package scheduler

/*

The interface should be :

INPUT : list of client's public keys

OUTPUT : list of slots

*/

import (
	"github.com/dedis/crypto/abstract"
)

type Scheduler interface {
	AddClientToSchedule(pk abstract.Point) error

	FinalizeClientSet() error

	RelayPerformShuffle()

	TrusteePerformShuffle()

	TrusteeValidateSchedule()

	RelayValidateSchedule()

	ClientPayloadEmbeddable(roundID int32) (int64, int64)
}
