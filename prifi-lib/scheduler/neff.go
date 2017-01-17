package scheduler

/**
 * Holds all the components to do a Neff Shuffle. Both the Relay and the Trustee have one instance of it, but uses only
 * their part in it.
 * The struct is aware of net/ and messages, but only to craft them. Sending the messages is the responsibility of the
 * caller
 */
type NeffShuffle struct {
	RelayView   *NeffShuffleRelay
	TrusteeView *NeffShuffleTrustee
	//client do not have a "view", no state to hold
}

/**
 * Instanciates both the relay and the trustee view (but you still need to call init on the correct one)
 */
func (n *NeffShuffle) Init() {
	n.RelayView = new(NeffShuffleRelay)
	n.TrusteeView = new(NeffShuffleTrustee)
}
