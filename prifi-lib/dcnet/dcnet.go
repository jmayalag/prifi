package dcnet

import (
	"fmt"
	"github.com/lbarman/prifi/prifi-lib/config"
	"gopkg.in/dedis/kyber.v2"
	"gopkg.in/dedis/kyber.v2/suites"
	"gopkg.in/dedis/onet.v2/log"
	"strconv"
)

// Relay, Trustee or Client
type DCNET_ENTITY int

const (
	// Define this DCNET entity as a client
	DCNET_CLIENT DCNET_ENTITY = iota

	// Define this DCNET entity as a trustee
	DCNET_TRUSTEE

	// Define this DCNET entity as a relay
	DCNET_RELAY
)

// A struct with all methods to encode and decode dc-net messages
type DCNetEntity struct {
	//Global for all nodes
	EntityID                      int
	Entity                        DCNET_ENTITY
	EquivocationProtectionEnabled bool
	DCNetPayloadSize              int

	cryptoSuite  suites.Suite
	sharedKeys   []kyber.Point // keys shared with other DC-net members
	sharedPRNGs  []kyber.XOF // PRNGs shared with other DC-net members (seeded with sharedKeys)
	currentRound int32

	//Used by the relay
	DCNetRoundDecoder *DCNetRoundDecoder //nil if unused

	//Equivocation protection
	equivocationProtection    *EquivocationProtection //nil if unused
	equivocationContribLength int                     //0 if equivocation protection is disabled

	verbose bool
}

// DCNetRoundDecoder is used by the relay to decode the dcnet ciphers
type DCNetRoundDecoder struct {
	currentRoundBeingDecoded int32
	xorBuffer                []byte
	equivTrusteeContribs     [][]byte
	equivClientContribs      [][]byte
}

// Used by clients, trustees
func NewDCNetEntity(
	entityID int,
	entity DCNET_ENTITY,
	PayloadSize int,
	equivocationProtection bool,
	sharedKeys []kyber.Point) *DCNetEntity {

	e := new(DCNetEntity)
	e.EntityID = entityID
	e.Entity = entity
	e.DCNetPayloadSize = PayloadSize
	e.EquivocationProtectionEnabled = equivocationProtection
	e.DCNetRoundDecoder = nil
	e.currentRound = 0

	e.verbose = false // todo: wire in the .toml

	if equivocationProtection {
		e.equivocationProtection = NewEquivocation()
	}

	e.cryptoSuite = config.CryptoSuite

	// if the node participates in the DC-net
	if entity != DCNET_RELAY {
		e.sharedKeys = sharedKeys

		// Use the provided shared secrets to seed a pseudorandom DC-nets ciphers shared with each peer.
		e.sharedPRNGs = make([]kyber.XOF, len(sharedKeys))
		for i := range sharedKeys {
			e.verbosePrint("key", i, ":", sharedKeys[i])
			seed, err := sharedKeys[i].Data()
			if err != nil {
				log.Fatal("Could not extract data from shared key", err)
			}
			e.sharedPRNGs[i] = e.cryptoSuite.XOF(seed)
		}
	} else {
		e.sharedKeys = make([]kyber.Point, 0)
		e.sharedPRNGs = make([]kyber.XOF, 0)
	}

	// if the equivocation protection is enabled
	if equivocationProtection {
		e.verbosePrint("equivocation = true")
		e.equivocationProtection = NewEquivocation()
		zero := e.equivocationProtection.suite.Scalar().Zero()
		one := e.equivocationProtection.suite.Scalar().One()
		minusOne := e.equivocationProtection.suite.Scalar().Sub(zero, one) //max value
		e.equivocationContribLength = minusOne.MarshalSize()
	} else {
		e.verbosePrint("equivocation = false")
	}

	// make sure we can still encode stuff !
	if e.DCNetPayloadSize <= 0 {
		panic("Payload length is" + strconv.Itoa(e.DCNetPayloadSize))
	}

	return e
}

func (e *DCNetEntity) verbosePrint(info ...interface{}) {
	if !e.verbose {
		return
	}

	s := "DCNet"

	if e.Entity == DCNET_RELAY {
		s += "[relay]:"
	} else if e.Entity == DCNET_CLIENT {
		s += "[client-" + strconv.Itoa(e.EntityID) + "]:"
	} else if e.Entity == DCNET_TRUSTEE {
		s += "[trustee-" + strconv.Itoa(e.EntityID) + "]:"
	} else {
		s += "[???]"
	}

	s2 := fmt.Sprint(info...)
	log.Lvl1(s, s2)
}

// Encodes "Payload" in the correct round. Will skip PRNG material if the round is in the future,
// and crash if the round is in the past or the Payload is too long
func (e *DCNetEntity) TrusteeEncodeForRound(roundID int32) []byte {
	return e.EncodeForRound(roundID, false, nil)
}

// Encodes "Payload" in the correct round. Will skip PRNG material if the round is in the future,
// and crash if the round is in the past or the Payload is too long
func (e *DCNetEntity) EncodeForRound(roundID int32, slotOwner bool, payload []byte) []byte {
	if len(payload) > e.DCNetPayloadSize {
		panic("DCNet: cannot encode Payload of length " + strconv.Itoa(int(len(payload))) + " max length is " + strconv.Itoa(len(payload)))
	}

	if roundID < e.currentRound {
		panic("DCNet: asked to encode for round " + strconv.Itoa(int(roundID)) + " but we are at  round " + strconv.Itoa(int(e.currentRound)))
	}

	for e.currentRound < roundID {
		//discard crypto material
		log.Lvl4("DCNet: Discarding round", e.currentRound)

		// consume the PRNGs
		for i := range e.sharedPRNGs {
			dummy := make([]byte, e.DCNetPayloadSize)
			e.sharedPRNGs[i].XORKeyStream(dummy, dummy)
		}

		e.currentRound++
	}

	var c *DCNetCipher
	if e.Entity == DCNET_CLIENT {
		c = e.clientEncode(slotOwner, payload)
	} else {
		c = e.trusteeEncode()
	}
	e.currentRound++

	e.verbosePrint("r[", roundID, "]:\n", c.Payload)
	e.verbosePrint("r[", roundID, "]: equiv\n", c.EquivocationProtectionTag)

	return c.ToBytes()
}

// Adds `newdata` into the sponge representing the received downstream data
func (e *DCNetEntity) UpdateReceivedMessageHistory(newData []byte) {
	if e.EquivocationProtectionEnabled {
		e.equivocationProtection.UpdateHistory(newData)
	}
}

func (e *DCNetEntity) clientEncode(slotOwner bool, payload []byte) *DCNetCipher {
	c := new(DCNetCipher)

	if payload == nil {
		payload = make([]byte, e.DCNetPayloadSize)
	} else {
		// deep clone and pad
		payload2 := make([]byte, e.DCNetPayloadSize)
		copy(payload2[0:len(payload)], payload)
		payload = payload2
	}
	c.Payload = payload

	// prepare the pads
	p_ij := make([][]byte, len(e.sharedPRNGs))
	for i := range p_ij {
		p_ij[i] = make([]byte, e.DCNetPayloadSize)
		e.sharedPRNGs[i].XORKeyStream(p_ij[i], p_ij[i])
	}

	// if the equivocation protection is enabled, encrypt the Payload, and add the tag
	if e.EquivocationProtectionEnabled {
		payload, sigma_j := e.equivocationProtection.ClientEncryptPayload(slotOwner, payload, p_ij)
		e.verbosePrint("payload\n", payload)
		e.verbosePrint("sigma_j\n", sigma_j)
		c.Payload = payload // replace the Payload with the encrypted version
		c.EquivocationProtectionTag = sigma_j
	}

	// DC-net encrypt the Payload
	for i := range p_ij {
		for k := range c.Payload {
			c.Payload[k] ^= p_ij[i][k] // XORs in the pads
		}
	}

	return c
}

func (e *DCNetEntity) trusteeEncode() *DCNetCipher {
	c := new(DCNetCipher)

	c.Payload = make([]byte, e.DCNetPayloadSize)

	// prepare the pads
	p_ij := make([][]byte, len(e.sharedPRNGs))
	for i := range p_ij {
		p_ij[i] = make([]byte, e.DCNetPayloadSize)
		e.sharedPRNGs[i].XORKeyStream(p_ij[i], p_ij[i])
	}

	// DC-net encrypt the Payload
	for i := range p_ij {
		for k := range c.Payload {
			c.Payload[k] ^= p_ij[i][k] // XORs in the pads
		}
	}

	// if the equivocation protection is enabled, encrypt the Payload, and add the tag
	if e.EquivocationProtectionEnabled {
		sigma_j := e.equivocationProtection.TrusteeGetContribution(p_ij)
		c.EquivocationProtectionTag = sigma_j
	}

	return c
}

// Used by the relay to start decoding a round
func (e *DCNetEntity) DecodeStart(roundID int32) {
	e.DCNetRoundDecoder = new(DCNetRoundDecoder)
	e.DCNetRoundDecoder.currentRoundBeingDecoded = roundID
	e.DCNetRoundDecoder.xorBuffer = make([]byte, e.DCNetPayloadSize)
	e.DCNetRoundDecoder.equivClientContribs = make([][]byte, 0)
	e.DCNetRoundDecoder.equivTrusteeContribs = make([][]byte, 0)
}

// called by the relay to decode a client contribution
func (e *DCNetEntity) DecodeClient(roundID int32, slice []byte) {

	dcNetCipher := DCNetCipherFromBytes(slice)

	if roundID != e.DCNetRoundDecoder.currentRoundBeingDecoded {
		panic("Cannot DecodeClient for round" +
			strconv.Itoa(int(roundID)) + ", we are in round " + strconv.Itoa(int(e.DCNetRoundDecoder.currentRoundBeingDecoded)))
	}

	for i := range dcNetCipher.Payload {
		e.DCNetRoundDecoder.xorBuffer[i] ^= dcNetCipher.Payload[i]
	}

	if e.EquivocationProtectionEnabled {
		e.DCNetRoundDecoder.equivClientContribs = append(e.DCNetRoundDecoder.equivClientContribs, dcNetCipher.EquivocationProtectionTag)
	}
}

// called by the relay to decode a client contribution
func (e *DCNetEntity) DecodeTrustee(roundID int32, slice []byte) {

	dcNetCipher := DCNetCipherFromBytes(slice)

	if roundID != e.DCNetRoundDecoder.currentRoundBeingDecoded {
		panic("Cannot DecodeClient for round" +
			strconv.Itoa(int(roundID)) + ", we are in round " + strconv.Itoa(int(e.DCNetRoundDecoder.currentRoundBeingDecoded)))
	}

	for i := range dcNetCipher.Payload {
		e.DCNetRoundDecoder.xorBuffer[i] ^= dcNetCipher.Payload[i]
	}

	if e.EquivocationProtectionEnabled {
		e.DCNetRoundDecoder.equivTrusteeContribs = append(e.DCNetRoundDecoder.equivTrusteeContribs, dcNetCipher.EquivocationProtectionTag)
	}
}

// Called on the relay to decode the cell, after having stored the cryptographic materials
func (e *DCNetEntity) DecodeCell() []byte {
	//No Equivocation -> just XOR
	d := e.DCNetRoundDecoder

	decoded := d.xorBuffer
	if e.EquivocationProtectionEnabled {
		decoded = e.equivocationProtection.RelayDecode(d.xorBuffer, d.equivTrusteeContribs, d.equivClientContribs)
	}

	return decoded
}
