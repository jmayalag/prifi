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

type IdConnectionAndPublicKey struct{
	Id 			int
	Conn 		net.Conn
	PublicKey 	abstract.Point
}

func startRelay(payloadLength int, relayPort string, nClients int, nTrustees int, trusteesIp []string, reportingLimit int) {

	relayState := initiateRelayState(relayPort, nTrustees, nClients, payloadLength, reportingLimit, trusteesIp)

	//start the server waiting for clients
	newClientConnectionsChan        := make(chan net.Conn) 					//channel with unparsed clients
	go relayServerListener(relayPort, newClientConnectionsChan)

	//start the client parser
	newClientWithIdAndPublicKeyChan := make(chan IdConnectionAndPublicKey)  //channel with parsed clients
	go welcomeNewClients(newClientConnectionsChan, newClientWithIdAndPublicKeyChan, relayState)

	//start the actual protocol
	relayState.connectToAllTrustees()
	relayState.waitForDefaultNumberOfClients(newClientWithIdAndPublicKeyChan)
	relayState.advertisePublicKeys()	

	//inputs and feedbacks for "processMessageLoop"
	protocolFailed        := make(chan bool)
	indicateEndOfProtocol := make(chan int)
	go relayState.processMessageLoop(protocolFailed, indicateEndOfProtocol) //CAREFUL RELAYSTATE IS SHARED BETWEEN THREADS

	//control loop
	var protocolHasFailed bool
	var endOfProtocolState int
	var newClient IdConnectionAndPublicKey
	for {
		select {
			case protocolHasFailed = <- protocolFailed:
				//re-run setup, something went wrong
				fmt.Println(protocolHasFailed)

			case newClient = <- newClientWithIdAndPublicKeyChan:
				//we tell processMessageLoop to stop ASAP
				indicateEndOfProtocol <- 1

			case endOfProtocolState = <- indicateEndOfProtocol:
				if endOfProtocolState != 2 {
					panic("something went wrong, should not happen")
				}

				//a new client has connected
				//1. copy the previous relayState
				newRelayState := relayState.copyStateAndAddNewClient(newClient)
				//2. disconnect the trustees (but not the clients)
				relayState.disconnectFromAllTrustees()
				//3. compose new client list
				//(done in 1, but this is a bit ugly)
				//4. reconnect to trustees
				newRelayState.connectToAllTrustees()
				//5. exchange the public keys 
				newRelayState.advertisePublicKeys()
				//6. process message loop (on the new relayState)
				go newRelayState.processMessageLoop(protocolFailed, indicateEndOfProtocol)

			default: 
				//all clear!
				time.Sleep(1000)
		}
	}
}

func (relayState *RelayState) connectToAllTrustees() {
	//connect to the trustees
	for i:= 0; i < relayState.nTrustees; i++ {
		connectToTrustee(i, relayState.trusteesHosts[i], relayState)
	}
	fmt.Println("Trustees connecting done, ", len(relayState.trusteesPublicKeys), "trustees connected")
}

func (relayState *RelayState) disconnectFromAllTrustees() {
	//disconnect to the trustees
	for i:= 0; i < relayState.nTrustees; i++ {
		relayState.trusteesConnections[i].Close()
	}
	fmt.Println("Trustees connecting done, ", len(relayState.trusteesPublicKeys), "trustees connected")
}

func (relayState *RelayState) waitForDefaultNumberOfClients(newClientConnectionsChan chan IdConnectionAndPublicKey) {
	currentClients := 0
	var newClientConnection IdConnectionAndPublicKey

	fmt.Printf("Waiting for %d clients (on port %s)\n", relayState.nClients - currentClients, relayState.RelayPort)
	for currentClients < relayState.nClients {
		select{
				case newClientConnection = <-newClientConnectionsChan: 

					//todo : this needs to be done better
					id := newClientConnection.Id
					relayState.clientsConnections[id] = newClientConnection.Conn
					relayState.clientPublicKeys[id] = newClientConnection.PublicKey

					currentClients += 1
					fmt.Printf("Waiting for %d clients (on port %s)\n", relayState.nClients - currentClients, relayState.RelayPort)
				default: 
					time.Sleep(100 * time.Millisecond)
		}
	}
	fmt.Println("Client connecting done, ", len(relayState.clientPublicKeys), "clients connected")
}

func (relayState *RelayState) copyStateAndAddNewClient(newClient *IdConnectionAndPublicKey){
	newNClients := relayState.nClients + 1
	newRelayState := initiateRelayState(relayState.RelayPort, relayState.nTrustees, newNClients, relayState.PayloadLength, relayState.ReportingLimit, relayState.trusteesHosts)

	//we keep the previous client params
	copy(newRelayState.clientPublicKeys, relayState.clientPublicKeys)
	copy(newRelayState.clientsConnections, relayState.clientsConnections)

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


func (relayState *RelayState) processMessageLoop(protocolFailed chan bool, indicateEndOfProtocol chan bool){
	//TODO : if something fail, send true->protocolFailed

	stats := emptyStatistics(relayState.ReportingLimit)

	// Create ciphertext slice bufferfers for all clients and trustees
	clientPayloadLength := relayState.CellCoder.ClientCellSize(relayState.PayloadLength)
	clientsPayloadData  := make([][]byte, relayState.nClients)
	for i := 0; i < relayState.nClients; i++ {
		clientsPayloadData[i] = make([]byte, clientPayloadLength)
	}

	trusteePayloadLength := relayState.CellCoder.TrusteeCellSize(relayState.PayloadLength)
	trusteesPayloadData  := make([][]byte, relayState.nTrustees)
	for i := 0; i < relayState.nTrustees; i++ {
		trusteesPayloadData[i] = make([]byte, trusteePayloadLength)
	}

	conns := make(map[int]chan<- []byte)
	downstream := make(chan dataWithConnectionId)
	nulldown := dataWithConnectionId{} // default empty downstream cell
	window := 2           // Maximum cells in-flight
	inflight := 0         // Current cells in-flight

	for {

		//if the main thread tells us to stop (for re-setup)
		tellClientsToResync := false
		select {
			case tellClientsToResync = <- indicateEndOfProtocol:
				//nothing to do, we updated the variable already
			default:
		}

		//we report the speed, bytes exchanged, etc
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

		//compute the message type; if 1, the client know they will resync
		msgType := 0
		if tellClientsToResync{
			msgType = 1
		}

		//craft the message for clients
		downstreamDataPayloadLength := len(downbuffer.data)
		downstreamData := make([]byte, 6+downstreamDataPayloadLength)
		binary.BigEndian.PutUint32(downstreamData[0:4], uint32(msgType))
		//binary.BigEndian.PutUint32(downstreamData[0:4], uint32(downbuffer.connectionId))
		binary.BigEndian.PutUint16(downstreamData[4:6], uint16(downstreamDataPayloadLength))
		copy(downstreamData[6:], downbuffer.data)

		// Broadcast the downstream data to all clients.
		util.BroadcastMessage(relayState.clientsConnections, downstreamData)
		stats.addDownstreamCell(int64(downstreamDataPayloadLength))

		inflight++
		if inflight < window {
			continue // Get more cells in flight
		}

		relayState.CellCoder.DecodeStart(relayState.PayloadLength, relayState.MessageHistory)

		// Collect a cell ciphertext from each trustee
		for i := 0; i < relayState.nTrustees; i++ {			
			//TODO: this looks blocking
			n, err := io.ReadFull(relayState.trusteesConnections[i], trusteesPayloadData[i])
			if n < trusteePayloadLength {
				panic("Relay : Read from trustee failed, read "+strconv.Itoa(n)+" where "+strconv.Itoa(trusteePayloadLength)+" was expected: " + err.Error())
			}

			relayState.CellCoder.DecodeTrustee(trusteesPayloadData[i])
		}

		// Collect an upstream ciphertext from each client
		for i := 0; i < relayState.nClients; i++ {
			//TODO: this looks blocking
			n, err := io.ReadFull(relayState.clientsConnections[i], clientsPayloadData[i])
			if n < clientPayloadLength {
				panic("Relay : Read from client failed, read "+strconv.Itoa(n)+" where "+strconv.Itoa(clientPayloadLength)+" was expected: " + err.Error())
			}

			relayState.CellCoder.DecodeClient(clientsPayloadData[i])
		}

		upstreamPlaintext := relayState.CellCoder.DecodeCell()
		inflight--

		stats.addUpstreamCell(int64(relayState.PayloadLength))

		// Process the decoded cell
		if upstreamPlaintext == nil {
			continue // empty or corrupt upstream cell
		}
		if len(upstreamPlaintext) != relayState.PayloadLength {
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

		if 6+upstreamPlainTextDataLength > relayState.PayloadLength {
			log.Printf("upstream cell invalid length %d", 6+upstreamPlainTextDataLength)
			continue
		}

		conn <- upstreamPlaintext[6 : 6+upstreamPlainTextDataLength]
	}
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

func welcomeNewClients(newClientConnectionsChan chan net.Conn, newClientWithPkChan chan IdConnectionAndPublicKey, relayState *RelayState) {	
	newClientsToParse := make(chan IdConnectionAndPublicKey)
	var newClientConnection net.Conn
	var newClientWithIdAndPk IdConnectionAndPublicKey

	for {
		select{
			//accept the TCP connection, and parse the parameters
			case newClientConnection = <-newClientConnectionsChan: 
				go relayParseClientParams(newClientConnection, relayState, newClientsToParse)
			
			//once client is ready (we have params+pk), forward to the other channel
			case newClientWithIdAndPk = <-newClientsToParse: 
				fmt.Println("New client is ready !")
				fmt.Println(newClientWithIdAndPk)
				newClientWithPkChan <- newClientWithIdAndPk
			default: 
				time.Sleep(1000) //todo : check this duration
		}
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

func relayParseClientParams(conn net.Conn, relayState *RelayState, newConnAndPk chan IdConnectionAndPublicKey) {

	nodeId, conn, publicKey := relayParseClientParamsAux(conn, relayState)
	s := IdConnectionAndPublicKey{Id: nodeId, Conn: conn, PublicKey: publicKey}
	newConnAndPk <- s
}