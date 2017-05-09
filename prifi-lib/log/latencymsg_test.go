package log

import (
	"testing"
	"time"
	"fmt"
	"encoding/hex"
)

func TestLatencyMessages(t *testing.T) {

	latencyTests := &LatencyTests{	}

	now := time.Now()
	newLatTest := &LatencyTestToSend{
		createdAt: now,
	}
	latencyTests.LatencyTestsToSend = append(latencyTests.LatencyTestsToSend, newLatTest)
	newLatTest = &LatencyTestToSend{
		createdAt: now.Add(10*time.Second),
	}
	latencyTests.LatencyTestsToSend = append(latencyTests.LatencyTestsToSend, newLatTest)

	clientID := 2
	roundID := 4
	payloadLength := 100
	logFn := func(timeDiff int64) {
		fmt.Println(timeDiff)
	}
	bytes, outMsgs := LatencyMessagesToBytes(latencyTests.LatencyTestsToSend, clientID, roundID, payloadLength, logFn)
	latencyTests.LatencyTestsToSend = outMsgs

	fmt.Println(hex.Dump(bytes))
}