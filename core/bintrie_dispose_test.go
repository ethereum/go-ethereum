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
	"encoding/binary"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/triedb"
)

// merkleStateFixture commits genesis on the merkle trie and plants a binary
// tree record beside it, returning the database and a flat-state account.
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

// plantChainHeads writes the chain head pointers. Every one begins "Last",
// which is also the state-id prefix byte, so a family scan over it deletes
// the chain's head. Planting them makes that regression visible without a
// full node; only for tests that do not open a BlockChain.
func plantChainHeads(db ethdb.Database) {
	rawdb.WriteHeadHeaderHash(db, common.Hash{0x11})
	rawdb.WriteHeadBlockHash(db, common.Hash{0x22})
	rawdb.WriteHeadFastBlockHash(db, common.Hash{0x33})
}

// assertMerkleStateGone: every merkle family and marker is gone, and the
// binary namespace is intact.
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
	for _, head := range []struct {
		name string
		read func(ethdb.KeyValueReader) common.Hash
	}{
		{"head header", rawdb.ReadHeadHeaderHash},
		{"head block", rawdb.ReadHeadBlockHash},
		{"head fast block", rawdb.ReadHeadFastBlockHash},
	} {
		if head.read(db) == (common.Hash{}) {
			t.Fatalf("the disposal deleted the %s pointer: a state family scan reached the chain head keys", head.name)
		}
	}
}

// TestMerkleDisposalDeletesTheState pins what a full node reclaims: the whole
// merkle state, markers included, and nothing of the binary tree.
func TestMerkleDisposalDeletesTheState(t *testing.T) {
	db, _ := merkleStateFixture(t, merkleGenesis(t))
	plantChainHeads(db)
	disposeMerkleState(db, "", nil)
	assertMerkleStateGone(t, db)
}

// TestMerkleDisposalRespectsArchiveMode: serving pre-fork history is what an
// archive node is for, so it neither condemns nor deletes the state.
func TestMerkleDisposalRespectsArchiveMode(t *testing.T) {
	// A migration genesis, so a dual-tree datadir is legitimate.
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

// TestMerkleDisposalDeletesUnderALiveNode: a node that crossed the fork
// in-run holds the merkle tree as its canonical handle, and still reclaims
// the state without being restarted - the handle is retired first, so the
// deletion cannot be read as absent accounts.
func TestMerkleDisposalDeletesUnderALiveNode(t *testing.T) {
	genesis := migrationGenesis(t)
	db, addrHash := merkleStateFixture(t, genesis)

	chain, err := NewBlockChain(db, genesis, beacon.New(ethash.NewFaker()),
		DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	if chain.TrieDB().IsPBT() {
		t.Fatal("a migration chain at a pre-fork head opened on the binary tree")
	}
	// The window closes on the follower's loop, with no sync in flight.
	chain.follower.close()

	chain.disposeMerkle()
	if !rawdb.ReadPBTMerkleDisposed(db) {
		t.Fatal("the disposal was neither performed nor recorded")
	}
	<-chain.disposer.Load().done
	if rawdb.ReadAccountSnapshot(db, addrHash) != nil {
		t.Fatal("the running node kept its merkle flat state, so the space is only reclaimed by a restart")
	}
	assertMerkleStateGone(t, db)
	if _, err := chain.treeFor(false); err == nil {
		t.Fatal("a merkle handle was served after the state was disposed of")
	}
}

// TestDisposedDatadirRefusesAMerkleHead: a node whose head is back under the
// boundary would execute on condemned state, so opening fails loudly.
func TestDisposedDatadirRefusesAMerkleHead(t *testing.T) {
	genesis := migrationGenesis(t)
	db, _ := merkleStateFixture(t, genesis)
	rawdb.WritePBTMerkleDisposed(db)

	chain, err := NewBlockChain(db, genesis, beacon.New(ethash.NewFaker()),
		DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err == nil {
		chain.Stop()
		t.Fatal("a datadir whose merkle state was disposed of opened on a pre-fork head")
	}
	if !strings.Contains(err.Error(), "disposed of") {
		t.Fatalf("opened with %q, want the disposal refusal", err)
	}
}

// TestMerkleDisposalResumes: a deletion cut short reports itself unfinished
// and leaves the rest, which is what lets shutdown interrupt it.
func TestMerkleDisposalResumes(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	// Past one batch: the interrupt is only consulted after a batch write.
	var value [256]byte
	for i := range 4096 {
		var hash common.Hash
		binary.BigEndian.PutUint64(hash[:8], uint64(i))
		rawdb.WriteAccountSnapshot(db, hash, value[:])
	}
	interrupt := make(chan struct{})
	close(interrupt)

	deleted, done, err := rawdb.DeleteMerkleState(db, interrupt)
	if err != nil {
		t.Fatalf("interrupted deletion failed: %v", err)
	}
	if done {
		t.Fatal("an interrupted deletion reported itself finished")
	}
	if deleted == 0 {
		t.Fatal("the interrupted deletion deleted nothing at all")
	}
	it := db.NewIterator(rawdb.SnapshotAccountPrefix, nil)
	left := it.Next()
	it.Release()
	if !left {
		t.Fatal("the interrupt did not stop the deletion before the end")
	}

	// The next start, with nothing to interrupt it.
	rest, done, err := rawdb.DeleteMerkleState(db, nil)
	if err != nil {
		t.Fatalf("resumed deletion failed: %v", err)
	}
	if !done {
		t.Fatal("the resumed deletion did not finish")
	}
	if rest == 0 {
		t.Fatal("the resumed deletion found nothing left, so the first one was not interrupted")
	}
	for _, family := range rawdb.MerkleKeyFamilies {
		fit := db.NewIterator(family, nil)
		over := fit.Next()
		fit.Release()
		if over {
			t.Fatalf("merkle family %q survived the resumed deletion", family)
		}
	}
}

// TestArchiveKeepsTheMerkleHandle: the follower owns the only route to a
// merkle handle, so an archive node past the window kept the state and lost
// every way to read it - and the live node's decision keeps it too.
func TestArchiveKeepsTheMerkleHandle(t *testing.T) {
	genesis := migrationGenesis(t)
	db, _ := merkleStateFixture(t, genesis)
	rawdb.WritePBTMigrationDone(db)

	chain, err := NewBlockChain(db, genesis, beacon.New(ethash.NewFaker()),
		DefaultConfig().WithStateScheme(rawdb.PathScheme).WithArchive(true))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if err := chain.SettleMerkleDisposal(); err != nil {
		t.Fatalf("an archive node refused its own start: %v", err)
	}
	if _, err := chain.treeFor(false); err != nil {
		t.Fatalf("an archive node cannot reach the merkle state it kept: %v", err)
	}
	if rawdb.ReadPBTMerkleDisposed(db) {
		t.Fatal("an archive node condemned the state it is meant to keep")
	}
}

// TestOfflineStartLeavesTheDecision: NewBlockChain runs for every geth
// command, with whatever gcmode that command defaults to, so it must not
// decide the disposal - an archive node's state would go from an unrelated
// invocation. The live node decides, and a full node whose head is still
// under the boundary is refused rather than left running on a retired handle.
func TestOfflineStartLeavesTheDecision(t *testing.T) {
	genesis := migrationGenesis(t)
	db, addrHash := merkleStateFixture(t, genesis)
	rawdb.WritePBTMigrationDone(db)

	chain, err := NewBlockChain(db, genesis, beacon.New(ethash.NewFaker()),
		DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	if rawdb.ReadPBTMerkleDisposed(db) {
		t.Fatal("opening the chain condemned the merkle state; an offline command would do the same to an archive node")
	}
	if err := chain.SettleMerkleDisposal(); err == nil {
		t.Fatal("a full node at a pre-fork head settled the disposal instead of refusing")
	}
	if rawdb.ReadPBTMerkleDisposed(db) {
		t.Fatal("the refusal condemned the state")
	}
	if rawdb.ReadAccountSnapshot(db, addrHash) == nil {
		t.Fatal("the refusal deleted the state")
	}
}
