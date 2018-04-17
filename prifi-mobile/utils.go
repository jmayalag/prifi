package prifiMobile

import (
	"strconv"
	"gopkg.in/dedis/onet.v1/network"
	"errors"
)

const relayIndex = 0
const separatorHostPort = ":"

// Relay Address
func GetRelayAddress() (string, error) {
	c, err := getGroupConfig()
	relayAddress := c.Roster.Get(relayIndex).Address.Host()
	return relayAddress, err
}

func SetRelayAddress(host string) error {
	c, err := getGroupConfig()
	if err != nil {
		return err
	}

	port := c.Roster.Get(relayIndex).Address.Port()
	fullAddress := network.NewAddress(network.PlainTCP, host + separatorHostPort + port)
	if fullAddress.Valid() {
		c.Roster.Get(relayIndex).Address = fullAddress
		return nil
	} else {
		return errors.New("not a host:port address")
	}
}


// Relay Port
func GetRelayPort() (int, error) {
	c, err := getGroupConfig()
	portString := c.Roster.Get(relayIndex).Address.Port()
	port, _ := strconv.Atoi(portString)
	return port, err
}

func SetRelayPort(port int) error {
	c, err := getGroupConfig()
	if err != nil {
		return err
	}

	relayAddress := c.Roster.Get(relayIndex).Address.Host()
	newPort := strconv.Itoa(port)
	fullAddress := network.NewAddress(network.PlainTCP, relayAddress + separatorHostPort + newPort)
	if fullAddress.Valid() {
		c.Roster.Get(relayIndex).Address = fullAddress
		return nil
	} else {
		return errors.New("not a host:port address")
	}
}


// Relay Socks Port
func GetRelaySocksPort() (int, error) {
	c, err := getPrifiConfig()
	return c.SocksClientPort, err
}

func SetRelaySocksPort(port int) error {
	c, err := getPrifiConfig()
	if err != nil {
		return err
	}
	c.SocksClientPort = port
	return nil
}


// Keys
func GetPublicKey() (string, error) {
	c, err := getCothorityConfig()
	return c.Public, err
}

func GetPrivateKey() (string, error) {
	c, err := getCothorityConfig()
	return c.Private, err
}


// Support functions
func getFullAddress() (string) {
	c, _ := getGroupConfig()
	return c.Roster.Get(relayIndex).Address.String()
}
