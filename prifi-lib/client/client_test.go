package client

import (
	"testing"
	"errors"
	"github.com/lbarman/prifi/prifi-lib/net"
	"github.com/dedis/cothority/log"
)

/**
 * Message Sender
 */
type TestMessageSender struct {
}

func (t *TestMessageSender) SendToClient(i int, msg interface{}) error {
	return errors.New("Clients should never sent to other clients")
}
func (t *TestMessageSender) SendToTrustee(i int, msg interface{}) error {
	return errors.New("Clients should never sent to other trustees")
}
var sentToRelay []interface{}
func (t *TestMessageSender) SendToRelay(msg interface{}) error {
	sentToRelay = append(sentToRelay, msg)
	return nil
}
func (t *TestMessageSender) BroadcastToAllClients(msg interface{}) error {
	return errors.New("Clients should never sent to other clients")
}
func (t *TestMessageSender) ClientSubscribeToBroadcast(clientName string, messageReceived func(interface{}) error, startStopChan chan bool) error {
	return nil
}

/**
 * Message Sender Wrapper
 */

func newTestMessageSenderWrapper(msgSender net.MessageSender) *net.MessageSenderWrapper {

	errHandling := func(e error) {}
	loggingSuccessFunction := func(e interface{}) { log.Lvl3(e) }
	loggingErrorFunction := func(e interface{}) { log.Error(e) }

	msw, err := net.NewMessageSenderWrapper(true, loggingSuccessFunction, loggingErrorFunction, errHandling, msgSender)
	if err != nil {
		log.Fatal("Could not create a MessageSenderWrapper, error is", err)
	}
	return msw
}

func TestPrifi(t *testing.T) {

	msgSender := new(TestMessageSender)
	msw := newTestMessageSenderWrapper(msgSender)
	sentToRelay = make([]interface{}, 0)
	in := make(chan []byte)
	out := make(chan []byte)

	client := NewClient(true, true, in, out, msw)

	//when receiving no message, client should have some parameters ready
	cs := client.clientState
	if cs.DataOutputEnabled != true {
		t.Error("DataOutputEnabled was not set correctly")
	}
	if cs.CellCoder == nil {
		t.Error("CellCoder should have been created")
	}
	if cs.currentState != CLIENT_STATE_BEFORE_INIT {
		t.Error("State was not set correctly")
	}
	if cs.privateKey == nil || cs.PublicKey == nil {
		t.Error("Private/Public key not set")
	}
	if cs.statistics == nil {
		t.Error("Statistics should have been set")
	}
}
