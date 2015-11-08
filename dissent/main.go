package main

import (
	"flag"
	"fmt"
	"strconv"
	"os"
	"os/signal"
	"net"
	"encoding/binary"
	"github.com/lbarman/crypto/nist"
	"github.com/lbarman/prifi/dcnet"
	"github.com/lbarman/crypto/abstract"

	log2 "github.com/lbarman/prifi/log"
)

//used to make sure everybody has the same version of the software. must be updated manually
const LLD_PROTOCOL_VERSION = 2

//sets the crypto suite used
var suite = nist.NewAES128SHA256P256()

//sets the factory for the dcnet's cell encoder/decoder
var factory = dcnet.SimpleCoderFactory

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
	nClients          := flag.Int("nclients", 3, "The number of clients.")
	nTrustees         := flag.Int("ntrustees", 2, "The number of trustees.")
	cellSize          := flag.Int("cellsize", 128, "Sets the size of one cell, in bytes.")
	relayPort         := flag.Int("relayport", 9876, "Sets listening port of the relay, waiting for clients.")
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

	//exception
	if(*nTrustees > 5) {
		fmt.Println("Only up to 5 trustees are supported")
		os.Exit(1)
	}

	relayPortAddr := "localhost:"+strconv.Itoa(*relayPort)

	if *isRelay {
		startRelay(*cellSize, relayPortAddr, *nClients, *nTrustees, trusteesIp, *relayReceiveLimit)
	} else if *clientId >= 0 {
		startClient(*clientId, relayPortAddr, *nClients, *nTrustees, *cellSize, *useSocksProxy)
	} else if *isTrusteeServer {
		startTrusteeServer()
	} else {
		println("Error: must specify -relay, -trusteesrv, -client=n, or -trustee=n")
	}
}

func broadcastMessage(conns []net.Conn, message []byte) {
	for i:=0; i<len(conns); i++ {
		n, err := conns[i].Write(message)

		if n < len(message) || err != nil {
			fmt.Println("Could not broadcast to conn", i)
			panic("Error writing to socket:" + err.Error())
		}
	}
}

func tellPublicKey(conn net.Conn, publicKey abstract.Point) {
	publicKeyBytes, _ := publicKey.MarshalBinary()
	keySize := len(publicKeyBytes)

	//tell the relay our public key (assume user verify through second channel)
	buffer := make([]byte, 8+keySize)
	copy(buffer[8:], publicKeyBytes)
	binary.BigEndian.PutUint32(buffer[0:4], uint32(LLD_PROTOCOL_VERSION))
	binary.BigEndian.PutUint32(buffer[4:8], uint32(keySize))

	n, err := conn.Write(buffer)

	if n < len(buffer) || err != nil {
		panic("Error writing to socket:" + err.Error())
	}
}

func MarshalPublicKeyArrayToByteArray(publicKeys []abstract.Point) []byte {
	var byteArray []byte

	for i:=0; i<len(publicKeys); i++ {
		publicKeysBytes, err := publicKeys[i].MarshalBinary()
		publicKeyLength := make([]byte, 4)
		binary.BigEndian.PutUint32(publicKeyLength, uint32(len(publicKeysBytes)))

		byteArray = append(byteArray, publicKeyLength...)
		byteArray = append(byteArray, publicKeysBytes...)

		//fmt.Println(hex.Dump(publicKeysBytes))
		if err != nil{
			panic("can't marshal client public key n°"+strconv.Itoa(i))
		}
	}

	return byteArray
}

func UnMarshalPublicKeyArrayFromConnection(conn net.Conn) []abstract.Point {

	//collect the public keys from the trustees
	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	if err != nil {
		panic("Read error:" + err.Error())
	}

	//will hold the public keys
	var publicKeys []abstract.Point

	//parse message
	currentByte := 0
	currentPkId := 0
	for {
		keyLength := int(binary.BigEndian.Uint32(buffer[currentByte:currentByte+4]))

		if keyLength == 0 {
			break; //we reached the end of the array
		}

		keyBytes := buffer[currentByte+4:currentByte+4+keyLength]

		publicKey := suite.Point()
		err2 := publicKey.UnmarshalBinary(keyBytes)
		if err2 != nil {
			panic(">>>>can't unmarshal key n°"+strconv.Itoa(currentPkId)+" ! " + err2.Error())
		}

		publicKeys = append(publicKeys, publicKey)

		currentByte += 4 + keyLength
		currentPkId += 1
	}

	return publicKeys;
}