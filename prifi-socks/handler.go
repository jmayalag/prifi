package prifisocks

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/dedis/cothority/log"
	"strconv"
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

/*
 * This is a socks CLIENT. It is run by the PriFi's relay to connect to some SOCKS server. dataForServer contains the
 * relay's output, i.e. the anonymized traffic. dataFromServer should be the downstream traffic for the PriFi clients.
 */
func StartSocksClient(serverAddress string, upstreamChan chan []byte, downstreamChan chan []byte) {

	socksConnections := make(map[uint32]net.Conn)      // Stores all live connections
	controlChannels := make(map[uint32]chan uint16)    // Stores the control channels for each live connection
	currentSendingState := uint16(ResumeCommunication) // Keeps track of current state (stalled or resumed)

	for {
		// Block on receiving a packet
		data := <-upstreamChan

		// Extract the data from the packet
		packet := ParseSocksPacketFromBytes(data)
		socksConnectionId := uint32(packet.Id)
		packetPayload := packet.Data[:packet.MessageLength]

		// Retrieve the requested connection if possible
		socksConnection := socksConnections[socksConnectionId]

		// Check the type of the packet
		switch packet.Type {
		case SocksConnect: // Indicates a new connection established (this is important to identify if a connection ID overlap has occured)

			// If no channel exists yet, create one and setup a channel handler (this means no connection ID overlap occured)
			if socksConnection == nil {

				// Create a new connection with the SOCKS server
				newConn, err := net.Dial("tcp", serverAddress)
				if err != nil {
					log.Error("SOCKS PriFi Client: Could not connect to SOCKS server.", err)
				} else {
					socksConnections[socksConnectionId] = newConn

					// Create a control channel for this connection
					controlChannels[socksConnectionId] = make(chan uint16, 1)

					// Instantiate a connection handler and pass it the current state
					controlChannels[socksConnectionId] <- currentSendingState
					go handleSocksClientConnection(newConn, socksConnectionId, 4000, controlChannels[socksConnectionId], downstreamChan, upstreamChan)

					// Send the received packet to the SOCKS server
					newConn.Write(packetPayload)
				}

			} else { // Otherwise a connection ID overlap has occured, reject the connection

				// Create Socks Error message and send it back to the client
				newData := NewSocksPacket(SocksError, packet.Id, 0, 0, []byte{})
				downstreamChan <- newData.ToBytes()

				log.Error("SOCKS PriFi Client: Received a SocksConnect message with ID", packet.Id, "but this connection already exists. Rejecting.")

				continue

			}

		case StallCommunication: // Indicates that all connection handlers should be stalled from sending data back to the relay

			// Send the control message to all existing control channels
			for i := range controlChannels {
				controlChannels[i] <- StallCommunication
			}

			// Change the Current State to stalled
			currentSendingState = StallCommunication

			log.Lvl2("SOCKS PriFi Client: All communications stalled.")

		case ResumeCommunication: // Indicates that all connection handlers should resume sending data back to the relay

			// Send the control message to all existing control channels
			for i := range controlChannels {
				controlChannels[i] <- ResumeCommunication
			}

			// Change the Current State to resumed
			currentSendingState = ResumeCommunication

			log.Lvl2("SOCKS PriFi Client: All communications resumed.")

		case SocksData: // Indicates this is a normal packet that needs to be proxied to the SOCKS server

			log.Lvl2("SOCKS PriFi Client: Got a SOCKS data message.")

			// Check if the connection exist, and send the packet to the SOCKS server
			if socksConnections[socksConnectionId] != nil {
				socksConnections[socksConnectionId].Write(packetPayload)
			} else {
				//log.Error("SOCKS PriFi Client: Got data for an unexisting connection "+strconv.Itoa(int(socksConnectionId))+", dropping.")
			}

		case SocksClosed: // Indicates a connection has been closed

			// Delete the connection and it's corresponding control channel
			delete(socksConnections, socksConnectionId)
			delete(controlChannels, socksConnectionId)

			log.Lvl2("SOCKS PriFi Client: Freeing ressources for connection " + strconv.Itoa(int(socksConnectionId)) + ".")

		default:
			log.Error("SOCKS PriFi Client: Unrecognized message type.")
			break

		}

	}

}

/*
 * This is started by the PriFi clients, so the app on their computer can connect to this "fake" Socks server (fake in the sense that the
 * output is actually forwarded to prifi and anonymized).
 */
func StartSocksServer(localListeningAddress string, payloadLength int, upstreamChan chan []byte, downstreamChan chan []byte, downStreamContainsLatencyMessages bool) {

	// Setup a thread to listen at the assigned port
	socksConnections := make(chan net.Conn, 1)
	go acceptSocksClients(localListeningAddress, socksConnections)

	socksProxyActiveConnections := make(map[uint32]net.Conn) // Stores all live connections (Reserve socksProxyActiveConnections[0])
	controlChannels := make(map[uint32]chan uint16)          // Stores the control channels for each live connection
	counter := make(map[uint32]int)                          // Counts the number of messages received for a certain channel (this helps us identify the 2nd message that contains IP & Port info)
	currentState := uint16(ResumeCommunication)              // Keeps track of current state (stalled or resumed)

	for {

		select {

		// New TCP connection to the SOCKS proxy received
		case conn := <-socksConnections:

			// Generate Socks connection ID
			newSocksProxyId := generateRandomUniqueId(socksProxyActiveConnections)
			socksProxyActiveConnections[newSocksProxyId] = conn
			controlChannels[newSocksProxyId] = make(chan uint16, 1)
			counter[newSocksProxyId] = 1

			log.Lvl2("SOCKS PriFi Server : got a new connection, assigned id " + strconv.Itoa(int(newSocksProxyId)))

			// Instantiate a connection handler and pass it the current state
			controlChannels[newSocksProxyId] <- currentState
			go handleSocksClientConnection(conn, newSocksProxyId, payloadLength, controlChannels[newSocksProxyId], upstreamChan, downstreamChan)

		// Plaintext downstream data (relay->client->Socks proxy)
		case bufferData := <-downstreamChan:

			// Extract the data from the packet
			myData := ParseSocksPacketFromBytes(bufferData)
			socksConnId := myData.Id
			packetType := myData.Type
			dataLength := myData.MessageLength
			data := myData.Data[:dataLength]

			// Skip if the connection doesn't exist or the connection ID is 0, unless it's a control message (stall, resume)
			if !downStreamContainsLatencyMessages { //of course, if the downstream contains latency messages, don't show error
				if (packetType != StallCommunication && packetType != ResumeCommunication) && (socksConnId == 0 || socksProxyActiveConnections[socksConnId] == nil) {
					log.Error("SOCKS PriFi Server: Got data for an unexisting connection " + strconv.Itoa(int(socksConnId)) + ", dropping.")
					continue
				}
			}

			// Check the type of message
			switch packetType {
			case SocksError: // Indicates SOCKS connection error (usually happens if the connection ID being used is not globally unique)
				log.Error("SOCKS PriFi Server: Got an error for connection " + strconv.Itoa(int(socksConnId)) + ", closing.")

				// Close the connection and Delete the connection and it's corresponding control channel
				socksProxyActiveConnections[socksConnId].Close()
				delete(socksProxyActiveConnections, socksConnId)
				delete(controlChannels, socksConnId)

			case StallCommunication: // Indicates that all connection handlers should be stalled from sending data back to the relay

				// Send the control message to all existing control channels
				for i := range controlChannels {
					controlChannels[i] <- StallCommunication
				}

				// Change the Current State to stalled
				currentState = StallCommunication

				log.Lvl2("SOCKS PriFi Server: All communications stalled.")

			case ResumeCommunication: // Indicates that all connection handlers should resume sending data back to the relay

				// Send the control message to all existing control channels
				for i := range controlChannels {
					controlChannels[i] <- ResumeCommunication
				}

				// Change the Current State to resumed
				currentState = ResumeCommunication

				log.Lvl2("SOCKS PriFi Server: All communications resumed.")

			case SocksData, SocksConnect: // Indicates receiving SOCKS data

				// If it's the second messages
				if counter[socksConnId] == 2 {
					// Replace the IP & Port fields set to the servers IP & PORT to the clients IP & PORT
					data = ipMaskerade(data, socksProxyActiveConnections[socksConnId].LocalAddr())
				}

				// Write the data back to the browser
				socksProxyActiveConnections[socksConnId].Write(data)
				counter[socksConnId]++

			case SocksClosed: // Indicates a connection has been closed

				// Delete the connection and it's corresponding control channel
				delete(socksProxyActiveConnections, socksConnId)
				delete(controlChannels, socksConnId)

				log.Error("SOCKS PriFi Server: Connection " + strconv.Itoa(int(socksConnId)) + " closed.")

			default:
				break
			}
		}
	}
}

/*
handleConnection handles reading data from a connection with a SOCKS entity (Browser, SOCKS Server) and forwarding it to a PriFi entity (Client, Relay).
*/
func handleSocksClientConnection(tcpConnection net.Conn, connectionId uint32, socksPacketLength int, controlChannel chan uint16, upstreamChan chan []byte, downstreamChan chan []byte) {

	log.Lvl2("SOCKS started to handle connection " + strconv.Itoa(int(connectionId)))

	dataFromSocksClientToSocksServer := make(chan []byte, 1) // Channel to communicate the data read from the connection with the SOCKS entity
	var dataBuffer [][]byte                                  // Buffer to store data to be sent later
	sendingOK := true                                        // Indicates if forwarding data to the PriFi entity is permitted
	messageType := uint16(SocksConnect)                      // This variable is used to ensure that the first packet sent back to the PriFi entity is a SocksConnect packet (It is only useful for the client-side handler)

	// Create connection reader to read from the connection with the SOCKS entity
	go connectionReader(tcpConnection, socksPacketLength-int(SocksPacketHeaderSize), dataFromSocksClientToSocksServer)

	for {

		// Block on either receiving a control message or data
		select {

		case controlMessage := <-controlChannel: // Read control message

			// Check the type of control (Stall, Resume)
			switch controlMessage {

			case StallCommunication:
				log.Lvl2("SOCKS handler for connection " + strconv.Itoa(int(connectionId)) + " just got a Stall message.")
				sendingOK = false // Indicate that forwarding to the PriFi entity is not permitted

			case ResumeCommunication:
				log.Lvl2("SOCKS handler for connection " + strconv.Itoa(int(connectionId)) + " just got a Resume message.")
				sendingOK = true // Indicate that forwarding to the PriFi entity is permitted

				// For all buffered data, send it to the PriFi entity
				for i := 0; i < len(dataBuffer); i++ {
					// Create data packet
					newData := NewSocksPacket(messageType, connectionId, uint16(len(dataBuffer[i])), uint16(socksPacketLength), dataBuffer[i])
					if messageType == SocksConnect {
						messageType = SocksData
					}

					// Send the data to the PriFi entity
					upstreamChan <- newData.ToBytes()

					// Connection Close Indicator
					if newData.MessageLength == 0 {
						tcpConnection.Close() // Close the connection

						// Create a connection closed message and send it back
						closeConn := NewSocksPacket(SocksClosed, connectionId, 0, 0, []byte{})
						downstreamChan <- closeConn.ToBytes()

						return
					}
				}

				// Empty the data buffer
				dataBuffer = nil

			default:
				break
			}

		case data := <-dataFromSocksClientToSocksServer: // Read data from SOCKS entity

			// Check if forwarding to the PriFi entity is permitted
			if sendingOK {

				// Create the packet from the data
				newData := NewSocksPacket(messageType, connectionId, uint16(len(data)), uint16(socksPacketLength), data)
				if messageType == SocksConnect {
					messageType = SocksData
				}

				// Send the data to the PriFi entity
				upstreamChan <- newData.ToBytes()

				// Connection Close Indicator
				if newData.MessageLength == 0 {
					log.Lvl2("Connection", connectionId, "closed")
					tcpConnection.Close() // Close the connection

					// Create a connection closed message and send it back
					closeConn := NewSocksPacket(SocksClosed, connectionId, 0, 0, []byte{})
					downstreamChan <- closeConn.ToBytes()

					return
				}

			} else { // Otherwise if sending is not permitted

				// Store the data in a buffer
				dataBuffer = append(dataBuffer, data)

			}

		}

	}
}

/*
browserConnectionListener listens and accepts connections at a certain port
*/
func acceptSocksClients(port string, newConnections chan<- net.Conn) {

	log.Lvl2("SOCKS listener is listening for connections on port " + port)

	lsock, err := net.Listen("tcp", port)

	if err != nil {
		log.Error("SOCKS listener cannot start listening, shutting down :", err.Error())
		return
	}

	for {
		conn, err := lsock.Accept()

		log.Lvl2("SOCKS listener just accepted a connection.")

		if err != nil {
			log.Error("SOCKS listener got an error with this new connection, shutting down :", err.Error())
			lsock.Close()
			return
		}

		newConnections <- conn
	}
}

/**
connectionReader reads data from a connection and forwards it into a data channel, maximum data read must be specified.
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

/*
replaceData replaces the IP & PORT data in the SOCKS5 connect server reply
*/
func ipMaskerade(buf []byte, addr net.Addr) []byte {
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

			buf[3] = addrIPv4             // Insert Addres Type
			buf = append(buf, host4...)   // Add IPv6 Address
			buf = append(buf, port[:]...) // Add Port

		} else if host6 != nil { // IPv6

			buf[3] = addrIPv6             // Insert Addres Type
			buf = append(buf, host6...)   // Add IPv6 Address
			buf = append(buf, port[:]...) // Add Port

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

/*
generateUniqueID generates a unique SOCK connection ID from a private key
*/
func generateRandomUniqueId(connections map[uint32]net.Conn) uint32 {

	id := generateRandomId()
	for connections[id] != nil { // If local conflicts occur, keep trying until we find a locally unique ID
		id = generateRandomId()
	}

	return id
}

/*
generateID generates an ID from a private key
*/
func generateRandomId() uint32 {
	var n uint32
	binary.Read(rand.Reader, binary.LittleEndian, &n)

	return n
}

/*
StallTester sends a stall message after "timeBeforeStall" seconds, and a resume message after "stallFor" seconds.
*/
func StallTester(timeBeforeStall time.Duration, stallFor time.Duration, myChannel chan []byte, payload int) {

	time.Sleep(timeBeforeStall)
	log.Lvl2("SOCKS : Sending stall message...")
	stallConn := NewSocksPacket(StallCommunication, 0, 0, uint16(payload), []byte{})
	myChannel <- stallConn.ToBytes()

	time.Sleep(stallFor)
	log.Lvl2("SOCKS : Sending resume message...")
	resumeConn := NewSocksPacket(ResumeCommunication, 0, 0, uint16(payload), []byte{})
	myChannel <- resumeConn.ToBytes()
}
