package log

import (
	"fmt"
	"net"
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
)

type SinkClient struct {
	conn 			net.Conn
	copyToStdOut 	bool
	logLevel		int
}

func StartSinkClient(logLevel int, entity string, remoteHost string, copyToStdout bool) *SinkClient {

	conn, err := net.Dial("tcp", remoteHost)

	if err != nil {
		fmt.Println("Can't reach log server...")
		panic("Exiting")
	}

	sc := SinkClient{conn, copyToStdout, logLevel}
	sc.writeData([]byte(entity))

	fmt.Println("Connected to sink server...")

	return &sc
}

func (sc *SinkClient) WriteMessage(severity int, message string) error {

	if severity > sc.logLevel { //unintuitive : severity 0 is highest
		return nil
	}

	s := "<"+SeverityToString(severity)+"> "+message

	if sc.copyToStdOut && !strings.HasPrefix(message, "{")  {
		fmt.Println(s)
	}

	return sc.writeData([]byte(s))
}

func (sc *SinkClient) writeData(message []byte) error {

	if sc.conn == nil {
		panic("SinkServer : Not connected")
	}

	length := len(message)

	//compose new message
	buffer := make([]byte, length+4)
	binary.BigEndian.PutUint32(buffer[0:4], uint32(length))
	copy(buffer[4:], message)

	n, err := sc.conn.Write(buffer)

	if n < length+4 {
		return errors.New("SinkServer : Couldn't write the full"+strconv.Itoa(length+4)+" bytes, only wrote "+strconv.Itoa(n))
	}

	if err != nil {
		return err
	}

	return nil
}