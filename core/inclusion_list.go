// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// blockGas is the gas used by a block's transactions in each EIP-8037 dimension.
type blockGas struct {
	execution uint64
	state     uint64
}

// BlockGasUsed returns the execution and state gas used by the block's
// transactions. A block processed before the node started is re-executed.
func (bc *BlockChain) BlockGasUsed(block *types.Block) (uint64, uint64, error) {
	if gas, ok := bc.blockGasCache.Get(block.Hash()); ok {
		return gas.execution, gas.state, nil
	}
	parent := bc.GetHeader(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return 0, 0, consensus.ErrUnknownAncestor
	}
	statedb, err := bc.StateAt(parent.Root, parent.Number, parent.Time)
	if err != nil {
		return 0, 0, err
	}
	res, err := bc.processor.Process(context.Background(), block, statedb, nil, nil, bc.overrideTracerActivation(false), nil)
	if err != nil {
		return 0, 0, err
	}
	bc.blockGasCache.Add(block.Hash(), blockGas{res.ExecutionGasUsed, res.StateGasUsed})
	return res.ExecutionGasUsed, res.StateGasUsed, nil
}

// CheckInclusionList reports whether the block satisfies the inclusion list
// (EIP-7805), given the execution and state gas used by its transactions.
func (bc *BlockChain) CheckInclusionList(block *types.Block, gasExecution, gasState uint64, inclusionList [][]byte) (bool, error) {
	parent := bc.GetHeader(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return false, consensus.ErrUnknownAncestor
	}
	accessList := block.AccessList()
	if accessList == nil {
		return false, errors.New("missing block access list")
	}
	parentState, err := bc.StateAt(parent.Root, parent.Number, parent.Time)
	if err != nil {
		return false, err
	}
	// The list is checked after the last transaction and before the withdrawals,
	// which is the access list view at the post-execution index.
	statedb, err := newAccessListState(parentState.Database(), parent.Root, parentState.Reader(), accessList.Lookup(), len(block.Transactions())+1)
	if err != nil {
		return false, err
	}
	gp := &GasPool{
		initial:             block.GasLimit(),
		cumulativeExecution: gasExecution,
		cumulativeState:     gasState,
	}
	satisfied := checkInclusionList(bc.chainConfig, bc, block.Header(), block.Transactions(), statedb, gp, inclusionList)
	if err := statedb.Error(); err != nil {
		return false, err
	}
	return satisfied, nil
}

// checkInclusionList reports whether none of the inclusion list transactions
// missing from the block could have been appended to it. The state is the one
// after the block's last transaction, and the gas pool holds the gas used by
// the transactions.
func checkInclusionList(config *params.ChainConfig, chain ChainContext, header *types.Header, txs types.Transactions, statedb *state.StateDB, gp *GasPool, inclusionList [][]byte) bool {
	var (
		signer   = types.MakeSigner(config, header.Number, header.Time)
		rules    = config.Rules(header.Number, true, header.Time)
		evm      = vm.NewEVM(NewEVMBlockContext(header, chain, nil), statedb, config, vm.Config{})
		included = make(map[common.Hash]struct{}, len(txs))
	)
	defer evm.Release()

	for _, tx := range txs {
		included[tx.Hash()] = struct{}{}
	}
	for _, raw := range inclusionList {
		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(raw); err != nil {
			continue
		}
		// Transactions already in the block and blob transactions are exempt.
		if _, ok := included[tx.Hash()]; ok || tx.Type() == types.BlobTxType {
			continue
		}
		// A bad signature or a foreign chain ID fails here.
		msg, err := TransactionToMessage(tx, signer, header.BaseFee)
		if err != nil {
			continue
		}
		if canAppend(evm, msg, gp, rules) {
			return false
		}
	}
	return true
}

// canAppend reports whether the message passes every check a transaction has
// to pass to be included in the block, without executing it.
func canAppend(evm *vm.EVM, msg *Message, gp *GasPool, rules params.Rules) bool {
	// Buying gas debits the sender, undo it so the next check starts clean.
	snapshot := evm.StateDB.Snapshot()
	defer evm.StateDB.RevertToSnapshot(snapshot)

	evm.SetTxContext(NewEVMTxContext(msg))
	st := newStateTransition(evm, msg, gp)
	if err := st.preCheck(rules); err != nil {
		return false
	}
	_, _, err := st.checkIntrinsicGas(rules)
	return err == nil
}
