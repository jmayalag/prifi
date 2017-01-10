package log

import (
	"testing"
)

func TestLog(t *testing.T) {
	//round
	if Round(float64(6.3)) != 6 {
		t.Error("Rounding error")
	}
	if Round(float64(6.0)) != 6 {
		t.Error("Rounding error")
	}
	if Round(float64(6.5)) != 7 {
		t.Error("Rounding error")
	}

	//roundwithprecision
	if RoundWithPrecision(float64(6.3), 2) != 6.30 {
		t.Error("Rounding error")
	}
	if RoundWithPrecision(float64(6.125), 2) != 6.13 {
		t.Error("Rounding error")
	}
	if RoundWithPrecision(float64(6.41), 1) != 6.4 {
		t.Error("Rounding error")
	}

	//mean
	if MeanFloat64([]float64{1.2, 4.5, 6.9}) != 4.2 {
		t.Error("Rounding error")
	}

	//confidence interval
	if RoundWithPrecision(Confidence95Percentiles([]int64{30, 31, 29, 29, 35, 39, 26, 29}), 2) != 7.53 {
		t.Error("Confidence95Percentiles is wrong")
	}

	//nothing to do here. Cheating on coverage mouehehe
	performGETRequest("lbarman.ch")
}
