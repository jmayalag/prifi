package scheduler

type NeffShuffle struct {
	RelayView   *NeffShuffleRelay
	TrusteeView *NeffShuffleTrustee
	//client do not have a "view", no state to hold
}

func (n *NeffShuffle) Init() {
	n.RelayView = new(NeffShuffleRelay)
	n.TrusteeView = new(NeffShuffleTrustee)
}
