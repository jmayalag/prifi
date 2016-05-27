package node

import (
	"github.com/dedis/crypto/random"
	"encoding/binary"
	"net"

	"github.com/lbarman/prifi/config"
	"github.com/lbarman/prifi/crypto"
	prifinet "github.com/lbarman/prifi/net"
	"github.com/dedis/crypto/abstract"
	"errors"
)

func InitNodeState(nodeConfig config.NodeConfig, nClients int, nTrustees int, cellSize int) NodeState {

	nodeState := new(NodeState)

	nodeState.Name = nodeConfig.Name
	nodeState.Id = nodeConfig.Id

	nodeState.NumClients = nClients
	nodeState.NumTrustees = nTrustees

	nodeState.PublicKey = nodeConfig.PublicKey
	nodeState.PrivateKey = nodeConfig.PrivateKey

	nodeState.CellSize = cellSize
	nodeState.CellCoder = config.Factory()

	nodeState.SharedSecrets = make([]abstract.Point, nClients)
	return *nodeState
}

func NodeAuthentication(relayTCPConn net.Conn, nodeState NodeState) error {

	// Send an auth request along with my ID to the relay
	reqMsg := make([]byte, 5)
	reqMsg[0] = AUTH_METHOD_PUBLIC_KEY
	binary.BigEndian.PutUint32(reqMsg[1:5], uint32(nodeState.Id))

	if err := prifinet.WriteMessage(relayTCPConn, reqMsg); err != nil {
		return errors.New("Cannot write to the relay. " + err.Error())
	}

	// Receive a challenge message from the relay
	challengeMsg, err := prifinet.ReadMessage(relayTCPConn)
	if err != nil {
		return errors.New("Relay disconnected. " + err.Error())
	}

	fields := prifinet.UnmarshalArrays(challengeMsg)
	K := config.CryptoSuite.Point()
	C := config.CryptoSuite.Point()
	K.UnmarshalBinary(fields[0])
	C.UnmarshalBinary(fields[1])

	// Decrypt the challenge
	challenge, err := crypto.ElGamalDecrypt(config.CryptoSuite, nodeState.PrivateKey, K, C)

	// Sign the challenge with my private key
	rand := random.Stream
	challengeSig := crypto.SchnorrSign(config.CryptoSuite, rand, challenge, nodeState.PrivateKey)

	// Send the signature to the relay
	if err = prifinet.WriteMessage(relayTCPConn, challengeSig); err != nil {
		return errors.New("Cannot write to the relay. " + err.Error())
	}

	// Check if authenticated
	resultMsg, err := prifinet.ReadMessage(relayTCPConn)
	if err != nil {
		return errors.New("Relay disconnected. " + err.Error())
	}
	if resultMsg[0] == 0 {
		return errors.New("Authentication rejected by the relay")
	}
	return nil
}

func RelayAuthentication(tcpConn net.Conn, publicKeyRoster map[int]abstract.Point) (prifinet.NodeRepresentation, error) {

	// TODO: Check if the node wants to perform Trust-On-First-Use (TOFU) authentication. If so, accept his temporary key pair.
	reqMsg, err := prifinet.ReadMessage(tcpConn)
	if err != nil {
		return prifinet.NodeRepresentation{}, errors.New("Node disconnected")
	}
	authMethod := reqMsg[0]
	nodeId := int(binary.BigEndian.Uint32(reqMsg[1:5]))

	if authMethod == AUTH_METHOD_PUBLIC_KEY {
		// Send a random challenge message to the client
		rand := random.Stream
		challenge := random.Bytes(20, rand)        // TODO: Provides 160-bit security

		// Encrypt the challenge with the node's public key
		K, C, remainder := crypto.ElGamalEncrypt(config.CryptoSuite, publicKeyRoster[nodeId], challenge)

		if len(remainder) > 0 {
			return prifinet.NodeRepresentation{}, errors.New("Challenge is too large to be encrypted in a single ElGamal cipher")
		}

		kBytes, _ := K.MarshalBinary()
		cBytes, _ := C.MarshalBinary()
		challengeMsg := prifinet.MarshalArrays(kBytes, cBytes)

		if err := prifinet.WriteMessage(tcpConn, challengeMsg); err != nil {
			return prifinet.NodeRepresentation{}, errors.New("Node disconnected. " + err.Error())
		}

		// Read the signed challenge
		challengeSig, err := prifinet.ReadMessage(tcpConn)
		if err != nil {
			return prifinet.NodeRepresentation{}, errors.New("Node disconnected")
		}

		// Verify the signature
		err1 := crypto.SchnorrVerify(config.CryptoSuite, challenge, publicKeyRoster[nodeId], challengeSig)
		if err1 == nil {
			// Authentication accepted; Send an acceptance message to the node
			if err := prifinet.WriteMessage(tcpConn, []byte{1}); err != nil {
				return prifinet.NodeRepresentation{}, errors.New("Node disconnected. " + err.Error())
			}
			// TODO: DAGA -- The long-term public key embedded here in node representation is later advertised to trustees and clients. This should be changed.
			newClient := prifinet.NodeRepresentation{nodeId, tcpConn, true, publicKeyRoster[nodeId]}
			return newClient, nil
		}

		// Send a rejection message to the node
		if err := prifinet.WriteMessage(tcpConn, []byte{0}); err != nil {
			return prifinet.NodeRepresentation{}, errors.New("Node disconnected. " + err.Error())
		}
	} else {
		// TODO: TOFU authentication mechanism
		return prifinet.NodeRepresentation{}, errors.New("Authentication method not supported!")
	}
	return prifinet.NodeRepresentation{}, errors.New("Node authentication failed")
}