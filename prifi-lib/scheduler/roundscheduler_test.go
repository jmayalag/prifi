package scheduler

import (
	"fmt"
	"testing"
)

func Test1Client(t *testing.T) {

	bmc := new(BitMaskScheduler_Client)

	//arbitrary constants for this test
	currentRound := int32(12)
	mySlot := 4 //we are c2, assigned to slot 2
	nClients := 5

	//0:c0, 1:c1, ... 4:c4, 5:c0, ... 10:c0, ... 12:c2, 13:c3, 14:c4, 15:c0

	//relay starts by sending a downstream message (for round currentRound) with OpenClosedSchedRequest=true
	//client notice it, instead of sending the next upstream traffic, he will send the OCSchedule
	bmc.Client_ReceivedScheduleRequest(currentRound+1, nClients)

	//rotate schedule since in this slot no data will be sent
	mySlot = (mySlot + 1) % nClients // we are c2, assigned to slot 3

	//12:SCHEDULE, 13:c2, 14:c3, 15:c4, 16:c0

	//he first compute which slot will be ours (starting next, after the schedule)
	i := currentRound + 1
	for i%int32(nClients) != int32(mySlot) {
		i++
	}
	mySlotInNextRound := int32(i)
	fmt.Println("Gonna reserve round", mySlotInNextRound)
	bmc.Client_ReserveRound(mySlotInNextRound)

	contribution := bmc.Client_GetOpenScheduleContribution()

	if len(contribution) != 1 {
		t.Error("Contribution should have length 1, has length", len(contribution))
	}

	//The contribution is sent to the relay
	bmr := new(BitMaskScheduler_Relay)

	bmr.Relay_ReceiveScheduleContribution(contribution)
	finalSched := bmr.Relay_ComputeFinalSchedule(currentRound + 1)

	if len(finalSched) > 1 {
		t.Error("finalSched should have length 1, has length", len(finalSched))
	}

	if !finalSched[mySlotInNextRound] {
		t.Error("finalSched should have slot", mySlotInNextRound, "open")
	}

	fmt.Println(finalSched)
	fmt.Println("Done")

}
