package scheduler

import (
	"fmt"
	"github.com/dedis/cothority/network"
	"github.com/dedis/crypto/abstract"
	cryptoconfig "github.com/dedis/crypto/config"
	"github.com/dedis/crypto/random"
	"github.com/lbarman/prifi/prifi-lib/config"
	"github.com/lbarman/prifi/prifi-lib/net"
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

	nClientsRange := []int{1, 2, 3, 4, 5, 10, 100}
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
		pub, priv := genKeyPair()
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
		trustee := cryptoconfig.NewKeyPair(network.Suite)
		trustees[i].TrusteeView.Init(i, trustee.Secret, trustee.Public)
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
		toSend2, err := trustees[i].TrusteeView.ReceivedShuffleFromRelay(parsed.Base, parsed.Pks, shuffleKeyPos)
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
		B_i := config.CryptoSuite.Point().Mul(B_i_minus_1, c_i)

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
				LHS := config.CryptoSuite.Point().Mul(B, p1_c_s)

				if !parsed2.NewEphPks[clientID].Equal(LHS) {
					t.Error("P" + strconv.Itoa(clientID) + "'[" + strconv.Itoa(i+1) + "] is computed incorrectly")
				}
			}

			//Specialized test for trustee n°1 (0-th trustee)
			if i == 0 {
				// Trustee 1 compute B1 = B * c1
				B := n.RelayView.InitialBase
				c1 := trustees[0].TrusteeView.SecretCoeff
				if !parsed2.NewBase.Equal(config.CryptoSuite.Point().Mul(B, c1)) {
					t.Error("B1 is computed incorrectly")
				}

				// Trustee 1 compute P1' = P1 * c1 = p1 * c1 * B
				for clientID := 0; clientID < nClients; clientID++ {
					p1prime := config.CryptoSuite.Scalar().Mul(clients[clientID].Private, c1)
					if !parsed2.NewEphPks[clientID].Equal(config.CryptoSuite.Point().Mul(B, p1prime)) {
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
				if !parsed2.NewBase.Equal(config.CryptoSuite.Point().Mul(B, c1c2)) {
					t.Error("B2 is computed incorrectly (2)")
				}

				//* Trustee 2 compute P1'' = P1' * c2 = p1 * c1 * c2 * B
				for clientID := 0; clientID < nClients; clientID++ {
					p1prime2 := config.CryptoSuite.Scalar().Mul(clients[clientID].Private, c1c2)
					if !parsed2.NewEphPks[clientID].Equal(config.CryptoSuite.Point().Mul(B, p1prime2)) {
						t.Error("P" + strconv.Itoa(clientID) + "'' is computed incorrectly")
					}
				}
			}
		}

		//and answers, the relay receives it
		isDone, err = n.RelayView.ReceivedShuffleFromTrustee(parsed2.NewBase, parsed2.NewEphPks, parsed2.Proof)

		i++
	}

	toSend3, err := n.RelayView.SendTranscript()
	if err != nil {
		t.Error(err)
	}
	parsed3 := toSend3.(*net.REL_TRU_TELL_TRANSCRIPT)

	for j := 0; j < nTrustees; j++ {
		toSend4, err := trustees[j].TrusteeView.ReceivedTranscriptFromRelay(parsed3.Bases, parsed3.EphPks, parsed3.Proofs)
		if err != nil {
			t.Error(err)
		}
		parsed4 := toSend4.(*net.TRU_REL_SHUFFLE_SIG)

		done, err := n.RelayView.ReceivedSignatureFromTrustee(parsed4.TrusteeID, parsed4.Sig)

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

	toSend5, err := n.RelayView.VerifySigsAndSendToClients(trusteesPks)
	if err != nil {
		t.Error(err)
	}
	parsed5 := toSend5.(*net.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG)

	mapping := make([]int, nClients)

	//client verify the sig and recognize their slot
	for j := 0; j < nClients; j++ {
		mySlot, err := n.ClientVerifySigAndRecognizeSlot(clients[j].Private, trusteesPks, parsed5.Base, parsed5.EphPks, parsed5.TrusteesSigs)
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

func genKeyPair() (abstract.Point, abstract.Scalar) {

	base := config.CryptoSuite.Point().Base()
	priv := config.CryptoSuite.Scalar().Pick(random.Stream)
	pub := config.CryptoSuite.Point().Mul(base, priv)

	return pub, priv
}
