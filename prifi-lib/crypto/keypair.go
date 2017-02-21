package crypto

import (
	"github.com/lbarman/prifi/prifi-lib/config"
	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/crypto.v0/random"
)

/**
 * creates a public, private key pair using the cryptosuite in config
 */
func NewKeyPair() (abstract.Point, abstract.Scalar) {

	base := config.CryptoSuite.Point().Base()
	priv := config.CryptoSuite.Scalar().Pick(random.Stream)
	pub := config.CryptoSuite.Point().Mul(base, priv)

	return pub, priv
}
