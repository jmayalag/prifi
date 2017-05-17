package client

import (
	"github.com/lbarman/prifi/prifi-lib/config"
	"github.com/lbarman/prifi/prifi-lib/crypto"
	"github.com/lbarman/prifi/prifi-lib/dcnet"
	"gopkg.in/dedis/crypto.v0/abstract"
	"testing"
)

func TestDCNetFF(test *testing.T) {
	dc := new(DCNet_FastForwarder)
	dc.CellCoder = dcnet.SimpleCoderFactory()

	dc2 := new(DCNet_FastForwarder)
	dc2.CellCoder = dcnet.SimpleCoderFactory()

	//set up the DC-nets
	_, clientp := crypto.NewKeyPair()
	trusteeP, _ := crypto.NewKeyPair()
	sharedSecret := config.CryptoSuite.Point().Mul(trusteeP, clientp)
	sharedPRNGs := make([]abstract.Cipher, 1)
	sharedPRNGs2 := make([]abstract.Cipher, 1)
	bytes, err := sharedSecret.MarshalBinary()
	if err != nil {
		test.Fatal("Could not marshal point !")
	}
	sharedPRNGs[0] = config.CryptoSuite.Cipher(bytes)
	sharedPRNGs2[0] = config.CryptoSuite.Cipher(bytes)

	dc.CellCoder.ClientSetup(config.CryptoSuite, sharedPRNGs)
	dc2.CellCoder.ClientSetup(config.CryptoSuite, sharedPRNGs2)

	cellSize := 8
	var history abstract.Cipher

	//the "real" rounds
	round0 := dc.CellCoder.ClientEncode(nil, cellSize, history)
	_ = dc.CellCoder.ClientEncode(nil, cellSize, history)
	round2 := dc.CellCoder.ClientEncode(nil, cellSize, history)

	//the skipped rounds
	round02 := dc2.ClientEncodeForRound(0, nil, cellSize, history)
	round22 := dc2.ClientEncodeForRound(2, nil, cellSize, history)

	if !testEq(round0, round02) {
		test.Error("Round 0 should be the same at both places")
	}
	if !testEq(round2, round22) {
		test.Error("Round 2 should be the same at both places")
	}
}

func testEq(a, b []byte) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
