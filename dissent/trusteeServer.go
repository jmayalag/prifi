package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	//"github.com/lbarman/prifi/dcnet"
	//"log"
	"encoding/hex"
	"time"
	"net"
	"github.com/lbarman/prifi/dcnet"
	"github.com/lbarman/crypto/abstract"
	//log2 "github.com/lbarman/prifi/log"
)


const listeningPort = ":9000"

func startTrusteeServer() {

	fmt.Printf("Starting Trustee Server \n")

	//async listen for incoming connections
	newConnections := make(chan net.Conn)
	go startListening(listeningPort, newConnections)

	//active connections will be hold there
	activeConnections := make([]net.Conn, 0)

	//handler warns the handler when a connection closes
	closedConnections := make(chan int)

	for {
		select {

			// New TCP connection
			case newConn := <-newConnections:
				newConnId := len(activeConnections)
				activeConnections = append(activeConnections, newConn)

				go handleConnection(newConnId, newConn, closedConnections)

		}
	}
}


func startListening(listenport string, newConnections chan<- net.Conn) {
	fmt.Printf("Listening on port %s\n", listenport)

	lsock, err := net.Listen("tcp", listenport)

	if err != nil {
		fmt.Printf("Can't open listen socket at port %s: %s", listenport, err.Error())
		return
	}
	for {
		conn, err := lsock.Accept()
		fmt.Printf("Accepted on port %s\n", listenport)

		if err != nil {
			fmt.Printf("Accept error: %s", err.Error())
			lsock.Close()
			return
		}
		newConnections <- conn
	}
}

type TrusteeCryptoParams struct {
	Name				string

	PublicKey			abstract.Point
	privateKey			abstract.Secret
	
	ClientPublicKeys	[]abstract.Point
	sharedSecrets		[]abstract.Point
	
	CellCoder			dcnet.CellCoder
	
	MessageHistory		abstract.Cipher
}


func initateTrusteeCrypto(trusteeId int, nClients int) *TrusteeCryptoParams {

	params := new(TrusteeCryptoParams)

	params.Name = "Trustee-"+strconv.Itoa(trusteeId)

	//prepare the crypto parameters
	rand 	:= suite.Cipher([]byte(params.Name))
	base	:= suite.Point().Base()

	//generate own parameters
	params.privateKey       = suite.Secret().Pick(rand)
	params.PublicKey        = suite.Point().Mul(base, params.privateKey)

	//placeholders for pubkeys and secrets
	params.ClientPublicKeys = make([]abstract.Point, nClients)
	params.sharedSecrets    = make([]abstract.Point, nClients)

	//sets the cell coder, and the history
	params.CellCoder = factory()

	return params
}

func handleConnection(connId int,conn net.Conn, closedConnections chan int){
	
	defer conn.Close()

	buffer := make([]byte, 1024)
	
	// Read the incoming connection into the bufferfer.
	_, err := conn.Read(buffer)
	if err != nil {
	    fmt.Println(">>>> Trustee", connId, "error reading:", err.Error())
	    return;
	}

	//Check the protocol version against ours
	version := int(binary.BigEndian.Uint32(buffer[0:4]))

	if(version != LLD_PROTOCOL_VERSION) {
		fmt.Println(">>>> Trustee", connId, "client version", version, "!= server version", LLD_PROTOCOL_VERSION)
		return;
	}

	//Extract the global parameters
	cellSize := int(binary.BigEndian.Uint32(buffer[4:8]))
	nClients := int(binary.BigEndian.Uint32(buffer[8:12]))
	nTrustees := int(binary.BigEndian.Uint32(buffer[12:16]))
	trusteeId := int(binary.BigEndian.Uint32(buffer[16:20]))
	fmt.Println(">>>> Trustee", connId, "setup is", nClients, "clients", nTrustees, "trustees, role is", trusteeId, "cellSize ", cellSize)

	
	//prepare the crypto parameters
	cryptoParams := initateTrusteeCrypto(trusteeId, nClients)
	tellPublicKey(conn, cryptoParams.PublicKey)

	//Read the clients' public keys from the connection
	clientsPublicKeys := UnMarshalPublicKeyArrayFromConnection(conn)
	for i:=0; i<len(clientsPublicKeys); i++ {
		cryptoParams.ClientPublicKeys[i] = clientsPublicKeys[i]
		cryptoParams.sharedSecrets[i] = suite.Point().Mul(clientsPublicKeys[i], cryptoParams.privateKey)
	}

	//check that we got all keys
	for i := 0; i<nClients; i++ {
		if cryptoParams.ClientPublicKeys[i] == nil {
			panic("Trustee : didn't get the public key from client "+strconv.Itoa(i))
		}
	}

	//print all shared secrets
	for i:=0; i<nClients; i++ {
		fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
		fmt.Println("            Client", i)
		d1, _ := cryptoParams.ClientPublicKeys[i].MarshalBinary()
		d2, _ := cryptoParams.sharedSecrets[i].MarshalBinary()
		fmt.Println(hex.Dump(d1))
		fmt.Println("+++")
		fmt.Println(hex.Dump(d2))
		fmt.Println("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
	}

	println("All crypto stuff exchanged !")

	for {
		time.Sleep(5000 * time.Millisecond)
	}

	startTrusteeSlave(conn, trusteeId, cellSize, nClients, nTrustees, cellSize, closedConnections)

	fmt.Println(">>>> Trustee", connId, "shutting down.")
	conn.Close()
}


func startTrusteeSlave(conn net.Conn, tno int, payloadLength int, nClients int, nTrustees int, cellSize int, closedConnections chan int) {
	tg := dcnet.TestSetup(nil, suite, factory, nClients, nTrustees)
	me := tg.Trustees[tno]

	//me.Dump(tno)

	upload := make(chan []byte)
	go trusteeConnRead(tno, payloadLength, conn, upload, closedConnections)

	// Just generate ciphertext cells and stream them to the server.
	exit := false
	i := 0
	for !exit {
		select {
			case readByte := <- upload:
				fmt.Println("Received byte ! ", readByte)

			case connClosed := <- closedConnections:
				if connClosed == tno {
					fmt.Println("[safely stopping handler "+strconv.Itoa(tno)+"]")
					return;
				}

			default:
				// Produce a cell worth of trustee ciphertext
				tslice := me.Coder.TrusteeEncode(cellSize)

				// Send it to the relay
				//println("trustee slice")
				//println(hex.Dump(tslice))
				n, err := conn.Write(tslice)

				i += 1
				fmt.Printf("["+strconv.Itoa(i)+":"+strconv.Itoa(tno)+"/"+strconv.Itoa(nClients)+","+strconv.Itoa(nTrustees)+"]")
				
				if n < len(tslice) || err != nil {
					//fmt.Println("can't write to socket: " + err.Error())
					//fmt.Println("\nShutting down handler", tno, "of conn", conn.RemoteAddr())
					fmt.Println("[error, stopping handler "+strconv.Itoa(tno)+"]")
					exit = true
				}

		}
	}
}


func trusteeConnRead(tno int, payloadLength int, conn net.Conn, readChan chan<- []byte, closedConnections chan<- int) {

	for {
		// Read up to a cell worth of data to send upstream
		buf := make([]byte, 512)
		n, err := conn.Read(buf)

		// Connection error or EOF?
		if n == 0 {
			if err == io.EOF {
				fmt.Println("[read EOF, trustee "+strconv.Itoa(tno)+"]")
			} else {
				fmt.Println("[read error, trustee "+strconv.Itoa(tno)+" ("+err.Error()+")]")
				conn.Close()
				return
			}
		} else {
			readChan <- buf
		}
	}
}
