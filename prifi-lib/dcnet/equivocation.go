package dcnet

import (
	"crypto/rand"
	"crypto/sha256"
	"github.com/lbarman/prifi/prifi-lib/config"
	"gopkg.in/dedis/crypto.v0/abstract"
	//"gopkg.in/dedis/onet.v1/log"
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
	history    abstract.Scalar
	randomness abstract.Cipher
	suite      abstract.Suite
}

// NewEquivocation creates the structure that handle equivocation protection
func NewEquivocation() *EquivocationProtection {
	e := new(EquivocationProtection)
	e.suite = config.CryptoSuite
	e.history = e.suite.Scalar()

	randomKey := make([]byte, e.suite.Cipher(nil).KeySize())
	rand.Read(randomKey)
	e.randomness = e.suite.Cipher(randomKey)

	return e
}

func (e *EquivocationProtection) randomScalar() abstract.Scalar {
	return e.suite.Scalar().Pick(e.randomness)
}

func (e *EquivocationProtection) hashInGroup(data []byte) abstract.Scalar {
	return e.suite.Scalar().SetBytes(data)
}

// Update History adds those bits to the history hash chain
func (e *EquivocationProtection) UpdateHistory(data []byte) {
	historyB := e.history.Bytes()
	toBeHashed := make([]byte, len(historyB)+len(data))
	newPayload := sha256.Sum256(toBeHashed)
	e.history.SetBytes(newPayload[:])
}

// a function that takes a payload x, encrypt it as x' = x + k, and returns x' and kappa = k + history * (sum of the (hashes of pads))
func (e *EquivocationProtection) ClientEncryptPayload(x []byte, p_j [][]byte) ([]byte, []byte) {

	// hash the pads p_i into q_i
	q_j := make([]abstract.Scalar, len(p_j))
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
	if x == nil {
		kappa_i := product
		return x, kappa_i.Bytes()
	}

	k_i := e.randomScalar()
	k_i_bytes := k_i.Bytes()

	// encrypt payload
	for i := range x {
		x[i] ^= k_i_bytes[i%len(k_i_bytes)]
	}

	// compute kappa
	kappa_i := k_i.Add(k_i, product)
	return x, kappa_i.Bytes()
}

// a function that takes returns the byte[] version of sigma_j
func (e *EquivocationProtection) TrusteeGetContribution(s_i [][]byte) []byte {

	// hash the pads p_i into q_i
	q_i := make([]abstract.Scalar, len(s_i))
	for client_i := range q_i {
		q_i[client_i] = e.hashInGroup(s_i[client_i])
	}

	// sum of q_i
	sum := e.suite.Scalar().Zero()
	for _, p := range q_i {
		sum = sum.Add(sum, p)
	}

	kappa_j := sum

	return kappa_j.Bytes()
}

// given all contributions, decodes the payload
func (e *EquivocationProtection) RelayDecode(encryptedPayload []byte, trusteesContributions [][]byte, clientsContributions [][]byte) []byte {

	//reconstitute the abstract.Point values
	trustee_kappa_j := make([]abstract.Scalar, len(trusteesContributions))
	for k, v := range trusteesContributions {
		trustee_kappa_j[k] = e.suite.Scalar().SetBytes(v)
	}
	client_kappa_i := make([]abstract.Scalar, len(clientsContributions))
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
		sumClients = sumClients.Add(sumClients, v)
	}

	prod := sumTrustees.Mul(sumTrustees, e.history)
	k_i := sumClients.Sub(sumClients, prod)

	//now use k to decrypt the payload
	k_bytes := k_i.Bytes()

	// decrypt the data
	for i := range encryptedPayload {
		encryptedPayload[i] ^= k_bytes[i%len(k_bytes)]
	}

	return encryptedPayload
}
