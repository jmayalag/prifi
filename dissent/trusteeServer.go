package main

import (
	"encoding/binary"
	"fmt"
	"github.com/dedis/prifi/dcnet"
	//"log"
	"net"
	//log2 "github.com/lbarman/prifi/log"
)


const srvListenPort = ":9000"

func startTrusteeSrv() {

	fmt.Printf("Starting Trustee Server \n")

	//async listen for incoming connections
	newconn := make(chan net.Conn)
	go srvListen(srvListenPort, newconn)

	//active connections will be hold there
	conns := make([]net.Conn, 0)

	//handler warns us when a connection closes
	close := make(chan int)


	for {
		select {
			case conn := <-newconn: // New TCP connection
				cno := len(conns)
				conns = append(conns, conn)

				go handleConnection(cno, conn, close)
		}
	}
}


func srvListen(listenport string, newconn chan<- net.Conn) {
	fmt.Printf("Listening on port %s\n", listenport)
	lsock, err := net.Listen("tcp", listenport)
	if err != nil {
		fmt.Printf("Can't open listen socket at port %s: %s",
			listenport, err.Error())
		return
	}
	for {
		conn, err := lsock.Accept()
		fmt.Printf("Accept on port %s\n", listenport)
		if err != nil {
			//log.Printf("Accept error: %s", err.Error())
			lsock.Close()
			return
		}
		newconn <- conn
	}
}

func handleConnection(cno int, conn net.Conn, close chan<- int){
	
	buf := make([]byte, 1024)
	
	// Read the incoming connection into the buffer.
	reqLen, err := conn.Read(buf)
	if err != nil {
	    fmt.Println("Handler", cno, "error reading:", err.Error())
	}

	fmt.Println("Handler", cno, "len", reqLen)

	ver := int(binary.BigEndian.Uint32(buf[0:4]))

	if(ver != LLD_PROTOCOL_VERSION) {
		fmt.Println("Handler", cno, "client version", ver, "!= server version", LLD_PROTOCOL_VERSION)

		conn.Close()
		close <- cno
	}

	cellsize := int(binary.BigEndian.Uint32(buf[4:8]))
	nClients := int(binary.BigEndian.Uint32(buf[8:12]))
	nTrustees := int(binary.BigEndian.Uint32(buf[12:16]))
	trusteeId := int(binary.BigEndian.Uint32(buf[16:20]))


	fmt.Println("Handler", cno, "setup is", nClients, nTrustees, "role is", trusteeId, "cellSize ", cellsize)

	startTrusteeSlave(conn, trusteeId, nClients, nTrustees, cellsize)

	fmt.Println("Handler", cno, "shutting down.")
	conn.Close()
}


func startTrusteeSlave(conn net.Conn, tno int, nclients int, ntrustees int, cellSize int) {
	tg := dcnet.TestSetup(nil, suite, factory, nclients, ntrustees)
	me := tg.Trustees[tno]

	upload := make(chan []byte)
	//go trusteeConnRead(conn, upload)

	// Just generate ciphertext cells and stream them to the server.
	for {
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
				if n < len(tslice) || err != nil {
					panic("can't write to socket: " + err.Error())
				}

		}
	}
}
