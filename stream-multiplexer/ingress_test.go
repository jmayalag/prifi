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
)

// Tests that the multiplexer produces messages of at most "payloadLength"
func TestSizes(t *testing.T) {

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
	fmt.Println("Writing long data...")
	conn1.Write(longData)
	fmt.Println("Done writing.")

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

	fmt.Println("Test done.")
	stopChan <- true
	fmt.Println("Exiting.")
}

// First test: two different connections send interleaved messages.
// Checks that all messages are multiplexed, with the correct IDs
func TestMultiplexer(t *testing.T) {

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
		id_conn1_bytes = data[0:MULTIPLEXER_HEADER_SIZE]

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
		if !bytes.Equal(id_conn1_bytes, data[0:MULTIPLEXER_HEADER_SIZE]){
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
		id_conn2_bytes = data[0:MULTIPLEXER_HEADER_SIZE]

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
		if !bytes.Equal(id_conn2_bytes, data[0:MULTIPLEXER_HEADER_SIZE]){
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
		if !bytes.Equal(id_conn1_bytes, data[0:MULTIPLEXER_HEADER_SIZE]){
			t.Error("Data on the same stream gets different IDs")
		}

	case <-time.After(1 * time.Second):
		t.Error("No data written on the upstreamchannel")
	}

	stopChan <- true
}