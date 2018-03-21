package prifiMobile

import (
	"golang.org/x/mobile/asset"
	"github.com/BurntSushi/toml"
	"gopkg.in/dedis/onet.v1/log"
	"bytes"
	"sync"
)

const mobileClientConfigFilename = "prifi.toml"

var prifiMobileClientConfigSingleton *PrifiMobileClientConfig
var once sync.Once

// Exposed Constructor (singleton)
func NewPrifiMobileClientConfig() *PrifiMobileClientConfig {
	once.Do(func() {
		prifiMobileClientConfigSingleton = initPrifiMobileClientConfig()
	})
	return prifiMobileClientConfigSingleton
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

func initPrifiMobileClientConfig() *PrifiMobileClientConfig {
	file, err := asset.Open(mobileClientConfigFilename)
	defer file.Close()

	if err != nil {
		log.Error("Could not open file ", mobileClientConfigFilename)
		return nil
	}

	tomlRawDataBuffer := new(bytes.Buffer)
	_, err = tomlRawDataBuffer.ReadFrom(file)

	if err != nil {
		log.Error("Could not read file ", mobileClientConfigFilename)
		return nil
	}

	config := &PrifiMobileClientConfig{}
	_, err = toml.Decode(tomlRawDataBuffer.String(), config)
	if err != nil {
		log.Error("Could not parse toml file ", mobileClientConfigFilename)
		return nil
	}

	return config
}
