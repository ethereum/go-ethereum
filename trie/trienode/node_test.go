// Copyright 2023 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package trienode

import (
	"bytes"
	"crypto/rand"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/testrand"
)

func makeTestSet(owner common.Hash, n int, paths [][]byte) *NodeSet {
	set := NewNodeSet(owner)
	for i := 0; i < n*3/4; i++ {
		path := testrand.Bytes(10)
		blob := testrand.Bytes(100)
		set.AddNode(path, NewNodeWithPrev(crypto.Keccak256Hash(blob), blob, testrand.Bytes(100)))
	}
	for i := 0; i < n/4; i++ {
		path := testrand.Bytes(10)
		set.AddNode(path, NewDeletedWithPrev(testrand.Bytes(100)))
	}
	for i := 0; i < len(paths); i++ {
		if i%3 == 0 {
			set.AddNode(paths[i], NewDeletedWithPrev(testrand.Bytes(100)))
		} else {
			blob := testrand.Bytes(100)
			set.AddNode(paths[i], NewNodeWithPrev(crypto.Keccak256Hash(blob), blob, testrand.Bytes(100)))
		}
	}
	return set
}

func copyNodeSet(set *NodeSet) *NodeSet {
	cpy := &NodeSet{
		Owner:   set.Owner,
		Leaves:  slices.Clone(set.Leaves),
		updates: set.updates,
		deletes: set.deletes,
		Nodes:   maps.Clone(set.Nodes),
		Origins: maps.Clone(set.Origins),
	}
	return cpy
}

func TestNodeSetMerge(t *testing.T) {
	var shared [][]byte
	for i := 0; i < 2; i++ {
		shared = append(shared, testrand.Bytes(10))
	}
	owner := testrand.Hash()
	setA := makeTestSet(owner, 20, shared)
	cpyA := copyNodeSet(setA)

	setB := makeTestSet(owner, 20, shared)
	setA.Merge(setB)

	for path, node := range setA.Nodes {
		nA, inA := cpyA.Nodes[path]
		nB, inB := setB.Nodes[path]

		switch {
		case inA && inB:
			origin := setA.Origins[path]
			if !bytes.Equal(origin, cpyA.Origins[path]) {
				t.Errorf("Unexpected origin, path %v: want: %v, got: %v", []byte(path), cpyA.Origins[path], origin)
			}
			if !reflect.DeepEqual(node, nB) {
				t.Errorf("Unexpected node, path %v: want: %v, got: %v", []byte(path), spew.Sdump(nB), spew.Sdump(node))
			}
		case !inA && inB:
			origin := setA.Origins[path]
			if !bytes.Equal(origin, setB.Origins[path]) {
				t.Errorf("Unexpected origin, path %v: want: %v, got: %v", []byte(path), setB.Origins[path], origin)
			}
			if !reflect.DeepEqual(node, nB) {
				t.Errorf("Unexpected node, path %v: want: %v, got: %v", []byte(path), spew.Sdump(nB), spew.Sdump(node))
			}
		case inA && !inB:
			origin := setA.Origins[path]
			if !bytes.Equal(origin, cpyA.Origins[path]) {
				t.Errorf("Unexpected origin, path %v: want: %v, got: %v", []byte(path), cpyA.Origins[path], origin)
			}
			if !reflect.DeepEqual(node, nA) {
				t.Errorf("Unexpected node, path %v: want: %v, got: %v", []byte(path), spew.Sdump(nA), spew.Sdump(node))
			}
		default:
			t.Errorf("Unexpected node, %v", []byte(path))
		}
	}
}

func BenchmarkMerge(b *testing.B) {
	b.Run("1K", func(b *testing.B) {
		benchmarkMerge(b, 1000)
	})
	b.Run("10K", func(b *testing.B) {
		benchmarkMerge(b, 10_000)
	})
}

func benchmarkMerge(b *testing.B, count int) {
	x := NewNodeSet(common.Hash{})
	y := NewNodeSet(common.Hash{})
	addNode := func(s *NodeSet) {
		path := make([]byte, 4)
		rand.Read(path)
		blob := make([]byte, 32)
		rand.Read(blob)
		hash := crypto.Keccak256Hash(blob)
		s.AddNode(path, NewNodeWithPrev(hash, blob, nil))
	}
	for i := 0; i < count; i++ {
		// Random path of 4 nibbles
		addNode(x)
		addNode(y)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Store set x into a backup
		z := NewNodeSet(common.Hash{})
		z.Merge(x)
		// Merge y into x
		x.Merge(y)
		x = z
	}
}
