package stream_multiplexer

import (
	"testing"
	"fmt"
	"os"
	"net"
	"strconv"
	"time"
	"bytes"
	"math"
	"encoding/binary"
)

// Tests that the multiplexer produces messages of at most "payloadLength"
func TestIngressSizes(t *testing.T) {

	port := 3000
	payloadLength := 20
	upstreamChan := make(chan []byte)
	downstreamChan := make(chan []byte)
	stopChan := make(chan bool)

	go StartIngressServer(port, payloadLength, upstreamChan, downstreamChan, stopChan)

	time.Sleep(2*time.Second)

	conn1, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(3000))
	if err != nil {
		fmt.Println("Could not connect client", err)
		os.Exit(1)
	}

	// c1 sends "test"
	longData := make([]byte, 10005)
	conn1.Write(longData)

	expectedNumberOfMessages := int(math.Ceil(float64(len(longData)) / float64(payloadLength - MULTIPLEXER_HEADER_SIZE)))
	lastPlaintextSize := int(math.Mod(float64(len(longData)), float64(payloadLength - MULTIPLEXER_HEADER_SIZE)))
	lastMessageSize := lastPlaintextSize + MULTIPLEXER_HEADER_SIZE

	for i:=0; i< expectedNumberOfMessages-1; i++ {
		select {
		case data := <-upstreamChan:
			if len(data) != payloadLength {
				t.Error("Expected multiplexed data of length " + strconv.Itoa(payloadLength) +
					" for message "+strconv.Itoa(i)+", but instead got " + strconv.Itoa(len(data)))
			}

		case <-time.After(1 * time.Second):
			t.Error("No data written on the upstreamchannel")
		}
	}

	select {
	case data := <-upstreamChan:
		if len(data) != lastMessageSize {
			t.Error("Expected multiplexed data of length " + strconv.Itoa(lastMessageSize) +
				" for last message, but instead got " + strconv.Itoa(len(data)))
		}

	case <-time.After(1 * time.Second):
		t.Error("No data written on the upstreamchannel")
	}

	stopChan <- true
	time.Sleep(2*time.Second)
}

// First test: two different connections send interleaved messages.
// Checks that all messages are multiplexed, with the correct IDs
func TestUpstreamIngressMultiplexer(t *testing.T) {

	port := 3000
	payloadLength := 20
	upstreamChan := make(chan []byte)
	downstreamChan := make(chan []byte)
	stopChan := make(chan bool, 1)

	go StartIngressServer(port, payloadLength, upstreamChan, downstreamChan, stopChan)

	time.Sleep(2 * time.Second)

	conn1, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(3000))
	if err != nil {
		fmt.Println("Could not connect client", err)
		os.Exit(1)
	}

	// c1 sends "test"
	conn1.Write([]byte("test"))
	var id_conn1_bytes []byte
	select {
	case data := <-upstreamChan:
		if !bytes.Equal([]byte("test"), data[MULTIPLEXER_HEADER_SIZE:]) {
			t.Error("Data not recovered")
		}
		id_conn1_bytes = data[0:4]

	case <-time.After(1 * time.Second):
		t.Error("No data written on the upstreamchannel")
	}

	// c1 sends "ninja"
	conn1.Write([]byte("ninja"))
	select {
	case data := <-upstreamChan:
		if !bytes.Equal([]byte("ninja"), data[MULTIPLEXER_HEADER_SIZE:]) {
			t.Error("Data not recovered")
		}
		if !bytes.Equal(id_conn1_bytes, data[0:4]) {
			t.Error("Data on the same stream gets different IDs")
		}

	case <-time.After(1 * time.Second):
		t.Error("No data written on the upstreamchannel")
	}

	conn2, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(3000))
	if err != nil {
		fmt.Println("Could not connect client", err)
		os.Exit(1)
	}

	// c2 sends "connexion2"
	conn2.Write([]byte("connexion2"))
	var id_conn2_bytes []byte
	select {
	case data := <-upstreamChan:
		if !bytes.Equal([]byte("connexion2"), data[MULTIPLEXER_HEADER_SIZE:]) {
			t.Error("Data not recovered")
		}
		id_conn2_bytes = data[0:4]

	case <-time.After(1 * time.Second):
		t.Error("No data written on the upstreamchannel")
	}

	// c2 sends "ninja2"
	conn2.Write([]byte("ninja2"))
	select {
	case data := <-upstreamChan:
		if !bytes.Equal([]byte("ninja2"), data[MULTIPLEXER_HEADER_SIZE:]) {
			t.Error("Data not recovered")
		}
		if !bytes.Equal(id_conn2_bytes, data[0:4]) {
			t.Error("Data on the same stream gets different IDs")
		}

	case <-time.After(1 * time.Second):
		t.Error("No data written on the upstreamchannel")
	}

	// c1 sends "newdata"
	conn1.Write([]byte("newdata"))
	select {
	case data := <-upstreamChan:
		if !bytes.Equal([]byte("newdata"), data[MULTIPLEXER_HEADER_SIZE:]) {
			t.Error("Data not recovered")
		}
		if !bytes.Equal(id_conn1_bytes, data[0:4]) {
			t.Error("Data on the same stream gets different IDs")
		}

	case <-time.After(1 * time.Second):
		t.Error("No data written on the upstreamchannel")
	}

	stopChan <- true
	time.Sleep(2*time.Second)
}



// First test: two different connections receive interleaved messages.
// Checks that all messages are multiplexed, with the correct IDs
func TestDownstreamIngressMultiplexer(t *testing.T) {

	port := 3000
	payloadLength := 20
	upstreamChan := make(chan []byte)
	downstreamChan := make(chan []byte)
	stopChan := make(chan bool, 1)

	go StartIngressServer(port, payloadLength, upstreamChan, downstreamChan, stopChan)

	time.Sleep(2*time.Second)

	conn1, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(3000))
	if err != nil {
		fmt.Println("Could not connect client", err)
		os.Exit(1)
	}

	// c1 sends "test"
	conn1.Write([]byte("test"))
	var id_conn1_bytes []byte
	select {
	case data := <-upstreamChan:
		if !bytes.Equal([]byte("test"), data[MULTIPLEXER_HEADER_SIZE:]) {
			t.Error("Data not recovered")
		}
		id_conn1_bytes = data[0:4]

	case <-time.After(1 * time.Second):
		t.Error("No data written on the upstreamchannel")
	}

	conn2, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(3000))
	if err != nil {
		fmt.Println("Could not connect client", err)
		os.Exit(1)
	}

	// c2 sends "connexion2"
	conn2.Write([]byte("connexion2"))
	var id_conn2_bytes []byte
	select {
	case data := <-upstreamChan:
		if !bytes.Equal([]byte("connexion2"), data[MULTIPLEXER_HEADER_SIZE:]) {
			t.Error("Data not recovered")
		}
		id_conn2_bytes = data[0:4]

	case <-time.After(1 * time.Second):
		t.Error("No data written on the upstreamchannel")
	}

	// now tests receiving messages (for c1)

	payload := []byte("hello")
	messageForC1 := make([]byte, MULTIPLEXER_HEADER_SIZE + len(payload))
	copy(messageForC1[:4], id_conn1_bytes[:])
	binary.BigEndian.PutUint32(messageForC1[4:8], uint32(len(payload)))
	copy(messageForC1[MULTIPLEXER_HEADER_SIZE:], payload)
	downstreamChan <- messageForC1

	conn1.SetDeadline(time.Now().Add(time.Second))
	messageRead := make([]byte, 2*len(payload)) // just a bigger size
	n, err := conn1.Read(messageRead)

	if err != nil {
		t.Error("Ingress could not forward downstream message to reader connection, ", err)
	}
	if n != len(payload) {
		t.Error("Ingress read the wrong number of bytes, expected" + strconv.Itoa(len(payload))+", instead got "+
			strconv.Itoa(n))
	}
	if !bytes.Equal(messageRead[:n], payload){
		t.Error("Ingress read the wrong message, expected" + string(payload) + ", instead got "+
			string(messageRead[:n]))
	}
	//make sure Connection 2 did not receive anything !
	conn2.SetDeadline(time.Now().Add(time.Second))
	messageRead = make([]byte, 2*len(payload)) // just a bigger size
	n, err = conn2.Read(messageRead)

	if n != 0 {
		t.Error("Connection2 should not have received anything!")
	}

	// now tests receiving messages (for c2)

	payload = []byte("something longer than 20 characters")
	//fmt.Println("Payload:", payload)
	maxPayloadLength := payloadLength - MULTIPLEXER_HEADER_SIZE
	nMessages := int(math.Ceil(float64(len(payload)) / float64(maxPayloadLength)))

	plaintextsForC2 := make([][]byte, nMessages)

	for i:=0; i<nMessages; i++ {

		startPos := i*maxPayloadLength
		endPos := startPos + maxPayloadLength
		if endPos > len(payload) {
			endPos = len(payload)
		}

		plaintextsForC2[i] = make([]byte, endPos - startPos)
		copy(plaintextsForC2[i][:], payload[startPos:endPos])
		//fmt.Println("Produced plaintext message for round", i, "data", plaintextsForC2[i])
	}
	messagesForC2 := make([][]byte, nMessages)

	for i:=0; i<nMessages; i++ {
		messagesForC2[i] = make([]byte, payloadLength)
		copy(messagesForC2[i][:4], id_conn2_bytes[:MULTIPLEXER_HEADER_SIZE])
		binary.BigEndian.PutUint32(messagesForC2[i][4:8], uint32(len(plaintextsForC2[i])))
		copy(messagesForC2[i][MULTIPLEXER_HEADER_SIZE:], plaintextsForC2[i])
		//fmt.Println("Produced message", i, "bytes", messagesForC2[i])

		downstreamChan <- messagesForC2[i]
	}

	dataRead := make([]byte, 0)
	errorCount := 0
	maxError := 3

	for len(dataRead) < len(payload) {
		conn2.SetDeadline(time.Now().Add(time.Second))
		messageRead = make([]byte, len(payload)) // just a bigger size
		n, err = conn2.Read(messageRead)

		if err != nil {
			errorCount++
			if errorCount >= maxError {
				t.Error("Ingress could not forward downstream to reader connection, ", err)
			}
		} else {
			for i := 0; i < n; i++ {
				dataRead = append(dataRead, messageRead[i])
			}
		}
	}

	if len(dataRead) != len(payload) {
		t.Error("Ingress read the wrong number of bytes, expected " + strconv.Itoa(len(payload)) + ", instead got " +
			strconv.Itoa(len(dataRead)))
	}
	if !bytes.Equal(dataRead, payload) {
		t.Error("Ingress read the wrong data for message, expected " + string(payload) + ", instead got " +
			string(dataRead))
	}
	//make sure Connection 2 did not receive anything !
	conn1.SetDeadline(time.Now().Add(time.Second))
	messageRead = make([]byte, 2*len(payload)) // just a bigger size
	n, err = conn1.Read(messageRead)

	if n != 0 {
		t.Error("Connection2 should not have received anything!")
	}

	stopChan <- true
	time.Sleep(2*time.Second)
}