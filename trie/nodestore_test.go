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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestNodeStoreCopy(t *testing.T) {
	env := fillDB(t)
	head := env.roots[len(env.roots)-1]
	store, err := newNodeStore(head, env.db)
	if err != nil {
		t.Fatalf("Failed to create store %v", err)
	}
	keys, vals := env.keys[len(env.keys)-1], env.vals[len(env.vals)-1]

	// Create the node store copy, ensure all nodes can be retrieved back.
	storeCopy := store.copy()
	for i := 0; i < len(keys); i++ {
		if len(vals[i]) == 0 {
			continue
		}
		_, path := DecodeStorageKey([]byte(keys[i]))
		blob1, err1 := store.readBlob(common.Hash{}, crypto.Keccak256Hash(vals[i]), path)
		blob2, err2 := storeCopy.readBlob(common.Hash{}, crypto.Keccak256Hash(vals[i]), path)
		if err1 != nil || err2 != nil {
			t.Fatalf("Failed to read node, %v, %v", err1, err2)
		}
		if !bytes.Equal(blob1, blob2) {
			t.Fatal("Node is mismatched")
		}
	}

	// Flush items into the origin reader, it shouldn't affect the copy
	var (
		node = randomNode()
		path = randomHash()
	)
	store.commit(map[string]*cachedNode{
		string(EncodeStorageKey(common.Hash{}, path.Bytes())): node,
	})
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
