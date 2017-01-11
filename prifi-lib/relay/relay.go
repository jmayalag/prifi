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
	"encoding/binary"
	"errors"
	"strconv"
	"time"

	"github.com/dedis/cothority/log"
	"github.com/dedis/crypto/abstract"
	"github.com/lbarman/prifi/prifi-lib/config"
	"github.com/lbarman/prifi/prifi-lib/dcnet"
	prifilog "github.com/lbarman/prifi/prifi-lib/log"
	"github.com/lbarman/prifi/prifi-lib/net"
	"github.com/lbarman/prifi/prifi-lib/scheduler"

	"github.com/lbarman/prifi/prifi-lib/crypto"
	socks "github.com/lbarman/prifi/prifi-socks"
)

// PriFiLibInstance contains the mutable state of a PriFi entity.
type PriFiLibRelayInstance struct {
	messageSender *net.MessageSenderWrapper
	relayState   RelayState
}

// NewPriFiRelay creates a new PriFi relay entity state.
// Note: the returned state is not sufficient for the PrFi protocol
// to start; this entity will expect a ALL_ALL_PARAMETERS message as
// first received message to complete it's state.
func NewPriFiRelay(msgSender *net.MessageSenderWrapper) *PriFiLibRelayInstance {
	prifi := PriFiLibRelayInstance{
		messageSender: msgSender,
	}

	log.Lvl1("Relay (but not its state) has been initialized by function call. ")
	return &prifi
}

// NewPriFiRelayWithState creates a new PriFi relay entity state.
func NewPriFiRelayWithState(msgSender *net.MessageSenderWrapper, state *RelayState) *PriFiLibRelayInstance {
	prifi := PriFiLibRelayInstance{
		messageSender: msgSender,
		relayState:    *state,
	}

	log.Lvl1("Relay (and its state) has been initialized by function call. ")
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

/*
NewRelayState initializes the state of this relay.
It must be called before anything else.
*/
func NewRelayState(nTrustees int, nClients int, upstreamCellSize int, downstreamCellSize int, windowSize int, useDummyDataDown bool, experimentRoundLimit int, experimentResultChan chan interface{}, useUDP bool, dataOutputEnabled bool, dataForClients chan []byte, dataFromDCNet chan []byte, timeoutHandler func([]int, []int)) *RelayState {
	params := new(RelayState)
	params.Name = "Relay"
	params.CellCoder = config.Factory()
	params.clients = make([]NodeRepresentation, 0)
	params.DataForClients = dataForClients
	params.PriorityDataForClients = make(chan []byte, 10) // This is used for relay's control message (like latency-tests)
	params.DataFromDCNet = dataFromDCNet
	params.DataOutputEnabled = dataOutputEnabled
	params.DownstreamCellSize = downstreamCellSize
	params.nClients = nClients
	params.ExperimentResultChannel = experimentResultChan
	params.nTrustees = nTrustees
	params.nTrusteesPkCollected = 0
	params.ExperimentRoundLimit = experimentRoundLimit
	params.trustees = make([]NodeRepresentation, nTrustees)
	params.UpstreamCellSize = upstreamCellSize
	params.UseDummyDataDown = useDummyDataDown
	params.UseUDP = useUDP
	params.WindowSize = windowSize
	params.nextDownStreamRoundToSend = int32(1) //since first round is half-round
	params.numberOfNonAckedDownstreamPackets = 0
	params.timeoutHandler = timeoutHandler

	//init the statistics
	params.statistics = prifilog.NewBitRateStatistics()

	//init the neff shuffle
	neffShuffle := new(scheduler.NeffShuffle)
	neffShuffle.Init()
	params.neffShuffle = neffShuffle.RelayView

	//init the buffer manager
	params.bufferManager = new(BufferManager)
	params.bufferManager.Init(nClients, nTrustees)

	// Generate pk
	params.PublicKey, params.privateKey = crypto.NewKeyPair()

	// Sets the new state
	params.currentState = RELAY_STATE_COLLECTING_TRUSTEES_PKS

	return params
}

/*
Received_ALL_REL_SHUTDOWN handles ALL_REL_SHUTDOWN messages.
When we receive this message, we should warn other protocol participants and clean resources.
*/
func (p *PriFiLibRelayInstance) Received_ALL_REL_SHUTDOWN(msg net.ALL_ALL_SHUTDOWN) error {
	log.Lvl1("Relay : Received a SHUTDOWN message. ")

	p.relayState.currentState = RELAY_STATE_SHUTDOWN

	msg2 := &net.ALL_ALL_SHUTDOWN{}

	var err error

	// Send this shutdown to all trustees
	for j := 0; j < p.relayState.nTrustees; j++ {
		p.messageSender.SendToTrusteeWithLog(j, msg2, "")
	}

	// Send this shutdown to all clients
	for j := 0; j < p.relayState.nClients; j++ {
		p.messageSender.SendToClientWithLog(j, msg2, "")
	}

	// TODO : stop all go-routines we created

	return err
}

/*
Received_ALL_REL_PARAMETERS handles ALL_REL_PARAMETERS.
It initializes the relay with the parameters contained in the message.
*/
func (p *PriFiLibRelayInstance) Received_ALL_ALL_PARAMETERS(msg net.ALL_ALL_PARAMETERS) error {

	// This can only happens in the state RELAY_STATE_BEFORE_INIT
	if p.relayState.currentState != RELAY_STATE_BEFORE_INIT && !msg.ForceParams {
		log.Lvl1("Relay : Received a ALL_ALL_PARAMETERS, but not in state RELAY_STATE_BEFORE_INIT, ignoring. ")
		return nil
	} else if p.relayState.currentState != RELAY_STATE_BEFORE_INIT && msg.ForceParams {
		log.Lvl1("Relay : Received a ALL_ALL_PARAMETERS && ForceParams = true, processing. ")
	} else {
		log.Lvl3("Relay : received ALL_ALL_PARAMETERS")
	}

	startNow := net.ValueOrElse(msg.Params, "StartNow", false).(bool)
	nTrustees := net.ValueOrElse(msg.Params, "NTrustees", p.relayState.nTrustees).(int)
	nClients := net.ValueOrElse(msg.Params, "nClients", p.relayState.nClients).(int)
	upCellSize := net.ValueOrElse(msg.Params, "UpstreamCellSize", p.relayState.UpstreamCellSize).(int)
	downCellSize := net.ValueOrElse(msg.Params, "DownstreamCellSize", p.relayState.DownstreamCellSize).(int)
	windowSize := net.ValueOrElse(msg.Params, "WindowSize", p.relayState.WindowSize).(int)
	useDummyDown := net.ValueOrElse(msg.Params, "UseDummyDataDown", p.relayState.UseDummyDataDown).(bool)
	reportingLimit := net.ValueOrElse(msg.Params, "ExperimentRoundLimit", p.relayState.ExperimentRoundLimit).(int)
	useUDP := net.ValueOrElse(msg.Params, "UseUDP", p.relayState.UseUDP).(bool)
	dataOutputEnabled := net.ValueOrElse(msg.Params, "DataOutputEnabled", p.relayState.DataOutputEnabled).(bool)

	//this is never set in the message
	dataForClients := make(chan []byte)
	dataFromDCNet := make(chan []byte)
	experimentResultChan := p.relayState.ExperimentResultChannel
	timeoutHandler := p.relayState.timeoutHandler


	p.relayState = *NewRelayState(nTrustees, nClients, upCellSize, downCellSize, windowSize,
		useDummyDown, reportingLimit, experimentResultChan, useUDP, dataOutputEnabled,
		dataForClients, dataFromDCNet, timeoutHandler)

	//this should be in NewRelayState, but we need p
	if !p.relayState.bufferManager.DoSendStopResumeMessages {
		//Add rate-limiting component to buffer manager
		stopFn := func(trusteeID int) {
			toSend := &net.REL_TRU_TELL_RATE_CHANGE{WindowCapacity: 0}
			p.messageSender.SendToTrusteeWithLog(trusteeID, toSend, "(trustee "+strconv.Itoa(trusteeID)+")")
		}
		resumeFn := func(trusteeID int) {
			toSend := &net.REL_TRU_TELL_RATE_CHANGE{WindowCapacity: 1}
			p.messageSender.SendToTrusteeWithLog(trusteeID, toSend, "(trustee "+strconv.Itoa(trusteeID)+")")
		}
		p.relayState.bufferManager.AddRateLimiter(TRUSTEE_CACHE_LOWBOUND, TRUSTEE_CACHE_HIGHBOUND, stopFn, resumeFn)
	}

	log.Lvlf3("%+v\n", p.relayState)
	log.Lvl1("Relay has been initialized by message. ")

	// Broadcast those parameters to the other nodes, then tell the trustees which ID they are.
	if startNow {
		p.relayState.currentState = RELAY_STATE_COLLECTING_TRUSTEES_PKS
		p.SendParameters()
	}
	log.Lvl1("Relay setup done, and setup sent to the trustees.")

	return nil
}

// ConnectToTrustees connects to the trustees and initializes them with default parameters.
func (p *PriFiLibRelayInstance) SendParameters() error {

	// Craft default parameters
	params := make(map[string]interface{})
	params["NClients"] = p.relayState.nClients
	params["NTrustees"] = p.relayState.nTrustees
	params["StartNow"] = true
	params["UpCellSize"] = p.relayState.UpstreamCellSize
	var msg = &net.ALL_ALL_PARAMETERS{
		Params: params,
		ForceParams:       true,
	}

	log.Lvl1("Sending ALL_TRU_PARAMETERS 2")
	log.Lvlf1("%+v\n", msg)

	// Send those parameters to all trustees
	for j := 0; j < p.relayState.nTrustees; j++ {

		// The ID is unique !
		msg.Params["NextFreeTrusteeID"] = j
		p.messageSender.SendToTrusteeWithLog(j, msg, "")
	}

	log.Lvl1("Sending ALL_CLI_PARAMETERS 2")
	log.Lvlf1("%+v\n", msg)

	// Send those parameters to all trustees
	for j := 0; j < p.relayState.nClients; j++ {

		// The ID is unique !
		msg.Params["NextFreeClientID"] = j
		p.messageSender.SendToClientWithLog(j, msg, "")
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
func (p *PriFiLibRelayInstance) Received_CLI_REL_UPSTREAM_DATA(msg net.CLI_REL_UPSTREAM_DATA) error {
	// This can only happens in the state RELAY_STATE_COMMUNICATING
	if p.relayState.currentState != RELAY_STATE_COMMUNICATING {
		e := "Relay : Received a CLI_REL_UPSTREAM_DATA, but not in state RELAY_STATE_COMMUNICATING, in state " + relayStateStr(p.relayState.currentState)
		log.Error(e)
		// return errors.New(e)
	}
	log.Lvl3("Relay : received CLI_REL_UPSTREAM_DATA from client " + strconv.Itoa(msg.ClientID) + " for round " + strconv.Itoa(int(msg.RoundID)))

	p.relayState.bufferManager.AddClientCipher(msg.RoundID, msg.ClientID, msg.Data)

	if p.relayState.bufferManager.HasAllCiphersForCurrentRound() {

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
	}

	return nil
}

/*
Received_TRU_REL_DC_CIPHER handles TRU_REL_DC_CIPHER messages. Those contain a DC-net cipher from a Trustee.
If it's for this round, we call decode on it, and remember we received it.
If for a future round we need to Buffer it.
*/
func (p *PriFiLibRelayInstance) Received_TRU_REL_DC_CIPHER(msg net.TRU_REL_DC_CIPHER) error {

	// this can only happens in the state RELAY_STATE_COMMUNICATING
	if p.relayState.currentState != RELAY_STATE_COMMUNICATING && p.relayState.currentState != RELAY_STATE_COLLECTING_SHUFFLE_SIGNATURES {
		e := "Relay : Received a TRU_REL_DC_CIPHER, but not in state RELAY_STATE_COMMUNICATING, in state " + relayStateStr(p.relayState.currentState)
		log.Error(e)
		return errors.New(e)
	}
	log.Lvl3("Relay : received TRU_REL_DC_CIPHER for round " + strconv.Itoa(int(msg.RoundID)) + " from trustee " + strconv.Itoa(msg.TrusteeID))

	p.relayState.bufferManager.AddTrusteeCipher(msg.RoundID, msg.TrusteeID, msg.Data)

	if p.relayState.bufferManager.HasAllCiphersForCurrentRound() {

		log.Lvl2("Relay has collected all ciphers for round", p.relayState.currentDCNetRound.currentRound, ", decoding...")
		p.finalizeUpstreamData()

		// send the data down
		for i := p.relayState.numberOfNonAckedDownstreamPackets; i < p.relayState.WindowSize; i++ {
			log.Lvl3("Relay : Gonna send, non-acked packets is", p.relayState.numberOfNonAckedDownstreamPackets, "(window is", p.relayState.WindowSize, ")")
			p.sendDownstreamData()
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
func (p *PriFiLibRelayInstance) finalizeUpstreamData() error {

	// we decode the DC-net cell
	clientSlices, trusteesSlices, err := p.relayState.bufferManager.FinalizeRound()
	if err != nil {
		return err
	}

	//decode all clients and trustees
	for _, s := range clientSlices {
		p.relayState.CellCoder.DecodeClient(s)
	}
	for _, s := range trusteesSlices {
		p.relayState.CellCoder.DecodeTrustee(s)
	}
	upstreamPlaintext := p.relayState.CellCoder.DecodeCell()

	p.relayState.statistics.AddUpstreamCell(int64(len(upstreamPlaintext)))

	// check if we have a latency test message
	if len(upstreamPlaintext) >= 2 {
		pattern := int(binary.BigEndian.Uint16(upstreamPlaintext[0:2]))
		if pattern == 43690 { // 1010101010101010
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
func (p *PriFiLibRelayInstance) sendDownstreamData() error {

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
			log.Error("Relay : We have some real data for the clients. ")

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

	flagResync := false
	log.Lvl3("Relay is gonna broadcast messages for round " + strconv.Itoa(int(p.relayState.nextDownStreamRoundToSend)) + ".")
	toSend := &net.REL_CLI_DOWNSTREAM_DATA{
		RoundID:    p.relayState.nextDownStreamRoundToSend,
		Data:       downstreamCellContent,
		FlagResync: flagResync}
	p.relayState.currentDCNetRound.dataAlreadySent = *toSend

	if !p.relayState.UseUDP {
		// broadcast to all clients
		for i := 0; i < p.relayState.nClients; i++ {
			//send to the i-th client
			p.messageSender.SendToClientWithLog(i, toSend, "(client "+strconv.Itoa(i)+", round "+strconv.Itoa(int(p.relayState.nextDownStreamRoundToSend))+")")
		}

		p.relayState.statistics.AddDownstreamCell(int64(len(downstreamCellContent)))
	} else {
		toSend2 := &net.REL_CLI_DOWNSTREAM_DATA_UDP{REL_CLI_DOWNSTREAM_DATA: *toSend}
		p.messageSender.MessageSender.BroadcastToAllClients(toSend2)

		p.relayState.statistics.AddDownstreamUDPCell(int64(len(downstreamCellContent)), p.relayState.nClients)
	}
	log.Lvl2("Relay is done broadcasting messages for round " + strconv.Itoa(int(p.relayState.nextDownStreamRoundToSend)) + ".")

	p.relayState.nextDownStreamRoundToSend++
	p.relayState.numberOfNonAckedDownstreamPackets++

	return nil
}

func (p *PriFiLibRelayInstance) roundFinished() error {

	p.relayState.numberOfNonAckedDownstreamPackets--

	timeSpent := time.Since(p.relayState.currentDCNetRound.startTime)
	log.Lvl2("Relay finished round "+strconv.Itoa(int(p.relayState.currentDCNetRound.currentRound))+" (after", timeSpent, ").")
	p.relayState.statistics.Report()

	//prepare for the next round
	nextRound := p.relayState.currentDCNetRound.currentRound + 1
	nilMessage := &net.REL_CLI_DOWNSTREAM_DATA{
		RoundID:    -1,
		Data:       make([]byte, 0),
		FlagResync: false}
	p.relayState.currentDCNetRound = DCNetRound{currentRound: nextRound, dataAlreadySent: *nilMessage, startTime: time.Now()}
	p.relayState.CellCoder.DecodeStart(p.relayState.UpstreamCellSize, p.relayState.MessageHistory) //this empties the buffer, making them ready for a new round

	//we just sent the data down, initiating a round. Let's prevent being blocked by a dead client
	go p.checkIfRoundHasEndedAfterTimeOut_Phase1(p.relayState.currentDCNetRound.currentRound)

	// Test if we are doing an experiment, and if we need to stop at some point.
	if nextRound == int32(p.relayState.ExperimentRoundLimit) {
		log.Lvl1("Relay : Experiment round limit (", nextRound, ") reached")

		// this can be set anywhere, anytime before
		p.relayState.ExperimentResultData = &struct {
			Data1 string
			Data2 []int
		}{
			"This is an experiment",
			[]int{0, -1, 1023},
		}
		p.relayState.ExperimentResultChannel <- p.relayState.ExperimentResultData

		// shut down everybody
		msg := net.ALL_ALL_SHUTDOWN{}
		p.Received_ALL_REL_SHUTDOWN(msg)
	}

	return nil
}

/*
Received_TRU_REL_TELL_PK handles TRU_REL_TELL_PK messages. Those are sent by the trustees message when we connect them.
We do nothing, until we have received one per trustee; Then, we pack them in one message, and broadcast it to the clients.
*/
func (p *PriFiLibRelayInstance) Received_TRU_REL_TELL_PK(msg net.TRU_REL_TELL_PK) error {

	// this can only happens in the state RELAY_STATE_COLLECTING_TRUSTEES_PKS
	if p.relayState.currentState != RELAY_STATE_COLLECTING_TRUSTEES_PKS {
		e := "Relay : Received a TRU_REL_TELL_PK, but not in state RELAY_STATE_COLLECTING_TRUSTEES_PKS, in state " + relayStateStr(p.relayState.currentState)
		log.Error(e)
		return errors.New(e)
	}
	log.Lvl3("Relay : received TRU_REL_TELL_PK")

	p.relayState.trustees[msg.TrusteeID] = NodeRepresentation{msg.TrusteeID, true, msg.Pk, msg.Pk}
	p.relayState.nTrusteesPkCollected++

	log.Lvl2("Relay : received TRU_REL_TELL_PK (" + strconv.Itoa(p.relayState.nTrusteesPkCollected) + "/" + strconv.Itoa(p.relayState.nTrustees) + ")")

	// if we have them all...
	if p.relayState.nTrusteesPkCollected == p.relayState.nTrustees {

		// prepare the message for the clients
		trusteesPk := make([]abstract.Point, p.relayState.nTrustees)
		for i := 0; i < p.relayState.nTrustees; i++ {
			trusteesPk[i] = p.relayState.trustees[i].PublicKey
		}

		// Send the pack to the clients
		toSend := &net.REL_CLI_TELL_TRUSTEES_PK{Pks: trusteesPk}
		for i := 0; i < p.relayState.nClients; i++ {
			p.messageSender.SendToClientWithLog(i, toSend, "(client "+strconv.Itoa(i)+")")
		}

		p.relayState.currentState = RELAY_STATE_COLLECTING_CLIENT_PKS
	}
	return nil
}

/*
Received_CLI_REL_TELL_PK_AND_EPH_PK handles CLI_REL_TELL_PK_AND_EPH_PK messages.
Those are sent by the client to tell their identity.
We do nothing until we have collected one per client; then, we pack them in one message
and send them to the first trustee for it to Neff-Shuffle them.
*/
func (p *PriFiLibRelayInstance) Received_CLI_REL_TELL_PK_AND_EPH_PK(msg net.CLI_REL_TELL_PK_AND_EPH_PK) error {

	// this can only happens in the state RELAY_STATE_COLLECTING_CLIENT_PKS
	if p.relayState.currentState != RELAY_STATE_COLLECTING_CLIENT_PKS {
		e := "Relay : Received a CLI_REL_TELL_PK_AND_EPH_PK, but not in state RELAY_STATE_COLLECTING_CLIENT_PKS, in state " + relayStateStr(p.relayState.currentState)
		log.Error(e)
		return errors.New(e)
	}
	log.Lvl3("Relay : received CLI_REL_TELL_PK_AND_EPH_PK")

	// collect this client information
	nextID := len(p.relayState.clients)
	newClient := NodeRepresentation{nextID, true, msg.Pk, msg.EphPk}

	p.relayState.clients = append(p.relayState.clients, newClient)

	// TODO : sanity check that we don't have twice the same client

	log.Lvl3("Relay : Received a CLI_REL_TELL_PK_AND_EPH_PK, registered client ID" + strconv.Itoa(nextID))

	log.Lvl2("Relay : received CLI_REL_TELL_PK_AND_EPH_PK (" + strconv.Itoa(len(p.relayState.clients)) + "/" + strconv.Itoa(p.relayState.nClients) + ")")

	// if we have collected all clients, continue
	if len(p.relayState.clients) == p.relayState.nClients {

		p.relayState.neffShuffle.Init(p.relayState.nTrustees)

		for i := 0; i < p.relayState.nClients; i++ {
			p.relayState.neffShuffle.AddClient(p.relayState.clients[i].EphemeralPublicKey)
		}

		msg, trusteeID, err := p.relayState.neffShuffle.SendToNextTrustee()
		if err != nil {
			e := "Could not do p.relayState.neffShuffle.SendToNextTrustee, error is " + err.Error()
			log.Error(e)
			return errors.New(e)
		}
		toSend := msg.(*net.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE)

		// send to the 1st trustee
		p.messageSender.SendToTrusteeWithLog(trusteeID, toSend, "(0-th iteration)")

		// changing state
		p.relayState.currentState = RELAY_STATE_COLLECTING_SHUFFLES
	}

	return nil
}

/*
Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS handles TRU_REL_TELL_NEW_BASE_AND_EPH_PKS messages.
Those are sent by the trustees once they finished a Neff-Shuffle.
In that case, we forward the result to the next trustee.
We do nothing until the last trustee sends us this message.
When this happens, we pack a transcript, and broadcast it to all the trustees who will sign it.
*/
func (p *PriFiLibRelayInstance) Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS(msg net.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS) error {

	// this can only happens in the state RELAY_STATE_COLLECTING_SHUFFLES
	if p.relayState.currentState != RELAY_STATE_COLLECTING_SHUFFLES {
		e := "Relay : Received a TRU_REL_TELL_NEW_BASE_AND_EPH_PKS, but not in state RELAY_STATE_COLLECTING_SHUFFLES, in state " + relayStateStr(p.relayState.currentState)
		log.Error(e)
		return errors.New(e)
	}
	log.Lvl3("Relay : received TRU_REL_TELL_NEW_BASE_AND_EPH_PKS")

	done, err := p.relayState.neffShuffle.ReceivedShuffleFromTrustee(msg.NewBase, msg.NewEphPks, msg.Proof)
	if err != nil {
		e := "Relay : error in p.relayState.neffShuffle.ReceivedShuffleFromTrustee " + err.Error()
		log.Error(e)
		return errors.New(e)
	}

	// if we're still waiting on some trustees, send them the new shuffle
	if !done {

		msg, trusteeID, err := p.relayState.neffShuffle.SendToNextTrustee()
		if err != nil {
			e := "Could not do p.relayState.neffShuffle.SendToNextTrustee, error is " + err.Error()
			log.Error(e)
			return errors.New(e)
		}
		toSend := msg.(*net.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE)

		// send to the i-th trustee
		p.messageSender.SendToTrusteeWithLog(trusteeID, toSend, "("+strconv.Itoa(trusteeID)+"-th iteration)")

	} else {
		// if we have all the shuffles

		msg, err := p.relayState.neffShuffle.SendTranscript()
		if err != nil {
			e := "Could not do p.relayState.neffShuffle.SendTranscript(), error is " + err.Error()
			log.Error(e)
			return errors.New(e)
		}

		toSend := msg.(*net.REL_TRU_TELL_TRANSCRIPT)

		// broadcast to all trustees
		for j := 0; j < p.relayState.nTrustees; j++ {
			// send to the j-th trustee
			p.messageSender.SendToTrusteeWithLog(j, toSend, "(trustee "+strconv.Itoa(j+1)+")")
		}

		// prepare to collect the ciphers
		p.relayState.currentDCNetRound = DCNetRound{currentRound: 0, dataAlreadySent: net.REL_CLI_DOWNSTREAM_DATA{}, startTime: time.Now()}
		p.relayState.CellCoder.DecodeStart(p.relayState.UpstreamCellSize, p.relayState.MessageHistory)

		// changing state
		p.relayState.currentState = RELAY_STATE_COLLECTING_SHUFFLE_SIGNATURES

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
func (p *PriFiLibRelayInstance) Received_TRU_REL_SHUFFLE_SIG(msg net.TRU_REL_SHUFFLE_SIG) error {

	// this can only happens in the state RELAY_STATE_COLLECTING_SHUFFLE_SIGNATURES
	if p.relayState.currentState != RELAY_STATE_COLLECTING_SHUFFLE_SIGNATURES {
		e := "Relay : Received a TRU_REL_SHUFFLE_SIG, but not in state RELAY_STATE_COLLECTING_SHUFFLE_SIGNATURES, in state " + relayStateStr(p.relayState.currentState)
		log.Error(e)
		return errors.New(e)
	}
	log.Lvl3("Relay : received TRU_REL_SHUFFLE_SIG")

	done, err := p.relayState.neffShuffle.ReceivedSignatureFromTrustee(msg.TrusteeID, msg.Sig)
	if err != nil {
		e := "Could not do p.relayState.neffShuffle.ReceivedSignatureFromTrustee(), error is " + err.Error()
		log.Error(e)
		return errors.New(e)
	}

	// if we have all the signatures
	if done {
		trusteesPks := make([]abstract.Point, p.relayState.nTrustees)
		i := 0
		for _, v := range p.relayState.trustees {
			trusteesPks[i] = v.PublicKey
			i++
		}

		toSend5, err := p.relayState.neffShuffle.VerifySigsAndSendToClients(trusteesPks)
		if err != nil {
			e := "Could not do p.relayState.neffShuffle.VerifySigsAndSendToClients(), error is " + err.Error()
			log.Error(e)
			return errors.New(e)
		}
		msg := toSend5.(*net.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG)
		// changing state
		log.Lvl2("Relay : ready to communicate.")
		p.relayState.currentState = RELAY_STATE_COMMUNICATING

		// broadcast to all clients
		for i := 0; i < p.relayState.nClients; i++ {
			// send to the i-th client
			p.messageSender.SendToClientWithLog(i, msg, "(client "+strconv.Itoa(i+1)+")")
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
func (p *PriFiLibRelayInstance) checkIfRoundHasEndedAfterTimeOut_Phase1(roundID int32) {

	time.Sleep(TIMEOUT_PHASE_1)

	if p.relayState.currentDCNetRound.currentRound != roundID {
		return //everything went well, it's great !
	}

	if p.relayState.currentState == RELAY_STATE_SHUTDOWN {
		return //nothing to ensure in that case
	}

	allGood := true

	if p.relayState.bufferManager.CurrentRound() == roundID {
		log.Error("waitAndCheckIfClientsSentData : We seem to be stuck in round", roundID, ". Phase 1 timeout.")

		missingClientCiphers, missingTrusteesCiphers := p.relayState.bufferManager.MissingCiphersForCurrentRound()

		//If we're using UDP, client might have missed the broadcast, re-sending
		if p.relayState.UseUDP {
			for clientID := range missingClientCiphers {
				log.Error("Relay : Client " + strconv.Itoa(clientID) + " didn't sent us is cipher for round " + strconv.Itoa(int(roundID)) + ". Phase 1 timeout. Re-sending...")
				extraInfo := "(client " + strconv.Itoa(clientID) + ", round " + strconv.Itoa(int(p.relayState.currentDCNetRound.currentRound)) + ")"
				p.messageSender.SendToClientWithLog(clientID, &p.relayState.currentDCNetRound.dataAlreadySent, extraInfo)
			}
		}

		if len(missingClientCiphers) > 0 || len(missingTrusteesCiphers) > 0 {
			allGood = false
		}
	}

	if !allGood {
		//if we're not done (we miss data), wait another timeout, after which clients/trustees will be considered offline
		go p.checkIfRoundHasEndedAfterTimeOut_Phase2(roundID)
	}

	//this shouldn't happen frequently (it means that the timeout 1 was fired, but the round finished almost at the same time)
}

/*
This second timeout happens after a longer delay. Clients and trustees will be considered offline if they haven't send data yet
*/
func (p *PriFiLibRelayInstance) checkIfRoundHasEndedAfterTimeOut_Phase2(roundID int32) {

	time.Sleep(TIMEOUT_PHASE_2)

	if p.relayState.currentDCNetRound.currentRound != roundID {
		//everything went well, it's great !
		return
	}

	if p.relayState.currentState == RELAY_STATE_SHUTDOWN {
		//nothing to ensure in that case
		return
	}

	if p.relayState.bufferManager.CurrentRound() == roundID {
		log.Error("waitAndCheckIfClientsSentData : We seem to be stuck in round", roundID, ". Phase 2 timeout.")

		missingClientCiphers, missingTrusteesCiphers := p.relayState.bufferManager.MissingCiphersForCurrentRound()
		p.relayState.timeoutHandler(missingClientCiphers, missingTrusteesCiphers)
	}
}

// ReceivedMessage must be called when a PriFi host receives a message.
// It takes care to call the correct message handler function.
func (p *PriFiLibRelayInstance) ReceivedMessage(msg interface{}) error {

	var err error

	switch typedMsg := msg.(type) {
	case net.ALL_ALL_PARAMETERS:
		err = p.Received_ALL_ALL_PARAMETERS(typedMsg)
	case net.ALL_ALL_SHUTDOWN:
		err = p.Received_ALL_REL_SHUTDOWN(typedMsg)
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
		panic("unrecognized message !")
	}

	//no need to push the error further up. display it here !
	if err != nil {
		log.Error("ReceivedMessage: got an error, " + err.Error())
		return err
	}

	return nil
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
