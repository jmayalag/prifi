package dcnet

import (

)
/*
func TestEquivocation(t *testing.T) {

	rangeTest := []int{100, 1000, 10000}

	for _, dataLen := range rangeTest {
		log.Lvl1("Testing for payload length", dataLen)
		equivocationTestForDataLength(t, dataLen)
	}
}

func equivocationTestForDataLength(t *testing.T, cellSize int) {

	history := config.CryptoSuite.Cipher([]byte("init"))

	// set up the Shared secrets
	tpub, _ := crypto.NewKeyPair()
	_, c1priv := crypto.NewKeyPair()
	_, c2priv := crypto.NewKeyPair()

	sharedSecret_c1 := config.CryptoSuite.Point().Mul(tpub, c1priv)
	sharedSecret_c2 := config.CryptoSuite.Point().Mul(tpub, c2priv)

	sharedPRNGs_t := make([]abstract.Cipher, 2)
	sharedPRNGs_c1 := make([]abstract.Cipher, 1)
	sharedPRNGs_c2 := make([]abstract.Cipher, 1)

	ssBytes, err := sharedSecret_c1.MarshalBinary()
	if err != nil {
		t.Error("Could not marshal point !")
	}
	sharedPRNGs_c1[0] = config.CryptoSuite.Cipher(ssBytes)
	sharedPRNGs_t[0] = config.CryptoSuite.Cipher(ssBytes)
	ssBytes, err = sharedSecret_c2.MarshalBinary()
	if err != nil {
		t.Error("Could not marshal point !")
	}
	sharedPRNGs_c2[0] = config.CryptoSuite.Cipher(ssBytes)
	sharedPRNGs_t[1] = config.CryptoSuite.Cipher(ssBytes)

	// set up the CellCoders
	cellCodert := NewSimpleDCNet(false)
	cellCodert.TrusteeSetup(config.CryptoSuite, sharedPRNGs_t)

	cellCoderc1 := NewSimpleDCNet(false)
	cellCoderc1.ClientSetup(config.CryptoSuite, sharedPRNGs_c1)

	cellCoderc2 := NewSimpleDCNet(false)
	cellCoderc2.ClientSetup(config.CryptoSuite, sharedPRNGs_c2)

	data := make([]byte, 0) // payload is zero for both, none transmitting

	// get the pads
	padRound1_c1 := cellCoderc1.ClientEncode(data, cellSize, history)
	padRound1_c2 := cellCoderc2.ClientEncode(data, cellSize, history)
	padRound2_t := cellCodert.TrusteeEncode(cellSize)

	res := make([]byte, cellSize)
	for i := range padRound1_c2 {
		v := padRound1_c1[i]
		v ^= padRound1_c2[i] ^ padRound2_t[i]
		res[i] = v
	}

	// assert that the pads works
	for _, v := range res {
		if v != 0 {
			t.Fatal("Res is non zero, DC-nets did not cancel out! go test dcnet.old/")
		}
	}

	// prepare for equivocation

	payload := make([]byte, cellSize)
	payload[0] = 0
	payload[1] = 1

	e_client0 := NewEquivocation()
	e_client1 := NewEquivocation()
	e_trustee := NewEquivocation()
	e_relay := NewEquivocation()

	// set some payload as downstream history

	historyBytes := make([]byte, 10)
	historyBytes[1] = 1

	e_client0.UpdateHistory(historyBytes)
	e_client1.UpdateHistory(historyBytes)
	e_trustee.UpdateHistory(historyBytes)
	e_relay.UpdateHistory(historyBytes)

	// start the actual equivocation

	pads1 := make([][]byte, 1)
	pads1[0] = padRound1_c1
	x_prim1, kappa1 := e_client0.ClientEncryptPayload(payload, pads1)

	pads2 := make([][]byte, 1)
	pads2[0] = padRound1_c2
	_, kappa2 := e_client1.ClientEncryptPayload(nil, pads2)

	pads3 := make([][]byte, 2)
	pads3[0] = padRound1_c1
	pads3[1] = padRound1_c2
	sigma := e_trustee.TrusteeGetContribution(pads3)

	// relay decodes
	trusteesContrib := make([][]byte, 1)
	trusteesContrib[0] = sigma

	clientContrib := make([][]byte, 2)
	clientContrib[0] = kappa1
	clientContrib[1] = kappa2

	payloadPlaintext := e_relay.RelayDecode(x_prim1, trusteesContrib, clientContrib)

	if bytes.Compare(payload, payloadPlaintext) != 0 {
		log.Lvl1(payload)
		log.Lvl1(payloadPlaintext)
		t.Error("payloads don't match")
	}
}
*/