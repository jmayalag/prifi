// Package prifi-sda-service contains the SDA service responsible
// for starting the SDA protocols required to enable PriFi
// communications.
package services

/*
* This is the internal part of the API. As probably the prifi-service will
* not have an external API, this will not have any API-functions.
 */

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/dedis/cothority/app/lib/config"
	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/network"
	"github.com/dedis/cothority/sda"
	prifi_lib "github.com/lbarman/prifi/prifi-lib"
	prifi_socks "github.com/lbarman/prifi/prifi-socks"
	prifi_protocol "github.com/lbarman/prifi/sda/protocols"
	"sync"
)

//The name of the service, used by SDA's internals
const ServiceName = "PriFiService"

var serviceID sda.ServiceID

// Register Service with SDA
func init() {
	sda.RegisterNewService(ServiceName, newService)
	serviceID = sda.ServiceFactory.ServiceID(ServiceName)
}

//The configuration read in prifi.toml
type PrifiTomlConfig struct {
	CellSizeUp            int
	CellSizeDown          int
	RelayWindowSize       int
	RelayUseDummyDataDown bool
	RelayReportingLimit   int
	UseUDP                bool
	DoLatencyTests        bool
	SocksServerPort       int
	SocksClientPort       int
}

// contains the identity map, a direct link to the relay, and a mutex
type SDANodesAndIDs struct {
	mutex             sync.Mutex
	identitiesMap     map[string]prifi_protocol.PriFiIdentity
	relayIdentity     *network.ServerIdentity
	group             *config.Group
	nextFreeClientID  int
	nextFreeTrusteeID int
}

//Set the config, from the prifi.toml. Is called by sda/app.
func (s *ServiceState) SetConfigFromToml(config *PrifiTomlConfig) {
	log.Lvl3("Setting PriFi configuration...")
	log.Lvlf3("%+v\n", config)
	s.prifiTomlConfig = config
}

//Service contains the state of the service
type ServiceState struct {
	// We need to embed the ServiceProcessor, so that incoming messages
	// are correctly handled.
	*sda.ServiceProcessor
	prifiTomlConfig  *PrifiTomlConfig
	Storage          *Storage
	path             string
	role             prifi_protocol.PriFiRole
	nodesAndIDs      *SDANodesAndIDs
	waitQueue        *waitQueue
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

	log.Lvl2("A network error occurred, warning other clients...")

	if s.role == prifi_protocol.Relay {
		s.waitQueue.mutex.Lock()
		for k, v := range s.waitQueue.clients {
			s.SendRaw(v.Identity, &StopProtocol{})
			s.waitQueue.clients[k].IsWaiting = false
		}
		for k, v := range s.waitQueue.trustees {
			s.SendRaw(v.Identity, &StopProtocol{})
			s.waitQueue.trustees[k].IsWaiting = false
		}
		s.waitQueue.mutex.Unlock()
	}

	if s.IsPriFiProtocolRunning() {

		log.Lvl2("A network error occurred, killing the PriFi protocol.")

		if s.priFiSDAProtocol != nil {
			s.priFiSDAProtocol.Stop()
		}
		s.priFiSDAProtocol = nil

		return
	}

	log.Lvl3("A network error occurred, would kill PriFi protocol, but it's not running.")
}

// StartTrustee starts the necessary
// protocols to enable the trustee-mode.
func (s *ServiceState) StartTrustee(group *config.Group) error {
	log.Info("Service", s, "running in trustee mode")
	s.role = prifi_protocol.Trustee
	s.readGroup(group)

	go s.autoConnect()

	return nil
}

// StartRelay starts the necessary
// protocols to enable the relay-mode.
// In this example it simply starts the demo protocol
func (s *ServiceState) StartRelay(group *config.Group) error {
	log.Info("Service", s, "running in relay mode")
	s.role = prifi_protocol.Relay
	s.readGroup(group)
	s.waitQueue = &waitQueue{
		clients:  make(map[string]*WaitQueueEntry),
		trustees: make(map[string]*WaitQueueEntry),
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
	s.readGroup(group)

	socksClientConfig = &prifi_protocol.SOCKSConfig{
		Port:              ":" + strconv.Itoa(s.prifiTomlConfig.SocksServerPort),
		PayloadLength:     s.prifiTomlConfig.CellSizeUp,
		UpstreamChannel:   make(chan []byte),
		DownstreamChannel: make(chan []byte),
	}

	go prifi_socks.StartSocksServer(socksClientConfig.Port, socksClientConfig.PayloadLength, socksClientConfig.UpstreamChannel, socksClientConfig.DownstreamChannel, s.prifiTomlConfig.DoLatencyTests)

	go s.autoConnect()

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

func (s *ServiceState) setConfigToPriFiProtocol(wrapper *prifi_protocol.PriFiSDAProtocol) {

	log.Lvl1("setConfigToPriFiProtocol called")
	log.Lvlf1("%+v\n", s.prifiTomlConfig)

	prifiParams := prifi_lib.ALL_ALL_PARAMETERS{
		ClientDataOutputEnabled: true,
		DoLatencyTests:          s.prifiTomlConfig.DoLatencyTests,
		DownCellSize:            s.prifiTomlConfig.CellSizeDown,
		ForceParams:             true,
		NClients:                -1, //computer later
		NextFreeClientID:        0,
		NextFreeTrusteeID:       0,
		NTrustees:               -1, //computer later
		RelayDataOutputEnabled:  true,
		RelayReportingLimit:     s.prifiTomlConfig.RelayReportingLimit,
		RelayUseDummyDataDown:   s.prifiTomlConfig.RelayUseDummyDataDown,
		RelayWindowSize:         s.prifiTomlConfig.RelayWindowSize,
		StartNow:                false,
		UpCellSize:              s.prifiTomlConfig.CellSizeUp,
		UseUDP:                  s.prifiTomlConfig.UseUDP,
	}

	//deep-clone the identityMap
	s.nodesAndIDs.mutex.Lock()
	idMapCopy := make(map[string]prifi_protocol.PriFiIdentity)
	for k, v := range s.nodesAndIDs.identitiesMap {
		idMapCopy[k] = prifi_protocol.PriFiIdentity{
			ID:      v.ID,
			Role:    v.Role,
			Address: v.Address,
		}
	}
	s.nodesAndIDs.mutex.Unlock()

	configMsg := &prifi_protocol.PriFiSDAWrapperConfig{
		ALL_ALL_PARAMETERS: prifiParams,
		Identities:         idMapCopy,
		Role:               s.role,
		ClientSideSocksConfig: socksClientConfig,
		RelaySideSocksConfig:  socksServerConfig,
	}

	log.Error("Setting config to PriFi-SDA-Protocol")
	log.Lvlf1("%+v\n", configMsg)
	log.Lvlf1("%+v\n", configMsg.Identities)

	wrapper.SetConfig(configMsg)

	//when PriFi-protocol (via PriFi-lib) detects a slow client, call "handleTimeout"
	wrapper.SetTimeoutHandler(s.handleTimeout)

}

// NewProtocol is called on all nodes of a Tree (except the root, since it is
// the one starting the protocol) so it's the Service that will be called to
// generate the PI on all others node.
// If you use CreateProtocolSDA, this will not be called, as the SDA will
// instantiate the protocol on its own. If you need more control at the
// instantiation of the protocol, use CreateProtocolService, and you can
// give some extra-configuration to your protocol in here.
func (s *ServiceState) NewProtocol(tn *sda.TreeNodeInstance, conf *sda.GenericConfig) (sda.ProtocolInstance, error) {

	log.Lvl1("New protocol called...")

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

// tryLoad tries to load the configuration and updates if a configuration
// is found, else it returns an error.
func (s *ServiceState) tryLoad() error {
	configFile := s.path + "/identity.bin"
	b, err := ioutil.ReadFile(configFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Error while reading %s: %s", configFile, err)
	}
	if len(b) > 0 {
		_, msg, err := network.UnmarshalRegistered(b)
		if err != nil {
			return fmt.Errorf("Couldn't unmarshal: %s", err)
		}
		log.Lvl3("Successfully loaded")
		s.Storage = msg.(*Storage)
	}
	return nil
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
	c.RegisterProcessorFunc(network.TypeFromData(ConnectionRequest{}), s.HandleConnection)
	c.RegisterProcessorFunc(network.TypeFromData(DisconnectionRequest{}), s.HandleDisconnection)

	if err := s.tryLoad(); err != nil {
		log.Error(err)
	}
	return s
}

// mapIdentities reads the group configuration to assign PriFi roles
// to server addresses and returns them with the server
// identity of the relay.
func mapIdentities(group *config.Group) (map[string]prifi_protocol.PriFiIdentity, network.ServerIdentity) {
	m := make(map[string]prifi_protocol.PriFiIdentity)
	var relay network.ServerIdentity

	// Read the description of the nodes in the config file to assign them PriFi roles.
	nodeList := group.Roster.List
	for i := 0; i < len(nodeList); i++ {
		si := nodeList[i]
		nodeDescription := group.GetDescription(si)

		var id *prifi_protocol.PriFiIdentity

		if nodeDescription == "relay" {
			id = &prifi_protocol.PriFiIdentity{
				Role: prifi_protocol.Relay,
				ID:   0,
			}
		} else if nodeDescription == "trustee" {
			id = &prifi_protocol.PriFiIdentity{
				Role: prifi_protocol.Trustee,
				ID:   -1,
			}
		}

		if id != nil {
			m[si.Address.String()] = *id
			if id.Role == prifi_protocol.Relay {
				relay = *si
			}
		} else {
			log.Error("Cannot parse node description, skipping:", si)
		}

	}

	// Check that there is exactly one relay and at least one trustee and client
	t, r := 0, 0

	for _, v := range m {
		switch v.Role {
		case prifi_protocol.Relay:
			r++
		case prifi_protocol.Trustee:
			t++
		}
	}

	if !(t > 0 && r == 1) {
		log.Fatal("Config file does not contain exactly one relay, and at least one trustee.")
	}

	return m, relay
}

// readGroup reads the group description and sets up the Service struct fields
// accordingly. It *MUST* be called first when the node is started.
func (s *ServiceState) readGroup(group *config.Group) {
	IDs, relayID := mapIdentities(group)
	s.nodesAndIDs = &SDANodesAndIDs{
		identitiesMap: IDs,
		relayIdentity: &relayID,
		group:         group,
	}
}
