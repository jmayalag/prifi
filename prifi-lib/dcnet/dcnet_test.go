package dcnet

import (
	"testing"
	"github.com/lbarman/prifi/prifi-lib/config"
	"gopkg.in/dedis/crypto.v0/abstract"
	"fmt"
	"bytes"
)

type TestGroup struct {
	Relay    *TestNode
	Clients  []*TestNode
	Trustees []*TestNode
}

type TestNode struct {
	name string
	pubKey  abstract.Point
	privKey abstract.Scalar
	peerKeys      []abstract.Point
	sharedSecrets []abstract.Cipher
	History abstract.Cipher
	DCNetEntity *DCNetEntity
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
	tg := NewTestGroup(t, false, false, dcNetMessageSize, NClients, NTrustees)
	SimulateRounds(t, tg, nRounds)
	tg = NewTestGroup(t, true, false, dcNetMessageSize, NClients, NTrustees)
	SimulateRounds(t, tg, nRounds)
	tg = NewTestGroup(t, false, true, dcNetMessageSize, NClients, NTrustees)
	SimulateRounds(t, tg, nRounds)
	tg = NewTestGroup(t, true, true, dcNetMessageSize, NClients, NTrustees)
	SimulateRounds(t, tg, nRounds)
}

func NewTestGroup(t *testing.T, disruptionProtectionEnabled, equivocationProtectionEnabled bool, dcNetMessageSize, nclients, ntrustees int) *TestGroup  {

	// Use a pseudorandom stream from a well-known seed
	// for all our setup randomness,
	// so we can reproduce the same keys etc on each node.
	rand := config.CryptoSuite.Cipher([]byte("DCTest"))

	nodes := make([]*TestNode, nclients+ntrustees)
	base := config.CryptoSuite.Point().Base()
	for i := range nodes {
		nodes[i] = new(TestNode)
		nodes[i].privKey = config.CryptoSuite.Scalar().Pick(rand)
		nodes[i].pubKey = config.CryptoSuite.Point().Mul(base, nodes[i].privKey)
	}

	clients := nodes[:nclients]
	trustees := nodes[nclients:]

	relay := new(TestNode)
	relay.name = "Relay"
	relay.DCNetEntity = NewDCNetEntity(0, DCNET_RELAY, dcNetMessageSize, equivocationProtectionEnabled, disruptionProtectionEnabled, nil)


	// Create tables of the clients' and the trustees' public session keys
	clientsKeys := make([]abstract.Point, nclients)
	trusteesKeys := make([]abstract.Point, ntrustees)
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
		n.sharedSecrets = make([]abstract.Cipher, len(n.peerKeys))
		for i := range n.peerKeys {
			dh := config.CryptoSuite.Point().Mul(n.peerKeys[i], n.privKey)
			data, _ := dh.MarshalBinary()
			n.sharedSecrets[i] = config.CryptoSuite.Cipher(data)
		}
		n.DCNetEntity = NewDCNetEntity(i, DCNET_CLIENT, dcNetMessageSize, equivocationProtectionEnabled, disruptionProtectionEnabled, n.sharedSecrets)
	}

	for i, n := range trustees {
		// Form Diffie-Hellman secret shared with each peer,
		// and a pseudorandom cipher derived from each.
		n.name = fmt.Sprintf("Trustee-%d", i)
		n.peerKeys = clientsKeys
		n.sharedSecrets = make([]abstract.Cipher, len(n.peerKeys))
		for i := range n.peerKeys {
			dh := config.CryptoSuite.Point().Mul(n.peerKeys[i], n.privKey)
			data, _ := dh.MarshalBinary()
			n.sharedSecrets[i] = config.CryptoSuite.Cipher(data)
		}
		n.DCNetEntity = NewDCNetEntity(i, DCNET_TRUSTEE, dcNetMessageSize, equivocationProtectionEnabled, disruptionProtectionEnabled, n.sharedSecrets)
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
		"DC-net with protections: disruption=",d.DisruptionProtectionEnabled,"equiv=", d.EquivocationProtectionEnabled)

	for roundID := int32(0); roundID <= maxRounds; roundID+=2 {
		clientMessages := make([][]byte, 0)
		trusteesMessages := make([][]byte, 0)
		first := true
		message := randomBytes(d.GetPayloadSize())

		downstreamMessage := randomBytes(d.GetPayloadSize()) //used only to update the history
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
			m := tg.Trustees[i].DCNetEntity.EncodeForRound(roundID, false, nil)
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