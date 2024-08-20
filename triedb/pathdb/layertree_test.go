// Copyright 2024 The go-ethereum Authors
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

package pathdb

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/triestate"
)

func newTestLayerTree() *layerTree {
	db := New(rawdb.NewMemoryDatabase(), nil, false)
	l := newDiskLayer(common.Hash{0x1}, 0, db, nil, newNodeBuffer(0, nil, 0))
	t := newLayerTree(l)
	return t
}

func TestLayerCap(t *testing.T) {
	var cases = []struct {
		init     func() *layerTree
		head     common.Hash
		layers   int
		base     common.Hash
		snapshot map[common.Hash]struct{}
	}{
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			// Chain:
			//   C2->C3->C4 (HEAD)
			head:   common.Hash{0x4},
			layers: 2,
			base:   common.Hash{0x2},
			snapshot: map[common.Hash]struct{}{
				common.Hash{0x2}: {},
				common.Hash{0x3}: {},
				common.Hash{0x4}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			// Chain:
			//   C3->C4 (HEAD)
			head:   common.Hash{0x4},
			layers: 1,
			base:   common.Hash{0x3},
			snapshot: map[common.Hash]struct{}{
				common.Hash{0x3}: {},
				common.Hash{0x4}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			// Chain:
			//   C4 (HEAD)
			head:   common.Hash{0x4},
			layers: 0,
			base:   common.Hash{0x4},
			snapshot: map[common.Hash]struct{}{
				common.Hash{0x4}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			//     ->C2'->C3'->C4'
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2a}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3a}, common.Hash{0x2a}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4a}, common.Hash{0x3a}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x2b}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3b}, common.Hash{0x2b}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4b}, common.Hash{0x3b}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			// Chain:
			//   C2->C3->C4 (HEAD)
			head:   common.Hash{0x4a},
			layers: 2,
			base:   common.Hash{0x2a},
			snapshot: map[common.Hash]struct{}{
				common.Hash{0x4a}: {},
				common.Hash{0x3a}: {},
				common.Hash{0x2a}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			//     ->C2'->C3'->C4'
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2a}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3a}, common.Hash{0x2a}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4a}, common.Hash{0x3a}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x2b}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3b}, common.Hash{0x2b}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4b}, common.Hash{0x3b}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			// Chain:
			//   C3->C4 (HEAD)
			head:   common.Hash{0x4a},
			layers: 1,
			base:   common.Hash{0x3a},
			snapshot: map[common.Hash]struct{}{
				common.Hash{0x4a}: {},
				common.Hash{0x3a}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			//         ->C3'->C4'
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3a}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4a}, common.Hash{0x3a}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3b}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4b}, common.Hash{0x3b}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			// Chain:
			//   C2->C3->C4 (HEAD)
			//     ->C3'->C4'
			head:   common.Hash{0x4a},
			layers: 2,
			base:   common.Hash{0x2},
			snapshot: map[common.Hash]struct{}{
				common.Hash{0x4a}: {},
				common.Hash{0x3a}: {},
				common.Hash{0x4b}: {},
				common.Hash{0x3b}: {},
				common.Hash{0x2}:  {},
			},
		},
	}
	for _, c := range cases {
		tr := c.init()
		if err := tr.cap(c.head, c.layers); err != nil {
			t.Fatalf("Failed to cap the layer tree %v", err)
		}
		if tr.bottom().root != c.base {
			t.Fatalf("Unexpected bottom layer tree root, want %v, got %v", c.base, tr.bottom().root)
		}
		if len(c.snapshot) != len(tr.layers) {
			t.Fatalf("Unexpected layer tree size, want %v, got %v", len(c.snapshot), len(tr.layers))
		}
		for h := range tr.layers {
			if _, ok := c.snapshot[h]; !ok {
				t.Fatalf("Unexpected layer %v", h)
			}
		}
	}
}

func TestBaseLayer(t *testing.T) {
	tr := newTestLayerTree()

	var cases = []struct {
		op   func()
		base common.Hash
	}{
		// no operation
		{
			func() {},
			common.Hash{0x1},
		},
		// add layers on top
		{
			func() {
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
			},
			common.Hash{0x1},
		},
		// forcibly flush all the layers
		{
			func() {
				tr.cap(common.Hash{0x3}, 0)
			},
			common.Hash{0x3},
		},
		// add layers on top and cap
		{
			func() {
				tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x5}, common.Hash{0x4}, 4, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x6}, common.Hash{0x5}, 5, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.cap(common.Hash{0x6}, 2)
			},
			common.Hash{0x4},
		},
	}
	for _, c := range cases {
		c.op()
		if tr.base.rootHash() != c.base {
			t.Fatalf("Unexpected base root, want %v, got: %v", c.base, tr.base.rootHash())
		}
	}
}

func TestDescendant(t *testing.T) {
	var cases = []struct {
		init      func() *layerTree
		snapshotA map[common.Hash]map[common.Hash]struct{}
		op        func(tr *layerTree)
		snapshotB map[common.Hash]map[common.Hash]struct{}
	}{
		{
			// Chain:
			//   C1->C2 (HEAD)
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			snapshotA: map[common.Hash]map[common.Hash]struct{}{},
			// Chain:
			//   C1->C2->C3 (HEAD)
			op: func(tr *layerTree) {
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
			},
			snapshotB: map[common.Hash]map[common.Hash]struct{}{
				common.Hash{0x2}: {
					common.Hash{0x3}: {},
				},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			snapshotA: map[common.Hash]map[common.Hash]struct{}{
				common.Hash{0x2}: {
					common.Hash{0x3}: {},
					common.Hash{0x4}: {},
				},
				common.Hash{0x3}: {
					common.Hash{0x4}: {},
				},
			},
			// Chain:
			//   C2->C3->C4 (HEAD)
			op: func(tr *layerTree) {
				tr.cap(common.Hash{0x4}, 2)
			},
			snapshotB: map[common.Hash]map[common.Hash]struct{}{
				common.Hash{0x3}: {
					common.Hash{0x4}: {},
				},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			snapshotA: map[common.Hash]map[common.Hash]struct{}{
				common.Hash{0x2}: {
					common.Hash{0x3}: {},
					common.Hash{0x4}: {},
				},
				common.Hash{0x3}: {
					common.Hash{0x4}: {},
				},
			},
			// Chain:
			//   C3->C4 (HEAD)
			op: func(tr *layerTree) {
				tr.cap(common.Hash{0x4}, 1)
			},
			snapshotB: map[common.Hash]map[common.Hash]struct{}{},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			snapshotA: map[common.Hash]map[common.Hash]struct{}{
				common.Hash{0x2}: {
					common.Hash{0x3}: {},
					common.Hash{0x4}: {},
				},
				common.Hash{0x3}: {
					common.Hash{0x4}: {},
				},
			},
			// Chain:
			//   C4 (HEAD)
			op: func(tr *layerTree) {
				tr.cap(common.Hash{0x4}, 0)
			},
			snapshotB: map[common.Hash]map[common.Hash]struct{}{},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			//     ->C2'->C3'->C4'
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2a}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3a}, common.Hash{0x2a}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4a}, common.Hash{0x3a}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x2b}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3b}, common.Hash{0x2b}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4b}, common.Hash{0x3b}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			snapshotA: map[common.Hash]map[common.Hash]struct{}{
				common.Hash{0x2a}: {
					common.Hash{0x3a}: {},
					common.Hash{0x4a}: {},
				},
				common.Hash{0x3a}: {
					common.Hash{0x4a}: {},
				},
				common.Hash{0x2b}: {
					common.Hash{0x3b}: {},
					common.Hash{0x4b}: {},
				},
				common.Hash{0x3b}: {
					common.Hash{0x4b}: {},
				},
			},
			// Chain:
			//   C2->C3->C4 (HEAD)
			op: func(tr *layerTree) {
				tr.cap(common.Hash{0x4a}, 2)
			},
			snapshotB: map[common.Hash]map[common.Hash]struct{}{
				common.Hash{0x3a}: {
					common.Hash{0x4a}: {},
				},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			//     ->C2'->C3'->C4'
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2a}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3a}, common.Hash{0x2a}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4a}, common.Hash{0x3a}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x2b}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3b}, common.Hash{0x2b}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4b}, common.Hash{0x3b}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			snapshotA: map[common.Hash]map[common.Hash]struct{}{
				common.Hash{0x2a}: {
					common.Hash{0x3a}: {},
					common.Hash{0x4a}: {},
				},
				common.Hash{0x3a}: {
					common.Hash{0x4a}: {},
				},
				common.Hash{0x2b}: {
					common.Hash{0x3b}: {},
					common.Hash{0x4b}: {},
				},
				common.Hash{0x3b}: {
					common.Hash{0x4b}: {},
				},
			},
			// Chain:
			//   C3->C4 (HEAD)
			op: func(tr *layerTree) {
				tr.cap(common.Hash{0x4a}, 1)
			},
			snapshotB: map[common.Hash]map[common.Hash]struct{}{},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			//         ->C3'->C4'
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3a}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4a}, common.Hash{0x3a}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3b}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4b}, common.Hash{0x3b}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			snapshotA: map[common.Hash]map[common.Hash]struct{}{
				common.Hash{0x2}: {
					common.Hash{0x3a}: {},
					common.Hash{0x4a}: {},
					common.Hash{0x3b}: {},
					common.Hash{0x4b}: {},
				},
				common.Hash{0x3a}: {
					common.Hash{0x4a}: {},
				},
				common.Hash{0x3b}: {
					common.Hash{0x4b}: {},
				},
			},
			// Chain:
			//   C2->C3->C4 (HEAD)
			//     ->C3'->C4'
			op: func(tr *layerTree) {
				tr.cap(common.Hash{0x4a}, 2)
			},
			snapshotB: map[common.Hash]map[common.Hash]struct{}{
				common.Hash{0x3a}: {
					common.Hash{0x4a}: {},
				},
				common.Hash{0x3b}: {
					common.Hash{0x4b}: {},
				},
			},
		},
	}
	check := func(setA, setB map[common.Hash]map[common.Hash]struct{}) bool {
		if len(setA) != len(setB) {
			return false
		}
		for h, subA := range setA {
			subB, ok := setB[h]
			if !ok {
				return false
			}
			if len(subA) != len(subB) {
				return false
			}
			for hh := range subA {
				if _, ok := subB[hh]; !ok {
					return false
				}
			}
		}
		return true
	}
	for _, c := range cases {
		tr := c.init()
		if !check(c.snapshotA, tr.descendants) {
			t.Fatalf("Unexpected descendants")
		}
		c.op(tr)
		if !check(c.snapshotB, tr.descendants) {
			t.Fatalf("Unexpected descendants")
		}
	}
}

func TestStale(t *testing.T) {
	var cases = []struct {
		init  func() *layerTree
		op    func(tr *layerTree)
		stale map[common.Hash]struct{}
		live  map[common.Hash]struct{}
	}{
		{
			// Chain:
			//   C1->C2 (HEAD)
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			// Chain:
			//   C1->C2->C3 (HEAD)
			op: func(tr *layerTree) {
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
			},
			stale: map[common.Hash]struct{}{},
			live: map[common.Hash]struct{}{
				common.Hash{0x1}: {},
				common.Hash{0x2}: {},
				common.Hash{0x3}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			// Chain:
			//   C2->C3->C4 (HEAD)
			op: func(tr *layerTree) {
				tr.cap(common.Hash{0x4}, 2)
			},
			stale: map[common.Hash]struct{}{
				common.Hash{0x1}: {},
				common.Hash{0x2}: {}, // old diff layer
			},
			live: map[common.Hash]struct{}{
				common.Hash{0x2}: {}, // new disk layer
				common.Hash{0x3}: {},
				common.Hash{0x4}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			// Chain:
			//   C3->C4 (HEAD)
			op: func(tr *layerTree) {
				tr.cap(common.Hash{0x4}, 1)
			},
			stale: map[common.Hash]struct{}{
				common.Hash{0x1}: {},
				common.Hash{0x2}: {},
				common.Hash{0x3}: {}, // old diff
			},
			live: map[common.Hash]struct{}{
				common.Hash{0x3}: {}, // new disk
				common.Hash{0x4}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			// Chain:
			//   C4 (HEAD)
			op: func(tr *layerTree) {
				tr.cap(common.Hash{0x4}, 0)
			},
			stale: map[common.Hash]struct{}{
				common.Hash{0x1}: {},
				common.Hash{0x2}: {},
				common.Hash{0x3}: {},
				common.Hash{0x4}: {}, // old diff
			},
			live: map[common.Hash]struct{}{
				common.Hash{0x4}: {}, // new disk
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			//     ->C2'->C3'->C4'
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2a}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3a}, common.Hash{0x2a}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4a}, common.Hash{0x3a}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x2b}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3b}, common.Hash{0x2b}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4b}, common.Hash{0x3b}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			// Chain:
			//   C2->C3->C4 (HEAD)
			op: func(tr *layerTree) {
				tr.cap(common.Hash{0x4a}, 2)
			},
			stale: map[common.Hash]struct{}{
				common.Hash{0x1}:  {},
				common.Hash{0x2a}: {}, // old diff
				common.Hash{0x2b}: {},
				common.Hash{0x3b}: {},
				common.Hash{0x4b}: {},
			},
			live: map[common.Hash]struct{}{
				common.Hash{0x2a}: {}, // new disk
				common.Hash{0x3a}: {},
				common.Hash{0x4a}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			//     ->C2'->C3'->C4'
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2a}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3a}, common.Hash{0x2a}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4a}, common.Hash{0x3a}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x2b}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3b}, common.Hash{0x2b}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4b}, common.Hash{0x3b}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			// Chain:
			//   C3->C4 (HEAD)
			op: func(tr *layerTree) {
				tr.cap(common.Hash{0x4a}, 1)
			},
			stale: map[common.Hash]struct{}{
				common.Hash{0x1}:  {},
				common.Hash{0x2a}: {},
				common.Hash{0x2b}: {},
				common.Hash{0x3a}: {}, // old diff
				common.Hash{0x3b}: {},
				common.Hash{0x4b}: {},
			},
			live: map[common.Hash]struct{}{
				common.Hash{0x3a}: {}, // new disk
				common.Hash{0x4a}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			//         ->C3'->C4'
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3a}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4a}, common.Hash{0x3a}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x3b}, common.Hash{0x2}, 2, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				tr.add(common.Hash{0x4b}, common.Hash{0x3b}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))
				return tr
			},
			// Chain:
			//   C2->C3->C4 (HEAD)
			//     ->C3'->C4'
			op: func(tr *layerTree) {
				tr.cap(common.Hash{0x4a}, 2)
			},
			stale: map[common.Hash]struct{}{
				common.Hash{0x1}: {},
				common.Hash{0x2}: {}, // old diff
			},
			live: map[common.Hash]struct{}{
				common.Hash{0x2}:  {}, // new disk
				common.Hash{0x3a}: {},
				common.Hash{0x4a}: {},
				common.Hash{0x3b}: {},
				common.Hash{0x4b}: {},
			},
		},
	}
	for _, c := range cases {
		tr := c.init()
		var stale []layer
		for h := range c.stale {
			stale = append(stale, tr.get(h))
		}
		c.op(tr)

		for _, l := range stale {
			if !l.isStale() {
				t.Fatalf("the layer is expected to be stale, %x", l.rootHash())
			}
		}
		for h := range c.live {
			l := tr.get(h)
			if l == nil {
				t.Fatalf("the layer is not reachable, %x", h)
			}
			if l.isStale() {
				t.Fatalf("the layer is expected to be non-stale, %x", l.rootHash())
			}
		}
	}
}
