package protocols

/*
 * PRIFI SDA WRAPPER
 *
 * Caution : this is not the "PriFi protocol", which is really a "PriFi Library" which you need to import, and feed with some network methods.
 * This is the "PriFi-SDA-Wrapper" protocol, which imports the PriFi lib, gives it "SendToXXX()" methods and calls the "prifi_library.MessageReceived()"
 * methods (it build a map that converts the SDA tree into identities), and starts the PriFi Library.
 *
 * The call order is :
 * 1) the sda/app is called by the user/scripts
 * 2) the clients/trustees/relay start their services
 * 3) the clients/trustees services use their autoconnect() function
 * 4) when he decides so, the relay (via ChurnHandler) spawns a new protocol :
 * 5) this file is called; in order :
 * 5.1) init() that registers the messages
 * 5.2) NewPriFiSDAWrapperProtocol() that creates a protocol (and contains the tree given by the service)
 * 5.3) in the service, setConfigToPriFiProtocol() is called, which calls the protocol (this file) 's SetConfigFromPriFiService()
 * 5.3.1) SetConfigFromPriFiService() calls both buildMessageSender() and registerHandlers()
 * 5.3.2) SetConfigFromPriFiService() calls New[Relay|Client|Trustee]State(); at this point, the protocol is ready to run
 * 6) the relay's service calls protocol.Start(), which happens here
 * 7) on the other entities, steps 5-6) will be repeated when a new message from the prifi protocols comes
 */

import (
	"errors"

	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/network"
	"github.com/dedis/cothority/sda"
	prifi_lib "github.com/lbarman/prifi/prifi-lib"
	"github.com/lbarman/prifi/prifi-lib/net"
)

// ProtocolName is the name used to register the SDA wrapper protocol with SDA.
const ProtocolName = "PrifiProtocol"

//PriFiSDAProtocol is the SDA-protocol struct. It contains the SDA-tree, and a chanel that stops the simulation when it receives a "true"
type PriFiSDAProtocol struct {
	*sda.TreeNodeInstance
	configSet     bool
	config        PriFiSDAWrapperConfig
	role          PriFiRole
	ms            MessageSender
	toHandler     func([]string, []string)
	ResultChannel chan interface{}

	//this is the actual "PriFi" (DC-net) protocol/library, defined in prifi-lib/prifi.go
	prifiLibInstance *prifi_lib.PriFiLibInstance
	HasStopped       bool //when set to true, the protocol has been stopped by PriFi-lib and should be destroyed
}

//Start is called on the Relay by the service when ChurnHandler decides so
func (p *PriFiSDAProtocol) Start() error {
	if !p.configSet {
		log.Fatal("Trying to start PriFi-lib, but config not set !")
	}

	//At the protocol is ready,

	log.Lvl3("Starting PriFi-SDA-Wrapper Protocol")

	//emulate the reception of a ALL_ALL_PARAMETERS with StartNow=true
	builder := net.NewALL_ALL_PARAMETERS_BUILDER()
	builder.Add("StartNow", true)
	msg := builder.BuildMessage(true)
	p.prifiLibInstance.ReceivedMessage(msg)

	return nil
}

// Stop aborts the current execution of the protocol.
func (p *PriFiSDAProtocol) Stop() {

	switch p.role {
	case Relay:
		p.prifiLibInstance.ReceivedMessage(net.ALL_ALL_SHUTDOWN{})
	case Trustee:
		p.prifiLibInstance.ReceivedMessage(net.ALL_ALL_SHUTDOWN{})
	case Client:
		p.prifiLibInstance.ReceivedMessage(net.ALL_ALL_SHUTDOWN{})
	}

	p.HasStopped = true

	p.Shutdown()
	//TODO : sureley we're missing some allocated resources here...
}

/**
 * On initialization of the PriFi-SDA-Wrapper protocol, it need to register the PriFi-Lib messages to be able to marshall them.
 * If we forget some messages there, it will crash when PriFi-Lib will call SendToXXX() with this message !
 */
func init() {

	//register the prifi_lib's message with the network lib here
	network.RegisterPacketType(net.ALL_ALL_PARAMETERS{})
	network.RegisterPacketType(net.ALL_ALL_PARAMETERS_NEW{})
	network.RegisterPacketType(net.CLI_REL_TELL_PK_AND_EPH_PK{})
	network.RegisterPacketType(net.CLI_REL_UPSTREAM_DATA{})
	network.RegisterPacketType(net.REL_CLI_DOWNSTREAM_DATA{})
	network.RegisterPacketType(net.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG{})
	network.RegisterPacketType(net.REL_CLI_TELL_TRUSTEES_PK{})
	network.RegisterPacketType(net.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE{})
	network.RegisterPacketType(net.REL_TRU_TELL_TRANSCRIPT{})
	network.RegisterPacketType(net.TRU_REL_DC_CIPHER{})
	network.RegisterPacketType(net.REL_TRU_TELL_RATE_CHANGE{})
	network.RegisterPacketType(net.TRU_REL_SHUFFLE_SIG{})
	network.RegisterPacketType(net.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS{})
	network.RegisterPacketType(net.TRU_REL_TELL_PK{})

	sda.GlobalProtocolRegister(ProtocolName, NewPriFiSDAWrapperProtocol)
}

// handleTimeout translates ids int ServerIdentities
// and calls the timeout handler.
func (p *PriFiSDAProtocol) handleTimeout(clientsIds []int, trusteesIds []int) {
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

// NewPriFiSDAWrapperProtocol creates a bare PrifiSDAWrapper struct.
// SetConfig **MUST** be called on it before it can participate
// to the protocol.
func NewPriFiSDAWrapperProtocol(n *sda.TreeNodeInstance) (sda.ProtocolInstance, error) {
	p := &PriFiSDAProtocol{
		TreeNodeInstance: n,
		ResultChannel:    make(chan interface{}),
	}

	return p, nil
}

// registerHandlers contains the verbose code
// that registers handlers for all prifi messages.
func (p *PriFiSDAProtocol) registerHandlers() error {
	//register handlers
	err := p.RegisterHandler(p.Received_ALL_ALL_PARAMETERS)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}
	err = p.RegisterHandler(p.Received_ALL_ALL_SHUTDOWN)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}

	//register client handlers
	err = p.RegisterHandler(p.Received_REL_CLI_DOWNSTREAM_DATA)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}
	err = p.RegisterHandler(p.Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}
	err = p.RegisterHandler(p.Received_REL_CLI_TELL_TRUSTEES_PK)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}

	//register relay handlers
	err = p.RegisterHandler(p.Received_CLI_REL_TELL_PK_AND_EPH_PK)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}
	err = p.RegisterHandler(p.Received_CLI_REL_UPSTREAM_DATA)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}
	err = p.RegisterHandler(p.Received_TRU_REL_DC_CIPHER)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}
	err = p.RegisterHandler(p.Received_TRU_REL_SHUFFLE_SIG)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}
	err = p.RegisterHandler(p.Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}
	err = p.RegisterHandler(p.Received_TRU_REL_TELL_PK)
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
	err = p.RegisterHandler(p.Received_REL_TRU_TELL_RATE_CHANGE)
	if err != nil {
		return errors.New("couldn't register handler: " + err.Error())
	}

	return nil
}
