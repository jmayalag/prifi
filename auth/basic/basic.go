package basicAuth

import (
	"encoding/binary"
	"errors"
	"github.com/dedis/crypto/abstract"
	"github.com/dedis/crypto/random"
	"github.com/lbarman/prifi/auth"
	"github.com/lbarman/prifi/config"
	"github.com/lbarman/prifi/crypto"
	prifinet "github.com/lbarman/prifi/net"
	"net"
)

// Called by a client who wants to be authenticated
func ClientAuthentication(relayConn net.Conn, nodeId int, privateKey abstract.Scalar) error {

	// Send an auth request to the relay
	idMsg := make([]byte, 5)
	idMsg[0] = auth.AUTH_METHOD_BASIC
	binary.BigEndian.PutUint32(idMsg[1:5], uint32(nodeId))

	if err := prifinet.WriteMessage(relayConn, idMsg); err != nil {
		return errors.New("Cannot write to the relay. " + err.Error())
	}

	// Receive a challenge message from the relay
	challengeMsg, err := prifinet.ReadMessage(relayConn)
	if err != nil {
		return errors.New("Relay disconnected. " + err.Error())
	}

	fields := prifinet.UnmarshalByteArrays(challengeMsg)
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
	if err = prifinet.WriteMessage(relayConn, challengeSig); err != nil {
		return errors.New("Cannot write to the relay. " + err.Error())
	}

	// Check if authenticated
	resultMsg, err := prifinet.ReadMessage(relayConn)
	if err != nil {
		return errors.New("Relay disconnected. " + err.Error())
	}
	if resultMsg[0] == 0 {
		return errors.New("Authentication rejected by the relay")
	}
	return nil
}

// Called by the relay to authenticate a client
func RelayAuthentication(clientConn net.Conn, publicKeyRoster map[int]abstract.Point) (prifinet.NodeRepresentation, error) {

	// Receive an auth request from the client
	idMsg, err := prifinet.ReadMessage(clientConn)
	if err != nil {
		return prifinet.NodeRepresentation{}, errors.New("Node disconnected")
	}
	if len(idMsg) != 5 {
		return prifinet.NodeRepresentation{}, errors.New("Unexpected message from client. Expecting client ID.")
	}
	nodeId := int(binary.BigEndian.Uint32(idMsg))

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
	challengeMsg := prifinet.MarshalByteArrays(kBytes, cBytes)

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
	return prifinet.NodeRepresentation{}, errors.New("Node authentication failed")
}
