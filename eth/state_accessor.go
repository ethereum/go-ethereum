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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
)

// stateAtBlock retrieves the state database associated with a certain block.
// If no state is locally available for the given block, a number of blocks are
// attempted to be reexecuted to generate the desired state.
func (eth *Ethereum) stateAtBlock(block *types.Block, reexec uint64) (statedb *state.StateDB, release func(), err error) {
	// If we have the state fully available, use that
	statedb, err = eth.blockchain.StateAt(block.Root())
	if err == nil {
		return statedb, func() {}, nil
	}
	// Otherwise try to reexec blocks until we find a state or reach our limit
	origin := block.NumberU64()
	database := state.NewDatabaseWithConfig(eth.chainDb, &trie.Config{Cache: 16, Preimages: true})

	for i := uint64(0); i < reexec; i++ {
		if block.NumberU64() == 0 {
			return nil, nil, errors.New("genesis state is missing")
		}
		parent := eth.blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1)
		if parent == nil {
			return nil, nil, fmt.Errorf("missing block %v %d", block.ParentHash(), block.NumberU64()-1)
		}
		block = parent

		statedb, err = state.New(block.Root(), database, nil)
		if err == nil {
			break
		}
	}
	if err != nil {
		switch err.(type) {
		case *trie.MissingNodeError:
			return nil, nil, fmt.Errorf("required historical state unavailable (reexec=%d)", reexec)
		default:
			return nil, nil, err
		}
	}
	// State was available at historical point, regenerate
	var (
		start  = time.Now()
		logged time.Time
		parent common.Hash
	)
	defer func() {
		if err != nil && parent != (common.Hash{}) {
			database.TrieDB().Dereference(parent)
		}
	}()
	for block.NumberU64() < origin {
		// Print progress logs if long enough time elapsed
		if time.Since(logged) > 8*time.Second {
			log.Info("Regenerating historical state", "block", block.NumberU64()+1, "target", origin, "remaining", origin-block.NumberU64()-1, "elapsed", time.Since(start))
			logged = time.Now()
		}
		// Retrieve the next block to regenerate and process it
		if block = eth.blockchain.GetBlockByNumber(block.NumberU64() + 1); block == nil {
			return nil, nil, fmt.Errorf("block #%d not found", block.NumberU64()+1)
		}
		_, _, _, err := eth.blockchain.Processor().Process(block, statedb, vm.Config{})
		if err != nil {
			return nil, nil, fmt.Errorf("processing block %d failed: %v", block.NumberU64(), err)
		}
		// Finalize the state so any modifications are written to the trie
		root, err := statedb.Commit(eth.blockchain.Config().IsEIP158(block.Number()))
		if err != nil {
			return nil, nil, err
		}
		statedb, err = state.New(root, database, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("state reset after block %d failed: %v", block.NumberU64(), err)
		}
		database.TrieDB().Reference(root, common.Hash{})
		if parent != (common.Hash{}) {
			database.TrieDB().Dereference(parent)
		}
		parent = root
	}
	nodes, imgs := database.TrieDB().Size()
	log.Info("Historical state regenerated", "block", block.NumberU64(), "elapsed", time.Since(start), "nodes", nodes, "preimages", imgs)
	return statedb, func() { database.TrieDB().Dereference(parent) }, nil
}

// statesInRange retrieves a batch of state databases associated with the specific
// block ranges. If no state is locally available for the given range, a number of
// blocks are attempted to be reexecuted to generate the ancestor state.
func (eth *Ethereum) statesInRange(fromBlock, toBlock *types.Block, reexec uint64) (states []*state.StateDB, release func(), err error) {
	statedb, err := eth.blockchain.StateAt(fromBlock.Root())
	if err != nil {
		statedb, _, err = eth.stateAtBlock(fromBlock, reexec)
	}
	if err != nil {
		return nil, nil, err
	}
	states = append(states, statedb.Copy())

	var (
		logged   time.Time
		parent   common.Hash
		start    = time.Now()
		refs     = []common.Hash{fromBlock.Root()}
		database = state.NewDatabaseWithConfig(eth.chainDb, &trie.Config{Cache: 16, Preimages: true})
	)
	// Release all resources(including the states referenced by `stateAtBlock`)
	// if error is returned.
	defer func() {
		if err != nil {
			for _, ref := range refs {
				database.TrieDB().Dereference(ref)
			}
		}
	}()
	for i := fromBlock.NumberU64() + 1; i <= toBlock.NumberU64(); i++ {
		// Print progress logs if long enough time elapsed
		if time.Since(logged) > 8*time.Second {
			logged = time.Now()
			log.Info("Regenerating historical state", "block", i, "target", fromBlock.NumberU64(), "remaining", toBlock.NumberU64()-i, "elapsed", time.Since(start))
		}
		// Retrieve the next block to regenerate and process it
		block := eth.blockchain.GetBlockByNumber(i)
		if block == nil {
			return nil, nil, fmt.Errorf("block #%d not found", i)
		}
		_, _, _, err := eth.blockchain.Processor().Process(block, statedb, vm.Config{})
		if err != nil {
			return nil, nil, fmt.Errorf("processing block %d failed: %v", block.NumberU64(), err)
		}
		// Finalize the state so any modifications are written to the trie
		root, err := statedb.Commit(eth.blockchain.Config().IsEIP158(block.Number()))
		if err != nil {
			return nil, nil, err
		}
		statedb, err := eth.blockchain.StateAt(root)
		if err != nil {
			return nil, nil, fmt.Errorf("state reset after block %d failed: %v", block.NumberU64(), err)
		}
		states = append(states, statedb.Copy())

		// Reference the trie twice, once for us, once for the tracer
		database.TrieDB().Reference(root, common.Hash{})
		database.TrieDB().Reference(root, common.Hash{})
		refs = append(refs, root)

		// Dereference all past tries we ourselves are done working with
		if parent != (common.Hash{}) {
			database.TrieDB().Dereference(parent)
		}
		parent = root
	}
	// release is handler to release all states referenced, including
	// the one referenced in `stateAtBlock`.
	release = func() {
		for _, ref := range refs {
			database.TrieDB().Dereference(ref)
		}
	}
	return states, release, nil
}

// stateAtTransaction returns the execution environment of a certain transaction.
func (eth *Ethereum) stateAtTransaction(block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, func(), error) {
	// Short circuit if it's genesis block.
	if block.NumberU64() == 0 {
		return nil, vm.BlockContext{}, nil, nil, errors.New("no transaction in genesis")
	}
	// Create the parent state database
	parent := eth.blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("parent %#x not found", block.ParentHash())
	}
	statedb, release, err := eth.stateAtBlock(parent, reexec)
	if err != nil {
		return nil, vm.BlockContext{}, nil, nil, err
	}
	if txIndex == 0 && len(block.Transactions()) == 0 {
		return nil, vm.BlockContext{}, statedb, release, nil
	}
	// Recompute transactions up to the target index.
	signer := types.MakeSigner(eth.blockchain.Config(), block.Number())
	for idx, tx := range block.Transactions() {
		// Assemble the transaction call message and return if the requested offset
		msg, _ := tx.AsMessage(signer)
		txContext := core.NewEVMTxContext(msg)
		context := core.NewEVMBlockContext(block.Header(), eth.blockchain, nil)
		if idx == txIndex {
			return msg, context, statedb, release, nil
		}
		// Not yet the searched for transaction, execute on top of the current state
		vmenv := vm.NewEVM(context, txContext, statedb, eth.blockchain.Config(), vm.Config{})
		statedb.Prepare(tx.Hash(), block.Hash(), idx)
		if _, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(tx.Gas())); err != nil {
			release()
			return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("transaction %#x failed: %v", tx.Hash(), err)
		}
		// Ensure any modifications are committed to the state
		// Only delete empty objects if EIP158/161 (a.k.a Spurious Dragon) is in effect
		statedb.Finalise(vmenv.ChainConfig().IsEIP158(block.Number()))
	}
	release()
	return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("transaction index %d out of range for block %#x", txIndex, block.Hash())
}
