package dcnet

import (
	"bytes"
	"crypto/rand"
	"testing"
	"fmt"
)

func randomBytes(length int) []byte {
	b := make([]byte, length)
	_, _ = rand.Read(b)
	return b
}

func assertEqual(a, b *DCNetCipher) bool {
	if !bytes.Equal(a.equivocationProtectionTag, b.equivocationProtectionTag) {
		return false
	}
	if !bytes.Equal(a.disruptionProtectionTag, b.disruptionProtectionTag) {
		return false
	}
	if !bytes.Equal(a.payload, b.payload) {
		return false
	}
	return true
}

func TestDCNetSerialization(t *testing.T) {

	ChangeLength(10, t)
}

func ChangeLength(length int, t *testing.T) {

	a := DCNetCipher{
		equivocationProtectionTag: randomBytes(length),
		disruptionProtectionTag: nil,
		payload: nil,
	}
	if !assertEqual(&a, DCNetCipherFromBytes(a.ToBytes())) {
		t.Error("DCNetCipher could not be marshalled-unmarshalled")
		fmt.Printf("%+v\n",a)
		fmt.Printf("%+v\n",a.ToBytes())
		fmt.Printf("%+v\n",DCNetCipherFromBytes(a.ToBytes()))
	}

	a = DCNetCipher{
		equivocationProtectionTag: nil,
		disruptionProtectionTag: randomBytes(length),
		payload: nil,
	}
	if !assertEqual(&a, DCNetCipherFromBytes(a.ToBytes())) {
		t.Error("DCNetCipher could not be marshalled-unmarshalled")
		fmt.Printf("%+v\n",a)
		fmt.Printf("%+v\n",DCNetCipherFromBytes(a.ToBytes()))
	}

	a = DCNetCipher{
		equivocationProtectionTag: nil,
		disruptionProtectionTag: nil,
		payload: randomBytes(length),
	}
	if !assertEqual(&a, DCNetCipherFromBytes(a.ToBytes())) {
		t.Error("DCNetCipher could not be marshalled-unmarshalled")
		fmt.Printf("%+v\n",a)
		fmt.Printf("%+v\n",DCNetCipherFromBytes(a.ToBytes()))
	}

	a = DCNetCipher{
		equivocationProtectionTag: randomBytes(length),
		disruptionProtectionTag: randomBytes(length),
		payload: nil,
	}
	if !assertEqual(&a, DCNetCipherFromBytes(a.ToBytes())) {
		t.Error("DCNetCipher could not be marshalled-unmarshalled")
		fmt.Printf("%+v\n",a)
		fmt.Printf("%+v\n",DCNetCipherFromBytes(a.ToBytes()))
	}

	a = DCNetCipher{
		equivocationProtectionTag: randomBytes(length),
		disruptionProtectionTag: nil,
		payload: randomBytes(length),
	}
	if !assertEqual(&a, DCNetCipherFromBytes(a.ToBytes())) {
		t.Error("DCNetCipher could not be marshalled-unmarshalled")
		fmt.Printf("%+v\n",a)
		fmt.Printf("%+v\n",DCNetCipherFromBytes(a.ToBytes()))
	}

	a = DCNetCipher{
		equivocationProtectionTag: nil,
		disruptionProtectionTag: randomBytes(length),
		payload: randomBytes(length),
	}
	if !assertEqual(&a, DCNetCipherFromBytes(a.ToBytes())) {
		t.Error("DCNetCipher could not be marshalled-unmarshalled")
		fmt.Printf("%+v\n",a)
		fmt.Printf("%+v\n",DCNetCipherFromBytes(a.ToBytes()))
	}

	a = DCNetCipher{
		equivocationProtectionTag: randomBytes(length),
		disruptionProtectionTag: randomBytes(length),
		payload: randomBytes(length),
	}
	if !assertEqual(&a, DCNetCipherFromBytes(a.ToBytes())) {
		t.Error("DCNetCipher could not be marshalled-unmarshalled")
		fmt.Printf("%+v\n",a)
		fmt.Printf("%+v\n",DCNetCipherFromBytes(a.ToBytes()))
	}

	a = DCNetCipher{
		equivocationProtectionTag: nil,
		disruptionProtectionTag: nil,
		payload: nil,
	}
	if !assertEqual(&a, DCNetCipherFromBytes(a.ToBytes())) {
		t.Error("DCNetCipher could not be marshalled-unmarshalled")
		fmt.Printf("%+v\n",a)
		fmt.Printf("%+v\n",DCNetCipherFromBytes(a.ToBytes()))
	}
}