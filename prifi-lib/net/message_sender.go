package net

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
