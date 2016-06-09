package auth

import (
	"errors"
	"github.com/dedis/crypto/abstract"
	prifinet "github.com/lbarman/prifi/net"
	"net"
)

func ClientAuthentication(authMethod int, serverConn net.Conn, nodeId int, privateKey abstract.Secret) error {

	switch authMethod {
	case AUTH_METHOD_BASIC:

		if err := clientBasicAuthentication(serverConn, nodeId, privateKey); err != nil {
			return errors.New("Authentication failed.")
		}

	case AUTH_METHOD_DAGA:

	default:
		return errors.New("Authentication method not supported.")
	}
	return nil
}

func ServerAuthentication(authMethod int, clientConn net.Conn, publicKeyRoster map[int]abstract.Point) (prifinet.NodeRepresentation, error) {

	var nodeRep prifinet.NodeRepresentation
	var err error

	switch authMethod {
	case AUTH_METHOD_BASIC:

		nodeRep, err = serverBasicAuthentication(clientConn, publicKeyRoster)
		if err != nil {
			return prifinet.NodeRepresentation{}, errors.New("Authentication failed.")
		}

	case AUTH_METHOD_DAGA:

	default:
		return prifinet.NodeRepresentation{}, errors.New("Authentication method not supported.")
	}
	return nodeRep, nil
}
