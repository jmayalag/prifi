package crypto

import (
	"github.com/dedis/crypto/abstract"
	"github.com/lbarman/prifi/prifi-lib/config"
	"testing"

	"github.com/dedis/crypto/random"
)

func TestSchnorr(t *testing.T) {

	pub, priv := genKeyPair()
	pub2, priv2 := genKeyPair()

	//with empty data
	data := make([]byte, 0)
	sig := SchnorrSign(config.CryptoSuite, random.Stream, data, priv)
	err := SchnorrVerify(config.CryptoSuite, data, pub, sig)

	if err != nil {
		t.Error("Should validate with nil message, err is " + err.Error())
	}

	//with empty data
	data = random.Bits(100, false, random.Stream)
	sig = SchnorrSign(config.CryptoSuite, random.Stream, data, priv)
	err = SchnorrVerify(config.CryptoSuite, data, pub, sig)

	if err != nil {
		t.Error("Should validate with random message, err is " + err.Error())
	}

	//should trivially not validate with other keys
	data = random.Bits(100, false, random.Stream)
	sig = SchnorrSign(config.CryptoSuite, random.Stream, data, priv2)
	err = SchnorrVerify(config.CryptoSuite, data, pub, sig)

	if err == nil {
		t.Error("Should not validate with wrong keys")
	}
	data = random.Bits(100, false, random.Stream)
	sig = SchnorrSign(config.CryptoSuite, random.Stream, data, priv)
	err = SchnorrVerify(config.CryptoSuite, data, pub2, sig)

	if err == nil {
		t.Error("Should not validate with wrong keys")
	}

}

func TestSchnorrHash(t *testing.T) {

	pub, _ := genKeyPair()
	data := random.Bits(100, false, random.Stream)
	secret := hashSchnorr(config.CryptoSuite, data, pub)

	if secret == nil {
		t.Error("Secret should not be nil")
	}
}

func genKeyPair() (abstract.Point, abstract.Scalar) {

	base := config.CryptoSuite.Point().Base()
	priv := config.CryptoSuite.Scalar().Pick(random.Stream)
	pub := config.CryptoSuite.Point().Mul(base, priv)

	return pub, priv
}
