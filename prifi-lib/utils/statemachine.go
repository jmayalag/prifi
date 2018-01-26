package utils

import "sync"

// is used to asset that an entity is in a given state
type StateMachine struct {
	sync.Mutex
	entity       string
	currentState string
	states       []string
	logInfo      func(interface{})
	logErr       func(interface{})
}

// creates a StateMachine with two logging functions. The initial state will be states[0]
func (s *StateMachine) Init(states []string, logInfo, logErr func(interface{})) {
	s.entity = ""
	s.logInfo = logInfo
	s.logErr = logErr
	s.states = states
	s.currentState = states[0]
}

func allowedState(states []string, input string) bool {
	for i := 0; i < len(states); i++ {
		if states[i] == input {
			return true
		}
	}
	return false
}

// sets the entity used in printing the log messages
func (s *StateMachine) SetEntity(e string) {
	s.entity = e
}

// asserts (and returns true/false) that the state is the one given. Fails if the given state is invalid
func (s *StateMachine) AssertState(state string) bool {
	s.Lock()
	defer s.Unlock()
	if !allowedState(s.states, state) {
		s.logErr(s.entity + ": Required State " + state + " which is not a valid state.")
		return false
	}
	if s.currentState != state {
		s.logErr(s.entity + ": Required State " + state + ", but in state " + s.currentState)
		return false
	}
	return true
}

// asserts (and returns true/false) that the state is the one given. Fails if the given state is invalid
func (s *StateMachine) AssertStateOrState(state1 string, state2 string) bool {
	s.Lock()
	defer s.Unlock()
	if !allowedState(s.states, state1) {
		s.logErr(s.entity + ": Required State1 " + state1 + " which is not a valid state.")
		return false
	}
	if !allowedState(s.states, state2) {
		s.logErr(s.entity + ": Required State2 " + state2 + " which is not a valid state.")
		return false
	}
	if s.currentState != state1 && s.currentState != state2 {
		s.logErr(s.entity + ": Required State " + state1 + " or " + state2 + ", but in state " + s.currentState)
		return false
	}
	return true
}

// changes state if it is valid
func (s *StateMachine) ChangeState(newState string) {
	s.Lock()
	defer s.Unlock()

	if !allowedState(s.states, newState) {
		s.logErr(s.entity + ": Cannot change state to " + newState + " which is not valid.")
		return
	}
	s.currentState = newState
}

// returns the current state
func (s *StateMachine) State() string {
	s.Lock()
	defer s.Unlock()

	return s.currentState
}
