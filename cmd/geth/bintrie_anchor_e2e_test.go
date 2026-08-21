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
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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

// TestAnchorSeededCatchup pairs the importer with the follower: a migrating
// chain's state is converted at an anchor, the artifacts imported into a
// wiped consumer, and the follower replays anchor→head to the same record a
// genesis-seeded follower produced - two derivations of one root.
func TestAnchorSeededCatchup(t *testing.T) {
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
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, 6, func(i int, b *core.BlockGen) {
		b.AddTx(types.MustSignNewTx(key, signer, &types.LegacyTx{
			Nonce: uint64(i), Gas: 1_000_000, GasPrice: gasPrice, To: &recipient, Value: big.NewInt(1000),
		}))
	})

	chaindb := rawdb.NewMemoryDatabase()
	cfg := core.DefaultConfig()
	cfg.Preimages = true
	cfg.StateScheme = rawdb.PathScheme
	chain, err := core.NewBlockChain(chaindb, gspec, engine, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatal(err)
	}
	head := blocks[len(blocks)-1]
	if _, err := chain.SetCanonical(head); err != nil {
		t.Fatal(err)
	}
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
	if _, err := convertState(chaindb, src, anchor.Root(), conversionOptions{
		snapshotPath: snapPath,
		preimagePath: prePath,
	}); err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	src.Close()
	if err := wipeBinaryTrieState(chaindb, triedbDir); err != nil {
		t.Fatal(err)
	}
	if _, err := importState(chaindb, importOptions{
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
