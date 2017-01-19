package relay

import (
	"github.com/lbarman/prifi/prifi-lib/net"
	"testing"
)

func TestDCNetRound(test *testing.T) {

	data := net.REL_CLI_DOWNSTREAM_DATA{
		RoundID:    101,
		Data:       make([]byte, 101),
		FlagResync: true,
	}
	dc := NewDCNetRound(100, &data)

	if dc.CurrentRound() != 100 {
		test.Error("Should be in round 100")
	}
	if !dc.isStillInRound(100) {
		test.Error("Should still be in round 100")
	}

}
