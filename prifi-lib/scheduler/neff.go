package scheduler

/*

The interface should be :

INPUT : list of client's public keys

OUTPUT : list of slots

*/

import (
	"github.com/dedis/crypto/abstract"
	"github.com/dedis/crypto/random"
	"github.com/lbarman/prifi/prifi-lib"
	"github.com/lbarman/prifi/prifi-lib/config"
	"github.com/dedis/crypto/shuffle"
	crypto_proof "github.com/dedis/crypto/proof"
	"github.com/lbarman/prifi/prifi-lib/crypto"
	"math/rand"
	"strconv"
	"errors"
	"bytes"
	"fmt"
)

type neffShuffleRelayView struct {
	NTrustees               int
	Pks                     []abstract.Point
	G_s                     []abstract.Scalar
	ShuffledPks_s           [][]abstract.Point
	Proof_s                 [][]byte
	Signature_s             [][]byte
	SignatureCount          int
	LastSecret              abstract.Scalar
	currentTrusteeShuffling int
}

type neffShuffleTrusteeView struct {
	TrusteeId int
	Base abstract.Scalar
	EphemeralKeys []abstract.Point
	Proof []byte
	PrivateKey abstract.Scalar
	PublicKey abstract.Point
	SecretCoeff abstract.Scalar
}

type neffShuffleScheduler struct {
	RelayView *neffShuffleRelayView
	TrusteeView *neffShuffleTrusteeView
}

func (n *neffShuffleScheduler) init(){
	n.RelayView = new(neffShuffleRelayView)
	n.TrusteeView = new(neffShuffleTrusteeView)
}

func (r *neffShuffleRelayView) AddClient(pk abstract.Point) error {

	if pk == nil {
		return errors.New("Cannot shuffle a nil key, refusing to add public key.")
	}

	if r.Pks == nil {
		r.Pks = make([]abstract.Point, 0)
	}
	r.Pks = append(r.Pks, pk)

	return nil
}

func (r *neffShuffleRelayView) init(nTrustees int) error {

	if nTrustees < 1 {
		return errors.New("Cannot prepare a shuffle for less than one trustee ("+strconv.Itoa(nTrustees)+")")
	}

	// prepare the empty transcript
	r.G_s = make([]abstract.Scalar, nTrustees)
	r.ShuffledPks_s = make([][]abstract.Point, nTrustees)
	r.Proof_s = make([][]byte, nTrustees)
	r.Signature_s = make([][]byte, nTrustees)
	r.currentTrusteeShuffling = 0
	r.NTrustees = nTrustees
	r.LastSecret = config.CryptoSuite.Scalar().One()

	return nil
}

func (t *neffShuffleTrusteeView) init(trusteeId int, private abstract.Scalar, public abstract.Point) error {
	if trusteeId < 0{
		return errors.New("Cannot shuffle without a valid id (>= 0)")
	}
	if private == nil {
		return errors.New("Cannot shuffle without a private key.")
	}
	if public == nil {
		return errors.New("Cannot shuffle without a public key.")
	}
	t.TrusteeId = trusteeId
	t.PrivateKey = private
	t.PublicKey = public
	return nil
}

func (r *neffShuffleRelayView) SendToNextTrustee() (error, interface{}) {

	if r.Pks == nil {
		return errors.New("RelayView's public keys is nil"), nil
	}
	if len(r.Pks) == 0 {
		return errors.New("RelayView's public key array is empty"), nil
	}

	// send to the next trustee
	toSend := &prifi_lib.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE{
		Pks: r.Pks,
		EphPks: r.Pks,
		Base: r.LastSecret}

	return nil, toSend
}

func (t *neffShuffleTrusteeView) ReceivedShuffleFromRelay(base abstract.Scalar, clientPublicKeys []abstract.Point, shuffleKeyPositions bool) (error, interface{}) {

	secretCoeff := config.CryptoSuite.Scalar().Pick(random.Stream)
	t.SecretCoeff = secretCoeff
	base2 := config.CryptoSuite.Scalar().Mul(base, secretCoeff)

	ephPublicKeys2 := clientPublicKeys

	fmt.Println("------------")
	fmt.Println("base is", base)
	fmt.Println("secretCoeff is", secretCoeff)
	fmt.Println("base2 is", base2)
	//transform the public keys with the secret coeff

	for i := 0; i < len(clientPublicKeys); i++ {
		oldKey := clientPublicKeys[i]
		ephPublicKeys2[i] = config.CryptoSuite.Point().Mul(oldKey, secretCoeff)
		//fmt.Println(oldKey, " changed to ", ephPublicKeys2[i])
	}

	//shuffle the array
	if shuffleKeyPositions {
		ephPublicKeys3 := make([]abstract.Point, len(ephPublicKeys2))
		perm := rand.Perm(len(ephPublicKeys2))
		for i, v := range perm {
			ephPublicKeys3[v] = ephPublicKeys2[i]
			//fmt.Println(i, " now goes to ", v)
		}
		ephPublicKeys2 = ephPublicKeys3
	}

	proof := make([]byte, 50) // TODO : the proof should be done

	//send the answer
	toSend := &prifi_lib.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS{base2, ephPublicKeys2, proof}

	//store the result
	t.Base = base2
	t.EphemeralKeys = ephPublicKeys2
	t.Proof = proof

	return nil, toSend
}

func (r *neffShuffleRelayView) ReceivedShuffleFromTrustee(newBase abstract.Scalar, newPublicKeys []abstract.Point, proof []byte) (error, bool) {

	// store this shuffle's result in our transcript
	j := r.currentTrusteeShuffling
	r.ShuffledPks_s[j] = newPublicKeys
	r.Proof_s[j] = proof
	r.G_s[j] = newBase

	//will be used by next trustee
	r.Pks = newPublicKeys
	r.LastSecret = newBase

	r.currentTrusteeShuffling = j + 1

	return nil, r.currentTrusteeShuffling == r.NTrustees
}


func (r *neffShuffleRelayView) SendTranscript() (error, interface{}) {

	if len(r.G_s) != len(r.ShuffledPks_s) || len(r.G_s) != len(r.Proof_s) {
		return errors.New("Size not matching, G_s is "+strconv.Itoa(len(r.G_s))+", ShuffledPks_s is "+strconv.Itoa(len(r.ShuffledPks_s))+", Proof_s is "+strconv.Itoa(len(r.Proof_s))+"."), nil
	}
	toSend := &prifi_lib.REL_TRU_TELL_TRANSCRIPT{r.G_s, r.ShuffledPks_s, r.Proof_s}
	return nil, toSend
}


func (t *neffShuffleTrusteeView) ReceivedTranscriptFromRelay(G_s []abstract.Scalar, shuffledPublicKeys_s [][]abstract.Point, proof_s [][]byte) (error, interface{}) {

	if len(G_s) != len(shuffledPublicKeys_s) || len(G_s) != len(proof_s) {
		return errors.New("Size not matching, G_s is "+strconv.Itoa(len(G_s))+", shuffledPublicKeys_s is "+strconv.Itoa(len(shuffledPublicKeys_s))+", proof_s is "+strconv.Itoa(len(proof_s))+"."), nil
	}

	if t.Base == nil {
		return errors.New("Cannot verify the shuffle, we didn't store the base"), nil
	}

	if t.EphemeralKeys == nil || len(t.EphemeralKeys) == 0 {
		return errors.New("Cannot verify the shuffle, we didn't store the ephemeral keys"), nil
	}

	if t.Proof == nil {
		return errors.New("Cannot verify the shuffle, we didn't store the proof"), nil
	}

	// PROTOBUF FLATTENS MY 2-DIMENSIONAL ARRAY. THIS IS A PATCH
	nTrustees := len(G_s)
	nClients := len(shuffledPublicKeys_s[0])
	a := shuffledPublicKeys_s
	b := make([][]abstract.Point, nTrustees)
	if len(a) > nTrustees {
		for i := 0; i < nTrustees; i++ {
			b[i] = make([]abstract.Point, nClients)
			for j := 0; j < nClients; j++ {
				v := a[i*nTrustees+j][0]
				b[i][j] = v
			}
		}
		shuffledPublicKeys_s = b
	} else {
		//log.Print("Probably the Protobuf lib has been patched ! you might remove this code.")
	}
	// END OF PATCH


	//Todo : verify each individual permutations
	var err error
	for j := 0; j < nTrustees; j++ {

		verify := true
		if j > 0 {
			X := shuffledPublicKeys_s[j-1]
			Y := shuffledPublicKeys_s[j-1]
			Xbar := shuffledPublicKeys_s[j]
			Ybar := shuffledPublicKeys_s[j]
			if len(X) > 1 {
				verifier := shuffle.Verifier(config.CryptoSuite, nil, X[0], X, Y, Xbar, Ybar)
				err = crypto_proof.HashVerify(config.CryptoSuite, "PairShuffle", verifier, proof_s[j])
			}
			if err != nil {
				verify = false
			}
		}
		verify = true // TODO: This shuffle needs to be fixed

		if !verify {
			return errors.New( "Could not verify the " + strconv.Itoa(j) + "th neff shuffle, error is " + err.Error()), nil
		}
	}

	//we verify that our shuffle was included
	ownPermutationFound := false
	for j := 0; j < nTrustees; j++ {

		if G_s[j].Equal(t.Base) && bytes.Equal(t.Proof, proof_s[j]) {

			allKeyEqual := true
			for k := 0; k < nClients; k++ {
				if !t.EphemeralKeys[k].Equal(shuffledPublicKeys_s[j][k]) {
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
		return errors.New("Could not locate our own permutation in the transcript..."), nil
	}

	//prepare the transcript signature. Since it is OK, we're gonna sign the latest permutation
	var M []byte
	G_s_j_bytes, err := G_s[nTrustees-1].MarshalBinary()

	if err != nil {
		return errors.New("Can't marshall the last signature..."), nil
	}
	M = append(M, G_s_j_bytes...)


	for j := 0; j < nClients; j++ {
		pkBytes, err := shuffledPublicKeys_s[nTrustees-1][j].MarshalBinary()
		if err != nil {
			return errors.New("Can't marshall public key" + strconv.Itoa(j)), nil
		}
		M = append(M, pkBytes...)
	}

	sig := crypto.SchnorrSign(config.CryptoSuite, random.Stream, M, t.PrivateKey)


	//send the answer
	toSend := &prifi_lib.TRU_REL_SHUFFLE_SIG{t.TrusteeId, sig}

	return nil, toSend
}

func (r *neffShuffleRelayView) ReceivedSignatureFromTrustee (trusteeId int, sig []byte) (error, bool){

	// store this shuffle's signature in our transcript
	r.Signature_s[trusteeId] = sig
	r.SignatureCount++

	return nil, r.SignatureCount == r.NTrustees
}

func multiSigVerify(trusteesPublicKeys []abstract.Point, G abstract.Scalar, ephPubKeys []abstract.Point, signatures [][]byte) (error, bool){

	nTrustees := len(trusteesPublicKeys)

	G_bytes, err := G.MarshalBinary()
	if err != nil {
		return errors.New("Can't marshall the last signature..."), false
	}
	var M []byte
	M = append(M, G_bytes...)
	for k := 0; k < len(ephPubKeys); k++ {
		pkBytes, err := ephPubKeys[k].MarshalBinary()

		if err != nil {
			return errors.New("Can't marshall the last signature..."), false
		}

		M = append(M, pkBytes...)
	}

	for j := 0; j < nTrustees; j++ {
		err := crypto.SchnorrVerify(config.CryptoSuite, M, trusteesPublicKeys[j], signatures[j])

		if err != nil {
			return errors.New("Can't verify sig n°"+strconv.Itoa(j)+"; "+err.Error()), false
		}
	}

	return nil, true
}

func (r *neffShuffleRelayView) VerifySigsAndSendToClients(trusteesPublicKeys []abstract.Point) (error, interface{}){

	//verify the signature
	lastPermutationIndex := r.NTrustees - 1
	G := r.G_s[lastPermutationIndex]
	ephPubKeys := r.ShuffledPks_s[lastPermutationIndex]
	signatures := r.Signature_s

	err, success := multiSigVerify(trusteesPublicKeys, G, ephPubKeys, signatures)
	if success != true {
		return err, nil
	}

	toSend := &prifi_lib.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG{G, ephPubKeys, signatures}
	return nil, toSend
}

func (n *neffShuffleScheduler) ClientVerifySigAndRecognizeSlot(privateKey abstract.Scalar, trusteesPublicKeys []abstract.Point, lastBase abstract.Scalar, ephPubKeys []abstract.Point, signatures [][]byte) (error, int) {

	fmt.Println("#########################")
	fmt.Println("My private key", privateKey)
	fmt.Println("secret", lastBase) //c1 * c2 * ...

	err, success := multiSigVerify(trusteesPublicKeys, lastBase, ephPubKeys, signatures)
	if success != true {
		return err, -1
	}

	base := config.CryptoSuite.Point().Base()
	newBaseFromTrusteesG := config.CryptoSuite.Point().Mul(base, lastBase)
	ephPubInNewBase := config.CryptoSuite.Point().Mul(newBaseFromTrusteesG, privateKey)


	fmt.Println("base", base) // G
	fmt.Println("newBaseFromTrusteesG", newBaseFromTrusteesG) // G * c1 * c2
	fmt.Println("ephPubInNewBase", ephPubInNewBase) // G * c1 * c2 * p

	mySlot := -1

	for j := 0; j < len(ephPubKeys); j++ {
		fmt.Println("... comparing with", ephPubKeys[j]) // p * G * c1 * c2
		if ephPubKeys[j].Equal(ephPubInNewBase) {
			mySlot = j
		}
	}

	if mySlot == -1 {
		return errors.New("Could not locate my slot"), -1
	}
	return nil, mySlot
}