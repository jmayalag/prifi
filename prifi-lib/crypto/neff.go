package crypto

import (
	"math/rand"

	"errors"
	"github.com/dedis/prifi/prifi-lib/config"
	"gopkg.in/dedis/kyber.v2"
)

// NeffShuffle implements Andrew Neff's verifiable shuffle proof scheme as described in the
// paper "Verifiable Mixing (Shuffling) of ElGamal Pairs", April 2004.
// The function randomly shuffles and re-randomizes a set of ElGamal pairs,
// producing a correctness proof in the process.
// Returns (Xbar,Ybar), the shuffled and randomized pairs.
func NeffShuffle(publicKeys []kyber.Point, base kyber.Point, doShufflePositions bool) ([]kyber.Point, kyber.Point, kyber.Scalar, []byte, error) {

	if base == nil {
		return nil, nil, nil, nil, errors.New("Cannot perform a shuffle is base is nil")
	}
	if publicKeys == nil {
		return nil, nil, nil, nil, errors.New("Cannot perform a shuffle is publicKeys is nil")
	}
	if len(publicKeys) == 0 {
		return nil, nil, nil, nil, errors.New("Cannot perform a shuffle is len(publicKeys) is 0")
	}
	suite := config.CryptoSuite

	//compute new shares
	secretCoeff := suite.Scalar().Pick(suite.RandomStream())
	newBase := suite.Point().Mul(secretCoeff, base)

	//transform the public keys with the secret coeff
	publicKeys2 := make([]kyber.Point, len(publicKeys))
	for i := 0; i < len(publicKeys); i++ {
		oldKey := publicKeys[i]
		publicKeys2[i] = suite.Point().Mul(secretCoeff, oldKey)
	}

	//shuffle the array
	if doShufflePositions {
		publicKeys3 := make([]kyber.Point, len(publicKeys2))
		perm := rand.Perm(len(publicKeys2))
		for i, v := range perm {
			publicKeys3[v] = publicKeys2[i]
		}
		publicKeys2 = publicKeys3
	}

	proof := make([]byte, 50) // TODO : the proof should be done

	return publicKeys2, newBase, secretCoeff, proof, nil
}
