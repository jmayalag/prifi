package relay

import (
	"gopkg.in/dedis/onet.v1/log"
	"time"
)

/*
This first timeout happens after a short delay. Clients will not be considered disconnected yet,
but if we use UDP, it can mean that a client missed a broadcast, and we re-sent the message.
If the round was *not* done, we do another timeout (Phase 2), and then, clients/trustees will be considered
online if they didn't answer by that time.
*/
func (p *PriFiLibRelayInstance) checkIfRoundHasEndedAfterTimeOut_Phase1(roundID int32) {

	time.Sleep(time.Duration(p.relayState.RoundTimeOut) * time.Millisecond)

	// never start treating two timeout concurrently
	p.relayState.timeoutMutex.Lock()
	defer p.relayState.timeoutMutex.Unlock()

	if !p.relayState.roundManager.IsRoundOpenend(roundID) {
		return //everything went dwell, it's great !
	}

	if p.stateMachine.State() == "SHUTDOWN" {
		return //nothing to ensure in that case
	}

	// new policy : just kill that round, do not retransmit, let SOCKS take care of the loss

	p.relayState.numberOfConsecutiveFailedRounds++
	log.Lvl1("WARNING: Timeout for round", roundID, ", force closing. Already", p.relayState.numberOfConsecutiveFailedRounds,
		"consecutive missed rounds (killing when =>", p.relayState.MaxNumberOfConsecutiveFailedRounds, ")")

	// if we missed too many rounds, kill the experiment
	missingClientCiphers, missingTrusteeCiphers := p.relayState.roundManager.MissingCiphersForCurrentRound()
	log.Lvl1("missing clients", missingClientCiphers, "and trustees", missingTrusteeCiphers)

	if p.relayState.numberOfConsecutiveFailedRounds >= p.relayState.MaxNumberOfConsecutiveFailedRounds {
		log.Error("MAX_NUMBER_OF_CONSECUTIVE_FAILED_ROUNDS (", p.relayState.MaxNumberOfConsecutiveFailedRounds,
			") reached, killing protocol.")

		log.Lvl3("Stopping experiment, if any.")
		missingClientCiphers, missingTrusteesCiphers := p.relayState.roundManager.MissingCiphersForCurrentRound()
		p.relayState.timeoutHandler(missingClientCiphers, missingTrusteesCiphers)
	} else {
		// cleanup, start the transition to next round
		log.Lvl1("Gonna Force close...")
		p.relayState.roundManager.Dump()
		p.relayState.roundManager.ForceCloseRound()
		p.relayState.roundManager.Dump()

		p.relayState.numberOfNonAckedDownstreamPacketsLock.Lock()
		p.relayState.numberOfNonAckedDownstreamPackets-- // packet is not "in-flight" because it is lost
		p.relayState.numberOfNonAckedDownstreamPacketsLock.Unlock()

		p.relayState.DCNet.DecodeStart(p.relayState.roundManager.CurrentRound())

		// if we can, open new rounds
		p.downstreamPhase_sendMany()

		// we should also try to finalize the next round
		if p.relayState.roundManager.HasAllCiphersForCurrentRound() {
			log.Lvl1("Timeouts: Following round was ready, calling hasAllCiphersForUpstream(true)")
			p.upstreamPhase1_processCiphers(true)
		}
	}
}
