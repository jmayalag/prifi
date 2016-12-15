package prifi

import (
	"encoding/binary"
	"errors"
	"strconv"

	"github.com/dedis/cothority/log"
	"github.com/dedis/crypto/abstract"
)

/*
 * Messages used by PriFi.
 * Syntax : SOURCE_DEST_CONTENT_CONTENT
 *
 * Below : Message-Switch that calls the correct function when one of this message arrives.
 */

// ALL_ALL_SHUTDOWN
// ALL_ALL_PARAMETERS
// CLI_REL_TELL_PK_AND_EPH_PK
// CLI_REL_UPSTREAM_DATA
// REL_CLI_DOWNSTREAM_DATA
// REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG
// REL_CLI_TELL_TRUSTEES_PK
// REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE
// REL_TRU_TELL_TRANSCRIPT
// TRU_REL_DC_CIPHER
// TRU_REL_SHUFFLE_SIG
// REL_TRU_TELL_RATE_CHANGE
// TRU_REL_TELL_NEW_BASE_AND_EPH_PKS
// TRU_REL_TELL_PK
// REL_TRU_TELL_RATE_CHANGE

//not used yet :
// REL_CLI_DOWNSTREAM_DATA
// CLI_REL_DOWNSTREAM_NACK

// ALL_ALL_SHUTDOWN message tells the participants to stop the protocol.
type ALL_ALL_SHUTDOWN struct {
}

// ALL_ALL_PARAMETERS message contains all the parameters used by the protocol.
type ALL_ALL_PARAMETERS struct {
	ClientDataOutputEnabled bool
	DoLatencyTests          bool
	DownCellSize            int
	ForceParams             bool
	NClients                int
	NextFreeClientId        int
	NextFreeTrusteeId       int
	NTrustees               int
	RelayDataOutputEnabled  bool
	RelayReportingLimit     int
	RelayUseDummyDataDown   bool
	RelayWindowSize         int
	StartNow                bool
	UpCellSize              int
	UseUDP                  bool
}

// CLI_REL_TELL_PK_AND_EPH_PK message contains the public key and ephemeral key of a client
// and is sent to the relay.
type CLI_REL_TELL_PK_AND_EPH_PK struct {
	Pk    abstract.Point
	EphPk abstract.Point
}

// CLI_REL_UPSTREAM_DATA message contains the upstream data of a client for a given round
// and is sent to the relay.
type CLI_REL_UPSTREAM_DATA struct {
	ClientId int
	RoundId  int32
	Data     []byte
}

// REL_CLI_DOWNSTREAM_DATA message contains the downstream data for a client for a given round
// and is sent by the relay to the clients.
type REL_CLI_DOWNSTREAM_DATA struct {
	RoundId    int32
	Data       []byte
	FlagResync bool
}

// REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG message contains the ephemeral public keys and the signatures
// of the trustees and is sent by the relay to the client.
type REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG struct {
	Base         abstract.Point
	EphPks       []abstract.Point
	TrusteesSigs [][]byte
}

// REL_CLI_TELL_TRUSTEES_PK message contains the public keys of the trustees
// and is sent by the relay to the clients.
type REL_CLI_TELL_TRUSTEES_PK struct {
	Pks []abstract.Point
}

// REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE message contains the public keys and ephemeral keys
// of the clients and is sent by the relay to the trustees.
type REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE struct {
	Pks    []abstract.Point
	EphPks []abstract.Point
	Base   abstract.Point
}

// REL_TRU_TELL_TRANSCRIPT message contains all the shuffles perfomrmed in a Neff shuffle round.
// It is sent by the relay to the trustees to be verified.
type REL_TRU_TELL_TRANSCRIPT struct {
	G_s    []abstract.Point
	EphPks [][]abstract.Point
	Proofs [][]byte
}

// TRU_REL_DC_CIPHER message contains the DC-net cipher of a trustee for a given round and is sent to the relay.
type TRU_REL_DC_CIPHER struct {
	RoundId   int32
	TrusteeId int
	Data      []byte
}

// TRU_REL_SHUFFLE_SIG contains the signatures shuffled by a trustee and is sent to the relay.
type TRU_REL_SHUFFLE_SIG struct {
	TrusteeId int
	Sig       []byte
}

// REL_TRU_TELL_RATE_CHANGE message asks the trustees to update their window capacity to adapt their
// sending rate and is sent by the relay.
type REL_TRU_TELL_RATE_CHANGE struct {
	WindowCapacity int
}

// TRU_REL_TELL_NEW_BASE_AND_EPH_PKS message contains the new ephemeral key of a trustee and
// is sent to the relay.
type TRU_REL_TELL_NEW_BASE_AND_EPH_PKS struct {
	NewBase   abstract.Point
	NewEphPks []abstract.Point
	Proof     []byte
}

// TRU_REL_TELL_PK message contains the public key of a trustee and is sent to the relay.
type TRU_REL_TELL_PK struct {
	TrusteeId int
	Pk        abstract.Point
}

/*
REL_CLI_DOWNSTREAM_DATA_UDP message is a bit special. It's a REL_CLI_DOWNSTREAM_DATA, simply named with _UDP postfix to be able to distinguish them from type,
and theoretically that should be it. But since it doesn't go through SDA (which does not support UDP yet), we have to manually convert it to bytes.
For that purpose, this message implements MarshallableMessage, defined in prifi-sda-wrapper/udp.go.
Hence, it has methods Print(), used for debug, ToBytes(), that converts it to a raw byte array, SetByte(), which simply store a byte array in the
structure (but does not decode it), and FromBytes(), which decodes the REL_CLI_DOWNSTREAM_DATA from the inner buffer set by SetBytes()
*/
type REL_CLI_DOWNSTREAM_DATA_UDP struct {
	REL_CLI_DOWNSTREAM_DATA
	byteEncoded []byte
}

// Print prints the raw value of this message.
func (m REL_CLI_DOWNSTREAM_DATA_UDP) Print() {
	log.Printf("%+v\n", m)
}

// SetBytes sets the bytes contained in this message.
func (m *REL_CLI_DOWNSTREAM_DATA_UDP) SetBytes(data []byte) {
	m.byteEncoded = make([]byte, len(data))
	copy(m.byteEncoded, data)
}

// ToBytes encodes a message into a slice of bytes.
func (m *REL_CLI_DOWNSTREAM_DATA_UDP) ToBytes() ([]byte, error) {

	//convert the message to bytes
	buf := make([]byte, 4+4+len(m.REL_CLI_DOWNSTREAM_DATA.Data)+4)
	resyncInt := 0
	if m.REL_CLI_DOWNSTREAM_DATA.FlagResync {
		resyncInt = 1
	}

	binary.BigEndian.PutUint32(buf[0:4], uint32(len(buf)))
	binary.BigEndian.PutUint32(buf[4:8], uint32(m.REL_CLI_DOWNSTREAM_DATA.RoundId))
	binary.BigEndian.PutUint32(buf[len(buf)-4:], uint32(resyncInt)) //todo : to be coded on one byte
	copy(buf[8:len(buf)-4], m.REL_CLI_DOWNSTREAM_DATA.Data)

	return buf, nil

}

// FromBytes decodes the message contained in the message's byteEncoded field.
func (m *REL_CLI_DOWNSTREAM_DATA_UDP) FromBytes() (interface{}, error) {

	buffer := m.byteEncoded

	//the smallest message is 4 bytes, indicating a length of 0
	if len(buffer) < 4 {
		e := "Messages.go : FromBytes() : cannot decode, smaller than 4 bytes"
		log.Error(e)
		return REL_CLI_DOWNSTREAM_DATA_UDP{}, errors.New(e)
	}

	messageSize := int(binary.BigEndian.Uint32(buffer[0:4]))

	if len(buffer) != messageSize {
		e := "Messages.go : FromBytes() : cannot decode, advertised length is " + strconv.Itoa(messageSize) + ", actual length is " + strconv.Itoa(len(buffer))
		log.Error(e)
		return REL_CLI_DOWNSTREAM_DATA_UDP{}, errors.New(e)
	}

	roundId := int32(binary.BigEndian.Uint32(buffer[4:8]))
	flagResyncInt := int(binary.BigEndian.Uint32(buffer[len(buffer)-4:]))
	data := buffer[8 : len(buffer)-4]

	flagResync := false
	if flagResyncInt == 1 {
		flagResync = true
	}

	innerMessage := REL_CLI_DOWNSTREAM_DATA{roundId, data, flagResync} //This wrapping feels wierd
	resultMessage := REL_CLI_DOWNSTREAM_DATA_UDP{innerMessage, make([]byte, 0)}

	return resultMessage, nil
}


// ReceivedMessage must be called when a PriFi host receives a message.
// It takes care to call the correct message handler function.
func (prifi *PriFiProtocol) ReceivedMessage(msg interface{}) error {

	if prifi == nil {
		log.Print("Received a message ", msg)
		panic("But prifi is nil !")
	}

	var err error

	switch typedMsg := msg.(type) {
	case ALL_ALL_PARAMETERS:
		switch prifi.role {
		case PRIFI_ROLE_RELAY:
			go prifi.Received_ALL_REL_PARAMETERS(typedMsg)
		case PRIFI_ROLE_CLIENT:
			err = prifi.Received_ALL_CLI_PARAMETERS(typedMsg)
		case PRIFI_ROLE_TRUSTEE:
			err = prifi.Received_ALL_TRU_PARAMETERS(typedMsg)
		default:
			panic("Received parameters, but we have no role yet !")
		}
	case ALL_ALL_SHUTDOWN:
		switch prifi.role {
		case PRIFI_ROLE_RELAY:
			go prifi.Received_ALL_REL_SHUTDOWN(typedMsg)
		case PRIFI_ROLE_CLIENT:
			err = prifi.Received_ALL_CLI_SHUTDOWN(typedMsg)
		case PRIFI_ROLE_TRUSTEE:
			err = prifi.Received_ALL_TRU_SHUTDOWN(typedMsg)
		default:
			panic("Received SHUTDOWN, but we have no role yet !")
		}
	case CLI_REL_TELL_PK_AND_EPH_PK:
		go prifi.Received_CLI_REL_TELL_PK_AND_EPH_PK(typedMsg)
	case CLI_REL_UPSTREAM_DATA:
		go prifi.Received_CLI_REL_UPSTREAM_DATA(typedMsg)
	case REL_CLI_DOWNSTREAM_DATA:
		err = prifi.Received_REL_CLI_DOWNSTREAM_DATA(typedMsg)
	/*
	 * this message is a bit special. At this point, we don't care anymore that's it's UDP, and cast it back to REL_CLI_DOWNSTREAM_DATA.
	 * the relay only handles REL_CLI_DOWNSTREAM_DATA
	 */
	case REL_CLI_DOWNSTREAM_DATA_UDP:
		err = prifi.Received_REL_CLI_UDP_DOWNSTREAM_DATA(typedMsg.REL_CLI_DOWNSTREAM_DATA)
	case REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG:
		err = prifi.Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG(typedMsg)
	case REL_CLI_TELL_TRUSTEES_PK:
		err = prifi.Received_REL_CLI_TELL_TRUSTEES_PK(typedMsg)
	case REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE:
		err = prifi.Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE(typedMsg)
	case REL_TRU_TELL_TRANSCRIPT:
		err = prifi.Received_REL_TRU_TELL_TRANSCRIPT(typedMsg)
	case TRU_REL_DC_CIPHER:
		go prifi.Received_TRU_REL_DC_CIPHER(typedMsg)
	case TRU_REL_SHUFFLE_SIG:
		go prifi.Received_TRU_REL_SHUFFLE_SIG(typedMsg)
	case TRU_REL_TELL_NEW_BASE_AND_EPH_PKS:
		go prifi.Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS(typedMsg)
	case TRU_REL_TELL_PK:
		go prifi.Received_TRU_REL_TELL_PK(typedMsg)
	case REL_TRU_TELL_RATE_CHANGE:
		err = prifi.Received_REL_TRU_TELL_RATE_CHANGE(typedMsg)
	default:
		panic("unrecognized message !")
	}

	//no need to push the error further up. display it here !
	if err != nil {
		log.Error("ReceivedMessage: got an error, " + err.Error())
		return err
	}

	return nil
}