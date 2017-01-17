package utils

import (
	"testing"
)

func TestSM(t *testing.T) {

	sm := new(StateMachine)
	logFn := func(s interface{}) {
		//log.Lvl1(s)
	}
	errFn := func(s interface{}) {
		//log.Error(s)
	}
	sm.Init([]string{"INIT", "COMM", "SHUTDOWN"}, logFn, errFn)

	if !sm.AssertState("INIT") {
		t.Error("We are in state init")
	}

	if sm.AssertState("ninja") {
		t.Error("ninja is an invalid state")
	}

	if sm.AssertState("SHUTDOWN") {
		t.Error("We are not in state SHUTDOWN")
	}

	sm.ChangeState("SHUTDOWN")

	sm.ChangeState("ninja")

	if sm.State() != "SHUTDOWN" {
		t.Error("we are not in state shutdown", sm.State())
	}

}
