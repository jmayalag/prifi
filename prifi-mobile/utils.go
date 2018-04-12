package prifiMobile

import "strconv"

func GetRelayAddress() (string, error) {
	c, err := getGroupConfig()
	relayAddress := c.Roster.Get(0).Address.Host()
	return relayAddress, err
}

func GetRelaySocksPort() (int, error) {
	c, err := getPrifiConfig()
	return c.SocksClientPort, err
}

func GetRelayPort() (int, error) {
	c, err := getGroupConfig()
	portString := c.Roster.Get(0).Address.Port()
	port, _ := strconv.Atoi(portString)
	return port, err
}

func GetPublicKey() (string, error) {
	c, err := getCothorityConfig()
	return c.Public, err
}

func GetPrivateKey() (string, error) {
	c, err := getCothorityConfig()
	return c.Private, err
}