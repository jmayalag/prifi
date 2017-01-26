package log

import (
	"fmt"
	"time"

	"gopkg.in/dedis/onet.v1/log"
)

//BitrateStatistics holds statistics about the bitrate, such as instant/total up/down/down (via udp)/retransmitted bits
type BitrateStatistics struct {
	begin      time.Time
	nextReport time.Time
	period     time.Duration

	totalUpstreamCells int64
	totalUpstreamBytes int64

	totalDownstreamCells int64
	totalDownstreamBytes int64

	instantUpstreamCells   int64
	instantUpstreamBytes   int64
	instantDownstreamBytes int64

	totalDownstreamUDPCells               int64
	totalDownstreamUDPBytes               int64
	instantDownstreamUDPBytes             int64
	totalDownstreamUDPBytesTimesClients   int64
	instantDownstreamUDPBytesTimesClients int64

	totalDownstreamRetransmitCells   int64
	totalDownstreamRetransmitBytes   int64
	instantDownstreamRetransmitBytes int64

	reportNo int
}

//NewBitRateStatistics create a new BitrateStatistics struct, with a period (for reporting) of 5 second
func NewBitRateStatistics() *BitrateStatistics {
	fiveSec := time.Duration(5) * time.Second
	now := time.Now()
	stats := BitrateStatistics{
		begin:      now,
		nextReport: now,
		reportNo:   0,
		period:     fiveSec}
	return &stats
}

// Dump prints all the contents of the BitrateStatistics
func (stats *BitrateStatistics) Dump() {
	log.Lvlf1("%+v\n", stats)
}

//AddDownstreamCell adds N bytes to the count of downstream bits
func (stats *BitrateStatistics) AddDownstreamCell(nBytes int64) {
	stats.totalDownstreamCells++
	stats.totalDownstreamBytes += nBytes
	stats.instantDownstreamBytes += nBytes
}

//AddDownstreamUDPCell adds N bytes to the count of downstream (via udp) bits
func (stats *BitrateStatistics) AddDownstreamUDPCell(nBytes int64, nclients int) {
	stats.totalDownstreamUDPCells++
	stats.totalDownstreamUDPBytes += nBytes
	stats.instantDownstreamRetransmitBytes += nBytes

	stats.totalDownstreamUDPBytesTimesClients += (nBytes * int64(nclients))
	stats.instantDownstreamUDPBytesTimesClients += (nBytes * int64(nclients))
}

//AddDownstreamRetransmitCell adds N bytes to the count of retransmitted bits
func (stats *BitrateStatistics) AddDownstreamRetransmitCell(nBytes int64) {
	stats.totalDownstreamRetransmitCells++
	stats.totalDownstreamRetransmitBytes += nBytes
	stats.instantDownstreamUDPBytes += nBytes
}

//AddUpstreamCell adds N bytes to the count of upstream bits
func (stats *BitrateStatistics) AddUpstreamCell(nBytes int64) {
	stats.totalUpstreamCells++
	stats.totalUpstreamBytes += nBytes
	stats.instantUpstreamCells++
	stats.instantUpstreamBytes += nBytes
}

//Report prints (if t>period=5 seconds have passed since the last report) all the information, without extra data
func (stats *BitrateStatistics) Report() {
	stats.ReportWithInfo("")
}

//ReportWithInfo prints (if t>period=5 seconds have passed since the last report) all the information, with extra data "info"
func (stats *BitrateStatistics) ReportWithInfo(info string) {
	now := time.Now()
	if now.After(stats.nextReport) {
		//percentage of retransmitted packet is not supported yet
		/*
			instantRetransmitPercentage := float64(0)
			if stats.instantDownstreamRetransmitBytes + stats.totalDownstreamUDPBytes != 0 {
				instantRetransmitPercentage = float64(100 * stats.instantDownstreamRetransmitBytes)/float64(stats.instantDownstreamUDPBytesTimesClients)
			}

			totalRetransmitPercentage := float64(0)
			if stats.instantDownstreamRetransmitBytes + stats.totalDownstreamUDPBytes != 0 {
				totalRetransmitPercentage = float64(100 * stats.totalDownstreamRetransmitBytes)/float64(stats.totalDownstreamUDPBytesTimesClients)
			}
		*/

		log.Lvlf1("[%v] %0.1f round/sec, %0.1f kB/s up, %0.1f kB/s down, %0.1f kB/s down(udp)",
			stats.reportNo,
			float64(stats.instantUpstreamCells)/stats.period.Seconds(),
			float64(stats.instantUpstreamBytes)/1024/stats.period.Seconds(),
			float64(stats.instantDownstreamBytes)/1024/stats.period.Seconds(),
			float64(stats.instantDownstreamUDPBytes)/1024/stats.period.Seconds())

		data := fmt.Sprintf("no=%v&round=%0.1f&up=%0.1f&down=%0.1f&udp_down%0.1f&info=%s",
			stats.reportNo,
			float64(stats.instantUpstreamCells)/stats.period.Seconds(),
			float64(stats.instantUpstreamBytes)/1024/stats.period.Seconds(),
			float64(stats.instantDownstreamBytes)/1024/stats.period.Seconds(),
			float64(stats.instantDownstreamUDPBytes)/1024/stats.period.Seconds(),
			info)

		go performGETRequest("http://lbarman.ch/prifi/?" + data)

		// Next report time
		stats.instantUpstreamCells = 0
		stats.instantUpstreamBytes = 0
		stats.instantDownstreamBytes = 0
		stats.instantDownstreamUDPBytes = 0
		stats.instantDownstreamRetransmitBytes = 0
		stats.instantDownstreamUDPBytesTimesClients = 0

		stats.nextReport = now.Add(stats.period)
		stats.reportNo++
	}
}
