package services

// This file contains the logic to handle churn.

import (
	"sync"
	"time"

	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/network"
	"github.com/dedis/cothority/sda"
	"github.com/lbarman/prifi/sda/protocols"
)

//Delay before each host re-tried to connect
const DELAY_BEFORE_KEEPALIVE = 5 * time.Second

func init() {
	network.RegisterPacketType(StopProtocol{})
	network.RegisterPacketType(ConnectionRequest{})
	network.RegisterPacketType(DisconnectionRequest{})
}

//A wait queue entry is en identity(=address), and a flag IsWaiting (if false, we know about this host, but he didn't try to connect)
type WaitQueueEntry struct {
	IsWaiting bool
	Identity  *network.ServerIdentity
}

// waitQueue contains the list of nodes that are currently willing
// to participate to the protocol.
type waitQueue struct {
	mutex    sync.Mutex
	trustees map[string]*WaitQueueEntry
	clients  map[string]*WaitQueueEntry
}

// Packet send by relay when some node disconnected
type StopProtocol struct{}

// ConnectionRequest messages are sent to the relay
// by nodes that want to join the protocol.
type ConnectionRequest struct{}

// DisconnectionRequest messages are sent to the relay
// by nodes that want to leave the protocol.
type DisconnectionRequest struct{}

// Packet send by relay when some node disconnected
func (s *ServiceState) HandleStop(msg *network.Packet) {

	log.Lvl1("Received a Handle Stop")

	s.KillPriFiProtocol()

}

// HandleConnection receives connection requests from other nodes.
// It decides when another PriFi protocol should be started.
func (s *ServiceState) HandleConnection(msg *network.Packet) {
	addr := msg.ServerIdentity.Address.String()
	log.Lvl3("Received new connection request from ", addr)

	// If we are not the relay, ignore the message
	if s.role != protocols.Relay {
		return
	}

	s.nodesAndIDs.mutex.Lock()
	id, found := s.nodesAndIDs.identitiesMap[addr]
	s.nodesAndIDs.mutex.Unlock()
	if !found {
		log.Lvl2("New previously-unknown client :", addr)

		s.nodesAndIDs.mutex.Lock()
		newID := &protocols.PriFiIdentity{
			ID:   s.nodesAndIDs.nextFreeClientID,
			Role: protocols.Client,
		}
		s.nodesAndIDs.nextFreeClientID++
		s.nodesAndIDs.identitiesMap[addr] = *newID
		id = *newID
		s.nodesAndIDs.mutex.Unlock()
	}

	s.waitQueue.mutex.Lock()
	defer s.waitQueue.mutex.Unlock()

	nodeAlreadyIn := false

	// Add node to the waiting queue
	switch id.Role {
	case protocols.Client:
		if _, ok := s.waitQueue.clients[addr]; !ok {
			s.waitQueue.clients[addr] = &WaitQueueEntry{
				IsWaiting: true,
				Identity:  msg.ServerIdentity,
			}
		} else {
			s.waitQueue.clients[addr].IsWaiting = true
			nodeAlreadyIn = true
		}
	case protocols.Trustee:

		//assign an ID to this trustee
		if id.ID == -1 {
			s.nodesAndIDs.mutex.Lock()
			id.ID = s.nodesAndIDs.nextFreeTrusteeID
			s.nodesAndIDs.identitiesMap[addr] = protocols.PriFiIdentity{
				ID:   s.nodesAndIDs.nextFreeTrusteeID,
				Role: protocols.Trustee,
			}
			log.Lvl2("Trustee", msg.ServerIdentity.Address, "got assigned ID", id.ID)
			s.nodesAndIDs.nextFreeTrusteeID++
			s.nodesAndIDs.mutex.Unlock()
		}
		if _, ok := s.waitQueue.trustees[addr]; !ok {
			s.waitQueue.trustees[addr] = &WaitQueueEntry{
				IsWaiting: true,
				Identity:  msg.ServerIdentity,
			}
		} else {
			s.waitQueue.trustees[addr].IsWaiting = true
			nodeAlreadyIn = true
		}
	default:
		log.Error("Ignoring connection request from node with invalid role.")
	}

	// If the nodes is already participating we do not need to restart
	if nodeAlreadyIn && s.IsPriFiProtocolRunning() {
		return
	}

	// Start (or restart) PriFi if there are enough participants
	nWaitingClients := 0
	for _, v := range s.waitQueue.clients {
		if v.IsWaiting {
			nWaitingClients++
		}
	}
	nWaitingTrustees := 0
	for _, v := range s.waitQueue.trustees {
		if v.IsWaiting {
			nWaitingTrustees++
		}
	}
	if nWaitingClients >= 2 && nWaitingTrustees >= 1 {
		log.Lvl1("We meet the parameters, starting ...")
		if s.IsPriFiProtocolRunning() {
			log.Lvl1("We meet the parameters and not running ,starting...")
			s.stopPriFiCommunicateProtocol()
		}
		s.startPriFiCommunicateProtocol()
	} else {
		log.Lvl1("Too few participants (", nWaitingClients, "clients and", nWaitingTrustees, "trustees), waiting...")
	}
}

// HandleDisconnection receives disconnection requests.
// It must stop the current PriFi protocol.
func (s *ServiceState) HandleDisconnection(msg *network.Packet) {
	addr := msg.ServerIdentity.Address.String()
	log.Lvl2("Received disconnection request from ", msg.ServerIdentity.Address)

	// If we are not the relay, ignore the message
	if s.role != protocols.Relay {
		return
	}

	s.nodesAndIDs.mutex.Lock()
	id, ok := s.nodesAndIDs.identitiesMap[addr]
	s.nodesAndIDs.mutex.Unlock()

	if !ok {
		log.Info("Ignoring disconnection from unknown node:", addr)
		return
	}

	s.waitQueue.mutex.Lock()
	defer s.waitQueue.mutex.Unlock()

	// Remove node to the waiting queue
	switch id.Role {
	case protocols.Client:
		delete(s.waitQueue.clients, addr)
	case protocols.Trustee:
		delete(s.waitQueue.trustees, addr)
	default:
		log.Info("Ignoring disconnection request from node with invalid role.")
	}

	// Stop PriFi and restart if there are enough participants left.
	if s.IsPriFiProtocolRunning() {
		s.stopPriFiCommunicateProtocol()
	}

	if len(s.waitQueue.clients) >= 2 && len(s.waitQueue.trustees) >= 1 {
		s.startPriFiCommunicateProtocol()
	}
}

// sendConnectionRequest sends a connection request to the relay.
// It is called by the client and trustee services at startup to
// announce themselves to the relay.
func (s *ServiceState) sendConnectionRequest() {
	log.Lvl2("Sending connection request")
	err := s.SendRaw(s.nodesAndIDs.relayIdentity, &ConnectionRequest{})

	if err != nil {
		log.Error("Connection failed:", err)
	}
}

// autoConnect sends a connection request to the relay
// every 10 seconds if the node is not participating to
// a PriFi protocol.
func (s *ServiceState) autoConnect() {
	s.sendConnectionRequest()

	tick := time.Tick(DELAY_BEFORE_KEEPALIVE)
	for range tick {
		if !s.IsPriFiProtocolRunning() {
			s.sendConnectionRequest()
		}
	}
}

// handleTimeout is a callback that should be called on the relay
// when a round times out. It tries to restart PriFi with the nodes
// that sent their ciphertext in time.
func (s *ServiceState) handleTimeout(lateClients []string, lateTrustees []string) {
	s.waitQueue.mutex.Lock()
	defer s.waitQueue.mutex.Unlock()

	for _, v := range lateClients {
		delete(s.waitQueue.clients, v)
	}

	for _, v := range lateTrustees {
		delete(s.waitQueue.trustees, v)
	}

	s.stopPriFiCommunicateProtocol()

	if len(s.waitQueue.clients) >= 2 && len(s.waitQueue.trustees) >= 1 {
		s.startPriFiCommunicateProtocol()
	}
}

// startPriFi starts a PriFi protocol. It is called
// by the relay as soon as enough participants are
// ready (one trustee and two clients).
func (s *ServiceState) startPriFiCommunicateProtocol() {
	log.Lvl1("Starting PriFi protocol")

	if s.role != protocols.Relay {
		log.Error("Trying to start PriFi protocol from a non-relay node.")
		return
	}

	var wrapper *protocols.PriFiSDAProtocol

	participants := make([]*network.ServerIdentity, len(s.waitQueue.trustees)+len(s.waitQueue.clients)+1)
	participants[0] = s.nodesAndIDs.relayIdentity
	i := 1
	for _, v := range s.waitQueue.clients {
		participants[i] = v.Identity
		i++
	}
	for _, v := range s.waitQueue.trustees {
		participants[i] = v.Identity
		i++
	}

	roster := sda.NewRoster(participants)

	// Start the PriFi protocol on a flat tree with the relay as root
	tree := roster.GenerateNaryTreeWithRoot(100, s.nodesAndIDs.relayIdentity)
	pi, err := s.CreateProtocolService(protocols.ProtocolName, tree)

	if err != nil {
		log.Fatal("Unable to start Prifi protocol:", err)
	}

	// Assert that pi has type PriFiSDAWrapper
	wrapper = pi.(*protocols.PriFiSDAProtocol)

	//assign and start the protocol
	s.priFiSDAProtocol = wrapper

	s.setConfigToPriFiProtocol(wrapper)

	wrapper.Start()
}

// stopPriFi stops the PriFi protocol currently running.
func (s *ServiceState) stopPriFiCommunicateProtocol() {
	log.Lvl1("Stopping PriFi protocol")

	if s.role != protocols.Relay {
		log.Error("Trying to stop PriFi protocol from a non-relay node.")
		return
	}

	s.KillPriFiProtocol()
}

func (s *ServiceState) printWaitQueue() {
	log.Info("Current state of the wait queue:")

	log.Info("Clients:")
	for k, v := range s.waitQueue.clients {
		log.Info(k, "->", v)
	}
	log.Info("Trustees:")
	for k, v := range s.waitQueue.trustees {
		log.Info(k, "->", v)
	}
}

func (s *ServiceState) printIdentifiersMap() {
	s.nodesAndIDs.mutex.Lock()
	defer s.nodesAndIDs.mutex.Unlock()

	log.Info("Current state of the identity map:")

	for k, v := range s.nodesAndIDs.identitiesMap {
		switch v.Role {

		case protocols.Relay:
			log.Info(k, " -> Relay ", v.ID)
		case protocols.Client:
			log.Info(k, " -> Client ", v.ID)
		case protocols.Trustee:
			log.Info(k, " -> Trustee ", v.ID)
		}
	}
}
