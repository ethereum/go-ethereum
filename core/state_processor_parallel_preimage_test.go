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
	"context"
	"maps"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// hashCalldataCode hashes its own calldata, turning that calldata into a SHA3
// preimage observed by the VM:
//
//	CALLDATASIZE PUSH1 0 PUSH1 0 CALLDATACOPY   memory[0:size] = calldata
//	CALLDATASIZE PUSH1 0 KECCAK256 POP STOP     keccak256(memory[0:size])
var hashCalldataCode = []byte{
	0x36, 0x60, 0x00, 0x60, 0x00, 0x37,
	0x36, 0x60, 0x00, 0x20, 0x50, 0x00,
}

// TestParallelPreimageRecording verifies that preimage recording survives the
// BAL-driven parallel execution path. Preimages are recorded by opKeccak256
// into the EVM's current StateDB; on the parallel path the transactions and the
// system calls each run against an ephemeral StateDB, so what they record must
// be folded into the canonical statedb — the one WriteBlockWithState persists
// from — before those states are discarded.
func TestParallelPreimageRecording(t *testing.T) {
	contract := common.HexToAddress("0xc1")

	// Substituting two system contracts makes the pre- and post-execution
	// system calls hash something too, covering the two folds the transactions
	// cannot reach: system calls run against ephemeral states of their own.
	// Both call sites tolerate the substitution — ProcessBeaconBlockRoot
	// discards the call result, and processRequestsSystemCall skips empty
	// output. The withdrawal queue is called with no calldata, so it hashes the
	// empty string.
	env := newBALTestEnv(types.GenesisAlloc{
		contract:                      {Code: hashCalldataCode, Balance: common.Big0},
		params.BeaconRootsAddress:     {Nonce: 1, Code: hashCalldataCode, Balance: common.Big0},
		params.WithdrawalQueueAddress: {Nonce: 1, Code: hashCalldataCode, Balance: common.Big0},
	})

	// Several transactions with distinct calldata, so every worker result must
	// contribute its own preimage to the canonical state.
	inputs := make([][]byte, 4)
	for i := range inputs {
		inputs[i] = bytes.Repeat([]byte{0xa0 + byte(i)}, 32)
	}
	engine := beacon.New(ethash.NewFaker())
	_, blocks, _ := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, g *BlockGen) {
		for i, input := range inputs {
			g.AddTx(env.tx(uint64(i), &contract, common.Big0, 100_000, 0, input))
		}
	})
	block := blocks[0]
	beaconRoot := block.BeaconRoot()
	if beaconRoot == nil {
		t.Fatal("block carries no beacon root, the system-call fold would go unexercised")
	}
	bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, nil)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}
	defer bc.Stop()

	// process re-executes the block from the chain's genesis state and returns
	// the preimages the canonical statedb ends up holding. It pins the
	// execution path down using the same inputs Process itself evaluates, so a
	// run that quietly took the other path fails instead of passing vacuously,
	// and it requires every transaction to succeed: a missing preimage then
	// means the KECCAK256 was never reached. Note that a revert would not
	// explain one either, since preimages are not journaled.
	process := func(cfg vm.Config, wantParallel bool) map[common.Hash][]byte {
		t.Helper()

		statedb, err := bc.State()
		if err != nil {
			t.Fatalf("state: %v", err)
		}
		parallel := supportsParallelExecution(block, bc.Config(), statedb.Witness() != nil, cfg.Tracer != nil, cfg.DisableParallelExecution)
		if parallel != wantParallel {
			t.Fatalf("parallel execution eligibility is %v, want %v", parallel, wantParallel)
		}
		res, err := NewStateProcessor(bc).Process(context.Background(), block, statedb, nil, nil, cfg, nil)
		if err != nil {
			t.Fatalf("process: %v", err)
		}
		for i, receipt := range res.Receipts {
			if receipt.Status != types.ReceiptStatusSuccessful {
				t.Fatalf("tx %d failed", i)
			}
		}
		// Snapshot the map: Preimages returns the statedb's live one.
		return maps.Clone(statedb.Preimages())
	}

	// Sequential execution is the reference behavior, and asserting its content
	// keeps the comparison below from being satisfied by two empty maps.
	seqPreimages := process(vm.Config{EnablePreimageRecording: true, DisableParallelExecution: true}, false)
	for i, input := range inputs {
		if got, ok := seqPreimages[crypto.Keccak256Hash(input)]; !ok || !bytes.Equal(got, input) {
			t.Fatalf("sequential run lost preimage of tx %d: present=%v", i, ok)
		}
	}
	if got, ok := seqPreimages[crypto.Keccak256Hash(beaconRoot[:])]; !ok || !bytes.Equal(got, beaconRoot[:]) {
		t.Fatalf("sequential run lost the pre-execution system call preimage: present=%v", ok)
	}
	if _, ok := seqPreimages[crypto.Keccak256Hash(nil)]; !ok {
		t.Fatal("sequential run lost the post-execution system call preimage")
	}

	// Parallel execution must report exactly the same preimages.
	parPreimages := process(vm.Config{EnablePreimageRecording: true}, true)
	if !maps.EqualFunc(parPreimages, seqPreimages, bytes.Equal) {
		var missing, unexpected, differing []common.Hash
		for hash, want := range seqPreimages {
			switch got, ok := parPreimages[hash]; {
			case !ok:
				missing = append(missing, hash)
			case !bytes.Equal(got, want):
				differing = append(differing, hash)
			}
		}
		for hash := range parPreimages {
			if _, ok := seqPreimages[hash]; !ok {
				unexpected = append(unexpected, hash)
			}
		}
		t.Fatalf("parallel recorded %d preimages, sequential %d: missing %x, unexpected %x, differing %x",
			len(parPreimages), len(seqPreimages), missing, unexpected, differing)
	}
}
