package trustee

import (
	"github.com/dedis/crypto/abstract"
	"github.com/lbarman/prifi/auth/daga"
	"github.com/lbarman/prifi/node"
	"net"
)

// const TRUSTEE_SERVER_LISTENING_PORT = ":9000"

type TrusteeState struct {
	node.NodeState
	activeConnection net.Conn
	ClientPublicKeys []abstract.Point
	dagaProtocol     daga.TrusteeProtocol
}
