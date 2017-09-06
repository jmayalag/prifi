package crypto

import (
	"crypto/sha256"
	"math/big"
	"math/rand"
)

// Equivocation holds the functions needed for equivocation protection
type Equivocation struct {
	p          *big.Int
	q          *big.Int
	phi        *big.Int
	modulus    *big.Int
	history    *big.Int
	randomness *rand.Rand
}

// NewEquivocation creates the structure that handle equivocation protection
// Usually, make sure that bitLength is greater than the data payload bit size. Otherwise, the blinding key
// will repeat, making it a non-one-time-pad
func NewEquivocation(bitLength int) *Equivocation {
	e := new(Equivocation)
	e.randomness = rand.New(rand.NewSource(0)) // TODO do not use this in production, deterministic !

	e.modulus = big.NewInt(0)
	e.history = big.NewInt(1)

	upperBound := big.NewInt(1)
	lowerBound := big.NewInt(1)
	upperBound.Lsh(upperBound, uint(bitLength))
	lowerBound.Lsh(lowerBound, uint(bitLength-1))

	for lowerBound.Cmp(e.modulus) > -1 || !e.modulus.ProbablyPrime(10) {
		e.modulus.Rand(e.randomness, upperBound)
	}

	return e
}

// Update History adds those bits to the history hash chain
func (e *Equivocation) UpdateHistory(data []byte) {

	oldHistory := make([]byte, 0)
	if e.history != nil {
		oldHistory = e.history.Bytes()
	}
	newHistoryBytes := hash(append(oldHistory, data...))

	e.history.SetBytes(newHistoryBytes[0:2])
}

func hash(b []byte) []byte {
	t := sha256.Sum256(b)
	return t[:]
}

// a function that takes a payload x, encrypt it as x' = x + k, and returns x' and kappa = k * history ^ (sum of the (hashes of pads))
func (e *Equivocation) ClientEncryptPayload(payload []byte, pads [][]byte) ([]byte, []byte) {

	// hash the pads
	hashOfPads := make([][]byte, len(pads))
	for k := range hashOfPads {
		hashOfPads[k] = hash(pads[k])
	}

	// sum the hash
	sum := new(big.Int)
	for _, v := range hashOfPads {
		v2 := new(big.Int)
		v2.SetBytes(v)
		sum = sum.Add(sum, v2)
	}

	blindingFactor := new(big.Int)
	blindingFactor.Exp(e.history, sum, e.modulus)

	//we're not the slot owner
	if payload == nil {
		// compute kappa
		blindingFactor_bytes := blindingFactor.Bytes()
		return nil, blindingFactor_bytes
	}

	// pick random key k
	k := new(big.Int)
	k.Rand(e.randomness, e.modulus)
	k_bytes := k.Bytes()

	// encrypt payload
	for i := range payload {
		payload[i] ^= k_bytes[i%len(k_bytes)]
	}

	// compute kappa
	kappa := new(big.Int)
	kappa = k.Mul(k, blindingFactor)

	kappa_bytes := kappa.Bytes()

	return payload, kappa_bytes
}

// a function that takes a payload x, encrypt it as x' = x + k, and returns x' and kappa = k * history ^ (sum of the (hashes of pads))
func (e *Equivocation) TrusteeGetContribution(pads [][]byte) []byte {

	// hash the pads
	hashOfPads := make([][]byte, len(pads))
	for k := range hashOfPads {
		hashOfPads[k] = hash(pads[k])
	}

	// sum the hash
	sum := new(big.Int)
	for _, v := range hashOfPads {
		v2 := new(big.Int)
		v2.SetBytes(v)
		sum.Add(sum, v2)
	}

	return sum.Bytes()
}

// given all contributions, decodes the payload
func (e *Equivocation) RelayDecode(encryptedPayload []byte, trusteesContributions [][]byte, clientsContributions [][]byte) []byte {

	//reconstitute the bigInt values
	trusteesContrib := make([]*big.Int, len(trusteesContributions))
	for k, v := range trusteesContributions {
		trusteesContrib[k] = new(big.Int)
		trusteesContrib[k].SetBytes(v)
	}
	clientsContrib := make([]*big.Int, len(clientsContributions))
	for k, v := range clientsContributions {
		clientsContrib[k] = new(big.Int)
		clientsContrib[k].SetBytes(v)
	}

	// compute sum of trustees contribs
	sum := new(big.Int)
	for _, v := range trusteesContrib {
		sum.Add(sum, v)
	}

	inverse := big.NewInt(0)
	inverse.ModInverse(e.history, e.modulus)

	firstPart := inverse
	firstPart.Exp(firstPart, sum, e.modulus)

	k := firstPart
	for _, v := range clientsContrib {
		k.Mul(k, v)
		k.Mod(k, e.modulus)
	}
	k.Mod(k, e.modulus)

	//now use k to decrypt the payload
	k_bytes := k.Bytes()

	// decrypt the data
	for i := range encryptedPayload {
		encryptedPayload[i] ^= k_bytes[i%len(k_bytes)]
	}

	return encryptedPayload
}
