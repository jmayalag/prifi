package log

import (
	"fmt"
	"github.com/dedis/cothority/log"
	"time"
)

//TODO : this file is so dirty it belongs to /r/programminghorror. I'm ashamed of having written this.

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
}

func NewBitRateStatistics() *BitrateStatistics {
	fiveSec := time.Duration(5) * time.Second
	now := time.Now()
	stats := BitrateStatistics{
		begin:      now,
		nextReport: now,
		period:     fiveSec}
	return &stats
}

func (stats *BitrateStatistics) Dump() {
	log.Lvlf1("%+v\n", stats)
}

func (stats *BitrateStatistics) AddDownstreamCell(nBytes int64) {
	stats.totalDownstreamCells += 1
	stats.totalDownstreamBytes += nBytes
	stats.instantDownstreamBytes += nBytes
}

func (stats *BitrateStatistics) AddDownstreamUDPCell(nBytes int64, nclients int) {
	stats.totalDownstreamUDPCells += 1
	stats.totalDownstreamUDPBytes += nBytes
	stats.instantDownstreamRetransmitBytes += nBytes

	stats.totalDownstreamUDPBytesTimesClients += (nBytes * int64(nclients))
	stats.instantDownstreamUDPBytesTimesClients += (nBytes * int64(nclients))
}

func (stats *BitrateStatistics) AddDownstreamRetransmitCell(nBytes int64) {
	stats.totalDownstreamRetransmitCells += 1
	stats.totalDownstreamRetransmitBytes += nBytes
	stats.instantDownstreamUDPBytes += nBytes
}

func (stats *BitrateStatistics) AddUpstreamCell(nBytes int64) {
	stats.totalUpstreamCells += 1
	stats.totalUpstreamBytes += nBytes
	stats.instantUpstreamCells += 1
	stats.instantUpstreamBytes += nBytes
}

func (stats *BitrateStatistics) Report() {
	stats.ReportWithInfo("")
}

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

		log.Lvlf1("%0.1f round/sec, %0.1f kB/s up, %0.1f kB/s down, %0.1f kB/s down(udp)",
			float64(stats.instantUpstreamCells)/stats.period.Seconds(),
			float64(stats.instantUpstreamBytes)/1024/stats.period.Seconds(),
			float64(stats.instantDownstreamBytes)/1024/stats.period.Seconds(),
			float64(stats.instantDownstreamUDPBytes)/1024/stats.period.Seconds())

		data := fmt.Sprintf("round=%0.1f&up=%0.1f&down=%0.1f&udp_down%0.1f&info=%s",
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
	}
}
