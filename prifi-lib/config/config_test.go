package config

import (
	"gopkg.in/dedis/crypto.v0/abstract"
	"testing"
)

func TestConfig(t *testing.T) {

	if CryptoSuite == nil {
		t.Error("CryptoSuite can't be nil")
	}

	//cryptoSuite must be an "abstract.Suite"
	_ = CryptoSuite.(abstract.Suite)

}
