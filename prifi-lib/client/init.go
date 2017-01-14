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
	clientState   *ClientState
}

// NewClient creates a new PriFi client entity state.
func NewClient(doLatencyTest bool, dataOutputEnabled bool, dataForDCNet chan []byte, dataFromDCNet chan []byte, msgSender *net.MessageSenderWrapper) *PriFiLibClientInstance {

	clientState := new(ClientState)

	//instantiates the static stuff
	clientState.statistics = prifilog.NewLatencyStatistics()
	clientState.PublicKey, clientState.privateKey = crypto.NewKeyPair()
	clientState.RoundNo = int32(0)
	clientState.BufferedRoundData = make(map[int32]net.REL_CLI_DOWNSTREAM_DATA)
	clientState.StartStopReceiveBroadcast = make(chan bool)
	clientState.LatencyTest = doLatencyTest
	clientState.CellCoder = config.Factory()
	clientState.DataForDCNet = dataForDCNet
	clientState.DataFromDCNet = dataFromDCNet
	clientState.DataOutputEnabled = dataOutputEnabled

	prifi := PriFiLibClientInstance{
		messageSender: msgSender,
		clientState:   clientState,
	}

	return &prifi
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
