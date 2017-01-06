package scheduler

import (
	"errors"
	"github.com/dedis/crypto/abstract"
	"github.com/dedis/crypto/random"
	"github.com/lbarman/prifi/prifi-lib"
	"github.com/lbarman/prifi/prifi-lib/config"
	"github.com/lbarman/prifi/prifi-lib/crypto"
	"strconv"
)

type neffShuffleRelayView struct {
	NTrustees    int
	InitialCoeff abstract.Scalar

	//this is the transcript, i.e. we keep everything
	Shares             []abstract.Scalar
	ShuffledPublicKeys [][]abstract.Point
	Proofs             [][]byte
	Signatures         [][]byte
	SignatureCount     int

	//this is the mutable state, i.e. it change with every shuffling from trustee
	PublicKeyBeingShuffled  []abstract.Point
	CurrentShares           abstract.Scalar
	currentTrusteeShuffling int
	CannotAddNewKeys        bool
}

/**
 * Prepares the relay-view to hold the answers from the trustees, etc
 */
func (r *neffShuffleRelayView) init(nTrustees int) error {

	if nTrustees < 1 {
		return errors.New("Cannot prepare a shuffle for less than one trustee (" + strconv.Itoa(nTrustees) + ")")
	}

	// prepare the empty transcript
	r.Shares = make([]abstract.Scalar, nTrustees)
	r.ShuffledPublicKeys = make([][]abstract.Point, nTrustees)
	r.Proofs = make([][]byte, nTrustees)
	r.Signatures = make([][]byte, nTrustees)
	r.currentTrusteeShuffling = 0
	r.NTrustees = nTrustees

	//the relay picks c0
	r.InitialCoeff = config.CryptoSuite.Scalar().Pick(random.Stream)

	//the share of products is c0 (will become c1*c0, c2*c1*c0, ...)
	r.CurrentShares = r.InitialCoeff

	return nil
}

/**
 * Adds a (ephemeral if possible) public key to the shuffle pool.
 */
func (r *neffShuffleRelayView) AddClient(publicKey abstract.Point) error {

	if publicKey == nil {
		return errors.New("Cannot shuffle a nil key, refusing to add public key.")
	}
	if r.CannotAddNewKeys {
		return errors.New("Cannot add key, the shuffling already started.")
	}

	if r.PublicKeyBeingShuffled == nil {
		r.PublicKeyBeingShuffled = make([]abstract.Point, 0)
	}
	r.PublicKeyBeingShuffled = append(r.PublicKeyBeingShuffled, publicKey)

	return nil
}

/**
 * Packs a message for the next trustee. Contains the current state of the shuffle, i.e. PublicKeyBeingShuffled + LastShareProduct
 */
func (r *neffShuffleRelayView) SendToNextTrustee() (error, interface{}) {

	if r.PublicKeyBeingShuffled == nil {
		return errors.New("RelayView's public keys is nil"), nil
	}
	if len(r.PublicKeyBeingShuffled) == 0 {
		return errors.New("RelayView's public key array is empty"), nil
	}
	r.CannotAddNewKeys = true

	// send to the next trustee
	msg := &prifi_lib.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE{
		Pks:    r.PublicKeyBeingShuffled,
		EphPks: r.PublicKeyBeingShuffled,
		Base:   r.CurrentShares}

	return nil, msg
}

/**
 * Simply holds the new shares and public keys, so we can use this in the next call to SendToNextTrustee()
 */
func (r *neffShuffleRelayView) ReceivedShuffleFromTrustee(newShares abstract.Scalar, newPublicKeys []abstract.Point, proof []byte) (error, bool) {

	if newShares == nil {
		return errors.New("Received a shuffle from the trustee, but newShares is nil"), false
	}
	if newPublicKeys == nil {
		return errors.New("Received a shuffle from the trustee, but newPublicKeys is nil"), false
	}
	if proof == nil {
		return errors.New("Received a shuffle from the trustee, but proof is nil"), false
	}
	if len(newPublicKeys) == 0 {
		return errors.New("Received a shuffle from the trustee, but len(newPublicKeys) is 0"), false
	}

	// store this shuffle's result in our transcript
	j := r.currentTrusteeShuffling
	r.ShuffledPublicKeys[j] = newPublicKeys
	r.Proofs[j] = proof
	r.Shares[j] = newShares

	//will be used by next trustee
	r.PublicKeyBeingShuffled = newPublicKeys
	r.CurrentShares = newShares

	r.currentTrusteeShuffling = j + 1

	return nil, r.currentTrusteeShuffling == r.NTrustees
}

/**
 * Packages the Shares, ShuffledPublicKeys and Proofs
 */
func (r *neffShuffleRelayView) SendTranscript() (error, interface{}) {

	if len(r.Shares) != len(r.ShuffledPublicKeys) || len(r.Shares) != len(r.Proofs) {
		return errors.New("Size not matching, G_s is " + strconv.Itoa(len(r.Shares)) + ", ShuffledPks_s is " + strconv.Itoa(len(r.ShuffledPublicKeys)) + ", Proof_s is " + strconv.Itoa(len(r.Proofs)) + "."), nil
	}
	if len(r.ShuffledPublicKeys) == 0 {
		return errors.New("Cannot send a transcript of empty array of public keys"), nil
	}

	msg := &prifi_lib.REL_TRU_TELL_TRANSCRIPT{
		Gs:     r.Shares,
		EphPks: r.ShuffledPublicKeys,
		Proofs: r.Proofs}
	return nil, msg
}

/**
 * Simply stores the signatures
 */
func (r *neffShuffleRelayView) ReceivedSignatureFromTrustee(trusteeId int, signature []byte) (error, bool) {

	if signature == nil {
		return errors.New("Received a signature from a trustee, but sig is nil"), false
	}
	if trusteeId < 0 {
		return errors.New("Received a signature from a trustee, trusteeId is invalid (" + strconv.Itoa(trusteeId) + ")"), false
	}

	// store this shuffle's signature in our transcript
	r.Signatures[trusteeId] = signature
	r.SignatureCount++

	return nil, r.SignatureCount == r.NTrustees
}

/**
 * Packages the shares, the shuffledPublicKeys in a byte array, and test the signatures from the trustees.
 * Fails if any one signature is invalid
 */
func multiSigVerify(trusteesPublicKeys []abstract.Point, shares abstract.Scalar, shuffledPublicKeys []abstract.Point, signatures [][]byte) (error, bool) {

	nTrustees := len(trusteesPublicKeys)

	if nTrustees == 0 {
		return errors.New("no point in calling multiSigVerify is we have 0 public keys from trustees"), false
	}
	if shares == nil {
		return errors.New("shares is nil"), false
	}
	if shuffledPublicKeys == nil {
		return errors.New("shuffledPublicKeys is nil"), false
	}
	if signatures == nil {
		return errors.New("signatures is nil"), false
	}
	if len(trusteesPublicKeys) != len(signatures) {
		return errors.New("len(trusteesPublicKeys)=" + strconv.Itoa(len(trusteesPublicKeys)) + " not matching len(signatures)=" + strconv.Itoa(len(signatures))), false
	}

	//we reproduce the signed blob
	G_bytes, err := shares.MarshalBinary()
	if err != nil {
		return errors.New("Can't marshall the last signature..."), false
	}
	var M []byte
	M = append(M, G_bytes...)
	for k := 0; k < len(shuffledPublicKeys); k++ {
		pkBytes, err := shuffledPublicKeys[k].MarshalBinary()

		if err != nil {
			return errors.New("Can't marshall the last signature..."), false
		}

		M = append(M, pkBytes...)
	}

	//we test the signatures
	for j := 0; j < nTrustees; j++ {
		err := crypto.SchnorrVerify(config.CryptoSuite, M, trusteesPublicKeys[j], signatures[j])

		if err != nil {
			return errors.New("Can't verify sig n°" + strconv.Itoa(j) + "; " + err.Error()), false
		}
	}

	return nil, true
}

/**
 * Verify all signatures, and sends to client the last shuffle (and the signatures)
 */
func (r *neffShuffleRelayView) VerifySigsAndSendToClients(trusteesPublicKeys []abstract.Point) (error, interface{}) {

	if trusteesPublicKeys == nil {
		return errors.New("shuffledPublicKeys is nil"), nil
	}

	if len(trusteesPublicKeys) != len(r.Shares) || len(trusteesPublicKeys) != len(r.ShuffledPublicKeys) || len(trusteesPublicKeys) != len(r.Signatures) {
		return errors.New("Some size mismatch, len(trusteesPublicKeys)=" + strconv.Itoa(len(trusteesPublicKeys)) + ", len(r.Shares)=" + strconv.Itoa(len(r.Shares)) + ", len(r.ShuffledPublicKeys)=" + strconv.Itoa(len(r.ShuffledPublicKeys)) + ", len(r.Signatures)=" + strconv.Itoa(len(r.Signatures)) + ""), nil
	}

	//verify the signature
	lastPermutationIndex := r.NTrustees - 1
	G := r.Shares[lastPermutationIndex]
	ephPubKeys := r.ShuffledPublicKeys[lastPermutationIndex]
	signatures := r.Signatures

	err, success := multiSigVerify(trusteesPublicKeys, G, ephPubKeys, signatures)
	if success != true {
		return err, nil
	}

	msg := &prifi_lib.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG{
		Base:         G,
		EphPks:       ephPubKeys,
		TrusteesSigs: signatures}
	return nil, msg
}
