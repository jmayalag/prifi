package log

import (
	"fmt"
	"time"

	"gopkg.in/dedis/onet.v2/log"
	"sort"
	"strconv"
)

//LatencyStatistics holds the latencies reported
type SchedulesStatistics struct {
	begin                      time.Time
	nextReport                 time.Time
	period                     time.Duration
	reportNo                   int
	scheduleLengthRepartitions map[int]int
}

//NewSchedulesStatistics create a new TimeStatistics struct, with a period (for reporting) of 5 second
func NewSchedulesStatistics() *SchedulesStatistics {
	fiveSec := time.Duration(5) * time.Second
	now := time.Now()
	stats := SchedulesStatistics{
		begin:                      now,
		nextReport:                 now,
		period:                     fiveSec,
		reportNo:                   0,
		scheduleLengthRepartitions: make(map[int]int)}
	return &stats
}

//AddLatency adds a latency to the stored latency array, and removes the oldest one if there are more than MAX_LATENCY_STORED
func (stats *SchedulesStatistics) AddSchedule(newSchedule map[int]bool) {
	scheduleLength := 0

	for _, v := range newSchedule {
		if v {
			scheduleLength++
		}
	}

	stats.scheduleLengthRepartitions[scheduleLength]++
}

//Report prints (if t>period=5 seconds have passed since the last report) all the information, without extra data
func (stats *SchedulesStatistics) Report() string {
	return stats.ReportWithInfo("")
}

//ReportWithInfo prints (if t>period=5 seconds have passed since the last report) all the information, with extra data
func (stats *SchedulesStatistics) ReportWithInfo(info string) string {
	now := time.Now()
	if now.After(stats.nextReport) {

		var keys []int
		for k := range stats.scheduleLengthRepartitions {
			keys = append(keys, k)
		}
		sort.Ints(keys)

		str := ""
		for _, k := range keys {
			str += strconv.Itoa(k) + "->" + strconv.Itoa(stats.scheduleLengthRepartitions[k]) + "; "
		}

		//human-readable output
		str2 := fmt.Sprintf("[%v] Schedules %s Info: %s", stats.reportNo, str, info)
		log.Lvl1(str2)

		stats.nextReport = now.Add(stats.period)
		stats.reportNo++

		return str2
	}
	return ""
}
