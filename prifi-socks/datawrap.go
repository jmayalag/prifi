package prifisocks

import (
	"encoding/binary"
)

// Possible message/packet types.
const (
	DummyData = iota
	SocksConnect
	StallCommunication
	ResumeCommunication
	SocksData
	SocksClosed
	SocksError
)

// Size of the data wrapper header.
const (
	SocksPacketHeaderSize = uint16(10)
)

//SocksPacket represents the packet communicated across the network. It contains the header components and the data.
type SocksPacket() struct {
	// Header
	Type          uint16
	ID            uint32 // SOCKS5 Connection ID
	MessageLength uint16 // The length of useful data
	PacketLength  uint16 // The length of the packet including the header

	// Data
	Data []byte // The data segment of the packet (Always of size PacketLength-HeaderLength)
}

//ToBytes converts the SocksPacket to a byte array
func (d *SocksPacket) ToBytes() []byte {

	// Make sure the data and messagelength are of appropriate length
	d.Data, d.MessageLength = socksTrimAndPadPayload(d.Data, d.MessageLength, d.PacketLength)

	buffer := make([]byte, int(SocksPacketHeaderSize)) // Temporary byte buffer to store the header

	// Fill up the header buffer
	binary.BigEndian.PutUint16(buffer[0:2], d.Type)
	binary.BigEndian.PutUint32(buffer[2:6], d.ID)
	binary.BigEndian.PutUint16(buffer[6:8], d.MessageLength)
	binary.BigEndian.PutUint16(buffer[8:10], d.PacketLength)

	return append(buffer, d.Data...) // Append the data to the header
}

//NewSocksPacket creates a new socks packet
func NewSocksPacket(Type uint16, ID uint32, MessageLength uint16, PacketLength uint16, Data []byte) SocksPacket {

	// Make sure the received data and messagelength are of appropriate length
	Data, MessageLength = socksTrimAndPadPayload(Data, MessageLength, PacketLength)

	return SocksPacket{Type, ID, MessageLength, PacketLength, Data} //Create the new packet
}

/*
socksTrimAndPadPayload checks for consistency withing the data in the packet and fixes any inconsistencies

Properties:
	- Actual message length cannot exceed the maximum possible length (which is PacketLength - HeaderLength)
	- Data should always be at maximum possible length, padding is added if needed
*/
func socksTrimAndPadPayload(data []byte, messageLength uint16, packetLength uint16) ([]byte, uint16) {

	// Get the maximum possible length of the data
	maxMessageLength := packetLength - SocksPacketHeaderSize

	// Check if the message length exceeds the maximum possible length, and truncate the length
	if messageLength > maxMessageLength {
		messageLength = maxMessageLength
	}

	// Add the necessary padding to the data and truncate it to maximum length
	padding := make([]byte, maxMessageLength)
	data = append(data, padding...)
	data = data[:maxMessageLength]

	// Return the modified data and message length
	return data, messageLength
}

//ParseSocksPacketFromBytes extracts the SocksPacket packet from an array of bytes.
func ParseSocksPacketFromBytes(buffer []byte) SocksPacket {

	if len(buffer) < int(SocksPacketHeaderSize) {
		return NewSocksPacket(0, 0, 0, 0, make([]byte, 0))
	}
	//Extract the content of the header
	packetType, connID, messageLength, packetLength := ParseSocksHeaderFromBytes(buffer)

	//Construct and return a new packet
	if int(messageLength) <= len(buffer)-int(SocksPacketHeaderSize) {
		return NewSocksPacket(packetType, connID, messageLength, packetLength, buffer[SocksPacketHeaderSize:SocksPacketHeaderSize+messageLength])
	}
	return NewSocksPacket(packetType, connID, messageLength, packetLength, buffer[SocksPacketHeaderSize:])

}

//ParseSocksHeaderFromBytes extracts the SocksPacket header from an array of bytes.
func ParseSocksHeaderFromBytes(buffer []byte) (uint16, uint32, uint16, uint16) {

	//Extract the content of the header from the buffer
	packetType := binary.BigEndian.Uint16(buffer[0:2])
	connID := binary.BigEndian.Uint32(buffer[2:6])
	messageLength := binary.BigEndian.Uint16(buffer[6:8])
	packetLength := binary.BigEndian.Uint16(buffer[8:10])

	//Return the header components
	return packetType, connID, messageLength, packetLength
}
