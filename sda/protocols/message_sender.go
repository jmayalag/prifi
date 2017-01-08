package protocols

import (
	"errors"
	"strconv"
	"time"

	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/sda"
	prifi_lib "github.com/lbarman/prifi/prifi-lib"
	"github.com/lbarman/prifi/prifi-lib/net"
)

//MessageSender is the struct we need to give PriFi-Lib so it can send messages.
//It needs to implement the "MessageSender interface" defined in prifi_lib/prifi.go
type MessageSender struct {
	tree     *sda.TreeNodeInstance
	relay    *sda.TreeNode
	clients  map[int]*sda.TreeNode
	trustees map[int]*sda.TreeNode
}

//SendToClient sends a message to client i, or fails if it is unknown
func (ms MessageSender) SendToClient(i int, msg interface{}) error {

	if client, ok := ms.clients[i]; ok {
		log.Lvl5("Sending a message to client ", i, " (", client.Name(), ") - ", msg)
		return ms.tree.SendTo(client, msg)
	}

	e := "Client " + strconv.Itoa(i) + " is unknown !"
	log.Error(e)
	return errors.New(e)
}

//SendToTrustee sends a message to trustee i, or fails if it is unknown
func (ms MessageSender) SendToTrustee(i int, msg interface{}) error {

	if trustee, ok := ms.trustees[i]; ok {
		log.Lvl5("Sending a message to trustee ", i, " (", trustee.Name(), ") - ", msg)
		return ms.tree.SendTo(trustee, msg)
	}

	e := "Trustee " + strconv.Itoa(i) + " is unknown !"
	log.Error(e)
	return errors.New(e)
}

//SendToRelay sends a message to the unique relay
func (ms MessageSender) SendToRelay(msg interface{}) error {
	log.Lvl5("Sending a message to relay ", " - ", msg)
	return ms.tree.SendTo(ms.relay, msg)
}

//BroadcastToAllClients broadcasts a message (must be a REL_CLI_DOWNSTREAM_DATA_UDP) to all clients using UDP
func (ms MessageSender) BroadcastToAllClients(msg interface{}) error {

	castedMsg, canCast := msg.(*net.REL_CLI_DOWNSTREAM_DATA_UDP)
	if !canCast {
		log.Error("Message sender : could not cast msg to REL_CLI_DOWNSTREAM_DATA_UDP, and I don't know how to send other messages.")
	}
	udpChan.Broadcast(castedMsg)

	return nil
}

//ClientSubscribeToBroadcast allows a client to subscribe to UDP broadcast
func (ms MessageSender) ClientSubscribeToBroadcast(clientName string, prifiLibInstance *prifi_lib.PriFiLibInstance, startStopChan chan bool) error {

	log.Lvl3(clientName, " started UDP-listener helper.")
	listening := false
	lastSeenMessage := 0 //the first real message has ID 1; this means that we saw the empty struct.

	for {
		select {
		case val := <-startStopChan:
			if val {
				listening = true //either we listen or we stop
				log.Lvl3(clientName, " switched on broadcast-listening.")
			} else {
				log.Lvl3(clientName, " killed broadcast-listening.")
				return nil
			}
		default:
		}

		if listening {
			emptyMessage := net.REL_CLI_DOWNSTREAM_DATA_UDP{}
			//listen and decode
			filledMessage, err := udpChan.ListenAndBlock(&emptyMessage, lastSeenMessage)
			lastSeenMessage++

			if err != nil {
				log.Error(clientName, " an error occured : ", err)
			}

			log.Lvl3(clientName, " Received an UDP message n°"+strconv.Itoa(lastSeenMessage))

			if err != nil {
				log.Error(clientName, " an error occured : ", err)
			}

			//forward to PriFi
			prifiLibInstance.ReceivedMessage(filledMessage)

		}

		time.Sleep(time.Second)
	}
}
