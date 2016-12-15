// Package prifi-sda-service contains the SDA service responsible
// for starting the SDA protocols required to enable PriFi
// communications.
package services

/*
* This is the internal part of the API. As probably the prifi-service will
* not have an external API, this will not have any API-functions.
 */

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"github.com/dedis/cothority/app/lib/config"
	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/network"
	"github.com/dedis/cothority/sda"
	"github.com/lbarman/prifi_dev/sda/protocols"
	"sync"
	"time"

	socks "github.com/lbarman/prifi_dev/prifi-socks"
)

const ServiceName = "PriFiService"

var serviceID sda.ServiceID

// Register Service with SDA
func init() {
	sda.RegisterNewService(ServiceName, newService)
	serviceID = sda.ServiceFactory.ServiceID(ServiceName)
	network.RegisterPacketType(ConnectionRequest{})
	network.RegisterPacketType(DisconnectionRequest{})
}


type PriFiconfig struct {
	CellSizeUp            int
	CellSizeDown          int
	RelayWindowSize       int
	RelayUseDummyDataDown bool
	RelayReportingLimit   int
	UseUDP                bool
	DoLatencyTests        bool
	SocksServerPort		int
	SocksClientPort		int
}

func (s *Service) SetConfig(config *PriFiconfig) {
	log.Lvlf3("%+v\n", config)
}

// waitQueue contains the list of nodes that are currently willing
// to participate to the protocol.
type waitQueue struct {
	mutex sync.Mutex
	trustees map[*network.ServerIdentity]bool
	clients map[*network.ServerIdentity]bool
}

// Service contains the state of the service
type Service struct {
	// We need to embed the ServiceProcessor, so that incoming messages
	// are correctly handled.
	*sda.ServiceProcessor
	group   *config.Group
	Storage *Storage
	path    string
	role protocols.PriFiRole
	identityMap map[network.Address]protocols.PriFiIdentity
	relayIdentity *network.ServerIdentity
	waitQueue *waitQueue
	prifiWrapper *protocols.PriFiSDAWrapper
	isPrifiRunning bool
}

// Storage will be saved, on the contrary of the 'Service'-structure
// which has per-service information stored.
type Storage struct {
	TrusteeID string
}

// ConnectionRequest messages are sent to the relay
// by nodes that want to join the protocol.
type ConnectionRequest struct {}

// DisconnectionRequest messages are sent to the relay
// by nodes that want to leave the protocol.
type DisconnectionRequest struct {}

// HandleConnection receives connection requests from other nodes.
// It decides when another PriFi protocol should be started.
func (s *Service) HandleConnection(msg *network.Packet) {
	log.Lvl3("Received new connection request from ", msg.ServerIdentity.Address)

	// If we are not the relay, ignore the message
	if s.role != protocols.Relay {
		return
	}

	id, ok := s.identityMap[msg.ServerIdentity.Address]
	if !ok {
		log.Info("Ignoring connection from unknown node:", msg.ServerIdentity.Address)
		return
	}

	s.waitQueue.mutex.Lock()
	defer s.waitQueue.mutex.Unlock()

	// Add node to the waiting queue
	switch id.Role {
	case protocols.Client:
		if _, ok := s.waitQueue.clients[msg.ServerIdentity]; !ok {
			s.waitQueue.clients[msg.ServerIdentity] = true
		}
	case protocols.Trustee:
		if _, ok := s.waitQueue.trustees[msg.ServerIdentity]; !ok {
			s.waitQueue.trustees[msg.ServerIdentity] = true
		}
	default:
		log.Info("Ignoring connection request from node with invalid role.")
	}

	// Start (or restart) PriFi if there are enough participants
	if len(s.waitQueue.clients) >= 2 && len(s.waitQueue.trustees) >= 1 {
		if s.isPrifiRunning {
			s.stopPriFiCommunicateProtocol()
		}
		s.startPriFiCommunicateProtocol()
	}
}

// HandleDisconnection receives disconnection requests.
// It must stop the current PriFi protocol.
func (s *Service) HandleDisconnection(msg *network.Packet)  {
	log.Lvl2("Received disconnection request from ", msg.ServerIdentity.Address)

	// If we are not the relay, ignore the message
	if s.role != protocols.Relay {
		return
	}

	id, ok := s.identityMap[msg.ServerIdentity.Address]
	if !ok {
		log.Info("Ignoring connection from unknown node:", msg.ServerIdentity.Address)
		return
	}

	s.waitQueue.mutex.Lock()
	defer s.waitQueue.mutex.Unlock()

	// Remove node to the waiting queue
	switch id.Role {
	case protocols.Client: delete(s.waitQueue.clients, msg.ServerIdentity)
	case protocols.Trustee: delete(s.waitQueue.trustees, msg.ServerIdentity)
	default: log.Info("Ignoring disconnection request from node with invalid role.")
	}

	// Stop PriFi and restart if there are enough participants left.
	if s.isPrifiRunning {
		s.stopPriFiCommunicateProtocol()
	}

	if len(s.waitQueue.clients) >= 2 && len(s.waitQueue.trustees) >= 1 {
		s.startPriFiCommunicateProtocol()
	}
}

// StartTrustee starts the necessary
// protocols to enable the trustee-mode.
func (s *Service) StartTrustee(group *config.Group) error {
	log.Info("Service", s, "running in trustee mode")
	s.role = protocols.Trustee
	s.readGroup(group)

	s.autoConnect()

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
		clients: make(map[*network.ServerIdentity]bool),
		trustees: make(map[*network.ServerIdentity]bool),
	}

	socksServerConfig = &protocols.SOCKSConfig{
		Port: "127.0.0.1:8081",
		PayloadLength: 100, //todo : this is wrong
		UpstreamChannel: make(chan []byte),
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
		Port: ":6789",
		PayloadLength: 100, //todo : this is wrong
		UpstreamChannel: make(chan []byte),
		DownstreamChannel: make(chan []byte),
	}

	go socks.StartSocksServer(socksClientConfig.Port, socksClientConfig.PayloadLength, socksClientConfig.UpstreamChannel, socksClientConfig.DownstreamChannel)

	s.autoConnect()

	return nil
}


// StartClient starts the necessary
// protocols to enable the client-mode.
func (s *Service) StartSocksTunnelOnly() error {
	log.Info("Service", s, "running in socks-tunnel-only mode")

	socksClientConfig = &protocols.SOCKSConfig{
		Port: ":6789",
		PayloadLength: 100,
		UpstreamChannel: make(chan []byte),
		DownstreamChannel: make(chan []byte),
	}

	socksServerConfig = &protocols.SOCKSConfig{
		Port: "127.0.0.1:8081",
		PayloadLength: 100,
		UpstreamChannel: socksClientConfig.UpstreamChannel,
		DownstreamChannel: socksClientConfig.DownstreamChannel,
	}
	go socks.StartSocksServer(socksClientConfig.Port, socksClientConfig.PayloadLength, socksClientConfig.UpstreamChannel, socksClientConfig.DownstreamChannel)
	go socks.StartSocksClient(socksServerConfig.Port, socksServerConfig.UpstreamChannel, socksServerConfig.DownstreamChannel)

	return nil
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

	// Assert that pi has type PriFiSDAWrapper
	wrapper := pi.(*protocols.PriFiSDAWrapper)

	s.isPrifiRunning = true

	wrapper.SetConfig(&protocols.PriFiSDAWrapperConfig{
		Identities: s.identityMap,
		Role: s.role,
		ClientSideSocksConfig: socksClientConfig,
		RelaySideSocksConfig: socksServerConfig,
	})

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

// parseDescription extracts a PriFiIdentity from a string
func parseDescription(description string) (*protocols.PriFiIdentity, error) {
	desc := strings.Split(description, " ")
	if len(desc) == 1 && desc[0] == "relay" {
		return &protocols.PriFiIdentity{
			Role: protocols.Relay,
			Id:   0,
		}, nil
	} else if len(desc) == 2 {
		id, err := strconv.Atoi(desc[1])
		if err != nil {
			return nil, errors.New("Unable to parse id:")
		} else {
			pid := protocols.PriFiIdentity{
				Id: id,
			}
			if desc[0] == "client" {
				pid.Role = protocols.Client
			} else if desc[0] == "trustee" {
				pid.Role = protocols.Trustee
			} else {
				return nil, errors.New("Invalid role.")
			}
			return &pid, nil
		}
	} else {
		return nil, errors.New("Invalid description.")
	}
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
		id, err := parseDescription(group.GetDescription(si))
		if err != nil {
			log.Info("Cannot parse node description, skipping:", err)
		} else {
			m[si.Address] = *id
			if id.Role == protocols.Relay {
				relay = *si
			}
		}
	}

	// Check that there is exactly one relay and at least one trustee and client
	t, c, r := 0, 0, 0

	for _, v := range m {
		switch v.Role {
		case protocols.Relay:
			r++
		case protocols.Client:
			c++
		case protocols.Trustee:
			t++
		}
	}

	if !(t > 0 && c > 0 && r == 1) {
		log.Fatal("Config file does not contain exactly one relay and at least one trustee and client.")
	}

	return m, relay
}

// readGroup reads the group description and sets up the Service struct fields
// accordingly. It *MUST* be called first when the node is started.
func (s *Service) readGroup(group *config.Group) {
	ids, relayId := mapIdentities(group)
	s.identityMap = ids
	s.relayIdentity = &relayId
	s.group = group
}

// sendConnectionRequest sends a connection request to the relay.
// It is called by the client and trustee services at startup to
// announce themselves to the relay.
func (s *Service) sendConnectionRequest() {
	log.Lvl2("Sending connection request")
	err := s.SendRaw(s.relayIdentity, &ConnectionRequest{})

	if err != nil {
		log.Error("Connection failed:", err)
	}
}

// autoConnect sends a connection request to the relay
// every 10 seconds if node is not participating to
// a PriFi protocol.
func (s *Service) autoConnect() {
	s.sendConnectionRequest()

	tick := time.Tick(10 * time.Second)
	for range tick {
		if !s.isPrifiRunning {
			s.sendConnectionRequest()
		}
	}
}

// startPriFi starts a PriFi protocol. It is called
// by the relay as soon as enough participants are
// ready (one trustee and two clients).
func (s *Service) startPriFiCommunicateProtocol() {
	log.Lvl1("Starting PriFi protocol")

	if s.role != protocols.Relay {
		log.Error("Trying to start PriFi protocol from a non-relay node.")
		return
	}

	var wrapper *protocols.PriFiSDAWrapper

	participants := make([]*network.ServerIdentity, len(s.waitQueue.trustees) + len(s.waitQueue.clients) + 1)
	participants[0] = s.relayIdentity
	i := 1;
	for k := range s.waitQueue.clients {
		participants[i] = k
		i++
	}
	for k := range s.waitQueue.trustees {
		participants[i] = k
		i++
	}

	roster := sda.NewRoster(participants)

	// Start the PriFi protocol on a flat tree with the relay as root
	tree := roster.GenerateNaryTreeWithRoot(100, s.relayIdentity)
	pi, err := s.CreateProtocolService(protocols.ProtocolName, tree)

	if err != nil {
		log.Fatal("Unable to start Prifi protocol:", err)
	}

	// Assert that pi has type PriFiSDAWrapper
	wrapper = pi.(*protocols.PriFiSDAWrapper)
	s.prifiWrapper = wrapper


	wrapper.SetConfig(&protocols.PriFiSDAWrapperConfig{
		Identities: s.identityMap,
		Role: s.role,
		ClientSideSocksConfig: socksClientConfig, //this is nil, it's ok
		RelaySideSocksConfig: socksServerConfig,
	})
	wrapper.Start()

	s.isPrifiRunning = true;
}

// stopPriFi stops the PriFi protocol currently running.
func (s *Service) stopPriFiCommunicateProtocol() {
	log.Lvl1("Stopping PriFi protocol")

	if s.role != protocols.Relay {
		log.Error("Trying to stop PriFi protocol from a non-relay node.")
		return
	}

	if !s.isPrifiRunning || s.prifiWrapper == nil {
		log.Error("Trying to stop PriFi protocol but it has not started.")
		return
	}

	s.prifiWrapper.Stop()
	s.prifiWrapper = nil
	s.isPrifiRunning = false;
}
