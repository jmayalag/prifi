package scheduler

import (
	"fmt"
	"github.com/dedis/cothority/network"
	"github.com/dedis/crypto/abstract"
	cryptoconfig "github.com/dedis/crypto/config"
	"github.com/dedis/crypto/random"
	"github.com/lbarman/prifi/prifi-lib"
	"github.com/lbarman/prifi/prifi-lib/config"
	"strconv"
	"testing"
)

type PrivatePublicPair struct {
	Private abstract.Scalar
	Public  abstract.Point
}

/*
 * Client 1 computes : p1 * B = P1
 * Client 2 computes : p2 * B = P2
 * Relay sends P1, P2, s0 to trustee
 * Trustee 1 pick c1
 * Trustee 1 compute s1 = s0 * c1
 * Trustee 1 compute P1' = P1 * c1 = p1 * c1 * B
 * Trustee 1 compute P2' = P2 * c1 = p2 * c1 * B
 * Relay sends P1', P2', s1
 * Trustee 2 pick c2
 * Trustee 2 compute s2 = s1 * c2 = s0 * c1 * c2
 * Trustee 2 compute P1'' = P1' * c2 = p1 * c1 * c2 * B
 * Trustee 2 compute P2'' = P2' * c2 = p2 * c1 * c2 * B
 * Relay sends P1'', P2'', s2 to clients
 * Client 1 compute p1 * s2 * B = p1 * s0 * c1 * c2 * B = P1''
 * Client 2 compute p2 * s2 * B = p2 * s0 * c1 * c2 * B = P2''
 */
func TestNeff(t *testing.T) {

	nTrustees := 2
	nClients := 2

	clients := make([]*PrivatePublicPair, nClients)
	for i := 0; i < nClients; i++ {
		pub, priv := genKeyPair()
		clients[i] = new(PrivatePublicPair)
		clients[i].Public = pub
		clients[i].Private = priv
	}

	//create the scheduler
	n := new(neffShuffleScheduler) //this will hold 1 relay, 1 trustee at most. Recreate n for >1 trustee
	n.init()

	//init the trustees
	trustees := make([]*neffShuffleScheduler, nTrustees)
	for i := 0; i < nTrustees; i++ {
		trustees[i] = new(neffShuffleScheduler)
		trustees[i].init()
		trustee := cryptoconfig.NewKeyPair(network.Suite)
		trustees[i].TrusteeView.init(i, trustee.Secret, trustee.Public)
	}

	//init the relay
	n.RelayView.init(nTrustees)
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
		err, toSend := n.RelayView.SendToNextTrustee()
		if err != nil {
			t.Error(err)
		}
		parsed := toSend.(*prifi_lib.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE)

		//Specialized test for trustee n°2 (1st trustee)
		if i == 1 {
			//Those test are just to see if we transmit (content of "parsed") correctly, maths are already checked below
			//s1 (=base) = s0 * c1
			s0 := n.RelayView.InitialCoeff
			c1 := trustees[0].TrusteeView.SecretCoeff
			s0c1 := config.CryptoSuite.Scalar().Mul(s0, c1)
			if !parsed.Base.Equal(s0c1) {
				t.Error("s1 was not computed/transmitted correctly to trustee 2 !")
			}

			//P1' = P1 * c1 = p1 * B * c1
			p1 := clients[0].Private
			B := config.CryptoSuite.Point().Base()
			p1c1 := config.CryptoSuite.Scalar().Mul(p1, c1)
			P1prime := config.CryptoSuite.Point().Mul(B, p1c1)
			if !parsed.Pks[0].Equal(P1prime) {
				t.Error("P1' was not computed/transmitted correctly to trustee 2 !")
			}

			//P2' = P2 * c1 = p2 * B * c1
			p2 := clients[1].Private
			p2c1 := config.CryptoSuite.Scalar().Mul(p2, c1)
			P2prime := config.CryptoSuite.Point().Mul(B, p2c1)
			if !parsed.Pks[1].Equal(P2prime) {
				t.Error("P2' was not computed/transmitted correctly to trustee 2 !")
			}
		}

		//who receives it
		shuffleKeyPos := false //so we can test easily
		err, toSend2 := trustees[i].TrusteeView.ReceivedShuffleFromRelay(parsed.Base, parsed.Pks, shuffleKeyPos)
		if err != nil {
			t.Error(err)
		}
		parsed2 := toSend2.(*prifi_lib.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS)

		// TEST: Trustee i compute s[i] = s[i-1] * c[i]
		s_i_minus_1 := n.RelayView.CurrentShares
		c_i := trustees[i].TrusteeView.SecretCoeff
		s_i := config.CryptoSuite.Scalar().Mul(s_i_minus_1, c_i)

		if !parsed2.NewBase.Equal(s_i) {
			t.Error("S[" + strconv.Itoa(i+1) + "] is computed incorrectly")
		}

		// TEST: Trustee i compute P1'[i] = P1'[i-1] * c[i] = p1 * c[1] ... c[i] * B
		p1 := clients[0].Private
		c_s := config.CryptoSuite.Scalar().One()
		for j := 0; j <= i; j++ {
			c_j := trustees[j].TrusteeView.SecretCoeff
			c_s = config.CryptoSuite.Scalar().Mul(c_s, c_j)
		}
		B := config.CryptoSuite.Point().Base()
		p1_c_s := config.CryptoSuite.Scalar().Mul(p1, c_s)
		LHS := config.CryptoSuite.Point().Mul(B, p1_c_s)

		if !parsed2.NewEphPks[0].Equal(LHS) {
			t.Error("P1'[" + strconv.Itoa(i+1) + "] is computed incorrectly")
		}

		// TEST: Trustee i compute P2'[i] = P2'[i-1] * c[i] = p2 * c[1] ... c[i] * B
		p2 := clients[1].Private
		p2_c_s := config.CryptoSuite.Scalar().Mul(p2, c_s)
		LHS = config.CryptoSuite.Point().Mul(B, p2_c_s)

		if !parsed2.NewEphPks[1].Equal(LHS) {
			t.Error("P2'[" + strconv.Itoa(i+1) + "] is computed incorrectly")
		}

		//Specialized test for trustee n°1 (0-th trustee)
		if i == 0 {
			// Trustee 1 compute s1 = s0 * c1
			B := config.CryptoSuite.Point().Base()
			c1 := trustees[0].TrusteeView.SecretCoeff
			s0 := n.RelayView.InitialCoeff
			if !parsed2.NewBase.Equal(config.CryptoSuite.Scalar().Mul(s0, c1)) {
				t.Error("S1 is computed incorrectly")
			}

			// Trustee 1 compute P1' = P1 * c1 = p1 * c1 * B
			p1prime := config.CryptoSuite.Scalar().Mul(clients[0].Private, c1)
			if !parsed2.NewEphPks[0].Equal(config.CryptoSuite.Point().Mul(B, p1prime)) {
				t.Error("P1' is computed incorrectly")
			}

			// Trustee 1 compute P2' = P2 * c1 = p2 * c1 * B
			p2prime := config.CryptoSuite.Scalar().Mul(clients[1].Private, c1)
			if !parsed2.NewEphPks[1].Equal(config.CryptoSuite.Point().Mul(B, p2prime)) {
				t.Error("P2' is computed incorrectly")
			}
		}

		//Specialized test for trustee n°2 (1st trustee)
		if i == 1 {

			//* Trustee 2 compute s2 = s1 * c2 = s0 * c1 * c2
			B := config.CryptoSuite.Point().Base()
			c1 := trustees[0].TrusteeView.SecretCoeff
			c2 := trustees[1].TrusteeView.SecretCoeff
			c1c2 := config.CryptoSuite.Scalar().Mul(c1, c2)
			s0 := n.RelayView.InitialCoeff
			if !parsed2.NewBase.Equal(config.CryptoSuite.Scalar().Mul(s0, c1c2)) {
				t.Error("S2 is computed incorrectly (2)")
			}

			//* Trustee 2 compute P1'' = P1' * c2 = p1 * c1 * c2 * B
			p1prime2 := config.CryptoSuite.Scalar().Mul(clients[0].Private, c1c2)
			if !parsed2.NewEphPks[0].Equal(config.CryptoSuite.Point().Mul(B, p1prime2)) {
				t.Error("P1'' is computed incorrectly")
			}
			//* Trustee 2 compute P2'' = P2' * c2 = p2 * c1 * c2 * B
			p2prime2 := config.CryptoSuite.Scalar().Mul(clients[1].Private, c1c2)
			if !parsed2.NewEphPks[1].Equal(config.CryptoSuite.Point().Mul(B, p2prime2)) {
				t.Error("P2'' is computed incorrectly")
			}
		}

		//and answers, the relay receives it
		err, isDone = n.RelayView.ReceivedShuffleFromTrustee(parsed2.NewBase, parsed2.NewEphPks, parsed2.Proof)

		i++
	}

	err, toSend3 := n.RelayView.SendTranscript()
	if err != nil {
		t.Error(err)
	}
	parsed3 := toSend3.(*prifi_lib.REL_TRU_TELL_TRANSCRIPT)

	for j := 0; j < nTrustees; j++ {
		err, toSend4 := trustees[j].TrusteeView.ReceivedTranscriptFromRelay(parsed3.Gs, parsed3.EphPks, parsed3.Proofs)
		if err != nil {
			t.Error(err)
		}
		parsed4 := toSend4.(*prifi_lib.TRU_REL_SHUFFLE_SIG)

		err, done := n.RelayView.ReceivedSignatureFromTrustee(parsed4.TrusteeID, parsed4.Sig)

		if done && j != nTrustees-1 {
			t.Error("Relay collecting signature, but is done too early, only received " + strconv.Itoa(j+1) + " signatures out of " + strconv.Itoa(nTrustees))
		}
		if !done && j == nTrustees-1 {
			t.Error("Relay collecting signature, but is not done, yet we have all signatures")
		}
	}

	trusteesPks := make([]abstract.Point, nTrustees)
	for j := 0; j < nTrustees; j++ {
		trusteesPks[j] = trustees[j].TrusteeView.PublicKey
	}

	err, toSend5 := n.RelayView.VerifySigsAndSendToClients(trusteesPks)
	if err != nil {
		t.Error(err)
	}
	parsed5 := toSend5.(*prifi_lib.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG)

	//client verify the sig and recognize their slot
	for j := 0; j < nClients; j++ {
		err, mySlot := n.ClientVerifySigAndRecognizeSlot(clients[j].Private, trusteesPks, parsed5.Base, parsed5.EphPks, parsed5.TrusteesSigs)
		if err != nil {
			t.Error(err)
		}
		fmt.Println("Client", j, "got assigned slot", mySlot)
	}

}

func genKeyPair() (abstract.Point, abstract.Scalar) {

	base := config.CryptoSuite.Point().Base()
	priv := config.CryptoSuite.Scalar().Pick(random.Stream)
	pub := config.CryptoSuite.Point().Mul(base, priv)

	return pub, priv
}
