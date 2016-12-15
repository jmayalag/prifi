/*
Prifi-app starts a cothority node in either trustee, relay or client mode.
*/
package main

import (
	"fmt"
	"os"

	"github.com/dedis/cothority/log"
	"github.com/lbarman/prifi_dev/sda/services"
	"gopkg.in/urfave/cli.v1"
	"github.com/dedis/cothority/app/lib/server"
	"github.com/dedis/cothority/sda"
	"path"
	"runtime"
	"io/ioutil"
	"github.com/BurntSushi/toml"
	"os/user"
	"github.com/dedis/cothority/app/lib/config"
)

// DefaultName is the name of the binary we produce and is used to create a directory
// folder with this name
const DefaultName = "prifi"

// Default name of configuration file
const DefaultCothorityConfigFile = "config.toml"

// Default name of group file
const DefaultCothorityGroupConfigFile = "group.toml"

// Default name of prifi's config file
const DefaultPriFiConfigFile = "group.toml"

// This app can launch the prifi service in either client, trustee or relay mode
func main() {
	app := cli.NewApp()
	app.Name = "prifi"
	app.Usage = "Starts PriFi in either Trustee, Relay or Client mode."
	app.Version = "0.1"
	app.Commands = []cli.Command{
		{
			Name:    "setup",
			Aliases: []string{"s"},
			Usage:   "setup the configuration for the server",
			Action:  setupNewCothorityNode,
		},
		{
			Name:    "trustee",
			Usage:   "start in trustee mode",
			Aliases: []string{"t"},
			Action:  startTrustee,
		},
		{
			Name:      "relay",
			Usage:     "start in relay mode",
			ArgsUsage: "group [id-name]",
			Aliases:   []string{"r"},
			Action:    startRelay,
		},
		{
			Name:    "client",
			Usage:   "start in client mode",
			Aliases: []string{"c"},
			Action:  startClient,
		},
		{
			Name:    "sockstest",
			Usage:   "only starts the socks server and the socks clients without prifi",
			Aliases: []string{"socks"},
			Action:  startSocksTunnelOnly,
		},
	}
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "debug, d",
			Value: 0,
			Usage: "debug-level: 1 for terse, 5 for maximal",
		},
		cli.StringFlag{
			Name:  "cothority_config, cc",
			Value: getDefaultFilePathForName(DefaultCothorityConfigFile),
			Usage: "configuration-file",
		},
		cli.StringFlag{
			Name:  "prifi_config, pc",
			Value: getDefaultFilePathForName(DefaultPriFiConfigFile),
			Usage: "configuration-file",
		},
		cli.IntFlag{
			Name:  "port, p",
			Value: -1, // will depend on if we run a relay or client
			Usage: "port for the socks handler",
		},
		cli.StringFlag{
			Name:  "group, g",
			Value: getDefaultFilePathForName(DefaultCothorityGroupConfigFile),
			Usage: "Group file",
		},
		cli.BoolFlag{
			Name:  "nowait",
			Usage: "Return immediately",
		},
	}
	app.Before = func(c *cli.Context) error {
		log.SetDebugVisible(c.Int("debug"))
		return nil
	}
	app.Run(os.Args)
}

/**
 * Every "app" require reading config files and starting cothority beforehand
 */
func readConfigAndStartCothority(c *cli.Context) (*sda.Conode, *config.Group, *services.Service) {
	//parse PriFi parameters
	prifiConfig, err := readPriFiConfigFile(c)
	fmt.Println(prifiConfig)
	if err != nil {
		log.Error("Could not read prifi config:", err)
		os.Exit(1)
	}

	//start cothority server
	host, err := startCothorityNode(c)
	if err != nil {
		log.Error("Could not start Cothority server:", err)
		os.Exit(1)
	}

	//finds the PriFi service
	service := host.GetService(services.ServiceName).(*services.Service)

	//reads the group description
	group := readCothorityGroupConfig(c)
	if err != nil {
		log.Error("Could not read the group description:", err)
		os.Exit(1)
	}

	return host, group, service
}

// trustee start the cothority in trustee-mode using the already stored configuration.
func startTrustee(c *cli.Context) error {
	log.Info("Starting trustee")

	host, group, service := readConfigAndStartCothority(c)

	if err := service.StartTrustee(group); err != nil {
		log.Error("Could not start the prifi service:", err)
		os.Exit(1)
	}

	host.Start()
	return nil
}

// relay starts the cothority in relay-mode using the already stored configuration.
func startRelay(c *cli.Context) error {
	log.Info("Starting relay")

	host, group, service := readConfigAndStartCothority(c)

	if err := service.StartRelay(group); err != nil {
		log.Error("Could not start the prifi service:", err)
		os.Exit(1)
	}

	host.Start()
	return nil
}

// client starts the cothority in client-mode using the already stored configuration.
func startClient(c *cli.Context) error {
	log.Info("Starting client")

	host, group, service := readConfigAndStartCothority(c)

	if err := service.StartClient(group); err != nil {
		log.Error("Could not start the prifi service:", err)
		os.Exit(1)
	}

	host.Start()
	return nil
}

// this is used to test the socks server and clients integrated to PriFi, without using DC-nets.
func startSocksTunnelOnly(c *cli.Context) error {
	log.Info("Starting socks tunnel (bypassing PriFi)")

	host, _, service := readConfigAndStartCothority(c)

	if err := service.StartSocksTunnelOnly(); err != nil {
		log.Error("Could not start the prifi service:", err)
		os.Exit(1)
	}

	host.Start()
	return nil
}

/**
 * COTHORITY
 */


// setupCothorityd sets up a new cothority node configuration (used by all prifi modes)
func setupNewCothorityNode(c *cli.Context) error {
	server.InteractiveConfig("cothorityd")
	return nil
}

// Starts the cothority node to enable communication with the prifi-service.
func startCothorityNode(c *cli.Context) (*sda.Conode, error) {
	// first check the options
	cfile := c.GlobalString("cothority_config")

	if _, err := os.Stat(cfile); os.IsNotExist(err) {
		log.Error("Could not open file \"", cfile, "\" (specified by flag cothority_config)")
		return nil, err
	}
	// Let's read the config
	_, host, err := config.ParseCothorityd(cfile)
	if err != nil {
		log.Error("Could not parse file", cfile)
		return nil, err
	}
	return host, nil
}

/**
 * CONFIG
 */

type PriFiToml struct {
	CellSizeUp            int
	CellSizeDown          int
	RelayWindowSize       int
	RelayUseDummyDataDown bool
	RelayReportingLimit   int
	UseUDP                bool
	DoLatencyTests        bool
}

func readPriFiConfigFile(c *cli.Context) (*PriFiToml, error) {

	cfile := c.GlobalString("prifi_config")

	if _, err := os.Stat(cfile); os.IsNotExist(err) {
		log.Error("Could not open file \"", cfile, "\" (specified by flag prifi_config)")
		return nil, err
	}

	tomlRawData, err := ioutil.ReadFile(cfile)
	if err != nil {
		log.Error("Could not read file \"", cfile, "\" (specified by flag prifi_config)")
	}

	tomlConfig := &PriFiToml{}
	_, err = toml.Decode(string(tomlRawData), tomlConfig)
	if err != nil {
		log.Error("Could not parse toml file", cfile)
		return nil, err
	}

	fmt.Print(tomlConfig)

	return tomlConfig, nil
}


// getDefaultFile creates a path to the default config folder and appends fileName to it.
func getDefaultFilePathForName(fileName string) string {
	u, err := user.Current()
	// can't get the user dir, so fallback to current working dir
	if err != nil {
		fmt.Print("[-] Could not get your home's directory. Switching back to current dir.")
		if curr, err := os.Getwd(); err != nil {
			log.Fatalf("Impossible to get the current directory. %v", err)
		} else {
			return path.Join(curr, fileName)
		}
	}
	// let's try to stick to usual OS folders
	switch runtime.GOOS {
	case "darwin":
		return path.Join(u.HomeDir, "Library", DefaultName, fileName)
	default:
		return path.Join(u.HomeDir, ".config", DefaultName, fileName)
	// TODO WIndows ? FreeBSD ?
	}
}

// getGroup reads the group-file and returns it.
func readCothorityGroupConfig(c *cli.Context) *config.Group {

	gfile := c.GlobalString("group")

	if _, err := os.Stat(gfile); os.IsNotExist(err) {
		log.Error("Could not open file \"", gfile, "\" (specified by flag group)")
		return nil
	}

	gr, err := os.Open(gfile)

	if err != nil {
		log.Error("Could not read file \"", gfile, "\"")
		return nil
	}

	defer gr.Close()

	groups, err := config.ReadGroupDescToml(gr)

	if err != nil {
		log.Error("Could not parse toml file \"", gfile, "\"")
		return nil
	}

	if groups == nil || groups.Roster == nil || len(groups.Roster.List) == 0 {
		log.Error("No servers found in roster from", gfile)
		return nil
	}
	return groups
}