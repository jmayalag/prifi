package log

import (
	"math"
	"net/http"
)

func Round(f float64) float64 {
	return math.Floor(f + .5)
}

func RoundWithPrecision(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return Round(f*shift) / shift
}

func MeanInt64(data []int64) float64 {
	sum := int64(0)
	for i := 0; i < len(data); i++ {
		sum += data[i]
	}

	mean := float64(sum) / float64(len(data))
	return mean
}

func MeanFloat64(data []float64) float64 {
	sum := float64(0)
	for i := 0; i < len(data); i++ {
		sum += data[i]
	}

	mean := float64(sum) / float64(len(data))
	return mean
}

func Confidence95Percentiles(data []int64) float64 {

	if len(data) == 0 {
		return 0
	}
	mean_val := MeanInt64(data)

	var deviations float64
	for i := 0; i < len(data); i++ {
		diff := mean_val - float64(data[i])
		deviations = append(deviations, diff*diff)
	}

	std := MeanFloat64(deviations)
	stderr := math.Sqrt(std)
	z_value_95 := 1.96
	margin_error := stderr * z_value_95

	return margin_error
}

func performGETRequest(url string) {
	_, _ = http.Get(url)
}
