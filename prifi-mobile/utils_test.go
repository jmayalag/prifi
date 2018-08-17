package prifiMobile

import (
	"fmt"
	"testing"
)

func TestGetterSetter(t *testing.T) {
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
}

func TestKeyGeneration(t *testing.T) {
	pub, _ := GetPublicKey()
	fmt.Println("Pub key", pub)

	pri, _ := GetPrivateKey()
	fmt.Println("Private key", pri)

	GenerateNewKeyPairAndAssign()

	pub, _ = GetPublicKey()
	fmt.Println("New Pub key", pub)

	pri, _ = GetPrivateKey()
	fmt.Println("New Private key", pri)
}

func TestGetPrifiPort(t *testing.T) {
	port, _ := GetPrifiPort()
	fmt.Println("Prifi port", port)
}

func TestGetMobileDisconnectWhenNetworkError(t *testing.T) {
	b, _ := GetMobileDisconnectWhenNetworkError()
	fmt.Println(b)

	SetMobileDisconnectWhenNetworkError(true)
	fmt.Println(GetMobileDisconnectWhenNetworkError())

	SetMobileDisconnectWhenNetworkError(false)
	fmt.Println(GetMobileDisconnectWhenNetworkError())

	SetMobileDisconnectWhenNetworkError(true)
	fmt.Println(GetMobileDisconnectWhenNetworkError())
}

func TestRelayKeys(t *testing.T) {
	k, e := GetRelayPublicKey()
	fmt.Println(k, e)

	e = SetRelayPublicKey("55873114984971f38b63f7ac1b51abdb42e773e1e42a6ef7bf76b")
	if e != nil {
		fmt.Println("invalid key")
	}

	k, e = GetRelayPublicKey()
	fmt.Println(k, e)

	e = SetRelayPublicKey("55873114984971f38b63f7ac1b51abdb42e773e1e42a6ef7bf762c131ee341db")
	if e != nil {
		fmt.Println("invalid key")
	}

	k, e = GetRelayPublicKey()
	fmt.Println(k, e)
}
