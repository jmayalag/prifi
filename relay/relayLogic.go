package relay

import (
	"encoding/binary"
	"fmt"
	"time"
	"log"
	"net"
	"strconv"
	prifinet "github.com/lbarman/prifi/net"
	prifilog "github.com/lbarman/prifi/log"
)

var relayState 			*RelayState 
var stateMachineLogger 	*prifilog.StateMachineLogger

var	protocolFailed          = make(chan bool)
var	messagesTowardsProcessingLoop = make(chan int)
var	messagesFromProcessingLoop    = make(chan int)
var	deconnectedClients      = make(chan int)
var	timedOutClients         = make(chan int)
var	deconnectedTrustees     = make(chan int)

func StartRelay(upstreamCellSize int, downstreamCellSize int, dummyDataDown bool, relayPort string, nClients int, nTrustees int, trusteesIp []string, reportingLimit int, useUDP bool) {

	prifilog.SimpleStringDump(prifilog.NOTIFICATION, "Relay started")

	stateMachineLogger = prifilog.NewStateMachineLogger("relay")
	stateMachineLogger.StateChange("relay-init")

	relayState = initiateRelayState(relayPort, nTrustees, nClients, upstreamCellSize, downstreamCellSize, dummyDataDown, reportingLimit, trusteesIp, useUDP)

	//start the server waiting for clients
	newClientConnectionsChan        := make(chan net.Conn) 	          			//channel with unparsed clients
	go relayServerListener(relayPort, newClientConnectionsChan)

	//start the client parser
	newClientWithIdAndPublicKeyChan := make(chan prifinet.NodeRepresentation)  //channel with parsed clients
	go welcomeNewClients(newClientConnectionsChan, newClientWithIdAndPublicKeyChan, useUDP)

	//prepare the UDP broadcast
	if relayState.UseUDP {
		addr := prifinet.IPV4_BROADCAST_ADDR+relayPort
		prifilog.Println(prifilog.INFORMATION, "Preparing UDP broadcast socket on " + addr)
		udpConn, err3 := net.Dial("udp4", addr)

		if err3 != nil {
			prifilog.Println(prifilog.RECOVERABLE_ERROR, "Failed preparing UDP broadcast socket on " + addr + ", switching off UDP")
			relayState.UseUDP = false
		} else {
			relayState.UDPBroadcastConn = udpConn
		}
	}

	stateMachineLogger.StateChange("protocol-setup")

	//start the actual protocol
	relayState.connectToAllTrustees()
	relayState.waitForDefaultNumberOfClients(newClientWithIdAndPublicKeyChan)
	relayState.advertisePublicKeys()	
	err := relayState.organizeRoundScheduling()

	var isProtocolRunning = false
	if err != nil {
		prifilog.Println(prifilog.RECOVERABLE_ERROR, "Relay Handler : round scheduling went wrong, restarting the configuration protocol")

		//disconnect all clients
		for i:=0; i<len(relayState.clients); i++{
			relayState.clients[i].Conn.Close()
			relayState.clients[i].Connected = false
		}	
		restartProtocol(relayState, make([]prifinet.NodeRepresentation, 0));
	} else {
		//copy for subthread, run subthread for processing the messages
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
				prifilog.Println(prifilog.NOTIFICATION, "Relay Handler : Processing loop has failed with status", protocolHasFailed)
				isProtocolRunning = false
				//TODO : re-run setup, something went wrong. Maybe restart from 0 ?

			case deconnectedClient := <- deconnectedClients:
				prifilog.Println(prifilog.WARNING, "Client", deconnectedClient, " has been indicated offline")
				relayState.clients[deconnectedClient].Connected = false

			case timedOutClient := <- timedOutClients:
				prifilog.Println(prifilog.WARNING, "Client", timedOutClient, " has been indicated offline (time out)")
				relayState.clients[timedOutClient].Conn.Close()
				relayState.clients[timedOutClient].Connected = false

			case deconnectedTrustee := <- deconnectedTrustees:
				prifilog.Println(prifilog.RECOVERABLE_ERROR, "Trustee", deconnectedTrustee, " has been indicated offline")

			case newClient := <- newClientWithIdAndPublicKeyChan:
				//we tell processMessageLoop to stop when possible
				newClients = append(newClients, newClient)
				if isProtocolRunning {
					prifilog.Println(prifilog.NOTIFICATION, "Relay Handler : new Client is ready, stopping processing loop")
					messagesTowardsProcessingLoop <- PROTOCOL_STATUS_GONNA_RESYNC
				} else {
					prifilog.Println(prifilog.NOTIFICATION, "Relay Handler : new Client is ready, restarting processing loop")
					isProtocolRunning = restartProtocol(relayState, newClients)
					newClients = make([]prifinet.NodeRepresentation, 0)
					prifilog.Println(prifilog.INFORMATION, "Done...")
				}

			case endOfProtocolState = <- messagesFromProcessingLoop:
				prifilog.Println(prifilog.INFORMATION, "Relay Handler : main loop stopped (status",endOfProtocolState,"), resyncing")

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
		prifilog.Println(prifilog.NOTIFICATION, "Adding new client", newClients[i])
	}
	relayState.nClients = len(relayState.clients)

	//if we dont have enough client, stop.
	if len(relayState.clients) == 0{
		prifilog.Println(prifilog.WARNING, "Relay Handler : not enough client, stopping and waiting...")
		return false
	} else {
		//re-advertise the configuration 	
		relayState.connectToAllTrustees()
		relayState.advertisePublicKeys()
		err := relayState.organizeRoundScheduling()
		if err != nil {
			prifilog.Println(prifilog.RECOVERABLE_ERROR, "Relay Handler : round scheduling went wrong, restarting the configuration protocol")

			//disconnect all clients
			for i:=0; i<len(relayState.clients); i++{
				relayState.clients[i].Conn.Close()
				relayState.clients[i].Connected = false
			}	
			return restartProtocol(relayState, make([]prifinet.NodeRepresentation, 0));
		}

		if INBETWEEN_CONFIG_SLEEP_TIME != 0 {
			time.Sleep(INBETWEEN_CONFIG_SLEEP_TIME)
			prifilog.StatisticReport("relay", "INBETWEEN_CONFIG_SLEEP_TIME", INBETWEEN_CONFIG_SLEEP_TIME.String())
		}

		//process message loop
		relayStateCopy := relayState.deepClone()
		go processMessageLoop(relayStateCopy)

		return true
	}
}

func processMessageLoop(relayState *RelayState){
	//TODO : if something fail, send true->protocolFailed

	stateMachineLogger.StateChange("protocol-mainloop")

	prifilog.InfoReport(prifilog.NOTIFICATION, "relay", fmt.Sprintf("new setup, %v clients and %v trustees", relayState.nClients, relayState.nTrustees))

	for i := 0; i<len(relayState.clients); i++ {
		prifilog.InfoReport(prifilog.NOTIFICATION, "relay", fmt.Sprintf("new setup, client %v on %v", relayState.clients[i].Id, relayState.clients[i].Conn.LocalAddr()))
	}
	for i := 0; i<len(relayState.trustees); i++ {
		prifilog.InfoReport(prifilog.NOTIFICATION, "relay", fmt.Sprintf("new setup, trustee %v on %v", relayState.trustees[i].Id, relayState.trustees[i].Conn.LocalAddr()))
	}

	stats := prifilog.EmptyStatistics(relayState.ReportingLimit)

	// Create ciphertext slice bufferfers for all clients and trustees
	clientupstreamCellSize := relayState.CellCoder.ClientCellSize(relayState.UpstreamCellSize)
	clientsPayloadData  := make([][]byte, relayState.nClients)
	for i := 0; i < relayState.nClients; i++ {
		clientsPayloadData[i] = make([]byte, clientupstreamCellSize)
	}

	trusteeupstreamCellSize := relayState.CellCoder.TrusteeCellSize(relayState.UpstreamCellSize)
	trusteesPayloadData  := make([][]byte, relayState.nTrustees)
	for i := 0; i < relayState.nTrustees; i++ {
		trusteesPayloadData[i] = make([]byte, trusteeupstreamCellSize)
	}

	socksProxyConnections := make(map[int]chan<- []byte)
	downstream            := make(chan prifinet.DataWithConnectionId)
	priorityDownStream    := make([]prifinet.DataWithConnectionId, 0)
	nulldown              := prifinet.DataWithConnectionId{} // default empty downstream cell
	
	//window                := 2           // Maximum cells in-flight
	//inflight              := 0         // Current cells in-flight

	currentSetupContinues := true
	
	for currentSetupContinues {

		//prifilog.Println(".")

		//if needed, we bound the number of round per second
		if PROCESSING_LOOP_SLEEP_TIME != 0 {
			time.Sleep(PROCESSING_LOOP_SLEEP_TIME)
			prifilog.StatisticReport("relay", "PROCESSING_LOOP_SLEEP_TIME", PROCESSING_LOOP_SLEEP_TIME.String())
		}

		//if the main thread tells us to stop (for re-setup)
		tellClientsToResync := false
		var mainThreadStatus int
		select {
			case mainThreadStatus = <- messagesTowardsProcessingLoop:
				if mainThreadStatus == PROTOCOL_STATUS_GONNA_RESYNC {
					prifilog.Println(prifilog.NOTIFICATION, "Main thread status is PROTOCOL_STATUS_GONNA_RESYNC, gonna warn the clients")
					tellClientsToResync = true
				}
			default:
		}

		//we report the speed, bytes exchanged, etc
		stats.ReportWithInfo("upCellSize "+strconv.Itoa(relayState.UpstreamCellSize)+" downCellSize "+
			strconv.Itoa(relayState.DownstreamCellSize)+" nClients"+strconv.Itoa(relayState.nClients)+" nTrustees"+strconv.Itoa(relayState.nTrustees))
		if stats.ReportingDone() {
			prifilog.Println(prifilog.WARNING, "Reporting limit matched; exiting the relay")
			break;
		}

		// See if there's any downstream data to forward.
		var downbuffer prifinet.DataWithConnectionId 
		if len(priorityDownStream) > 0 {
			downbuffer         = priorityDownStream[0]

			if len(priorityDownStream) == 1 {
				priorityDownStream = nil
			} else {
				priorityDownStream = priorityDownStream[1:]
			}
		} else {
			select {
				case downbuffer = <-downstream: // some data to forward downstream
				default: 
					downbuffer = nulldown
			}
		}

		//compute the message type; if MESSAGE_TYPE_DATA_AND_RESYNC, the clients know they will resync
		msgType := prifinet.MESSAGE_TYPE_DATA
		if tellClientsToResync{
			msgType = prifinet.MESSAGE_TYPE_DATA_AND_RESYNC
			currentSetupContinues = false
		}

		//craft the message for clients
		downstreamDataCellSize := len(downbuffer.Data)		
		if relayState.UseDummyDataDown && relayState.DownstreamCellSize > len(downbuffer.Data){
			//if we want dummy traffic down, force the size to be as big as the specified down cell size. The rest will be 0
			downstreamDataCellSize = relayState.DownstreamCellSize
		}
		downstreamData := make([]byte, 6+downstreamDataCellSize)
		binary.BigEndian.PutUint16(downstreamData[0:2], uint16(msgType))
		binary.BigEndian.PutUint32(downstreamData[2:6], uint32(downbuffer.ConnectionId)) //this is the SOCKS connection ID
		copy(downstreamData[6:], downbuffer.Data)
		prifilog.Println(prifilog.RECOVERABLE_ERROR, "MESSAGE TYPE "+strconv.Itoa(msgType)+" SOCKS# "+strconv.Itoa(downbuffer.ConnectionId)+" DATALEN "+strconv.Itoa(len(downbuffer.Data)))
					

		// Broadcast the downstream data to all clients.

		if !relayState.UseUDP {
			//simple version, N unicast through TCP

			//prifilog.Println(prifilog.NOTIFICATION, "Sending", len(downstreamData), "bytes over NUnicast TCP")
			prifinet.NUnicastMessageToNodes(relayState.clients, downstreamData)
			stats.AddDownstreamCell(int64(downstreamDataCellSize))
		} else {

			//prifilog.Println(prifilog.NOTIFICATION, "Sending", len(downstreamData), "bytes over UDP Broadcast")

			//1. tell via TCP the length of the message
			sizeMessage := make([]byte, 4)
			binary.BigEndian.PutUint32(sizeMessage[0:4], uint32(len(downstreamData)))
			prifinet.NUnicastMessageToNodes(relayState.clients, sizeMessage)
			stats.AddDownstreamCell(int64(4))
			
			//2. broadcast message through UDP
			prifinet.WriteMessage(relayState.UDPBroadcastConn, downstreamData)
			stats.AddDownstreamUDPCell(int64(downstreamDataCellSize))

			//TODO : this could happen in parallel
			//acks := make([]bool, relayState.nClients)
			for i := 0; i < relayState.nClients; i++ {
				buffer, err := prifinet.ReadMessage(relayState.clients[i].Conn)
				
				if err != nil || len(buffer) != 1 || buffer[0] == 0{
					prifilog.Println(prifilog.RECOVERABLE_ERROR, "Client", i, "did not fully get the UDP broadcast, re-transmitting "+strconv.Itoa(len(downstreamData))+" bytes over TCP...")
					prifinet.WriteMessage(relayState.clients[i].Conn, downstreamData)
					stats.AddDownstreamRetransmitCell(int64(6+downstreamDataCellSize))
				} 
			}
		}

		/* LBARMAN : disabled, until effect on performance is clear
		inflight++
		if inflight < window {
			continue // Get more cells in flight
		}*/

		relayState.CellCoder.DecodeStart(relayState.UpstreamCellSize, relayState.MessageHistory)

		// Collect a cell ciphertext from each trustee
		errorInThisCell := false
		for i := 0; i < relayState.nTrustees; i++ {	

			//TODO : add a channel for timeout trustee
			data, err := prifinet.ReadWithTimeOut(i, relayState.trustees[i].Conn, trusteeupstreamCellSize, CLIENT_READ_TIMEOUT, deconnectedTrustees, deconnectedTrustees)

			if err {
				errorInThisCell = true
				break
			} else {
				relayState.CellCoder.DecodeTrustee(data)
			}			
		}

		// Collect an upstream ciphertext from each client
		if !errorInThisCell {
			for i := 0; i < relayState.nClients; i++ {
				data, err := prifinet.ReadWithTimeOut(i, relayState.clients[i].Conn, clientupstreamCellSize, CLIENT_READ_TIMEOUT, timedOutClients, deconnectedClients)

				if err {
					errorInThisCell = true
					break
				} else {
					relayState.CellCoder.DecodeClient(data)
				}
			}
		}

		if errorInThisCell {
			
			prifilog.Println(prifilog.WARNING, "Relay main loop : Cell will be invalid, some party disconnected. Warning the clients...")

			//craft the message for clients
			downstreamData := make([]byte, 10)
			binary.BigEndian.PutUint16(downstreamData[0:2], uint16(prifinet.MESSAGE_TYPE_LAST_UPLOAD_FAILED))
			binary.BigEndian.PutUint32(downstreamData[2:6], uint32(downbuffer.ConnectionId)) //this is the SOCKS connection ID
			prifinet.NUnicastMessageToNodes(relayState.clients, downstreamData)

			break
		} else {
			upstreamPlaintext := relayState.CellCoder.DecodeCell()
			//inflight--

			stats.AddUpstreamCell(int64(relayState.UpstreamCellSize))

			// Process the decoded cell

			//check if we have a latency test message
			pattern     := int(binary.BigEndian.Uint16(upstreamPlaintext[0:2]))
			if pattern == 43690 { //1010101010101010
				//clientId  := uint16(binary.BigEndian.Uint16(upstreamPlaintext[2:4]))
				//timestamp := uint64(binary.BigEndian.Uint64(upstreamPlaintext[4:12]))

				cellDown := prifinet.DataWithConnectionId{-1, upstreamPlaintext}
				priorityDownStream = append(priorityDownStream, cellDown)

				continue //the rest is for SOCKS
			}

			if upstreamPlaintext == nil {
				continue // empty or corrupt upstream cell
			}
			if len(upstreamPlaintext) != relayState.UpstreamCellSize {
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

			if 6+socksDataLength > relayState.UpstreamCellSize {
				log.Printf("upstream cell invalid length %d", 6+socksDataLength)
				continue
			}

			socksConn <- upstreamPlaintext[6 : 6+socksDataLength]
		}
	}

	if INBETWEEN_CONFIG_SLEEP_TIME != 0 {
		time.Sleep(INBETWEEN_CONFIG_SLEEP_TIME)
		prifilog.StatisticReport("relay", "INBETWEEN_CONFIG_SLEEP_TIME", INBETWEEN_CONFIG_SLEEP_TIME.String())
	}

	messagesFromProcessingLoop <- PROTOCOL_STATUS_RESYNCING

	stateMachineLogger.StateChange("protocol-resync")
}

func relayParseClientParams(tcpConn net.Conn, newClientChan chan prifinet.NodeRepresentation, clientsUseUDP bool) {

	newClient, success := relayParseClientParamsAux(tcpConn, clientsUseUDP)
	if success {
		prifilog.Println(prifilog.INFORMATION, "Client parameter parsed, sending back...")
		newClientChan <- newClient
	} else {
		prifilog.Println(prifilog.WARNING, "Could not parse client parameters, ignoring him...")
	}
}