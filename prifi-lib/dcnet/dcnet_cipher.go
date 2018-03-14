package dcnet

import (
	"encoding/binary"
)

// DCNetCipher is the output of a DC-net round
type DCNetCipher struct {
	disruptionProtectionTag   []byte
	equivocationProtectionTag []byte
	payload                   []byte
}

// Converts the DCNetCipher to []byte
func (c *DCNetCipher) ToBytes() []byte {
	out := make([]byte, 12)
	disruptionTagStart := -1
	equivocationTagStart := -1
	payloadStart := 12

	if c.disruptionProtectionTag != nil {
		disruptionTagStart = 12
		payloadStart += len(c.disruptionProtectionTag)
	}
	if c.equivocationProtectionTag != nil {
		equivocationTagStart = 12 + len(c.disruptionProtectionTag)
		payloadStart += len(c.equivocationProtectionTag)
	}

	binary.BigEndian.PutUint32(out[0:4], uint32(disruptionTagStart))
	binary.BigEndian.PutUint32(out[4:8], uint32(equivocationTagStart))
	binary.BigEndian.PutUint32(out[8:12], uint32(payloadStart))

	if c.disruptionProtectionTag != nil {
		out = append(out, c.disruptionProtectionTag...)
	}
	if c.equivocationProtectionTag != nil {
		out = append(out, c.equivocationProtectionTag...)
	}
	out = append(out, c.payload...)

	return out
}

// Decodes some bytes into a DCNetCipher
func DCNetCipherFromBytes(data []byte) *DCNetCipher {
	c := new(DCNetCipher)

	if len(data) < 12 {
		panic("DCNetCipherFromBytes: data too short")
	}

	minusOneInUint32 := 4294967295

	disruptionTagStart := int(binary.BigEndian.Uint32(data[0:4]))
	equivocationTagStart := int(binary.BigEndian.Uint32(data[4:8]))
	payloadStart := int(binary.BigEndian.Uint32(data[8:12]))

	if disruptionTagStart != minusOneInUint32 { // -1
		// then disruptionTagStart = 12
		end := payloadStart
		if equivocationTagStart != minusOneInUint32 { // -1
			end = equivocationTagStart
		}
		c.disruptionProtectionTag = data[12:end]
	}

	if equivocationTagStart != minusOneInUint32 {
		c.equivocationProtectionTag = data[equivocationTagStart:payloadStart]
	}

	c.payload = data[payloadStart:]

	return c
}