package log

import (
	"fmt"
	"github.com/dedis/cothority/log"
	"time"
)

const MAX_LATENCY_STORED = 100

//TODO : this file is so dirty it belongs to /r/programminghorror. I'm ashamed of having written this.

type LatencyStatistics struct {
	begin      time.Time
	nextReport time.Time
	period     time.Duration

	latencies []int64
}

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

func (stats *LatencyStatistics) LatencyStatistics() (string, string, string) {

	if len(stats.latencies) == 0 {
		return "-1", "-1", "-1"
	}

	m := RoundWithPrecision(MeanInt64(stats.latencies), 2)
	v := RoundWithPrecision(Confidence95Percentiles(stats.latencies), 2)

	return fmt.Sprintf("%v", m), fmt.Sprintf("%v", v), fmt.Sprintf("%v", len(stats.latencies))
}

func (stats *LatencyStatistics) AddLatency(latency int64) {
	stats.latencies = append(stats.latencies, latency)

	//we remove the first items
	if len(stats.latencies) > MAX_LATENCY_STORED {
		start := len(stats.latencies) - MAX_LATENCY_STORED
		stats.latencies = stats.latencies[start:]
	}
}

func (stats *LatencyStatistics) Report() {
	stats.ReportWithInfo("")
}

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
