package relay

import (
	"github.com/lbarman/prifi/prifi-lib/net"
	"testing"
)

func TestDCNetRound(test *testing.T) {

	data := make([]byte, 101)
	window := 10
	dcmr := NewDCNetRoundManager(window)

	if dcmr.CurrentRound() != 0 {
		test.Error("Should be in round 0")
	}
	if !dcmr.CurrentRoundIsStill(0) {
		test.Error("Should still be in round 0")
	}

	_ = data

}
