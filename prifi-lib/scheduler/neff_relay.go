package scheduler

import (
	"errors"
	"github.com/lbarman/prifi/prifi-lib/config"
	"github.com/lbarman/prifi/prifi-lib/crypto"
	"github.com/lbarman/prifi/prifi-lib/net"
	"gopkg.in/dedis/crypto.v0/abstract"
	"strconv"
)

/**
 * The view of the relay for the Neff Shuffle
 */
type NeffShuffleRelay struct {
	NTrustees   int
	InitialBase abstract.Point

	//this is the transcript, i.e. we keep everything
	Bases              []abstract.Point
	ShuffledPublicKeys []net.PublicKeyArray
	Proofs             []net.ByteArray
	Signatures         []net.ByteArray
	SignatureCount     int

	//this is the mutable state, i.e. it change with every shuffling from trustee
	PublicKeyBeingShuffled  []abstract.Point
	LastBase                abstract.Point
	currentTrusteeShuffling int
	CannotAddNewKeys        bool
}

/**
 * Prepares the relay-view to hold the answers from the trustees, etc
 */
func (r *NeffShuffleRelay) Init(nTrustees int) error {

	if nTrustees < 1 {
		return errors.New("Cannot prepare a shuffle for less than one trustee (" + strconv.Itoa(nTrustees) + ")")
	}

	// prepare the empty transcript
	r.Bases = make([]abstract.Point, nTrustees)
	r.ShuffledPublicKeys = make([]net.PublicKeyArray, nTrustees)
	r.Proofs = make([]net.ByteArray, nTrustees)
	r.Signatures = make([]net.ByteArray, nTrustees)
	r.currentTrusteeShuffling = 0
	r.NTrustees = nTrustees

	//the relay picks c0
	r.InitialBase = config.CryptoSuite.Point().Base()

	//the share of products is c0 (will become c1*c0, c2*c1*c0, ...)
	r.LastBase = r.InitialBase

	return nil
}

/**
 * Adds a (ephemeral if possible) public key to the shuffle pool.
 */
func (r *NeffShuffleRelay) AddClient(publicKey abstract.Point) error {

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
func (r *NeffShuffleRelay) SendToNextTrustee() (interface{}, int, error) {

	if r.PublicKeyBeingShuffled == nil {
		return nil, -1, errors.New("RelayView's public keys is nil")
	}
	if len(r.PublicKeyBeingShuffled) == 0 {
		return nil, -1, errors.New("RelayView's public key array is empty")
	}
	r.CannotAddNewKeys = true

	// send to the next trustee
	msg := &net.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE{
		Pks:    nil,
		EphPks: r.PublicKeyBeingShuffled,
		Base:   r.LastBase}

	return msg, r.currentTrusteeShuffling, nil
}

/**
 * Simply holds the new shares and public keys, so we can use this in the next call to SendToNextTrustee()
 */
func (r *NeffShuffleRelay) ReceivedShuffleFromTrustee(newBase abstract.Point, newPublicKeys []abstract.Point, proof []byte) (bool, error) {

	if newBase == nil {
		return false, errors.New("Received a shuffle from the trustee, but newShares is nil")
	}
	if newPublicKeys == nil {
		return false, errors.New("Received a shuffle from the trustee, but newPublicKeys is nil")
	}
	if proof == nil {
		return false, errors.New("Received a shuffle from the trustee, but proof is nil")
	}
	if len(newPublicKeys) == 0 {
		return false, errors.New("Received a shuffle from the trustee, but len(newPublicKeys) is 0")
	}

	// store this shuffle's result in our transcript
	j := r.currentTrusteeShuffling
	r.ShuffledPublicKeys[j] = net.PublicKeyArray{Keys: newPublicKeys}
	r.Proofs[j] = net.ByteArray{Bytes: proof}
	r.Bases[j] = newBase

	//will be used by next trustee
	r.PublicKeyBeingShuffled = newPublicKeys
	r.LastBase = newBase

	r.currentTrusteeShuffling = j + 1

	return r.currentTrusteeShuffling == r.NTrustees, nil
}

/**
 * Packages the Shares, ShuffledPublicKeys and Proofs
 */
func (r *NeffShuffleRelay) SendTranscript() (interface{}, error) {

	if len(r.Bases) != len(r.ShuffledPublicKeys) || len(r.Bases) != len(r.Proofs) {
		return nil, errors.New("Size not matching, Bases is " + strconv.Itoa(len(r.Bases)) + ", ShuffledPublicKeys is " + strconv.Itoa(len(r.ShuffledPublicKeys)) + ", Proofs is " + strconv.Itoa(len(r.Proofs)) + ".")
	}
	if len(r.ShuffledPublicKeys) == 0 {
		return nil, errors.New("Cannot send a transcript of empty array of public keys")
	}

	msg := &net.REL_TRU_TELL_TRANSCRIPT{
		Bases:  r.Bases,
		EphPks: r.ShuffledPublicKeys,
		Proofs: r.Proofs}
	return msg, nil
}

/**
 * Simply stores the signatures
 */
func (r *NeffShuffleRelay) ReceivedSignatureFromTrustee(trusteeID int, signature []byte) (bool, error) {

	if signature == nil {
		return false, errors.New("Received a signature from a trustee, but sig is nil")
	}
	if trusteeID < 0 {
		return false, errors.New("Received a signature from a trustee, trusteeId is invalid (" + strconv.Itoa(trusteeID) + ")")
	}

	// store this shuffle's signature in our transcript
	r.Signatures[trusteeID] = net.ByteArray{Bytes: signature}
	r.SignatureCount++

	return r.SignatureCount == r.NTrustees, nil
}

/**
 * Packages the shares, the shuffledPublicKeys in a byte array, and test the signatures from the trustees.
 * Fails if any one signature is invalid
 */
func multiSigVerify(trusteesPublicKeys []abstract.Point, lastBase abstract.Point, shuffledPublicKeys []abstract.Point, signatures [][]byte) (bool, error) {

	nTrustees := len(trusteesPublicKeys)

	if nTrustees == 0 {
		return false, errors.New("no point in calling multiSigVerify is we have 0 public keys from trustees")
	}
	if lastBase == nil {
		return false, errors.New("lastBase is nil")
	}
	if shuffledPublicKeys == nil {
		return false, errors.New("shuffledPublicKeys is nil")
	}
	if signatures == nil {
		return false, errors.New("signatures is nil")
	}
	if len(trusteesPublicKeys) != len(signatures) {
		return false, errors.New("len(trusteesPublicKeys)=" + strconv.Itoa(len(trusteesPublicKeys)) + " not matching len(signatures)=" + strconv.Itoa(len(signatures)))
	}

	//we reproduce the signed blob
	G_bytes, err := lastBase.MarshalBinary()
	if err != nil {
		return false, errors.New("Can't marshall the last base...")
	}
	var M []byte
	M = append(M, G_bytes...)
	for k := 0; k < len(shuffledPublicKeys); k++ {
		pkBytes, err := shuffledPublicKeys[k].MarshalBinary()

		if err != nil {
			return false, errors.New("Can't marshall the last signature...")
		}

		M = append(M, pkBytes...)
	}

	//we test the signatures
	for j := 0; j < nTrustees; j++ {
		err := crypto.SchnorrVerify(config.CryptoSuite, M, trusteesPublicKeys[j], signatures[j])

		if err != nil {
			return false, errors.New("Can't verify sig n°" + strconv.Itoa(j) + "; " + err.Error())
		}
	}

	return true, nil
}

/**
 * Verify all signatures, and sends to client the last shuffle (and the signatures)
 */
func (r *NeffShuffleRelay) VerifySigsAndSendToClients(trusteesPublicKeys []abstract.Point) (interface{}, error) {

	if trusteesPublicKeys == nil {
		return nil, errors.New("shuffledPublicKeys is nil")
	}

	if len(trusteesPublicKeys) != len(r.Bases) || len(trusteesPublicKeys) != len(r.ShuffledPublicKeys) || len(trusteesPublicKeys) != len(r.Signatures) {
		return nil, errors.New("Some size mismatch, len(trusteesPublicKeys)=" + strconv.Itoa(len(trusteesPublicKeys)) + ", len(r.Bases)=" + strconv.Itoa(len(r.Bases)) + ", len(r.ShuffledPublicKeys)=" + strconv.Itoa(len(r.ShuffledPublicKeys)) + ", len(r.Signatures)=" + strconv.Itoa(len(r.Signatures)) + "")
	}

	//verify the signature
	lastPermutationIndex := r.NTrustees - 1
	lastBase := r.Bases[lastPermutationIndex]
	ephPubKeys := r.ShuffledPublicKeys[lastPermutationIndex]
	signatures := r.Signatures

	sigArray := make([][]byte, 0)
	for k := range r.Signatures {
		sigArray = append(sigArray, r.Signatures[k].Bytes)
	}

	success, err := multiSigVerify(trusteesPublicKeys, lastBase, ephPubKeys.Keys, sigArray)
	if success != true {
		return nil, err
	}

	msg := &net.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG{
		Base:         lastBase,
		EphPks:       ephPubKeys.Keys,
		TrusteesSigs: signatures}
	return msg, nil
}
