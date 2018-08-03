package scheduler

import (
	"fmt"
	"github.com/dedis/prifi/prifi-lib/config"
	"github.com/dedis/prifi/prifi-lib/crypto"
	"github.com/dedis/prifi/prifi-lib/net"
	"gopkg.in/dedis/kyber.v2"
	"strconv"
	"testing"
)

type PrivatePublicPair struct {
	Private kyber.Scalar
	Public  kyber.Point
}

/*
 * Client 1 computes : p1 * B = P1
 * Client 2 computes : p2 * B = P2
 * Relay picks a base B
 * Relay sends P1, P2, B to trustee
 * Trustee 1 pick c1
 * Trustee 1 compute B1 = B * c1
 * Trustee 1 compute P1' = P1 * c1 = p1 * c1 * B
 * Trustee 1 compute P2' = P2 * c1 = p2 * c1 * B
 * Relay collect, then sends P1', P2', B1
 * Trustee 2 pick c2
 * Trustee 2 compute B2 = B1 * c2 = B * c1 * c2
 * Trustee 2 compute P1'' = P1' * c2 = p1 * c1 * c2 * B
 * Trustee 2 compute P2'' = P2' * c2 = p2 * c1 * c2 * B
 * Relay sends P1'', P2'', s2 to clients
 * Client 1 compute p1 * s2 * B = p1 * s0 * c1 * c2 * B = P1''
 * Client 2 compute p2 * s2 * B = p2 * s0 * c1 * c2 * B = P2''
 */
func TestWholeNeffShuffle(t *testing.T) {

	nClientsRange := []int{1, 2, 3, 4, 5, 10}
	nTrusteeRange := []int{1, 2, 3, 5, 10}

	//standard testing. shuffleKeyPos=false to allow testing
	for _, nClients := range nClientsRange {
		for _, nTrustees := range nTrusteeRange {
			//standard testing
			fmt.Println("Testing for", nClients, "clients,", nTrustees, "trustees...")
			NeffShuffleTestHelper(t, nClients, nTrustees, false)
		}
	}
}

func NeffShuffleTestHelper(t *testing.T, nClients int, nTrustees int, shuffleKeyPos bool) []int {
	clients := make([]*PrivatePublicPair, nClients)
	for i := 0; i < nClients; i++ {
		pub, priv := crypto.NewKeyPair()
		clients[i] = new(PrivatePublicPair)
		clients[i].Public = pub
		clients[i].Private = priv
	}

	//create the scheduler
	n := new(NeffShuffle) //this will hold 1 relay, 1 trustee at most. Recreate n for >1 trustee
	n.Init()

	//init the trustees
	trustees := make([]*NeffShuffle, nTrustees)
	for i := 0; i < nTrustees; i++ {
		trustees[i] = new(NeffShuffle)
		trustees[i].Init()
		pub, priv := crypto.NewKeyPair()
		trustees[i].TrusteeView.Init(i, priv, pub)
	}

	//init the relay
	err := n.RelayView.Init(nTrustees)
	if err != nil {
		t.Error(err)
	}
	for i := 0; i < nClients; i++ {
		n.RelayView.AddClient(clients[i].Public)
	}

	isDone := false
	i := 0
	for !isDone {
		if i >= nTrustees {
			t.Error("Should only shuffle" + strconv.Itoa(nTrustees) + ", but we did one more loop")
		}

		//the relay send the shuffle send it to the next trustee
		toSend, _, err := n.RelayView.SendToNextTrustee()
		if err != nil {
			t.Error(err)
		}
		parsed := toSend.(*net.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE)

		//who receives it
		toSend2, err := trustees[i].TrusteeView.ReceivedShuffleFromRelay(parsed.Base, parsed.EphPks, shuffleKeyPos, make([]byte, 1))
		if err != nil {
			t.Error(err)
		}
		parsed2 := toSend2.(*net.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS)

		// TEST: Trustee i compute B[i] = B[i-1] * c[i]
		B_i_minus_1 := n.RelayView.InitialBase
		if i > 0 {
			B_i_minus_1 = n.RelayView.Bases[i-1]
		}
		c_i := trustees[i].TrusteeView.SecretCoeff
		B_i := config.CryptoSuite.Point().Mul(c_i, B_i_minus_1)

		if !parsed2.NewBase.Equal(B_i) {
			t.Error("B[" + strconv.Itoa(i+1) + "] is computed incorrectly")
		}

		//if shuffle the key pos, we cannot test this easily
		if !shuffleKeyPos {
			// TEST: Trustee i compute P1'[i] = P1'[i-1] * c[i] = p1 * c[1] ... c[i] * B

			for clientID := 0; clientID < nClients; clientID++ {
				p1 := clients[clientID].Private
				c_s := config.CryptoSuite.Scalar().One()
				for j := 0; j <= i; j++ {
					c_j := trustees[j].TrusteeView.SecretCoeff
					c_s = config.CryptoSuite.Scalar().Mul(c_s, c_j)
				}
				B := config.CryptoSuite.Point().Base()
				p1_c_s := config.CryptoSuite.Scalar().Mul(p1, c_s)
				LHS := config.CryptoSuite.Point().Mul(p1_c_s, B)

				if !parsed2.NewEphPks[clientID].Equal(LHS) {
					t.Error("P" + strconv.Itoa(clientID) + "'[" + strconv.Itoa(i+1) + "] is computed incorrectly")
				}
			}

			//Specialized test for trustee n°1 (0-th trustee)
			if i == 0 {
				// Trustee 1 compute B1 = B * c1
				B := n.RelayView.InitialBase
				c1 := trustees[0].TrusteeView.SecretCoeff
				if !parsed2.NewBase.Equal(config.CryptoSuite.Point().Mul(c1, B)) {
					t.Error("B1 is computed incorrectly")
				}

				// Trustee 1 compute P1' = P1 * c1 = p1 * c1 * B
				for clientID := 0; clientID < nClients; clientID++ {
					p1prime := config.CryptoSuite.Scalar().Mul(clients[clientID].Private, c1)
					if !parsed2.NewEphPks[clientID].Equal(config.CryptoSuite.Point().Mul(p1prime, B)) {
						t.Error("P" + strconv.Itoa(clientID) + "' is computed incorrectly")
					}
				}
			}

			//Specialized test for trustee n°2 (1st trustee)
			if i == 1 {

				//* Trustee 2 compute B2 = B1 * c2 = B * c1 * c2
				B := n.RelayView.InitialBase
				c1 := trustees[0].TrusteeView.SecretCoeff
				c2 := trustees[1].TrusteeView.SecretCoeff
				c1c2 := config.CryptoSuite.Scalar().Mul(c1, c2)
				if !parsed2.NewBase.Equal(config.CryptoSuite.Point().Mul(c1c2, B)) {
					t.Error("B2 is computed incorrectly (2)")
				}

				//* Trustee 2 compute P1'' = P1' * c2 = p1 * c1 * c2 * B
				for clientID := 0; clientID < nClients; clientID++ {
					p1prime2 := config.CryptoSuite.Scalar().Mul(clients[clientID].Private, c1c2)
					if !parsed2.NewEphPks[clientID].Equal(config.CryptoSuite.Point().Mul(p1prime2, B)) {
						t.Error("P" + strconv.Itoa(clientID) + "'' is computed incorrectly")
					}
				}
			}
		}

		//and answers, the relay receives it
		isDone, err = n.RelayView.ReceivedShuffleFromTrustee(parsed2.NewBase, parsed2.NewEphPks, parsed2.Proof)

		if err != nil {
			t.Error("Shouldn't have an error here," + err.Error())
		}

		i++
	}

	toSend3, err := n.RelayView.SendTranscript()
	if err != nil {
		t.Error(err)
	}
	parsed3 := toSend3.(*net.REL_TRU_TELL_TRANSCRIPT)

	for j := 0; j < nTrustees; j++ {
		toSend4, err := trustees[j].TrusteeView.ReceivedTranscriptFromRelay(parsed3.Bases, parsed3.GetKeys(), parsed3.GetProofs())
		if err != nil {
			t.Error(err)
		}
		parsed4 := toSend4.(*net.TRU_REL_SHUFFLE_SIG)

		done, err := n.RelayView.ReceivedSignatureFromTrustee(int(parsed4.TrusteeID), parsed4.Sig)

		if done && j != nTrustees-1 {
			t.Error("Relay collecting signature, but is done too early, only received " + strconv.Itoa(j+1) + " signatures out of " + strconv.Itoa(nTrustees))
		}
		if !done && j == nTrustees-1 {
			t.Error("Relay collecting signature, but is not done, yet we have all signatures")
		}
		if err != nil {
			t.Error("Shouldn't have an error here," + err.Error())
		}
	}

	trusteesPks := make([]kyber.Point, nTrustees)
	for j := 0; j < nTrustees; j++ {
		trusteesPks[j] = trustees[j].TrusteeView.PublicKey
	}

	toSend5, err := n.RelayView.VerifySigsAndSendToClients(trusteesPks)
	if err != nil {
		t.Error(err)
	}
	parsed5 := toSend5.(*net.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG)

	mapping := make([]int, nClients)

	//client verify the sig and recognize their slot
	for j := 0; j < nClients; j++ {
		mySlot, err := n.ClientVerifySigAndRecognizeSlot(clients[j].Private, trusteesPks, parsed5.Base, parsed5.EphPks, parsed5.GetSignatures())
		if err != nil {
			t.Error(err)
		}
		mapping[j] = mySlot
	}

	//test that mapping is valid
	for j := 0; j < nClients; j++ {

		if mapping[j] < 0 || mapping[j] >= nClients {
			t.Error("Final mapping invalid,", j, "->", mapping[j])
		}

		//test for duplicate
		mySlot := mapping[j]
		for k := 0; k < nClients; k++ {
			if k != j {
				if mapping[k] == mySlot {
					t.Error("Collision, mapping[", j, "]=mapping[", k, "]=", mySlot)
				}
			}
		}
	}

	return mapping
}

func TestWholeNeffShuffleClientErrors(t *testing.T) {
	n := new(NeffShuffle) //this will hold 1 relay, 1 trustee at most. Recreate n for >1 trustee
	n.Init()
	_, priv := crypto.NewKeyPair()

	//init the trustees
	nTrustees := 2
	trusteesPks := make([]kyber.Point, nTrustees)
	for i := 0; i < nTrustees; i++ {
		pub, _ := crypto.NewKeyPair()
		trusteesPks[i] = pub
	}

	//init the ephemeral keys
	nClients := 4
	ephPks := make([]kyber.Point, nClients)
	for i := 0; i < nTrustees; i++ {
		pub, _ := crypto.NewKeyPair()
		ephPks[i] = pub
	}

	base := config.CryptoSuite.Point().Base()

	//init the sigs
	trusteesSigs := make([][]byte, nTrustees)
	for i := 0; i < nTrustees; i++ {
		trusteesSigs[i] = make([]byte, 2)
	}

	_, err := n.ClientVerifySigAndRecognizeSlot(nil, trusteesPks, base, ephPks, trusteesSigs)
	if err == nil {
		t.Error("ClientVerifySigAndRecognizeSlot should fail without private key")
	}
	_, err = n.ClientVerifySigAndRecognizeSlot(priv, nil, base, ephPks, trusteesSigs)
	if err == nil {
		t.Error("ClientVerifySigAndRecognizeSlot should fail without public keys from trustees")
	}
	_, err = n.ClientVerifySigAndRecognizeSlot(priv, trusteesPks, nil, ephPks, trusteesSigs)
	if err == nil {
		t.Error("ClientVerifySigAndRecognizeSlot should fail without base")
	}
	_, err = n.ClientVerifySigAndRecognizeSlot(priv, trusteesPks, base, nil, trusteesSigs)
	if err == nil {
		t.Error("ClientVerifySigAndRecognizeSlot should fail without the ephemeral keys")
	}
	_, err = n.ClientVerifySigAndRecognizeSlot(priv, trusteesPks, base, ephPks, nil)
	if err == nil {
		t.Error("ClientVerifySigAndRecognizeSlot should fail without signatures from trustees")
	}
	_, err = n.ClientVerifySigAndRecognizeSlot(priv, trusteesPks, base, make([]kyber.Point, 0), trusteesSigs)
	if err == nil {
		t.Error("ClientVerifySigAndRecognizeSlot should fail with 0 ephemeral keys")
	}
	_, err = n.ClientVerifySigAndRecognizeSlot(priv, trusteesPks, base, ephPks, trusteesSigs[0:1])
	if err == nil {
		t.Error("ClientVerifySigAndRecognizeSlot should fail with mismatching sizes between sigs and pks (trustees)")
	}
}

func TestWholeNeffShuffleRelayErrors(t *testing.T) {

	pub, _ := crypto.NewKeyPair()
	//create the scheduler
	n := new(NeffShuffle)
	n.Init()

	err := n.RelayView.Init(0)
	if err == nil {
		t.Error("Should not be able to init a shuffle with 0 trustees")
	}
	err = n.RelayView.Init(1)

	//cannot start without keys
	_, _, err = n.RelayView.SendToNextTrustee()
	if err == nil {
		t.Error("Should not be able to start without any keys")
	}
	n.RelayView.PublicKeyBeingShuffled = make([]kyber.Point, 0)
	_, _, err = n.RelayView.SendToNextTrustee()
	if err == nil {
		t.Error("Should not be able to start without any keys")
	}

	//try adding invalid clients
	err = n.RelayView.AddClient(nil)
	if err == nil {
		t.Error("Should not be able to add an nil key")
	}
	err = n.RelayView.AddClient(pub)
	n.RelayView.SendToNextTrustee()
	err = n.RelayView.AddClient(pub)
	if err == nil {
		t.Error("Should not be able to add a key once SendToNextTrustee has been called")
	}

	//should not accept invalid shares
	base := config.CryptoSuite.Point().Base()
	//init the ephemeral keys
	nClients := 4
	ephPks := make([]kyber.Point, nClients)
	for i := 0; i < nClients; i++ {
		pub, _ := crypto.NewKeyPair()
		ephPks[i] = pub
	}
	proof := make([]byte, 10)

	_, err = n.RelayView.ReceivedShuffleFromTrustee(nil, ephPks, proof)
	if err == nil {
		t.Error("shouldn't accept a shuffle from a trustee if base is nil")
	}
	_, err = n.RelayView.ReceivedShuffleFromTrustee(base, nil, proof)
	if err == nil {
		t.Error("shouldn't accept a shuffle from a trustee if ephPks is nil")
	}
	_, err = n.RelayView.ReceivedShuffleFromTrustee(base, ephPks, nil)
	if err == nil {
		t.Error("shouldn't accept a shuffle from a trustee if proof is nil")
	}
	_, err = n.RelayView.ReceivedShuffleFromTrustee(base, make([]kyber.Point, 0), proof)
	if err == nil {
		t.Error("shouldn't accept a shuffle from a trustee if it contains 0 public keys")
	}

	// cannot send transcript when inner state is invalid
	n.RelayView.Bases = make([]kyber.Point, 1)
	n.RelayView.ShuffledPublicKeys = make([]net.PublicKeyArray, 0)
	n.RelayView.Proofs = make([]net.ByteArray, 2)
	_, err = n.RelayView.SendTranscript()
	if err == nil {
		t.Error("Relay shouldn't try to send obviously wrong transcript")
	}
	n.RelayView.Bases = make([]kyber.Point, 0)
	n.RelayView.ShuffledPublicKeys = make([]net.PublicKeyArray, 0)
	n.RelayView.Proofs = make([]net.ByteArray, 0)
	_, err = n.RelayView.SendTranscript()
	if err == nil {
		t.Error("Relay shouldn't try to send empty transcript")
	}

	//cannot accept a signed shuffle if its invalid !
	_, err = n.RelayView.ReceivedSignatureFromTrustee(0, nil)
	if err == nil {
		t.Error("Relay shouldn't accept a signature if signature's bytes are nil")
	}
	_, err = n.RelayView.ReceivedSignatureFromTrustee(-1, make([]byte, 10))
	if err == nil {
		t.Error("Relay shouldn't accept a signature if trustee signing doesn't give its correct ID")
	}

	//cannot verify if inner state is wrong
	_, err = n.RelayView.VerifySigsAndSendToClients(nil)
	if err == nil {
		t.Error("Relay shouldn't accept to verify without trustees public keys")
	}
	n.RelayView.ShuffledPublicKeys = make([]net.PublicKeyArray, 1)
	n.RelayView.Signatures = make([]net.ByteArray, 2)
	_, err = n.RelayView.VerifySigsAndSendToClients(make([]kyber.Point, 3))
	if err == nil {
		t.Error("Relay shouldn't accept to verify when sizes are mismatching")
	}
}

func TestWholeNeffShuffleTrusteeErrors(t *testing.T) {

	pub, priv := crypto.NewKeyPair()
	//create the scheduler
	n := new(NeffShuffle)
	n.Init()

	err := n.TrusteeView.Init(-1, priv, pub)
	if err == nil {
		t.Error("Should not be able to init a shuffle with an invalid ID")
	}
	err = n.TrusteeView.Init(1, nil, pub)
	if err == nil {
		t.Error("Should not be able to init a shuffle without public key")
	}
	err = n.TrusteeView.Init(1, priv, nil)
	if err == nil {
		t.Error("Should not be able to init a shuffle without private key")
	}

	//sanity check on receivedshuffle
	base := config.CryptoSuite.Point().Base()
	nClients := 4
	ephPks := make([]kyber.Point, nClients)
	for i := 0; i < nClients; i++ {
		pub, _ := crypto.NewKeyPair()
		ephPks[i] = pub
	}
	_, err = n.TrusteeView.ReceivedShuffleFromRelay(nil, ephPks, true, make([]byte, 1))
	if err == nil {
		t.Error("Shouldn't accept a shuffle from the relay when base is nil")
	}
	_, err = n.TrusteeView.ReceivedShuffleFromRelay(base, nil, true, make([]byte, 1))
	if err == nil {
		t.Error("Shouldn't accept a shuffle from the relay when ephPks is nil")
	}
	_, err = n.TrusteeView.ReceivedShuffleFromRelay(base, make([]kyber.Point, 0), true, make([]byte, 1))
	if err == nil {
		t.Error("Shouldn't accept a shuffle from the relay with no keys to shuffle")
	}

	//sanity check on ReceivedTranscriptFromRelay
	bases := make([]kyber.Point, 2)
	shuffledPublicKeys := make([][]kyber.Point, 3)
	proofs := make([][]byte, 4)
	_, err = n.TrusteeView.ReceivedTranscriptFromRelay(nil, shuffledPublicKeys, proofs)
	if err == nil {
		t.Error("Shouldn't accept a transcript with nil instead of bases")
	}
	_, err = n.TrusteeView.ReceivedTranscriptFromRelay(bases, nil, proofs)
	if err == nil {
		t.Error("Shouldn't accept a transcript with nil instead of bases")
	}
	_, err = n.TrusteeView.ReceivedTranscriptFromRelay(bases, shuffledPublicKeys, nil)
	if err == nil {
		t.Error("Shouldn't accept a transcript with nil instead of bases")
	}
	_, err = n.TrusteeView.ReceivedTranscriptFromRelay(bases, shuffledPublicKeys, proofs)
	if err == nil {
		t.Error("Shouldn't accept a transcript when elements mismatch in sizes")
	}

	//more in-depth checks
	bases = make([]kyber.Point, 1)
	bases[0] = config.CryptoSuite.Point().Base()
	n.TrusteeView.NewBase = bases[0] //base match

	proofs = make([][]byte, 1)
	proofs[0] = []byte{1, 2}
	n.TrusteeView.Proof = []byte{1, 2} //proof match

	n.TrusteeView.EphemeralKeys = ephPks

	newPub, _ := crypto.NewKeyPair()
	ephPks_s := make([][]kyber.Point, 1)
	for i := 0; i < len(ephPks_s); i++ {
		ephPks_s[i] = make([]kyber.Point, len(ephPks))
	}
	for i := 0; i < len(ephPks); i++ {
		ephPks_s[0][i] = ephPks[i]
	}
	ephPks_s[0][0] = newPub

	_, err = n.TrusteeView.ReceivedTranscriptFromRelay(bases, ephPks_s, proofs)
	if err == nil {
		t.Error("Shouldn't accept a transcript when one key has been changed !")
	}
}
