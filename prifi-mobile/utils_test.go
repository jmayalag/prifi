package prifiMobile

import (
	"testing"
	"fmt"
)

func TestUtils(t *testing.T) {
	a, _ := GetRelayAddress()
	fmt.Println(a)

	p, _ := GetRelayPort()
	fmt.Println(p)

	sp, _ := GetRelaySocksPort()
	fmt.Println(sp)

	pub, _ := GetPublicKey()
	fmt.Println(pub)

	pri, _ := GetPrivateKey()
	fmt.Println(pri)
}