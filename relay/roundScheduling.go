package relay

import (
	"encoding/binary"
	"github.com/dedis/crypto/abstract"
	"github.com/lbarman/prifi/config"
	prifilog "github.com/lbarman/prifi/log"
	prifinet "github.com/lbarman/prifi/net"
	"time"
)

func (relayState *RelayState) organizeRoundScheduling() error {
	defer prifilog.TimeTrack("relay", "organizeRoundScheduling", time.Now())

	ephPublicKeys := make([]abstract.Point, relayState.nClients)

	//collect ephemeral keys
	prifilog.Println(prifilog.INFORMATION, "Waiting for", relayState.nClients, "ephemeral keys")
	for i := 0; i < relayState.nClients; i++ {
		ephPublicKeys[i] = nil
		for ephPublicKeys[i] == nil {

			pkRead := false
			var pk abstract.Point = nil

			for !pkRead {

				prifilog.Println(prifilog.INFORMATION, "Waiting on client ", i, "'s ephemeral key")
				buffer, err := prifinet.ReadMessage(relayState.clients[i].Conn)
				publicKey := config.CryptoSuite.Point()
				msgType := int(binary.BigEndian.Uint16(buffer[0:2]))

				if msgType == prifinet.MESSAGE_TYPE_PUBLICKEYS {
					err2 := publicKey.UnmarshalBinary(buffer[2:])

					if err2 != nil {
						prifilog.Println(prifilog.INFORMATION, "Reading client ", i, "ephemeral key")
						return err
					}
					pk = publicKey
					break

				} else if msgType != prifinet.MESSAGE_TYPE_PUBLICKEYS {
					//append data in the buffer
					prifilog.Println(prifilog.WARNING, "organizeRoundScheduling: trying to read a public key message, got a data message; discarding, checking for public key in next message...")
					continue
				}
			}

			ephPublicKeys[i] = pk
		}
	}

	prifilog.Println(prifilog.INFORMATION, "Relay: collected all ephemeral public keys")

	// prepare transcript
	G_s := make([]abstract.Point, relayState.nTrustees)
	ephPublicKeys_s := make([][]abstract.Point, relayState.nTrustees)
	proof_s := make([][]byte, relayState.nTrustees)

	//ask each trustee in turn to do the oblivious shuffle
	G := config.CryptoSuite.Point().Base()
	for j := 0; j < relayState.nTrustees; j++ {

		prifinet.WriteBaseAndPublicKeyToConn(relayState.trustees[j].Conn, G, ephPublicKeys)
		prifilog.Println(prifilog.INFORMATION, "Trustee", j, "is shuffling...")

		base2, ephPublicKeys2, proof, err := prifinet.ParseBasePublicKeysAndProofFromConn(relayState.trustees[j].Conn)

		if err != nil {
			return err
		}

		prifilog.Println(prifilog.INFORMATION, "Trustee", j, "is done shuffling")

		//collect transcript
		G_s[j] = base2
		ephPublicKeys_s[j] = ephPublicKeys2
		proof_s[j] = proof

		//next trustee get this trustee's output
		G = base2
		ephPublicKeys = ephPublicKeys2
	}

	prifilog.Println(prifilog.INFORMATION, "All trustees have shuffled, sending the transcript...")

	//pack transcript
	transcriptBytes := make([]byte, 0)
	for i := 0; i < len(G_s); i++ {
		G_s_i_bytes, _ := G_s[i].MarshalBinary()
		transcriptBytes = append(transcriptBytes, prifinet.IntToBA(len(G_s_i_bytes))...)
		transcriptBytes = append(transcriptBytes, G_s_i_bytes...)
	}
	for i := 0; i < len(ephPublicKeys_s); i++ {

		pkArray := make([]byte, 0)
		for k := 0; k < len(ephPublicKeys_s[i]); k++ {
			pkBytes, _ := ephPublicKeys_s[i][k].MarshalBinary()
			pkArray = append(pkArray, prifinet.IntToBA(len(pkBytes))...)
			pkArray = append(pkArray, pkBytes...)
		}

		transcriptBytes = append(transcriptBytes, prifinet.IntToBA(len(pkArray))...)
		transcriptBytes = append(transcriptBytes, pkArray...)
	}

	for i := 0; i < len(proof_s); i++ {
		transcriptBytes = append(transcriptBytes, prifinet.IntToBA(len(proof_s[i]))...)
		transcriptBytes = append(transcriptBytes, proof_s[i]...)
	}

	//broadcast to trustees
	prifinet.NUnicastMessageToNodes(relayState.trustees, transcriptBytes)

	//wait for the signature for each trustee
	signatures := make([][]byte, relayState.nTrustees)
	for j := 0; j < relayState.nTrustees; j++ {

		buffer, err := prifinet.ReadMessage(relayState.trustees[j].Conn)
		if err != nil {
			prifilog.Println(prifilog.RECOVERABLE_ERROR, "Relay, couldn't read signature from trustee "+err.Error())
			return err
		}

		sigSize := int(binary.BigEndian.Uint32(buffer[0:4]))
		sig := make([]byte, sigSize)
		copy(sig[:], buffer[4:4+sigSize])

		signatures[j] = sig

		prifilog.Println(prifilog.INFORMATION, "Collected signature from trustee", j)
	}

	prifilog.Println(prifilog.INFORMATION, "Crafting signature message for clients...")

	sigMsg := make([]byte, 0)

	//the final shuffle is the one from the latest trustee
	lastPermutation := relayState.nTrustees - 1
	G_s_i_bytes, err := G_s[lastPermutation].MarshalBinary()
	if err != nil {
		return err
	}

	//pack the final base
	sigMsg = append(sigMsg, prifinet.IntToBA(len(G_s_i_bytes))...)
	sigMsg = append(sigMsg, G_s_i_bytes...)

	//pack the ephemeral shuffle
	pkArray, err := prifinet.MarshalPublicKeyArrayToByteArray(ephPublicKeys_s[lastPermutation])

	if err != nil {
		return err
	}

	sigMsg = append(sigMsg, prifinet.IntToBA(len(pkArray))...)
	sigMsg = append(sigMsg, pkArray...)

	//pack the trustee's signatures
	packedSignatures := make([]byte, 0)
	for j := 0; j < relayState.nTrustees; j++ {
		packedSignatures = append(packedSignatures, prifinet.IntToBA(len(signatures[j]))...)
		packedSignatures = append(packedSignatures, signatures[j]...)
	}
	sigMsg = append(sigMsg, prifinet.IntToBA(len(packedSignatures))...)
	sigMsg = append(sigMsg, packedSignatures...)

	//send to clients
	prifinet.NUnicastMessageToNodes(relayState.clients, sigMsg)

	prifilog.Println(prifilog.INFORMATION, "Oblivious shuffle & signatures sent !")
	return nil
}
