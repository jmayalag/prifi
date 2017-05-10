package scheduler

import (
	"fmt"
	"testing"
)

func Test1Client(t *testing.T) {

	bmc := new(BitMaskSlotScheduler_Client)

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
	bmr := new(BitMaskSlotScheduler_Relay)

	finalSched := bmr.Relay_ComputeFinalSchedule(contribution, currentRound+1, nClients)

	if len(finalSched) != nClients {
		t.Error("finalSched should have length", nClients, ", has length", len(finalSched))
	}

	if !finalSched[mySlotInNextRound] {
		t.Error("finalSched should have slot", mySlotInNextRound, "open")
	}

	fmt.Println(finalSched)
}

func Test2Client(t *testing.T) {

	bmc1 := new(BitMaskSlotScheduler_Client)
	bmc2 := new(BitMaskSlotScheduler_Client)

	//arbitrary constants for this test
	currentRound := int32(12)
	mySlot1 := 3 //we are c0 assigned to 3
	mySlot2 := 4 //we are c1 assigned to 4
	nClients := 5

	//relay starts by sending a downstream message (for round currentRound) with OpenClosedSchedRequest=true
	//client notice it, instead of sending the next upstream traffic, he will send the OCSchedule
	bmc1.Client_ReceivedScheduleRequest(currentRound+1, nClients)
	bmc2.Client_ReceivedScheduleRequest(currentRound+1, nClients)

	//rotate schedule since in this slot no data will be sent
	mySlot1 = (mySlot1 + 1) % nClients // we are c0, assigned to slot 4
	mySlot2 = (mySlot2 + 1) % nClients // we are c1, assigned to slot 0

	//he first compute which slot will be ours (starting next, after the schedule)
	i := currentRound + 1
	for i%int32(nClients) != int32(mySlot1) {
		i++
	}
	mySlotInNextRound1 := int32(i)
	i = currentRound + 1
	for i%int32(nClients) != int32(mySlot2) {
		i++
	}
	mySlotInNextRound2 := int32(i)
	fmt.Println("Client 0: Gonna reserve round", mySlotInNextRound1)
	fmt.Println("Client 1: Gonna reserve round", mySlotInNextRound2)
	bmc1.Client_ReserveRound(mySlotInNextRound1)
	bmc2.Client_ReserveRound(mySlotInNextRound2)

	contribution1 := bmc1.Client_GetOpenScheduleContribution()
	contribution2 := bmc2.Client_GetOpenScheduleContribution()

	if len(contribution1) != 1 {
		t.Error("Contribution should have length 1, has length", len(contribution1))
	}
	if len(contribution2) != 1 {
		t.Error("Contribution should have length 1, has length", len(contribution2))
	}

	//The contribution is sent to the relay
	bmr := new(BitMaskSlotScheduler_Relay)

	contributions := bmr.Relay_CombineContributions(contribution1, contribution2)
	finalSched := bmr.Relay_ComputeFinalSchedule(contributions, currentRound+1, nClients)

	if len(finalSched) != nClients {
		t.Error("finalSched should have length", nClients, ", has length", len(finalSched))
	}

	if !finalSched[mySlotInNextRound1] {
		t.Error("finalSched should have slot", mySlotInNextRound1, "open")
	}

	if !finalSched[mySlotInNextRound2] {
		t.Error("finalSched should have slot", mySlotInNextRound2, "open")
	}

	fmt.Println(finalSched)
}
