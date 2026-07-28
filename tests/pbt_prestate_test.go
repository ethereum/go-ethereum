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

package tests

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestBinaryTreeForkRegistered checks that the fork the execution specs use
// for EIP-8297 resolves here and activates the binary tree, so ported
// fixtures have a target to run against.
func TestBinaryTreeForkRegistered(t *testing.T) {
	config, ok := Forks["BinaryTree"]
	if !ok {
		t.Fatal("BinaryTree fork is not registered")
	}
	if !config.IsPBT(big.NewInt(0), 0) {
		t.Fatal("BinaryTree fork does not activate the binary tree")
	}
	// The tree replaces the state commitment only: execution rules are
	// those of the latest scheduled fork.
	for _, tc := range []struct {
		name string
		on   bool
	}{
		{"cancun", config.IsCancun(big.NewInt(0), 0)},
		{"prague", config.IsPrague(big.NewInt(0), 0)},
		{"osaka", config.IsOsaka(big.NewInt(0), 0)},
	} {
		if !tc.on {
			t.Fatalf("BinaryTree fork does not carry %s rules", tc.name)
		}
	}
}

// TestPBTPreState builds a pre-state through the state-test harness and
// checks it is a binary tree holding the allocation.
func TestPBTPreState(t *testing.T) {
	addr := common.Address{1}
	alloc := types.GenesisAlloc{
		addr: {
			Balance: big.NewInt(1000),
			Nonce:   3,
			Code:    []byte{0x60, 0x01, 0x60, 0x02},
			Storage: map[common.Hash]common.Hash{
				{31: 5}: {31: 7}, // header stem
				{30: 4}: {31: 8}, // storage bucket
			},
		},
	}
	db := rawdb.NewMemoryDatabase()
	state := MakePBTPreState(db, alloc, false)
	defer state.Close()

	if !state.TrieDB.IsPBT() {
		t.Fatal("pre-state is not a binary tree")
	}
	if got := state.StateDB.GetBalance(addr).Uint64(); got != 1000 {
		t.Fatalf("balance %d, want 1000", got)
	}
	if got := state.StateDB.GetNonce(addr); got != 3 {
		t.Fatalf("nonce %d, want 3", got)
	}
	if got := state.StateDB.GetCodeSize(addr); got != 4 {
		t.Fatalf("code size %d, want 4", got)
	}
	for slot, want := range alloc[addr].Storage {
		if got := state.StateDB.GetState(addr, slot); got != want {
			t.Fatalf("slot %x = %x, want %x", slot, got, want)
		}
	}
	if !state.StateDB.HasStorage(addr) {
		t.Fatal("account reports no storage")
	}
	// The root must be the binary tree's, not the merkle-patricia empty.
	if root := state.StateDB.IntermediateRoot(false); root == types.EmptyRootHash {
		t.Fatal("state root is the merkle-patricia empty root")
	}
}
