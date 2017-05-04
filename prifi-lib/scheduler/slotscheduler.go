package scheduler

import (
	"math"
	"strconv"
)

// SlotScheduler is a protocol between the relay and the clients that allows to decide which slots are gonna be
// "open" (fixed-length byte array) or "closed" (inexistant, no message at all)
type SlotScheduler interface {

	//the client receives a new schedule request from the relay
	Client_ReceivedScheduleRequest()

	//the client alters the schedule being computed, and ask to transmit
	Client_ReserveSlot(slotID int32)

	//return the schedule to send as payload
	Client_GetOpenScheduleContribution() []byte

	//Called with each client's contribution
	Relay_ReceiveScheduleContribution(contribution []byte)

	//returns all contributions in forms of a map of open slots
	Relay_ComputeFinalSchedule() map[int32]bool
}

// BitMaskScheduler_Client holds the info necessary for a client to compute his "contribution", or part of the bitmask
type BitMaskSlotScheduler_Client struct {
	NClients               int
	beginningOfRoundSlotID int32
	ClientWantsToSend      bool
	MySlotInNextRound      int
}

// BitMaskScheduler_Relay holds all contributions sent by clients, before converting it to a map of slots
type BitMaskSlotScheduler_Relay struct {
	ReceivedContributions []byte
}

// Client_ReceivedScheduleRequest instantiates the fields of BitMaskScheduler_Client
func (bmc *BitMaskSlotScheduler_Client) Client_ReceivedScheduleRequest(beginningOfRoundSlotID int32, nClients int) {
	bmc.NClients = nClients
	bmc.beginningOfRoundSlotID = beginningOfRoundSlotID
	bmc.ClientWantsToSend = false
	bmc.MySlotInNextRound = -1
}

// Client_ReserveRound indicates to reserve a slot in the next round
func (bmc *BitMaskSlotScheduler_Client) Client_ReserveSlot(slotID int32) {
	if slotID < bmc.beginningOfRoundSlotID {
		panic("Cannot reserve slot " + strconv.Itoa(int(slotID)) + " since next scheduled round starts at slot " + strconv.Itoa(int(bmc.beginningOfRoundSlotID)))
	}

	slotIDInNextRound := slotID - bmc.beginningOfRoundSlotID
	bmc.MySlotInNextRound = int(slotIDInNextRound)
	bmc.ClientWantsToSend = true
}

// Client_GetOpenScheduleContribution computes their contribution as a bit array
func (bmc *BitMaskSlotScheduler_Client) Client_GetOpenScheduleContribution() []byte {
	//length of the contribution is nClients/8 bytes
	nBytes := int(math.Ceil(float64(bmc.NClients) / 8))
	payload := make([]byte, nBytes)

	if !bmc.ClientWantsToSend {
		return payload //all zeros
	}

	//set a bit to 1 at the correct position
	whichByte := int(math.Floor(float64(bmc.MySlotInNextRound) / 8))
	whichBit := uint(bmc.MySlotInNextRound % 8)
	payload[whichByte] = 1 << whichBit
	return payload
}

// Relay_ReceiveScheduleContribution stores the received schedule contribution
func (bmr *BitMaskSlotScheduler_Relay) Relay_ReceiveScheduleContribution(contribution []byte) {
	if bmr.ReceivedContributions == nil {
		bmr.ReceivedContributions = make([]byte, len(contribution))
	}
	for i := range contribution {
		bmr.ReceivedContributions[i] ^= contribution[i]
	}
}

// Relay_ComputeFinalSchedule computes the map[int32]bool of open slots in the next round given the stored contributions
func (bmr *BitMaskSlotScheduler_Relay) Relay_ComputeFinalSchedule(baseRoundID int32, maxSlots int) map[int32]bool {

	allContribs := bmr.ReceivedContributions
	res := make(map[int32]bool)

	for byteIndex, b := range allContribs {
		for bitPos := uint(0); bitPos < 8; bitPos++ {
			roundID := baseRoundID + int32(byteIndex*8+int(bitPos))
			val := b & (1 << bitPos)
			if val > 0 { //the bit was set
				res[roundID] = true
			} else {
				res[roundID] = false
			}
			if bitPos == uint(maxSlots) - 1{
				return res
			}
		}
	}

	return res
}
