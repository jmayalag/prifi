// Launches a SOCKS5 server that listens to PriFi traffic
// and forwards all connections.
package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"flag"
	"strconv"

	"github.com/dedis/cothority/log"
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

// Launches a SOCKS5 server that listens to PriFi traffic
// and forwards all connections.
func main() {

	//manually parse debug flag, since there's only one
	var debugFlag = flag.Int("debug", 3, "debug-level")
	var portFlag = flag.Int("port", 8081, "port")
	flag.Parse()
	log.SetDebugVisible(*debugFlag)

	//check if the port is valid
	if *portFlag <= 1024 {
		log.Lvl1("Port number below 1024. Without super-admin privileges, this server will crash.")
	}
	if *portFlag > 65535 {
		log.Fatal("Port number above 65535. Exiting.")
	}

	//starts the SOCKS exit
	port := ":" + strconv.Itoa(*portFlag)

	log.Lvl2("Starting SOCKS exit...")

	// listen on all interfaces
	ln, _ := net.Listen("tcp", port)

	log.Lvl1("Server listening on port " + port)

	for {
		// accept connection on port
		conn, _ := ln.Accept()

		log.Lvl1("Accepted new socks-client connection")

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
	socksVersion, err := readBytes(connReader, 1)
	if err != nil {
		// handle error
		log.Error("Socks Server : Cannot read version.")
		return
	} else if int(socksVersion[0]) != 5 {
		// handle socks version
		log.Error("Socks Server : Version is ", int(socksVersion[0]), " only 5 is supported.")
		return
	} else {
		log.Lvl2("Socks Server : Version is ", int(socksVersion[0]))
	}

	// Read SOCKS Number of Methods
	socksNumOfMethods, err := readBytes(connReader, 1)
	if err != nil {
		//handle error
		log.Error("Socks Server : Cannot read number of methods.")
		return
	}

	// Read SOCKS Methods
	numOfMethods := uint16(socksNumOfMethods[0])
	socksMethods, err := readBytes(connReader, numOfMethods)
	if err != nil {
		//handle error
		log.Error("Socks Server : Cannot read methods.")
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
		log.Error("Socks Server : No supported method found.")
		return
	}

	//Construct Response Message
	methodSelectionResponse := []byte{socksVersion[0], byte(methNoAuth)}
	log.Lvl2("Socks Server : Writing negotiation response...")
	conn.Write(methodSelectionResponse)

	/* SOCKS5 Web Server Request Phase */

	// Read SOCKS Request Header (Version, Command, Address Type)
	requestHeader, err := readBytes(connReader, 4)
	if err != nil {
		//handle error
		log.Error("Socks Server : cannot read header.")
		return
	}

	// Read Web Server IP
	destinationIP, err := readSocksAddr(connReader, int(requestHeader[3]))
	if err != nil {
		//handle error
		log.Error("Socks Server : cannot read IP.")
		return
	}

	// Read Web Server Port
	destinationPortBytes, err := readBytes(connReader, 2)
	if err != nil {
		//handle error
		log.Error("Socks Server : cannot read Port.")
		return
	}

	// Process Address and Port
	destinationPort := binary.BigEndian.Uint16(destinationPortBytes)
	destinationAddress := (&net.TCPAddr{IP: destinationIP, Port: int(destinationPort)}).String()

	// Process the command
	switch int(requestHeader[1]) {
	case cmdConnect: // Process "Connect" command

		//Connect to the web server
		log.Lvl2("Socks Server : contacting server ", destinationAddress)
		webConn, err := net.Dial("tcp", destinationAddress)
		if err != nil {
			log.Error("Socks Server : error contacting server ", destinationAddress)
			return
		}

		// Send success reply downstream
		sucessMessage := createSocksReply(0, conn.LocalAddr())
		conn.Write(sucessMessage)

		// Commence forwarding raw data on the connection
		go proxyPackets(webConn, conn)
		go proxyPackets(conn, webConn)

	default:
		log.Error("Socks Server : unknown command", int(requestHeader[1]))
	}

}

func proxyPackets(fromConn net.Conn, toConn net.Conn) {
	for {
		buf := make([]byte, 4096)
		messageLength, _ := fromConn.Read(buf)

		if messageLength == 0 { // connection close indicator
			log.Lvl3("Connection ended.")
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

func readBytes(connReader io.Reader, length uint16) ([]byte, error) {

	message := make([]byte, length)            // Byte buffer to store the data
	_, err := io.ReadFull(connReader, message) // Read the data
	if err != nil {
		return nil, err
	}

	//Return the content of the data
	return message, nil
}
