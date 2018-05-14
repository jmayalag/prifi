package stream_multiplexer

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"gopkg.in/dedis/onet.v2/log"
	"io"
	"net"
	"time"
)

// EgressServer takes data from a go channel and recreates the multiplexed TCP streams
type EgressServer struct {
	activeConnections map[string]*MultiplexedConnection
	maxMessageSize    int
	maxPayloadSize    int
	upstreamChan      chan []byte
	downstreamChan    chan []byte
	stopChan          chan bool
	verbose           bool
}

// StartEgressHandler creates (and block) an Egress Server
func StartEgressHandler(serverAddress string, maxMessageSize int, upstreamChan chan []byte, downstreamChan chan []byte, stopChan chan bool, verbose bool) {
	eg := new(EgressServer)
	eg.maxMessageSize = maxMessageSize
	eg.maxPayloadSize = maxMessageSize - MULTIPLEXER_HEADER_SIZE //we use 8 bytes for the multiplexing
	eg.upstreamChan = upstreamChan
	eg.downstreamChan = downstreamChan
	eg.stopChan = stopChan
	eg.activeConnections = make(map[string]*MultiplexedConnection)
	eg.verbose = verbose

	if verbose {
		log.Lvl1("Egress Server in verbose mode")
	}

	for {
		dataRead := <-upstreamChan

		// if too short or all bytes are zero, there was no data usptream, discard the frame
		if len(dataRead) < 4 || bytes.Equal(dataRead[0:4], make([]byte, 4)) {
			log.Lvl3("Egress Server: no upstream Data, continuing")
			continue
		}

		if len(dataRead) < MULTIPLEXER_HEADER_SIZE {
			// we cannot demultiplex, skip
			log.Lvl3("Egress Server: frame too short, continuing")
			continue
		}

		ID := string(dataRead[0:4])
		size := int(binary.BigEndian.Uint32(dataRead[4:8]))
		data := dataRead[8:]

		// trim the data if needed
		if len(data) > size {
			data = data[:size]
		}

		if eg.verbose {
			log.Lvl1("Clients -> Egress Server:\n" + hex.Dump(data))
		}

		// if this a new connection, dial it first
		if mc, ok := eg.activeConnections[ID]; !ok || mc.conn == nil {
			c, err := net.Dial("tcp", serverAddress)
			if err != nil {
				log.Error("Egress server: Could not connect to server, discarding data. Do you have a SOCKS server running on",
					serverAddress, "? You need one!", err)
				continue
			} else {

				mc := new(MultiplexedConnection)
				mc.conn = c
				mc.ID = ID
				mc.ID_bytes = []byte(ID)
				mc.stopChan = make(chan bool, 1)
				mc.maxMessageLength = eg.maxMessageSize

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
}

func (eg *EgressServer) egressConnectionReader(mc *MultiplexedConnection) {
	for {
		// Check if we need to stop
		select {
		case _ = <-mc.stopChan:
			mc.conn.Close()
			return
		default:
		}

		// Read data from the connection
		buffer := make([]byte, eg.maxPayloadSize)
		n, err := mc.conn.Read(buffer)

		if err != nil {
			if err, ok := err.(*net.OpError); ok && err.Timeout() {
				// it was a timeout
				continue
			}

			if err == io.EOF {
				// Connection closed indicator
				return
			}

			log.Error("Egress server: connectionReader error (reading will stop),", err)
			return
		}

		// Trim the data and send it through the data channel
		slice := make([]byte, n+MULTIPLEXER_HEADER_SIZE)
		copy(slice[0:4], mc.ID_bytes[:])
		binary.BigEndian.PutUint32(slice[4:8], uint32(n))
		copy(slice[MULTIPLEXER_HEADER_SIZE:], buffer[:n])
		eg.downstreamChan <- slice

		if eg.verbose {
			log.Lvl1("Egress Server -> Clients:\n", hex.Dump(slice))
		}

	}
}
