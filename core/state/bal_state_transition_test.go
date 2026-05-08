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

package state

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/holiman/uint256"
)

// TestBALStateTransition_EIP7702DelegationClear locks in the fix for the
// `if len(code) > 0` regression that skipped legitimate EIP-7702 delegation
// clears (encoded as a non-nil empty byte slice in the BAL). After the fix
// the gate is `if code != nil`, so a delegation clear correctly resets
// `acct.CodeHash` to `EmptyCodeHash` and is NOT counted as a deletion.
func TestBALStateTransition_EIP7702DelegationClear(t *testing.T) {
	addr := common.HexToAddress("0x000000000000000000000000000000000000aaaa")

	// Pre-state: account already holds a 7702 delegation (non-empty code).
	sdb := NewDatabaseForTesting()
	prestate, _ := New(types.EmptyRootHash, sdb)
	delegationCode := append([]byte{0xef, 0x01, 0x00}, common.HexToAddress("0xbeef").Bytes()...)
	prestate.SetBalance(addr, uint256.NewInt(1e18), tracing.BalanceChangeUnspecified)
	prestate.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
	prestate.SetCode(addr, delegationCode, tracing.CodeChangeUnspecified)
	parentRoot, err := prestate.Commit(0, false, false)
	if err != nil {
		t.Fatalf("Commit prestate: %v", err)
	}
	if err := sdb.TrieDB().Commit(parentRoot, false); err != nil {
		t.Fatalf("TrieDB Commit: %v", err)
	}

	// Build a BAL whose only mutation is a 7702 delegation clear.
	// Code is non-nil but length zero — the canonical encoding.
	construction := make(bal.ConstructionBlockAccessList)
	construction.AccumulateMutations(bal.StateMutations{
		addr: bal.AccountMutations{Code: bal.ContractCode{}},
	}, 0)
	accessList := construction.ToEncodingObj()

	// Synthesize a zero-tx block carrying the access list.
	block := types.NewBlockWithHeader(&types.Header{Number: common.Big1}).WithAccessList(accessList)

	// Run the BAL state transition.
	reader, err := sdb.Reader(parentRoot)
	if err != nil {
		t.Fatalf("Reader: %v", err)
	}
	bst, err := NewBALStateTransition(block, reader, sdb, parentRoot)
	if err != nil {
		t.Fatalf("NewBALStateTransition: %v", err)
	}
	bst.IntermediateRoot(false)
	if err := bst.Error(); err != nil {
		t.Fatalf("IntermediateRoot: %v", err)
	}

	post, ok := bst.postStates[addr]
	if !ok {
		t.Fatal("post-state must exist for the address (account is updated, not deleted)")
	}
	if !bytes.Equal(post.CodeHash, types.EmptyCodeHash.Bytes()) {
		t.Fatalf("CodeHash: got %x, want %x", post.CodeHash, types.EmptyCodeHash.Bytes())
	}
	if d := bst.Deletions().Accounts; d != 0 {
		t.Fatalf("Deletions.Accounts: got %d, want 0 (delegation clear is not a deletion)", d)
	}
	if _, deleted := bst.deletions[addr]; deleted {
		t.Fatal("address must not be in s.deletions after a 7702 clear")
	}
}
