package log

import (
	"fmt"
	"time"

	"github.com/dedis/cothority/log"
)

//This class hold latencies values, and performs the average/std distribution of it. That is the max number of value stored.
const MAX_LATENCY_STORED = 100

//LatencyStatistics holds the latencies reported
type LatencyStatistics struct {
	begin      time.Time
	nextReport time.Time
	period     time.Duration

	latencies []int64
}

//NewLatencyStatistics create a new LatencyStatistics struct, with a period (for reporting) of 5 second
func NewLatencyStatistics() *LatencyStatistics {
	fiveSec := time.Duration(5) * time.Second
	now := time.Now()
	stats := LatencyStatistics{
		begin:      now,
		nextReport: now,
		period:     fiveSec,
		latencies:  make([]int64, 0)}
	return &stats
}

//LatencyStatistics returns a triplet (mean, variance, number of samples) as formatted strings (2-digit precision)
func (stats *LatencyStatistics) LatencyStatistics() (string, string, string) {

	if len(stats.latencies) == 0 {
		return "-1", "-1", "-1"
	}

	m := RoundWithPrecision(MeanInt64(stats.latencies), 2)
	v := RoundWithPrecision(Confidence95Percentiles(stats.latencies), 2)

	return fmt.Sprintf("%v", m), fmt.Sprintf("%v", v), fmt.Sprintf("%v", len(stats.latencies))
}

//AddLatency adds a latency to the stored latency array, and removes the oldest one if there are more than MAX_LATENCY_STORED
func (stats *LatencyStatistics) AddLatency(latency int64) {
	stats.latencies = append(stats.latencies, latency)

	//we remove the first items
	if len(stats.latencies) > MAX_LATENCY_STORED {
		start := len(stats.latencies) - MAX_LATENCY_STORED
		stats.latencies = stats.latencies[start:]
	}
}

//Report prints (if t>period=5 seconds have passed since the last report) all the information, without extra data
func (stats *LatencyStatistics) Report() {
	stats.ReportWithInfo("")
}

//ReportWithInfo prints (if t>period=5 seconds have passed since the last report) all the information, without extra data
func (stats *LatencyStatistics) ReportWithInfo(info string) {
	now := time.Now()
	if now.After(stats.nextReport) {

		mean, variance, n := stats.LatencyStatistics()

		log.Lvlf1("Measured latency : %s +- %s (over %s)", mean, variance, n)

		data := fmt.Sprintf("mean=%s&var=%s&n=%s&info=%s", mean, variance, n, info)

		go performGETRequest("http://lbarman.ch/prifi/?" + data)

		stats.nextReport = now.Add(stats.period)
	}
}
