package client

import (
	"fmt"
	"encoding/hex"
	"github.com/lbarman/prifi/dcnet"
	"github.com/lbarman/crypto/abstract"
	"strconv"
	"time"
	"github.com/lbarman/prifi/config"
)

// Number of bytes of cell payload to reserve for connection header, length
const socksHeaderLength = 6

const WAIT_FOR_PUBLICKEY_SLEEP_TIME       = 100 * time.Millisecond
const FAILED_CONNECTION_WAIT_BEFORE_RETRY = 1000 * time.Millisecond
const UDP_DATAGRAM_WAIT_TIMEOUT           = 1 * time.Second

type ParamsFromRelay struct {
	trusteesPublicKeys 	[]abstract.Point
	nClients  			int
}

type ClientState struct {
	Id					int
	Name				string

	PublicKey			abstract.Point
	privateKey			abstract.Secret

	EphemeralPublicKey	abstract.Point
	ephemeralPrivateKey	abstract.Secret

	nClients			int
	nTrustees			int

	PayloadLength		int
	UsablePayloadLength	int
	UseSocksProxy		bool
	LatencyTest			bool
	UseUDP				bool

	TrusteePublicKey	[]abstract.Point
	sharedSecrets		[]abstract.Point
	
	CellCoder			dcnet.CellCoder
	
	MessageHistory		abstract.Cipher
}

func newClientState(clientId int, nTrustees int, nClients int, payloadLength int, useSocksProxy bool, latencyTest bool, useUDP bool) *ClientState {

	params := new(ClientState)

	params.Name                = "Client-"+strconv.Itoa(clientId)
	params.Id 				   = clientId
	params.nClients            = nClients
	params.nTrustees           = nTrustees
	params.PayloadLength       = payloadLength
	params.UseSocksProxy       = useSocksProxy
	params.LatencyTest 		   = latencyTest
	params.UseUDP 			   = useUDP

	//prepare the crypto parameters
	rand 	:= config.CryptoSuite.Cipher([]byte(params.Name))
	base	:= config.CryptoSuite.Point().Base()

	//generate own parameters
	params.privateKey       = config.CryptoSuite.Secret().Pick(rand)
	params.PublicKey        = config.CryptoSuite.Point().Mul(base, params.privateKey)

	//placeholders for pubkeys and secrets
	params.TrusteePublicKey = make([]abstract.Point,  nTrustees)
	params.sharedSecrets    = make([]abstract.Point, nTrustees)

	//sets the cell coder, and the history
	params.CellCoder           = config.Factory()
	params.UsablePayloadLength = params.CellCoder.ClientCellSize(payloadLength)

	return params
}

func (clientState *ClientState) generateEphemeralKeys() {

	//prepare the crypto parameters
	rand 	:= config.CryptoSuite.Cipher([]byte(clientState.Name))
	base	:= config.CryptoSuite.Point().Base()

	//generate ephemeral keys
	Epriv := config.CryptoSuite.Secret().Pick(rand)
	Epub := config.CryptoSuite.Point().Mul(base, Epriv)

	clientState.EphemeralPublicKey  = Epub
	clientState.ephemeralPrivateKey = Epriv

}

func (clientState *ClientState) printSecrets() {
	//print all secrets
	

	k1, _ := clientState.PublicKey.MarshalBinary()
	k2, _ := clientState.privateKey.MarshalBinary()
	var k3, k4 []byte

	if clientState.EphemeralPublicKey != nil {
		k3, _ = clientState.EphemeralPublicKey.MarshalBinary()
		k4, _ = clientState.ephemeralPrivateKey.MarshalBinary()
	}

	fmt.Println("")
	fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
	fmt.Println("            CLIENT ", clientState.Id)
	fmt.Println("Public key :")
	fmt.Println(hex.Dump(k1))
	fmt.Println("private key :")
	fmt.Println(hex.Dump(k2))

	if clientState.EphemeralPublicKey != nil {
		fmt.Println("Ephemeral public key :")
		fmt.Println(hex.Dump(k3))
		fmt.Println("Ephemeral private key :")
		fmt.Println(hex.Dump(k4))
	}

	for i:=0; i<clientState.nTrustees; i++ {
		fmt.Println("> > > > > > > > > > > > > > > > >")
		fmt.Println("   Shared Params With Trustee", i)
		d1, _ := clientState.TrusteePublicKey[i].MarshalBinary()
		d2, _ := clientState.sharedSecrets[i].MarshalBinary()
		fmt.Println("Trustee public key :")
		fmt.Println(hex.Dump(d1))
		fmt.Println("Shared secret :")
		fmt.Println(hex.Dump(d2))
	}
	fmt.Println("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
	fmt.Println("")
}