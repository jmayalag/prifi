package scheduler

import (
	"errors"
	"github.com/lbarman/prifi/prifi-lib/config"
	"gopkg.in/dedis/kyber.v2"
	"strconv"
)

/**
 * Tests that all trustees signed correctly the [lastBase, ephPubKey array].
 * Locate our slot (position in the shuffle) given the ephemeral public key and the new base
 */
func (n *NeffShuffle) ClientVerifySigAndRecognizeSlot(privateKey kyber.Scalar, trusteesPublicKeys []kyber.Point, lastBase kyber.Point, shuffledPublicKeys []kyber.Point, signatures [][]byte) (int, error) {

	if privateKey == nil {
		return -1, errors.New("Can't verify without private key")
	}
	if trusteesPublicKeys == nil {
		return -1, errors.New("Can't verify without trustee's public keys")
	}
	if signatures == nil {
		return -1, errors.New("Can't verify without trustee's public signatures")
	}
	if shuffledPublicKeys == nil {
		return -1, errors.New("Can't verify without ephemeral public keys")
	}
	if lastBase == nil {
		return -1, errors.New("Can't verify without last base")
	}
	if len(shuffledPublicKeys) < 1 {
		return -1, errors.New("Can't verify without ephemeral public keys (len=0)")
	}
	if len(signatures) != len(trusteesPublicKeys) {
		return -1, errors.New("Can't verify if len(sig) != len(trusteesPublicKeys), " + strconv.Itoa(len(signatures)) + " != " + strconv.Itoa(len(trusteesPublicKeys)) + ".")
	}

	//batch-verify all signatures
	success, err := multiSigVerify(trusteesPublicKeys, lastBase, shuffledPublicKeys, signatures)
	if success != true {
		return -1, err
	}

	//locate our public key in shuffle
	publicKeyInNewBase := config.CryptoSuite.Point().Mul(privateKey, lastBase)

	mySlot := -1

	for j := 0; j < len(shuffledPublicKeys); j++ {
		if shuffledPublicKeys[j].Equal(publicKeyInNewBase) {
			mySlot = j
		}
	}

	if mySlot == -1 {
		return -1, errors.New("Could not locate my slot")
	}
	return mySlot, nil
}
