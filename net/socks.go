package net

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
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


func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

const downcellmax = 16 * 1024 // downstream cell max size
var errAddressTypeNotSupported = errors.New("SOCKS5 address type not supported")

type chanreader struct {
	b   []byte
	c   <-chan []byte
	eof bool
}

func (cr *chanreader) Read(p []byte) (n int, err error) {
	if cr.eof {
		return 0, io.EOF
	}
	blen := len(cr.b)
	if blen == 0 {
		cr.b = <-cr.c // read next block from channel
		blen = len(cr.b)
		if blen == 0 { // channel sender signaled EOF
			cr.eof = true
			return 0, io.EOF
		}
	}

	act := min(blen, len(p))
	copy(p, cr.b[:act])
	cr.b = cr.b[act:]
	return act, nil
}

func newChanReader(c <-chan []byte) *chanreader {
	return &chanreader{[]byte{}, c, false}
}


// Read an IPv4 or IPv6 address from an io.Reader and return it as a string
func readIP(r io.Reader, len int) (string, error) {
	addr := make([]byte, len)
	_, err := io.ReadFull(r, addr)
	if err != nil {
		return "", err
	}
	return net.IP(addr).String(), nil
}

func readSocksAddr(cr io.Reader, addrtype int) (string, error) {
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
			return "", err
		}

		// Now the domain name itself
		domain := make([]byte, int(dlen[0]))
		_, err = io.ReadFull(cr, domain)
		if err != nil {
			return "", err
		}
		log.Printf("SOCKS: domain '%s'\n", string(domain))

		return string(domain), nil

	default:
		msg := fmt.Sprintf("unknown SOCKS address type %d", addrtype)
		return "", errors.New(msg)
	}
}

func socksRelayDown(cno int, conn net.Conn, downstream chan<- DataWithConnectionId) {
	//log.Printf("socksRelayDown: cno %d\n", cno)
	for {
		buf := make([]byte, downcellmax)
		n, err := conn.Read(buf)
		buf = buf[:n]
		//fmt.Printf("socksRelayDown: %d bytes on cno %d\n", n, cno)
		//fmt.Print(hex.Dump(buf[:n]))

		// Forward the data (or close indication if n==0) downstream
		downstream <- DataWithConnectionId{cno, buf}

		// Connection error or EOF?
		if n == 0 {
			log.Println("socksRelayDown: " + err.Error())
			conn.Close()
			return
		}
	}
}

func socksRelayUp(cno int, conn net.Conn, upstream <-chan []byte) {
	//log.Printf("socksRelayUp: cno %d\n", cno)
	for {
		// Get the next upstream data buffer
		buf := <-upstream
		dlen := len(buf)
		//fmt.Printf("socksRelayUp: %d bytes on cno %d\n", len(buf), cno)
		//fmt.Print(hex.Dump(buf))

		if dlen == 0 { // connection close indicator
			log.Printf("socksRelayUp: closing stream %d\n", cno)
			conn.Close()
			return
		}
		//println(hex.Dump(buf))
		n, err := conn.Write(buf)
		if n != dlen {
			log.Printf("socksRelayUp: " + err.Error())
			conn.Close()
			return
		}
	}
}

func socks5Reply(cno int, err error, addr net.Addr) DataWithConnectionId {

	buf := make([]byte, 4)
	buf[0] = byte(5) // version

	// buf[1]: Reply field
	switch err {
	case nil: // succeeded
		buf[1] = repSucceeded
	// XXX recognize some specific errors
	default:
		buf[1] = repGeneralFailure
	}

	// Address type
	if addr != nil {
		tcpaddr := addr.(*net.TCPAddr)
		host4 := tcpaddr.IP.To4()
		host6 := tcpaddr.IP.To16()
		port := [2]byte{}
		binary.BigEndian.PutUint16(port[:], uint16(tcpaddr.Port))
		if host4 != nil { // it's an IPv4 address
			buf[3] = addrIPv4
			buf = append(buf, host4...)
			buf = append(buf, port[:]...)
		} else if host6 != nil { // it's an IPv6 address
			buf[3] = addrIPv6
			buf = append(buf, host6...)
			buf = append(buf, port[:]...)
		} else { // huh???
			log.Printf("SOCKS: neither IPv4 nor IPv6 addr?")
			addr = nil
			err = errAddressTypeNotSupported
		}
	}
	if addr == nil { // attach a null IPv4 address
		buf[3] = addrIPv4
		buf = append(buf, make([]byte, 4+2)...)
	}

	// Reply code
	var rep int
	switch err {
	case nil:
		rep = repSucceeded
	case errAddressTypeNotSupported:
		rep = repAddressTypeNotSupported
	default:
		rep = repGeneralFailure
	}
	buf[1] = byte(rep)

	//log.Printf("SOCKS5 reply:\n" + hex.Dump(buf))
	return DataWithConnectionId{cno, buf}
}



// Main loop of our relay-side SOCKS proxy.
func RelaySocksProxy(connId int, upstream <-chan []byte, downstream chan<- DataWithConnectionId) {

	// Send downstream close indication when we bail for whatever reason
	defer func() {
		downstream <- DataWithConnectionId{connId, []byte{}}
	}()

	// Put a convenient I/O wrapper around the raw upstream channel
	cr := newChanReader(upstream)

	// Read the SOCKS client's version/methods header
	vernmeth := [2]byte{}
	_, err := io.ReadFull(cr, vernmeth[:])
	if err != nil {
		log.Printf("SOCKS: no version/method header: " + err.Error())
		return
	}
	//log.Printf("SOCKS proxy: version %d nmethods %d \n",
	//	vernmeth[0], vernmeth[1])
	ver := int(vernmeth[0])
	if ver != 5 {
		log.Printf("SOCKS: unsupported version number %d", ver)

		if ver == 71 {
			log.Printf("Tips : 71 is for HTTP, but this is a SOCKS proxy, not an HTTP proxy !", ver)
		}
		return
	}
	nmeth := int(vernmeth[1])
	methods := make([]byte, nmeth)
	_, err = io.ReadFull(cr, methods)
	if err != nil {
		log.Printf("SOCKS: short version/method header: " + err.Error())
		return
	}

	// Find a supported method (currently only NoAuth)
	for i := 0; ; i++ {
		if i >= len(methods) {
			log.Printf("SOCKS: no supported method")
			resp := [2]byte{byte(ver), byte(methNone)}
			downstream <- DataWithConnectionId{connId, resp[:]}
			return
		}
		if methods[i] == methNoAuth {
			break
		}
	}

	// Reply with the chosen method
	methresp := [2]byte{byte(ver), byte(methNoAuth)}
	downstream <- DataWithConnectionId{connId, methresp[:]}

	// Receive client request
	req := make([]byte, 4)
	_, err = io.ReadFull(cr, req)
	if err != nil {
		log.Printf("SOCKS: missing client request: " + err.Error())
		return
	}
	if req[0] != byte(ver) {
		log.Printf("SOCKS: client changed versions")
		return
	}
	host, err := readSocksAddr(cr, int(req[3]))
	if err != nil {
		log.Printf("SOCKS: invalid destination address: " + err.Error())
		return
	}
	portb := [2]byte{}
	_, err = io.ReadFull(cr, portb[:])
	if err != nil {
		log.Printf("SOCKS: invalid destination port: " + err.Error())
		return
	}
	port := binary.BigEndian.Uint16(portb[:])
	hostport := fmt.Sprintf("%s:%d", host, port)

	// Process the command
	cmd := int(req[1])
	//log.Printf("SOCKS proxy: request %d for %s\n", cmd, hostport)
	switch cmd {
	case cmdConnect:
		conn, err := net.Dial("tcp", hostport)
		if err != nil {
			log.Printf("SOCKS: error connecting to destionation: " +
				err.Error())
			downstream <- socks5Reply(connId, err, nil)
			return
		}

		// Send success reply downstream
		downstream <- socks5Reply(connId, nil, conn.LocalAddr())

		// Commence forwarding raw data on the connection
		go socksRelayDown(connId, conn, downstream)
		socksRelayUp(connId, conn, upstream)

	default:
		log.Printf("SOCKS: unsupported command %d", cmd)
	}
}