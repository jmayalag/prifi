package datasources

import (
	"golang.org/x/tools/go/gcimporter15/testdata"
	"sync"
	"time"
)

type DataSource interface {
	HasData() bool

	GetDataFromSource() (int, []byte)

	AckDataToSource(int)

	SendDataToSource([]byte)
}

// One buffered latency test message. We only need to store the "createdAt" time.
type LatencyTestToSend struct {
	CreatedAt time.Time
}

//The DataSource structure
type DataSourcePings struct {
	sync.Mutex
	LatencyTestsInterval time.Duration
	LatencyTestsToSend   []*LatencyTestToSend //lock the mutex before accessing this
}

func NewDataSourcePings(interval time.Duration) *DataSourcePings {

	dsp := &DataSourcePings{
		LatencyTestsInterval: interval,
		LatencyTestsToSend:   make([]*LatencyTestToSend, 0),
	}

	go latencyMsgGenerator(interval)

	return dsp
}

// periodically generates latency test messages
func (dsp *DataSourcePings) latencyMsgGenerator(interval time.Duration) {
	for {
		time.Sleep(interval)

		dsp.Lock()
		// create a new latency test message
		newLatTest := &LatencyTestToSend{
			CreatedAt: time.Now(),
		}
		dsp.LatencyTestsToSend = append(dsp.LatencyTestsToSend, newLatTest)
		dsp.Unlock()
	}
}

//
func (dsp *DataSourcePings) HasData() bool {
	dsp.Lock()
	defer dsp.Unlock()

	return (len(dsp.LatencyTestsToSend) != 0)
}

func (dsp *DataSourcePings) GetDataFromSource() (int, []byte) {
	dsp.Lock()
	defer dsp.Unlock()

}

func (dsp *DataSourcePings) AckDataToSource(int) {
	//no ACK for pings
}

func (dsp *DataSourcePings) SendDataToSource([]byte) {

}
