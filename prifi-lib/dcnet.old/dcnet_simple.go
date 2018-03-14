package dcnet_old

import (
	"gopkg.in/dedis/crypto.v0/abstract"
)

type simpleDCNet struct {
	suite abstract.Suite

	equivocationProtection *EquivocationProtection
	equivContribLength     int

	// Pseudorandom DC-nets ciphers shared with each peer.
	// On clients, there is one cipher per trustee.
	// On trustees, there is one cipher per client.
	dcCiphers []abstract.Cipher

	// Used only on the relay for decoding
	xorBuffer            []byte
	equivTrusteeContribs [][]byte
	equivClientContribs  [][]byte
}

// SimpleCoderFactory is a simple DC-net encoder providing no disruption or equivocation protection,
// for experimentation and baseline performance evaluations.
func NewSimpleDCNet(equivocationProtectionEnabled bool) DCNet {
	dc := new(simpleDCNet)
	if equivocationProtectionEnabled {
		dc.equivocationProtection = NewEquivocation()
		zero := dc.equivocationProtection.suite.Scalar().Zero()
		one := dc.equivocationProtection.suite.Scalar().One()
		minusOne := dc.equivocationProtection.suite.Scalar().Sub(zero, one) //max value
		dc.equivContribLength = len(minusOne.Bytes())
	}
	return dc
}

func (c *simpleDCNet) UpdateHistory(data []byte) {
	c.equivocationProtection.UpdateHistory(data)
}

func (c *simpleDCNet) GetClientCipherSize(payloadLength int) int {
	if c.equivocationProtection != nil {
		return payloadLength + c.equivContribLength
	}
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

	//No Equivocation -> just XOR
	if c.equivocationProtection == nil {
		for i := range c.dcCiphers {
			c.dcCiphers[i].XORKeyStream(payload, payload)
		}
		return payload
	}

	//equivocation -> split, get payload, keep trustee contrib
	p_ij := make([][]byte, len(c.dcCiphers))
	for i := range c.dcCiphers {
		p_ij[i] = make([]byte, payloadLength)
		c.dcCiphers[i].XORKeyStream(p_ij[i], p_ij[i])

		for k := range payload {
			payload[k] ^= p_ij[i][k]
		}
	}

	payload2, sigma_j := c.equivocationProtection.ClientEncryptPayload(payload, p_ij)

	result := make([]byte, len(sigma_j)+payloadLength)
	copy(result[0:len(sigma_j)], sigma_j[:])
	copy(result[len(sigma_j):], payload2[:])
	return result
}

func (c *simpleDCNet) GetTrusteeCipherSize(payloadLength int) int {
	if c.equivocationProtection != nil {
		return payloadLength + c.equivContribLength
	}
	return payloadLength // no expansion
}

func (c *simpleDCNet) TrusteeSetup(suite abstract.Suite, sharedSecrets []abstract.Cipher) []byte {
	c.ClientSetup(suite, sharedSecrets)
	return nil
}

func (c *simpleDCNet) TrusteeEncode(payloadLength int) []byte {

	payload := make([]byte, payloadLength)

	//No Equivocation -> just XOR
	if c.equivocationProtection == nil {
		for i := range c.dcCiphers {
			c.dcCiphers[i].XORKeyStream(payload, payload)
		}
		return payload
	}

	//equivocation -> split, get payload, keep trustee contrib
	p_ij := make([][]byte, len(c.dcCiphers))
	for i := range c.dcCiphers {
		p_ij[i] = make([]byte, payloadLength)
		c.dcCiphers[i].XORKeyStream(p_ij[i], p_ij[i])

		for k := range payload {
			payload[k] ^= p_ij[i][k]
		}
	}

	sigma_j := c.equivocationProtection.TrusteeGetContribution(p_ij)

	result := make([]byte, len(sigma_j)+payloadLength)
	copy(result[0:len(sigma_j)], sigma_j[:])
	copy(result[len(sigma_j):], payload[:])
	return result
}

func (c *simpleDCNet) RelaySetup(suite abstract.Suite, trusteeInfo [][]byte) {
}

func (c *simpleDCNet) DecodeStart(payloadLength int, history abstract.Cipher) {
	c.xorBuffer = make([]byte, payloadLength)
	c.equivTrusteeContribs = make([][]byte, 0)
	c.equivClientContribs = make([][]byte, 0)
}

func (c *simpleDCNet) DecodeClient(slice []byte) {
	//No Equivocation -> just XOR
	if c.equivocationProtection == nil {
		for i := range slice {
			c.xorBuffer[i] ^= slice[i]
		}
		return
	}

	//equivocation -> split, get payload, keep trustee contrib
	sigma_j_length := c.equivContribLength

	clientContrib := slice[0:sigma_j_length]
	payload := slice[sigma_j_length:]

	for i := range payload {
		c.xorBuffer[i] ^= payload[i]
	}

	c.equivClientContribs = append(c.equivClientContribs, clientContrib)
}

func (c *simpleDCNet) DecodeTrustee(slice []byte) {
	//No Equivocation -> just XOR
	if c.equivocationProtection == nil {
		for i := range slice {
			c.xorBuffer[i] ^= slice[i]
		}
		return
	}

	//equivocation -> split, get payload, keep trustee contrib
	sigma_j_length := c.equivContribLength

	trusteeContrib := slice[0:sigma_j_length]
	payload := slice[sigma_j_length:]

	for i := range payload {
		c.xorBuffer[i] ^= payload[i]
	}

	c.equivTrusteeContribs = append(c.equivTrusteeContribs, trusteeContrib)
}

func (c *simpleDCNet) DecodeCell() []byte {
	//No Equivocation -> just XOR
	if c.equivocationProtection == nil {
		return c.xorBuffer
	}

	return c.equivocationProtection.RelayDecode(c.xorBuffer, c.equivTrusteeContribs, c.equivClientContribs)
}
