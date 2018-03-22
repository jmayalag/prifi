package prifiMobile

import (
	"testing"
)

func TestConfig(t *testing.T) {
	p := NewPrifiMobileClientConfig()

	if p.SocksClientPort != 8090 {
		t.Error("wrong info %v", p.SocksClientPort)
	}

	c := NewCothorityConfig()

	if c.Address != "tcp://127.0.0.1:6000" {
		t.Error("wrong", c.Address)
	}

	//fmt.Println(p.parseToOriginalPrifiConfig().SocksClientPort)
}