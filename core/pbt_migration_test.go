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

	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

// migrationForkTime is far enough in the future that no generated block
// reaches it.
const migrationForkTime = uint64(1) << 40

// migrationGenesis clones the PBT chain genesis with the fork pushed past
// genesis - the migration configuration.
func migrationGenesis(t *testing.T) *Genesis {
	genesis, _, _, _ := pbtChainGenesis(t)
	cfg := *genesis.Config
	forkTime := migrationForkTime
	cfg.BinaryTrieTime = &forkTime
	genesis.Config = &cfg
	return genesis
}

// merkleGenesis clones the PBT chain genesis with the fork removed entirely.
func merkleGenesis(t *testing.T) *Genesis {
	genesis, _, _, _ := pbtChainGenesis(t)
	cfg := *genesis.Config
	cfg.BinaryTrieTime = nil
	genesis.Config = &cfg
	return genesis
}

// TestStateModeResolution pins the three-way split - no fork, fork at
// genesis, fork past genesis - from a supplied genesis and from a stored one.
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

	// The stored path answers the same with no genesis in hand.
	db := rawdb.NewMemoryDatabase()
	engine := beacon.New(ethash.NewFaker())
	chain, err := NewBlockChain(db, migrationGenesis(t), engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
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

// TestMigrationGenesisIsMerkle pins that scheduling the fork past genesis
// does not move the genesis hash: the chain starts as a plain merkle chain.
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
// merkle trie, and that a populated binary namespace - an anchor import may
// precede the first start - is not treated as corruption.
func TestNewBlockChainMigrationMode(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	rawdb.WritePBTFlatState(rawdb.NewTable(db, string(rawdb.PBTPrefix)))

	engine := beacon.New(ethash.NewFaker())
	chain, err := NewBlockChain(db, migrationGenesis(t), engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	if chain.TrieDB().IsPBT() {
		t.Fatal("migration chain opened on the binary tree")
	}
}

// TestMigrationRequiresPathScheme pins that the migration refuses the hash
// scheme: the second tree cannot exist there.
func TestMigrationRequiresPathScheme(t *testing.T) {
	engine := beacon.New(ethash.NewFaker())
	_, err := NewBlockChain(rawdb.NewMemoryDatabase(), migrationGenesis(t), engine, DefaultConfig().WithStateScheme(rawdb.HashScheme))
	if err == nil {
		t.Fatal("migration accepted the hash scheme")
	}
}

// TestUnscheduledPBTStateStillRefused pins the narrowed guard: binary state
// under a config that never schedules the fork stays corruption.
func TestUnscheduledPBTStateStillRefused(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	rawdb.WritePBTFlatState(rawdb.NewTable(db, string(rawdb.PBTPrefix)))

	engine := beacon.New(ethash.NewFaker())
	if _, err := NewBlockChain(db, merkleGenesis(t), engine, DefaultConfig().WithStateScheme(rawdb.PathScheme)); err == nil {
		t.Fatal("binary state accepted under a config that never schedules the fork")
	}
}
