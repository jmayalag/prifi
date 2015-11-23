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

const WAIT_FOR_PUBLICKEY_SLEEP_TIME =  100 * time.Millisecond

type ParamsFromRelay struct {
	publicKeys []abstract.Point
	nClients  	int
}

type ClientState struct {
	Id					int
	Name				string

	PublicKey			abstract.Point
	privateKey			abstract.Secret

	nClients			int
	nTrustees			int

	PayloadLength		int
	UsablePayloadLength	int
	UseSocksProxy		bool
	
	TrusteePublicKey	[]abstract.Point
	sharedSecrets		[]abstract.Point
	
	CellCoder			dcnet.CellCoder
	
	MessageHistory		abstract.Cipher
}

func newClientState(clientId int, nTrustees int, nClients int, payloadLength int, useSocksProxy bool) *ClientState {

	params := new(ClientState)

	params.Name                = "Client-"+strconv.Itoa(clientId)
	params.Id 				   = clientId
	params.nClients            = nClients
	params.nTrustees           = nTrustees
	params.PayloadLength       = payloadLength
	params.UseSocksProxy       = useSocksProxy

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

func (clientState *ClientState) printSecrets() {
	//print all shared secrets
	
	fmt.Println("")
	for i:=0; i<clientState.nTrustees; i++ {
		fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
		fmt.Println("            TRUSTEE", i)
		d1, _ := clientState.TrusteePublicKey[i].MarshalBinary()
		d2, _ := clientState.sharedSecrets[i].MarshalBinary()
		fmt.Println(hex.Dump(d1))
		fmt.Println("+++")
		fmt.Println(hex.Dump(d2))
	}
	fmt.Println("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
	fmt.Println("")
}