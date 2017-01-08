package crypto

import (
	"github.com/dedis/crypto/abstract"
	"github.com/lbarman/prifi/prifi-lib/config"
	"testing"

	"fmt"
	"github.com/dedis/crypto/random"
	"strconv"
)

func TestSchnorr(t *testing.T) {

	pub, priv := genKeyPair()
	pub2, priv2 := genKeyPair()

	//with empty data
	data := make([]byte, 0)
	sig := SchnorrSign(config.CryptoSuite, random.Stream, data, priv)
	err := SchnorrVerify(config.CryptoSuite, data, pub, sig)

	if err != nil {
		t.Error("Should validate with nil message, err is " + err.Error())
	}

	//with empty data
	data = random.Bits(100, false, random.Stream)
	sig = SchnorrSign(config.CryptoSuite, random.Stream, data, priv)
	err = SchnorrVerify(config.CryptoSuite, data, pub, sig)

	if err != nil {
		t.Error("Should validate with random message, err is " + err.Error())
	}

	//should trivially not validate with other keys
	data = random.Bits(100, false, random.Stream)
	sig = SchnorrSign(config.CryptoSuite, random.Stream, data, priv2)
	err = SchnorrVerify(config.CryptoSuite, data, pub, sig)

	if err == nil {
		t.Error("Should not validate with wrong keys")
	}
	data = random.Bits(100, false, random.Stream)
	sig = SchnorrSign(config.CryptoSuite, random.Stream, data, priv)
	err = SchnorrVerify(config.CryptoSuite, data, pub2, sig)

	if err == nil {
		t.Error("Should not validate with wrong keys")
	}

}

func TestSchnorrHash(t *testing.T) {

	pub, _ := genKeyPair()
	data := random.Bits(100, false, random.Stream)
	secret := hashSchnorr(config.CryptoSuite, data, pub)

	if secret == nil {
		t.Error("Secret should not be nil")
	}
}

func genKeyPair() (abstract.Point, abstract.Scalar) {

	base := config.CryptoSuite.Point().Base()
	priv := config.CryptoSuite.Scalar().Pick(random.Stream)
	pub := config.CryptoSuite.Point().Mul(base, priv)

	return pub, priv
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
		clientPks := make([]abstract.Point, nClients)
		clientPrivKeys := make([]abstract.Scalar, nClients)
		for i := 0; i < nClients; i++ {
			pub, priv := genKeyPair()
			clientPks[i] = pub
			clientPrivKeys[i] = priv
		}

		//shuffle
		shuffledKeys, newBase, secretCoeff, proof, err := NeffShuffle(clientPks, base, config.CryptoSuite, true)

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
		transformedKeys := make([]abstract.Point, nClients)
		for i := 0; i < nClients; i++ {
			transformedKeys[i] = config.CryptoSuite.Point().Mul(newBase, clientPrivKeys[i])
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
			shuffledKeys, newBase, secretCoeff, proof, err = NeffShuffle(clientPks, base, config.CryptoSuite, true)

			mapping := make([]int, nClients)
			transformedKeys := make([]abstract.Point, nClients)
			for i := 0; i < nClients; i++ {
				transformedKeys[i] = config.CryptoSuite.Point().Mul(newBase, clientPrivKeys[i])
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
