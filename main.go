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

	// Roles
	isLogSink         := flag.Bool("logsink", false, "Start log sink node")
	nodeName          := flag.String("nodeName", "", "Node name")
	useSocksProxy     := flag.Bool("socks", true, "Starts a useSocksProxy proxy for the client")
	isConfig          := flag.Bool("config", false, "Configure the system")

	// Parameters config
	nClients           := flag.Int("nclients", 1, "The number of clients.")
	nTrustees          := flag.Int("ntrustees", 1, "The number of trustees.")
	upstreamCellSize   := flag.Int("upcellsize", 30720, "Sets the size of one upstream cell, in bytes.")
	downstreamCellSize := flag.Int("downcellsize", 30720, "Sets the size of one downstream cell, in bytes.")
	latencyTest        := flag.Bool("latencytest", true, "Makes the client run a latency test. Disables the SOCKS proxy.")
	useUdp             := flag.Bool("udp", true, "Improves performances by adding UDP broadcast from relay to clients")
	windowSize         := flag.Int("window", 1, "The size of the relay's window")

	// Logging stuff
	logLevel          := flag.Int("loglvl", prifilog.INFORMATION, "The minimum level of logs to display.")
	logType           := flag.String("logtype", "file", "Choices : file, or netlogger.")
	netLogPort		  := flag.String("logport", ":10305", "The network port of the log server")
	netLogHost		  := flag.String("loghost", "localhost:10305", "The network host+port of the log server")
	netLogStdOut	  := flag.Bool("logtostdout", true, "If the log is also copied to stdout")
	logPath			  := flag.String("logpath", "", "The path to the folder of the log files, by default empty, i.e. current directory")

	// Relay parameters
	relayPort         := flag.Int("relayport", 9876, "Sets listening port of the relay, waiting for clients.")
	relayHostAddr     := flag.String("relayhostaddr", "localhost:9876", "The address of the relay, for the client to contact.")
	relayReceiveLimit := flag.Int("reportlimit", -1, "Sets the limit of cells to receive before stopping the relay")
	relayDummyDown    := flag.Bool("relaydummydown", false, "The relays sends dummy data down, instead of empty cells.")

	// Trustees host
	trustee1Host      := flag.String("t1host", "localhost:9000", "The Ip address of the 1st trustee, or localhost")
	trustee2Host      := flag.String("t2host", "localhost", "The Ip address of the 2nd trustee, or localhost")
	trustee3Host      := flag.String("t3host", "localhost", "The Ip address of the 3rd trustee, or localhost")
	trustee4Host      := flag.String("t4host", "localhost", "The Ip address of the 4th trustee, or localhost")
	trustee5Host      := flag.String("t5host", "localhost", "The Ip address of the 5th trustee, or localhost")

	flag.Parse()
	trusteesIp := []string{*trustee1Host, *trustee2Host, *trustee3Host, *trustee4Host, *trustee5Host}

	var nodeConfig config.NodeConfig
	if !*isConfig {

		if *nodeName == "" {
			println("Error: Must specify -config or -nodeName=[name of the node]")
			os.Exit(1)
		}

		// Read node's configuration from file
		nodeConfig = config.NodeConfig{}
		if err := nodeConfig.Load(*nodeName); err != nil {
			fmt.Println("Error: Cannot load configuration file for " + *nodeName + ". " + err.Error())
			os.Exit(1)
		}
		fmt.Println("Node configuration loaded successfully. Node name: " + nodeConfig.Name + ", Node ID: " + strconv.Itoa(nodeConfig.Id) + ", Pub ID: " + nodeConfig.PubId)
	}

	// Hard reset
	*useUdp = false

	// Starts the LOG sink server
	if *isLogSink {
		prifilog.StartSinkServer(*netLogPort, *logPath + "sink.log")
	}

	if *latencyTest {
		*useSocksProxy = false
	}

	// Set up the log - default is a file
	if *logType == "netlogger" {
		prifilog.SetUpNetworkLogEngine(*logLevel, *nodeName, *netLogHost, *netLogStdOut)
	} else {
		logFilename := *nodeName + ".log"
		prifilog.SetUpFileLogEngine(*logLevel, *logPath + logFilename, *netLogStdOut)
	}

	if(*nTrustees > 5) {
		fmt.Println("Only up to 5 trustees are supported for this prototype") //only limited because of the input parameters
		os.Exit(1)
	}

	relayPortAddr := ":" + strconv.Itoa(*relayPort)	// NOT "localhost:xxxx", or it will not listen on any interfaces

	switch {
	case *isConfig:
		if err := config.GenerateConfig(*nClients, *nTrustees, config.CryptoSuite); err != nil {
            fmt.Println("Error: Configuration generation failed. " + err.Error())
            os.Exit(1)
        }
		confDir, _ := config.ConfigDir("")
		fmt.Println("Configurations generated successfully at", confDir)

	case nodeConfig.Type == config.NODE_TYPE_TRUSTEE:
		trustee.StartTrustee(nodeConfig)

	case nodeConfig.Type == config.NODE_TYPE_RELAY:
		relay.StartRelay(nodeConfig, *upstreamCellSize, *downstreamCellSize, *windowSize, *relayDummyDown,
			relayPortAddr, *nClients, *nTrustees, trusteesIp, *relayReceiveLimit, *useUdp)

	case nodeConfig.Type == config.NODE_TYPE_CLIENT:
		client.StartClient(nodeConfig, *relayHostAddr, *nClients, *nTrustees, *upstreamCellSize, *useSocksProxy, *latencyTest, *useUdp)
	}
}