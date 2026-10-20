// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// awaitRecord polls for a block's shadow-root record.
func awaitRecord(t *testing.T, db ethdb.Database, number uint64, hash common.Hash) common.Hash {
	t.Helper()
	for start := time.Now(); ; time.Sleep(10 * time.Millisecond) {
		if root, ok := rawdb.ReadShadowStateRoot(db, hash, number); ok {
			return root
		}
		if time.Since(start) > 15*time.Second {
			t.Fatalf("shadow never recorded block %d", number)
		}
	}
}

// anchorTestChain is a migrating chain - access lists from genesis, the fork
// far off - of n blocks that each move state, with the config its nodes open.
func anchorTestChain(t *testing.T, n int) (*core.Genesis, consensus.Engine, []*types.Block, *core.BlockChainConfig) {
	t.Helper()
	u64 := func(v uint64) *uint64 { return &v }
	config := *params.MergedTestChainConfig
	config.AmsterdamTime = u64(0)
	config.BinaryTrieTime = u64(1 << 40)
	config.Ethash = nil

	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	var (
		sender    = crypto.PubkeyToAddress(key.PublicKey)
		recipient = common.Address{0x0a, 0x0c}
		signer    = types.LatestSigner(&config)
		gasPrice  = big.NewInt(params.InitialBaseFee + 1)
		engine    = beacon.New(ethash.NewFaker())
	)
	gspec := &core.Genesis{
		Config:     &config,
		Difficulty: common.Big0,
		BaseFee:    big.NewInt(params.InitialBaseFee),
		GasLimit:   30_000_000,
		Alloc: types.GenesisAlloc{
			sender: {Balance: new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))},
		},
	}
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, n, func(i int, b *core.BlockGen) {
		b.AddTx(types.MustSignNewTx(key, signer, &types.LegacyTx{
			Nonce: uint64(i), Gas: 1_000_000, GasPrice: gasPrice, To: &recipient, Value: big.NewInt(1000),
		}))
	})
	cfg := core.DefaultConfig()
	cfg.Preimages = true
	cfg.StateScheme = rawdb.PathScheme
	return gspec, engine, blocks, cfg
}

// insertBlocks inserts blocks and makes the last one the head.
func insertBlocks(t *testing.T, chain *core.BlockChain, blocks []*types.Block) {
	t.Helper()
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatal(err)
	}
	if _, err := chain.SetCanonical(blocks[len(blocks)-1]); err != nil {
		t.Fatal(err)
	}
}

// TestAnchorSeededCatchup pairs the importer with the follower: a migrating
// chain's state is converted at an anchor, the artifacts imported into a
// wiped consumer, and the follower replays anchor→head to the same record a
// genesis-seeded follower produced - two derivations of one root.
func TestAnchorSeededCatchup(t *testing.T) {
	gspec, engine, blocks, cfg := anchorTestChain(t, 6)
	chaindb := rawdb.NewMemoryDatabase()
	chain, err := core.NewBlockChain(chaindb, gspec, engine, cfg)
	if err != nil {
		t.Fatal(err)
	}
	insertBlocks(t, chain, blocks)
	head := blocks[len(blocks)-1]
	want := awaitRecord(t, chaindb, head.NumberU64(), head.Hash())
	chain.Stop()

	// Consumer simulation: drop the genesis-seeded shadow, convert at the
	// anchor with artifacts, wipe again, and import.
	triedbDir := t.TempDir()
	if err := wipeBinaryTrieState(chaindb, triedbDir); err != nil {
		t.Fatal(err)
	}
	var (
		anchor   = blocks[3]
		snapPath = filepath.Join(t.TempDir(), "snapshot")
		prePath  = filepath.Join(t.TempDir(), "preimages")
	)
	src := triedb.NewDatabase(chaindb, &triedb.Config{Preimages: true, PathDB: pathdb.ReadOnly})
	if _, err := convertState(context.Background(), chaindb, src, anchor.Root(), conversionOptions{
		snapshotPath: snapPath,
		preimagePath: prePath,
	}); err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	src.Close()
	if err := wipeBinaryTrieState(chaindb, triedbDir); err != nil {
		t.Fatal(err)
	}
	if _, err := importState(context.Background(), chaindb, importOptions{
		snapshot:  snapPath,
		preimages: prePath,
		anchor:    anchor.Header(),
	}); err != nil {
		t.Fatalf("import failed: %v", err)
	}

	// The reopened follower must resolve through the anchor and catch up.
	chain, err = core.NewBlockChain(chaindb, gspec, engine, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	if got := awaitRecord(t, chaindb, head.NumberU64(), head.Hash()); got != want {
		t.Fatalf("anchor-seeded catch-up root %x, genesis-seeded said %x", got, want)
	}
	// The wipe cleared all records, so one below the anchor means the
	// follower re-seeded from genesis instead of adopting the anchor.
	if _, ok := rawdb.ReadShadowStateRoot(chaindb, blocks[1].Hash(), 2); ok {
		t.Fatal("follower replayed below the anchor")
	}
}

// onlineFixture is a migrating chain, the shadow root a genesis-seeded
// follower records at its head, and artifact pairs converted at some blocks.
type onlineFixture struct {
	gspec  *core.Genesis
	engine consensus.Engine
	cfg    *core.BlockChainConfig
	blocks []*types.Block
	want   common.Hash
	pairs  map[int][2]string // block index -> snapshot and preimage file
}

func newOnlineFixture(t *testing.T, anchors ...int) *onlineFixture {
	t.Helper()
	gspec, engine, blocks, cfg := anchorTestChain(t, 8)
	db := rawdb.NewMemoryDatabase()
	chain, err := core.NewBlockChain(db, gspec, engine, cfg)
	if err != nil {
		t.Fatal(err)
	}
	insertBlocks(t, chain, blocks)
	head := blocks[len(blocks)-1]
	f := &onlineFixture{gspec: gspec, engine: engine, cfg: cfg, blocks: blocks,
		want: awaitRecord(t, db, head.NumberU64(), head.Hash()), pairs: make(map[int][2]string)}
	chain.Stop()

	src := triedb.NewDatabase(db, &triedb.Config{Preimages: true, PathDB: pathdb.ReadOnly})
	defer src.Close()
	for _, i := range anchors {
		// The converter writes the namespace it converts into: clear it.
		if err := wipeBinaryTrieState(db, ""); err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		pair := [2]string{filepath.Join(dir, "snapshot"), filepath.Join(dir, "preimages")}
		if _, err := convertState(context.Background(), db, src, blocks[i].Root(), conversionOptions{snapshotPath: pair[0], preimagePath: pair[1]}); err != nil {
			t.Fatalf("conversion at block %d failed: %v", blocks[i].NumberU64(), err)
		}
		f.pairs[i] = pair
	}
	return f
}

// node opens a node on db that imports beside its chain.
func (f *onlineFixture) node(t *testing.T, db ethdb.Database, force bool) *core.BlockChain {
	t.Helper()
	cfg := *f.cfg
	cfg.PBTImport, cfg.PBTImportForce = true, force
	chain, err := core.NewBlockChain(db, f.gspec, f.engine, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	return chain
}

// importer runs the import of the pair converted at block i, by hash.
func (f *onlineFixture) importer(t *testing.T, chain *core.BlockChain, db ethdb.Database, i int, force bool) *pbtImporter {
	pair := f.pairs[i]
	return &pbtImporter{
		chain: chain,
		db:    db,
		cfg: ethconfig.PBTImportConfig{Snapshot: pair[0], Preimages: pair[1],
			Anchor: f.blocks[i].Hash().Hex(), Force: force},
		triedbDir: t.TempDir(),
		tmpDir:    filepath.Join(t.TempDir(), "pbt-import"),
		poll:      10 * time.Millisecond,
	}
}

// TestOnlineImportFollowsTheChain: the import waits for its anchor to be
// finalized, clears an interrupted run's leftovers, and runs while blocks
// keep landing; once it attests, the follower adopts the anchor and reaches
// the head with the root a genesis-seeded follower produced, and the pair it
// consumed is left empty. Restarting with the same flags imports nothing;
// another anchor needs force, which replaces the tree.
func TestOnlineImportFollowsTheChain(t *testing.T) {
	f := newOnlineFixture(t, 3, 4)
	db := rawdb.NewMemoryDatabase()
	chain := f.node(t, db, false)
	insertBlocks(t, chain, f.blocks[:5])
	pbtdb := rawdb.NewTable(db, string(rawdb.PBTPrefix))
	leftover := []byte{0x01} // no tree node path: those carry a 2-byte bit count
	rawdb.WriteAccountTrieNode(pbtdb, leftover, []byte{0xde, 0xad})

	p := f.importer(t, chain, db, 3, false)
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * p.poll)
	if rawdb.HasPBTState(db) || len(rawdb.ReadAccountTrieNode(pbtdb, leftover)) == 0 {
		t.Fatal("the import ran before its anchor was finalized")
	}
	if prog := chain.MigrationProgress(); prog.Phase != "importing" {
		t.Fatalf("progress phase %q while waiting, want importing", prog.Phase)
	}
	chain.SetFinalized(f.blocks[4].Header())
	insertBlocks(t, chain, f.blocks[5:])
	<-p.done

	head := f.blocks[len(f.blocks)-1]
	if got := awaitRecord(t, db, head.NumberU64(), head.Hash()); got != f.want {
		t.Fatalf("online import's catch-up root %x, genesis-seeded said %x", got, f.want)
	}
	if _, ok := rawdb.ReadShadowStateRoot(db, f.blocks[1].Hash(), 2); ok {
		t.Fatal("follower replayed below the anchor")
	}
	if len(rawdb.ReadAccountTrieNode(pbtdb, leftover)) != 0 {
		t.Fatal("the interrupted run's leftovers survived")
	}
	for _, path := range f.pairs[3] {
		if info, err := os.Stat(path); err != nil || info.Size() != 0 {
			t.Fatalf("the import did not consume %s (err %v)", path, err)
		}
	}
	p.Stop()
	chain.Stop()

	// The same flags again: already imported, nothing held or run.
	chain = f.node(t, db, false)
	p = f.importer(t, chain, db, 3, false)
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	if p.done != nil || rawdb.ReadPBTImportPending(db) {
		t.Fatal("restarting with the same flags imported again")
	}
	chain.Stop()

	// Another anchor is refused without force...
	chain = f.node(t, db, false)
	if err := f.importer(t, chain, db, 4, false).Start(); err == nil || !strings.Contains(err.Error(), "pbt.import.force") {
		t.Fatalf("err = %v, want a refusal naming --pbt.import.force", err)
	}
	chain.Stop()

	// ...and replaces the tree with it, named by number this time.
	chain = f.node(t, db, true)
	defer chain.Stop()
	chain.SetFinalized(head.Header())
	p = f.importer(t, chain, db, 4, true)
	p.cfg.Anchor = strconv.FormatUint(f.blocks[4].NumberU64(), 10)
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	<-p.done
	if num, hash, _ := rawdb.ReadPBTAnchor(pbtdb); num != f.blocks[4].NumberU64() || hash != f.blocks[4].Hash() {
		t.Fatalf("anchor %d (%x) after the forced import, want block %d", num, hash, f.blocks[4].NumberU64())
	}
	chain.KickMigration()
	if got := awaitRecord(t, db, head.NumberU64(), head.Hash()); got != f.want {
		t.Fatalf("replaced tree's catch-up root %x, genesis-seeded said %x", got, f.want)
	}
	if _, ok := rawdb.ReadShadowStateRoot(db, f.blocks[3].Hash(), f.blocks[3].NumberU64()); ok {
		t.Fatal("the replaced tree's records survived")
	}
}

// TestOnlineImportNeverCanonicalAnchor: an anchor whose height is finalized
// with another block will never qualify; the import fails, the tree stays
// held, and the chain keeps following.
func TestOnlineImportNeverCanonicalAnchor(t *testing.T) {
	f := newOnlineFixture(t, 3)
	_, side, _ := core.GenerateChainWithGenesis(f.gspec, f.engine, 4, func(i int, b *core.BlockGen) {
		b.SetExtra([]byte("side"))
	})
	db := rawdb.NewMemoryDatabase()
	chain := f.node(t, db, false)
	defer chain.Stop()
	insertBlocks(t, chain, f.blocks[:5])
	for _, b := range side {
		if _, err := chain.InsertBlockWithoutSetHead(context.Background(), b, false); err != nil {
			t.Fatal(err)
		}
	}
	chain.SetFinalized(f.blocks[4].Header())

	p := f.importer(t, chain, db, 3, false)
	p.cfg.Anchor = side[3].Hash().Hex()
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	<-p.done
	prog := chain.MigrationProgress()
	if prog.Phase != "importFailed" || prog.Binary == nil || !strings.Contains(prog.Binary.Error, "never be canonical") {
		t.Fatalf("progress %+v, want importFailed on the anchor", prog)
	}
	if !rawdb.ReadPBTImportPending(db) {
		t.Fatal("a failed import released the tree")
	}
	insertBlocks(t, chain, f.blocks[5:])
	if head := chain.CurrentBlock(); head.Number.Uint64() != f.blocks[len(f.blocks)-1].NumberU64() {
		t.Fatalf("head %d after the failed import, want %d", head.Number, f.blocks[len(f.blocks)-1].NumberU64())
	}
}

// TestOnlineImportStopAndFailure: stopping a waiting import keeps the hold for
// the next start; an anchor that leaves the canonical chain before the
// attestation is refused; a pair that fails verification after writing
// leaves the tree held and its namespace clean, scratch included, while the
// chain keeps inserting blocks; and a restart over the pair that failed run
// consumed says so instead of reading it.
func TestOnlineImportStopAndFailure(t *testing.T) {
	f := newOnlineFixture(t, 3)
	db := rawdb.NewMemoryDatabase()
	chain := f.node(t, db, false)
	defer chain.Stop()
	insertBlocks(t, chain, f.blocks[:5])

	p := f.importer(t, chain, db, 3, false)
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	if err := p.Stop(); err != nil {
		t.Fatal(err)
	}
	if !rawdb.ReadPBTImportPending(db) {
		t.Fatal("a stopped import released the tree")
	}

	anchor := f.blocks[3]
	rawdb.WriteCanonicalHash(db, common.Hash{0x0b, 0xad}, anchor.NumberU64())
	p.cfg.KeepArtifacts = true // the pair serves the next run too
	err := p.importAt(context.Background(), anchor.Header())
	rawdb.WriteCanonicalHash(db, anchor.Hash(), anchor.NumberU64())
	if err == nil || !strings.Contains(err.Error(), "left the canonical chain") {
		t.Fatalf("err = %v, want the anchor's re-check to refuse", err)
	}
	if rawdb.HasPBTState(db) {
		t.Fatal("an import whose anchor left the chain attested")
	}

	// The pair converted at block 4 claimed as block 5's state: the tree
	// rebuilds and is written, then the merkle re-derivation refuses it.
	chain.SetFinalized(f.blocks[4].Header())
	p = f.importer(t, chain, db, 3, false)
	p.cfg.Anchor = f.blocks[4].Hash().Hex()
	if err := os.MkdirAll(p.tmpDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.tmpDir, "killed-run-spill"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	<-p.done
	prog := chain.MigrationProgress()
	if prog.Phase != "importFailed" || prog.Binary == nil || !strings.Contains(prog.Binary.Error, "anchor commits") {
		t.Fatalf("progress %+v, want importFailed on the merkle re-derivation", prog)
	}
	if dirty, err := hasBinaryTrieState(db); err != nil || dirty {
		t.Fatalf("the failed import's writes survived (err %v)", err)
	}
	if _, err := os.Stat(p.tmpDir); !os.IsNotExist(err) {
		t.Fatalf("scratch survived the import: %v", err)
	}
	if !rawdb.ReadPBTImportPending(db) {
		t.Fatal("a failed import released the tree")
	}
	insertBlocks(t, chain, f.blocks[5:])

	p = f.importer(t, chain, db, 3, false)
	p.cfg.Anchor = f.blocks[4].Hash().Hex()
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	<-p.done
	if prog := chain.MigrationProgress(); prog.Phase != "importFailed" || prog.Binary == nil || !strings.Contains(prog.Binary.Error, "consumed") {
		t.Fatalf("progress %+v, want importFailed naming the consumed pair", prog)
	}
}
