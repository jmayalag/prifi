/*
Package config contains the cryptographic primitives that are used by the PriFi library.
*/
package config

import (
	"gopkg.in/dedis/kyber.v2"
	"gopkg.in/dedis/kyber.v2/suites"
)

var CryptoSuite Suite

type Suite struct {
	kyber.Group
	kyber.HashFactory
	kyber.XOF
}

func Init() {
	CryptoSuite = suites.MustFind("Ed25519")
}