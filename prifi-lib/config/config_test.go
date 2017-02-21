package config

import (
	"github.com/lbarman/prifi/prifi-lib/dcnet"
	"gopkg.in/dedis/crypto.v0/abstract"
	"testing"
)

func TestConfig(t *testing.T) {

	if CryptoSuite == nil {
		t.Error("CryptoSuite can't be nil")
	}
	if Factory == nil {
		t.Error("DC-net factory can't be nil")
	}

	//cryptoSuite must be an "abstract.Suite"
	_ = CryptoSuite.(abstract.Suite)

	//and the result of Factory() must be a dcnet.CellCoder
	cellCoder := Factory()
	_ = cellCoder.(dcnet.CellCoder)

}
