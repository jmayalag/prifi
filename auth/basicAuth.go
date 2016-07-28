package auth

import (
	"encoding/binary"
	"errors"
	"github.com/dedis/crypto/abstract"
	"github.com/dedis/crypto/random"
	"github.com/lbarman/prifi/config"
	"github.com/lbarman/prifi/crypto"
	prifinet "github.com/lbarman/prifi/net"
	"net"
)

// Performs basic public-key authentication using Schnorr's signature
func clientBasicAuthentication(serverConn net.Conn, nodeId int, privateKey abstract.Scalar) error {

	// Send an auth request along with my ID to the relay
	reqMsg := make([]byte, 5)
	reqMsg[0] = AUTH_METHOD_BASIC
	binary.BigEndian.PutUint32(reqMsg[1:5], uint32(nodeId))

	if err := prifinet.WriteMessage(serverConn, reqMsg); err != nil {
		return errors.New("Cannot write to the relay. " + err.Error())
	}

	// Receive a challenge message from the relay
	challengeMsg, err := prifinet.ReadMessage(serverConn)
	if err != nil {
		return errors.New("Relay disconnected. " + err.Error())
	}

	fields := prifinet.UnmarshalArrays(challengeMsg)
	K := config.CryptoSuite.Point()
	C := config.CryptoSuite.Point()
	K.UnmarshalBinary(fields[0])
	C.UnmarshalBinary(fields[1])

	// Decrypt the challenge
	challenge, err := crypto.ElGamalDecrypt(config.CryptoSuite, privateKey, K, C)

	// Sign the challenge with my private key
	rand := random.Stream
	challengeSig := crypto.SchnorrSign(config.CryptoSuite, rand, challenge, privateKey)

	// Send the signature to the relay
	if err = prifinet.WriteMessage(serverConn, challengeSig); err != nil {
		return errors.New("Cannot write to the relay. " + err.Error())
	}

	// Check if authenticated
	resultMsg, err := prifinet.ReadMessage(serverConn)
	if err != nil {
		return errors.New("Relay disconnected. " + err.Error())
	}
	if resultMsg[0] == 0 {
		return errors.New("Authentication rejected by the relay")
	}
	return nil
}

func serverBasicAuthentication(clientConn net.Conn, publicKeyRoster map[int]abstract.Point) (prifinet.NodeRepresentation, error) {

	// TODO: Check if the node wants to perform Trust-On-First-Use (TOFU) authentication. If so, accept his temporary key pair.
	reqMsg, err := prifinet.ReadMessage(clientConn)
	if err != nil {
		return prifinet.NodeRepresentation{}, errors.New("Node disconnected")
	}
	authMethod := reqMsg[0]
	nodeId := int(binary.BigEndian.Uint32(reqMsg[1:5]))

	if authMethod == AUTH_METHOD_BASIC {
		// Send a random challenge message to the client
		rand := random.Stream
		challenge := random.Bytes(20, rand) // TODO: Provides 160-bit security

		// Encrypt the challenge with the node's public key
		K, C, remainder := crypto.ElGamalEncrypt(config.CryptoSuite, publicKeyRoster[nodeId], challenge)

		if len(remainder) > 0 {
			return prifinet.NodeRepresentation{}, errors.New("Challenge is too large to be encrypted in a single ElGamal cipher")
		}

		kBytes, _ := K.MarshalBinary()
		cBytes, _ := C.MarshalBinary()
		challengeMsg := prifinet.MarshalArrays(kBytes, cBytes)

		if err := prifinet.WriteMessage(clientConn, challengeMsg); err != nil {
			return prifinet.NodeRepresentation{}, errors.New("Node disconnected. " + err.Error())
		}

		// Read the signed challenge
		challengeSig, err := prifinet.ReadMessage(clientConn)
		if err != nil {
			return prifinet.NodeRepresentation{}, errors.New("Node disconnected")
		}

		// Verify the signature
		err1 := crypto.SchnorrVerify(config.CryptoSuite, challenge, publicKeyRoster[nodeId], challengeSig)
		if err1 == nil {
			// Authentication accepted; Send an acceptance message to the node
			if err := prifinet.WriteMessage(clientConn, []byte{1}); err != nil {
				return prifinet.NodeRepresentation{}, errors.New("Node disconnected. " + err.Error())
			}
			// TODO: DAGA -- The long-term public key embedded here in node representation is later advertised to trustees and clients. This should be changed.
			newClient := prifinet.NodeRepresentation{nodeId, clientConn, true, publicKeyRoster[nodeId]}
			return newClient, nil
		}

		// Send a rejection message to the node
		if err := prifinet.WriteMessage(clientConn, []byte{0}); err != nil {
			return prifinet.NodeRepresentation{}, errors.New("Node disconnected. " + err.Error())
		}
	} else {
		// TODO: TOFU authentication mechanism
		return prifinet.NodeRepresentation{}, errors.New("Authentication method not supported!")
	}
	return prifinet.NodeRepresentation{}, errors.New("Node authentication failed")
}
