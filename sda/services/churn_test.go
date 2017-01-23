package services

import (
	"github.com/lbarman/prifi/prifi-lib/crypto"
	"github.com/lbarman/prifi/sda/protocols"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
	"strconv"
	"testing"
)

func genSI(addrPort string) *network.ServerIdentity {
	pub, _ := crypto.NewKeyPair()
	addr := network.NewAddress(network.Local, addrPort)
	return network.NewServerIdentity(pub, addr)
}

func genPacketFromSource(source *network.ServerIdentity) *network.Packet {
	return &network.Packet{
		ServerIdentity: source,
	}
}

func TestIDHasing(t *testing.T) {

	si := genSI("127.0.0.1:1")
	id1 := idFromServerIdentity(si)
	if id1 == "" {
		t.Error("idFromServerIdentity can't return nil on a valid ServerIdentity")
	}

	packet := genPacketFromSource(si)

	id2 := idFromMsg(packet)
	if id2 == "" {
		t.Error("idFromMsg can't return nil on a valid message")
	}

	if id1 != id2 {
		t.Error("idFromMsg and idFromServerIdentity must return the same value on the same ServerID")
	}
}

var stopProtocolCalled bool = false
var startProtocolCalled bool = false

func stopProtocol() {
	stopProtocolCalled = true
}
func startProtocol() {
	startProtocolCalled = true
}

func testIfInRoster(roster *onet.Roster, ID *network.ServerIdentity) bool {
	for _, v := range roster.List {
		if v.Equal(ID) {
			return true
		}
	}
	return false
}

func testIfInIDMap(idmap map[string]protocols.PriFiIdentity, ID *network.ServerIdentity) bool {
	for _, v := range idmap {
		if v.ServerID.Equal(ID) {
			return true
		}
	}
	return false
}

func testIDMapForCollisions(idmap map[string]protocols.PriFiIdentity) bool {
	clients := make([]bool, len(idmap))
	trustees := make([]bool, len(idmap))
	relay := false

	for _, v := range idmap {
		if v.Role == protocols.Client {
			if clients[v.ID] {
				return false
			}
			clients[v.ID] = true
		}
		if v.Role == protocols.Trustee {
			if trustees[v.ID] {
				return false
			}
			trustees[v.ID] = true
		}
		if v.Role == protocols.Relay {
			if relay {
				return false
			}
			relay = true
		}
	}

	//detect holes in clients map
	hasSeenFirstFalse := false
	for _, v := range clients {
		if v && !hasSeenFirstFalse {
			continue
		}
		if v && hasSeenFirstFalse {
			return false
		}
		hasSeenFirstFalse = true
	}

	//detect holes in trustees map
	hasSeenFirstFalse = false
	for _, v := range trustees {
		if v && !hasSeenFirstFalse {
			continue
		}
		if v && hasSeenFirstFalse {
			return false
		}
		hasSeenFirstFalse = true
	}

	return true
}

func TestChurn(t *testing.T) {

	//gen some IDs
	relayID := genSI("127.0.0.0:1")
	trustees := make([]*network.ServerIdentity, 3)
	for i := 0; i < len(trustees); i++ {
		trustees[i] = genSI("0.127.0.0:" + strconv.Itoa(i))
	}
	clients := make([]*network.ServerIdentity, 3)
	for i := 0; i < len(clients); i++ {
		clients[i] = genSI("0.0.127.0:" + strconv.Itoa(i))
	}

	//init the struct
	c := new(churnHandler)
	c.init(relayID, trustees)
	c.stopProtocol = stopProtocol
	c.startProtocol = startProtocol
	c.isProtocolRunning = func() bool { return false }

	//if test that trustees are correctly recognized
	if c.isATrustee(relayID) {
		t.Error("Relay shouldn't be considered as a trustee")
	}
	for i := 0; i < len(trustees); i++ {
		if !c.isATrustee(trustees[i]) {
			t.Error("Trustee " + strconv.Itoa(i) + " should be considered as a trustee")
		}
	}
	for i := 0; i < len(clients); i++ {
		if c.isATrustee(clients[i]) {
			t.Error("Clients " + strconv.Itoa(i) + " shouldn't be considered as a trustee")
		}
	}

	//test the number of waiting entities, should be 0
	nClients, nTrustees := c.waitQueue.count()
	if nClients != 0 {
		t.Error("nClients should be 0, is", nClients)
	}
	if nTrustees != 0 {
		t.Error("nTrustees should be 0, is", nTrustees)
	}
	if startProtocolCalled {
		t.Error("Protocol should not have started at that point")
	}
	if stopProtocolCalled {
		t.Error("Protocol should not have been stopped at that point")
	}
	roster := c.createRoster()
	if len(roster.List) != 1 {
		t.Error("Roster should have length 1")
	}
	if !testIfInRoster(roster, relayID) {
		t.Error("Relay should be in roster")
	}
	idMap := c.createIdentitiesMap()
	if len(idMap) != len(roster.List) {
		t.Error("IDmap should have the same length as the roster")
	}
	if !testIfInIDMap(idMap, relayID) {
		t.Error("Relay should be in idMap")
	}
	if !testIDMapForCollisions(idMap) {
		t.Error("Something is wrong in the ID map")
		log.Lvlf1("%+v", idMap)
	}

	//add one trustee
	c.handleConnection(genPacketFromSource(trustees[0]))
	nClients, nTrustees = c.waitQueue.count()
	if nClients != 0 {
		t.Error("nClients should be 0, is", nClients)
	}
	if nTrustees != 1 {
		t.Error("nTrustees should be 1, is", nTrustees)
	}
	if startProtocolCalled {
		t.Error("Protocol should not have started at that point")
	}
	if stopProtocolCalled {
		t.Error("Protocol should not have been stopped at that point")
	}
	roster = c.createRoster()
	if len(roster.List) != 2 {
		t.Error("Roster should have length 2")
	}
	if !testIfInRoster(roster, relayID) {
		t.Error("Relay should be in roster")
	}
	if !testIfInRoster(roster, trustees[0]) {
		t.Error("Trustee 0 should be in roster")
	}
	idMap = c.createIdentitiesMap()
	if len(idMap) != len(roster.List) {
		t.Error("IDmap should have the same length as the roster")
	}
	if !testIfInIDMap(idMap, relayID) {
		t.Error("Relay should be in idMap")
	}
	if !testIfInIDMap(idMap, trustees[0]) {
		t.Error("Trustee 0 should be in idMap")
	}
	if !testIDMapForCollisions(idMap) {
		t.Error("Something is wrong in the ID map")
		log.Lvlf1("%+v", idMap)
	}
	stopProtocolCalled = false
	startProtocolCalled = false

	//add the same trustee
	c.handleConnection(genPacketFromSource(trustees[0]))
	nClients, nTrustees = c.waitQueue.count()
	if nClients != 0 {
		t.Error("nClients should be 0, is", nClients)
	}
	if nTrustees != 1 {
		t.Error("nTrustees should be 1, is", nTrustees)
	}
	if startProtocolCalled {
		t.Error("Protocol should not have started at that point")
	}
	if stopProtocolCalled {
		t.Error("Protocol should not have been stopped at that point")
	}
	roster = c.createRoster()
	if len(roster.List) != 2 {
		t.Error("Roster should have length 2")
	}
	if !testIfInRoster(roster, relayID) {
		t.Error("Relay should be in roster")
	}
	if !testIfInRoster(roster, trustees[0]) {
		t.Error("Trustee 0 should be in roster")
	}
	idMap = c.createIdentitiesMap()
	if len(idMap) != len(roster.List) {
		t.Error("IDmap should have the same length as the roster")
	}
	if !testIfInIDMap(idMap, relayID) {
		t.Error("Relay should be in idMap")
	}
	if !testIfInIDMap(idMap, trustees[0]) {
		t.Error("Trustee 0 should be in idMap")
	}
	if !testIDMapForCollisions(idMap) {
		t.Error("Something is wrong in the ID map")
		log.Lvlf1("%+v", idMap)
	}
	stopProtocolCalled = false
	startProtocolCalled = false

	//remove this trustee
	c.handleDisconnection(genPacketFromSource(trustees[0]))
	nClients, nTrustees = c.waitQueue.count()
	if nClients != 0 {
		t.Error("nClients should be 0, is", nClients)
	}
	if nTrustees != 0 {
		t.Error("nTrustees should be 0, is", nTrustees)
	}
	if startProtocolCalled {
		t.Error("Protocol should not have started at that point")
	}
	if !stopProtocolCalled {
		t.Error("Protocol should have been stopped, we had a disconnection")
	}
	roster = c.createRoster()
	if len(roster.List) != 1 {
		t.Error("Roster should have length 1")
	}
	if !testIfInRoster(roster, relayID) {
		t.Error("Relay should be in roster")
	}
	idMap = c.createIdentitiesMap()
	if len(idMap) != len(roster.List) {
		t.Error("IDmap should have the same length as the roster")
	}
	if !testIfInIDMap(idMap, relayID) {
		t.Error("Relay should be in idMap")
	}
	if !testIDMapForCollisions(idMap) {
		t.Error("Something is wrong in the ID map")
		log.Lvlf1("%+v", idMap)
	}
	stopProtocolCalled = false
	startProtocolCalled = false

	//add one trustee
	c.handleConnection(genPacketFromSource(trustees[1]))
	nClients, nTrustees = c.waitQueue.count()
	if nClients != 0 {
		t.Error("nClients should be 0, is", nClients)
	}
	if nTrustees != 1 {
		t.Error("nTrustees should be 1, is", nTrustees)
	}
	if startProtocolCalled {
		t.Error("Protocol should not have started at that point")
	}
	if stopProtocolCalled {
		t.Error("Protocol should not have been stopped at that point")
	}
	roster = c.createRoster()
	if len(roster.List) != 2 {
		t.Error("Roster should have length 2")
	}
	if !testIfInRoster(roster, relayID) {
		t.Error("Relay should be in roster")
	}
	if !testIfInRoster(roster, trustees[1]) {
		t.Error("Trustee 1 should be in roster")
	}
	idMap = c.createIdentitiesMap()
	if len(idMap) != len(roster.List) {
		t.Error("IDmap should have the same length as the roster")
	}
	if !testIfInIDMap(idMap, relayID) {
		t.Error("Relay should be in idMap")
	}
	if !testIfInIDMap(idMap, trustees[1]) {
		t.Error("Trustee 1 should be in idMap")
	}
	if !testIDMapForCollisions(idMap) {
		t.Error("Something is wrong in the ID map")
		log.Lvlf1("%+v", idMap)
	}
	stopProtocolCalled = false
	startProtocolCalled = false

	//add one client
	c.handleConnection(genPacketFromSource(clients[0]))
	nClients, nTrustees = c.waitQueue.count()
	if nClients != 1 {
		t.Error("nClients should be 1, is", nClients)
	}
	if nTrustees != 1 {
		t.Error("nTrustees should be 1, is", nTrustees)
	}
	if !startProtocolCalled {
		t.Error("Protocol should have started at that point (1 client, 1 trustee)")
	}
	if stopProtocolCalled {
		t.Error("Protocol should not have been stopped at that point")
	}
	roster = c.createRoster()
	if len(roster.List) != 3 {
		t.Error("Roster should have length 3")
	}
	if !testIfInRoster(roster, relayID) {
		t.Error("Relay should be in roster")
	}
	if !testIfInRoster(roster, trustees[1]) {
		t.Error("Trustee 1 should be in roster")
	}
	if !testIfInRoster(roster, clients[0]) {
		t.Error("Client 0 should be in roster")
	}
	idMap = c.createIdentitiesMap()
	if len(idMap) != len(roster.List) {
		t.Error("IDmap should have the same length as the roster")
	}
	if !testIfInIDMap(idMap, relayID) {
		t.Error("Relay should be in idMap")
	}
	if !testIfInIDMap(idMap, trustees[1]) {
		t.Error("Trustee 1 should be in idMap")
	}
	if !testIfInIDMap(idMap, clients[0]) {
		t.Error("Client 0 should be in idMap")
	}
	if !testIDMapForCollisions(idMap) {
		t.Error("Something is wrong in the ID map")
		log.Lvlf1("%+v", idMap)
	}
	stopProtocolCalled = false
	startProtocolCalled = false
	c.isProtocolRunning = func() bool { return true } //protocol is now running

	//add one client
	c.handleConnection(genPacketFromSource(clients[1]))
	nClients, nTrustees = c.waitQueue.count()
	if nClients != 2 {
		t.Error("nClients should be 2, is", nClients)
	}
	if nTrustees != 1 {
		t.Error("nTrustees should be 1, is", nTrustees)
	}
	if !stopProtocolCalled {
		t.Error("Protocol should have been stopped at that point, new client connected")
	}
	if !startProtocolCalled {
		t.Error("Protocol should have started at that point (2 client, 1 trustee)")
	}
	roster = c.createRoster()
	if len(roster.List) != 4 {
		t.Error("Roster should have length 4")
	}
	if !testIfInRoster(roster, relayID) {
		t.Error("Relay should be in roster")
	}
	if !testIfInRoster(roster, trustees[1]) {
		t.Error("Trustee 1 should be in roster")
	}
	if !testIfInRoster(roster, clients[0]) {
		t.Error("Client 0 should be in roster")
	}
	if !testIfInRoster(roster, clients[1]) {
		t.Error("Client 1 should be in roster")
	}
	idMap = c.createIdentitiesMap()
	if len(idMap) != len(roster.List) {
		t.Error("IDmap should have the same length as the roster")
	}
	if !testIfInIDMap(idMap, relayID) {
		t.Error("Relay should be in idMap")
	}
	if !testIfInIDMap(idMap, trustees[1]) {
		t.Error("Trustee 1 should be in idMap")
	}
	if !testIfInIDMap(idMap, clients[0]) {
		t.Error("Client 0 should be in idMap")
	}
	if !testIfInIDMap(idMap, clients[1]) {
		t.Error("Client 1 should be in idMap")
	}
	if !testIDMapForCollisions(idMap) {
		t.Error("Something is wrong in the ID map")
		log.Lvlf1("%+v", idMap)
	}
	stopProtocolCalled = false
	startProtocolCalled = false
	c.isProtocolRunning = func() bool { return true } //protocol is now running

	//re-add the same client
	c.handleConnection(genPacketFromSource(clients[0]))
	nClients, nTrustees = c.waitQueue.count()
	if nClients != 2 {
		t.Error("nClients should be 2, is", nClients)
	}
	if nTrustees != 1 {
		t.Error("nTrustees should be 1, is", nTrustees)
	}
	if stopProtocolCalled {
		t.Error("Protocol should not have been stopped at that point, same client re-tried to connect")
	}
	if startProtocolCalled {
		t.Error("Protocol should not have re-started at that point, same client re-tried to connect")
	}
	roster = c.createRoster()
	if len(roster.List) != 4 {
		t.Error("Roster should have length 4")
	}
	if !testIfInRoster(roster, relayID) {
		t.Error("Relay should be in roster")
	}
	if !testIfInRoster(roster, trustees[1]) {
		t.Error("Trustee 1 should be in roster")
	}
	if !testIfInRoster(roster, clients[0]) {
		t.Error("Client 0 should be in roster")
	}
	if !testIfInRoster(roster, clients[1]) {
		t.Error("Client 1 should be in roster")
	}
	idMap = c.createIdentitiesMap()
	if len(idMap) != len(roster.List) {
		t.Error("IDmap should have the same length as the roster")
	}
	if !testIfInIDMap(idMap, relayID) {
		t.Error("Relay should be in idMap")
	}
	if !testIfInIDMap(idMap, trustees[1]) {
		t.Error("Trustee 1 should be in idMap")
	}
	if !testIfInIDMap(idMap, clients[0]) {
		t.Error("Client 0 should be in idMap")
	}
	if !testIfInIDMap(idMap, clients[1]) {
		t.Error("Client 1 should be in idMap")
	}
	if !testIDMapForCollisions(idMap) {
		t.Error("Something is wrong in the ID map")
		log.Lvlf1("%+v", idMap)
	}
	stopProtocolCalled = false
	startProtocolCalled = false
	c.isProtocolRunning = func() bool { return true } //protocol is now running

	//trigger network error
	c.handleUnknownDisconnection()
	nClients, nTrustees = c.waitQueue.count()
	if nClients != 0 {
		t.Error("nClients should be 0, is", nClients)
	}
	if nTrustees != 0 {
		t.Error("nTrustees should be 0, is", nTrustees)
	}
	if !stopProtocolCalled {
		t.Error("Protocol should have been stopped at that point, network error occurred")
	}
	if startProtocolCalled {
		t.Error("Protocol should not have re-started at that point, no client have tried to connect yet")
	}
	roster = c.createRoster()
	if len(roster.List) != 1 {
		t.Error("Roster should have length 1")
	}
	idMap = c.createIdentitiesMap()
	if len(idMap) != len(roster.List) {
		t.Error("IDmap should have the same length as the roster")
	}
	if !testIfInIDMap(idMap, relayID) {
		t.Error("Relay should be in idMap")
	}
	if !testIDMapForCollisions(idMap) {
		t.Error("Something is wrong in the ID map")
		log.Lvlf1("%+v", idMap)
	}
	stopProtocolCalled = false
	startProtocolCalled = false
	c.isProtocolRunning = func() bool { return false } //protocol is now running

	//add a bunch of entities
	c.handleConnection(genPacketFromSource(clients[0]))
	c.handleConnection(genPacketFromSource(clients[1]))
	c.handleConnection(genPacketFromSource(clients[2]))
	c.handleConnection(genPacketFromSource(trustees[1]))
	nClients, nTrustees = c.waitQueue.count()
	if nClients != 3 {
		t.Error("nClients should be 3, is", nClients)
	}
	if nTrustees != 1 {
		t.Error("nTrustees should be 1, is", nTrustees)
	}
	if stopProtocolCalled {
		t.Error("Protocol should not have been stopped at that point, it was not running")
	}
	if !startProtocolCalled {
		t.Error("Protocol should have re-started at that point, we have enough clients")
	}
	roster = c.createRoster()
	if len(roster.List) != 5 {
		t.Error("Roster should have length 5")
	}
	if !testIDMapForCollisions(idMap) {
		t.Error("Something is wrong in the ID map")
		log.Lvlf1("%+v", idMap)
	}
	stopProtocolCalled = false
	startProtocolCalled = false
	c.isProtocolRunning = func() bool { return true } //protocol is now running

	//trigger one disconnection - it should kick everybody
	c.handleDisconnection(genPacketFromSource(clients[1]))
	nClients, nTrustees = c.waitQueue.count()
	if nClients != 0 {
		t.Error("nClients should be 0, is", nClients)
	}
	if nTrustees != 0 {
		t.Error("nTrustees should be 0, is", nTrustees)
	}
	if !stopProtocolCalled {
		t.Error("Protocol should have stopped, we got a disconnection")
	}
	if startProtocolCalled {
		t.Error("Protocol should not have re-started at that point, we don't have enough clients")
	}
	roster = c.createRoster()
	if len(roster.List) != 1 {
		t.Error("Roster should have length 1")
	}
	if !testIfInRoster(roster, relayID) {
		t.Error("Relay should be in roster")
	}
	if !testIDMapForCollisions(idMap) {
		t.Error("Something is wrong in the ID map")
		log.Lvlf1("%+v", idMap)
	}
	stopProtocolCalled = false
	startProtocolCalled = false
	c.isProtocolRunning = func() bool { return false } //protocol is now running

	//other tests; do we call StopProtocol only when it is not running
	c.handleConnection(genPacketFromSource(trustees[0]))
	c.handleConnection(genPacketFromSource(clients[0]))
	c.handleConnection(genPacketFromSource(clients[1]))
	stopProtocolCalled = false
	startProtocolCalled = false
	c.isProtocolRunning = func() bool { return false }
	c.tryStartProtocol()
	if stopProtocolCalled {
		t.Error("Protocol should not have been stopped at that point, it was not running")
	}
	if !startProtocolCalled {
		t.Error("Protocol should have restarted")
	}

	//other tests; do we call StopProtocol when it was running
	stopProtocolCalled = false
	startProtocolCalled = false
	c.isProtocolRunning = func() bool { return true }
	c.tryStartProtocol()
	if !stopProtocolCalled {
		t.Error("Protocol should have been stopped at that point, it was running")
	}
	if !startProtocolCalled {
		t.Error("Protocol should have restarted")
	}
}
