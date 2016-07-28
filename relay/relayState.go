package relay

import (
	"errors"
	"github.com/dedis/crypto/abstract"
	"github.com/lbarman/prifi/auth"
	"github.com/lbarman/prifi/auth/basic"
	"github.com/lbarman/prifi/config"
	prifilog "github.com/lbarman/prifi/log"
	prifinet "github.com/lbarman/prifi/net"
	"net"
	"strconv"
	"time"
)

func initRelayState(nodeConfig config.NodeConfig, relayPort string, nTrustees int, nClients int, upstreamCellSize int, downstreamCellSize int, windowSize int, useDummyDataDown bool, reportingLimit int, trusteesHosts []string, useUDP bool) *RelayState {

	s := new(RelayState)

	s.Name = nodeConfig.Name
	s.RelayPort = relayPort
	s.UpstreamCellSize = upstreamCellSize
	s.DownstreamCellSize = downstreamCellSize
	s.WindowSize = windowSize
	s.ReportingLimit = reportingLimit
	s.UseUDP = useUDP
	s.UseDummyDataDown = useDummyDataDown

	s.privateKey = nodeConfig.PrivateKey
	s.PublicKey = nodeConfig.PublicKey

	s.nClients = nClients
	s.nTrustees = nTrustees
	s.TrusteesHosts = trusteesHosts

	// Load client public keys
	s.ClientPublicKeys = make(map[int]abstract.Point)
	for _, nodeInfo := range nodeConfig.NodesInfo {
		if nodeInfo.Type == "Client" {
			s.ClientPublicKeys[nodeInfo.Id] = nodeConfig.PublicKeyRoster[nodeInfo.Id]
		}
	}

	// Load trustee public keys
	s.TrusteePublicKeys = make(map[int]abstract.Point)
	for _, nodeInfo := range nodeConfig.NodesInfo {
		if nodeInfo.Type == "Trustee" {
			s.TrusteePublicKeys[nodeInfo.Id] = nodeConfig.PublicKeyRoster[nodeInfo.Id]
		}
	}

	s.CellCoder = config.Factory()
	return s
}

func (relayState *RelayState) deepClone() *RelayState {
	s := new(RelayState)

	s.Name = relayState.Name
	s.RelayPort = relayState.RelayPort
	s.PublicKey = relayState.PublicKey
	s.privateKey = relayState.privateKey
	s.nClients = relayState.nClients
	s.nTrustees = relayState.nTrustees
	s.TrusteesHosts = make([]string, len(relayState.TrusteesHosts))
	s.clients = make([]prifinet.NodeRepresentation, len(relayState.clients))
	s.trustees = make([]prifinet.NodeRepresentation, len(relayState.trustees))
	s.CellCoder = config.Factory()
	s.MessageHistory = relayState.MessageHistory
	s.UpstreamCellSize = relayState.UpstreamCellSize
	s.DownstreamCellSize = relayState.DownstreamCellSize
	s.WindowSize = relayState.WindowSize
	s.ReportingLimit = relayState.ReportingLimit
	s.UseUDP = relayState.UseUDP
	s.UseDummyDataDown = relayState.UseDummyDataDown
	s.UDPBroadcastConn = relayState.UDPBroadcastConn

	copy(s.TrusteesHosts, relayState.TrusteesHosts)

	for i := 0; i < len(relayState.clients); i++ {
		s.clients[i].Id = relayState.clients[i].Id
		s.clients[i].Conn = relayState.clients[i].Conn
		s.clients[i].Connected = relayState.clients[i].Connected
		s.clients[i].PublicKey = relayState.clients[i].PublicKey
	}
	for i := 0; i < len(relayState.trustees); i++ {
		s.trustees[i].Id = relayState.trustees[i].Id
		s.trustees[i].Conn = relayState.trustees[i].Conn
		s.trustees[i].Connected = relayState.trustees[i].Connected
		s.trustees[i].PublicKey = relayState.trustees[i].PublicKey
	}
	return s
}

func (relayState *RelayState) addNewClient(newClient prifinet.NodeRepresentation) {
	relayState.nClients = relayState.nClients + 1
	relayState.clients = append(relayState.clients, newClient)
}

func connectToTrusteeAsync(trusteeChan chan prifinet.NodeRepresentation, id int, host string, relayState *RelayState) {

	var err error = errors.New("empty")
	trustee := prifinet.NodeRepresentation{}

	for i := 0; i < config.NUM_RETRY_CONNECT && err != nil; i++ {
		trustee, err = connectToTrustee(host, relayState)

		if err != nil {
			prifilog.Println(prifilog.RECOVERABLE_ERROR, "Failed to connect to trustee "+strconv.Itoa(id)+" host "+host+", retrying after two second...")
			time.Sleep(2 * time.Second)
		}
	}

	if err == nil {
		trusteeChan <- trustee
	}
	prifilog.Println(prifilog.RECOVERABLE_ERROR, "Cannot connect to the trustee.")
}

func (relayState *RelayState) connectToAllTrustees() {

	defer prifilog.TimeTrack("relay", "connectToAllTrustees", time.Now())

	trusteeChan := make(chan prifinet.NodeRepresentation, relayState.nTrustees)

	// Connect to all the trustees
	for i := 0; i < relayState.nTrustees; i++ {
		go connectToTrusteeAsync(trusteeChan, i, relayState.TrusteesHosts[i], relayState)
	}

	// Wait for all the trustees to be connected
	i := 0
	for i < relayState.nTrustees {
		select {
		case trustee := <-trusteeChan:
			relayState.trustees = append(relayState.trustees, trustee)
			i++

		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	prifilog.Println(prifilog.INFORMATION, "Trustee connected,", len(relayState.trustees), "trustees connected")
}

func (relayState *RelayState) disconnectFromAllTrustees() {
	defer prifilog.TimeTrack("relay", "disconnectToAllTrustees", time.Now())

	//disconnect to the trustees
	for i := 0; i < len(relayState.trustees); i++ {
		relayState.trustees[i].Conn.Close()
	}
	relayState.trustees = make([]prifinet.NodeRepresentation, 0)
	prifilog.Println(prifilog.INFORMATION, "Trustees disonnecting done, ", len(relayState.trustees), "trustees disconnected")
}

func welcomeNewClients(newConnectionsChan chan net.Conn, newClientChan chan prifinet.NodeRepresentation,
	clientsUseUDP bool, authMethod int) {

	newClientsToParse := make(chan prifinet.NodeRepresentation)

	for {
		select {
		// Accept the TCP connection and authenticate the client
		case newConnection := <-newConnectionsChan:
			go func() {
				// Authenticate the client
				newClient, err := authenticateClient(newConnection, relayState.ClientPublicKeys)

				if err == nil {
					prifilog.Println(prifilog.INFORMATION,
						"Client "+strconv.Itoa(newClient.Id)+" authenticated successfully")
					newClientsToParse <- newClient
				} else {
					prifilog.Println(prifilog.WARNING, "Client "+strconv.Itoa(newClient.Id)+" authentication failed")
				}
			}()

		// Once client is ready, forward to the other channel
		case newClient := <-newClientsToParse:
			newClientChan <- newClient

		default:
			time.Sleep(NEWCLIENT_CHECK_SLEEP_TIME) //todo : check this duration
		}
	}
}

func (relayState *RelayState) waitForDefaultNumberOfClients(newClientConnectionsChan chan prifinet.NodeRepresentation) {
	defer prifilog.TimeTrack("relay", "waitForDefaultNumberOfClients", time.Now())

	currentClients := 0

	prifilog.Printf(prifilog.INFORMATION, "Waiting for %d clients (on port %s)", relayState.nClients-currentClients, relayState.RelayPort)

	for currentClients < relayState.nClients {
		select {
		case newClient := <-newClientConnectionsChan:
			relayState.clients = append(relayState.clients, newClient)
			currentClients += 1
			prifilog.Printf(prifilog.INFORMATION, "Waiting for %d clients (on port %s)", relayState.nClients-currentClients, relayState.RelayPort)
		default:
			time.Sleep(100 * time.Millisecond)
			//prifilog.StatisticReport("relay", "SLEEP_100ms", "100ms")
		}
	}
	prifilog.Println(prifilog.INFORMATION, "Client connected,", len(relayState.clients), "clients connected")
}

func (relayState *RelayState) excludeDisconnectedClients() {
	defer prifilog.TimeTrack("relay", "excludeDisconnectedClients", time.Now())

	//count the clients that disconnected
	nClientsDisconnected := 0
	for i := 0; i < len(relayState.clients); i++ {
		if !relayState.clients[i].Connected {
			prifilog.Println(prifilog.INFORMATION, "Relay Handler : Client ", i, " discarded, seems he disconnected...")
			nClientsDisconnected++
		}
	}

	//count the actual number of clients, and init the new state with the old parameters
	newNClients := relayState.nClients - nClientsDisconnected

	//copy the connected clients
	newClients := make([]prifinet.NodeRepresentation, newNClients)
	j := 0
	for i := 0; i < len(relayState.clients); i++ {
		if relayState.clients[i].Connected {
			newClients[j] = relayState.clients[i]
			prifilog.Println(prifilog.INFORMATION, "Adding Client ", i, "who's not disconnected")
			j++
		}
	}

	relayState.clients = newClients
}

func authenticateClient(clientConn net.Conn, publicKeyRoster map[int]abstract.Point) (prifinet.NodeRepresentation, error) {

	// Receive the client's preferred method of authentication
	var newClient prifinet.NodeRepresentation
	reqMsg, err := prifinet.ReadMessage(clientConn)
	if err != nil {
		return newClient, errors.New("Relay disconnected. " + err.Error())
	}

	authMethod := int(reqMsg[0])
	switch authMethod {
	case auth.AUTH_METHOD_BASIC:
		newClient, err = basicAuth.RelayAuthentication(clientConn, publicKeyRoster)

	case auth.AUTH_METHOD_DAGA:
		if !relayState.dagaProtocol.Initialized {
			return prifinet.NodeRepresentation{}, errors.New("DAGA protocol is not initialized")
		}
		newClient, err = relayState.dagaProtocol.AuthenticateClient(clientConn)
	}
	if err != nil {
		return prifinet.NodeRepresentation{}, errors.New("Client authentication failed. " + err.Error())
	}
	return newClient, nil
}
