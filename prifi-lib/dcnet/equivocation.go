package dcnet

import (
	"crypto/rand"
	"crypto/sha256"
	"github.com/dedis/prifi/prifi-lib/config"
	"gopkg.in/dedis/kyber.v2"
	"gopkg.in/dedis/kyber.v2/suites"
	"gopkg.in/dedis/onet.v2/log"
)

// Clients compute:
// kappa_i = k_i + h * SUM_j(q_ij), where q_ij = H(p_ij) in group
// c' = k_i + c
//
// Trustees compute:
// sigma_i = SUM_i(q_ij), where q_ij = H(s_ij) in group
//
// Relay compute:
// k_i = SUM_i(kappa_i) - h * (SUM_j(sigma_i))
//     = SUM_i(h * SUM_j(q_ij)) + k_i - h * SUM_j(SUM_i(q_ij))
// c = k_i + c'
//

// Equivocation holds the functions needed for equivocation protection
type EquivocationProtection struct {
	history    kyber.Scalar
	randomness kyber.XOF
	suite      suites.Suite
}

// NewEquivocation creates the structure that handle equivocation protection
func NewEquivocation() *EquivocationProtection {
	e := new(EquivocationProtection)
	e.suite = config.CryptoSuite
	e.history = e.suite.Scalar().One()

	randomKey := make([]byte, 100)
	rand.Read(randomKey)
	e.randomness = e.suite.XOF(randomKey)

	return e
}

func (e *EquivocationProtection) randomScalar() kyber.Scalar {
	return e.suite.Scalar().Pick(e.randomness)
}

func (e *EquivocationProtection) hashInGroup(data []byte) kyber.Scalar {
	return e.suite.Scalar().SetBytes(data)
}

// Update History adds those bits to the history hash chain
func (e *EquivocationProtection) UpdateHistory(data []byte) {
	historyB, err := e.history.MarshalBinary()
	if err != nil {
		log.Fatal("Could not unmarshall bytes", err)
	}
	toBeHashed := make([]byte, len(historyB)+len(data))
	newPayload := sha256.Sum256(toBeHashed)
	e.history.SetBytes(newPayload[:])
}

// a function that takes a payload x, encrypt it as x' = x + k, and returns x' and kappa = k + history * (sum of the (hashes of pads))
func (e *EquivocationProtection) ClientEncryptPayload(slotOwner bool, x []byte, p_j [][]byte) ([]byte, []byte) {

	// hash the pads p_i into q_i
	q_j := make([]kyber.Scalar, len(p_j))
	for trustee_j := range q_j {
		q_j[trustee_j] = e.hashInGroup(p_j[trustee_j])
	}

	// sum of q_i
	sum := e.suite.Scalar().Zero()
	for _, p := range q_j {
		sum = sum.Add(sum, p)
	}

	product := sum.Mul(sum, e.history)

	//we're not the slot owner
	if !slotOwner {
		kappa_i := product
		kappa_i_bytes, err := kappa_i.MarshalBinary()
		if err != nil {
			log.Fatal("Couldn't marshall", err)
		}
		return x, kappa_i_bytes
	}

	k_i := e.randomScalar()
	k_i_bytes, err := k_i.MarshalBinary()
	if err != nil {
		log.Fatal("Couldn't marshall", err)
	}

	// encrypt payload
	for i := range x {
		x[i] ^= k_i_bytes[i%len(k_i_bytes)]
	}

	// compute kappa
	kappa_i := k_i.Add(k_i, product)
	kappa_i_bytes, err := kappa_i.MarshalBinary()
	if err != nil {
		log.Fatal("Couldn't marshall", err)
	}
	return x, kappa_i_bytes
}

// a function that takes returns the byte[] version of sigma_j
func (e *EquivocationProtection) TrusteeGetContribution(s_i [][]byte) []byte {

	// hash the pads p_i into q_i
	q_i := make([]kyber.Scalar, len(s_i))
	for client_i := range q_i {
		q_i[client_i] = e.hashInGroup(s_i[client_i])
	}

	// sum of q_i
	sum := e.suite.Scalar().Zero()
	for _, p := range q_i {
		sum = sum.Add(sum, p)
	}

	kappa_j := sum

	kappa_j_bytes, err := kappa_j.MarshalBinary()
	if err != nil {
		log.Fatal("Couldn't marshall", err)
	}
	return kappa_j_bytes
}

// given all contributions, decodes the payload
func (e *EquivocationProtection) RelayDecode(encryptedPayload []byte, trusteesContributions [][]byte, clientsContributions [][]byte) []byte {

	//reconstitute the abstract.Point values
	trustee_kappa_j := make([]kyber.Scalar, len(trusteesContributions))
	for k, v := range trusteesContributions {
		trustee_kappa_j[k] = e.suite.Scalar().SetBytes(v)
	}
	client_kappa_i := make([]kyber.Scalar, len(clientsContributions))
	for k, v := range clientsContributions {
		client_kappa_i[k] = e.suite.Scalar().SetBytes(v)
	}

	// compute sum of trustees contribs
	sumTrustees := e.suite.Scalar().Zero()
	for _, v := range trustee_kappa_j {
		sumTrustees = sumTrustees.Add(sumTrustees, v)
	}

	// compute sum of clients contribs
	sumClients := e.suite.Scalar().Zero()
	for _, v := range client_kappa_i {
		//log.Lvl1("Adding in", v, "value is now", sumClients)
		sumClients = sumClients.Add(sumClients, v)
	}

	prod := sumTrustees.Mul(sumTrustees, e.history)
	k_i := sumClients.Sub(sumClients, prod)

	//now use k to decrypt the payload
	k_bytes, err := k_i.MarshalBinary()
	if err != nil {
		log.Fatal("Couldn't marshall", err)
	}

	if len(k_bytes) == 0 {
		log.Lvl1("Error: Equivocation couldn't recover the k_bytes value")
		log.Lvl1("encryptedPayload:", encryptedPayload)
		for k, v := range trusteesContributions {
			log.Lvl1("trusteesContributions:", v, "mapped to", trustee_kappa_j[k])
		}
		for k, v := range clientsContributions {
			log.Lvl1("clientsContributions:", v, "mapped to", client_kappa_i[k])
		}
		log.Lvl1("sumTrustees:", sumTrustees)
		log.Lvl1("sumClients:", sumClients)
		log.Lvl1("history:", e.history)
		log.Lvl1("prod:", prod)
		log.Lvl1("k_i:", k_i)
		return make([]byte, 0)
	}

	// decrypt the payload
	for i := range encryptedPayload {
		encryptedPayload[i] ^= k_bytes[i%len(k_bytes)]
	}

	return encryptedPayload
}
