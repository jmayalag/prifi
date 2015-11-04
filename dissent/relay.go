package main

import (
	"encoding/binary"
	"fmt"
	"github.com/lbarman/crypto/abstract"
	"github.com/lbarman/prifi/dcnet"
	"io"
	//"os"
	"strconv"
	"log"
	"strings"
	"net"
	"time"
	log2 "github.com/lbarman/prifi/log"
)

type Trustee struct {
	pubkey abstract.Point
}

type AnonSet struct {
	suite    abstract.Suite
	trustees []Trustee
}

// Periodic stats reporting
var begin = time.Now()
var report = begin
var numberOfReports = 0
var period, _ = time.ParseDuration("3s")
var totupcells = int64(0)
var totupbytes = int64(0)
var totdowncells = int64(0)
var totdownbytes = int64(0)

var parupcells = int64(0)
var parupbytes = int64(0)
var pardownbytes = int64(0)

func reportStatistics(payloadLength int, reportingLimit int) bool {
	now := time.Now()
	if now.After(report) {
		duration := now.Sub(begin).Seconds()

		instantUpSpeed := (float64(parupbytes)/period.Seconds())

		fmt.Printf("@ %fs; cell %f (%f) /sec, up %f (%f) B/s, down %f (%f) B/s\n",
			duration,
			 float64(totupcells)/duration, float64(parupcells)/period.Seconds(),
			 float64(totupbytes)/duration, instantUpSpeed,
			 float64(totdownbytes)/duration, float64(pardownbytes)/period.Seconds())

			// Next report time
		parupcells = 0
		parupbytes = 0
		pardownbytes = 0

		//log2.BenchmarkFloat(fmt.Sprintf("cellsize-%d-upstream-bytes", payloadLength), instantUpSpeed)

		data := struct {
		    Experiment string
		    CellSize int
		    Speed float64
		}{
		    "upstream-speed-given-cellsize",
		    payloadLength,
		    instantUpSpeed,
		}

		log2.JsonDump(data)

		report = now.Add(period)
		numberOfReports += 1

		if(reportingLimit > -1 && numberOfReports >= reportingLimit) {
			return false
		}
	}

	return true
}

func startRelay(payloadLength int, relayPort string, nClients int, nTrustees int, trusteesIp []string, reportingLimit int) {

	//the crypto parameters are static
	tg := dcnet.TestSetup(nil, suite, factory, nClients, nTrustees)
	me := tg.Relay

	//connect to the trustees

	trusteesConnections := make([]net.Conn, nTrustees)

	for i:= 0; i < nTrustees; i++ {
		currentTrusteeIp := strings.Replace(trusteesIp[i], "_", ":", -1)

		//connect
		fmt.Println("Relay connecting to trustee", i, "on address", currentTrusteeIp)
		conn, err := net.Dial("tcp", currentTrusteeIp)
		if err != nil {
			panic("Can't connect to trustee:" + err.Error())

			//TODO : maybe code something less brutal here
		}

		//tell the trustee server our parameters
		buffer := make([]byte, 20)
		binary.BigEndian.PutUint32(buffer[0:4], uint32(LLD_PROTOCOL_VERSION))
		binary.BigEndian.PutUint32(buffer[4:8], uint32(payloadLength))
		binary.BigEndian.PutUint32(buffer[8:12], uint32(nClients))
		binary.BigEndian.PutUint32(buffer[12:16], uint32(nTrustees))
		binary.BigEndian.PutUint32(buffer[16:20], uint32(i))

		fmt.Println("Writing", LLD_PROTOCOL_VERSION, "setup is", nClients, nTrustees, "role is", i, "cellSize ", payloadLength)

		n, err := conn.Write(buffer)

		if n < 1 || err != nil {
			panic("Error writing to socket:" + err.Error())
		}

		fmt.Println("Trustee", i, "is connected.")
		trusteesConnections[i] = conn
	}

	//starts the client server
	lsock, err := net.Listen("tcp", relayPort)
	if err != nil {
		panic("Can't open listen socket:" + err.Error())
	}

	// Wait for all the clients to connect
	clientsConnections := make([]net.Conn, nClients)

	for j := 0; j < nClients; j++ {
		fmt.Printf("Waiting for %d clients\n", nClients-j)

		conn, err := lsock.Accept()
		if err != nil {
			panic("Listen error:" + err.Error())
		}

		b := make([]byte, 1)
		n, err := conn.Read(b)
		if n < 1 || err != nil {
			panic("Read error:" + err.Error())
		}

		//TODO : not happy with this, reserve one byte for the client id
		nodeId := int(b[0] & 0x7f)
		if b[0]&0x80 == 0 && nodeId < nClients {
			if clientsConnections[nodeId] != nil {
				panic("Oops, client connected twice")
				j -= 1
			}
			clientsConnections[nodeId] = conn
		} else {
			panic("illegal node number")
		}
	}
	println("All clients and trustees connected.")

	// Create ciphertext slice bufferfers for all clients and trustees
	clientPayloadLength := me.Coder.ClientCellSize(payloadLength)
	clientsPayloadData  := make([][]byte, nClients)
	for i := 0; i < nClients; i++ {
		clientsPayloadData[i] = make([]byte, clientPayloadLength)
	}

	trusteePayloadLength := me.Coder.TrusteeCellSize(payloadLength)
	trusteesPayloadData  := make([][]byte, nTrustees)
	for i := 0; i < nTrustees; i++ {
		trusteesPayloadData[i] = make([]byte, trusteePayloadLength)
	}

	conns := make(map[int]chan<- []byte)
	downstream := make(chan dataWithConnectionId)
	nulldown := dataWithConnectionId{} // default empty downstream cell
	window := 2           // Maximum cells in-flight
	inflight := 0         // Current cells in-flight


	for {

		//TODO: change this way of breaking the loop, it's not very elegant..
		// Show periodic reports
		if(!reportStatistics(payloadLength, reportingLimit)) {
			println("Reporting limit matched; exiting the relay")
			break;
		}

		//TODO : check if it is required to send empty cell
		// See if there's any downstream data to forward.
		var downbuffer dataWithConnectionId
		select {
			case downbuffer = <-downstream: // some data to forward downstream
				//fmt.Println("Downstream data...")
				//fmt.Printf("v %d\n", len(downbuffer)-6)
			default: // nothing at the moment to forward
				downbuffer = nulldown
		}

		downstreamDataPayloadLength := len(downbuffer.data)
		downstreamData := make([]byte, 6+downstreamDataPayloadLength)
		binary.BigEndian.PutUint32(downstreamData[0:4], uint32(downbuffer.connectionId))
		binary.BigEndian.PutUint16(downstreamData[4:6], uint16(downstreamDataPayloadLength))
		copy(downstreamData[6:], downbuffer.data)

		// Broadcast the downstream data to all clients.
		for i := 0; i < nClients; i++ {
			n, err := clientsConnections[i].Write(downstreamData)

			if n != 6+downstreamDataPayloadLength {
				panic("Relay : Write to client failed, wrote "+strconv.Itoa(downstreamDataPayloadLength+6)+" where "+strconv.Itoa(n)+" was expected : " + err.Error())
			}
		}

		totdowncells++
		totdownbytes += int64(downstreamDataPayloadLength)
		pardownbytes += int64(downstreamDataPayloadLength)

		inflight++
		if inflight < window {
			continue // Get more cells in flight
		}

		me.Coder.DecodeStart(payloadLength, me.History)

		// Collect a cell ciphertext from each trustee
		for i := 0; i < nTrustees; i++ {
			
			//TODO: this looks blocking
			n, err := io.ReadFull(trusteesConnections[i], trusteesPayloadData[i])
			if n < trusteePayloadLength {
				panic("Relay : Read from trustee failed, read "+strconv.Itoa(n)+" where "+strconv.Itoa(trusteePayloadLength)+" was expected: " + err.Error())
			}

			me.Coder.DecodeTrustee(trusteesPayloadData[i])
		}

		// Collect an upstream ciphertext from each client
		for i := 0; i < nClients; i++ {

			//TODO: this looks blocking
			n, err := io.ReadFull(clientsConnections[i], clientsPayloadData[i])
			if n < clientPayloadLength {
				panic("Relay : Read from client failed, read "+strconv.Itoa(n)+" where "+strconv.Itoa(clientPayloadLength)+" was expected: " + err.Error())
			}

			me.Coder.DecodeClient(clientsPayloadData[i])
		}

		upstreamPlaintext := me.Coder.DecodeCell()
		inflight--

		totupcells++
		totupbytes += int64(payloadLength)
		parupcells++
		parupbytes += int64(payloadLength)

		// Process the decoded cell
		if upstreamPlaintext == nil {
			continue // empty or corrupt upstream cell
		}
		if len(upstreamPlaintext) != payloadLength {
			panic("DecodeCell produced wrong-size payload")
		}

		// Decode the upstream cell header (may be empty, all zeros)
		upstreamPlainTextConnId     := int(binary.BigEndian.Uint32(upstreamPlaintext[0:4]))
		upstreamPlainTextDataLength := int(binary.BigEndian.Uint16(upstreamPlaintext[4:6]))

		if upstreamPlainTextConnId == 0 {
			continue // no upstream data
		}

		//check which connection it belongs to
		//TODO: what is that ?? this is supposed to be anonymous
		conn := conns[upstreamPlainTextConnId]

		// client initiating new connection
		if conn == nil { 
			conn = relayNewConn(upstreamPlainTextConnId, downstream)
			conns[upstreamPlainTextConnId] = conn
		}

		if 6+upstreamPlainTextDataLength > payloadLength {
			log.Printf("upstream cell invalid length %d", 6+upstreamPlainTextDataLength)
			continue
		}

		conn <- upstreamPlaintext[6 : 6+upstreamPlainTextDataLength]
	}
}


func relayNewConn(connId int, downstreamData chan<- dataWithConnectionId) chan<- []byte {

	upstreamData := make(chan []byte)
	go relaySocksProxy(connId, upstreamData, downstreamData)
	return upstreamData
}
