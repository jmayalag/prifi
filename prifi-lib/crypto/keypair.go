package crypto

import (
	"github.com/dedis/prifi/prifi-lib/config"
	"gopkg.in/dedis/kyber.v2"
)

/**
 * creates a public, private key pair using the cryptosuite in config
 */
func NewKeyPair() (kyber.Point, kyber.Scalar) {

	base := config.CryptoSuite.Point().Base()
	priv := config.CryptoSuite.Scalar().Pick(config.CryptoSuite.RandomStream())
	pub := config.CryptoSuite.Point().Mul(priv, base)

	return pub, priv
}
