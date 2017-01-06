package scheduler

type NeffShuffle struct {
	RelayView   *neffShuffleRelayView
	TrusteeView *neffShuffleTrusteeView
	//client do not have a "view", no state to hold
}

func (n *NeffShuffle) Init() {
	n.RelayView = new(neffShuffleRelayView)
	n.TrusteeView = new(neffShuffleTrusteeView)
}
