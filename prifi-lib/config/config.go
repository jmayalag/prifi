/*
Package config contains the cryptographic primitives that are used by the PriFi library.
*/
package config

import (
	"gopkg.in/dedis/kyber.v2/suites"
)

var CryptoSuite suites.Suite = suites.MustFind("Ed25519")