package protocols

/*
 * This class represent communication through UDP, and implements Broadcast, and ListenAndBlock (wait until there is one message).
 * When emulating in localhost with thread, we cannot use UDP broadcast (network interfaces usually ignore their self-sent messages),
 * hence this UDPChannel has two implementations : the classical UDP, and a cheating, localhost, fake-UDP broadcast done through go
 * channels.
 */

import (
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"encoding/binary"
	"gopkg.in/dedis/onet.v1/log"
	"encoding/hex"
)

//UPD_PORT is the port used for UDP broadcast
const UDP_PORT int = 10101

//MAX_UDP_SIZE is the max size of one broadcasted packet
const MAX_UDP_SIZE int = 65507

//FAKE_LOCAL_UDP_SIMULATED_LOSS_PERCENTAGE is the simulated loss percentage when we use a non-lossy local chanel
const FAKE_LOCAL_UDP_SIMULATED_LOSS_PERCENTAGE = 0

//MarshallableMessage . Since we can only send []byte over UDP, each interface{} we want to send needs to implement MarshallableMessage.
//It has methods Print(), used for debug, ToBytes(), that converts it to a raw byte array, SetByte(), which simply store a byte array in the
//structure (but does not decode it), and FromBytes(), which decodes the interface{} from the inner buffer set by SetBytes()
type MarshallableMessage interface {
	Print()

	ToBytes() ([]byte, error)

	FromBytes(data []byte) (interface{}, error)
}

//UDPChannel is the interface for UDP channel, since this class has two implementation.
type UDPChannel interface {
	Broadcast(msg MarshallableMessage) error

	//we take an empty MarshallableMessage as input, because the method does know how to parse the message
	ListenAndBlock(msg MarshallableMessage, lastSeenMessage int, identityListening string) (interface{}, error)
}

/**
 * The localhost, non-udp, cheating udp channel that uses go-channels to transmit information.
 * It has perfect orderding, and no loss.
 */
func newLocalhostUDPChannel() UDPChannel {
	return &LocalhostChannel{}
}

/**
 * The real UDP thing. IT DOES NOT WORK IN LOCAL, as network interfaces usually ignore self-sent broadcasted messages.
 */
func newRealUDPChannel() UDPChannel {
	return &RealUDPChannel{}
}

//LocalhostChannel is the fake, local UDP channel that uses channels
type LocalhostChannel struct {
	sync.RWMutex
	lastMessageID int //the first real message has ID 1, as the struct puts in a 0 when initialized
	lastMessage   []byte
}

//RealUDPChannel is the real UDP channel
type RealUDPChannel struct {
	relayConn *net.UDPConn
	localConn *net.UDPConn
}

//Broadcast of LocalhostChannel is the implementation of broadcast for the fake localhost channel
func (lc *LocalhostChannel) Broadcast(msg MarshallableMessage) error {

	lc.Lock()
	defer lc.Unlock()

	if lc.lastMessage == nil {

		log.Lvl3("Broadcast - setting msg # to 0")
		lc.lastMessageID = 0
		lc.lastMessage = make([]byte, 0)
	}

	data, err := msg.ToBytes()
	if err != nil {
		log.Error("Broadcast: could not marshal message, error is", err.Error())
	}

	//append message to the buffer bool
	lc.lastMessage = data
	lc.lastMessageID++
	log.Lvl3("Broadcast - added message, new message has Id ", lc.lastMessageID, ".")

	return nil
}

//ListenAndBlock of LocalhostChannel is the implementation of message reception for the fake localhost channel
func (lc *LocalhostChannel) ListenAndBlock(emptyMessage MarshallableMessage, lastSeenMessage int, identityListening string) (interface{}, error) {

	//we wait until there is a new message
	lc.RLock()
	defer lc.RUnlock()

	//our channel is lossy. we decide "in advance" if we will miss next message
	r := rand.Intn(100)
	willIgnoreNextMessage := false
	if r < FAKE_LOCAL_UDP_SIMULATED_LOSS_PERCENTAGE {
		willIgnoreNextMessage = true
	}

	if willIgnoreNextMessage {
		log.Lvl3("ListenAndBlock : Lossy UDP (loss", FAKE_LOCAL_UDP_SIMULATED_LOSS_PERCENTAGE, "%), we will ignore message coming after", lastSeenMessage, "and wait for message after", (lastSeenMessage + 1))
		lastSeenMessage++
	}

	log.Lvl3("ListenAndBlock - waiting on message ", (lastSeenMessage + 1), ".")

	for lc.lastMessageID == lastSeenMessage {
		//unlock before wait !
		lc.RUnlock()

		log.Lvl5("ListenAndBlock - last message is ", (lc.lastMessageID + 1), ", waiting.")
		time.Sleep(5 * time.Millisecond)
		lc.RLock()
	}

	log.Lvl3("ListenAndBlock - returning message n°" + strconv.Itoa(lastSeenMessage+1) + ".")
	//there's one
	lastMsg := lc.lastMessage

	emptyMessage.FromBytes(lastMsg)

	return emptyMessage, nil
}

//Broadcast of RealUDPChannel is the implementation of broadcast for the real UDP channel
func (c *RealUDPChannel) Broadcast(msg MarshallableMessage) error {

	//if we're not ready with the connnection yet
	if c.relayConn == nil {
		ServerAddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:"+strconv.Itoa(UDP_PORT))
		if err != nil {
			log.Error("Broadcast: could not resolve BCast address, error is", err.Error())
		}

		LocalAddr, err := net.ResolveUDPAddr("udp", ":0")
		if err != nil {
			log.Error("Broadcast: could not resolve Local address, error is", err.Error())
		}

		c.relayConn, err = net.DialUDP("udp", LocalAddr, ServerAddr)
		if err != nil {
			log.Error("Broadcast: could not UDP Dial, error is", err.Error())
		}

		//TODO : connection is never closed
	}

	data, err := msg.ToBytes()
	if err != nil {
		log.Error("Broadcast: could not marshal message, error is", err.Error())
	}

	header := make([]byte, 8)
	pattern := uint16(43690) //1010101010101010
	binary.BigEndian.PutUint16(header[0:2], pattern)
	binary.BigEndian.PutUint32(header[2:6], uint32(len(data)))
	binary.BigEndian.PutUint16(header[6:8], pattern)
	_, err = c.relayConn.Write(header)
	if err != nil {
		log.Error("Broadcast: could not write header, error is", err.Error())
	}

	_, err = c.relayConn.Write(data)
	if err != nil {
		log.Error("Broadcast: could not write message, error is", err.Error())
	} else {
		log.Lvl3("Broadcast: broadcasted one message of length", len(data))
		log.Info(hex.Dump(header))
		log.Info(hex.Dump(data))
	}

	return nil
}

//ListenAndBlock of RealUDPChannel is the implementation of message reception for the real UDP channel
func (c *RealUDPChannel) ListenAndBlock(emptyMessage MarshallableMessage, lastSeenMessage int, identityListening string) (interface{}, error) {

	//if we're not ready with the connection yet

	if c.localConn == nil {

		/* Lets prepare a address at any address at port 10001*/
		ServerAddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(UDP_PORT))
		if err != nil {
			log.Error("ListenAndBlock(",identityListening,"): could not resolve BCast address, error is", err.Error())
		}

		/* Now listen at selected port */
		c.localConn, err = net.ListenUDP("udp", ServerAddr)
		if err != nil {
			log.Error("ListenAndBlock(",identityListening,"): could not UDP Dial, error is", err.Error())
		}
	}

	header := make([]byte, 8)
	log.Info("ListenAndBlock(",identityListening,"): Ready to receive")
	n, addr, err := c.localConn.ReadFromUDP(header)
	log.Info(hex.Dump(header))
	if err != nil {
		log.Error("ListenAndBlock(",identityListening,"): could not receive header, error is", err.Error())
	}

	messageSize := int(binary.BigEndian.Uint32(header[2:6]))
	message := make([]byte, messageSize)
	log.Info("ListenAndBlock(",identityListening,"): Received a header from",addr,"gonna read message of length",messageSize,"...", n)
	n2, addr2, err2 := c.localConn.ReadFromUDP(message)
	log.Info(hex.Dump(message))

	if err2 != nil {
		log.Error("ListenAndBlock(",identityListening,"): could not receive message, error2 is", err.Error())
	} else {
		log.Lvl4("ListenAndBlock(",identityListening,"): Received a message of", n2, "bytes, from addr", addr2)
	}

	newMessage, err3 := emptyMessage.FromBytes(message)
	if err3 != nil {
		log.Error("ListenAndBlock(",identityListening,"): could not unmarshall message, error3 is", err3.Error())
	}

	return newMessage, nil
}
