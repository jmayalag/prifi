package crypto

import (
	"github.com/dedis/crypto/abstract"
	"github.com/dedis/crypto/random"
)

func ElGamalEncrypt(suite abstract.Suite, pubkey abstract.Point, message []byte) (
	K, C abstract.Point, remainder []byte) {

	// Embed the message (or as much of it as will fit) into a curve point.
	M, remainder := suite.Point().Pick(message, random.Stream)

	// ElGamal-encrypt the point to produce ciphertext (K,C).
	k := suite.Scalar().Pick(random.Stream) // ephemeral private key
	K = suite.Point().Mul(nil, k)           // ephemeral DH public key
	S := suite.Point().Mul(pubkey, k)       // ephemeral DH shared secret
	C = S.Add(S, M)                         // message blinded with secret
	return
}

func ElGamalDecrypt(suite abstract.Suite, prikey abstract.Scalar, K, C abstract.Point) (
	message []byte, err error) {

	// ElGamal-decrypt the ciphertext (K,C) to reproduce the message.
	S := suite.Point().Mul(K, prikey) // regenerate shared secret
	M := suite.Point().Sub(C, S)      // use to un-blind the message
	message, err = M.Data()           // extract the embedded data
	return
}
