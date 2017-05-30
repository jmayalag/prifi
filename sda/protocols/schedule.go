package protocols

/*
 * PRIFI SCHEDULE WRAPPER
 *
 * Caution : this is not the "PriFi protocol", which is really a "PriFi Library" which you need to import,
 * and feed with some network methods. This is the "PriFi-SCHEDULE-Wrapper" protocol, which imports the PriFi lib,
 * gives it "SendToXXX()" methods and calls the "prifi_library.MessageReceived()" methods
 * (it build a map that converts the SDA tree into identities), and starts the PriFi Library.
 *
 * The call order is :
 * 1) the sda/app is called by the user/scripts
 * 2) the clients/trustees/relay start their services
 * 3) the clients/trustees services use their autoconnect() function
 * 4) when he decides so, the relay (via ChurnHandler) spawns a new protocol :
 * 5) this file is called; in order :
 * 5.1) init() that registers the messages
 * 5.2) NewPriFiScheduleWrapperProtocol() that creates a protocol (and contains the tree given by the service)
 * 5.3) in the service, setConfigToPriFiProtocol() is called, which calls the protocol (this file) 's SetConfigFromPriFiService()
 * 5.3.1) SetConfigFromPriFiService() calls both buildMessageSender() and registerHandlers()
 * 5.3.2) SetConfigFromPriFiService() calls New[Relay|Client|Trustee]State(); at this point, the protocol is ready to run
 * 6) the relay's service calls protocol.Start(), which happens here
 * 7) on the other entities, steps 5-6) will be repeated when a new message from the prifi protocols comes
 */

import (
	"errors"

	"github.com/lbarman/prifi/prifi-lib"
	"github.com/lbarman/prifi/prifi-lib/net"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
)

// PriFiScheduleProtocol is the SDA-protocol struct. It contains the SDA-tree,
// and a chanel that stops the simulation when it receives a "true"
type PriFiScheduleProtocol struct {
	*onet.TreeNodeInstance
	configSet     bool
	config        PriFiWrapperConfig
	role          PriFiRole
	ms            MessageSender
	toHandler     func([]string, []string)
	ResultChannel chan interface{}
	WhenFinished  func()

	//this is the actual "PriFi" (DC-net) protocol/library, defined in prifi-lib/prifi.go
	prifiLibInstance prifi_lib.SpecializedLibInstance
	HasStopped       bool //when set to true, the protocol has been stopped by PriFi-lib and should be destroyed
}

//Start is called on the Relay by the service when ChurnHandler decides so
func (p *PriFiScheduleProtocol) Start() error {

	if !p.configSet {
		log.Fatal("Trying to start PriFi-lib, but config not set !")
	}

	//At the protocol is ready,

	log.Lvl3("Starting PriFi-Schedule-Wrapper Protocol")

	//Perform the suffling once all keys are shared
	msg := new(net.CLI_REL_TELL_PK_AND_EPH_PK_2)

	p.SendTo(p.TreeNode(), msg)

	return nil
}

// Stop aborts the current execution of the protocol.
func (p *PriFiScheduleProtocol) Stop() {

	if p.prifiLibInstance != nil {
		switch p.role {
		case Relay:
			p.prifiLibInstance.ReceivedMessage(net.ALL_ALL_SHUTDOWN{})
		case Trustee:
			p.prifiLibInstance.ReceivedMessage(net.ALL_ALL_SHUTDOWN{})
		case Client:
			p.prifiLibInstance.ReceivedMessage(net.ALL_ALL_SHUTDOWN{})
		}
	}

	p.HasStopped = true

	p.Shutdown()
	//TODO : sureley we're missing some allocated resources here...
}

/**
 * On initialization of the PriFi-Schedule-Wrapper protocol, it need to register the PriFi-Lib messages to be able
 * to marshall them. If we forget some messages there, it will crash when PriFi-Lib will call SendToXXX() with this message !
 */
func init() {

	//register the prifi_lib's message with the network lib here
	network.RegisterMessage(net.CLI_REL_TELL_PK_AND_EPH_PK_2{})
	network.RegisterMessage(net.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE{})
	network.RegisterMessage(net.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS{})
	network.RegisterMessage(net.REL_TRU_TELL_TRANSCRIPT{})
	network.RegisterMessage(net.TRU_REL_SHUFFLE_SIG_1{})

	onet.GlobalProtocolRegister("PrifiScheduleProtocol", NewPriFiScheduleWrapperProtocol)
}

// handleTimeout translates ids int ServerIdentities
// and calls the timeout handler.
func (p *PriFiScheduleProtocol) handleTimeout(clientsIds []int, trusteesIds []int) {
	clients := make([]string, len(clientsIds))
	trustees := make([]string, len(trusteesIds))

	for i, v := range clientsIds {
		clients[i] = p.ms.clients[v].ServerIdentity.Address.String()
	}

	for i, v := range trusteesIds {
		trustees[i] = p.ms.trustees[v].ServerIdentity.Address.String()
	}

	p.toHandler(clients, trustees)
}

// NewPriFiScheduleWrapperProtocol creates a bare PrifiScheduleWrapper struct.
// SetConfig **MUST** be called on it before it can participate
// to the protocol.
func NewPriFiScheduleWrapperProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	p := &PriFiScheduleProtocol{
		TreeNodeInstance: n,
		ResultChannel:    make(chan interface{}),
	}

	return p, nil
}

// registerHandlers contains the verbose code
// that registers handlers for all prifi messages.
func (p *PriFiScheduleProtocol) registerHandlers() error {
	//register handlers
	err := p.RegisterHandler(p.Received_ALL_ALL_SHUTDOWN)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}

	//register relay handlers
	err = p.RegisterHandler(p.Received_CLI_REL_TELL_PK_AND_EPH_PK_2)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}
	err = p.RegisterHandler(p.Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}
	err = p.RegisterHandler(p.Received_TRU_REL_SHUFFLE_SIG_1)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}

	//register trustees handlers
	err = p.RegisterHandler(p.Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}
	err = p.RegisterHandler(p.Received_REL_TRU_TELL_TRANSCRIPT)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}

	return nil
}
