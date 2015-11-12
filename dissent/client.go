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
)

// Number of bytes of cell payload to reserve for connection header, length
const socksHeaderLength = 6

type ClientCryptoParams struct {
	Name				string

	PublicKey			abstract.Point
	privateKey			abstract.Secret
	
	TrusteePublicKey	[]abstract.Point
	sharedSecrets		[]abstract.Point
	
	CellCoder			dcnet.CellCoder
	
	MessageHistory		abstract.Cipher
}

func initateClientCrypto(clientId int, nTrustees int) *ClientCryptoParams {

	params := new(ClientCryptoParams)

	params.Name = "Client-"+strconv.Itoa(clientId)

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
	params.CellCoder = factory()

	return params
}

func startClient(clientId int, relayHostAddr string, nClients int, nTrustees int, payloadLength int, useSocksProxy bool) {
	fmt.Printf("startClient %d\n", clientId)

	//crypto parameters
	cryptoParams := initateClientCrypto(clientId, nTrustees)
	clientPayloadSize := cryptoParams.CellCoder.ClientCellSize(payloadLength)

	relayConn := connectToRelay(relayHostAddr, clientId, cryptoParams)

	//Read the trustee's public keys from the connection
	trusteesPublicKeys := util.UnMarshalPublicKeyArrayFromConnection(relayConn, suite)
	for i:=0; i<len(trusteesPublicKeys); i++ {
		cryptoParams.TrusteePublicKey[i] = trusteesPublicKeys[i]
		cryptoParams.sharedSecrets[i] = suite.Point().Mul(trusteesPublicKeys[i], cryptoParams.privateKey)
	}

	//check that we got all keys
	for i := 0; i<nTrustees; i++ {
		if cryptoParams.TrusteePublicKey[i] == nil {
			panic("Client : didn't get the public key from trustee "+strconv.Itoa(i))
		}
	}

	//print all shared secrets
	for i:=0; i<nTrustees; i++ {
		fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
		fmt.Println("            TRUSTEE", i)
		d1, _ := cryptoParams.TrusteePublicKey[i].MarshalBinary()
		d2, _ := cryptoParams.sharedSecrets[i].MarshalBinary()
		fmt.Println(hex.Dump(d1))
		fmt.Println("+++")
		fmt.Println(hex.Dump(d2))
		fmt.Println("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
	}

	println("All crypto stuff exchanged !")

	//initiate downstream stream
	dataFromRelay := make(chan dataWithConnectionId)
	go readDataFromRelay(relayConn, dataFromRelay)

	println("client", clientId, "connected")

	// We're the "slot owner" - start a socks relay
	socksProxyNewConnections    := make(chan net.Conn)
	socksProxyData              := make(chan []byte)
	socksProxyConnClosed        := make(chan int)
	socksProxyActiveConnections := make([]net.Conn, 1) // reserve socksProxyActiveConnections[0]
	
	if(useSocksProxy){
		port := ":" + strconv.Itoa(1080+clientId)
		go startSocksProxy(port, socksProxyNewConnections)
	}

	// This will hold the data to be sent later on to the relay, anonymized
	dataForRelayBuffer := make([][]byte, 0)

	// Client/proxy main loop
	totupcells := uint64(0)
	totupbytes := uint64(0)
	for {
		select {

			// New TCP connection to the SOCKS proxy
			case conn := <-socksProxyNewConnections: 
				newClientId := len(socksProxyActiveConnections)
				socksProxyActiveConnections = append(socksProxyActiveConnections, conn)
				go readDataFromSocksProxy(newClientId, payloadLength, conn, socksProxyData, socksProxyConnClosed)

			// Data to anonymize from SOCKS proxy
			case data := <-socksProxyData: 
				dataForRelayBuffer = append(dataForRelayBuffer, data)

			//connection closed from SOCKS proxy
			case clientId := <-socksProxyConnClosed:
				socksProxyActiveConnections[clientId] = nil

			//downstream slice from relay (normal DC-net cycle)
			case dataWithConnId := <-dataFromRelay:
				print(".")

				connId := dataWithConnId.connectionId
				
				//Handle the connections, forwards the downstream slice to the SOCKS proxy
				if connId > 0 && connId < len(socksProxyActiveConnections) && socksProxyActiveConnections[connId] != nil {
					data       := dataWithConnId.data
					dataLength := len(data)

					if dataLength > 0 {

						//if there is no socks proxy, nothing to do (useless case indeed, only for debug)
						if useSocksProxy {
							n, err := socksProxyActiveConnections[clientId].Write(data)
							if n < dataLength {
								panic("Write to socks proxy: expected "+strconv.Itoa(dataLength)+" bytes, got "+strconv.Itoa(n)+", " + err.Error())
							}
						}
					} else {
						// Relay indicating EOF on this conn
						fmt.Printf("Relay to client : closed conn %d", connId)
						socksProxyActiveConnections[clientId].Close()
					}
				}

				// Should account the downstream cell in the history

				// Produce and ship the next upstream slice
				writeNextUpstreamSlice(dataForRelayBuffer, payloadLength, clientPayloadSize, relayConn, cryptoParams)

				//statistics
				totupcells++
				totupbytes += uint64(payloadLength)
				//fmt.Printf("sent %d upstream cells, %d bytes\n", totupcells, totupbytes)
			}
	}
}

/*
 * Creates the next cell
 */

func writeNextUpstreamSlice(dataForRelayBuffer [][]byte, payloadLength int, clientPayloadSize int, relayConn net.Conn, cryptoParams *ClientCryptoParams) {
	var nextUpstreamBytes []byte
	if len(dataForRelayBuffer) > 0 {
		nextUpstreamBytes  = dataForRelayBuffer[0]
		dataForRelayBuffer = dataForRelayBuffer[1:]
		//fmt.Printf("\n^ %v (len : %d)\n", p)
	}

	//produce the next upstream cell
	upstreamSlice := cryptoParams.CellCoder.ClientEncode(nextUpstreamBytes, payloadLength, cryptoParams.MessageHistory)

	if len(upstreamSlice) != clientPayloadSize {
		panic("Client slice wrong size, expected "+strconv.Itoa(clientPayloadSize)+", but got "+strconv.Itoa(len(upstreamSlice)))
	}

	n, err := relayConn.Write(upstreamSlice)
	if n != len(upstreamSlice) {
		panic("Client write to relay error, expected writing "+strconv.Itoa(len(upstreamSlice))+", but wrote "+strconv.Itoa(n)+", err : " + err.Error())
	}
}


/*
 * RELAY CONNECTION
 */

func connectToRelay(relayHost string, connectionId int, params *ClientCryptoParams) net.Conn {
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

func readDataFromRelay(relayConn net.Conn, datadataFromRelay chan<- dataWithConnectionId) {
	header := [6]byte{}
	totcells := uint64(0)
	totbytes := uint64(0)

	for {
		// Read the next (downstream) header from the relay
		n, err := io.ReadFull(relayConn, header[:])

		if n != len(header) {
			panic("clientReadRelay: " + err.Error())
		}

		connectionId := int(binary.BigEndian.Uint32(header[0:4]))
		dataLength := int(binary.BigEndian.Uint16(header[4:6]))

		// Read the downstream data
		data := make([]byte, dataLength)
		n, err = io.ReadFull(relayConn, data)

		if n != dataLength {
			panic("readDataFromRelay: read data length ("+strconv.Itoa(n)+") not matching expected length ("+strconv.Itoa(dataLength)+")" + err.Error())
		}

		datadataFromRelay <- dataWithConnectionId{connectionId, data}

		totcells++
		totbytes += uint64(dataLength)
	}
}

/*
 * SOCKS PROXY
 */

func startSocksProxy(port string, newConnections chan<- net.Conn) {
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


func readDataFromSocksProxy(clientId int, payloadLength int, conn net.Conn, data chan<- []byte, closed chan<- int) {
	for {
		// Read up to a cell worth of data to send upstream
		buffer := make([]byte, payloadLength)
		n, err := conn.Read(buffer[socksHeaderLength:])

		// Encode the connection number and actual data length
		binary.BigEndian.PutUint32(buffer[0:4], uint32(clientId))
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
			closed <- clientId // signal that channel is closed
			return
		}
	}
}