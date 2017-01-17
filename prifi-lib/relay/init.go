package relay

/*
PriFi Relay
************
This regroups the behavior of the PriFi relay.
Needs to be instantiated via the PriFiProtocol in prifi.go
Then, this file simple handle the answer to the different message kind :

- ALL_ALL_SHUTDOWN - kill this relay
- ALL_ALL_PARAMETERS (specialized into ALL_REL_PARAMETERS) - used to initialize the relay over the network / overwrite its configuration
- TRU_REL_TELL_PK - when a trustee connects, he tells us his public key
- CLI_REL_TELL_PK_AND_EPH_PK - when they receive the list of the trustees, each clients tells his identity. when we have all client's IDs,
								  we send them to the trustees to shuffle (Schedule protocol)
- TRU_REL_TELL_NEW_BASE_AND_EPH_PKS - when we receive the result of one shuffle, we forward it to the next trustee
- TRU_REL_SHUFFLE_SIG - when the shuffle has been done by all trustee, we send the transcript, and they answer with a signature, which we
						   broadcast to the clients
- CLI_REL_UPSTREAM_DATA - data for the DC-net
- REL_CLI_UDP_DOWNSTREAM_DATA - is NEVER received here, but casted to CLI_REL_UPSTREAM_DATA by messages.go
- TRU_REL_DC_CIPHER - data for the DC-net

local functions :

ConnectToTrustees() - simple helper
finalizeUpstreamData() - called after some Receive_CLI_REL_UPSTREAM_DATA, when we have all ciphers.
sendDownstreamData() - called after a finalizeUpstreamData(), to continue the communication
checkIfRoundHasEndedAfterTimeOut_Phase1() - called by sendDownstreamData(), which starts a new round. After some short time, if the round hasn't changed, and we used UDP,
											   retransmit messages to client over TCP
checkIfRoundHasEndedAfterTimeOut_Phase2() - called by checkIfRoundHasEndedAfterTimeOut_Phase1(). After some long time, entities that didn't send us data should be
considered disconnected

*/

import (
	"errors"
	"strconv"
	"time"

	"github.com/dedis/crypto/abstract"
	"github.com/lbarman/prifi/prifi-lib/config"
	"github.com/lbarman/prifi/prifi-lib/dcnet"
	prifilog "github.com/lbarman/prifi/prifi-lib/log"
	"github.com/lbarman/prifi/prifi-lib/net"
	"github.com/lbarman/prifi/prifi-lib/scheduler"

	"github.com/lbarman/prifi/prifi-lib/crypto"
	"reflect"
)

// PriFiLibInstance contains the mutable state of a PriFi entity.
type PriFiLibRelayInstance struct {
	messageSender *net.MessageSenderWrapper
	relayState    *RelayState
}

// NewPriFiRelay creates a new PriFi relay entity state.
// Note: the returned state is not sufficient for the PrFi protocol
// to start; this entity will expect a ALL_ALL_PARAMETERS message as
// first received message to complete it's state.
func NewRelay(dataOutputEnabled bool, dataForClients chan []byte, dataFromDCNet chan []byte, experimentResultChan chan interface{}, timeoutHandler func([]int, []int), msgSender *net.MessageSenderWrapper) *PriFiLibRelayInstance {
	relayState := new(RelayState)

	//init the static stuff
	relayState.CellCoder = config.Factory()
	relayState.DataForClients = dataForClients
	relayState.DataFromDCNet = dataFromDCNet
	relayState.DataOutputEnabled = dataOutputEnabled
	relayState.timeoutHandler = timeoutHandler
	relayState.ExperimentResultChannel = experimentResultChan
	relayState.PriorityDataForClients = make(chan []byte, 10) // This is used for relay's control message (like latency-tests)
	relayState.statistics = prifilog.NewBitRateStatistics()
	relayState.PublicKey, relayState.privateKey = crypto.NewKeyPair()
	relayState.bufferManager = new(BufferManager)
	neffShuffle := new(scheduler.NeffShuffle)
	neffShuffle.Init()
	relayState.neffShuffle = neffShuffle.RelayView
	relayState.Name = "Relay"

	prifi := PriFiLibRelayInstance{
		messageSender: msgSender,
		relayState:    relayState,
	}
	return &prifi
}

//The time slept between each round
const PROCESSING_LOOP_SLEEP_TIME = 0 * time.Millisecond

//The timeout before retransmission. Here of 0, since we have only TCP. to be increase with UDP
const TIMEOUT_PHASE_1 = 1 * time.Second

//The timeout before kicking a client/trustee
const TIMEOUT_PHASE_2 = 1 * time.Second

// Number of ciphertexts buffered by trustees. When <= TRUSTEE_CACHE_LOWBOUND, resume sending
const TRUSTEE_CACHE_LOWBOUND = 1

// Number of ciphertexts buffered by trustees. When >= TRUSTEE_CACHE_HIGHBOUND, stop sending
const TRUSTEE_CACHE_HIGHBOUND = 10

// Possible states the trustees are in. This restrict the kind of messages they can receive at a given point in time.
const (
	RELAY_STATE_BEFORE_INIT int16 = iota
	RELAY_STATE_COLLECTING_TRUSTEES_PKS
	RELAY_STATE_COLLECTING_CLIENT_PKS
	RELAY_STATE_COLLECTING_SHUFFLES
	RELAY_STATE_COLLECTING_SHUFFLE_SIGNATURES
	RELAY_STATE_COMMUNICATING
	RELAY_STATE_SHUTDOWN
)

// NodeRepresentation regroups the information about one client or trustee.
type NodeRepresentation struct {
	ID                 int
	Connected          bool
	PublicKey          abstract.Point
	EphemeralPublicKey abstract.Point
}

// DCNetRound counts how many (upstream) messages we received for a given DC-net round.
type DCNetRound struct {
	currentRound    int32
	dataAlreadySent net.REL_CLI_DOWNSTREAM_DATA
	startTime       time.Time
}

// RelayState contains the mutable state of the relay.
type RelayState struct {
	bufferManager                     *BufferManager
	CellCoder                         dcnet.CellCoder
	clients                           []NodeRepresentation
	currentDCNetRound                 DCNetRound
	neffShuffle                       *scheduler.NeffShuffleRelay
	currentState                      int16
	DataForClients                    chan []byte // VPN / SOCKS should put data there !
	PriorityDataForClients            chan []byte
	DataFromDCNet                     chan []byte // VPN / SOCKS should read data from there !
	DataOutputEnabled                 bool        // If FALSE, nothing will be written to DataFromDCNet
	DownstreamCellSize                int
	MessageHistory                    abstract.Cipher
	Name                              string
	nClients                          int
	nClientsPkCollected              int
	nTrustees                         int
	nTrusteesPkCollected              int
	privateKey                        abstract.Scalar
	PublicKey                         abstract.Point
	ExperimentRoundLimit              int
	trustees                          []NodeRepresentation
	UpstreamCellSize                  int
	UseDummyDataDown                  bool
	UseUDP                            bool
	numberOfNonAckedDownstreamPackets int
	WindowSize                        int
	nextDownStreamRoundToSend         int32
	ExperimentResultChannel           chan interface{}
	ExperimentResultData              interface{}
	timeoutHandler                    func([]int, []int)
	statistics                        *prifilog.BitrateStatistics
}

// ReceivedMessage must be called when a PriFi host receives a message.
// It takes care to call the correct message handler function.
func (p *PriFiLibRelayInstance) ReceivedMessage(msg interface{}) error {

	var err error

	switch typedMsg := msg.(type) {
	case *net.ALL_ALL_PARAMETERS_NEW:
		err = p.Received_ALL_ALL_PARAMETERS(*typedMsg)
	case net.ALL_ALL_SHUTDOWN:
		err = p.Received_ALL_ALL_SHUTDOWN(typedMsg)
	case net.CLI_REL_TELL_PK_AND_EPH_PK:
		err = p.Received_CLI_REL_TELL_PK_AND_EPH_PK(typedMsg)
	case net.CLI_REL_UPSTREAM_DATA:
		err = p.Received_CLI_REL_UPSTREAM_DATA(typedMsg)
	case net.TRU_REL_DC_CIPHER:
		err = p.Received_TRU_REL_DC_CIPHER(typedMsg)
	case net.TRU_REL_SHUFFLE_SIG:
		err = p.Received_TRU_REL_SHUFFLE_SIG(typedMsg)
	case net.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS:
		err = p.Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS(typedMsg)
	case net.TRU_REL_TELL_PK:
		err = p.Received_TRU_REL_TELL_PK(typedMsg)
	default:
		err = errors.New("Unrecognized message, type" + reflect.TypeOf(msg).String())
	}

	return err
}

func relayStateStr(state int16) string {
	switch state {
	case RELAY_STATE_BEFORE_INIT:
		return "RELAY_STATE_BEFORE_INIT"
	case RELAY_STATE_COLLECTING_TRUSTEES_PKS:
		return "RELAY_STATE_COLLECTING_TRUSTEES_PKS"
	case RELAY_STATE_COLLECTING_CLIENT_PKS:
		return "RELAY_STATE_COLLECTING_CLIENT_PKS"
	case RELAY_STATE_COLLECTING_SHUFFLES:
		return "RELAY_STATE_COLLECTING_SHUFFLES"
	case RELAY_STATE_COLLECTING_SHUFFLE_SIGNATURES:
		return "RELAY_STATE_COLLECTING_SHUFFLE_SIGNATURES"
	case RELAY_STATE_COMMUNICATING:
		return "RELAY_STATE_COMMUNICATING"
	case RELAY_STATE_SHUTDOWN:
		return "RELAY_STATE_SHUTDOWN"
	default:
		return "unknown state (" + strconv.Itoa(int(state)) + ")"
	}
}
