package log

import (
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func TestLatencyMessages(t *testing.T) {

	latencyTests := &LatencyTests{}

	now := time.Now()
	newLatTest := &LatencyTestToSend{
		createdAt: now.Add(-10 * time.Second),
	}
	latencyTests.LatencyTestsToSend = append(latencyTests.LatencyTestsToSend, newLatTest)
	newLatTest = &LatencyTestToSend{
		createdAt: now.Add(-1 * time.Second),
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

	actionFunction := func(roundRec int32, roundDiff int32, timeDiff int64) {
		fmt.Println("Latency is", timeDiff, "received on round", roundRec, "=> round diff is", roundDiff)
	}
	receptionRoundID := int32(20)
	DecodeLatencyMessages(bytes, clientID, receptionRoundID, actionFunction)
}
