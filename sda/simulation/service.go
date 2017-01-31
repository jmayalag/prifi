package main

import (
	"github.com/BurntSushi/toml"
	prifi_service "github.com/lbarman/prifi/sda/services"
	prifi_protocol "github.com/lbarman/prifi/sda/protocols"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/simul/monitor"
	"time"
	"github.com/dedis/cothority/services"
	"os"
	"io/ioutil"
)

/*
 * Defines the simulation for the service-template
 */

func init() {
	log.Error("Called")
	onet.SimulationRegister("PriFi", NewSimulationService)
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
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

var prifiService *prifi_service.ServiceState

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

	//read the config
	prifiTomlConfig, err := readPriFiConfigFile("filepath.txt")
	if err != nil {
		log.Fatal("Could not read PriFi config", err)
	}

	//finds the PriFi service
	service := config.GetService(prifi_service.ServiceName).(*prifi_service.ServiceState)

	//set the config from the .toml file
	service.SetConfigFromToml(prifiTomlConfig)


	switch index{
	case 0:
		log.Lvl1("Intiating this node as relay")
		prifiService = service.StartRelay( /* need app.Group here */ )

	}

	/*
	host.Router.AddErrorHandler(service.NetworkErrorHappened)
	host.Start()
	 */
	return nil
}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
	size := config.Tree.Size()
	log.Lvl2("Size is:", size, "rounds:", s.Rounds)
	service, ok := config.GetService("PriFi").(*services.ServiceState)
	if service == nil || !ok {
		log.Fatal("Didn't find service PriFi")
	}
	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)
		round := monitor.NewTimeMeasure("round")

		/*
			ret, err := service.ClockRequest(&template.ClockRequest{Roster: config.Roster})
			if err != nil {
				log.Error(err)
			}
			resp, ok := ret.(*template.ClockResponse)$
		*/
		time.Sleep(time.Second)

		if !ok {
			log.Fatal("Didn't get a ClockResponse")
		}
		round.Record()
	}
	return nil
}


func readPriFiConfigFile(filePath string) (*prifi_protocol.PrifiTomlConfig, error) {

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Error("Could not open file \"", filePath, "\" (specified by flag prifi_config)")
		return nil, err
	}

	tomlRawData, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Error("Could not read file \"", filePath, "\" (specified by flag prifi_config)")
	}

	tomlConfig := &prifi_protocol.PrifiTomlConfig{}
	_, err = toml.Decode(string(tomlRawData), tomlConfig)
	if err != nil {
		log.Error("Could not parse toml file", filePath)
		return nil, err
	}

	return tomlConfig, nil
}