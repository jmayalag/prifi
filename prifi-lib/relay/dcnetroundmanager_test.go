package relay

import (
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

	//requesting the next downstream round to send should not return an open round
	if dcmr.NextDownStreamRoundToSent() != 1 {
		test.Error("NextDownStreamRoundToSent should be equal to 1", dcmr.NextDownStreamRoundToSent())
	}
	//but should still return the same number
	if dcmr.NextDownStreamRoundToSent() != 1 {
		test.Error("NextDownStreamRoundToSent should still be equal to 1", dcmr.NextDownStreamRoundToSent())
	}

	//opening another round should not change current round
	dcmr.OpenRound(1)
	if dcmr.CurrentRound() != 0 {
		test.Error("Should be in round 0")
	}

	//requesting the next downstream round to send should not return an open round
	if dcmr.NextDownStreamRoundToSent() != 2 {
		test.Error("NextDownStreamRoundToSent should be equal to 2", dcmr.NextDownStreamRoundToSent())
	}

	//setting a round to closed should skip it
	s := make(map[int32]bool, 1)
	s[2] = false
	dcmr.SetStoredRoundSchedule(s)
	if dcmr.storedRoundsSchedule == nil || len(dcmr.storedRoundsSchedule) != 1 || dcmr.storedRoundsSchedule[0] != s[0] {
		test.Error("dcmr.storedRoundsSchedule should be s")
	}
	if dcmr.NextDownStreamRoundToSent() != 3 {
		test.Error("NextDownStreamRoundToSent should be equal to 3", dcmr.NextDownStreamRoundToSent())
	}

	//should be able to open a round while skipping another round
	dcmr.OpenRound(3)
	if dcmr.CurrentRound() != 0 {
		test.Error("Should be in round 0")
	}
	dcmr.CloseRound(0)
	if dcmr.CurrentRound() != 1 {
		test.Error("Should be in round 1")
	}
	dcmr.CloseRound(1)
	if dcmr.CurrentRound() != 3 {
		test.Error("Should be in round 3", dcmr.CurrentRound())
	}

	_ = data

}
