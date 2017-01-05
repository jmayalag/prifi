package scheduler

import (
	"github.com/dedis/cothority/network"
	"github.com/dedis/crypto/abstract"
	cryptoconfig "github.com/dedis/crypto/config"
	"github.com/lbarman/prifi/prifi-lib"
	"testing"
	"strconv"
	"fmt"
)

type PrivatePublicPair struct {
	Private abstract.Scalar
	Public abstract.Point
}

func TestNeff(t *testing.T) {

	nTrustees := 2
	nClients := 2

	clients := make([]*PrivatePublicPair, nClients)
	for i:=0; i<nClients; i++ {
		clientI := cryptoconfig.NewKeyPair(network.Suite)
		clients[i] = new(PrivatePublicPair)
		clients[i].Public = clientI.Public
		clients[i].Private = clientI.Secret
	}

	//create the scheduler
	n := new(neffShuffleScheduler) //this will hold 1 relay, 1 trustee at most. Recreate n for >1 trustee
	n.init()

	//init the trustees
	trustees := make([]*neffShuffleScheduler, nTrustees)
	for i:=0; i<nTrustees; i++{
		trustees[i] = new(neffShuffleScheduler)
		trustees[i].init()
		trustee := cryptoconfig.NewKeyPair(network.Suite)
		trustees[i].TrusteeView.init(i, trustee.Secret, trustee.Public)
	}

	//init the relay
	n.RelayView.init(nTrustees)
	for i:=0; i<nClients; i++ {
		n.RelayView.AddClient(clients[i].Public)
	}

	isDone := false
	i := 0
	for !isDone {
		if i >= nTrustees {
			t.Error("Should only shuffle" + strconv.Itoa(nTrustees) +", but we did one more loop")
		}

		//the relay send the shuffle send it to the next trustee
		err, toSend := n.RelayView.SendToNextTrustee()
		if err != nil {
			t.Error(err)
		}
		parsed := toSend.(*prifi_lib.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE)

		//who receives it
		err, toSend2 := trustees[i].TrusteeView.ReceivedShuffleFromRelay(parsed.Base, parsed.Pks)
		if err != nil {
			t.Error(err)
		}
		parsed2 := toSend2.(*prifi_lib.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS)

		//and answers
		err, isDone = n.RelayView.ReceivedShuffleFromTrustee(parsed2.NewBase, parsed2.NewEphPks, parsed2.Proof)

		i ++
	}

	err, toSend3 := n.RelayView.SendTranscript()
	if err != nil {
		t.Error(err)
	}
	parsed3 := toSend3.(*prifi_lib.REL_TRU_TELL_TRANSCRIPT)

	for j := 0; j<nTrustees; j++ {
		err, toSend4 := trustees[j].TrusteeView.ReceivedTranscriptFromRelay(parsed3.Gs, parsed3.EphPks, parsed3.Proofs)
		if err != nil {
			t.Error(err)
		}
		parsed4 := toSend4.(*prifi_lib.TRU_REL_SHUFFLE_SIG)

		err, done := n.RelayView.ReceivedSignatureFromTrustee(parsed4.TrusteeID, parsed4.Sig)

		if done && j != nTrustees - 1 {
			t.Error("Relay collecting signature, but is done too early, only received "+strconv.Itoa(j+1)+" signatures out of "+strconv.Itoa(nTrustees))
		}
		if !done && j == nTrustees - 1 {
			t.Error("Relay collecting signature, but is not done, yet we have all signatures")
		}
	}

	trusteesPks := make([]abstract.Point, nTrustees)
	for j := 0; j<nTrustees; j++ {
		trusteesPks[j] = trustees[j].TrusteeView.PublicKey
	}

	err, toSend5 := n.RelayView.VerifySigsAndSendToClients(trusteesPks)
	if err != nil {
		t.Error(err)
	}
	parsed5 := toSend5.(*prifi_lib.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG)

	//client verify the sig and recognize their slot
	for j := 0; j<nClients; j++ {
		err, mySlot := n.ClientVerifySigAndRecognizeSlot(clients[j].Private, trusteesPks, parsed5.Base, parsed5.EphPks, parsed5.TrusteesSigs)
		if err != nil {
			t.Error(err)
		}
		fmt.Println("Client", j, "got assigned slot", mySlot)
	}

}
