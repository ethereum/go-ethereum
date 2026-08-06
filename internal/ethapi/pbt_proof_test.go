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

package ethapi

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	corestate "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

// pbtProofState commits one account holding code and returns a state at the
// resulting root, alongside the trie database so a second trie can be opened.
func pbtProofState(t *testing.T, addr common.Address, code []byte) (*corestate.StateDB, *triedb.Database, common.Hash) {
	t.Helper()

	db := triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.PBTDefaults)
	sdb, err := corestate.New(types.EmptyBinaryHash, corestate.NewDatabase(db, nil))
	if err != nil {
		t.Fatal(err)
	}
	sdb.CreateAccount(addr)
	sdb.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
	sdb.AddBalance(addr, uint256.NewInt(1000), tracing.BalanceChangeUnspecified)
	sdb.SetCode(addr, code, tracing.CodeChangeUnspecified)
	root, err := sdb.Commit(1, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(root, false); err != nil {
		t.Fatal(err)
	}
	next, err := corestate.New(root, corestate.NewDatabase(db, nil))
	if err != nil {
		t.Fatal(err)
	}
	return next, db, root
}

// pathProof returns the node set a single-key proof of key emits.
func pathProof(t *testing.T, db *triedb.Database, root common.Hash, key []byte) []string {
	t.Helper()

	tr, err := bintrie.NewBinaryTrie(root, db)
	if err != nil {
		t.Fatal(err)
	}
	list := newDedupProofList()
	if err := tr.Prove(key, list); err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(list.list))
	for i, blob := range list.list {
		out[i] = string(blob)
	}
	return out
}

// TestPBTProofCoversResidentCodeLeaf pins that eth_getProof covers whichever
// of the code-hash and delegation leaves the account actually holds.
//
// The two are exclusive, so naming only one answers for half the accounts.
// Proofs of this tree do not ship whole stems - proof.go expands a group along
// the queried key alone - so a delegated account proved at 0 and 1 comes back
// with an absence witness where the code hash would be and nothing that
// substantiates the code hash the response reports.
func TestPBTProofCoversResidentCodeLeaf(t *testing.T) {
	for _, tc := range []struct {
		name     string
		code     []byte
		resident func(common.Address) []byte
		absent   func(common.Address) []byte
	}{
		{
			name:     "delegated account",
			code:     types.AddressToDelegation(common.Address{9}),
			resident: bintrie.DelegationKey,
			absent:   bintrie.CodeHashKey,
		},
		{
			name:     "contract",
			code:     []byte{0x60, 0x01, 0x60, 0x00, 0x55},
			resident: bintrie.CodeHashKey,
			absent:   bintrie.DelegationKey,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			addr := common.Address{1}
			sdb, db, root := pbtProofState(t, addr, tc.code)

			res, err := pbtProof(sdb, root, addr, nil, nil, sdb.GetCodeHash(addr))
			if err != nil {
				t.Fatal(err)
			}
			got := make(map[string]struct{}, len(res.AccountProof))
			for _, blob := range res.AccountProof {
				got[string(blob)] = struct{}{}
			}
			// Every node the resident leaf's own proof needs has to be in the
			// account proof, or the leaf is not provable from it.
			for _, want := range pathProof(t, db, root, tc.resident(addr)) {
				if _, ok := got[want]; !ok {
					t.Fatalf("the account proof is missing a node on the path to the resident code leaf: %s", want)
				}
			}
			// The absent one is proved too, as absence: which leaf is resident
			// is itself part of what the proof establishes.
			for _, want := range pathProof(t, db, root, tc.absent(addr)) {
				if _, ok := got[want]; !ok {
					t.Fatalf("the account proof does not witness the absence of the other code leaf: %s", want)
				}
			}
			if res.ProofFormat != "eip8297" {
				t.Fatalf("proof format %q, want eip8297", res.ProofFormat)
			}
		})
	}
}
