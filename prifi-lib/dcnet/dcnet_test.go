package dcnet

import (
	"testing"

	"bytes"
	"fmt"
	"os"
	"time"

	"gopkg.in/dedis/crypto.v0/nist"

	"gopkg.in/dedis/crypto.v0/abstract"
)

func TestSimple(t *testing.T) {
	CellCoderTest(t, nist.NewAES128SHA256P256(), SimpleCoderFactory)
}

func TestOwned(t *testing.T) {
	CellCoderTest(t, nist.NewAES128SHA256P256(), OwnedCoderFactory)
}

type TestNode struct {

	// General parameters
	suite abstract.Suite
	name  string

	// Asymmetric keypair for this node
	pub abstract.Point
	pri abstract.Scalar

	npeers        int
	peerkeys      []abstract.Point  // each peer's session public key
	sharedsecrets []abstract.Cipher // shared secrets

	// Owner keypair for this cell series.
	// Public key is known by and common to all nodes.
	// Private key is held only by owner client.
	opub abstract.Point
	opri abstract.Scalar

	Coder DCNet

	// Cipher representing history as seen by this node.
	History abstract.Cipher
}

func (n *TestNode) Dump(tno int) {
	fmt.Println("[", tno, ": pub ", n.pub, ", pri ", n.pri, ", opub ", n.opub, " opri", n.opri, "]")
}

type TestGroup struct {
	Relay    *TestNode
	Clients  []*TestNode
	Trustees []*TestNode
}

func (n *TestNode) nodeSetup(name string, peerkeys []abstract.Point) {
	n.name = name

	// Form Diffie-Hellman secret shared with each peer,
	// and a pseudorandom cipher derived from each.
	n.npeers = len(peerkeys)
	n.peerkeys = peerkeys
	n.sharedsecrets = make([]abstract.Cipher, n.npeers)
	for i := range peerkeys {
		dh := n.suite.Point().Mul(peerkeys[i], n.pri)
		data, _ := dh.MarshalBinary()
		n.sharedsecrets[i] = n.suite.Cipher(data)
	}
}

func SetupTest(t *testing.T, suite abstract.Suite, factory CellFactory,
	nclients, ntrustees int) *TestGroup {

	// Use a pseudorandom stream from a well-known seed
	// for all our setup randomness,
	// so we can reproduce the same keys etc on each node.
	rand := suite.Cipher([]byte("DCTest"))

	nodes := make([]*TestNode, nclients+ntrustees)
	base := suite.Point().Base()
	for i := range nodes {
		nodes[i] = new(TestNode)
		nodes[i].suite = suite

		// Each client and trustee gets a session keypair
		nodes[i].pri = suite.Scalar().Pick(rand)
		nodes[i].pub = suite.Point().Mul(base, nodes[i].pri)
		if t != nil {
			t.Logf("node %d key %s\n", i, nodes[i].pri.String())
		}

		nodes[i].Coder = factory()
	}

	clients := nodes[:nclients]
	trustees := nodes[nclients:]

	relay := new(TestNode)
	relay.name = "Relay"
	relay.Coder = factory()

	// Create tables of the clients' and the trustees' public session keys
	ckeys := make([]abstract.Point, nclients)
	tkeys := make([]abstract.Point, ntrustees)
	for i := range clients {
		ckeys[i] = clients[i].pub
	}
	for i := range trustees {
		tkeys[i] = trustees[i].pub
	}

	// Pick an "owner" for the (one) transmission series we'll have.
	// For now the owner will be the first client.
	opri := suite.Scalar().Pick(rand)
	opub := suite.Point().Mul(base, opri)
	clients[0].opri = opri
	for i := range nodes {
		nodes[i].opub = opub // Everyone knows owner public key
	}

	// Setup the clients and servers to know each others' session keys.
	// XXX this should by something generic across multiple cell types,
	// producing master shared ciphers that each cell type derives from.
	for i := range clients {
		n := clients[i]
		n.nodeSetup(fmt.Sprintf("Client%d", i), tkeys)
		n.Coder = factory()
		n.Coder.ClientSetup(suite, n.sharedsecrets)
	}

	tinfo := make([][]byte, ntrustees)
	for i := range trustees {
		n := trustees[i]
		n.nodeSetup(fmt.Sprintf("Trustee%d", i), ckeys)
		n.Coder = factory()
		tinfo[i] = n.Coder.TrusteeSetup(suite, n.sharedsecrets)
	}
	relay.Coder.RelaySetup(suite, tinfo)

	// Create a set of fake history streams for the relay and clients
	hist := []byte("xyz")
	relay.History = suite.Cipher(hist)
	for i := range clients {
		clients[i].History = suite.Cipher(hist)
	}

	tg := new(TestGroup)
	tg.Relay = relay
	tg.Clients = clients
	tg.Trustees = trustees
	return tg
}

func CellCoderTest(t *testing.T, suite abstract.Suite, factory CellFactory) {

	nclients := 1
	ntrustees := 3

	tg := SetupTest(t, suite, factory, nclients, ntrustees)
	relay := tg.Relay
	clients := tg.Clients
	trustees := tg.Trustees

	// Get some data to transmit
	t.Log("Simulating DC-nets")
	payloadlen := 1200
	inb := make([]byte, payloadlen)
	inf, err := os.Open("./dcnet_simple.go")
	if err != nil {
		t.Error("Could not run test, unable to read file")
	}
	beg := time.Now()
	ncells := 0
	nbytes := 0
	cslice := make([][]byte, nclients)
	tslice := make([][]byte, ntrustees)
	for {
		n, _ := inf.Read(inb)
		if n <= 0 {
			break
		}
		payloadlen = n

		// Client processing
		// first client (owner) gets the payload data
		p := make([]byte, payloadlen)
		copy(p, inb)
		for i := range clients {
			cslice[i] = clients[i].Coder.ClientEncode(p, payloadlen,
				clients[i].History)
			p = nil // for remaining clients
		}

		// Trustee processing
		for i := range trustees {
			tslice[i] = trustees[i].Coder.TrusteeEncode(payloadlen)
		}

		// Relay processing
		relay.Coder.DecodeStart(payloadlen, relay.History)
		for i := range clients {
			relay.Coder.DecodeClient(cslice[i])
		}
		for i := range trustees {
			relay.Coder.DecodeTrustee(tslice[i])
		}
		outb := relay.Coder.DecodeCell()

		//os.Stdout.Write(outb)
		if outb == nil || len(outb) != payloadlen ||
			!bytes.Equal(inb[:payloadlen], outb[:payloadlen]) {
			t.Log("oops, data corrupted")
			t.FailNow()
		}

		ncells++
		nbytes += payloadlen
	}
	end := time.Now()
	t.Logf("Time %f cells %d bytes %d nclients %d ntrustees %d\n",
		float64(end.Sub(beg))/1000000000.0,
		ncells, nbytes, nclients, ntrustees)
}

func TestOthers(t *testing.T) {
	cellCoder := new(ownedCoder)
	cellCoder.suite = nist.NewAES128SHA256P256()
	cellCoder.random = cellCoder.suite.Cipher([]byte{0, 1, 2})

	if size := cellCoder.GetClientCipherSize(1500); size != 1532 {
		t.Error("Size should be", size)
	}
	if size := cellCoder.GetClientCipherSize(0); size != 32 {
		t.Error("Size should be", size)
	}
	if size := cellCoder.GetTrusteeCipherSize(1500); size != 1500 {
		t.Error("Size should be", 1500)
	}
	if size := cellCoder.GetTrusteeCipherSize(0); size != 0 {
		t.Error("Size should be", 0)
	}

	data := make([]byte, 10)
	p := cellCoder.suite.Point().Base()
	cellCoder.inlineEncode(data, p)

	p.Sub(p, cellCoder.suite.Point().Base())
	hdr, err := p.Data()
	if err != nil {
		t.Error("Can't decode inline point")
	}
	cellCoder.inlineDecode(hdr)

	// Testing inline encoding and no transmission cases
	nclients := 1
	ntrustees := 3

	tg := SetupTest(t, nist.NewAES128SHA256P256(), OwnedCoderFactory, nclients, ntrustees)
	relay := tg.Relay
	clients := tg.Clients
	trustees := tg.Trustees

	// Get some data to transmit
	t.Log("Simulating DC-nets")
	payloadlen := 10
	cslice := make([][]byte, nclients)
	tslice := make([][]byte, ntrustees)

	// Client processing
	// first client (owner) gets the payload data
	dat := []byte("Testing")

	for i := range clients {
		cslice[i] = clients[i].Coder.ClientEncode(dat, payloadlen,
			clients[i].History)
		dat = nil // for remaining clients
	}

	// Trustee processing
	for i := range trustees {
		tslice[i] = trustees[i].Coder.TrusteeEncode(payloadlen)
	}

	// Relay processing
	relay.Coder.DecodeStart(payloadlen, relay.History)
	for i := range clients {
		relay.Coder.DecodeClient(cslice[i])
	}
	for i := range trustees {
		relay.Coder.DecodeTrustee(tslice[i])
	}
	outb := relay.Coder.DecodeCell()

	if !bytes.Equal(outb, []byte("Testing")) {
		t.Error("Inline encoding-decoding failed")
	}
}
