package trustee

import (
	"net"
	"github.com/dedis/crypto/abstract"
	"github.com/lbarman/prifi/node"
)

const TRUSTEE_SERVER_LISTENING_PORT = ":9000"

type TrusteeState struct {

	node.NodeState
	activeConnection	net.Conn
	ClientPublicKeys	[]abstract.Point
}