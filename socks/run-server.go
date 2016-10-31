package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
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

// Launches a SOCKS5 server that listens on port 8081 and forwards
// all connections.
func main() {

	fmt.Println("Launching server...")

	// listen on all interfaces
	ln, _ := net.Listen("tcp", ":8081")

	for {
		// accept connection on port
		conn, _ := ln.Accept()

		fmt.Println("Accepted Client Connection")

		go HandleClient(conn)
	}

}

/**
 * HandleClient is a channel handler assigned for a certain connection ID which handles the packets sent by the client with that ID
 */
func HandleClient(conn net.Conn) {

	// Create a channel reader
	connReader := bufio.NewReader(conn)

	/* SOCKS5 Method Selection Phase */

	// Read SOCKS Version
	socksVersion, err := readMessage(connReader, 1)
	if err != nil {
		// handle error
		fmt.Println("Version Error")
		return
	} else if int(socksVersion[0]) != 5 {
		// handle socks version
		fmt.Println("Version:", int(socksVersion[0]))
		return
	}

	// Read SOCKS Number of Methods
	socksNumOfMethods, err := readMessage(connReader, 1)
	if err != nil {
		//handle error
		return
	}

	// Read SOCKS Methods
	numOfMethods := uint16(socksNumOfMethods[0])
	socksMethods, err := readMessage(connReader, numOfMethods)
	if err != nil {
		//handle error
		return
	}

	// Find a supported method (currently only NoAuth)
	foundMethod := false
	for i := 0; i < len(socksMethods); i++ {
		if socksMethods[i] == methNoAuth {
			foundMethod = true
			break
		}
	}
	if !foundMethod {
		//handle not finding method
		return
	}

	//Construct Response Message
	methodSelectionResponse := []byte{socksVersion[0], byte(methNoAuth)}
	conn.Write(methodSelectionResponse)

	/* SOCKS5 Web Server Request Phase */

	// Read SOCKS Request Header (Version, Command, Address Type)
	requestHeader, err := readMessage(connReader, 4)
	if err != nil {
		//handle error
		fmt.Println("Request Header Error")
		return
	}

	// Read Web Server IP
	destinationIP, err := readSocksAddr(connReader, int(requestHeader[3]))
	if err != nil {
		//handle error
		fmt.Println("IP Address Error")
		return
	}

	// Read Web Server Port
	destinationPortBytes, err := readMessage(connReader, 2)
	if err != nil {
		//handle error
		fmt.Println("Destination Port Error")
		return
	}

	// Process Address and Port
	destinationPort := binary.BigEndian.Uint16(destinationPortBytes)
	destinationAddress := (&net.TCPAddr{IP: destinationIP, Port: int(destinationPort)}).String()

	// Process the command
	switch int(requestHeader[1]) {
	case cmdConnect: // Process "Connect" command

		//Connect to the web server
		fmt.Println("Connecting to Web Server @", destinationAddress)
		webConn, err := net.Dial("tcp", destinationAddress)
		if err != nil {
			fmt.Println("Failed to connect to web server")
			return
		}

		// Send success reply downstream
		sucessMessage := createSocksReply(0, conn.LocalAddr())
		conn.Write(sucessMessage)

		// Commence forwarding raw data on the connection
		go proxyPackets(webConn, conn)
		go proxyPackets(conn, webConn)

	default:
		fmt.Println("Cannot Process Command")
	}

}

func proxyPackets(fromConn net.Conn, toConn net.Conn) {
	for {
		buf := make([]byte, 4096)
		messageLength, _ := fromConn.Read(buf)

		if messageLength == 0 { // connection close indicator
			fmt.Println("Connection Disconnected")
			fromConn.Close()
			toConn.Close()
			return
		}

		n, _ := toConn.Write(buf[:messageLength])
		if n != messageLength {
			fmt.Println("Write Error")
			return
		}
	}
}

/*
readIP reads an IPv4 or IPv6 address from an io.Reader and return it as a string.
 */
func readIP(r io.Reader, len int) (net.IP, error) {
	errorIP := make(net.IP, net.IPv4len)

	addr := make([]byte, len)
	_, err := io.ReadFull(r, addr)
	if err != nil {
		return errorIP, err
	}
	return net.IP(addr), nil
}

/*
readSocksAddr extracts the address content from a SOCKS message.
 */
func readSocksAddr(cr io.Reader, addrtype int) (net.IP, error) {

	errorIP := make(net.IP, net.IPv4len)

	switch addrtype {
	case addrIPv4:
		return readIP(cr, net.IPv4len)

	case addrIPv6:
		return readIP(cr, net.IPv6len)

	case addrDomain:

		// First read the 1-byte domain name length
		dlen := [1]byte{}
		_, err := io.ReadFull(cr, dlen[:])
		if err != nil {
			return errorIP, err
		}

		// Now the domain name itself
		domain := make([]byte, int(dlen[0]))
		_, err = io.ReadFull(cr, domain)
		if err != nil {
			return errorIP, err
		}

		return net.IP(domain), nil

	default:
		msg := fmt.Sprintf("unknown SOCKS address type %d", addrtype)
		fmt.Println(msg)
		return errorIP, errors.New(msg)
	}

}

/*
createSocksReply creates a reply for the SOCKS5 client Request.
 */
func createSocksReply(replyCode int, addr net.Addr) []byte {

	buf := make([]byte, 4)   // Create byte buffer to store reply message
	buf[0] = byte(5)         // Insert Version
	buf[1] = byte(replyCode) // Insert Reply Code

	//Check if address exists
	if addr != nil {

		// Extract Address type
		tcpaddr := addr.(*net.TCPAddr)
		host4 := tcpaddr.IP.To4()
		host6 := tcpaddr.IP.To16()

		//i, _ := strconv.Atoi("6789")

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

func readMessage(connReader io.Reader, length uint16) ([]byte, error) {

	message := make([]byte, length)            // Byte buffer to store the data
	_, err := io.ReadFull(connReader, message) // Read the data
	if err != nil {
		return nil, err
	}

	//Return the content of the data
	return message, nil
}
