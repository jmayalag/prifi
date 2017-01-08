package timing

import (
	"time"
	"fmt"
	"github.com/lbarman/prifi/utils/output"
	"sync"
)

var startTimes map[string]time.Time = make(map[string]time.Time)
var mutex sync.Mutex
var outputInterface output.Output = output.PrintOutput{}

// StartMeasure starts a time measure identified by a name.
func StartMeasure(name string) {
	mutex.Lock()

	if _, present := startTimes[name]; present {
		// Unlock before potentially expensive writing to output.
		mutex.Unlock()
		msg := fmt.Sprint("WARNING: starting a measure that already exists with name: ", name, " (nothing will happen)")
		outputInterface.Print(msg)
	} else {
		startTimes[name] = time.Now()
		mutex.Unlock()
	}
}

// StopMeasure stops a time measure identified by a name,
// prints the result to the current output interface and
// returns the measured time. Returns 0 if no measure was
// started with that name.
func StopMeasure(name string) time.Duration {
	// Store call time in case we have to wait for the mutex.
	now := time.Now()

	mutex.Lock()

	if start, ok := startTimes[name]; ok {
		duration := now.Sub(start)
		delete(startTimes, name)
		// Unlock before potentially expensive writing to output.
		mutex.Unlock()

		msg := fmt.Sprint("Measured time for ", name, " : ", duration)
		outputInterface.Print(msg)

		return duration
	} else {
		// Unlock before potentially expensive writing to output.
		mutex.Unlock()

		msg := fmt.Sprint("WARNING: stopping a measure that was not started with name: ", name)
		outputInterface.Print(msg)

		return time.Duration(0)
	}
}

// SetOutputInterface sets the output interface to use
// to print measure results.
func SetOutputInterface(out output.Output) {
	outputInterface = out
}
