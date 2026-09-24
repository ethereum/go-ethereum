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
	"crypto/ecdsa"
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

// migrationForkTime is past every generated block.
const migrationForkTime = uint64(1) << 40

// migrationChainGenesis is the PBT chain genesis with the fork past genesis.
func migrationChainGenesis(t *testing.T) (*Genesis, *ecdsa.PrivateKey, common.Address, common.Address) {
	t.Helper()
	genesis, key, sender, recipient := pbtChainGenesis(t)
	cfg := *genesis.Config
	forkTime := migrationForkTime
	cfg.BinaryTrieTime = &forkTime
	genesis.Config = &cfg
	return genesis, key, sender, recipient
}

// migrationGenesis is migrationChainGenesis for tests that only need the spec.
func migrationGenesis(t *testing.T) *Genesis {
	t.Helper()
	genesis, _, _, _ := migrationChainGenesis(t)
	return genesis
}

// merkleGenesis clones the PBT chain genesis with the fork removed entirely.
func merkleGenesis(t *testing.T) *Genesis {
	t.Helper()
	genesis, _, _, _ := pbtChainGenesis(t)
	cfg := *genesis.Config
	cfg.BinaryTrieTime = nil
	genesis.Config = &cfg
	return genesis
}

// TestStateModeResolution pins the three-way split.
func TestStateModeResolution(t *testing.T) {
	native, _, _, _ := pbtChainGenesis(t)
	for _, tc := range []struct {
		name    string
		genesis *Genesis
		want    stateMode
	}{
		{"no fork", merkleGenesis(t), modeMPT},
		{"fork at genesis", native, modePBTNative},
		{"fork past genesis", migrationGenesis(t), modeMigration},
	} {
		mode, _, err := resolveStateMode(rawdb.NewMemoryDatabase(), tc.genesis)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if mode != tc.want {
			t.Errorf("%s: mode = %d, want %d", tc.name, mode, tc.want)
		}
	}
}

// TestMigrationGenesisIsMerkle pins that the fork does not move the genesis.
func TestMigrationGenesisIsMerkle(t *testing.T) {
	var (
		migration       = migrationGenesis(t)
		merkle          = merkleGenesis(t)
		native, _, _, _ = pbtChainGenesis(t)
	)
	if migration.IsPBT() {
		t.Fatal("a migration genesis claims to commit with the binary tree")
	}
	if !native.IsPBT() {
		t.Fatal("a genesis-active fork does not commit with the binary tree")
	}
	if got, want := migration.ToBlock().Hash(), merkle.ToBlock().Hash(); got != want {
		t.Fatalf("migration genesis hash %x, want the merkle hash %x", got, want)
	}
	if got := migration.ToBlock().Hash(); got == native.ToBlock().Hash() {
		t.Fatalf("migration genesis hash %x equals the binary-tree hash", got)
	}
}

// TestNewBlockChainMigrationMode pins that a migration chain opens on the
// merkle trie, tolerates a pre-populated binary namespace (an anchor import
// may precede the first start), and resolves its stored mode on reopen.
func TestNewBlockChainMigrationMode(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	rawdb.WritePBTFlatState(rawdb.NewTable(db, string(rawdb.PBTPrefix)))

	chain := openMigrationChain(t, db, migrationGenesis(t))
	if chain.TrieDB().IsPBT() {
		t.Fatal("migration chain opened on the binary tree")
	}
	chain.Stop()
	mode, _, err := resolveStateMode(db, nil)
	if err != nil {
		t.Fatal(err)
	}
	if mode != modeMigration {
		t.Fatalf("stored mode = %d, want %d", mode, modeMigration)
	}
}

// TestMigrationRequiresPathScheme pins the path-scheme requirement.
func TestMigrationRequiresPathScheme(t *testing.T) {
	engine := beacon.New(ethash.NewFaker())
	_, err := NewBlockChain(rawdb.NewMemoryDatabase(), migrationGenesis(t), engine, DefaultConfig().WithStateScheme(rawdb.HashScheme))
	if err == nil {
		t.Fatal("migration accepted the hash scheme")
	}
}

// TestUnscheduledPBTStateStillRefused pins the narrowed reopen guard.
func TestUnscheduledPBTStateStillRefused(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	rawdb.WritePBTFlatState(rawdb.NewTable(db, string(rawdb.PBTPrefix)))

	engine := beacon.New(ethash.NewFaker())
	if _, err := NewBlockChain(db, merkleGenesis(t), engine, DefaultConfig().WithStateScheme(rawdb.PathScheme)); err == nil {
		t.Fatal("binary state accepted under a config that never schedules the fork")
	}
}

// TestMigrationDoneSkipsFollower pins the terminal marker.
func TestMigrationDoneSkipsFollower(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	rawdb.WritePBTMigrationDone(db)

	chain := openMigrationChain(t, db, migrationGenesis(t))
	defer chain.Stop()
	if chain.follower != nil {
		t.Fatal("a finished migration started a follower")
	}
	if p := chain.MigrationProgress(); p.Phase != "done" {
		t.Fatalf("progress phase %q, want done", p.Phase)
	}
}

// merkleStateFixture commits genesis on the merkle trie beside a binary record;
// it returns the database and a flat-state account hash.
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
		t.Fatal("fixture has no merkle flat state")
	}
	// The binary namespace shares the database.
	pbtdb := rawdb.NewTable(db, string(rawdb.PBTPrefix))
	rawdb.WritePBTFlatState(pbtdb)
	rawdb.WriteAccountTrieNode(pbtdb, nil, []byte{0x01, 0x02})
	return db, addrHash
}

// TestMerkleDisposalDeletesUnderALiveNode: a node that crossed the fork in-run
// retires bc.triedb and deletes the merkle state without a restart.
func TestMerkleDisposalDeletesUnderALiveNode(t *testing.T) {
	genesis := migrationGenesis(t)
	db, addrHash := merkleStateFixture(t, genesis)

	chain := openMigrationChain(t, db, genesis)
	defer chain.Stop()
	if chain.TrieDB().IsPBT() {
		t.Fatal("opened on the binary tree")
	}
	// Stop the loop the close would run on.
	chain.follower.close()

	chain.live.Store(true)
	chain.disposeMerkle()
	<-chain.disposer.Load().done

	if rawdb.ReadAccountSnapshot(db, addrHash) != nil {
		t.Fatal("merkle flat state survived")
	}
	for _, family := range rawdb.MerkleKeyFamilies {
		it := db.NewIterator(family, nil)
		left := it.Next()
		it.Release()
		if left {
			t.Fatalf("merkle family %q survived", family)
		}
	}
	if rawdb.HasSnapshotRoot(db) {
		t.Fatal("snapshot root survived")
	}
	// Head pointers begin "Last", the state-id prefix byte.
	if rawdb.ReadHeadBlockHash(db) == (common.Hash{}) || rawdb.ReadHeadHeaderHash(db) == (common.Hash{}) {
		t.Fatal("head pointers deleted")
	}
	pbtdb := rawdb.NewTable(db, string(rawdb.PBTPrefix))
	if !rawdb.ReadPBTFlatState(pbtdb) || len(rawdb.ReadAccountTrieNode(pbtdb, nil)) == 0 {
		t.Fatal("binary namespace touched")
	}
	if _, err := chain.treeFor(false); err == nil {
		t.Fatal("merkle handle served after disposal")
	}
}

// TestArchiveKeepsTheMerkleState: an archive node keeps the merkle state and
// the follower that reaches it.
func TestArchiveKeepsTheMerkleState(t *testing.T) {
	genesis := migrationGenesis(t)
	db, addrHash := merkleStateFixture(t, genesis)
	rawdb.WritePBTMigrationDone(db)

	chain, err := NewBlockChain(db, genesis, beacon.New(ethash.NewFaker()),
		DefaultConfig().WithStateScheme(rawdb.PathScheme).WithArchive(true))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if err := chain.SettleMerkleDisposal(); err != nil {
		t.Fatalf("settle: %v", err)
	}
	if rawdb.ReadPBTMerkleDisposed(db) {
		t.Fatal("archive node wrote the disposal marker")
	}
	if rawdb.ReadAccountSnapshot(db, addrHash) == nil {
		t.Fatal("merkle flat state deleted")
	}
	if chain.follower == nil {
		t.Fatal("archive node dropped the follower")
	}
}

// TestOfflineStartLeavesTheDecision: only the live node decides the disposal,
// and it refuses a full node at a pre-fork head.
func TestOfflineStartLeavesTheDecision(t *testing.T) {
	genesis := migrationGenesis(t)
	db, addrHash := merkleStateFixture(t, genesis)
	rawdb.WritePBTMigrationDone(db)

	chain := openMigrationChain(t, db, genesis)
	defer chain.Stop()
	// A window closing in an offline command, e.g. geth import.
	chain.disposeMerkle()
	if rawdb.ReadPBTMerkleDisposed(db) {
		t.Fatal("offline process wrote the disposal marker")
	}
	if err := chain.SettleMerkleDisposal(); err == nil {
		t.Fatal("settle succeeded at a pre-fork head")
	}
	if rawdb.ReadAccountSnapshot(db, addrHash) == nil {
		t.Fatal("merkle flat state deleted")
	}
}

// TestDisposedDatadirRefusesAMerkleHead: a disposed datadir refuses to open at a
// pre-fork head.
func TestDisposedDatadirRefusesAMerkleHead(t *testing.T) {
	genesis := migrationGenesis(t)
	db, _ := merkleStateFixture(t, genesis)
	rawdb.WritePBTMerkleDisposed(db)

	_, err := NewBlockChain(db, genesis, beacon.New(ethash.NewFaker()),
		DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err == nil || !strings.Contains(err.Error(), "disposed") {
		t.Fatalf("err = %v, want disposal refusal", err)
	}
}
