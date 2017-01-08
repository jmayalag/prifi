// Package prifi-sda-service contains the SDA service responsible
// for starting the SDA protocols required to enable PriFi
// communications.
package services

/*
* This is the internal part of the API. As probably the prifi-service will
* not have an external API, this will not have any API-functions.
 */

import (
	"io/ioutil"
	"strconv"

	"github.com/dedis/cothority/app/lib/config"
	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/network"
	"github.com/dedis/cothority/sda"
	prifi_socks "github.com/lbarman/prifi/prifi-socks"
	prifi_protocol "github.com/lbarman/prifi/sda/protocols"
	"time"
)

//The name of the service, used by SDA's internals
const ServiceName = "PriFiService"

var serviceID sda.ServiceID

// Register Service with SDA
func init() {
	sda.RegisterNewService(ServiceName, newService)
	serviceID = sda.ServiceFactory.ServiceID(ServiceName)
}

// contains the identity map, a direct link to the relay, and a mutex
type SDANodesAndIDs struct {
	currentIdentitiesMap  map[string]prifi_protocol.PriFiIdentity //contains relay+trustee + whoever connects
	originalIdentitiesMap map[string]prifi_protocol.PriFiIdentity //contains relay+trustee for the relay
	relayIdentity         *network.ServerIdentity
	group                 *config.Group
}

//Service contains the state of the service
type ServiceState struct {
	// We need to embed the ServiceProcessor, so that incoming messages
	// are correctly handled.
	*sda.ServiceProcessor
	prifiTomlConfig *PrifiTomlConfig
	Storage         *Storage
	path            string
	role            prifi_protocol.PriFiRole

	//this hold the churn handler; protocol is started there. Only relay has this != nil
	churnHandler *churnHandler

	//this hold the running protocol (when it runs)
	priFiSDAProtocol *prifi_protocol.PriFiSDAProtocol
}

// returns true if the PriFi SDA protocol is running (in any state : init, communicate, etc)
func (s *ServiceState) IsPriFiProtocolRunning() bool {
	if s.priFiSDAProtocol != nil {
		return !s.priFiSDAProtocol.HasStopped
	}
	return false
}

// Storage will be saved, on the contrary of the 'Service'-structure
// which has per-service information stored.
type Storage struct {
	//our service has no state to be saved
}

// This is a handler passed to the SDA when starting a host. The SDA usually handle all the network by itself,
// but in our case it is useful to know when a network RESET occured, so we can kill protocols (otherwise they
// remain in some weird state)
func (s *ServiceState) NetworkErrorHappened(e error) {

	if s.role != prifi_protocol.Relay {
		log.Error("A network error occurred, but we're not the relay, nothing to do.")
		return
	}

	s.stopPriFiCommunicateProtocol()
}

// StartTrustee starts the necessary
// protocols to enable the trustee-mode.
func (s *ServiceState) StartTrustee(group *config.Group) error {
	log.Info("Service", s, "running in trustee mode")
	s.role = prifi_protocol.Trustee

	_, relayID := mapIdentities(group)
	go s.autoConnect(relayID)

	return nil
}

// StartRelay starts the necessary
// protocols to enable the relay-mode.
// In this example it simply starts the demo protocol
func (s *ServiceState) StartRelay(group *config.Group) error {
	log.Info("Service", s, "running in relay mode")

	//set state to the correct info, parse .toml
	s.role = prifi_protocol.Relay
	idMap, relayID := mapIdentities(group)

	s.churnHandler = new(churnHandler)
	s.churnHandler.init(relayID)
	s.churnHandler.isATrustee = func(ID *network.ServerIdentity) bool {

		for _, v := range idMap {
			if v.ServerID.Equal(ID) {
				return true //only trustees and relay are in idMap
			}
		}
		return false
	}

	socksServerConfig = &prifi_protocol.SOCKSConfig{
		Port:              "127.0.0.1:" + strconv.Itoa(s.prifiTomlConfig.SocksClientPort),
		PayloadLength:     s.prifiTomlConfig.CellSizeUp,
		UpstreamChannel:   make(chan []byte),
		DownstreamChannel: make(chan []byte),
	}

	go prifi_socks.StartSocksClient(socksServerConfig.Port, socksServerConfig.UpstreamChannel, socksServerConfig.DownstreamChannel)

	return nil
}

var socksClientConfig *prifi_protocol.SOCKSConfig
var socksServerConfig *prifi_protocol.SOCKSConfig

// StartClient starts the necessary
// protocols to enable the client-mode.
func (s *ServiceState) StartClient(group *config.Group) error {
	log.Info("Service", s, "running in client mode")
	s.role = prifi_protocol.Client

	_, relayID := mapIdentities(group)

	socksClientConfig = &prifi_protocol.SOCKSConfig{
		Port:              ":" + strconv.Itoa(s.prifiTomlConfig.SocksServerPort),
		PayloadLength:     s.prifiTomlConfig.CellSizeUp,
		UpstreamChannel:   make(chan []byte),
		DownstreamChannel: make(chan []byte),
	}

	log.Lvl1("Starting SOCKS server on port", socksClientConfig.Port)
	go prifi_socks.StartSocksServer(socksClientConfig.Port, socksClientConfig.PayloadLength, socksClientConfig.UpstreamChannel, socksClientConfig.DownstreamChannel, s.prifiTomlConfig.DoLatencyTests)

	go s.autoConnect(relayID)

	return nil
}

// StartClient starts the necessary
// protocols to enable the client-mode.
func (s *ServiceState) StartSocksTunnelOnly() error {
	log.Info("Service", s, "running in socks-tunnel-only mode")

	socksClientConfig = &prifi_protocol.SOCKSConfig{
		Port:              ":" + strconv.Itoa(s.prifiTomlConfig.SocksServerPort),
		PayloadLength:     s.prifiTomlConfig.CellSizeUp,
		UpstreamChannel:   make(chan []byte),
		DownstreamChannel: make(chan []byte),
	}

	socksServerConfig = &prifi_protocol.SOCKSConfig{
		Port:              "127.0.0.1:" + strconv.Itoa(s.prifiTomlConfig.SocksClientPort),
		PayloadLength:     s.prifiTomlConfig.CellSizeUp,
		UpstreamChannel:   socksClientConfig.UpstreamChannel,
		DownstreamChannel: socksClientConfig.DownstreamChannel,
	}
	go prifi_socks.StartSocksServer(socksClientConfig.Port, socksClientConfig.PayloadLength, socksClientConfig.UpstreamChannel, socksClientConfig.DownstreamChannel, false)
	go prifi_socks.StartSocksClient(socksServerConfig.Port, socksServerConfig.UpstreamChannel, socksServerConfig.DownstreamChannel)

	return nil
}

// NewProtocol is called on all nodes of a Tree (except the root, since it is
// the one starting the protocol) so it's the Service that will be called to
// generate the PI on all others node.
// If you use CreateProtocolSDA, this will not be called, as the SDA will
// instantiate the protocol on its own. If you need more control at the
// instantiation of the protocol, use CreateProtocolService, and you can
// give some extra-configuration to your protocol in here.
func (s *ServiceState) NewProtocol(tn *sda.TreeNodeInstance, conf *sda.GenericConfig) (sda.ProtocolInstance, error) {

	pi, err := prifi_protocol.NewPriFiSDAWrapperProtocol(tn)
	if err != nil {
		return nil, err
	}

	wrapper := pi.(*prifi_protocol.PriFiSDAProtocol)
	s.priFiSDAProtocol = wrapper
	s.setConfigToPriFiProtocol(wrapper)

	return wrapper, nil
}

// save saves the actual identity
func (s *ServiceState) save() {
	log.Lvl3("Saving service")
	b, err := network.MarshalRegisteredType(s.Storage)
	if err != nil {
		log.Error("Couldn't marshal service:", err)
	} else {
		err = ioutil.WriteFile(s.path+"/prifi.bin", b, 0660)
		if err != nil {
			log.Error("Couldn't save file:", err)
		}
	}
}

// newService receives the context and a path where it can write its
// configuration, if desired. As we don't know when the service will exit,
// we need to save the configuration on our own from time to time.
func newService(c *sda.Context, path string) sda.Service {
	log.LLvl4("Calling newService")
	s := &ServiceState{
		ServiceProcessor: sda.NewServiceProcessor(c),
		path:             path,
	}

	c.RegisterProcessorFunc(network.TypeFromData(StopProtocol{}), s.HandleStop)
	c.RegisterProcessorFunc(network.TypeFromData(ConnectionRequest{}), s.churnHandler.handleDisconnection)
	c.RegisterProcessorFunc(network.TypeFromData(DisconnectionRequest{}), s.churnHandler.handleDisconnection)

	if err := s.tryLoad(); err != nil {
		log.Error(err)
	}
	return s
}

// Packet send by relay when some node disconnected
func (s *ServiceState) HandleStop(msg *network.Packet) {

	log.Lvl1("Received a Handle Stop")

	s.stopPriFiCommunicateProtocol()

}

// handleTimeout is a callback that should be called on the relay
// when a round times out. It tries to restart PriFi with the nodes
// that sent their ciphertext in time.
func (s *ServiceState) handleTimeout(lateClients []string, lateTrustees []string) {

	// we can probably do something more clever here, since we know who disconnected. Yet let's just restart everything
	s.stopPriFiCommunicateProtocol()
}

// startPriFi starts a PriFi protocol. It is called
// by the relay as soon as enough participants are
// ready (one trustee and two clients).
func (s *ServiceState) startPriFiCommunicateProtocol() {
	log.Lvl1("Starting PriFi protocol")

	if s.role != prifi_protocol.Relay {
		log.Error("Trying to start PriFi protocol from a non-relay node.")
		return
	}

	var wrapper *prifi_protocol.PriFiSDAProtocol
	roster := s.churnHandler.createRoster()

	// Start the PriFi protocol on a flat tree with the relay as root
	tree := roster.GenerateNaryTreeWithRoot(100, s.churnHandler.relayIdentity)
	pi, err := s.CreateProtocolService(prifi_protocol.ProtocolName, tree)

	if err != nil {
		log.Fatal("Unable to start Prifi protocol:", err)
	}

	// Assert that pi has type PriFiSDAWrapper
	wrapper = pi.(*prifi_protocol.PriFiSDAProtocol)

	//assign and start the protocol
	s.priFiSDAProtocol = wrapper

	s.setConfigToPriFiProtocol(wrapper)

	wrapper.Start()
}

// stopPriFi stops the PriFi protocol currently running.
func (s *ServiceState) stopPriFiCommunicateProtocol() {
	log.Lvl1("Stopping PriFi protocol")

	if !s.IsPriFiProtocolRunning() {
		log.Lvl3("Would stop PriFi protocol, but it's not running.")
		return
	}

	log.Lvl2("A network error occurred, killing the PriFi protocol.")

	if s.priFiSDAProtocol != nil {
		s.priFiSDAProtocol.Stop()
	}
	s.priFiSDAProtocol = nil

	if s.role == prifi_protocol.Relay {

		log.Lvl2("A network error occurred, we're the relay, warning other clients...")

		for _, v := range s.churnHandler.getClientsIdentities() {
			s.SendRaw(v, &StopProtocol{})
		}
		for _, v := range s.churnHandler.getTrusteesIdentities() {
			s.SendRaw(v, &StopProtocol{})
		}
	}
}

// autoConnect sends a connection request to the relay
// every 10 seconds if the node is not participating to
// a PriFi protocol.
func (s *ServiceState) autoConnect(relayID *network.ServerIdentity) {
	s.sendConnectionRequest(relayID)

	tick := time.Tick(DELAY_BEFORE_KEEPALIVE)
	for range tick {
		if !s.IsPriFiProtocolRunning() {
			s.sendConnectionRequest(relayID)
		}
	}
}

// sendConnectionRequest sends a connection request to the relay.
// It is called by the client and trustee services at startup to
// announce themselves to the relay.
func (s *ServiceState) sendConnectionRequest(relayID *network.ServerIdentity) {
	log.Lvl2("Sending connection request")
	err := s.SendRaw(relayID, &ConnectionRequest{})

	if err != nil {
		log.Error("Connection failed:", err)
	}
}
