package trustee

import (
	"errors"
	"github.com/dedis/crypto/abstract"
	"github.com/lbarman/prifi/prifi-lib/config"
	"github.com/lbarman/prifi/prifi-lib/crypto"
	"github.com/lbarman/prifi/prifi-lib/dcnet"
	"github.com/lbarman/prifi/prifi-lib/net"
	"github.com/lbarman/prifi/prifi-lib/scheduler"
	"reflect"
	"time"
)

// Possible states the trustees are in. This restrict the kind of messages they can receive at a given point in time.
const (
	TRUSTEE_STATE_BEFORE_INIT int16 = iota
	TRUSTEE_STATE_INITIALIZING
	TRUSTEE_STATE_SHUFFLE_DONE
	TRUSTEE_STATE_READY
	TRUSTEE_STATE_SHUTDOWN
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
}

// NewPriFiClientWithState creates a new PriFi client entity state.
func NewTrustee(msgSender *net.MessageSenderWrapper) *PriFiLibTrusteeInstance {

	trusteeState := new(TrusteeState)

	//init the static stuff
	trusteeState.sendingRate = make(chan int16, 10)
	trusteeState.CellCoder = config.Factory()
	trusteeState.PublicKey, trusteeState.privateKey = crypto.NewKeyPair()
	neffShuffle := new(scheduler.NeffShuffle)
	neffShuffle.Init()
	trusteeState.neffShuffle = neffShuffle.TrusteeView

	prifi := PriFiLibTrusteeInstance{
		messageSender: msgSender,
		trusteeState:  trusteeState,
	}
	return &prifi
}

// TrusteeState contains the mutable state of the trustee.
type TrusteeState struct {
	CellCoder        dcnet.CellCoder
	ClientPublicKeys []abstract.Point
	currentState     int16
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
func (p *PriFiLibTrusteeInstance) ReceivedMessage(msg interface{}) error {

	var err error

	switch typedMsg := msg.(type) {
	case net.ALL_ALL_PARAMETERS_NEW:
		err = p.Received_ALL_ALL_PARAMETERS(typedMsg) //todo change this name
	case net.ALL_ALL_SHUTDOWN:
		err = p.Received_ALL_ALL_SHUTDOWN(typedMsg)
	case net.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE:
		err = p.Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE(typedMsg)
	case net.REL_TRU_TELL_TRANSCRIPT:
		err = p.Received_REL_TRU_TELL_TRANSCRIPT(typedMsg)
	case net.REL_TRU_TELL_RATE_CHANGE:
		err = p.Received_REL_TRU_TELL_RATE_CHANGE(typedMsg)
	default:
		err = errors.New("Unrecognized message, type" + reflect.TypeOf(msg).String())
	}

	return err
}
