package prifi

import (
	"errors"
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

func (ms MessageSender) BroadcastToAllClients(msg interface{}) error {

	castedMsg, canCast := msg.(*prifi_lib.REL_CLI_DOWNSTREAM_DATA_UDP)
	if !canCast {
		dbg.Error("Message sender : could not cast msg to REL_CLI_DOWNSTREAM_DATA_UDP, and I don't know how to send other messages.")
	}
	udpChan.Broadcast(castedMsg)

	return nil
}

func (ms MessageSender) ClientSubscribeToBroadcast(clientName string, protocolInstance *prifi_lib.PriFiProtocol, startStopChan chan bool) error {

	dbg.Lvl3(clientName, " started UDP-listener helper.")
	listening := false
	lastSeenMessage := 0 //the first real message has ID 1; this means that we saw the empty struct.

	for {
		select {
		case val := <-startStopChan:
			if val {
				listening = true //either we listen or we stop
				dbg.Lvl3(clientName, " switched on broadcast-listening.")
			} else {
				dbg.Lvl3(clientName, " killed broadcast-listening.")
				return nil
			}
		default:
		}

		if listening {
			emptyMessage := prifi_lib.REL_CLI_DOWNSTREAM_DATA_UDP{}
			//listen
			filledMessage, err := udpChan.ListenAndBlock(&emptyMessage, lastSeenMessage)

			if err != nil {
				dbg.Error(clientName, " an error occured : ", err)
			}

			//decode
			msg, err := filledMessage.FromBytes()
			dbg.Lvl3(clientName, " Received an UDP Message !")

			if err != nil {
				dbg.Error(clientName, " an error occured : ", err)
			}

			//forward to PriFi
			protocolInstance.ReceivedMessage(msg)

		}

		time.Sleep(time.Second)
	}
	return nil
}
