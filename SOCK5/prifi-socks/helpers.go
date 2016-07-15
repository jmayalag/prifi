package prifi_socks


import (
  "io"
  "net"
  "encoding/binary"
)



type dataWrap struct {
	ID 					uint32
	MessageLength 		uint16
	PacketLength 		uint16
	Data 				[] byte
}

func NewDataWrap(ID uint32, MessageLength uint16, PacketLength uint16, Data []byte ) dataWrap {
    return dataWrap{ID, MessageLength, PacketLength, Data}
}

func (d *dataWrap) ToBytes() []byte {
	// Read up to a cell worth of data to send upstream
	buffer := make([]byte, 8)
	// Encode the connection number and actual data length
	binary.BigEndian.PutUint32(buffer[0:4], d.ID)
	binary.BigEndian.PutUint16(buffer[4:6], d.MessageLength)
	binary.BigEndian.PutUint16(buffer[6:8], d.PacketLength)

	return append(buffer,d.Data...)  //This needs modifying to account for big packets
}





type chanreader struct {
  b   []byte
  c   <-chan []byte
  eof bool
}

func (cr *chanreader) Read(p []byte) (n int, err error) {
  if cr.eof {
    return 0, io.EOF
  }
  blen := len(cr.b)
  if blen == 0 {
    cr.b = <-cr.c // read next block from channel
    blen = len(cr.b)
    if blen == 0 { // channel sender signaled EOF
      cr.eof = true
      return 0, io.EOF
    }
  }

  act := min(blen, len(p))
  copy(p, cr.b[:act])
  cr.b = cr.b[act:]
  return act, nil
}

func newChanReader(c <-chan []byte) *chanreader {
  return &chanreader{[]byte{}, c, false}
}

func min(x, y int) int {
  if x < y {
    return x
  }
  return y
}







func sendMessage(conn net.Conn, data dataWrap) {
  conn.Write(data.ToBytes())
}


func readHeader(connReader io.Reader) (uint32, uint16, uint16, error) {
  
  controlHeader := make([]byte, 8)

  _, err := io.ReadFull(connReader,controlHeader)  
  if err != nil {
    return 0, 0, 0, err
  }

  connID := binary.BigEndian.Uint32(controlHeader[0:4])
  messageLength := binary.BigEndian.Uint16(controlHeader[4:6])
  packetLength := binary.BigEndian.Uint16(controlHeader[6:8])

  return connID, messageLength, packetLength, nil

}

func extractData(buffer []byte) dataWrap {
 
  connID := binary.BigEndian.Uint32(buffer[0:4])
  messageLength := binary.BigEndian.Uint16(buffer[4:6])
  payloadLength := binary.BigEndian.Uint16(buffer[6:8])

  return NewDataWrap(connID,messageLength,payloadLength,buffer[8:8+messageLength])

}



