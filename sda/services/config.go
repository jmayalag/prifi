package services

import (
	"fmt"
	prifi_net "github.com/lbarman/prifi/prifi-lib/net"
	"io/ioutil"
	"os"

	"github.com/dedis/cothority/app/lib/config"
	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/network"
	prifi_protocol "github.com/lbarman/prifi/sda/protocols"
)

var socksClientConfig *prifi_protocol.SOCKSConfig
var socksServerConfig *prifi_protocol.SOCKSConfig

//The configuration read in prifi.toml
type PrifiTomlConfig struct {
	CellSizeUp            int
	CellSizeDown          int
	RelayWindowSize       int
	RelayUseDummyDataDown bool
	RelayReportingLimit   int
	UseUDP                bool
	DoLatencyTests        bool
	SocksServerPort       int
	SocksClientPort       int
}

//Set the config, from the prifi.toml. Is called by sda/app.
func (s *ServiceState) SetConfigFromToml(config *PrifiTomlConfig) {
	log.Lvl3("Setting PriFi configuration...")
	log.Lvlf3("%+v\n", config)
	s.prifiTomlConfig = config
}

// tryLoad tries to load the configuration and updates if a configuration
// is found, else it returns an error.
func (s *ServiceState) tryLoad() error {
	configFile := s.path + "/identity.bin"
	b, err := ioutil.ReadFile(configFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Error while reading %s: %s", configFile, err)
	}
	if len(b) > 0 {
		_, msg, err := network.UnmarshalRegistered(b)
		if err != nil {
			return fmt.Errorf("Couldn't unmarshal: %s", err)
		}
		log.Lvl3("Successfully loaded")
		s.Storage = msg.(*Storage)
	}
	return nil
}

// mapIdentities reads the group configuration to assign PriFi roles
// to server addresses and returns them with the server
// identity of the relay.
func mapIdentities(group *config.Group) (*network.ServerIdentity, []*network.ServerIdentity) {
	trustees := make([]*network.ServerIdentity, 0)
	var relay *network.ServerIdentity

	// Read the description of the nodes in the config file to assign them PriFi roles.
	nodeList := group.Roster.List
	for i := 0; i < len(nodeList); i++ {
		si := nodeList[i]
		nodeDescription := group.GetDescription(si)

		if nodeDescription == "relay" {
			relay = si
		} else if nodeDescription == "trustee" {
			trustees = append(trustees, si)
		}
	}

	return relay, trustees
}
func (s *ServiceState) setConfigToPriFiProtocol(wrapper *prifi_protocol.PriFiSDAProtocol) {

	prifiParams := prifi_net.ALL_ALL_PARAMETERS{
		ClientDataOutputEnabled: true,
		DoLatencyTests:          s.prifiTomlConfig.DoLatencyTests,
		DownCellSize:            s.prifiTomlConfig.CellSizeDown,
		ForceParams:             true,
		NClients:                -1, //computer later
		NextFreeClientID:        0,
		NextFreeTrusteeID:       0,
		NTrustees:               -1, //computer later
		RelayDataOutputEnabled:  true,
		RelayReportingLimit:     s.prifiTomlConfig.RelayReportingLimit,
		RelayUseDummyDataDown:   s.prifiTomlConfig.RelayUseDummyDataDown,
		RelayWindowSize:         s.prifiTomlConfig.RelayWindowSize,
		StartNow:                false,
		UpCellSize:              s.prifiTomlConfig.CellSizeUp,
		UseUDP:                  s.prifiTomlConfig.UseUDP,
	}

	configMsg := &prifi_protocol.PriFiSDAWrapperConfig{
		ALL_ALL_PARAMETERS: prifiParams,
		Identities:         s.churnHandler.createIdentitiesMap(),
		Role:               s.role,
		ClientSideSocksConfig: socksClientConfig,
		RelaySideSocksConfig:  socksServerConfig,
	}

	wrapper.SetConfig(configMsg)

	//when PriFi-protocol (via PriFi-lib) detects a slow client, call "handleTimeout"
	wrapper.SetTimeoutHandler(s.handleTimeout)
}
