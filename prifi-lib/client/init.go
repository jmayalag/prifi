package client

/**
 * PriFi Client
 * ************
 * This regroups the behavior of the PriFi client.
 * Needs to be instantiated via the PriFiProtocol in prifi.go
 * Then, this file simple handle the answer to the different message kind :
 *
 * - ALL_ALL_SHUTDOWN - kill this client
 * - ALL_ALL_PARAMETERS (specialized into ALL_CLI_PARAMETERS) - used to initialize the client over the network / overwrite its configuration
 * - REL_CLI_TELL_TRUSTEES_PK - the trustee's identities. We react by sending our identity + ephemeral identity
 * - REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG - the shuffle from the trustees. We do some check, if they pass, we can communicate. We send the first round to the relay.
 * - REL_CLI_DOWNSTREAM_DATA - the data from the relay, for one round. We react by finishing the round (sending our data to the relay)
 *
 * local functions :
 *
 * ProcessDownStreamData() <- is called by Received_REL_CLI_DOWNSTREAM_DATA; it handles the raw data received
 * SendUpstreamData() <- it is called at the end of ProcessDownStreamData(). Hence, after getting some data down, we send some data up.
 *
 * TODO : traffic need to be encrypted
 */

import (
	"errors"
	"strconv"

	"github.com/dedis/cothority/log"
	"github.com/dedis/crypto/abstract"
	"github.com/lbarman/prifi/prifi-lib/config"
	"github.com/lbarman/prifi/prifi-lib/crypto"
	"github.com/lbarman/prifi/prifi-lib/dcnet"
	prifilog "github.com/lbarman/prifi/prifi-lib/log"
	"github.com/lbarman/prifi/prifi-lib/net"
	"reflect"
)

// Possible states of the clients. This restrict the kind of messages they can receive at a given point in time.
const (
	CLIENT_STATE_BEFORE_INIT int16 = iota
	CLIENT_STATE_INITIALIZING
	CLIENT_STATE_EPH_KEYS_SENT
	CLIENT_STATE_READY
	CLIENT_STATE_SHUTDOWN
)

// ClientState contains the mutable state of the client.
type ClientState struct {
	CellCoder                 dcnet.CellCoder
	currentState              int16
	DataForDCNet              chan []byte //Data to the relay : VPN / SOCKS should put data there !
	DataFromDCNet             chan []byte //Data from the relay : VPN / SOCKS should read data from there !
	DataOutputEnabled         bool        //if FALSE, nothing will be written to DataFromDCNet
	ephemeralPrivateKey       abstract.Scalar
	EphemeralPublicKey        abstract.Point
	ID                        int
	LatencyTest               bool
	MySlot                    int
	Name                      string
	nClients                  int
	nTrustees                 int
	PayloadLength             int
	privateKey                abstract.Scalar
	PublicKey                 abstract.Point
	sharedSecrets             []abstract.Point
	TrusteePublicKey          []abstract.Point
	UsablePayloadLength       int
	UseSocksProxy             bool
	UseUDP                    bool
	MessageHistory            abstract.Cipher
	StartStopReceiveBroadcast chan bool
	statistics                *prifilog.LatencyStatistics

	//concurrent stuff
	RoundNo           int32
	BufferedRoundData map[int32]net.REL_CLI_DOWNSTREAM_DATA
}

// PriFiLibInstance contains the mutable state of a PriFi entity.
type PriFiLibClientInstance struct {
	messageSender *net.MessageSenderWrapper
	clientState   ClientState
}

// NewPriFiClient creates a new PriFi client entity state.
// Note: the returned state is not sufficient for the PrFi protocol
// to start; this entity will expect a ALL_ALL_PARAMETERS message as
// first received message to complete it's state.
func NewPriFiClient(msgSender *net.MessageSenderWrapper) *PriFiLibClientInstance {
	prifi := PriFiLibClientInstance{
		messageSender: msgSender,
	}
	return &prifi
}

// NewPriFiClientWithState creates a new PriFi client entity state.
func NewPriFiClientWithState(msgSender *net.MessageSenderWrapper, state *ClientState) *PriFiLibClientInstance {
	prifi := PriFiLibClientInstance{
		messageSender: msgSender,
		clientState:   *state,
	}
	log.Lvl1("Client has been initialized by function call. ")

	log.Lvl2("Client " + strconv.Itoa(prifi.clientState.ID) + " : starting the broadcast-listener goroutine")
	go prifi.messageSender.MessageSender.ClientSubscribeToBroadcast(prifi.clientState.Name, prifi.ReceivedMessage, prifi.clientState.StartStopReceiveBroadcast)
	return &prifi
}

// NewClientState is used to initialize the state of the client. Must be called before anything else.
func NewClientState(clientID int, nTrustees int, nClients int, payloadLength int, latencyTest bool, useUDP bool, dataOutputEnabled bool, dataForDCNet chan []byte, dataFromDCNet chan []byte) *ClientState {

	//set the defaults
	params := new(ClientState)
	params.ID = clientID
	params.Name = "Client-" + strconv.Itoa(clientID)
	params.CellCoder = config.Factory()
	params.DataForDCNet = dataForDCNet
	params.DataFromDCNet = dataFromDCNet
	params.DataOutputEnabled = dataOutputEnabled
	params.LatencyTest = latencyTest
	//params.MessageHistory =
	params.MySlot = -1
	params.nClients = nClients
	params.nTrustees = nTrustees
	params.PayloadLength = payloadLength
	params.UsablePayloadLength = params.CellCoder.ClientCellSize(payloadLength)
	params.UseSocksProxy = false //deprecated
	params.UseUDP = useUDP
	params.RoundNo = int32(0)
	params.BufferedRoundData = make(map[int32]net.REL_CLI_DOWNSTREAM_DATA)
	params.StartStopReceiveBroadcast = make(chan bool)
	params.statistics = prifilog.NewLatencyStatistics()

	// Generate pk
	params.PublicKey, params.privateKey = crypto.NewKeyPair()

	//placeholders for pubkeys and secrets
	params.TrusteePublicKey = make([]abstract.Point, nTrustees)
	params.sharedSecrets = make([]abstract.Point, nTrustees)

	//sets the new state
	params.currentState = CLIENT_STATE_INITIALIZING

	return params
}

// ReceivedMessage must be called when a PriFi host receives a message.
// It takes care to call the correct message handler function.
func (p *PriFiLibClientInstance) ReceivedMessage(msg interface{}) error {

	var err error

	switch typedMsg := msg.(type) {
	case net.ALL_ALL_PARAMETERS_NEW:
		err = p.Received_ALL_ALL_PARAMETERS(typedMsg) //todo change this name
	case net.ALL_ALL_SHUTDOWN:
		err = p.Received_ALL_ALL_SHUTDOWN(typedMsg)
	case net.REL_CLI_DOWNSTREAM_DATA:
		err = p.Received_REL_CLI_DOWNSTREAM_DATA(typedMsg)
	/*
	 * this message is a bit special. At this point, we don't care anymore that's it's UDP, and cast it back to REL_CLI_DOWNSTREAM_DATA.
	 * the relay only handles REL_CLI_DOWNSTREAM_DATA
	 */
	case net.REL_CLI_DOWNSTREAM_DATA_UDP:
		err = p.Received_REL_CLI_UDP_DOWNSTREAM_DATA(typedMsg.REL_CLI_DOWNSTREAM_DATA)
	case net.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG:
		err = p.Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG(typedMsg)
	case net.REL_CLI_TELL_TRUSTEES_PK:
		err = p.Received_REL_CLI_TELL_TRUSTEES_PK(typedMsg)
	default:
		err = errors.New("Unrecognized message, type" + reflect.TypeOf(msg).String())
	}

	return err
}
