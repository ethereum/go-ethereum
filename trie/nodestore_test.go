// Copyright 2022 The go-ethereum Authors
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

package trie

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestNodeStoreCopy(t *testing.T) {
	// Insert a batch of entries into trie
	triedb := NewDatabase(rawdb.NewMemoryDatabase())
	trie := NewEmpty(triedb)
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"dog", "puppy"},
		{"somethingveryoddindeedthis is", "myothernodedata"},
	}
	for _, val := range vals {
		trie.Update([]byte(val.k), []byte(val.v))
	}
	trie.Commit(false) // all nodes should be committed into store

	seen := make(map[string][]byte)
	iter := trie.NodeIterator(nil)
	for iter.Next(true) {
		if iter.Hash() != (common.Hash{}) {
			seen[string(iter.Path())] = common.CopyBytes(iter.NodeBlob())
		}
	}

	// Create the node store copy, ensure all nodes can be retrieved back.
	store := trie.nodes
	storeCopy := store.copy()

	for path, blob := range seen {
		blob1, err1 := store.readBlob(common.Hash{}, crypto.Keccak256Hash(blob), []byte(path))
		blob2, err2 := storeCopy.readBlob(common.Hash{}, crypto.Keccak256Hash(blob), []byte(path))
		if err1 != nil || err2 != nil {
			t.Fatalf("Failed to read node, %v, %v", err1, err2)
		}
		if !bytes.Equal(blob1, blob) || !bytes.Equal(blob2, blob) {
			t.Fatal("Node is mismatched")
		}
	}
	// Flush items into the origin reader, it shouldn't affect the copy
	var (
		node = randomNode()
		path = randomHash()
	)
	store.write(string(path.Bytes()), node)
	blob, err := store.readBlob(common.Hash{}, node.hash, path.Bytes())
	if err != nil {
		t.Fatalf("Failed to read blob %v", err)
	}
	if !bytes.Equal(blob, node.rlp()) {
		t.Fatal("Unexpected node")
	}
	_, err = storeCopy.readBlob(common.Hash{}, node.hash, path.Bytes())
	missing, ok := err.(*MissingNodeError)
	if !ok || missing.NodeHash != node.hash {
		t.Fatal("didn't hit missing node, got", err)
	}

	// Create a new copy, it should retrieve the node correctly
	copyTwo := store.copy()
	blob, err = copyTwo.readBlob(common.Hash{}, node.hash, path.Bytes())
	if err != nil {
		t.Fatalf("Failed to read blob %v", err)
	}
	if !bytes.Equal(blob, node.rlp()) {
		t.Fatal("Unexpected node")
	}
}

// randomHash generates a random blob of data and returns it as a hash.
func randomHash() common.Hash {
	var hash common.Hash
	if n, err := rand.Read(hash[:]); n != common.HashLength || err != nil {
		panic(err)
	}
	return hash
}

func randomNode() *memoryNode {
	val := randBytes(100)
	return &memoryNode{
		hash: crypto.Keccak256Hash(val),
		node: rawNode(val),
		size: 100,
	}
}
