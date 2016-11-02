// Package prifi-sda-protocol contains the SDA protocol that transmits
// prifi-lib messages through the network.
// As of now it only contains a dummy example protocol for demonstration purpose.
package prifi

import (
	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/network"
	"github.com/dedis/cothority/sda"
)

const ProtocolName = "Protocool"

// MyMsg contains a single message string
type MyMsg struct {
	Text string
}

// MyMsgStruct is used by the SDA to transmit messages of type MyMsg.
// The message handler functions must take a struct containing a
// network registered message struct (here MyMsg) and a *sda.TreeNode
// which will contain the TreeNode that sent the message.
type MyMsgStruct struct {
	*sda.TreeNode
	MyMsg
}

// MyProtocol contains the protocol's state on one node and must implement
// the sda.ProtocolInstance interface (most of it is implemented by
// the contained TreeNodeInstance, we only have to implement Start())
type MyProtocol struct {
	*sda.TreeNodeInstance
}

/* Register protocol and packet types with SDA on initialization */
func init() {
	network.RegisterPacketType(MyMsg{})

	sda.GlobalProtocolRegister(ProtocolName, newProtocolInstance)
}

// Start is called by the service when starting the protocol.
// It is part of the sda.Protocol interface.
func (p *MyProtocol) Start() error {
	log.Info("Starting PriFi protocol...")

	// This naive protocol sends a kind message to all the children nodes
	msg := MyMsg{
		Text: "Hello children !",
	}

	log.Info("Sending message to children...")

	for _, c := range p.Children() {
		log.Info(c.ServerIdentity)
		err := p.SendTo(c, &msg)
		if err != nil {
			log.Info("Error while sending msg:", err)
		}
	}

	log.Info("Messages sent !")

	return nil
}

// HandleMsg handles a message of type MyMsg received by the SDA.
// It is automatically called by the SDA when a message of type MyMsg is received.
// See also MyMsgStruct comments.
func (p *MyProtocol) HandleMsg(msg MyMsgStruct) error {
	text := msg.Text
	log.Info("Received message:", text)
	return nil
}

// newProtocolInstance creates a new protocol instance (see MyProtocol comments)
func newProtocolInstance(n *sda.TreeNodeInstance) (sda.ProtocolInstance, error) {
	// Initialize protocol state
	pi := &MyProtocol{
		TreeNodeInstance: n,
	}

	// Register message handler(s)
	if err := pi.RegisterHandler(pi.HandleMsg); err != nil {
		log.Fatal("Could not register handler:", err)
	}

	return pi, nil
}
