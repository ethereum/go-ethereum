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
	"strings"
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

// faultyStorageReader wraps a state.Reader, failing reads of a single storage
// slot to emulate a backend fault during execution.
type faultyStorageReader struct {
	state.Reader
	addr common.Address
	slot common.Hash
	err  error
}

func (r *faultyStorageReader) Storage(addr common.Address, slot common.Hash) (common.Hash, error) {
	if addr == r.addr && slot == r.slot {
		return common.Hash{}, r.err
	}
	return r.Reader.Storage(addr, slot)
}

// faultyAccountReader wraps a state.Reader, failing reads of a single account
// to emulate a backend fault during execution.
type faultyAccountReader struct {
	state.Reader
	addr common.Address
	err  error
}

func (r *faultyAccountReader) Account(addr common.Address) (*types.StateAccount, error) {
	if addr == r.addr {
		return nil, r.err
	}
	return r.Reader.Account(addr)
}

// runParallelWithReader generates one block, then re-executes it through the
// parallel processor on a state whose reader is wrapped by wrap, returning the
// processing error.
func runParallelWithReader(t *testing.T, env *balTestEnv, gen func(*BlockGen), wrap func(state.Reader) state.Reader) error {
	t.Helper()
	engine := beacon.New(ethash.NewFaker())
	_, blocks, _ := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, b *BlockGen) {
		gen(b)
	})
	block := blocks[0]
	if block.AccessList() == nil {
		t.Fatal("expected non-nil block access list")
	}
	bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, nil)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}
	defer bc.Stop()

	head, err := bc.State()
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	// The chain head is still genesis, i.e. the executed block's parent.
	statedb, err := state.NewWithReader(bc.CurrentBlock().Root, head.Database(), wrap(head.Reader()))
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	_, err = NewStateProcessor(bc).Process(context.Background(), block, statedb, nil, nil, vm.Config{}, nil)
	return err
}

// TestParallelExecutionDatabaseErrorStorage: a storage read failing inside a
// transaction's ephemeral state must fail parallel processing with the
// database fault, not produce a silently wrong execution result.
func TestParallelExecutionDatabaseErrorStorage(t *testing.T) {
	var (
		contract = common.Address{0xcc}
		slot     = common.Hash{}
		injected = errors.New("injected storage fault")
	)
	// The contract copies slot 0 into slot 1: with a swallowed read fault the
	// transaction would silently store zero instead of the actual value. The
	// slot is only ever read, so neither the access list overlay nor the
	// canonical state applier resolves it: the fault stays confined to the
	// transaction's ephemeral state.
	env := newBALTestEnv(types.GenesisAlloc{
		contract: {
			Nonce:   1,
			Balance: common.Big0,
			Code: []byte{
				0x60, 0x00, 0x54, // PUSH1 0, SLOAD
				0x60, 0x01, 0x55, // PUSH1 1, SSTORE
				0x00, // STOP
			},
			Storage: map[common.Hash]common.Hash{slot: common.HexToHash("0x2a")},
		},
	})
	err := runParallelWithReader(t, env,
		func(b *BlockGen) {
			b.AddTx(env.tx(0, &contract, big.NewInt(0), 100_000, 1, nil))
		},
		func(base state.Reader) state.Reader {
			return &faultyStorageReader{Reader: base, addr: contract, slot: slot, err: injected}
		},
	)
	if !errors.Is(err, injected) {
		t.Fatalf("expected the injected database fault, got %v", err)
	}
}

// TestParallelExecutionDatabaseErrorAccount: an account read failing inside a
// transaction's ephemeral state must fail parallel processing with the
// database fault. Unlike storage reads, account faults surface through the
// getStateObject path.
func TestParallelExecutionDatabaseErrorAccount(t *testing.T) {
	var (
		contract = common.Address{0xcc}
		target   = common.Address{0xbb}
		injected = errors.New("injected account fault")
	)
	// The contract stores BALANCE(target) into slot 1: with a swallowed read
	// fault the probed account resolves as nonexistent and the transaction
	// silently stores zero. The target's balance never changes, so only the
	// transaction's ephemeral state resolves its account.
	code := append([]byte{0x73}, target.Bytes()...)   // PUSH20 target
	code = append(code, 0x31, 0x60, 0x01, 0x55, 0x00) // BALANCE, PUSH1 1, SSTORE, STOP

	env := newBALTestEnv(types.GenesisAlloc{
		contract: {Nonce: 1, Balance: common.Big0, Code: code},
		target:   {Balance: newGwei(1000)},
	})
	err := runParallelWithReader(t, env,
		func(b *BlockGen) {
			b.AddTx(env.tx(0, &contract, big.NewInt(0), 100_000, 1, nil))
		},
		func(base state.Reader) state.Reader {
			return &faultyAccountReader{Reader: base, addr: target, err: injected}
		},
	)
	if !errors.Is(err, injected) {
		t.Fatalf("expected the injected database fault, got %v", err)
	}
}

// TestParallelExecutionDatabaseErrorPreExecution: a database fault confined to
// the pre-execution system-call state must fail parallel processing. The beacon
// roots contract is substituted with code performing a pure SLOAD of a slot it
// never writes: the slot has no post-value in the access list, so neither the
// canonical applier nor any transaction resolves it — only the pre-execution
// state does. The substitution is safe because ProcessBeaconBlockRoot discards
// the call's result.
func TestParallelExecutionDatabaseErrorPreExecution(t *testing.T) {
	var (
		slot     = common.HexToHash("0x42")
		injected = errors.New("injected storage fault")
	)
	env := newBALTestEnv(types.GenesisAlloc{
		params.BeaconRootsAddress: {
			Nonce:   1,
			Balance: common.Big0,
			Code:    []byte{0x60, 0x42, 0x54, 0x00}, // PUSH1 0x42, SLOAD, STOP
		},
	})
	err := runParallelWithReader(t, env,
		func(b *BlockGen) {
			b.SetCoinbase(common.Address{0xc0})
		},
		func(base state.Reader) state.Reader {
			return &faultyStorageReader{Reader: base, addr: params.BeaconRootsAddress, slot: slot, err: injected}
		},
	)
	if !errors.Is(err, injected) {
		t.Fatalf("expected the injected database fault, got %v", err)
	}
	if !strings.Contains(err.Error(), "pre-execution system calls") {
		t.Fatalf("expected the pre-execution phase in the error, got %v", err)
	}
}

// TestParallelExecutionDatabaseErrorFinalize: a database fault confined to the
// post-execution system-call state must fail parallel processing. A zero-amount
// withdrawal touches its recipient in Engine.Finalize without changing its
// balance, so the account is resolved only by the post-execution state, never
// by the transactions or the canonical access-list applier.
func TestParallelExecutionDatabaseErrorFinalize(t *testing.T) {
	var (
		recipient = common.Address{0xdd}
		injected  = errors.New("injected account fault")
	)
	env := newBALTestEnv(nil)
	err := runParallelWithReader(t, env,
		func(b *BlockGen) {
			b.SetCoinbase(common.Address{0xc0})
			b.AddWithdrawal(&types.Withdrawal{Validator: 1, Address: recipient, Amount: 0})
		},
		func(base state.Reader) state.Reader {
			return &faultyAccountReader{Reader: base, addr: recipient, err: injected}
		},
	)
	if !errors.Is(err, injected) {
		t.Fatalf("expected the injected database fault, got %v", err)
	}
	if !strings.Contains(err.Error(), "post-execution system calls") {
		t.Fatalf("expected the post-execution phase in the error, got %v", err)
	}
}
