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
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestAttestFlatState pins the gate protecting binary-tree flat state.
//
// Flat state for the binary tree can only be accumulated forward from block
// commits: the tree keys its leaves by a hash of the address and keeps no
// preimages, so nothing can walk it back into an address-keyed store. A store
// that missed writes therefore cannot be repaired, only rebuilt.
//
// That makes opening one a correctness question rather than a performance one.
// Once flat state is treated as complete, a miss is not an error but the answer
// "this account does not exist", returned with no error so the trie behind it
// is never consulted, and then cached. A database written before flat state was
// accumulated has to be refused, not served.
func TestAttestFlatState(t *testing.T) {
	t.Run("fresh database attests itself", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		if err := attestFlatState(db, false); err != nil {
			t.Fatalf("a fresh database was refused: %v", err)
		}
		if !rawdb.ReadPBTFlatState(db) {
			t.Fatal("attestation was not recorded")
		}
		// Reopening keeps working.
		if err := attestFlatState(db, false); err != nil {
			t.Fatalf("an attested database was refused on reopen: %v", err)
		}
	})

	t.Run("fresh database in read-only mode is refused", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		err := attestFlatState(db, true)
		if err == nil {
			t.Fatal("a read-only database was attested, which requires a write")
		}
		if !strings.Contains(err.Error(), "read-only") {
			t.Fatalf("refusal does not name the cause: %v", err)
		}
	})

	t.Run("database with state but no attestation is refused", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		// Stand in for a database written while flat state was discarded:
		// it carries persisted state, but was never attested.
		rawdb.WritePersistentStateID(db, 42)

		err := attestFlatState(db, false)
		if err == nil {
			t.Fatal("a database predating flat state was accepted; every account in it would read as non-existent")
		}
		if !strings.Contains(err.Error(), "resync") {
			t.Fatalf("refusal does not say what to do about it: %v", err)
		}
		if rawdb.ReadPBTFlatState(db) {
			t.Fatal("a refused database was attested anyway")
		}
	})

	t.Run("database with trie data but no attestation is refused", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		rawdb.WriteAccountTrieNode(db, nil, []byte{0x02, 0xff})

		if err := attestFlatState(db, false); err == nil {
			t.Fatal("a database carrying trie data but no attestation was accepted")
		}
	})

	t.Run("chain data sharing the namespace prefix is not debris", func(t *testing.T) {
		// The namespace prefix is a block body's prefix too; chain data is
		// not tree state.
		outer := rawdb.NewMemoryDatabase()
		rawdb.WriteBody(outer, common.Hash{0x01}, 1, &types.Body{})

		db := rawdb.NewTable(outer, string(rawdb.PBTPrefix))
		if err := attestFlatState(db, false); err != nil {
			t.Fatalf("a block body was mistaken for binary tree state: %v", err)
		}
	})

	t.Run("database with conversion debris but no attestation is refused", func(t *testing.T) {
		// Scan-phase conversion debris: flat records only, no state id, no
		// root node, no attestation.
		db := rawdb.NewMemoryDatabase()
		rawdb.WriteAccountSnapshot(db, common.Hash{0x01}, []byte{0x01})

		if err := attestFlatState(db, false); err == nil {
			t.Fatal("a database carrying flat-state debris but no attestation was accepted")
		}
		if rawdb.ReadPBTFlatState(db) {
			t.Fatal("a refused database was attested anyway")
		}
	})
}
