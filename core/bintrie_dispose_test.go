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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/triedb"
)

// merkleStateFixture commits the given genesis to disk on the merkle trie and
// plants a binary tree record beside it, returning the database and an
// account hash the merkle flat state holds.
func merkleStateFixture(t *testing.T, genesis *Genesis) (ethdb.Database, common.Hash) {
	t.Helper()
	db := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db, triedb.PathDefaults)
	genesis.MustCommit(db, tdb)
	if err := tdb.Close(); err != nil {
		t.Fatal(err)
	}
	var addrHash common.Hash
	for addr := range genesis.Alloc {
		addrHash = crypto.Keccak256Hash(addr.Bytes())
		break
	}
	if rawdb.ReadAccountSnapshot(db, addrHash) == nil {
		t.Fatal("fixture wrote no merkle flat state, so deleting it would prove nothing")
	}
	if len(rawdb.ReadAccountTrieNode(db, nil)) == 0 {
		t.Fatal("fixture wrote no merkle trie nodes")
	}
	// The binary namespace shares the database and must survive untouched.
	pbtdb := rawdb.NewTable(db, string(rawdb.PBTPrefix))
	rawdb.WritePBTFlatState(pbtdb)
	rawdb.WriteAccountTrieNode(pbtdb, nil, []byte{0x01, 0x02})
	rawdb.WriteSnapshotRoot(pbtdb, common.Hash{0xbb})
	return db, addrHash
}

// assertMerkleStateGone: every merkle family is empty, the markers that bless
// the flat state are gone, and the binary namespace is intact.
func assertMerkleStateGone(t *testing.T, db ethdb.Database) {
	t.Helper()
	for _, family := range rawdb.MerkleKeyFamilies {
		it := db.NewIterator(family, nil)
		left := it.Next()
		key := common.CopyBytes(it.Key())
		it.Release()
		if left {
			t.Fatalf("merkle family %q still holds %x", family, key)
		}
	}
	if rawdb.HasSnapshotRoot(db) {
		t.Fatal("the merkle snapshot root outlived the state it blesses")
	}
	if len(rawdb.ReadSnapshotGenerator(db)) != 0 {
		t.Fatal("the merkle snapshot generator marker survived the disposal")
	}
	pbtdb := rawdb.NewTable(db, string(rawdb.PBTPrefix))
	if !rawdb.ReadPBTFlatState(pbtdb) {
		t.Fatal("the disposal took the binary tree's attestation")
	}
	if len(rawdb.ReadAccountTrieNode(pbtdb, nil)) == 0 {
		t.Fatal("the disposal took the binary tree's nodes")
	}
}

// TestMerkleDisposalDeletesTheState pins what a full node reclaims once the
// migration window closes: the whole merkle state, markers included, and
// nothing of the binary tree that replaced it.
func TestMerkleDisposalDeletesTheState(t *testing.T) {
	db, _ := merkleStateFixture(t, merkleGenesis(t))
	disposeMerkleState(db, "", nil)
	assertMerkleStateGone(t, db)
}

// TestMerkleDisposalIsIdempotent: the disposal marker is never cleared, so
// every later start runs the deletion again. Doing so must be a no-op rather
// than an error, which is also what makes an interrupted run resumable.
func TestMerkleDisposalIsIdempotent(t *testing.T) {
	db, _ := merkleStateFixture(t, merkleGenesis(t))
	disposeMerkleState(db, "", nil)

	deleted, done, err := rawdb.DeleteMerkleState(db, nil)
	if err != nil {
		t.Fatalf("second disposal failed: %v", err)
	}
	if !done {
		t.Fatal("second disposal reported itself unfinished")
	}
	if deleted != 0 {
		t.Fatalf("second disposal deleted %d records, want none", deleted)
	}
	assertMerkleStateGone(t, db)
}

// TestMerkleDisposalRespectsArchiveMode pins the one node that keeps the
// merkle state: serving pre-fork history is what an archive node is for, so
// it must neither mark the state disposed of nor delete a record of it.
func TestMerkleDisposalRespectsArchiveMode(t *testing.T) {
	// A migration genesis, so a datadir holding both trees is legitimate: it
	// commits the same block as the merkle one, with the fork scheduled past
	// it.
	genesis := migrationGenesis(t)
	db, addrHash := merkleStateFixture(t, genesis)

	chain, err := NewBlockChain(db, genesis, beacon.New(ethash.NewFaker()),
		DefaultConfig().WithStateScheme(rawdb.PathScheme).WithArchive(true))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	chain.disposeMerkle()
	if rawdb.ReadPBTMerkleDisposed(db) {
		t.Fatal("an archive node marked its merkle state disposed of")
	}
	if rawdb.ReadAccountSnapshot(db, addrHash) == nil {
		t.Fatal("an archive node deleted its merkle flat state")
	}
	if !rawdb.HasSnapshotRoot(db) {
		t.Fatal("an archive node deleted its snapshot marker")
	}
}
