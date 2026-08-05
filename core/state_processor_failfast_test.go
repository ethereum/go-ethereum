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
	"math/big"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// storageReadRecorder wraps a state.Reader and records which accounts had a
// storage slot read through it. It is the observation channel for the
// fail-fast test: each transaction below touches the storage of a distinct
// contract, so the set of recorded contract addresses identifies exactly
// which transactions were executed by the parallel workers. Embedding the
// interface deliberately forwards only the plain state.Reader method set;
// optional capabilities such as state.ReaderStater are dropped, which merely
// disables read-statistics reporting.
type storageReadRecorder struct {
	state.Reader
	mu    sync.Mutex
	reads map[common.Address]struct{}
}

func (r *storageReadRecorder) Storage(addr common.Address, slot common.Hash) (common.Hash, error) {
	r.mu.Lock()
	r.reads[addr] = struct{}{}
	r.mu.Unlock()
	return r.Reader.Storage(addr, slot)
}

// failFastTxGas is the gas allowance of each looping test transaction; the
// SLOAD loop burns it in full, keeping every valid transaction in flight far
// longer than the failing transaction needs to error out.
const failFastTxGas = 500_000

// runRecorded processes the block through the parallel path on a fresh state
// whose reads are observed by a storageReadRecorder, and returns how many of
// the given contract addresses had their storage read, along with the
// processing error.
func runRecorded(t *testing.T, bc *BlockChain, block *types.Block, contracts []common.Address) (int, error) {
	t.Helper()
	base, err := bc.State()
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	recorder := &storageReadRecorder{Reader: base.Reader(), reads: make(map[common.Address]struct{})}
	statedb, err := state.NewWithReader(bc.CurrentBlock().Root, base.Database(), recorder)
	if err != nil {
		t.Fatalf("state with reader: %v", err)
	}
	_, perr := NewStateProcessor(bc).Process(context.Background(), block, statedb, nil, nil, vm.Config{}, nil)

	executed := 0
	recorder.mu.Lock()
	for _, addr := range contracts {
		if _, ok := recorder.reads[addr]; ok {
			executed++
		}
	}
	recorder.mu.Unlock()
	return executed, perr
}

// TestParallelExecutionFailFast asserts that when one transaction fails during
// parallel execution, the remaining workers stop claiming transactions instead
// of executing the rest of the (already invalid) block. Sequential execution
// aborts at the first failing transaction; this pins the parallel path to the
// same behavior.
func TestParallelExecutionFailFast(t *testing.T) {
	if runtime.GOMAXPROCS(0) < 2 {
		t.Skip("fail-fast discrimination requires at least two parallel workers")
	}
	// One distinct contract per transaction, each burning its full gas
	// allowance in an SLOAD loop: JUMPDEST; PUSH1 0; SLOAD; POP; PUSH1 0; JUMP.
	numTxs := 4*runtime.GOMAXPROCS(0) + 4
	code := []byte{0x5b, 0x60, 0x00, 0x54, 0x50, 0x60, 0x00, 0x56}
	contracts := make([]common.Address, numTxs)
	extra := types.GenesisAlloc{}
	for i := range contracts {
		contracts[i] = common.BigToAddress(big.NewInt(int64(0x10000 + i)))
		extra[contracts[i]] = types.Account{Nonce: 1, Code: code, Balance: common.Big0}
	}
	env := newBALTestEnv(extra)
	// Size the block gas limit to the core count so the workload cannot
	// outgrow it on many-core machines.
	env.gspec.GasLimit = uint64(numTxs)*failFastTxGas + 1_000_000

	// Generate a fully valid block calling every contract once.
	engine := beacon.New(ethash.NewFaker())
	_, blocks, _ := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, b *BlockGen) {
		for i := 0; i < numTxs; i++ {
			b.AddTx(env.tx(uint64(i), &contracts[i], common.Big0, failFastTxGas, 1, nil))
		}
	})
	block := blocks[0]

	bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, nil)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}
	defer bc.Stop()

	// Positive control: the valid block must take the parallel path and
	// execute every transaction, pinning the observation channel before it
	// is used to assert an early stop.
	if !supportsParallelExecution(block, env.cfg, false, false, false) {
		t.Fatal("block is not eligible for parallel execution")
	}
	executed, err := runRecorded(t, bc, block, contracts)
	if err != nil {
		t.Fatalf("control run failed: %v", err)
	}
	if executed != numTxs {
		t.Fatalf("control run executed %d of %d transactions", executed, numTxs)
	}

	// Swap the first transaction for one whose value exceeds the sender's
	// balance: it passes signature and nonce checks but fails the balance
	// precheck inside the worker. The header and access list are kept as-is;
	// Process does not re-validate them and the access list only steers reads
	// of the still valid remainder.
	badTx := env.tx(0, &contracts[0], newGwei(2_000_000_000), failFastTxGas, 1, nil)
	body := *block.Body()
	body.Transactions = append([]*types.Transaction{badTx}, block.Transactions()[1:]...)
	doctored := block.WithBody(body)
	if !supportsParallelExecution(doctored, env.cfg, false, false, false) {
		t.Fatal("doctored block is not eligible for parallel execution")
	}
	executed, err = runRecorded(t, bc, doctored, contracts)
	if err == nil {
		t.Fatal("expected parallel execution to fail on the doctored transaction")
	}
	if !errors.Is(err, ErrInsufficientFunds) || !strings.Contains(err.Error(), "could not apply tx 0") {
		t.Fatalf("unexpected error: %v", err)
	}
	// contracts[0] is targeted by the failing transaction and can never be
	// read, so the executable ceiling is numTxs-1.
	if valid := numTxs - 1; executed >= valid {
		t.Fatalf("workers executed all %d valid transactions despite tx 0 failing; want an early stop", valid)
	}
	t.Logf("executed %d of %d valid transactions before the failure propagated", executed, numTxs-1)
}

// TestParallelExecutionDetachedFromCallerContext asserts that the worker
// group's cancellation is not wired to the caller's context: a valid block
// must process successfully even under an already-cancelled context. Block
// validity must never depend on the lifetime of the RPC request that
// delivered the block — an aborted engine-API call would otherwise turn a
// valid block into a bad-block report and an INVALID payload status.
func TestParallelExecutionDetachedFromCallerContext(t *testing.T) {
	env := newBALTestEnv(nil)
	engine := beacon.New(ethash.NewFaker())
	to := common.BigToAddress(big.NewInt(0xdead))
	_, blocks, _ := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, b *BlockGen) {
		b.AddTx(env.tx(0, &to, big.NewInt(1), params.TxGas, 1, nil))
	})
	block := blocks[0]
	if !supportsParallelExecution(block, env.cfg, false, false, false) {
		t.Fatal("block is not eligible for parallel execution")
	}
	bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, nil)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}
	defer bc.Stop()
	statedb, err := bc.State()
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewStateProcessor(bc).Process(ctx, block, statedb, nil, nil, vm.Config{}, nil); err != nil {
		t.Fatalf("valid block failed under cancelled caller context: %v", err)
	}
}
