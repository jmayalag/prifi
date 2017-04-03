package protocols

import (
	"encoding/binary"
	"errors"
	"gopkg.in/dedis/onet.v1/log"
	"io"
	"net"
	"strconv"
)

//RealUDPChannel is the real UDP channel
type TCPChannel struct {
	conn           net.Conn
	ready          bool
	MessageHandler func([]byte)

	stop bool
}

// StartListener creates a server listener on the given port, and process up to one TCP connection on it
func (t *TCPChannel) StartListener(port int) error {

	// listen on all interfaces
	log.LLvl3("Starting tcp server for fast delivery on port", port)
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}

	// accept exactly connection
	conn, err := ln.Accept()
	if err != nil {
		return err
	}
	log.LLvl3("Accepted one fast delivery tcp connection.")
	t.conn = conn
	t.ready = true

	//loop over exactly one connection
	for !t.stop {
		message, err := readMessage(t.conn)
		if err != nil {
			log.Error("Could not read message on the fast-channel, err is", err)
		}

		t.MessageHandler(message)
	}

	return nil
}

// ConnectToServer connects to the fast delivery server
func (t *TCPChannel) ConnectToServer(addr string) error {
	// connect to this socket

	log.LLvl3("Connecting to tcp server at", addr, "for fast delivery")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	log.LLvl3("Connected to fast-delivery server.")
	t.conn = conn
	t.ready = true

	//loop over exactly one connection
	for !t.stop {
		message, err := readMessage(t.conn)
		if err != nil {
			log.Error("Could not read message on the fast-channel, err is", err)
		}

		t.MessageHandler(message)
	}
	return nil
}

// If the connection is ready, write the byte message into it.
func (t *TCPChannel) WriteMessage(msg []byte) {
	if !t.ready {
		log.Error("Cannot send when not ready")
	}
	err := writeMessage(t.conn, msg)
	if err != nil {
		log.Error("Could not read write on the fast-channel, err is", err)
	}
}

func writeMessage(conn net.Conn, message []byte) error {

	length := len(message)

	//compose new message
	buffer := make([]byte, length)
	binary.BigEndian.PutUint32(buffer[0:4], uint32(length))
	copy(buffer[4:], message)

	n, err := conn.Write(buffer)

	if n < length+4 {
		return errors.New("Couldn't write the full" + strconv.Itoa(length+4) + " bytes, only wrote " + strconv.Itoa(n))
	}

	if err != nil {
		return err
	}

	return nil
}

func readMessage(conn net.Conn) ([]byte, error) {

	header := make([]byte, 4)
	emptyMessage := make([]byte, 0)

	//read header
	n, err := io.ReadFull(conn, header)

	if err != nil {
		return emptyMessage, err
	}

	if n != 4 {
		return emptyMessage, errors.New("Couldn't read the full 4 header bytes, only read " + strconv.Itoa(n))
	}

	//parse header
	bodySize := int(binary.BigEndian.Uint32(header[0:4]))

	//read body
	body := make([]byte, bodySize)
	n2, err2 := io.ReadFull(conn, body)

	if err2 != nil {
		return emptyMessage, err2
	}

	if n2 != bodySize {
		return emptyMessage, errors.New("Couldn't read the full" + strconv.Itoa(bodySize) + " body bytes, only read " + strconv.Itoa(n2))
	}

	return body, nil
}
