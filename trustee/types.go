package trustee

import (
	"net"
	"github.com/lbarman/prifi/dcnet"
	"github.com/lbarman/crypto/abstract"
)

const TRUSTEE_SERVER_LISTENING_PORT = ":9000"

type TrusteeState struct {
	Name				string
	TrusteeId			int
	PayloadLength		int
	activeConnection	net.Conn

	PublicKey			abstract.Point
	privateKey			abstract.Secret
	
	nClients			int
	nTrustees			int

	ClientPublicKeys	[]abstract.Point
	sharedSecrets		[]abstract.Point
	
	CellCoder			dcnet.CellCoder
	
	MessageHistory		abstract.Cipher
}