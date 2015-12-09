package log

import (
	"fmt"
	"time"
	"net"
	"io"
	"errors"
	"strconv"
	"encoding/binary"
	"strings"
	"github.com/fatih/color"
)

type EntityAndMessage struct {
	entity  string
	message string
}

func StartSinkServer(listeningPort string, logFile string) {

	//dump to a file
	fileLogger := StartFileClient(INFORMATION, logFile, false)

	dataChan := make(chan EntityAndMessage)
	go serverListener(listeningPort, dataChan)

	for {
		select {
			case d := <- dataChan :
				entity := d.entity
				msg    := d.message
				s := "["+entity+"] "+msg
				fileLogger.writeMessage(s)

				if !strings.HasPrefix(msg, "{") {
					switch(entity) {
						case "relay":
							color.Set(color.FgGreen)
							fmt.Println(s)
							color.Unset()
							break
						case "trusteeServer":
							color.Set(color.FgYellow)
							fmt.Println(s)
							color.Unset()
							break
						case "trustee0":
							color.Set(color.FgRed)
							fmt.Println(s)
							color.Unset()
							break
						case "trustee1":
							color.Set(color.FgYellow)
							fmt.Println(s)
							color.Unset()
							break
						case "trustee2":
							color.Set(color.FgBlue)
							fmt.Println(s)
							color.Unset()
							break
						case "trustee3":
							color.Set(color.FgMagenta)
							fmt.Println(s)
							color.Unset()
							break
						case "trustee4":
							color.Set(color.FgCyan)
							fmt.Println(s)
							color.Unset()
							break
						case "trustee5":
							color.Set(color.FgGreen)
							fmt.Println(s)
							color.Unset()
							break
						case "client0":
							color.Set(color.FgCyan)
							fmt.Println(s)
							color.Unset()
							break
						case "client1":
							color.Set(color.FgMagenta)
							fmt.Println(s)
							color.Unset()
							break
						case "client2":
							color.Set(color.FgBlue)
							fmt.Println(s)
							color.Unset()
							break
						case "client3":
							color.Set(color.FgYellow)
							fmt.Println(s)
							color.Unset()
							break
						case "client4":
							color.Set(color.FgRed)
							fmt.Println(s)
							color.Unset()
							break
						case "client5":
							color.Set(color.FgGreen)
							fmt.Println(s)
							color.Unset()
							break
						default:
							color.Set(color.FgWhite)
							fmt.Println(s)
							color.Unset()
							break					
					}
				}

			default:
				time.Sleep(100*time.Millisecond)
		}
	}
}

func serverListener(listeningPort string, dataChan chan EntityAndMessage) {
	
	fmt.Println("SinkServer : listening on " + listeningPort)
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

func handleClient(conn net.Conn, dataChan chan<- EntityAndMessage) {
	defer fmt.Println("Stopping handler.")

	//the first message is the entity
	entityBytes, err := readMessage(conn)

	entity := string(entityBytes)

	if err != nil {
		fmt.Println(err)
		panic("SinkServer error, could not read entity...")
		return
	}

	color.Set(color.FgWhite)
	fmt.Println("[Sink] connected to entity ", entity)
	color.Unset()

	for {
		message, err := readMessage(conn)

		if err != nil {
			fmt.Println(err)
			fmt.Println("SinkServer error, client probably disconnected, stopping goroutine...")
			break
		}

		dataChan <- EntityAndMessage{entity, string(message)}
	}

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