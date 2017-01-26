package relay

import (
	"errors"
	"sync"
)

/**
 * Stores ciphers for different rounds
 */
type BufferManager struct {
	sync.Mutex

	//immutable
	nClients  int
	nTrustees int

	//changes every call to Finalize()
	currentRoundID int32

	//the ACK map for this round
	clientAckMap  map[int]bool
	trusteeAckMap map[int]bool

	//hold the real data. map(trustee/clientID -> map( roundID -> data))
	bufferedClientCiphers  map[int]map[int32][]byte
	bufferedTrusteeCiphers map[int]map[int32][]byte

	//stop/resume functions when we have too much/little ciphers
	DoSendStopResumeMessages bool
	LowBound                 int //restart sending at lowerbound
	HighBound                int //stop sending at higherbound
	stopFunction             func(int)
	stopSent                 bool
	resumeFunction           func(int)
	resumeSent               bool
}

/**
 * Creates a BufferManager that will expect nClients + nTrustees ciphers per round
 */
func (b *BufferManager) Init(nClients, nTrustees int) error {
	if nClients+nTrustees == 0 {
		return errors.New("Can't init a bufferManager with no clients nor trustees")
	}

	b.currentRoundID = int32(0)
	b.nClients = nClients
	b.nTrustees = nTrustees
	b.resetACKmaps()

	b.bufferedClientCiphers = make(map[int]map[int32][]byte)
	b.bufferedTrusteeCiphers = make(map[int]map[int32][]byte)

	return nil
}

/**
 * Adds a component to the BufferManager, that reacts to the # of buffered cipher (per trustees), and call stopFn()
 * and resumeFn() when the bounds are reached
 */
func (b *BufferManager) AddRateLimiter(lowBound, highBound int, stopFunction, resumeFunction func(int)) error {
	if lowBound < 0 || lowBound > highBound {
		return errors.New("Lowbound must be > 0 and < highBound")
	}
	if highBound < lowBound {
		return errors.New("Highbound must be > lowBound")
	}
	if stopFunction == nil {
		return errors.New("Can't initiate a RateLimiter without a stop function")
	}
	if resumeFunction == nil {
		return errors.New("Can't initiate a RateLimiter without a resume function")
	}

	b.DoSendStopResumeMessages = true
	b.LowBound = lowBound
	b.HighBound = highBound
	b.stopFunction = stopFunction
	b.resumeFunction = resumeFunction

	return nil
}

func addToBuffer(bufferPtr *map[int]map[int32][]byte, roundID int32, entityID int, data []byte) {

	buffer := *bufferPtr
	if buffer[entityID] == nil {
		buffer[entityID] = make(map[int32][]byte)
	}

	buffer[entityID][roundID] = data
}

/**
 * Adds a trustee cipher for a given round
 */
func (b *BufferManager) AddTrusteeCipher(roundID int32, trusteeID int, data []byte) error {
	b.Lock()
	defer b.Unlock()

	if data == nil {
		return errors.New("Can't accept a nil trustee cipher")
	}
	if roundID < b.currentRoundID {
		return errors.New("Can't accept a trustee cipher in the past")
	}
	addToBuffer(&b.bufferedTrusteeCiphers, roundID, trusteeID, data)

	if roundID == b.currentRoundID {
		b.trusteeAckMap[trusteeID] = true
	}

	b.sendRateChangeIfNeeded(trusteeID)

	return nil
}

func (b *BufferManager) sendRateChangeIfNeeded(trusteeID int) {
	if b.DoSendStopResumeMessages {
		n := b.NumberOfBufferedCiphers(trusteeID)
		if n >= b.HighBound && !b.stopSent {
			b.stopFunction(trusteeID)
			b.stopSent = true
			b.resumeSent = false
		} else if n <= b.LowBound && !b.resumeSent {
			b.resumeFunction(trusteeID)
			b.stopSent = false
			b.resumeSent = true
		}
	}
}

/**
 * Adds a client cipher for a given round
 */
func (b *BufferManager) AddClientCipher(roundID int32, clientID int, data []byte) error {
	b.Lock()
	defer b.Unlock()

	if data == nil {
		return errors.New("Can't accept a nil client cipher")
	}
	if roundID < b.currentRoundID {
		return errors.New("Can't accept a client cipher in the past")
	}
	addToBuffer(&b.bufferedClientCiphers, roundID, clientID, data)

	if roundID == b.currentRoundID {
		b.clientAckMap[clientID] = true
	}

	return nil
}

/**
 * Returns the current round we are in.
 */
func (b *BufferManager) CurrentRound() int32 {
	return b.currentRoundID
}

/**
 * Returns true iff we received exactly one cipher for every client and trustee for this round
 */
func (b *BufferManager) HasAllCiphersForCurrentRound() bool {
	b.Lock()
	defer b.Unlock()
	return b.hasAllCiphersForCurrentRound()
}

/**
 * Returns true iff we received exactly one cipher for every client and trustee for this round
 */
func (b *BufferManager) hasAllCiphersForCurrentRound() bool {

	for _, v := range b.clientAckMap {
		if !v {
			return false
		}
	}
	for _, v := range b.trusteeAckMap {
		if !v {
			return false
		}
	}
	return true
}

/**
 * Returns the number of buffered ciphers for this trustee.
 */
func (b *BufferManager) NumberOfBufferedCiphers(trusteeID int) int {
	return len(b.bufferedTrusteeCiphers[trusteeID])
}

/**
 * Returns a pair of (clientIDs, trusteesIDs) where those entities did not send a cipher for this round
 */
func (b *BufferManager) MissingCiphersForCurrentRound() ([]int, []int) {
	b.Lock()
	defer b.Unlock()

	clientMissing := make([]int, 0)
	for k, v := range b.clientAckMap {
		if !v {
			clientMissing = append(clientMissing, k)
		}
	}
	trusteeMissing := make([]int, 0)
	for k, v := range b.trusteeAckMap {
		if !v {
			trusteeMissing = append(trusteeMissing, k)
		}
	}

	return clientMissing, trusteeMissing
}

/**
 * Finalizes this round, returning all ciphers stored, then increasing the round number.
 * Should only be called when HasAllCiphersForCurrentRound() == true
 */
func (b *BufferManager) FinalizeRound() ([][]byte, [][]byte, error) {
	b.Lock()
	defer b.Unlock()

	if !b.hasAllCiphersForCurrentRound() {
		return nil, nil, errors.New("Cannot finalize round yet, missing ciphers")
	}

	//prepare the output, discard those ciphers
	clientsOut := make([][]byte, 0)
	for i := 0; i < b.nClients; i++ {
		clientsOut = append(clientsOut, b.bufferedClientCiphers[i][b.currentRoundID])
		delete(b.bufferedClientCiphers[i], b.currentRoundID)
	}
	trusteesOut := make([][]byte, 0)
	for i := 0; i < b.nTrustees; i++ {
		trusteesOut = append(trusteesOut, b.bufferedTrusteeCiphers[i][b.currentRoundID])
		delete(b.bufferedTrusteeCiphers[i], b.currentRoundID)
	}

	//change round
	b.currentRoundID++

	//reset the map
	b.resetACKmaps()

	//use the cipher we already stored
	for i := 0; i < b.nClients; i++ {
		if _, exists := b.bufferedClientCiphers[i][b.currentRoundID]; exists {
			b.clientAckMap[i] = true
		}
	}
	for i := 0; i < b.nTrustees; i++ {
		if _, exists := b.bufferedTrusteeCiphers[i][b.currentRoundID]; exists {
			b.trusteeAckMap[i] = true
		}
	}

	//send rate changes if needed
	for trusteeID := range b.bufferedTrusteeCiphers {
		b.sendRateChangeIfNeeded(trusteeID)
	}

	return clientsOut, trusteesOut, nil
}

/**
 * Resets to 0 (all false) the two acks maps
 */
func (b *BufferManager) resetACKmaps() {

	b.clientAckMap = make(map[int]bool)
	b.trusteeAckMap = make(map[int]bool)

	for i := 0; i < b.nClients; i++ {
		b.clientAckMap[i] = false
	}
	for i := 0; i < b.nTrustees; i++ {
		b.trusteeAckMap[i] = false
	}
}
