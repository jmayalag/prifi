package stream_multiplexer

import (
	"net"

	"strconv"

	"gopkg.in/dedis/onet.v1/log"
	"encoding/binary"
	"crypto/rand"
	"bytes"
	"time"
)

const MULTIPLEXER_HEADER_SIZE = 4

type MultiplexedConnection struct {
	ID string
	ID_bytes []byte
	conn net.Conn
	stopChan chan bool
	maxPayloadLength int
}


func StartIngressServer(port int, payloadLength int, upstreamChan chan []byte, downstreamChan chan []byte, stopChan chan bool) {

	maxPayloadLength := payloadLength - MULTIPLEXER_HEADER_SIZE //we use 4 bytes for the multiplexing
	activeConnections := make([]*MultiplexedConnection, 0)

	var socket *net.TCPListener
	var err error
	s, err := net.Listen("tcp", ":"+strconv.Itoa(port))

	if err != nil {
		log.Error("Ingress server cannot start listening, shutting down :", err.Error())
		return
	} else {
		log.Lvl2("Ingress server is listening for connections on port ", port)
	}

	// cast as TCPListener to get the SetDeadline method
	socket =s.(*net.TCPListener)

	for {
		socket.SetDeadline(time.Now().Add(time.Second))
		conn, err := socket.Accept()

		select {
		case <-stopChan:
			log.Lvl2("Ingress server stopped.")

			//stops all subroutines
			for _, mc := range activeConnections {
				mc.stopChan <- true
			}
			socket.Close()
			return
		default:
		}

		if err != nil {
			if err, ok := err.(*net.OpError); ok && err.Timeout() {
				// it was a timeout
				continue
			}
			log.Lvl3("Ingress server error:", err)
		}

		id := generateRandomID()
		log.Lvl2("Ingress server just accepted a connection, assigning ID",id)

		if err != nil {
			log.Error("Ingress server got an error with this new connection, shutting down :", err.Error())
			socket.Close()
			return
		}

		mc := new(MultiplexedConnection)
		mc.conn = conn
		mc.ID = id
		ID_bytes := []byte(id)
		mc.ID_bytes = ID_bytes[0:MULTIPLEXER_HEADER_SIZE]
		mc.stopChan = make(chan bool, 1)
		mc.maxPayloadLength = maxPayloadLength

		activeConnections = append(activeConnections, mc)
		go handleActiveConnection(mc, upstreamChan, downstreamChan)
	}
}

func handleActiveConnection(mc *MultiplexedConnection, upstreamChan chan []byte, downstreamChan chan []byte) {
	// read the traffic from the connection, pipe it to the upstream channel
	go multiplexedConnectionReader(mc, upstreamChan)
	go multiplexedChannelReader(downstreamChan, mc)
}

func multiplexedChannelReader(dataChannel chan []byte, mc *MultiplexedConnection) {
	for {
		// poll the downstream chanel
		slice := <- dataChannel

		if len(slice) < MULTIPLEXER_HEADER_SIZE {
			// we cannot de-multiplex data without the ID, just ignore
			continue
		}

		ID := slice[0:MULTIPLEXER_HEADER_SIZE]
		data := slice[MULTIPLEXER_HEADER_SIZE:]

		if !bytes.Equal(ID, mc.ID_bytes) {
			// data is not for us
			continue
		}

		mc.conn.Write(data)
	}
}

func multiplexedConnectionReader(mc *MultiplexedConnection, dataChannel chan []byte) {
	for {
		// Read data from the connection
		buffer := make([]byte, mc.maxPayloadLength)
		n, err := mc.conn.Read(buffer)

		if err != nil {
			log.Error("SOCKS connectionReader error,", err)
			return
		}

		// Trim the data and send it through the data channel
		slice := make([]byte, n+MULTIPLEXER_HEADER_SIZE)
		copy(slice[0:MULTIPLEXER_HEADER_SIZE], mc.ID_bytes[0:MULTIPLEXER_HEADER_SIZE])
		copy(slice[MULTIPLEXER_HEADER_SIZE:], buffer[:n])
		dataChannel <- slice

		// Connection Closed Indicator
		if n == 0 {
			return
		}
	}
}

//generateID generates an ID from a private key
func generateRandomID() string {
	var n uint32
	binary.Read(rand.Reader, binary.LittleEndian, &n)

	return strconv.Itoa(int(n))
}