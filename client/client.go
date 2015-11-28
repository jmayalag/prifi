package client

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"io"
	"github.com/lbarman/prifi/crypto"
	"net"
	"github.com/lbarman/prifi/config"
	prifinet "github.com/lbarman/prifi/net"
	prifilog "github.com/lbarman/prifi/log"
	"os"
)

func StartClient(clientId int, relayHostAddr string, expectedNumberOfClients int, nTrustees int, payloadLength int, useSocksProxy bool) {
	fmt.Printf("startClient %d\n", clientId)

	clientState := newClientState(clientId, nTrustees, expectedNumberOfClients, payloadLength, useSocksProxy)
	stats := prifilog.EmptyStatistics(-1) //no limit

	//connect to relay
	relayConn := connectToRelay(relayHostAddr, clientState)

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

	exitClient            := false

	for !exitClient {

		if relayConn == nil {
			fmt.Println("Client: trying to configure, but relay not connected. connecting...")
			relayConn = connectToRelay(relayHostAddr, clientState)
		}

		println(">>>> Configurating... ")

		params := readParamsFromRelay(relayConn)
		clientState.nClients = params.nClients

		//Parse the trustee's public keys, generate the shared secrets
		for i:=0; i<len(params.trusteesPublicKeys); i++ {
			clientState.TrusteePublicKey[i] = params.trusteesPublicKeys[i]
			clientState.sharedSecrets[i] = config.CryptoSuite.Point().Mul(params.trusteesPublicKeys[i], clientState.privateKey)
		}

		//check that we got all keys
		for i := 0; i<clientState.nTrustees; i++ {
			if clientState.TrusteePublicKey[i] == nil {
				panic("Client : didn't get the public key from trustee "+strconv.Itoa(i))
			}
		}

		//TODO: Shuffle to detect if we own the slot
		myRound := roundScheduling(relayConn, clientState)
		clientState.printSecrets()
		println(">>>> All crypto stuff exchanged !")

		//define downstream stream (relay -> client)
		stopReadRelay := make(chan bool, 1)
		go readDataFromRelay(relayConn, dataFromRelay, stopReadRelay)

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
					fmt.Println("[", currentRound, "-",myRound, "]")

					switch data.MessageType {

						case prifinet.MESSAGE_TYPE_LAST_UPLOAD_FAILED :
							//relay wants to re-setup (new key exchanges)
							fmt.Println("Relay warns that a client disconnected, gonna resync..")
							continueToNextRound = false

						case prifinet.MESSAGE_TYPE_DATA_AND_RESYNC :
							//relay wants to re-setup (new key exchanges)
							fmt.Println("Relay wants to resync")
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

			//DEBUG : client 1 hard-fails after 10 loops
			if roundCount > 10 && clientId == 1 {
				fmt.Println("10/1 GONNA EXIT")
				os.Exit(1)
			}

			roundCount++
		}
	}
}

func roundScheduling(relayConn net.Conn, clientState *ClientState) int{

	fmt.Println("Generating ephemeral keys....")
	clientState.generateEphemeralKeys()

	//tell the relay our public key
	ephPublicKeyBytes, _ := clientState.EphemeralPublicKey.MarshalBinary()
	keySize := len(ephPublicKeyBytes)

	buffer := make([]byte, 12+keySize)
	binary.BigEndian.PutUint32(buffer[0:4], uint32(config.LLD_PROTOCOL_VERSION))
	binary.BigEndian.PutUint32(buffer[4:8], uint32(prifinet.SOCKS_CONNECTION_ID_EMPTY))
	binary.BigEndian.PutUint32(buffer[8:12], uint32(keySize))
	copy(buffer[12:], ephPublicKeyBytes)

	n, err := relayConn.Write(buffer)

	if n < 12+keySize || err != nil {
		panic("Error writing to socket:" + err.Error())
	}

	fmt.Println("Ephemeral public key sent to relay, waiting for the trustees signatures")

	G, ephPubKeys, signatures := prifinet.ParseBasePublicKeysAndTrusteeSignaturesFromConn(relayConn)


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
		fmt.Println("Verifying signature for trustee", j)
		err := crypto.SchnorrVerify(config.CryptoSuite, M, clientState.TrusteePublicKey[j], signatures[j])

		if err == nil {
			fmt.Println("Signature OK !")
		} else {
			fmt.Println("Trustee", j, "signature is not valid. Something fishy is going on...")
			panic(err.Error())
		}
	}

	//identify which slot we own
	myPrivKey := clientState.ephemeralPrivateKey
	ephPubInBaseG := config.CryptoSuite.Point().Mul(G, myPrivKey)
	mySlot := -1

	for j:=0; j<len(ephPubKeys); j++ {
		if(ephPubKeys[j].Equal(ephPubInBaseG)) {
			mySlot = j
		}
	}

	if mySlot == -1 {
		panic("We don't have a slot !")
	} else {
		fmt.Println("Our slot is", mySlot)
	}

	return mySlot
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
		panic("Client slice wrong size, expected "+strconv.Itoa(clientState.UsablePayloadLength)+", but got "+strconv.Itoa(len(upstreamSlice)))
	}

	n, err := relayConn.Write(upstreamSlice)

	if n != len(upstreamSlice) {
		fmt.Println("Client write to relay error, expected writing "+strconv.Itoa(len(upstreamSlice))+", but wrote "+strconv.Itoa(n)+", err : " + err.Error())
		return -1 //relay probably disconnected
	}

	return n
}

/*
 * RELAY CONNECTION
 */

func connectToRelay(relayHost string, params *ClientState) net.Conn {
	conn, err := net.Dial("tcp", relayHost)
	if err != nil {
		panic("Can't connect to relay:" + err.Error())
	}


	//tell the relay our public key
	publicKeyBytes, _ := params.PublicKey.MarshalBinary()
	keySize := len(publicKeyBytes)

	buffer := make([]byte, 12+keySize)
	binary.BigEndian.PutUint32(buffer[0:4], uint32(config.LLD_PROTOCOL_VERSION))
	binary.BigEndian.PutUint32(buffer[4:8], uint32(prifinet.SOCKS_CONNECTION_ID_EMPTY))
	binary.BigEndian.PutUint32(buffer[8:12], uint32(keySize))
	copy(buffer[12:], publicKeyBytes)

	n, err := conn.Write(buffer)

	if n < 12+keySize || err != nil {
		panic("Error writing to socket:" + err.Error())
	}

	return conn
}

func readParamsFromRelay(relayConn net.Conn) ParamsFromRelay {

	// Read the next (downstream) header from the relay
	header := [10]byte{}
	n, err := io.ReadFull(relayConn, header[:])

	if n != len(header) {
		panic("readParamsFromRelay: " + err.Error())
	}

	//parse the header
	messageType := int(binary.BigEndian.Uint32(header[0:4]))
	nClients := int(binary.BigEndian.Uint32(header[4:8]))
	dataLength  := int(binary.BigEndian.Uint16(header[8:10]))

	// Read the data
	data := make([]byte, dataLength)
	n, err = io.ReadFull(relayConn, data)

	if err != nil {
		panic("readParamsFromRelay: " + err.Error())
	}

	if messageType != prifinet.MESSAGE_TYPE_PUBLICKEYS {
		fmt.Println("MessageType", messageType, "nClients/SocksConnID", nClients, "DataLength", dataLength)
		panic("Expecting params from relay, got another message")
	}
		
	publicKeys := prifinet.UnMarshalPublicKeyArrayFromByteArray(data, config.CryptoSuite)
	fmt.Println("Got the public key from the trustees...")

	params := ParamsFromRelay{publicKeys, nClients}
	return  params
}

func readDataFromRelay(relayConn net.Conn, dataFromRelay chan<- prifinet.DataWithMessageTypeAndConnId, stopReadRelay chan bool) {
	header := [10]byte{}
	totcells := uint64(0)
	totbytes := uint64(0)

	for {
		// Read the next (downstream) header from the relay
		n, err := io.ReadFull(relayConn, header[:])

		if err != nil {
			fmt.Println("ClientReadRelay error, relay probably disconnected, stopping goroutine...")
			return
		}
		if n != len(header) {
			panic("clientReadRelay: " + err.Error())
		}

		//parse the header
		messageType := int(binary.BigEndian.Uint32(header[0:4]))
		socksConnId := int(binary.BigEndian.Uint32(header[4:8]))
		dataLength  := int(binary.BigEndian.Uint16(header[8:10]))

		// Read the body
		body := make([]byte, dataLength)
		n, err = io.ReadFull(relayConn, body)

		if err != nil {
			fmt.Println("ClientReadRelay error, relay probably disconnected, stopping goroutine...")
			return
		}		
		if n != dataLength {
			panic("readDataFromRelay: read body length ("+strconv.Itoa(n)+") not matching expected length ("+strconv.Itoa(dataLength)+")" + err.Error())
		}

		//communicate to main thread
		dataFromRelay <- prifinet.DataWithMessageTypeAndConnId{messageType, socksConnId, body}

		//statistics
		totcells++
		totbytes += uint64(dataLength)

		if messageType == prifinet.MESSAGE_TYPE_DATA_AND_RESYNC || messageType == prifinet.MESSAGE_TYPE_LAST_UPLOAD_FAILED {
			fmt.Println("next message is gonna be some parameters, not data. Stop this goroutine")
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
							panic("Write to socks proxy: expected "+strconv.Itoa(dataLength)+" bytes, got "+strconv.Itoa(n)+", " + err.Error())
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