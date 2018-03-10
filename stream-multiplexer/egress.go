package stream_multiplexer

import (
	"time"
	"net"
	"encoding/binary"
	"github.com/dedis/onet/log"
)

type EgressServer struct {
	activeConnections map[string]*MultiplexedConnection
	maxMessageLength int
	maxPayloadLength int
	upstreamChan chan []byte
	downstreamChan chan []byte
	stopChan chan bool
}

func StartEgressHandler(serverAddress string, maxMessageLength int, upstreamChan chan []byte, downstreamChan chan []byte, stopChan chan bool) *EgressServer {
	eg := new(EgressServer)
	eg.maxMessageLength = maxMessageLength
	eg.maxPayloadLength = maxMessageLength - MULTIPLEXER_HEADER_SIZE //we use 8 bytes for the multiplexing
	eg.upstreamChan = upstreamChan
	eg.downstreamChan = downstreamChan
	eg.stopChan = stopChan
	eg.activeConnections = make(map[string]*MultiplexedConnection)

	for {
		dataRead := <- upstreamChan

		if len(dataRead) < MULTIPLEXER_HEADER_SIZE {
			// we cannot demultiplex, skip
			continue
		}

		ID := string(dataRead[0:4])
		size := int(binary.BigEndian.Uint32(dataRead[4:8]))
		data := dataRead[8:]

		// trim the data if needed
		if len(data) > size {
			data = data[:size]
		}

		// if this a new connection, dial it first
		if _, ok := eg.activeConnections[ID]; !ok {
			c, err := net.Dial("tcp", serverAddress)
			if err != nil {
				log.Error("Egress server: Could not connect to server, discarding data.", err)
			} else {

				mc := new(MultiplexedConnection)
				mc.conn = c
				mc.ID = ID
				mc.ID_bytes = []byte(ID)
				mc.stopChan = make(chan bool, 1)
				mc.maxMessageLength = eg.maxMessageLength

				eg.activeConnections[ID] = mc
				go eg.egressConnectionReader(mc)
			}
		}

		mc, _ := eg.activeConnections[ID]

		// Try to write to it; if it fails, clean it
		mc.conn.SetWriteDeadline(time.Now().Add(time.Second))
		n, err := mc.conn.Write(data)

		if err != nil || n != len(data) {
			log.Error("Egress server: could not write the whole", len(data), "bytes, only", n, "error", err)
			mc.conn.Close()
			mc.stopChan <- true
			eg.activeConnections[ID] = nil
		}
	}

	return eg
}

func (eg *EgressServer) egressConnectionReader(mc *MultiplexedConnection) {
	for {
		// Check if we need to stop
		select {
		case _ = <- mc.stopChan:
			mc.conn.Close()
			return
		default:
		}

		// Read data from the connection
		buffer := make([]byte, eg.maxPayloadLength)
		n, err := mc.conn.Read(buffer)

		if err != nil {
			if err, ok := err.(*net.OpError); ok && err.Timeout() {
				// it was a timeout
				continue
			}
			log.Error("Egress server: connectionReader error,", err)
			return
		}

		// Trim the data and send it through the data channel
		slice := make([]byte, n+MULTIPLEXER_HEADER_SIZE)
		copy(slice[0:4], mc.ID_bytes[:])
		binary.BigEndian.PutUint32(slice[4:8], uint32(n))
		copy(slice[MULTIPLEXER_HEADER_SIZE:], buffer[:n])
		eg.downstreamChan <- slice

		// Connection Closed Indicator
		if n == 0 {
			return
		}
	}
}