package main 

import (
	"net"

	socks "github.com/lbarman/prifi_dev/SOCK5/prifi-socks"
)

func main() {
	toServer := make(chan []byte, 1)
	fromServer := make(chan []byte, 1)
	socksConnections := make(chan net.Conn, 1)

  	go socks.StartSocksProxyServerListener(":6789",socksConnections)
  	go socks.StartSocksProxyServerHandler(socksConnections, toServer, fromServer)

  	socks.ConnectToServer("127.0.0.1:8081",toServer, fromServer)
}