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
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/holiman/uint256"
)

// counterRuntime increments storage slot 0 on every call:
//
//	PUSH1 0x00 SLOAD PUSH1 0x01 ADD PUSH1 0x00 SSTORE STOP
var counterRuntime = common.FromHex("0x60005460010160005500")

// counterDeployer is init code that returns counterRuntime, so a creation
// transaction carrying it deploys the counter contract (exercising CodeChanges).
var counterDeployer = common.FromHex("0x600a600c600039600a6000f3" + "60005460010160005500")

var counterAddr = common.Address{0xc0, 0x11, 0xec}

// TestApplyBlockAccessList checks that ApplyBlockAccessList writes each kind of
// post-block value — balance, nonce, code, a storage set and a storage clear —
// into the state exactly as recorded in the access list.
func TestApplyBlockAccessList(t *testing.T) {
	var (
		eoa      = common.Address{0xaa}
		contract = common.Address{0xbb}
		keySet   = common.Hash{0x01}
		keyClear = common.Hash{0x02}
		valSet   = common.Hash{0x09}
	)

	// Build a synthetic access list via the construction API, then encode it to
	// the decoded form the sync path consumes.
	cb := bal.NewConstructionBlockAccessList()
	cb.BalanceChange(1, eoa, uint256.NewInt(1234))
	cb.NonceChange(eoa, 1, 7)
	cb.CodeChange(contract, 1, counterRuntime)
	cb.NonceChange(contract, 1, 1)
	cb.StorageWrite(1, contract, keySet, valSet)
	cb.StorageWrite(1, contract, keyClear, common.Hash{}) // post-value 0 clears the slot
	list := cb.ToEncodingObj()

	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatalf("new state: %v", err)
	}
	// Seed a non-zero value in the slot that the access list clears, so the clear
	// is actually observable.
	statedb.SetState(contract, keyClear, common.Hash{0x7f})

	ApplyBlockAccessList(statedb, list)

	if got := statedb.GetBalance(eoa); got.Uint64() != 1234 {
		t.Errorf("eoa balance = %d, want 1234", got.Uint64())
	}
	if got := statedb.GetNonce(eoa); got != 7 {
		t.Errorf("eoa nonce = %d, want 7", got)
	}
	if got := statedb.GetCode(contract); !bytes.Equal(got, counterRuntime) {
		t.Errorf("contract code = %x, want %x", got, counterRuntime)
	}
	if got := statedb.GetNonce(contract); got != 1 {
		t.Errorf("contract nonce = %d, want 1", got)
	}
	if got := statedb.GetState(contract, keySet); got != valSet {
		t.Errorf("set slot = %x, want %x", got, valSet)
	}
	if got := statedb.GetState(contract, keyClear); got != (common.Hash{}) {
		t.Errorf("cleared slot = %x, want zero", got)
	}
}

// balTx builds a signed transaction for the funded sender of the given test env.
func balTx(t *testing.T, env *balTestEnv, nonce uint64, to *common.Address, value *big.Int, data []byte) *types.Transaction {
	t.Helper()
	tx, err := types.SignTx(types.NewTx(&types.DynamicFeeTx{
		ChainID:   env.cfg.ChainID,
		Nonce:     nonce,
		To:        to,
		Value:     value,
		Gas:       500000, // a value-bearing transfer runs EIP-7708 machinery (~205k gas)
		GasFeeCap: big.NewInt(1_000_000_000_000),
		GasTipCap: big.NewInt(1_000_000_000),
		Data:      data,
	}), env.signer, env.key)
	if err != nil {
		t.Fatalf("sign tx: %v", err)
	}
	return tx
}

// balBlocks builds a short chain that exercises every access-list change kind:
// value transfers (balance/nonce, recipient credit), calls to a pre-deployed
// contract (storage writes), and a contract creation (code). The counter's slot
// 0 ends up incremented once per call across the chain.
func balBlocks(t *testing.T) (*balTestEnv, consensus.Engine, []*types.Block, int) {
	t.Helper()
	env := newBALTestEnv(types.GenesisAlloc{
		counterAddr: {Code: counterRuntime, Balance: common.Big0},
	})
	engine := beacon.New(ethash.NewFaker())
	addr1 := common.Address{0x11}

	calls := 0
	_, blocks, _ := GenerateChainWithGenesis(env.gspec, engine, 3, func(i int, b *BlockGen) {
		n := b.TxNonce(env.from)
		switch i {
		case 0: // two contract calls: storage writes on an existing contract
			b.AddTx(balTx(t, env, n, &counterAddr, common.Big0, nil))
			b.AddTx(balTx(t, env, n+1, &counterAddr, common.Big0, nil))
			calls += 2
		case 1: // a value transfer (balance/nonce/recipient) plus a call
			b.AddTx(balTx(t, env, n, &addr1, big.NewInt(5_000_000_000), nil))
			b.AddTx(balTx(t, env, n+1, &counterAddr, common.Big0, nil))
			calls++
		case 2: // a contract creation (code change) plus a call
			b.AddTx(balTx(t, env, n, nil, common.Big0, counterDeployer))
			b.AddTx(balTx(t, env, n+1, &counterAddr, common.Big0, nil))
			calls++
		}
	})
	for _, blk := range blocks {
		if blk.AccessList() == nil {
			t.Fatalf("block #%d carries no access list", blk.NumberU64())
		}
	}
	return env, engine, blocks, calls
}

// TestInsertChainReconstructsFinalizedState checks the wired-in fast path: with
// reconstruction enabled and the whole run finalized, InsertChain rebuilds each
// block's state from its access list (no execution), reaches the correct head
// and state, and stores no receipts.
func TestInsertChainReconstructsFinalizedState(t *testing.T) {
	env, engine, blocks, calls := balBlocks(t)

	cfg := DefaultConfig()
	cfg.BALStateReconstruction = true
	bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, cfg)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}
	defer bc.Stop()

	// Mark the whole run as finalized, then catch up to it.
	bc.SetFinalized(blocks[len(blocks)-1].Header())
	if n, err := bc.InsertChain(blocks); err != nil {
		t.Fatalf("insert %d/%d: %v", n, len(blocks), err)
	}

	head := blocks[len(blocks)-1]
	if got := bc.CurrentBlock(); got.Hash() != head.Hash() || got.Root != head.Root() {
		t.Fatalf("head = #%d %s (root %s), want #%d %s (root %s)", got.Number, got.Hash(), got.Root, head.NumberU64(), head.Hash(), head.Root())
	}
	// No block was executed, so none stored receipts, even though all carry txs.
	for _, blk := range blocks {
		if r := bc.GetReceiptsByHash(blk.Hash()); len(r) != 0 {
			t.Fatalf("block #%d has %d receipts, want 0 (executionless)", blk.NumberU64(), len(r))
		}
	}
	// The reconstructed state is real: the counter slot equals the call count.
	st, err := bc.State()
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if got := st.GetState(counterAddr, common.Hash{}).Big().Int64(); got != int64(calls) {
		t.Fatalf("counter slot = %d, want %d", got, calls)
	}
}

// TestInsertChainExecutesWhenReconstructionDisabled is the control: with the
// fast path off, the same blocks execute normally, reaching the identical state
// root but storing receipts.
func TestInsertChainExecutesWhenReconstructionDisabled(t *testing.T) {
	env, engine, blocks, _ := balBlocks(t)

	bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, DefaultConfig())
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}
	defer bc.Stop()

	bc.SetFinalized(blocks[len(blocks)-1].Header()) // finality alone must not trigger the fast path
	if n, err := bc.InsertChain(blocks); err != nil {
		t.Fatalf("insert %d/%d: %v", n, len(blocks), err)
	}

	head := blocks[len(blocks)-1]
	if got := bc.CurrentBlock(); got.Root != head.Root() {
		t.Fatalf("head root = %s, want %s", got.Root, head.Root())
	}
	// Executed blocks store receipts for their transactions.
	if r := bc.GetReceiptsByHash(blocks[0].Hash()); len(r) != len(blocks[0].Transactions()) {
		t.Fatalf("block #%d has %d receipts, want %d (executed)", blocks[0].NumberU64(), len(r), len(blocks[0].Transactions()))
	}
}

// TestProcessBlockFromAccessListRejectsWrongState checks the safety anchor:
// applying a correct access list onto the wrong parent state yields a state root
// that does not match the header, so reconstruction is rejected (and the caller
// falls back to full execution).
func TestProcessBlockFromAccessListRejectsWrongState(t *testing.T) {
	env, engine, blocks, _ := balBlocks(t)

	cfg := DefaultConfig()
	cfg.BALStateReconstruction = true
	bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, cfg)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}
	defer bc.Stop()

	// Reconstruct block #1 from an empty parent state instead of genesis.
	_, err = bc.processBlockFromAccessList(types.EmptyRootHash, blocks[0], ExecuteConfig{})
	if err == nil {
		t.Fatal("expected reconstruction onto the wrong parent state to be rejected")
	}
	if !strings.Contains(err.Error(), "state root mismatch") {
		t.Fatalf("expected state root mismatch, got: %v", err)
	}
}
