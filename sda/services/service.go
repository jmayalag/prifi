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
	"github.com/lbarman/prifi/sda/protocols"

	prifi "github.com/lbarman/prifi/prifi-lib"
	socks "github.com/lbarman/prifi/prifi-socks"
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
type PriFiConfig struct {
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
	identitiesMap     map[network.Address]protocols.PriFiIdentity
	relayIdentity     *network.ServerIdentity
	group             *config.Group
	nextFreeClientID  int
	nextFreeTrusteeID int
}

//Set the config, from the prifi.toml. Is called by sda/app.
func (s *Service) SetConfig(config *PriFiConfig) {
	log.Lvl3("Setting PriFi configuration...")
	log.Lvlf3("%+v\n", config)
	s.prifiConfig = config
}

//Service contains the state of the service
type Service struct {
	// We need to embed the ServiceProcessor, so that incoming messages
	// are correctly handled.
	*sda.ServiceProcessor
	prifiConfig    *PriFiConfig
	Storage        *Storage
	path           string
	role           protocols.PriFiRole
	nodesAndIDs    *SDANodesAndIDs
	waitQueue      *waitQueue
	prifiWrapper   *protocols.PriFiSDAWrapper
	isPrifiRunning bool
}

// Storage will be saved, on the contrary of the 'Service'-structure
// which has per-service information stored.
type Storage struct {
	TrusteeID string
}

// This is a handler passed to the SDA when starting a host. The SDA usually handle all the network by itself,
// but in our case it is useful to know when a network RESET occured, so we can kill protocols (otherwise they
// remain in some weird state)
func (s *Service) NetworkErrorHappened(e error) {
	log.Error("Error occurred, killing PriFi protocol")
	return
}

// StartTrustee starts the necessary
// protocols to enable the trustee-mode.
func (s *Service) StartTrustee(group *config.Group) error {
	log.Info("Service", s, "running in trustee mode")
	s.role = protocols.Trustee
	s.readGroup(group)

	go s.autoConnect()

	return nil
}

// StartRelay starts the necessary
// protocols to enable the relay-mode.
// In this example it simply starts the demo protocol
func (s *Service) StartRelay(group *config.Group) error {
	log.Info("Service", s, "running in relay mode")
	s.role = protocols.Relay
	s.readGroup(group)
	s.waitQueue = &waitQueue{
		clients:  make(map[*network.ServerIdentity]bool),
		trustees: make(map[*network.ServerIdentity]bool),
	}

	socksServerConfig = &protocols.SOCKSConfig{
		Port:              "127.0.0.1:" + strconv.Itoa(s.prifiConfig.SocksClientPort),
		PayloadLength:     s.prifiConfig.CellSizeUp,
		UpstreamChannel:   make(chan []byte),
		DownstreamChannel: make(chan []byte),
	}

	go socks.StartSocksClient(socksServerConfig.Port, socksServerConfig.UpstreamChannel, socksServerConfig.DownstreamChannel)

	return nil
}

var socksClientConfig *protocols.SOCKSConfig
var socksServerConfig *protocols.SOCKSConfig

// StartClient starts the necessary
// protocols to enable the client-mode.
func (s *Service) StartClient(group *config.Group) error {
	log.Info("Service", s, "running in client mode")
	s.role = protocols.Client
	s.readGroup(group)

	socksClientConfig = &protocols.SOCKSConfig{
		Port:              ":" + strconv.Itoa(s.prifiConfig.SocksServerPort),
		PayloadLength:     s.prifiConfig.CellSizeUp,
		UpstreamChannel:   make(chan []byte),
		DownstreamChannel: make(chan []byte),
	}

	go socks.StartSocksServer(socksClientConfig.Port, socksClientConfig.PayloadLength, socksClientConfig.UpstreamChannel, socksClientConfig.DownstreamChannel, s.prifiConfig.DoLatencyTests)

	go s.autoConnect()

	return nil
}

// StartClient starts the necessary
// protocols to enable the client-mode.
func (s *Service) StartSocksTunnelOnly() error {
	log.Info("Service", s, "running in socks-tunnel-only mode")

	socksClientConfig = &protocols.SOCKSConfig{
		Port:              ":" + strconv.Itoa(s.prifiConfig.SocksServerPort),
		PayloadLength:     s.prifiConfig.CellSizeUp,
		UpstreamChannel:   make(chan []byte),
		DownstreamChannel: make(chan []byte),
	}

	socksServerConfig = &protocols.SOCKSConfig{
		Port:              "127.0.0.1:" + strconv.Itoa(s.prifiConfig.SocksClientPort),
		PayloadLength:     s.prifiConfig.CellSizeUp,
		UpstreamChannel:   socksClientConfig.UpstreamChannel,
		DownstreamChannel: socksClientConfig.DownstreamChannel,
	}
	go socks.StartSocksServer(socksClientConfig.Port, socksClientConfig.PayloadLength, socksClientConfig.UpstreamChannel, socksClientConfig.DownstreamChannel, false)
	go socks.StartSocksClient(socksServerConfig.Port, socksServerConfig.UpstreamChannel, socksServerConfig.DownstreamChannel)

	return nil
}

func (s *Service) setConfigToPriFiProtocol(wrapper *protocols.PriFiSDAWrapper) {

	log.Lvl1("setConfigToPriFiProtocol called")
	log.Lvlf1("%+v\n", s.prifiConfig)

	prifiParams := prifi.ALL_ALL_PARAMETERS{
		ClientDataOutputEnabled: true,
		DoLatencyTests:          s.prifiConfig.DoLatencyTests,
		DownCellSize:            s.prifiConfig.CellSizeDown,
		ForceParams:             true,
		NClients:                -1, //computer later
		NextFreeClientID:        0,
		NextFreeTrusteeID:       0,
		NTrustees:               -1, //computer later
		RelayDataOutputEnabled:  true,
		RelayReportingLimit:     s.prifiConfig.RelayReportingLimit,
		RelayUseDummyDataDown:   s.prifiConfig.RelayUseDummyDataDown,
		RelayWindowSize:         s.prifiConfig.RelayWindowSize,
		StartNow:                false,
		UpCellSize:              s.prifiConfig.CellSizeUp,
		UseUDP:                  s.prifiConfig.UseUDP,
	}

	//deep-clone the identityMap
	s.nodesAndIDs.mutex.Lock()
	idMapCopy := make(map[network.Address]protocols.PriFiIdentity)
	for k, v := range s.nodesAndIDs.identitiesMap {
		idMapCopy[k] = protocols.PriFiIdentity{
			ID:   v.ID,
			Role: v.Role,
		}
	}
	s.nodesAndIDs.mutex.Unlock()

	wrapper.SetConfig(&protocols.PriFiSDAWrapperConfig{
		ALL_ALL_PARAMETERS: prifiParams,
		Identities:         idMapCopy,
		Role:               s.role,
		ClientSideSocksConfig: socksClientConfig,
		RelaySideSocksConfig:  socksServerConfig,
	})

}

// NewProtocol is called on all nodes of a Tree (except the root, since it is
// the one starting the protocol) so it's the Service that will be called to
// generate the PI on all others node.
// If you use CreateProtocolSDA, this will not be called, as the SDA will
// instantiate the protocol on its own. If you need more control at the
// instantiation of the protocol, use CreateProtocolService, and you can
// give some extra-configuration to your protocol in here.
func (s *Service) NewProtocol(tn *sda.TreeNodeInstance, conf *sda.GenericConfig) (sda.ProtocolInstance, error) {
	log.Lvl5("Setting node configuration from service")

	pi, err := protocols.NewPriFiSDAWrapperProtocol(tn)
	if err != nil {
		return nil, err
	}

	wrapper := pi.(*protocols.PriFiSDAWrapper)
	s.isPrifiRunning = true
	s.setConfigToPriFiProtocol(wrapper)

	wrapper.Running = &s.isPrifiRunning

	return wrapper, nil
}

// save saves the actual identity
func (s *Service) save() {
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
func (s *Service) tryLoad() error {
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
	log.Lvl4("Calling newService")
	s := &Service{
		ServiceProcessor: sda.NewServiceProcessor(c),
		path:             path,
		isPrifiRunning:   false,
	}

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
func mapIdentities(group *config.Group) (map[network.Address]protocols.PriFiIdentity, network.ServerIdentity) {
	m := make(map[network.Address]protocols.PriFiIdentity)
	var relay network.ServerIdentity

	// Read the description of the nodes in the config file to assign them PriFi roles.
	nodeList := group.Roster.List
	for i := 0; i < len(nodeList); i++ {
		si := nodeList[i]
		nodeDescription := group.GetDescription(si)

		var id *protocols.PriFiIdentity

		if nodeDescription == "relay" {
			id = &protocols.PriFiIdentity{
				Role: protocols.Relay,
				ID:   0,
			}
		} else if nodeDescription == "trustee" {
			id = &protocols.PriFiIdentity{
				Role: protocols.Trustee,
				ID:   -1,
			}
		}

		if id != nil {
			m[si.Address] = *id
			if id.Role == protocols.Relay {
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
		case protocols.Relay:
			r++
		case protocols.Trustee:
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
func (s *Service) readGroup(group *config.Group) {
	IDs, relayID := mapIdentities(group)
	s.nodesAndIDs = &SDANodesAndIDs{
		identitiesMap: IDs,
		relayIdentity: &relayID,
		group:         group,
	}
}
