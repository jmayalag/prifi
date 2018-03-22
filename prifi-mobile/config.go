package prifiMobile

import (
	"golang.org/x/mobile/asset"
	"github.com/BurntSushi/toml"
	"gopkg.in/dedis/onet.v1/log"
	"bytes"
	"sync"
	"gopkg.in/dedis/onet.v1/network"
	prifi_protocol "github.com/lbarman/prifi/sda/protocols"
	"gopkg.in/dedis/onet.v1/app"
)

const mobileClientConfigFilename = "prifi.toml"
const cothorityConfigFilename = "identity.toml"
const cothorityGroupConfigFilename = "group.toml"

var prifiMobileClientConfigSingleton *PrifiMobileClientConfig
var cothorityConfigSingleton *CothorityConfig

var onceClient, onceCothority sync.Once


// Exposed Constructor (singleton)
func NewPrifiMobileClientConfig() *PrifiMobileClientConfig {
	onceClient.Do(func() {
		prifiMobileClientConfigSingleton = initPrifiMobileClientConfig()
	})
	return prifiMobileClientConfigSingleton
}

func NewCothorityConfig() *CothorityConfig {
	onceCothority.Do(func() {
		cothorityConfigSingleton = initCothorityConfig()
	})
	return cothorityConfigSingleton
}

// Exposed Getters and Setters
type PrifiMobileClientConfig struct {
	ForceConsoleColor                       bool
	OverrideLogLevel                        int
	ClientDataOutputEnabled                 bool
	RelayDataOutputEnabled                  bool
	CellSizeUp                              int
	CellSizeDown                            int
	RelayWindowSize                         int
	RelayUseOpenClosedSlots                 bool
	RelayUseDummyDataDown                   bool
	RelayReportingLimit                     int
	UseUDP                                  bool
	DoLatencyTests                          bool
	SocksServerPort                         int
	SocksClientPort                         int
	ProtocolVersion                         string
	DCNetType                               string
	ReplayPCAP                              bool
	PCAPFolder                              string
	TrusteeSleepTimeBetweenMessages         int
	TrusteeAlwaysSlowDown                   bool
	TrusteeNeverSlowDown                    bool
	SimulDelayBetweenClients                int
	DisruptionProtectionEnabled             bool
	EquivocationProtectionEnabled           bool
	OpenClosedSlotsMinDelayBetweenRequests  int
	RelayMaxNumberOfConsecutiveFailedRounds int
	RelayProcessingLoopSleepTime            int
	RelayRoundTimeOut                       int
	RelayTrusteeCacheLowBound               int
	RelayTrusteeCacheHighBound              int
}

type CothorityConfig struct {
	Public      string
	Private     string
	Address     network.Address
	Description string
}

func (c *CothorityConfig) GetAddress() string {
	return c.Address.String()
}

// TODO: Hanlde more carefully
func (c *PrifiMobileClientConfig) parseToOriginalPrifiConfig() *prifi_protocol.PrifiTomlConfig {
	config := prifi_protocol.PrifiTomlConfig(*c)
	return &config
}

// TODO: Reduce Code Duplication
func initPrifiMobileClientConfig() *PrifiMobileClientConfig {
	tomlRawDataString := readTomlFromAssets(mobileClientConfigFilename)

	config := &PrifiMobileClientConfig{}
	_, err := toml.Decode(tomlRawDataString, config)
	if err != nil {
		log.Error("Could not parse toml file ", mobileClientConfigFilename)
		return nil
	}

	return config
}

func initCothorityConfig() *CothorityConfig {
	tomlRawDataString := readTomlFromAssets(cothorityConfigFilename)

	config := &CothorityConfig{}
	_, err := toml.Decode(tomlRawDataString, config)
	if err != nil {
		log.Error("Could not parse toml file ", cothorityConfigFilename)
		return nil
	}

	return config
}

func readCothorityGroupConfig() *app.Group {
	file, err := asset.Open(cothorityGroupConfigFilename)
	defer file.Close()

	groups, err := app.ReadGroupDescToml(file)

	if err != nil {
		log.Error("Could not parse toml file ", cothorityGroupConfigFilename)
		return nil
	}

	if groups == nil || groups.Roster == nil || len(groups.Roster.List) == 0 {
		log.Error("No servers found in roster from ", cothorityGroupConfigFilename)
		return nil
	}

	return groups
}

func readTomlFromAssets(filename string) string {
	file, err := asset.Open(filename)
	defer file.Close()

	if err != nil {
		log.Error("Could not open file ", filename)
		return ""
	}

	tomlRawDataBuffer := new(bytes.Buffer)
	_, err = tomlRawDataBuffer.ReadFrom(file)

	if err != nil {
		log.Error("Could not read file ", mobileClientConfigFilename)
		return ""
	}

	return tomlRawDataBuffer.String()
}
