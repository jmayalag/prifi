package relay

import (
	"github.com/lbarman/prifi/prifi-lib/net"
	"sync"
	"time"
)

// DCNetRound counts how many (upstream) messages we received for a given DC-net round.
type DCNetRound struct {
	sync.Mutex
	currentRound    int32
	dataAlreadySent *net.REL_CLI_DOWNSTREAM_DATA
	startTime       time.Time
}

// Creates a DCNetRound that hold a roundID, some data (sent at the beginning of the round, in case some client missed it), and the time the round started
func NewDCNetRound(currentRound int32, dataAlreadySent *net.REL_CLI_DOWNSTREAM_DATA) *DCNetRound {
	dcnetRound := new(DCNetRound)
	dcnetRound.startTime = time.Now()
	dcnetRound.currentRound = currentRound
	dcnetRound.dataAlreadySent = dataAlreadySent
	return dcnetRound
}

//Check if we are still in the given round
func (dc *DCNetRound) ChangeRound(newRound int32) {
	dc.Lock()
	defer dc.Unlock()
	dc.currentRound = newRound
	dc.dataAlreadySent = nil
	dc.startTime = time.Now()
}

//Check if we are still in the given round
func (dc *DCNetRound) isStillInRound(round int32) bool {
	dc.Lock()
	defer dc.Unlock()
	return (dc.currentRound == round)
}

//Returns the current round
func (dc *DCNetRound) CurrentRound() int32 {
	dc.Lock()
	defer dc.Unlock()
	return dc.currentRound
}

//return the time delta since the creation of the DCNetRound struct
func (dc *DCNetRound) TimeSpentInRound() time.Duration {
	dc.Lock()
	defer dc.Unlock()
	return time.Since(dc.startTime)
}

//Set the "DataAlreadySent" field
func (dc *DCNetRound) SetDataAlreadySent(data *net.REL_CLI_DOWNSTREAM_DATA) {
	dc.Lock()
	defer dc.Unlock()
	dc.dataAlreadySent = data
}

//Gets the "DataAlreadySent" field
func (dc *DCNetRound) GetDataAlreadySent() *net.REL_CLI_DOWNSTREAM_DATA {
	dc.Lock()
	defer dc.Unlock()
	return dc.dataAlreadySent
}
