/*
Prifi-app starts a cothority node in either trustee, relay or client mode.
*/
package main

import (
	"fmt"
	"github.com/dedis/cothority/app/lib/config"
	"github.com/dedis/cothority/app/lib/server"
	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/sda"
	"github.com/lbarman/prifi_dev/sda/services"
	"gopkg.in/urfave/cli.v1"
	"os"
	"os/user"
	"path"
	"runtime"
)

// DefaultName is the name of the binary we produce and is used to create a directory
// folder with this name
const DefaultName = "prifi"

// Default name of configuration file
const DefaultServerConfig = "config.toml"

// Default name of group file
const DefaultGroupConfig = "group.toml"

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
			Action:  setupCothorityd,
		},
		{
			Name:    "trustee",
			Usage:   "start in trustee mode",
			Aliases: []string{"t"},
			Action:  trustee,
		},
		{
			Name:      "relay",
			Usage:     "start in relay mode",
			ArgsUsage: "group [id-name]",
			Aliases:   []string{"r"},
			Action:    relay,
		},
		{
			Name:    "client",
			Usage:   "start in client mode",
			Aliases: []string{"c"},
			Action:  client,
		},
	}
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "debug, d",
			Value: 0,
			Usage: "debug-level: 1 for terse, 5 for maximal",
		},
		cli.StringFlag{
			Name:  "config, c",
			Value: getDefaultFile(DefaultServerConfig),
			Usage: "configuration-file",
		},
		cli.StringFlag{
			Name:  "group, g",
			Value: getDefaultFile(DefaultGroupConfig),
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

// trustee start the cothority in trustee-mode using the already stored configuration.
func trustee(c *cli.Context) error {
	log.Info("Starting trustee")
	host, err := cothorityd(c)
	log.ErrFatal(err)
	service := host.GetService(prifi.ServiceName).(*prifi.Service)
	// Do other setups
	log.ErrFatal(service.StartTrustee())

	host.Start()
	return nil
}

// relay starts the cothority in relay-mode using the already stored configuration.
func relay(c *cli.Context) error {
	log.Info("Starting relay")
	host, err := cothorityd(c)
	log.ErrFatal(err)
	service := host.GetService(prifi.ServiceName).(*prifi.Service)
	// Do other setups
	group := getGroup(c)
	log.ErrFatal(service.StartRelay(group))

	host.Start()
	return nil
}

// client starts the cothority in client-mode using the already stored configuration.
func client(c *cli.Context) error {
	log.Info("Starting client")
	host, err := cothorityd(c)
	log.ErrFatal(err)
	service := host.GetService(prifi.ServiceName).(*prifi.Service)
	// Do other setups
	log.ErrFatal(service.StartClient())

	host.Start()
	return nil
}

// setupCothorityd sets up a new cothority node configuration (used by all prifi modes)
func setupCothorityd(c *cli.Context) error {
	server.InteractiveConfig(DefaultName)
	return nil
}

// Starts the cothority node to enable communication with the prifi-service.
// Returns the prifi-service.
func cothorityd(c *cli.Context) (*sda.Conode, error) {
	// first check the options
	cfile := c.GlobalString("config")

	if _, err := os.Stat(cfile); os.IsNotExist(err) {
		return nil, err
	}
	// Let's read the config
	_, host, err := config.ParseCothorityd(cfile)
	if err != nil {
		return nil, err
	}
	return host, nil
}

// getDefaultFile creates a path to the default config folder and appends fileName to it.
func getDefaultFile(fileName string) string {
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
func getGroup(c *cli.Context) *config.Group {
	gfile := c.GlobalString("group")
	gr, err := os.Open(gfile)
	log.ErrFatal(err)
	defer gr.Close()
	groups, err := config.ReadGroupDescToml(gr)
	log.ErrFatal(err)
	if groups == nil || groups.Roster == nil || len(groups.Roster.List) == 0 {
		log.Fatal("No servers found in roster from", gfile)
	}
	return groups
}
