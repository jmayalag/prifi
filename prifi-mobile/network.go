package prifiMobile

import (
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/app"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/crypto"
	"gopkg.in/dedis/onet.v1/network"
	prifi_service "github.com/lbarman/prifi/sda/services"
)

func startCothorityNode() (*onet.Server, *app.Group, *prifi_service.ServiceState) {
	host, err := parseCothority()
	if err != nil {
		log.Error("Could not start cothority")
		return nil, nil, nil
	}

	//reads the group description
	group := readCothorityGroupConfig()
	if err != nil {
		log.Error("Could not read the group description:", err)
		return nil, nil, nil
	}

	config := NewPrifiMobileClientConfig().parseToOriginalPrifiConfig()
	service := host.Service(prifi_service.ServiceName).(*prifi_service.ServiceState)
	service.SetConfigFromToml(config)
	config.ProtocolVersion = "v1" //getGitCommitID()

	return host, group, service
}

func parseCothority() (*onet.Server, error) {
	c := NewCothorityConfig()

	secret, err := crypto.StringHexToScalar(network.Suite, c.Private)
	if err != nil {
		return nil, err
	}
	point, err := crypto.StringHexToPoint(network.Suite, c.Public)
	if err != nil {
		return nil, err
	}
	si := network.NewServerIdentity(point, c.Address)
	si.Description = c.Description
	server := onet.NewServerTCP(si, secret)
	return server, nil
}


