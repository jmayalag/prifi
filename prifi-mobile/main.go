/*
 * The package name must NOT contain underscore.
 */
package prifiMobile

import (
	"gopkg.in/dedis/onet.v1/log"
	"time"
)

var stop chan struct{}

// The "main" function that is called by Mobile OS in order to launch a client server
func StartClient() error {
	stop = make(chan struct{})

	host, group, service, err := startCothorityNode()

	if err != nil {
		log.Error("Could not start the cothority node:", err)
		return err
	}

	if err := service.StartClient(group, time.Duration(0)); err != nil {
		log.Error("Could not start the prifi service:", err)
		return err
	}

	host.Router.AddErrorHandler(service.NetworkErrorHappened)

	select {
		case <-stop:
			host.Close()
			service.ShutdownSocks()
	}

	host.Start()

	return nil
}

func StopClient() {
	close(stop)
}
