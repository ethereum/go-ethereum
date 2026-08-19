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
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

// generateMigrationChain pre-generates n transfer blocks on a migrating
// genesis; every block pays the recipient so each moves state.
func generateMigrationChain(t *testing.T, n int) (*Genesis, ethdb.Database, []*types.Block, *replayUniverse) {
	genesis, key, sender, recipient := migrationChainGenesis(t)
	signer := types.LatestSigner(genesis.Config)

	u := newReplayUniverse()
	u.account(recipient)

	engine := beacon.New(ethash.NewFaker())
	db, blocks, _ := GenerateChainWithGenesis(genesis, engine, n, payTo(t, key, sender, recipient, signer, 1000))
	return genesis, db, blocks, u
}

// openMigrationChain opens a chain over the given database; the follower
// starts with it in migration mode.
func openMigrationChain(t *testing.T, db ethdb.Database, genesis *Genesis) *BlockChain {
	t.Helper()
	chain, err := NewBlockChain(db, genesis, beacon.New(ethash.NewFaker()), DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	return chain
}

// advance imports the blocks and promotes the last to canonical head.
func advance(t *testing.T, chain *BlockChain, blocks []*types.Block) {
	t.Helper()
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := chain.SetCanonical(blocks[len(blocks)-1]); err != nil {
		t.Fatalf("set canonical: %v", err)
	}
}

// awaitShadow waits for the follower to record the block's shadow root.
func awaitShadow(t *testing.T, chain *BlockChain, block *types.Block) common.Hash {
	t.Helper()
	if chain.follower == nil {
		t.Fatal("migration chain has no follower")
	}
	if err := chain.follower.waitCaughtUp(block.NumberU64(), block.Hash(), 10*time.Second); err != nil {
		t.Fatal(err)
	}
	return rawdb.ReadShadowStateRoot(chain.db, block.NumberU64(), block.Hash())
}

// universeWithAlloc folds the genesis allocation and fee recipient into the
// scenario's universe.
func universeWithAlloc(u *replayUniverse, genesis *Genesis) *replayUniverse {
	for addr := range genesis.Alloc {
		u.account(addr)
	}
	u.account(common.Address{})
	return u
}

// TestFollowerTracksChain pins the base loop: the follower seeds the shadow
// from genesis, replays each imported block, records per-hash roots that
// match the converter, and reports itself synced.
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
	if p := chain.MigrationProgress(); p.Phase != "synced" {
		t.Fatalf("progress phase %q, want synced (%+v)", p.Phase, p)
	}
}

// TestFollowerBatchedCatchup pins the deep-behind path: far from the head the
// follower folds ranges and records no intermediate roots, near the head it
// records every block, and the end state matches the converter.
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
	if r := rawdb.ReadShadowStateRoot(db, blocks[2].NumberU64(), blocks[2].Hash()); r != (common.Hash{}) {
		t.Fatalf("batched-over block 3 has a recorded root %x", r)
	}
}

// TestFollowerFollowsReorg pins the reorg walk: when the canonical chain
// switches branches, the follower rewinds to the fork point through its own
// records and replays the winning branch; the loser's records remain.
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

	db, branchA, _ := GenerateChainWithGenesis(genesis, engine, 2, payTo(t, key, sender, onlyOnA, signer, 1000))
	branchB, _ := GenerateChain(genesis.Config, genesis.ToBlock(), engine, db, 3, payTo(t, key, sender, onlyOnB, signer, 5000))

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
	if r := rawdb.ReadShadowStateRoot(db, tipA.NumberU64(), tipA.Hash()); r == (common.Hash{}) {
		t.Fatal("the losing branch lost its recorded shadow root")
	}
}

// TestFollowerRestartResumes pins the clean-shutdown path: the shadow is
// journaled at the follower's own root and a reopened chain resumes replay
// where it stopped.
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

// standaloneFollower drives follow() synchronously against a hand-built
// database, with no chain and no event loop.
func standaloneFollower(genesis *Genesis, db ethdb.Database) *bintrieFollower {
	cfg := DefaultConfig().WithStateScheme(rawdb.PathScheme)
	cfg.TrieNoAsyncFlush = true
	return &bintrieFollower{
		db:     db,
		config: genesis.Config,
		cfg:    cfg,
	}
}

// writeChainShape persists the generated blocks the way an import would:
// bodies, headers, access lists and canonical assignments.
func writeChainShape(db ethdb.Database, blocks []*types.Block) {
	for _, b := range blocks {
		rawdb.WriteBlock(db, b)
		rawdb.WriteCanonicalHash(db, b.Hash(), b.NumberU64())
	}
}

// TestFollowerRecoversWithoutJournal pins the crash rule: with the in-memory
// layers gone and the cursor pointing past the durable state, the follower
// walks its records back to a live root and re-replays; the end state is
// unchanged.
func TestFollowerRecoversWithoutJournal(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 4)
	writeChainShape(db, blocks)
	head := blocks[len(blocks)-1].Header()

	first := standaloneFollower(genesis, db)
	if err := first.follow(head, nil); err != nil {
		t.Fatalf("first pass: %v", err)
	}
	want := rawdb.ReadShadowStateRoot(db, head.Number.Uint64(), head.Hash())
	if want == (common.Hash{}) {
		t.Fatal("first pass recorded no head root")
	}
	// Crash: no journal - the diff layers die with the process.
	if err := first.shadow.Close(); err != nil {
		t.Fatal(err)
	}

	second := standaloneFollower(genesis, db)
	if err := second.follow(head, nil); err != nil {
		t.Fatalf("recovery pass: %v", err)
	}
	if got := rawdb.ReadShadowStateRoot(db, head.Number.Uint64(), head.Hash()); got != want {
		t.Fatalf("recovered head root %x, want %x", got, want)
	}
	if num, _, _, ok := rawdb.ReadPBTMigrationCursor(db); !ok || num != head.Number.Uint64() {
		t.Fatalf("cursor at %d (ok=%v), want %d", num, ok, head.Number.Uint64())
	}
}

// TestFollowerStallsOnMissingAccessList pins the availability rule: a block
// whose header commits to an access list that the database does not hold
// stalls the follower rather than silently skipping state.
func TestFollowerStallsOnMissingAccessList(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 2)
	writeChainShape(db, blocks)
	rawdb.DeleteAccessList(db, blocks[0].Hash(), blocks[0].NumberU64())

	f := standaloneFollower(genesis, db)
	err := f.follow(blocks[len(blocks)-1].Header(), nil)
	if err == nil || !strings.Contains(err.Error(), "access list") {
		t.Fatalf("follow = %v, want a missing-access-list error", err)
	}
}

// TestFollowerRequiresArtifactWithoutGenesisLists pins the seeding rule: a
// chain whose early blocks have no access lists cannot grow a shadow from
// genesis and must be anchor-seeded instead.
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
	if err == nil || !strings.Contains(err.Error(), "artifact") {
		t.Fatalf("follow = %v, want a conversion-artifact error", err)
	}
}

// TestFollowerIdlesDuringSnapSync pins the interlock: while the merkle side
// is snap-syncing the shadow neither opens nor replays.
func TestFollowerIdlesDuringSnapSync(t *testing.T) {
	genesis, db, blocks, _ := generateMigrationChain(t, 1)
	writeChainShape(db, blocks)
	rawdb.WriteSnapSyncStatusFlag(db, rawdb.StateSyncRunning)

	f := standaloneFollower(genesis, db)
	if err := f.follow(blocks[0].Header(), nil); err != nil {
		t.Fatalf("follow during snap sync = %v, want quiet idle", err)
	}
	if f.shadow != nil {
		t.Fatal("shadow opened during snap sync")
	}
}
