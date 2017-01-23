package services

import (
	"testing"

	"github.com/dedis/onet/log"
)

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestServiceTemplate(t *testing.T) {

	/*
		local := sda.NewLocalTest()
		defer local.CloseAll()
		hosts, roster, _ := local.MakeHELS(5, serviceID)
		log.Lvl1("Roster is", roster)

		var services []sda.Service
		for _, h := range hosts {
			service := local.Services[h.ServerIdentity.ID][serviceID].(sda.Service)
			services = append(services, service)
		}

		services[0].StartTrustee()
		services[1].StartTrustee()
		services[2].StartRelay()
		services[3].StartClient()
		services[4].StartClient()
	*/
}
