package trustee

import (
	"errors"
	"testing"

	"github.com/dedis/prifi/prifi-lib/config"
	"github.com/dedis/prifi/prifi-lib/crypto"
	"github.com/dedis/prifi/prifi-lib/net"
	"github.com/dedis/prifi/prifi-lib/scheduler"
	"gopkg.in/dedis/kyber.v2"
	"gopkg.in/dedis/onet.v2/log"
	"time"
)

/**
 * Message Sender
 */
type TestMessageSender struct {
	sentToRelay chan interface{}
}

func (t *TestMessageSender) SendToClient(i int, msg interface{}) error {
	return errors.New("Trustees should never sent to clients")
}
func (t *TestMessageSender) SendToTrustee(i int, msg interface{}) error {
	return errors.New("Trustees should never sent to other trustees")
}

func (t *TestMessageSender) SendToRelay(msg interface{}) error {
	t.sentToRelay <- msg
	return nil
}
func (t *TestMessageSender) BroadcastToAllClients(msg interface{}) error {
	return errors.New("Clients should never sent to other clients")
}
func (t *TestMessageSender) ClientSubscribeToBroadcast(clientID int, messageReceived func(interface{}) error, startStopChan chan bool) error {
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

func TestTrustee(t *testing.T) {

	msgSender := new(TestMessageSender)
	msgSender.sentToRelay = make(chan interface{}, 15)
	msw := newTestMessageSenderWrapper(msgSender)
	neverSlowDown := false
	alwaysSlowDown := false
	baseSleepTime := 1000
	trustee := NewTrustee(neverSlowDown, alwaysSlowDown, baseSleepTime, msw)

	ts := trustee.trusteeState
	if ts.sendingRate == nil {
		t.Error("sendingRate should not be nil")
	}
	if trustee.stateMachine.State() != "BEFORE_INIT" {
		t.Error("State was not set correctly")
	}
	if ts.privateKey == nil || ts.PublicKey == nil {
		t.Error("Private/Public key not set")
	}
	if ts.neffShuffle == nil {
		t.Error("NeffShuffle should not be nil")
	}

	//should not be able to receive those weird messages
	weird := new(net.ALL_ALL_PARAMETERS)
	weird.Add("NextFreeTrusteeID", -1)
	if err := trustee.ReceivedMessage(*weird); err == nil {
		t.Error("Trustee should not accept this message")
	}
	weird.Add("NextFreeTrusteeID", 0)
	weird.Add("NTrustees", 0)
	if err := trustee.ReceivedMessage(*weird); err == nil {
		t.Error("Trustee should not accept this message")
	}
	weird.Add("NTrustees", 1)
	weird.Add("NClients", 0)
	if err := trustee.ReceivedMessage(*weird); err == nil {
		t.Error("Trustee should not accept this message")
	}
	weird.Add("NClients", 1)
	weird.Add("PayloadSize", 0)
	if err := trustee.ReceivedMessage(*weird); err == nil {
		t.Error("Trustee should not accept this message")
	}

	//we start by receiving a ALL_ALL_PARAMETERS from relay
	msg := new(net.ALL_ALL_PARAMETERS)
	msg.ForceParams = true
	trusteeID := 3
	nClients := 3
	nTrustees := 2
	upCellSize := 1500
	dcNetType := "Simple"
	msg.Add("StartNow", true)
	msg.Add("NClients", nClients)
	msg.Add("NTrustees", nTrustees)
	msg.Add("PayloadSize", upCellSize)
	msg.Add("NextFreeTrusteeID", trusteeID)
	msg.Add("DCNetType", dcNetType)

	if err := trustee.ReceivedMessage(*msg); err != nil {
		t.Error("Trustee should be able to receive this message:", err)
	}

	if ts.nClients != 3 {
		t.Error("NClients should be 3")
	}
	if ts.nTrustees != nTrustees {
		t.Error("nTrustees should be 2")
	}
	if ts.PayloadSize != 1500 {
		t.Error("PayloadSize should be 1500")
	}
	if ts.ID != trusteeID {
		t.Error("ID should be 3")
	}
	if len(ts.ClientPublicKeys) != nClients {
		t.Error("Len(TrusteePKs) should be equal to NTrustees")
	}
	if len(ts.sharedSecrets) != nClients {
		t.Error("Len(SharedSecrets) should be equal to NTrustees")
	}
	if trustee.stateMachine.State() != "INITIALIZING" {
		t.Error("Trustee should be in state INITIALIZING")
	}

	//Should send a TRU_REL_TELL_PK
	select {
	case msg3 := <-msgSender.sentToRelay:
		msg3_parsed := msg3.(*net.TRU_REL_TELL_PK)
		if int(msg3_parsed.TrusteeID) != trusteeID {
			t.Error("Trustee sent a wrong trustee ID")
		}
		if !msg3_parsed.Pk.Equal(ts.PublicKey) {
			t.Error("Trustee did not send his public key")
		}
	default:
		t.Error("Trustee should have sent a TRU_REL_TELL_PK to the relay")
	}

	//do the shuffle
	n := new(scheduler.NeffShuffle)
	n.Init()
	n.RelayView.Init(1)

	clientPubKeys := make([]kyber.Point, nClients)
	clientPrivKeys := make([]kyber.Scalar, nClients)
	for i := 0; i < nClients; i++ {
		clientPubKeys[i], clientPrivKeys[i] = crypto.NewKeyPair()
		n.RelayView.AddClient(clientPubKeys[i])
	}
	toSend, _, err := n.RelayView.SendToNextTrustee()
	if err != nil {
		t.Error(err)
	}
	msg4 := toSend.(*net.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE)

	//we inject the public keys
	msg4.Pks = make([]kyber.Point, nClients)
	for i := 0; i < nClients; i++ {
		msg4.Pks[i] = clientPubKeys[i]
	}

	//we receive the shuffle
	if err := trustee.ReceivedMessage(*msg4); err != nil {
		t.Error("Trustee should be able to receive this message:", err)
	}

	for i := 0; i < nClients; i++ {
		if !ts.ClientPublicKeys[i].Equal(clientPubKeys[i]) {
			t.Error("Pub key", i, "has not been stored correctly")
		}
		myPrivKey := ts.privateKey
		if !ts.sharedSecrets[i].Equal(config.CryptoSuite.Point().Mul(myPrivKey, clientPubKeys[i])) {
			t.Error("Shared secret", i, "has not been computed correctly")
		}
	}

	if trustee.stateMachine.State() != "SHUFFLE_DONE" {
		t.Error("Trustee should be in state SHUFFLE DONE")
	}

	//Should have sent a TRU_REL_TELL_NEW_BASE_AND_EPH_PKS
	select {
	case msg5 := <-msgSender.sentToRelay:
		msg5_parsed := msg5.(*net.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS)
		_, err = n.RelayView.ReceivedShuffleFromTrustee(msg5_parsed.NewBase, msg5_parsed.NewEphPks, msg5_parsed.Proof)
		if err != nil {
			t.Error("This should be fine, yet", err)
		}
	default:
		t.Error("Trustee should have sent a TRU_REL_TELL_NEW_BASE_AND_EPH_PKS to the relay")
	}

	//should receive the transcript
	toSend3, err := n.RelayView.SendTranscript()
	if err != nil {
		t.Error(err)
	}
	msg6 := toSend3.(*net.REL_TRU_TELL_TRANSCRIPT)

	if err := trustee.ReceivedMessage(*msg6); err != nil {
		t.Error("Trustee should be able to receive this message:", err)
	}

	//should the signed shuffle
	select {
	case msgX := <-msgSender.sentToRelay:
		_ = msgX.(*net.TRU_REL_SHUFFLE_SIG)
	default:
		t.Error("Trustee should have sent a TRU_REL_SHUFFLE_SIG to the relay")
	}

	if trustee.stateMachine.State() != "READY" {
		t.Error("Trustee should be in state READY")
	}

	stopMsg := &net.REL_TRU_TELL_RATE_CHANGE{
		WindowCapacity: 0,
	}

	time.Sleep(time.Duration(baseSleepTime/2) * time.Millisecond) //just time for one message

	if err := trustee.ReceivedMessage(*stopMsg); err != nil {
		t.Error("Should handle this stop message, but", err)
	}

	//should have sent a few ciphers before getting the stop message
	select {
	case msg8 := <-msgSender.sentToRelay:
		msg8_parsed := msg8.(*net.TRU_REL_DC_CIPHER)

		if int(msg8_parsed.TrusteeID) != trusteeID {
			t.Error("TRU_REL_DC_CIPHER has the wrong trustee ID")
		}
		if msg8_parsed.RoundID != 0 {
			t.Error("TRU_REL_DC_CIPHER has the wrong round ID")
		}
		if len(msg8_parsed.Data) != upCellSize+8 {
			t.Error("TRU_REL_DC_CIPHER sent a payload with wrong size")
		}

	default:
		t.Fatal("Trustee should have sent a TRU_REL_DC_CIPHER to the relay")
	}

	time.Sleep(time.Duration(baseSleepTime*2) * time.Millisecond)

	//empty the chan
	empty := false
	for !empty {
		select {
		case <-msgSender.sentToRelay:
			//nothing
		default:
			empty = true
		}
	}

	time.Sleep(time.Duration(baseSleepTime*2) * time.Millisecond)

	select {
	case _ = <-msgSender.sentToRelay:
		//t.Error("Trustee should not have sent a TRU_REL_DC_CIPHER to the relay")
	default:
	}

	startMsg := &net.REL_TRU_TELL_RATE_CHANGE{
		WindowCapacity: 1,
	}

	time.Sleep(time.Duration(baseSleepTime) * time.Millisecond) //just time for one message

	if err := trustee.ReceivedMessage(*startMsg); err != nil {
		t.Error("Should handle this start message, but", err)
	}

	time.Sleep(time.Duration(baseSleepTime*2) * time.Millisecond)

	select {
	case msg8 := <-msgSender.sentToRelay:
		msg8_parsed := msg8.(*net.TRU_REL_DC_CIPHER)

		if int(msg8_parsed.TrusteeID) != trusteeID {
			t.Error("TRU_REL_DC_CIPHER has the wrong trustee ID")
		}
		if len(msg8_parsed.Data) != upCellSize+8 {
			t.Error("TRU_REL_DC_CIPHER sent a payload with wrong size")
		}

	default:
		t.Error("Trustee should have sent a TRU_REL_DC_CIPHER to the relay")
	}

	randomMsg := net.CLI_REL_TELL_PK_AND_EPH_PK{}
	if err := trustee.ReceivedMessage(randomMsg); err == nil {
		t.Error("Should not accept this CLI_REL_TELL_PK_AND_EPH_PK message")
	}

	shutdownMsg := net.ALL_ALL_SHUTDOWN{}
	if err := trustee.ReceivedMessage(shutdownMsg); err != nil {
		t.Error("Should handle this ALL_ALL_SHUTDOWN message, but", err)
	}
	if trustee.stateMachine.State() != "SHUTDOWN" {
		t.Error("Trustee should be in state SHUTDOWN")
	}

	t.SkipNow() //we started a goroutine, let's kill everything, we're good
}
