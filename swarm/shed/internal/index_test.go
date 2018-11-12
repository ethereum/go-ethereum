// Copyright 2018 The go-ethereum Authors
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

package internal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

var retrievalIndexFuncs = IndexFuncs{
	EncodeKey: func(fields IndexItem) (key []byte, err error) {
		return fields.Hash, nil
	},
	DecodeKey: func(key []byte) (e IndexItem, err error) {
		e.Hash = key
		return e, nil
	},
	EncodeValue: func(fields IndexItem) (value []byte, err error) {
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(fields.StoreTimestamp))
		value = append(b, fields.Data...)
		return value, nil
	},
	DecodeValue: func(value []byte) (e IndexItem, err error) {
		e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[:8]))
		e.Data = value[8:]
		return e, nil
	},
}

func TestIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	index, err := db.NewIndex("retrieval", retrievalIndexFuncs)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("put", func(t *testing.T) {
		want := IndexItem{
			Hash:           []byte("put-hash"),
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}

		err = index.Put(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := index.Get(IndexItem{
			Hash: want.Hash,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkIndexItem(t, got, want)

		t.Run("overwrite", func(t *testing.T) {
			want := IndexItem{
				Hash:           []byte("put-hash"),
				Data:           []byte("New DATA"),
				StoreTimestamp: time.Now().UTC().UnixNano(),
			}

			err = index.Put(want)
			if err != nil {
				t.Fatal(err)
			}
			got, err := index.Get(IndexItem{
				Hash: want.Hash,
			})
			if err != nil {
				t.Fatal(err)
			}
			checkIndexItem(t, got, want)
		})
	})

	t.Run("put in batch", func(t *testing.T) {
		want := IndexItem{
			Hash:           []byte("put-in-batch-hash"),
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}

		batch := new(leveldb.Batch)
		index.PutInBatch(batch, want)
		db.WriteBatch(batch)
		if err != nil {
			t.Fatal(err)
		}
		got, err := index.Get(IndexItem{
			Hash: want.Hash,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkIndexItem(t, got, want)

		t.Run("overwrite", func(t *testing.T) {
			want := IndexItem{
				Hash:           []byte("put-in-batch-hash"),
				Data:           []byte("New DATA"),
				StoreTimestamp: time.Now().UTC().UnixNano(),
			}

			batch := new(leveldb.Batch)
			index.PutInBatch(batch, want)
			db.WriteBatch(batch)
			if err != nil {
				t.Fatal(err)
			}
			got, err := index.Get(IndexItem{
				Hash: want.Hash,
			})
			if err != nil {
				t.Fatal(err)
			}
			checkIndexItem(t, got, want)
		})
	})

	t.Run("delete", func(t *testing.T) {
		want := IndexItem{
			Hash:           []byte("delete-hash"),
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}

		err = index.Put(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := index.Get(IndexItem{
			Hash: want.Hash,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkIndexItem(t, got, want)

		err = index.Delete(IndexItem{
			Hash: want.Hash,
		})
		if err != nil {
			t.Fatal(err)
		}

		got, err = index.Get(IndexItem{
			Hash: want.Hash,
		})
		if err != leveldb.ErrNotFound {
			t.Fatalf("got error %v, want %v", err, leveldb.ErrNotFound)
		}
	})

	t.Run("delete in batch", func(t *testing.T) {
		want := IndexItem{
			Hash:           []byte("delete-in-batch-hash"),
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}

		err = index.Put(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := index.Get(IndexItem{
			Hash: want.Hash,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkIndexItem(t, got, want)

		batch := new(leveldb.Batch)
		index.DeleteInBatch(batch, IndexItem{
			Hash: want.Hash,
		})
		err = db.WriteBatch(batch)
		if err != nil {
			t.Fatal(err)
		}

		got, err = index.Get(IndexItem{
			Hash: want.Hash,
		})
		if err != leveldb.ErrNotFound {
			t.Fatalf("got error %v, want %v", err, leveldb.ErrNotFound)
		}
	})
}

func TestIndex_iterate(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	index, err := db.NewIndex("retrieval", retrievalIndexFuncs)
	if err != nil {
		t.Fatal(err)
	}

	items := []IndexItem{
		{
			Hash: []byte("iterate-hash-01"),
			Data: []byte("data80"),
		},
		{
			Hash: []byte("iterate-hash-03"),
			Data: []byte("data22"),
		},
		{
			Hash: []byte("iterate-hash-05"),
			Data: []byte("data41"),
		},
		{
			Hash: []byte("iterate-hash-02"),
			Data: []byte("data84"),
		},
		{
			Hash: []byte("iterate-hash-06"),
			Data: []byte("data1"),
		},
	}
	batch := new(leveldb.Batch)
	for _, i := range items {
		index.PutInBatch(batch, i)
	}
	err = db.WriteBatch(batch)
	if err != nil {
		t.Fatal(err)
	}
	item04 := IndexItem{
		Hash: []byte("iterate-hash-04"),
		Data: []byte("data0"),
	}
	err = index.Put(item04)
	if err != nil {
		t.Fatal(err)
	}
	items = append(items, item04)

	sort.SliceStable(items, func(i, j int) bool {
		return bytes.Compare(items[i].Hash, items[j].Hash) < 0
	})

	t.Run("all", func(t *testing.T) {
		var i int
		err := index.IterateAll(func(item IndexItem) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkIndexItem(t, item, want)
			i++
			return false, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("from", func(t *testing.T) {
		startIndex := 2
		i := startIndex
		err := index.IterateFrom(items[startIndex], func(item IndexItem) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkIndexItem(t, item, want)
			i++
			return false, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("stop", func(t *testing.T) {
		var i int
		stopIndex := 3
		var count int
		err := index.IterateAll(func(item IndexItem) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkIndexItem(t, item, want)
			count++
			if i == stopIndex {
				return true, nil
			}
			i++
			return false, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		wantItemsCount := stopIndex + 1
		if count != wantItemsCount {
			t.Errorf("got %v items, expected %v", count, wantItemsCount)
		}
	})

	t.Run("no overflow", func(t *testing.T) {
		secondIndex, err := db.NewIndex("second-index", retrievalIndexFuncs)
		if err != nil {
			t.Fatal(err)
		}

		secondIndexItem := IndexItem{
			Hash: []byte("iterate-hash-10"),
			Data: []byte("data-second"),
		}
		err = secondIndex.Put(secondIndexItem)
		if err != nil {
			t.Fatal(err)
		}

		var i int
		err = index.IterateAll(func(item IndexItem) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkIndexItem(t, item, want)
			i++
			return false, nil
		})
		if err != nil {
			t.Fatal(err)
		}

		i = 0
		err = secondIndex.IterateAll(func(item IndexItem) (stop bool, err error) {
			if i > 1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			checkIndexItem(t, item, secondIndexItem)
			i++
			return false, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func checkIndexItem(t *testing.T, got, want IndexItem) {
	t.Helper()

	if !bytes.Equal(got.Hash, want.Hash) {
		t.Errorf("got hash %q, expected %q", string(got.Hash), string(want.Hash))
	}
	if !bytes.Equal(got.Data, want.Data) {
		t.Errorf("got data %q, expected %q", string(got.Data), string(want.Data))
	}
	if got.StoreTimestamp != want.StoreTimestamp {
		t.Errorf("got store timestamp %v, expected %v", got.StoreTimestamp, want.StoreTimestamp)
	}
	if got.AccessTimestamp != want.AccessTimestamp {
		t.Errorf("got access timestamp %v, expected %v", got.AccessTimestamp, want.AccessTimestamp)
	}
}
