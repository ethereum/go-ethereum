package storage

import (
	"bytes"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func initDbStore() (m *DbStore) {
	os.RemoveAll("/tmp/bzz")
	m, err := NewDbStore("/tmp/bzz", MakeHashFunc(defaultHash), defaultDbCapacity, defaultRadius)
	if err != nil {
		panic("no dbStore")
	}
	return
}

func testDbStore(l int64, branches int64, t *testing.T) {
	m := initDbStore()
	defer m.close()
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
	m := initDbStore()
	defer m.close()
	_, err := m.Get(ZeroKey)
	if err != notFound {
		t.Errorf("Expected notFound, got %v", err)
	}
}

func TestDbStoreSyncIterator(t *testing.T) {
	m := initDbStore()
	defer m.close()
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

	it, err = m.NewSyncIterator(DbSyncState{
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

	it, err = m.NewSyncIterator(DbSyncState{
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
