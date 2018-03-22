/*
 * The package name must NOT contain underscore.
 */
package prifiMobile

import (
	"time"
	"os"
	"gopkg.in/dedis/onet.v1/log"
)

func StartClient() {
	host, group, service := startCothorityNode()

	if err := service.StartClient(group, time.Duration(0)); err != nil {
		log.Error("Could not start the prifi service:", err)
		os.Exit(1)
	}

	host.Router.AddErrorHandler(service.NetworkErrorHappened)
	host.Start()
}
