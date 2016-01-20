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
	prifilog "github.com/lbarman/prifi/log"
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

	//roles
	isLogSink         := flag.Bool("logsink", false, "Start log sink node")
	isRelay           := flag.Bool("relay", false, "Start relay node")
	clientId          := flag.Int("client", -1, "Start client node")
	useSocksProxy     := flag.Bool("socks", true, "Starts a useSocksProxy proxy for the client")
	isTrusteeServer   := flag.Bool("trusteesrv", false, "Start a trustee server")

	//parameters config
	nClients           := flag.Int("nclients", 1, "The number of clients.")
	nTrustees          := flag.Int("ntrustees", 1, "The number of trustees.")
	upstreamCellSize   := flag.Int("upcellsize", 30720, "Sets the size of one upstream cell, in bytes.")
	downstreamCellSize := flag.Int("downcellsize", 30720, "Sets the size of one downstream cell, in bytes.")
	latencyTest        := flag.Bool("latencytest", true, "Makes the client run a latency test. Disables the SOCKS proxy.")
	useUdp             := flag.Bool("udp", true, "Improves performances by adding UDP broadcast from relay to clients")

	//logging stuff
	logLevel          := flag.Int("loglvl", prifilog.INFORMATION, "The minimum level of logs to display.")
	logType           := flag.String("logtype", "file", "Choices : file, or netlogger.")
	netLogPort		  := flag.String("logport", ":10305", "The network port of the log server")
	netLogHost		  := flag.String("loghost", "localhost:10305", "The network host+port of the log server")
	netLogStdOut	  := flag.Bool("logtostdout", true, "If the log is also copied to stdout")
	logPath			  := flag.String("logpath", "", "The path to the folder of the log files, by default empty, i.e. current directory")

	//relay parameters
	relayPort         := flag.Int("relayport", 9876, "Sets listening port of the relay, waiting for clients.")
	relayHostAddr     := flag.String("relayhostaddr", "localhost:9876", "The address of the relay, for the client to contact.")
	relayReceiveLimit := flag.Int("reportlimit", -1, "Sets the limit of cells to receive before stopping the relay")
	relayDummyDown    := flag.Bool("relaydummydown", false, "The relays sends dummy data down, instead of empty cells.")

	//trustees host
	trustee1Host      := flag.String("t1host", "localhost", "The Ip address of the 1st trustee, or localhost")
	trustee2Host      := flag.String("t2host", "localhost", "The Ip address of the 2nd trustee, or localhost")
	trustee3Host      := flag.String("t3host", "localhost", "The Ip address of the 3rd trustee, or localhost")
	trustee4Host      := flag.String("t4host", "localhost", "The Ip address of the 4th trustee, or localhost")
	trustee5Host      := flag.String("t5host", "localhost", "The Ip address of the 5th trustee, or localhost")

	flag.Parse()
	trusteesIp := []string{*trustee1Host, *trustee2Host, *trustee3Host, *trustee4Host, *trustee5Host}
	
	config.ReadConfig()

	//starts the LOG sink server
	if *isLogSink {
		prifilog.StartSinkServer(*netLogPort, *logPath+"sink.log")
	}

	if *latencyTest {
		*useSocksProxy = false
	}

	//set up the log - default is a file
	if *logType == "netlogger" {
		var entity string
		if *isRelay {
			entity = "relay"
		} else if *clientId >= 0 {
			entity = "client"+strconv.Itoa(*clientId)
		} else if *isTrusteeServer {
			entity = "trusteeServer"
		}else{
			entity = "unknown"
		}
		prifilog.SetUpNetworkLogEngine(*logLevel, entity, *netLogHost, *netLogStdOut)
	}else{
		var logFile string
		if *isRelay {
			logFile = "relay.log"
		} else if *clientId >= 0 {
			logFile = "client"+strconv.Itoa(*clientId)+".log"
		} else if *isTrusteeServer {
			logFile = "trusteeServer.log"
		}else{
			logFile = "dissent.log"
		}
		prifilog.SetUpFileLogEngine(*logLevel, *logPath+logFile, *netLogStdOut)
	}

	//exception
	if(*nTrustees > 5) {
		fmt.Println("Only up to 5 trustees are supported for this prototype") //only limited because of the input parameters
		os.Exit(1)
	}

	relayPortAddr := ":"+strconv.Itoa(*relayPort) //NOT "localhost:xxxx", or it will not listen on any interfaces

	if *isRelay {
		relay.StartRelay(*upstreamCellSize, *downstreamCellSize, *relayDummyDown, relayPortAddr, *nClients, *nTrustees, trusteesIp, *relayReceiveLimit, *useUdp)
	} else if *clientId >= 0 {
		client.StartClient(*clientId, *relayHostAddr, *nClients, *nTrustees, *upstreamCellSize, *downstreamCellSize, *useSocksProxy, *latencyTest, *useUdp)
	} else if *isTrusteeServer {
		trustee.StartTrusteeServer()
	} else {
		println("Error: must specify -relay, -trusteesrv, -client=n, or -trustee=n")
	}
}