package relay

import (
	"gopkg.in/dedis/onet.v1/log"
	"strconv"
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

	if p.relayState.roundManager.HasAllCiphersForCurrentRound() {
		p.hasAllCiphersForUpstream(true)
	} else {
		log.Error("waitAndCheckIfClientsSentData : We seem to be stuck in round", roundID, ". Phase 2 timeout.")

		log.Lvl3("Stopping experiment, if any.")
		output := p.relayState.ExperimentResultData
		output = append(output, "!!aborted-round-"+strconv.Itoa(int(roundID)))
		p.relayState.ExperimentResultChannel <- output

		missingClientCiphers, missingTrusteesCiphers := p.relayState.roundManager.MissingCiphersForCurrentRound()
		p.relayState.timeoutHandler(missingClientCiphers, missingTrusteesCiphers)
	}
}
