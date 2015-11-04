package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"strconv"
	"os"
	"os/signal"
	"github.com/lbarman/crypto/nist"
	"github.com/lbarman/prifi/dcnet"

	log2 "github.com/lbarman/prifi/log"
)

var suite = nist.NewAES128SHA256P256() // XXX should only have defaultSuite
//var suite = openssl.NewAES128SHA256P256()
//var suite = ed25519.NewAES128SHA256Ed25519()
var factory = dcnet.SimpleCoderFactory

var defaultSuite = suite

const LLD_PROTOCOL_VERSION = 1

const nClients = 2
const nTrustees = 3

const relayhost = "localhost:9876" // XXX
const bindport = ":9876"

//const payloadlen = 1200			// upstream cell size
var payloadlen = 7680 // upstream cell size

const downcellmax = 16 * 1024 // downstream cell max size

// Number of bytes of cell payload to reserve for connection header, length
const proxyhdrlen = 6

type connbuf struct {
	cno int    // connection number
	buf []byte // data buffer
}

type dataWithConnectionId struct {
	connectionId 	int    // connection number
	data 			[]byte // data buffer
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

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

var errAddressTypeNotSupported = errors.New("SOCKS5 address type not supported")

/*
 * MAIN
 */

func interceptCtrlC() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			panic("signal: " + sig.String()) // with stacktrace
		}
	}()
}

func main() {
	interceptCtrlC()

	log2.StringDump("New run...")

	//roles...
	isRelay           := flag.Bool("relay", false, "Start relay node")
	clientId          := flag.Int("client", -1, "Start client node")
	useSocksProxy     := flag.Bool("socks", true, "Starts a useSocksProxy proxy for the client")
	isTrusteeServer   := flag.Bool("trusteesrv", false, "Start a trustee server")

	//parameters config
	nClients          := flag.Int("nClients", nClients, "The number of clients.")
	nTrustees         := flag.Int("nTrustees", nTrustees, "The number of trustees.")
	cellSize          := flag.Int("cellSize", -1, "Sets the size of one cell, in bytes.")
	relayPort         := flag.Int("relayPort", 9876, "Sets listening port of the relay, waiting for clients.")
	relayReceiveLimit := flag.Int("reportlimit", -1, "Sets the limit of cells to receive before stopping the relay")
	trustee1Host      := flag.String("t1host", "localhost", "The Ip address of the 1st trustee, or localhost")
	trustee2Host      := flag.String("t2host", "localhost", "The Ip address of the 2nd trustee, or localhost")
	trustee3Host      := flag.String("t3host", "localhost", "The Ip address of the 3rd trustee, or localhost")
	trustee4Host      := flag.String("t4host", "localhost", "The Ip address of the 4th trustee, or localhost")
	trustee5Host      := flag.String("t5host", "localhost", "The Ip address of the 5th trustee, or localhost")

	flag.Parse()
	trusteesIp := []string{*trustee1Host, *trustee2Host, *trustee3Host, *trustee4Host, *trustee5Host}

	fmt.Println(*trustee1Host)

	readConfig()

	if(*cellSize > -1) {
		payloadlen = *cellSize
	}

	//exception
	if(*nTrustees > 5) {
		fmt.Println("Only up to 5 trustees are supported")
		os.Exit(1)
	}

	relayPortAddr := ":"+strconv.Itoa(*relayPort)

	if *isRelay {
		startRelay(*cellSize, relayPortAddr, *nClients, *nTrustees, trusteesIp, *relayReceiveLimit)
	} else if *clientId >= 0 {
		startClient(*clientId, relayPortAddr, *nClients, *nTrustees, *useSocksProxy)
	} else if *isTrusteeServer {
		startTrusteeServer()
	} else {
		println("Error: must specify -relay, -trusteesrv, -client=n, or -trustee=n")
	}
}


/*
 * CLIENT
 */


/*
 * TRUSTEE
 */

func openRelay(connectionId int) net.Conn {
	conn, err := net.Dial("tcp", relayhost)
	if err != nil {
		panic("Can't connect to relay:" + err.Error())
	}

	// Tell the relay our client or trustee number
	b := make([]byte, 1)
	b[0] = byte(connectionId)
	n, err := conn.Write(b)

	if n < 1 || err != nil {
		panic("Error writing to socket:" + err.Error())
	}

	return conn
}