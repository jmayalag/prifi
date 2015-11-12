package main

import (
	"encoding/binary"
	"fmt"
	"encoding/hex"
	"github.com/lbarman/crypto/abstract"
	"github.com/lbarman/prifi/dcnet"
	"io"
	"time"
	"strconv"
	"log"
	"net"
	log2 "github.com/lbarman/prifi/log"
	"github.com/lbarman/prifi/util"
)

type RelayState struct {
	Name				string
	RelayPort			string

	PublicKey			abstract.Point
	privateKey			abstract.Secret
	
	nClients			int
	nTrustees			int

	trusteesHosts		[]string

	clientsConnections  []net.Conn
	trusteesConnections []net.Conn
	trusteesPublicKeys  []abstract.Point
	clientPublicKeys    []abstract.Point
	
	CellCoder			dcnet.CellCoder
	
	MessageHistory		abstract.Cipher

	PayloadLength		int
	ReportingLimit		int
}

func (state *RelayState) connectToAllTrustees() {
	//connect to the trustees
	for i:= 0; i < state.nTrustees; i++ {
		connectToTrustee(i, state.trusteesHosts[i], state)
	}
	fmt.Println("Trustees connecting done, ", len(state.trusteesPublicKeys), "trustees connected")
}

func (state *RelayState) disconnectFromAllTrustees() {
	//disconnect to the trustees
	for i:= 0; i < state.nTrustees; i++ {
		state.trusteesConnections[i].Close()
	}
	fmt.Println("Trustees connecting done, ", len(state.trusteesPublicKeys), "trustees connected")
}

func (state *RelayState) waitForClientsToConnect(newClientConnections chan net.Conn) {
	currentClients := 0
	var newClientConnection net.Conn

	fmt.Printf("Waiting for %d clients (on port %s)\n", state.nClients - currentClients, state.RelayPort)
	for currentClients < state.nClients {
		select{
				case newClientConnection = <-newClientConnections: 
					relayParseClientParams(newClientConnection, state)
					currentClients += 1
					fmt.Printf("Waiting for %d clients (on port %s)\n", state.nClients - currentClients, state.RelayPort)
				default: 
					time.Sleep(100 * time.Millisecond)
		}
	}
	fmt.Println("Client connecting done, ", len(state.clientPublicKeys), "clients connected")
}

func (state *RelayState) stateWithNewClient(newClient *IdConnectionAndPublicKey){
	newNClients := state.nClients + 1
	newRelayState := initiateRelayState(state.RelayPort, state.nTrustees, newNClients, state.PayloadLength, state.ReportingLimit, state.trusteesHosts)

	//we keep the previous client params
	copy(newRelayState.clientPublicKeys, state.clientPublicKeys)
	copy(newRelayState.clientsConnections, state.clientsConnections)

	//we add the new client
	newRelayState.clientPublicKeys = append(newRelayState.clientPublicKeys, newClient.PublicKey)
	newRelayState.clientsConnections = append(newRelayState.clientsConnections, newClient.Conn)
}

func (relayState *RelayState) advertisePublicKeys(){
	//Prepare the messages
	messageForClient   := util.MarshalPublicKeyArrayToByteArray(relayState.trusteesPublicKeys)
	messageForTrustees := util.MarshalPublicKeyArrayToByteArray(relayState.clientPublicKeys)

	//broadcast to the clients
	util.BroadcastMessage(relayState.clientsConnections, messageForClient)
	util.BroadcastMessage(relayState.trusteesConnections, messageForTrustees)
	fmt.Println("Advertising done, to", len(relayState.clientsConnections), "clients and", len(relayState.trusteesConnections))
}


func (state *RelayState) processMessageLoop(newClientConnections chan net.Conn){

	stats := emptyStatistics(state.ReportingLimit)

	// Create ciphertext slice bufferfers for all clients and trustees
	clientPayloadLength := state.CellCoder.ClientCellSize(state.PayloadLength)
	clientsPayloadData  := make([][]byte, state.nClients)
	for i := 0; i < state.nClients; i++ {
		clientsPayloadData[i] = make([]byte, clientPayloadLength)
	}

	trusteePayloadLength := state.CellCoder.TrusteeCellSize(state.PayloadLength)
	trusteesPayloadData  := make([][]byte, state.nTrustees)
	for i := 0; i < state.nTrustees; i++ {
		trusteesPayloadData[i] = make([]byte, trusteePayloadLength)
	}

	conns := make(map[int]chan<- []byte)
	downstream := make(chan dataWithConnectionId)
	nulldown := dataWithConnectionId{} // default empty downstream cell
	window := 2           // Maximum cells in-flight
	inflight := 0         // Current cells in-flight

	newClientsToParse := make(chan IdConnectionAndPublicKey)
	var newClientConnection net.Conn
	var newClientWithIdAndPk IdConnectionAndPublicKey

	for {

		tellClientsToResync := false
		select{
			//accept the TCP connection, and parse the parameters
			case newClientConnection = <-newClientConnections: 
				go relayParseClientParamsAsync(newClientConnection, state, newClientsToParse)
			
			//once client is ready (we have params+pk), trigger the re-setup
			case newClientWithIdAndPk = <-newClientsToParse: 
				fmt.Println("New client is ready !")
				fmt.Println(newClientWithIdAndPk)
				tellClientsToResync = true
			default: 
		}

		stats.report(state)
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

		msgType := 0
		if tellClientsToResync{
			msgType = 1
		}

		downstreamDataPayloadLength := len(downbuffer.data)
		downstreamData := make([]byte, 6+downstreamDataPayloadLength)
		binary.BigEndian.PutUint32(downstreamData[0:4], uint32(msgType))
		//binary.BigEndian.PutUint32(downstreamData[0:4], uint32(downbuffer.connectionId))
		binary.BigEndian.PutUint16(downstreamData[4:6], uint16(downstreamDataPayloadLength))
		copy(downstreamData[6:], downbuffer.data)

		// Broadcast the downstream data to all clients.
		util.BroadcastMessage(state.clientsConnections, downstreamData)
		stats.addDownstreamCell(int64(downstreamDataPayloadLength))

		inflight++
		if inflight < window {
			continue // Get more cells in flight
		}

		state.CellCoder.DecodeStart(state.PayloadLength, state.MessageHistory)

		// Collect a cell ciphertext from each trustee
		for i := 0; i < state.nTrustees; i++ {			
			//TODO: this looks blocking
			n, err := io.ReadFull(state.trusteesConnections[i], trusteesPayloadData[i])
			if n < trusteePayloadLength {
				panic("Relay : Read from trustee failed, read "+strconv.Itoa(n)+" where "+strconv.Itoa(trusteePayloadLength)+" was expected: " + err.Error())
			}

			state.CellCoder.DecodeTrustee(trusteesPayloadData[i])
		}

		// Collect an upstream ciphertext from each client
		for i := 0; i < state.nClients; i++ {
			//TODO: this looks blocking
			n, err := io.ReadFull(state.clientsConnections[i], clientsPayloadData[i])
			if n < clientPayloadLength {
				panic("Relay : Read from client failed, read "+strconv.Itoa(n)+" where "+strconv.Itoa(clientPayloadLength)+" was expected: " + err.Error())
			}

			state.CellCoder.DecodeClient(clientsPayloadData[i])
		}

		upstreamPlaintext := state.CellCoder.DecodeCell()
		inflight--

		stats.addUpstreamCell(int64(state.PayloadLength))

		// Process the decoded cell
		if upstreamPlaintext == nil {
			continue // empty or corrupt upstream cell
		}
		if len(upstreamPlaintext) != state.PayloadLength {
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

		if 6+upstreamPlainTextDataLength > state.PayloadLength {
			log.Printf("upstream cell invalid length %d", 6+upstreamPlainTextDataLength)
			continue
		}

		conn <- upstreamPlaintext[6 : 6+upstreamPlainTextDataLength]
	}
}

func startRelay(payloadLength int, relayPort string, nClients int, nTrustees int, trusteesIp []string, reportingLimit int) {

	relayState := initiateRelayState(relayPort, nTrustees, nClients, payloadLength, reportingLimit, trusteesIp)

	//start the server waiting for clients
	newClientConnections := make(chan net.Conn)
	go relayServerListener(relayPort, newClientConnections)

	//connect to all trustees
	relayState.connectToAllTrustees()
	relayState.waitForClientsToConnect(newClientConnections)
	relayState.advertisePublicKeys()	
	relayState.processMessageLoop(newClientConnections)
}

func initiateRelayState(relayPort string, nTrustees int, nClients int, payloadLength int, reportingLimit int, trusteesHosts []string) *RelayState {
	params := new(RelayState)

	params.Name           = "Relay"
	params.RelayPort      = relayPort
	params.PayloadLength  = payloadLength
	params.ReportingLimit = reportingLimit

	//prepare the crypto parameters
	rand 	:= suite.Cipher([]byte(params.Name))
	base	:= suite.Point().Base()

	//generate own parameters
	params.privateKey       = suite.Secret().Pick(rand)
	params.PublicKey        = suite.Point().Mul(base, params.privateKey)

	params.nClients      = nClients
	params.nTrustees     = nTrustees
	params.trusteesHosts = trusteesHosts

	//placeholders for pubkeys and connections
	params.trusteesPublicKeys = make([]abstract.Point, nTrustees)
	params.clientPublicKeys   = make([]abstract.Point, nClients)

	params.trusteesConnections = make([]net.Conn, nTrustees)
	params.clientsConnections  = make([]net.Conn, nClients)

	//sets the cell coder, and the history
	params.CellCoder = factory()

	return params
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

func relayServerListener(listeningPort string, newConnection chan net.Conn) {
	listeningSocket, err := net.Listen("tcp", listeningPort)
	if err != nil {
		panic("Can't open listen socket:" + err.Error())
	}

	for {
		conn, err2 := listeningSocket.Accept()
		if err != nil {
			fmt.Println("Relay : can't accept client. ", err2.Error())
		}
		newConnection <- conn
	}
}

func relayParseClientParamsAux(conn net.Conn, relayState *RelayState) (int, net.Conn, abstract.Point) {
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

	//check that the node ID is not used
	if(nodeId <= len(relayState.clientsConnections) && relayState.clientsConnections[nodeId] != nil) {
		fmt.Println(nodeId, "is used")
		newId := len(relayState.clientsConnections)
		fmt.Println("Client with ID ", nodeId, "tried to connect, but some client already took that ID. changing ID to", newId)
		nodeId = newId
	}

	keySize := int(binary.BigEndian.Uint32(buffer[8:12]))
	keyBytes := buffer[12:(12+keySize)] 

	publicKey := suite.Point()
	err3 := publicKey.UnmarshalBinary(keyBytes)

	if err3 != nil {
		panic(">>>>  Relay : can't unmarshal client key ! " + err3.Error())
	}

	return nodeId, conn, publicKey
}

type IdConnectionAndPublicKey struct{
	Id 			int
	Conn 		net.Conn
	PublicKey 	abstract.Point
}

func relayParseClientParamsAsync(conn net.Conn, relayState *RelayState, newConnAndPk chan IdConnectionAndPublicKey) {

	nodeId, conn, publicKey := relayParseClientParamsAux(conn, relayState)
	s := IdConnectionAndPublicKey{Id: nodeId, Conn: conn, PublicKey: publicKey}
	newConnAndPk <- s
}

func relayParseClientParams(conn net.Conn,relayState *RelayState) {

	nodeId, conn, publicKey := relayParseClientParamsAux(conn, relayState)

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

	totalUpstreamCells		int64
	totalUpstreamBytes 		int64
	totalDownstreamCells 	int64
	totalDownstreamBytes 	int64
	instantUpstreamCells	int64
	instantUpstreamBytes 	int64
	instantDownstreamBytes	int64
}

func emptyStatistics(reportingLimit int) *Statistics{
	stats := Statistics{time.Now(), time.Now(), 0, reportingLimit, time.Duration(3)*time.Second, 0, 0, 0, 0, 0, 0, 0}
	return &stats
}

func (stats *Statistics) reportingDone() bool {
	return stats.nReports >= stats.maxNReports
}

func (stats *Statistics) addDownstreamCell(nBytes int64) {
	stats.totalDownstreamCells += 1
	stats.totalDownstreamBytes += nBytes
	stats.instantDownstreamBytes += nBytes
}

func (stats *Statistics) addUpstreamCell(nBytes int64) {
	stats.totalUpstreamCells += 1
	stats.totalUpstreamBytes += nBytes
	stats.instantUpstreamCells += 1
	stats.instantUpstreamBytes += nBytes
}

func (stats *Statistics) report(state *RelayState) {
	now := time.Now()
	if now.After(stats.nextReport) {
		duration := now.Sub(stats.begin).Seconds()
		instantUpSpeed := (float64(stats.instantUpstreamBytes)/stats.period.Seconds())

		fmt.Printf("@ %fs; cell %f (%f) /sec, up %f (%f) B/s, down %f (%f) B/s\n",
			duration,
			 float64(stats.totalUpstreamCells)/duration, float64(stats.instantUpstreamCells)/stats.period.Seconds(),
			 float64(stats.totalUpstreamBytes)/duration, instantUpSpeed,
			 float64(stats.totalDownstreamBytes)/duration, float64(stats.instantDownstreamBytes)/stats.period.Seconds())

		// Next report time
		stats.instantUpstreamCells = 0
		stats.instantUpstreamBytes = 0
		stats.instantDownstreamBytes = 0

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