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

package shed

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

// Index functions for the index that is used in tests in this file.
var retrievalIndexFuncs = IndexFuncs{
	EncodeKey: func(fields Item) (key []byte, err error) {
		return fields.Address, nil
	},
	DecodeKey: func(key []byte) (e Item, err error) {
		e.Address = key
		return e, nil
	},
	EncodeValue: func(fields Item) (value []byte, err error) {
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(fields.StoreTimestamp))
		value = append(b, fields.Data...)
		return value, nil
	},
	DecodeValue: func(keyItem Item, value []byte) (e Item, err error) {
		e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[:8]))
		e.Data = value[8:]
		return e, nil
	},
}

// TestIndex validates put, get, has and delete functions of the Index implementation.
func TestIndex(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	index, err := db.NewIndex("retrieval", retrievalIndexFuncs)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("put", func(t *testing.T) {
		want := Item{
			Address:        []byte("put-hash"),
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}

		err := index.Put(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := index.Get(Item{
			Address: want.Address,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkItem(t, got, want)

		t.Run("overwrite", func(t *testing.T) {
			want := Item{
				Address:        []byte("put-hash"),
				Data:           []byte("New DATA"),
				StoreTimestamp: time.Now().UTC().UnixNano(),
			}

			err = index.Put(want)
			if err != nil {
				t.Fatal(err)
			}
			got, err := index.Get(Item{
				Address: want.Address,
			})
			if err != nil {
				t.Fatal(err)
			}
			checkItem(t, got, want)
		})
	})

	t.Run("put in batch", func(t *testing.T) {
		want := Item{
			Address:        []byte("put-in-batch-hash"),
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}

		batch := new(leveldb.Batch)
		index.PutInBatch(batch, want)
		err := db.WriteBatch(batch)
		if err != nil {
			t.Fatal(err)
		}
		got, err := index.Get(Item{
			Address: want.Address,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkItem(t, got, want)

		t.Run("overwrite", func(t *testing.T) {
			want := Item{
				Address:        []byte("put-in-batch-hash"),
				Data:           []byte("New DATA"),
				StoreTimestamp: time.Now().UTC().UnixNano(),
			}

			batch := new(leveldb.Batch)
			index.PutInBatch(batch, want)
			db.WriteBatch(batch)
			if err != nil {
				t.Fatal(err)
			}
			got, err := index.Get(Item{
				Address: want.Address,
			})
			if err != nil {
				t.Fatal(err)
			}
			checkItem(t, got, want)
		})
	})

	t.Run("put in batch twice", func(t *testing.T) {
		// ensure that the last item of items with the same db keys
		// is actually saved
		batch := new(leveldb.Batch)
		address := []byte("put-in-batch-twice-hash")

		// put the first item
		index.PutInBatch(batch, Item{
			Address:        address,
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		})

		want := Item{
			Address:        address,
			Data:           []byte("New DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}
		// then put the item that will produce the same key
		// but different value in the database
		index.PutInBatch(batch, want)
		db.WriteBatch(batch)
		if err != nil {
			t.Fatal(err)
		}
		got, err := index.Get(Item{
			Address: address,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkItem(t, got, want)
	})

	t.Run("has", func(t *testing.T) {
		want := Item{
			Address:        []byte("has-hash"),
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}

		dontWant := Item{
			Address:        []byte("do-not-has-hash"),
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}

		err := index.Put(want)
		if err != nil {
			t.Fatal(err)
		}

		has, err := index.Has(want)
		if err != nil {
			t.Fatal(err)
		}
		if !has {
			t.Error("item is not found")
		}

		has, err = index.Has(dontWant)
		if err != nil {
			t.Fatal(err)
		}
		if has {
			t.Error("unwanted item is found")
		}
	})

	t.Run("delete", func(t *testing.T) {
		want := Item{
			Address:        []byte("delete-hash"),
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}

		err := index.Put(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := index.Get(Item{
			Address: want.Address,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkItem(t, got, want)

		err = index.Delete(Item{
			Address: want.Address,
		})
		if err != nil {
			t.Fatal(err)
		}

		wantErr := leveldb.ErrNotFound
		got, err = index.Get(Item{
			Address: want.Address,
		})
		if err != wantErr {
			t.Fatalf("got error %v, want %v", err, wantErr)
		}
	})

	t.Run("delete in batch", func(t *testing.T) {
		want := Item{
			Address:        []byte("delete-in-batch-hash"),
			Data:           []byte("DATA"),
			StoreTimestamp: time.Now().UTC().UnixNano(),
		}

		err := index.Put(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := index.Get(Item{
			Address: want.Address,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkItem(t, got, want)

		batch := new(leveldb.Batch)
		index.DeleteInBatch(batch, Item{
			Address: want.Address,
		})
		err = db.WriteBatch(batch)
		if err != nil {
			t.Fatal(err)
		}

		wantErr := leveldb.ErrNotFound
		got, err = index.Get(Item{
			Address: want.Address,
		})
		if err != wantErr {
			t.Fatalf("got error %v, want %v", err, wantErr)
		}
	})
}

// TestIndex_Iterate validates index Iterate
// functions for correctness.
func TestIndex_Iterate(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	index, err := db.NewIndex("retrieval", retrievalIndexFuncs)
	if err != nil {
		t.Fatal(err)
	}

	items := []Item{
		{
			Address: []byte("iterate-hash-01"),
			Data:    []byte("data80"),
		},
		{
			Address: []byte("iterate-hash-03"),
			Data:    []byte("data22"),
		},
		{
			Address: []byte("iterate-hash-05"),
			Data:    []byte("data41"),
		},
		{
			Address: []byte("iterate-hash-02"),
			Data:    []byte("data84"),
		},
		{
			Address: []byte("iterate-hash-06"),
			Data:    []byte("data1"),
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
	item04 := Item{
		Address: []byte("iterate-hash-04"),
		Data:    []byte("data0"),
	}
	err = index.Put(item04)
	if err != nil {
		t.Fatal(err)
	}
	items = append(items, item04)

	sort.SliceStable(items, func(i, j int) bool {
		return bytes.Compare(items[i].Address, items[j].Address) < 0
	})

	t.Run("all", func(t *testing.T) {
		var i int
		err := index.Iterate(func(item Item) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkItem(t, item, want)
			i++
			return false, nil
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("start from", func(t *testing.T) {
		startIndex := 2
		i := startIndex
		err := index.Iterate(func(item Item) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkItem(t, item, want)
			i++
			return false, nil
		}, &IterateOptions{
			StartFrom: &items[startIndex],
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("skip start from", func(t *testing.T) {
		startIndex := 2
		i := startIndex + 1
		err := index.Iterate(func(item Item) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkItem(t, item, want)
			i++
			return false, nil
		}, &IterateOptions{
			StartFrom:         &items[startIndex],
			SkipStartFromItem: true,
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("stop", func(t *testing.T) {
		var i int
		stopIndex := 3
		var count int
		err := index.Iterate(func(item Item) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkItem(t, item, want)
			count++
			if i == stopIndex {
				return true, nil
			}
			i++
			return false, nil
		}, nil)
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

		secondItem := Item{
			Address: []byte("iterate-hash-10"),
			Data:    []byte("data-second"),
		}
		err = secondIndex.Put(secondItem)
		if err != nil {
			t.Fatal(err)
		}

		var i int
		err = index.Iterate(func(item Item) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkItem(t, item, want)
			i++
			return false, nil
		}, nil)
		if err != nil {
			t.Fatal(err)
		}

		i = 0
		err = secondIndex.Iterate(func(item Item) (stop bool, err error) {
			if i > 1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			checkItem(t, item, secondItem)
			i++
			return false, nil
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
	})
}

// TestIndex_Iterate_withPrefix validates index Iterate
// function for correctness.
func TestIndex_Iterate_withPrefix(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	index, err := db.NewIndex("retrieval", retrievalIndexFuncs)
	if err != nil {
		t.Fatal(err)
	}

	allItems := []Item{
		{Address: []byte("want-hash-00"), Data: []byte("data80")},
		{Address: []byte("skip-hash-01"), Data: []byte("data81")},
		{Address: []byte("skip-hash-02"), Data: []byte("data82")},
		{Address: []byte("skip-hash-03"), Data: []byte("data83")},
		{Address: []byte("want-hash-04"), Data: []byte("data84")},
		{Address: []byte("want-hash-05"), Data: []byte("data85")},
		{Address: []byte("want-hash-06"), Data: []byte("data86")},
		{Address: []byte("want-hash-07"), Data: []byte("data87")},
		{Address: []byte("want-hash-08"), Data: []byte("data88")},
		{Address: []byte("want-hash-09"), Data: []byte("data89")},
		{Address: []byte("skip-hash-10"), Data: []byte("data90")},
	}
	batch := new(leveldb.Batch)
	for _, i := range allItems {
		index.PutInBatch(batch, i)
	}
	err = db.WriteBatch(batch)
	if err != nil {
		t.Fatal(err)
	}

	prefix := []byte("want")

	items := make([]Item, 0)
	for _, item := range allItems {
		if bytes.HasPrefix(item.Address, prefix) {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return bytes.Compare(items[i].Address, items[j].Address) < 0
	})

	t.Run("with prefix", func(t *testing.T) {
		var i int
		err := index.Iterate(func(item Item) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkItem(t, item, want)
			i++
			return false, nil
		}, &IterateOptions{
			Prefix: prefix,
		})
		if err != nil {
			t.Fatal(err)
		}
		if i != len(items) {
			t.Errorf("got %v items, want %v", i, len(items))
		}
	})

	t.Run("with prefix and start from", func(t *testing.T) {
		startIndex := 2
		var count int
		i := startIndex
		err := index.Iterate(func(item Item) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkItem(t, item, want)
			i++
			count++
			return false, nil
		}, &IterateOptions{
			StartFrom: &items[startIndex],
			Prefix:    prefix,
		})
		if err != nil {
			t.Fatal(err)
		}
		wantCount := len(items) - startIndex
		if count != wantCount {
			t.Errorf("got %v items, want %v", count, wantCount)
		}
	})

	t.Run("with prefix and skip start from", func(t *testing.T) {
		startIndex := 2
		var count int
		i := startIndex + 1
		err := index.Iterate(func(item Item) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkItem(t, item, want)
			i++
			count++
			return false, nil
		}, &IterateOptions{
			StartFrom:         &items[startIndex],
			SkipStartFromItem: true,
			Prefix:            prefix,
		})
		if err != nil {
			t.Fatal(err)
		}
		wantCount := len(items) - startIndex - 1
		if count != wantCount {
			t.Errorf("got %v items, want %v", count, wantCount)
		}
	})

	t.Run("stop", func(t *testing.T) {
		var i int
		stopIndex := 3
		var count int
		err := index.Iterate(func(item Item) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkItem(t, item, want)
			count++
			if i == stopIndex {
				return true, nil
			}
			i++
			return false, nil
		}, &IterateOptions{
			Prefix: prefix,
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

		secondItem := Item{
			Address: []byte("iterate-hash-10"),
			Data:    []byte("data-second"),
		}
		err = secondIndex.Put(secondItem)
		if err != nil {
			t.Fatal(err)
		}

		var i int
		err = index.Iterate(func(item Item) (stop bool, err error) {
			if i > len(items)-1 {
				return true, fmt.Errorf("got unexpected index item: %#v", item)
			}
			want := items[i]
			checkItem(t, item, want)
			i++
			return false, nil
		}, &IterateOptions{
			Prefix: prefix,
		})
		if err != nil {
			t.Fatal(err)
		}
		if i != len(items) {
			t.Errorf("got %v items, want %v", i, len(items))
		}
	})
}

// TestIndex_count tests if Index.Count and Index.CountFrom
// returns the correct number of items.
func TestIndex_count(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	index, err := db.NewIndex("retrieval", retrievalIndexFuncs)
	if err != nil {
		t.Fatal(err)
	}

	items := []Item{
		{
			Address: []byte("iterate-hash-01"),
			Data:    []byte("data80"),
		},
		{
			Address: []byte("iterate-hash-02"),
			Data:    []byte("data84"),
		},
		{
			Address: []byte("iterate-hash-03"),
			Data:    []byte("data22"),
		},
		{
			Address: []byte("iterate-hash-04"),
			Data:    []byte("data41"),
		},
		{
			Address: []byte("iterate-hash-05"),
			Data:    []byte("data1"),
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

	t.Run("Count", func(t *testing.T) {
		got, err := index.Count()
		if err != nil {
			t.Fatal(err)
		}

		want := len(items)
		if got != want {
			t.Errorf("got %v items count, want %v", got, want)
		}
	})

	t.Run("CountFrom", func(t *testing.T) {
		got, err := index.CountFrom(Item{
			Address: items[1].Address,
		})
		if err != nil {
			t.Fatal(err)
		}

		want := len(items) - 1
		if got != want {
			t.Errorf("got %v items count, want %v", got, want)
		}
	})

	// update the index with another item
	t.Run("add item", func(t *testing.T) {
		item04 := Item{
			Address: []byte("iterate-hash-06"),
			Data:    []byte("data0"),
		}
		err = index.Put(item04)
		if err != nil {
			t.Fatal(err)
		}

		count := len(items) + 1

		t.Run("Count", func(t *testing.T) {
			got, err := index.Count()
			if err != nil {
				t.Fatal(err)
			}

			want := count
			if got != want {
				t.Errorf("got %v items count, want %v", got, want)
			}
		})

		t.Run("CountFrom", func(t *testing.T) {
			got, err := index.CountFrom(Item{
				Address: items[1].Address,
			})
			if err != nil {
				t.Fatal(err)
			}

			want := count - 1
			if got != want {
				t.Errorf("got %v items count, want %v", got, want)
			}
		})
	})

	// delete some items
	t.Run("delete items", func(t *testing.T) {
		deleteCount := 3

		for _, item := range items[:deleteCount] {
			err := index.Delete(item)
			if err != nil {
				t.Fatal(err)
			}
		}

		count := len(items) + 1 - deleteCount

		t.Run("Count", func(t *testing.T) {
			got, err := index.Count()
			if err != nil {
				t.Fatal(err)
			}

			want := count
			if got != want {
				t.Errorf("got %v items count, want %v", got, want)
			}
		})

		t.Run("CountFrom", func(t *testing.T) {
			got, err := index.CountFrom(Item{
				Address: items[deleteCount+1].Address,
			})
			if err != nil {
				t.Fatal(err)
			}

			want := count - 1
			if got != want {
				t.Errorf("got %v items count, want %v", got, want)
			}
		})
	})
}

// checkItem is a test helper function that compares if two Index items are the same.
func checkItem(t *testing.T, got, want Item) {
	t.Helper()

	if !bytes.Equal(got.Address, want.Address) {
		t.Errorf("got hash %q, expected %q", string(got.Address), string(want.Address))
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

// TestIndex_firstAndLast validates that index First and Last methods
// are returning expected results based on the provided prefix.
func TestIndex_firstAndLast(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	index, err := db.NewIndex("retrieval", retrievalIndexFuncs)
	if err != nil {
		t.Fatal(err)
	}

	addrs := [][]byte{
		{0, 0, 0, 0, 0},
		{0, 1},
		{0, 1, 0, 0, 0},
		{0, 1, 0, 0, 1},
		{0, 1, 0, 0, 2},
		{0, 2, 0, 0, 1},
		{0, 4, 0, 0, 0},
		{0, 10, 0, 0, 10},
		{0, 10, 0, 0, 11},
		{0, 10, 0, 0, 20},
		{1, 32, 255, 0, 1},
		{1, 32, 255, 0, 2},
		{1, 32, 255, 0, 3},
		{255, 255, 255, 255, 32},
		{255, 255, 255, 255, 64},
		{255, 255, 255, 255, 255},
	}

	// ensure that the addresses are sorted for
	// validation of nil prefix
	sort.Slice(addrs, func(i, j int) (less bool) {
		return bytes.Compare(addrs[i], addrs[j]) == -1
	})

	batch := new(leveldb.Batch)
	for _, addr := range addrs {
		index.PutInBatch(batch, Item{
			Address: addr,
		})
	}
	err = db.WriteBatch(batch)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		prefix []byte
		first  []byte
		last   []byte
		err    error
	}{
		{
			prefix: nil,
			first:  addrs[0],
			last:   addrs[len(addrs)-1],
		},
		{
			prefix: []byte{0, 0},
			first:  []byte{0, 0, 0, 0, 0},
			last:   []byte{0, 0, 0, 0, 0},
		},
		{
			prefix: []byte{0},
			first:  []byte{0, 0, 0, 0, 0},
			last:   []byte{0, 10, 0, 0, 20},
		},
		{
			prefix: []byte{0, 1},
			first:  []byte{0, 1},
			last:   []byte{0, 1, 0, 0, 2},
		},
		{
			prefix: []byte{0, 10},
			first:  []byte{0, 10, 0, 0, 10},
			last:   []byte{0, 10, 0, 0, 20},
		},
		{
			prefix: []byte{1, 32, 255},
			first:  []byte{1, 32, 255, 0, 1},
			last:   []byte{1, 32, 255, 0, 3},
		},
		{
			prefix: []byte{255},
			first:  []byte{255, 255, 255, 255, 32},
			last:   []byte{255, 255, 255, 255, 255},
		},
		{
			prefix: []byte{255, 255, 255, 255, 255},
			first:  []byte{255, 255, 255, 255, 255},
			last:   []byte{255, 255, 255, 255, 255},
		},
		{
			prefix: []byte{0, 3},
			err:    leveldb.ErrNotFound,
		},
		{
			prefix: []byte{222},
			err:    leveldb.ErrNotFound,
		},
	} {
		got, err := index.Last(tc.prefix)
		if tc.err != err {
			t.Errorf("got error %v for Last with prefix %v, want %v", err, tc.prefix, tc.err)
		} else {
			if !bytes.Equal(got.Address, tc.last) {
				t.Errorf("got %v for Last with prefix %v, want %v", got.Address, tc.prefix, tc.last)
			}
		}

		got, err = index.First(tc.prefix)
		if tc.err != err {
			t.Errorf("got error %v for First with prefix %v, want %v", err, tc.prefix, tc.err)
		} else {
			if !bytes.Equal(got.Address, tc.first) {
				t.Errorf("got %v for First with prefix %v, want %v", got.Address, tc.prefix, tc.first)
			}
		}
	}
}

// TestIncByteSlice validates returned values of incByteSlice function.
func TestIncByteSlice(t *testing.T) {
	for _, tc := range []struct {
		b    []byte
		want []byte
	}{
		{b: nil, want: nil},
		{b: []byte{}, want: nil},
		{b: []byte{0}, want: []byte{1}},
		{b: []byte{42}, want: []byte{43}},
		{b: []byte{255}, want: nil},
		{b: []byte{0, 0}, want: []byte{0, 1}},
		{b: []byte{1, 0}, want: []byte{1, 1}},
		{b: []byte{1, 255}, want: []byte{2, 0}},
		{b: []byte{255, 255}, want: nil},
		{b: []byte{32, 0, 255}, want: []byte{32, 1, 0}},
	} {
		got := incByteSlice(tc.b)
		if !bytes.Equal(got, tc.want) {
			t.Errorf("got %v, want %v", got, tc.want)
		}
	}
}
