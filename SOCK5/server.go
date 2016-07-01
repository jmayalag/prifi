package main

import "net"
import "fmt"
import "bufio"
import "strings" // only needed below for sample processing

func main() {

  fmt.Println("Launching server...")

  // listen on all interfaces
  ln, _ := net.Listen("tcp", ":8081")

  // accept connection on port
  conn, _ := ln.Accept()

  // run loop forever (or until ctrl-c)
  for {
    // will listen for message to process ending in newline (\n)
    message, _ := bufio.NewReader(conn).ReadBytes('\n')
    
    client, _ := conn.RemoteAddr().(*net.TCPAddr)
    fmt.Println(client.IP)

    //sendMessage("127.0.0.1:6789", message)

    // output message received
    fmt.Print("Message Received:", string(message))
    // sample process for string received
    newmessage := strings.ToUpper(string(message))
    // send new string back to client
    conn.Write([]byte(newmessage + "\n"))
  }
}


func sendMessage(IP string, message []byte) {

  conn, _ := net.Dial("tcp", IP)
  conn.Write(message)

  m, _ := bufio.NewReader(conn).ReadBytes('\n')

  fmt.Println(string(m))

  conn.Close()

}