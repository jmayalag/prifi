package prifiMobile

import "testing"

func TestConfig(t *testing.T) {
	c := NewPrifiMobileClientConfig()

	if c.SocksClientPort != 8090 {
		t.Error("wrong info %v", c.SocksClientPort)
	}
}