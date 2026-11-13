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

package traceapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/triedb/database"
)

type testBackend struct {
	chainConfig *params.ChainConfig
	engine      consensus.Engine
	chaindb     ethdb.Database
	chain       *core.BlockChain
	evmTimeout  time.Duration
}

// newTestBackend creates a new test backend. OBS: After test is done, teardown must be
// invoked in order to release associated resources.
func newTestBackend(t *testing.T, n int, gspec *core.Genesis, generator func(i int, b *core.BlockGen)) *testBackend {
	backend := &testBackend{
		chainConfig: gspec.Config,
		engine:      ethash.NewFaker(),
		chaindb:     rawdb.NewMemoryDatabase(),
		evmTimeout:  5 * time.Second,
	}
	// Generate blocks for testing
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, backend.engine, n, generator)

	// Import the canonical chain
	options := &core.BlockChainConfig{
		TrieCleanLimit: 256,
		TrieDirtyLimit: 256,
		TrieTimeLimit:  5 * time.Minute,
		SnapshotLimit:  0,
		ArchiveMode:    true, // Archive mode
	}
	chain, err := core.NewBlockChain(backend.chaindb, gspec, backend.engine, options)
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}
	backend.chain = chain
	return backend
}

func (b *testBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.chain.GetHeaderByHash(hash), nil
}

func (b *testBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	if number == rpc.PendingBlockNumber || number == rpc.LatestBlockNumber {
		return b.chain.CurrentHeader(), nil
	}
	return b.chain.GetHeaderByNumber(uint64(number)), nil
}

func (b *testBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.chain.GetBlockByHash(hash), nil
}

func (b *testBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	if number == rpc.PendingBlockNumber || number == rpc.LatestBlockNumber {
		return b.chain.GetBlockByNumber(b.chain.CurrentBlock().Number.Uint64()), nil
	}
	return b.chain.GetBlockByNumber(uint64(number)), nil
}

func (b *testBackend) GetCanonicalTransaction(txHash common.Hash) (bool, *types.Transaction, common.Hash, uint64, uint64) {
	tx, hash, blockNumber, index := rawdb.ReadCanonicalTransaction(b.chaindb, txHash)
	return tx != nil, tx, hash, blockNumber, index
}

func (b *testBackend) TxIndexDone() bool {
	return true
}

func (b *testBackend) RPCGasCap() uint64 {
	return 25000000
}

func (b *testBackend) ChainConfig() *params.ChainConfig {
	return b.chainConfig
}

func (b *testBackend) Engine() consensus.Engine {
	return b.engine
}

func (b *testBackend) ChainDb() ethdb.Database {
	return b.chaindb
}

func (b *testBackend) CurrentHeader() *types.Header {
	return b.chain.CurrentHeader()
}

// Pending reports no pending block.
func (b *testBackend) Pending() (*types.Block, types.Receipts, *state.StateDB) {
	return nil, nil, nil
}

func (b *testBackend) StateAtBlock(ctx context.Context, block *types.Block, base *state.StateDB, readOnly bool, preferDisk bool) (*state.StateDB, tracers.StateReleaseFunc, error) {
	statedb, err := b.chain.StateAt(block.Root(), block.Number(), block.Time())
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", database.ErrStateUnavailable, err)
	}
	return statedb, func() {}, nil
}

func (b *testBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int) (*types.Transaction, vm.BlockContext, *state.StateDB, tracers.StateReleaseFunc, error) {
	parent := b.chain.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("%w: parent %#x not found", database.ErrStateUnavailable, block.ParentHash())
	}
	statedb, release, err := b.StateAtBlock(ctx, parent, nil, true, false)
	if err != nil {
		return nil, vm.BlockContext{}, nil, nil, err
	}
	vmctx := core.NewEVMBlockContext(block.Header(), b.chain, nil)
	evm := vm.NewEVM(vmctx, statedb, b.chainConfig, vm.Config{})
	defer evm.Release()
	core.PreExecution(ctx, block.BeaconRoot(), parent.Header(), b.chainConfig, evm, block.Number(), block.Time())
	signer := types.MakeSigner(b.chainConfig, block.Number(), block.Time())
	for idx, tx := range block.Transactions() {
		if idx == txIndex {
			return tx, vmctx, statedb, release, nil
		}
		msg, err := core.TransactionToMessage(tx, signer, block.BaseFee())
		if err != nil {
			return nil, vm.BlockContext{}, nil, nil, err
		}
		statedb.SetTxContext(tx.Hash(), idx, uint32(idx+1))
		if _, err := core.ApplyMessage(evm, msg, nil); err != nil {
			return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("transaction %#x failed: %w", tx.Hash(), err)
		}
		statedb.Finalise(evm.GetRules())
	}
	if txIndex == 0 {
		return nil, vmctx, statedb, release, nil
	}
	return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("transaction index %d out of range for block %#x", txIndex, block.Hash())
}

func (b *testBackend) RPCEVMTimeout() time.Duration {
	return b.evmTimeout
}

// teardown releases the associated resources.
func (b *testBackend) teardown() {
	b.chain.Stop()
}
