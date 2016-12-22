package prifi

// This file contains the logic to handle churn.

import (
	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/network"
	"github.com/lbarman/prifi_dev/sda/protocols"
	"time"
	"github.com/dedis/cothority/sda"
	"sync"
)

func init() {
	network.RegisterPacketType(ConnectionRequest{})
	network.RegisterPacketType(DisconnectionRequest{})
}

// waitQueue contains the list of nodes that are currently willing
// to participate to the protocol.
type waitQueue struct {
	mutex sync.Mutex
	trustees map[*network.ServerIdentity]bool
	clients map[*network.ServerIdentity]bool
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
	log.Info("Received new connection request from ", msg.ServerIdentity.Address)

	// If we are not the relay, ignore the message
	if s.role != prifi.Relay {
		return
	}

	id, ok := s.identityMap[msg.ServerIdentity.Address]
	if !ok {
		log.Info("Ignoring connection from unknown node:", msg.ServerIdentity.Address)
		return
	}

	s.waitQueue.mutex.Lock()
	defer s.waitQueue.mutex.Unlock()

	nodeAlreadyIn := false

	// Add node to the waiting queue
	switch id.Role {
	case prifi.Client:
		if _, ok := s.waitQueue.clients[msg.ServerIdentity]; !ok {
			s.waitQueue.clients[msg.ServerIdentity] = true
		} else {
			nodeAlreadyIn  = true
		}
	case prifi.Trustee:
		if _, ok := s.waitQueue.trustees[msg.ServerIdentity]; !ok {
			s.waitQueue.trustees[msg.ServerIdentity] = true
		} else {
			nodeAlreadyIn = true
		}
	default:
		log.Info("Ignoring connection request from node with invalid role.")
	}

	// If the nodes is already participating we do not need to restart
	if nodeAlreadyIn && s.isPrifiRunning {
		return
	}

	// Start (or restart) PriFi if there are enough participants
	if len(s.waitQueue.clients) >= 2 && len(s.waitQueue.trustees) >= 1 {
		if s.isPrifiRunning {
			s.stopPriFi()
		}
		s.startPriFi()
	}
}

// HandleDisconnection receives disconnection requests.
// It must stop the current PriFi protocol.
func (s *Service) HandleDisconnection(msg *network.Packet)  {
	log.Lvl2("Received disconnection request from ", msg.ServerIdentity.Address)

	// If we are not the relay, ignore the message
	if s.role != prifi.Relay {
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
	case prifi.Client: delete(s.waitQueue.clients, msg.ServerIdentity)
	case prifi.Trustee: delete(s.waitQueue.trustees, msg.ServerIdentity)
	default: log.Info("Ignoring disconnection request from node with invalid role.")
	}

	// Stop PriFi and restart if there are enough participants left.
	if s.isPrifiRunning {
		s.stopPriFi()
	}

	if len(s.waitQueue.clients) >= 2 && len(s.waitQueue.trustees) >= 1 {
		s.startPriFi()
	}
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
// every 10 seconds if the node is not participating to
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

// handleTimeout is a callback that should be called on the relay
// when a round times out. It tries to restart PriFi with the nodes
// that sent their ciphertext in time.
func (s *Service) handleTimeout(lateClients []*network.ServerIdentity, lateTrustees []*network.ServerIdentity) {
	s.waitQueue.mutex.Lock()
	defer s.waitQueue.mutex.Unlock()

	for _, v := range lateClients {
		delete(s.waitQueue.clients, v)
	}

	for _, v := range lateTrustees {
		delete(s.waitQueue.trustees, v)
	}

	s.stopPriFi()

	if len(s.waitQueue.clients) >= 2 && len(s.waitQueue.trustees) >= 1 {
		s.startPriFi()
	}
}

// startPriFi starts a PriFi protocol. It is called
// by the relay as soon as enough participants are
// ready (one trustee and two clients).
func (s *Service) startPriFi() {
	log.Lvl1("startPriFi with nodes")

	if s.role != prifi.Relay {
		log.Error("Trying to start PriFi protocol from a non-relay node.")
		return
	}

	var wrapper *prifi.PriFiSDAWrapper

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
	pi, err := s.CreateProtocolService(prifi.ProtocolName, tree)

	if err != nil {
		log.Fatal("Unable to start Prifi protocol:", err)
	}

	// Assert that pi has type PriFiSDAWrapper
	wrapper = pi.(*prifi.PriFiSDAWrapper)
	s.prifiWrapper = wrapper

	wrapper.SetConfig(&prifi.PriFiSDAWrapperConfig{
		Identities: s.identityMap,
		Role: prifi.Relay,
	})
	wrapper.SetTimeoutHandler(s.handleTimeout)
	wrapper.Start()

	s.isPrifiRunning = true;
}

// stopPriFi stops the PriFi protocol currently running.
func (s *Service) stopPriFi() {
	log.Lvl1("Stopping PriFi protocol")

	if s.role != prifi.Relay {
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

func (s *Service) printWaitQueue() {
	log.Info("Current state of the wait queue:")

	log.Info("Clients:")
	for k := range s.waitQueue.clients {
		log.Info(k.Address)
	}
	log.Info("Trustees:")
	for k := range s.waitQueue.trustees {
		log.Info(k.Address)
	}
}
