package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	//"github.com/lbarman/prifi/dcnet"
	//"log"
	"encoding/hex"
	"net"
	"github.com/lbarman/prifi/dcnet"
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

func handleConnection(connId int,conn net.Conn, closedConnections chan int){
	
	buffer := make([]byte, 1024)
	
	// Read the incoming connection into the bufferfer.
	reqLen, err := conn.Read(buffer)
	if err != nil {
	    fmt.Println(">>>> Handler", connId, "error reading:", err.Error())
	}

	fmt.Println(">>>> Handler", connId, "len", reqLen)

	ver := int(binary.BigEndian.Uint32(buffer[0:4]))

	if(ver != LLD_PROTOCOL_VERSION) {
		fmt.Println(">>>> Handler", connId, "client version", ver, "!= server version", LLD_PROTOCOL_VERSION)
		conn.Close()
		closedConnections <- connId
	}

	cellSize := int(binary.BigEndian.Uint32(buffer[4:8]))
	nClients := int(binary.BigEndian.Uint32(buffer[8:12]))
	nTrustees := int(binary.BigEndian.Uint32(buffer[12:16]))
	trusteeId := int(binary.BigEndian.Uint32(buffer[16:20]))
	fmt.Println(">>>> Handler", connId, "setup is", nClients, nTrustees, "role is", trusteeId, "cellSize ", cellSize)

	
	//prepare the crypto parameters
	rand 	:= suite.Cipher([]byte("trustee-"+strconv.Itoa(connId)))
	base	:= suite.Point().Base()
	privateKey  := suite.Secret().Pick(rand)
	publicKey   := suite.Point().Mul(base, privateKey)
	publicKeyBytes, _ := publicKey.MarshalBinary()
	keySize := len(publicKeyBytes)

	fmt.Println("TrusteeSrv >>>>> keylen is ", keySize)
	fmt.Println(hex.Dump(publicKeyBytes))

	//tell the relay our public key (assume user verify through second channel)
	buffer2 := make([]byte, 8+keySize)
	copy(buffer2[8:], publicKeyBytes)
	binary.BigEndian.PutUint32(buffer2[0:4], uint32(LLD_PROTOCOL_VERSION))
	binary.BigEndian.PutUint32(buffer2[4:8], uint32(keySize))

	fmt.Println("Writing", LLD_PROTOCOL_VERSION, "key of length", keySize, ", key is ", publicKey)

	n, err := conn.Write(buffer2)

	if n < 1 || err != nil {
		panic("Error writing to socket:" + err.Error())
	}

	//TODO : wait for crypto parameters from clients

	startTrusteeSlave(conn, trusteeId, cellSize, nClients, nTrustees, cellSize, closedConnections)

	fmt.Println(">>>> Handler", connId, "shutting down.")
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
