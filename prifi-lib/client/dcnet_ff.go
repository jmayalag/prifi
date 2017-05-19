package client

import (
	"github.com/lbarman/prifi/prifi-lib/dcnet"
	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/onet.v1/log"
)

// DCNet_FastForwarder allows to request DC-net pads for a specific round
type DCNet_FastForwarder struct {
	CellCoder    dcnet.CellCoder
	currentRound int32
}

// ClientEncodeForRound allows to request DC-net pads for a specific round
func (dc *DCNet_FastForwarder) ClientEncodeForRound(roundID int32, payload []byte, payloadSize int, history abstract.Cipher) []byte {

	for dc.currentRound < roundID {
		//discard crypto material
		log.Lvl4("Discarding round", dc.currentRound)
		_ = dc.CellCoder.ClientEncode(nil, payloadSize, history)
		dc.currentRound++
	}

	log.Lvl4("Producing round", dc.currentRound)
	//produce the real round
	data := dc.CellCoder.ClientEncode(payload, payloadSize, history)
	dc.currentRound++
	return data
}
