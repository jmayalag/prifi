package relay

import (
	"bytes"
	"gopkg.in/dedis/crypto.v0/random"
	"testing"
)

func genDataSlice() []byte {
	return random.Bits(100, false, random.Stream)
}

func TestBuffers(test *testing.T) {

	b := new(BufferManager)

	err := b.Init(0, 0)
	if err == nil {
		test.Error("Shouldn't be able to init a bufferManager with 0 client/trustees")
	}

	err = b.Init(1, 1)
	if err != nil {
		test.Error("Should be able to init a bufferManager")
	}

	//check the initial state
	if b.CurrentRound() != 0 {
		test.Error("BufferManager should start in round 0")
	}
	if b.HasAllCiphersForCurrentRound() {
		test.Error("BufferManager does not have all ciphers yet, but claimed so")
	}
	c, t := b.MissingCiphersForCurrentRound()
	if len(c) != 1 || len(t) != 1 {
		test.Error("BufferManager did not compute correctly the missing ciphers")
	}
	_, _, err = b.FinalizeRound()
	if err == nil {
		test.Error("BufferManager should not be able to finalize round")
	}
	if b.NumberOfBufferedCiphers(0) != 0 {
		test.Error("Number of ciphers for trustee 0 should be 0")
	}

	clientSlice := genDataSlice()
	trusteeSlice := genDataSlice()

	//add a slice from a trustee
	b.AddTrusteeCipher(0, 0, trusteeSlice)

	if b.CurrentRound() != 0 {
		test.Error("BufferManager should still be in round 0")
	}
	if b.HasAllCiphersForCurrentRound() {
		test.Error("BufferManager does not have all ciphers yet, but claimed so")
	}
	c, t = b.MissingCiphersForCurrentRound()
	if len(c) != 1 || len(t) != 0 {
		test.Error("BufferManager did not compute correctly the missing ciphers")
	}
	if b.NumberOfBufferedCiphers(0) != 1 {
		test.Error("Number of ciphers for trustee 0 should be 1")
	}
	_, _, err = b.FinalizeRound()
	if err == nil {
		test.Error("BufferManager should not be able to finalize round")
	}
	if b.NumberOfBufferedCiphers(0) != 1 {
		test.Error("Number of ciphers for trustee 0 should be 1")
	}

	//add a slice from a client
	b.AddClientCipher(0, 0, clientSlice)

	if b.CurrentRound() != 0 {
		test.Error("BufferManager should still be in round 0")
	}
	if !b.HasAllCiphersForCurrentRound() {
		test.Error("BufferManager does have all ciphers")
	}
	c, t = b.MissingCiphersForCurrentRound()
	if len(c) != 0 || len(t) != 0 {
		test.Error("BufferManager did not compute correctly the missing ciphers")
	}
	clientSlices, trusteesSlices, err := b.FinalizeRound()
	if err != nil {
		test.Error("BufferManager should be able to finalize round")
	}
	if !bytes.Equal(clientSlices[0], clientSlice) {
		test.Error("Client slice should be the same")
	}
	if !bytes.Equal(trusteesSlices[0], trusteeSlice) {
		test.Error("Trustee slice should be the same")
	}

	//post round
	if b.NumberOfBufferedCiphers(0) != 0 {
		test.Error("Number of ciphers for trustee 0 should be 0")
	}
	if b.CurrentRound() != 1 {
		test.Error("BufferManager should now be in round 1")
	}
	if b.HasAllCiphersForCurrentRound() {
		test.Error("BufferManager does not have all ciphers for round 1")
	}
	c, t = b.MissingCiphersForCurrentRound()
	if len(c) != 1 || len(t) != 1 {
		test.Error("BufferManager did not compute correctly the missing ciphers")
	}

	clientSlice1 := genDataSlice()
	trusteeSlice1 := genDataSlice()
	clientSlice2 := genDataSlice()
	trusteeSlice2 := genDataSlice()
	trusteeSlice3 := genDataSlice()

	//should refuse to add slices in the apst
	if err = b.AddClientCipher(0, 0, clientSlice1); err == nil {
		test.Error("Should refuse to add client slices in the past")
	}
	if err = b.AddTrusteeCipher(0, 0, trusteeSlice1); err == nil {
		test.Error("Should refuse to add trustee slices in the past")
	}

	b.AddClientCipher(1, 0, clientSlice1)
	b.AddClientCipher(2, 0, clientSlice2)
	b.AddTrusteeCipher(3, 0, trusteeSlice3)

	//r1[c,]
	//r2[c,]
	//r3[,t]
	//nothing should have changed
	if b.CurrentRound() != 1 {
		test.Error("BufferManager should still be in round 1")
	}
	if b.HasAllCiphersForCurrentRound() {
		test.Error("BufferManager does not have all ciphers for round 1")
	}
	c, t = b.MissingCiphersForCurrentRound()
	if len(c) != 0 || len(t) != 1 {
		test.Error("BufferManager did not compute correctly the missing ciphers")
	}

	b.AddClientCipher(1, 0, clientSlice1)

	//r1[c,]
	//r2[c,]
	//r3[,t]
	//nothing should have changed if we re-add cipher 1 for client 0
	if b.CurrentRound() != 1 {
		test.Error("BufferManager should still be in round 1")
	}
	if b.HasAllCiphersForCurrentRound() {
		test.Error("BufferManager does not have all ciphers for round 1")
	}
	c, t = b.MissingCiphersForCurrentRound()
	if len(c) != 0 || len(t) != 1 {
		test.Error("BufferManager did not compute correctly the missing ciphers")
	}

	//r1[c,t]
	//r2[c,]
	//r3[,t]
	//add one trustee
	b.AddTrusteeCipher(1, 0, trusteeSlice1)
	clientSlices, trusteesSlices, err = b.FinalizeRound()
	if err != nil {
		test.Error("Should be able to finalize without error")
	}
	if !bytes.Equal(clientSlices[0], clientSlice1) {
		test.Error("Client slice should be the same")
	}
	if !bytes.Equal(trusteesSlices[0], trusteeSlice1) {
		test.Error("Trustee slice should be the same")
	}

	//r2[c,]
	//r3[,t]
	//we should be in round 2, but already have the cipher for the client
	if b.CurrentRound() != 2 {
		test.Error("BufferManager should now be in round 2")
	}
	if b.HasAllCiphersForCurrentRound() {
		test.Error("BufferManager does not have all ciphers for round 2")
	}
	c, t = b.MissingCiphersForCurrentRound()
	if len(c) != 0 || len(t) != 1 {
		test.Error("BufferManager did not compute correctly the missing ciphers")
	}

	//r2[c,t]
	//r3[,t]
	//add one trustee
	b.AddTrusteeCipher(2, 0, trusteeSlice2)
	clientSlices, trusteesSlices, err = b.FinalizeRound()
	if err != nil {
		test.Error("Should be able to finalize without error")
	}
	if !bytes.Equal(clientSlices[0], clientSlice2) {
		test.Error("Client slice should be the same")
	}
	if !bytes.Equal(trusteesSlices[0], trusteeSlice2) {
		test.Error("Trustee slice should be the same")
	}

	//r3[,t]
	//we should be in round 3 with one trustee
	if b.CurrentRound() != 3 {
		test.Error("BufferManager should now be in round 3")
	}
	if b.HasAllCiphersForCurrentRound() {
		test.Error("BufferManager does not have all ciphers for round 3")
	}
	c, t = b.MissingCiphersForCurrentRound()
	if len(c) != 1 || len(t) != 0 {
		test.Error("BufferManager did not compute correctly the missing ciphers")
	}

	//should both fail on nil slice
	err = b.AddClientCipher(0, 0, nil)
	if err == nil {
		test.Error("Shouldn't be able to add a nil slice")
	}
	err = b.AddTrusteeCipher(0, 0, nil)
	if err == nil {
		test.Error("Shouldn't be able to add a nil slice")
	}
}

func TestRateLimiter(test *testing.T) {

	b := new(BufferManager)
	low := 1
	high := 3

	stopCalled := false
	resumeCalled := false

	stopFn := func(int) {
		stopCalled = true
	}
	resFn := func(int) {
		resumeCalled = true
	}

	b.Init(0, 1)
	b.AddRateLimiter(low, high, stopFn, resFn)
	trusteeSlice := genDataSlice()

	b.AddTrusteeCipher(0, 0, trusteeSlice)
	if !resumeCalled {
		test.Error("Resume should have been called")
	}
	resumeCalled = false

	b.AddTrusteeCipher(1, 0, trusteeSlice)
	b.AddTrusteeCipher(2, 0, trusteeSlice)
	b.AddTrusteeCipher(3, 0, trusteeSlice)
	if !stopCalled {
		test.Error("Stop should have been called")
	}
	stopCalled = false
	b.AddTrusteeCipher(4, 0, trusteeSlice)
	if stopCalled {
		test.Error("Stop not should have been called again")
	}
	stopCalled = false

	b.AddClientCipher(0, 0, trusteeSlice)
	b.FinalizeRound()
	if stopCalled {
		test.Error("Stop not should have been called again (2)")
	}
	stopCalled = false

	b.AddClientCipher(1, 0, trusteeSlice)
	b.FinalizeRound()
	if stopCalled {
		test.Error("Stop not should have been called again (3)")
	}
	stopCalled = false
	b.AddClientCipher(2, 0, trusteeSlice)
	b.FinalizeRound()
	if stopCalled {
		test.Error("Stop should not have been called")
	}
	b.AddClientCipher(3, 0, trusteeSlice)
	b.FinalizeRound()

	if !resumeCalled {
		test.Error("Resume should have been called")
	}
}
