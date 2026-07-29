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

package pathdb

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/holiman/uint256"
)

// TestPBTFlatStateFollowsTheBranch pins the assumption the whole phase ordering
// rests on: that a reorg inside the layer window leaves flat state consistent.
//
// The binary tree cannot roll its persisted state back, so reorgs are handled
// by re-executing the winning branch forward from the closest ancestor whose
// state is still live. That only works if flat state is per-layer rather than
// global - if the data a layer contributed goes away with the layer, and a read
// on one branch never sees another branch's writes.
//
// This matters because `BINTRIE_FLAT_STATE_REORG_GAP.md` reached the opposite
// conclusion ("bintrie nodes with flat state enabled cannot handle reorgs") for
// an earlier flat-state design. That reasoning does not carry to the layer tree,
// but the claim is load-bearing enough to check rather than argue.
//
// Scope: this covers the window where the competing branches are still diff
// layers, which is the window reorgs are served from. Once a branch has been
// flushed into the disk layer its writes are no longer separable, and a fork
// below that point is exactly the case Recoverable reports as unreachable and
// the chain refuses by name - so there is no third behaviour to test here.
func TestPBTFlatStateFollowsTheBranch(t *testing.T) {
	var (
		disk    = rawdb.NewMemoryDatabase()
		db      = New(disk, &Config{TrieCleanSize: 1024, StateCleanSize: 1024, NoAsyncFlush: true}, true)
		account = common.Hash{0xaa}

		// base -> A -> B      (canonical)
		//      -> A'          (the branch that wins)
		base   = types.EmptyBinaryHash
		rootA  = common.Hash{0x0a}
		rootB  = common.Hash{0x0b}
		rootA2 = common.Hash{0x1a}
	)
	defer db.Close()

	// Distinct balances make it obvious which branch a read resolved on.
	set := func(balance uint64) *StateSetWithOrigin {
		blob := types.SlimAccountRLP(types.StateAccount{
			Nonce:    balance,
			Balance:  uint256.NewInt(balance),
			Root:     types.EmptyRootHash,
			CodeHash: types.EmptyCodeHash.Bytes(),
		})
		return NewStateSetWithOrigin(
			map[common.Hash][]byte{account: blob}, nil, nil, nil, false,
		)
	}
	if err := db.Update(rootA, base, 1, trienode.NewMergedNodeSet(), set(1)); err != nil {
		t.Fatal(err)
	}
	if err := db.Update(rootB, rootA, 2, trienode.NewMergedNodeSet(), set(2)); err != nil {
		t.Fatal(err)
	}
	// The competing branch forks from A, not from B.
	if err := db.Update(rootA2, rootA, 2, trienode.NewMergedNodeSet(), set(3)); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		root common.Hash
		want uint64
	}{
		{"canonical tip", rootB, 2},
		{"fork point", rootA, 1},
		{"competing branch", rootA2, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := db.StateReader(tc.root)
			if err != nil {
				t.Fatalf("state is not readable at %x: %v", tc.root, err)
			}
			acct, err := reader.Account(account)
			if err != nil {
				t.Fatal(err)
			}
			if acct == nil {
				t.Fatalf("account is absent at %x", tc.root)
			}
			if acct.Nonce != tc.want {
				t.Fatalf("read resolved on the wrong branch: nonce %d, want %d", acct.Nonce, tc.want)
			}
		})
	}
}
