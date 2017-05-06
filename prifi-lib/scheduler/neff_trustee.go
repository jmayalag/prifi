package scheduler

/*

The interface should be :

INPUT : list of client's public keys

OUTPUT : list of slots

*/

import (
	"bytes"
	"errors"
	"github.com/lbarman/prifi/prifi-lib/config"
	"github.com/lbarman/prifi/prifi-lib/crypto"
	"github.com/lbarman/prifi/prifi-lib/net"
	"gopkg.in/dedis/crypto.v0/abstract"
	crypto_proof "gopkg.in/dedis/crypto.v0/proof"
	"gopkg.in/dedis/crypto.v0/random"
	"gopkg.in/dedis/crypto.v0/shuffle"
	"strconv"
)

/**
 * The view of one trustee for the Neff Shuffle
 */
type NeffShuffleTrustee struct {
	TrusteeID  int
	PrivateKey abstract.Scalar
	PublicKey  abstract.Point

	SecretCoeff   abstract.Scalar // c[i]
	NewBase       abstract.Point  // s[i] = G * c[1] ... c[1]
	Proof         []byte
	EphemeralKeys []abstract.Point
}

/**
 * Creates a new trustee-view for the neff shuffle, and initiates the fields correctly
 */
func (t *NeffShuffleTrustee) Init(trusteeID int, private abstract.Scalar, public abstract.Point) error {
	if trusteeID < 0 {
		return errors.New("Cannot shuffle without a valid id (>= 0)")
	}
	if private == nil {
		return errors.New("Cannot shuffle without a private key.")
	}
	if public == nil {
		return errors.New("Cannot shuffle without a public key.")
	}
	t.TrusteeID = trusteeID
	t.PrivateKey = private
	t.PublicKey = public
	return nil
}

/**
 * Received s[i-1], and the public keys. Do the shuffle, store locally, and send back the new s[i], shuffle array
 * If shuffleKeyPositions is false, do not shuffle the key's position (useful for testing - 0 anonymity)
 */
func (t *NeffShuffleTrustee) ReceivedShuffleFromRelay(lastBase abstract.Point, clientPublicKeys []abstract.Point, shuffleKeyPositions bool, vkey []byte) (interface{}, error) {

	if lastBase == nil {
		return nil, errors.New("Cannot perform a shuffle is lastBase is nil")
	}
	if clientPublicKeys == nil {
		return nil, errors.New("Cannot perform a shuffle is clientPublicKeys is nil")
	}
	if len(clientPublicKeys) == 0 {
		return nil, errors.New("Cannot perform a shuffle is len(clientPublicKeys) is 0")
	}

	shuffledKeys, newBase, secretCoeff, proof, err := crypto.NeffShuffle(clientPublicKeys, lastBase, config.CryptoSuite, shuffleKeyPositions)
	if err != nil {
		return nil, err
	}

	t.SecretCoeff = secretCoeff

	//store the result
	t.NewBase = newBase
	t.EphemeralKeys = shuffledKeys
	t.Proof = proof

	//send the answer
	msg := &net.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS{
		NewBase:            newBase,
		NewEphPks:          shuffledKeys,
		Proof:              proof,
		VerifiableDCNetKey: vkey}

	return msg, nil
}

/**
 * We received a transcript of the whole shuffle from the relay. Check that we are included, and sign
 */
func (t *NeffShuffleTrustee) ReceivedTranscriptFromRelay(bases []abstract.Point, shuffledPublicKeys [][]abstract.Point, proofs [][]byte) (interface{}, error) {

	if t.NewBase == nil {
		return nil, errors.New("Cannot verify the shuffle, we didn't store the base")
	}
	if t.EphemeralKeys == nil || len(t.EphemeralKeys) == 0 {
		return nil, errors.New("Cannot verify the shuffle, we didn't store the ephemeral keys")
	}
	if t.Proof == nil {
		return nil, errors.New("Cannot verify the shuffle, we didn't store the proof")
	}
	if len(bases) != len(shuffledPublicKeys) || len(bases) != len(proofs) {
		return nil, errors.New("Size not matching, bases is " + strconv.Itoa(len(bases)) + ", shuffledPublicKeys_s is " + strconv.Itoa(len(shuffledPublicKeys)) + ", proof_s is " + strconv.Itoa(len(proofs)) + ".")
	}

	nTrustees := len(bases)
	nClients := len(shuffledPublicKeys[0])

	//Todo : verify each individual permutations. No verification is done yet
	var err error
	for j := 0; j < nTrustees; j++ {

		verify := true
		if j > 0 {
			X := shuffledPublicKeys[j-1]
			Y := shuffledPublicKeys[j-1]
			Xbar := shuffledPublicKeys[j]
			Ybar := shuffledPublicKeys[j]
			if len(X) > 1 {
				verifier := shuffle.Verifier(config.CryptoSuite, nil, X[0], X, Y, Xbar, Ybar)
				err = crypto_proof.HashVerify(config.CryptoSuite, "PairShuffle", verifier, proofs[j])
			}
			if err != nil {
				verify = false
			}
		}
		verify = true // TODO: This shuffle needs to be fixed

		if !verify {
			return nil, errors.New("Could not verify the " + strconv.Itoa(j) + "th neff shuffle, error is " + err.Error())
		}
	}

	//we verify that our shuffle was included
	ownPermutationFound := false
	for j := 0; j < nTrustees; j++ {
		if bases[j].Equal(t.NewBase) && bytes.Equal(t.Proof, proofs[j]) {
			allKeyEqual := true
			for k := 0; k < nClients; k++ {
				if !t.EphemeralKeys[k].Equal(shuffledPublicKeys[j][k]) {
					allKeyEqual = false
					break
				}
			}
			if allKeyEqual {
				ownPermutationFound = true
			}
		}
	}

	if !ownPermutationFound {
		return nil, errors.New("Could not locate our own permutation in the transcript...")
	}

	//prepare the transcript signature. Since it is OK, we're gonna sign only the latest permutation
	var blob []byte
	lastPerm := nTrustees - 1

	lastSharesByte, err := bases[lastPerm].MarshalBinary()
	if err != nil {
		return nil, errors.New("Can't marshall the last shares...")
	}
	blob = append(blob, lastSharesByte...)

	for j := 0; j < nClients; j++ {
		pkBytes, err := shuffledPublicKeys[lastPerm][j].MarshalBinary()
		if err != nil {
			return nil, errors.New("Can't marshall shuffled public key" + strconv.Itoa(j))
		}
		blob = append(blob, pkBytes...)
	}

	//sign this blob
	signature := crypto.SchnorrSign(config.CryptoSuite, random.Stream, blob, t.PrivateKey)

	//send the answer
	msg := &net.TRU_REL_SHUFFLE_SIG{
		TrusteeID: t.TrusteeID,
		Sig:       signature}

	return msg, nil
}
