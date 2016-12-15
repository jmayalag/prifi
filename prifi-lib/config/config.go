/*
Package config contains the cryptographic primitives that are used by the PriFi library.
*/
package config

import (
	"github.com/dedis/crypto/config"
	"github.com/dedis/crypto/ed25519"
	"github.com/dedis/crypto/suites"
	"github.com/lbarman/prifi_dev/prifi-lib/dcnet"
)

// LLD_PROTOCOL_VERSION is used to make sure everybody has the same version of the software.
// It must be updated manually.
const LLD_PROTOCOL_VERSION = 3

//sets the crypto suite used
var CryptoSuite = ed25519.NewAES128SHA256Ed25519(false) //nist.NewAES128SHA256P256()

//Factory contains the factory for the DC-net's cell encoder/decoder.
var Factory = dcnet.SimpleCoderFactory

var configFile config.File

// ConfigData is Dissent config file format
type ConfigData struct {
	Keys config.Keys // Info on configured key-pairs
}

var configData ConfigData
var keyPairs []config.KeyPair

// ReadConfig reads a configuration form a config file.
func ReadConfig() error {

	// Load the configuration file
	configFile.Load("dissent", &configData)

	// Read or create our public/private keypairs
	pairs, err := configFile.Keys(&configData.Keys, suites.All(), CryptoSuite)
	if err != nil {
		return err
	}
	keyPairs = pairs
	println("Loaded", len(pairs), "key-pairs")

	return nil
}
