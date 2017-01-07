package crypto

import (
	"math/rand"

	"errors"
	"github.com/dedis/crypto/abstract"
	"github.com/dedis/crypto/random"
)

// NeffShuffle implements Andrew Neff's verifiable shuffle proof scheme as described in the
// paper "Verifiable Mixing (Shuffling) of ElGamal Pairs", April 2004.
// The function randomly shuffles and re-randomizes a set of ElGamal pairs,
// producing a correctness proof in the process.
// Returns (Xbar,Ybar), the shuffled and randomized pairs.
func NeffShuffle(publicKeys []abstract.Point, base abstract.Point, suite abstract.Suite, doShufflePositions bool) ([]abstract.Point, abstract.Point, abstract.Scalar, []byte, error) {

	if base == nil {
		return nil, nil, nil, nil, errors.New("Cannot perform a shuffle is base is nil")
	}
	if publicKeys == nil {
		return nil, nil, nil, nil, errors.New("Cannot perform a shuffle is publicKeys is nil")
	}
	if len(publicKeys) == 0 {
		return nil, nil, nil, nil, errors.New("Cannot perform a shuffle is len(publicKeys) is 0")
	}
	if suite == nil {
		return nil, nil, nil, nil, errors.New("Cannot perform a shuffle without a suite")
	}

	//compute new shares
	secretCoeff := suite.Scalar().Pick(random.Stream)
	newBase := suite.Point().Mul(base, secretCoeff)

	//transform the public keys with the secret coeff
	publicKeys2 := make([]abstract.Point, len(publicKeys))
	for i := 0; i < len(publicKeys); i++ {
		oldKey := publicKeys[i]
		publicKeys2[i] = suite.Point().Mul(oldKey, secretCoeff)
	}

	//shuffle the array
	if doShufflePositions {
		publicKeys3 := make([]abstract.Point, len(publicKeys2))
		perm := rand.Perm(len(publicKeys2))
		for i, v := range perm {
			publicKeys3[v] = publicKeys2[i]
		}
		publicKeys2 = publicKeys3
	}

	proof := make([]byte, 50) // TODO : the proof should be done

	return publicKeys2, newBase, secretCoeff, proof, nil
}
