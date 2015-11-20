package util

import (
	"fmt"
	"time"
	log2 "github.com/lbarman/prifi/log"
)

type Statistics struct {
	begin			time.Time
	nextReport		time.Time
	nReports		int
	maxNReports		int
	period			time.Duration

	totalUpstreamCells		int64
	totalUpstreamBytes 		int64
	totalDownstreamCells 	int64
	totalDownstreamBytes 	int64
	instantUpstreamCells	int64
	instantUpstreamBytes 	int64
	instantDownstreamBytes	int64
}

func EmptyStatistics(reportingLimit int) *Statistics{
	stats := Statistics{time.Now(), time.Now(), 0, reportingLimit, time.Duration(3)*time.Second, 0, 0, 0, 0, 0, 0, 0}
	return &stats
}

func (stats *Statistics) ReportingDone() bool {
	return stats.nReports >= stats.maxNReports
}

func (stats *Statistics) AddDownstreamCell(nBytes int64) {
	stats.totalDownstreamCells += 1
	stats.totalDownstreamBytes += nBytes
	stats.instantDownstreamBytes += nBytes
}

func (stats *Statistics) AddUpstreamCell(nBytes int64) {
	stats.totalUpstreamCells += 1
	stats.totalUpstreamBytes += nBytes
	stats.instantUpstreamCells += 1
	stats.instantUpstreamBytes += nBytes
}

func (stats *Statistics) ReportJson() {
	now := time.Now()
	if now.After(stats.nextReport) {
		duration := now.Sub(stats.begin).Seconds()
		instantUpSpeed := (float64(stats.instantUpstreamBytes)/stats.period.Seconds())

		fmt.Printf("@ %fs; cell %f (%f) /sec, up %f (%f) B/s, down %f (%f) B/s\n",
			duration,
			 float64(stats.totalUpstreamCells)/duration, float64(stats.instantUpstreamCells)/stats.period.Seconds(),
			 float64(stats.totalUpstreamBytes)/duration, instantUpSpeed,
			 float64(stats.totalDownstreamBytes)/duration, float64(stats.instantDownstreamBytes)/stats.period.Seconds())

		// Next report time
		stats.instantUpstreamCells = 0
		stats.instantUpstreamBytes = 0
		stats.instantDownstreamBytes = 0

		//log2.BenchmarkFloat(fmt.Sprintf("cellsize-%d-upstream-bytes", payloadLength), instantUpSpeed)

		//write JSON
		data := struct {
		    Experiment string
		    CellSize int
		    Speed float64
		}{
		    "upstream-speed-given-cellsize",
		    42,//relayState.PayloadLength,
		    instantUpSpeed,
		}
		log2.JsonDump(data)

		stats.nextReport = now.Add(stats.period)
		stats.nReports += 1
	}
}

func (stats *Statistics) Report() {
	now := time.Now()
	if now.After(stats.nextReport) {
		duration := now.Sub(stats.begin).Seconds()
		instantUpSpeed := (float64(stats.instantUpstreamBytes)/stats.period.Seconds())

		fmt.Printf("@ %fs; cell %f (%f) /sec, up %f (%f) B/s, down %f (%f) B/s\n",
			duration,
			 float64(stats.totalUpstreamCells)/duration, float64(stats.instantUpstreamCells)/stats.period.Seconds(),
			 float64(stats.totalUpstreamBytes)/duration, instantUpSpeed,
			 float64(stats.totalDownstreamBytes)/duration, float64(stats.instantDownstreamBytes)/stats.period.Seconds())

		// Next report time
		stats.instantUpstreamCells = 0
		stats.instantUpstreamBytes = 0
		stats.instantDownstreamBytes = 0

		stats.nextReport = now.Add(stats.period)
		stats.nReports += 1
	}
}