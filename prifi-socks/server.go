package prifi_socks

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


/** 
  * SOCKS5 Client Connection Handler.  
  */

func HandleClient(conn net.Conn) {

    fmt.Println("Handling Conection")

    // For every SOCKS5 client, we keep track of the channels for each SOCKS connection made with the browser at the client side
    allChannels := make( map[uint32]chan []byte )

    // Create connection reader
    connReader := bufio.NewReader(conn)

    // This loop reads packets continuously and sends them through a channel to the appropriate channel handler
    for {
      //Read a full datawrap packet
      newData, err :=  readFull(connReader)
      if err != nil {
        //handle error
        fmt.Println("Data Read Error")
        return
      }

      //Extract the needed content from the packet (Connection ID & Data)
      connID := newData.ID 
      clientPacket :=  newData.Data[:newData.MessageLength]

      //Datrawrap packets with ID=0 are discarded (this indicates a useless packet)
      if connID == 0 {
        fmt.Println("Dummy Message Received")
        continue
      }

      // Get the channel associated with the connection ID
      myChan := allChannels[connID]

      // If no channel exists yet, create one and setup a channel handler
      if myChan == nil {

        // Create a new channel for the new connection ID
        newChan := make(chan []byte)
        allChannels[connID] = newChan

        // Instantiate a channel handler
        go hanndleChannel(conn, newChan, connID)

        myChan = newChan

      } 

      // Send the data through the appropriate channel
      myChan <- clientPacket
  }
    
}

/** 
  * Channel Handler assigned for a certain connection ID which handles the packets sent by the client with that ID  
  */

func hanndleChannel(conn net.Conn, clientPacket chan []byte, connID uint32) {
  
  // Create a channel reader
  connReader := newChanReader(clientPacket)


  /* SOCKS5 Method Selection Phase */

  // Read SOCKS Version
  socksVersion, err := readMessage(connReader,1)
  if err != nil {
    // handle error
    fmt.Println("Version Error")
    return
  } else if int(socksVersion[0]) != 5 {
    // handle socks version
    fmt.Println("Version:", int( socksVersion[0] ) )
    return
  }
  
  // Read SOCKS Number of Methods
  socksNumOfMethods, err := readMessage(connReader,1)
  if err != nil {
    //handle error
     return
  }
  
  // Read SOCKS Methods
  numOfMethods := uint16( socksNumOfMethods[0] )
  socksMethods, err := readMessage(connReader,numOfMethods)
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
  
  //Construct Response Message
  methodSelectionResponse := []byte{ socksVersion[0] , byte(methNoAuth) }
  sendMessage(conn, NewDataWrap(connID,uint16(len(methodSelectionResponse)),uint16(len(methodSelectionResponse))+dataWrapHeaderSize,methodSelectionResponse))


  /* SOCKS5 Web Server Request Phase */

  // Read SOCKS Request Header (Version, Command, Address Type)
  requestHeader, err := readMessage(connReader,4)
  if err != nil {
    //handle error
    fmt.Println("Request Header Error")
    return
  }
  
  // Read Web Server IP
  destinationIP, err :=  readSocksAddr(connReader, int(requestHeader[3]))
  if err != nil {
    //handle error
    fmt.Println("IP Address Error")
    return
  }
  
  // Read Web Server Port
  destinationPortBytes, err := readMessage(connReader,2)
  if err != nil {
    //handle error
    fmt.Println("Destination Port Error")
    return
  }
  
  // Process Address and Port  
  destinationPort := binary.BigEndian.Uint16(destinationPortBytes)
  destinationAddress := (&net.TCPAddr{IP: destinationIP, Port: int(destinationPort)}).String()
  

  // Process the command
  switch int(requestHeader[1]) {
     case cmdConnect: // Process "Connect" command
        
        //Connect to the web server
        fmt.Println("Connecting to Web Server @",destinationAddress)
        webConn, err := net.Dial("tcp", destinationAddress)
        if err != nil {
          fmt.Println("Failed to connect to web server")
          return
        }

        
        // Send success reply downstream
        sucessMessage := createSocksReply(0, conn.LocalAddr())
        sendMessage(conn, NewDataWrap(connID,uint16(len(sucessMessage)),uint16(len(sucessMessage))+dataWrapHeaderSize,sucessMessage))
        

        // Commence forwarding raw data on the connection
        go proxyClientPackets(webConn, conn, connID)
        go proxyWebServerPackets(webConn, connReader, connID)
       
      default:
        fmt.Println("Cannot Process Command")
  }

}


/** 
  * Forwards data incoming from the web server to the SOCKS5 client  
  */

func proxyClientPackets(webConn net.Conn, conn net.Conn, connID uint32) {
  for {
    buf := make([]byte, 1000-8)
    n, _ := webConn.Read(buf)
    buf = buf[:n]
    // Forward the data (or close indication if n==0) downstream
    sendMessage(conn, NewDataWrap(connID,uint16(n),uint16(n)+dataWrapHeaderSize,buf))

    // Connection error or EOF?
    if n == 0 {
      fmt.Println("Disconnected from Web Server")
      webConn.Close()
      return
    }
  }
}


/** 
  * Forwards data incoming from the SOCKS5 client to the web server
  */

func proxyWebServerPackets(webConn net.Conn, connReader io.Reader, connID uint32) {

  for {
    // Get the next upstream data buffer
    buf := make([]byte, 1000-8)
    messageLength, err := connReader.Read(buf)
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






/** 
  * Read an IPv4 or IPv6 address from an io.Reader and return it as a string  
  */
func readIP(r io.Reader, len int) (net.IP, error) {
  errorIP := make(net.IP, net.IPv4len)

  addr := make([]byte, len)
  _, err := io.ReadFull(r, addr)
  if err != nil {
    return errorIP, err
  }
  return net.IP(addr), nil
}


/** 
  * Extracts the address content from a SOCKS message  
  */
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

/** 
  * Creates a reply for the SOCKS5 client Request
  */
func createSocksReply(replyCode int, addr net.Addr) []byte {
  
  buf := make([]byte, 4)    // Create byte buffer to store reply message
  buf[0] = byte(5)          // Insert Version
  buf[1] = byte(replyCode)  // Insert Reply Code

  //Check if address exists
  if addr != nil {

    // Extract Address type
    tcpaddr := addr.(*net.TCPAddr)
    host4 := tcpaddr.IP.To4()
    host6 := tcpaddr.IP.To16()

    i, _ := strconv.Atoi("6789") 

    port := [2]byte{} // Create byte buffer for the port
    binary.BigEndian.PutUint16(port[:], uint16(i))//tcpaddr.Port)) 

    // Check address type
    if host4 != nil { //IPv4

      buf[3] = addrIPv4               // Insert Addres Type
      buf = append(buf, host4...)     // Add IPv6 Address
      buf = append(buf, port[:]...)   // Add Port

    } else if host6 != nil { // IPv6

      buf[3] = addrIPv6               // Insert Addres Type
      buf = append(buf, host6...)     // Add IPv6 Address 
      buf = append(buf, port[:]...)   // Add Port

    } else { // Unknown...

      fmt.Println("SOCKS: neither IPv4 nor IPv6 addr?")
      addr = nil
      buf[1] = byte(repAddressTypeNotSupported)

    }

  } else { // otherwise, attach a null IPv4 address
    buf[3] = addrIPv4
    buf = append(buf, make([]byte, 4+2)...)
  }

  // Return reply message
  return buf
 
}

