/*
Package config contains the cryptographic primitives that are used by the PriFi library.
*/
package config

import (
	//"github.com/lbarman/prifi/prifi-lib/dcnet"
	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/crypto.v0/ed25519"
)

//CryptoSuite contains crypto suite used, here ED25519
var CryptoSuite abstract.Suite

//Factory contains the factory for the DC-net's cell encoder/decoder.
//var Factory func() dcnet.CellCoder

func init() {
	CryptoSuite = ed25519.NewAES128SHA256Ed25519(false) //nist.NewAES128SHA256P256()
	//Factory = dcnet.SimpleCoderFactory
	//Factory = dcnet.OwnedCoderFactory
}
