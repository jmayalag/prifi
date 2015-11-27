package relay

import (
	"encoding/binary"
	"fmt"
	"github.com/lbarman/prifi/config"
	"github.com/lbarman/crypto/abstract"
	"time"
	"log"
	"net"
	prifinet "github.com/lbarman/prifi/net"
	prifilog "github.com/lbarman/prifi/log"
)

var relayState 			*RelayState 
var stateMachineLogger 	*prifilog.StateMachineLogger

var	protocolFailed        = make(chan bool)
var	indicateEndOfProtocol = make(chan int)
var	deconnectedClients	  = make(chan int)
var	timedOutClients   	  = make(chan int)
var	deconnectedTrustees	  = make(chan int)

func StartRelay(payloadLength int, relayPort string, nClients int, nTrustees int, trusteesIp []string, reportingLimit int) {

	stateMachineLogger = prifilog.NewStateMachineLogger()
	stateMachineLogger.StateChange("relay-init")

	relayState = initiateRelayState(relayPort, nTrustees, nClients, payloadLength, reportingLimit, trusteesIp)

	//start the server waiting for clients
	newClientConnectionsChan        := make(chan net.Conn) 	          //channel with unparsed clients
	go relayServerListener(relayPort, newClientConnectionsChan)

	//start the client parser
	newClientWithIdAndPublicKeyChan := make(chan prifinet.NodeRepresentation)  //channel with parsed clients
	go welcomeNewClients(newClientConnectionsChan, newClientWithIdAndPublicKeyChan)

	stateMachineLogger.StateChange("protocol-setup")

	//start the actual protocol
	relayState.connectToAllTrustees()
	relayState.waitForDefaultNumberOfClients(newClientWithIdAndPublicKeyChan)
	relayState.advertisePublicKeys()	
	relayState.organizeRoundScheduling()

	//copy for subtrhead
	relayStateCopy := relayState.deepClone()
	go processMessageLoop(relayStateCopy)
	var isProtocolRunning = true

	//control loop
	var endOfProtocolState int
	newClients := make([]prifinet.NodeRepresentation, 0)

	for {

		select {
			case protocolHasFailed := <- protocolFailed:
				fmt.Println(protocolHasFailed)
				fmt.Println("Relay Handler : Processing loop has failed")
				isProtocolRunning = false
				//TODO : re-run setup, something went wrong. Maybe restart from 0 ?

			case deconnectedClient := <- deconnectedClients:
				fmt.Println("Client", deconnectedClient, " has been indicated offline")
				relayState.clients[deconnectedClient].Connected = false

			case timedOutClient := <- timedOutClients:
				fmt.Println("Client", timedOutClient, " has been indicated offline (time out)")
				relayState.clients[timedOutClient].Conn.Close()
				relayState.clients[timedOutClient].Connected = false

			case deconnectedTrustee := <- deconnectedTrustees:
				fmt.Println("Trustee", deconnectedTrustee, " has been indicated offline")

			case newClient := <- newClientWithIdAndPublicKeyChan:
				//we tell processMessageLoop to stop when possible
				newClients = append(newClients, newClient)
				if isProtocolRunning {
					fmt.Println("Relay Handler : new Client is ready, stopping processing loop")
					indicateEndOfProtocol <- PROTOCOL_STATUS_GONNA_RESYNC
				} else {
					fmt.Println("Relay Handler : new Client is ready, restarting processing loop")
					isProtocolRunning = restartProtocol(relayState, newClients)
					newClients = make([]prifinet.NodeRepresentation, 0)
					fmt.Println("Done...")
				}

			case endOfProtocolState = <- indicateEndOfProtocol:
				fmt.Println("Relay Handler : main loop stopped, resyncing")

				if endOfProtocolState != PROTOCOL_STATUS_RESYNCING {
					panic("something went wrong, should not happen")
				}

				isProtocolRunning = restartProtocol(relayState, newClients)
				newClients = make([]prifinet.NodeRepresentation, 0)
			default: 
				//all clear! keep this thread handler load low, (accept changes every X millisecond)
				time.Sleep(CONTROL_LOOP_SLEEP_TIME)
		}
	}
}

func restartProtocol(relayState *RelayState, newClients []prifinet.NodeRepresentation) bool {
	relayState.excludeDisconnectedClients() 				
	relayState.disconnectFromAllTrustees()

	//add the new clients to the previous (filtered) list
	for i:=0; i<len(newClients); i++{
		relayState.addNewClient(newClients[i])
		fmt.Println("Adding new client")
		fmt.Println(newClients[i])
	}
	relayState.nClients = len(relayState.clients)

	//if we dont have enough client, stop.
	if len(relayState.clients) == 0{
		fmt.Println("Relay Handler : not enough client, stopping and waiting...")
		return false
	} else {
		//re-advertise the configuration 	
		relayState.connectToAllTrustees()
		relayState.advertisePublicKeys()

		time.Sleep(INBETWEEN_CONFIG_SLEEP_TIME)

		//process message loop
		relayStateCopy := relayState.deepClone()
		go processMessageLoop(relayStateCopy)

		return true
	}
}

func (relayState *RelayState) advertisePublicKeys(){
	//Prepare the messages
	dataForClients   := prifinet.MarshalNodeRepresentationArrayToByteArray(relayState.trustees)
	dataForTrustees := prifinet.MarshalNodeRepresentationArrayToByteArray(relayState.clients)

	//craft the message for clients
	messageForClientsLength := len(dataForClients)
	messageForClients := make([]byte, 10+messageForClientsLength)
	binary.BigEndian.PutUint32(messageForClients[0:4], uint32(prifinet.MESSAGE_TYPE_PUBLICKEYS))
	binary.BigEndian.PutUint32(messageForClients[4:8], uint32(relayState.nClients))
	binary.BigEndian.PutUint16(messageForClients[8:10], uint16(messageForClientsLength))
	copy(messageForClients[10:], dataForClients)

	//TODO : would be cleaner if the trustees used the same structure for the message

	//broadcast to the clients
	prifinet.BroadcastMessageToNodes(relayState.clients, messageForClients)
	prifinet.BroadcastMessageToNodes(relayState.trustees, dataForTrustees)
	fmt.Println("Advertising done, to", len(relayState.clients), "clients and", len(relayState.trustees), "trustees")
}

func (relayState *RelayState) organizeRoundScheduling(){

	ephPublicKey := make([]abstract.Point, relayState.nClients)

	for i := 0; i < relayState.nClients; i++ {

		buffer := make([]byte, 512)
		_, err := relayState.clients[i].Conn.Read(buffer)
		if err != nil {
			panic("Read error:" + err.Error())
		}

		keySize := int(binary.BigEndian.Uint32(buffer[8:12]))
		keyBytes := buffer[12:(12+keySize)] 

		publicKey := config.CryptoSuite.Point()
		err2 := publicKey.UnmarshalBinary(keyBytes)

		if err2 != nil {
			panic(">>>>  Relay : can't unmarshal ephemeral client key ! " + err2.Error())
		}

		ephPublicKey[i] = publicKey
	}

	fmt.Println("Relay: collected all ephemeral public keys")

	G := config.CryptoSuite.Point().Base()
	for j := 0; j < relayState.nTrustees; j++ {

		prifinet.WriteBaseAndPublicKeyToConn(relayState.trustees[j].Conn, G, ephPublicKey)
		
		fmt.Println(G)
		for i := 0; i < relayState.nClients; i++ {
			fmt.Println(ephPublicKey[i])
		}

		fmt.Println("Trustee", j, "is shuffling...")

		base2, ephPublicKeys2, proof := prifinet.ParseBasePublicKeysAndProofFromConn(relayState.trustees[j].Conn)

		fmt.Println("Trustee", j, "is done shuffling")
		fmt.Println(base2)
		for i := 0; i < relayState.nClients; i++ {
			fmt.Println(ephPublicKeys2[i])
		}
		fmt.Println(proof)

		for {
			fmt.Println("all done, waiting forever")
			time.Sleep(5 * time.Second)
		}
	}

	for {
		fmt.Println("all done, waiting forever")
		time.Sleep(5 * time.Second)
	}
}


func processMessageLoop(relayState *RelayState){
	//TODO : if something fail, send true->protocolFailed

	stateMachineLogger.StateChange("protocol-mainloop")

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

	stats := prifilog.EmptyStatistics(relayState.ReportingLimit)

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

	socksProxyConnections := make(map[int]chan<- []byte)
	downstream            := make(chan prifinet.DataWithConnectionId)
	nulldown              := prifinet.DataWithConnectionId{} // default empty downstream cell
	window                := 2           // Maximum cells in-flight
	inflight              := 0         // Current cells in-flight

	currentSetupContinues := true
	
	for currentSetupContinues {

		fmt.Println(".")

		//if needed, we bound the number of round per second
		time.Sleep(INBETWEEN_ROUND_SLEEP_TIME)

		//if the main thread tells us to stop (for re-setup)
		tellClientsToResync := false
		var mainThreadStatus int
		select {
			case mainThreadStatus = <- indicateEndOfProtocol:
				if mainThreadStatus == PROTOCOL_STATUS_GONNA_RESYNC {
					fmt.Println("Main thread status is 1, gonna warn the clients")
					tellClientsToResync = true
				}
			default:
		}

		//we report the speed, bytes exchanged, etc
		stats.Report()
		if stats.ReportingDone() {
			fmt.Println("Reporting limit matched; exiting the relay")
			break;
		}

		// See if there's any downstream data to forward.
		var downbuffer prifinet.DataWithConnectionId
		select {
			case downbuffer = <-downstream: // some data to forward downstream
			default: 
				downbuffer = nulldown
		}

		//compute the message type; if MESSAGE_TYPE_DATA_AND_RESYNC, the clients know they will resync
		msgType := prifinet.MESSAGE_TYPE_DATA
		if tellClientsToResync{
			msgType = prifinet.MESSAGE_TYPE_DATA_AND_RESYNC
			currentSetupContinues = false
		}

		//craft the message for clients
		downstreamDataPayloadLength := len(downbuffer.Data)
		downstreamData := make([]byte, 10+downstreamDataPayloadLength)
		binary.BigEndian.PutUint32(downstreamData[0:4], uint32(msgType))
		binary.BigEndian.PutUint32(downstreamData[4:8], uint32(downbuffer.ConnectionId)) //this is the SOCKS connection ID
		binary.BigEndian.PutUint16(downstreamData[8:10], uint16(downstreamDataPayloadLength))
		copy(downstreamData[10:], downbuffer.Data)

		// Broadcast the downstream data to all clients.
		prifinet.BroadcastMessageToNodes(relayState.clients, downstreamData)
		stats.AddDownstreamCell(int64(downstreamDataPayloadLength))

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

			//TODO : add a channel for timeout trustee
			data, err := prifinet.ReadWithTimeOut(i, relayState.trustees[i].Conn, trusteePayloadLength, CLIENT_READ_TIMEOUT, deconnectedTrustees, deconnectedTrustees)

			if err {
				errorInThisCell = true
			}

			relayState.CellCoder.DecodeTrustee(data)
		}

		// Collect an upstream ciphertext from each client
		for i := 0; i < relayState.nClients; i++ {

			if errorInThisCell {
				break
			}

			data, err := prifinet.ReadWithTimeOut(i, relayState.clients[i].Conn, clientPayloadLength, CLIENT_READ_TIMEOUT, timedOutClients, deconnectedClients)

			if err {
				errorInThisCell = true
			}

			relayState.CellCoder.DecodeClient(data)
		}

		if errorInThisCell {
			
			fmt.Println("Relay main loop : Cell will be invalid, some party disconnected. Warning the clients...")

			//craft the message for clients
			downstreamData := make([]byte, 10)
			binary.BigEndian.PutUint32(downstreamData[0:4], uint32(3))
			binary.BigEndian.PutUint32(downstreamData[4:8], uint32(downbuffer.ConnectionId)) //this is the SOCKS connection ID
			binary.BigEndian.PutUint16(downstreamData[8:10], uint16(0))
			prifinet.BroadcastMessageToNodes(relayState.clients, downstreamData)

			break
		} else {
			upstreamPlaintext := relayState.CellCoder.DecodeCell()
			inflight--

			stats.AddUpstreamCell(int64(relayState.PayloadLength))

			// Process the decoded cell
			if upstreamPlaintext == nil {
				continue // empty or corrupt upstream cell
			}
			if len(upstreamPlaintext) != relayState.PayloadLength {
				panic("DecodeCell produced wrong-size payload")
			}

			// Decode the upstream cell header (may be empty, all zeros)
			socksConnId     := int(binary.BigEndian.Uint32(upstreamPlaintext[0:4]))
			socksDataLength := int(binary.BigEndian.Uint16(upstreamPlaintext[4:6]))

			if socksConnId == prifinet.SOCKS_CONNECTION_ID_EMPTY {
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
				continue
			}

			socksConn <- upstreamPlaintext[6 : 6+socksDataLength]
		}
	}

	fmt.Println("Relay main loop : waiting ",INBETWEEN_CONFIG_SLEEP_TIME," seconds, client should now be waiting for new parameters...")
	time.Sleep(INBETWEEN_CONFIG_SLEEP_TIME)

	indicateEndOfProtocol <- PROTOCOL_STATUS_RESYNCING

	stateMachineLogger.StateChange("protocol-resync")
}

func newSOCKSProxyHandler(connId int, downstreamData chan<- prifinet.DataWithConnectionId) chan<- []byte {
	upstreamData := make(chan []byte)
	go prifinet.RelaySocksProxy(connId, upstreamData, downstreamData)
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
	binary.BigEndian.PutUint32(buffer[0:4], uint32(config.LLD_PROTOCOL_VERSION))
	binary.BigEndian.PutUint32(buffer[4:8], uint32(relayState.PayloadLength))
	binary.BigEndian.PutUint32(buffer[8:12], uint32(relayState.nClients))
	binary.BigEndian.PutUint32(buffer[12:16], uint32(relayState.nTrustees))
	binary.BigEndian.PutUint32(buffer[16:20], uint32(trusteeId))

	fmt.Println("Writing", config.LLD_PROTOCOL_VERSION, "setup is", relayState.nClients, relayState.nTrustees, "role is", trusteeId, "cellSize ", relayState.PayloadLength)

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
	publicKey := config.CryptoSuite.Point()
	err3 := publicKey.UnmarshalBinary(keyBytes)

	if err3 != nil {
		panic(">>>>  Relay : can't unmarshal trustee key ! " + err2.Error())
	}

	fmt.Println("Trustee", trusteeId, "is connected.")
	
	newTrustee := prifinet.NodeRepresentation{trusteeId, conn, true, publicKey}

	//side effects
	relayState.trustees = append(relayState.trustees, newTrustee)
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

func relayParseClientParamsAux(conn net.Conn) prifinet.NodeRepresentation {
	buffer := make([]byte, 512)
	_, err2 := conn.Read(buffer)
	if err2 != nil {
		panic("Read error:" + err2.Error())
	}

	version := int(binary.BigEndian.Uint32(buffer[0:4]))

	if(version != config.LLD_PROTOCOL_VERSION) {
		fmt.Println(">>>> Relay client version", version, "!= relay version", config.LLD_PROTOCOL_VERSION)
		panic("fatal error")
	}

	nodeId := int(binary.BigEndian.Uint32(buffer[4:8]))

	//check that the node ID is not used
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
	if idExists {
		fmt.Println("Client with ID ", nodeId, "tried to connect, but some client already took that ID. changing ID to", nextFreeId)
		nodeId = nextFreeId
	}

	keySize := int(binary.BigEndian.Uint32(buffer[8:12]))
	keyBytes := buffer[12:(12+keySize)] 

	publicKey := config.CryptoSuite.Point()
	err3 := publicKey.UnmarshalBinary(keyBytes)

	if err3 != nil {
		panic(">>>>  Relay : can't unmarshal client key ! " + err3.Error())
	}

	newClient := prifinet.NodeRepresentation{nodeId, conn, true, publicKey}

	return newClient
}

func relayParseClientParams(conn net.Conn, newClientChan chan prifinet.NodeRepresentation) {

	newClient := relayParseClientParamsAux(conn)
	newClientChan <- newClient
}