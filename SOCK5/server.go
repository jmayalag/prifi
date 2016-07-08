package main

import (
  "net"
  "fmt"
  "bufio"
  "io"
  "encoding/binary"
  "errors"
  "strconv"
)

// Authentication methods
const (
  methNoAuth = iota
  methGSS
  methUserPass
  methNone = 0xff
)

// Address types
const (
  addrIPv4   = 0x01
  addrDomain = 0x03
  addrIPv6   = 0x04
)

// Commands
const (
  cmdConnect   = 0x01
  cmdBind      = 0x02
  cmdAssociate = 0x03
)

// Reply codes
const (
  repSucceeded = iota
  repGeneralFailure
  repConnectionNotAllowed
  repNetworkUnreachable
  repHostUnreachable
  repConnectionRefused
  repTTLExpired
  repCommandNotSupported
  repAddressTypeNotSupported
)


func main() {

  fmt.Println("Launching server...")

  // listen on all interfaces
  ln, _ := net.Listen("tcp", ":8081")

  for {
    // accept connection on port
    conn, _ := ln.Accept()

   fmt.Println("Accepted Client Connection")   

    go handleClient(conn)
  }
 
}

func handleClient(conn net.Conn) {
    
    allChannels := make( map[uint32]chan []byte )

    fmt.Println("Handling Conection")

    /*   Client SOCKS Request Handling     */

    connReader := bufio.NewReader(conn)

    for {

    connID, messageLength, err :=  readHeader(connReader)
    if err != nil {
      //handle error
      fmt.Println("Header Error")
      return
    }

    fmt.Println("Connection ID:", connID  )
    fmt.Println("Length:", messageLength )


    myChan := allChannels[connID]

    if myChan == nil {

      newChan := make(chan []byte)
      allChannels[connID] = newChan

      go hanndleChannel(conn, newChan, connID)

      myChan = newChan

    } 

    clientPacket := make([]byte, messageLength)
    _ , err = io.ReadFull(connReader,clientPacket)
    if err != nil {
      //handle error
      fmt.Println("Data Read Error")
      return
    }

    myChan <- clientPacket
}
    
}



func hanndleChannel(conn net.Conn, clientPacket chan []byte, connID uint32) {
  
  connReader := newChanReader(clientPacket)

  //Read SOCKS Version
    socksVersion := []byte{0}
    _, err := io.ReadFull(connReader,socksVersion)
    if err != nil {
      //handle error
      fmt.Println("Version Error")
      return
    } else if int(socksVersion[0]) != 5 {
      //handle socks version
      fmt.Println("Version:", int( socksVersion[0] ) )
      return
    }

    fmt.Println( "Socks Version is:", int( socksVersion[0] ) )

    //Read SOCKS Number of Methods
    socksNumOfMethods := []byte{0}
    _ , err = io.ReadFull(connReader,socksNumOfMethods)
    if err != nil {
      //handle error

      return
    }

    fmt.Println("Socks Num of Methods is:", int(socksNumOfMethods[0] ) )

    //Read SOCKS Methods
    numOfMethods := int( socksNumOfMethods[0] )
    socksMethods := make([]byte, numOfMethods)
    _, err = io.ReadFull(connReader,socksMethods)
    if err != nil {
      //handle error

      return
    }



    // Find a supported method (currently only NoAuth)
    foundMethod := false
    for i := 0; i< len(socksMethods); i++ {
      if socksMethods[i] == methNoAuth {
        foundMethod = true
        break
      }
    }

    if !foundMethod {
      //handle not finding method

      return
    }

    fmt.Println("Constructing Response")

    //Construct Response Message
    methodSelectionResponse := []byte{ socksVersion[0] , byte(methNoAuth) }
    replyToClient(conn, connID, methodSelectionResponse)

    fmt.Println("Sent Response")


    /*   Client Web Request Handling    */

    requestHeader := make([] byte, 4)
    _, err = io.ReadFull(connReader,requestHeader)
    if err != nil {
      //handle error
      fmt.Println("Request Header Error")
      return
    }

    fmt.Println("Version Type:", int( requestHeader[0] ) )
    fmt.Println("CMD Type:", int( requestHeader[1] ) )
    fmt.Println("Reserved Type:", int( requestHeader[2] ) )
    fmt.Println("Address Type:", int( requestHeader[3] ) )

    destinationIP, err :=  readSocksAddr(connReader, int(requestHeader[3]))
    if err != nil {
      //handle error
      fmt.Println("IP Address Error")
      return
    }


    destinationPortBytes := make([]byte, 2)
    _, err = io.ReadFull(connReader,destinationPortBytes)
    if err != nil {
      //handle error
      fmt.Println("Destination Port Error")
      return
    }
    destinationPort := binary.BigEndian.Uint16(destinationPortBytes)


    destinationAddress := (&net.TCPAddr{IP: destinationIP, Port: int(destinationPort)}).String()
    //destinationAddress := fmt.Sprintf("%s:%d", destinationIP, destinationPort)

    fmt.Println("Web server @",destinationAddress)

    // Process the command
    switch int(requestHeader[1]) {
      case cmdConnect:
        webConn, err := net.Dial("tcp", destinationAddress)
        if err != nil {
          fmt.Println("Failed to connect to web server")
          return
        }

        // Send success reply downstream
        sucessMessage := createSocksReply(repSucceeded, conn.LocalAddr())
        replyToClient(conn, connID, sucessMessage)

        // Commence forwarding raw data on the connection
        go socksRelayDown(webConn, conn, connID)
        go socksRelayUp(webConn, connReader, connID)

      default:
        fmt.Println("Cannot Process Command")
    }

}




func socksRelayDown(webConn net.Conn, conn net.Conn, connID uint32) {
  for {
    buf := make([]byte, 100000)
    n, _ := webConn.Read(buf)
    buf = buf[:n]
    // Forward the data (or close indication if n==0) downstream
    replyToClient(conn, connID, buf)

    // Connection error or EOF?
    if n == 0 {
      webConn.Close()
      return
    }
  }
}


func socksRelayUp(webConn net.Conn, connReader io.Reader, connID uint32) {

  for {
    // Get the next upstream data buffer
    buf := make([]byte, 100000)
    messageLength, err := io.ReadFull(connReader,buf)
    if err != nil {
      //handle error
      fmt.Println("Header Error")
      return
    }

    if messageLength == 0 { // connection close indicator
      return
    }
    //println(hex.Dump(buf))
    n, err := webConn.Write(buf[:messageLength])
    if n != messageLength {
      return
    }

  }
}






func replyToClient(conn net.Conn, connID uint32, message []byte) {
  byteID := make([]byte,4)
  binary.BigEndian.PutUint32(byteID, connID)

  conn.Write(byteID)
  conn.Write([]byte( string(message) + "\n"))

  fmt.Println("Replied to Client with ID:", connID)
}

// Read an IPv4 or IPv6 address from an io.Reader and return it as a string
func readIP(r io.Reader, len int) (net.IP, error) {
  errorIP := make(net.IP, net.IPv4len)

  addr := make([]byte, len)
  _, err := io.ReadFull(r, addr)
  if err != nil {
    return errorIP, err
  }
  return net.IP(addr), nil
}

func readSocksAddr(cr io.Reader, addrtype int) (net.IP, error) {
  
 errorIP := make(net.IP, net.IPv4len)

  switch addrtype {
  case addrIPv4:
    return readIP(cr, net.IPv4len)

  case addrIPv6:
    return readIP(cr, net.IPv6len)

  case addrDomain:

    // First read the 1-byte domain name length
    dlen := [1]byte{}
    _, err := io.ReadFull(cr, dlen[:])
    if err != nil {
      return errorIP, err
    }

    // Now the domain name itself
    domain := make([]byte, int(dlen[0]))
    _, err = io.ReadFull(cr, domain)
    if err != nil {
      return errorIP, err
    }

    return net.IP(domain), nil

  default:
    msg := fmt.Sprintf("unknown SOCKS address type %d", addrtype)
    fmt.Println(msg)
    return errorIP, errors.New(msg)
  }

}


func createSocksReply(replyCode int, addr net.Addr) []byte {
  
  buf := make([]byte, 4)
  buf[0] = byte(5) // version
  buf[1] = byte(replyCode)

   // Address type
  if addr != nil {

    tcpaddr := addr.(*net.TCPAddr)
    host4 := tcpaddr.IP.To4()
    host6 := tcpaddr.IP.To16()

    i, _ := strconv.Atoi("6789")

    fmt.Println("Success Address",tcpaddr.IP)
    fmt.Println("Success Port",i)

    port := [2]byte{}
    binary.BigEndian.PutUint16(port[:], uint16(i))//tcpaddr.Port))

    if host4 != nil { // it's an IPv4 address

      buf[3] = addrIPv4
      buf = append(buf, host4...)
      buf = append(buf, port[:]...)

    } else if host6 != nil { // it's an IPv6 address

      buf[3] = addrIPv6
      buf = append(buf, host6...)
      buf = append(buf, port[:]...)

    } else { // huh???

      fmt.Println("SOCKS: neither IPv4 nor IPv6 addr?")
      addr = nil
      buf[1] = byte(repAddressTypeNotSupported)

    }

  } else { // attach a null IPv4 address
    buf[3] = addrIPv4
    buf = append(buf, make([]byte, 4+2)...)
  }

   return buf
 }



func sendMessage(IP string, message []byte) {

  conn, _ := net.Dial("tcp", IP)
  conn.Write(message)

  m, _ := bufio.NewReader(conn).ReadBytes('\n')

  fmt.Println(string(m))

  conn.Close()

}

func readHeader(connReader io.Reader) (uint32, uint16, error) {
  
  controlHeader := make([]byte, 6)

  _, err := io.ReadFull(connReader,controlHeader)  
  if err != nil {
    return 0, 0, err
  }

  connID := binary.BigEndian.Uint32(controlHeader[0:4])
  messageLength := binary.BigEndian.Uint16(controlHeader[4:6])

  return connID, messageLength, nil

}

type chanreader struct {
  b   []byte
  c   <-chan []byte
  eof bool
}

func (cr *chanreader) Read(p []byte) (n int, err error) {
  if cr.eof {
    return 0, io.EOF
  }
  blen := len(cr.b)
  if blen == 0 {
    cr.b = <-cr.c // read next block from channel
    blen = len(cr.b)
    if blen == 0 { // channel sender signaled EOF
      cr.eof = true
      return 0, io.EOF
    }
  }

  act := min(blen, len(p))
  copy(p, cr.b[:act])
  cr.b = cr.b[act:]
  return act, nil
}

func newChanReader(c <-chan []byte) *chanreader {
  return &chanreader{[]byte{}, c, false}
}

func min(x, y int) int {
  if x < y {
    return x
  }
  return y
}