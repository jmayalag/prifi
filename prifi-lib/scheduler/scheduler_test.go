package scheduler

import (
	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/network"
	cryptoconfig "github.com/dedis/crypto/config"
	"github.com/lbarman/prifi/prifi-lib"
	"testing"
)

func TestDoUser(t *testing.T) {

	nTrustees := 1
	userA := cryptoconfig.NewKeyPair(network.Suite)
	userB := cryptoconfig.NewKeyPair(network.Suite)

	//create the scheduler
	n := new(neffShuffleScheduler)
	n.init()

	//the relay receives the keys
	n.AddClient(userA.Public)
	n.AddClient(userB.Public)

	for i :=0; i < nTrustees; i++ {

		//send it to the trustee
		toSend := n.SendToFirstTrustee(nTrustees).(*prifi_lib.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE)
		log.Lvlf1("%+v\n", toSend)

		//who receives it
		toSend2 := n.ReceivedShuffleFromRelay(toSend.Base, toSend.Pks).(*prifi_lib.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS)
		log.Lvlf1("%+v\n", toSend2)

		//and answers
		toSend3 := n.ReceivedShuffleFromTrustee(toSend2.NewBase, toSend2.NewEphPks, toSend2.Proof)
		log.Lvlf1("%+v\n", toSend3)
	}

	//the relay sends it to the clients

	t.Log("Done")
}
