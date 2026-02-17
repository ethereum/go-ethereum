// Copyright 2025 The go-ethereum Authors
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

package trie

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/holiman/uint256"
)

// TestInspect inspects a randomly generated account trie. It's useful for
// quickly verifying changes to the results display.
func TestInspect(t *testing.T) {
	db := newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme)
	trie, err := NewStateTrie(TrieID(types.EmptyRootHash), db)
	if err != nil {
		t.Fatalf("failed to create state trie: %v", err)
	}
	// Create a realistic looking account trie with storage.
	addresses, accounts := makeAccountsWithStorage(db, 11, true)
	for i := 0; i < len(addresses); i++ {
		trie.MustUpdate(crypto.Keccak256(addresses[i][:]), accounts[i])
	}
	// Insert the accounts into the trie and hash it
	root, nodes := trie.Commit(true)
	db.Update(root, types.EmptyRootHash, trienode.NewWithNodeSet(nodes))
	db.Commit(root)

	tempDir := t.TempDir()
	dumpPath := filepath.Join(tempDir, "trie-dump.bin")
	if err := Inspect(db, root, &InspectConfig{
		TopN:     1,
		DumpPath: dumpPath,
		Path:     filepath.Join(tempDir, "trie-summary.json"),
	}); err != nil {
		t.Fatalf("inspect failed: %v", err)
	}
	if err := Summarize(dumpPath, &InspectConfig{
		TopN: 1,
		Path: filepath.Join(tempDir, "trie-summary-reanalysis.json"),
	}); err != nil {
		t.Fatalf("summarize failed: %v", err)
	}
}

func makeAccountsWithStorage(db *testDb, size int, storage bool) (addresses [][20]byte, accounts [][]byte) {
	// Make the random benchmark deterministic
	random := rand.New(rand.NewSource(0))

	addresses = make([][20]byte, size)
	for i := 0; i < len(addresses); i++ {
		data := make([]byte, 20)
		random.Read(data)
		copy(addresses[i][:], data)
	}
	accounts = make([][]byte, len(addresses))
	for i := 0; i < len(accounts); i++ {
		var (
			nonce = uint64(random.Int63())
			root  = types.EmptyRootHash
			code  = crypto.Keccak256(nil)
		)
		if storage {
			trie := NewEmpty(db)
			for range random.Uint32()%256 + 1 { // non-zero
				k, v := make([]byte, 32), make([]byte, 32)
				random.Read(k)
				random.Read(v)
				trie.MustUpdate(k, v)
			}
			var nodes *trienode.NodeSet
			root, nodes = trie.Commit(true)
			db.Update(root, types.EmptyRootHash, trienode.NewWithNodeSet(nodes))
			db.Commit(root)
		}
		numBytes := random.Uint32() % 33 // [0, 32] bytes
		balanceBytes := make([]byte, numBytes)
		random.Read(balanceBytes)
		balance := new(uint256.Int).SetBytes(balanceBytes)
		data, _ := rlp.EncodeToBytes(&types.StateAccount{
			Nonce:    nonce,
			Balance:  balance,
			Root:     root,
			CodeHash: code,
		})
		accounts[i] = data
	}
	return addresses, accounts
}
