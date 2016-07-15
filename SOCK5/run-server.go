package main 

import (
	"fmt"
	"net"

	socks "github.com/lbarman/prifi_dev/SOCK5/prifi-socks"
)

func main() {

  fmt.Println("Launching server...")

  // listen on all interfaces
  ln, _ := net.Listen("tcp", ":8081")

  for {
    // accept connection on port
    conn, _ := ln.Accept()

   fmt.Println("Accepted Client Connection")   

    go socks.HandleClient(conn)
  }
 
}