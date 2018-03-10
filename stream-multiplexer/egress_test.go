package stream_multiplexer

import (
	"testing"
	"net"
	"time"
	"bytes"
	"fmt"
	"sync"
	"encoding/binary"
)

func handleConnection(id int, conn net.Conn, expect []byte, t *testing.T, wg *sync.WaitGroup) {
	defer wg.Done()

	buffer := make([]byte, 0)
	errorCount := 0
	maxError := 3

	for len(buffer) < len(expect) {
		buffer2 := make([]byte, 20)
		conn.SetReadDeadline(time.Now().Add(time.Second))
		n, err := conn.Read(buffer2)

		if err != nil {
			errorCount++
			if errorCount >= maxError {
				t.Error("Could not read", err)
			}
			continue
		}

		for i :=0; i<n; i++ {
			buffer = append(buffer, buffer2[i])
		}
	}

	if !bytes.Equal(buffer, expect) {
		t.Error("StartServerAndExpect failed, handler",id,"expected", expect, "got", buffer)
	} else {
		fmt.Println("StartServerAndExpect handler",id," indeed received", buffer)
	}

	conn.SetWriteDeadline(time.Now().Add(time.Second))
	n, err := conn.Write(buffer)

	if err != nil || n != len(buffer) {
		t.Error("Could not echo back the", len(buffer), "bytes, only", n, ":", err)
	}
}

func StartServerAndExpect(data map[int][]byte, remote string, t *testing.T, done chan bool) {
	var socketListener *net.TCPListener
	s, err := net.Listen("tcp", remote)
	socketListener = s.(*net.TCPListener)

	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	id := 0

	wait := 0
	maxWait := 2

	for {
		socketListener.SetDeadline(time.Now().Add(time.Second))
		conn, err := socketListener.Accept()

		if err != nil {
			if err, ok := err.(*net.OpError); ok && err.Timeout() {
				// it was a timeout
				if wait >= maxWait {
					break
				}
				wait++
				continue
			}
			t.Error("StartServerAndExpect accept error:", err)
		} else {
			wg.Add(1)
			go handleConnection(id, conn, data[id], t, &wg)
			id++
		}
	}

	wg.Wait()
	socketListener.Close()
	time.Sleep(time.Second)
	done <- true
}

// Tests that the multiplexer forwards short messages
func TestEgress1(t *testing.T) {

	remote := "127.0.0.1:3000"
	payloadLength := 20
	upstreamChan := make(chan []byte)
	downstreamChan := make(chan []byte)
	stopChan := make(chan bool)

	go StartEgressHandler(remote, payloadLength, upstreamChan, downstreamChan, stopChan)

	// prepare a dummy message
	payload := []byte("hello")
	multiplexedMsg := make([]byte, MULTIPLEXER_HEADER_SIZE + len(payload))
	ID_str := generateRandomID()
	ID := []byte(ID_str[0:4])
	copy(multiplexedMsg[0:4], ID)
	multiplexedMsg[7]=byte(len(payload))
	copy(multiplexedMsg[8:], payload)

	doneChan := make(chan bool, 1)

	expected := make(map[int][]byte)
	expected[0] = payload
	go StartServerAndExpect(expected, remote, t, doneChan)

	upstreamChan <- multiplexedMsg

	<- doneChan

	echo := <- downstreamChan
	echoID := echo[0:4]
	size := int(binary.BigEndian.Uint32(echo[4:8]))
	data := echo[8:]
	if !bytes.Equal(echoID, ID) {
		t.Error("Echoed message ID is wrong", ID, echoID)
	}
	if !bytes.Equal(payload, data[:size]) {
		t.Error("Echoed message data is wrong", payload, data[:size])
	}
}

// Tests that the multiplexer forwards double messages into one stream
func TestEgress2(t *testing.T) {

	remote := "127.0.0.1:3000"
	payloadLength := 20
	upstreamChan := make(chan []byte)
	downstreamChan := make(chan []byte)
	stopChan := make(chan bool)

	go StartEgressHandler(remote, payloadLength, upstreamChan, downstreamChan, stopChan)

	// prepare a dummy message
	payload := []byte("hello")
	doubleHello := make([]byte, 2*len(payload))
	copy(doubleHello[0:5], payload)
	copy(doubleHello[5:10], payload)

	multiplexedMsg := make([]byte, MULTIPLEXER_HEADER_SIZE + len(payload))
	ID_str := generateRandomID()
	ID := []byte(ID_str[0:4])
	copy(multiplexedMsg[0:4], ID)
	multiplexedMsg[7]=byte(len(payload))
	copy(multiplexedMsg[8:], payload)

	doneChan := make(chan bool, 1)

	expected := make(map[int][]byte)
	expected[0] = doubleHello
	go StartServerAndExpect(expected, remote, t, doneChan)

	upstreamChan <- multiplexedMsg
	upstreamChan <- multiplexedMsg

	<- doneChan

	echo := <- downstreamChan
	echoID := echo[0:4]
	size := int(binary.BigEndian.Uint32(echo[4:8]))
	data := echo[8:]
	if !bytes.Equal(echoID, ID) {
		t.Error("Echoed message ID is wrong", ID, echoID)
	}
	if !bytes.Equal(doubleHello, data[:size]) {
		t.Error("Echoed message data is wrong", doubleHello, data[:size])
	}
}

// Tests that the multiplexer multiplexes short messages
func TestEgressMultiplex(t *testing.T) {

	remote := "127.0.0.1:3000"
	payloadLength := 20
	upstreamChan := make(chan []byte)
	downstreamChan := make(chan []byte)
	stopChan := make(chan bool)

	go StartEgressHandler(remote, payloadLength, upstreamChan, downstreamChan, stopChan)

	// prepare a dummy message
	payload := []byte("hello")
	multiplexedMsg := make([]byte, MULTIPLEXER_HEADER_SIZE + len(payload))
	ID_str := generateRandomID()
	ID := []byte(ID_str[0:4])
	copy(multiplexedMsg[0:4], ID)
	multiplexedMsg[7]=byte(len(payload))
	copy(multiplexedMsg[8:], payload)


	// prepare a dummy message 2
	payload2 := []byte("hello2")
	multiplexedMsg2 := make([]byte, MULTIPLEXER_HEADER_SIZE + len(payload2))
	ID2_str := generateRandomID()
	ID2 := []byte(ID2_str[0:4])
	copy(multiplexedMsg2[0:4], ID2)
	multiplexedMsg2[7]=byte(len(payload2))
	copy(multiplexedMsg2[8:], payload2)

	doneChan := make(chan bool, 1)

	expected := make(map[int][]byte)
	expected[0] = payload
	expected[1] = payload2
	go StartServerAndExpect(expected, remote, t, doneChan)

	upstreamChan <- multiplexedMsg
	upstreamChan <- multiplexedMsg2

	<- doneChan

	echo1 := <- downstreamChan
	echo2 := <- downstreamChan

	//swap messages if needed
	if bytes.Equal(ID, echo2[0:4]) && bytes.Equal(ID2, echo1[0:4]) {
		tmp := echo1
		echo1 = echo2
		echo2 = tmp
	}

	echoID1 := echo1[0:4]
	size1 := int(binary.BigEndian.Uint32(echo1[4:8]))
	data1 := echo1[8:]
	if !bytes.Equal(echoID1, ID) {
		t.Error("Echoed message ID is wrong", ID, echoID1)
	}
	if !bytes.Equal(payload, data1[:size1]) {
		t.Error("Echoed message data is wrong", payload, data1[:size1])
	}

	echoID2 := echo2[0:4]
	size2 := int(binary.BigEndian.Uint32(echo2[4:8]))
	data2 := echo2[8:]
	if !bytes.Equal(echoID2, ID2) {
		t.Error("Echoed message ID is wrong", ID2, echoID2)
	}
	if !bytes.Equal(payload2, data2[:size2]) {
		t.Error("Echoed message data is wrong", payload2, data2[:size2])
	}
}

// Tests that the multiplexer multiplexes long messages
func TestEgressMultiplexLong(t *testing.T) {

	remote := "127.0.0.1:3000"
	payloadLength := 20
	upstreamChan := make(chan []byte)
	downstreamChan := make(chan []byte)
	stopChan := make(chan bool)

	go StartEgressHandler(remote, payloadLength, upstreamChan, downstreamChan, stopChan)

	// prepare a dummy message
	payload := []byte("hello")
	multiplexedMsg := make([]byte, MULTIPLEXER_HEADER_SIZE+len(payload))
	ID_str := generateRandomID()
	ID := []byte(ID_str[0:4])
	copy(multiplexedMsg[0:4], ID)
	multiplexedMsg[7] = byte(len(payload))
	copy(multiplexedMsg[8:], payload)

	// prepare a dummy message 2
	payload2 := []byte("hello2")
	multiplexedMsg2 := make([]byte, MULTIPLEXER_HEADER_SIZE+len(payload2))
	ID2_str := generateRandomID()
	ID2 := []byte(ID2_str[0:4])
	copy(multiplexedMsg2[0:4], ID2)
	multiplexedMsg2[7] = byte(len(payload2))
	copy(multiplexedMsg2[8:], payload2)

	doneChan := make(chan bool, 1)

	doubleHello := make([]byte, 2*len(payload))
	copy(doubleHello[0:5], payload)
	copy(doubleHello[5:10], payload)

	doubleHello2 := make([]byte, 2*len(payload2))
	copy(doubleHello2[0:6], payload2)
	copy(doubleHello2[6:12], payload2)

	expected := make(map[int][]byte)
	expected[0] = doubleHello
	expected[1] = doubleHello2
	go StartServerAndExpect(expected, remote, t, doneChan)

	upstreamChan <- multiplexedMsg
	upstreamChan <- multiplexedMsg2
	upstreamChan <- multiplexedMsg2
	upstreamChan <- multiplexedMsg

	<-doneChan

	echo1 := <- downstreamChan
	echo2 := <- downstreamChan

	//swap messages if needed
	if bytes.Equal(ID, echo2[0:4]) && bytes.Equal(ID2, echo1[0:4]) {
		tmp := echo1
		echo1 = echo2
		echo2 = tmp
	}

	echoID1 := echo1[0:4]
	size1 := int(binary.BigEndian.Uint32(echo1[4:8]))
	data1 := echo1[8:]
	if !bytes.Equal(echoID1, ID) {
		t.Error("Echoed message ID is wrong", ID, echoID1)
	}
	if !bytes.Equal(doubleHello, data1[:size1]) {
		t.Error("Echoed message data is wrong", doubleHello, data1[:size1])
	}

	echoID2 := echo2[0:4]
	size2 := int(binary.BigEndian.Uint32(echo2[4:8]))
	data2 := echo2[8:]
	if !bytes.Equal(echoID2, ID2) {
		t.Error("Echoed message ID is wrong", ID2, echoID2)
	}
	if !bytes.Equal(doubleHello2, data2[:size2]) {
		t.Error("Echoed message data is wrong", doubleHello2, data2[:size2])
	}
}