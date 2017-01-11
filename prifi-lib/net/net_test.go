package net

import (
	"bytes"
	"errors"
	"github.com/dedis/crypto/random"
	"testing"
)

type TestMessageSender struct {
}

func (t *TestMessageSender) SendToClient(i int, msg interface{}) error {
	return errors.New("Tried to send to client!")
}
func (t *TestMessageSender) SendToTrustee(i int, msg interface{}) error {
	return nil
}
func (t *TestMessageSender) SendToRelay(msg interface{}) error {
	if msg == "trigger-error" {
		return errors.New("error triggered")
	}
	return nil
}
func (t *TestMessageSender) BroadcastToAllClients(msg interface{}) error {
	return nil
}
func (t *TestMessageSender) ClientSubscribeToBroadcast(clientName string, messageReceived func(interface{}) error, startStopChan chan bool) error {
	return nil
}

func TestMessageSenderWrapper(t *testing.T) {

	_, err := NewMessageSenderWrapper(true, nil, nil, nil, nil)
	if err == nil {
		t.Error("If logging=true, should provide a logging function")
	}

	_, err = NewMessageSenderWrapper(false, nil, nil, nil, nil)
	if err == nil {
		t.Error("Should provide a error handling function")
	}

	errHandling := func(e error) {}
	_, err = NewMessageSenderWrapper(false, nil, nil, errHandling, nil)
	if err == nil {
		t.Error("Should provide a messageSender")
	}

	//test the real stuff
	var errorHandlerCalled bool = false
	errHandling = func(e error) { errorHandlerCalled = true }
	msgSender := new(TestMessageSender)
	msw, err := NewMessageSenderWrapper(false, nil, nil, errHandling, msgSender)
	if err != nil {
		t.Error("Should be able to create a MessageSenderWrapper")
	}

	success := msw.SendToTrusteeWithLog(0, "hello", "")
	if !success || errorHandlerCalled {
		t.Error("this call should not trigger an error")
	}
	var loggingFunctionCalled bool = false
	logging := func(e interface{}) { loggingFunctionCalled = true }
	msw, err = NewMessageSenderWrapper(true, logging, logging, errHandling, msgSender)
	success = msw.SendToClientWithLog(0, "hello", "")
	if success || !errorHandlerCalled {
		t.Error("this call should trigger an error")
	}
}

func TestMessageSenderWrapperRelay(t *testing.T) {

	//test the real stuff
	var errorHandlerCalled bool = false
	var loggingFunctionCalled bool = false

	errHandling := func(e error) { errorHandlerCalled = true }
	logging := func(e interface{}) { loggingFunctionCalled = true }

	msgSender := new(TestMessageSender)
	msw, err := NewMessageSenderWrapper(true, logging, logging, errHandling, msgSender)
	if err != nil {
		t.Error("Should be able to create a MessageSenderWrapper")
	}

	success := msw.SendToRelayWithLog("hello", "")
	if !success || errorHandlerCalled {
		t.Error("this call should not trigger an error")
	}
	success = msw.SendToRelayWithLog("trigger-error", "")
	if success || !errorHandlerCalled {
		t.Error("this call should trigger an error")
	}
}

func TestUDPMessage(t *testing.T) {

	msg := new(REL_CLI_DOWNSTREAM_DATA_UDP)

	//random content
	content := new(REL_CLI_DOWNSTREAM_DATA)
	content.RoundID = 1
	content.FlagResync = true
	content.Data = random.Bits(100, false, random.Stream)

	msg.SetContent(*content)

	//test marshalling
	msgBytes, err := msg.ToBytes()

	if err != nil {
		t.Error(err)
	}
	if msgBytes == nil {
		t.Error("msgBytes can't be nil")
	}

	void := new(REL_CLI_DOWNSTREAM_DATA_UDP)
	msg2, err2 := void.FromBytes(msgBytes)

	if err2 != nil {
		t.Error(err2)
	}

	parsedMsg := msg2.(REL_CLI_DOWNSTREAM_DATA_UDP)

	if parsedMsg.RoundID != content.RoundID {
		t.Error("RoundID unparsed incorrectly")
	}
	if parsedMsg.FlagResync != content.FlagResync {
		t.Error("FlagResync unparsed incorrectly")
	}
	if !bytes.Equal(parsedMsg.Data, content.Data) {
		t.Error("Data unparsed incorrectly")
	}

	//this should fail, cannot read the size if len<4
	void = new(REL_CLI_DOWNSTREAM_DATA_UDP)
	_, err2 = void.FromBytes(msgBytes[0:3])

	if err2 == nil {
		t.Error("REL_CLI_DOWNSTREAM_DATA_UDP should not allow to decode message < 4 bytes")
	}

	//this should fail, the size is wrong
	void = new(REL_CLI_DOWNSTREAM_DATA_UDP)
	msgBytes[0] = byte(10)
	_, err2 = void.FromBytes(msgBytes)

	if err2 == nil {
		t.Error("REL_CLI_DOWNSTREAM_DATA_UDP should not allow to decode wrong-size messages")
	}
}

func TestUtils(t *testing.T) {

	m := make(map[string]interface{})
	m["test"] = 123
	m["test2"] = "abc"

	if ValueOrElse(m, "test", 456) != 123 {
		t.Error("ValueOrElse computed a wrong value")
	}
	if ValueOrElse(m, "test2", "def") != "abc" {
		t.Error("ValueOrElse computed a wrong value")
	}
	if ValueOrElse(m, "test3", "newval") != "newval" {
		t.Error("ValueOrElse computed a wrong value")
	}
	if ValueOrElse(m, "test4", float64(1.2)) != float64(1.2) {
		t.Error("ValueOrElse computed a wrong value")
	}
}