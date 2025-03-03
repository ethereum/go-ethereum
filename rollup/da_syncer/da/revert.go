package da

import (
	"github.com/scroll-tech/go-ethereum/rollup/l1"
)

type RevertBatch struct {
	event l1.RollupEvent
}

func NewRevertBatch(event l1.RollupEvent) *RevertBatch {
	return &RevertBatch{
		event: event,
	}
}

func (r *RevertBatch) Type() Type {
	return RevertBatchType
}

func (r *RevertBatch) L1BlockNumber() uint64 {
	return r.event.BlockNumber()
}

func (r *RevertBatch) BatchIndex() uint64 {
	return r.event.BatchIndex().Uint64()
}

func (r *RevertBatch) Event() l1.RollupEvent {
	return r.event
}

func (r *RevertBatch) CompareTo(other Entry) int {
	if r.BatchIndex() < other.BatchIndex() {
		return -1
	} else if r.BatchIndex() > other.BatchIndex() {
		return 1
	}
	return 0
}
