package dcnet

import (
	"bytes"
	"github.com/lbarman/prifi/prifi-lib/config"
	"github.com/lbarman/prifi/prifi-lib/crypto"
	"gopkg.in/dedis/kyber.v2"
	"gopkg.in/dedis/onet.v1/log"
	"testing"
)

func TestEquivocation(t *testing.T) {

	rangeTest := []int{100, 1000, 10000}
	repeat := 100

	for _, dataLen := range rangeTest {
		log.Lvl1("Testing for data length", dataLen)
		for i := 0; i < repeat; i++ {
			equivocationTestForDataLength(t, dataLen)
		}
	}
}

func equivocationTestForDataLength(t *testing.T, payloadSize int) {

	// set up the Shared secrets
	tpub, _ := crypto.NewKeyPair()
	_, c1priv := crypto.NewKeyPair()
	_, c2priv := crypto.NewKeyPair()

	sharedSecret_c1 := config.CryptoSuite.Point().Mul(tpub, c1priv)
	sharedSecret_c2 := config.CryptoSuite.Point().Mul(tpub, c2priv)

	sharedPRNGs_t := make([]kyber.Cipher, 2)
	sharedPRNGs_c1 := make([]kyber.Cipher, 1)
	sharedPRNGs_c2 := make([]kyber.Cipher, 1)

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

	// set up the DC-nets
	dcnet_Trustee := NewDCNetEntity(0, DCNET_TRUSTEE, payloadSize, false, sharedPRNGs_t)
	dcnet_Client1 := NewDCNetEntity(0, DCNET_CLIENT, payloadSize, false, sharedPRNGs_c1)
	dcnet_Client2 := NewDCNetEntity(1, DCNET_CLIENT, payloadSize, false, sharedPRNGs_c2)

	data := randomBytes(payloadSize)

	// get the pads
	padRound2_t := DCNetCipherFromBytes(dcnet_Trustee.TrusteeEncodeForRound(0))
	padRound1_c1 := DCNetCipherFromBytes(dcnet_Client1.EncodeForRound(0, true, data))
	padRound1_c2 := DCNetCipherFromBytes(dcnet_Client2.EncodeForRound(0, false, nil))

	res := make([]byte, payloadSize)
	for i := range padRound1_c2.Payload {
		v := padRound1_c1.Payload[i]
		v ^= padRound1_c2.Payload[i] ^ padRound2_t.Payload[i]
		res[i] = v
	}

	// assert that the pads works
	for i, v := range res {
		if v != data[i] {
			t.Fatal("Res is not equal to data, DC-nets did not cancel out! go test dcnet/")
		}
	}

	// prepare for equivocation

	payload := randomBytes(payloadSize)

	e_client0 := NewEquivocation()
	e_client1 := NewEquivocation()
	e_trustee := NewEquivocation()
	e_relay := NewEquivocation()

	// set some data as downstream history

	historyBytes := make([]byte, 10)
	historyBytes[1] = 1

	e_client0.UpdateHistory(historyBytes)
	e_client1.UpdateHistory(historyBytes)
	e_trustee.UpdateHistory(historyBytes)
	e_relay.UpdateHistory(historyBytes)

	// start the actual equivocation

	pads1 := make([][]byte, 1)
	pads1[0] = padRound1_c1.Payload
	x_prim1, kappa1 := e_client0.ClientEncryptPayload(true, payload, pads1)

	pads2 := make([][]byte, 1)
	pads2[0] = padRound1_c2.Payload
	_, kappa2 := e_client1.ClientEncryptPayload(false, nil, pads2)

	pads3 := make([][]byte, 2)
	pads3[0] = padRound1_c1.Payload
	pads3[1] = padRound1_c2.Payload
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
