package scheduler

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
)

type RoundScheduler interface {

	//the client receives a new schedule request from the relay
	Client_ReceivedScheduleRequest()

	//the client alters the schedule being computed, and ask to transmit
	Client_ReserveRound(slotID int32) error

	//return the schedule to send as payload
	Client_GetOpenScheduleContribution() []byte

	//Called with each client's contribution
	Relay_ReceiveScheduleContribution(contribution []byte)

	//returns all contributions in forms of a map of open slots
	Relay_ComputeFinalSchedule() map[int32]bool
}

type StoredSchedule struct {
	startRoundID       int32
	endRoundID         int32
	openClosedSchedule map[int32]bool
}

type BitMaskScheduler_Client struct {
	NClients    int
	BaseRoundID int32
	WantToSend  bool
	MySlot      int
}

type BitMaskScheduler_Relay struct {
	ReceivedContributions []byte
}

func (bmc *BitMaskScheduler_Client) Client_ReceivedScheduleRequest(baseRoundID int32, nClients int) {
	bmc.NClients = nClients
	bmc.BaseRoundID = baseRoundID
	bmc.WantToSend = false
	bmc.MySlot = -1
}

func (bmc *BitMaskScheduler_Client) Client_ReserveRound(roundID int32) {
	if roundID < bmc.BaseRoundID {
		panic("Cannot reserve round" + strconv.Itoa(int(roundID)) + "since next schedule starts at" + strconv.Itoa(int(bmc.BaseRoundID)))
	}

	fmt.Println("bmc.NextRoundSchedule.BaseRoundID", bmc.BaseRoundID, "roundID", roundID)
	slotID := roundID - bmc.BaseRoundID
	bmc.MySlot = int(slotID)
	bmc.WantToSend = true
}

func (bmc *BitMaskScheduler_Client) Client_GetOpenScheduleContribution() []byte {
	nBytes := int(math.Ceil(float64(bmc.NClients) / 8))

	payload := make([]byte, nBytes)

	if !bmc.WantToSend {
		return payload //all zeros
	}

	whichByte := int(math.Floor(float64(bmc.MySlot) / 8))
	whichBit := uint(bmc.MySlot % 8)
	fmt.Println("bmc.NextScheduleMySlot", bmc.MySlot, "whichByte", whichByte, "whichBit", whichBit)
	payload[whichByte] = 1 << whichBit
	return payload
}

func (bmr *BitMaskScheduler_Relay) Relay_ReceiveScheduleContribution(contribution []byte) {
	if bmr.ReceivedContributions == nil {
		bmr.ReceivedContributions = make([]byte, len(contribution))
	}
	fmt.Println("Contribution was", hex.Dump(bmr.ReceivedContributions))
	for i := range contribution {
		bmr.ReceivedContributions[i] ^= contribution[i]
	}
	fmt.Println("Contribution  is", hex.Dump(bmr.ReceivedContributions))
}

func (bmr *BitMaskScheduler_Relay) Relay_ComputeFinalSchedule(baseRoundID int32) map[int32]bool {

	allContribs := bmr.ReceivedContributions
	res := make(map[int32]bool)

	for byteIndex, b := range allContribs {
		for bitPos := uint(0); bitPos < 8; bitPos++ {
			val := b & (1 << bitPos)
			if val > 0 { //the bit was set
				roundID := baseRoundID + int32(byteIndex*8+int(bitPos))
				res[roundID] = true
			}
		}
	}

	return res
}
