package trustee

import (
	"errors"
	"github.com/lbarman/prifi/prifi-lib/crypto"
	"github.com/lbarman/prifi/prifi-lib/dcnet"
	"github.com/lbarman/prifi/prifi-lib/net"
	"github.com/lbarman/prifi/prifi-lib/scheduler"
	"github.com/lbarman/prifi/prifi-lib/utils"
	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/onet.v1/log"
	"reflect"
	"strings"
	"time"
)

// Possible sending rates for the trustees.
const (
	TRUSTEE_KILL_SEND_PROCESS int16 = iota // kills the goroutine responsible for sending messages
	TRUSTEE_RATE_ACTIVE
	TRUSTEE_RATE_STOPPED
)

// TRUSTEE_BASE_SLEEP_TIME is the base unit for how much time the trustee sleeps between sending ciphers to the relay.
const TRUSTEE_BASE_SLEEP_TIME = 10 * time.Millisecond

// PriFiLibTrusteeInstance contains the mutable state of a PriFi entity.
type PriFiLibTrusteeInstance struct {
	messageSender *net.MessageSenderWrapper
	trusteeState  *TrusteeState
	stateMachine  *utils.StateMachine
}

// NewPriFiClientWithState creates a new PriFi client entity state.
func NewTrustee(msgSender *net.MessageSenderWrapper) *PriFiLibTrusteeInstance {

	trusteeState := new(TrusteeState)

	//init the static stuff
	trusteeState.sendingRate = make(chan int16, 10)
	//trusteeState.CellCoder = config.Factory()
	trusteeState.PublicKey, trusteeState.privateKey = crypto.NewKeyPair()
	neffShuffle := new(scheduler.NeffShuffle)
	neffShuffle.Init()
	trusteeState.neffShuffle = neffShuffle.TrusteeView

	//init the state machine
	states := []string{"BEFORE_INIT", "INITIALIZING", "SHUFFLE_DONE", "READY", "COMMUNICATING", "SHUTDOWN"}
	sm := new(utils.StateMachine)
	logFn := func(s interface{}) {
		log.Lvl3(s)
	}
	errFn := func(s interface{}) {
		if strings.Contains(s.(string), ", but in state SHUTDOWN") { //it's an "acceptable error"
			log.Lvl2(s)
		} else {
			log.Error(s)
		}
	}
	sm.Init(states, logFn, errFn)

	prifi := PriFiLibTrusteeInstance{
		messageSender: msgSender,
		trusteeState:  trusteeState,
		stateMachine:  sm,
	}
	return &prifi
}

// TrusteeState contains the mutable state of the trustee.
type TrusteeState struct {
	CellCoder        dcnet.CellCoder
	ClientPublicKeys []abstract.Point
	ID               int
	MessageHistory   abstract.Cipher
	Name             string
	nClients         int
	neffShuffle      *scheduler.NeffShuffleTrustee
	nTrustees        int
	PayloadLength    int
	privateKey       abstract.Scalar
	PublicKey        abstract.Point
	sendingRate      chan int16
	sharedSecrets    []abstract.Point
	TrusteeID        int
}

// NeffShuffleResult holds the result of the NeffShuffle,
// since it needs to be verified when we receive REL_TRU_TELL_TRANSCRIPT.
type NeffShuffleResult struct {
	base  abstract.Point
	pks   []abstract.Point
	proof []byte
}

// ReceivedMessage must be called when a PriFi host receives a message.
// It takes care to call the correct message handler function.
func (p *PriFiLibTrusteeInstance) ReceivedMessage(msg interface{}) (bool, interface{}, error) {

	var err error
	var endStep bool
	var state interface{}

	switch typedMsg := msg.(type) {
	case net.ALL_ALL_PARAMETERS_NEW:
		if typedMsg.ForceParams || p.stateMachine.AssertState("BEFORE_INIT") {
			endStep, state, err = p.Received_ALL_ALL_PARAMETERS(typedMsg) //todo change this name
		}
	case net.ALL_ALL_SHUTDOWN:
		endStep, state, err = p.Received_ALL_ALL_SHUTDOWN(typedMsg)
	case net.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE:
		if p.stateMachine.AssertState("INITIALIZING") {
			endStep, state, err = p.Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE(typedMsg)
		}
	case net.REL_TRU_TELL_TRANSCRIPT:
		if p.stateMachine.AssertState("SHUFFLE_DONE") {
			endStep, state, err = p.Received_REL_TRU_TELL_TRANSCRIPT(typedMsg)
		}
	case net.REL_TRU_TELL_READY:
		if p.stateMachine.AssertState("READY") {
			endStep, state, err = p.Received_REL_TRU_TELL_READY(typedMsg)
		}
	case net.REL_TRU_TELL_RATE_CHANGE:
		if p.stateMachine.AssertState("COMMUNICATING") {
			endStep, state, err = p.Received_REL_TRU_TELL_RATE_CHANGE(typedMsg)
		}
	default:
		err = errors.New("Unrecognized message, type" + reflect.TypeOf(msg).String())
		endStep = false
		state = nil
	}

	return endStep, state, err
}

// SetMessageSender is used to change the message sender of the current Trustee Instance
func (p *PriFiLibTrusteeInstance) SetMessageSender(msgSender net.MessageSender) error {
	errHandling := func(e error) { /* do nothing yet, we are alerted of errors via the SDA */ }
	loggingSuccessFunction := func(e interface{}) { log.Lvl3(e) }
	loggingErrorFunction := func(e interface{}) { log.Error(e) }

	msw, err := net.NewMessageSenderWrapper(true, loggingSuccessFunction, loggingErrorFunction, errHandling, msgSender)
	if err != nil {
		log.Fatal("Could not create a MessageSenderWrapper, error is", err)
	}
	p.messageSender = msw
	return nil
}
