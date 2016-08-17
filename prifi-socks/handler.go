package prifi_socks

import (
	"net"
	"fmt"
  	"encoding/binary"
	"crypto/rand"

	"github.com/dedis/crypto/abstract"
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
  * Connects the relay to the proxy server and proxies messages between the relay and the server
  */

func ConnectToServer(IP string, toServer chan []byte, fromServer chan []byte) {
	
    allConnections := make( map[uint32] net.Conn )		// Stores all live connections
	controlChannels := make( map[uint32] chan uint16 )	// Stores the control channels for each live connection
    currentState := uint16( ResumeCommunication )		// Keeps track of current state (stalled or resumed)

	
	for {
		// Block on receiving a packet
		data := <-toServer

		// Extract the data from the packet
		myPacket := ExtractFull(data)
		packetType := myPacket.Type
		connID := myPacket.ID 
    	clientPacket :=  myPacket.Data[:myPacket.MessageLength]

    	// Retreive the requested connection if possible
    	myConn := allConnections[connID]

    	// Check the type of the packet
    	switch packetType {
    		case SocksConnect: 	// Indicates a new connection established (this is important to identify if a connection ID overlap has occured)

    			// If no channel exists yet, create one and setup a channel handler (this means no connection ID overlap occured)
		      	if myConn == nil {

		        	// Create a new connection with the SOCKS server
					newConn, _ := net.Dial("tcp", IP)
			        allConnections[connID] = newConn

			        // Create a control channel for this connection
			        controlChannels[connID] = make(chan uint16, 1)

		        	// Instantiate a connection handler and pass it the current state
					controlChannels[connID] <- currentState
		        	go handleConnection(newConn, connID, 4000, controlChannels[connID], fromServer, toServer)

		        	// Send the received packet to the SOCKS server 
					newConn.Write(clientPacket)

		        } else { // Otherwise a connection ID overlap has occured, reject the connection

		        	// Create Socks Error message and send it back to the client
					newData := NewDataWrap(SocksError, myPacket.ID, 0, 0, []byte{})
					fromServer <- newData.ToBytes()
					continue

		        }

    		case StallCommunication:	// Indicates that all connection handlers should be stalled from sending data back to the relay

    			// Send the control message to all existing control channels
    			for i := range controlChannels {
    				controlChannels[i] <- StallCommunication
    			}

    			// Change the Current State to stalled
				currentState = StallCommunication

    		case ResumeCommunication:	// Indicates that all connection handlers should resume sending data back to the relay
    			    			
    			// Send the control message to all existing control channels
				for i := range controlChannels {
    				controlChannels[i] <- ResumeCommunication
    			} 

				// Change the Current State to resumed
				currentState = ResumeCommunication

    		case SocksData:		// Indicates this is a normal packet that needs to be proxied to the SOCKS server

    			// Check if the connection exist, and send the packet to the SOCKS server
    			if myConn != nil {
    				myConn.Write(clientPacket)
    			}

			case SocksClosed:	// Indicates a connection has been closed

				// Delete the connection and it's corresponding control channel 
				delete(allConnections,connID)
				delete(controlChannels,connID)

			default:
				break
  
    	}      	


	}

}


/** 
  * Handles reading data from a connection with a SOCKS entity (Browser, SOCKS Server) and forwarding it to a PriFi entity (Client, Relay)
  */

func handleConnection( conn net.Conn, connID uint32, payloadLength int, control chan uint16, sendData chan []byte, closedChan chan []byte ) {

	dataChannel := make(chan []byte, 1)	// Channel to communicate the data read from the connection with the SOCKS entity
	var dataBuffer [][]byte 			// Buffer to store data to be sent later
	sendingOK := true 					// Indicates if forwarding data to the PriFi entity is permitted
	messageType := uint16(SocksConnect) // This variable is used to ensure that the first packet sent back to the PriFi entity is a SocksConnect packet (It is only useful for the client-side handler)

	// Create connection reader to read from the connection with the SOCKS entity
	go connectionReader(conn, payloadLength-int(DataWrapHeaderSize), dataChannel)

	for {
		
		// Block on either receiving a control message or data
		select {

	    	case controlMessage := <- control: // Read control message

	    		// Check the type of control (Stall, Resume)
	    		switch controlMessage {

	    			case StallCommunication:
		    			sendingOK = false 	// Indicate that forwarding to the PriFi entity is not permitted

	    			case ResumeCommunication:
		    			sendingOK = true	// Indicate that forwarding to the PriFi entity is permitted

		    			// For all buffered data, send it to the PriFi entity
		    			for i := 0; i < len(dataBuffer); i++ {
		    				// Create data packate
	    					newData := NewDataWrap(messageType, connID, uint16(len(dataBuffer[i])), uint16(payloadLength), dataBuffer[i])
	    					if messageType == SocksConnect {
					    		messageType = SocksData
					    	}

					    	// Send the data to the PriFi entity
		    				sendData <- newData.ToBytes()
					
							// Connection Close Indicator
							if newData.MessageLength == 0 { 
								conn.Close()	// Close the connection

								// Create a connection closed message and send it back
								closeConn := NewDataWrap(SocksClosed, connID, 0, 0, []byte{})
								closedChan <- closeConn.ToBytes()
								
								return
							}
		    			}

		    			// Empty the data buffer
		    			dataBuffer = nil

	    			default:
	    				break
	    		}

	    	case data := <- dataChannel: // Read data from SOCKS entity

	    		// Check if forwarding to the PriFi entity is permitted
	    		if sendingOK {

	    			// Create the packet from the data
	    			newData := NewDataWrap(messageType, connID, uint16(len(data)), uint16(payloadLength), data)
	    			if messageType == SocksConnect {
	    				messageType = SocksData
	    			}

	    			// Send the data to the PriFi entity
					sendData <- newData.ToBytes()

					// Connection Close Indicator
					if newData.MessageLength == 0 {
						fmt.Println("Connection", connID, "Closed")
						conn.Close()	// Close the connection

						// Create a connection closed message and send it back
						closeConn := NewDataWrap(SocksClosed, connID, 0, 0, []byte{})
						closedChan <- closeConn.ToBytes()

						return
					}

		    	} else { 	// Otherwise if sending is not permitted

		    		// Store the data in a buffer
		    		dataBuffer = append(dataBuffer, data)

		    	}

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
  * Handles SOCKS5 connections with the browser
  */
func StartSocksProxyServerHandler(newConnections chan net.Conn, payloadLength int, key abstract.Secret, toServer chan []byte, fromServer chan []byte) {

	socksProxyActiveConnections := make( map[uint32] net.Conn ) // Stores all live connections (Reserve socksProxyActiveConnections[0])
	controlChannels := make( map[uint32] chan uint16 )			// Stores the control channels for each live connection
    counter := make( map[uint32] int )							// Counts the number of messages received for a certain channel (this helps us identify the 2nd message that contains IP & Port info)
    currentState := uint16(ResumeCommunication)					// Keeps track of current state (stalled or resumed)
	
	for {

		select {

		// New TCP connection to the SOCKS proxy received
		case conn := <-newConnections:

			// Generate Socks connection ID
			newSocksProxyId := generateID(key)
			for socksProxyActiveConnections[newSocksProxyId] != nil { // If local conflicts occur, keep trying untill we find a locally unique ID
				newSocksProxyId = generateID(key)
			}
			socksProxyActiveConnections[newSocksProxyId] = conn
			controlChannels[newSocksProxyId] = make(chan uint16, 1)
			counter[newSocksProxyId] = 1

		    // Instantiate a connection handler and pass it the current state
			controlChannels[newSocksProxyId] <- currentState
			go handleConnection( conn, newSocksProxyId, payloadLength, controlChannels[newSocksProxyId], toServer, fromServer )
			fmt.Println("Connection", newSocksProxyId, "Established")

		// Plaintext downstream data (relay->client->Socks proxy)
		case bufferData := <-fromServer:
		
			// Extract the data from the packet
			myData := ExtractFull(bufferData)
			socksConnId := myData.ID
			packetType := myData.Type
			dataLength := myData.MessageLength
			data := myData.Data[:dataLength]

			// Skip if the connection doesn't exist or the connection ID is 0, unless it's a control message (stall, resume)
			if packetType == StallCommunication || packetType == ResumeCommunication {
				//fmt.Println("Stall/Resume Control Received")
			} else if socksConnId == 0 || socksProxyActiveConnections[socksConnId] == nil {
				continue
			} 
				
			// Check the type of message
			switch packetType {
				case SocksError: // Indicates SOCKS connection error (usually happens if the connection ID being used is not globally unique)
    					fmt.Println("DUPLICATE ID",socksConnId)

    					// Close the connection and Delete the connection and it's corresponding control channel
						socksProxyActiveConnections[socksConnId].Close()
						delete(socksProxyActiveConnections,socksConnId)
						delete(controlChannels,socksConnId)

	    		case StallCommunication:	// Indicates that all connection handlers should be stalled from sending data back to the relay

	    			// Send the control message to all existing control channels
	    			for i := range controlChannels {
	    				controlChannels[i] <- StallCommunication
	    			}

	    			// Change the Current State to stalled
					currentState = StallCommunication

    			case ResumeCommunication:	// Indicates that all connection handlers should resume sending data back to the relay
    			    			
	    			// Send the control message to all existing control channels
					for i := range controlChannels {
	    				controlChannels[i] <- ResumeCommunication
	    			} 

					// Change the Current State to resumed
					currentState = ResumeCommunication

	    		case SocksData, SocksConnect:	// Indicates receiving SOCKS data

	    			// If it's the second messages
	    			if counter[socksConnId] == 2 {
	    				// Replace the IP & Port fields set to the servers IP & PORT to the clients IP & PORT
						data = replaceData(data, socksProxyActiveConnections[socksConnId].LocalAddr())
					}

					// Write the data back to the browser
					socksProxyActiveConnections[socksConnId].Write(data)
					counter[socksConnId]++

				case SocksClosed:	// Indicates a connection has been closed

					// Delete the connection and it's corresponding control channel 
					delete(socksProxyActiveConnections,socksConnId)
					delete(controlChannels,socksConnId)

				default:
					break
			}
		}
	}
}


/**
	Reads data from a connection and forwards it into a data channel, maximum data read must be specified
*/

func connectionReader(conn net.Conn, readLength int, dataChannel chan []byte) {
	for {
		// Read data from the connection
		buffer := make([]byte, readLength)
		n, _ := conn.Read(buffer)

		// Trim the data and send it through the data channel
		dataChannel <- buffer[:n]

		// Connection Closed Indicator
		if n == 0 { 
			return
		}
	}
}


/**
	Replaces the IP & PORT data in the SOCKS5 connect server reply
*/

func replaceData( buf []byte, addr net.Addr ) []byte {
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
	Generates a SOCK connection ID
*/

func generateID(key abstract.Secret) uint32 {
	var n uint32
    binary.Read(rand.Reader, binary.LittleEndian, &n)

    return n
}

