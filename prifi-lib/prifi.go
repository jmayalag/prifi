package prifi_lib

import (
	"github.com/dedis/cothority/log"
	"github.com/lbarman/prifi/prifi-lib/client"
	"github.com/lbarman/prifi/prifi-lib/net"
	"github.com/lbarman/prifi/prifi-lib/relay"
	"github.com/lbarman/prifi/prifi-lib/trustee"
	"os"
)

/*
PriFi - Library
***************
This is a network-agnostic PriFi library. Feed it with a MessageSender interface (that knows how to contact the different entities),
and call ReceivedMessage(msg) with the received messages.
Then, it runs the PriFi anonymous communication network among those entities.
*/

// PriFiLibInstance contains the mutable state of a PriFi entity.
type PriFiLibInstance struct { //todo remove this, like it was done for client
	role                   int16
	messageSender          *net.MessageSenderWrapper
	specializedLibInstance SpecializedLibInstance
}

//Prifi's "Relay", "Client" and "Trustee" instance all can receive a message
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
func NewPriFiClient(doLatencyTest bool, dataOutputEnabled bool, dataForDCNet chan []byte, dataFromDCNet chan []byte, msgSender net.MessageSender) SpecializedLibInstance {
	msw := newMessageSenderWrapper(msgSender)
	c := client.NewClient(doLatencyTest, dataOutputEnabled, dataForDCNet, dataFromDCNet, msw)
	return c
}

//Creates a new PriFi relay //todo do like client
func NewPriFiRelay(msgSender net.MessageSender) *PriFiLibInstance {
	prifi := PriFiLibInstance{
		role: PRIFI_ROLE_RELAY,
		specializedLibInstance: relay.NewPriFiRelay(newMessageSenderWrapper(msgSender)),
	}
	return &prifi
}

//Creates a new PriFi trustee //todo do like client
func NewPriFiRelayWithState(msgSender net.MessageSender, state *relay.RelayState) *PriFiLibInstance {
	prifi := PriFiLibInstance{
		role: PRIFI_ROLE_RELAY,
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

// NewPriFiTrusteeWithState creates a new PriFi trustee entity state.
func NewPriFiTrusteeWithState(msgSender net.MessageSender, state *trustee.TrusteeState) *PriFiLibInstance {
	prifi := PriFiLibInstance{
		role: PRIFI_ROLE_TRUSTEE,
		specializedLibInstance: trustee.NewPriFiTrusteeWithState(newMessageSenderWrapper(msgSender), state),
	}
	return &prifi
}

// ReceivedMessage must be called when a PriFi host receives a message.
// It takes care to call the correct message handler function.
func (p *PriFiLibInstance) ReceivedMessage(msg interface{}) error {
	err := p.specializedLibInstance.ReceivedMessage(msg)
	if err != nil {
		log.Error(err)
		os.Exit(1) //todo we can cleanly call shutdown, and keep the service
	}
	return nil
}
