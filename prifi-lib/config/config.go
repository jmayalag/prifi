/*
Package config contains the cryptographic primitives that are used by the PriFi library.
*/
package config

import (
	"gopkg.in/dedis/kyber.v2/suites"
)

// the suite used in the prifi-lib
var CryptoSuite = suites.MustFind("Ed25519")
