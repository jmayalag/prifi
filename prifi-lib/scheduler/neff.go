package scheduler

type neffShuffleScheduler struct {
	RelayView   *neffShuffleRelayView
	TrusteeView *neffShuffleTrusteeView
}

func (n *neffShuffleScheduler) init() {
	n.RelayView = new(neffShuffleRelayView)
	n.TrusteeView = new(neffShuffleTrusteeView)
}
