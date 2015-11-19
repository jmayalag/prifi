package main

import (
	"encoding/binary"
	"fmt"
	"github.com/lbarman/prifi/util"
	"encoding/hex"
	"strconv"
	"io"
	"net"
	"github.com/lbarman/prifi/dcnet"
	"github.com/lbarman/crypto/abstract"
	//log2 "github.com/lbarman/prifi/log"
	"time"
	"os"
)

// Number of bytes of cell payload to reserve for connection header, length
const socksHeaderLength = 6

type ClientState struct {
	Name				string

	PublicKey			abstract.Point
	privateKey			abstract.Secret

	nClients			int
	nTrustees			int

	PayloadLength		int
	UsablePayloadLength	int
	UseSocksProxy		bool
	
	TrusteePublicKey	[]abstract.Point
	sharedSecrets		[]abstract.Point
	
	CellCoder			dcnet.CellCoder
	
	MessageHistory		abstract.Cipher
}

func initiateClientState(socksConnId int, nTrustees int, nClients int, payloadLength int, useSocksProxy bool) *ClientState {

	params := new(ClientState)

	params.Name                = "Client-"+strconv.Itoa(socksConnId)
	params.nClients            = nClients
	params.nTrustees           = nTrustees
	params.PayloadLength       = payloadLength
	params.UseSocksProxy       = useSocksProxy

	//prepare the crypto parameters
	rand 	:= suite.Cipher([]byte(params.Name))
	base	:= suite.Point().Base()

	//generate own parameters
	params.privateKey       = suite.Secret().Pick(rand)
	params.PublicKey        = suite.Point().Mul(base, params.privateKey)

	//placeholders for pubkeys and secrets
	params.TrusteePublicKey = make([]abstract.Point,  nTrustees)
	params.sharedSecrets    = make([]abstract.Point, nTrustees)

	//sets the cell coder, and the history
	params.CellCoder           = factory()
	params.UsablePayloadLength = params.CellCoder.ClientCellSize(payloadLength)

	return params
}

func (clientState *ClientState) printSecrets() {
	//print all shared secrets
	for i:=0; i<clientState.nTrustees; i++ {
		fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
		fmt.Println("            TRUSTEE", i)
		d1, _ := clientState.TrusteePublicKey[i].MarshalBinary()
		d2, _ := clientState.sharedSecrets[i].MarshalBinary()
		fmt.Println(hex.Dump(d1))
		fmt.Println("+++")
		fmt.Println(hex.Dump(d2))
		fmt.Println("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
	}
}

func startClient(socksConnId int, relayHostAddr string, nClients int, nTrustees int, payloadLength int, useSocksProxy bool) {
	fmt.Printf("startClient %d\n", socksConnId)

	clientState := initiateClientState(socksConnId, nTrustees, nClients, payloadLength, useSocksProxy)
	stats := emptyStatistics(-1) //no limit

	//connect to relay
	relayConn := connectToRelay(relayHostAddr, socksConnId, clientState)

	//initiate downstream stream (relay -> client)
	dataFromRelay       := make(chan dataWithMessageTypeAndConnId)
	publicKeysFromRelay := make(chan []abstract.Point)
	go readDataFromRelay(relayConn, dataFromRelay, publicKeysFromRelay)

	//start the socks proxy
	socksProxyNewConnections := make(chan net.Conn)
	dataForRelayBuffer       := make(chan []byte, 0) // This will hold the data to be sent later on to the relay, anonymized
	dataForSocksProxy        := make(chan dataWithMessageTypeAndConnId, 0) // This hold the data from the relay to one of the SOCKS connection
	
	if(clientState.UseSocksProxy){
		port := ":" + strconv.Itoa(1080+socksConnId)
		go startSocksProxyServerListener(port, socksProxyNewConnections)
		go startSocksProxyServerHandler(socksProxyNewConnections, dataForRelayBuffer, dataForSocksProxy, clientState)
	} else {
		go channelCleaner(dataForSocksProxy)
	}

	exitClient := false
	for !exitClient {
		println(">>>> Configurating... ")

		var trusteesPublicKeys []abstract.Point
		publicKeysMessageReceived := false

		for !publicKeysMessageReceived{
			select {
				case trusteesPublicKeys = <- publicKeysFromRelay:
					println("Received the public keys !")
					publicKeysMessageReceived = true

				default: 
					time.Sleep(1000*time.Millisecond)
			}
		}

		//Parse the trustee's public keys
		for i:=0; i<len(trusteesPublicKeys); i++ {
			clientState.TrusteePublicKey[i] = trusteesPublicKeys[i]
			clientState.sharedSecrets[i] = suite.Point().Mul(trusteesPublicKeys[i], clientState.privateKey)
		}

		//check that we got all keys
		for i := 0; i<clientState.nTrustees; i++ {
			if clientState.TrusteePublicKey[i] == nil {
				panic("Client : didn't get the public key from trustee "+strconv.Itoa(i))
			}
		}

		clientState.printSecrets()
		println(">>>> All crypto stuff exchanged !")

		roundCount          := 0
		continueToNextRound := true
		for continueToNextRound {
			select {
				//downstream slice from relay (normal DC-net cycle)
				case data := <-dataFromRelay:
					print(".")

					switch data.messageType {

						case 3 : //relay wants to re-setup (new key exchanges)
							fmt.Println("Relay warns that a client disconnected, gonna resync..")
							continueToNextRound = false

						case 1 : //relay wants to re-setup (new key exchanges)
							fmt.Println("Relay wants to resync")
							continueToNextRound = false

						case 0 : //data for SOCKS proxy, just hand it over to the dedicated thread
							dataForSocksProxy <- data
					}

					// TODO Should account the downstream cell in the history

					// Produce and ship the next upstream slice
					writeNextUpstreamSlice(dataForRelayBuffer, relayConn, clientState)

					//we report the speed, bytes exchanged, etc
					stats.report()
			}

			if roundCount > 10 && socksConnId == 1 {
				fmt.Println("10/1 GONNA EXIT")
				os.Exit(1)
			}

			roundCount++
		}
	}
}

/*
 * Creates the next cell
 */

func writeNextUpstreamSlice(dataForRelayBuffer chan []byte, relayConn net.Conn, clientState *ClientState) {
	var nextUpstreamBytes []byte

	select
	{
		case nextUpstreamBytes = <-dataForRelayBuffer:

		default:
	}

	//produce the next upstream cell
	upstreamSlice := clientState.CellCoder.ClientEncode(nextUpstreamBytes, clientState.PayloadLength, clientState.MessageHistory)

	if len(upstreamSlice) != clientState.UsablePayloadLength {
		panic("Client slice wrong size, expected "+strconv.Itoa(clientState.UsablePayloadLength)+", but got "+strconv.Itoa(len(upstreamSlice)))
	}

	n, err := relayConn.Write(upstreamSlice)
	if n != len(upstreamSlice) {
		panic("Client write to relay error, expected writing "+strconv.Itoa(len(upstreamSlice))+", but wrote "+strconv.Itoa(n)+", err : " + err.Error())
	}
}


/*
 * RELAY CONNECTION
 */

func connectToRelay(relayHost string, connectionId int, params *ClientState) net.Conn {
	conn, err := net.Dial("tcp", relayHost)
	if err != nil {
		panic("Can't connect to relay:" + err.Error())
	}


	//tell the relay our public key
	publicKeyBytes, _ := params.PublicKey.MarshalBinary()
	keySize := len(publicKeyBytes)

	buffer := make([]byte, 12+keySize)
	binary.BigEndian.PutUint32(buffer[0:4], uint32(LLD_PROTOCOL_VERSION))
	binary.BigEndian.PutUint32(buffer[4:8], uint32(connectionId))
	binary.BigEndian.PutUint32(buffer[8:12], uint32(keySize))
	copy(buffer[12:], publicKeyBytes)

	n, err := conn.Write(buffer)

	if n < 12+keySize || err != nil {
		panic("Error writing to socket:" + err.Error())
	}

	return conn
}

func readDataFromRelay(relayConn net.Conn, dataFromRelay chan<- dataWithMessageTypeAndConnId, publicKeysFromRelay chan<- []abstract.Point) {
	header := [10]byte{}
	totcells := uint64(0)
	totbytes := uint64(0)

	for {
		// Read the next (downstream) header from the relay
		n, err := io.ReadFull(relayConn, header[:])

		if n != len(header) {
			panic("clientReadRelay: " + err.Error())
		}

		//parse the header
		messageType := int(binary.BigEndian.Uint32(header[0:4]))
		socksConnId := int(binary.BigEndian.Uint32(header[4:8]))
		dataLength  := int(binary.BigEndian.Uint16(header[8:10]))

		fmt.Println("Read a message with type", messageType, " socks id ", socksConnId, "data length", dataLength)

		// Read the data
		data := make([]byte, dataLength)
		n, err = io.ReadFull(relayConn, data)

		if messageType == MESSAGE_TYPE_PUBLICKEYS {
			//Public key arrays

			println("<<<<<<<<<<<<<<<<<<")
			println("Data has size")
			println(len(data))

			publicKeys := util.UnMarshalPublicKeyArrayFromByteArray(data, suite)
			println("unmarshalling done")
			publicKeysFromRelay <- publicKeys
			println("Exiting...")

		}  else {
			// Data
			data := make([]byte, dataLength)
			n, err = io.ReadFull(relayConn, data)

			if n != dataLength {
				panic("readDataFromRelay: read data length ("+strconv.Itoa(n)+") not matching expected length ("+strconv.Itoa(dataLength)+")" + err.Error())
			}

			dataFromRelay <- dataWithMessageTypeAndConnId{messageType, socksConnId, data}

			totcells++
			totbytes += uint64(dataLength)
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

func startSocksProxyServerHandler(socksProxyNewConnections chan net.Conn, dataForRelayBuffer chan []byte, dataForSOCKSProxy chan dataWithMessageTypeAndConnId, clientState *ClientState) {

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
				messageType := dataTypeConn.messageType //we know it's data for relay
				socksConnId   := dataTypeConn.connectionId
				data          := dataTypeConn.data
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

func channelCleaner(channel chan dataWithMessageTypeAndConnId){
	for{
		select{
			case x := <- channel:
				_ = x //could this be more ugly ?
				//do nothing
			default:
				time.Sleep(1000 * time.Millisecond)
		}
	}
}