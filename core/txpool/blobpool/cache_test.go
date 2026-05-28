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

package blobpool

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/billy"
)

func newTestStore(t *testing.T) billy.Database {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "store")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	store, err := billy.Open(billy.Options{Path: dir}, newSlotter(testMaxBlobsPerBlock), nil)
	if err != nil {
		t.Fatalf("billy open: %v", err)
	}
	return store
}

// putTx stores a blob transaction in the given store and returns its metadata
// and blobTxForPool.
func putTx(t *testing.T, store billy.Database, nonce, tip uint64) (*blobTxMeta, *blobTxForPool) {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tx := makeTx(nonce, tip, tip, 1, key)
	ptx := newBlobTxForPool(tx)
	id, err := store.Put(encodeForPool(tx))
	if err != nil {
		t.Fatalf("store put: %v", err)
	}
	return newBlobTxMeta(id, ptx.TxSize(), store.Size(id), ptx), ptx
}

func TestCacheRefreshLoadsAndEvicts(t *testing.T) {
	store := newTestStore(t)
	m1, _ := putTx(t, store, 1, 10)
	m2, _ := putTx(t, store, 2, 10)
	m3, _ := putTx(t, store, 3, 10)
	id1, id2, id3 := m1.id, m2.id, m3.id

	c := newCache(store, 8)
	c.reset([]uint64{id1, id2})

	// cache should have id1, id2 entries
	want := []uint64{id1, id2}
	missing, redundant := checkCacheContents(want, c)
	if len(missing) != 0 || len(redundant) != 0 {
		t.Fatalf("cache contents mismatch: want=%v missing=%v redundant=%v",
			want, missing, redundant)
	}

	c.reset([]uint64{id2, id3})

	// cache should have id2, id3 entries
	// since the capacity is biggere than 3, it should
	// also still hold id1.
	want = []uint64{id2, id3}
	missing, redundant = checkCacheContents(want, c)
	if len(missing) != 0 || len(redundant) != 0 {
		t.Fatalf("cache contents mismatch: want=%v missing=%v redundant=%v",
			want, missing, redundant)
	}
}

func TestCacheUpdate(t *testing.T) {
	store := newTestStore(t)

	// three txs with increasing tips
	m1, p1 := putTx(t, store, 1, 10)
	m2, p2 := putTx(t, store, 2, 20)
	m3, p3 := putTx(t, store, 3, 30)
	id1, id2, id3 := m1.id, m2.id, m3.id

	c := newCache(store, 2)
	// case 1: add two transactions should succeed
	c.update([]*addedTx{{id: m1.id, ptx: p1}}, nil)
	c.update([]*addedTx{{id: m2.id, ptx: p2}}, nil)
	want := []uint64{id1, id2}
	missing, redundant := checkCacheContents(want, c)
	if len(missing) != 0 || len(redundant) != 0 {
		t.Fatalf("cache contents mismatch: want=%v missing=%v redundant=%v",
			want, missing, redundant)
	}

	// case 2: adding a tx with higher tip (id3) should evict
	// the least profitable tx (id1)
	c.update([]*addedTx{{id: m3.id, ptx: p3}}, nil)
	want = []uint64{id2, id3}
	missing, redundant = checkCacheContents(want, c)
	if len(missing) != 0 || len(redundant) != 0 {
		t.Fatalf("cache contents mismatch: want=%v missing=%v redundant=%v",
			want, missing, redundant)
	}

	// case 3: adding a tx with lower tip (id1) should be rejected
	c.update([]*addedTx{{id: m1.id, ptx: p1}}, nil)
	want = []uint64{id2, id3}
	missing, redundant = checkCacheContents(want, c)
	if len(missing) != 0 || len(redundant) != 0 {
		t.Fatalf("cache contents mismatch: want=%v missing=%v redundant=%v",
			want, missing, redundant)
	}
}

func TestCacheDrop(t *testing.T) {
	store := newTestStore(t)

	m1, p1 := putTx(t, store, 1, 10)
	id1 := m1.id

	c := newCache(store, 4)
	c.update([]*addedTx{{id: m1.id, ptx: p1}}, nil)
	want := []uint64{m1.id}
	missing, redundant := checkCacheContents(want, c)
	if len(missing) != 0 || len(redundant) != 0 {
		t.Fatalf("cache contents mismatch: want=%v missing=%v redundant=%v",
			want, missing, redundant)
	}

	c.update(nil, []uint64{id1})
	want = []uint64{}
	missing, redundant = checkCacheContents(want, c)
	if len(missing) != 0 || len(redundant) != 0 {
		t.Fatalf("cache contents mismatch: want=%v missing=%v redundant=%v",
			want, missing, redundant)
	}

	// Should be okay to call drop for non-cached id
	c.update(nil, []uint64{99999})
}

// checkCacheContents checks whether the cache has exact content as `want`
func checkCacheContents(want []uint64, c *cache) (missing []uint64, redundant []uint64) {
	for _, wantId := range want {
		if c.get(wantId) == nil {
			missing = append(missing, wantId)
		}
	}
	for _, e := range c.entries {
		if !slices.Contains(want, e.txID) {
			redundant = append(redundant, e.txID)
		}
	}
	return missing, redundant
}
