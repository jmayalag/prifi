package scheduler

import (
	"github.com/dedis/crypto/abstract"
)

//not implemented
type Scheduler interface {
	AddClientToSchedule(pk abstract.Point) error

	FinalizeClientSet() error

	RelayPerformShuffle()

	TrusteePerformShuffle()

	TrusteeValidateSchedule()

	RelayValidateSchedule()

	ClientPayloadEmbeddable(roundID int32) (int64, int64)
}
