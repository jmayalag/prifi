package relay

import (
	"encoding/binary"
	"encoding/hex"
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

	prifilog.SimpleStringDump("Relay started")

	stateMachineLogger = prifilog.NewStateMachineLogger("relay")
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
	err := relayState.organizeRoundScheduling()

	var isProtocolRunning = false
	if err != nil {
		prifilog.Println("Relay Handler : round scheduling went wrong, restarting the configuration protocol")

		//disconnect all clients
		for i:=0; i<len(relayState.clients); i++{
			relayState.clients[i].Conn.Close()
			relayState.clients[i].Connected = false
		}	
		restartProtocol(relayState, make([]prifinet.NodeRepresentation, 0));
	} else {
		//copy for subtrhead
		relayStateCopy := relayState.deepClone()
		go processMessageLoop(relayStateCopy)
		isProtocolRunning = true
	}

	//control loop
	var endOfProtocolState int
	newClients := make([]prifinet.NodeRepresentation, 0)

	for {

		select {
			case protocolHasFailed := <- protocolFailed:
				prifilog.Println(protocolHasFailed)
				prifilog.Println("Relay Handler : Processing loop has failed")
				isProtocolRunning = false
				//TODO : re-run setup, something went wrong. Maybe restart from 0 ?

			case deconnectedClient := <- deconnectedClients:
				prifilog.Println("Client", deconnectedClient, " has been indicated offline")
				relayState.clients[deconnectedClient].Connected = false

			case timedOutClient := <- timedOutClients:
				prifilog.Println("Client", timedOutClient, " has been indicated offline (time out)")
				relayState.clients[timedOutClient].Conn.Close()
				relayState.clients[timedOutClient].Connected = false

			case deconnectedTrustee := <- deconnectedTrustees:
				prifilog.Println("Trustee", deconnectedTrustee, " has been indicated offline")

			case newClient := <- newClientWithIdAndPublicKeyChan:
				//we tell processMessageLoop to stop when possible
				newClients = append(newClients, newClient)
				if isProtocolRunning {
					prifilog.Println("Relay Handler : new Client is ready, stopping processing loop")
					indicateEndOfProtocol <- PROTOCOL_STATUS_GONNA_RESYNC
				} else {
					prifilog.Println("Relay Handler : new Client is ready, restarting processing loop")
					isProtocolRunning = restartProtocol(relayState, newClients)
					newClients = make([]prifinet.NodeRepresentation, 0)
					prifilog.Println("Done...")
				}

			case endOfProtocolState = <- indicateEndOfProtocol:
				prifilog.Println("Relay Handler : main loop stopped, resyncing")

				if endOfProtocolState != PROTOCOL_STATUS_RESYNCING {
					panic("something went wrong, should not happen")
				}

				isProtocolRunning = restartProtocol(relayState, newClients)
				newClients = make([]prifinet.NodeRepresentation, 0)
			default: 
				//all clear! keep this thread handler load low, (accept changes every X millisecond)
				time.Sleep(CONTROL_LOOP_SLEEP_TIME)
				//prifilog.StatisticReport("relay", "CONTROL_LOOP_SLEEP_TIME", CONTROL_LOOP_SLEEP_TIME.String())
		}
	}
}

func restartProtocol(relayState *RelayState, newClients []prifinet.NodeRepresentation) bool {
	relayState.excludeDisconnectedClients() 				
	relayState.disconnectFromAllTrustees()

	//add the new clients to the previous (filtered) list
	for i:=0; i<len(newClients); i++{
		relayState.addNewClient(newClients[i])
		prifilog.Println("Adding new client")
		prifilog.Println(newClients[i])
	}
	relayState.nClients = len(relayState.clients)

	//if we dont have enough client, stop.
	if len(relayState.clients) == 0{
		prifilog.Println("Relay Handler : not enough client, stopping and waiting...")
		return false
	} else {
		//re-advertise the configuration 	
		relayState.connectToAllTrustees()
		relayState.advertisePublicKeys()
		err := relayState.organizeRoundScheduling()
		if err != nil {
			prifilog.Println("Relay Handler : round scheduling went wrong, restarting the configuration protocol")

			//disconnect all clients
			for i:=0; i<len(relayState.clients); i++{
				relayState.clients[i].Conn.Close()
				relayState.clients[i].Connected = false
			}	
			return restartProtocol(relayState, make([]prifinet.NodeRepresentation, 0));
		}

		time.Sleep(INBETWEEN_CONFIG_SLEEP_TIME)
		prifilog.StatisticReport("relay", "INBETWEEN_CONFIG_SLEEP_TIME", INBETWEEN_CONFIG_SLEEP_TIME.String())

		//process message loop
		relayStateCopy := relayState.deepClone()
		go processMessageLoop(relayStateCopy)

		return true
	}
}

func (relayState *RelayState) advertisePublicKeys() error{
	defer prifilog.TimeTrack("relay", "advertisePublicKeys", time.Now())

	//Prepare the messages
	dataForClients, err  := prifinet.MarshalNodeRepresentationArrayToByteArray(relayState.trustees)

	if err != nil {
		return err
	}

	dataForTrustees, err := prifinet.MarshalNodeRepresentationArrayToByteArray(relayState.clients)

	if err != nil {
		return err
	}

	//craft the message for clients
	messageForClients := make([]byte, 6 + len(dataForClients))
	binary.BigEndian.PutUint16(messageForClients[0:2], uint16(prifinet.MESSAGE_TYPE_PUBLICKEYS))
	binary.BigEndian.PutUint32(messageForClients[2:6], uint32(relayState.nClients))
	copy(messageForClients[6:], dataForClients)

	//TODO : would be cleaner if the trustees used the same structure for the message

	//broadcast to the clients
	prifinet.NUnicastMessageToNodes(relayState.clients, messageForClients)
	prifinet.NUnicastMessageToNodes(relayState.trustees, dataForTrustees)
	prifilog.Println("Advertising done, to", len(relayState.clients), "clients and", len(relayState.trustees), "trustees")

	return nil
}

func (relayState *RelayState) organizeRoundScheduling() error {
	defer prifilog.TimeTrack("relay", "organizeRoundScheduling", time.Now())

	ephPublicKeys := make([]abstract.Point, relayState.nClients)

	//collect ephemeral keys
	prifilog.Println("Waiting for", relayState.nClients, "ephemeral keys")
	for i := 0; i < relayState.nClients; i++ {
		ephPublicKeys[i] = nil
		for ephPublicKeys[i] == nil {

			pkRead := false
			var pk abstract.Point = nil

			for !pkRead {

				buffer, err := prifinet.ReadMessage(relayState.clients[i].Conn)
				publicKey := config.CryptoSuite.Point()
				msgType := int(binary.BigEndian.Uint16(buffer[0:2]))

				if msgType == prifinet.MESSAGE_TYPE_PUBLICKEYS {
					err2 := publicKey.UnmarshalBinary(buffer[2:])

					if err2 != nil {
						prifilog.Println("Reading client", i, "ephemeral key")
						return err
					}
					pk = publicKey
					break

				} else if msgType != prifinet.MESSAGE_TYPE_PUBLICKEYS {
					//append data in the buffer
					prifilog.Println("organizeRoundScheduling: trying to read a public key message, got a data message; discarding, checking for public key in next message...")
					continue
				}
			}

			ephPublicKeys[i] = pk
		}
	}

	prifilog.Println("Relay: collected all ephemeral public keys")

	// prepare transcript
	G_s             := make([]abstract.Point, relayState.nTrustees)
	ephPublicKeys_s := make([][]abstract.Point, relayState.nTrustees)
	proof_s         := make([][]byte, relayState.nTrustees)

	//ask each trustee in turn to do the oblivious shuffle
	G := config.CryptoSuite.Point().Base()
	for j := 0; j < relayState.nTrustees; j++ {

		prifinet.WriteBaseAndPublicKeyToConn(relayState.trustees[j].Conn, G, ephPublicKeys)
		prifilog.Println("Trustee", j, "is shuffling...")

		base2, ephPublicKeys2, proof, err := prifinet.ParseBasePublicKeysAndProofFromConn(relayState.trustees[j].Conn)

		if err != nil {
			return err
		}

		prifilog.Println("Trustee", j, "is done shuffling")

		//collect transcript
		G_s[j]             = base2
		ephPublicKeys_s[j] = ephPublicKeys2
		proof_s[j]         = proof

		//next trustee get this trustee's output
		G            = base2
		ephPublicKeys = ephPublicKeys2
	}

	prifilog.Println("All trustees have shuffled, sending the transcript...")

	//pack transcript
	transcriptBytes := make([]byte, 0)
	for i:=0; i<len(G_s); i++ {
		G_s_i_bytes, _ := G_s[i].MarshalBinary()
		transcriptBytes = append(transcriptBytes, prifinet.IntToBA(len(G_s_i_bytes))...)
		transcriptBytes = append(transcriptBytes, G_s_i_bytes...)

		prifilog.Println("G_S_", i)
		prifilog.Println(hex.Dump(G_s_i_bytes))
	}
	for i:=0; i<len(ephPublicKeys_s); i++ {

		pkArray := make([]byte, 0)
		for k:=0; k<len(ephPublicKeys_s[i]); k++{
			pkBytes, _ := ephPublicKeys_s[i][k].MarshalBinary()
			pkArray = append(pkArray, prifinet.IntToBA(len(pkBytes))...)
			pkArray = append(pkArray, pkBytes...)
			prifilog.Println("Packing key", k)
		}

		transcriptBytes = append(transcriptBytes, prifinet.IntToBA(len(pkArray))...)
		transcriptBytes = append(transcriptBytes, pkArray...)

		prifilog.Println("pkArray_", i)
		prifilog.Println(hex.Dump(pkArray))
	}
	for i:=0; i<len(proof_s); i++ {
		transcriptBytes = append(transcriptBytes, prifinet.IntToBA(len(proof_s[i]))...)
		transcriptBytes = append(transcriptBytes, proof_s[i]...)

		prifilog.Println("G_S_", i)
		prifilog.Println(hex.Dump(proof_s[i]))
	}

	//broadcast to trustees
	prifinet.NUnicastMessageToNodes(relayState.trustees, transcriptBytes)

	//wait for the signature for each trustee
	signatures := make([][]byte, relayState.nTrustees)
	for j := 0; j < relayState.nTrustees; j++ {
 
 		buffer, err := prifinet.ReadMessage(relayState.trustees[j].Conn)
		if err != nil {
			prifilog.Println("Relay, couldn't read signature from trustee " + err.Error())
			return err
		}

		sigSize := int(binary.BigEndian.Uint32(buffer[0:4]))
		sig := make([]byte, sigSize)
		copy(sig[:], buffer[4:4+sigSize])
		
		signatures[j] = sig

		prifilog.Println("Collected signature from trustee", j)
	}

	prifilog.Println("Crafting signature message for clients...")

	sigMsg := make([]byte, 0)

	//the final shuffle is the one from the latest trustee
	lastPermutation := relayState.nTrustees - 1
	G_s_i_bytes, err := G_s[lastPermutation].MarshalBinary()
	if err != nil {
		return err
	}

	//pack the final base
	sigMsg = append(sigMsg, prifinet.IntToBA(len(G_s_i_bytes))...)
	sigMsg = append(sigMsg, G_s_i_bytes...)

	//pack the ephemeral shuffle
	pkArray, err := prifinet.MarshalPublicKeyArrayToByteArray(ephPublicKeys_s[lastPermutation])

	if err != nil {
		return err
	}

	sigMsg = append(sigMsg, prifinet.IntToBA(len(pkArray))...)
	sigMsg = append(sigMsg, pkArray...)

	//pack the trustee's signatures
	packedSignatures := make([]byte, 0)
	for j := 0; j < relayState.nTrustees; j++ {
		packedSignatures = append(packedSignatures, prifinet.IntToBA(len(signatures[j]))...)
		packedSignatures = append(packedSignatures, signatures[j]...)
	}
	sigMsg = append(sigMsg, prifinet.IntToBA(len(packedSignatures))...)
	sigMsg = append(sigMsg, packedSignatures...)

	//send to clients
	prifinet.NUnicastMessageToNodes(relayState.clients, sigMsg)

	prifilog.Println("Oblivious shuffle & signatures sent !")
	return nil

	/* 
	//obsolete, of course in practice the client do the verification (relay is untrusted)
	prifilog.Println("We verify on behalf of client")

	M := make([]byte, 0)
	M = append(M, G_s_i_bytes...)
	for k:=0; k<len(ephPublicKeys_s[lastPermutation]); k++{
		prifilog.Println("Embedding eph key")
		prifilog.Println(ephPublicKeys_s[lastPermutation][k])
		pkBytes, _ := ephPublicKeys_s[lastPermutation][k].MarshalBinary()
		M = append(M, pkBytes...)
	}

	prifilog.Println("The message we're gonna verify is :")
	prifilog.Println(hex.Dump(M))

	for j := 0; j < relayState.nTrustees; j++ {
		sigMsg = append(sigMsg, prifinet.IntToBA(len(signatures[j]))...)
		sigMsg = append(sigMsg, signatures[j]...)

		prifilog.Println("Verifying for trustee", j)
		err := crypto.SchnorrVerify(config.CryptoSuite, M, relayState.trustees[j].PublicKey, signatures[j])

		prifilog.Println("Signature was :")
		prifilog.Println(hex.Dump(signatures[j]))

		if err == nil {
			prifilog.Println("Signature OK !")
		} else {
			panic(err.Error())
		}
	}
	*/
}


func processMessageLoop(relayState *RelayState){
	//TODO : if something fail, send true->protocolFailed

	stateMachineLogger.StateChange("protocol-mainloop")

	prifilog.Println("")
	prifilog.Println("#################################")
	prifilog.Println("# Configuration updated, running")
	prifilog.Println("#", relayState.nClients, "clients", relayState.nTrustees, "trustees")

	for i := 0; i<len(relayState.clients); i++ {
		prifilog.Println("# Client", relayState.clients[i].Id, " on port ", relayState.clients[i].Conn.LocalAddr())
	}
	for i := 0; i<len(relayState.trustees); i++ {
		prifilog.Println("# Trustee", relayState.trustees[i].Id, " on port ", relayState.trustees[i].Conn.LocalAddr())
	}
	prifilog.Println("#################################")
	prifilog.Println("")


	prifilog.InfoReport("relay", fmt.Sprintf("new setup, %v clients and %v trustees", relayState.nClients, relayState.nTrustees))

	for i := 0; i<len(relayState.clients); i++ {
		prifilog.InfoReport("relay", fmt.Sprintf("new setup, client %v on %v", relayState.clients[i].Id, relayState.clients[i].Conn.LocalAddr()))
	}
	for i := 0; i<len(relayState.trustees); i++ {
		prifilog.InfoReport("relay", fmt.Sprintf("new setup, trustee %v on %v", relayState.trustees[i].Id, relayState.trustees[i].Conn.LocalAddr()))
	}

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

		//prifilog.Println(".")

		//if needed, we bound the number of round per second
		if INBETWEEN_ROUND_SLEEP_TIME != 0 {
			time.Sleep(INBETWEEN_ROUND_SLEEP_TIME)
			prifilog.StatisticReport("relay", "INBETWEEN_ROUND_SLEEP_TIME", INBETWEEN_ROUND_SLEEP_TIME.String())
		}

		//if the main thread tells us to stop (for re-setup)
		tellClientsToResync := false
		var mainThreadStatus int
		select {
			case mainThreadStatus = <- indicateEndOfProtocol:
				if mainThreadStatus == PROTOCOL_STATUS_GONNA_RESYNC {
					prifilog.Println("Main thread status is 1, gonna warn the clients")
					tellClientsToResync = true
				}
			default:
		}

		//we report the speed, bytes exchanged, etc
		stats.Report()
		if stats.ReportingDone() {
			prifilog.Println("Reporting limit matched; exiting the relay")
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
		downstreamData := make([]byte, 6+downstreamDataPayloadLength)
		binary.BigEndian.PutUint16(downstreamData[0:2], uint16(msgType))
		binary.BigEndian.PutUint32(downstreamData[2:6], uint32(downbuffer.ConnectionId)) //this is the SOCKS connection ID
		copy(downstreamData[6:], downbuffer.Data)

		// Broadcast the downstream data to all clients.
		prifinet.NUnicastMessageToNodes(relayState.clients, downstreamData)
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
			
			prifilog.Println("Relay main loop : Cell will be invalid, some party disconnected. Warning the clients...")

			//craft the message for clients
			downstreamData := make([]byte, 10)
			binary.BigEndian.PutUint16(downstreamData[0:2], uint16(prifinet.MESSAGE_TYPE_LAST_UPLOAD_FAILED))
			binary.BigEndian.PutUint32(downstreamData[2:6], uint32(downbuffer.ConnectionId)) //this is the SOCKS connection ID
			prifinet.NUnicastMessageToNodes(relayState.clients, downstreamData)

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

	if INBETWEEN_CONFIG_SLEEP_TIME != 0 {
		prifilog.Println("Relay main loop : waiting ",INBETWEEN_CONFIG_SLEEP_TIME," seconds, client should now be waiting for new parameters...")
		time.Sleep(INBETWEEN_CONFIG_SLEEP_TIME)
		prifilog.StatisticReport("relay", "INBETWEEN_CONFIG_SLEEP_TIME", INBETWEEN_CONFIG_SLEEP_TIME.String())
	}

	indicateEndOfProtocol <- PROTOCOL_STATUS_RESYNCING

	stateMachineLogger.StateChange("protocol-resync")
}

func newSOCKSProxyHandler(connId int, downstreamData chan<- prifinet.DataWithConnectionId) chan<- []byte {
	upstreamData := make(chan []byte)
	go prifinet.RelaySocksProxy(connId, upstreamData, downstreamData)
	return upstreamData
}

func connectToTrustee(trusteeId int, trusteeHostAddr string, relayState *RelayState) error {
	prifilog.Println("Relay connecting to trustee", trusteeId, "on address", trusteeHostAddr)

	var conn net.Conn = nil
	var err error = nil

	//connect
	for conn == nil{
		conn, err = net.Dial("tcp", trusteeHostAddr)
		if err != nil {
			prifilog.Println("Can't connect to trustee on "+trusteeHostAddr+"; "+err.Error())
			conn = nil
			time.Sleep(FAILED_CONNECTION_WAIT_BEFORE_RETRY)
		}
	}

	//tell the trustee server our parameters
	buffer := make([]byte, 16)
	binary.BigEndian.PutUint32(buffer[0:4], uint32(relayState.PayloadLength))
	binary.BigEndian.PutUint32(buffer[4:8], uint32(relayState.nClients))
	binary.BigEndian.PutUint32(buffer[8:12], uint32(relayState.nTrustees))
	binary.BigEndian.PutUint32(buffer[12:16], uint32(trusteeId))

	prifilog.Println("Writing; setup is", relayState.nClients, relayState.nTrustees, "role is", trusteeId, "cellSize ", relayState.PayloadLength)

	err2 := prifinet.WriteMessage(conn, buffer)

	if err2 != nil {
		prifilog.Println("Error writing to socket:" + err2.Error())
		return err2
	}
	
	// Read the incoming connection into the buffer.
	buffer2, err2 := prifinet.ReadMessage(conn)
	if err2 != nil {
	    prifilog.Println(">>>> Relay : error reading:", err.Error())
	    return err2
	}

	publicKey := config.CryptoSuite.Point()
	err3 := publicKey.UnmarshalBinary(buffer2)

	if err3 != nil {
		prifilog.Println(">>>>  Relay : can't unmarshal trustee key ! " + err3.Error())
		return err3
	}

	prifilog.Println("Trustee", trusteeId, "is connected.")
	
	newTrustee := prifinet.NodeRepresentation{trusteeId, conn, true, publicKey}

	//side effects
	relayState.trustees = append(relayState.trustees, newTrustee)

	return nil
}

func relayServerListener(listeningPort string, newConnection chan net.Conn) {
	listeningSocket, err := net.Listen("tcp", listeningPort)
	if err != nil {
		panic("Can't open listen socket:" + err.Error())
	}

	for {
		conn, err2 := listeningSocket.Accept()
		if err != nil {
			prifilog.Println("Relay : can't accept client. ", err2.Error())
		}
		newConnection <- conn
	}
}

func relayParseClientParamsAux(conn net.Conn) prifinet.NodeRepresentation {

	message, err := prifinet.ReadMessage(conn)
	if err != nil {
		panic("Read error:" + err.Error())
	}

	//check that the node ID is not used
	nextFreeId := 0
	for i:=0; i<len(relayState.clients); i++{

		if relayState.clients[i].Id == nextFreeId {
			nextFreeId++
		}
	}
	prifilog.Println("Client connected, assigning his ID to", nextFreeId)
	nodeId := nextFreeId

	publicKey := config.CryptoSuite.Point()
	err2 := publicKey.UnmarshalBinary(message)

	if err2 != nil {
		panic(">>>>  Relay : can't unmarshal client key ! " + err2.Error())
	}

	newClient := prifinet.NodeRepresentation{nodeId, conn, true, publicKey}

	return newClient
}

func relayParseClientParams(conn net.Conn, newClientChan chan prifinet.NodeRepresentation) {

	newClient := relayParseClientParamsAux(conn)
	newClientChan <- newClient
}