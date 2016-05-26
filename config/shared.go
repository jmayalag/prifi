package config

import (
	"github.com/dedis/crypto/nist"
	"github.com/lbarman/prifi/dcnet"
)

//used to make sure everybody has the same version of the software. must be updated manually
const LLD_PROTOCOL_VERSION = 3

const NUM_RETRY_CONNECT = 3

//sets the crypto suite used
var CryptoSuite = nist.NewAES128SHA256P256()

//sets the factory for the dcnet's cell encoder/decoder
var Factory = dcnet.SimpleCoderFactory
