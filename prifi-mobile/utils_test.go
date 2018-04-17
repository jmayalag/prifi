package prifiMobile

import (
	"testing"
	"fmt"
)

func TestUtils(t *testing.T) {
	// Relay Address
	a, _ := GetRelayAddress()
	fmt.Println("Address ", a)

	_ = SetRelayAddress("111.111.111.111")
	a, _ = GetRelayAddress()
	fmt.Println("Address ", a)

	_ = SetRelayPort(1021)
	fmt.Println("Full Address ", getFullAddress())


	// Relay Port
	p, _ := GetRelayPort()
	fmt.Println("Relay Port ", p)

	_ = SetRelayPort(1000)
	p, _ = GetRelayPort()
	fmt.Println("Relay Port ", p)

	_ = SetRelayPort(7000)
	p, _ = GetRelayPort()
	fmt.Println("Full Address ", getFullAddress())

	// Relay Socks Port
	sp, _ := GetRelaySocksPort()
	fmt.Println("Relay Socks Port", sp)

	_ = SetRelaySocksPort(8000)
	sp, _ = GetRelaySocksPort()
	fmt.Println("Relay Socks Port", sp)

	_ = SetRelaySocksPort(8090)
	sp, _ = GetRelaySocksPort()
	fmt.Println("Relay Socks Port", sp)

	// Keys
	pub, _ := GetPublicKey()
	fmt.Println("Pub key", pub)

	pri, _ := GetPrivateKey()
	fmt.Println("Private key", pri)
}