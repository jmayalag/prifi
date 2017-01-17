package log

import (
	"math"
	"net/http"
)

//Round rounds up a float64, without digits after the comma
func Round(f float64) float64 {
	return math.Floor(f + .5)
}

//RoundWithPrecision rounds up a float64, with a specified amount of digits after the comma
func RoundWithPrecision(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return Round(f*shift) / shift
}

//MeanInt64 returns the mean for a []int64
func MeanInt64(data []int64) float64 {
	sum := int64(0)
	for i := 0; i < len(data); i++ {
		sum += data[i]
	}

	mean := float64(sum) / float64(len(data))
	return mean
}

//MeanFloat64 returns the mean for a []float64
func MeanFloat64(data []float64) float64 {
	sum := float64(0)
	for i := 0; i < len(data); i++ {
		sum += data[i]
	}

	mean := float64(sum) / float64(len(data))
	return mean
}

//Confidence95Percentiles returns the confidence interval for 95 percentile
func Confidence95Percentiles(data []int64) float64 {
	if len(data) == 0 {
		return 0
	}
	mean_val := MeanInt64(data)

	var deviations []float64
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

//performGETRequest performs a GET request and ignores all errors
func performGETRequest(url string) error {
	_, err := http.Get(url)
	return err
}
