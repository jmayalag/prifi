package prifi

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/dedis/cothority/lib/dbg"
	"github.com/dedis/cothority/lib/sda"
	prifi_lib "github.com/lbarman/prifi_dev/prifi-lib"
)

/**
 * This is the struct we need to give PriFi-Lib so it can send messages.
 * It need to implement the "MessageSender interface" defined in prifi_lib/prifi.go
 */
type MessageSender struct {
	tree     *sda.TreeNodeInstance
	relay    *sda.TreeNode
	clients  map[int]*sda.TreeNode
	trustees map[int]*sda.TreeNode
}

func (ms MessageSender) SendToClient(i int, msg interface{}) error {

	if client, ok := ms.clients[i]; ok {
		dbg.Lvl5("Sending a message to client ", i, " (", client.Name(), ") - ", msg)
		return ms.tree.SendTo(client, msg)
	} else {
		e := "Client " + strconv.Itoa(i) + " is unknown !"
		dbg.Error(e)
		return errors.New(e)
	}

	return nil
}

func (ms MessageSender) SendToTrustee(i int, msg interface{}) error {

	if trustee, ok := ms.trustees[i]; ok {
		dbg.Lvl5("Sending a message to trustee ", i, " (", trustee.Name(), ") - ", msg)
		return ms.tree.SendTo(trustee, msg)
	} else {
		e := "Trustee " + strconv.Itoa(i) + " is unknown !"
		dbg.Error(e)
		return errors.New(e)
	}

	return nil
}

func (ms MessageSender) SendToRelay(msg interface{}) error {
	dbg.Lvl5("Sending a message to relay ", " - ", msg)
	return ms.tree.SendTo(ms.relay, msg)
}

var udpChan UDPChannel = newLocalhostUDPChannel()

func (ms MessageSender) BroadcastToAllClients(msg interface{}) error {

	c := 10
	b := make([]byte, c)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("error:", err)
		return nil
	}

	udpChan.Broadcast(b)
	return nil
}

func (ms MessageSender) ClientSubscribeToBroadcast(clientName string, protocolInstance *prifi_lib.PriFiProtocol, startStopChan chan bool) error {

	listening := false
	lastSeenMessage := -1

	for {
		select {
		case val := <-startStopChan:
			if val {
				listening = true //either we listen or we stop
				dbg.Lvl3("Client ", clientName, " switched on broadcast-listening.")
			} else {
				dbg.Lvl3("Client ", clientName, " killed broadcast-listening.")
				return nil
			}
		default:
		}

		if listening {
			//listen, then call
			msg, _ := udpChan.ListenAndBlock(lastSeenMessage)
			dbg.Error("Client ", clientName, "Received an UDP message !")
			protocolInstance.ReceivedMessage(msg)

		}

		time.Sleep(time.Second)
	}
	return nil
}
