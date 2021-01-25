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

package tracers

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	errStateNotFound       = errors.New("state not found")
	errBlockNotFound       = errors.New("block not found")
	errTransactionNotFound = errors.New("transaction not found")
)

type testBackend struct {
	chainConfig *params.ChainConfig
	engine      consensus.Engine
	chaindb     ethdb.Database
	chain       *core.BlockChain
}

func newTestBackend(t *testing.T, n int, gspec *core.Genesis, generator func(i int, b *core.BlockGen)) *testBackend {
	backend := &testBackend{
		chainConfig: params.TestChainConfig,
		engine:      ethash.NewFaker(),
		chaindb:     rawdb.NewMemoryDatabase(),
	}
	// Generate blocks for testing
	gspec.Config = backend.chainConfig
	var (
		gendb   = rawdb.NewMemoryDatabase()
		genesis = gspec.MustCommit(gendb)
	)
	blocks, _ := core.GenerateChain(backend.chainConfig, genesis, backend.engine, gendb, n, generator)

	// Import the canonical chain
	gspec.MustCommit(backend.chaindb)
	cacheConfig := &core.CacheConfig{
		TrieCleanLimit:    256,
		TrieDirtyLimit:    256,
		TrieTimeLimit:     5 * time.Minute,
		SnapshotLimit:     0,
		TrieDirtyDisabled: true, // Archive mode
	}
	chain, err := core.NewBlockChain(backend.chaindb, cacheConfig, backend.chainConfig, backend.engine, vm.Config{}, nil, nil)
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
		return b.chain.CurrentBlock(), nil
	}
	return b.chain.GetBlockByNumber(uint64(number)), nil
}

func (b *testBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, hash, blockNumber, index := rawdb.ReadTransaction(b.chaindb, txHash)
	if tx == nil {
		return nil, common.Hash{}, 0, 0, errTransactionNotFound
	}
	return tx, hash, blockNumber, index, nil
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

func (b *testBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64) (*state.StateDB, func(), error) {
	statedb, err := b.chain.StateAt(block.Root())
	if err != nil {
		return nil, nil, errStateNotFound
	}
	return statedb, func() {}, nil
}

func (b *testBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, func(), error) {
	parent := b.chain.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return nil, vm.BlockContext{}, nil, nil, errBlockNotFound
	}
	statedb, err := b.chain.StateAt(parent.Root())
	if err != nil {
		return nil, vm.BlockContext{}, nil, nil, errStateNotFound
	}
	if txIndex == 0 && len(block.Transactions()) == 0 {
		return nil, vm.BlockContext{}, statedb, func() {}, nil
	}
	// Recompute transactions up to the target index.
	signer := types.MakeSigner(b.chainConfig, block.Number())
	for idx, tx := range block.Transactions() {
		msg, _ := tx.AsMessage(signer)
		txContext := core.NewEVMTxContext(msg)
		context := core.NewEVMBlockContext(block.Header(), b.chain, nil)
		if idx == txIndex {
			return msg, context, statedb, func() {}, nil
		}
		vmenv := vm.NewEVM(context, txContext, statedb, b.chainConfig, vm.Config{})
		if _, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(tx.Gas())); err != nil {
			return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("transaction %#x failed: %v", tx.Hash(), err)
		}
		statedb.Finalise(vmenv.ChainConfig().IsEIP158(block.Number()))
	}
	return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("transaction index %d out of range for block %#x", txIndex, block.Hash())
}

func (b *testBackend) StatesInRange(ctx context.Context, fromBlock *types.Block, toBlock *types.Block, reexec uint64) ([]*state.StateDB, func(), error) {
	var result []*state.StateDB
	for number := fromBlock.NumberU64(); number <= toBlock.NumberU64(); number += 1 {
		block := b.chain.GetBlockByNumber(number)
		if block == nil {
			return nil, nil, errBlockNotFound
		}
		statedb, err := b.chain.StateAt(block.Root())
		if err != nil {
			return nil, nil, errStateNotFound
		}
		result = append(result, statedb)
	}
	return result, func() {}, nil
}

func TestTraceCall(t *testing.T) {
	t.Parallel()

	// Initialize test accounts
	accounts := newAccounts(3)
	genesis := &core.Genesis{Alloc: core.GenesisAlloc{
		accounts[0].addr: {Balance: big.NewInt(params.Ether)},
		accounts[1].addr: {Balance: big.NewInt(params.Ether)},
		accounts[2].addr: {Balance: big.NewInt(params.Ether)},
	}}
	genBlocks := 10
	signer := types.HomesteadSigner{}
	api := NewAPI(newTestBackend(t, genBlocks, genesis, func(i int, b *core.BlockGen) {
		// Transfer from account[0] to account[1]
		//    value: 1000 wei
		//    fee:   0 wei
		tx, _ := types.SignTx(types.NewTransaction(uint64(i), accounts[1].addr, big.NewInt(1000), params.TxGas, big.NewInt(0), nil), signer, accounts[0].key)
		b.AddTx(tx)
	}))

	var testSuite = []struct {
		blockNumber rpc.BlockNumber
		call        ethapi.CallArgs
		config      *TraceConfig
		expectErr   error
		expect      interface{}
	}{
		// Standard JSON trace upon the genesis, plain transfer.
		{
			blockNumber: rpc.BlockNumber(0),
			call: ethapi.CallArgs{
				From:  &accounts[0].addr,
				To:    &accounts[1].addr,
				Value: (*hexutil.Big)(big.NewInt(1000)),
			},
			config:    nil,
			expectErr: nil,
			expect: &ethapi.ExecutionResult{
				Gas:         params.TxGas,
				Failed:      false,
				ReturnValue: "",
				StructLogs:  []ethapi.StructLogRes{},
			},
		},
		// Standard JSON trace upon the head, plain transfer.
		{
			blockNumber: rpc.BlockNumber(genBlocks),
			call: ethapi.CallArgs{
				From:  &accounts[0].addr,
				To:    &accounts[1].addr,
				Value: (*hexutil.Big)(big.NewInt(1000)),
			},
			config:    nil,
			expectErr: nil,
			expect: &ethapi.ExecutionResult{
				Gas:         params.TxGas,
				Failed:      false,
				ReturnValue: "",
				StructLogs:  []ethapi.StructLogRes{},
			},
		},
		// Standard JSON trace upon the non-existent block, error expects
		{
			blockNumber: rpc.BlockNumber(genBlocks + 1),
			call: ethapi.CallArgs{
				From:  &accounts[0].addr,
				To:    &accounts[1].addr,
				Value: (*hexutil.Big)(big.NewInt(1000)),
			},
			config:    nil,
			expectErr: fmt.Errorf("block #%d not found", genBlocks+1),
			expect:    nil,
		},
		// Standard JSON trace upon the latest block
		{
			blockNumber: rpc.LatestBlockNumber,
			call: ethapi.CallArgs{
				From:  &accounts[0].addr,
				To:    &accounts[1].addr,
				Value: (*hexutil.Big)(big.NewInt(1000)),
			},
			config:    nil,
			expectErr: nil,
			expect: &ethapi.ExecutionResult{
				Gas:         params.TxGas,
				Failed:      false,
				ReturnValue: "",
				StructLogs:  []ethapi.StructLogRes{},
			},
		},
		// Standard JSON trace upon the pending block
		{
			blockNumber: rpc.PendingBlockNumber,
			call: ethapi.CallArgs{
				From:  &accounts[0].addr,
				To:    &accounts[1].addr,
				Value: (*hexutil.Big)(big.NewInt(1000)),
			},
			config:    nil,
			expectErr: nil,
			expect: &ethapi.ExecutionResult{
				Gas:         params.TxGas,
				Failed:      false,
				ReturnValue: "",
				StructLogs:  []ethapi.StructLogRes{},
			},
		},
	}
	for _, testspec := range testSuite {
		result, err := api.TraceCall(context.Background(), testspec.call, rpc.BlockNumberOrHash{BlockNumber: &testspec.blockNumber}, testspec.config)
		if testspec.expectErr != nil {
			if err == nil {
				t.Errorf("Expect error %v, get nothing", testspec.expectErr)
				continue
			}
			if !reflect.DeepEqual(err, testspec.expectErr) {
				t.Errorf("Error mismatch, want %v, get %v", testspec.expectErr, err)
			}
		} else {
			if err != nil {
				t.Errorf("Expect no error, get %v", err)
				continue
			}
			if !reflect.DeepEqual(result, testspec.expect) {
				t.Errorf("Result mismatch, want %v, get %v", testspec.expect, result)
			}
		}
	}
}

func TestTraceTransaction(t *testing.T) {
	t.Parallel()

	// Initialize test accounts
	accounts := newAccounts(2)
	genesis := &core.Genesis{Alloc: core.GenesisAlloc{
		accounts[0].addr: {Balance: big.NewInt(params.Ether)},
		accounts[1].addr: {Balance: big.NewInt(params.Ether)},
	}}
	target := common.Hash{}
	signer := types.HomesteadSigner{}
	api := NewAPI(newTestBackend(t, 1, genesis, func(i int, b *core.BlockGen) {
		// Transfer from account[0] to account[1]
		//    value: 1000 wei
		//    fee:   0 wei
		tx, _ := types.SignTx(types.NewTransaction(uint64(i), accounts[1].addr, big.NewInt(1000), params.TxGas, big.NewInt(0), nil), signer, accounts[0].key)
		b.AddTx(tx)
		target = tx.Hash()
	}))
	result, err := api.TraceTransaction(context.Background(), target, nil)
	if err != nil {
		t.Errorf("Failed to trace transaction %v", err)
	}
	if !reflect.DeepEqual(result, &ethapi.ExecutionResult{
		Gas:         params.TxGas,
		Failed:      false,
		ReturnValue: "",
		StructLogs:  []ethapi.StructLogRes{},
	}) {
		t.Error("Transaction tracing result is different")
	}
}

func TestTraceBlock(t *testing.T) {
	t.Parallel()

	// Initialize test accounts
	accounts := newAccounts(3)
	genesis := &core.Genesis{Alloc: core.GenesisAlloc{
		accounts[0].addr: {Balance: big.NewInt(params.Ether)},
		accounts[1].addr: {Balance: big.NewInt(params.Ether)},
		accounts[2].addr: {Balance: big.NewInt(params.Ether)},
	}}
	genBlocks := 10
	signer := types.HomesteadSigner{}
	api := NewAPI(newTestBackend(t, genBlocks, genesis, func(i int, b *core.BlockGen) {
		// Transfer from account[0] to account[1]
		//    value: 1000 wei
		//    fee:   0 wei
		tx, _ := types.SignTx(types.NewTransaction(uint64(i), accounts[1].addr, big.NewInt(1000), params.TxGas, big.NewInt(0), nil), signer, accounts[0].key)
		b.AddTx(tx)
	}))

	var testSuite = []struct {
		blockNumber rpc.BlockNumber
		config      *TraceConfig
		expect      interface{}
		expectErr   error
	}{
		// Trace genesis block, expect error
		{
			blockNumber: rpc.BlockNumber(0),
			config:      nil,
			expect:      nil,
			expectErr:   errors.New("genesis is not traceable"),
		},
		// Trace head block
		{
			blockNumber: rpc.BlockNumber(genBlocks),
			config:      nil,
			expectErr:   nil,
			expect: []*txTraceResult{
				{
					Result: &ethapi.ExecutionResult{
						Gas:         params.TxGas,
						Failed:      false,
						ReturnValue: "",
						StructLogs:  []ethapi.StructLogRes{},
					},
				},
			},
		},
		// Trace non-existent block
		{
			blockNumber: rpc.BlockNumber(genBlocks + 1),
			config:      nil,
			expectErr:   fmt.Errorf("block #%d not found", genBlocks+1),
			expect:      nil,
		},
		// Trace latest block
		{
			blockNumber: rpc.LatestBlockNumber,
			config:      nil,
			expectErr:   nil,
			expect: []*txTraceResult{
				{
					Result: &ethapi.ExecutionResult{
						Gas:         params.TxGas,
						Failed:      false,
						ReturnValue: "",
						StructLogs:  []ethapi.StructLogRes{},
					},
				},
			},
		},
		// Trace pending block
		{
			blockNumber: rpc.PendingBlockNumber,
			config:      nil,
			expectErr:   nil,
			expect: []*txTraceResult{
				{
					Result: &ethapi.ExecutionResult{
						Gas:         params.TxGas,
						Failed:      false,
						ReturnValue: "",
						StructLogs:  []ethapi.StructLogRes{},
					},
				},
			},
		},
	}
	for _, testspec := range testSuite {
		result, err := api.TraceBlockByNumber(context.Background(), testspec.blockNumber, testspec.config)
		if testspec.expectErr != nil {
			if err == nil {
				t.Errorf("Expect error %v, get nothing", testspec.expectErr)
				continue
			}
			if !reflect.DeepEqual(err, testspec.expectErr) {
				t.Errorf("Error mismatch, want %v, get %v", testspec.expectErr, err)
			}
		} else {
			if err != nil {
				t.Errorf("Expect no error, get %v", err)
				continue
			}
			if !reflect.DeepEqual(result, testspec.expect) {
				t.Errorf("Result mismatch, want %v, get %v", testspec.expect, result)
			}
		}
	}
}

type Account struct {
	key  *ecdsa.PrivateKey
	addr common.Address
}

type Accounts []Account

func (a Accounts) Len() int           { return len(a) }
func (a Accounts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Accounts) Less(i, j int) bool { return bytes.Compare(a[i].addr.Bytes(), a[j].addr.Bytes()) < 0 }

func newAccounts(n int) (accounts Accounts) {
	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(key.PublicKey)
		accounts = append(accounts, Account{key: key, addr: addr})
	}
	sort.Sort(accounts)
	return accounts
}
