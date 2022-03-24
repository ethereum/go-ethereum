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
	t.SkipNow()

	env := fillDB()
	defer env.teardown()

	reader, err := newSnapStore(env.roots[len(env.roots)-1], common.Hash{}, env.db)
	if err != nil {
		t.Fatalf("Failed to create reader %v", err)
	}
	keys, vals := env.keys[len(env.keys)-1], env.vals[len(env.vals)-1]

	readerCopy := reader.copy()

	for i := 0; i < len(keys); i++ {
		if len(vals[i]) == 0 {
			continue
		}
		_, path := DecodeStorageKey([]byte(keys[i]))
		blob1, err1 := reader.readBlob(common.Hash{}, crypto.Keccak256Hash(vals[i]), path)
		blob2, err2 := readerCopy.readBlob(common.Hash{}, crypto.Keccak256Hash(vals[i]), path)
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
	blob, err := reader.readBlob(common.Hash{}, node.hash, path.Bytes())
	if err != nil {
		t.Fatalf("Failed to read blob %v", err)
	}
	if !bytes.Equal(blob, node.rlp()) {
		t.Fatal("Unexpected node")
	}
	_, err = readerCopy.readBlob(common.Hash{}, node.hash, path.Bytes())
	missing, ok := err.(*MissingNodeError)
	if !ok || missing.NodeHash != node.hash {
		t.Fatal("didn't hit missing node, got", err)
	}

	// Create a new copy, it should retrieve the node correctly
	copyTwo := reader.copy()
	blob, err = copyTwo.readBlob(common.Hash{}, node.hash, path.Bytes())
	if err != nil {
		t.Fatalf("Failed to read blob %v", err)
	}
	if !bytes.Equal(blob, node.rlp()) {
		t.Fatal("Unexpected node")
	}
}
