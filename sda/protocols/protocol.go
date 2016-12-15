package prifi

import (
	"github.com/dedis/cothority/sda"
	"github.com/dedis/cothority/network"
	"github.com/dedis/cothority/log"
)

// Define message struct
type MyMsg struct {
	Text string
}

// The message handler functions must take a struct containing a
// network registered message struct (here MyMsg) and a *sda.TreeNode
// which will contain the TreeNode that sent the message.
type MyMsgStruct struct {
	*sda.TreeNode
	MyMsg
}

// This contains the protocol's state on one node and must implement
// the sda.ProtocolInstance interface (most of it is implemented by
// the contained TreeNodeInstance, we only have to implement Start())
type MyProtocol struct {
	*sda.TreeNodeInstance
}

/* Register protocol and packet types with SDA on initialization */
func init() {
	network.RegisterPacketType(MyMsg{})

	//sda.GlobalProtocolRegister(ProtocolName, newProtocolInstance)
}

// This is called by the service when starting the protocol
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

// This is automatically called by the SDA when a message of type MyMsg is received
// See also MyMsgStruct comments
func (p *MyProtocol) HandleMsg(msg MyMsgStruct) error {
	text := msg.Text
	log.Info("Received message:", text)
	return nil
}

// Creates a new protocol instance (see MyProtocol comments)
func newProtocolInstance(n *sda.TreeNodeInstance) (sda.ProtocolInstance, error) {
	// Initialize protocol state
	pi := &MyProtocol{
		TreeNodeInstance: n,
	}

	// Register message handler(s)
	if err:= pi.RegisterHandler(pi.HandleMsg); err != nil {
		log.Fatal("Could not register handler:", err)
	}

	return pi, nil
}
