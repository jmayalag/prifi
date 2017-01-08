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
func mapIdentities(group *config.Group) (map[string]prifi_protocol.PriFiIdentity, *network.ServerIdentity) {
	m := make(map[string]prifi_protocol.PriFiIdentity)
	var relay *network.ServerIdentity

	// Read the description of the nodes in the config file to assign them PriFi roles.
	nodeList := group.Roster.List
	for i := 0; i < len(nodeList); i++ {
		si := nodeList[i]
		nodeDescription := group.GetDescription(si)

		var id *prifi_protocol.PriFiIdentity

		if nodeDescription == "relay" {
			id = &prifi_protocol.PriFiIdentity{
				Role:     prifi_protocol.Relay,
				ID:       0,
				ServerID: si,
			}
		} else if nodeDescription == "trustee" {
			id = &prifi_protocol.PriFiIdentity{
				Role:     prifi_protocol.Trustee,
				ID:       -1,
				ServerID: si,
			}
		}

		if id != nil {
			identifier := si.Address.String() + "=" + si.Public.String()
			m[identifier] = *id
			if id.Role == prifi_protocol.Relay {
				relay = si
			}
		} else {
			log.Error("Cannot parse node description, skipping:", si)
		}

	}

	return m, relay
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
