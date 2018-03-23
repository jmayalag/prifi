package prifiMobile

// Configuration files read & write and related structs

import (
	"bytes"
	"github.com/BurntSushi/toml"
	prifi_protocol "github.com/lbarman/prifi/sda/protocols"
	"golang.org/x/mobile/asset"
	"gopkg.in/dedis/onet.v1/app"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
	"sync"
)

const mobileClientConfigFilename = "prifi.toml"
const cothorityConfigFilename = "identity.toml"
const cothorityGroupConfigFilename = "group.toml"

var prifiMobileClientConfigSingleton *PrifiMobileClientConfig
var cothorityConfigSingleton *CothorityConfig

var onceClient, onceCothority sync.Once

// Exposed singleton constructors (Functions with New... will become constructors that can be called by Java and ObjC)
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

// Exposed structs and their getters and setters
// Design Choice: PrifiConfig and Cothority Config can be manipulated by Mobile OS, that's why they are exposed
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

// network.Address a type that is currently not supported by gomobile (23 March 2018)
// Thus, we have a function to return address under a supported type (string)
type CothorityConfig struct {
	Public      string
	Private     string
	Address     network.Address
	Description string
}

func (c *CothorityConfig) GetAddress() string {
	return c.Address.String()
}

// Convert mobile config into original prifi config
// TODO: Hanlde more carefully. What if mobile config has less or more members
func (c *PrifiMobileClientConfig) parseToOriginalPrifiConfig() *prifi_protocol.PrifiTomlConfig {
	config := prifi_protocol.PrifiTomlConfig(*c)
	return &config
}

// TODO: Reduce Code Duplication of both inits
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

// TODO: less code duplication?
func readCothorityGroupConfig() (*app.Group, error) {
	file, err := asset.Open(cothorityGroupConfigFilename)
	defer file.Close()

	groups, err := app.ReadGroupDescToml(file)

	if err != nil {
		log.Error("Could not parse toml file ", cothorityGroupConfigFilename)
		return nil, err
	}

	if groups == nil || groups.Roster == nil || len(groups.Roster.List) == 0 {
		log.Error("No servers found in roster from ", cothorityGroupConfigFilename)
		return nil, err
	}

	return groups, nil
}

// TODO: Read form any given paths
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
