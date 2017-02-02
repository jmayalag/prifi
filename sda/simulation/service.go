package main

import (
	"github.com/BurntSushi/toml"
	prifi_protocol "github.com/lbarman/prifi/sda/protocols"
	prifi_service "github.com/lbarman/prifi/sda/services"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/app"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
	"time"
)

/*
 * Defines the simulation for the service-template
 */

func init() {
	onet.SimulationRegister("PriFi", NewSimulationService)
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
	prifi_protocol.PrifiTomlConfig
	NTrustees int
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewSimulationService(config string) (onet.Simulation, error) {
	es := &SimulationService{}
	_, err := toml.Decode(config, es)
	if err != nil {
		return nil, err
	}
	return es, nil
}

// Setup creates the tree used for that simulation
func (s *SimulationService) Setup(dir string, hosts []string) (*onet.SimulationConfig, error) {
	sc := &onet.SimulationConfig{}
	s.CreateRoster(sc, hosts, 2000)
	err := s.CreateTree(sc)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

// Node can be used to initialize each node before it will be run
// by the server. Here we call the 'Node'-method of the
// SimulationBFTree structure which will load the roster- and the
// tree-structure to speed up the first round.
func (s *SimulationService) Node(config *onet.SimulationConfig) error {

	index, _ := config.Roster.Search(config.Server.ServerIdentity.ID)
	if index < 0 {
		log.Fatal("Didn't find this node in roster")
	}
	log.Lvl3("Initializing node-index", index)
	if err := s.SimulationBFTree.Node(config); err != nil {
		log.Fatal("Could not register node in SDA Tree", err)
	}

	s.RelayReportingLimit = 10 * 100
	s.SocksServerPort = 8080 + index

	//assign the roles
	roles := make(map[*network.ServerIdentity]string)
	for k, v := range config.Roster.List {
		if k == 0 {
			roles[v] = "relay"
		} else if k > 0 && k <= s.NTrustees {
			roles[v] = "trustee"
		} else {
			roles[v] = "client"
		}
		//no, we don't need clients (handled by churn.go)
	}
	group := &app.Group{Roster: config.Roster, Description: roles}

	//finds the PriFi service
	service := config.GetService(prifi_service.ServiceName).(*prifi_service.ServiceState)

	//set the config from the .toml file
	service.SetConfigFromToml(&s.PrifiTomlConfig)

	//start this node in the correct setup
	var err error
	if index == 0 {
		log.Lvl1("Initiating this node as relay")
		err = service.StartRelay(group)
	} else if index > 0 && index <= s.NTrustees {
		log.Lvl1("Initiating this node as trustee")
		err = service.StartTrustee(group)
	} else {
		log.Lvl1("Initiating this node as client")
		err = service.StartClient(group)
	}

	if err != nil {
		log.Fatal("Error instantiating this node, ", err)
	}

	return nil
}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting experiment round", round)

		//finds the PriFi service
		service := config.GetService(prifi_service.ServiceName).(*prifi_service.ServiceState)

		//the art of programming : waiting for an event
		for service.PriFiSDAProtocol == nil {
			time.Sleep(10 * time.Millisecond)
		}
		for service.PriFiSDAProtocol.ResultChannel == nil {
			time.Sleep(10 * time.Millisecond)
		}

		//block and get the result from the channel
		res := <-service.PriFiSDAProtocol.ResultChannel
		log.Error("Res is", res)

		time.Sleep(time.Second)
	}
	return nil
}
