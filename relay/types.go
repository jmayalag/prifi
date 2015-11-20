package relay

import (
	"github.com/lbarman/prifi/dcnet"
	"github.com/lbarman/crypto/abstract"
	"net"
	prifinet "github.com/lbarman/prifi/net"
)

const (
	PROTOCOL_STATUS_OK = iota
	PROTOCOL_STATUS_GONNA_RESYNC
	PROTOCOL_STATUS_RESYNCING
)

type IdConnectionAndPublicKey struct{
	Id 			int
	Conn 		net.Conn
	PublicKey 	abstract.Point
}

type RelayState struct {
	Name				string
	RelayPort			string

	PublicKey			abstract.Point
	privateKey			abstract.Secret
	
	nClients			int
	nTrustees			int

	trusteesHosts		[]string

	clients  			[]prifinet.NodeRepresentation
	trustees  			[]prifinet.NodeRepresentation
	
	CellCoder			dcnet.CellCoder
	
	MessageHistory		abstract.Cipher

	PayloadLength		int
	ReportingLimit		int
}