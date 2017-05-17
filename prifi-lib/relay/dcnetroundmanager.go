package relay

import (
	"github.com/lbarman/prifi/prifi-lib/net"
	"gopkg.in/dedis/onet.v1/log"
	"sync"
	"time"
)

// DCNetRound counts how many (upstream) messages we received for a given DC-net round.
type DCNetRoundManager struct {
	sync.Mutex
	currentRound                int32
	maxNumberOfConcurrentRounds int
	dataAlreadySent             map[int32]*net.REL_CLI_DOWNSTREAM_DATA
	startTimes                  map[int32]time.Time
	storedRoundsSchedule        map[int32]bool
}

// Creates a DCNetRound that hold a roundID, some data (sent at the beginning of the round, in case some client missed it), and the time the round started
func NewDCNetRoundManager(maxNumberOfConcurrentRounds int) *DCNetRoundManager {
	dcRM := new(DCNetRoundManager)
	dcRM.currentRound = 0
	dcRM.maxNumberOfConcurrentRounds = maxNumberOfConcurrentRounds
	dcRM.startTimes = make(map[int32]time.Time)
	dcRM.dataAlreadySent = make(map[int32]*net.REL_CLI_DOWNSTREAM_DATA)

	return dcRM
}

//Check if we are still in the given round
func (dc *DCNetRoundManager) OpenRound(roundID int32) {
	dc.Lock()
	defer dc.Unlock()

	//make sure not to open more rounds than allowed (or we won't count correctly)
	if len(dc.startTimes) >= dc.maxNumberOfConcurrentRounds {
		log.Fatal("Tried to OpenRound(", roundID, "), but we have already", len(dc.startTimes), "rounds opened.")
	}

	dc.dataAlreadySent[roundID] = nil
	dc.startTimes[roundID] = time.Now()
}

//Check if we are still in the given round
func (dc *DCNetRoundManager) CloseRound(roundID int32) {
	dc.Lock()
	defer dc.Unlock()

	//this is a DC-net, devices are in lock-step, never close a round if another with smaller ID is open
	for i := 1; i <= dc.maxNumberOfConcurrentRounds; i++ {
		indexToCheck := roundID - int32(i)
		if indexToCheck >= 0 {
			if _, found := dc.startTimes[indexToCheck]; found {
				log.Fatal("Tried to CloseRound(", roundID, "), but round", indexToCheck, "is still opened.")
			}
		}
	}

	dc.currentRound = roundID + 1
	for dc.roundShouldBeSkipped(dc.currentRound) {
		dc.currentRound++
	}

	delete(dc.dataAlreadySent, roundID)
	delete(dc.startTimes, roundID)
}

//Check if we are still in the given round
func (dc *DCNetRoundManager) CurrentRoundIsStill(round int32) bool {
	dc.Lock()
	defer dc.Unlock()
	return (dc.currentRound == round)
}

//Returns the current round
func (dc *DCNetRoundManager) CurrentRound() int32 {
	dc.Lock()
	defer dc.Unlock()
	return dc.currentRound
}

//return the time delta since the creation of the DCNetRound struct
func (dc *DCNetRoundManager) TimeSpentInRound(roundID int32) time.Duration {
	dc.Lock()
	defer dc.Unlock()

	if startTime, found := dc.startTimes[roundID]; found {
		return time.Since(startTime)
	}
	log.Fatal("Requested duration for round", roundID, ", but round has been closed already (or was not found).")
	return time.Hour
}

//Set the "DataAlreadySent" field for the given round
func (dc *DCNetRoundManager) SetDataAlreadySent(roundID int32, data *net.REL_CLI_DOWNSTREAM_DATA) {
	dc.Lock()
	defer dc.Unlock()
	dc.dataAlreadySent[roundID] = data
}

//Gets the "DataAlreadySent" field for the given round
func (dc *DCNetRoundManager) GetDataAlreadySent(roundID int32) *net.REL_CLI_DOWNSTREAM_DATA {
	dc.Lock()
	defer dc.Unlock()
	if data, found := dc.dataAlreadySent[roundID]; found {
		return data
	}
	log.Fatal("Requested data already sent for round", roundID, ", but round has been closed already (or was not found).")
	return nil
}

func (dc *DCNetRoundManager) roundAlreadyOpened(roundID int32) bool {
	for k := range dc.startTimes {
		if roundID == k {
			return true
		}
	}
	return false
}

func (dc *DCNetRoundManager) roundShouldBeSkipped(roundID int32) bool {
	if dc.storedRoundsSchedule == nil {
		return false
	}

	if isOpen, ok := dc.storedRoundsSchedule[roundID]; ok {
		return !isOpen
	}
	return false
}

//NextDownStreamRoundToSent returns the next downstream round to send, and takes cares of closed slots
func (dc *DCNetRoundManager) NextDownStreamRoundToSent() int32 {
	dc.Lock()
	defer dc.Unlock()

	nextDownstreamRound := dc.currentRound

	//do not return an open round
	for dc.roundAlreadyOpened(nextDownstreamRound) {
		nextDownstreamRound++
	}

	//if we don't have a closed schedule, return this
	if dc.storedRoundsSchedule == nil {
		return nextDownstreamRound
	}

	//else, increment it
	doSend, found := dc.storedRoundsSchedule[nextDownstreamRound]
	for found && !doSend {
		//log.Error("Going to skip round", nextDownstreamRound, doSend, bmr.storedSchedule)
		nextDownstreamRound++
		doSend, found = dc.storedRoundsSchedule[nextDownstreamRound]
	}

	return nextDownstreamRound
}

//IsNextDownstreamRoundForOpenClosedRequest return true if the next downstream round has flagOpenCloseScheduleRequest == true
func (dc *DCNetRoundManager) IsNextDownstreamRoundForOpenClosedRequest(nClients int) bool {
	return (dc.NextDownStreamRoundToSent()%int32(nClients+1) == 0)
}

//SetStoredRoundSchedule simply stores s
func (dc *DCNetRoundManager) SetStoredRoundSchedule(s map[int32]bool) {
	//only accessed by a single thread
	dc.storedRoundsSchedule = s
}
