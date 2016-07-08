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

    		myBytes := data.toBytes()
    		offset := 6

    		fmt.Println("Connection",binary.BigEndian.Uint32(myBytes[0:4]),": Sending Message of length",data.N,"to Server")

			if(data.N>3) {
				fmt.Println("version", myBytes[offset+0])
				fmt.Println("command", myBytes[offset+1])
				fmt.Println("reserved", myBytes[offset+2])
				fmt.Println("adderess type", myBytes[offset+3])
				if int(myBytes[offset+3]) == 4 {
					fmt.Println("IP adderess", net.IP(myBytes[offset+4:offset+20]).String())
					fmt.Println("Port", binary.BigEndian.Uint16(myBytes[offset+20:offset+22]))
				} else {
					fmt.Println("IP adderess", net.IP(myBytes[offset+4:offset+8]).String())
					fmt.Println("Port", binary.BigEndian.Uint16(myBytes[offset+8:offset+10]))
				}

				conn.Write(data.toBytes())
			} else {
				fmt.Println("version", myBytes[offset+0])
				
			}


	   		conn.Write(data.toBytes())

		}
	}()

	for {
		newIDBytes := make([]byte, 4)
    	_, errID := io.ReadFull(connReader,newIDBytes)
   
		fmt.Println("Received ConnID from server")	
		message, errMessage := connReader.ReadBytes('\n')

		fmt.Println("Received the data from server")	
		newID := binary.BigEndian.Uint32( newIDBytes )
		
		fmt.Println("Received reply from server for connection",newID)		


		if errID == nil && errMessage == nil {
			fromServer <- NewDataWrap( newID , uint16(len(message)), message)
   			fmt.Println("Message from server: "+ string(message))		
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
			fmt.Println("We have a reply message to send to the broswer!")	
			socksConnId := myData.ID
			data := myData.Data
			dataLength := len(data)

			//Handle the connections, forwards the downstream slice to the SOCKS proxy
			//if there is no socks proxy, nothing to do (useless case indeed, only for debug)
			if dataLength > 0 && socksProxyActiveConnections[socksConnId] != nil {
				fmt.Println("Sending Reply to Browser")	
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
