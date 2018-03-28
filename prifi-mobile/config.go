package prifiMobile

// Configuration files read & write and related structs

import (
	"bytes"
	"github.com/BurntSushi/toml"
	prifi_protocol "github.com/lbarman/prifi/sda/protocols"
	"golang.org/x/mobile/asset"
	"gopkg.in/dedis/onet.v1/app"
	"gopkg.in/dedis/onet.v1/log"
	"sync"
)

const prifiConfigFilename = "prifi.toml"
const cothorityConfigFilename = "identity.toml"
const cothorityGroupConfigFilename = "group.toml"

var prifiConfigSingleton *prifi_protocol.PrifiTomlConfig
var cothorityConfigSingleton *app.CothorityConfig

var onceClient, onceCothority sync.Once
var globalErr error


func getPrifiConfig() (*prifi_protocol.PrifiTomlConfig, error) {
	onceClient.Do(func() {
		prifiConfigSingleton, globalErr = initPrifiConfig()
	})
	return prifiConfigSingleton, globalErr
}

func getCothorityConfig() (*app.CothorityConfig, error) {
	onceCothority.Do(func() {
		cothorityConfigSingleton, globalErr = initCothorityConfig()
	})
	return cothorityConfigSingleton, globalErr
}

// TODO: Reduce Code Duplication of both inits
func initPrifiConfig() (*prifi_protocol.PrifiTomlConfig, error) {
	tomlRawDataString, err := readTomlFromAssets(prifiConfigFilename)

	if err != nil {
		return nil, err
	}

	config := &prifi_protocol.PrifiTomlConfig{}
	_, err = toml.Decode(tomlRawDataString, config)
	if err != nil {
		log.Error("Could not parse toml file ", prifiConfigFilename)
		return nil, err
	}

	return config, nil
}

func initCothorityConfig() (*app.CothorityConfig, error) {
	tomlRawDataString, err := readTomlFromAssets(cothorityConfigFilename)

	if err != nil {
		return nil, err
	}

	config := &app.CothorityConfig{}
	_, err = toml.Decode(tomlRawDataString, config)
	if err != nil {
		log.Error("Could not parse toml file ", cothorityConfigFilename)
		return nil, err
	}

	return config, nil
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

// TODO: Read from any given paths
func readTomlFromAssets(filename string) (string, error) {
	file, err := asset.Open(filename)
	defer file.Close()

	if err != nil {
		log.Error("Could not open file ", filename)
		return "", err
	}

	tomlRawDataBuffer := new(bytes.Buffer)
	_, err = tomlRawDataBuffer.ReadFrom(file)

	if err != nil {
		log.Error("Could not read file ", prifiConfigFilename)
		return "", err
	}

	return tomlRawDataBuffer.String(), err
}
