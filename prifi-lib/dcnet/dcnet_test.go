package dcnet

import (
	"bytes"
	"fmt"
	"github.com/lbarman/prifi/prifi-lib/config"
	"gopkg.in/dedis/kyber.v2"
	"testing"
)

type TestGroup struct {
	Relay    *TestNode
	Clients  []*TestNode
	Trustees []*TestNode
}

type TestNode struct {
	name          string
	pubKey        kyber.Point
	privKey       kyber.Scalar
	peerKeys      []kyber.Point
	sharedSecrets []kyber.Point
	History       kyber.XOF
	DCNetEntity   *DCNetEntity
}

func TestDCNetCreation(t *testing.T) {

	nRounds := int32(100)
	dcNetMessageLength := 100

	for nTrustees := 1; nTrustees < 10; nTrustees++ {
		for nClients := 1; nClients < 10; nClients++ {
			VariousLevelsOfProtection(t, nRounds, dcNetMessageLength, nClients, nTrustees)
		}
	}
}

func VariousLevelsOfProtection(t *testing.T, nRounds int32, dcNetMessageSize, NClients, NTrustees int) {
	tg := NewTestGroup(t, false, dcNetMessageSize, NClients, NTrustees)
	SimulateRounds(t, tg, nRounds)
	tg = NewTestGroup(t, true, dcNetMessageSize, NClients, NTrustees)
	SimulateRounds(t, tg, nRounds)
}

func NewTestGroup(t *testing.T, equivocationProtectionEnabled bool, dcNetMessageSize, nclients, ntrustees int) *TestGroup {

	// Use a pseudorandom stream from a well-known seed
	// for all our setup randomness,
	// so we can reproduce the same keys etc on each node.
	rand := config.CryptoSuite.XOF([]byte("DCTest"))

	nodes := make([]*TestNode, nclients+ntrustees)
	base := config.CryptoSuite.Point().Base()
	for i := range nodes {
		nodes[i] = new(TestNode)
		nodes[i].privKey = config.CryptoSuite.Scalar().Pick(rand)
		nodes[i].pubKey = config.CryptoSuite.Point().Mul(nodes[i].privKey, base)
	}

	clients := nodes[:nclients]
	trustees := nodes[nclients:]

	relay := new(TestNode)
	relay.name = "Relay"
	relay.DCNetEntity = NewDCNetEntity(0, DCNET_RELAY, dcNetMessageSize, equivocationProtectionEnabled, nil)

	// Create tables of the clients' and the trustees' public session keys
	clientsKeys := make([]kyber.Point, nclients)
	trusteesKeys := make([]kyber.Point, ntrustees)
	for i := range clients {
		clientsKeys[i] = clients[i].pubKey
	}
	for i := range trustees {
		trusteesKeys[i] = trustees[i].pubKey
	}

	// Setup the clients and servers to know each others' session keys.
	for i, n := range clients {
		// Form Diffie-Hellman secret shared with each peer,
		// and a pseudorandom cipher derived from each.
		n.name = fmt.Sprintf("Client-%d", i)
		n.peerKeys = trusteesKeys
		n.sharedSecrets = make([]kyber.Point, len(n.peerKeys))
		for i := range n.peerKeys {
			n.sharedSecrets[i] = config.CryptoSuite.Point().Mul(n.privKey, n.peerKeys[i])
		}
		n.DCNetEntity = NewDCNetEntity(i, DCNET_CLIENT, dcNetMessageSize, equivocationProtectionEnabled, n.sharedSecrets)
	}

	for i, n := range trustees {
		// Form Diffie-Hellman secret shared with each peer,
		// and a pseudorandom cipher derived from each.
		n.name = fmt.Sprintf("Trustee-%d", i)
		n.peerKeys = clientsKeys
		n.sharedSecrets = make([]kyber.Point, len(n.peerKeys))
		for i := range n.peerKeys {
			n.sharedSecrets[i] = config.CryptoSuite.Point().Mul(n.privKey, n.peerKeys[i])
		}
		n.DCNetEntity = NewDCNetEntity(i, DCNET_TRUSTEE, dcNetMessageSize, equivocationProtectionEnabled, n.sharedSecrets)
	}

	// Create a set of fake history streams for the relay and clients
	//hist := []byte("xyz")
	//relay.History = suite.Cipher(hist)
	//for i := range clients {
	//	clients[i].History = suite.Cipher(hist)
	//}

	tg := new(TestGroup)
	tg.Relay = relay
	tg.Clients = clients
	tg.Trustees = trustees
	return tg
}

func SimulateRounds(t *testing.T, tg *TestGroup, maxRounds int32) {

	d := tg.Relay.DCNetEntity
	fmt.Println("Testing for ", len(tg.Clients), "/", len(tg.Trustees),
		"DC-net with equiv=", d.EquivocationProtectionEnabled)

	for roundID := int32(0); roundID <= maxRounds; roundID += 2 {
		clientMessages := make([][]byte, 0)
		trusteesMessages := make([][]byte, 0)
		first := true
		message := randomBytes(d.DCNetPayloadSize)

		downstreamMessage := randomBytes(d.DCNetPayloadSize) //used only to update the history
		for i := range tg.Clients {
			tg.Clients[i].DCNetEntity.UpdateReceivedMessageHistory(downstreamMessage)
		}
		tg.Relay.DCNetEntity.UpdateReceivedMessageHistory(downstreamMessage)

		// Generate the clients dc-net cryptographic material
		for i := range tg.Clients {
			var m []byte
			if first {
				//fmt.Println("Embedding message:", message)
				m = tg.Clients[i].DCNetEntity.EncodeForRound(roundID, true, message)
				first = false
			} else {
				m = tg.Clients[i].DCNetEntity.EncodeForRound(roundID, false, nil)
			}
			clientMessages = append(clientMessages, m)
		}

		// Generate the trustees dc-net cryptographic material
		for i := range tg.Trustees {
			m := tg.Trustees[i].DCNetEntity.TrusteeEncodeForRound(roundID)
			trusteesMessages = append(trusteesMessages, m)
		}

		// The relay decodes the cryptographic material
		tg.Relay.DCNetEntity.DecodeStart(roundID)
		for _, m := range clientMessages {
			tg.Relay.DCNetEntity.DecodeClient(roundID, m)
		}
		for _, m := range trusteesMessages {
			tg.Relay.DCNetEntity.DecodeTrustee(roundID, m)
		}

		output := tg.Relay.DCNetEntity.DecodeCell()

		//fmt.Println("-----------------")
		//fmt.Println(output)
		//fmt.Println(message)

		if !bytes.Equal(output, message) {
			t.Error("DC-net encoding failed")
		}
	}
}
