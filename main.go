package main

import (
	"flag"
	"fmt"
	"strconv"
	"os"
	"os/signal"
	"github.com/lbarman/prifi/client"
	"github.com/lbarman/prifi/relay"
	"github.com/lbarman/prifi/trustee"
	"github.com/lbarman/prifi/config"
	log2 "github.com/lbarman/prifi/log"
)

func interceptCtrlC() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			panic("signal: " + sig.String()) // with stacktrace
		}
	}()
}

func main() {
	interceptCtrlC()

	log2.StringDump("New run...")

	//roles...
	isRelay           := flag.Bool("relay", false, "Start relay node")
	clientId          := flag.Int("client", -1, "Start client node")
	useSocksProxy     := flag.Bool("socks", true, "Starts a useSocksProxy proxy for the client")
	isTrusteeServer   := flag.Bool("trusteesrv", false, "Start a trustee server")

	//parameters config
	nClients          := flag.Int("nclients", 1, "The number of clients.")
	nTrustees         := flag.Int("ntrustees", 1, "The number of trustees.")
	cellSize          := flag.Int("cellsize", 128, "Sets the size of one cell, in bytes.")
	relayPort         := flag.Int("relayport", 9876, "Sets listening port of the relay, waiting for clients.")
	relayReceiveLimit := flag.Int("reportlimit", -1, "Sets the limit of cells to receive before stopping the relay")
	trustee1Host      := flag.String("t1host", "localhost", "The Ip address of the 1st trustee, or localhost")
	trustee2Host      := flag.String("t2host", "localhost", "The Ip address of the 2nd trustee, or localhost")
	trustee3Host      := flag.String("t3host", "localhost", "The Ip address of the 3rd trustee, or localhost")
	trustee4Host      := flag.String("t4host", "localhost", "The Ip address of the 4th trustee, or localhost")
	trustee5Host      := flag.String("t5host", "localhost", "The Ip address of the 5th trustee, or localhost")

	flag.Parse()
	trusteesIp := []string{*trustee1Host, *trustee2Host, *trustee3Host, *trustee4Host, *trustee5Host}
	
	config.ReadConfig()

	//exception
	if(*nTrustees > 5) {
		fmt.Println("Only up to 5 trustees are supported")
		os.Exit(1)
	}

	relayPortAddr := "localhost:"+strconv.Itoa(*relayPort)

	if *isRelay {
		relay.StartRelay(*cellSize, relayPortAddr, *nClients, *nTrustees, trusteesIp, *relayReceiveLimit)
	} else if *clientId >= 0 {
		client.StartClient(*clientId, relayPortAddr, *nClients, *nTrustees, *cellSize, *useSocksProxy)
	} else if *isTrusteeServer {
		trustee.StartTrusteeServer()
	} else {
		println("Error: must specify -relay, -trusteesrv, -client=n, or -trustee=n")
	}
}