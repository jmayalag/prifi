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
	"github.com/dedis/prifi/prifi-lib/crypto"
	"github.com/dedis/prifi/prifi-lib/dcnet"
	prifilog "github.com/dedis/prifi/prifi-lib/log"
	"github.com/dedis/prifi/prifi-lib/net"
	"github.com/dedis/prifi/prifi-lib/utils"
	"gopkg.in/dedis/kyber.v2"
	"gopkg.in/dedis/onet.v2/log"
	"reflect"
	"strings"
	"time"
)

// ClientState contains the mutable state of the client.
type ClientState struct {
	DCNet                         *dcnet.DCNetEntity
	currentState                  int16
	DataForDCNet                  chan []byte //Data to the relay : VPN / SOCKS should put data there !
	NextDataForDCNet              *[]byte     //if not nil, send this before polling DataForDCNet
	DataFromDCNet                 chan []byte //Data from the relay : VPN / SOCKS should read data from there !
	DataOutputEnabled             bool        //if FALSE, nothing will be written to DataFromDCNet
	ephemeralPrivateKey           kyber.Scalar
	EphemeralPublicKey            kyber.Point
	ID                            int
	LatencyTest                   *prifilog.LatencyTests
	MySlot                        int
	Name                          string
	nClients                      int
	nTrustees                     int
	PayloadSize                   int
	privateKey                    kyber.Scalar
	PublicKey                     kyber.Point
	sharedSecrets                 []kyber.Point
	TrusteePublicKey              []kyber.Point
	UseSocksProxy                 bool
	UseUDP                        bool
	MessageHistory                kyber.XOF
	StartStopReceiveBroadcast     chan bool
	timeStatistics                map[string]*prifilog.TimeStatistics
	pcapReplay                    *PCAPReplayer
	DisruptionProtectionEnabled   bool
	LastWantToSend                time.Time
	EquivocationProtectionEnabled bool

	//concurrent stuff
	RoundNo           int32
	BufferedRoundData map[int32]net.REL_CLI_DOWNSTREAM_DATA
}

// PCAPReplayer handles the data needed to replay some .pcap file
type PCAPReplayer struct {
	Enabled       bool
	PCAPFolder    string
	PCAPFile      string
	Packets       []utils.Packet
	currentPacket int
	time0         uint64
}

// PriFiLibInstance contains the mutable state of a PriFi entity.
type PriFiLibClientInstance struct {
	messageSender *net.MessageSenderWrapper
	clientState   *ClientState
	stateMachine  *utils.StateMachine
}

// NewClient creates a new PriFi client entity state.
func NewClient(doLatencyTest bool, dataOutputEnabled bool, dataForDCNet chan []byte, dataFromDCNet chan []byte, doReplayPcap bool, pcapFolder string, msgSender *net.MessageSenderWrapper) *PriFiLibClientInstance {

	clientState := new(ClientState)

	//instantiates the static stuff
	clientState.PublicKey, clientState.privateKey = crypto.NewKeyPair()
	//clientState.StartStopReceiveBroadcast = make(chan bool) //this should stay nil, !=nil -> we have a listener goroutine active
	clientState.LatencyTest = &prifilog.LatencyTests{
		DoLatencyTests:       doLatencyTest,
		LatencyTestsInterval: 2 * time.Second,
		NextLatencyTest:      time.Now(),
		LatencyTestsToSend:   make([]*prifilog.LatencyTestToSend, 0),
	}
	clientState.timeStatistics = make(map[string]*prifilog.TimeStatistics)
	clientState.timeStatistics["latency-msg-stayed-in-buffer"] = prifilog.NewTimeStatistics()
	clientState.timeStatistics["measured-latency"] = prifilog.NewTimeStatistics()
	clientState.timeStatistics["round-processing"] = prifilog.NewTimeStatistics()
	clientState.DataForDCNet = dataForDCNet
	clientState.NextDataForDCNet = nil
	clientState.DataFromDCNet = dataFromDCNet
	clientState.DataOutputEnabled = dataOutputEnabled
	clientState.LastWantToSend = time.Now()
	clientState.pcapReplay = &PCAPReplayer{
		Enabled:    doReplayPcap,
		PCAPFolder: pcapFolder,
		time0:      uint64(MsTimeStampNow()),
	}

	//init the state machine
	states := []string{"BEFORE_INIT", "EPH_KEYS_SENT", "READY", "BLAMING", "SHUTDOWN"}
	sm := new(utils.StateMachine)
	logFn := func(s interface{}) {
		log.Lvl2(s)
	}
	errFn := func(s interface{}) {
		if strings.Contains(s.(string), ", but in state SHUTDOWN") { //it's an "acceptable error"
			log.Lvl2(s)
		} else {
			log.Fatal(s)
		}
	}
	sm.Init(states, logFn, errFn)

	prifi := PriFiLibClientInstance{
		messageSender: msgSender,
		clientState:   clientState,
		stateMachine:  sm,
	}

	return &prifi
}

// ReceivedMessage must be called when a PriFi host receives a message.
// It takes care to call the correct message handler function.
func (p *PriFiLibClientInstance) ReceivedMessage(msg interface{}) error {

	var err error

	switch typedMsg := msg.(type) {
	case net.ALL_ALL_PARAMETERS:
		if typedMsg.ForceParams || p.stateMachine.AssertState("BEFORE_INIT") {
			err = p.Received_ALL_ALL_PARAMETERS(typedMsg)
		}
	case net.ALL_ALL_SHUTDOWN:
		err = p.Received_ALL_ALL_SHUTDOWN(typedMsg)
	case net.REL_CLI_DOWNSTREAM_DATA:
		if p.stateMachine.AssertState("READY") {
			err = p.Received_REL_CLI_DOWNSTREAM_DATA(typedMsg)
		}
	case net.REL_CLI_DOWNSTREAM_DATA_UDP:
		if p.stateMachine.AssertState("READY") {
			err = p.Received_REL_CLI_UDP_DOWNSTREAM_DATA(typedMsg)
		}
	case net.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG:
		if p.stateMachine.AssertState("EPH_KEYS_SENT") {
			err = p.Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG(typedMsg)
		}
	default:
		err = errors.New("Unrecognized message, type" + reflect.TypeOf(msg).String())
	}

	return err
}
