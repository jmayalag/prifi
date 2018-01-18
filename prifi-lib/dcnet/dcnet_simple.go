package dcnet

import (
	"gopkg.in/dedis/crypto.v0/abstract"
)

type simpleDCNet struct {
	suite abstract.Suite

	equivocationProtection bool

	// Pseudorandom DC-nets ciphers shared with each peer.
	// On clients, there is one cipher per trustee.
	// On trustees, there is one cipher per client.
	dcCiphers []abstract.Cipher

	// Used only on the relay for decoding
	xorBuffer []byte
}

// SimpleCoderFactory is a simple DC-net encoder providing no disruption or equivocation protection,
// for experimentation and baseline performance evaluations.
func NewSimpleDCNet(equivocationProtectionEnabled bool) DCNet {
	dc := new(simpleDCNet)
	dc.equivocationProtection = equivocationProtectionEnabled
	return dc
}

func (c *simpleDCNet) GetClientCipherSize(payloadLength int) int {
	return payloadLength // no expansion
}

func (c *simpleDCNet) ClientSetup(suite abstract.Suite, sharedSecrets []abstract.Cipher) {
	c.suite = suite
	keySize := suite.Cipher(nil).KeySize()

	n_peers := len(sharedSecrets)
	c.dcCiphers = make([]abstract.Cipher, n_peers)

	// Use the provided shared secrets to seed a pseudorandom DC-nets ciphers shared with each peer.
	for i := range sharedSecrets {
		key := make([]byte, keySize)
		sharedSecrets[i].Partial(key, key, nil)
		c.dcCiphers[i] = suite.Cipher(key)
	}
}

func (c *simpleDCNet) ClientEncode(payload []byte, payloadLength int, history abstract.Cipher) []byte {

	if payload == nil {
		payload = make([]byte, payloadLength)
	}
	if len(payload) < payloadLength {
		payload2 := make([]byte, payloadLength)
		copy(payload2[0:len(payload)], payload)
		payload = payload2
	}
	for i := range c.dcCiphers {
		c.dcCiphers[i].XORKeyStream(payload, payload)
	}
	return payload
}

func (c *simpleDCNet) GetTrusteeCipherSize(payloadLength int) int {
	return payloadLength // no expansion
}

func (c *simpleDCNet) TrusteeSetup(suite abstract.Suite, sharedSecrets []abstract.Cipher) []byte {
	c.ClientSetup(suite, sharedSecrets)
	return nil
}

func (c *simpleDCNet) TrusteeEncode(payloadLength int) []byte {
	emptyCode := abstract.Cipher{}
	return c.ClientEncode(nil, payloadLength, emptyCode)
}

func (c *simpleDCNet) RelaySetup(suite abstract.Suite, trusteeInfo [][]byte) {
}

func (c *simpleDCNet) DecodeStart(payloadLength int, history abstract.Cipher) {
	c.xorBuffer = make([]byte, payloadLength)
}

func (c *simpleDCNet) DecodeClient(slice []byte) {
	for i := range slice {
		c.xorBuffer[i] ^= slice[i]
	}
}

func (c *simpleDCNet) DecodeTrustee(slice []byte) {
	c.DecodeClient(slice)
}

func (c *simpleDCNet) DecodeCell() []byte {
	return c.xorBuffer
}
