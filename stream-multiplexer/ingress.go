package stream_multiplexer

import (
	"net"

	"strconv"

	"gopkg.in/dedis/onet.v1/log"
	"encoding/binary"
	"crypto/rand"
	"bytes"
)

type MultiplexedConnection struct {
	ID string
	ID_bytes []byte
	conn net.Conn
	stopChan chan bool
	maxPayloadLength int
}


func StartIngressServer(port int, payloadLength int, upstreamChan chan []byte, downstreamChan chan []byte, stopChan chan bool) {

	maxPayloadLength := payloadLength - 4 //we use 4 bytes for the multiplexing
	activeConnections := make([]*MultiplexedConnection, 0)

	socket, err := net.Listen("tcp", ":"+strconv.Itoa(port))

	if err != nil {
		log.Error("Ingress server cannot start listening, shutting down :", err.Error())
		return
	} else {
		log.Lvl2("Ingress server is listening for connections on port ", port)
	}

	for {
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
		mc.ID_bytes = ID_bytes[0:4]
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

		if len(slice) < 4 {
			// we cannot de-multiplex data without the ID, just ignore
			continue
		}

		ID := slice[0:4]
		data := slice[4:]

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
		slice := make([]byte, n+4)
		copy(slice[0:4], mc.ID_bytes[0:4])
		copy(slice[4:], buffer[:n])
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