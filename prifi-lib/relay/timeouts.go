package relay

import (
	"strconv"
	"time"

	"github.com/dedis/cothority/log"
)

/*
This first timeout happens after a short delay. Clients will not be considered disconnected yet,
but if we use UDP, it can mean that a client missed a broadcast, and we re-sent the message.
If the round was *not* done, we do another timeout (Phase 2), and then, clients/trustees will be considered
online if they didn't answer by that time.
*/
func (p *PriFiLibRelayInstance) checkIfRoundHasEndedAfterTimeOut_Phase1(roundID int32) {

	time.Sleep(TIMEOUT_PHASE_1)

	if !p.relayState.currentDCNetRound.isStillInRound(roundID) {
		return //everything went well, it's great !
	}

	if p.stateMachine.State() == "SHUTDOWN" {
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
				extraInfo := "(client " + strconv.Itoa(clientID) + ", round " + strconv.Itoa(int(p.relayState.currentDCNetRound.CurrentRound())) + ")"
				p.messageSender.SendToClientWithLog(clientID, p.relayState.currentDCNetRound.GetDataAlreadySent(), extraInfo)
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

	if !p.relayState.currentDCNetRound.isStillInRound(roundID) {
		//everything went well, it's great !
		return
	}

	if p.stateMachine.State() == "SHUTDOWN" {
		//nothing to ensure in that case
		return
	}

	if p.relayState.bufferManager.CurrentRound() == roundID {
		log.Error("waitAndCheckIfClientsSentData : We seem to be stuck in round", roundID, ". Phase 2 timeout.")

		missingClientCiphers, missingTrusteesCiphers := p.relayState.bufferManager.MissingCiphersForCurrentRound()
		p.relayState.timeoutHandler(missingClientCiphers, missingTrusteesCiphers)
	}
}
