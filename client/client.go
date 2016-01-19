package client

import (
	"encoding/binary"
	"strconv"
	"fmt"
	"errors"
	"io"
	"time"
	"strings"
	"github.com/lbarman/crypto/abstract"
	"github.com/lbarman/prifi/crypto"
	"net"
	"github.com/lbarman/prifi/config"
	prifinet "github.com/lbarman/prifi/net"
	prifilog "github.com/lbarman/prifi/log"
)

func StartClient(clientId int, relayHostAddr string, expectedNumberOfClients int, nTrustees int, payloadLength int, useSocksProxy bool, latencyTest bool, useUDP bool) {
	prifilog.SimpleStringDump(prifilog.NOTIFICATION, "Client " + strconv.Itoa(clientId) + "; Started")

	clientState := newClientState(clientId, nTrustees, expectedNumberOfClients, payloadLength, useSocksProxy, latencyTest, useUDP)
	stats := prifilog.EmptyStatistics(-1) //no limit

	//connect to relay
	relayTCPConn, relayUDPConn, err := connectToRelay(relayHostAddr, clientState)

	//define downstream stream (relay -> client)
	dataFromRelay       := make(chan prifinet.DataWithMessageTypeAndConnId)

	//start the socks proxy
	socksProxyNewConnections := make(chan net.Conn)
	dataForRelayBuffer       := make(chan []byte, 0) // This will hold the data to be sent later on to the relay, anonymized
	dataForSocksProxy        := make(chan prifinet.DataWithMessageTypeAndConnId, 0) // This hold the data from the relay to one of the SOCKS connection
	
	if(clientState.UseSocksProxy){
		port := ":" + strconv.Itoa(1080+clientId)
		prifilog.SimpleStringDump(prifilog.INFORMATION, "Client " + strconv.Itoa(clientId) + "; Starting SOCKS proxy on port "+port)
		go startSocksProxyServerListener(port, socksProxyNewConnections)
		go startSocksProxyServerHandler(socksProxyNewConnections, dataForRelayBuffer, dataForSocksProxy, clientState)
	}

	exitClient := false

	for !exitClient {

		if relayTCPConn == nil {
			prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(clientId) + "; trying to configure, but relay not connected. connecting...")
			relayTCPConn, relayUDPConn, err = connectToRelay(relayHostAddr, clientState)

			if err != nil {

				prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(clientId) + "; Could not TCP connect to the relay, restarting")
				relayTCPConn.Close()
				relayTCPConn = nil

				if relayUDPConn != nil {
					relayUDPConn.Close()
					relayUDPConn = nil
				}

				continue //redo everything
			}
		}

		prifilog.SimpleStringDump(prifilog.INFORMATION, "Client " + strconv.Itoa(clientId) + "; Waiting for relay params + public keys...")

		params, err := readPublicKeysFromRelay(relayTCPConn, clientState)

		prifilog.SimpleStringDump(prifilog.INFORMATION, "Client " + strconv.Itoa(clientId) + "; Got the parameters from relay")

		if err != nil {
			prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(clientId) + "; Could not get the parameters from the relay, restarting")
			relayTCPConn.Close()
			relayTCPConn = nil

			if relayUDPConn != nil {
				relayUDPConn.Close()
				relayUDPConn = nil
			}
			continue //redo everything
		}
		clientState.nClients = params.nClients

		//Parse the trustee's public keys, generate the shared secrets
		clientState.nTrustees        = len(params.trusteesPublicKeys)
		clientState.TrusteePublicKey = make([]abstract.Point, clientState.nTrustees)
		clientState.sharedSecrets    = make([]abstract.Point, clientState.nTrustees)

		for i:=0; i<len(params.trusteesPublicKeys); i++ {
			clientState.TrusteePublicKey[i] = params.trusteesPublicKeys[i]
			clientState.sharedSecrets[i] = config.CryptoSuite.Point().Mul(params.trusteesPublicKeys[i], clientState.privateKey)
		}

		//check that we got all keys
		for i := 0; i<clientState.nTrustees; i++ {
			if clientState.TrusteePublicKey[i] == nil {
				prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(clientId) + "; Didn't get public key from trustee "+strconv.Itoa(i)+", restarting")
				relayTCPConn.Close()
				relayTCPConn = nil
				continue //redo everything
			}
		}

		//TODO: Shuffle to detect if we own the slot
		myRound, err := roundScheduling(relayTCPConn, clientState)

		if err != nil {
			relayTCPConn.Close()
			relayTCPConn = nil
			continue //redo everything
		}

		//clientState.printSecrets()
		prifilog.SimpleStringDump(prifilog.NOTIFICATION, "Client " + strconv.Itoa(clientId) + "; Everything ready, assigned to round "+strconv.Itoa(myRound)+" out of "+strconv.Itoa(clientState.nClients))

		//define downstream stream (relay -> client)
		stopReadRelay     := make(chan bool, 1)
		relayDisconnected := make(chan bool, 1)
		go readDataFromRelay(relayTCPConn, relayUDPConn, dataFromRelay, stopReadRelay, relayDisconnected, clientState)

		roundCount          := 0
		continueToNextRound := true
		for continueToNextRound {
			select {

				case d := <- relayDisconnected:
					if d {
						prifilog.SimpleStringDump(prifilog.WARNING, "Client " + strconv.Itoa(clientId) + "; Connection to relay lost, reconfigurating...")
						continueToNextRound = false
					}

				//downstream slice from relay (normal DC-net cycle)
				case data := <-dataFromRelay:
					
					//compute in which round we are (respective to the number of Clients)
					currentRound := roundCount % clientState.nClients
					isMySlot := false
					if currentRound == myRound {
						isMySlot = true
					}

					switch data.MessageType {

						case prifinet.MESSAGE_TYPE_LAST_UPLOAD_FAILED :
							//relay wants to re-setup (new key exchanges)
							prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(clientId) + "; Relay warns that a client disconnected, gonna resync..")
							continueToNextRound = false

						case prifinet.MESSAGE_TYPE_DATA_AND_RESYNC :
							//relay wants to re-setup (new key exchanges)
							prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(clientId) + "; Relay wants to resync...")
							continueToNextRound = false

						case prifinet.MESSAGE_TYPE_DATA :

							//test if it is the answer from a ping
							if len(data.Data) > 2 {
								pattern     := int(binary.BigEndian.Uint16(data.Data[0:2]))
								if pattern == 43690 { //1010101010101010
									clientId  := int(binary.BigEndian.Uint16(data.Data[2:4]))

									if clientId == clientState.Id {
										timestamp := int64(binary.BigEndian.Uint64(data.Data[4:12]))
										diff := prifilog.MsTimeStamp() - timestamp

										//prifilog.Println(prifilog.EXPERIMENT_OUTPUT, "Client "+strconv.Itoa(clientId) +"; Latency is ", fmt.Sprintf("%v", diff) + "ms")

										stats.AddLatency(diff)
									}
								} 
							} else {
								//data for SOCKS proxy, just hand it over to the dedicated thread
								if(clientState.UseSocksProxy){
									dataForSocksProxy <- data
								}
							}
							stats.AddDownstreamCell(int64(len(data.Data)))
					}

					// TODO Should account the downstream cell in the history

					// Produce and ship the next upstream slice
					nBytes := writeNextUpstreamSlice(isMySlot, dataForRelayBuffer, relayTCPConn, clientState)
					if nBytes == -1 {
						//couldn't write anything, relay is disconnected
						relayTCPConn = nil
						continueToNextRound = false
					}
					stats.AddUpstreamCell(int64(nBytes))

					//we report the speed, bytes exchanged, etc
					stats.ReportWithInfo(fmt.Sprintf("cellsize=%v ", payloadLength))
			}
			roundCount++
		}
	}
}

func roundScheduling(relayTCPConn net.Conn, clientState *ClientState) (int, error) {

	prifilog.SimpleStringDump(prifilog.INFORMATION, "Client " + strconv.Itoa(clientState.Id) + "; Generating ephemeral keys...")
	clientState.generateEphemeralKeys()

	//tell the relay our public key
	ephPublicKeyBytes, _ := clientState.EphemeralPublicKey.MarshalBinary()
	keySize := len(ephPublicKeyBytes)

	buffer := make([]byte, 2+keySize)
	binary.BigEndian.PutUint16(buffer[0:2], uint16(prifinet.MESSAGE_TYPE_PUBLICKEYS))
	copy(buffer[2:], ephPublicKeyBytes)

	err := prifinet.WriteMessage(relayTCPConn, buffer)
	if err != nil {
		prifilog.SimpleStringDump(prifilog.SEVERE_ERROR, "Client " + strconv.Itoa(clientState.Id) + "; Error writing ephemeral key "+err.Error())
		return -1, err
	}

	prifilog.SimpleStringDump(prifilog.INFORMATION, "Client " + strconv.Itoa(clientState.Id) + "; waiting for the trustees signatures ")

	G, ephPubKeys, signatures, err := prifinet.ParseBasePublicKeysAndTrusteeSignaturesFromConn(relayTCPConn)

	if err != nil {
		return -1, err
	}

	//composing the signed message
	G_bytes, _ := G.MarshalBinary()
	M := make([]byte, 0)
	M = append(M, G_bytes...)
	for k:=0; k<len(ephPubKeys); k++{
		pkBytes, _ := ephPubKeys[k].MarshalBinary()
		M = append(M, pkBytes...)
	}

	//verifying the signature for all trustees
	for j := 0; j < clientState.nTrustees; j++ {
		err := crypto.SchnorrVerify(config.CryptoSuite, M, clientState.TrusteePublicKey[j], signatures[j])

		if err != nil {
			prifilog.SimpleStringDump(prifilog.SEVERE_ERROR, "Client " + strconv.Itoa(clientState.Id) + "; signature from trustee "+strconv.Itoa(j)+" is invalid")
			return -1, err
		}
	}
	prifilog.SimpleStringDump(prifilog.INFORMATION, "Client " + strconv.Itoa(clientState.Id) + "; all signatures Ok")

	//identify which slot we own
	myPrivKey     := clientState.ephemeralPrivateKey
	ephPubInBaseG := config.CryptoSuite.Point().Mul(G, myPrivKey)
	mySlot        := -1

	for j:=0; j<len(ephPubKeys); j++ {
		if(ephPubKeys[j].Equal(ephPubInBaseG)) {
			mySlot = j
		}
	}

	if mySlot == -1 {
		prifilog.SimpleStringDump(prifilog.SEVERE_ERROR, "Client " + strconv.Itoa(clientState.Id) + "; Can't recognize our slot !")
		return -1, errors.New("We don't have a slot !")
	} 

	return mySlot, nil
}

/*
 * Creates the next cell
 */

func writeNextUpstreamSlice(isMySlot bool, dataForRelayBuffer chan []byte, relayTCPConn net.Conn, clientState *ClientState) int {
	var nextUpstreamBytes []byte

	if isMySlot {
		select
		{
			case nextUpstreamBytes = <-dataForRelayBuffer:

			default:
				if clientState.LatencyTest {

					if clientState.PayloadLength < 12 {
						panic("Trying to do a Latency test, but payload is smaller than 10 bytes.")
					}

					buffer   := make([]byte, clientState.PayloadLength)
					pattern  := uint16(43690) //1010101010101010
					currTime := prifilog.MsTimeStamp() //timestamp in Ms
					
					binary.BigEndian.PutUint16(buffer[0:2], pattern)
					binary.BigEndian.PutUint16(buffer[2:4], uint16(clientState.Id))
					binary.BigEndian.PutUint64(buffer[4:12], uint64(currTime))

					nextUpstreamBytes = buffer
				}
		}
	}		

	//produce the next upstream cell
	upstreamSlice := clientState.CellCoder.ClientEncode(nextUpstreamBytes, clientState.PayloadLength, clientState.MessageHistory)

	if len(upstreamSlice) != clientState.UsablePayloadLength {
		prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(clientState.Id) + "; Client slice wrong size, expected "+strconv.Itoa(clientState.UsablePayloadLength)+", but got "+strconv.Itoa(len(upstreamSlice)))
		return -1 //relay probably disconnected
	}

	err := prifinet.WriteMessage(relayTCPConn, upstreamSlice)

	if err != nil {
		prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(clientState.Id) + "; Client write to relay error, err : " + err.Error())
		return -1 //relay probably disconnected
	}

	return len(upstreamSlice)
}

/*
 * RELAY CONNECTION
 */

func connectToRelay(relayHost string, params *ClientState) (net.Conn, net.Conn, error) {
	
	var tcpConn net.Conn = nil
	var udpConn net.Conn = nil
	var err error = nil

	//connect with TCP
	for tcpConn == nil{
		tcpConn, err = net.Dial("tcp", relayHost)
		if err != nil {
			prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(params.Id) + "; Can't TCP connect to relay on "+relayHost+", gonna retry")
			tcpConn = nil
			time.Sleep(FAILED_CONNECTION_WAIT_BEFORE_RETRY)
		}
	}

	if !params.UseUDP {
		prifilog.SimpleStringDump(prifilog.INFORMATION, "Client " + strconv.Itoa(params.Id) + "; Skipping UDP connection.")
	} else {
		//connect with UDP
		startIndex := strings.LastIndex(relayHost, ":")+1
		localPort := ":"+relayHost[startIndex:] //remove localhost

		for udpConn == nil{
			addr, err := net.ResolveUDPAddr("udp4", localPort)

			if err != nil {
				prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(params.Id) + "; Can't resolve UDP address "+localPort+", gonna retry")
				time.Sleep(FAILED_CONNECTION_WAIT_BEFORE_RETRY)
				continue
			}

			udpConn, err = net.ListenUDP("udp4", addr)
			if err != nil {
				prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(params.Id) + "; Can't listen on UDP on "+addr.String()+", gonna retry")
				udpConn = nil
				time.Sleep(FAILED_CONNECTION_WAIT_BEFORE_RETRY)
			}
		}
		prifilog.Println(prifilog.INFORMATION, "Listening on UDP addr "+localPort)
	}

	//tell the relay our public key
	publicKeyBytes, _ := params.PublicKey.MarshalBinary()
	err2              := prifinet.WriteMessage(tcpConn, publicKeyBytes)

	if err2 != nil {
		prifilog.SimpleStringDump(prifilog.SEVERE_ERROR, "Client " + strconv.Itoa(params.Id) + "; Error writing message, "+err2.Error())
		return nil, nil, err2
	}
	
	return tcpConn, udpConn, nil
}

func readPublicKeysFromRelay(relayTCPConn net.Conn, clientState *ClientState) (ParamsFromRelay, error) {

	// Read the next (downstream) header from the relay
	message, err := prifinet.ReadMessage(relayTCPConn)

	if err != nil {
		return ParamsFromRelay{}, errors.New("readPublicKeysFromRelay: " + err.Error())
	}

	//parse the header
	messageType := int(binary.BigEndian.Uint16(message[0:2]))
	nClients    := int(binary.BigEndian.Uint32(message[2:6]))
	data        := message[6:]

	if messageType != prifinet.MESSAGE_TYPE_PUBLICKEYS {
		prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(clientState.Id) + "; Expected message type "+strconv.Itoa(prifinet.MESSAGE_TYPE_PUBLICKEYS)+", got "+strconv.Itoa(messageType))
		return ParamsFromRelay{}, errors.New("Expecting params from relay, got another message")
	}
		
	publicKeys, err := prifinet.UnMarshalPublicKeyArrayFromByteArray(data, config.CryptoSuite)
	if err != nil {
		prifilog.SimpleStringDump(prifilog.SEVERE_ERROR, "Client " + strconv.Itoa(clientState.Id) + "; Could not unmarshall the keys")
		return ParamsFromRelay{}, err
	}

	params := ParamsFromRelay{publicKeys, nClients}
	return params, nil
}

func readDataFromRelay(relayTCPConn net.Conn, relayUDPConn net.Conn, dataFromRelay chan<- prifinet.DataWithMessageTypeAndConnId, stopReadRelay chan bool, relayDisconnected chan bool, params *ClientState) {

	for {
		// Read the next (downstream) header from the relay
		message, err := prifinet.ReadMessage(relayTCPConn)

		if err != nil {
			prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(params.Id) + "; ClientReadRelay error, relay probably disconnected, stopping goroutine")
			relayDisconnected <- true
			return
		}

		//switch : if it's 4 bytes, it indicates the size of a message of UDP
		if relayUDPConn != nil && len(message) == 4 {

			udpMessageLength := int(binary.BigEndian.Uint32(message[0:4]))
			prifilog.Println(prifilog.RECOVERABLE_ERROR, "Great ! waiting on an UDP message " + strconv.Itoa(udpMessageLength) + ";")
			udpMessage, err2  := prifinet.ReadDatagram(relayUDPConn, udpMessageLength)
			ack              := make([]byte, 1)

			if len(udpMessage) == udpMessageLength && err2 == nil {
				//send ACK
				ack[0] = 1
				prifinet.WriteMessage(relayTCPConn, ack)
				//prifilog.Println(prifilog.INFORMATION, "Client " + strconv.Itoa(params.Id) + "; ClientReadRelay UDP success, received", len(udpMessage), "bytes over UDP")
				message = udpMessage
			} else {
				//send NACK
				ack[0] = 0
				if err2 != nil {
					prifilog.Println(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(params.Id) + "; ClientReadRelay UDP warning, received", len(udpMessage), "instead of advertised", udpMessageLength, "bytes, err is", err2.Error() ,"gonna send NACK and read over TCP")
				} else {
					prifilog.Println(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(params.Id) + "; ClientReadRelay UDP warning, received", len(udpMessage), "instead of advertised", udpMessageLength, "bytes, gonna send NACK read over TCP")
				}
				prifinet.WriteMessage(relayTCPConn, ack)
				messageRetransmittedOverTCP, err := prifinet.ReadMessage(relayTCPConn)

				if err != nil {
					prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(params.Id) + "; ClientReadRelay error (over retransmitted TCP message), relay probably disconnected, stopping goroutine")
					relayDisconnected <- true
					return
				}

				message = messageRetransmittedOverTCP
			}
		}

		//parse the header
		messageType := int(binary.BigEndian.Uint16(message[0:2]))
		socksConnId := int(binary.BigEndian.Uint32(message[2:6]))
		data        := message[6:]

		prifilog.Println(prifilog.SEVERE_ERROR, "MESSAGE TYPE "+strconv.Itoa(messageType)+" SOCKS# "+strconv.Itoa(socksConnId)+" DATALEN "+strconv.Itoa(len(data)))

		//communicate to main thread
		dataFromRelay <- prifinet.DataWithMessageTypeAndConnId{messageType, socksConnId, data}

		if messageType == prifinet.MESSAGE_TYPE_DATA_AND_RESYNC || messageType == prifinet.MESSAGE_TYPE_LAST_UPLOAD_FAILED {
			prifilog.SimpleStringDump(prifilog.RECOVERABLE_ERROR, "Client " + strconv.Itoa(params.Id) + " next message is gonna be some parameters, not data. Stop this goroutine")
			return
		}		
	}
}

/*
 * SOCKS PROXY
 */

func startSocksProxyServerListener(port string, newConnections chan<- net.Conn) {
	prifilog.SimpleStringDump(prifilog.NOTIFICATION, "Listening on port " + port)
	
	lsock, err := net.Listen("tcp", port)

	if err != nil {
		prifilog.Printf(prifilog.RECOVERABLE_ERROR, "Can't open listen socket at port %s: %s", port, err.Error())
		return
	}

	for {
		conn, err := lsock.Accept()
		prifilog.Printf(prifilog.INFORMATION, "Accept on port %s\n", port)

		if err != nil {
			lsock.Close()
			return
		}
		newConnections <- conn
	}
}

func startSocksProxyServerHandler(socksProxyNewConnections chan net.Conn, dataForRelayBuffer chan []byte, dataForSOCKSProxy chan prifinet.DataWithMessageTypeAndConnId, clientState *ClientState) {

	socksProxyActiveConnections := make([]net.Conn, 1) // reserve socksProxyActiveConnections[0]
	socksProxyConnClosed        := make(chan int)
	socksProxyData              := make(chan []byte)

	for {
		select {

			// New TCP connection to the SOCKS proxy
			case conn := <-socksProxyNewConnections: 
				newSocksProxyId := len(socksProxyActiveConnections)
				socksProxyActiveConnections = append(socksProxyActiveConnections, conn)
				go readDataFromSocksProxy(newSocksProxyId, clientState.PayloadLength, conn, socksProxyData, socksProxyConnClosed)

			// Data to anonymize from SOCKS proxy
			case data := <-socksProxyData: 
				dataForRelayBuffer <- data

			// Plaintext downstream data (relay->client->Socks proxy)
			case dataTypeConn := <-dataForSOCKSProxy:
				messageType := dataTypeConn.MessageType //we know it's data for relay
				socksConnId   := dataTypeConn.ConnectionId
				data          := dataTypeConn.Data
				dataLength    := len(data)

				prifilog.Printf(prifilog.INFORMATION, "Read a message with type", messageType, " socks id ", socksConnId)
				
				//Handle the connections, forwards the downstream slice to the SOCKS proxy
				//if there is no socks proxy, nothing to do (useless case indeed, only for debug)
				if clientState.UseSocksProxy {
					if dataLength > 0 && socksProxyActiveConnections[socksConnId] != nil {
						n, err := socksProxyActiveConnections[socksConnId].Write(data)
						if n < dataLength {
							prifilog.Printf(prifilog.RECOVERABLE_ERROR, "Write to socks proxy: expected "+strconv.Itoa(dataLength)+" bytes, got "+strconv.Itoa(n)+", " + err.Error())
						}
					} else {
						// Relay indicating EOF on this conn
						prifilog.Printf(prifilog.RECOVERABLE_ERROR, "Relay to client : closed socks conn %d", socksConnId)
						socksProxyActiveConnections[socksConnId].Close()
					}
				}

			//connection closed from SOCKS proxy
			case socksConnId := <-socksProxyConnClosed:
				socksProxyActiveConnections[socksConnId] = nil
		}
	}
}


func readDataFromSocksProxy(socksConnId int, payloadLength int, conn net.Conn, data chan<- []byte, closed chan<- int) {
	for {
		// Read up to a cell worth of data to send upstream
		buffer := make([]byte, payloadLength)
		n, err := conn.Read(buffer[socksHeaderLength:])

		// Encode the connection number and actual data length
		binary.BigEndian.PutUint32(buffer[0:4], uint32(socksConnId))
		binary.BigEndian.PutUint16(buffer[4:6], uint16(n))

		data <- buffer

		// Connection error or EOF?
		if n == 0 {
			if err == io.EOF {
				prifilog.Printf(prifilog.RECOVERABLE_ERROR, "clientUpload: EOF, closing")
			} else {
				prifilog.Printf(prifilog.RECOVERABLE_ERROR, "clientUpload: " + err.Error())
			}
			conn.Close()
			closed <- socksConnId // signal that channel is closed
			return
		}
	}
}