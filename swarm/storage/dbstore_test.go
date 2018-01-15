// Copyright 2016 The go-ethereum Authors
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

package storage

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
	"github.com/ethereum/go-ethereum/swarm/storage/mock/mem"
)

func initDbStore(t *testing.T) *DbStore {
	dir, err := ioutil.TempDir("", "bzz-storage-test")
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewDbStore(dir, MakeHashFunc(SHA3Hash), defaultDbCapacity, defaultRadius)
	if err != nil {
		t.Fatal("can't create store:", err)
	}
	return m
}

func testDbStore(l int64, branches int64, t *testing.T) {
	m := initDbStore(t)
	defer m.Close()
	testStore(m, l, branches, t)
}

func TestDbStore128_0x1000000(t *testing.T) {
	testDbStore(0x1000000, 128, t)
}

func TestDbStore128_10000_(t *testing.T) {
	testDbStore(10000, 128, t)
}

func TestDbStore128_1000_(t *testing.T) {
	testDbStore(1000, 128, t)
}

func TestDbStore128_100_(t *testing.T) {
	testDbStore(100, 128, t)
}

func TestDbStore2_100_(t *testing.T) {
	testDbStore(100, 2, t)
}

func TestDbStoreNotFound(t *testing.T) {
	m := initDbStore(t)
	defer m.Close()
	_, err := m.Get(ZeroKey)
	if err != notFound {
		t.Errorf("Expected notFound, got %v", err)
	}
}

func TestDbStoreSyncIterator(t *testing.T) {
	m := initDbStore(t)
	defer m.Close()
	keys := []Key{
		Key(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")),
		Key(common.Hex2Bytes("4000000000000000000000000000000000000000000000000000000000000000")),
		Key(common.Hex2Bytes("5000000000000000000000000000000000000000000000000000000000000000")),
		Key(common.Hex2Bytes("3000000000000000000000000000000000000000000000000000000000000000")),
		Key(common.Hex2Bytes("2000000000000000000000000000000000000000000000000000000000000000")),
		Key(common.Hex2Bytes("1000000000000000000000000000000000000000000000000000000000000000")),
	}
	for _, key := range keys {
		m.Put(NewChunk(key, nil))
	}
	it, err := m.NewSyncIterator(DbSyncState{
		Start: Key(common.Hex2Bytes("1000000000000000000000000000000000000000000000000000000000000000")),
		Stop:  Key(common.Hex2Bytes("4000000000000000000000000000000000000000000000000000000000000000")),
		First: 2,
		Last:  4,
	})
	if err != nil {
		t.Fatalf("unexpected error creating NewSyncIterator")
	}

	var chunk Key
	var res []Key
	for {
		chunk = it.Next()
		if chunk == nil {
			break
		}
		res = append(res, chunk)
	}
	if len(res) != 1 {
		t.Fatalf("Expected 1 chunk, got %v: %v", len(res), res)
	}
	if !bytes.Equal(res[0][:], keys[3]) {
		t.Fatalf("Expected %v chunk, got %v", keys[3], res[0])
	}

	if err != nil {
		t.Fatalf("unexpected error creating NewSyncIterator")
	}

	it, err = m.NewSyncIterator(DbSyncState{
		Start: Key(common.Hex2Bytes("1000000000000000000000000000000000000000000000000000000000000000")),
		Stop:  Key(common.Hex2Bytes("5000000000000000000000000000000000000000000000000000000000000000")),
		First: 2,
		Last:  4,
	})

	res = nil
	for {
		chunk = it.Next()
		if chunk == nil {
			break
		}
		res = append(res, chunk)
	}
	if len(res) != 2 {
		t.Fatalf("Expected 2 chunk, got %v: %v", len(res), res)
	}
	if !bytes.Equal(res[0][:], keys[3]) {
		t.Fatalf("Expected %v chunk, got %v", keys[3], res[0])
	}
	if !bytes.Equal(res[1][:], keys[2]) {
		t.Fatalf("Expected %v chunk, got %v", keys[2], res[1])
	}

	if err != nil {
		t.Fatalf("unexpected error creating NewSyncIterator")
	}

	it, _ = m.NewSyncIterator(DbSyncState{
		Start: Key(common.Hex2Bytes("1000000000000000000000000000000000000000000000000000000000000000")),
		Stop:  Key(common.Hex2Bytes("4000000000000000000000000000000000000000000000000000000000000000")),
		First: 2,
		Last:  5,
	})
	res = nil
	for {
		chunk = it.Next()
		if chunk == nil {
			break
		}
		res = append(res, chunk)
	}
	if len(res) != 2 {
		t.Fatalf("Expected 2 chunk, got %v", len(res))
	}
	if !bytes.Equal(res[0][:], keys[4]) {
		t.Fatalf("Expected %v chunk, got %v", keys[4], res[0])
	}
	if !bytes.Equal(res[1][:], keys[3]) {
		t.Fatalf("Expected %v chunk, got %v", keys[3], res[1])
	}

	it, _ = m.NewSyncIterator(DbSyncState{
		Start: Key(common.Hex2Bytes("2000000000000000000000000000000000000000000000000000000000000000")),
		Stop:  Key(common.Hex2Bytes("4000000000000000000000000000000000000000000000000000000000000000")),
		First: 2,
		Last:  5,
	})
	res = nil
	for {
		chunk = it.Next()
		if chunk == nil {
			break
		}
		res = append(res, chunk)
	}
	if len(res) != 1 {
		t.Fatalf("Expected 1 chunk, got %v", len(res))
	}
	if !bytes.Equal(res[0][:], keys[3]) {
		t.Fatalf("Expected %v chunk, got %v", keys[3], res[0])
	}
}

func initMockDbStore(t *testing.T, mockStore mock.NodeStorer) *DbStore {
	dir, err := ioutil.TempDir("", "bzz-storage-test-mock")
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewMockDbStore(dir, MakeHashFunc(SHA3Hash), defaultDbCapacity, defaultRadius, mockStore)
	if err != nil {
		t.Fatal("can't create store:", err)
	}
	return m
}

func testMockDbStore(l int64, branches int64, t *testing.T) {
	globalStore := mem.NewGlobalStore()
	addr := common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	mockStore := globalStore.NewNodeStore(addr)
	m := initMockDbStore(t, mockStore)
	defer m.Close()

	key := Key(common.Hex2Bytes("fed1911825fc6a02ebfd19ab218a20455d8d7d275f8bf4d8244eb04364fae6f7"))
	data := common.Hex2BytesFixed(strings.Repeat("1234567890abcdf", 10), 4096)

	m.Put(&Chunk{
		Key:   key,
		SData: data,
	})

	_, err := globalStore.Get(addr, key)
	if err != nil {
		t.Errorf("unexpected error getting the data from global mock store: %v", err)
	}

	if !globalStore.HasKey(addr, key) {
		t.Error("key not found in global store")
	}

	testStore(m, l, branches, t)

}

func TestMockDbStore128_0x1000000(t *testing.T) {
	testMockDbStore(0x1000000, 128, t)
}

func TestMockDbStore128_10000_(t *testing.T) {
	testMockDbStore(10000, 128, t)
}

func TestMockDbStore128_1000_(t *testing.T) {
	testMockDbStore(1000, 128, t)
}

func TestMockDbStore128_100_(t *testing.T) {
	testMockDbStore(100, 128, t)
}

func TestMockDbStore2_100_(t *testing.T) {
	testMockDbStore(100, 2, t)
}

func TestMockDbStoreNotFound(t *testing.T) {
	globalStore := mem.NewGlobalStore()
	mockStore := globalStore.NewNodeStore(common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"))
	m := initMockDbStore(t, mockStore)
	defer m.Close()
	_, err := m.Get(ZeroKey)
	if err != notFound {
		t.Errorf("Expected notFound, got %v", err)
	}
}
