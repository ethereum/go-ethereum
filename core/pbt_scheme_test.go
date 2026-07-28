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
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// pbtSchemeGenesis is a minimal binary-tree genesis, active from block zero.
func pbtSchemeGenesis() *Genesis {
	config := *testPBTChainConfig
	config.Ethash = nil
	return &Genesis{
		Config:     &config,
		Difficulty: big.NewInt(0),
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Alloc:      types.GenesisAlloc{{1}: {Balance: big.NewInt(1)}},
	}
}

// TestPBTTriedbConfigRequiresPathScheme pins that deriving a trie database
// config for the binary tree refuses every scheme but path. hashdb keys nodes
// by their keccak hash and decodes account leaves as RLP, so a binary node set
// handed to it silently becomes a merkle-patricia one.
//
// The unset case matters as much as the hash one: neither branch of the
// derivation fires for it, and triedb.NewDatabase then falls through to hashdb
// because it selects pathdb only when PathDB is non-nil.
func TestPBTTriedbConfigRequiresPathScheme(t *testing.T) {
	for _, tc := range []struct {
		name    string
		scheme  string
		wantErr bool
	}{
		{"hash", rawdb.HashScheme, true},
		{"unset", "", true},
		{"path", rawdb.PathScheme, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig().WithStateScheme(tc.scheme)

			// A merkle-patricia chain accepts all three unchanged.
			if _, err := cfg.triedbConfig(false); err != nil {
				t.Fatalf("non-PBT chain rejected the %q scheme: %v", tc.scheme, err)
			}
			got, err := cfg.triedbConfig(true)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("binary tree accepted the %q scheme", tc.scheme)
				}
				return
			}
			if err != nil {
				t.Fatalf("binary tree rejected the path scheme: %v", err)
			}
			if !got.IsPBT {
				t.Fatal("derived config lost the binary tree flag")
			}
			if got.PathDB == nil || got.HashDB != nil {
				t.Fatal("derived config is not path-scheme backed")
			}
		})
	}
}

// TestPBTBlockChainRejectsHashScheme pins the same rule on the path a caller
// actually takes. DefaultConfig selects the hash scheme, so constructing a
// binary-tree chain from it - which every direct NewBlockChain caller does by
// default - used to produce a hashdb over binary nodes rather than an error.
func TestPBTBlockChainRejectsHashScheme(t *testing.T) {
	engine := beacon.New(ethash.NewFaker())

	if _, err := NewBlockChain(rawdb.NewMemoryDatabase(), pbtSchemeGenesis(), engine, DefaultConfig()); err == nil {
		t.Fatal("binary-tree chain opened on the default (hash) scheme")
	}
	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), pbtSchemeGenesis(), engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatalf("binary-tree chain rejected the path scheme: %v", err)
	}
	defer chain.Stop()

	if !chain.TrieDB().IsPBT() {
		t.Fatal("chain is not running on the binary tree")
	}
	if chain.TrieDB().Scheme() != rawdb.PathScheme {
		t.Fatalf("chain state scheme is %q, want %q", chain.TrieDB().Scheme(), rawdb.PathScheme)
	}
}

// TestPBTHistoricStateRefused pins that the historic database refuses the
// binary tree - it opens merkle-patricia tries keyed by the hash of the
// address - and, just as importantly, that live state is still served.
//
// The distinction is the whole point: a check placed above the dispatch in
// stateAtBlock would also reject the recent blocks that come from the live
// database, which the tree handles perfectly well, and which is what tracing
// asks for most of the time.
//
// What this pins is that the refusal is explicit and names the tree. It does
// not reach the construction it protects against: the historic readers need
// indexed history to get as far as building a trie, which a chain this short
// has none of, so without the guard the request fails anyway - for the wrong
// reason, reported as state simply being unavailable.
func TestPBTHistoricStateRefused(t *testing.T) {
	cfg := DefaultConfig().WithStateScheme(rawdb.PathScheme).WithArchive(true)
	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), pbtSchemeGenesis(), beacon.New(ethash.NewFaker()), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	root := chain.CurrentBlock().Root
	if _, err := chain.StateAt(root); err != nil {
		t.Fatalf("live state is unavailable on the binary tree: %v", err)
	}
	_, err = chain.HistoricState(root)
	if err == nil {
		t.Fatal("historic database opened a merkle-patricia trie over binary nodes")
	}
	if !strings.Contains(err.Error(), "binary tree") {
		t.Fatalf("historic database failed for an unrelated reason: %v", err)
	}
}
