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

	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"github.com/dedis/prifi/prifi-lib/config"
	"github.com/dedis/prifi/prifi-lib/dcnet"
	prifilog "github.com/dedis/prifi/prifi-lib/log"
	"github.com/dedis/prifi/prifi-lib/net"
	"github.com/dedis/prifi/prifi-lib/utils"
	"github.com/dedis/prifi/utils"
	"gopkg.in/dedis/kyber.v2"
	"gopkg.in/dedis/onet.v2/log"
)

/*
Received_ALL_REL_SHUTDOWN handles ALL_REL_SHUTDOWN messages.
When we receive this message, we should warn other protocol participants and clean resources.
*/
func (p *PriFiLibRelayInstance) Received_ALL_ALL_SHUTDOWN(msg net.ALL_ALL_SHUTDOWN) error {
	log.Lvl1("Relay : Received a SHUTDOWN message. ")

	p.stateMachine.ChangeState("SHUTDOWN")

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

	startNow := msg.BoolValueOrElse("StartNow", false)
	nTrustees := msg.IntValueOrElse("NTrustees", p.relayState.nTrustees)
	nClients := msg.IntValueOrElse("NClients", p.relayState.nClients)
	payloadSize := msg.IntValueOrElse("PayloadSize", p.relayState.PayloadSize)
	downCellSize := msg.IntValueOrElse("DownstreamCellSize", p.relayState.DownstreamCellSize)
	windowSize := msg.IntValueOrElse("WindowSize", p.relayState.WindowSize)
	useDummyDown := msg.BoolValueOrElse("UseDummyDataDown", p.relayState.UseDummyDataDown)
	useOpenClosedSlots := msg.BoolValueOrElse("UseOpenClosedSlots", p.relayState.UseOpenClosedSlots)
	reportingLimit := msg.IntValueOrElse("ExperimentRoundLimit", p.relayState.ExperimentRoundLimit)
	useUDP := msg.BoolValueOrElse("UseUDP", p.relayState.UseUDP)
	dcNetType := msg.StringValueOrElse("DCNetType", p.relayState.dcNetType)
	disruptionProtection := msg.BoolValueOrElse("DisruptionProtectionEnabled", false)
	openClosedSlotsMinDelayBetweenRequests := msg.IntValueOrElse("OpenClosedSlotsMinDelayBetweenRequests", p.relayState.OpenClosedSlotsMinDelayBetweenRequests)
	maxNumberOfConsecutiveFailedRounds := msg.IntValueOrElse("RelayMaxNumberOfConsecutiveFailedRounds", p.relayState.MaxNumberOfConsecutiveFailedRounds)
	processingLoopSleepTime := msg.IntValueOrElse("RelayProcessingLoopSleepTime", p.relayState.ProcessingLoopSleepTime)
	roundTimeOut := msg.IntValueOrElse("RelayRoundTimeOut", p.relayState.RoundTimeOut)
	trusteeCacheLowBound := msg.IntValueOrElse("RelayTrusteeCacheLowBound", p.relayState.TrusteeCacheLowBound)
	trusteeCacheHighBound := msg.IntValueOrElse("RelayTrusteeCacheHighBound", p.relayState.TrusteeCacheHighBound)
	equivocationProtectionEnabled := msg.BoolValueOrElse("EquivocationProtectionEnabled", p.relayState.EquivocationProtectionEnabled)

	if payloadSize < 1 {
		return errors.New("payloadSize cannot be 0")
	}

	p.relayState.clients = make([]NodeRepresentation, nClients)
	p.relayState.trustees = make([]NodeRepresentation, nTrustees)
	p.relayState.nClients = nClients
	p.relayState.nTrustees = nTrustees
	p.relayState.nTrusteesPkCollected = 0
	p.relayState.nClientsPkCollected = 0
	p.relayState.ExperimentRoundLimit = reportingLimit
	p.relayState.PayloadSize = payloadSize
	p.relayState.DownstreamCellSize = downCellSize
	p.relayState.bitrateStatistics = prifilog.NewBitRateStatistics(payloadSize)
	p.relayState.UseDummyDataDown = useDummyDown
	p.relayState.UseOpenClosedSlots = useOpenClosedSlots
	p.relayState.UseUDP = useUDP
	p.relayState.WindowSize = windowSize
	p.relayState.numberOfNonAckedDownstreamPackets = 0
	p.relayState.OpenClosedSlotsMinDelayBetweenRequests = openClosedSlotsMinDelayBetweenRequests
	p.relayState.MaxNumberOfConsecutiveFailedRounds = maxNumberOfConsecutiveFailedRounds
	p.relayState.ProcessingLoopSleepTime = processingLoopSleepTime
	p.relayState.RoundTimeOut = roundTimeOut
	p.relayState.TrusteeCacheLowBound = trusteeCacheLowBound
	p.relayState.TrusteeCacheHighBound = trusteeCacheHighBound
	p.relayState.EquivocationProtectionEnabled = equivocationProtectionEnabled
	p.relayState.MessageHistory = config.CryptoSuite.XOF([]byte("init")) //any non-nil, non-empty, constant array
	p.relayState.VerifiableDCNetKeys = make([][]byte, nTrustees)
	p.relayState.nVkeysCollected = 0
	p.relayState.roundManager = NewBufferableRoundManager(nClients, nTrustees, windowSize)
	p.relayState.dcNetType = dcNetType
	p.relayState.pcapLogger = utils.NewPCAPLog()
	p.relayState.DisruptionProtectionEnabled = disruptionProtection
	p.relayState.clientBitMap = make(map[int]map[int]int)
	p.relayState.trusteeBitMap = make(map[int]map[int]int)
	p.relayState.blamingData = make([]int, 6)
	p.relayState.OpenClosedSlotsRequestsRoundID = make(map[int32]bool)

	switch dcNetType {
	case "Verifiable":
		panic("Verifiable DCNet not implemented yet")
	}

	//this should be in NewRelayState, but we need p
	if !p.relayState.roundManager.DoSendStopResumeMessages {
		//Add rate-limiting component to buffer manager

		stopFn := func(trusteeID int) {
			toSend := &net.REL_TRU_TELL_RATE_CHANGE{WindowCapacity: 0}
			p.messageSender.SendToTrusteeWithLog(trusteeID, toSend, "(trustee "+strconv.Itoa(trusteeID)+")")
		}
		resumeFn := func(trusteeID int) {
			toSend := &net.REL_TRU_TELL_RATE_CHANGE{WindowCapacity: 1}
			p.messageSender.SendToTrusteeWithLog(trusteeID, toSend, "(trustee "+strconv.Itoa(trusteeID)+")")
		}
		p.relayState.roundManager.AddRateLimiter(p.relayState.TrusteeCacheLowBound, p.relayState.TrusteeCacheHighBound, stopFn, resumeFn)
	}

	log.Lvlf3("Relay new state: %+v\n", p.relayState)
	log.Lvl1("Relay has been initialized by message; StartNow is", startNow)

	// Broadcast those parameters to the other nodes, then tell the trustees which ID they are.
	if startNow {
		p.stateMachine.ChangeState("COLLECTING_TRUSTEES_PKS")
		p.BroadcastParameters()
	}
	log.Lvl1("Relay setup done, and setup sent to the trustees.")

	timing.StopMeasureAndLogWithInfo("resync-boot", strconv.Itoa(p.relayState.nClients))
	timing.StartMeasure("resync-shuffle")
	timing.StartMeasure("resync-shuffle-collect-client-pk")

	return nil
}

// ConnectToTrustees connects to the trustees and initializes them with default parameters.
func (p *PriFiLibRelayInstance) BroadcastParameters() error {

	// Craft default parameters
	msg := new(net.ALL_ALL_PARAMETERS)
	msg.Add("NClients", p.relayState.nClients)
	msg.Add("NTrustees", p.relayState.nTrustees)
	msg.Add("UseUDP", p.relayState.UseUDP)
	msg.Add("StartNow", true)
	msg.Add("PayloadSize", p.relayState.PayloadSize)
	msg.Add("DCNetType", p.relayState.dcNetType)
	msg.Add("DisruptionProtectionEnabled", p.relayState.DisruptionProtectionEnabled)
	msg.Add("EquivocationProtectionEnabled", p.relayState.EquivocationProtectionEnabled)
	msg.ForceParams = true

	// Send those parameters to all trustees
	for j := 0; j < p.relayState.nTrustees; j++ {

		// The ID is unique !
		msg.Add("NextFreeTrusteeID", j)
		p.messageSender.SendToTrusteeWithLog(j, msg, "")
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
	p.relayState.roundManager.AddClientCipher(msg.RoundID, int(msg.ClientID), msg.Data)
	if p.relayState.roundManager.HasAllCiphersForCurrentRound() {
		p.upstreamPhase1_processCiphers(true)
	}

	return nil
}

/*
Received_TRU_REL_DC_CIPHER handles TRU_REL_DC_CIPHER messages. Those contain a DC-net cipher from a Trustee.
If it's for this round, we call decode on it, and remember we received it.
If for a future round we need to Buffer it.
*/
func (p *PriFiLibRelayInstance) Received_TRU_REL_DC_CIPHER(msg net.TRU_REL_DC_CIPHER) error {
	p.relayState.roundManager.AddTrusteeCipher(msg.RoundID, int(msg.TrusteeID), msg.Data)
	if p.relayState.roundManager.HasAllCiphersForCurrentRound() {
		p.upstreamPhase1_processCiphers(true)
	}

	return nil
}

// Received_CLI_REL_OPENCLOSED_DATA handles the reception of the OpenClosed map, which details which
// pseudonymous clients want to transmit in a given round
func (p *PriFiLibRelayInstance) Received_CLI_REL_OPENCLOSED_DATA(msg net.CLI_REL_OPENCLOSED_DATA) error {
	p.relayState.roundManager.AddClientCipher(msg.RoundID, int(msg.ClientID), msg.OpenClosedData)
	if p.relayState.roundManager.HasAllCiphersForCurrentRound() {
		p.upstreamPhase1_processCiphers(false)
	}

	return nil
}

// upstreamPhase1_processCiphers collects all DC-net ciphers, and decides what to do with them (is it a OCMap message ?
// a data message ?)
// it then proceed accordingly, finalizes the round, and calls downstreamPhase_sendMany()
func (p *PriFiLibRelayInstance) upstreamPhase1_processCiphers(finishedByTrustee bool) {

	// keep statistics on who finished the round, to check on who the system is waiting
	if finishedByTrustee {
		timeMs := timing.StopMeasure("waiting-on-someone").Nanoseconds() / 1e6
		p.relayState.timeStatistics["waiting-on-trustees"].AddTime(timeMs)
	} else {
		timeMs := timing.StopMeasure("waiting-on-someone").Nanoseconds() / 1e6
		p.relayState.timeStatistics["waiting-on-clients"].AddTime(timeMs)
	}

	roundID := p.relayState.roundManager.CurrentRound()
	_, isOCRound := p.relayState.OpenClosedSlotsRequestsRoundID[roundID]

	log.Lvl3("Relay has collected all ciphers for round", roundID, "(isOCRound", isOCRound, "), decoding...")

	// most important switch of this method
	if isOCRound {
		err := p.upstreamPhase2a_extractOCMap(roundID)
		if err != nil {
			log.Lvl3("upstreamPhase2a_extractOCMap: error", err.Error())
		}
	} else {
		err := p.upstreamPhase2b_extractPayload()
		if err != nil {
			log.Lvl3("upstreamPhase2b_extractPayload: error", err.Error())
		}
	}

	// used if we're replaying a pcap. The first message we decode is "time0"
	if roundID == 0 {
		p.relayState.time0 = uint64(prifilog.MsTimeStampNow())
	}

	// one round has just passed ! Round start with downstream data, and end with upstream data, like here.
	p.upstreamPhase3_finalizeRound(roundID)

	// inter-round sleep
	if p.relayState.ProcessingLoopSleepTime > 0 {
		time.Sleep(time.Duration(p.relayState.ProcessingLoopSleepTime) * time.Millisecond)
	}

	// downstream phase
	p.downstreamPhase_sendMany()
}

// downstreamPhase_sendMany starts as many rounds (by opening the round and sending downstream data) as specified
// by the window
func (p *PriFiLibRelayInstance) downstreamPhase_sendMany() {
	// send the data down
	for i := p.relayState.numberOfNonAckedDownstreamPackets; i < p.relayState.WindowSize; i++ {
		log.Lvl3("Relay : Gonna send, non-acked packets is", p.relayState.numberOfNonAckedDownstreamPackets, "(window is", p.relayState.WindowSize, ")")
		p.downstreamPhase1_openRoundAndSendData()
	}
}

// upstreamPhase2a_extractOCMap extracts the open-closed request map, updates the inner OCMap stored, potentially
// sleeps if all slots are closed.
func (p *PriFiLibRelayInstance) upstreamPhase2a_extractOCMap(roundID int32) error {
	//classical DC-net decoding
	clientSlices, trusteesSlices, err := p.relayState.roundManager.CollectRoundData()
	if err != nil {
		return err
	}
	for _, s := range clientSlices {
		p.relayState.DCNet.DecodeClient(roundID, s)
	}
	for _, s := range trusteesSlices {
		p.relayState.DCNet.DecodeTrustee(roundID, s)
	}

	//here we have the plaintext map
	openClosedData := p.relayState.DCNet.DecodeCell()

	//compute the map
	newSchedule := p.relayState.slotScheduler.Relay_ComputeFinalSchedule(openClosedData, p.relayState.nClients)
	p.relayState.roundManager.SetStoredRoundSchedule(newSchedule)
	p.relayState.schedulesStatistics.AddSchedule(newSchedule)

	// if all slots are closed, do not immediately send the next downstream data (which will be a OCSlots schedule)
	hasOpenSlot := false
	for _, v := range newSchedule {
		if v {
			hasOpenSlot = true
			break
		}
	}
	if !hasOpenSlot {
		log.Lvl3("All slots closed, sleeping for", p.relayState.OpenClosedSlotsMinDelayBetweenRequests, "ms")
		d := time.Duration(p.relayState.OpenClosedSlotsMinDelayBetweenRequests) * time.Millisecond
		time.Sleep(d)
	}

	return nil
}

// upstreamPhase2b_extractPayload is called when we know the payload is data (and not an OCMap message)
// If enabled, it checks the Disruption protection, and perhaps starts a blame
// If it's a latency-test message, we send it back to the clients.
// If we use SOCKS/VPN, give them the data.
// If it's a pcap message, update the statistics accordingly
func (p *PriFiLibRelayInstance) upstreamPhase2b_extractPayload() error {

	// we decode the DC-net cell
	roundID := p.relayState.roundManager.CurrentRound()
	clientSlices, trusteesSlices, err := p.relayState.roundManager.CollectRoundData()
	if err != nil {
		return err
	}

	//decode all clients and trustees
	for _, s := range clientSlices {
		p.relayState.DCNet.DecodeClient(roundID, s)
	}
	for _, s := range trusteesSlices {
		p.relayState.DCNet.DecodeTrustee(roundID, s)
	}
	upstreamPlaintext := p.relayState.DCNet.DecodeCell()

	p.relayState.bitrateStatistics.AddUpstreamCell(int64(len(upstreamPlaintext)))

	//disruption-protection
	if p.relayState.DisruptionProtectionEnabled {

		log.Lvl3("Verifying HMAC for disruption protection")
		hmac := upstreamPlaintext[0:32]
		upstreamPlaintext = upstreamPlaintext[32:]

		clientID := -1 // todo loop with the schedule
		valid := ValidateHmac256(upstreamPlaintext, hmac, clientID)

		if !valid {
			// start blame
			log.Error("Warning: Disruption Protection check failed")
		}
	}

	log.Lvl4("Decoded cell is", upstreamPlaintext)

	// check if we have a latency test message, or a pcap meta message
	if len(upstreamPlaintext) >= 2 {
		pattern := int(binary.BigEndian.Uint16(upstreamPlaintext[0:2]))
		if pattern == 43690 {
			// 1010101010101010
			// then, we simply have to send it down
			// log.Info("Relay noticed a latency-test message on round", p.relayState.dcnetRoundManager.CurrentRound())
			p.relayState.PriorityDataForClients <- upstreamPlaintext
		} else if pattern == 21845 {
			//0101010101010101
			ID := int32(binary.BigEndian.Uint32(upstreamPlaintext[2:6]))
			timestamp := int64(binary.BigEndian.Uint64(upstreamPlaintext[6:14]))
			frag := false
			if upstreamPlaintext[14] == byte(1) {
				frag = true
			}
			now := prifilog.MsTimeStampNow() - int64(p.relayState.time0)
			diff := now - timestamp

			log.Lvl2("Got a PCAP meta-message (id", ID, ",frag", frag, ") at", now, ", delay since original is", diff, "ms")
			p.relayState.timeStatistics["pcap-delay"].AddTime(diff)
			p.relayState.pcapLogger.ReceivedPcap(uint32(ID), frag, uint64(timestamp), p.relayState.time0, uint32(len(upstreamPlaintext)))

			//also decode other messages
			pos := 15
			for pos+15 <= len(upstreamPlaintext) {
				pattern := int(binary.BigEndian.Uint16(upstreamPlaintext[pos : pos+2]))
				if pattern != 21845 {
					break
				}
				ID := int32(binary.BigEndian.Uint32(upstreamPlaintext[pos+2 : pos+6]))
				timestamp := int64(binary.BigEndian.Uint64(upstreamPlaintext[pos+6 : pos+14]))
				frag := false
				if upstreamPlaintext[pos+14] == byte(1) {
					frag = true
				}

				now := prifilog.MsTimeStampNow() - int64(p.relayState.time0)
				diff := now - timestamp

				log.Lvl2("Got a PCAP meta-message (id", ID, ",frag", frag, ") at", now, ", delay since original is", diff, "ms")
				p.relayState.timeStatistics["pcap-delay"].AddTime(diff)
				p.relayState.pcapLogger.ReceivedPcap(uint32(ID), frag, uint64(timestamp), p.relayState.time0, uint32(len(upstreamPlaintext)))

				pos += 15
			}

		}
	}

	if upstreamPlaintext != nil {
		// verify that the decoded payload has the correct size
		expectedSize := p.relayState.PayloadSize
		if p.relayState.DisruptionProtectionEnabled {
			expectedSize -= 32
		}
		if len(upstreamPlaintext) != expectedSize {
			e := "Relay : DecodeCell produced wrong-size payload, " + strconv.Itoa(len(upstreamPlaintext)) + "!=" + strconv.Itoa(p.relayState.PayloadSize)
			log.Error(e)
			return errors.New(e)
		}

		if p.relayState.DataOutputEnabled {
			p.relayState.DataFromDCNet <- upstreamPlaintext
		}
	}

	return nil
}

// upstreamPhase3_FinalizeRound happens when the data for the upstream round has been collected, and essentially
// close the current round
func (p *PriFiLibRelayInstance) upstreamPhase3_finalizeRound(roundID int32) error {

	p.relayState.numberOfNonAckedDownstreamPackets--
	p.relayState.numberOfConsecutiveFailedRounds = 0

	// collects timing experiments
	if roundID == 0 {
		log.Lvl2("Relay finished round " + strconv.Itoa(int(roundID)) + " .")
	} else {
		log.Lvl2("Relay finished round "+strconv.Itoa(int(roundID))+" (after", p.relayState.roundManager.TimeSpentInRound(roundID), ").")
		p.collectExperimentResult(p.relayState.bitrateStatistics.Report())
		p.collectExperimentResult(p.relayState.schedulesStatistics.Report())
		timeSpent := p.relayState.roundManager.TimeSpentInRound(roundID)
		p.relayState.timeStatistics["round-duration"].AddTime(timeSpent.Nanoseconds() / 1e6) //ms
		for k, v := range p.relayState.timeStatistics {
			p.collectExperimentResult(v.ReportWithInfo(k))
		}
	}

	// Test if we are doing an experiment, and if we need to stop at some point.
	newRound := p.relayState.roundManager.CurrentRound()
	if newRound == int32(p.relayState.ExperimentRoundLimit) {
		log.Lvl1("Relay : Experiment round limit (", newRound, ") reached")
		p.relayState.ExperimentResultChannel <- p.relayState.ExperimentResultData

		// shut down everybody
		msg := net.ALL_ALL_SHUTDOWN{}
		p.Received_ALL_ALL_SHUTDOWN(msg)
	}

	p.relayState.roundManager.CloseRound()

	// we just closed that round. If there is any other round opened (window > 1), directly prepare to decode it
	if roundOpened, nextRoundID := p.relayState.roundManager.currentRound(); roundOpened {
		p.relayState.DCNet.DecodeStart(nextRoundID)
	}

	return nil
}

/*
sendDownstreamData is simply called when the Relay has processed the upstream cell from all clients, and is ready to finalize the round by sending the data down.
If it's a latency-test message, we send it back to the clients.
If we use SOCKS/VPN, give them the data.
Since after this function, we'll start receiving data for the next round, if we have buffered data for this next round, tell the state that we
have the data already (and we're not waiting on it). Clean the old data.
*/
func (p *PriFiLibRelayInstance) downstreamPhase1_openRoundAndSendData() error {

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

		default:
			downstreamCellContent = make([]byte, 1)
		}
	}

	// if we want to use dummy data down, pad to the correct size
	if p.relayState.UseDummyDataDown && len(downstreamCellContent) < p.relayState.DownstreamCellSize {
		data := make([]byte, p.relayState.DownstreamCellSize)
		copy(data[0:], downstreamCellContent)
		downstreamCellContent = data
	}

	nextDownstreamRoundID := p.relayState.roundManager.NextRoundToOpen()

	// TODO : if something went wrong before, this flag should be used to warn the clients that the config has changed
	flagResync := false

	// periodically set to True so client can advertise their bitmap
	flagOpenClosedRequest := p.relayState.UseOpenClosedSlots &&
		p.relayState.roundManager.IsNextDownstreamRoundForOpenClosedRequest(p.relayState.nClients)
	if flagOpenClosedRequest {
		p.relayState.OpenClosedSlotsRequestsRoundID[nextDownstreamRoundID] = true
	}

	//compute next owner
	nextOwner := p.relayState.roundManager.UpdateAndGetNextOwnerID()

	//sending data part
	timing.StartMeasure("sending-data")
	if flagOpenClosedRequest {
		log.Lvl2("Relay is gonna broadcast messages for round "+strconv.Itoa(int(nextDownstreamRoundID))+" (OCRequest=true), owner=", nextOwner, ", len", len(downstreamCellContent))
	} else {
		log.Lvl2("Relay is gonna broadcast messages for round "+strconv.Itoa(int(nextDownstreamRoundID))+" (OCRequest=false), owner=", nextOwner, ", len", len(downstreamCellContent))
	}

	toSend := &net.REL_CLI_DOWNSTREAM_DATA{
		RoundID:               nextDownstreamRoundID,
		OwnershipID:           int32(nextOwner),
		Data:                  downstreamCellContent,
		FlagResync:            flagResync,
		FlagOpenClosedRequest: flagOpenClosedRequest}

	if roundOpened, _ := p.relayState.roundManager.currentRound(); !roundOpened {
		//prepare for the next round (this empties the dc-net buffer, making them ready for a new round)
		p.relayState.DCNet.DecodeStart(nextDownstreamRoundID)
	}

	p.relayState.roundManager.OpenNextRound()
	p.relayState.roundManager.SetDataAlreadySent(nextDownstreamRoundID, toSend)

	if !p.relayState.UseUDP {
		// broadcast to all clients
		for i := 0; i < p.relayState.nClients; i++ {
			//send to the i-th client
			p.messageSender.SendToClientWithLog(i, toSend, "(client "+strconv.Itoa(i)+", round "+strconv.Itoa(int(nextDownstreamRoundID))+")")
		}

		p.relayState.bitrateStatistics.AddDownstreamCell(int64(len(downstreamCellContent)))
	} else {
		toSend2 := &net.REL_CLI_DOWNSTREAM_DATA_UDP{REL_CLI_DOWNSTREAM_DATA: *toSend}
		p.messageSender.BroadcastToAllClientsWithLog(toSend2, "(UDP broadcast, round "+strconv.Itoa(int(nextDownstreamRoundID))+")")

		p.relayState.bitrateStatistics.AddDownstreamUDPCell(int64(len(downstreamCellContent)), p.relayState.nClients)
	}

	timeMs := timing.StopMeasure("sending-data").Nanoseconds() / 1e6
	p.relayState.timeStatistics["sending-data"].AddTime(timeMs)

	log.Lvl3("Relay is done broadcasting messages for round " + strconv.Itoa(int(nextDownstreamRoundID)) + ".")

	//we just sent the data down, initiating a round. Let's prevent being blocked by a dead client
	go p.checkIfRoundHasEndedAfterTimeOut_Phase1(nextDownstreamRoundID)

	//now relay enters a waiting state (collecting all ciphers from clients/trustees)
	timing.StartMeasure("waiting-on-someone")

	p.relayState.numberOfNonAckedDownstreamPackets++

	return nil
}

/*
Received_TRU_REL_TELL_PK handles TRU_REL_TELL_PK messages. Those are sent by the trustees message when we connect them.
We do nothing, until we have received one per trustee; Then, we pack them in one message, and broadcast it to the clients.
*/
func (p *PriFiLibRelayInstance) Received_TRU_REL_TELL_PK(msg net.TRU_REL_TELL_PK) error {

	p.relayState.trustees[msg.TrusteeID] = NodeRepresentation{int(msg.TrusteeID), true, msg.Pk, msg.Pk}
	p.relayState.nTrusteesPkCollected++

	log.Lvl2("Relay : received TRU_REL_TELL_PK (" + strconv.Itoa(p.relayState.nTrusteesPkCollected) + "/" + strconv.Itoa(p.relayState.nTrustees) + ")")

	// if we have them all...
	if p.relayState.nTrusteesPkCollected == p.relayState.nTrustees {

		// prepare the message for the clients
		trusteesPk := make([]kyber.Point, p.relayState.nTrustees)
		for i := 0; i < p.relayState.nTrustees; i++ {
			trusteesPk[i] = p.relayState.trustees[i].PublicKey
		}

		//send that to the clients, along with the parameters
		toSend := new(net.ALL_ALL_PARAMETERS)
		toSend.Add("NClients", p.relayState.nClients)
		toSend.Add("NTrustees", p.relayState.nTrustees)
		toSend.Add("UseUDP", p.relayState.UseUDP)
		toSend.Add("StartNow", true)
		toSend.Add("PayloadSize", p.relayState.PayloadSize)
		toSend.Add("DCNetType", p.relayState.dcNetType)
		toSend.Add("DisruptionProtectionEnabled", p.relayState.DisruptionProtectionEnabled)
		toSend.Add("EquivocationProtectionEnabled", p.relayState.EquivocationProtectionEnabled)
		toSend.TrusteesPks = trusteesPk

		// Send those parameters to all clients
		for j := 0; j < p.relayState.nClients; j++ {
			// The ID is unique !
			toSend.Add("NextFreeClientID", j)
			p.messageSender.SendToClientWithLog(j, toSend, "")
		}

		p.stateMachine.ChangeState("COLLECTING_CLIENT_PKS")
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

	p.relayState.clients[msg.ClientID] = NodeRepresentation{int(msg.ClientID), true, msg.Pk, msg.EphPk}
	p.relayState.nClientsPkCollected++

	log.Lvl2("Relay : received CLI_REL_TELL_PK_AND_EPH_PK (" + strconv.Itoa(p.relayState.nClientsPkCollected) + "/" + strconv.Itoa(p.relayState.nClients) + ")")

	// if we have collected all clients, continue
	if p.relayState.nClientsPkCollected == p.relayState.nClients {

		timing.StopMeasureAndLogWithInfo("resync-shuffle-collect-client-pk", strconv.Itoa(p.relayState.nClients))
		timing.StartMeasure("resync-shuffle-trustee-1step")

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

		//todo: fix this. The neff shuffle now stores twices the ephemeral public keys
		toSend.Pks = make([]kyber.Point, p.relayState.nClients)
		for i := 0; i < p.relayState.nClients; i++ {
			toSend.Pks[i] = p.relayState.clients[i].PublicKey
		}

		// send to the 1st trustee
		p.messageSender.SendToTrusteeWithLog(trusteeID, toSend, "(0-th iteration)")

		p.stateMachine.ChangeState("COLLECTING_SHUFFLES")
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

	p.relayState.VerifiableDCNetKeys[p.relayState.nVkeysCollected] = msg.VerifiableDCNetKey
	p.relayState.nVkeysCollected++

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

		//todo: fix this. The neff shuffle now stores twices the ephemeral public keys
		toSend.Pks = make([]kyber.Point, p.relayState.nClients)
		for i := 0; i < p.relayState.nClients; i++ {
			toSend.Pks[i] = p.relayState.clients[i].PublicKey
		}

		// send to the i-th trustee
		p.messageSender.SendToTrusteeWithLog(trusteeID, toSend, "("+strconv.Itoa(trusteeID)+"-th iteration)")

	} else {
		// if we have all the shuffles

		timing.StopMeasureAndLogWithInfo("resync-shuffle-trustee-1step", strconv.Itoa(p.relayState.nClients))
		timing.StartMeasure("resync-shuffle-trustee-2step")

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

		p.relayState.DCNet = dcnet.NewDCNetEntity(0, dcnet.DCNET_RELAY, p.relayState.PayloadSize,
			p.relayState.EquivocationProtectionEnabled, nil)

		// prepare to collect the ciphers
		p.relayState.DCNet.DecodeStart(0)

		p.stateMachine.ChangeState("COLLECTING_SHUFFLE_SIGNATURES")

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

	done, err := p.relayState.neffShuffle.ReceivedSignatureFromTrustee(int(msg.TrusteeID), msg.Sig)
	if err != nil {
		e := "Could not do p.relayState.neffShuffle.ReceivedSignatureFromTrustee(), error is " + err.Error()
		log.Error(e)
		return errors.New(e)
	}

	// if we have all the signatures
	if done {
		trusteesPks := make([]kyber.Point, p.relayState.nTrustees)
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
		p.relayState.roundManager.OpenNextRound()
		log.Lvl2("Relay : ready to communicate.")
		p.stateMachine.ChangeState("COMMUNICATING")

		timing.StopMeasureAndLogWithInfo("resync-shuffle-trustee-2step", strconv.Itoa(p.relayState.nClients))
		timing.StopMeasureAndLogWithInfo("resync-shuffle", strconv.Itoa(p.relayState.nClients))
		timing.StopMeasureAndLogWithInfo("resync", strconv.Itoa(p.relayState.nClients))

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

// ValidateHmac256 returns true iff the recomputed HMAC is equal to the given one
func ValidateHmac256(message, inputHmac []byte, clientID int) bool {
	key := []byte("client-secret" + strconv.Itoa(clientID)) // quick hack, this should be a random shared secret
	h := hmac.New(sha256.New, key)
	h.Write(message)
	computedHmac := h.Sum(nil)
	return bytes.Equal(inputHmac, computedHmac)
}

// updates p.relayState.ExperimentResultData
func (p *PriFiLibRelayInstance) collectExperimentResult(str string) {
	if str == "" {
		return
	}

	// if this is not an experiment, simply return
	if p.relayState.ExperimentRoundLimit == -1 {
		return
	}

	p.relayState.ExperimentResultData = append(p.relayState.ExperimentResultData, str)
}
