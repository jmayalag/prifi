package protocols

/*
 * PRIFI SDA WRAPPER
 *
 * Caution : this is not the "PriFi protocol", which is really a "PriFi Library" which you need to import, and feed with some network methods.
 * This is the "PriFi-SDA-Wrapper" protocol, which imports the PriFi lib, gives it "SendToXXX()" methods and calls the "prifi_library.MessageReceived()"
 * methods (it build a map that converts the SDA tree into identities), and starts the PriFi Library.
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

//the UDP channel we provide to PriFi. check udp.go for more details.
var udpChan = newRealUDPChannel() // Cannot use localhost channel anymore for real deployment

//PriFiRole is the type of the enum to qualify the role of a SDA node (Relay, Client, Trustee)
type PriFiRole int

//The possible states of a SDA node, of type PriFiRole
const (
	Relay PriFiRole = iota
	Client
	Trustee
)

//PriFiIdentity is the identity (role + ID)
type PriFiIdentity struct {
	Role     PriFiRole
	ID       int
	ServerID *network.ServerIdentity
}

//SOCKSConfig contains the port, payload, and up/down channels for data
type SOCKSConfig struct {
	Port              string
	PayloadLength     int
	UpstreamChannel   chan []byte
	DownstreamChannel chan []byte
}

//PriFiSDAWrapperConfig is all the information the SDA-Protocols needs. It contains the network map of identities, our role, and the socks parameters if we are the corresponding role
type PriFiSDAWrapperConfig struct {
	net.ALL_ALL_PARAMETERS
	Identities            map[string]PriFiIdentity
	Role                  PriFiRole
	ClientSideSocksConfig *SOCKSConfig
	RelaySideSocksConfig  *SOCKSConfig
}

//PriFiSDAProtocol is the SDA-protocol struct. It contains the SDA-tree, and a chanel that stops the simulation when it receives a "true"
type PriFiSDAProtocol struct {
	*sda.TreeNodeInstance
	configSet     bool
	config        PriFiSDAWrapperConfig
	role          PriFiRole
	ms            MessageSender
	toHandler     func([]string, []string)
	ResultChannel chan interface{}
	// running is a pointer to the service's variable
	// indicating if the protocol is running. It should
	// be set to false when the protocol is stopped.
	//IsRunning bool // TODO: We should use a lock before modifying it

	//this is the actual "PriFi" (DC-net) protocol/library, defined in prifi-lib/prifi.go
	prifiLibInstance *prifi_lib.PriFiLibInstance
	HasStopped       bool //when set to true, the protocol has been stopped by PriFi-lib and should be destro
}

//Start implements the sda.Protocol interface.
func (p *PriFiSDAProtocol) Start() error {
	if !p.configSet {
		log.Fatal("Trying to start PriFi Library, but config not set !")
	}

	log.Lvl3("Starting PriFi-SDA-Wrapper Protocol")

	p.prifiLibInstance.ConnectToTrustees()

	return nil
}

// Stop aborts the current execution of the protocol.
func (p *PriFiSDAProtocol) Stop() {

	switch p.role {
	case Relay:
		p.prifiLibInstance.Received_ALL_REL_SHUTDOWN(net.ALL_ALL_SHUTDOWN{})
	case Trustee:
		p.prifiLibInstance.Received_ALL_TRU_SHUTDOWN(net.ALL_ALL_SHUTDOWN{})
	case Client:
		p.prifiLibInstance.Received_ALL_CLI_SHUTDOWN(net.ALL_ALL_SHUTDOWN{})
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

// SetConfig configures the PriFi node.
// It **MUST** be called in service.newProtocol or before Start().
func (p *PriFiSDAProtocol) SetConfig(config *PriFiSDAWrapperConfig) {
	p.config = *config
	p.role = config.Role

	ms := p.buildMessageSender(config.Identities)
	p.ms = ms

	//sanity check
	switch config.Role {
	case Trustee:
		if ms.relay == nil {
			log.Fatal("Relay is not reachable (I'm a trustee, and I need it) !")
		}
	case Client:
		if ms.relay == nil {
			log.Fatal("Relay is not reachable (I'm a client, and I need it) !")
		}
	case Relay:
		if len(ms.clients) < 1 {
			log.Fatal("Less than one client reachable (I'm a relay, and there's no use starting the protocol) !")
		}
		if len(ms.trustees) < 1 {
			log.Fatal("No trustee reachable (I'm a relay, and I cannot start the protocol) !")
		}
	}

	nClients := len(ms.clients)
	nTrustees := len(ms.trustees)
	experimentResultChan := p.ResultChannel

	switch config.Role {
	case Relay:
		relayState := prifi_lib.NewRelayState(
			nTrustees,
			nClients,
			config.UpCellSize,
			config.DownCellSize,
			config.RelayWindowSize,
			config.RelayUseDummyDataDown,
			config.RelayReportingLimit,
			experimentResultChan,
			config.UseUDP,
			config.RelayDataOutputEnabled,
			config.RelaySideSocksConfig.DownstreamChannel,
			config.RelaySideSocksConfig.UpstreamChannel)

		p.prifiLibInstance = prifi_lib.NewPriFiRelayWithState(ms, relayState)

		p.prifiLibInstance.SetTimeoutHandler(p.handleTimeout)

	case Trustee:
		id := config.Identities[p.ServerIdentity().Address.String()].ID
		trusteeState := prifi_lib.NewTrusteeState(id, nClients, nTrustees, config.UpCellSize)
		p.prifiLibInstance = prifi_lib.NewPriFiTrusteeWithState(ms, trusteeState)

	case Client:
		id := config.Identities[p.ServerIdentity().Address.String()].ID
		clientState := prifi_lib.NewClientState(id,
			nTrustees,
			nClients,
			config.UpCellSize,
			config.DoLatencyTests,
			config.UseUDP,
			config.ClientDataOutputEnabled,
			config.ClientSideSocksConfig.UpstreamChannel,
			config.ClientSideSocksConfig.DownstreamChannel)
		p.prifiLibInstance = prifi_lib.NewPriFiClientWithState(ms, clientState)
	}

	p.registerHandlers()

	p.configSet = true
}

// SetTimeoutHandler sets the function that will be called on round timeout
// if the protocol runs as the relay.
func (p *PriFiSDAProtocol) SetTimeoutHandler(handler func([]string, []string)) {
	p.toHandler = handler
}

// buildMessageSender creates a MessageSender struct
// given a mep between server identities and PriFi identities.
func (p *PriFiSDAProtocol) buildMessageSender(identities map[string]PriFiIdentity) MessageSender {
	nodes := p.List() // Has type []*sda.TreeNode
	trustees := make(map[int]*sda.TreeNode)
	clients := make(map[int]*sda.TreeNode)
	trusteeID := 0
	clientID := 0
	var relay *sda.TreeNode

	for i := 0; i < len(nodes); i++ {
		identifier := nodes[i].ServerIdentity.Address.String() + "=" + nodes[i].ServerIdentity.Public.String()
		id, ok := identities[identifier]

		if !ok {
			log.Lvl3("Skipping unknow node with address", identifier)
			continue
		}
		switch id.Role {
		case Client:
			clients[clientID] = nodes[i] //TODO : wrong
			clientID++
		case Trustee:
			trustees[trusteeID] = nodes[i]
			trusteeID++
		case Relay:
			if relay == nil {
				relay = nodes[i]
			} else {
				log.Fatal("Multiple relays")
			}
		}
	}

	return MessageSender{p.TreeNodeInstance, relay, clients, trustees}
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
