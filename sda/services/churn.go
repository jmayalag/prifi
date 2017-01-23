package services

// This file contains the logic to handle churn.

import (
	"time"

	"github.com/dedis/onet/log"
	"github.com/dedis/onet/network"
	"github.com/dedis/onet"
	"github.com/lbarman/prifi/sda/protocols"
)

/*
 * When the relay boots :
 * he reads group.toml, identifies the trustees
 * he inits an empty list of nodes
 *
 * When a node connects :
 * the relay identifies him as client or trustee using the stored group.toml
 * he adds it to the list of nodes
 * if PriFi was running, he kills it, and rerun it if > threshold
 *
 * When a node disconnect :
 * He sends STOP messages to every other node
 * He kills his local instance of PriFi protocol
 * He empties the list of waiting nodes
 *
 * Every X seconds :
 * if the protocol is not running
 * count the number of participants, if > threshold, start prifi
 */

//Delay before each host re-tried to connect
const DELAY_BEFORE_KEEPALIVE = 5 * time.Second

type waitQueueEntry struct {
	serverID  *network.ServerIdentity
	numericID int
	role      protocols.PriFiRole
}

// waitQueue contains the list of nodes that are currently willing
// to participate to the protocol.
type waitQueue struct {
	trustees map[string]*waitQueueEntry
	clients  map[string]*waitQueueEntry
}

func idFromMsg(msg *network.Packet) string {
	return idFromServerIdentity(msg.ServerIdentity)
}
func idFromServerIdentity(si *network.ServerIdentity) string {
	public := si.Public.String()
	identifier := public
	return identifier
}

type churnHandler struct {
	waitQueue         *waitQueue
	nextFreeClientID  int
	nextFreeTrusteeID int
	relayIdentity     *network.ServerIdentity //necessary to call createRoster
	trusteesIDs       []*network.ServerIdentity

	//to be specified when instantiated
	startProtocol     func()
	stopProtocol      func()
	isProtocolRunning func() bool
}

func (c *churnHandler) init(relayID *network.ServerIdentity, trusteesIDs []*network.ServerIdentity) {

	if relayID == nil {
		log.Fatal("Can't start the churnHandler without the relayID")
	}
	if trusteesIDs == nil {
		log.Fatal("Can't start the churnHandler without the trusteesIDs")
	}

	c.waitQueue = &waitQueue{
		clients:  make(map[string]*waitQueueEntry),
		trustees: make(map[string]*waitQueueEntry),
	}
	c.nextFreeClientID = 0
	c.nextFreeTrusteeID = 0
	c.relayIdentity = relayID
	c.trusteesIDs = trusteesIDs
}

/**
 * Checks whether an ID is in the waiting clients/trustees (given isTrustee)
 */
func (wq *waitQueue) contains(stringID string, isTrustee bool) bool {
	if isTrustee {
		_, ok := wq.trustees[stringID]
		return ok
	}
	_, ok := wq.clients[stringID]
	return ok
}

/**
 * Returns nClients, nTrustees waiting
 */
func (wq *waitQueue) count() (int, int) {
	return len(wq.clients), len(wq.trustees)
}

/**
 * Creates a roster from waiting nodes, used by SDA
 */
func (c *churnHandler) createRoster() *sda.Roster {

	n, m := c.waitQueue.count()
	nParticipants := n + m + 1

	participants := make([]*network.ServerIdentity, nParticipants)
	participants[0] = c.relayIdentity
	i := 1
	for _, v := range c.waitQueue.clients {
		participants[i] = v.serverID
		i++
	}
	for _, v := range c.waitQueue.trustees {
		participants[i] = v.serverID
		i++
	}

	roster := sda.NewRoster(participants)
	return roster
}

/**
 * Tests if the given serverIdentity represents a trustee
 */
func (c *churnHandler) isATrustee(ID *network.ServerIdentity) bool {
	for _, v := range c.trusteesIDs {
		if v.Equal(ID) {
			return true
		}
	}
	return false
}

/**
 * Creates an IdentityMap from the waiting nodes, used by PriFi-lib
 */
func (c *churnHandler) createIdentitiesMap() map[string]protocols.PriFiIdentity {
	res := make(map[string]protocols.PriFiIdentity)

	//add relay
	res[idFromServerIdentity(c.relayIdentity)] = protocols.PriFiIdentity{
		Role:     protocols.Relay,
		ID:       0,
		ServerID: c.relayIdentity,
	}

	//add clients
	for _, v := range c.waitQueue.clients {
		res[idFromServerIdentity(v.serverID)] = protocols.PriFiIdentity{
			Role:     protocols.Client,
			ID:       v.numericID,
			ServerID: v.serverID,
		}
	}

	//add trustees
	for _, v := range c.waitQueue.trustees {
		res[idFromServerIdentity(v.serverID)] = protocols.PriFiIdentity{
			Role:     protocols.Trustee,
			ID:       v.numericID,
			ServerID: v.serverID,
		}
	}

	return res
}

func (c *churnHandler) getClientsIdentities() []*network.ServerIdentity {
	nClients := len(c.waitQueue.clients)
	clients := make([]*network.ServerIdentity, nClients)
	i := 0
	for _, v := range c.waitQueue.clients {
		clients[i] = v.serverID
		i++
	}
	return clients
}

func (c *churnHandler) getTrusteesIdentities() []*network.ServerIdentity {
	nTrustees := len(c.waitQueue.trustees)
	trustees := make([]*network.ServerIdentity, nTrustees)
	i := 0
	for _, v := range c.waitQueue.trustees {
		trustees[i] = v.serverID
		i++
	}
	return trustees
}

/**
 * Handles a "Connection" message
 */
func (c *churnHandler) handleConnection(msg *network.Packet) {

	ID := idFromMsg(msg)
	isTrustee := c.isATrustee(msg.ServerIdentity)

	if c.waitQueue.contains(ID, isTrustee) {
		log.Lvl3("Ignored new connection request from", ID, " (isATrustee:", isTrustee, "), already in the list")
		return
	}

	log.Lvl3("Received new connection request from", ID, " (isATrustee:", isTrustee, ")")

	if isTrustee {
		c.waitQueue.trustees[ID] = &waitQueueEntry{
			serverID:  msg.ServerIdentity,
			role:      protocols.Trustee,
			numericID: c.nextFreeTrusteeID,
		}
		log.Lvl3("ID ", ID, " assigned to trustee #", c.nextFreeTrusteeID)
		c.nextFreeTrusteeID++
	} else {
		c.waitQueue.clients[ID] = &waitQueueEntry{
			serverID:  msg.ServerIdentity,
			role:      protocols.Client,
			numericID: c.nextFreeClientID,
		}
		log.Lvl3("ID ", ID, " assigned to client #", c.nextFreeClientID)
		c.nextFreeClientID++
	}

	c.tryStartProtocol()
}

func (c *churnHandler) handleUnknownDisconnection() {

	c.waitQueue.clients = make(map[string]*waitQueueEntry)
	c.waitQueue.trustees = make(map[string]*waitQueueEntry)
	c.nextFreeClientID = 0
	c.nextFreeTrusteeID = 0

	c.stopProtocol()
	c.tryStartProtocol()
}

/**
 * Handles a "Disconnection" message
 */
func (c *churnHandler) handleDisconnection(msg *network.Packet) {

	ID := idFromMsg(msg)
	isTrustee := c.isATrustee(msg.ServerIdentity)

	if !c.waitQueue.contains(ID, isTrustee) {
		log.Lvl3("Ignored new disconnection request from", ID, " (isATrustee:", isTrustee, "), not in the list")
		return
	}

	log.Lvl3("Received new disconnection request from", ID, " (isATrustee:", isTrustee, ")")

	/* This is the smart way. Dumb way first
	if isTrustee {
		delete(c.waitQueue.trustees, ID)
		c.nextFreeTrusteeID = 0
		for k := range c.waitQueue.trustees {
			c.waitQueue.trustees[k].numericID = c.nextFreeTrusteeID
			c.nextFreeTrusteeID++
		}
		c.nextFreeTrusteeID += 1
	} else {
		delete(c.waitQueue.clients, ID)
		c.nextFreeClientID = 0
		for k := range c.waitQueue.clients {
			c.waitQueue.clients[k].numericID = c.nextFreeClientID
			c.nextFreeClientID++
		}
		c.nextFreeClientID += 1
	}
	*/
	c.handleUnknownDisconnection()
}

/**
 * restarts the protocol (stop + start) if nClients waiting & nTrustees waiting both > 1
 */
func (c *churnHandler) tryStartProtocol() {
	nClients, nTrustees := c.waitQueue.count()

	if nClients >= 1 && nTrustees >= 1 {
		if c.isProtocolRunning() {
			c.stopProtocol()
		}
		c.startProtocol()
	} else {
		log.Lvl1("Too few participants (", nClients, "clients and", nTrustees, "trustees), waiting...")
	}
}
