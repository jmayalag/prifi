package scheduler

/*

The interface should be :

INPUT : list of client's public keys

OUTPUT : list of slots

*/

import (
	"github.com/dedis/cothority/log"
	"github.com/dedis/crypto/abstract"
	"github.com/dedis/crypto/random"
	"github.com/lbarman/prifi/prifi-lib"
	"github.com/lbarman/prifi/prifi-lib/config"
	"math/rand"
	"github.com/btcsuite/golangcrypto/openpgp/errors"
	"strconv"
)

type neffShuffleRelayView struct {
	NTrustees               int
	Pks                     []abstract.Point
	G_s                     []abstract.Scalar
	ShuffledPks_s           [][]abstract.Point
	Proof_s                 [][]byte
	Signature_s             [][]byte
	lastG 			abstract.Scalar
	currentTrusteeShuffling int
}

type neffShuffleScheduler struct {
	RelayView *neffShuffleRelayView
}

func (n *neffShuffleScheduler) AddClient(pk abstract.Point) error {

	if n.RelayView == nil {
		n.RelayView = new(neffShuffleRelayView)
	}
	if pk == nil {
		return errors.New("Cannot shuffle a nil key, refusing to add public key.")
	}

	if n.RelayView.Pks == nil {
		n.RelayView.Pks = make([]abstract.Point, 0)
	}
	n.RelayView.Pks = append(n.RelayView.Pks, pk)

	return nil
}

func (n *neffShuffleScheduler) PrepareNeffShuffle(nTrustees int) error {

	if nTrustees < 1 {
		return errors.New("Cannot prepare a shuffle for less than one trustee ("+strconv.Itoa(nTrustees)+")"), nil
	}

	// prepare the empty transcript
	n.RelayView.G_s = make([]abstract.Scalar, nTrustees)
	n.RelayView.ShuffledPks_s = make([][]abstract.Point, nTrustees)
	n.RelayView.Proof_s = make([][]byte, nTrustees)
	n.RelayView.Signature_s = make([][]byte, nTrustees)
	n.RelayView.currentTrusteeShuffling = 0
	n.RelayView.NTrustees = nTrustees
	n.RelayView.lastG = config.CryptoSuite.Scalar().One(

	return nil
}

func (n *neffShuffleScheduler) SendToTrustee() (error, interface{}) {

	if n.RelayView == nil {
		return errors.New("RelayView is nil"), nil
	}
	if n.RelayView.Pks == nil {
		return errors.New("RelayView's public keys is nil"), nil
	}
	if len(n.RelayView.Pks) == 0 {
		return errors.New("RelayView's public key array is empty"), nil
	}

	// send to the 1st trustee
	toSend := &prifi_lib.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE{
		Pks: n.RelayView.Pks,
		EphPks: n.RelayView.Pks,
		Base: n.RelayView.lastG}

	return toSend
}

func (n *neffShuffleScheduler) ReceivedShuffleFromRelay(base abstract.Scalar, clientPublicKeys []abstract.Point) (error, interface{}) {

	if n.RelayView == nil {
		return errors.New("RelayView is nil"), nil
	}

	secretCoeff := config.CryptoSuite.Scalar().Pick(random.Stream)
	base2 := config.CryptoSuite.Scalar().Mul(base, secretCoeff)

	ephPublicKeys2 := clientPublicKeys

	//transform the public keys with the secret coeff
	for i := 0; i < len(clientPublicKeys); i++ {
		ephPublicKeys2[i] = config.CryptoSuite.Point().Mul(clientPublicKeys[i], base2)
	}

	//shuffle the array
	ephPublicKeys3 := make([]abstract.Point, len(ephPublicKeys2))
	perm := rand.Perm(len(ephPublicKeys2))
	for i, v := range perm {
		ephPublicKeys3[v] = ephPublicKeys2[i]
	}
	ephPublicKeys2 = ephPublicKeys3

	proof := make([]byte, 50) // TODO : the proof should be done

	//send the answer
	toSend := &prifi_lib.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS{base2, ephPublicKeys2, proof}

	return toSend
}

func (n *neffShuffleScheduler) ReceivedShuffleFromTrustee(newBase abstract.Scalar, newPublicKeys []abstract.Point, proof []byte) interface{} {

	// store this shuffle's result in our transcript
	j := n.RelayView.currentTrusteeShuffling
	n.RelayView.G_s[j] = newBase
	n.RelayView.ShuffledPks_s[j] = newPublicKeys
	n.RelayView.Proof_s[j] = proof

	n.RelayView.currentTrusteeShuffling = j + 1

	// if we're still waiting on some trustees, send them the new shuffle
	if n.RelayView.currentTrusteeShuffling != n.RelayView.NTrustees {

		// send to the i-th trustee
		toSend := &prifi_lib.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE{newPublicKeys, newPublicKeys, newBase}
		return toSend

	} else {
		toSend := &prifi_lib.REL_TRU_TELL_TRANSCRIPT{n.RelayView.G_s, n.RelayView.ShuffledPks_s, n.RelayView.Proof_s}
		return toSend
	}
}
