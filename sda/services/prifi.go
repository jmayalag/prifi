package services

import (
	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/network"
	prifi_protocol "github.com/lbarman/prifi/sda/protocols"
	"time"
)

// Packet send by relay when some node disconnected
type StopProtocol struct{}

// ConnectionRequest messages are sent to the relay
// by nodes that want to join the protocol.
type ConnectionRequest struct{}

// DisconnectionRequest messages are sent to the relay
// by nodes that want to leave the protocol.
type DisconnectionRequest struct{}

func init() {
	network.RegisterPacketType(StopProtocol{})
	network.RegisterPacketType(ConnectionRequest{})
	network.RegisterPacketType(DisconnectionRequest{})
}

// returns true if the PriFi SDA protocol is running (in any state : init, communicate, etc)
func (s *ServiceState) IsPriFiProtocolRunning() bool {
	if s.priFiSDAProtocol != nil {
		return !s.priFiSDAProtocol.HasStopped
	}
	return false
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
