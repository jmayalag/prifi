package log

import (
	"time"
	"sync"
	"github.com/fatih/color"
)

const (
	ACTION_ENTER_STATE = iota
	ACTION_EXIT_STATE
)

type StateMachineStateChange struct {
	Name			string
	Action			int16
	Time			time.Time
}

type StateMachineLogs struct {
	sync.Mutex
	currentState	string
	timeEnterState	time.Time
	measures		[]StateMachineStateChange
}

//output is meaningful only when EXITING a state
func (sml *StateMachineLogs) addStateChange(newState string, action int16) time.Duration {
	//NOT thread safe, but private
	currentTime        := time.Now()
	newMeasure         := StateMachineStateChange{newState, action, currentTime}
	sml.measures       = append(sml.measures, newMeasure)

	timeSpentInPrevState := time.Since(sml.timeEnterState)
	sml.timeEnterState = currentTime

	return timeSpentInPrevState
}

func (sml *StateMachineLogs) Init () {
	sml.Lock()

	initialState     := "init"
	sml.currentState = initialState
	sml.measures     = make([]StateMachineStateChange, 0)

	sml.addStateChange(initialState, ACTION_ENTER_STATE)

	sml.Unlock()
}

func (sml *StateMachineLogs) StateChange(newState string){
	sml.Lock()

	//exit
	oldState := sml.currentState
	timeSpendInState := sml.addStateChange(oldState, ACTION_EXIT_STATE)
	sml.addStateChange(newState, ACTION_ENTER_STATE)

	color.Blue("[Timings] Left state %s after %s sec.", oldState, timeSpendInState)

	sml.Unlock()
}

/*
 * SOURCE : https://github.com/DeDiS/cothority/blob/development/lib/monitor/measure.go
 */
/*
// Convert microseconds to seconds
func iiToF(sec int64, usec int64) float64 {
	return float64(sec) + float64(usec)/1000000.0
}

// Gets the sytem and the user time so far
func GetRTime() (tSys, tUsr float64) {
	rusage := &syscall.Rusage{}
	syscall.Getrusage(syscall.RUSAGE_SELF, rusage)
	s, u := rusage.Stime, rusage.Utime
	return iiToF(int64(s.Sec), int64(s.Usec)), iiToF(int64(u.Sec), int64(u.Usec))
}
*/