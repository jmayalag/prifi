package prifi_lib

import (
	"strconv"

	"github.com/dedis/cothority/log"
	"github.com/lbarman/prifi/prifi-lib/net"
	"github.com/lbarman/prifi/prifi-lib/relay"
)

/*
PriFi - Library
***************
This is a network-agnostic PriFi library. Feed it with a MessageSender interface (that knows how to contact the different entities),
and call ReceivedMessage(msg) with the received messages.
Then, it runs the PriFi anonymous communication network among those entities.
*/

// PriFiLibInstance contains the mutable state of a PriFi entity.
type PriFiLibInstance struct {
	role          int16
	messageSender *net.MessageSenderWrapper

	clientState  ClientState
	trusteeState TrusteeState

	specializedLibInstance SpecializedLibInstance
}

type SpecializedLibInstance interface {
	ReceivedMessage(msg interface{}) error
}

// Possible role of PriFi entities.
// The role restricts the kind of messages an entity can receive at
// a given point in time. The roles are mutually exclusive.
const (
	PRIFI_ROLE_UNDEFINED int16 = iota
	PRIFI_ROLE_RELAY
	PRIFI_ROLE_CLIENT
	PRIFI_ROLE_TRUSTEE
)

/*
call the functions below on the appropriate machine on the network.
if you call *without state* (one of the first 3 methods), IT IS NOT SUFFICIENT FOR PRIFI to start; this entity will expect a ALL_ALL_PARAMETERS as a
first message to finish initializing itself (this is handy if only the Relay has access to the configuration file).
Otherwise, the 3 last methods fully initialize the entity.
*/

func newMessageSenderWrapper(msgSender net.MessageSender) *net.MessageSenderWrapper {

	errHandling := func(e error) { /* do nothing yet, we are alerted of errors via the SDA */ }
	loggingSuccessFunction := func(e interface{}) { log.Lvl3(e) }
	loggingErrorFunction := func(e interface{}) { log.Error(e) }

	msw, err := net.NewMessageSenderWrapper(true, loggingSuccessFunction, loggingErrorFunction, errHandling, msgSender)
	if err != nil {
		log.Fatal("Could not create a MessageSenderWrapper, error is", err)
	}
	return msw
}


// NewPriFiClient creates a new PriFi client entity state.
// Note: the returned state is not sufficient for the PrFi protocol
// to start; this entity will expect a ALL_ALL_PARAMETERS message as
// first received message to complete it's state.
func NewPriFiClient(msgSender net.MessageSender) *PriFiLibInstance {
	prifi := PriFiLibInstance{
		role:          PRIFI_ROLE_CLIENT,
		messageSender: newMessageSenderWrapper(msgSender),
	}
	return &prifi
}
func NewPriFiRelay(msgSender net.MessageSender) *PriFiLibInstance {
	prifi := PriFiLibInstance{
		role:          PRIFI_ROLE_RELAY,
		specializedLibInstance: relay.NewPriFiRelay(newMessageSenderWrapper(msgSender)),
	}
	return &prifi
}
func NewPriFiRelayWithState(msgSender net.MessageSender, state *relay.RelayState) *PriFiLibInstance {
	prifi := PriFiLibInstance{
		role:          PRIFI_ROLE_RELAY,
		specializedLibInstance: relay.NewPriFiRelayWithState(newMessageSenderWrapper(msgSender), state),
	}
	return &prifi
}

// NewPriFiTrustee creates a new PriFi trustee entity state.
// Note: the returned state is not sufficient for the PrFi protocol
// to start; this entity will expect a ALL_ALL_PARAMETERS message as
// first received message to complete it's state.
func NewPriFiTrustee(msgSender net.MessageSender) *PriFiLibInstance {
	prifi := PriFiLibInstance{
		role:          PRIFI_ROLE_TRUSTEE,
		messageSender: newMessageSenderWrapper(msgSender),
	}
	return &prifi
}

// NewPriFiClientWithState creates a new PriFi client entity state.
func NewPriFiClientWithState(msgSender net.MessageSender, state *ClientState) *PriFiLibInstance {
	prifi := PriFiLibInstance{
		role:          PRIFI_ROLE_CLIENT,
		messageSender: newMessageSenderWrapper(msgSender),
		clientState:   *state,
	}
	log.Lvl1("Client has been initialized by function call. ")

	log.Lvl2("Client " + strconv.Itoa(prifi.clientState.ID) + " : starting the broadcast-listener goroutine")
	go prifi.messageSender.ClientSubscribeToBroadcast(prifi.clientState.Name, prifi.ReceivedMessage, prifi.clientState.StartStopReceiveBroadcast)
	return &prifi
}

// NewPriFiTrusteeWithState creates a new PriFi trustee entity state.
func NewPriFiTrusteeWithState(msgSender net.MessageSender, state *TrusteeState) *PriFiLibInstance {
	prifi := PriFiLibInstance{
		role:          PRIFI_ROLE_TRUSTEE,
		messageSender: newMessageSenderWrapper(msgSender),
		trusteeState:  *state,
	}

	log.Lvl1("Trustee has been initialized by function call. ")
	return &prifi
}

// ReceivedMessage must be called when a PriFi host receives a message.
// It takes care to call the correct message handler function.
func (prifi *PriFiLibInstance) ReceivedMessage(msg interface{}) error {

	if prifi == nil {
		log.Print("Received a message ", msg)
		panic("But prifi is nil !")
	}

	var err error

	switch typedMsg := msg.(type) {
	case net.ALL_ALL_PARAMETERS_NEW:
		switch prifi.role {
		case PRIFI_ROLE_CLIENT:
			err = prifi.Received_ALL_CLI_PARAMETERS(typedMsg)
		case PRIFI_ROLE_TRUSTEE:
			err = prifi.Received_ALL_TRU_PARAMETERS(typedMsg)
		default:
			prifi.specializedLibInstance.ReceivedMessage(typedMsg)
		}
	case net.ALL_ALL_SHUTDOWN:
		switch prifi.role {
		case PRIFI_ROLE_CLIENT:
			err = prifi.Received_ALL_CLI_SHUTDOWN(typedMsg)
		case PRIFI_ROLE_TRUSTEE:
			err = prifi.Received_ALL_TRU_SHUTDOWN(typedMsg)
		default:
			prifi.specializedLibInstance.ReceivedMessage(typedMsg)
		}
	case net.REL_CLI_DOWNSTREAM_DATA:
		err = prifi.Received_REL_CLI_DOWNSTREAM_DATA(typedMsg)
	/*
	 * this message is a bit special. At this point, we don't care anymore that's it's UDP, and cast it back to REL_CLI_DOWNSTREAM_DATA.
	 * the relay only handles REL_CLI_DOWNSTREAM_DATA
	 */
	case net.REL_CLI_DOWNSTREAM_DATA_UDP:
		err = prifi.Received_REL_CLI_UDP_DOWNSTREAM_DATA(typedMsg.REL_CLI_DOWNSTREAM_DATA)
	case net.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG:
		err = prifi.Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG(typedMsg)
	case net.REL_CLI_TELL_TRUSTEES_PK:
		err = prifi.Received_REL_CLI_TELL_TRUSTEES_PK(typedMsg)
	case net.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE:
		err = prifi.Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE(typedMsg)
	case net.REL_TRU_TELL_TRANSCRIPT:
		err = prifi.Received_REL_TRU_TELL_TRANSCRIPT(typedMsg)
	case net.REL_TRU_TELL_RATE_CHANGE:
		err = prifi.Received_REL_TRU_TELL_RATE_CHANGE(typedMsg)
	default:
		panic("unrecognized message !")
	}

	//no need to push the error further up. display it here !
	if err != nil {
		log.Error("ReceivedMessage: got an error, " + err.Error())
		return err
	}

	return nil
}
