package log

import (
	"time"
	"fmt"
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

type StateMachineLogger struct {
	sync.Mutex
	currentState	string
	timeEnterState	time.Time
	measures		[]StateMachineStateChange
}

//output is meaningful only when EXITING a state
func (sml *StateMachineLogger) addStateChange(newState string, action int16) time.Duration {
	//NOT thread safe, but private
	currentTime        := time.Now()
	newMeasure         := StateMachineStateChange{newState, action, currentTime}
	sml.measures       = append(sml.measures, newMeasure)

	timeSpentInPrevState := time.Since(sml.timeEnterState)
	sml.timeEnterState   = currentTime
	sml.currentState     = newState

	return timeSpentInPrevState
}

func NewStateMachineLogger() *StateMachineLogger {
	sml := StateMachineLogger{}
	sml.Init()

	return &sml
}

func (sml *StateMachineLogger) Init () {
	fmt.Println("requesting lock...")
	sml.Lock()

	initialState       := "statemachinelogger-init"
	sml.timeEnterState = time.Now()
	sml.measures       = make([]StateMachineStateChange, 0)

	sml.addStateChange(initialState, ACTION_ENTER_STATE)

	sml.Unlock()
	fmt.Println("releasing lock...")
}

func (sml *StateMachineLogger) StateChange(newState string){
	fmt.Println("requesting lock...")
	sml.Lock()

	//exit
	oldState := sml.currentState
	timeSpendInState := sml.addStateChange(oldState, ACTION_EXIT_STATE)
	sml.addStateChange(newState, ACTION_ENTER_STATE)

	color.White("[Timings] Left state %s after %s", oldState, timeSpendInState)

	sml.Unlock()
	fmt.Println("releasing lock...")
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