package relay

import (
	"errors"
	"github.com/dedis/cothority/log"
	"github.com/lbarman/prifi/prifi-lib/crypto"
	"github.com/lbarman/prifi/prifi-lib/net"
	"testing"
)

/**
 * Message Sender
 */
type TestMessageSender struct {
}

func (t *TestMessageSender) SendToClient(i int, msg interface{}) error {
	sentToClient = append(sentToClient, msg)
	return nil
}
func (t *TestMessageSender) SendToTrustee(i int, msg interface{}) error {
	sentToTrustee = append(sentToTrustee, msg)
	return nil
}

var sentToClient []interface{}
var sentToTrustee []interface{}

func (t *TestMessageSender) SendToRelay(msg interface{}) error {
	return errors.New("Relay sending to relay !?")
}
func (t *TestMessageSender) BroadcastToAllClients(msg interface{}) error {
	return t.SendToClient(0, msg)
}
func (t *TestMessageSender) ClientSubscribeToBroadcast(clientName string, messageReceived func(interface{}) error, startStopChan chan bool) error {
	return errors.New("Not for relay")
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

func getClientMessage(wantedMessage string) (interface{}, error) {
	return getMessage(&sentToClient, wantedMessage)
}
func getTrusteeMessage(wantedMessage string) (interface{}, error) {
	return getMessage(&sentToTrustee, wantedMessage)
}

func getMessage(bufferPtr *[]interface{}, wantedMessage string) (interface{}, error) {
	buffer := *bufferPtr
	if buffer == nil {
		panic("Buffer nil")
	}
	if len(buffer) == 0 {
		return nil, errors.New("Tried to fetch a " + wantedMessage + "but buffer is empty.")
	}
	msg := buffer[0]
	*bufferPtr = buffer[1:]
	return msg, nil
}

func TestClient(t *testing.T) {

	timeoutHandler := func(clients, trustees []int) { log.Error(clients, trustees) }
	resultChan := make(chan interface{}, 1)

	msgSender := new(TestMessageSender)
	msw := newTestMessageSenderWrapper(msgSender)
	sentToClient = make([]interface{}, 0)
	sentToTrustee = make([]interface{}, 0)
	dataForClients := make(chan []byte, 6)
	dataFromDCNet := make(chan []byte, 3)

	relay := NewRelay(true, dataForClients, dataFromDCNet, resultChan, timeoutHandler, msw)

	//when receiving no message, client should have some parameters ready
	rs := relay.relayState
	if rs.DataOutputEnabled != true {
		t.Error("DataOutputEnabled was not set correctly")
	}
	if rs.DataFromDCNet == nil {
		t.Error("DataFromDCNet was not set correctly")
	}
	if rs.DataForClients == nil {
		t.Error("DataForClients was not set correctly")
	}
	if rs.timeoutHandler == nil {
		t.Error("timeoutHandler was not set correctly")
	}
	if rs.ExperimentResultChannel == nil {
		t.Error("ExperimentResultChannel was not set correctly")
	}
	if rs.PriorityDataForClients == nil {
		t.Error("PriorityDataForClients was not set correctly")
	}
	if rs.CellCoder == nil {
		t.Error("CellCoder should have been created")
	}
	if relay.stateMachine.State() != "BEFORE_INIT" {
		t.Error("State was not set correctly")
	}
	if rs.privateKey == nil || rs.PublicKey == nil {
		t.Error("Private/Public key not set")
	}
	if rs.statistics == nil {
		t.Error("Statistics should have been set")
	}

	//we start by receiving a ALL_ALL_PARAMETERS from relay
	msg := new(net.ALL_ALL_PARAMETERS_NEW)
	msg.ForceParams = true
	nClients := 1
	nTrustees := 1
	upCellSize := 1500
	msg.Add("StartNow", true)
	msg.Add("NClients", nClients)
	msg.Add("NTrustees", nTrustees)
	msg.Add("UpstreamCellSize", upCellSize)
	msg.Add("DownstreamCellSize", 10*upCellSize)
	msg.Add("WindowSize", 1)
	msg.Add("UseUDP", true)
	msg.Add("UseDummyDataDown", true)
	msg.Add("ExperimentRoundLimit", -1)

	if err := relay.ReceivedMessage(msg); err != nil {
		t.Error("Relay should be able to receive this message, but", err)
	}
	if rs.nClients != nClients {
		t.Error("nClients was not set correctly")
	}
	if rs.nTrustees != nTrustees {
		t.Error("nTrustees was not set correctly")
	}
	if rs.nTrusteesPkCollected != 0 {
		t.Error("nTrusteesPkCollected was not set correctly")
	}
	if rs.nClientsPkCollected != 0 {
		t.Error("nClientsPkCollected was not set correctly")
	}
	if rs.ExperimentRoundLimit != -1 {
		t.Error("ExperimentRoundLimit was not set correctly")
	}
	if rs.UpstreamCellSize != upCellSize {
		t.Error("UpstreamCellSize was not set correctly")
	}
	if rs.DownstreamCellSize != 10*upCellSize {
		t.Error("DownstreamCellSize was not set correctly")
	}
	if rs.UseDummyDataDown != true {
		t.Error("UseDummyDataDown was not set correctly")
	}
	if rs.UseUDP != true {
		t.Error("UseUDP was not set correctly")
	}
	if rs.nextDownStreamRoundToSend != 1 {
		t.Error("nextDownStreamRoundToSend was not set correctly; it should be equal to 1 since round 0 is a half-round, and does not contain downstream data from relay")
	}
	if rs.WindowSize != 1 {
		t.Error("WindowSize was not set correctly")
	}
	if rs.bufferManager == nil {
		t.Error("bufferManager was not set correctly")
	}
	if rs.bufferManager.resumeFunction == nil {
		t.Error("bufferManager.resumeFunction was not set correctly")
	}
	if rs.bufferManager.stopFunction == nil {
		t.Error("bufferManager.stopFunction was not set correctly")
	}
	if relay.stateMachine.State() != "COLLECTING_TRUSTEES_PKS" {
		t.Error("In wrong state ! we should be in COLLECTING_TRUSTEES_PKS, but are in ", relay.stateMachine.State())
	}

	// should send ALL_ALL_PARAMETERS to clients
	msg2, err := getClientMessage("ALL_ALL_PARAMETERS")
	if err != nil {
		t.Error(err)
	}
	msg3 := msg2.(*net.ALL_ALL_PARAMETERS_NEW)

	if msg3.ParamsInt["NClients"] != nClients {
		t.Error("nClients not set correctly")
	}
	if msg3.ParamsInt["NTrustees"] != nTrustees {
		t.Error("nTrustees not set correctly")
	}
	if msg3.ParamsBool["StartNow"] != true {
		t.Error("StartNow not set correctly")
	}
	if msg3.ParamsInt["UpstreamCellSize"] != upCellSize {
		t.Error("UpstreamCellSize not set correctly")
	}
	if msg3.ParamsInt["NextFreeClientID"] != 0 {
		t.Error("NextFreeTrusteeID not set correctly")
	}

	// should send ALL_ALL_PARAMETERS to trustees
	msg4, err := getTrusteeMessage("ALL_ALL_PARAMETERS")
	if err != nil {
		t.Error(err)
	}
	msg5 := msg4.(*net.ALL_ALL_PARAMETERS_NEW)

	if msg5.ParamsInt["NClients"] != nClients {
		t.Error("nClients not set correctly")
	}
	if msg5.ParamsInt["NTrustees"] != nTrustees {
		t.Error("nTrustees not set correctly")
	}
	if msg5.ParamsBool["StartNow"] != true {
		t.Error("StartNow not set correctly")
	}
	if msg5.ParamsInt["UpstreamCellSize"] != upCellSize {
		t.Error("UpstreamCellSize not set correctly")
	}
	if msg5.ParamsInt["NextFreeTrusteeID"] != 0 {
		t.Error("NextFreeTrusteeID not set correctly")
	}

	//since startNow = true, trustee sends TRU_REL_TELL_PK
	trusteePub, trusteePriv := crypto.NewKeyPair()
	_ = trusteePriv
	msg6 := net.TRU_REL_TELL_PK{
		TrusteeID: 0,
		Pk:        trusteePub,
	}
	if err := relay.ReceivedMessage(msg6); err != nil {
		t.Error("Relay should be able to receive this message, but", err)
	}
	if relay.stateMachine.State() != "COLLECTING_CLIENT_PKS" {
		t.Error("In wrong state ! we should be in COLLECTING_CLIENT_PKS, but are in ", relay.stateMachine.State())
	}

	// should send REL_CLI_TELL_TRUSTEES_PK to clients
	msg7, err := getClientMessage("REL_CLI_TELL_TRUSTEES_PK")
	if err != nil {
		t.Error(err)
	}
	msg8 := msg7.(*net.REL_CLI_TELL_TRUSTEES_PK)

	if !msg8.Pks[0].Equal(trusteePub) {
		t.Error("Relay sent wrong public key")
	}

	//should receive a CLI_REL_TELL_PK_AND_EPH_PK
	cliPub, cliPriv := crypto.NewKeyPair()
	cliEphPub, cliEphPriv := crypto.NewKeyPair()
	_ = cliPriv
	_ = cliEphPriv
	msg9 := net.CLI_REL_TELL_PK_AND_EPH_PK{
		ClientID: 0,
		Pk:       cliPub,
		EphPk:    cliEphPub,
	}
	if err := relay.ReceivedMessage(msg9); err != nil {
		t.Error("Relay should be able to receive this message, but", err)
	}
	if relay.stateMachine.State() != "COLLECTING_SHUFFLES" {
		t.Error("In wrong state ! we should be in COLLECTING_SHUFFLES, but are in ", relay.stateMachine.State())
	}

	// should send REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE to clients
	msg10, err := getTrusteeMessage("REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE")
	if err != nil {
		t.Error(err)
	}
	msg11 := msg10.(*net.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE)

	if !msg11.EphPks[0].Equal(cliEphPub) {
		t.Error("Relay sent wrong ephemeral public key")
	}

	//should receive a TRU_REL_TELL_NEW_BASE_AND_EPH_PKS

	msg12 := net.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS{
		NewBase:   msg11.Base,
		NewEphPks: msg11.EphPks,
		Proof:     make([]byte, 50),
	}

	if err := relay.ReceivedMessage(msg12); err != nil {
		t.Error("Relay should be able to receive this message, but", err)
	}

	if relay.stateMachine.State() != "COLLECTING_SHUFFLE_SIGNATURES" {
		t.Error("In wrong state ! we should be in COLLECTING_SHUFFLE_SIGNATURES, but are in ", relay.stateMachine.State())
	}

	// should send REL_TRU_TELL_TRANSCRIPT to clients
	msg13, err := getTrusteeMessage("REL_TRU_TELL_TRANSCRIPT")
	if err != nil {
		t.Error(err)
	}
	_ = msg13.(*net.REL_TRU_TELL_TRANSCRIPT)

}
