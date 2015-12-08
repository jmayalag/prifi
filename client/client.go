package client

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"errors"
	"io"
	"time"
	"github.com/lbarman/crypto/abstract"
	"github.com/lbarman/prifi/crypto"
	"net"
	"github.com/lbarman/prifi/config"
	prifinet "github.com/lbarman/prifi/net"
	prifilog "github.com/lbarman/prifi/log"
)

func StartClient(clientId int, relayHostAddr string, expectedNumberOfClients int, nTrustees int, payloadLength int, useSocksProxy bool) {
	prifilog.SimpleStringDump("Client " + strconv.Itoa(clientId) + "; Started")

	clientState := newClientState(clientId, nTrustees, expectedNumberOfClients, payloadLength, useSocksProxy)
	stats := prifilog.EmptyStatistics(-1) //no limit

	//connect to relay
	relayConn, err := connectToRelay(relayHostAddr, clientState)

	//define downstream stream (relay -> client)
	dataFromRelay       := make(chan prifinet.DataWithMessageTypeAndConnId)

	//start the socks proxy
	socksProxyNewConnections := make(chan net.Conn)
	dataForRelayBuffer       := make(chan []byte, 0) // This will hold the data to be sent later on to the relay, anonymized
	dataForSocksProxy        := make(chan prifinet.DataWithMessageTypeAndConnId, 0) // This hold the data from the relay to one of the SOCKS connection
	
	if(clientState.UseSocksProxy){
		port := ":" + strconv.Itoa(1080+clientId)
		go startSocksProxyServerListener(port, socksProxyNewConnections)
		go startSocksProxyServerHandler(socksProxyNewConnections, dataForRelayBuffer, dataForSocksProxy, clientState)
	}

	exitClient := false

	for !exitClient {

		if relayConn == nil {
			prifilog.SimpleStringDump("Client " + strconv.Itoa(clientId) + "; trying to configure, but relay not connected. connecting...")
			relayConn, err = connectToRelay(relayHostAddr, clientState)

			if err != nil {
				relayConn.Close()
				relayConn = nil
				continue //redo everything
			}
		}

		prifilog.SimpleStringDump("Client " + strconv.Itoa(clientId) + "; Waiting for relay params + public keys...")

		params, err := readPublicKeysFromRelay(relayConn, clientState)
		if err != nil {
			relayConn.Close()
			relayConn = nil
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
				prifilog.SimpleStringDump("Client " + strconv.Itoa(clientId) + "; Didn't get public key from trustee "+strconv.Itoa(i))
				relayConn.Close()
				relayConn = nil
				continue //redo everything
			}
		}

		//TODO: Shuffle to detect if we own the slot
		myRound, err := roundScheduling(relayConn, clientState)

		if err != nil {
			relayConn.Close()
			relayConn = nil
			continue //redo everything
		}

		clientState.printSecrets()
		prifilog.SimpleStringDump("Client " + strconv.Itoa(clientId) + "; Everything ready, assigned to round "+strconv.Itoa(myRound)+" out of "+strconv.Itoa(clientState.nClients))

		//define downstream stream (relay -> client)
		stopReadRelay := make(chan bool, 1)
		go readDataFromRelay(relayConn, dataFromRelay, stopReadRelay, clientState)

		roundCount          := 0
		continueToNextRound := true
		for continueToNextRound {
			select {
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
							prifilog.SimpleStringDump("Client " + strconv.Itoa(clientId) + "; Relay warns that a client disconnected, gonna resync..")
							continueToNextRound = false

						case prifinet.MESSAGE_TYPE_DATA_AND_RESYNC :
							//relay wants to re-setup (new key exchanges)
							prifilog.SimpleStringDump("Client " + strconv.Itoa(clientId) + "; Relay wants to resync...")
							continueToNextRound = false

						case prifinet.MESSAGE_TYPE_DATA :
							//data for SOCKS proxy, just hand it over to the dedicated thread
							if(clientState.UseSocksProxy){
								dataForSocksProxy <- data
							}
							stats.AddDownstreamCell(int64(len(data.Data)))
					}

					// TODO Should account the downstream cell in the history

					// Produce and ship the next upstream slice
					nBytes := writeNextUpstreamSlice(isMySlot, dataForRelayBuffer, relayConn, clientState)
					if nBytes == -1 {
						//couldn't write anything, relay is disconnected
						relayConn = nil
						continueToNextRound = false
					}
					stats.AddUpstreamCell(int64(nBytes))

					//we report the speed, bytes exchanged, etc
					stats.Report()
			}
			roundCount++
		}
	}
}

func roundScheduling(relayConn net.Conn, clientState *ClientState) (int, error) {

	prifilog.SimpleStringDump("Client " + strconv.Itoa(clientState.Id) + "; Generating ephemeral keys...")
	clientState.generateEphemeralKeys()

	//tell the relay our public key
	ephPublicKeyBytes, _ := clientState.EphemeralPublicKey.MarshalBinary()
	keySize := len(ephPublicKeyBytes)

	buffer := make([]byte, 2+keySize)
	binary.BigEndian.PutUint16(buffer[0:2], uint16(prifinet.MESSAGE_TYPE_PUBLICKEYS))
	copy(buffer[2:], ephPublicKeyBytes)

	err := prifinet.WriteMessage(relayConn, buffer)
	if err != nil {
		prifilog.SimpleStringDump("Client " + strconv.Itoa(clientState.Id) + "; Error writing ephemeral key "+err.Error())
		return -1, err
	}

	prifilog.SimpleStringDump("Client " + strconv.Itoa(clientState.Id) + "; waiting for the trustees signatures ")

	G, ephPubKeys, signatures, err := prifinet.ParseBasePublicKeysAndTrusteeSignaturesFromConn(relayConn)

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
			prifilog.SimpleStringDump("Client " + strconv.Itoa(clientState.Id) + "; signature from trustee "+strconv.Itoa(j)+" is invalid")
			return -1, err
		}
	}
	prifilog.SimpleStringDump("Client " + strconv.Itoa(clientState.Id) + "; all signatures Ok")

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
		prifilog.SimpleStringDump("Client " + strconv.Itoa(clientState.Id) + "; Can't recognize our slot !")
		return -1, errors.New("We don't have a slot !")
	} 

	return mySlot, nil
}

/*
 * Creates the next cell
 */

func writeNextUpstreamSlice(canWrite bool, dataForRelayBuffer chan []byte, relayConn net.Conn, clientState *ClientState) int {
	var nextUpstreamBytes []byte

	if canWrite {
		select
		{
			case nextUpstreamBytes = <-dataForRelayBuffer:

			default:
		}
	}		

	//produce the next upstream cell
	upstreamSlice := clientState.CellCoder.ClientEncode(nextUpstreamBytes, clientState.PayloadLength, clientState.MessageHistory)

	if len(upstreamSlice) != clientState.UsablePayloadLength {
		prifilog.SimpleStringDump("Client " + strconv.Itoa(clientState.Id) + "; Client slice wrong size, expected "+strconv.Itoa(clientState.UsablePayloadLength)+", but got "+strconv.Itoa(len(upstreamSlice)))
		return -1 //relay probably disconnected
	}

	err := prifinet.WriteMessage(relayConn, upstreamSlice)

	if err != nil {
		prifilog.SimpleStringDump("Client " + strconv.Itoa(clientState.Id) + "; Client write to relay error, err : " + err.Error())
		fmt.Println("Client write to relay error, err : " + err.Error())
		return -1 //relay probably disconnected
	}

	return len(upstreamSlice)
}

/*
 * RELAY CONNECTION
 */

func connectToRelay(relayHost string, params *ClientState) (net.Conn, error) {
	
	var conn net.Conn = nil
	var err error = nil

	//connect
	for conn == nil{
		conn, err = net.Dial("tcp", relayHost)
		if err != nil {
			prifilog.SimpleStringDump("Client " + strconv.Itoa(params.Id) + "; Can't connect to relay on "+relayHost+", gonna retry")
			conn = nil
			time.Sleep(FAILED_CONNECTION_WAIT_BEFORE_RETRY)
		}
	}

	//tell the relay our public key
	publicKeyBytes, _ := params.PublicKey.MarshalBinary()

	err2 := prifinet.WriteMessage(conn, publicKeyBytes)

	if err2 != nil {
		prifilog.SimpleStringDump("Client " + strconv.Itoa(params.Id) + "; Error writing message, "+err2.Error())
		return nil, err2
	}

	return conn, nil
}

func readPublicKeysFromRelay(relayConn net.Conn, clientState *ClientState) (ParamsFromRelay, error) {

	// Read the next (downstream) header from the relay
	message, err := prifinet.ReadMessage(relayConn)

	if err != nil {
		return ParamsFromRelay{}, errors.New("readPublicKeysFromRelay: " + err.Error())
	}

	//parse the header
	messageType := int(binary.BigEndian.Uint16(message[0:2]))
	nClients    := int(binary.BigEndian.Uint32(message[2:6]))
	data        := message[6:]

	if messageType != prifinet.MESSAGE_TYPE_PUBLICKEYS {
		prifilog.SimpleStringDump("Client " + strconv.Itoa(clientState.Id) + "; Expected message type "+strconv.Itoa(prifinet.MESSAGE_TYPE_PUBLICKEYS)+", got "+strconv.Itoa(messageType))
		return ParamsFromRelay{}, errors.New("Expecting params from relay, got another message")
	}
		
	publicKeys, err := prifinet.UnMarshalPublicKeyArrayFromByteArray(data, config.CryptoSuite)
	if err != nil {
		return ParamsFromRelay{}, err
	}

	params := ParamsFromRelay{publicKeys, nClients}
	return params, nil
}

func readDataFromRelay(relayConn net.Conn, dataFromRelay chan<- prifinet.DataWithMessageTypeAndConnId, stopReadRelay chan bool, params *ClientState) {
	totcells := uint64(0)
	totbytes := uint64(0)

	for {
		// Read the next (downstream) header from the relay
		message, err := prifinet.ReadMessage(relayConn)

		if err != nil {
			prifilog.SimpleStringDump("Client " + strconv.Itoa(params.Id) + "; ClientReadRelay error, relay probably disconnected, stopping goroutine")
			return
		}

		//parse the header
		messageType := int(binary.BigEndian.Uint16(message[0:2]))
		socksConnId := int(binary.BigEndian.Uint32(message[2:6]))
		data        := message[6:]

		if err != nil {
			prifilog.SimpleStringDump("Client " + strconv.Itoa(params.Id) + "; ClientReadRelay error, relay probably disconnected, stopping goroutine")
			return
		}

		//communicate to main thread
		dataFromRelay <- prifinet.DataWithMessageTypeAndConnId{messageType, socksConnId, data}

		//statistics
		totcells++
		totbytes += uint64(len(data))

		if messageType == prifinet.MESSAGE_TYPE_DATA_AND_RESYNC || messageType == prifinet.MESSAGE_TYPE_LAST_UPLOAD_FAILED {
			prifilog.SimpleStringDump("Client " + strconv.Itoa(params.Id) + "next message is gonna be some parameters, not data. Stop this goroutine")
			return
		} 
	}
}

/*
 * SOCKS PROXY
 */

func startSocksProxyServerListener(port string, newConnections chan<- net.Conn) {
	fmt.Printf("Listening on port %s\n", port)
	
	lsock, err := net.Listen("tcp", port)

	if err != nil {
		fmt.Printf("Can't open listen socket at port %s: %s", port, err.Error())
		return
	}

	for {
		conn, err := lsock.Accept()
		fmt.Printf("Accept on port %s\n", port)

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

				fmt.Println("Read a message with type", messageType, " socks id ", socksConnId)
				
				//Handle the connections, forwards the downstream slice to the SOCKS proxy
				//if there is no socks proxy, nothing to do (useless case indeed, only for debug)
				if clientState.UseSocksProxy {
					if dataLength > 0 && socksProxyActiveConnections[socksConnId] != nil {
						n, err := socksProxyActiveConnections[socksConnId].Write(data)
						if n < dataLength {
							fmt.Println("Write to socks proxy: expected "+strconv.Itoa(dataLength)+" bytes, got "+strconv.Itoa(n)+", " + err.Error())
						}
					} else {
						// Relay indicating EOF on this conn
						fmt.Printf("Relay to client : closed socks conn %d", socksConnId)
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
				println("clientUpload: EOF, closing")
			} else {
				println("clientUpload: " + err.Error())
			}
			conn.Close()
			closed <- socksConnId // signal that channel is closed
			return
		}
	}
}