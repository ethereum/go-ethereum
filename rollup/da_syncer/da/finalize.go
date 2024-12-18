package da

type FinalizeBatch struct {
	batchIndex uint64

	l1BlockNumber uint64
}

func NewFinalizeBatch(batchIndex uint64) *FinalizeBatch {
	return &FinalizeBatch{
		batchIndex: batchIndex,
	}
}

func (f *FinalizeBatch) Type() Type {
	return FinalizeBatchType
}

func (f *FinalizeBatch) L1BlockNumber() uint64 {
	return f.l1BlockNumber
}

func (f *FinalizeBatch) BatchIndex() uint64 {
	return f.batchIndex
}

func (f *FinalizeBatch) CompareTo(other Entry) int {
	if f.BatchIndex() < other.BatchIndex() {
		return -1
	} else if f.BatchIndex() > other.BatchIndex() {
		return 1
	}
	return 0
}
