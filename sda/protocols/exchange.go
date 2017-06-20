package protocols

/*
 * PRIFI EXCHANGE WRAPPER
 *
 * Caution : this is not the "PriFi protocol", which is really a "PriFi Library" which you need to import,
 * and feed with some network methods. This is the "PriFi-EXCHANGE-Wrapper" protocol, which imports the PriFi lib,
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
 * 5.2) NewPriFiExchangeWrapperProtocol() that creates a protocol (and contains the tree given by the service)
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

// PriFiExchangeProtocol is the SDA-protocol struct. It contains the SDA-tree,
// and a chanel that stops the simulation when it receives a "true"
type PriFiExchangeProtocol struct {
	*onet.TreeNodeInstance
	configSet     bool
	config        PriFiWrapperConfig
	role          PriFiRole
	ms            MessageSender
	toHandler     func([]string, []string)
	ResultChannel chan interface{}
	WhenFinished  func(prifi_lib.SpecializedLibInstance)

	//this is the actual "PriFi" (DC-net) protocol/library, defined in prifi-lib/prifi.go
	prifiLibInstance prifi_lib.SpecializedLibInstance
	HasStopped       bool //when set to true, the protocol has been stopped by PriFi-lib and should be destroyed
}

//Start is called on the Relay by the service when ChurnHandler decid 	es so
func (p *PriFiExchangeProtocol) Start() error {

	if !p.configSet {
		log.Fatal("Trying to start PriFi-lib, but config not set !")
	}

	//At the protocol is ready,

	log.Lvl3("Starting PriFi-Exchange-Wrapper Protocol")

	//emulate the reception of a ALL_ALL_PARAMETERS with StartNow=true
	msg := new(net.ALL_ALL_PARAMETERS_NEW)
	msg.Add("StartNow", true)
	msg.Add("NTrustees", len(p.ms.trustees))
	msg.Add("NClients", len(p.ms.clients))
	msg.Add("UpstreamCellSize", p.config.Toml.CellSizeUp)
	msg.Add("DownstreamCellSize", p.config.Toml.CellSizeDown)
	msg.Add("WindowSize", p.config.Toml.RelayWindowSize)
	msg.Add("UseOpenClosedSlots", p.config.Toml.RelayUseOpenClosedSlots)
	msg.Add("UseDummyDataDown", p.config.Toml.RelayUseDummyDataDown)
	msg.Add("ExperimentRoundLimit", p.config.Toml.RelayReportingLimit)
	msg.Add("UseUDP", p.config.Toml.UseUDP)
	msg.Add("DCNetType", p.config.Toml.DCNetType)
	msg.ForceParams = true

	p.SendTo(p.TreeNode(), msg)

	return nil
}

// Stop aborts the current execution of the protocol.
func (p *PriFiExchangeProtocol) Stop() {

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
	//TODO : surely we're missing some allocated resources here...
}

/**
 * On initialization of the PriFi-Exchange-Wrapper protocol, it need to register the PriFi-Lib messages to be able
 * to marshall them. If we forget some messages there, it will crash when PriFi-Lib will call SendToXXX() with this message !
 */
func init() {

	//register the prifi_lib's message with the network lib here
	network.RegisterMessage(net.ALL_ALL_PARAMETERS_NEW{})
	network.RegisterMessage(net.TRU_REL_TELL_PK{})
	network.RegisterMessage(net.REL_CLI_TELL_TRUSTEES_PK{})
	network.RegisterMessage(net.CLI_REL_TELL_PK_AND_EPH_PK{})

	onet.GlobalProtocolRegister("PrifiExchangeProtocol", NewPriFiExchangeWrapperProtocol)
}

// handleTimeout translates ids int ServerIdentities
// and calls the timeout handler.
func (p *PriFiExchangeProtocol) handleTimeout(clientsIds []int, trusteesIds []int) {
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

// NewPriFiExchangeWrapperProtocol creates a bare PrifiExchangeWrapper struct.
// SetConfig **MUST** be called on it before it can participate
// to the protocol.
func NewPriFiExchangeWrapperProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	p := &PriFiExchangeProtocol{
		TreeNodeInstance: n,
		ResultChannel:    make(chan interface{}),
	}

	return p, nil
}

// registerHandlers contains the verbose code
// that registers handlers for all prifi messages.
func (p *PriFiExchangeProtocol) registerHandlers() error {
	//register handlers
	err := p.RegisterHandler(p.Received_ALL_ALL_PARAMETERS_NEW)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}
	err = p.RegisterHandler(p.Received_ALL_ALL_SHUTDOWN)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}

	//register client handlers
	err = p.RegisterHandler(p.Received_REL_CLI_TELL_TRUSTEES_PK)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}

	//register relay handlers
	err = p.RegisterHandler(p.Received_TRU_REL_TELL_PK)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}
	err = p.RegisterHandler(p.Received_CLI_REL_TELL_PK_AND_EPH_PK)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}

	return nil
}
