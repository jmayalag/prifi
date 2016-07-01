package main

import (
	"net"
	"fmt"
	"bufio"
	"encoding/binary"
	"strconv"
)

type dataWrap struct {
	ID 		int
	N 		int
	Data 	[] byte
}

func NewDataWrap(ID int, N int, Data []byte ) *dataWrap {
    return &dataWrap{ID, N, Data}
}

func (d *dataWrap) toBytes() []byte {
	// Read up to a cell worth of data to send upstream
	n := len(d.Data)
	buffer := make([]byte, 6)
	// Encode the connection number and actual data length
	binary.BigEndian.PutUint32(buffer[0:4], uint32(d.ID))
	binary.BigEndian.PutUint16(buffer[4:6], uint16(n))

	return append(buffer,d.Data...)
}

func main() {
	toServer := make(chan dataWrap, 1)
	fromServer := make(chan dataWrap, 1)
	socksConnections := make(chan net.Conn, 1)

  	go startSocksProxyServerListener(":6789",socksConnections)
  	go startSocksProxyServerHandler(socksConnections, toServer, fromServer)

  	connectToServer("127.0.0.1:8081",toServer, fromServer)
}


func connectToServer(IP string, toServer chan dataWrap, fromServer chan dataWrap) {
	// connect to this socket
	conn, _ := net.Dial("tcp", IP)
	fmt.Println("Connected To Server")

	go func() {
		for {
			data := <-toServer
    		fmt.Println("Sending Text ")
    		conn.Write([]byte(string(data.Data) + "\n"))
		}
	}()

	for {
		newIDString, err1 := bufio.NewReader(conn).ReadString('\n')
		message, err2 := bufio.NewReader(conn).ReadBytes('\n')
		newID, _ := strconv.Atoi(newIDString)
		
		if err1 == nil && err2 == nil {
			fromServer <- *NewDataWrap( newID , len(message), message)
   			fmt.Print("Message from server: "+ string(message))		
		}
	}

}


func startSocksProxyServerListener(port string, newConnections chan<- net.Conn) {
	lsock, err := net.Listen("tcp", port)

	if err != nil {
		return
	}

	for {
		conn, err := lsock.Accept()

		if err != nil {
			lsock.Close()
			return
		}

		newConnections <- conn
	}
}

func startSocksProxyServerHandler(newConnections chan net.Conn, toServer chan dataWrap, fromServer chan dataWrap) {

	socksProxyActiveConnections := make([]net.Conn, 1) // reserve socksProxyActiveConnections[0]
	socksProxyConnClosed := make(chan int)

	for {
		select {

		// New TCP connection to the SOCKS proxy
		case conn := <-newConnections:
			newSocksProxyId := len(socksProxyActiveConnections)
			socksProxyActiveConnections = append(socksProxyActiveConnections, conn)
			go readDataFromSocksProxy(newSocksProxyId, 1000, conn, toServer, socksProxyConnClosed)

		// Plaintext downstream data (relay->client->Socks proxy)
		case myData := <-fromServer:
			socksConnId := myData.ID
			data := myData.Data
			dataLength := myData.N

			//Handle the connections, forwards the downstream slice to the SOCKS proxy
			//if there is no socks proxy, nothing to do (useless case indeed, only for debug)
			if dataLength > 0 && socksProxyActiveConnections[socksConnId] != nil {
				socksProxyActiveConnections[socksConnId].Write(data)
			} else {
				socksProxyActiveConnections[socksConnId].Close()
			}
			

		//connection closed from SOCKS proxy
		case socksConnId := <-socksProxyConnClosed:
			socksProxyActiveConnections[socksConnId] = nil
		}
	}
}

func readDataFromSocksProxy(socksConnId int, payloadLength int, conn net.Conn, toServer chan dataWrap, closed chan<- int) {
	for {
		// Read up to a cell worth of data to send upstream
		buffer := make([]byte, payloadLength)
		n, _ := conn.Read(buffer[6:])

		fmt.Println(string(buffer[7]))

//		toServer <- *NewDataWrap(socksConnId,n,buffer)

		// Connection error or EOF?
		if n == 0 {
			conn.Close()
			closed <- socksConnId // signal that channel is closed
			return
		}
	}
}