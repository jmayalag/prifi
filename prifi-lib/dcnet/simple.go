package dcnet

import (
	"gopkg.in/dedis/crypto.v0/abstract"
)

type simpleCoder struct {
	suite abstract.Suite

	// Pseudorandom DC-nets ciphers shared with each peer.
	// On clients, there is one DC-nets cipher per trustee.
	// On trustees, there is one DC-nets cipher per client.
	dcCiphers []abstract.Cipher

	xorBuffer []byte
}

// SimpleCoderFactory is a simple DC-net encoder providing no disruption or equivocation protection,
// for experimentation and baseline performance evaluations.
func SimpleCoderFactory() CellCoder {
	return new(simpleCoder)
}

///// Client methods /////

func (c *simpleCoder) ClientCellSize(payloadLength int) int {
	return payloadLength // no expansion
}

func (c *simpleCoder) ClientSetup(suite abstract.Suite,
	sharedSecrets []abstract.Cipher) {
	c.suite = suite
	keySize := suite.Cipher(nil).KeySize()

	// Use the provided shared secrets to seed
	// a pseudorandom DC-nets ciphers shared with each peer.
	npeers := len(sharedSecrets)
	c.dcCiphers = make([]abstract.Cipher, npeers)
	for i := range sharedSecrets {
		key := make([]byte, keySize)
		sharedSecrets[i].Partial(key, key, nil)
		c.dcCiphers[i] = suite.Cipher(key)
	}
}

func (c *simpleCoder) ClientEncode(payload []byte, payloadLength int,
	history abstract.Cipher) []byte {

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

///// Trustee methods /////

func (c *simpleCoder) TrusteeCellSize(payloadLength int) int {
	return payloadLength // no expansion
}

func (c *simpleCoder) TrusteeSetup(suite abstract.Suite,
	sharedSecrets []abstract.Cipher) []byte {
	c.ClientSetup(suite, sharedSecrets)
	return nil
}

func (c *simpleCoder) TrusteeEncode(payloadLength int) []byte {
	emptyCode := abstract.Cipher{}
	return c.ClientEncode(nil, payloadLength, emptyCode)
}

///// Relay methods /////

func (c *simpleCoder) RelaySetup(suite abstract.Suite, trusteeInfo [][]byte) {
	// nothing to do
}

func (c *simpleCoder) DecodeStart(payloadLength int, history abstract.Cipher) {
	c.xorBuffer = make([]byte, payloadLength)
}

func (c *simpleCoder) DecodeClient(slice []byte) {
	for i := range slice {
		c.xorBuffer[i] ^= slice[i]
	}
}

func (c *simpleCoder) DecodeTrustee(slice []byte) {
	c.DecodeClient(slice)
}

func (c *simpleCoder) DecodeCell() []byte {
	return c.xorBuffer
}
