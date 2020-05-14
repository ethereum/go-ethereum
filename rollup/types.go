package rollup

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

const (
	MinTxBytes               = uint64(100)
	MinTxGas                 = MinTxBytes*params.TxDataNonZeroGasEIP2028 + params.SstoreSetGas
	TransitionBatchGasBuffer = uint64(1_000_000)
)

type BlockStore interface {
	GetBlockByNumber(number uint64) *types.Block
}

type Transition struct {
	transaction *types.Transaction
	postState   common.Hash
}

func newTransition(tx *types.Transaction, postState common.Hash) *Transition {
	return &Transition{
		transaction: tx,
		postState:   postState,
	}
}

type TransitionBatch struct {
	transitions []*Transition
}

func NewTransitionBatch(defaultSize int) *TransitionBatch {
	return &TransitionBatch{transitions: make([]*Transition, 0, defaultSize)}
}

// addBlock adds a Geth Block to the TransitionBatch. This is just its transaction and state root.
func (r *TransitionBatch) addBlock(block *types.Block) {
	r.transitions = append(r.transitions, newTransition(block.Transactions()[0], block.Root()))
}
