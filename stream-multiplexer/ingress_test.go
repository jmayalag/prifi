package stream_multiplexer

import (
	"errors"
	"gopkg.in/dedis/onet.v1/log"
	"testing"
)


func TestMultiplexer(t *testing.T) {

	port := 3000
	payloadLength := 20
	upstreamChan := make(chan []byte)
	downstreamChan := make(chan []byte)
	stopChan := make(chan bool)

	go StartIngressServer(port, payloadLength, upstreamChan, downstreamChan, stopChan)

	
}