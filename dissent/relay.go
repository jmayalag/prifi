package main

import (
	"encoding/binary"
	"fmt"
	"encoding/hex"
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

type RelayState struct {
	Name				string

	PublicKey			abstract.Point
	privateKey			abstract.Secret
	
	nClients			int
	nTrustees			int

	clientsConnections  []net.Conn
	trusteesConnections []net.Conn
	trusteesPublicKeys  []abstract.Point
	clientPublicKeys    []abstract.Point
	
	CellCoder			dcnet.CellCoder
	
	MessageHistory		abstract.Cipher

	PayloadLength		int
	ReportingLimit		int
}

func initiateRelayState(nTrustees int, nClients int, payloadLength int, reportingLimit int) *RelayState {

	params := new(RelayState)

	params.Name           = "Relay"
	params.PayloadLength  = payloadLength
	params.ReportingLimit = reportingLimit

	//prepare the crypto parameters
	rand 	:= suite.Cipher([]byte(params.Name))
	base	:= suite.Point().Base()

	//generate own parameters
	params.privateKey       = suite.Secret().Pick(rand)
	params.PublicKey        = suite.Point().Mul(base, params.privateKey)

	params.nClients  = nClients
	params.nTrustees = nTrustees

	//placeholders for pubkeys and connections
	params.trusteesPublicKeys = make([]abstract.Point, nTrustees)
	params.clientPublicKeys   = make([]abstract.Point, nClients)

	params.trusteesConnections = make([]net.Conn, nTrustees)
	params.clientsConnections  = make([]net.Conn, nClients)

	//sets the cell coder, and the history
	params.CellCoder = factory()

	return params
}

func startRelay(payloadLength int, relayPort string, nClients int, nTrustees int, trusteesIp []string, reportingLimit int) {

	relayState := initiateRelayState(nTrustees, nClients, payloadLength, reportingLimit)
	stats := emptyStatistics(reportingLimit)

	//connect to the trustees
	for i:= 0; i < nTrustees; i++ {
		currentTrusteeIp := strings.Replace(trusteesIp[i], "_", ":", -1) //trick for windows shell, where ":" separates args
		connectToTrustee(i, currentTrusteeIp, relayState)
	}

	//starts the client server
	lsock, err := net.Listen("tcp", relayPort)
	if err != nil {
		panic("Can't open listen socket:" + err.Error())
	}

	// Wait for all the clients to connect
	for j := 0; j < nClients; j++ {
		fmt.Printf("Waiting for %d clients (on port %s)\n", nClients-j, relayPort)
		relayAcceptOneClient(lsock, relayState)		
	}
	println("All clients and trustees connected.")

	//Prepare the messages
	messageForClient   := MarshalPublicKeyArrayToByteArray(relayState.trusteesPublicKeys)
	messageForTrustees := MarshalPublicKeyArrayToByteArray(relayState.clientPublicKeys)

	//broadcast to the clients
	broadcastMessage(relayState.clientsConnections, messageForClient)
	broadcastMessage(relayState.trusteesConnections, messageForTrustees)
	
	println("All crypto stuff exchanged !")

	for {
		time.Sleep(5000 * time.Millisecond)
	}


	// Create ciphertext slice bufferfers for all clients and trustees
	clientPayloadLength := relayState.CellCoder.ClientCellSize(payloadLength)
	clientsPayloadData  := make([][]byte, nClients)
	for i := 0; i < nClients; i++ {
		clientsPayloadData[i] = make([]byte, clientPayloadLength)
	}

	trusteePayloadLength := relayState.CellCoder.TrusteeCellSize(payloadLength)
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

		stats.report(relayState)
		if stats.reportingDone() {
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
			n, err := relayState.clientsConnections[i].Write(downstreamData)

			if n != 6+downstreamDataPayloadLength {
				panic("Relay : Write to client failed, wrote "+strconv.Itoa(downstreamDataPayloadLength+6)+" where "+strconv.Itoa(n)+" was expected : " + err.Error())
			}
		}

		stats.addDownstreamCell(int64(downstreamDataPayloadLength))

		inflight++
		if inflight < window {
			continue // Get more cells in flight
		}

		relayState.CellCoder.DecodeStart(payloadLength, relayState.MessageHistory)

		// Collect a cell ciphertext from each trustee
		for i := 0; i < nTrustees; i++ {
			
			//TODO: this looks blocking
			n, err := io.ReadFull(relayState.trusteesConnections[i], trusteesPayloadData[i])
			if n < trusteePayloadLength {
				panic("Relay : Read from trustee failed, read "+strconv.Itoa(n)+" where "+strconv.Itoa(trusteePayloadLength)+" was expected: " + err.Error())
			}

			relayState.CellCoder.DecodeTrustee(trusteesPayloadData[i])
		}

		// Collect an upstream ciphertext from each client
		for i := 0; i < nClients; i++ {

			//TODO: this looks blocking
			n, err := io.ReadFull(relayState.clientsConnections[i], clientsPayloadData[i])
			if n < clientPayloadLength {
				panic("Relay : Read from client failed, read "+strconv.Itoa(n)+" where "+strconv.Itoa(clientPayloadLength)+" was expected: " + err.Error())
			}

			relayState.CellCoder.DecodeClient(clientsPayloadData[i])
		}

		upstreamPlaintext := relayState.CellCoder.DecodeCell()
		inflight--

		stats.addUpstreamCell(int64(payloadLength))

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

func connectToTrustee(trusteeId int, trusteeHostAddr string, relayState *RelayState) {
	//connect
	fmt.Println("Relay connecting to trustee", trusteeId, "on address", trusteeHostAddr)
	conn, err := net.Dial("tcp", trusteeHostAddr)
	if err != nil {
		panic("Can't connect to trustee:" + err.Error())
		//TODO : maybe code something less brutal here
	}

	//tell the trustee server our parameters
	buffer := make([]byte, 20)
	binary.BigEndian.PutUint32(buffer[0:4], uint32(LLD_PROTOCOL_VERSION))
	binary.BigEndian.PutUint32(buffer[4:8], uint32(relayState.PayloadLength))
	binary.BigEndian.PutUint32(buffer[8:12], uint32(relayState.nClients))
	binary.BigEndian.PutUint32(buffer[12:16], uint32(relayState.nTrustees))
	binary.BigEndian.PutUint32(buffer[16:20], uint32(trusteeId))

	fmt.Println("Writing", LLD_PROTOCOL_VERSION, "setup is", relayState.nClients, relayState.nTrustees, "role is", trusteeId, "cellSize ", relayState.PayloadLength)

	n, err := conn.Write(buffer)

	if n < 1 || err != nil {
		panic("Error writing to socket:" + err.Error())
	}

	// Now read the public key
	buffer2 := make([]byte, 1024)
	
	// Read the incoming connection into the buffer.
	reqLen, err := conn.Read(buffer2)
	if err != nil {
	    fmt.Println(">>>> Relay : error reading:", err.Error())
	}

	fmt.Println(">>>>  Relay : reading public key", reqLen)
	keySize := int(binary.BigEndian.Uint32(buffer2[4:8]))
	keyBytes := buffer2[8:(8+keySize)]


	fmt.Println(hex.Dump(keyBytes))

	publicKey := suite.Point()
	err2 := publicKey.UnmarshalBinary(keyBytes)

	if err2 != nil {
		panic(">>>>  Relay : can't unmarshal trustee key ! " + err2.Error())
	}

	fmt.Println("Trustee", trusteeId, "is connected.")
	

	//side effects
	relayState.trusteesConnections[trusteeId] = conn
	relayState.trusteesPublicKeys[trusteeId]  = publicKey
}

func relayAcceptOneClient(listeningSocket net.Listener, relayState *RelayState) {
	conn, err := listeningSocket.Accept()
	
	if err != nil {
		panic("Listen error:" + err.Error())
	}

	buffer := make([]byte, 512)
	_, err2 := conn.Read(buffer)
	if err2 != nil {
		panic("Read error:" + err2.Error())
	}

	version := int(binary.BigEndian.Uint32(buffer[0:4]))

	if(version != LLD_PROTOCOL_VERSION) {
		fmt.Println(">>>> Relay client version", version, "!= relay version", LLD_PROTOCOL_VERSION)
		panic("fatal error")
	}

	nodeId := int(binary.BigEndian.Uint32(buffer[4:8]))
	keySize := int(binary.BigEndian.Uint32(buffer[8:12]))
	keyBytes := buffer[12:(12+keySize)] 

	publicKey := suite.Point()
	err3 := publicKey.UnmarshalBinary(keyBytes)

	if err3 != nil {
		panic(">>>>  Relay : can't unmarshal client key ! " + err3.Error())
	}

	if nodeId < 0 || nodeId >= relayState.nClients {
		panic("illegal node number")
	}

	//side effect
	relayState.clientsConnections[nodeId] = conn
	relayState.clientPublicKeys[nodeId] = publicKey
}

type Statistics struct {
	begin			time.Time
	nextReport		time.Time
	nReports		int
	maxNReports		int
	period			time.Duration

	totupcells		int64
	totupbytes 		int64
	totdowncells 	int64
	totdownbytes 	int64
	parupcells		int64
	parupbytes 		int64
	pardownbytes	int64
}

func emptyStatistics(reportingLimit int) *Statistics{
	stats := Statistics{time.Now(), time.Now(), 0, reportingLimit, time.Duration(3)*time.Second, 0, 0, 0, 0, 0, 0, 0}
	return &stats
}

func (stats *Statistics) reportingDone() bool {
	return stats.nReports >= stats.maxNReports
}

func (stats *Statistics) addDownstreamCell(nBytes int64) {
	stats.totdowncells += 1
	stats.totdownbytes += nBytes
	stats.pardownbytes += nBytes
}

func (stats *Statistics) addUpstreamCell(nBytes int64) {
	stats.totupcells += 1
	stats.totupbytes += nBytes
	stats.parupcells += 1
	stats.parupbytes += nBytes
}

func (stats *Statistics) report(state *RelayState) {
	now := time.Now()
	if now.After(stats.nextReport) {
		duration := now.Sub(stats.begin).Seconds()
		instantUpSpeed := (float64(stats.parupbytes)/stats.period.Seconds())

		fmt.Printf("@ %fs; cell %f (%f) /sec, up %f (%f) B/s, down %f (%f) B/s\n",
			duration,
			 float64(stats.totupcells)/duration, float64(stats.parupcells)/stats.period.Seconds(),
			 float64(stats.totupbytes)/duration, instantUpSpeed,
			 float64(stats.totdownbytes)/duration, float64(stats.pardownbytes)/stats.period.Seconds())

		// Next report time
		stats.parupcells = 0
		stats.parupbytes = 0
		stats.pardownbytes = 0

		//log2.BenchmarkFloat(fmt.Sprintf("cellsize-%d-upstream-bytes", payloadLength), instantUpSpeed)

		//write JSON
		data := struct {
		    Experiment string
		    CellSize int
		    Speed float64
		}{
		    "upstream-speed-given-cellsize",
		    state.PayloadLength,
		    instantUpSpeed,
		}
		log2.JsonDump(data)

		stats.nextReport = now.Add(stats.period)
		stats.nReports += 1
	}
}