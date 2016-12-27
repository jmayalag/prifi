package prifi_lib

/*
PriFi Relay
************
This regroups the behavior of the PriFi relay.
Needs to be instantiated via the PriFiProtocol in prifi.go
Then, this file simple handle the answer to the different message kind :
Always make sure to use the locks in the lockPool in the order they appear:
Locking Guideline: Assume lock A appears before lock B in the lockPool and you want to lock A and B, lock A before B. If B is locked and you want to lock A, unlock B then lock A and re-lock B.

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

TODO : We should timeout if some client did not send anything after a while
TODO : given the number of already-buffered Ciphers (per trustee), we need to tell him to slow down
TODO : if something went wrong before, this flag should be used to warn the clients that the config has changed
TODO : sanity check that we don't have twice the same client
TODO : create a "send" function that takes as parameter the data we want and the message to print if an error occurs (since the sending block always looks the same)
*/

import (
	"encoding/binary"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/dedis/cothority/log"
	"github.com/dedis/crypto/abstract"
	"github.com/lbarman/prifi/prifi-lib/config"
	"github.com/lbarman/prifi/prifi-lib/dcnet"
	prifilog "github.com/lbarman/prifi/prifi-lib/log"

	socks "github.com/lbarman/prifi/prifi-socks"
)

//The time slept between each round
const PROCESSING_LOOP_SLEEP_TIME = 0 * time.Second

// Number of ciphertexts buffered by trustees
const TRUSTEE_WINDOW_SIZE = 10

// Trustees stop sending when capacity <= TRUSTEE_WINDOW_LOWER_LIMIT
const TRUSTEE_WINDOW_LOWER_LIMIT = 1

// Trustees resume sending when capacity = TRUSTEE_WINDOW_LOWER_LIMIT + TRUSTEE_RESUME_SENDING_RATIO*(TRUSTEE_WINDOW_SIZE - TRUSTEE_WINDOW_LOWER_LIMIT)
const TRUSTEE_RESUME_SENDING_RATIO = 0.9

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

// NeffShuffleState is where the Neff Shuffles are accumulated during the Schedule protocol.
type NeffShuffleState struct {
	ClientPublicKeys  []abstract.Point
	Gs                []abstract.Scalar
	ephPubKeys        [][]abstract.Point
	proofs            [][]byte
	nextFreeId_Proofs int
	signatures        [][]byte
	signatureCount    int
}

// DCNetRound counts how many (upstream) messages we received for a given DC-net round.
type DCNetRound struct {
	currentRound       int32
	trusteeCipherCount int
	clientCipherCount  int
	clientCipherAck    map[int]bool
	trusteeCipherAck   map[int]bool
	dataAlreadySent    REL_CLI_DOWNSTREAM_DATA
	startTime          time.Time
}

// hasAllCiphers tests if we received all DC-net ciphers (1 per client, 1 per trustee)
func (dcnet *DCNetRound) hasAllCiphers(p *PriFiLibInstance) bool {
	if p.relayState.nClients == dcnet.clientCipherCount && p.relayState.nTrustees == dcnet.trusteeCipherCount {
		return true
	}
	return false
}

// BufferedCipher holds the ciphertexts received in advance from the trustees.
type BufferedCipher struct {
	RoundID int32
	Data    map[int][]byte
}

// lockPool contains the locks used to ensure thread-safe concurrency
// To avoid deadlocks, make sure to ALWAYS use the locks in the order they appear in the lockPool (this means an unlock and a re-lock of a variable is sometimes required in places where it seems redundant to unlock that variable)
// DO NOT rearrange these locks, NEW locks should be appended to the lockPool
type lockPool struct {
	round         sync.RWMutex
	coder         sync.RWMutex
	trusteeBuffer sync.RWMutex
	clientBuffer  sync.RWMutex
	cipherTracker sync.RWMutex
	clients       sync.RWMutex
	shuffle       sync.RWMutex
	state         sync.RWMutex
	nTrusteePK    sync.RWMutex
	trustees      sync.RWMutex
	expData       sync.RWMutex

	// add new locks here
}

// RelayState contains the mutable state of the relay.
type RelayState struct {
	// RelayPort				string
	// PublicKey				abstract.Point
	// privateKey			abstract.Scalar
	// trusteesHosts			[]string

	bufferedTrusteeCiphers            map[int32]BufferedCipher
	bufferedClientCiphers             map[int32]BufferedCipher
	trusteeCipherTracker              []int
	CellCoder                         dcnet.CellCoder
	clients                           []NodeRepresentation
	currentDCNetRound                 DCNetRound
	currentShuffleTranscript          NeffShuffleState
	currentState                      int16
	DataForClients                    chan []byte // VPN / SOCKS should put data there !
	PriorityDataForClients            chan []byte
	DataFromDCNet                     chan []byte // VPN / SOCKS should read data from there !
	DataOutputEnabled                 bool        // If FALSE, nothing will be written to DataFromDCNet
	DownstreamCellSize                int
	MessageHistory                    abstract.Cipher
	Name                              string
	nClients                          int
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
	locks                             *lockPool
	timeoutHandler                    func([]int, []int)
	statistics                        *prifilog.BitrateStatistics
}

/*
NewRelayState initializes the state of this relay.
It must be called before anything else.
*/
func NewRelayState(nTrustees int, nClients int, upstreamCellSize int, downstreamCellSize int, windowSize int, useDummyDataDown bool, experimentRoundLimit int, experimentResultChan chan interface{}, useUDP bool, dataOutputEnabled bool, dataForClients chan []byte, dataFromDCNet chan []byte) *RelayState {
	params := new(RelayState)
	params.Name = "Relay"
	params.CellCoder = config.Factory()
	params.clients = make([]NodeRepresentation, 0)
	params.DataForClients = dataForClients
	params.PriorityDataForClients = make(chan []byte, 10) // This is used for relay's control message (like latency-tests)
	params.DataFromDCNet = dataFromDCNet
	params.DataOutputEnabled = dataOutputEnabled
	params.DownstreamCellSize = downstreamCellSize
	// params.MessageHistory =
	params.nClients = nClients
	params.ExperimentResultChannel = experimentResultChan
	params.nTrustees = nTrustees
	params.nTrusteesPkCollected = 0
	params.ExperimentRoundLimit = experimentRoundLimit
	params.trusteeCipherTracker = make([]int, nTrustees)
	params.trustees = make([]NodeRepresentation, nTrustees)
	params.UpstreamCellSize = upstreamCellSize
	params.UseDummyDataDown = useDummyDataDown
	params.UseUDP = useUDP
	params.WindowSize = windowSize
	params.nextDownStreamRoundToSend = int32(1) //since first round is half-round
	params.numberOfNonAckedDownstreamPackets = 0
	params.statistics = prifilog.NewBitRateStatistics()
	params.locks = &lockPool{}

	// Prepare the crypto parameters
	rand := config.CryptoSuite.Cipher([]byte(params.Name))
	base := config.CryptoSuite.Point().Base()

	// Generate own parameters
	params.privateKey = config.CryptoSuite.Scalar().Pick(rand)
	params.PublicKey = config.CryptoSuite.Point().Mul(base, params.privateKey)

	// Sets the new state
	params.currentState = RELAY_STATE_COLLECTING_TRUSTEES_PKS

	return params
}

/*
Received_ALL_REL_SHUTDOWN handles ALL_REL_SHUTDOWN messages.
When we receive this message, we should warn other protocol participants and clean resources.
*/
func (p *PriFiLibInstance) Received_ALL_REL_SHUTDOWN(msg ALL_ALL_SHUTDOWN) error {
	log.Lvl1("Relay : Received a SHUTDOWN message. ")

	p.relayState.locks.state.Lock()
	p.relayState.currentState = RELAY_STATE_SHUTDOWN
	p.relayState.locks.state.Unlock()

	msg2 := &ALL_ALL_SHUTDOWN{}

	var err error

	// Send this shutdown to all trustees
	for j := 0; j < p.relayState.nTrustees; j++ {
		err := p.messageSender.SendToTrustee(j, msg2)
		if err != nil {
			e := "Could not send ALL_TRU_SHUTDOWN to Trustee " + strconv.Itoa(j) + ", error is " + err.Error()
			log.Error(e)
			err = errors.New(e)
		} else {
			log.Lvl3("Relay : sent ALL_TRU_PARAMETERS to Trustee " + strconv.Itoa(j) + ".")
		}
	}

	// Send this shutdown to all clients
	for j := 0; j < p.relayState.nClients; j++ {
		err := p.messageSender.SendToClient(j, msg2)
		if err != nil {
			e := "Could not send ALL_TRU_SHUTDOWN to Client " + strconv.Itoa(j) + ", error is " + err.Error()
			log.Error(e)
			err = errors.New(e)
		} else {
			log.Lvl3("Relay : sent ALL_TRU_PARAMETERS to Client " + strconv.Itoa(j) + ".")
		}
	}

	// TODO : stop all go-routines we created

	return err
}

/*
Received_ALL_REL_PARAMETERS handles ALL_REL_PARAMETERS.
It initializes the relay with the parameters contained in the message.
*/
func (p *PriFiLibInstance) Received_ALL_REL_PARAMETERS(msg ALL_ALL_PARAMETERS) error {

	p.relayState.locks.state.Lock()

	// This can only happens in the state RELAY_STATE_BEFORE_INIT
	if p.relayState.currentState != RELAY_STATE_BEFORE_INIT && !msg.ForceParams {
		log.Lvl1("Relay : Received a ALL_ALL_PARAMETERS, but not in state RELAY_STATE_BEFORE_INIT, ignoring. ")
		p.relayState.locks.state.Unlock()
		return nil
	} else if p.relayState.currentState != RELAY_STATE_BEFORE_INIT && msg.ForceParams {
		log.Lvl1("Relay : Received a ALL_ALL_PARAMETERS && ForceParams = true, processing. ")
	} else {
		log.Lvl3("Relay : received ALL_ALL_PARAMETERS")
	}
	p.relayState.locks.state.Unlock()

	oldExperimentResultChan := p.relayState.ExperimentResultChannel
	p.relayState = *NewRelayState(msg.NTrustees, msg.NClients, msg.UpCellSize, msg.DownCellSize, msg.RelayWindowSize, msg.RelayUseDummyDataDown, msg.RelayReportingLimit, oldExperimentResultChan, msg.UseUDP, msg.RelayDataOutputEnabled, make(chan []byte), make(chan []byte))

	log.Lvlf5("%+v\n", p.relayState)
	log.Lvl1("Relay has been initialized by message. ")

	p.relayState.locks.state.Lock()
	// Broadcast those parameters to the other nodes, then tell the trustees which ID they are.
	if msg.StartNow {
		p.relayState.currentState = RELAY_STATE_COLLECTING_TRUSTEES_PKS
		p.ConnectToTrustees()
	}
	log.Lvl1("Relay setup done, and setup sent to the trustees.")
	p.relayState.locks.state.Unlock()

	return nil
}

// ConnectToTrustees connects to the trustees and initializes them with default parameters.
func (p *PriFiLibInstance) ConnectToTrustees() error {

	// Craft default parameters
	// TODO : if the parameters are not constants anymore, we need a way to change those fields. For now, trustees don't need much information
	var msg = &ALL_ALL_PARAMETERS{
		NClients:          p.relayState.nClients,
		NextFreeTrusteeID: 0,
		NTrustees:         p.relayState.nTrustees,
		StartNow:          true,
		ForceParams:       true,
		UpCellSize:        p.relayState.UpstreamCellSize,
	}

	log.Lvl1("Sending ALL_TRU_PARAMETERS")
	log.Lvlf1("%+v\n", msg)

	// Send those parameters to all trustees
	for j := 0; j < p.relayState.nTrustees; j++ {

		// The ID is unique !
		msg.NextFreeTrusteeID = j
		err := p.messageSender.SendToTrustee(j, msg)
		if err != nil {
			e := "Could not send ALL_TRU_PARAMETERS to Trustee " + strconv.Itoa(j) + ", error is " + err.Error()
			log.Error(e)
			return errors.New(e)
		}
		log.Lvl3("Relay : sent ALL_TRU_PARAMETERS to Trustee " + strconv.Itoa(j) + ".")
	}

	return nil
}

/*
Received_CLI_REL_UPSTREAM_DATA handles CLI_REL_UPSTREAM_DATA messages and is part of PriFi's main loop.
This is what happens in one round, for the relay. We receive some upstream data.
If we have collected data from all entities for this round, we can call DecodeCell() and get the output.
If we get data for another round (in the future) we should buffer it.
If we finished a round (we had collected all data, and called DecodeCell()), we need to finish the round by sending some data down.
Either we send something from the SOCKS/VPN buffer, or we answer the latency-test message if we received any, or we send 1 bit.
*/
func (p *PriFiLibInstance) Received_CLI_REL_UPSTREAM_DATA(msg CLI_REL_UPSTREAM_DATA) error {
	p.relayState.locks.state.Lock()
	// This can only happens in the state RELAY_STATE_COMMUNICATING
	if p.relayState.currentState != RELAY_STATE_COMMUNICATING {
		e := "Relay : Received a CLI_REL_UPSTREAM_DATA, but not in state RELAY_STATE_COMMUNICATING, in state " + relayStateStr(p.relayState.currentState)
		log.Error(e)
		p.relayState.locks.state.Unlock()
		// return errors.New(e)
	} else {
		p.relayState.locks.state.Unlock()
		log.Lvl3("Relay : received CLI_REL_UPSTREAM_DATA from client " + strconv.Itoa(msg.ClientID) + " for round " + strconv.Itoa(int(msg.RoundID)))

	}

	p.relayState.locks.round.Lock()

	// if this is not the message destinated for this round, discard it ! (we are in lock-step)
	if p.relayState.currentDCNetRound.currentRound != msg.RoundID {

		e := "Relay : Client sent DC-net cipher for round " + strconv.Itoa(int(msg.RoundID)) + " but current round is " + strconv.Itoa(int(p.relayState.currentDCNetRound.currentRound))
		log.Lvl3(e)

		p.relayState.locks.clientBuffer.Lock()
		// else, we need to buffer this message somewhere
		if _, ok := p.relayState.bufferedClientCiphers[msg.RoundID]; ok {
			// the roundId already exists, simply add data
			p.relayState.bufferedClientCiphers[msg.RoundID].Data[msg.ClientID] = msg.Data
		} else {
			// else, create the key in the map, and store the data
			newKey := BufferedCipher{msg.RoundID, make(map[int][]byte)}
			newKey.Data[msg.ClientID] = msg.Data
			p.relayState.bufferedClientCiphers[msg.RoundID] = newKey
		}

		p.relayState.locks.clientBuffer.Unlock()
		p.relayState.locks.round.Unlock()

	} else {
		// else, if this is the message we need for this round
		p.relayState.locks.coder.Lock()

		p.relayState.CellCoder.DecodeClient(msg.Data)
		p.relayState.currentDCNetRound.clientCipherCount++
		p.relayState.currentDCNetRound.clientCipherAck[msg.ClientID] = true

		p.relayState.locks.coder.Unlock()

		log.Lvl3("Relay collecting cells for round", p.relayState.currentDCNetRound.currentRound, ", ", p.relayState.currentDCNetRound.clientCipherCount, "/", p.relayState.nClients, ", ", p.relayState.currentDCNetRound.trusteeCipherCount, "/", p.relayState.nTrustees)

		if p.relayState.currentDCNetRound.hasAllCiphers(p) {
			p.relayState.locks.round.Unlock()

			log.Lvl2("Relay has collected all ciphers for round", p.relayState.currentDCNetRound.currentRound, ", decoding...")
			p.finalizeUpstreamData()

			//one round has just passed !

			// sleep so it does not go too fast for debug
			time.Sleep(PROCESSING_LOOP_SLEEP_TIME)

			// send the data down
			for i := p.relayState.numberOfNonAckedDownstreamPackets; i < p.relayState.WindowSize; i++ {
				log.Lvl3("Relay : Gonna send, non-acked packets is", p.relayState.numberOfNonAckedDownstreamPackets, "(window is", p.relayState.WindowSize, ")")
				p.sendDownstreamData()
			}

		} else {
			p.relayState.locks.round.Unlock()
		}

	}

	return nil
}

/*
Received_TRU_REL_DC_CIPHER handles TRU_REL_DC_CIPHER messages. Those contain a DC-net cipher from a Trustee.
If it's for this round, we call decode on it, and remember we received it.
If for a future round we need to Buffer it.
*/
func (p *PriFiLibInstance) Received_TRU_REL_DC_CIPHER(msg TRU_REL_DC_CIPHER) error {
	// TODO : given the number of already-buffered Ciphers (per trustee), we need to tell him to slow down

	p.relayState.locks.state.Lock()
	// this can only happens in the state RELAY_STATE_COMMUNICATING
	if p.relayState.currentState != RELAY_STATE_COMMUNICATING && p.relayState.currentState != RELAY_STATE_COLLECTING_SHUFFLE_SIGNATURES {
		e := "Relay : Received a TRU_REL_DC_CIPHER, but not in state RELAY_STATE_COMMUNICATING, in state " + relayStateStr(p.relayState.currentState)
		log.Error(e)
		return errors.New(e)
	}
	p.relayState.locks.state.Unlock()
	log.Lvl3("Relay : received TRU_REL_DC_CIPHER for round " + strconv.Itoa(int(msg.RoundID)) + " from trustee " + strconv.Itoa(msg.TrusteeID))

	p.relayState.locks.round.Lock()

	// if this is the message we need for this round
	if p.relayState.currentDCNetRound.currentRound == msg.RoundID {

		log.Lvl3("Relay collecting cells for round", p.relayState.currentDCNetRound.currentRound, ", ", p.relayState.currentDCNetRound.clientCipherCount, "/", p.relayState.nClients, ", ", p.relayState.currentDCNetRound.trusteeCipherCount, "/", p.relayState.nTrustees)

		p.relayState.locks.coder.Lock()

		p.relayState.CellCoder.DecodeTrustee(msg.Data)
		p.relayState.currentDCNetRound.trusteeCipherCount++
		p.relayState.currentDCNetRound.trusteeCipherAck[msg.TrusteeID] = true

		p.relayState.locks.coder.Unlock()

		if p.relayState.currentDCNetRound.hasAllCiphers(p) {
			p.relayState.locks.round.Unlock()

			log.Lvl2("Relay has collected all ciphers for round", p.relayState.currentDCNetRound.currentRound, ", decoding...")
			p.finalizeUpstreamData()

			// send the data down
			for i := p.relayState.numberOfNonAckedDownstreamPackets; i < p.relayState.WindowSize; i++ {
				log.Lvl3("Relay : Gonna send, non-acked packets is", p.relayState.numberOfNonAckedDownstreamPackets, "(window is", p.relayState.WindowSize, ")")
				p.sendDownstreamData()
			}
		} else {
			p.relayState.locks.round.Unlock()
		}

	} else {

		defer p.relayState.locks.round.Unlock()

		p.relayState.locks.trusteeBuffer.Lock()

		// else, we need to buffer this message somewhere
		if _, ok := p.relayState.bufferedTrusteeCiphers[msg.RoundID]; ok {
			// the roundId already exists, simply add data
			p.relayState.bufferedTrusteeCiphers[msg.RoundID].Data[msg.TrusteeID] = msg.Data
		} else {
			// else, create the key in the map, and store the data
			newKey := BufferedCipher{msg.RoundID, make(map[int][]byte)}
			newKey.Data[msg.TrusteeID] = msg.Data
			p.relayState.bufferedTrusteeCiphers[msg.RoundID] = newKey
		}

		p.relayState.locks.trusteeBuffer.Unlock()

		// Here is the control to regulate the trustees ciphers in case they should stop sending
		p.relayState.locks.cipherTracker.Lock()
		p.relayState.trusteeCipherTracker[msg.TrusteeID]++                                        // Increment the currently buffered ciphers for this trustee
		currentCapacity := TRUSTEE_WINDOW_SIZE - p.relayState.trusteeCipherTracker[msg.TrusteeID] // Get our remaining allowed capacity
		p.relayState.locks.cipherTracker.Unlock()

		if currentCapacity <= TRUSTEE_WINDOW_LOWER_LIMIT { // Check if the capacity is lower then allowed
			toSend := &REL_TRU_TELL_RATE_CHANGE{currentCapacity}
			err := p.messageSender.SendToTrustee(msg.TrusteeID, toSend) // send the trustee a message informing them of the capacity

			if err != nil {
				e := "Could not send REL_TRU_TELL_RATE_CHANGE to " + strconv.Itoa(msg.TrusteeID) + "-th trustee for round " + strconv.Itoa(int(p.relayState.currentDCNetRound.currentRound)) + ", error is " + err.Error()
				log.Error(e)
				return errors.New(e)
			}
			log.Lvl3("Relay : sent REL_TRU_TELL_RATE_CHANGE to " + strconv.Itoa(msg.TrusteeID) + "-th trustee for round " + strconv.Itoa(int(p.relayState.currentDCNetRound.currentRound)))
		}

	}
	return nil
}

/*
finalizeUpstreamData is simply called when the Relay has received all ciphertexts (one per client, one per trustee),
and is ready to finalize the
DC-net round by XORing everything together.
If it's a latency-test message, we send it back to the clients.
If we use SOCKS/VPN, give them the data.
*/
func (p *PriFiLibInstance) finalizeUpstreamData() error {

	// we decode the DC-net cell
	p.relayState.locks.coder.Lock()
	upstreamPlaintext := p.relayState.CellCoder.DecodeCell()
	p.relayState.locks.coder.Unlock()

	p.relayState.statistics.AddUpstreamCell(int64(len(upstreamPlaintext)))

	// check if we have a latency test message
	if len(upstreamPlaintext) >= 2 {
		pattern := int(binary.BigEndian.Uint16(upstreamPlaintext[0:2]))
		if pattern == 43690 { // 1010101010101010
			log.Lvl3("Relay : noticed a latency-test message, sending answer...")
			// then, we simply have to send it down
			p.relayState.PriorityDataForClients <- upstreamPlaintext
		}
	}

	if upstreamPlaintext == nil {
		// empty upstream cell
	}

	if len(upstreamPlaintext) != p.relayState.UpstreamCellSize {
		e := "Relay : DecodeCell produced wrong-size payload, " + strconv.Itoa(len(upstreamPlaintext)) + "!=" + strconv.Itoa(p.relayState.UpstreamCellSize)
		log.Error(e)
		return errors.New(e)
	}

	if p.relayState.DataOutputEnabled {
		packetType, _, _, _ := socks.ParseSocksHeaderFromBytes(upstreamPlaintext)

		switch packetType {
		case socks.SocksData, socks.SocksConnect, socks.StallCommunication, socks.ResumeCommunication:
			p.relayState.DataFromDCNet <- upstreamPlaintext

		default:
			break
		}

	}

	p.roundFinished()

	return nil
}

/*
sendDownstreamData is simply called when the Relay has processed the upstream cell from all clients, and is ready to finalize the round by sending the data down.
If it's a latency-test message, we send it back to the clients.
If we use SOCKS/VPN, give them the data.
Since after this function, we'll start receiving data for the next round, if we have buffered data for this next round, tell the state that we
have the data already (and we're not waiting on it). Clean the old data.
*/
func (p *PriFiLibInstance) sendDownstreamData() error {

	var downstreamCellContent []byte

	select {
	case downstreamCellContent = <-p.relayState.PriorityDataForClients:
		log.Lvl3("Relay : We have some priority data for the clients")
	// TODO : maybe we can pack more than one message here ?

	default:

	}

	// only if we don't have priority data for clients
	if downstreamCellContent == nil {
		select {

		// either select data from the data we have to send, if any
		case downstreamCellContent = <-p.relayState.DataForClients:
			log.Lvl3("Relay : We have some real data for the clients. ")

		default:
			downstreamCellContent = make([]byte, 1)
			log.Lvl3("Relay : Sending 1bit down. ")
		}
	}

	// if we want to use dummy data down, pad to the correct size
	if p.relayState.UseDummyDataDown && len(downstreamCellContent) < p.relayState.DownstreamCellSize {
		data := make([]byte, p.relayState.DownstreamCellSize)
		copy(data[0:], downstreamCellContent)
		downstreamCellContent = data
	}

	// TODO : if something went wrong before, this flag should be used to warn the clients that the config has changed
	p.relayState.locks.round.Lock()

	flagResync := false
	log.Lvl3("Relay is gonna broadcast messages for round " + strconv.Itoa(int(p.relayState.nextDownStreamRoundToSend)) + ".")
	toSend := &REL_CLI_DOWNSTREAM_DATA{p.relayState.nextDownStreamRoundToSend, downstreamCellContent, flagResync}
	p.relayState.currentDCNetRound.dataAlreadySent = *toSend

	if !p.relayState.UseUDP {
		// broadcast to all clients
		for i := 0; i < p.relayState.nClients; i++ {
			//send to the i-th client
			err := p.messageSender.SendToClient(i, toSend)
			if err != nil {
				e := "Could not send REL_CLI_DOWNSTREAM_DATA to client " + strconv.Itoa(i) + " for round " + strconv.Itoa(int(p.relayState.nextDownStreamRoundToSend)) + ", error is " + err.Error()
				log.Error(e)
				p.relayState.locks.round.Unlock()

				arr := make([]int, 1)
				arr[0] = i
				p.relayState.timeoutHandler(arr, make([]int, 0))

				return errors.New(e)
			}
			log.Lvl3("Relay : sent REL_CLI_DOWNSTREAM_DATA to client " + strconv.Itoa(i) + " for round " + strconv.Itoa(int(p.relayState.nextDownStreamRoundToSend)))
		}

		p.relayState.statistics.AddDownstreamCell(int64(len(downstreamCellContent)))
	} else {
		toSend2 := &REL_CLI_DOWNSTREAM_DATA_UDP{*toSend, make([]byte, 0)}
		p.messageSender.BroadcastToAllClients(toSend2)

		p.relayState.statistics.AddDownstreamUDPCell(int64(len(downstreamCellContent)), p.relayState.nClients)
	}
	log.Lvl2("Relay is done broadcasting messages for round " + strconv.Itoa(int(p.relayState.nextDownStreamRoundToSend)) + ".")

	p.relayState.nextDownStreamRoundToSend++
	p.relayState.numberOfNonAckedDownstreamPackets++

	p.relayState.locks.round.Unlock()

	return nil
}

func (p *PriFiLibInstance) roundFinished() error {

	p.relayState.locks.coder.Lock()
	p.relayState.locks.round.Lock()

	p.relayState.numberOfNonAckedDownstreamPackets--

	timeSpent := time.Since(p.relayState.currentDCNetRound.startTime)
	log.Lvl4("Relay finished round "+strconv.Itoa(int(p.relayState.currentDCNetRound.currentRound))+" (after", timeSpent, ").")
	p.relayState.statistics.Report()

	//prepare for the next round
	nextRound := p.relayState.currentDCNetRound.currentRound + 1
	nilMessage := &REL_CLI_DOWNSTREAM_DATA{-1, make([]byte, 0), false}
	p.relayState.currentDCNetRound = DCNetRound{nextRound, 0, 0, make(map[int]bool), make(map[int]bool), *nilMessage, time.Now()}
	p.relayState.CellCoder.DecodeStart(p.relayState.UpstreamCellSize, p.relayState.MessageHistory) //this empties the buffer, making them ready for a new round

	//we just sent the data down, initiating a round. Let's prevent being blocked by a dead client
	go p.checkIfRoundHasEndedAfterTimeOut_Phase1(p.relayState.currentDCNetRound.currentRound)

	// Test if we are doing an experiment, and if we need to stop at some point.
	if nextRound == int32(p.relayState.ExperimentRoundLimit) {
		log.Lvl1("Relay : Experiment round limit (", nextRound, ") reached")

		p.relayState.locks.expData.Lock()
		// this can be set anywhere, anytime before
		p.relayState.ExperimentResultData = &struct {
			Data1 string
			Data2 []int
		}{
			"This is an experiment",
			[]int{0, -1, 1023},
		}
		p.relayState.ExperimentResultChannel <- p.relayState.ExperimentResultData

		p.relayState.locks.expData.Unlock()

		// shut down everybody
		msg := ALL_ALL_SHUTDOWN{}
		p.Received_ALL_REL_SHUTDOWN(msg)
	}

	p.relayState.locks.trusteeBuffer.Lock()

	// if we have buffered messages for next round, use them now, so whenever we receive a client message, the trustee's message are counted correctly
	if _, ok := p.relayState.bufferedTrusteeCiphers[nextRound]; ok {

		threshhold := (TRUSTEE_WINDOW_LOWER_LIMIT + 1) + TRUSTEE_RESUME_SENDING_RATIO*(TRUSTEE_WINDOW_SIZE-(TRUSTEE_WINDOW_LOWER_LIMIT+1))

		for trusteeID, data := range p.relayState.bufferedTrusteeCiphers[nextRound].Data {
			// start decoding using this data
			log.Lvl3("Relay : using pre-cached DC-net cipher from trustee " + strconv.Itoa(trusteeID) + " for round " + strconv.Itoa(int(nextRound)))
			p.relayState.CellCoder.DecodeTrustee(data)
			p.relayState.currentDCNetRound.trusteeCipherCount++

			// Here is the control to regulate the trustees ciphers in case they should continue sending
			p.relayState.locks.cipherTracker.Lock()
			p.relayState.trusteeCipherTracker[trusteeID]--
			currentCapacity := TRUSTEE_WINDOW_SIZE - p.relayState.trusteeCipherTracker[trusteeID]
			p.relayState.locks.cipherTracker.Unlock()

			if currentCapacity >= int(threshhold) { // if the previous capacity was at the lower limit allowed
				toSend := &REL_TRU_TELL_RATE_CHANGE{currentCapacity}
				err := p.messageSender.SendToTrustee(trusteeID, toSend) // send the trustee informing them of the current capacity that has free'd up
				if err != nil {
					e := "Could not send REL_TRU_TELL_RATE_CHANGE to " + strconv.Itoa(trusteeID) + "-th trustee for round " + strconv.Itoa(int(p.relayState.currentDCNetRound.currentRound)) + ", error is " + err.Error()
					log.Error(e)
					p.relayState.locks.trusteeBuffer.Unlock()
					p.relayState.locks.round.Unlock()
					p.relayState.locks.coder.Unlock()
					return errors.New(e)
				}
				log.Lvl3("Relay : sent REL_TRU_TELL_RATE_CHANGE to " + strconv.Itoa(trusteeID) + "-th trustee for round " + strconv.Itoa(int(p.relayState.currentDCNetRound.currentRound)))
			}
		}

		delete(p.relayState.bufferedTrusteeCiphers, nextRound)
	}

	p.relayState.locks.trusteeBuffer.Unlock()

	p.relayState.locks.clientBuffer.Lock()

	if _, ok := p.relayState.bufferedClientCiphers[nextRound]; ok {
		for clientID, data := range p.relayState.bufferedClientCiphers[nextRound].Data {
			// start decoding using this data
			log.Lvl3("Relay : using pre-cached DC-net cipher from client " + strconv.Itoa(clientID) + " for round " + strconv.Itoa(int(nextRound)))
			p.relayState.CellCoder.DecodeClient(data)
			p.relayState.currentDCNetRound.clientCipherCount++
		}

		delete(p.relayState.bufferedClientCiphers, nextRound)
	}

	p.relayState.locks.clientBuffer.Unlock()
	p.relayState.locks.round.Unlock()
	p.relayState.locks.coder.Unlock()

	return nil
}

/*
Received_TRU_REL_TELL_PK handles TRU_REL_TELL_PK messages. Those are sent by the trustees message when we connect them.
We do nothing, until we have received one per trustee; Then, we pack them in one message, and broadcast it to the clients.
*/
func (p *PriFiLibInstance) Received_TRU_REL_TELL_PK(msg TRU_REL_TELL_PK) error {

	p.relayState.locks.state.Lock()
	// this can only happens in the state RELAY_STATE_COLLECTING_TRUSTEES_PKS
	if p.relayState.currentState != RELAY_STATE_COLLECTING_TRUSTEES_PKS {
		e := "Relay : Received a TRU_REL_TELL_PK, but not in state RELAY_STATE_COLLECTING_TRUSTEES_PKS, in state " + relayStateStr(p.relayState.currentState)
		log.Error(e)
		p.relayState.locks.state.Unlock()
		return errors.New(e)
	}
	p.relayState.locks.state.Unlock()
	log.Lvl3("Relay : received TRU_REL_TELL_PK")

	p.relayState.locks.nTrusteePK.Lock()
	p.relayState.locks.trustees.Lock()

	p.relayState.trustees[msg.TrusteeID] = NodeRepresentation{msg.TrusteeID, true, msg.Pk, msg.Pk}
	p.relayState.nTrusteesPkCollected++

	log.Lvl2("Relay : received TRU_REL_TELL_PK (" + strconv.Itoa(p.relayState.nTrusteesPkCollected) + "/" + strconv.Itoa(p.relayState.nTrustees) + ")")

	// if we have them all...
	if p.relayState.nTrusteesPkCollected == p.relayState.nTrustees {
		p.relayState.locks.nTrusteePK.Unlock()

		// prepare the message for the clients
		trusteesPk := make([]abstract.Point, p.relayState.nTrustees)
		for i := 0; i < p.relayState.nTrustees; i++ {
			trusteesPk[i] = p.relayState.trustees[i].PublicKey
		}
		p.relayState.locks.trustees.Unlock()

		// Send the pack to the clients
		toSend := &REL_CLI_TELL_TRUSTEES_PK{trusteesPk}
		for i := 0; i < p.relayState.nClients; i++ {
			err := p.messageSender.SendToClient(i, toSend)
			if err != nil {
				e := "Could not send REL_CLI_TELL_TRUSTEES_PK (" + strconv.Itoa(i) + "-th client), error is " + err.Error()
				log.Error(e)
				return errors.New(e)
			}
			log.Lvl3("Relay : sent REL_CLI_TELL_TRUSTEES_PK (" + strconv.Itoa(i) + "-th client)")
		}

		p.relayState.locks.state.Lock()
		p.relayState.currentState = RELAY_STATE_COLLECTING_CLIENT_PKS
		p.relayState.locks.state.Unlock()
	} else {
		p.relayState.locks.trustees.Unlock()
		p.relayState.locks.nTrusteePK.Unlock()
	}

	return nil
}

/*
Received_CLI_REL_TELL_PK_AND_EPH_PK handles CLI_REL_TELL_PK_AND_EPH_PK messages.
Those are sent by the client to tell their identity.
We do nothing until we have collected one per client; then, we pack them in one message
and send them to the first trustee for it to Neff-Shuffle them.
*/
func (p *PriFiLibInstance) Received_CLI_REL_TELL_PK_AND_EPH_PK(msg CLI_REL_TELL_PK_AND_EPH_PK) error {

	p.relayState.locks.state.Lock()
	// this can only happens in the state RELAY_STATE_COLLECTING_CLIENT_PKS
	if p.relayState.currentState != RELAY_STATE_COLLECTING_CLIENT_PKS {
		e := "Relay : Received a CLI_REL_TELL_PK_AND_EPH_PK, but not in state RELAY_STATE_COLLECTING_CLIENT_PKS, in state " + relayStateStr(p.relayState.currentState)
		log.Error(e)
		p.relayState.locks.state.Unlock()
		return errors.New(e)
	}
	p.relayState.locks.state.Unlock()
	log.Lvl3("Relay : received CLI_REL_TELL_PK_AND_EPH_PK")

	p.relayState.locks.clients.Lock()

	// collect this client information
	nextID := len(p.relayState.clients)
	newClient := NodeRepresentation{nextID, true, msg.Pk, msg.EphPk}

	p.relayState.clients = append(p.relayState.clients, newClient)

	// TODO : sanity check that we don't have twice the same client

	log.Lvl3("Relay : Received a CLI_REL_TELL_PK_AND_EPH_PK, registered client ID" + strconv.Itoa(nextID))

	log.Lvl2("Relay : received CLI_REL_TELL_PK_AND_EPH_PK (" + strconv.Itoa(len(p.relayState.clients)) + "/" + strconv.Itoa(p.relayState.nClients) + ")")

	// if we have collected all clients, continue
	if len(p.relayState.clients) == p.relayState.nClients {

		// prepare the arrays; pack the public keys and ephemeral public keys
		pks := make([]abstract.Point, p.relayState.nClients)
		ephPks := make([]abstract.Point, p.relayState.nClients)
		for i := 0; i < p.relayState.nClients; i++ {
			pks[i] = p.relayState.clients[i].PublicKey
			ephPks[i] = p.relayState.clients[i].EphemeralPublicKey
		}

		G := config.CryptoSuite.Scalar().One()

		// prepare the empty shuffle
		emptyG_s := make([]abstract.Scalar, p.relayState.nTrustees)
		emptyEphPks_s := make([][]abstract.Point, p.relayState.nTrustees)
		emptyProof_s := make([][]byte, p.relayState.nTrustees)
		emptySignature_s := make([][]byte, p.relayState.nTrustees)

		p.relayState.locks.shuffle.Lock()
		p.relayState.currentShuffleTranscript = NeffShuffleState{pks, emptyG_s, emptyEphPks_s, emptyProof_s, 0, emptySignature_s, 0}
		p.relayState.locks.shuffle.Unlock()

		// send to the 1st trustee
		toSend := &REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE{pks, ephPks, G}
		time.Sleep(time.Second * 2)
		err := p.messageSender.SendToTrustee(0, toSend)
		if err != nil {
			e := "Could not send REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE (0-th iteration), error is " + err.Error()
			log.Error(e)
			p.relayState.locks.clients.Unlock()
			return errors.New(e)
		}
		log.Lvl3("Relay : sent REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE (0-th iteration)")

		// changing state
		p.relayState.locks.state.Lock()
		p.relayState.currentState = RELAY_STATE_COLLECTING_SHUFFLES
		p.relayState.locks.state.Unlock()
	}

	p.relayState.locks.clients.Unlock()

	return nil
}

/*
Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS handles TRU_REL_TELL_NEW_BASE_AND_EPH_PKS messages.
Those are sent by the trustees once they finished a Neff-Shuffle.
In that case, we forward the result to the next trustee.
We do nothing until the last trustee sends us this message.
When this happens, we pack a transcript, and broadcast it to all the trustees who will sign it.
*/
func (p *PriFiLibInstance) Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS(msg TRU_REL_TELL_NEW_BASE_AND_EPH_PKS) error {

	p.relayState.locks.state.Lock()
	// this can only happens in the state RELAY_STATE_COLLECTING_SHUFFLES
	if p.relayState.currentState != RELAY_STATE_COLLECTING_SHUFFLES {
		e := "Relay : Received a TRU_REL_TELL_NEW_BASE_AND_EPH_PKS, but not in state RELAY_STATE_COLLECTING_SHUFFLES, in state " + relayStateStr(p.relayState.currentState)
		log.Error(e)
		p.relayState.locks.state.Unlock()
		return errors.New(e)
	}
	p.relayState.locks.state.Unlock()
	log.Lvl3("Relay : received TRU_REL_TELL_NEW_BASE_AND_EPH_PKS")

	p.relayState.locks.shuffle.Lock()

	// store this shuffle's result in our transcript
	j := p.relayState.currentShuffleTranscript.nextFreeId_Proofs
	p.relayState.currentShuffleTranscript.Gs[j] = msg.NewBase
	p.relayState.currentShuffleTranscript.ephPubKeys[j] = msg.NewEphPks
	p.relayState.currentShuffleTranscript.proofs[j] = msg.Proof

	p.relayState.currentShuffleTranscript.nextFreeId_Proofs = j + 1

	log.Lvl2("Relay : received TRU_REL_TELL_NEW_BASE_AND_EPH_PKS (" + strconv.Itoa(p.relayState.currentShuffleTranscript.nextFreeId_Proofs) + "/" + strconv.Itoa(p.relayState.nTrustees) + ")")

	// if we're still waiting on some trustees, send them the new shuffle
	if p.relayState.currentShuffleTranscript.nextFreeId_Proofs != p.relayState.nTrustees {

		pks := p.relayState.currentShuffleTranscript.ClientPublicKeys
		ephPks := msg.NewEphPks
		G := msg.NewBase

		// send to the i-th trustee
		toSend := &REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE{pks, ephPks, G}
		err := p.messageSender.SendToTrustee(j+1, toSend)
		if err != nil {
			e := "Could not send REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE (" + strconv.Itoa(j+1) + "-th iteration), error is " + err.Error()
			log.Error(e)
			p.relayState.locks.shuffle.Unlock()
			return errors.New(e)
		}
		log.Lvl3("Relay : sent REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE (" + strconv.Itoa(j+1) + "-th iteration)")

		p.relayState.locks.shuffle.Unlock()

	} else {
		// if we have all the shuffles

		// pack transcript
		G_s := p.relayState.currentShuffleTranscript.Gs
		ephPublicKeys_s := p.relayState.currentShuffleTranscript.ephPubKeys
		proof_s := p.relayState.currentShuffleTranscript.proofs

		p.relayState.locks.shuffle.Unlock()

		p.relayState.locks.trusteeBuffer.Lock()
		p.relayState.locks.clientBuffer.Lock()

		// when receiving the next message (and after processing it), trustees will start sending data. Prepare to buffer it
		p.relayState.bufferedTrusteeCiphers = make(map[int32]BufferedCipher)
		p.relayState.bufferedClientCiphers = make(map[int32]BufferedCipher)

		p.relayState.locks.trusteeBuffer.Unlock()
		p.relayState.locks.clientBuffer.Unlock()

		// broadcast to all trustees
		for j := 0; j < p.relayState.nTrustees; j++ {
			// send to the j-th trustee
			toSend := &REL_TRU_TELL_TRANSCRIPT{G_s, ephPublicKeys_s, proof_s}
			err := p.messageSender.SendToTrustee(j, toSend) // TODO : this should be the trustee X !
			if err != nil {
				e := "Could not send REL_TRU_TELL_TRANSCRIPT to " + strconv.Itoa(j+1) + "-th trustee, error is " + err.Error()
				log.Error(e)
				return errors.New(e)
			}
			log.Lvl3("Relay : sent REL_TRU_TELL_TRANSCRIPT to " + strconv.Itoa(j+1) + "-th trustee")
		}

		p.relayState.locks.round.Lock()
		p.relayState.locks.coder.Lock()
		p.relayState.locks.state.Lock()

		// prepare to collect the ciphers
		p.relayState.currentDCNetRound = DCNetRound{0, 0, 0, make(map[int]bool), make(map[int]bool), REL_CLI_DOWNSTREAM_DATA{}, time.Now()}
		p.relayState.CellCoder.DecodeStart(p.relayState.UpstreamCellSize, p.relayState.MessageHistory)

		// changing state
		p.relayState.currentState = RELAY_STATE_COLLECTING_SHUFFLE_SIGNATURES

		p.relayState.locks.state.Unlock()
		p.relayState.locks.coder.Unlock()
		p.relayState.locks.round.Unlock()

	}

	return nil
}

/*
Received_TRU_REL_SHUFFLE_SIG handles TRU_REL_SHUFFLE_SIG messages.
Those contain the signature from the NeffShuffleS-transcript from one trustee.
We do nothing until we have all signatures; when we do, we pack those
in one message with the result of the Neff-Shuffle and send them to the clients.
When this is done, we are finally ready to communicate. We wait for the client's messages.
*/
func (p *PriFiLibInstance) Received_TRU_REL_SHUFFLE_SIG(msg TRU_REL_SHUFFLE_SIG) error {

	p.relayState.locks.shuffle.Lock()
	p.relayState.locks.state.Lock()

	defer p.relayState.locks.state.Unlock()
	defer p.relayState.locks.shuffle.Unlock()

	// this can only happens in the state RELAY_STATE_COLLECTING_SHUFFLE_SIGNATURES
	if p.relayState.currentState != RELAY_STATE_COLLECTING_SHUFFLE_SIGNATURES {
		e := "Relay : Received a TRU_REL_SHUFFLE_SIG, but not in state RELAY_STATE_COLLECTING_SHUFFLE_SIGNATURES, in state " + relayStateStr(p.relayState.currentState)
		log.Error(e)
		return errors.New(e)
	}
	log.Lvl3("Relay : received TRU_REL_SHUFFLE_SIG")

	// sanity check
	if msg.TrusteeID < 0 || msg.TrusteeID > len(p.relayState.currentShuffleTranscript.signatures) {
		e := "Relay : One of the following check failed : msg.TrusteeId >= 0 && msg.TrusteeId < len(p.relayState.currentShuffleTranscript.signatures_s) ; msg.TrusteeId = " + strconv.Itoa(msg.TrusteeID) + ";"
		log.Error(e)
		return errors.New(e)
	}

	// store this shuffle's signature in our transcript
	p.relayState.currentShuffleTranscript.signatures[msg.TrusteeID] = msg.Sig
	p.relayState.currentShuffleTranscript.signatureCount++

	log.Lvl2("Relay : received TRU_REL_SHUFFLE_SIG (" + strconv.Itoa(p.relayState.currentShuffleTranscript.signatureCount) + "/" + strconv.Itoa(p.relayState.nTrustees) + ")")

	// if we have all the signatures
	if p.relayState.currentShuffleTranscript.signatureCount == p.relayState.nTrustees {

		// We could verify here before broadcasting to the clients, for performance (but this does not add security)

		// prepare the message for the clients
		lastPermutationIndex := p.relayState.nTrustees - 1
		G := p.relayState.currentShuffleTranscript.Gs[lastPermutationIndex]
		ephPks := p.relayState.currentShuffleTranscript.ephPubKeys[lastPermutationIndex]
		signatures := p.relayState.currentShuffleTranscript.signatures

		// changing state
		log.Lvl2("Relay : ready to communicate.")
		p.relayState.currentState = RELAY_STATE_COMMUNICATING

		// broadcast to all clients
		for i := 0; i < p.relayState.nClients; i++ {
			// send to the i-th client
			toSend := &REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG{G, ephPks, signatures} //TODO: remove from loop (it's loop independent)
			err := p.messageSender.SendToClient(i, toSend)
			if err != nil {
				e := "Could not send REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG to " + strconv.Itoa(i+1) + "-th client, error is " + err.Error()
				log.Error(e)
				return errors.New(e)
			}
			log.Lvl3("Relay : sent REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG to " + strconv.Itoa(i+1) + "-th client")
		}

		//client will answer will CLI_REL_UPSTREAM_DATA. There is no data down on round 0. We set the following variable to 1 since the reception of CLI_REL_UPSTREAM_DATA decrements it.
		p.relayState.numberOfNonAckedDownstreamPackets = 1
	}

	return nil
}

/*
This first timeout happens after a short delay. Clients will not be considered disconnected yet,
but if we use UDP, it can mean that a client missed a broadcast, and we re-sent the message.
If the round was *not* done, we do another timeout (Phase 2), and then, clients/trustees will be considered
online if they didn't answer by that time.
*/
func (p *PriFiLibInstance) checkIfRoundHasEndedAfterTimeOut_Phase1(roundID int32) {

	time.Sleep(5 * time.Second)

	p.relayState.locks.round.Lock()

	if p.relayState.currentDCNetRound.currentRound != roundID {
		//everything went well, it's great !
		p.relayState.locks.round.Unlock()
		return
	}

	p.relayState.locks.state.Lock()

	if p.relayState.currentState == RELAY_STATE_SHUTDOWN {
		//nothing to ensure in that case
		p.relayState.locks.round.Unlock()
		p.relayState.locks.state.Unlock()
		return
	}

	p.relayState.locks.state.Unlock()

	allGood := true

	if p.relayState.currentDCNetRound.currentRound == roundID {
		log.Error("waitAndCheckIfClientsSentData : We seem to be stuck in round", roundID, ". Phase 1 timeout.")

		//check for the trustees
		for j := 0; j < p.relayState.nTrustees; j++ {

			p.relayState.locks.trustees.Lock()
			trusteeID := p.relayState.trustees[j].ID
			p.relayState.locks.trustees.Unlock()

			//if we miss some message...
			if !p.relayState.currentDCNetRound.trusteeCipherAck[trusteeID] {
				allGood = false
			}
		}

		//check for the clients
		for i := 0; i < p.relayState.nClients; i++ {

			p.relayState.locks.clients.Lock()
			clientID := p.relayState.clients[i].ID
			p.relayState.locks.clients.Unlock()

			//if we miss some message...
			if !p.relayState.currentDCNetRound.clientCipherAck[clientID] {
				allGood = false

				//If we're using UDP, client might have missed the broadcast, re-sending
				if p.relayState.UseUDP {
					log.Error("Relay : Client " + strconv.Itoa(clientID) + " didn't sent us is cipher for round " + strconv.Itoa(int(roundID)) + ". Phase 1 timeout. Re-sending...")
					err := p.messageSender.SendToClient(i, &p.relayState.currentDCNetRound.dataAlreadySent)
					if err != nil {
						e := "Could not send REL_CLI_DOWNSTREAM_DATA to " + strconv.Itoa(i+1) + "-th client for round " + strconv.Itoa(int(p.relayState.currentDCNetRound.currentRound)) + ", error is " + err.Error()
						log.Error(e)
					} else {
						log.Lvl3("Relay : sent REL_CLI_DOWNSTREAM_DATA to " + strconv.Itoa(i+1) + "-th client for round " + strconv.Itoa(int(p.relayState.currentDCNetRound.currentRound)))
					}
				}
			}
		}
	}
	p.relayState.locks.round.Unlock()

	if !allGood {
		//if we're not done (we miss data), wait another timeout, after which clients/trustees will be considered offline
		go p.checkIfRoundHasEndedAfterTimeOut_Phase2(roundID)
	}

	//this shouldn't happen frequently (it means that the timeout 1 was fired, but the round finished almost at the same time)
}

/*
This second timeout happens after a longer delay. Clients and trustees will be considered offline if they haven't send data yet
*/
func (p *PriFiLibInstance) checkIfRoundHasEndedAfterTimeOut_Phase2(roundID int32) {

	time.Sleep(5 * time.Second)

	p.relayState.locks.round.Lock()
	if p.relayState.currentDCNetRound.currentRound != roundID {
		//everything went well, it's great !
		p.relayState.locks.round.Unlock()
		return
	}

	p.relayState.locks.state.Lock()
	if p.relayState.currentState == RELAY_STATE_SHUTDOWN {
		//nothing to ensure in that case
		p.relayState.locks.round.Unlock()
		p.relayState.locks.state.Unlock()
		return
	}
	p.relayState.locks.state.Unlock()

	var clientsIds []int
	var trusteesIds []int
	var stuck = false

	if p.relayState.currentDCNetRound.currentRound == roundID {
		log.Error("waitAndCheckIfClientsSentData : We seem to be stuck in round", roundID, ". Phase 2 timeout.")
		stuck = true

		//check for the trustees
		for j := 0; j < p.relayState.nTrustees; j++ {

			p.relayState.locks.trustees.Lock()
			trusteeID := p.relayState.trustees[j].ID
			p.relayState.locks.trustees.Unlock()

			if !p.relayState.currentDCNetRound.trusteeCipherAck[trusteeID] {
				e := "Relay : Trustee " + strconv.Itoa(trusteeID) + " didn't sent us is cipher for round " + strconv.Itoa(int(roundID)) + ". Phase 2 timeout. This is unacceptable !"
				log.Error(e)

				trusteesIds = append(trusteesIds, trusteeID)
			}
		}

		//check for the clients
		for i := 0; i < p.relayState.nClients; i++ {

			p.relayState.locks.clients.Lock()
			clientID := p.relayState.clients[i].ID
			p.relayState.locks.clients.Unlock()

			if !p.relayState.currentDCNetRound.clientCipherAck[clientID] {
				e := "Relay : Client " + strconv.Itoa(clientID) + " didn't sent us is cipher for round " + strconv.Itoa(int(roundID)) + ". Phase 2 timeout. This is unacceptable !"
				log.Error(e)

				clientsIds = append(clientsIds, clientID)
			}
		}
	}
	p.relayState.locks.round.Unlock()

	if stuck {
		p.relayState.timeoutHandler(clientsIds, trusteesIds)
	}
}

// SetTimeoutHandler sets the timeout handler, which is called with the dead/unresponsive client/trustee ID when a timeout occurs in a round
func (p *PriFiLibInstance) SetTimeoutHandler(handler func([]int, []int)) {
	p.relayState.timeoutHandler = handler
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
