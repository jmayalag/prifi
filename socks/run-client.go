package main 

import (
	"net"

	prifisocks "github.com/lbarman/prifi_dev/prifi-socks"
)

func main() {
	toServer := make(chan []byte, 1)
	fromServer := make(chan []byte, 1)
	socksConnections := make(chan net.Conn, 1)

  	go prifisocks.StartSocksProxyServerListener(":6789",socksConnections)
  	go prifisocks.StartSocksProxyServerHandler(socksConnections, 1000, nil, toServer, fromServer)

  	prifisocks.ConnectToServer("127.0.0.1:8081",toServer, fromServer)

}