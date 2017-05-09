package log

import (
	"encoding/binary"
	"time"
)

const pattern uint16 = uint16(43690) //1010101010101010

// Regroups the information about doing latency tests
type LatencyTests struct {
	DoLatencyTests       bool
	LatencyTestsInterval time.Duration
	NextLatencyTest      time.Time
	LatencyTestsToSend   []*LatencyTestToSend
}

// One buffered latency test message. We only need to store the "createdAt" time.
type LatencyTestToSend struct {
	createdAt time.Time
}

func genLatencyMessagePayload(creationTime time.Time, clientID, roundID int) []byte {
	latencyMsgBytes := make([]byte, 14)
	currTime := MsTimeStamp(creationTime) //timestamp in Ms
	binary.BigEndian.PutUint16(latencyMsgBytes[0:2], uint16(clientID))
	binary.BigEndian.PutUint32(latencyMsgBytes[2:6], uint32(roundID))
	binary.BigEndian.PutUint64(latencyMsgBytes[6:14], uint64(currTime))
	return latencyMsgBytes
}


// LatencyMessagesToBytes encoded the Latency messages in "msgs", returns the encoded bytes and the new "msgs" without the successfully-encoded messages
func LatencyMessagesToBytes(msgs []*LatencyTestToSend, clientID int, roundID int, payLoadLength int, reportFunction func(int64)) ([]byte, []*LatencyTestToSend) {
	if len(msgs) == 0 {
		return make([]byte, 0), msgs
	}

	if payLoadLength < 18 {
		panic("Trying to do a Latency test, but payload is smaller than 18 bytes.")
	}

	// [0:2] PATTERN
	// [2:4] Number of messages
	// [4:6] ClientID
	// [6:10] RoundID
	// [10:18] Time of sending

	buffer := make([]byte, payLoadLength)
	binary.BigEndian.PutUint16(buffer[0:2], pattern)
	//later, put number in [2;4]
	posInBuffer := 4
	latencyMsgLength := 14 // 2 + 4 + 8

	//pack all the latency messages we can in one
	numberOfMessagesPacked := uint16(0)
	for len(msgs) > 0 && posInBuffer+latencyMsgLength <= payLoadLength {

		//encode the first message
		b := genLatencyMessagePayload(msgs[0].createdAt, clientID, roundID)

		//save bytes in global buffer
		copy(buffer[posInBuffer:], b)

		//this is used to compute the "time stayed in buffer"
		reportFunction(MsTimeStampNow() - MsTimeStamp(msgs[0].createdAt))

		//pop the stack
		if len(msgs) == 1 {
			msgs = make([]*LatencyTestToSend, 0)
		} else {
			msgs = msgs[1:]
		}


		//we just packed one extra message
		numberOfMessagesPacked += 1
		posInBuffer += latencyMsgLength
	}
	binary.BigEndian.PutUint16(buffer[2:4], numberOfMessagesPacked)

	return buffer, msgs
}