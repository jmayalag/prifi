package crypto

import (
	"github.com/dedis/prifi/prifi-lib/config"
	"gopkg.in/dedis/kyber.v2"
	"testing"

	"fmt"
	"strconv"
)

func TestNeffErrors(t *testing.T) {

	nClients := 2
	base := config.CryptoSuite.Point().Base()

	//build the client's public key array
	clientPks := make([]kyber.Point, nClients)
	clientPrivKeys := make([]kyber.Scalar, nClients)
	for i := 0; i < nClients; i++ {
		pub, priv := NewKeyPair()
		clientPks[i] = pub
		clientPrivKeys[i] = priv
	}

	//each of those call should fail
	_, _, _, _, err := NeffShuffle(nil, base, true)
	if err == nil {
		t.Error("NeffShuffle without a public key array should fail")
	}
	_, _, _, _, err = NeffShuffle(clientPks, nil, true)
	if err == nil {
		t.Error("NeffShuffle without a base should fail")
	}
	_, _, _, _, err = NeffShuffle(make([]kyber.Point, 0), base, true)
	if err == nil {
		t.Error("NeffShuffle with 0 public keys should fail")
	}

}

func TestNeffShuffle(t *testing.T) {

	//output distribution testing.
	nClientsRange := []int{2, 3, 4, 5, 10}
	repetition := 100
	base := config.CryptoSuite.Point().Base()
	maxUnfairness := 30 //30% of difference between the shuffle's homogeneity

	for _, nClients := range nClientsRange {
		fmt.Println("Testing shuffle for ", nClients, " clients.")

		//build the client's public key array
		clientPks := make([]kyber.Point, nClients)
		clientPrivKeys := make([]kyber.Scalar, nClients)
		for i := 0; i < nClients; i++ {
			pub, priv := NewKeyPair()
			clientPks[i] = pub
			clientPrivKeys[i] = priv
		}

		//shuffle
		shuffledKeys, newBase, secretCoeff, proof, err := NeffShuffle(clientPks, base, true)

		if err != nil {
			t.Error(err)
		}
		if shuffledKeys == nil {
			t.Error("ShuffledKeys is nil")
		}
		if newBase == nil {
			t.Error("newBase is nil")
		}
		if secretCoeff == nil {
			t.Error("secretCoeff is nil")
		}
		if proof == nil {
			t.Error("proof is nil")
		}

		//now test that the shuffled keys are indeed the old keys in the new base
		transformedKeys := make([]kyber.Point, nClients)
		for i := 0; i < nClients; i++ {
			transformedKeys[i] = config.CryptoSuite.Point().Mul(clientPrivKeys[i], newBase)
		}

		//for every key, check that it exists in the remaining array
		for _, v := range transformedKeys {
			found := false
			for i := 0; i < nClients; i++ {
				if shuffledKeys[i].Equal(v) {
					found = true
				}
			}

			if !found {
				t.Error("Public key not found in outbound array")
			}
		}

		//test the distribution
		mappingDistrib := make([][]float64, nClients)
		for k := 0; k < nClients; k++ {
			mappingDistrib[k] = make([]float64, nClients)
		}
		fmt.Print("Testing distribution for ", nClients, " clients.")
		for i := 0; i < repetition; i++ {
			shuffledKeys, newBase, secretCoeff, proof, err = NeffShuffle(clientPks, base, true)

			if err != nil {
				t.Error("Shouldn't have an error here," + err.Error())
			}

			//todo : check the proofs !
			_ = proof
			_ = secretCoeff

			mapping := make([]int, nClients)
			transformedKeys := make([]kyber.Point, nClients)
			for i := 0; i < nClients; i++ {
				transformedKeys[i] = config.CryptoSuite.Point().Mul(clientPrivKeys[i], newBase)
			}
			for k, v := range transformedKeys {
				for i := 0; i < nClients; i++ {
					if shuffledKeys[i].Equal(v) {
						mapping[k] = i
					}
				}
			}

			for clientID, slot := range mapping {
				mappingDistrib[clientID][slot] += float64(1)
			}
		}
		maxDeviation := float64(-1)
		for clientID, _ := range mappingDistrib {
			for slot := 0; slot < nClients; slot++ {
				//compute deviation
				expectedValue := float64(100) / float64(len(mappingDistrib[clientID]))
				mappingDistrib[clientID][slot] -= expectedValue
				if mappingDistrib[clientID][slot] < 0 {
					mappingDistrib[clientID][slot] = -mappingDistrib[clientID][slot]
				}

				//store max deviation
				if mappingDistrib[clientID][slot] > maxDeviation {
					maxDeviation = mappingDistrib[clientID][slot]
				}
			}
		}
		fmt.Printf("+-%d%%\n", int(maxDeviation))
		if int(maxDeviation) > maxUnfairness {
			t.Error("Max allowed distribution biais is " + strconv.Itoa(maxUnfairness) + " percent.")
		}

	}

}
