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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// Byte-level pins for block zero under the binary-tree schedule. A
// binaryTrieTime AT the genesis timestamp commits the state with the binary
// tree from block zero; a binaryTrieTime AFTER it commits the merkle-patricia
// genesis and lets a shadow catch up. These constants hold both arms still:
// any drift in genesis construction or database selection fails here first,
// loudly, instead of surfacing as a devnet whose clients disagree about
// block zero.
//
// The fixture below is deliberately frozen: it shares nothing mutable with
// the other test files, so the pins can only move when genesis encoding
// itself moves. Recomputing them is never routine maintenance - a red run
// here is a consensus-affecting change until proven otherwise.
var (
	pinnedPBTGenesisHash = common.HexToHash("0x8ad6f3ae34ddccca3178397d05397aacbebf66bbb6cace92e248eca298c81f45")
	pinnedPBTGenesisRoot = common.HexToHash("0x6e434dbab050bc1c92cba65e0c3afbad2cd96345e5d1612ace1b79a5bd01bbe9")
	pinnedMPTGenesisHash = common.HexToHash("0x25a02bfaa978af588b051f2e2304f4751639983762a0ddd55b546675d4f7f269")
)

// pinFutureOffset places the future-schedule arm. The frozen config activates
// every fork at time zero, so no other time-based fork lies in (0, 1800] and
// the offset moves only the binary-tree schedule.
const pinFutureOffset = 1800

// pinnedGenesis builds the frozen fixture the pins were computed under: every
// fork at zero, the binary tree scheduled at the genesis timestamp, and one
// funded account. Literal on purpose - do not replace any of it with a shared
// config or helper.
func pinnedGenesis() *Genesis {
	return &Genesis{
		Config: &params.ChainConfig{
			ChainID:                 big.NewInt(1),
			HomesteadBlock:          big.NewInt(0),
			EIP150Block:             big.NewInt(0),
			EIP155Block:             big.NewInt(0),
			EIP158Block:             big.NewInt(0),
			ByzantiumBlock:          big.NewInt(0),
			ConstantinopleBlock:     big.NewInt(0),
			PetersburgBlock:         big.NewInt(0),
			IstanbulBlock:           big.NewInt(0),
			MuirGlacierBlock:        big.NewInt(0),
			BerlinBlock:             big.NewInt(0),
			LondonBlock:             big.NewInt(0),
			ArrowGlacierBlock:       big.NewInt(0),
			GrayGlacierBlock:        big.NewInt(0),
			MergeNetsplitBlock:      big.NewInt(0),
			ShanghaiTime:            u64(0),
			CancunTime:              u64(0),
			PragueTime:              u64(0),
			OsakaTime:               u64(0),
			AmsterdamTime:           u64(0),
			BinaryTrieTime:          u64(0),
			TerminalTotalDifficulty: big.NewInt(0),
			BlobScheduleConfig: &params.BlobScheduleConfig{
				Cancun: params.DefaultCancunBlobConfig,
				Prague: params.DefaultPragueBlobConfig,
			},
		},
		Difficulty: big.NewInt(0),
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Alloc: types.GenesisAlloc{
			common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7"): {
				Balance: new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether)),
			},
		},
	}
}

// TestPBTGenesisPins pins the at-genesis schedule's genesis block: with
// binaryTrieTime equal to the genesis timestamp, ToBlock commits the alloc
// with the binary tree, and the resulting hash and root must never move.
func TestPBTGenesisPins(t *testing.T) {
	block := pinnedGenesis().ToBlock()
	if got := block.Hash(); got != pinnedPBTGenesisHash {
		t.Errorf("pbt genesis hash drifted: got %s, pinned %s", got, pinnedPBTGenesisHash)
	}
	if got := block.Root(); got != pinnedPBTGenesisRoot {
		t.Errorf("pbt genesis root drifted: got %s, pinned %s", got, pinnedPBTGenesisRoot)
	}
}

// TestPBTGenesisSelectsBinaryTree pins database selection for the at-genesis
// schedule: a chain whose config schedules the tree at the genesis timestamp
// opens on the binary tree from block zero, carrying the pinned block.
func TestPBTGenesisSelectsBinaryTree(t *testing.T) {
	genesis := pinnedGenesis()
	engine := beacon.New(ethash.NewFaker())
	db, _, _ := GenerateChainWithGenesis(genesis, engine, 0, nil)

	chain, err := NewBlockChain(db, genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if !chain.TrieDB().IsPBT() {
		t.Fatal("an at-genesis binary tree schedule did not open on the binary tree")
	}
	head := chain.CurrentBlock()
	if head.Number.Uint64() != 0 || head.Root != pinnedPBTGenesisRoot {
		t.Fatalf("head is block %d root %s, want block 0 root %s", head.Number, head.Root, pinnedPBTGenesisRoot)
	}
	if got := head.Hash(); got != pinnedPBTGenesisHash {
		t.Fatalf("head hash is %s, want the pinned %s", got, pinnedPBTGenesisHash)
	}
}

// TestMigrationGenesisCommitsMerklePatricia pins block zero's commitment as
// positional, byte-level on both arms: nil and future schedules commit the
// identical pinned merkle-patricia genesis, at-genesis stays the pinned
// binary block above.
func TestMigrationGenesisCommitsMerklePatricia(t *testing.T) {
	variant := func(tree *uint64) *Genesis {
		g := pinnedGenesis()
		g.Config.BinaryTrieTime = tree
		return g
	}
	base := pinnedGenesis()
	var (
		hashNil    = variant(nil).ToBlock().Hash()
		hashFuture = variant(u64(base.Timestamp + pinFutureOffset)).ToBlock().Hash()
		hashAt     = variant(u64(base.Timestamp)).ToBlock().Hash()
	)
	if hashNil != pinnedMPTGenesisHash {
		t.Fatalf("merkle-patricia genesis hash drifted: got %s, pinned %s", hashNil, pinnedMPTGenesisHash)
	}
	if hashFuture != hashNil {
		t.Fatalf("a future schedule commits %s, the merkle-patricia genesis is %s — a migration must start on the merkle trie", hashFuture, hashNil)
	}
	if hashFuture == hashAt {
		t.Fatal("a future schedule still commits the binary-tree genesis")
	}
	if hashAt != pinnedPBTGenesisHash {
		t.Fatalf("the at-genesis arm drifted from its pin: %s != %s", hashAt, pinnedPBTGenesisHash)
	}

	// Genesis.IsPBT answers block zero's commitment, never the schedule; the
	// schedule question stays with ChainConfig.IsPBT.
	if variant(u64(base.Timestamp + pinFutureOffset)).IsPBT() {
		t.Fatal("a future schedule claims a binary-tree genesis")
	}
	if !variant(u64(base.Timestamp)).IsPBT() {
		t.Fatal("an at-genesis schedule denies its binary-tree genesis")
	}
	if !variant(u64(base.Timestamp + pinFutureOffset)).Config.IsPBT() {
		t.Fatal("a future schedule stopped counting as scheduled at all")
	}
}
