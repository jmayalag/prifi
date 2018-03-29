/*
 * The package name must NOT contain underscore.
 */
package prifiMobile

import (
	prifi_service "github.com/lbarman/prifi/sda/services"
	"time"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1"
)

var stopChan chan bool
var errorChan chan error
var globalHost *onet.Server
var globalService *prifi_service.ServiceState

// The "main" function that is called by Mobile OS in order to launch a client server
func StartClient() {
	stopChan = make(chan bool, 1)
	errorChan = make(chan error, 1)

	go func() {
		errorChan <- run()
	}()


	select {
		case err := <-errorChan:
			log.Error("Error occurs", err)
		case <-stopChan:
			globalHost.Close()
			globalService.ShutdownSocks()
			globalService.ShutdownConnexionToRelay()
			log.Info("PriFi Shutdown")
	}
}

func StopClient() {
	stopChan <- true
}

func run() error {
	host, group, service, err := startCothorityNode()
	globalHost = host
	globalService = service

	if err != nil {
		log.Error("Could not start the cothority node:", err)
		return err
	}

	if err := service.StartClient(group, time.Duration(0)); err != nil {
		log.Error("Could not start the PriFi service:", err)
		return err
	}

	host.Router.AddErrorHandler(service.NetworkErrorHappened)
	host.Start()

	// Never return
	return nil
}
