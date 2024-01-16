package builder

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

type builder struct {
	config   *builderConfig
	txns     []*types.Transaction
	receipts []*types.Receipt
	state    *state.StateDB
	gasPool  *core.GasPool
	gasUsed  *uint64
}

type builderConfig struct {
	preState *state.StateDB
	header   *types.Header
	config   *params.ChainConfig
	context  core.ChainContext
}

func newBuilder(config *builderConfig) *builder {
	gp := core.GasPool(config.header.GasLimit)
	var gasUsed uint64

	return &builder{
		config:  config,
		state:   config.preState.Copy(),
		gasPool: &gp,
		gasUsed: &gasUsed,
	}
}

func (b *builder) AddTransaction(txn *types.Transaction) (*types.SimulateTransactionResult, error) {
	dummyAuthor := common.Address{}

	vmConfig := vm.Config{
		NoBaseFee: true,
	}

	snap := b.state.Snapshot()

	b.state.SetTxContext(txn.Hash(), len(b.txns))
	receipt, err := core.ApplyTransaction(b.config.config, b.config.context, &dummyAuthor, b.gasPool, b.state, b.config.header, txn, b.gasUsed, vmConfig)
	if err != nil {
		b.state.RevertToSnapshot(snap)

		result := &types.SimulateTransactionResult{
			Success: false,
			Error:   err.Error(),
		}
		return result, nil
	}

	b.txns = append(b.txns, txn)
	b.receipts = append(b.receipts, receipt)

	result := &types.SimulateTransactionResult{
		Success: true,
		Logs:    []*types.SimulatedLog{},
	}
	for _, log := range receipt.Logs {
		result.Logs = append(result.Logs, &types.SimulatedLog{
			Addr:   log.Address,
			Topics: log.Topics,
			Data:   log.Data,
		})
	}

	return result, nil
}
