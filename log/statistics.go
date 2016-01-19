package log

import (
	"fmt"
	"time"
	"math"
)

const MAX_LATENCY_STORED = 100

type Statistics struct {
	begin			time.Time
	nextReport		time.Time
	nReports		int
	maxNReports		int
	period			time.Duration

	latencies				[]int64

	totalUpstreamCells		int64
	totalUpstreamBytes 		int64

	totalDownstreamCells 	int64
	totalDownstreamBytes 	int64

	instantUpstreamCells	int64
	instantUpstreamBytes 	int64
	instantDownstreamBytes	int64

	totalDownstreamUDPCells 	int64
	totalDownstreamUDPBytes 	int64
	instantDownstreamUDPBytes 	int64
}

func EmptyStatistics(reportingLimit int) *Statistics{
	stats := Statistics{time.Now(), time.Now(), 0, reportingLimit, time.Duration(5)*time.Second, make([]int64, 0), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	return &stats
}

func (stats *Statistics) ReportingDone() bool {
	if stats.maxNReports == 0 || stats.maxNReports == -1 {
		return false
	}
	return stats.nReports >= stats.maxNReports
}

func (stats *Statistics) Dump() {
	fmt.Println("Dumping Statistics...")
	fmt.Println("begin", stats.begin)
	fmt.Println("nextReport", stats.nextReport)
	fmt.Println("nReports", stats.nReports)
	fmt.Println("maxNReports", stats.maxNReports)
	fmt.Println("period", stats.period)

	fmt.Println(stats.totalUpstreamCells)
	fmt.Println(stats.totalUpstreamBytes)
	fmt.Println(stats.totalDownstreamCells)
	fmt.Println(stats.totalDownstreamBytes)
	fmt.Println(stats.totalDownstreamUDPCells)
	fmt.Println(stats.totalDownstreamUDPBytes)
	fmt.Println(stats.instantUpstreamCells)
	fmt.Println(stats.instantUpstreamBytes)
	fmt.Println(stats.instantDownstreamBytes)
	fmt.Println(stats.instantDownstreamUDPBytes)
}

func round(f float64) float64 {
    return math.Floor(f + .5)
}

func round2(f float64, places int) float64 {
    shift := math.Pow(10, float64(places))
    return round(f * shift) / shift;    
}

func mean(data []int64) float64 {
	sum := int64(0)
	for i:=0; i<len(data); i++ {
		sum += data[i]
	}

	mean := float64(sum) / float64(len(data))
	return mean
}

func mean2(data []float64) float64 {
	sum := float64(0)
	for i:=0; i<len(data); i++ {
		sum += data[i]
	}

	mean := float64(sum) / float64(len(data))
	return mean
}

func confidence(data []int64) float64 {

	if len(data) == 0{
		return 0
	}
	mean_val := mean(data)

	deviations := make([]float64, 0)
	for i:=0; i<len(data); i++ {
		diff := mean_val - float64(data[i])
		deviations = append(deviations, diff*diff)
	}

	std := mean2(deviations)
	stderr := math.Sqrt(std)
	z_value_95 := 1.96
	margin_error := stderr * z_value_95

	return margin_error
}

func (stats *Statistics) LatencyStatistics() (string, string, string) {

	if len(stats.latencies) == 0{
		return "-1", "-1", "-1"
	}

	m := round2(mean(stats.latencies), 2)
	v := round2(confidence(stats.latencies), 2)

	return fmt.Sprintf("%v", m), fmt.Sprintf("%v", v), fmt.Sprintf("%v", len(stats.latencies))
}

func (stats *Statistics) AddLatency(latency int64) {
	stats.latencies = append(stats.latencies, latency)

	//we remove the first items
	if len(stats.latencies) > MAX_LATENCY_STORED {
		start := len(stats.latencies) - MAX_LATENCY_STORED
		stats.latencies = stats.latencies[start:]
	}
}

func (stats *Statistics) AddDownstreamCell(nBytes int64) {
	stats.totalDownstreamCells += 1
	stats.totalDownstreamBytes += nBytes
	stats.instantDownstreamBytes += nBytes
}

func (stats *Statistics) AddDownstreamUDPCell(nBytes int64) {
	stats.totalDownstreamUDPCells += 1
	stats.totalDownstreamUDPBytes += nBytes
	stats.instantDownstreamUDPBytes += nBytes
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
		latm, latv, latn := stats.LatencyStatistics()

		Printf(EXPERIMENT_OUTPUT, "@ %fs; cell %f (%f) /sec, up %f (%f) B/s, down %f (%f) B/s, udp down %f (%f) B/s, lat %s += %s over %s",
			duration,
			 float64(stats.totalUpstreamCells)/duration, float64(stats.instantUpstreamCells)/stats.period.Seconds(),
			 float64(stats.totalUpstreamBytes)/duration, instantUpSpeed,
			 float64(stats.totalDownstreamBytes)/duration, float64(stats.instantDownstreamBytes)/stats.period.Seconds(),
			 float64(stats.totalDownstreamUDPBytes)/duration, float64(stats.instantDownstreamUDPBytes)/stats.period.Seconds(),
			 latm, latv, latn)

		// Next report time
		stats.instantUpstreamCells = 0
		stats.instantUpstreamBytes = 0
		stats.instantDownstreamBytes = 0
		stats.instantDownstreamUDPBytes = 0

		//prifilog.BenchmarkFloat(fmt.Sprintf("cellsize-%d-upstream-bytes", payloadLength), instantUpSpeed)

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
		JsonDump(data)

		stats.nextReport = now.Add(stats.period)
		stats.nReports += 1
	}
}

func (stats *Statistics) Report() {
	stats.ReportWithInfo("")
}


func (stats *Statistics) ReportWithInfo(info string) {
	now := time.Now()
	if now.After(stats.nextReport) {
		duration := now.Sub(stats.begin).Seconds()
		instantUpSpeed := (float64(stats.instantUpstreamBytes)/stats.period.Seconds())
		latm, latv, latn := stats.LatencyStatistics()

		Printf(EXPERIMENT_OUTPUT, "@ %fs; cell %f (%f) /sec, up %f (%f) B/s, down %f (%f) B/s, udp down %f (%f) B/s, lat %s += %s over %s "+info,
			duration,
			 float64(stats.totalUpstreamCells)/duration, float64(stats.instantUpstreamCells)/stats.period.Seconds(),
			 float64(stats.totalUpstreamBytes)/duration, instantUpSpeed,
			 float64(stats.totalDownstreamBytes)/duration, float64(stats.instantDownstreamBytes)/stats.period.Seconds(),
			 float64(stats.totalDownstreamUDPBytes)/duration, float64(stats.instantDownstreamUDPBytes)/stats.period.Seconds(),
			 latm, latv, latn)

		// Next report time
		stats.instantUpstreamCells = 0
		stats.instantUpstreamBytes = 0
		stats.instantDownstreamBytes = 0
		stats.instantDownstreamUDPBytes = 0

		stats.nextReport = now.Add(stats.period)
		stats.nReports += 1
	}
}