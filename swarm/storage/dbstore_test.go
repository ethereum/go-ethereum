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
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/log"
)

func initDbStore(t *testing.T) *DbStore {
	dir, err := ioutil.TempDir("", "bzz-storage-test")
	if err != nil {
		t.Fatal(err)
	}
	basekey := sha3.NewKeccak256().Sum([]byte("random"))
	m, err := NewDbStore(dir, MakeHashFunc(defaultHash), defaultDbCapacity, func(k Key) (ret uint8) { return uint8(proximity(basekey[:], k[:])) })
	if err != nil {
		t.Fatal("can't create store:", err)
	}
	return m
}

func testDbStore(indata io.Reader, l int64, branches int64, t *testing.T) {
	t.Skip()
	if indata == nil {
		indata = rand.Reader
	}
	m := initDbStore(t)
	defer m.Close()
	testStore(m, indata, l, branches, t)
}

func TestDbStore128_0x1000000(t *testing.T) {
	testDbStore(nil, 0x1000000, 128, t)
}

func TestDbStore128_10000_(t *testing.T) {
	testDbStore(nil, 10000, 128, t)
}

func TestDbStore128_1000_(t *testing.T) {
	testDbStore(nil, 1000, 128, t)
}

func TestDbStore128_100_(t *testing.T) {
	testDbStore(nil, 100, 128, t)
}

func TestDbStore2_100_(t *testing.T) {
	testDbStore(nil, 100, 2, t)
}

func TestDbStore128_1000000_fixed_(t *testing.T) {
	b := []byte{}
	br := getFixedData(b, 1000000, 254)
	testDbStore(br, 1000000, 2, t)
}

func TestDbStore2_100_fixed_(t *testing.T) {
	b := []byte{}
	br := getFixedData(b, 100, 0)
	testDbStore(br, 100, 2, t)
}

func getFixedData(b []byte, l uint32, p uint8) io.Reader {
	var i byte // it will wrap and still fit byte but not be of much use >255 cos its will only generate more of the same chunks
	var c uint32
	if p == 0 {
		p = 255
	}
	for c = 0; c < l; c++ {
		b = append(b, byte(i))
		if i == p {
			i = 0
		} else {
			i++
		}
	}
	return bytes.NewReader(b)
}

func TestDbStoreNotFound(t *testing.T) {
	m := initDbStore(t)
	defer m.Close()
	_, err := m.Get(ZeroKey)
	if err != notFound {
		t.Errorf("Expected notFound, got %v", err)
	}
}

// func TestDbStoreSyncIterator(t *testing.T) {
// 	m := initDbStore(t)
// 	defer m.Close()
// 	keys := []Key{
// 		Key(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")),
// 		Key(common.Hex2Bytes("4000000000000000000000000000000000000000000000000000000000000000")),
// 		Key(common.Hex2Bytes("5000000000000000000000000000000000000000000000000000000000000000")),
// 		Key(common.Hex2Bytes("3000000000000000000000000000000000000000000000000000000000000000")),
// 		Key(common.Hex2Bytes("2000000000000000000000000000000000000000000000000000000000000000")),
// 		Key(common.Hex2Bytes("1000000000000000000000000000000000000000000000000000000000000000")),
// 	}
// 	for _, key := range keys {
// 		m.Put(NewChunk(key, nil))
// 	}
// 	it, err := m.NewSyncIterator(DbSyncState{
// 		Start: Key(common.Hex2Bytes("1000000000000000000000000000000000000000000000000000000000000000")),
// 		Stop:  Key(common.Hex2Bytes("4000000000000000000000000000000000000000000000000000000000000000")),
// 		First: 2,
// 		Last:  4,
// 	})
// 	if err != nil {
// 		t.Fatalf("unexpected error creating NewSyncIterator")
// 	}

// 	var chunk Key
// 	var res []Key
// 	for {
// 		chunk = it.Next()
// 		if chunk == nil {
// 			break
// 		}
// 		res = append(res, chunk)
// 	}
// 	if len(res) != 1 {
// 		t.Fatalf("Expected 1 chunk, got %v: %v", len(res), res)
// 	}
// 	if !bytes.Equal(res[0][:], keys[3]) {
// 		t.Fatalf("Expected %v chunk, got %v", keys[3], res[0])
// 	}

// 	if err != nil {
// 		t.Fatalf("unexpected error creating NewSyncIterator")
// 	}

// 	it, err = m.NewSyncIterator(DbSyncState{
// 		Start: Key(common.Hex2Bytes("1000000000000000000000000000000000000000000000000000000000000000")),
// 		Stop:  Key(common.Hex2Bytes("5000000000000000000000000000000000000000000000000000000000000000")),
// 		First: 2,
// 		Last:  4,
// 	})

// 	res = nil
// 	for {
// 		chunk = it.Next()
// 		if chunk == nil {
// 			break
// 		}
// 		res = append(res, chunk)
// 	}
// 	if len(res) != 2 {
// 		t.Fatalf("Expected 2 chunk, got %v: %v", len(res), res)
// 	}
// 	if !bytes.Equal(res[0][:], keys[3]) {
// 		t.Fatalf("Expected %v chunk, got %v", keys[3], res[0])
// 	}
// 	if !bytes.Equal(res[1][:], keys[2]) {
// 		t.Fatalf("Expected %v chunk, got %v", keys[2], res[1])
// 	}

// 	if err != nil {
// 		t.Fatalf("unexpected error creating NewSyncIterator")
// 	}

// 	it, _ = m.NewSyncIterator(DbSyncState{
// 		Start: Key(common.Hex2Bytes("1000000000000000000000000000000000000000000000000000000000000000")),
// 		Stop:  Key(common.Hex2Bytes("4000000000000000000000000000000000000000000000000000000000000000")),
// 		First: 2,
// 		Last:  5,
// 	})
// 	res = nil
// 	for {
// 		chunk = it.Next()
// 		if chunk == nil {
// 			break
// 		}
// 		res = append(res, chunk)
// 	}
// 	if len(res) != 2 {
// 		t.Fatalf("Expected 2 chunk, got %v", len(res))
// 	}
// 	if !bytes.Equal(res[0][:], keys[4]) {
// 		t.Fatalf("Expected %v chunk, got %v", keys[4], res[0])
// 	}
// 	if !bytes.Equal(res[1][:], keys[3]) {
// 		t.Fatalf("Expected %v chunk, got %v", keys[3], res[1])
// 	}

// 	it, _ = m.NewSyncIterator(DbSyncState{
// 		Start: Key(common.Hex2Bytes("2000000000000000000000000000000000000000000000000000000000000000")),
// 		Stop:  Key(common.Hex2Bytes("4000000000000000000000000000000000000000000000000000000000000000")),
// 		First: 2,
// 		Last:  5,
// 	})
// 	res = brokenLimitReader(data, size, errAt)
// 	for {
// 		chunk = it.Next()
// 		if chunk == nil {
// 			break
// 		}
// 		res = append(res, chunk)
// 	}
// 	if len(res) != 1 {
// 		t.Fatalf("Expected 1 chunk, got %v", len(res))
// 	}
// 	if !bytes.Equal(res[0][:], keys[3]) {
// 		t.Fatalf("Expected %v chunk, got %v", keys[3], res[0])
// 	}
// }

func TestIterator(t *testing.T) {
	var chunkcount int = 32
	var i int
	var poc uint
	chunkkeys := NewKeyCollection(chunkcount)
	chunkkeys_results := NewKeyCollection(chunkcount)
	chunks := make([]Chunk, chunkcount)

	m := initDbStore(t)
	defer m.Close()

	FakeChunk(getDefaultChunkSize(), chunkcount, chunks)

	for i = 0; i < len(chunks); i++ {
		m.Put(&chunks[i])
		chunkkeys[i] = chunks[i].Key
	}

	//testSplit(m, l, 128, chunkkeys, t)

	for i = 0; i < len(chunkkeys); i++ {
		log.Trace(fmt.Sprintf("Chunk array pos %d/%d: '%v'", i, chunkcount, chunkkeys[i]))
	}

	i = 0
	for poc = 0; poc <= 255; poc++ {
		err := m.SyncIterator(0, uint64(chunkkeys.Len()), uint8(poc), func(k Key, n uint64) bool {
			log.Trace(fmt.Sprintf("Got key %v number %d poc %d", k, n, uint8(poc)))
			chunkkeys_results[n] = k
			i++
			return true
		})
		if err != nil {
			t.Fatalf("Iterator call failed: %v", err)
		}
	}

	for i = 0; i < chunkcount; i++ {
		if bytes.Compare(chunkkeys[i], chunkkeys_results[i]) != 0 {
			t.Fatalf("Chunk put #%d key '%v' does not match iterator's key '%v'", i, chunkkeys[i], chunkkeys_results[i])
		}
	}

}
