package dcnet

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"
)

func randomBytes(length int) []byte {
	b := make([]byte, length)
	_, _ = rand.Read(b)
	return b
}

func assertEqual(a, b *DCNetCipher) bool {
	if !bytes.Equal(a.EquivocationProtectionTag, b.EquivocationProtectionTag) {
		return false
	}
	if !bytes.Equal(a.Payload, b.Payload) {
		return false
	}
	return true
}

func TestDCNetSerialization(t *testing.T) {
	ChangeLength(10, t)
	ChangeLength(20, t)
	ChangeLength(50, t)
	ChangeLength(100, t)
	ChangeLength(1000, t)
}

func ChangeLength(length int, t *testing.T) {

	a := DCNetCipher{
		EquivocationProtectionTag: randomBytes(length),
		Payload:                   nil,
	}
	if !assertEqual(&a, DCNetCipherFromBytes(a.ToBytes())) {
		t.Error("DCNetCipher could not be marshalled-unmarshalled")
		fmt.Printf("%+v\n", a)
		fmt.Printf("%+v\n", a.ToBytes())
		fmt.Printf("%+v\n", DCNetCipherFromBytes(a.ToBytes()))
	}

	a = DCNetCipher{
		EquivocationProtectionTag: nil,
		Payload:                   nil,
	}
	if !assertEqual(&a, DCNetCipherFromBytes(a.ToBytes())) {
		t.Error("DCNetCipher could not be marshalled-unmarshalled")
		fmt.Printf("%+v\n", a)
		fmt.Printf("%+v\n", DCNetCipherFromBytes(a.ToBytes()))
	}

	a = DCNetCipher{
		EquivocationProtectionTag: nil,
		Payload:                   randomBytes(length),
	}
	if !assertEqual(&a, DCNetCipherFromBytes(a.ToBytes())) {
		t.Error("DCNetCipher could not be marshalled-unmarshalled")
		fmt.Printf("%+v\n", a)
		fmt.Printf("%+v\n", DCNetCipherFromBytes(a.ToBytes()))
	}

	a = DCNetCipher{
		EquivocationProtectionTag: randomBytes(length),
		Payload:                   randomBytes(length),
	}
	if !assertEqual(&a, DCNetCipherFromBytes(a.ToBytes())) {
		t.Error("DCNetCipher could not be marshalled-unmarshalled")
		fmt.Printf("%+v\n", a)
		fmt.Printf("%+v\n", DCNetCipherFromBytes(a.ToBytes()))
	}
}
