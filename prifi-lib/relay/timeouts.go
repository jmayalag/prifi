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

	time.Sleep(TIMEOUT_PHASE_1)

	if !p.relayState.roundManager.IsRoundOpenend(roundID) {
		return //everything went well, it's great !
	}

	if p.stateMachine.State() == "SHUTDOWN" {
		return //nothing to ensure in that case
	}

	// new policy : just kill that round, do not retransmit, let SOCKS take care of the loss

	p.relayState.roundManager.ForceCloseRound()
	p.relayState.numberOfConsecutiveFailedRounds++
	log.Lvl1("WARNING: consecutive missed round is", p.relayState.numberOfConsecutiveFailedRounds)

	if p.relayState.roundManager.HasAllCiphersForCurrentRound() {
		p.hasAllCiphersForUpstream(true)
	}

	if p.relayState.numberOfConsecutiveFailedRounds >= p.relayState.WindowSize ||
		p.relayState.numberOfConsecutiveFailedRounds >= MAX_NUMBER_OF_CONSECUTIVE_FAILED_ROUNDS {
		log.Error("MAX_NUMBER_OF_CONSECUTIVE_FAILED_ROUNDS reached, killing protocol.")

		log.Lvl3("Stopping experiment, if any.")
		//output := p.relayState.ExperimentResultData
		//output = append(output, "!!aborted-round-"+strconv.Itoa(int(roundID)))
		//p.relayState.ExperimentResultChannel <- output

		missingClientCiphers, missingTrusteesCiphers := p.relayState.roundManager.MissingCiphersForCurrentRound()
		p.relayState.timeoutHandler(missingClientCiphers, missingTrusteesCiphers)
	}
}
