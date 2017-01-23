package prifi_lib

import (
	"errors"
	"gopkg.in/dedis/onet.v1/log"
	"testing"
)

/**
 * Message Sender
 */
type TestMessageSender struct {
}

func (t *TestMessageSender) SendToClient(i int, msg interface{}) error {
	return errors.New("not implemented")
}
func (t *TestMessageSender) SendToTrustee(i int, msg interface{}) error {
	return errors.New("not implemented")
}
func (t *TestMessageSender) SendToRelay(msg interface{}) error {
	return errors.New("not implemented")
}
func (t *TestMessageSender) BroadcastToAllClients(msg interface{}) error {
	return errors.New("not implemented")
}
func (t *TestMessageSender) ClientSubscribeToBroadcast(clientName string, messageReceived func(interface{}) error, startStopChan chan bool) error {
	return errors.New("not implemented")
}

func TestPrifi(t *testing.T) {

	msgSender := new(TestMessageSender)
	in := make(chan []byte, 6)
	out := make(chan []byte, 3)

	client0 := NewPriFiClient(true, true, in, out, msgSender)
	client1 := NewPriFiClient(true, true, in, out, msgSender)

	timeoutHandler := func(clients, trustees []int) { log.Error(clients, trustees) }
	resultChan := make(chan interface{}, 1)

	relay := NewPriFiRelay(true, in, out, resultChan, timeoutHandler, msgSender)

	trustee0 := NewPriFiTrustee(msgSender)
	trustee1 := NewPriFiTrustee(msgSender)

	//TODO : emulate network connectivity, and run for a few rounds
	client0.ReceivedMessage("someMessage")
	_ = client0
	_ = client1
	_ = relay
	_ = trustee0
	_ = trustee1
}
