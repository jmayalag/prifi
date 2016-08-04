package prifi_socks

import (
	"net"
	"fmt"
  	"encoding/binary"
)

// Authentication methods
const (
  methNoAuth = iota
  methGSS
  methUserPass
  methNone = 0xff
)

// Address types
const (
  addrIPv4   = 0x01
  addrDomain = 0x03
  addrIPv6   = 0x04
)

// Commands
const (
  cmdConnect   = 0x01
  cmdBind      = 0x02
  cmdAssociate = 0x03
)

// Reply codes
const (
  repSucceeded = iota
  repGeneralFailure
  repConnectionNotAllowed
  repNetworkUnreachable
  repHostUnreachable
  repConnectionRefused
  repTTLExpired
  repCommandNotSupported
  repAddressTypeNotSupported
)

/** 
  * Connects the client to the proxy server and proxies messages between the client and the server
  */
func ConnectToServer(IP string, toServer chan []byte, fromServer chan []byte) {
	
    allConnections := make( map[uint32] net.Conn )

	// Forward client messages to the server
	for {
		data := <-toServer
		myPacket := ExtractFull(data)
	
		connID := myPacket.ID 
    	clientPacket :=  myPacket.Data[:myPacket.MessageLength]

      	//Datrawrap packets with ID=0 are discarded (this indicates a useless packet)
      	if connID == 0 {
        	continue
      	}

      	// Get the channel associated with the connection ID
      	myConn := allConnections[connID]

      	// If no channel exists yet, create one and setup a channel handler
      	if myConn == nil {

        	// Create a new channel for the new connection ID
			newConn, _ := net.Dial("tcp", IP)
	        allConnections[connID] = newConn

        	// Instantiate a channel handler
        	go handleConnection(newConn, fromServer, connID)

        	myConn = newConn
        }

		myConn.Write(clientPacket)

	}

}


func handleConnection(conn net.Conn, fromServer chan []byte, connID uint32) {
	
	// Forward server messages to the client
	for {
	    buf := make([]byte, 4096)
	    messageLength, err := conn.Read(buf)
	    if err != nil {
	      fmt.Println("Read Error")
	      return
	    }

    	newPacket := NewDataWrap(connID,uint16(messageLength),uint16(messageLength)+dataWrapHeaderSize,buf[:messageLength])  
		if err == nil {
			fromServer <- newPacket.ToBytes()
		}
	}

}


/** 
  * Listens and accepts connections at a certain port
  */
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

/** 
  * Handles SOCKS5 connections
  */
func StartSocksProxyServerHandler(newConnections chan net.Conn, payloadLength int, toServer chan []byte, fromServer chan []byte) {

	socksProxyActiveConnections := make([]net.Conn, 1) // reserve socksProxyActiveConnections[0]
	socksProxyConnClosed := make(chan uint32)
    counter := make( map[uint32] int )

	for {

		select {

		// New TCP connection to the SOCKS proxy
		case conn := <-newConnections:
			newSocksProxyId := uint32(len(socksProxyActiveConnections))
			socksProxyActiveConnections = append(socksProxyActiveConnections, conn)
			counter[newSocksProxyId] = 1

			go readDataFromSocksProxy(newSocksProxyId, payloadLength, conn, toServer, socksProxyConnClosed)
			fmt.Println("Connection", newSocksProxyId, "Established")

		// Plaintext downstream data (relay->client->Socks proxy)
		case bufferData := <-fromServer:

			myData := ExtractFull(bufferData)
			socksConnId := myData.ID
			dataLength := myData.MessageLength
			data := myData.Data[:dataLength]

			//Handle the connections, forwards the downstream slice to the SOCKS proxy
			//if there is no socks proxy, nothing to do (useless case indeed, only for debug)
			if socksConnId == 0 {
				continue
			} else if dataLength > 0 && socksProxyActiveConnections[socksConnId] != nil {
				
				if counter[socksConnId] == 2 {
					data = replaceData(data, socksProxyActiveConnections[socksConnId].LocalAddr())
				}
				socksProxyActiveConnections[socksConnId].Write(data)
				counter[socksConnId]++

			} else if socksProxyActiveConnections[socksConnId] != nil {
				socksProxyActiveConnections[socksConnId].Close()
			}
			

		//connection closed from SOCKS proxy
		case socksConnId := <-socksProxyConnClosed:
			socksProxyActiveConnections[socksConnId] = nil
		}
	}
}

func replaceData(buf []byte, addr net.Addr) []byte {
	buf = buf[:4]

	//Check if address exists
	if addr != nil {

		// Extract Address type
		tcpaddr := addr.(*net.TCPAddr)
		host4 := tcpaddr.IP.To4()
		host6 := tcpaddr.IP.To16()

		port := [2]byte{} // Create byte buffer for the port
		binary.BigEndian.PutUint16(port[:], uint16(tcpaddr.Port))

		// Check address type
		if host4 != nil { //IPv4

			buf[3] = addrIPv4               // Insert Addres Type
			buf = append(buf, host4...)     // Add IPv6 Address
			buf = append(buf, port[:]...)   // Add Port

		} else if host6 != nil { // IPv6

			buf[3] = addrIPv6               // Insert Addres Type
			buf = append(buf, host6...)     // Add IPv6 Address 
			buf = append(buf, port[:]...)   // Add Port

		} else { // Unknown...

			fmt.Println("SOCKS: neither IPv4 nor IPv6 addr?")
			addr = nil
			buf[1] = byte(repAddressTypeNotSupported)

		}

	} else { // otherwise, attach a null IPv4 address
		buf[3] = addrIPv4
		buf = append(buf, make([]byte, 4+2)...)
	}

	// Return reply message
	return buf
}

/** 
  * Handles reading the data sent through the SOCKS5 connections and preparing them to be sent to the proxy server
  */
func readDataFromSocksProxy(socksConnId uint32, payloadLength int, conn net.Conn, toServer chan []byte, closed chan<- uint32) {

	for {
		// Read up to a cell worth of data to send upstream
		buffer := make([]byte, payloadLength-int(dataWrapHeaderSize))
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



