package da

import (
	"github.com/scroll-tech/go-ethereum/rollup/l1"
)

type FinalizeBatch struct {
	event *l1.FinalizeBatchEvent
}

func NewFinalizeBatch(event *l1.FinalizeBatchEvent) *FinalizeBatch {
	return &FinalizeBatch{
		event: event,
	}
}

func (f *FinalizeBatch) Type() Type {
	return FinalizeBatchType
}

func (f *FinalizeBatch) L1BlockNumber() uint64 {
	return f.event.BlockNumber()
}

func (f *FinalizeBatch) BatchIndex() uint64 {
	return f.event.BatchIndex().Uint64()
}

func (f *FinalizeBatch) Event() l1.RollupEvent {
	return f.event
}

func (f *FinalizeBatch) CompareTo(other Entry) int {
	if f.BatchIndex() < other.BatchIndex() {
		return -1
	} else if f.BatchIndex() > other.BatchIndex() {
		return 1
	}
	return 0
}
