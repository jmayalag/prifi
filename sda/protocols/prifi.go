package protocols

import (
	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/network"
	prifi_lib "github.com/lbarman/prifi/prifi-lib"
	"github.com/lbarman/prifi/prifi-lib/net"
	"github.com/lbarman/prifi/prifi-lib/relay"
	"github.com/lbarman/prifi/prifi-lib/trustee"
	"github.com/lbarman/prifi/prifi-lib/client"
)

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

// SetConfig configures the PriFi node.
// It **MUST** be called in service.newProtocol or before Start().
func (p *PriFiSDAProtocol) SetConfigFromPriFiService(config *PriFiSDAWrapperConfig) {
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
		relayState := relay.NewRelayState(
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
			config.RelaySideSocksConfig.UpstreamChannel,
			p.handleTimeout)

		p.prifiLibInstance = prifi_lib.NewPriFiRelayWithState(ms, relayState)


	case Trustee:
		id := config.Identities[p.ServerIdentity().Address.String()].ID
		trusteeState := trustee.NewTrusteeState(id, nClients, nTrustees, config.UpCellSize)
		p.prifiLibInstance = prifi_lib.NewPriFiTrusteeWithState(ms, trusteeState)

	case Client:
		id := config.Identities[p.ServerIdentity().Address.String()].ID
		clientState := client.NewClientState(id,
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
