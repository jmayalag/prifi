package main

import (
	"net"
	"fmt"
	"bufio"
	"encoding/binary"
	"io"
)

type dataWrap struct {
	ID 		uint32
	N 		uint16
	Data 	[] byte
}

func NewDataWrap(ID uint32, N uint16, Data []byte ) dataWrap {
    return dataWrap{ID, N, Data}
}

func (d *dataWrap) toBytes() []byte {
	// Read up to a cell worth of data to send upstream
	buffer := make([]byte, 6)
	// Encode the connection number and actual data length
	binary.BigEndian.PutUint32(buffer[0:4], d.ID)
	binary.BigEndian.PutUint16(buffer[4:6], d.N)

	return append(buffer,d.Data[0:d.N]...)
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
	connReader := bufio.NewReader(conn)
	fmt.Println("Connected To Server")

	go func() {
		for {
			data := <-toServer
	   		conn.Write(data.toBytes())

		}
	}()

	for {
		connID, messageLength, err :=  readHeader(connReader)

		message := make([]byte, messageLength)
    	_, errMessage := io.ReadFull(connReader,message)

		if err == nil && errMessage == nil {
			fromServer <- NewDataWrap( connID , messageLength, message)
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
	socksProxyConnClosed := make(chan uint32)

	for {
		select {

		// New TCP connection to the SOCKS proxy
		case conn := <-newConnections:
			newSocksProxyId := uint32(len(socksProxyActiveConnections))
			socksProxyActiveConnections = append(socksProxyActiveConnections, conn)
			go readDataFromSocksProxy(newSocksProxyId, 50, conn, toServer, socksProxyConnClosed)
			fmt.Println("Connection", newSocksProxyId, "Established")

		// Plaintext downstream data (relay->client->Socks proxy)
		case myData := <-fromServer:
			socksConnId := myData.ID
			data := myData.Data
			dataLength := len(data)

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

func readDataFromSocksProxy(socksConnId uint32, payloadLength int, conn net.Conn, toServer chan dataWrap, closed chan<- uint32) {
	for {
		// Read up to a cell worth of data to send upstream
		buffer := make([]byte, payloadLength)
		n, _ := conn.Read(buffer)

		// Connection error or EOF?
		if n == 0 {
			fmt.Println("Connection", socksConnId, "Closed")
			conn.Close()
			closed <- socksConnId // signal that channel is closed
			return
		}

		toServer <- NewDataWrap(socksConnId,uint16(n),buffer)

	}
}


func readHeader(connReader io.Reader) (uint32, uint16, error) {
  
  controlHeader := make([]byte, 6)

  _, err := io.ReadFull(connReader,controlHeader)  
  if err != nil {
    return 0, 0, err
  }

  connID := binary.BigEndian.Uint32(controlHeader[0:4])
  messageLength := binary.BigEndian.Uint16(controlHeader[4:6])

  return connID, messageLength, nil

}