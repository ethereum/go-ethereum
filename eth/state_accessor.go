// Copyright 2021 The go-ethereum Authors
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

package eth

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/trie"
)

// noopReleaser is returned in case there is no operation expected
// for releasing the tracing state.
var noopReleaser = func() {}

// StateAtBlock retrieves the state database associated with a certain block.
// If no state is locally available for the given block, recover the state by
// applying the state reverse diffs.
//
// The optional base statedb can be passed then it's regarded as the statedb
// of the parent block.
//
// Parameters:
// - block:  The block for which we want the state (== state at the stateRoot of the parent)
// - parent: If the caller is tracing multiple blocks, the caller can provide the parent state
//           continuously from the callsite.
func (eth *Ethereum) StateAtBlock(block *types.Block, parent *state.StateDB) (*state.StateDB, func(), error) {
	// Check if the requested state is available in the live chain.
	statedb, err := eth.blockchain.StateAt(block.Root())
	if err == nil {
		return statedb, noopReleaser, nil
	}
	// Build the requested state based on the given parent state
	// by applying the block on top.
	if parent != nil {
		_, _, _, err := eth.blockchain.Processor().Process(block, parent, vm.Config{})
		if err != nil {
			return nil, nil, fmt.Errorf("processing block %d failed: %v", block.NumberU64(), err)
		}
		// Finalize the state so any modifications are written to the trie
		root, err := parent.Commit(eth.blockchain.Config().IsEIP158(block.Number()))
		if err != nil {
			return nil, nil, fmt.Errorf("stateAtBlock commit failed, number %d root %v: %w",
				block.NumberU64(), block.Root().Hex(), err)
		}
		statedb, err := state.New(root, parent.Database(), nil)
		if err != nil {
			return nil, nil, fmt.Errorf("state reset after block %d failed: %v", block.NumberU64(), err)
		}
		return statedb, noopReleaser, nil
	}
	// Create an isolated state snapshot from the live chain. All
	// the mutations caused later can be erased by invoking release
	// function. Note if the requested state is too old to recover
	// an error will be returned. And it's an expensive operation
	// can take a few minutes depends on how many reverts are required.
	snap, err := trie.NewDatabaseSnapshot(eth.BlockChain().TrieDB(), block.Root())
	if err != nil {
		return nil, nil, err
	}
	stateCache, err := state.NewDatabaseWithSnapshot(snap)
	if err != nil {
		return nil, nil, err
	}
	state, err := state.New(block.Root(), stateCache, nil)
	if err != nil {
		return nil, nil, err
	}
	return state, snap.Release, nil
}

// stateAtTransaction returns the execution environment of a certain transaction.
func (eth *Ethereum) stateAtTransaction(block *types.Block, txIndex int) (core.Message, vm.BlockContext, *state.StateDB, func(), error) {
	// Short circuit if it's genesis block.
	if block.NumberU64() == 0 {
		return nil, vm.BlockContext{}, nil, nil, errors.New("no transaction in genesis")
	}
	// Create the parent state database
	parent := eth.blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("parent %#x not found", block.ParentHash())
	}
	// Lookup the statedb of parent block from the live database,
	// otherwise regenerate it on the flight.
	statedb, rel, err := eth.StateAtBlock(parent, nil)
	if err != nil {
		return nil, vm.BlockContext{}, nil, nil, err
	}
	if txIndex == 0 && len(block.Transactions()) == 0 {
		return nil, vm.BlockContext{}, statedb, rel, nil
	}
	// Recompute transactions up to the target index.
	signer := types.MakeSigner(eth.blockchain.Config(), block.Number())
	for idx, tx := range block.Transactions() {
		// Assemble the transaction call message and return if the requested offset
		msg, _ := tx.AsMessage(signer, block.BaseFee())
		txContext := core.NewEVMTxContext(msg)
		context := core.NewEVMBlockContext(block.Header(), eth.blockchain, nil)
		if idx == txIndex {
			return msg, context, statedb, rel, nil
		}
		// Not yet the searched for transaction, execute on top of the current state
		vmenv := vm.NewEVM(context, txContext, statedb, eth.blockchain.Config(), vm.Config{})
		statedb.Prepare(tx.Hash(), idx)
		if _, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(tx.Gas())); err != nil {
			return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("transaction %#x failed: %v", tx.Hash(), err)
		}
		// Ensure any modifications are committed to the state
		// Only delete empty objects if EIP158/161 (a.k.a Spurious Dragon) is in effect
		statedb.Finalise(vmenv.ChainConfig().IsEIP158(block.Number()))
	}
	return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("transaction index %d out of range for block %#x", txIndex, block.Hash())
}
