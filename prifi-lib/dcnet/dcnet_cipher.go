package dcnet

import (
	"encoding/binary"
)

// DCNetCipher is the output of a DC-net round
type DCNetCipher struct {
	equivocationProtectionTag []byte
	payload                   []byte
}

// Converts the DCNetCipher to []byte
func (c *DCNetCipher) ToBytes() []byte {
	out := make([]byte, 8)
	equivocationTagStart := -1
	payloadStart := 8

	if c.equivocationProtectionTag != nil {
		equivocationTagStart = 8
		payloadStart += len(c.equivocationProtectionTag)
	}

	binary.BigEndian.PutUint32(out[0:4], uint32(equivocationTagStart))
	binary.BigEndian.PutUint32(out[4:8], uint32(payloadStart))

	if c.equivocationProtectionTag != nil {
		out = append(out, c.equivocationProtectionTag...)
	}
	out = append(out, c.payload...)

	return out
}

// Decodes some bytes into a DCNetCipher
func DCNetCipherFromBytes(data []byte) *DCNetCipher {
	c := new(DCNetCipher)

	if len(data) < 8 {
		panic("DCNetCipherFromBytes: data too short")
	}

	minusOneInUint32 := 4294967295

	equivocationTagStart := int(binary.BigEndian.Uint32(data[0:4]))
	payloadStart := int(binary.BigEndian.Uint32(data[4:8]))

	if equivocationTagStart != minusOneInUint32 {
		c.equivocationProtectionTag = data[8:payloadStart]
	}

	c.payload = data[payloadStart:]

	return c
}