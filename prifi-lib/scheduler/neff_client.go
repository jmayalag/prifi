package scheduler

import (
	"errors"
	"github.com/dedis/crypto/abstract"
	"github.com/lbarman/prifi/prifi-lib/config"
	"strconv"
)

/**
 * Tests that all trustees signed correctly the [lastBase, ephPubKey array].
 * Locate our slot (position in the shuffle) given the ephemeral public key and the new base
 */
func (n *neffShuffleScheduler) ClientVerifySigAndRecognizeSlot(privateKey abstract.Scalar, trusteesPublicKeys []abstract.Point, shares abstract.Scalar, shuffledPublicKeys []abstract.Point, signatures [][]byte) (error, int) {

	if privateKey == nil {
		return errors.New("Can't verify without private key"), -1
	}
	if trusteesPublicKeys == nil {
		return errors.New("Can't verify without trustee's public keys"), -1
	}
	if trusteesPublicKeys == nil {
		return errors.New("Can't verify without trustee's public signatures"), -1
	}
	if shuffledPublicKeys == nil {
		return errors.New("Can't verify without ephemeral public keys"), -1
	}
	if shares == nil {
		return errors.New("Can't verify without last base"), -1
	}
	if len(shuffledPublicKeys) < 1 {
		return errors.New("Can't verify without ephemeral public keys (len=0)"), -1
	}
	if len(signatures) != len(trusteesPublicKeys) {
		return errors.New("Can't verify if len(sig) != len(trusteesPublicKeys), " + strconv.Itoa(len(signatures)) + " != " + strconv.Itoa(len(trusteesPublicKeys)) + "."), -1
	}

	//batch-verify all signatures
	err, success := multiSigVerify(trusteesPublicKeys, shares, shuffledPublicKeys, signatures)
	if success != true {
		return err, -1
	}

	//locate our public key in shuffle
	G := config.CryptoSuite.Point().Base()
	newBase := config.CryptoSuite.Point().Mul(G, shares)
	publicKeyInNewBase := config.CryptoSuite.Point().Mul(newBase, privateKey)

	mySlot := -1

	for j := 0; j < len(shuffledPublicKeys); j++ {
		if shuffledPublicKeys[j].Equal(publicKeyInNewBase) {
			mySlot = j
		}
	}

	if mySlot == -1 {
		return errors.New("Could not locate my slot"), -1
	}
	return nil, mySlot
}
