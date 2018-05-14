package stream_multiplexer

import (
	"net"

	"strconv"

	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"gopkg.in/dedis/onet.v2/log"
	"io"
	"sync"
	"time"
)

// MULTIPLEXER_HEADER_SIZE is the size of the header for the multiplexed data,
// currently 4 byte for StreamID and 4 byte for length
const MULTIPLEXER_HEADER_SIZE = 8

// MultiplexedConnection represents a TCP connections to which we assigned
// a stream ID
type MultiplexedConnection struct {
	ID               string
	ID_bytes         []byte
	conn             net.Conn
	stopChan         chan bool
	maxMessageLength int
}

// IngressServer accepts TCPs connections and multiplexes them (read- and write-)
// over go channels
type IngressServer struct {
	activeConnectionsLock sync.Locker
	activeConnections     []*MultiplexedConnection
	socketListener        *net.TCPListener
	maxMessageSize        int
	maxPayloadSize        int
	upstreamChan          chan []byte
	downstreamChan        chan []byte
	stopChan              chan bool
	verbose               bool
}

// StartIngressServer creates (and block) an Ingress Server
func StartIngressServer(port int, maxMessageSize int, upstreamChan chan []byte, downstreamChan chan []byte, stopChan chan bool, verbose bool) {

	ig := new(IngressServer)
	ig.maxMessageSize = maxMessageSize
	ig.upstreamChan = upstreamChan
	ig.downstreamChan = downstreamChan
	ig.stopChan = stopChan
	ig.maxPayloadSize = maxMessageSize - MULTIPLEXER_HEADER_SIZE //we use 8 bytes for the multiplexing
	ig.activeConnectionsLock = new(sync.Mutex)
	ig.activeConnections = make([]*MultiplexedConnection, 0)
	ig.verbose = verbose
	if verbose {
		log.Lvl1("Ingress Server in verbose mode")
	}

	var err error
	s, err := net.Listen("tcp", ":"+strconv.Itoa(port))

	if err != nil {
		log.Error("Ingress server cannot start listening, shutting down :", err.Error())
		return
	}
	log.Lvl2("Ingress server is listening for connections on port ", port)

	// cast as TCPListener to get the SetDeadline method
	ig.socketListener = s.(*net.TCPListener)

	// starts a handler that dispatches the data from "downstreamChan" into the correct connection
	go ig.multiplexedChannelReader()

	for {
		ig.socketListener.SetDeadline(time.Now().Add(time.Second))
		conn, err := ig.socketListener.Accept()

		select {
		case <-stopChan:
			log.Lvl2("Ingress server stopped.")

			//stops all subroutines
			for _, mc := range ig.activeConnections {
				mc.stopChan <- true
			}
			ig.socketListener.Close()
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
		log.Lvl2("Ingress server just accepted a connection, assigning ID", id)

		if err != nil {
			log.Error("Ingress server got an error with this new connection, shutting down :", err.Error())
			ig.socketListener.Close()
			return
		}

		mc := new(MultiplexedConnection)
		mc.conn = conn
		mc.ID = id
		ID_bytes := []byte(id)
		mc.ID_bytes = ID_bytes[0:4]
		mc.stopChan = make(chan bool, 1)
		mc.maxMessageLength = ig.maxMessageSize

		// lock the list before editing it
		ig.activeConnectionsLock.Lock()
		ig.activeConnections = append(ig.activeConnections, mc)
		ig.activeConnectionsLock.Unlock()

		// starts a handler that pours "mc.connection" into upstreamChan
		go ig.ingressConnectionReader(mc)
	}
}

// multiplexedChannelReader reads the "downstreamChan" and dispatches the data to the correct connection
func (ig *IngressServer) multiplexedChannelReader() {
	for {
		// poll the downstream chanel
		slice := <-ig.downstreamChan

		if len(slice) < MULTIPLEXER_HEADER_SIZE {
			// we cannot de-multiplex data without the header, just ignore
			continue
		}

		if ig.verbose {
			log.Lvl1("Ingress Server <- DCNet: \n", hex.Dump(slice))
		}

		ID := slice[0:4]
		length := int(binary.BigEndian.Uint32(slice[4:MULTIPLEXER_HEADER_SIZE]))
		data := slice[MULTIPLEXER_HEADER_SIZE:]

		// trim the data if needed
		if len(data) > length {
			data = data[0:length]
		}

		ig.activeConnectionsLock.Lock()
		for _, v := range ig.activeConnections {
			if bytes.Equal(v.ID_bytes, ID) {
				v.conn.Write(data)
				break
			}
		}
		ig.activeConnectionsLock.Unlock()

	}
}

func (ig *IngressServer) ingressConnectionReader(mc *MultiplexedConnection) {
	for {
		// Check if we need to stop
		select {
		case _ = <-mc.stopChan:
			mc.conn.Close()
			return
		default:
		}

		// Read data from the connection
		buffer := make([]byte, ig.maxPayloadSize)
		mc.conn.SetReadDeadline(time.Now().Add(time.Second))
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

			log.Error("Ingress server: connectionReader error,", err)
			return
		}

		// Trim the data and send it through the data channel
		slice := make([]byte, n+MULTIPLEXER_HEADER_SIZE)
		copy(slice[0:4], mc.ID_bytes[:])
		binary.BigEndian.PutUint32(slice[4:8], uint32(n))
		copy(slice[MULTIPLEXER_HEADER_SIZE:], buffer[:n])

		if ig.verbose {
			log.Lvl1("Ingress Server -> DCNet:\n", hex.Dump(slice))
		}

		ig.upstreamChan <- slice
	}
}

//generateID generates an ID from a private key
func generateRandomID() string {
	var n uint32
	binary.Read(rand.Reader, binary.LittleEndian, &n)

	return strconv.Itoa(int(n))
}
