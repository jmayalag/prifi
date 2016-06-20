package prifi

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/dedis/cothority/lib/dbg"
)

const UDPPORT int = 10101
const MAXUDPSIZEINBYTES int = 65507

type MarshallableMessage interface {
	Print()

	SetBytes(data []byte)

	ToBytes() ([]byte, error)

	FromBytes() (interface{}, error)
}

type UDPChannel interface {
	Broadcast(msg MarshallableMessage) error

	//we take an empty MarshallableMessage as input, because the method does know how to parse the message
	ListenAndBlock(msg MarshallableMessage, lastSeenMessage int) (MarshallableMessage, error)
}

func newLocalhostUDPChannel() UDPChannel {
	return &LocalhostChannel{}
}

func newRealUDPChannel() UDPChannel {
	return &RealUDPChannel{}
}

type LocalhostChannel struct {
	sync.RWMutex
	lastMessageId int //the first real message has ID 1, as the struct puts in a 0 when initialized
	lastMessage   []byte
}

type RealUDPChannel struct {
	relayConn *net.UDPConn
	localConn *net.UDPConn
}

func (lc *LocalhostChannel) Broadcast(msg MarshallableMessage) error {

	dbg.Lvl3("Broadcast - aquiring lock")
	lc.Lock()
	defer lc.Unlock()

	if lc.lastMessage == nil {

		dbg.Lvl3("Broadcast - setting msg # to 1")
		lc.lastMessageId = 1
		lc.lastMessage = make([]byte, 0)
	}

	data, err := msg.ToBytes()
	if err != nil {
		dbg.Error("Broadcast: could not marshal message, error is", err.Error())
	}

	//append message to the buffer bool
	lc.lastMessage = data
	lc.lastMessageId++
	dbg.Lvl3("Broadcast - added message ", lc.lastMessageId, ".")

	return nil
}

func (lc *LocalhostChannel) ListenAndBlock(emptyMessage MarshallableMessage, lastSeenMessage int) (MarshallableMessage, error) {

	//we wait until there is a new message
	lc.RLock()
	defer lc.RUnlock()

	dbg.Lvl3("ListenAndBlock - waiting on message ", (lastSeenMessage + 1), ".")
	for lc.lastMessageId == lastSeenMessage {
		//unlock before wait !
		lc.RUnlock()

		dbg.Lvl5("ListenAndBlock - last message is ", (lc.lastMessageId + 1), ", waiting.")
		time.Sleep(5 * time.Millisecond)
		lc.RLock()
	}

	dbg.Lvl3("ListenAndBlock - returning message n°" + strconv.Itoa(lastSeenMessage+1) + ".")
	//there's one
	lastMsg := lc.lastMessage

	emptyMessage.SetBytes(lastMsg)

	return emptyMessage, nil
}

func (c *RealUDPChannel) Broadcast(msg MarshallableMessage) error {

	//if we're not ready with the connnection yet
	if c.relayConn == nil {
		ServerAddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:"+strconv.Itoa(UDPPORT))
		if err != nil {
			dbg.Error("Broadcast: could not resolve BCast address, error is", err.Error())
		}

		LocalAddr, err := net.ResolveUDPAddr("udp", ":0")
		if err != nil {
			dbg.Error("Broadcast: could not resolve Local address, error is", err.Error())
		}

		c.relayConn, err = net.DialUDP("udp", LocalAddr, ServerAddr)
		if err != nil {
			dbg.Error("Broadcast: could not UDP Dial, error is", err.Error())
		}

		//TODO : connection is never closed
	}

	data, err := msg.ToBytes()
	if err != nil {
		dbg.Error("Broadcast: could not marshal message, error is", err.Error())
	}

	_, err = c.relayConn.Write(data)
	if err != nil {
		dbg.Error("Broadcast: could not write message, error is", err.Error())
	} else {
		dbg.Lvl3("Broadcast: broadcasted one message")
	}

	return nil
}

func (c *RealUDPChannel) ListenAndBlock(emptyMessage MarshallableMessage, lastSeenMessage int) (MarshallableMessage, error) {

	//if we're not ready with the connnection yet

	if c.localConn == nil {

		/* Lets prepare a address at any address at port 10001*/
		ServerAddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(UDPPORT))
		if err != nil {
			dbg.Error("ListenAndBlock: could not resolve BCast address, error is", err.Error())
		}

		/* Now listen at selected port */
		c.localConn, err = net.ListenUDP("udp", ServerAddr)
		if err != nil {
			dbg.Error("ListenAndBlock: could not UDP Dial, error is", err.Error())
		}
	}

	buf := make([]byte, MAXUDPSIZEINBYTES)

	n, addr, err := c.localConn.ReadFromUDP(buf)
	fmt.Println("Received ", string(buf[0:n]), " from ", addr)

	if err != nil {
		dbg.Error("ListenAndBlock: could not receive message, error is", err.Error())
	} else {
		dbg.Error("ListenAndBlock: Received a message of", n, "bytes, from addr", addr)
	}

	emptyMessage.SetBytes(buf)

	return emptyMessage, nil
}
