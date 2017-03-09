package relay

import (
	"gopkg.in/dedis/onet.v1/log"
	"sync"
	"time"
)

type ByteArray struct {
	Data []byte
}

// DCNetRound counts how many (upstream) messages we received for a given DC-net round.
type DCNetRoundManager struct {
	sync.Mutex
	currentRound                int32
	maxNumberOfConcurrentRounds int
	dataAlreadySent             map[int32]*ByteArray
	startTimes                  map[int32]time.Time
}

// Creates a DCNetRound that hold a roundID, some data (sent at the beginning of the round, in case some client missed it), and the time the round started
func NewDCNetRoundManager(maxNumberOfConcurrentRounds int) *DCNetRoundManager {
	dcRM := new(DCNetRoundManager)
	dcRM.currentRound = 0
	dcRM.maxNumberOfConcurrentRounds = maxNumberOfConcurrentRounds
	dcRM.startTimes = make(map[int32]time.Time)
	dcRM.dataAlreadySent = make(map[int32]*ByteArray)

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

	dc.currentRound++

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
func (dc *DCNetRoundManager) SetDataAlreadySent(roundID int32, data []byte) {
	dc.Lock()
	defer dc.Unlock()
	dc.dataAlreadySent[roundID] = &ByteArray{data}
}

//Gets the "DataAlreadySent" field for the given round
func (dc *DCNetRoundManager) GetDataAlreadySent(roundID int32) []byte {
	dc.Lock()
	defer dc.Unlock()
	if data, found := dc.dataAlreadySent[roundID]; found {
		return data.Data
	}
	log.Fatal("Requested data already sent for round", roundID, ", but round has been closed already (or was not found).")
	return nil
}
