package prifi_lib

/**
 * Stores ciphers for different rounds
 */
type BufferManager struct {
}

/**
 * Creates a BufferManager that will expect nClients + nTrustees ciphers per round
 */
func (b *BufferManager) Init(nClients, nTrustees int) error {

	return nil
}

/**
 * Adds a trustee cipher for a given round
 */
func (b *BufferManager) AddTrusteeCipher(roundID int32, trusteeID int, data []byte) error {

	return nil
}

/**
 * Adds a client cipher for a given round
 */
func (b *BufferManager) AddClientCipher(roundID int32, clientID int, data []byte) error {

	return nil
}

/**
 * Returns the current round we are in.
 */
func (b *BufferManager) CurrentRound() int32 {

	return 0
}

/**
 * Returns true iff we received exactly one cipher for every client and trustee for this round
 */
func (b *BufferManager) HasAllCiphersForCurrentRound() bool {

	return false
}

/**
 * Returns a pair of (clientIDs, trusteesIDs) where those entities did not send a cipher for this round
 */
func (b *BufferManager) MissingCiphersForCurrentRound() ([]int, []int) {

	return make([]int, 0), make([]int, 0)
}

/**
 * Finalizes this round, returning all ciphers stored, then increasing the round number.
 * Should only be called when HasAllCiphersForCurrentRound() == true
 */
func (b *BufferManager) FinalizeRound() ([][]byte, error) {

	return make([][]byte, 0), nil
}

/**
 * When moving on to a next round, we should check which cipher we already have
 */
func (b *BufferManager) fetchCiphersForNextRound() ([][]byte, error) {

	return make([][]byte, 0), nil
}
