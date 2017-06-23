package prifi_lib

import (
	"errors"
	"github.com/lbarman/prifi/prifi-lib/client"
	"github.com/lbarman/prifi/prifi-lib/crypto"
	"github.com/lbarman/prifi/prifi-lib/net"
	"github.com/lbarman/prifi/prifi-lib/relay"
	"github.com/lbarman/prifi/prifi-lib/trustee"
	"gopkg.in/dedis/onet.v1/log"
	"testing"
)

/**
 * Message Sender
 */
type TestMessageSender struct {
}

func (t *TestMessageSender) SendToClient(i int, msg interface{}) error {
	return nil
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
func (t *TestMessageSender) ClientSubscribeToBroadcast(clientID int, messageReceived func(interface{}) (bool, interface{}, error), startStopChan chan bool) error {
	return errors.New("not implemented")
}

func TestPrifi(t *testing.T) {

	msgSender := new(TestMessageSender)
	msw := newMessageSenderWrapper(msgSender)
	in := make(chan []byte, 6)
	out := make(chan []byte, 3)

	client0 := NewPriFiClient(true, true, in, out, false, "./", msgSender)
	client1 := NewPriFiClient(true, true, in, out, false, "./", msgSender)
	client0_inst := client.NewClient(true, true, in, out, false, "./", msw)
	client1_inst := client.NewClient(true, true, in, out, false, "./", msw)
	client0.SetSpecializedLibInstance(client0_inst)
	client1.SetSpecializedLibInstance(client1_inst)
	client0.SetMessageSender(msgSender)
	client1.SetMessageSender(msgSender)

	timeoutHandler := func(clients, trustees []int) { log.Error(clients, trustees) }
	resultChan := make(chan interface{}, 1)

	relay0 := NewPriFiRelay(true, in, out, resultChan, timeoutHandler, msgSender)
	relay0_inst := relay.NewRelay(true, in, out, resultChan, timeoutHandler, msw)
	relay0.SetSpecializedLibInstance(relay0_inst)
	relay0.SetMessageSender(msgSender)

	trustee0 := NewPriFiTrustee(msgSender)
	trustee1 := NewPriFiTrustee(msgSender)
	trustee0_inst := trustee.NewTrustee(msw)
	trustee1_inst := trustee.NewTrustee(msw)
	trustee0.SetSpecializedLibInstance(trustee0_inst)
	trustee1.SetSpecializedLibInstance(trustee1_inst)
	trustee0.SetMessageSender(msgSender)
	trustee1.SetMessageSender(msgSender)

	//TODO : emulate network connectivity, and run for a few rounds

	//should trigger an error, not ready
	pub, _ := crypto.NewKeyPair()
	relay0.ReceivedMessage(net.TRU_REL_TELL_PK{Pk: pub})

	//emulate the reception of a ALL_ALL_PARAMETERS with StartNow=true
	msg := new(net.ALL_ALL_PARAMETERS_NEW)
	msg.Add("StartNow", true)
	msg.Add("NTrustees", 2)
	msg.Add("NClients", 2)
	msg.Add("UpstreamCellSize", 1000)
	msg.Add("DownstreamCellSize", 1000)
	msg.Add("WindowSize", 1)
	msg.Add("UseDummyDataDown", true)
	msg.Add("ExperimentRoundLimit", 10)
	msg.Add("UseUDP", false)
	msg.Add("DCNetType", "Simple")
	msg.ForceParams = true

	relay0.ReceivedMessage(*msg)
	_ = client0
	_ = client1
	_ = relay0
	_ = trustee0
	_ = trustee1
}
