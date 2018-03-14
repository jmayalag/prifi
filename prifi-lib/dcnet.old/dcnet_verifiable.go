package dcnet_old

import (
	"bytes"
	"crypto/rand"

	"gopkg.in/dedis/crypto.v0/abstract"
)

type verifiableDCNet struct {
	suite abstract.Suite

	// Length of Key and MAC part of verifiable DC-net point
	keyLength, macLength int

	// Verifiable DC-nets secrets shared with each peer.
	vkeys []abstract.Scalar

	// The sum of all our verifiable DC-nets secrets.
	vkey abstract.Scalar

	// Pseudorandom DC-nets ciphers shared with each peer.
	// On clients, there is one DC-nets cipher per trustee.
	// On trustees, there ois one DC-nets cipher per client.
	dcCiphers []abstract.Cipher

	// Pseudorandom stream
	random abstract.Cipher

	// Decoding state, used only by the relay
	point     abstract.Point
	pnull     abstract.Point // neutral/identity element
	xorBuffer []byte
}

// OwnedCoderFactory creates a DC-net cell coder for "owned" cells:
// cells having a single owner identified by a public pseudonym key.
//
// This CellCoder supports variable-length payloads.
// For small payloads that can be embedded into half a Point,
// the encoding consists of a single verifiable DC-net point.
// For larger payloads, we use one verifiable DC-net point
// to transmit a key and a MAC for the associated variable-length,
// symmetric-key crypto based part of the cell.
func NewVerifiableDCNet(equivocationProtectionEnabled bool) DCNet {
	return new(verifiableDCNet)
}

func (c *verifiableDCNet) UpdateHistory(data []byte) {
	panic("not supported")
}

// Compute the size of the symmetric AES-encoded part of an encoded ciphertext.
func (c *verifiableDCNet) symmCellSize(payloadLength int) int {

	// If data fits in the space reserved for the key
	// in the verifiable DC-net point,
	// we can just inline the data in the point instead of the key.
	// (We'll still use the MAC part of the point for validation.)
	if payloadLength <= c.keyLength {
		return 0
	}

	// Otherwise the point is used to hold an encryption key and a MAC,
	// and the payload is symmetric-key encrypted.
	// XXX trap encoding
	return payloadLength
}

func (c *verifiableDCNet) commonSetup(suite abstract.Suite) {
	c.suite = suite

	// Divide the embeddable data in the verifiable point
	// between an encryption key and a MAC check
	c.keyLength = suite.Cipher(nil).KeySize()
	c.macLength = suite.Point().PickLen() - c.keyLength
	if c.macLength < c.keyLength*3/4 {
		panic("misconfigured ciphersuite: MAC too small!")
	}

	randomKey := make([]byte, suite.Cipher(nil).KeySize())
	rand.Read(randomKey)
	c.random = suite.Cipher(randomKey)
}

func (c *verifiableDCNet) GetClientCipherSize(payloadLength int) int {
	// Clients must produce a point plus the symmetric ciphertext
	return 32 + c.symmCellSize(payloadLength)
}

func (c *verifiableDCNet) ClientSetup(suite abstract.Suite, sharedSecrets []abstract.Cipher) {
	c.commonSetup(suite)
	keySize := suite.Cipher(nil).KeySize()

	// Use the provided shared secrets to seed
	// a pseudorandom public-key encryption secret, and
	// a pseudorandom DC-nets cipher shared with each peer.
	nPeers := len(sharedSecrets)
	c.vkeys = make([]abstract.Scalar, nPeers)
	c.vkey = suite.Scalar()
	c.dcCiphers = make([]abstract.Cipher, nPeers)
	for i := range sharedSecrets {
		c.vkeys[i] = suite.Scalar().Pick(sharedSecrets[i])
		c.vkey.Add(c.vkey, c.vkeys[i])
		key := make([]byte, keySize)
		sharedSecrets[i].Partial(key, key, nil)
		c.dcCiphers[i] = suite.Cipher(key)
	}
}

func (c *verifiableDCNet) ClientEncode(payload []byte, payloadLength int, history abstract.Cipher) []byte {

	// Compute the verifiable blinding point for this cell.
	// To protect clients from equivocation by relays,
	// we choose the blinding generator for each cell pseudorandomly
	// based on the history of all past downstream messages
	// the client has received from the relay.
	// If any two honest clients disagree on this history,
	// they will produce encryptions based on unrelated generators,
	// rendering the cell unintelligible,
	// so that any data the client might be sending based on
	// having seen a divergent history gets suppressed.
	p := c.suite.Point()
	p.Pick(nil, history)
	p.Mul(p, c.vkey)

	// Encode the payload data, if any.
	payOut := make([]byte, c.symmCellSize(payloadLength))
	if payload != nil {
		// We're the owner of this cell.
		if len(payload) <= c.keyLength {
			c.inlineEncode(payload, p)
		} else {
			c.ownerEncode(payload, payOut, p)
		}
	}

	// XOR the symmetric DC-net streams into the payload part
	for i := range c.dcCiphers {
		c.dcCiphers[i].XORKeyStream(payOut, payOut)
	}

	// Build the full cell ciphertext
	out, _ := p.MarshalBinary()
	out = append(out, payOut...)
	return out
}

func (c *verifiableDCNet) inlineEncode(payload []byte, p abstract.Point) {

	// Hash the cleartext payload to produce the MAC
	hash := c.suite.Hash()
	hash.Write(payload)
	mac := hash.Sum(nil)[:c.macLength]

	// Embed the payload and MAC into a Point representing the message
	hdr := append(payload, mac...)
	mp, _ := c.suite.Point().Pick(hdr, c.random)

	// Add this to the blinding point we already computed to transmit.
	p.Add(p, mp)
}

func (c *verifiableDCNet) ownerEncode(payload, payOut []byte, p abstract.Point) {

	// XXX trap-encode

	// Pick a fresh random key with which to encrypt the payload
	key := make([]byte, c.keyLength)
	c.random.XORKeyStream(key, key)

	// Encrypt the payload with it
	c.suite.Cipher(key).XORKeyStream(payOut, payload)

	// Compute a MAC over the encrypted payload
	hash := c.suite.Hash()
	hash.Write(payOut)
	mac := hash.Sum(nil)[:c.macLength]

	// Combine the key and the MAC into the Point for this cell header
	hdr := append(key, mac...)
	if len(hdr) != p.PickLen() {
		panic("oops, length of key+mac turned out wrong")
	}
	mp, _ := c.suite.Point().Pick(hdr, c.random)

	// Add this to the blinding point we already computed to transmit.
	p.Add(p, mp)
}

func (c *verifiableDCNet) GetTrusteeCipherSize(payloadLength int) int {
	// Trustees produce only the symmetric ciphertext, if any
	return c.symmCellSize(payloadLength)
}

// Setup the trustee side.
// May produce coder configuration info to be passed to the relay,
// which will become available to the RelaySetup() method below.
func (c *verifiableDCNet) TrusteeSetup(suite abstract.Suite, clientStreams []abstract.Cipher) []byte {

	// Compute shared secrets
	c.ClientSetup(suite, clientStreams)

	// Release the negation of the composite shared verifiable secret
	// to the relay, so the relay can decode each cell's header.
	c.vkey.Neg(c.vkey)
	rv, _ := c.vkey.MarshalBinary()
	return rv
}

func (c *verifiableDCNet) TrusteeEncode(payloadLength int) []byte {
	// Trustees produce only symmetric DC-nets streams for the payload portion of each cell.
	payOut := make([]byte, payloadLength) // XXX trap expansion
	for i := range c.dcCiphers {
		c.dcCiphers[i].XORKeyStream(payOut, payOut)
	}
	return payOut
}

func (c *verifiableDCNet) RelaySetup(suite abstract.Suite, trusteeInfo [][]byte) {

	c.commonSetup(suite)

	// Decode the trustees' composite verifiable DC-net secrets
	nTrustees := len(trusteeInfo)
	c.vkeys = make([]abstract.Scalar, nTrustees)
	c.vkey = suite.Scalar()
	for i := range c.vkeys {
		c.vkeys[i] = c.suite.Scalar()
		c.vkeys[i].UnmarshalBinary(trusteeInfo[i])
		c.vkey.Add(c.vkey, c.vkeys[i])
	}

	c.pnull = c.suite.Point().Null()
}

func (c *verifiableDCNet) DecodeStart(payloadLength int, history abstract.Cipher) {

	// Compute the composite trustees-side verifiable DC-net unblinder
	// based on the appropriate message history.
	p := c.suite.Point()
	p.Pick(nil, history)
	p.Mul(p, c.vkey)
	c.point = p

	// Initialize the symmetric ciphertext XOR buffer
	if payloadLength > c.keyLength {
		c.xorBuffer = make([]byte, payloadLength)
	}
}

func (c *verifiableDCNet) DecodeClient(slice []byte) {
	// Decode and add in the point in the slice header
	pLength := c.suite.PointLen()
	p := c.suite.Point()
	if err := p.UnmarshalBinary(slice[:pLength]); err != nil {
		println("warning: error decoding point")
	}
	c.point.Add(c.point, p)

	// Combine in the symmetric ciphertext streams
	if c.xorBuffer != nil {
		slice = slice[pLength:]
		for i := range slice {
			c.xorBuffer[i] ^= slice[i]
		}
	}
}

func (c *verifiableDCNet) DecodeTrustee(slice []byte) {

	// Combine in the trustees' symmetric ciphertext streams
	if c.xorBuffer != nil {
		for i := range slice {
			c.xorBuffer[i] ^= slice[i]
		}
	}
}

func (c *verifiableDCNet) DecodeCell() []byte {

	if c.point.Equal(c.pnull) {
		//println("no transmission in cell")
		return nil
	}

	// Decode the header from the decrypted point.
	hdr, err := c.point.Data()
	if err != nil || len(hdr) < c.macLength {
		println("warning: undecipherable cell header")
		return nil // XXX differentiate from no transmission?
	}

	if c.xorBuffer == nil { // short inline cell
		return c.inlineDecode(hdr)
	}
	// long payload cell
	return c.ownerDecode(hdr)
}

func (c *verifiableDCNet) inlineDecode(hdr []byte) []byte {

	// Split the inline payload from the MAC
	dataLength := len(hdr) - c.macLength
	data := hdr[:dataLength]
	mac := hdr[dataLength:]

	// Check the MAC
	hash := c.suite.Hash()
	hash.Write(data)
	check := hash.Sum(nil)[:c.macLength]
	if !bytes.Equal(mac, check) {
		println("warning: MAC check failed on inline cell")
		return nil
	}

	return data
}

func (c *verifiableDCNet) ownerDecode(hdr []byte) []byte {

	// Split the payload encryption key from the MAC
	keyLength := len(hdr) - c.macLength
	if keyLength != c.keyLength {
		println("warning: wrong size cell encryption key")
		return nil
	}
	key := hdr[:keyLength]
	mac := hdr[keyLength:]
	data := c.xorBuffer

	// Check the MAC on the still-encrypted data
	hash := c.suite.Hash()
	hash.Write(data)
	check := hash.Sum(nil)[:c.macLength]
	if !bytes.Equal(mac, check) {
		println("warning: MAC check failed on out-of-line cell")
		return nil
	}

	// Decrypt and return the payload data
	c.suite.Cipher(key).XORKeyStream(data, data)
	return data
}
