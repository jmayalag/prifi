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
	//"github.com/lbarman/prifi/util"
)

type RelayState struct {
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

func startRelay(payloadLength int, relayPort string, nClients int, nTrustees int, trusteesIp []string, reportingLimit int) {

	var relayState *RelayState
	relayState = initiateRelayState(relayPort, nTrustees, nClients, payloadLength, reportingLimit, trusteesIp)

	//start the server waiting for clients
	newClientConnectionsChan        := make(chan net.Conn) 					//channel with unparsed clients
	go relayServerListener(relayPort, newClientConnectionsChan)

	//start the client parser
	newClientWithIdAndPublicKeyChan := make(chan NodeRepresentation)  //channel with parsed clients
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
	var endOfProtocolState int
	newClients := make([]NodeRepresentation, 0)

	for {
		select {
			case protocolHasFailed := <- protocolFailed:
				//re-run setup, something went wrong
				fmt.Println(protocolHasFailed)
				fmt.Println("protocolHasFailed")

			case newClient := <- newClientWithIdAndPublicKeyChan:
				//we tell processMessageLoop to stop ASAP
				fmt.Println("newClientWithIdAndPublicKeyChan")
				newClients = append(newClients, newClient)
				indicateEndOfProtocol <- 1

			case endOfProtocolState = <- indicateEndOfProtocol:
				fmt.Println("indicateEndOfProtocol")
				if endOfProtocolState != 2 {
					panic("something went wrong, should not happen")
				}

				fmt.Println("oooooooooooooooooooooooooooooooo")
				for i := 0; i<len(relayState.clients); i++ {
					fmt.Println(relayState.clients[i].Id, " - ", relayState.clients[i].PublicKey, " - ", relayState.clients[i].Conn)
				}
				fmt.Println("oooooooooooooooooooooooooooooooo")

				//1. copy the previous relayState
				newRelayState := relayState.clone() 
				
				//2. disconnect the trustees (but not the clients)
				relayState.disconnectFromAllTrustees()
				//3. compose new client list
				for i:=0; i<len(newClients); i++{
					newRelayState.addNewClient(newClients[i])
				}
				newClients = make([]NodeRepresentation, 0)


				fmt.Println("*******************************")
				for i := 0; i<len(newRelayState.clients); i++ {
					fmt.Println(newRelayState.clients[i].Id, " - ", newRelayState.clients[i].PublicKey, " - ", newRelayState.clients[i].Conn)
				}
				fmt.Println("*******************************")



				//4. reconnect to trustees
				newRelayState.connectToAllTrustees()
				//5. exchange the public keys 
				newRelayState.advertisePublicKeys()

				println("Client should be OK ...")
				time.Sleep(5*time.Second)
				//6. process message loop (on the new relayState)
				go newRelayState.processMessageLoop(protocolFailed, indicateEndOfProtocol)

				//replace old state
				relayState = newRelayState

			default: 
				//all clear!
				fmt.Println("timer1000")
				time.Sleep(1000 * time.Millisecond)
		}
	}
}

func (relayState *RelayState) connectToAllTrustees() {
	//connect to the trustees
	for i:= 0; i < relayState.nTrustees; i++ {
		connectToTrustee(i, relayState.trusteesHosts[i], relayState)
	}
	fmt.Println("Trustees connecting done, ", len(relayState.trustees), "trustees connected")
}

func (relayState *RelayState) disconnectFromAllTrustees() {
	//disconnect to the trustees
	for i:= 0; i < relayState.nTrustees; i++ {
		relayState.trustees[i].Conn.Close()
	}
	fmt.Println("Trustees connecting done, ", len(relayState.trustees), "trustees connected")
}

func (relayState *RelayState) waitForDefaultNumberOfClients(newClientConnectionsChan chan NodeRepresentation) {
	currentClients := 0

	fmt.Printf("Waiting for %d clients (on port %s)\n", relayState.nClients - currentClients, relayState.RelayPort)

	for currentClients < relayState.nClients {
		select{
				case newClient := <-newClientConnectionsChan: 
					relayState.clients = append(relayState.clients, newClient)

					currentClients += 1
					fmt.Printf("Waiting for %d clients (on port %s)\n", relayState.nClients - currentClients, relayState.RelayPort)
				default: 
					time.Sleep(100 * time.Millisecond)
		}
	}
	fmt.Println("Client connecting done, ", len(relayState.clients), "clients connected")
}

func (relayState *RelayState) clone() *RelayState{

	//count the clients that disconnected
	nClientsDisconnected := 0
	for i := 0; i<len(relayState.clients); i++ {
		if !relayState.clients[i].Connected {
			fmt.Println("CLIENT ", i, " IS NOT COPIED, DISCONNECTED")
			nClientsDisconnected++
		}
	}

	//count the actual number of clients, and init the new state with the old parameters
	newNClients   := relayState.nClients - nClientsDisconnected
	newRelayState := initiateRelayState(relayState.RelayPort, relayState.nTrustees, newNClients, relayState.PayloadLength, relayState.ReportingLimit, relayState.trusteesHosts)

	//copy the connected clients
	newRelayState.clients = make([]NodeRepresentation, newNClients)
	j := 0
	for i := 0; i<len(relayState.clients); i++ {
		if relayState.clients[i].Connected {
			newRelayState.clients[j] = relayState.clients[i]
			j++
		}
	}

	return newRelayState
}

func (relayState *RelayState) addNewClient(newClient NodeRepresentation){
	relayState.nClients = relayState.nClients + 1
	relayState.clients  = append(relayState.clients, newClient)
}

func (relayState *RelayState) advertisePublicKeys(){
	//Prepare the messages
	println("Preparing trustee array")
	dataForClients   := MarshalNodeRepresentationArrayToByteArray(relayState.trustees)
	println("Preparing client array")
			println(len(relayState.clients))
			for i:=0; i<len(relayState.clients); i++{
				fmt.Println(relayState.clients)
			}
	dataForTrustees := MarshalNodeRepresentationArrayToByteArray(relayState.clients)


			println("<<<<<<<<<<<<<<<<<<")
			println("Data has size")
			println(len(dataForClients))

	//craft the message for clients
	messageForClientsLength := len(dataForClients)
	messageForClients := make([]byte, 10+messageForClientsLength)
	binary.BigEndian.PutUint32(messageForClients[0:4], uint32(2)) //message type //TODO make this less ugly
	binary.BigEndian.PutUint32(messageForClients[4:8], uint32(0)) //socks ID
	binary.BigEndian.PutUint16(messageForClients[8:10], uint16(messageForClientsLength))
	copy(messageForClients[10:], dataForClients)

	//broadcast to the clients
	BroadcastMessage(relayState.clients, messageForClients)
	BroadcastMessage(relayState.trustees, dataForTrustees)
	fmt.Println("Advertising done, to", len(relayState.clients), "clients and", len(relayState.trustees), "trustees")
}


func (relayState *RelayState) processMessageLoop(protocolFailed chan bool, indicateEndOfProtocol chan int){
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

	currentSetupContinues := true
	
	for currentSetupContinues {
		println("<")

		//if the main thread tells us to stop (for re-setup)
		tellClientsToResync := false
		var mainThreadStatus int
		select {
			case mainThreadStatus = <- indicateEndOfProtocol:
				if mainThreadStatus == 1 {
					println("Main thread status is 1, gonna warn the clients")
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
			fmt.Println("Telling clients to resync")
			msgType = 1
			currentSetupContinues = false
		}

		//craft the message for clients
		downstreamDataPayloadLength := len(downbuffer.data)
		downstreamData := make([]byte, 10+downstreamDataPayloadLength)
		binary.BigEndian.PutUint32(downstreamData[0:4], uint32(msgType))
		binary.BigEndian.PutUint32(downstreamData[4:8], uint32(downbuffer.connectionId)) //this is the SOCKS connection ID
		binary.BigEndian.PutUint16(downstreamData[8:10], uint16(downstreamDataPayloadLength))
		copy(downstreamData[10:], downbuffer.data)


		fmt.Println("Writing a message with type", msgType, " socks id ", downbuffer.connectionId)

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
				fmt.Println("Trustee "+strconv.Itoa(i)+" deconnected")
			}
			if n < trusteePayloadLength {
				errorInThisCell                  = true
				relayState.trustees[i].Connected = false
				fmt.Println("Relay : Read from trustee failed, read "+strconv.Itoa(n)+" where "+strconv.Itoa(trusteePayloadLength)+" was expected: " + err.Error())
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
				fmt.Println("Client "+strconv.Itoa(i)+" deconnected")
				errorInThisCell                 = true
				relayState.clients[i].Connected = false
			}
			if n < clientPayloadLength {
				errorInThisCell                 = true
				relayState.clients[i].Connected = false
				fmt.Println("Relay : Read from client failed, read "+strconv.Itoa(n)+" where "+strconv.Itoa(trusteePayloadLength)+" was expected: " + err.Error())
			}

			relayState.CellCoder.DecodeClient(clientsPayloadData[i])
		}

		if errorInThisCell {
			//craft the message for clients
			downstreamData := make([]byte, 10)
			binary.BigEndian.PutUint32(downstreamData[0:4], uint32(3))
			binary.BigEndian.PutUint32(downstreamData[4:8], uint32(downbuffer.connectionId)) //this is the SOCKS connection ID
			binary.BigEndian.PutUint16(downstreamData[8:10], uint16(0))
			BroadcastMessage(relayState.clients, downstreamData)
			break
		} else {
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

	println("Main loop broken, waiting 5 sec")
	time.Sleep(5*time.Second)

	indicateEndOfProtocol <- 2
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

	//sets the cell coder, and the history
	params.CellCoder = factory()

	return params
}

func welcomeNewClients(newConnectionsChan chan net.Conn, newClientChan chan NodeRepresentation, relayState *RelayState) {	
	newClientsToParse := make(chan NodeRepresentation)

	for {
		select{
			//accept the TCP connection, and parse the parameters
			case newConnection := <-newConnectionsChan: 
				go relayParseClientParams(newConnection, relayState, newClientsToParse)
			
			//once client is ready (we have params+pk), forward to the other channel
			case newClient := <-newClientsToParse: 
				fmt.Println("New client is ready !")
				fmt.Println(newClient)
				newClientChan <- newClient
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
	newTrustee := NodeRepresentation{trusteeId, conn, true, publicKey}

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

func relayParseClientParamsAux(conn net.Conn, relayState *RelayState) NodeRepresentation {
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

	publicKey := suite.Point()
	err3 := publicKey.UnmarshalBinary(keyBytes)

	if err3 != nil {
		panic(">>>>  Relay : can't unmarshal client key ! " + err3.Error())
	}

	newClient := NodeRepresentation{nodeId, conn, true, publicKey}

	return newClient
}

func relayParseClientParams(conn net.Conn, relayState *RelayState, newClientChan chan NodeRepresentation) {

	newClient := relayParseClientParamsAux(conn, relayState)
	newClientChan <- newClient
}


// TODO : this should be somewhere else
func MarshalNodeRepresentationArrayToByteArray(nodes []NodeRepresentation) []byte {
	var byteArray []byte

	msgType := make([]byte, 4)
	binary.BigEndian.PutUint32(msgType, uint32(2))
	byteArray = append(byteArray, msgType...)

	for i:=0; i<len(nodes); i++ {
		publicKeysBytes, err := nodes[i].PublicKey.MarshalBinary()
		publicKeyLength := make([]byte, 4)
		binary.BigEndian.PutUint32(publicKeyLength, uint32(len(publicKeysBytes)))

		fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>> Adding key ", i)

		byteArray = append(byteArray, publicKeyLength...)
		byteArray = append(byteArray, publicKeysBytes...)

		//fmt.Println(hex.Dump(publicKeysBytes))
		if err != nil{
			panic("can't marshal client public key n°"+strconv.Itoa(i))
		}
	}

	return byteArray
}
func BroadcastMessage(nodes []NodeRepresentation, message []byte) {
	fmt.Println("Gonna broadcast this message")
	fmt.Println(hex.Dump(message))

	for i:=0; i<len(nodes); i++ {
		if  nodes[i].Connected {
			n, err := nodes[i].Conn.Write(message)

			fmt.Println("[", nodes[i].Conn.LocalAddr(), " - ", nodes[i].Conn.RemoteAddr(), "]")

			if n < len(message) || err != nil {
				fmt.Println("Could not broadcast to conn", i, "gonna set it to disconnected.")
				nodes[i].Connected = false
			}
		}
	}
}