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

package snap

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

func testTrieNodeChain(t *testing.T) *core.BlockChain {
	t.Helper()
	gspec := &core.Genesis{Config: params.MergedTestChainConfig}
	db := rawdb.NewMemoryDatabase()
	engine := beacon.New(ethash.NewFaker())
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, 1, nil)
	bc, err := core.NewBlockChain(db, gspec, engine, &core.BlockChainConfig{
		StateScheme:   rawdb.PathScheme,
		TrieTimeLimit: 5 * time.Minute,
		NoPrefetch:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bc.InsertChain(blocks); err != nil {
		t.Fatal(err)
	}
	return bc
}

// TestServiceGetTrieNodesQueryOversizedPath ensures structurally invalid trie paths
// do not trigger expensive trie lookups while preserving the empty-node response
// expected by the snap protocol test suite (see issue #34853).
func TestServiceGetTrieNodesQueryOversizedPath(t *testing.T) {
	bc := testTrieNodeChain(t)
	defer bc.Stop()

	longPath := make([]byte, 54)
	for i := range longPath {
		longPath[i] = byte(i)
	}
	paths, err := rlp.EncodeToRawList([]TrieNodePathSet{{longPath}})
	if err != nil {
		t.Fatal(err)
	}
	req := &GetTrieNodesPacket{
		Root:  bc.CurrentBlock().Root,
		Paths: paths,
		Bytes: 5000,
	}
	nodes, err := ServiceGetTrieNodesQuery(bc, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(nodes))
	}
	if hash := crypto.Keccak256Hash(nodes[0]); hash != types.EmptyCodeHash {
		t.Fatalf("got node hash %s, want %s", hash, types.EmptyCodeHash)
	}
}
