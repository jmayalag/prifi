package prifi

//TODO : BIG, BIG todo here. we should automate some tests !

import (
	"testing"

	"github.com/dedis/cothority/log"
)

func TestPrifi(t *testing.T) {
	log.TestOutput(testing.Verbose(), 4)

	log.Lvl1("Testing PriFi protocol...")

	nbrHosts := 2
	local := sda.NewLocalTest()
	hosts, el, tree := local.GenBigTree(nbrHosts, nbrHosts, 3, true, true)
	p, err := local.CreateProtocol("PriFi", tree)

	//var client1 *ClientState = initClient(0, 1, 1, 1000, false, false, false)

}
