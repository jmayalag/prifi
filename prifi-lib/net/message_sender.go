package net

import (
	"errors"
	"reflect"
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
	ClientSubscribeToBroadcast(clientName string, messageReceived func(interface{}) error, startStopChan chan bool) error
}

/**
 * A wrapper around a messageSender. will automatically print what it does (logFunction) if loggingEnabled, and
 * will call networkErrorHappened on error
 */
type MessageSenderWrapper struct {
	MessageSender
	loggingEnabled       bool
	logSuccessFunction   func(interface{})
	logErrorFunction     func(interface{})
	networkErrorHappened func(error)
}

/**
 * Creates a wrapper around a messageSender. will automatically print what it does (logFunction) if loggingEnabled, and
 * will call networkErrorHappened on error
 */
func NewMessageSenderWrapper(logging bool, logSuccessFunction func(interface{}), logErrorFunction func(interface{}), networkErrorHappened func(error), ms MessageSender) (*MessageSenderWrapper, error) {
	if logging && logSuccessFunction == nil {
		return nil, errors.New("Can't create a MessageSenderWrapper without logFunction if logging is enabled")
	}
	if logging && logErrorFunction == nil {
		return nil, errors.New("Can't create a MessageSenderWrapper without logFunction if logging is enabled")
	}
	if networkErrorHappened == nil {
		return nil, errors.New("Can't create a MessageSenderWrapper without networkErrorHappened. If you don't need error handling, set it to func(e error){}.")
	}
	if ms == nil {
		return nil, errors.New("Can't create a MessageSenderWrapper without messageSender.")
	}

	msw := &MessageSenderWrapper{
		loggingEnabled:       logging,
		logSuccessFunction:   logSuccessFunction,
		logErrorFunction:     logErrorFunction,
		networkErrorHappened: networkErrorHappened,
		MessageSender:        ms,
	}

	return msw, nil
}

/**
 * Send a message to client i. will automatically print what it does (Lvl3) if loggingenabled, and
 * will call networkErrorHappened on error
 */
func (m *MessageSenderWrapper) SendToClientWithLog(i int, msg interface{}, extraInfos string) bool {
	return m.sendToWithLog(m.MessageSender.SendToClient, i, msg, extraInfos)
}

/**
 * Send a message to trustee i. will automatically print what it does (Lvl3) if loggingenabled, and
 * will call networkErrorHappened on error
 */
func (m *MessageSenderWrapper) SendToTrusteeWithLog(i int, msg interface{}, extraInfos string) bool {
	return m.sendToWithLog(m.MessageSender.SendToTrustee, i, msg, extraInfos)
}

/**
 * Send a message to the relay. will automatically print what it does (Lvl3) if loggingenabled, and
 * will call networkErrorHappened on error
 */
func (m *MessageSenderWrapper) SendToRelayWithLog(msg interface{}, extraInfos string) bool {
	err := m.MessageSender.SendToRelay(msg)
	msgName := reflect.TypeOf(msg).String()
	if err != nil {
		e := "Tried to send a " + msgName + ", but some network error occurred. Err is: " + err.Error()
		if m.networkErrorHappened != nil {
			m.networkErrorHappened(errors.New(e))
		}
		if m.loggingEnabled {
			m.logErrorFunction(e + extraInfos)
		}
		return false
	}

	if m.loggingEnabled {
		m.logSuccessFunction("Sent a " + msgName + "." + extraInfos)
	}
	return true
}

/**
 * Helper function for both SendToClientWithLog and SendToTrusteeWithLog
 */
func (m *MessageSenderWrapper) sendToWithLog(sendingFunc func(int, interface{}) error, i int, msg interface{}, extraInfos string) bool {
	err := sendingFunc(i, msg)
	msgName := reflect.TypeOf(msg).String()
	if err != nil {
		e := "Tried to send a " + msgName + ", but some network error occurred. Err is: " + err.Error()
		if m.networkErrorHappened != nil {
			m.networkErrorHappened(errors.New(e))
		}
		if m.loggingEnabled {
			m.logErrorFunction(e + extraInfos)
		}
		return false
	}

	if m.loggingEnabled {
		m.logSuccessFunction("Sent a " + msgName + "." + extraInfos)
	}
	return true
}
