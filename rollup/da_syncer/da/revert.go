package da

type RevertBatch struct {
	batchIndex uint64

	l1BlockNumber uint64
}

func NewRevertBatch(batchIndex uint64) *RevertBatch {
	return &RevertBatch{
		batchIndex: batchIndex,
	}
}

func (r *RevertBatch) Type() Type {
	return RevertBatchType
}

func (r *RevertBatch) L1BlockNumber() uint64 {
	return r.l1BlockNumber
}
func (r *RevertBatch) BatchIndex() uint64 {
	return r.batchIndex
}

func (r *RevertBatch) CompareTo(other Entry) int {
	if r.BatchIndex() < other.BatchIndex() {
		return -1
	} else if r.BatchIndex() > other.BatchIndex() {
		return 1
	}
	return 0
}
