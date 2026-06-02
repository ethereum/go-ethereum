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

package triedb

import (
	"bytes"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

// testAccount is a helper for building test state with deterministic ordering.
type testAccount struct {
	hash    common.Hash
	account types.StateAccount
	storage []testSlot // must be sorted by hash
}

type testSlot struct {
	hash  common.Hash
	value []byte
}

// buildExpectedRoot computes the state root from sorted test accounts using
// StackTrie (which requires sorted key insertion).
func buildExpectedRoot(t *testing.T, accounts []testAccount) common.Hash {
	t.Helper()
	// Sort accounts by hash
	sort.Slice(accounts, func(i, j int) bool {
		return bytes.Compare(accounts[i].hash[:], accounts[j].hash[:]) < 0
	})
	acctTrie := trie.NewStackTrie(nil)
	for i := range accounts {
		data, err := rlp.EncodeToBytes(&accounts[i].account)
		if err != nil {
			t.Fatal(err)
		}
		acctTrie.Update(accounts[i].hash[:], data)
	}
	return acctTrie.Hash()
}

// computeStorageRoot computes the storage trie root from sorted slots.
func computeStorageRoot(slots []testSlot) common.Hash {
	sort.Slice(slots, func(i, j int) bool {
		return bytes.Compare(slots[i].hash[:], slots[j].hash[:]) < 0
	})
	st := trie.NewStackTrie(nil)
	for _, s := range slots {
		st.Update(s.hash[:], s.value)
	}
	return st.Hash()
}

func TestGenerateTrieEmpty(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	if err := GenerateTrie(db, rawdb.HashScheme, types.EmptyRootHash); err != nil {
		t.Fatalf("GenerateTrie on empty state failed: %v", err)
	}
}

func TestGenerateTrieAccountsOnly(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	accounts := []testAccount{
		{
			hash: common.HexToHash("0x01"),
			account: types.StateAccount{
				Nonce:    1,
				Balance:  uint256.NewInt(100),
				Root:     types.EmptyRootHash,
				CodeHash: types.EmptyCodeHash.Bytes(),
			},
		},
		{
			hash: common.HexToHash("0x02"),
			account: types.StateAccount{
				Nonce:    2,
				Balance:  uint256.NewInt(200),
				Root:     types.EmptyRootHash,
				CodeHash: types.EmptyCodeHash.Bytes(),
			},
		},
	}
	for _, a := range accounts {
		rawdb.WriteAccountSnapshot(db, a.hash, types.SlimAccountRLP(a.account))
	}
	root := buildExpectedRoot(t, accounts)

	if err := GenerateTrie(db, rawdb.HashScheme, root); err != nil {
		t.Fatalf("GenerateTrie failed: %v", err)
	}
}

func TestGenerateTrieWithStorage(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	slots := []testSlot{
		{hash: common.HexToHash("0xaa"), value: []byte{0x01, 0x02, 0x03}},
		{hash: common.HexToHash("0xbb"), value: []byte{0x04, 0x05, 0x06}},
	}
	storageRoot := computeStorageRoot(slots)

	accounts := []testAccount{
		{
			hash: common.HexToHash("0x01"),
			account: types.StateAccount{
				Nonce:    1,
				Balance:  uint256.NewInt(100),
				Root:     storageRoot,
				CodeHash: types.EmptyCodeHash.Bytes(),
			},
			storage: slots,
		},
		{
			hash: common.HexToHash("0x02"),
			account: types.StateAccount{
				Nonce:    0,
				Balance:  uint256.NewInt(50),
				Root:     types.EmptyRootHash,
				CodeHash: types.EmptyCodeHash.Bytes(),
			},
		},
	}
	// Write account snapshots
	for _, a := range accounts {
		rawdb.WriteAccountSnapshot(db, a.hash, types.SlimAccountRLP(a.account))
	}
	// Write storage snapshots
	for _, a := range accounts {
		for _, s := range a.storage {
			rawdb.WriteStorageSnapshot(db, a.hash, s.hash, s.value)
		}
	}
	root := buildExpectedRoot(t, accounts)

	if err := GenerateTrie(db, rawdb.HashScheme, root); err != nil {
		t.Fatalf("GenerateTrie failed: %v", err)
	}
}

func TestGenerateTrieRootMismatch(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	acct := types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(100),
		Root:     types.EmptyRootHash,
		CodeHash: types.EmptyCodeHash.Bytes(),
	}
	rawdb.WriteAccountSnapshot(db, common.HexToHash("0x01"), types.SlimAccountRLP(acct))

	wrongRoot := common.HexToHash("0xdeadbeef")
	err := GenerateTrie(db, rawdb.HashScheme, wrongRoot)
	if err == nil {
		t.Fatal("expected error for root mismatch, got nil")
	}
}
