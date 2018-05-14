package main

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"gopkg.in/dedis/kyber.v2"
	"gopkg.in/dedis/kyber.v2/util/key"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// HostsMappingFile is the file used to indicate the mapping host -> IP
const HostsMappingFile = "hosts_mapping.toml"

// SimulationManualAssignment is a second implementation of SimulationBFTree, but we change the method CreateRoster
type SimulationManualAssignment struct {
	Rounds     int
	BF         int
	Hosts      int
	SingleHost bool
	Depth      int
}

// HostMapping contains a mapping of ID (0 to n_hosts) and IP on which they need to run
type HostMapping struct {
	ID int
	IP string
}

// HostsMappingToml is used to parse the .toml
type HostsMappingToml struct {
	Hosts []*HostMapping `toml:"hosts"`
}

// decodeHostsMapping reads the .toml file
func decodeHostsMapping(filePath string) (*HostsMappingToml, error) {

	f, err := os.Open(filePath)
	if err != nil {
		e := fmt.Sprint("Could not read file \"", filePath, "\"")
		log.Error(e)
		return nil, errors.New(e)
	}

	defer f.Close()

	hosts := &HostsMappingToml{}
	_, err = toml.DecodeReader(f, hosts)
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

// CreateRoster creates an Roster with the host-names in 'addresses'.
// It creates 's.Hosts' entries, starting from 'port' for each round through
// 'addresses'. The network.Address(es) created are of type PlainTCP.
func (s *SimulationManualAssignment) CreateRoster(sc *onet.SimulationConfig, addresses []string, port int) {
	start := time.Now()
	nbrAddr := len(addresses)
	if sc.PrivateKeys == nil {
		sc.PrivateKeys = make(map[network.Address]kyber.Scalar)
	}
	hosts := s.Hosts
	if s.SingleHost {
		// If we want to work with a single host, we only make one
		// host per server
		log.Fatal("Not supported yet")
		hosts = nbrAddr
		if hosts > s.Hosts {
			hosts = s.Hosts
		}
	}
	localhosts := false
	listeners := make([]net.Listener, hosts)
	services := make([]net.Listener, hosts)
	if /*strings.Contains(addresses[0], "localhost") || */ strings.Contains(addresses[0], "127.0.0.") {
		localhosts = true
	}
	entities := make([]*network.ServerIdentity, hosts)
	log.Lvl3("Doing", hosts, "hosts")
	key := key.NewKeyPair(network.Suite)

	//replaces linus automatic assignement by the one read in hosts_mapping.toml
	mapping, err := decodeHostsMapping(HostsMappingFile)
	if err != nil {
		log.Fatal("Could not decode " + HostsMappingFile)
	}

	//prepare the ports mapping
	portsMapping := make(map[string]int, 0)
	for _, hostMapping := range mapping.Hosts {
		portsMapping[hostMapping.IP] = port
	}

	for c := 0; c < hosts; c++ {
		key.Secret.Add(key.Secret, key.Suite.Scalar().One())
		key.Public.Add(key.Public, key.Suite.Point().Base())

		address := ""
		for _, hostMapping := range mapping.Hosts {
			if hostMapping.ID == c {
				address = hostMapping.IP
			}
		}

		if address == "" {
			log.Fatal("Host index", c, "not specified in hosts_mapping.toml")
		}

		var add network.Address
		if localhosts {
			// If we have localhosts, we have to search for an empty port
			port := 0
			for port == 0 {

				var err error
				listeners[c], err = net.Listen("tcp", ":0")
				if err != nil {
					log.Fatal("Couldn't search for empty port:", err)
				}
				_, p, _ := net.SplitHostPort(listeners[c].Addr().String())
				port, _ = strconv.Atoi(p)
				services[c], err = net.Listen("tcp", ":"+strconv.Itoa(port+1))
				if err != nil {
					port = 0
				}
			}
			address += ":" + strconv.Itoa(port)
			add = network.NewTCPAddress(address)
			log.Lvl4("Found free port", address)
		} else {
			nextFreePort := portsMapping[address]
			portsMapping[address] += 2
			addressAndPort := address + ":" + strconv.Itoa(nextFreePort)
			add = network.NewTCPAddress(addressAndPort)
		}
		log.Lvl3("Adding server", address, "to Roster")
		entities[c] = network.NewServerIdentity(key.Public.Clone(), add)
		sc.PrivateKeys[entities[c].Address] = key.Secret.Clone()
	}
	if hosts > 1 {
		if sc.PrivateKeys[entities[0].Address].Equal(
			sc.PrivateKeys[entities[1].Address]) {
			log.Fatal("Something went terribly wrong.")
		}
	}

	// And close all our listeners
	if localhosts {
		for _, l := range listeners {
			err := l.Close()
			if err != nil {
				log.Fatal("Couldn't close port:", l, err)
			}
		}
		for _, l := range services {
			err := l.Close()
			if err != nil {
				log.Fatal("Couldn't close port:", l, err)
			}
		}
	}

	sc.Roster = onet.NewRoster(entities)
	log.Lvl3("Creating entity List took: " + time.Now().Sub(start).String())
}

// CreateTree the tree as defined in SimulationBFTree and stores the result
// in 'sc'
func (s *SimulationManualAssignment) CreateTree(sc *onet.SimulationConfig) error {
	log.Lvl3("CreateTree started")
	start := time.Now()
	if sc.Roster == nil {
		return errors.New("Empty Roster")
	}
	sc.Tree = sc.Roster.GenerateBigNaryTree(s.BF, s.Hosts)
	log.Lvl3("Creating tree took: " + time.Now().Sub(start).String())
	return nil
}

// Node - standard registers the entityList and the Tree with that Overlay,
// so we don't have to pass that around for the experiments.
func (s *SimulationManualAssignment) Node(sc *onet.SimulationConfig) error {
	sc.Overlay.RegisterRoster(sc.Roster)
	sc.Overlay.RegisterTree(sc.Tree)
	return nil
}
