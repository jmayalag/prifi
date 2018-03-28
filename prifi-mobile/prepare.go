package prifiMobile

// Functions that are needed to initialize a server are all here

import (
	prifi_service "github.com/lbarman/prifi/sda/services"
	prifi_protocol "github.com/lbarman/prifi/sda/protocols"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/app"
	"gopkg.in/dedis/onet.v1/crypto"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
)

func startCothorityNode() (*onet.Server, *app.Group, *prifi_service.ServiceState, error) {
	prifiConfig, err := parsePrifi()
	if err != nil {
		log.Error("Could not read prifi config")
		return nil, nil, nil, err
	}

	host, err := parseCothority()
	if err != nil {
		log.Error("Could not start cothority")
		return nil, nil, nil, err
	}

	group, err := readCothorityGroupConfig()
	if err != nil {
		log.Error("Could not read the group description:", err)
		return nil, nil, nil, err
	}

	service := host.Service(prifi_service.ServiceName).(*prifi_service.ServiceState)
	service.SetConfigFromToml(prifiConfig)

	// TODO Replace getCommitID
	prifiConfig.ProtocolVersion = "v1" // standard string for all nodes


	return host, group, service, nil
}

func parsePrifi() (*prifi_protocol.PrifiTomlConfig, error) {
	c, err := getPrifiConfig()
	if err != nil {
		return nil, err
	}

	if c.OverrideLogLevel > 0 {
		log.Lvl3("Log level set to", c.OverrideLogLevel)
		log.SetDebugVisible(c.OverrideLogLevel)
	}

	return c, nil
}

func parseCothority() (*onet.Server, error) {
	c, err := getCothorityConfig()
	if err != nil {
		return nil, err
	}

	secret, err := crypto.StringHexToScalar(network.Suite, c.Private)
	if err != nil {
		return nil, err
	}
	point, err := crypto.StringHexToPoint(network.Suite, c.Public)
	if err != nil {
		return nil, err
	}

	serverIdentity := network.NewServerIdentity(point, c.Address)
	serverIdentity.Description = c.Description
	server := onet.NewServerTCP(serverIdentity, secret)
	return server, nil
}
