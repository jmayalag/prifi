package main

import (
	"flag"
	"strconv"
	"gopkg.in/dedis/onet.v1/log"
	"github.com/ginuerzh/gost"
)

const DEFAULT_DEBUG_LEVEL = 3
const DEFAULT_PORT = 8090

var DEBUG_LEVELS = []int{1, 2, 3, 4, 5}


// Launch a SOCKS5 server that listens to PriFi traffic and forwards all connections
func main() {

	// Command-line flags
	var debugFlag = flag.Int("debug", DEFAULT_DEBUG_LEVEL, "debug-level")
	var portFlag = flag.Int("port", DEFAULT_PORT, "port")
	flag.Parse()
	log.SetDebugVisible(*debugFlag)

	// Enable or disable Gost logger based on the ONet debug level
	if contains(DEBUG_LEVELS, *debugFlag) {
		gost.SetLogger(&gost.LogLogger{})
	}

	// Check if the port is valid
	if *portFlag <= 1024 {
		log.Lvl1("Port number below 1024. Without super-admin privileges, this server will crash.")
	}

	if *portFlag > 65535 {
		log.Fatal("Port number above 65535. Exiting...")
	}

	//starts the SOCKS exit
	port := ":" + strconv.Itoa(*portFlag)

	log.Lvl2("Starting a SOCKS5 server...")

	gostTcpListener, err := gost.TCPListener(port)

	if err != nil {
		log.Fatal("Could not listen on port", port, "error is", err)
	}

	log.Lvl1("Server listening on port " + port)

	gostServer := gost.Server{Listener: gostTcpListener}
	gostHandler := gost.SOCKS5Handler()

	gostServer.Serve(gostHandler)
}

func contains(intSlice []int, searchInt int) bool {
	for _, value := range intSlice {
		if value == searchInt {
			return true
		}
	}
	return false
}
