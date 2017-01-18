package trustee

import (
	"errors"
	"testing"

	"github.com/dedis/cothority/log"
	"github.com/lbarman/prifi/prifi-lib/net"
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
	trustee := NewTrustee(msw)

	_ = trustee

	t.SkipNow() //we started a goroutine, let's kill everything, we're good
}
