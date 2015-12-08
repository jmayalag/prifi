package log

import (
	"fmt"
	"time"
	"net"
	"io"
	"errors"
	"strconv"
	"encoding/binary"
)

func StartSinkServer(listeningPort string, logFile string) {

	//dump to a file
	fileLogger := StartFileClient(logFile, false)

	dataChan := make(chan string)
	go serverListener(listeningPort, dataChan)

	for {
		select {
			case d := <- dataChan :
				fileLogger.WriteMessage(d)

			default:
				time.Sleep(100*time.Millisecond)
		}
	}
}

func serverListener(listeningPort string, dataChan chan string) {
	listeningSocket, err := net.Listen("tcp", listeningPort)
	if err != nil {
		panic("SinkServer : Can't open listen socket:" + err.Error())
	}

	for {
		conn, err2 := listeningSocket.Accept()
		if err != nil {
			fmt.Println("SinkServer : can't accept log client. ", err2.Error())
		}
		go handleClient(conn, dataChan)
	}
}

func handleClient(conn net.Conn, dataChan chan<- string) {

	for {
		message, err := readMessage(conn)

		if err != nil {
			fmt.Println(err)
			fmt.Println("SinkServer error, client probably disconnected, stopping goroutine...")
			break
		}

		dataChan <- string(message)
	}

	fmt.Println("Stopping handler.")
}

func readMessage(conn net.Conn) ([]byte, error) {

	header := make([]byte, 4)
	emptyMessage := make([]byte, 0)

	//read header
	n, err := io.ReadFull(conn, header)

	if err != nil{
		return emptyMessage, err
	}

	if n != 4 {
		return emptyMessage, errors.New("SinkServer Couldn't read the full 4 header bytes, only read "+strconv.Itoa(n))
	}

	bodySize := int(binary.BigEndian.Uint32(header[0:4]))

	//read body
	body := make([]byte, bodySize)
	n2, err2 := io.ReadFull(conn, body)

	if err2 != nil{
		return emptyMessage, err2
	}

	if n2 != bodySize {
		return emptyMessage, errors.New("SinkServer Couldn't read the full" + strconv.Itoa(bodySize) +" body bytes, only read "+strconv.Itoa(n2))
	}

	return body, nil
}