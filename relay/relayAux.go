package relay

import (
	"encoding/binary"
	"github.com/lbarman/prifi/auth/basic"
	"github.com/lbarman/prifi/config"
	prifilog "github.com/lbarman/prifi/log"
	prifinet "github.com/lbarman/prifi/net"
	"net"
	"strconv"
	"time"
)

// Invoked by the relay to advertise trustees' and clients' long-term public keys to each other
// TODO: DAGA -- This function should advertise ephemeral keys not long-term ones.
func (relayState *RelayState) advertisePublicKeys() error {
	defer prifilog.TimeTrack("relay", "advertisePublicKeys", time.Now())

	dataForClients, err := prifinet.MarshalNodeRepresentations(relayState.trustees)
	if err != nil {
		return err
	}

	dataForTrustees, err := prifinet.MarshalNodeRepresentations(relayState.clients)
	if err != nil {
		return err
	}

	// Craft the message for clients
	messageForClients := make([]byte, 6+len(dataForClients))
	binary.BigEndian.PutUint16(messageForClients[0:2], uint16(prifinet.MESSAGE_TYPE_PUBLICKEYS))
	binary.BigEndian.PutUint32(messageForClients[2:6], uint32(relayState.nClients))
	copy(messageForClients[6:], dataForClients)

	//TODO : would be cleaner if the trustees used the same structure for the message

	// Broadcast to the clients
	prifinet.NUnicastMessageToNodes(relayState.clients, messageForClients)
	prifinet.NUnicastMessageToNodes(relayState.trustees, dataForTrustees)
	prifilog.Println(prifilog.NOTIFICATION, "Advertising done, to", len(relayState.clients), "clients and", len(relayState.trustees), "trustees")

	return nil
}

func relayParseClientParamsAux(tcpConn net.Conn, clientsUseUDP bool) (prifinet.NodeRepresentation, bool) {

	message, err := prifinet.ReadMessage(tcpConn)
	if err != nil {
		prifilog.Println(prifilog.SEVERE_ERROR, "Can't read client parameters "+err.Error())
		return prifinet.NodeRepresentation{}, false
	}

	//check that the node ID is not used
	nextFreeId := 0
	for i := 0; i < len(relayState.clients); i++ {

		if relayState.clients[i].Id == nextFreeId {
			nextFreeId++
		}
	}
	prifilog.Println(prifilog.NOTIFICATION, "Client connected, assigning his ID to", nextFreeId)
	nodeId := nextFreeId

	publicKey := config.CryptoSuite.Point()
	err2 := publicKey.UnmarshalBinary(message)

	if err2 != nil {
		prifilog.Println(prifilog.SEVERE_ERROR, "can't unmarshal client key ! "+err2.Error())
		return prifinet.NodeRepresentation{}, false
	}

	newClient := prifinet.NodeRepresentation{nodeId, tcpConn, true, publicKey}

	return newClient, true
}

func newSOCKSProxyHandler(connId int, downstreamData chan<- prifinet.DataWithConnectionId) chan<- []byte {
	upstreamData := make(chan []byte)
	//go prifinet.RelaySocksProxy(connId, upstreamData, downstreamData)
	return upstreamData
}

func connectToTrustee(trusteeHostAddr string, relayState *RelayState) (prifinet.NodeRepresentation, error) {

	prifilog.Println(prifilog.NOTIFICATION, "Relay connecting to trustee on address", trusteeHostAddr)

	var conn net.Conn = nil
	var err error = nil

	// Connect to the trustee
	for conn == nil {
		conn, err = net.Dial("tcp", trusteeHostAddr)
		if err != nil {
			prifilog.Println(prifilog.RECOVERABLE_ERROR, "Can't connect to trustee on "+trusteeHostAddr+"; "+err.Error())
			conn = nil
			time.Sleep(FAILED_CONNECTION_WAIT_BEFORE_RETRY)
		}
	}

	// Tell the trustee server our parameters
	buffer := make([]byte, 16)
	binary.BigEndian.PutUint32(buffer[0:4], uint32(relayState.UpstreamCellSize))
	binary.BigEndian.PutUint32(buffer[4:8], uint32(relayState.nClients))
	binary.BigEndian.PutUint32(buffer[8:12], uint32(relayState.nTrustees))

	prifilog.Println(prifilog.NOTIFICATION, "Writing; setup is", relayState.nClients, relayState.nTrustees, "cellSize ", relayState.UpstreamCellSize)

	err2 := prifinet.WriteMessage(conn, buffer)

	if err2 != nil {
		prifilog.Println(prifilog.RECOVERABLE_ERROR, "Error writing to socket:"+err2.Error())
		return prifinet.NodeRepresentation{}, err2
	}

	// Authenticate the trustee (trustees are always authenticated via the basic public-key method)
	newTrustee, err := basicAuth.RelayAuthentication(conn, relayState.TrusteePublicKeys)
	if err != nil {
		prifilog.Println(prifilog.RECOVERABLE_ERROR, "Trustee authentication failed. "+err.Error())
		return prifinet.NodeRepresentation{}, err
	}
	prifilog.Println(prifilog.INFORMATION, "Trustee "+strconv.Itoa(newTrustee.Id)+" authenticated successfully.")
	return newTrustee, nil
}

func relayListener(listeningPort string, newConnection chan net.Conn) {
	listeningSocket, err := net.Listen("tcp", listeningPort)
	if err != nil {
		panic("Can't open listen socket:" + err.Error())
	}

	for {
		conn, err2 := listeningSocket.Accept()
		if err != nil {
			prifilog.Println(prifilog.RECOVERABLE_ERROR, "Relay : can't accept client. ", err2.Error())
		}
		newConnection <- conn
	}
}
