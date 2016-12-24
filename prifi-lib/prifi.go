package prifi

import (
	"strconv"

	"github.com/dedis/cothority/log"
)

/*
PriFi - Library
***************
This is a network-agnostic PriFi library. Feed it with a MessageSender interface (that knows how to contact the different entities),
and call ReceivedMessage(msg) with the received messages.
Then, it runs the PriFi anonymous communication network among those entities.
*/

// PriFiProtocol contains the mutable state of a PriFi entity.
type PriFiProtocol struct {
	role          int16
	messageSender MessageSender
	// TODO: combine states into a single interface
	clientState  ClientState  //only one of those will be set
	relayState   RelayState   //only one of those will be set
	trusteeState TrusteeState //only one of those will be set
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

// MessageSender is the interface that abstracts the network
// interactions.
type MessageSender interface {
	// SendToClient tries to deliver the message "msg" to the client i.
	SendToClient(i int, msg interface{}) error

	// SendToTrustee tries to deliver the message "msg" to the trustee i.
	SendToTrustee(i int, msg interface{}) error

	// SendToRelay tries to deliver the message "msg" to the relay.
	SendToRelay(msg interface{}) error

	/*
		BroadcastToAllClients tries to deliver the message "msg"
		to every client, possibly using broadcast.
	*/
	BroadcastToAllClients(msg interface{}) error

	/*
		ClientSubscribeToBroadcast should be called by the Clients
		in order to receive the Broadcast messages.
		Calling the function starts the handler but does not actually
		listen for broadcast messages.
		Sending true to startStopChan starts receiving the broadcasts.
		Sending false to startStopChan stops receiving the broadcasts.
	*/
	ClientSubscribeToBroadcast(clientName string, protocolInstance *PriFiProtocol, startStopChan chan bool) error
}

/*
call the functions below on the appropriate machine on the network.
if you call *without state* (one of the first 3 methods), IT IS NOT SUFFICIENT FOR PRIFI to start; this entity will expect a ALL_ALL_PARAMETERS as a
first message to finish initializing itself (this is handy if only the Relay has access to the configuration file).
Otherwise, the 3 last methods fully initialize the entity.
*/

// NewPriFiRelay creates a new PriFi relay entity state.
// Note: the returned state is not sufficient for the PrFi protocol
// to start; this entity will expect a ALL_ALL_PARAMETERS message as
// first received message to complete it's state.
func NewPriFiRelay(msgSender MessageSender) *PriFiProtocol {
	prifi := PriFiProtocol{
		role:          PRIFI_ROLE_RELAY,
		messageSender: msgSender,
	}

	return &prifi
}

// NewPriFiClient creates a new PriFi client entity state.
// Note: the returned state is not sufficient for the PrFi protocol
// to start; this entity will expect a ALL_ALL_PARAMETERS message as
// first received message to complete it's state.
func NewPriFiClient(msgSender MessageSender) *PriFiProtocol {
	prifi := PriFiProtocol{
		role:          PRIFI_ROLE_CLIENT,
		messageSender: msgSender,
	}
	return &prifi
}

// NewPriFiTrustee creates a new PriFi trustee entity state.
// Note: the returned state is not sufficient for the PrFi protocol
// to start; this entity will expect a ALL_ALL_PARAMETERS message as
// first received message to complete it's state.
func NewPriFiTrustee(msgSender MessageSender) *PriFiProtocol {
	prifi := PriFiProtocol{
		role:          PRIFI_ROLE_TRUSTEE,
		messageSender: msgSender,
	}
	return &prifi
}

// NewPriFiRelayWithState creates a new PriFi relay entity state.
func NewPriFiRelayWithState(msgSender MessageSender, state *RelayState) *PriFiProtocol {
	prifi := PriFiProtocol{
		role:          PRIFI_ROLE_RELAY,
		messageSender: msgSender,
		relayState:    *state,
	}

	log.Lvl1("Relay has been initialized by function call. ")
	return &prifi
}

// NewPriFiClientWithState creates a new PriFi client entity state.
func NewPriFiClientWithState(msgSender MessageSender, state *ClientState) *PriFiProtocol {
	prifi := PriFiProtocol{
		role:          PRIFI_ROLE_CLIENT,
		messageSender: msgSender,
		clientState:   *state,
	}
	log.Lvl1("Client has been initialized by function call. ")

	log.Lvl2("Client " + strconv.Itoa(prifi.clientState.Id) + " : starting the broadcast-listener goroutine")
	go prifi.messageSender.ClientSubscribeToBroadcast(prifi.clientState.Name, &prifi, prifi.clientState.StartStopReceiveBroadcast)
	return &prifi
}

// NewPriFiTrusteeWithState creates a new PriFi trustee entity state.
func NewPriFiTrusteeWithState(msgSender MessageSender, state *TrusteeState) *PriFiProtocol {
	prifi := PriFiProtocol{
		role:          PRIFI_ROLE_TRUSTEE,
		messageSender: msgSender,
		trusteeState:  *state,
	}

	log.Lvl1("Trustee has been initialized by function call. ")
	return &prifi
}

// WhoAmI prints a description of the state of the PriFi entity
// on which it is called.
func (prifi *PriFiProtocol) WhoAmI() {

	log.Print("###################### WHO AM I ######################")
	if prifi.role == PRIFI_ROLE_RELAY {
		log.Print("I' a relay, my name is ", prifi.relayState.Name)
		log.Printf("%+v\n", prifi.relayState)
		//log.Print("I'm not : ")
		//log.Printf("%+v\n", prifi.clientState)
		//log.Printf("%+v\n", prifi.trusteeState)
	} else if prifi.role == PRIFI_ROLE_CLIENT {
		log.Print("I' a client, my name is ", prifi.clientState.Name)
		log.Printf("%+v\n", prifi.clientState)
		//log.Print("I'm not : ")
		//log.Printf("%+v\n", prifi.relayState)
		//log.Printf("%+v\n", prifi.trusteeState)
	} else if prifi.role == PRIFI_ROLE_TRUSTEE {
		log.Print("I' a trustee, my name is ", prifi.trusteeState.Name)
		log.Printf("%+v\n", prifi.trusteeState)
		//log.Print("I'm not : ")
		//log.Printf("%+v\n", prifi.clientState)
		//log.Printf("%+v\n", prifi.relayState)
	}
	log.Print("###################### -------- ######################")
}
