package prifi_socks

import (
	"net"
	"fmt"
	"bufio"
	"io"
)








func ConnectToServer(IP string, toServer chan []byte, fromServer chan []byte) {
	// connect to this socket
	conn, _ := net.Dial("tcp", IP)
	connReader := bufio.NewReader(conn)
	fmt.Println("Connected To Server")

	go func() {
		for {
			data := <-toServer
	   		conn.Write(data)
		}
	}()

	for {
		connID, messageLength, payloadLength, err :=  readHeader(connReader)

		message := make([]byte, payloadLength)
    	_, errMessage := io.ReadFull(connReader,message)

		if err == nil && errMessage == nil {
			newData := NewDataWrap( connID , messageLength, payloadLength, message[:messageLength])
			fromServer <- newData.ToBytes()
		}
	}

}


func StartSocksProxyServerListener(port string, newConnections chan<- net.Conn) {
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

func StartSocksProxyServerHandler(newConnections chan net.Conn, payloadLength int, toServer chan []byte, fromServer chan []byte) {

	socksProxyActiveConnections := make([]net.Conn, 1) // reserve socksProxyActiveConnections[0]
	socksProxyConnClosed := make(chan uint32)

	for {
		select {

		// New TCP connection to the SOCKS proxy
		case conn := <-newConnections:
			newSocksProxyId := uint32(len(socksProxyActiveConnections))
			socksProxyActiveConnections = append(socksProxyActiveConnections, conn)
			go readDataFromSocksProxy(newSocksProxyId, payloadLength, conn, toServer, socksProxyConnClosed)
			fmt.Println("Connection", newSocksProxyId, "Established")

		// Plaintext downstream data (relay->client->Socks proxy)
		case bufferData := <-fromServer:

			myData := extractData(bufferData)
			socksConnId := myData.ID
			data := myData.Data
			dataLength := len(data)

			//Handle the connections, forwards the downstream slice to the SOCKS proxy
			//if there is no socks proxy, nothing to do (useless case indeed, only for debug)
			if dataLength > 0 && socksProxyActiveConnections[socksConnId] != nil {
				socksProxyActiveConnections[socksConnId].Write(data)
			} else if socksProxyActiveConnections[socksConnId] != nil {
				socksProxyActiveConnections[socksConnId].Close()
			}
			

		//connection closed from SOCKS proxy
		case socksConnId := <-socksProxyConnClosed:
			socksProxyActiveConnections[socksConnId] = nil
		}
	}
}


func readDataFromSocksProxy(socksConnId uint32, payloadLength int, conn net.Conn, toServer chan []byte, closed chan<- uint32) {
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

		newData := NewDataWrap(socksConnId, uint16(n), uint16(payloadLength), buffer)

		toServer <- newData.ToBytes()

	}
}



