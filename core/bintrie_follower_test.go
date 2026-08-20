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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/triedb"
)

// generateMigrationChain pre-generates n blocks, each paying the recipient
// (0x0ff1) 1000 wei so every block moves state.
func generateMigrationChain(t *testing.T, n int) (*Genesis, ethdb.Database, []*types.Block, *replayUniverse) {
	genesis, key, sender, recipient := migrationChainGenesis(t)
	signer := types.LatestSigner(genesis.Config)

	u := newReplayUniverse()
	u.account(recipient)

	engine := beacon.New(ethash.NewFaker())
	db, blocks, _ := GenerateChainWithGenesis(genesis, engine, n, payTo(t, key, sender, recipient, signer, 1000))
	return genesis, db, blocks, u
}

func openMigrationChain(t *testing.T, db ethdb.Database, genesis *Genesis) *BlockChain {
	t.Helper()
	chain, err := NewBlockChain(db, genesis, beacon.New(ethash.NewFaker()), DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	return chain
}

func advance(t *testing.T, chain *BlockChain, blocks []*types.Block) {
	t.Helper()
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := chain.SetCanonical(blocks[len(blocks)-1]); err != nil {
		t.Fatalf("set canonical: %v", err)
	}
}

func awaitShadow(t *testing.T, chain *BlockChain, block *types.Block) common.Hash {
	t.Helper()
	if chain.follower == nil {
		t.Fatal("migration chain has no follower")
	}
	if err := chain.follower.waitCaughtUp(block.NumberU64(), block.Hash(), 10*time.Second); err != nil {
		t.Fatal(err)
	}
	root, ok := rawdb.ReadShadowStateRoot(chain.db, block.NumberU64(), block.Hash())
	if !ok {
		t.Fatalf("caught up to block %d but no shadow root recorded", block.NumberU64())
	}
	return root
}

// universeWithAlloc adds the genesis allocation and fee recipient.
func universeWithAlloc(u *replayUniverse, genesis *Genesis) *replayUniverse {
	for addr := range genesis.Alloc {
		u.account(addr)
	}
	u.account(common.Address{})
	return u
}

// TestFollowerTracksChain pins the base loop: seed, replay, record, synced.
func TestFollowerTracksChain(t *testing.T) {
	genesis, db, blocks, u := generateMigrationChain(t, 3)
	universeWithAlloc(u, genesis)

	chain := openMigrationChain(t, db, genesis)
	defer chain.Stop()
	advance(t, chain, blocks)

	head := blocks[len(blocks)-1]
	root := awaitShadow(t, chain, head)
	if want := convertCanonical(t, chain, head.Header(), u); root != want {
		t.Fatalf("shadow root %x, converting the canonical state says %x", root, want)
	}
	if p := chain.MigrationProgress(); p.Binary == nil || p.Binary.Phase != "synced" {
		t.Fatalf("binary progress %+v, want synced", p.Binary)
	}
	st, err := chain.State()
	if err != nil {
		t.Fatal(err)
	}
	if got := st.GetBalance(common.Address{0x0f, 0xf1}); got.Uint64() != 3000 {
		t.Fatalf("merkle-side recipient balance = %v, want 3000", got)
	}
}

// TestFollowerBatchedCatchup pins the deep-behind fold path.
func TestFollowerBatchedCatchup(t *testing.T) {
	n := followTrackWindow + 12
	genesis, _, blocks, u := generateMigrationChain(t, n)
	universeWithAlloc(u, genesis)

	// Import into a fresh database: generating this many blocks advances the
	// generator's own disk layer past genesis, which the chain still needs.
	db := rawdb.NewMemoryDatabase()
	chain := openMigrationChain(t, db, genesis)
	defer chain.Stop()
	advance(t, chain, blocks)

	head := blocks[len(blocks)-1]
	root := awaitShadow(t, chain, head)
	if want := convertCanonical(t, chain, head.Header(), u); root != want {
		t.Fatalf("shadow root %x, converting the canonical state says %x", root, want)
	}
	// The earliest blocks were folded over: no per-hash roots for them.
	if r, ok := rawdb.ReadShadowStateRoot(db, blocks[2].NumberU64(), blocks[2].Hash()); ok {
		t.Fatalf("batched-over block 3 has a recorded root %x", r)
	}
}

// TestFollowerFollowsReorg pins the reorg walk to a non-genesis fork point;
// the loser's records remain.
func TestFollowerFollowsReorg(t *testing.T) {
	genesis, key, sender, _ := migrationChainGenesis(t)
	var (
		onlyOnA = common.Address{0xaa, 0xaa}
		onlyOnB = common.Address{0xbb, 0xbb}
		signer  = types.LatestSigner(genesis.Config)
		engine  = beacon.New(ethash.NewFaker())
	)
	u := universeWithAlloc(newReplayUniverse(), genesis)
	u.account(onlyOnA, onlyOnB)

	// The branches fork at block 1, not genesis: the walk must resolve at a
	// replayed non-genesis ancestor.
	db, branchA, _ := GenerateChainWithGenesis(genesis, engine, 2, payTo(t, key, sender, onlyOnA, signer, 1000))
	branchB, _ := GenerateChain(genesis.Config, branchA[0], engine, db, 3, payTo(t, key, sender, onlyOnB, signer, 5000))

	chain := openMigrationChain(t, db, genesis)
	defer chain.Stop()

	advance(t, chain, branchA)
	tipA := branchA[len(branchA)-1]
	awaitShadow(t, chain, tipA)

	advance(t, chain, branchB)
	tipB := branchB[len(branchB)-1]
	root := awaitShadow(t, chain, tipB)
	if want := convertCanonical(t, chain, tipB.Header(), u); root != want {
		t.Fatalf("post-reorg shadow root %x, converter says %x", root, want)
	}
	if _, ok := rawdb.ReadShadowStateRoot(db, tipA.NumberU64(), tipA.Hash()); !ok {
		t.Fatal("the losing branch lost its recorded shadow root")
	}
}

// TestFollowerRestartResumes pins the clean-shutdown resume.
func TestFollowerRestartResumes(t *testing.T) {
	genesis, db, blocks, u := generateMigrationChain(t, 4)
	universeWithAlloc(u, genesis)

	chain := openMigrationChain(t, db, genesis)
	advance(t, chain, blocks[:2])
	awaitShadow(t, chain, blocks[1])
	chain.Stop()

	chain = openMigrationChain(t, db, genesis)
	defer chain.Stop()
	advance(t, chain, blocks[2:])

	head := blocks[3]
	root := awaitShadow(t, chain, head)
	if want := convertCanonical(t, chain, head.Header(), u); root != want {
		t.Fatalf("post-restart shadow root %x, converter says %x", root, want)
	}
}

// standaloneFollower drives follow() synchronously, no chain, no loop.
func standaloneFollower(genesis *Genesis, db ethdb.Database) *bintrieFollower {
	cfg := DefaultConfig().WithStateScheme(rawdb.PathScheme)
	cfg.TrieNoAsyncFlush = true
	return &bintrieFollower{
		db:     db,
		config: genesis.Config,
		cfg:    cfg,
	}
}

// writeChainShape persists generated blocks the way an import would.
func writeChainShape(db ethdb.Database, blocks []*types.Block) {
	for _, b := range blocks {
		rawdb.WriteBlock(db, b)
		rawdb.WriteCanonicalHash(db, b.Hash(), b.NumberU64())
	}
}

// firstPass replays the chain with a standalone follower, returning it and
// the head record.
func firstPass(t *testing.T, genesis *Genesis, db ethdb.Database, head *types.Header) (*bintrieFollower, common.Hash) {
	t.Helper()
	f := standaloneFollower(genesis, db)
	if err := f.follow(head, nil); err != nil {
		t.Fatalf("first pass: %v", err)
	}
	root, ok := rawdb.ReadShadowStateRoot(db, head.Number.Uint64(), head.Hash())
	if !ok {
		t.Fatal("first pass recorded no head root")
	}
	return f, root
}

// TestFollowerRecoversWithoutJournal pins the crash rule: layers gone, cursor
// ahead of the durable state, the records walk back to a live root.
func TestFollowerRecoversWithoutJournal(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 4)
	writeChainShape(db, blocks)
	head := blocks[len(blocks)-1].Header()

	first, want := firstPass(t, genesis, db, head)
	if err := first.direction(true).handle.Close(); err != nil { // crash: no journal
		t.Fatal(err)
	}

	second := standaloneFollower(genesis, db)
	if err := second.follow(head, nil); err != nil {
		t.Fatalf("recovery pass: %v", err)
	}
	if got, ok := rawdb.ReadShadowStateRoot(db, head.Number.Uint64(), head.Hash()); !ok || got != want {
		t.Fatalf("recovered head root %x (ok=%v), want %x", got, ok, want)
	}
}

// TestFollowerStallsOnMissingAccessList pins the availability stall.
func TestFollowerStallsOnMissingAccessList(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 2)
	writeChainShape(db, blocks)
	rawdb.DeleteAccessList(db, blocks[0].Hash(), blocks[0].NumberU64())

	f := standaloneFollower(genesis, db)
	err := f.follow(blocks[len(blocks)-1].Header(), nil)
	var missing *MissingAccessListError
	if !errors.As(err, &missing) || missing.Number != blocks[0].NumberU64() {
		t.Fatalf("follow = %v, want a MissingAccessListError for block %d", err, blocks[0].NumberU64())
	}
}

// TestFollowerRequestsMissingLists pins the fetcher hook: a stall on a
// missing list hands the requester exactly the absent blocks, and replay
// completes once they land.
func TestFollowerRequestsMissingLists(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 4)
	writeChainShape(db, blocks)
	missing := blocks[1]
	list := rawdb.ReadAccessList(db, missing.Hash(), missing.NumberU64())
	rawdb.DeleteAccessList(db, missing.Hash(), missing.NumberU64())

	var got []BALRequest
	f := standaloneFollower(genesis, db)
	f.setRequester(func(reqs []BALRequest) { got = append(got, reqs...) })

	head := blocks[len(blocks)-1].Header()
	f.sync(head, nil)
	var stall *MissingAccessListError
	if err := f.direction(true).stall; !errors.As(err, &stall) || stall.Number != missing.NumberU64() || stall.Hash != missing.Hash() {
		t.Fatalf("stall = %v, want the missing list of block %d", err, missing.NumberU64())
	}
	if len(got) != 1 || got[0] != (BALRequest{Number: missing.NumberU64(), Hash: missing.Hash()}) {
		t.Fatalf("requested %v, want exactly the missing block", got)
	}

	rawdb.WriteAccessList(db, missing.Hash(), missing.NumberU64(), list)
	if err := f.follow(head, nil); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if _, ok := rawdb.ReadShadowStateRoot(db, head.Number.Uint64(), head.Hash()); !ok {
		t.Fatal("head never recorded after the lists landed")
	}
}

// TestFollowerRequiresArtifactWithoutGenesisLists pins the seeding rule.
func TestFollowerRequiresArtifactWithoutGenesisLists(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 1)
	writeChainShape(db, blocks)

	// Pretend Amsterdam activates past genesis: the early chain carries no
	// access lists, so a genesis seed cannot follow it.
	cfg := *genesis.Config
	later := uint64(1) << 39
	cfg.AmsterdamTime = &later
	genesis = genesis.copy()
	genesis.Config = &cfg

	f := standaloneFollower(genesis, db)
	err := f.follow(blocks[0].Header(), nil)
	if err == nil || !strings.Contains(err.Error(), "predates access lists") {
		t.Fatalf("follow = %v, want a seeding error", err)
	}
}

// TestFollowerIdlesDuringSnapSync pins the snap-sync interlock.
func TestFollowerIdlesDuringSnapSync(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 1)
	writeChainShape(db, blocks)
	rawdb.WriteSnapSyncStatusFlag(db, rawdb.StateSyncRunning)

	f := standaloneFollower(genesis, db)
	if err := f.follow(blocks[0].Header(), nil); err != nil {
		t.Fatalf("follow during snap sync = %v, want quiet idle", err)
	}
	if f.direction(true).handle != nil {
		t.Fatal("shadow opened during snap sync")
	}
}

// TestFollowerRefusesUnresolvablePosition pins the poisoned-fallback guard.
func TestFollowerRefusesUnresolvablePosition(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 2)
	writeChainShape(db, blocks)

	// A cursor with a dead root, no live records below it, and a disk root
	// that is NOT the genesis seed - the post-crash shape after deep damage.
	// The namespace is attested, as any real shadow is from creation.
	pbtdb := rawdb.NewTable(db, string(rawdb.PBTPrefix))
	rawdb.WritePBTFlatState(pbtdb)
	rawdb.WriteSnapshotRoot(pbtdb, common.Hash{0xd1, 0x5c})
	rawdb.WritePBTMigrationCursor(db, 2, blocks[1].Hash(), common.Hash{0xde, 0xad})

	f := standaloneFollower(genesis, db)
	err := f.follow(blocks[1].Header(), nil)
	if err == nil || !strings.Contains(err.Error(), "re-anchor") {
		t.Fatalf("follow = %v, want an unresolvable-position error", err)
	}
}

// TestFollowerStallsOnAccessListMismatch pins the integrity check.
func TestFollowerStallsOnAccessListMismatch(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 2)
	writeChainShape(db, blocks)
	rawdb.WriteAccessList(db, blocks[0].Hash(), blocks[0].NumberU64(), &bal.BlockAccessList{})

	f := standaloneFollower(genesis, db)
	err := f.follow(blocks[len(blocks)-1].Header(), nil)
	if err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("follow = %v, want an access-list mismatch error", err)
	}
}

// TestFollowerStallsBeforeAccessLists pins the coverage stall.
func TestFollowerStallsBeforeAccessLists(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 1)
	writeChainShape(db, blocks)

	// Strip the header's list commitment - and the optional fields behind it,
	// which cannot encode past a nil - as a pre-Amsterdam block would look
	// behind an anchor imported too early.
	header := blocks[0].Header()
	header.BlockAccessListHash = nil
	header.SlotNumber = nil
	rawdb.WriteHeader(db, header)
	rawdb.WriteCanonicalHash(db, header.Hash(), header.Number.Uint64())
	pbtdb := rawdb.NewTable(db, string(rawdb.PBTPrefix))
	rawdb.WritePBTFlatState(pbtdb)
	rawdb.WritePBTAnchor(pbtdb, 0, rawdb.ReadCanonicalHash(db, 0))

	f := standaloneFollower(genesis, db)
	err := f.follow(header, nil)
	if err == nil || !strings.Contains(err.Error(), "no access list") {
		t.Fatalf("follow = %v, want a no-access-list error", err)
	}
}

// TestFollowerResumesFromRecordAboveCursor pins the batch crash window: the
// record lands before the cursor, so the resolver scans up before down.
func TestFollowerResumesFromRecordAboveCursor(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 4)
	writeChainShape(db, blocks)
	head := blocks[len(blocks)-1].Header()

	first, want := firstPass(t, genesis, db, head)
	// Make the head root durable, then crash with the cursor rewound to a
	// block whose own root died with the in-memory layers.
	if err := first.direction(true).handle.Commit(want, false); err != nil {
		t.Fatal(err)
	}
	if err := first.direction(true).handle.Close(); err != nil {
		t.Fatal(err)
	}
	rawdb.WritePBTMigrationCursor(db, 2, blocks[1].Hash(), common.Hash{0xde, 0xad})

	second := standaloneFollower(genesis, db)
	if err := second.follow(head, nil); err != nil {
		t.Fatalf("recovery pass: %v", err)
	}
	if num, _, root := second.direction(true).cursor(); num != head.Number.Uint64() || root != want {
		t.Fatalf("resumed at %d root %x, want %d root %x", num, root, head.Number.Uint64(), want)
	}
}

// TestFollowerStopsOnCanonicalDiscontinuity pins the mid-run splice stop.
func TestFollowerStopsOnCanonicalDiscontinuity(t *testing.T) {
	genesis, key, sender, _ := migrationChainGenesis(t)
	var (
		onlyOnA = common.Address{0xaa, 0xaa}
		onlyOnB = common.Address{0xbb, 0xbb}
		signer  = types.LatestSigner(genesis.Config)
		engine  = beacon.New(ethash.NewFaker())
	)
	db, branchA, _ := GenerateChainWithGenesis(genesis, engine, 3, payTo(t, key, sender, onlyOnA, signer, 1000))
	branchB, _ := GenerateChain(genesis.Config, genesis.ToBlock(), engine, db, 3, payTo(t, key, sender, onlyOnB, signer, 5000))
	writeChainShape(db, branchA[:1])

	f := standaloneFollower(genesis, db)
	if err := f.follow(branchA[0].Header(), nil); err != nil {
		t.Fatal(err)
	}
	// A reorg to branch B caught mid-rewrite: block 2 already re-pointed,
	// block 1 still branch A. The next block does not chain onto the cursor.
	rawdb.WriteBlock(db, branchB[1])
	rawdb.WriteCanonicalHash(db, branchB[1].Hash(), 2)

	if err := f.follow(branchB[1].Header(), nil); err != nil {
		t.Fatalf("follow across the splice = %v, want a clean stop", err)
	}
	if num, hash, _ := f.direction(true).cursor(); num != 1 || hash != branchA[0].Hash() {
		t.Fatalf("cursor moved to %d %x across a splice", num, hash)
	}
	if _, ok := rawdb.ReadShadowStateRoot(db, 2, branchB[1].Hash()); ok {
		t.Fatal("a spliced block got a shadow root recorded")
	}
}

// TestFollowerBatchesDeepCatchup pins the fold path: a follower far behind
// folds ranges - sparse records - and lands on the root the per-block path
// produces.
func TestFollowerBatchesDeepCatchup(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 300)
	writeChainShape(db, blocks)
	head := blocks[len(blocks)-1].Header()

	batched := standaloneFollower(genesis, db)
	if err := batched.follow(head, nil); err != nil {
		t.Fatalf("batched catch-up: %v", err)
	}
	want, ok := rawdb.ReadShadowStateRoot(db, head.Number.Uint64(), head.Hash())
	if !ok {
		t.Fatal("batched catch-up recorded no head root")
	}
	if _, ok := rawdb.ReadShadowStateRoot(db, 50, blocks[49].Hash()); ok {
		t.Fatal("a folded-over block got a record")
	}
	if _, ok := rawdb.ReadShadowStateRoot(db, 128, blocks[127].Hash()); !ok {
		t.Fatal("the batch end got no record")
	}

	db2 := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db2, triedb.HashDefaults)
	defer tdb.Close()
	if _, err := genesis.Commit(db2, tdb, nil); err != nil {
		t.Fatal(err)
	}
	writeChainShape(db2, blocks)
	perBlock := standaloneFollower(genesis, db2)
	for _, n := range []int{100, 200, 300} {
		if err := perBlock.follow(blocks[n-1].Header(), nil); err != nil {
			t.Fatalf("per-block catch-up to %d: %v", n, err)
		}
	}
	if got, ok := rawdb.ReadShadowStateRoot(db2, head.Number.Uint64(), head.Hash()); !ok || got != want {
		t.Fatalf("per-block root %x (ok=%v), batched said %x", got, ok, want)
	}
}

// TestFollowerRefusesTreesDuringSnapSync pins the open guard: a probe while
// the sync flag is up must not poison the handle for the later replay.
func TestFollowerRefusesTreesDuringSnapSync(t *testing.T) {
	genesis, db, _, _ := generateMigrationChain(t, 2)
	rawdb.WriteSnapSyncStatusFlag(db, rawdb.StateSyncRunning)
	chain := openMigrationChain(t, db, genesis)
	defer chain.Stop()

	if chain.HasState(common.Hash{0x01}) {
		t.Fatal("probe found state during snap sync")
	}
	if _, err := chain.follower.tree(true); err == nil {
		t.Fatal("tree opened during snap sync")
	}
	rawdb.WriteSnapSyncStatusFlag(db, rawdb.StateSyncFinished)
	if _, err := chain.follower.tree(true); err != nil {
		t.Fatalf("tree still refused after snap sync: %v", err)
	}
	if err := chain.follower.direction(true).ensure(); err != nil {
		t.Fatalf("shadow never resolved after snap sync: %v", err)
	}
}

// TestWaitCaughtUpAbortsOnStop pins the wait against dead followers: a
// stopped loop - shutdown or a closed window - fails fast, not by timeout.
func TestWaitCaughtUpAbortsOnStop(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 1)
	chain := openMigrationChain(t, db, genesis)
	advance(t, chain, blocks)
	f := chain.follower
	chain.Stop()

	start := time.Now()
	err := f.waitCaughtUp(99, common.Hash{0xaa}, 30*time.Second)
	if err == nil || time.Since(start) > time.Second {
		t.Fatalf("wait on a stopped follower = %v after %v", err, time.Since(start))
	}
}
