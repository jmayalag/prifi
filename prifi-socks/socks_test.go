package prifisocks

import (
	"testing"

	"github.com/dedis/cothority/log"
	prifi_socks "github.com/lbarman/prifi/prifi-socks"
	"net"
)

func TestSocksConnect(t *testing.T) {

	log.Lvl1("Testing SOCKS server")

	payloadLength := 1000
	upstreamChannelServer := make(chan []byte)
	downstreamChannelServer := make(chan []byte)
	port := ":8080"

	upstreamChannelClient := make(chan []byte)
	downstreamChannelClient := make(chan []byte)
	portServer := "127.0.0.1:8080"

	go prifi_socks.StartSocksServer(port, payloadLength, upstreamChannelServer, downstreamChannelServer, false)
	go prifi_socks.StartSocksClient(portServer, upstreamChannelClient, downstreamChannelClient)

	conn, err := net.Dial("tcp", port)
	if err != nil {
		log.Error("SOCKS PriFi Client: Could not connect to SOCKS server.", err)
	}

	//First message is
	message_0 := make([]byte, 0)

	conn.Write(message_0)

}
