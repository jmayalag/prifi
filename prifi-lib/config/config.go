/*
Package config contains the cryptographic primitives that are used by the PriFi library.
*/
package config

import (
	"github.com/dedis/crypto/ed25519"
	"github.com/lbarman/prifi/prifi-lib/dcnet"
)

// LLD_PROTOCOL_VERSION is used to make sure everybody has the same version of the software.
// It must be updated manually.
const LLD_PROTOCOL_VERSION = 3

//CryptoSuite contains crypto suite used, here ED25519
var CryptoSuite = ed25519.NewAES128SHA256Ed25519(false) //nist.NewAES128SHA256P256()

//Factory contains the factory for the DC-net's cell encoder/decoder.
var Factory = dcnet.SimpleCoderFactory
