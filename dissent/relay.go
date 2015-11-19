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
	"sync"
	"net"
	//"github.com/lbarman/prifi/util"
)

type RelayState struct {
	sync.RWMutex

	Name				string
	RelayPort			string

	PublicKey			abstract.Point
	privateKey			abstract.Secret
	
	nClients			int
	nTrustees			int

	trusteesHosts		[]string

	clients  			[]NodeRepresentation
	trustees  			[]NodeRepresentation
	
	CellCoder			dcnet.CellCoder
	
	MessageHistory		abstract.Cipher

	PayloadLength		int
	ReportingLimit		int
}

func (relayState *RelayState) verboseRLock(){
	fmt.Println("Requesting RLock")
	relayState.RLock()
}
func (relayState *RelayState) verboseLock(){
	fmt.Println("Requesting Lock")
	relayState.Lock()
}
func (relayState *RelayState) verboseRUnlock(){
	fmt.Println("Requesting RUnlock")
	relayState.RUnlock()
}
func (relayState *RelayState) verboseUnlock(){
	fmt.Println("Requesting Unlock")
	relayState.Unlock()
}

type NodeRepresentation struct {
	Id			int
	Conn 		net.Conn
	Connected 	bool
	PublicKey	abstract.Point
}

type IdConnectionAndPublicKey struct{
	Id 			int
	Conn 		net.Conn
	PublicKey 	abstract.Point
}

var relayState 			*RelayState 

func startRelay(payloadLength int, relayPort string, nClients int, nTrustees int, trusteesIp []string, reportingLimit int) {

	relayState = new(RelayState)
	setupRelayState(relayPort, nTrustees, nClients, payloadLength, reportingLimit, trusteesIp)

	//start the server waiting for clients
	newClientConnectionsChan        := make(chan net.Conn) 	          //channel with unparsed clients
	go relayServerListener(relayPort, newClientConnectionsChan)

	//start the client parser
	newClientWithIdAndPublicKeyChan := make(chan NodeRepresentation)  //channel with parsed clients
	go welcomeNewClients(newClientConnectionsChan, newClientWithIdAndPublicKeyChan)

	//start the actual protocol
	connectToAllTrustees() //request RW lock
	waitForDefaultNumberOfClients(newClientWithIdAndPublicKeyChan) //ask for RW lock
	advertisePublicKeys()	

	//inputs and feedbacks for "processMessageLoop"
	protocolFailed        := make(chan bool)
	indicateEndOfProtocol := make(chan int)
	go processMessageLoop(protocolFailed, indicateEndOfProtocol)

	//control loop
	var endOfProtocolState int
	var isProtocolRunning = true //we just started processMessageLoop
	newClients := make([]NodeRepresentation, 0)

	for {
		select {
			case protocolHasFailed := <- protocolFailed:
				fmt.Println(protocolHasFailed)
				fmt.Println("Relay Handler : Processing loop has failed")
				isProtocolRunning = false
				//TODO : re-run setup, something went wrong. Maybe restart from 0 ?

			case newClient := <- newClientWithIdAndPublicKeyChan:
				//we tell processMessageLoop to stop when possible
				newClients = append(newClients, newClient)
				if isProtocolRunning {
					fmt.Println("Relay Handler : new Client is ready, stopping processing loop")
					indicateEndOfProtocol <- PROTOCOL_STATUS_GONNA_RESYNC
				} else {
					fmt.Println("Relay Handler : new Client is ready, restarting processing loop")
					isProtocolRunning = restartProtocol(newClients, protocolFailed, indicateEndOfProtocol) //request RW Lock
					fmt.Println("Done...")
				}

			case endOfProtocolState = <- indicateEndOfProtocol:
				fmt.Println("Relay Handler : main loop stopped, resyncing")

				if endOfProtocolState != PROTOCOL_STATUS_RESYNCING {
					panic("something went wrong, should not happen")
				}

				isProtocolRunning = restartProtocol(newClients, protocolFailed, indicateEndOfProtocol) //request RW Lock
			default: 
				//all clear! keep this thread handler load low, (accept changes every X millisecond)
				time.Sleep(1000 * time.Millisecond)
		}
	}
}

func restartProtocol(newClients []NodeRepresentation, protocolFailed chan bool, indicateEndOfProtocol chan int) bool {
	//copy the previous relayState
	println("a")
	excludeDisconnectedClients() //request RW lock	
	println("b")	
	disconnectFromAllTrustees()

	//add the new clients to the previous (filtered) list
	for i:=0; i<len(newClients); i++{
		addNewClient(newClients[i])  //request RW lock	
		fmt.Println("Adding new client")
		fmt.Println(newClients[i])
	}
	newClients = make([]NodeRepresentation, 0)

	//if we dont have enough client, stop.
	if len(relayState.clients) == 0{
		fmt.Println("Relay Handler : not enough client, stopping and waiting...")
		return false
	} else {
		//re-advertise the configuration 	
		connectToAllTrustees()  //request RW lock
		advertisePublicKeys()

		println("Client should be OK ...")
		time.Sleep(5*time.Second)

		//6. process message loop (on the new relayState)
		go processMessageLoop(protocolFailed, indicateEndOfProtocol)
		return true
	}
}

func connectToAllTrustees() {
	relayState.verboseRLock()
	trusteesHosts := make([]string, relayState.nTrustees)
	copy(trusteesHosts, relayState.trusteesHosts)
	relayState.verboseRUnlock()

	//connect to the trustees
	for i:= 0; i < len(trusteesHosts); i++ {
		connectToTrustee(i, trusteesHosts[i]) //request RW lock
	}
	fmt.Println("Trustees connecting done, ", len(relayState.trustees), "trustees connected")

}

func disconnectFromAllTrustees() {
	relayState.verboseRLock()

	//disconnect to the trustees
	for i:= 0; i < len(relayState.trustees); i++ {
		relayState.trustees[i].Conn.Close()
	}
	fmt.Println("Trustees disonnecting done, ", len(relayState.trustees), "trustees disconnected")

	relayState.verboseRUnlock()
}

func waitForDefaultNumberOfClients(newClientConnectionsChan chan NodeRepresentation) {

	relayState.verboseRLock()
	currentClients   := len(relayState.clients)
	expectedNClients := relayState.nClients
	relayState.verboseRUnlock()

	fmt.Printf("Waiting for %d clients (on port %s)\n", relayState.nClients - currentClients, relayState.RelayPort)

	for currentClients < expectedNClients {
		select{
				case newClient := <-newClientConnectionsChan: 

					relayState.verboseLock()
					relayState.clients = append(relayState.clients, newClient)
					relayState.verboseUnlock()
					
					fmt.Printf("Waiting for %d clients (on port %s)\n", relayState.nClients - currentClients, relayState.RelayPort)
				default: 
					time.Sleep(100 * time.Millisecond)
		}
		relayState.verboseRLock()
		currentClients = len(relayState.clients)
		expectedNClients = relayState.nClients
		relayState.verboseRUnlock()
	}
	fmt.Println("Client connecting done, ", len(relayState.clients), "clients connected")
}

func excludeDisconnectedClients(){
	relayState.verboseLock()

	//count the clients that disconnected
	nClientsDisconnected := 0
	for i := 0; i<len(relayState.clients); i++ {
		if !relayState.clients[i].Connected {
			fmt.Println("Relay Handler : Client ", i, " discarded, seems he disconnected...")
			nClientsDisconnected++
		}
	}

	//count the actual number of clients, and init the new state with the old parameters
	newNClients   := relayState.nClients - nClientsDisconnected

	//copy the connected clients
	newClients := make([]NodeRepresentation, newNClients)
	j := 0
	for i := 0; i<len(relayState.clients); i++ {
		if relayState.clients[i].Connected {
			newClients[j] = relayState.clients[i]
			fmt.Println("Adding Client ", i, "who's not disconnected")
			j++
		}
	}

	relayState.clients = newClients
	relayState.verboseUnlock()
}

func addNewClient(newClient NodeRepresentation){
	relayState.verboseLock()
	relayState.nClients = relayState.nClients + 1
	relayState.clients  = append(relayState.clients, newClient)
	relayState.verboseUnlock()
}

func advertisePublicKeys(){
	//Prepare the messages
	relayState.verboseRLock()
	dataForClients   := MarshalNodeRepresentationArrayToByteArray(relayState.trustees)
	dataForTrustees := MarshalNodeRepresentationArrayToByteArray(relayState.clients)

	//craft the message for clients
	messageForClientsLength := len(dataForClients)
	messageForClients := make([]byte, 10+messageForClientsLength)
	binary.BigEndian.PutUint32(messageForClients[0:4], uint32(MESSAGE_TYPE_PUBLICKEYS))
	binary.BigEndian.PutUint32(messageForClients[4:8], uint32(SOCKS_CONNECTION_ID_EMPTY))
	binary.BigEndian.PutUint16(messageForClients[8:10], uint16(messageForClientsLength))
	copy(messageForClients[10:], dataForClients)

	//TODO : would be cleaner if the trustees used the same structure for the message

	//broadcast to the clients
	BroadcastMessage(relayState.clients, messageForClients)
	BroadcastMessage(relayState.trustees, dataForTrustees)
	fmt.Println("Advertising done, to", len(relayState.clients), "clients and", len(relayState.trustees), "trustees")
	relayState.verboseRUnlock()
}


func processMessageLoop(protocolFailed chan bool, indicateEndOfProtocol chan int){
	//TODO : if something fail, send true->protocolFailed

	relayState.verboseRLock()

	fmt.Println("")
	fmt.Println("#################################")
	fmt.Println("# Configuration updated, running")
	fmt.Println("#", relayState.nClients, "clients", relayState.nTrustees, "trustees")

	for i := 0; i<len(relayState.clients); i++ {
		fmt.Println("# Client", relayState.clients[i].Id, " on port ", relayState.clients[i].Conn.LocalAddr())
	}
	for i := 0; i<len(relayState.trustees); i++ {
		fmt.Println("# Trustee", relayState.trustees[i].Id, " on port ", relayState.trustees[i].Conn.LocalAddr())
	}
	fmt.Println("#################################")
	fmt.Println("")

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
	relayState.verboseRUnlock()

	socksProxyConnections := make(map[int]chan<- []byte)
	downstream            := make(chan dataWithConnectionId)
	nulldown              := dataWithConnectionId{} // default empty downstream cell
	window                := 2           // Maximum cells in-flight
	inflight              := 0         // Current cells in-flight

	currentSetupContinues := true
	
	for currentSetupContinues {

		//if the main thread tells us to stop (for re-setup)
		tellClientsToResync := false
		var mainThreadStatus int
		select {
			case mainThreadStatus = <- indicateEndOfProtocol:
				if mainThreadStatus == PROTOCOL_STATUS_GONNA_RESYNC {
					println("Main thread status is PROTOCOL_STATUS_GONNA_RESYNC, gonna warn the clients")
					tellClientsToResync = true
				}
			default:
		}

		//we report the speed, bytes exchanged, etc
		stats.reportRelay(relayState)
		if stats.reportingDone() {
			println("Reporting limit matched; exiting the relay")
			break;
		}

		// See if there's any downstream data to forward.
		var downbuffer dataWithConnectionId
		select {
			case downbuffer = <-downstream: // some data to forward downstream
			default: 
				downbuffer = nulldown
		}

		//compute the message type; if MESSAGE_TYPE_DATA_AND_RESYNC, the clients know they will resync
		msgType := MESSAGE_TYPE_DATA
		if tellClientsToResync{
			msgType = MESSAGE_TYPE_DATA_AND_RESYNC
			currentSetupContinues = false
		}

		//craft the message for clients
		downstreamDataPayloadLength := len(downbuffer.data)
		downstreamData := make([]byte, 10+downstreamDataPayloadLength)
		binary.BigEndian.PutUint32(downstreamData[0:4], uint32(msgType))
		binary.BigEndian.PutUint32(downstreamData[4:8], uint32(downbuffer.connectionId)) //this is the SOCKS connection ID
		binary.BigEndian.PutUint16(downstreamData[8:10], uint16(downstreamDataPayloadLength))
		copy(downstreamData[10:], downbuffer.data)

		relayState.verboseRLock()

		// Broadcast the downstream data to all clients.
		BroadcastMessage(relayState.clients, downstreamData)
		stats.addDownstreamCell(int64(downstreamDataPayloadLength))

		inflight++
		if inflight < window {
			continue // Get more cells in flight
		}

		relayState.CellCoder.DecodeStart(relayState.PayloadLength, relayState.MessageHistory)

		// Collect a cell ciphertext from each trustee
		errorInThisCell := false
		for i := 0; i < relayState.nTrustees; i++ {	

			if errorInThisCell {
				break
			}

			//TODO: this looks blocking
			n, err := io.ReadFull(relayState.trustees[i].Conn, trusteesPayloadData[i])
			if err != nil {
				errorInThisCell                  = true
				relayState.trustees[i].Connected = false
				fmt.Println("Relay main loop : Trustee "+strconv.Itoa(i)+" deconnected")
			}
			if n < trusteePayloadLength {
				errorInThisCell                  = true
				relayState.trustees[i].Connected = false
				fmt.Println("Relay main loop : Read from trustee failed, read "+strconv.Itoa(n)+" where "+strconv.Itoa(trusteePayloadLength)+" was expected: " + err.Error())
			}

			relayState.CellCoder.DecodeTrustee(trusteesPayloadData[i])
		}

		// Collect an upstream ciphertext from each client
		for i := 0; i < relayState.nClients; i++ {

			if errorInThisCell {
				break
			}

			//TODO: this looks blocking
			n, err := io.ReadFull(relayState.clients[i].Conn, clientsPayloadData[i])
			if err != nil {
				errorInThisCell                 = true
				relayState.clients[i].Connected = false
				fmt.Println("Relay main loop : Client "+strconv.Itoa(i)+" deconnected")
			}
			if n < clientPayloadLength {
				errorInThisCell                 = true
				relayState.clients[i].Connected = false
				fmt.Println("Relay main loop : Read from client failed, read "+strconv.Itoa(n)+" where "+strconv.Itoa(trusteePayloadLength)+" was expected: " + err.Error())
			}

			relayState.CellCoder.DecodeClient(clientsPayloadData[i])
		}

		if errorInThisCell {
			
			fmt.Println("Relay main loop : Cell will be invalid, some party disconnected. Warning the clients...")

			//craft the message for clients
			downstreamData := make([]byte, 10)
			binary.BigEndian.PutUint32(downstreamData[0:4], uint32(3))
			binary.BigEndian.PutUint32(downstreamData[4:8], uint32(downbuffer.connectionId)) //this is the SOCKS connection ID
			binary.BigEndian.PutUint16(downstreamData[8:10], uint16(0))
			BroadcastMessage(relayState.clients, downstreamData)
			
			relayState.verboseRUnlock()
			break
		} else {
			upstreamPlaintext := relayState.CellCoder.DecodeCell()
			inflight--

			stats.addUpstreamCell(int64(relayState.PayloadLength))

			// Process the decoded cell
			if upstreamPlaintext == nil {
				relayState.verboseRUnlock()
				continue // empty or corrupt upstream cell
			}
			if len(upstreamPlaintext) != relayState.PayloadLength {
				panic("DecodeCell produced wrong-size payload")
			}

			// Decode the upstream cell header (may be empty, all zeros)
			socksConnId     := int(binary.BigEndian.Uint32(upstreamPlaintext[0:4]))
			socksDataLength := int(binary.BigEndian.Uint16(upstreamPlaintext[4:6]))

			if socksConnId == SOCKS_CONNECTION_ID_EMPTY {
				relayState.verboseRUnlock()
				continue 
			}

			socksConn := socksProxyConnections[socksConnId]

			// client initiating new connection
			if socksConn == nil { 
				socksConn = newSOCKSProxyHandler(socksConnId, downstream)
				socksProxyConnections[socksConnId] = socksConn
			}

			if 6+socksDataLength > relayState.PayloadLength {
				log.Printf("upstream cell invalid length %d", 6+socksDataLength)

				relayState.verboseRUnlock()
				continue
			}

			socksConn <- upstreamPlaintext[6 : 6+socksDataLength]
			relayState.verboseRUnlock()
		}
	}

	println("Relay main loop : waiting 5 seconds, client should now be waiting for new parameters...")
	time.Sleep(5*time.Second)

	indicateEndOfProtocol <- PROTOCOL_STATUS_RESYNCING
}

func setupRelayState(relayPort string, nTrustees int, nClients int, payloadLength int, reportingLimit int, trusteesHosts []string) {
	relayState.verboseLock()

	relayState.Name           = "Relay"
	relayState.RelayPort      = relayPort
	relayState.PayloadLength  = payloadLength
	relayState.ReportingLimit = reportingLimit

	//prepare the crypto parameters
	rand 	:= suite.Cipher([]byte(relayState.Name))
	base	:= suite.Point().Base()

	//generate own parameters
	relayState.privateKey       = suite.Secret().Pick(rand)
	relayState.PublicKey        = suite.Point().Mul(base, relayState.privateKey)

	relayState.nClients      = nClients
	relayState.nTrustees     = nTrustees
	relayState.trusteesHosts = trusteesHosts

	//sets the cell coder, and the history
	relayState.CellCoder = factory()
	
	relayState.verboseUnlock()
}

func welcomeNewClients(newConnectionsChan chan net.Conn, newClientChan chan NodeRepresentation) {	
	newClientsToParse := make(chan NodeRepresentation)

	for {
		select{
			//accept the TCP connection, and parse the parameters
			case newConnection := <-newConnectionsChan: 
				go relayParseClientParams(newConnection, newClientsToParse)
			
			//once client is ready (we have params+pk), forward to the other channel
			case newClient := <-newClientsToParse: 
				fmt.Println("welcomeNewClients : New client is ready !")
				newClientChan <- newClient
			default: 
				time.Sleep(1000) //todo : check this duration
		}
	}
}

func newSOCKSProxyHandler(connId int, downstreamData chan<- dataWithConnectionId) chan<- []byte {
	upstreamData := make(chan []byte)
	go relaySocksProxy(connId, upstreamData, downstreamData)
	return upstreamData
}

func connectToTrustee(trusteeId int, trusteeHostAddr string) {
	//connect
	fmt.Println("Relay connecting to trustee", trusteeId, "on address", trusteeHostAddr)
	conn, err := net.Dial("tcp", trusteeHostAddr)
	if err != nil {
		panic("Can't connect to trustee:" + err.Error())
		//TODO : maybe code something less brutal here
	}

	//tell the trustee server our parameters
	relayState.verboseRLock()
	buffer := make([]byte, 20)
	binary.BigEndian.PutUint32(buffer[0:4], uint32(LLD_PROTOCOL_VERSION))
	binary.BigEndian.PutUint32(buffer[4:8], uint32(relayState.PayloadLength))
	binary.BigEndian.PutUint32(buffer[8:12], uint32(relayState.nClients))
	binary.BigEndian.PutUint32(buffer[12:16], uint32(relayState.nTrustees))
	binary.BigEndian.PutUint32(buffer[16:20], uint32(trusteeId))

	fmt.Println("Writing", LLD_PROTOCOL_VERSION, "setup is", relayState.nClients, relayState.nTrustees, "role is", trusteeId, "cellSize ", relayState.PayloadLength)
	relayState.verboseRUnlock()

	n, err := conn.Write(buffer)

	if n < 1 || err != nil {
		panic("Error writing to socket:" + err.Error())
	}

	// Now read the public key
	buffer2 := make([]byte, 1024)
	
	// Read the incoming connection into the buffer.
	_, err2 := conn.Read(buffer2)
	if err2 != nil {
	    fmt.Println(">>>> Relay : error reading:", err.Error())
	}

	keySize := int(binary.BigEndian.Uint32(buffer2[4:8]))
	keyBytes := buffer2[8:(8+keySize)]
	publicKey := suite.Point()
	err3 := publicKey.UnmarshalBinary(keyBytes)

	if err3 != nil {
		panic(">>>>  Relay : can't unmarshal trustee key ! " + err2.Error())
	}

	fmt.Println("Trustee", trusteeId, "is connected.")
	
	newTrustee := NodeRepresentation{trusteeId, conn, true, publicKey}

	relayState.verboseLock()
	relayState.trustees = append(relayState.trustees, newTrustee)
	relayState.verboseUnlock()
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

func relayParseClientParamsAux(conn net.Conn) NodeRepresentation {
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
	relayState.verboseRLock()
	idExists := false
	nextFreeId := 0
	for i:=0; i<len(relayState.clients); i++{
		if relayState.clients[i].Id == nodeId {
			idExists = true
		}
		if relayState.clients[i].Id == nextFreeId {
			nextFreeId++
		}
	}
	relayState.verboseRUnlock()
	if idExists {
		fmt.Println("Client with ID ", nodeId, "tried to connect, but some client already took that ID. changing ID to", nextFreeId)
		nodeId = nextFreeId
	}

	keySize := int(binary.BigEndian.Uint32(buffer[8:12]))
	keyBytes := buffer[12:(12+keySize)] 

	publicKey := suite.Point()
	err3 := publicKey.UnmarshalBinary(keyBytes)

	if err3 != nil {
		panic(">>>>  Relay : can't unmarshal client key ! " + err3.Error())
	}

	newClient := NodeRepresentation{nodeId, conn, true, publicKey}

	return newClient
}

func relayParseClientParams(conn net.Conn, newClientChan chan NodeRepresentation) {

	newClient := relayParseClientParamsAux(conn)
	newClientChan <- newClient
}


// TODO : this should be somewhere else
func MarshalNodeRepresentationArrayToByteArray(nodes []NodeRepresentation) []byte {
	var byteArray []byte

	msgType := make([]byte, 4)
	binary.BigEndian.PutUint32(msgType, uint32(MESSAGE_TYPE_PUBLICKEYS))
	byteArray = append(byteArray, msgType...)

	for i:=0; i<len(nodes); i++ {
		publicKeysBytes, err := nodes[i].PublicKey.MarshalBinary()
		publicKeyLength := make([]byte, 4)
		binary.BigEndian.PutUint32(publicKeyLength, uint32(len(publicKeysBytes)))

		byteArray = append(byteArray, publicKeyLength...)
		byteArray = append(byteArray, publicKeysBytes...)

		if err != nil{
			panic("can't marshal client public key n°"+strconv.Itoa(i))
		}
	}

	return byteArray
}
func BroadcastMessage(nodes []NodeRepresentation, message []byte) {
	fmt.Println(hex.Dump(message))

	for i:=0; i<len(nodes); i++ {
		if  nodes[i].Connected {
			n, err := nodes[i].Conn.Write(message)

			//fmt.Println("[", nodes[i].Conn.LocalAddr(), " - ", nodes[i].Conn.RemoteAddr(), "]")

			if n < len(message) || err != nil {
				fmt.Println("Could not broadcast to conn", i, "gonna set it to disconnected.")
				nodes[i].Connected = false
			}
		}
	}
}