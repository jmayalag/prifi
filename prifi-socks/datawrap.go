package prifi_socks


import (
  "io"
  "net"
  "encoding/binary"
)


const (
  dataWrapHeaderSize = uint16(8)
)

/** 
  * The "dataWrap" struct represents the packet communicated accross the network. With the header components and the data.
  */

type dataWrap struct {
  // Header
	ID 					      uint32     // SOCKS5 Connection ID
	MessageLength 	  uint16     // The length of useful data
	PacketLength 	   	uint16     // The length of the packet including the header

  // Data
	Data 				      []byte     // The data segment of the packet (Always of size PacketLength-HeaderLength)
}

/** 
  * Converts the struct into bytes
  */

func (d *dataWrap) ToBytes() []byte {

  // Make sure the data and messagelength are of appropriate length
  d.Data, d.MessageLength = cleanData(d.Data, d.MessageLength, d.PacketLength)

	buffer := make([]byte, int(dataWrapHeaderSize))  // Temporary byte buffer to store the header
	
  // Fill up the header buffer
  binary.BigEndian.PutUint32(buffer[0:4], d.ID)
	binary.BigEndian.PutUint16(buffer[4:6], d.MessageLength)
	binary.BigEndian.PutUint16(buffer[6:8], d.PacketLength)

  return append(buffer,d.Data...)  // Append the data to the header
}




/** 
  * Creates a new datawrap packet
  */

func NewDataWrap(ID uint32, MessageLength uint16, PacketLength uint16, Data []byte ) dataWrap {
    // Make sure the received data and messagelength are of appropriate length
    Data, MessageLength = cleanData(Data, MessageLength, PacketLength)

    return dataWrap{ID, MessageLength, PacketLength, Data} //Create the new packet
}


/** 
  * Checks for consistency withing the data in the packet and fixes any inconsistencies
  *
  * Properties:
  *     - Actual message length cannot exceed the maximum possible length (which is PacketLength - HeaderLength)
  *     - Data should always be at maximum possible length, padding is added if needed
  */

func cleanData(data []byte, messageLength uint16, packetLength uint16) ([]byte, uint16) {
  
  // Get the maximum possible length of the data
  maxMessageLength := packetLength - dataWrapHeaderSize
  
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


func trimData(data dataWrap) dataWrap {
  return NewDataWrap(data.ID, data.MessageLength, data.MessageLength+dataWrapHeaderSize,data.Data[:data.MessageLength])
}

func trimBytes(data []byte) []byte {
  newData := trimData(ExtractFull(data))

  return newData.ToBytes()
}

/** 
  * Reads the full datawarp packet: Header + Data
  */

func readFull(connReader io.Reader) (dataWrap, error) {

  // Read the header
  connID, messageLength, packetLength, err :=  readHeader(connReader)
  if err != nil {
      return NewDataWrap( 0 , 0, 0, nil), err
  }

  // Read the data
  message, err := readMessage(connReader,packetLength - dataWrapHeaderSize)
  if err != nil {
      return NewDataWrap( 0 , 0, 0, nil), err
  }

  return NewDataWrap( connID , messageLength, packetLength, message), nil
}


/** 
  * Reads only the Header of the datawarp packet
  */

func readHeader(connReader io.Reader) (uint32, uint16, uint16, error) {
  
  controlHeader := make([]byte, dataWrapHeaderSize) // Byte buffer to store the header

  _, err := io.ReadFull(connReader,controlHeader)  // Read the header
  if err != nil {
    return 0, 0, 0, err
  }

  // Extract the content of the header
  connID, messageLength, packetLength := extractHeader(controlHeader)

  // Return the content of the header
  return connID, messageLength, packetLength, nil

}

/** 
  * Reads only the Data of the datawarp packet
  */

func readMessage(connReader io.Reader, length uint16) ([]byte , error) {
    
    message := make([]byte, length) // Byte buffer to store the data
    _ , err := io.ReadFull(connReader,message)  // Read the data
    if err != nil {
      return nil,err
    }

    //Return the content of the data
    return message,nil
}

/** 
  * Sends a packet thjrough an active connection
  */

func sendMessage(conn net.Conn, data dataWrap) {
  conn.Write(data.ToBytes())
}


/** 
  * Extracts the datawrap packet from an array of bytes
  */

func ExtractFull(buffer []byte) dataWrap {
  
  if(len(buffer) < 8) {
    return NewDataWrap(0,0,0,make([]byte,0))
  }
  //Extract the content of the header
  connID, messageLength, packetLength := extractHeader(buffer)

  //Construct and return a new packet
  if int(messageLength) <= len(buffer) - 8  {
    return NewDataWrap(connID,messageLength,packetLength,buffer[dataWrapHeaderSize:dataWrapHeaderSize+messageLength])  
  }
  return NewDataWrap(connID,messageLength,packetLength,buffer[dataWrapHeaderSize:])

}


/** 
  * Extracts the datawrap header from an array of bytes
  */
func extractHeader(buffer []byte) (uint32, uint16, uint16) {

  //Extract the content of the header from the buffer
  connID := binary.BigEndian.Uint32(buffer[0:4])
  messageLength := binary.BigEndian.Uint16(buffer[4:6])
  packetLength := binary.BigEndian.Uint16(buffer[6:8])

  //Return the header components
  return connID, messageLength, packetLength
}






















