package main

import (
	"encoding/binary"
	"fmt"
	"io"
	//"github.com/lbarman/prifi/dcnet"
	//"log"
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

	//handler warns us when a connection closedConnectionss
	closedConnections := make(chan int)

	for {
		select {

			// New TCP connection
			case newConn := <-newConnections:
				newConnId := len(activeConnections)
				activeConnections = append(activeConnections, newConn)

				go handleConnection(newConnId, newConn, closedConnections)

			//Maybe should handle stuff from closedConnections
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

func handleConnection(connId int, conn net.Conn, closedConnections chan<- int){
	
	buffer := make([]byte, 1024)
	
	// Read the incoming connection into the bufferfer.
	reqLen, err := conn.Read(buffer)
	if err != nil {
	    fmt.Println("Handler", connId, "error reading:", err.Error())
	}

	fmt.Println("Handler", connId, "len", reqLen)

	ver := int(binary.BigEndian.Uint32(buffer[0:4]))

	if(ver != LLD_PROTOCOL_VERSION) {
		fmt.Println("Handler", connId, "client version", ver, "!= server version", LLD_PROTOCOL_VERSION)
		conn.Close()
		closedConnections <- connId
	}

	cellSize := int(binary.BigEndian.Uint32(buffer[4:8]))
	nClients := int(binary.BigEndian.Uint32(buffer[8:12]))
	nTrustees := int(binary.BigEndian.Uint32(buffer[12:16]))
	trusteeId := int(binary.BigEndian.Uint32(buffer[16:20]))


	fmt.Println("Handler", connId, "setup is", nClients, nTrustees, "role is", trusteeId, "cellSize ", cellSize)

	//TODO : wait for crypto parameters from clients

	startTrusteeSlave(conn, trusteeId, nClients, nTrustees, cellSize)

	fmt.Println("Handler", connId, "shutting down.")
	conn.Close()
}


func startTrusteeSlave(conn net.Conn, tno int, nClients int, nTrustees int, cellSize int) {
	tg := dcnet.TestSetup(nil, suite, factory, nClients, nTrustees)
	me := tg.Trustees[tno]

	me.Dump(tno)

	upload := make(chan []byte)
	//go trusteeConnRead(conn, upload)

	// Just generate ciphertext cells and stream them to the server.
	exit := false
	for !exit {
		select {
			case readByte := <- upload:
				fmt.Println("Received byte ! ", readByte)

			default:
				// Produce a cell worth of trustee ciphertext
				tslice := me.Coder.TrusteeEncode(cellSize)

				// Send it to the relay
				//println("trustee slice")
				//println(hex.Dump(tslice))
				n, err := conn.Write(tslice)

				print(tno)
				
				if n < len(tslice) || err != nil {
					//fmt.Println("can't write to socket: " + err.Error())
					fmt.Println("\nShutting down handler", tno, "of conn", conn.RemoteAddr())
					exit = true
				}

		}
	}
}


func trusteeConnRead(conn net.Conn, readChan chan<- []byte) {

	for {
		// Read up to a cell worth of data to send upstream
		buf := make([]byte, payloadlen)
		n, err := conn.Read(buf[proxyhdrlen:])

		// Connection error or EOF?
		if n == 0 {
			if err == io.EOF {
				println("trusteeConnRead: EOF from relay, closing")
			} else {
				println("trusteeConnRead: " + err.Error())
			}
			conn.Close()
			return
		} else {
			readChan <- buf
		}
	}
}
