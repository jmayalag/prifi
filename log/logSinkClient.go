package log

import (
	"fmt"
	"net"
	"encoding/binary"
	"errors"
	"strconv"
)

type SinkClient struct {
	conn 			net.Conn
	copyToStdOut 	bool
}

func StartSinkClient(remoteHost string, copyToStdout bool) *SinkClient {

	conn, err := net.Dial("tcp", remoteHost)

	if err != nil {
		fmt.Println("Can't reach log server...")
		panic("Exiting")
	}

	return &SinkClient{conn, copyToStdout}
}

func (sc *SinkClient) WriteMessage(message string) error {
	return sc.writeData([]byte(message))
}

func (sc *SinkClient) writeData(message []byte) error {

	if sc.conn == nil {
		panic("SinkServer : Not connected")
	}

	length := len(message)

	//compose new message
	buffer := make([]byte, length+4)
	binary.BigEndian.PutUint32(buffer[0:4], uint32(length))
	copy(buffer[6:], message)

	n, err := sc.conn.Write(buffer)

	if n < length+4 {
		return errors.New("SinkServer : Couldn't write the full"+strconv.Itoa(length+4)+" bytes, only wrote "+strconv.Itoa(n))
	}

	if err != nil {
		return err
	}

	if sc.copyToStdOut {
		fmt.Println(message)
	}

	return nil
}