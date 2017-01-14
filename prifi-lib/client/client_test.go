package client

import (
	"errors"
	"github.com/dedis/cothority/log"
	"github.com/lbarman/prifi/prifi-lib/net"
	"testing"
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
	if cs.StartStopReceiveBroadcast != nil {
		t.Error("StartStopReceiveBroadcast should *not* have been set")
	}

	//we start by receiving a ALL_ALL_PARAMETERS from relay
	b := net.NewALL_ALL_PARAMETERS_BUILDER()
	b.Add("NClients", 3)
	b.Add("NTrustees", 2)
	b.Add("UpCellSize", 1500)
	b.Add("NextFreeClientID", 3)
	b.Add("UseUDP", true)
	msg := b.BuildMessage(true)

	if err := client.ReceivedMessage(*msg); err != nil {
		t.Error("Client should be able to receive this message:", err)
	}

	if cs.nClients != 3 {
		t.Error("NClients should be 3")
	}
	if cs.nTrustees != 2 {
		t.Error("nTrustees should be 2")
	}
	if cs.PayloadLength != 1500 {
		t.Error("PayloadLength should be 1500")
	}
	if cs.ID != 3 {
		t.Error("ID should be 3")
	}
	if cs.RoundNo != 0 {
		t.Error("RoundNumber should be 0")
	}
	if cs.StartStopReceiveBroadcast == nil {
		t.Error("StartStopReceiveBroadcast should now have been set")
	}
	if cs.UseUDP != true {
		t.Error("UseUDP should now have been set to true")
	}
	if len(cs.TrusteePublicKey) != 2 {
		t.Error("Len(TrusteePKs) should be equal to NTrustees")
	}
	if len(cs.sharedSecrets) != 2 {
		t.Error("Len(SharedSecrets) should be equal to NTrustees")
	}
	if cs.currentState != CLIENT_STATE_INITIALIZING {
		t.Error("Client should be in state CLIENT_STATE_INITIALIZING")
	}

	//Should receive a Received_REL_CLI_TELL_TRUSTEES_PK

	t.SkipNow() //we started a goroutine, let's kill everything, we're good
}
