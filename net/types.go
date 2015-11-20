package net

import (
	"net"
	"github.com/lbarman/crypto/abstract"
)

type NodeRepresentation struct {
	Id			int
	Conn 		net.Conn
	Connected 	bool
	PublicKey	abstract.Point
}

type DataWithConnectionId struct {
	ConnectionId 	int    // connection number
	Data 			[]byte // data buffer
}

type DataWithMessageType struct {
	MessageType 	int    
	Data 			[]byte
}

type DataWithMessageTypeAndConnId struct {
	MessageType 	int    
	ConnectionId 	int    // connection number (SOCKS id)
	Data 			[]byte
}


const SOCKS_CONNECTION_ID_EMPTY = 0

const (
	MESSAGE_TYPE_DATA = iota
	MESSAGE_TYPE_DATA_AND_RESYNC
	MESSAGE_TYPE_PUBLICKEYS
	MESSAGE_TYPE_LAST_UPLOAD_FAILED
)