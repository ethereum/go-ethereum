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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
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
