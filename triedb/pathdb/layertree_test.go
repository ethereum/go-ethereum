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
	"bytes"
	"errors"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

func newTestLayerTree() *layerTree {
	db := New(rawdb.NewMemoryDatabase(), nil, false)
	l := newDiskLayer(common.Hash{0x1}, 0, db, nil, nil, newBuffer(0, nil, nil, 0), nil)
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
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				return tr
			},
			// Chain:
			//   C2->C3->C4 (HEAD)
			head:   common.Hash{0x4},
			layers: 2,
			base:   common.Hash{0x2},
			snapshot: map[common.Hash]struct{}{
				{0x2}: {},
				{0x3}: {},
				{0x4}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				return tr
			},
			// Chain:
			//   C3->C4 (HEAD)
			head:   common.Hash{0x4},
			layers: 1,
			base:   common.Hash{0x3},
			snapshot: map[common.Hash]struct{}{
				{0x3}: {},
				{0x4}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				return tr
			},
			// Chain:
			//   C4 (HEAD)
			head:   common.Hash{0x4},
			layers: 0,
			base:   common.Hash{0x4},
			snapshot: map[common.Hash]struct{}{
				{0x4}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			//     ->C2'->C3'->C4'
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2a}, common.Hash{0x1}, 1, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x3a}, common.Hash{0x2a}, 2, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x4a}, common.Hash{0x3a}, 3, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x2b}, common.Hash{0x1}, 1, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x3b}, common.Hash{0x2b}, 2, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x4b}, common.Hash{0x3b}, 3, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				return tr
			},
			// Chain:
			//   C2->C3->C4 (HEAD)
			head:   common.Hash{0x4a},
			layers: 2,
			base:   common.Hash{0x2a},
			snapshot: map[common.Hash]struct{}{
				{0x4a}: {},
				{0x3a}: {},
				{0x2a}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			//     ->C2'->C3'->C4'
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2a}, common.Hash{0x1}, 1, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x3a}, common.Hash{0x2a}, 2, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x4a}, common.Hash{0x3a}, 3, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x2b}, common.Hash{0x1}, 1, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x3b}, common.Hash{0x2b}, 2, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x4b}, common.Hash{0x3b}, 3, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				return tr
			},
			// Chain:
			//   C3->C4 (HEAD)
			head:   common.Hash{0x4a},
			layers: 1,
			base:   common.Hash{0x3a},
			snapshot: map[common.Hash]struct{}{
				{0x4a}: {},
				{0x3a}: {},
			},
		},
		{
			// Chain:
			//   C1->C2->C3->C4 (HEAD)
			//         ->C3'->C4'
			init: func() *layerTree {
				tr := newTestLayerTree()
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x3a}, common.Hash{0x2}, 2, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x4a}, common.Hash{0x3a}, 3, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x3b}, common.Hash{0x2}, 2, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x4b}, common.Hash{0x3b}, 3, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				return tr
			},
			// Chain:
			//   C2->C3->C4 (HEAD)
			//     ->C3'->C4'
			head:   common.Hash{0x4a},
			layers: 2,
			base:   common.Hash{0x2},
			snapshot: map[common.Hash]struct{}{
				{0x4a}: {},
				{0x3a}: {},
				{0x4b}: {},
				{0x3b}: {},
				{0x2}:  {},
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
		// Chain:
		//   C1 (HEAD)
		{
			func() {},
			common.Hash{0x1},
		},
		// Chain:
		//   C1->C2->C3 (HEAD)
		{
			func() {
				tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
			},
			common.Hash{0x1},
		},
		// Chain:
		//   C3 (HEAD)
		{
			func() {
				tr.cap(common.Hash{0x3}, 0)
			},
			common.Hash{0x3},
		},
		// Chain:
		//   C4->C5->C6 (HEAD)
		{
			func() {
				tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x5}, common.Hash{0x4}, 4, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
				tr.add(common.Hash{0x6}, common.Hash{0x5}, 5, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(nil, nil, nil, nil, false))
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

func TestDuplicateRootLookup(t *testing.T) {
	// Chain:
	//   C1->C2->C3 (HEAD)
	tr := newTestLayerTree() // base = 0x1
	tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, NewNodeSetWithOrigin(nil, nil),
		NewStateSetWithOrigin(randomAccountSet("0xa"), randomStorageSet([]string{"0xa"}, [][]string{{"0x1"}}, nil), nil, nil, false))
	tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, NewNodeSetWithOrigin(nil, nil),
		NewStateSetWithOrigin(randomAccountSet("0xa"), randomStorageSet([]string{"0xa"}, [][]string{{"0x1"}}, nil), nil, nil, false))

	// A fork block with the same state root as C2; inserting it must not
	// pollute the lookup history for the canonical descendant C3.
	tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, NewNodeSetWithOrigin(nil, nil),
		NewStateSetWithOrigin(randomAccountSet("0xa"), randomStorageSet([]string{"0xa"}, [][]string{{"0x1"}}, nil), nil, nil, false))
	if n := tr.len(); n != 3 {
		t.Fatalf("duplicate root insert changed layer count, got %d, want 3", n)
	}

	l, err := tr.lookupAccount(common.HexToHash("0xa"), common.Hash{0x3})
	if err != nil {
		t.Fatalf("account lookup failed: %v", err)
	}
	if l.rootHash() != (common.Hash{0x3}) {
		t.Errorf("unexpected account tip, want %x, got %x", common.Hash{0x3}, l.rootHash())
	}

	l, err = tr.lookupStorage(common.HexToHash("0xa"), common.HexToHash("0x1"), common.Hash{0x3})
	if err != nil {
		t.Fatalf("storage lookup failed: %v", err)
	}
	if l.rootHash() != (common.Hash{0x3}) {
		t.Errorf("unexpected storage tip, want %x, got %x", common.Hash{0x3}, l.rootHash())
	}
}

func TestAccountLookup(t *testing.T) {
	// Chain:
	//   C1->C2->C3->C4 (HEAD)
	tr := newTestLayerTree() // base = 0x1
	tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, NewNodeSetWithOrigin(nil, nil),
		NewStateSetWithOrigin(randomAccountSet("0xa"), nil, nil, nil, false))
	tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, NewNodeSetWithOrigin(nil, nil),
		NewStateSetWithOrigin(randomAccountSet("0xb"), nil, nil, nil, false))
	tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, NewNodeSetWithOrigin(nil, nil),
		NewStateSetWithOrigin(randomAccountSet("0xa", "0xc"), nil, nil, nil, false))

	var cases = []struct {
		account common.Hash
		state   common.Hash
		expect  common.Hash
	}{
		{
			// unknown account
			common.HexToHash("0xd"), common.Hash{0x4}, common.Hash{0x1},
		},
		/*
			lookup account from the top
		*/
		{
			common.HexToHash("0xa"), common.Hash{0x4}, common.Hash{0x4},
		},
		{
			common.HexToHash("0xb"), common.Hash{0x4}, common.Hash{0x3},
		},
		{
			common.HexToHash("0xc"), common.Hash{0x4}, common.Hash{0x4},
		},
		/*
			lookup account from the middle
		*/
		{
			common.HexToHash("0xa"), common.Hash{0x3}, common.Hash{0x2},
		},
		{
			common.HexToHash("0xb"), common.Hash{0x3}, common.Hash{0x3},
		},
		{
			common.HexToHash("0xc"), common.Hash{0x3}, common.Hash{0x1}, // not found
		},
		{
			common.HexToHash("0xa"), common.Hash{0x2}, common.Hash{0x2},
		},
		{
			common.HexToHash("0xb"), common.Hash{0x2}, common.Hash{0x1}, // not found
		},
		{
			common.HexToHash("0xc"), common.Hash{0x2}, common.Hash{0x1}, // not found
		},
		/*
			lookup account from the bottom
		*/
		{
			common.HexToHash("0xa"), common.Hash{0x1}, common.Hash{0x1}, // not found
		},
		{
			common.HexToHash("0xb"), common.Hash{0x1}, common.Hash{0x1}, // not found
		},
		{
			common.HexToHash("0xc"), common.Hash{0x1}, common.Hash{0x1}, // not found
		},
	}
	for i, c := range cases {
		l, err := tr.lookupAccount(c.account, c.state)
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		if l.rootHash() != c.expect {
			t.Errorf("Unexpected tiphash, %d, want: %x, got: %x", i, c.expect, l.rootHash())
		}
	}

	// Chain:
	//   C3->C4 (HEAD)
	tr.cap(common.Hash{0x4}, 1)

	cases2 := []struct {
		account   common.Hash
		state     common.Hash
		expect    common.Hash
		expectErr error
	}{
		{
			// unknown account
			common.HexToHash("0xd"), common.Hash{0x4}, common.Hash{0x3}, nil,
		},
		/*
			lookup account from the top
		*/
		{
			common.HexToHash("0xa"), common.Hash{0x4}, common.Hash{0x4}, nil,
		},
		{
			common.HexToHash("0xb"), common.Hash{0x4}, common.Hash{0x3}, nil,
		},
		{
			common.HexToHash("0xc"), common.Hash{0x4}, common.Hash{0x4}, nil,
		},
		/*
			lookup account from the bottom
		*/
		{
			common.HexToHash("0xa"), common.Hash{0x3}, common.Hash{0x3}, nil,
		},
		{
			common.HexToHash("0xb"), common.Hash{0x3}, common.Hash{0x3}, nil,
		},
		{
			common.HexToHash("0xc"), common.Hash{0x3}, common.Hash{0x3}, nil, // not found
		},
		/*
			stale states
		*/
		{
			common.HexToHash("0xa"), common.Hash{0x2}, common.Hash{}, errSnapshotStale,
		},
		{
			common.HexToHash("0xb"), common.Hash{0x2}, common.Hash{}, errSnapshotStale,
		},
		{
			common.HexToHash("0xc"), common.Hash{0x2}, common.Hash{}, errSnapshotStale,
		},
		{
			common.HexToHash("0xa"), common.Hash{0x1}, common.Hash{}, errSnapshotStale,
		},
		{
			common.HexToHash("0xb"), common.Hash{0x1}, common.Hash{}, errSnapshotStale,
		},
		{
			common.HexToHash("0xc"), common.Hash{0x1}, common.Hash{}, errSnapshotStale,
		},
	}
	for i, c := range cases2 {
		l, err := tr.lookupAccount(c.account, c.state)
		if c.expectErr != nil {
			if !errors.Is(err, c.expectErr) {
				t.Fatalf("%d: unexpected error, want %v, got %v", i, c.expectErr, err)
			}
		}
		if c.expectErr == nil {
			if err != nil {
				t.Fatalf("%d: %v", i, err)
			}
			if l.rootHash() != c.expect {
				t.Errorf("Unexpected tiphash, %d, want: %x, got: %x", i, c.expect, l.rootHash())
			}
		}
	}
}

func TestStorageLookup(t *testing.T) {
	// Chain:
	//   C1->C2->C3->C4 (HEAD)
	tr := newTestLayerTree() // base = 0x1
	tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, NewNodeSetWithOrigin(nil, nil),
		NewStateSetWithOrigin(randomAccountSet("0xa"), randomStorageSet([]string{"0xa"}, [][]string{{"0x1"}}, nil), nil, nil, false))
	tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, NewNodeSetWithOrigin(nil, nil),
		NewStateSetWithOrigin(randomAccountSet("0xa"), randomStorageSet([]string{"0xa"}, [][]string{{"0x2"}}, nil), nil, nil, false))
	tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, NewNodeSetWithOrigin(nil, nil),
		NewStateSetWithOrigin(randomAccountSet("0xa"), randomStorageSet([]string{"0xa"}, [][]string{{"0x1", "0x3"}}, nil), nil, nil, false))

	var cases = []struct {
		storage common.Hash
		state   common.Hash
		expect  common.Hash
	}{
		{
			// unknown storage slot
			common.HexToHash("0x4"), common.Hash{0x4}, common.Hash{0x1},
		},
		/*
			lookup storage slot from the top
		*/
		{
			common.HexToHash("0x1"), common.Hash{0x4}, common.Hash{0x4},
		},
		{
			common.HexToHash("0x2"), common.Hash{0x4}, common.Hash{0x3},
		},
		{
			common.HexToHash("0x3"), common.Hash{0x4}, common.Hash{0x4},
		},
		/*
			lookup storage slot from the middle
		*/
		{
			common.HexToHash("0x1"), common.Hash{0x3}, common.Hash{0x2},
		},
		{
			common.HexToHash("0x2"), common.Hash{0x3}, common.Hash{0x3},
		},
		{
			common.HexToHash("0x3"), common.Hash{0x3}, common.Hash{0x1}, // not found
		},
		{
			common.HexToHash("0x1"), common.Hash{0x2}, common.Hash{0x2},
		},
		{
			common.HexToHash("0x2"), common.Hash{0x2}, common.Hash{0x1}, // not found
		},
		{
			common.HexToHash("0x3"), common.Hash{0x2}, common.Hash{0x1}, // not found
		},
		/*
			lookup storage slot from the bottom
		*/
		{
			common.HexToHash("0x1"), common.Hash{0x1}, common.Hash{0x1}, // not found
		},
		{
			common.HexToHash("0x2"), common.Hash{0x1}, common.Hash{0x1}, // not found
		},
		{
			common.HexToHash("0x3"), common.Hash{0x1}, common.Hash{0x1}, // not found
		},
	}
	for i, c := range cases {
		l, err := tr.lookupStorage(common.HexToHash("0xa"), c.storage, c.state)
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		if l.rootHash() != c.expect {
			t.Errorf("Unexpected tiphash, %d, want: %x, got: %x", i, c.expect, l.rootHash())
		}
	}

	// Chain:
	//   C3->C4 (HEAD)
	tr.cap(common.Hash{0x4}, 1)

	cases2 := []struct {
		storage   common.Hash
		state     common.Hash
		expect    common.Hash
		expectErr error
	}{
		{
			// unknown storage slot
			common.HexToHash("0x4"), common.Hash{0x4}, common.Hash{0x3}, nil,
		},
		/*
			lookup account from the top
		*/
		{
			common.HexToHash("0x1"), common.Hash{0x4}, common.Hash{0x4}, nil,
		},
		{
			common.HexToHash("0x2"), common.Hash{0x4}, common.Hash{0x3}, nil,
		},
		{
			common.HexToHash("0x3"), common.Hash{0x4}, common.Hash{0x4}, nil,
		},
		/*
			lookup account from the bottom
		*/
		{
			common.HexToHash("0x1"), common.Hash{0x3}, common.Hash{0x3}, nil,
		},
		{
			common.HexToHash("0x2"), common.Hash{0x3}, common.Hash{0x3}, nil,
		},
		{
			common.HexToHash("0x3"), common.Hash{0x3}, common.Hash{0x3}, nil, // not found
		},
		/*
			stale states
		*/
		{
			common.HexToHash("0x1"), common.Hash{0x2}, common.Hash{}, errSnapshotStale,
		},
		{
			common.HexToHash("0x2"), common.Hash{0x2}, common.Hash{}, errSnapshotStale,
		},
		{
			common.HexToHash("0x3"), common.Hash{0x2}, common.Hash{}, errSnapshotStale,
		},
		{
			common.HexToHash("0x1"), common.Hash{0x1}, common.Hash{}, errSnapshotStale,
		},
		{
			common.HexToHash("0x2"), common.Hash{0x1}, common.Hash{}, errSnapshotStale,
		},
		{
			common.HexToHash("0x3"), common.Hash{0x1}, common.Hash{}, errSnapshotStale,
		},
	}
	for i, c := range cases2 {
		l, err := tr.lookupStorage(common.HexToHash("0xa"), c.storage, c.state)
		if c.expectErr != nil {
			if !errors.Is(err, c.expectErr) {
				t.Fatalf("%d: unexpected error, want %v, got %v", i, c.expectErr, err)
			}
		}
		if c.expectErr == nil {
			if err != nil {
				t.Fatalf("%d: %v", i, err)
			}
			if l.rootHash() != c.expect {
				t.Errorf("Unexpected tiphash, %d, want: %x, got: %x", i, c.expect, l.rootHash())
			}
		}
	}
}

// TestLookupZeroBaseRootFallback is a regression test for a sentinel
// collision in accountTip/storageTip: before the fix they returned
// common.Hash{} as both the "stale" marker and the disk-layer fallback
// when the disk root itself happened to be zero. lookupAccount/Storage
// then misreported a legitimate fallback as errSnapshotStale.
//
// On the merkle path the collision was invisible because the empty
// merkle trie hashes to types.EmptyRootHash (a concrete non-zero
// keccak), so the disk layer's root was never the zero hash in
// practice. The bug only surfaces once the disk layer root can
// legitimately be zero (for example a fresh verkle/bintrie database
// where the empty binary trie hashes to EmptyVerkleHash ==
// common.Hash{}).
//
// The test constructs a layer tree whose base layer's root IS the zero
// hash, stacks diff layers on top, and exercises four cases:
//
//  1. Look up an account NEVER written → should fall through to the
//     disk layer and return (diskLayer, nil). Before the fix this
//     returned errSnapshotStale because the fallback hash collided
//     with the sentinel.
//  2. Symmetric case for lookupStorage.
//  3. Look up an account written in a diff layer → should return that
//     diff layer (the normal happy path is unaffected by the fix).
//  4. Look up any key at a state root that isn't part of the tree
//     (neither the disk root nor a descendant of it) → MUST still
//     return errSnapshotStale. This pins the "other half" of the
//     contract so a future refactor that always returns ok=true would
//     fail here.
func TestLookupZeroBaseRootFallback(t *testing.T) {
	// Build a layer tree whose disk-layer root is common.Hash{} —
	// mirrors the bintrie/verkle configuration where the empty trie
	// hashes to EmptyVerkleHash. newTestLayerTree can't be reused
	// because it hard-codes common.Hash{0x1}.
	db := New(rawdb.NewMemoryDatabase(), nil, false)
	base := newDiskLayer(common.Hash{}, 0, db, nil, nil, newBuffer(0, nil, nil, 0), nil)
	tr := newLayerTree(base)

	// Stack two diff layers on the zero-rooted disk layer, each
	// touching a known account and slot so we have something for the
	// happy-path lookups to find later.
	if err := tr.add(
		common.Hash{0x2}, common.Hash{},
		1,
		NewNodeSetWithOrigin(nil, nil),
		NewStateSetWithOrigin(
			randomAccountSet("0xa"),
			randomStorageSet([]string{"0xa"}, [][]string{{"0x1"}}, nil),
			nil, nil, false),
	); err != nil {
		t.Fatalf("add first diff layer: %v", err)
	}
	if err := tr.add(
		common.Hash{0x3}, common.Hash{0x2},
		2,
		NewNodeSetWithOrigin(nil, nil),
		NewStateSetWithOrigin(
			randomAccountSet("0xb"),
			nil, nil, nil, false),
	); err != nil {
		t.Fatalf("add second diff layer: %v", err)
	}

	// Case 1: unknown account queried at the head. The lookup must
	// fall through the diff layers, hit the disk-layer fallback at
	// base=common.Hash{}, and return the disk layer with no error —
	// NOT errSnapshotStale.
	l, err := tr.lookupAccount(common.HexToHash("0xdead"), common.Hash{0x3})
	if err != nil {
		t.Fatalf("lookupAccount on zero-base disk layer: unexpected error %v", err)
	}
	if l.rootHash() != (common.Hash{}) {
		t.Errorf("expected fall-through to disk layer (root=0), got %x", l.rootHash())
	}

	// Case 2: symmetric check for storage. Slot 0x99 was never written,
	// so the lookup must fall through to the disk layer just like
	// Case 1.
	l, err = tr.lookupStorage(
		common.HexToHash("0xdead"), common.HexToHash("0x99"), common.Hash{0x3})
	if err != nil {
		t.Fatalf("lookupStorage on zero-base disk layer: unexpected error %v", err)
	}
	if l.rootHash() != (common.Hash{}) {
		t.Errorf("expected fall-through to disk layer (root=0), got %x", l.rootHash())
	}

	// Case 3: happy path. Account 0xa was written at diff layer 0x2.
	// The lookup must return that layer, proving the fix didn't break
	// the normal resolution path.
	l, err = tr.lookupAccount(common.HexToHash("0xa"), common.Hash{0x3})
	if err != nil {
		t.Fatalf("lookupAccount(known): %v", err)
	}
	if l.rootHash() != (common.Hash{0x2}) {
		t.Errorf("known account tip: want %x, got %x",
			common.Hash{0x2}, l.rootHash())
	}

	// Case 4: truly stale state root. This pins the other half of the
	// contract — the boolean must actually signal not-found for an
	// unknown state, otherwise a refactor that always returned
	// ok=true would still pass cases 1–3.
	_, err = tr.lookupAccount(common.HexToHash("0xa"), common.HexToHash("0xdeadbeef"))
	if !errors.Is(err, errSnapshotStale) {
		t.Errorf("lookupAccount(stale state): want errSnapshotStale, got %v", err)
	}
	_, err = tr.lookupStorage(
		common.HexToHash("0xa"), common.HexToHash("0x1"),
		common.HexToHash("0xdeadbeef"))
	if !errors.Is(err, errSnapshotStale) {
		t.Errorf("lookupStorage(stale state): want errSnapshotStale, got %v", err)
	}
}

// walkLookup resolves the layer holding an entry in the specified state by
// checking the live layers one by one, as the reference of the indexed lookups.
// The persisted layers (still linked by the forks off the disk layer) are
// regarded as the disk layer, which holds their data.
func walkLookup(tr *layerTree, state common.Hash, has func(*diffLayer) bool) layer {
	l := tr.get(state)
	for {
		if tr.get(l.rootHash()) != l {
			return tr.bottom()
		}
		dl, ok := l.(*diffLayer)
		if !ok || has(dl) {
			return l
		}
		l = dl.parentLayer()
	}
}

// checkLookups checks the indexed lookups of the given entries in the specified
// state against walking down the layers one by one.
func checkLookups(t *testing.T, tr *layerTree, state common.Hash, acct common.Hash, slot common.Hash, owner common.Hash, path []byte) {
	t.Helper()

	got, err := tr.lookupAccount(acct, state)
	if err != nil {
		t.Fatalf("Failed to resolve account: %v", err)
	}
	want := walkLookup(tr, state, func(dl *diffLayer) bool {
		_, ok := dl.states.account(acct)
		return ok
	})
	gotBlob, err1 := got.account(acct, 0)
	wantBlob, err2 := want.account(acct, 0)
	if err1 != nil || err2 != nil || !bytes.Equal(gotBlob, wantBlob) {
		t.Fatalf("Unexpected account, want %x, got %x (%v, %v)", wantBlob, gotBlob, err1, err2)
	}
	got, err = tr.lookupStorage(acct, slot, state)
	if err != nil {
		t.Fatalf("Failed to resolve storage: %v", err)
	}
	want = walkLookup(tr, state, func(dl *diffLayer) bool {
		_, ok := dl.states.storage(acct, slot)
		return ok
	})
	gotBlob, err1 = got.storage(acct, slot, 0)
	wantBlob, err2 = want.storage(acct, slot, 0)
	if err1 != nil || err2 != nil || !bytes.Equal(gotBlob, wantBlob) {
		t.Fatalf("Unexpected storage, want %x, got %x (%v, %v)", wantBlob, gotBlob, err1, err2)
	}
	got, _, err = tr.lookupNode(owner, path, state)
	if err != nil {
		t.Fatalf("Failed to resolve node: %v", err)
	}
	want = walkLookup(tr, state, func(dl *diffLayer) bool {
		_, ok := dl.nodes.node(owner, path)
		return ok
	})
	gotBlob, _, _, err1 = got.node(owner, path, 0)
	wantBlob, _, _, err2 = want.node(owner, path, 0)
	if err1 != nil || err2 != nil || !bytes.Equal(gotBlob, wantBlob) {
		t.Fatalf("Unexpected node, want %x, got %x (%v, %v)", wantBlob, gotBlob, err1, err2)
	}
}

// TestLayerTreeLookupRandom checks the indexed lookups against walking down the
// layers one by one, over a random tree being extended with forks and capped.
func TestLayerTreeLookupRandom(t *testing.T) {
	for seed := int64(1); seed <= 8; seed++ {
		testLayerTreeLookupRandom(t, seed)
	}
}

func testLayerTreeLookupRandom(t *testing.T, seed int64) {
	var (
		r        = rand.New(rand.NewSource(seed))
		tr       = newTestLayerTree()
		accounts = make([]common.Hash, 32)
		slots    = make([]common.Hash, 16)
		paths    = [][]byte{nil, {0x1}, {0x2}, {0x1, 0x3}, {0x2, 0x4}, {0x5, 0x6, 0x7}}
		block    uint64
	)
	for i := range accounts {
		r.Read(accounts[i][:])
	}
	for i := range slots {
		r.Read(slots[i][:])
	}
	owners := append([]common.Hash{{}}, accounts[:3]...)

	newSets := func() (*nodeSetWithOrigin, *StateSetWithOrigin) {
		var (
			accts  = make(map[common.Hash][]byte)
			stores = make(map[common.Hash]map[common.Hash][]byte)
			nodes  = make(map[common.Hash]map[string]*trienode.Node)
		)
		for i := 0; i < 1+r.Intn(4); i++ {
			accts[accounts[r.Intn(len(accounts))]] = []byte{byte(r.Intn(256))}
		}
		for i := 0; i < r.Intn(4); i++ {
			acct := accounts[r.Intn(4)]
			if stores[acct] == nil {
				stores[acct] = make(map[common.Hash][]byte)
			}
			stores[acct][slots[r.Intn(len(slots))]] = []byte{byte(r.Intn(256))}
		}
		for i := 0; i < r.Intn(4); i++ {
			owner := owners[r.Intn(len(owners))]
			if nodes[owner] == nil {
				nodes[owner] = make(map[string]*trienode.Node)
			}
			blob := []byte{byte(r.Intn(256))}
			nodes[owner][string(paths[r.Intn(len(paths))])] = trienode.New(crypto.Keccak256Hash(blob), blob)
		}
		return NewNodeSetWithOrigin(nodes, nil), NewStateSetWithOrigin(accts, stores, nil, nil, false)
	}
	// live returns the live layers, along with the diff layers linked to the
	// disk layer through the live layers. The forks off a persisted layer still
	// link to it, capping them is not supported.
	live := func() (roots []common.Hash, diffs []common.Hash) {
		tr.forEach(func(l layer) {
			roots = append(roots, l.rootHash())
			if _, ok := l.(*diffLayer); !ok {
				return
			}
			for cur := l; cur != tr.base; cur = cur.parentLayer() {
				if tr.layers[cur.rootHash()] != cur {
					return
				}
			}
			diffs = append(diffs, l.rootHash())
		})
		return roots, diffs
	}
	for step := 0; step < 3000; step++ {
		// Extend a random layer, forking the tree, or cap a random diff layer
		roots, diffs := live()
		if len(diffs) == 0 || r.Intn(5) != 0 {
			var root common.Hash
			r.Read(root[:])
			block++
			nodes, states := newSets()
			if err := tr.add(root, roots[r.Intn(len(roots))], block, nodes, states); err != nil {
				t.Fatalf("Failed to add layer: %v", err)
			}
		} else if err := tr.cap(diffs[r.Intn(len(diffs))], 1+r.Intn(8)); err != nil {
			t.Fatalf("Failed to cap layers: %v", err)
		}
		// Resolve through the index only, or along with the unindexed layers
		if r.Intn(2) == 0 {
			tr.index.wait()
		}
		roots, _ = live()
		for i := 0; i < 16; i++ {
			checkLookups(t, tr, roots[r.Intn(len(roots))], accounts[r.Intn(len(accounts))], slots[r.Intn(len(slots))], owners[r.Intn(len(owners))], paths[r.Intn(len(paths))])
		}
	}
}

// TestLayerIndexGrow checks the index rows are regrown for a layer with many
// keys, with the entries of the layers indexed before still resolved.
func TestLayerIndexGrow(t *testing.T) {
	var (
		r      = rand.New(rand.NewSource(1))
		tr     = newTestLayerTree()
		accts  []common.Hash
		parent = tr.base.rootHash()
	)
	add := func(n int) {
		accounts := make(map[common.Hash][]byte)
		for i := 0; i < n; i++ {
			var acct common.Hash
			r.Read(acct[:])
			accounts[acct] = []byte{byte(i)}
			accts = append(accts, acct)
		}
		var root common.Hash
		r.Read(root[:])
		if err := tr.add(root, parent, 0, NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(accounts, nil, nil, nil, false)); err != nil {
			t.Fatalf("Failed to add layer: %v", err)
		}
		parent = root
	}
	for i := 0; i < 16; i++ {
		add(16)
	}
	tr.index.wait()
	rows := len(tr.index.rows.Load().rows)

	add(4 * indexMinRows / indexRowsPerKey)
	tr.index.wait()
	if grown := len(tr.index.rows.Load().rows); grown <= rows {
		t.Fatalf("Index rows not grown, before: %d, after: %d", rows, grown)
	}
	for _, acct := range accts {
		checkLookups(t, tr, parent, acct, common.Hash{}, common.Hash{}, nil)
	}
}

// TestLayerIndexWrap checks the lookups while the slots are reused by the new
// layers, along with the rejection of the layers beyond the span of the index.
func TestLayerIndexWrap(t *testing.T) {
	var (
		tr     = newTestLayerTree()
		parent = tr.base.rootHash()
		accts  []common.Hash
	)
	add := func(i int) error {
		acct := common.Hash{0xaa, byte(i >> 8), byte(i)}
		root := common.Hash{0xbb, byte(i >> 8), byte(i)}
		accounts := map[common.Hash][]byte{acct: {byte(i)}, {0xcc}: {byte(i)}}
		if err := tr.add(root, parent, uint64(i), NewNodeSetWithOrigin(nil, nil), NewStateSetWithOrigin(accounts, nil, nil, nil, false)); err != nil {
			return err
		}
		accts = append(accts, acct)
		parent = root
		return nil
	}
	// Fill up the span of the index, the layer beyond it is rejected
	for i := 0; i < indexSlots; i++ {
		if err := add(i); err != nil {
			t.Fatalf("Failed to add layer: %v", err)
		}
	}
	if err := add(indexSlots); err == nil {
		t.Fatal("Layer beyond the span of the index is accepted")
	}
	// Wrap around the slots with the bottom layers persisted
	for i := indexSlots; i < 3*indexSlots; i++ {
		if err := tr.cap(parent, maxDiffLayers); err != nil {
			t.Fatalf("Failed to cap layers: %v", err)
		}
		if err := add(i); err != nil {
			t.Fatalf("Failed to add layer: %v", err)
		}
	}
	tr.index.wait()
	for _, acct := range append(accts, common.Hash{0xcc}) {
		checkLookups(t, tr, parent, acct, common.Hash{}, common.Hash{}, nil)
	}
}
