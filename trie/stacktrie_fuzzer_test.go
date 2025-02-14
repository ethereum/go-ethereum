// Copyright 2020 The go-ethereum Authors
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
	"encoding/binary"
	"fmt"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

func FuzzStackTrie(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzz(data, false)
	})
}

func fuzz(data []byte, debugging bool) {
	var (
		input = bytes.NewReader(data)
		dbA   = newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme)
		trieA = NewEmpty(dbA)
		memDB = rawdb.NewMemoryDatabase()
		trieB = NewStackTrie(func(path []byte, hash common.Hash, blob []byte) {
			rawdb.WriteTrieNode(memDB, common.Hash{}, path, hash, blob, rawdb.HashScheme)
		})
		vals        []*kv
		maxElements = 10000
		// operate on unique keys only
		keys = make(map[string]struct{})
	)
	// Fill the trie with elements
	for i := 0; input.Len() > 0 && i < maxElements; i++ {
		// Build the key
		k := make([]byte, 32)
		input.Read(k)

		// Build the val
		var a uint16
		binary.Read(input, binary.LittleEndian, &a)
		a = 1 + a%100
		v := make([]byte, a)
		input.Read(v)

		if input.Len() == 0 {
			// If it was exhausted while reading, the value may be all zeroes,
			// thus 'deletion' which is not supported on stacktrie
			break
		}
		if _, present := keys[string(k)]; present {
			// This key is a duplicate, ignore it
			continue
		}
		keys[string(k)] = struct{}{}
		vals = append(vals, &kv{k: k, v: v})

		trieA.MustUpdate(k, v)
	}
	if len(vals) == 0 {
		return
	}
	// Flush trie -> database
	rootA, nodes := trieA.Commit(false)
	if nodes != nil {
		dbA.Update(rootA, types.EmptyRootHash, trienode.NewWithNodeSet(nodes))
	}
	// Flush memdb -> disk (sponge)
	dbA.Commit(rootA)

	// Stacktrie requires sorted insertion
	slices.SortFunc(vals, (*kv).cmp)

	for _, kv := range vals {
		if debugging {
			fmt.Printf("{\"%#x\" , \"%#x\"} // stacktrie.Update\n", kv.k, kv.v)
		}
		trieB.Update(kv.k, kv.v)
	}
	rootB := trieB.Hash()
	if rootA != rootB {
		panic(fmt.Sprintf("roots differ: (trie) %x != %x (stacktrie)", rootA, rootB))
	}

	// Ensure all the nodes are persisted correctly
	var (
		nodeset = make(map[string][]byte) // path -> blob
		trieC   = NewStackTrie(func(path []byte, hash common.Hash, blob []byte) {
			if crypto.Keccak256Hash(blob) != hash {
				panic("invalid node blob")
			}
			nodeset[string(path)] = common.CopyBytes(blob)
		})
		checked int
	)
	for _, kv := range vals {
		trieC.Update(kv.k, kv.v)
	}
	rootC := trieC.Hash()
	if rootA != rootC {
		panic(fmt.Sprintf("roots differ: (trie) %x != %x (stacktrie)", rootA, rootC))
	}
	trieA, _ = New(TrieID(rootA), dbA)
	iterA := trieA.MustNodeIterator(nil)
	for iterA.Next(true) {
		if iterA.Hash() == (common.Hash{}) {
			if _, present := nodeset[string(iterA.Path())]; present {
				panic("unexpected tiny node")
			}
			continue
		}
		nodeBlob, present := nodeset[string(iterA.Path())]
		if !present {
			panic("missing node")
		}
		if !bytes.Equal(nodeBlob, iterA.NodeBlob()) {
			panic("node blob is not matched")
		}
		checked += 1
	}
	if checked != len(nodeset) {
		panic("node number is not matched")
	}
}
