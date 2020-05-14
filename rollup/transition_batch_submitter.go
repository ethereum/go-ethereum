package rollup

type RollupTransitionBatchSubmitter interface {
	submit(block *TransitionBatch) error
}

type TransitionBatchSubmitter struct{}

func NewBlockSubmitter() *TransitionBatchSubmitter {
	return &TransitionBatchSubmitter{}
}
func (d *TransitionBatchSubmitter) submit(block *TransitionBatch) error {
	return nil
}
