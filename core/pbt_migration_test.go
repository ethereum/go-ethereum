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
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
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

// openImportChain opens a migration chain whose binary tree belongs to an
// import running beside it.
func openImportChain(t *testing.T, db ethdb.Database, genesis *Genesis, force bool) *BlockChain {
	t.Helper()
	cfg := DefaultConfig().WithStateScheme(rawdb.PathScheme)
	cfg.PBTImport, cfg.PBTImportForce = true, force
	chain, err := NewBlockChain(db, genesis, beacon.New(ethash.NewFaker()), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return chain
}

// awaitShadowRecord polls for a block's shadow root, whatever the follower
// latched before.
func awaitShadowRecord(t *testing.T, db ethdb.Database, block *types.Block) common.Hash {
	t.Helper()
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(10 * time.Millisecond) {
		if root, ok := rawdb.ReadShadowStateRoot(db, block.Hash(), block.NumberU64()); ok {
			return root
		}
	}
	t.Fatalf("shadow never recorded block %d", block.NumberU64())
	return common.Hash{}
}

// TestPBTImportNeedsAScheduledTree: an import seeds the tree the follower
// carries to the fork, so it needs a scheduled fork; past the fork the flags
// are dropped rather than refused.
func TestPBTImportNeedsAScheduledTree(t *testing.T) {
	cfg := DefaultConfig().WithStateScheme(rawdb.PathScheme)
	cfg.PBTImport = true
	engine := beacon.New(ethash.NewFaker())
	if _, err := NewBlockChain(rawdb.NewMemoryDatabase(), merkleGenesis(t), engine, cfg); err == nil || !strings.Contains(err.Error(), "binaryTrieTime") {
		t.Fatalf("err = %v, want the missing schedule", err)
	}
	db := rawdb.NewMemoryDatabase()
	head := types.NewBlockWithHeader(&types.Header{Number: big.NewInt(1), Time: migrationForkTime, Difficulty: common.Big0})
	rawdb.WriteBlock(db, head)
	rawdb.WriteCanonicalHash(db, head.Hash(), 1)
	rawdb.WriteHeadBlockHash(db, head.Hash())
	chain, err := NewBlockChain(db, migrationGenesis(t), engine, cfg)
	if err != nil {
		t.Fatalf("post-fork head refused the flags: %v", err)
	}
	defer chain.Stop()
	if chain.ImportsBinaryTree() {
		t.Fatal("import kept past the fork")
	}
}

// TestPBTImportHoldsTheTree: while an import owns the binary tree, blocks keep
// importing and nothing opens the namespace, neither the follower nor a state
// probe falling through to it; the release hands the tree to the follower.
func TestPBTImportHoldsTheTree(t *testing.T) {
	genesis, db, blocks, u := generateMigrationChain(t, 3)
	universeWithAlloc(u, genesis)
	chain := openImportChain(t, db, genesis, false)
	defer chain.Stop()
	advance(t, chain, blocks)
	head := blocks[len(blocks)-1]

	if err := chain.follower.waitCaughtUp(head.NumberU64(), head.Hash(), 10*time.Second); !errors.Is(err, errPBTImportPending) {
		t.Fatalf("follower stall %v, want the import hold", err)
	}
	if chain.HasState(common.Hash{0x01}) {
		t.Fatal("unknown root reported present")
	}
	pbtdb := rawdb.NewTable(db, string(rawdb.PBTPrefix))
	if _, _, ok := rawdb.ReadPBTAnchor(pbtdb); ok || rawdb.ReadPBTFlatState(pbtdb) {
		t.Fatal("the held namespace was seeded or attested")
	}
	if p := chain.MigrationProgress(); p.Phase != "importing" {
		t.Fatalf("progress phase %q, want importing", p.Phase)
	}

	chain.FinishPBTImport(nil)
	if rawdb.ReadPBTImportPending(db) {
		t.Fatal("hold survived the release")
	}
	if root, want := awaitShadowRecord(t, db, head), convertCanonical(t, chain, head.Header(), u); root != want {
		t.Fatalf("shadow root %x after the release, converting the canonical state says %x", root, want)
	}
}

// TestPBTImportFailureKeepsTheHold: a failed import leaves the tree held and
// says why.
func TestPBTImportFailureKeepsTheHold(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 2)
	chain := openImportChain(t, db, genesis, false)
	defer chain.Stop()
	advance(t, chain, blocks)

	chain.FinishPBTImport(errors.New("snapshot leaves rebuild to the wrong root"))
	p := chain.MigrationProgress()
	if p.Phase != "importFailed" || p.Binary == nil || !strings.Contains(p.Binary.Error, "wrong root") {
		t.Fatalf("progress %+v, want importFailed with the cause", p)
	}
	if !rawdb.ReadPBTImportPending(db) {
		t.Fatal("a failed import released the tree")
	}
	head := blocks[len(blocks)-1]
	if err := chain.follower.waitCaughtUp(head.NumberU64(), head.Hash(), 10*time.Second); !errors.Is(err, errPBTImportPending) {
		t.Fatalf("follower stall %v, want the import hold", err)
	}
}

// TestUnfinishedPBTImportHeldWithoutFlags: a restart without the import
// flags must neither open nor crash on what an unfinished import wrote -
// pathdb refuses an unattested namespace fatally - and keeps importing blocks.
func TestUnfinishedPBTImportHeldWithoutFlags(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 2)
	pbtdb := rawdb.NewTable(db, string(rawdb.PBTPrefix))
	rawdb.WriteAccountTrieNode(pbtdb, nil, []byte{0x01, 0x02})
	rawdb.WritePBTImportPending(db)

	chain := openMigrationChain(t, db, genesis)
	defer chain.Stop()
	advance(t, chain, blocks)
	head := blocks[len(blocks)-1]
	if err := chain.follower.waitCaughtUp(head.NumberU64(), head.Hash(), 10*time.Second); !errors.Is(err, errPBTImportPending) {
		t.Fatalf("follower stall %v, want the import hold", err)
	}
	if chain.HasState(common.Hash{0x01}) {
		t.Fatal("unknown root reported present")
	}
	p := chain.MigrationProgress()
	if p.Phase != "importFailed" || p.Binary == nil || p.Binary.Error != errPBTImportUnfinished.Error() {
		t.Fatalf("progress %+v, want importFailed asking for the flags", p)
	}
}

// TestFinishedPBTImportHold: an attested, anchored tree is finished. A hold
// left over from a run that died before releasing is cleared, an import of
// the same tree is not held for, and only force holds it for a replacement.
func TestFinishedPBTImportHold(t *testing.T) {
	finished := func() ethdb.Database {
		db := rawdb.NewMemoryDatabase()
		pbtdb := rawdb.NewTable(db, string(rawdb.PBTPrefix))
		rawdb.WritePBTAnchor(pbtdb, 7, common.Hash{0x07})
		rawdb.WritePBTFlatState(pbtdb)
		rawdb.WritePBTImportPending(db)
		return db
	}
	genesis := migrationGenesis(t)

	db := finished()
	openMigrationChain(t, db, genesis).Stop()
	if rawdb.ReadPBTImportPending(db) {
		t.Fatal("stale hold kept on a finished tree")
	}
	db = finished()
	openImportChain(t, db, genesis, false).Stop()
	if rawdb.ReadPBTImportPending(db) {
		t.Fatal("an import without force held a finished tree")
	}
	db = finished()
	openImportChain(t, db, genesis, true).Stop()
	if !rawdb.ReadPBTImportPending(db) {
		t.Fatal("a forced import did not hold the tree it replaces")
	}
}

// TestImportFlagsIgnoredWhenTreeInPlace: import flags on a node whose
// migration finished name nothing to do; the node starts without them.
func TestImportFlagsIgnoredWhenTreeInPlace(t *testing.T) {
	genesis := migrationGenesis(t)
	db, _ := merkleStateFixture(t, genesis)
	rawdb.WritePBTMigrationDone(db)

	cfg := DefaultConfig().WithStateScheme(rawdb.PathScheme)
	cfg.PBTImport = true
	chain, err := NewBlockChain(db, genesis, beacon.New(ethash.NewFaker()), cfg)
	if err != nil {
		t.Fatalf("flags refused on a finished node: %v", err)
	}
	defer chain.Stop()
	if chain.ImportsBinaryTree() {
		t.Fatal("chain still runs an import")
	}
	if rawdb.ReadPBTImportPending(db) {
		t.Fatal("import hold written for an ignored import")
	}
}
