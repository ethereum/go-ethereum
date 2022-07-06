package redisdb

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"sync"
)

var (
	errNotFound = errors.New("not found")

	// errSnapshotReleased is returned if callers want to retrieve data from a
	// released snapshot.
	errSnapshotReleased = errors.New("snapshot released")
)

// snapshot wraps a batch of key-value entries deep copied from
// database for implementing the Snapshot interface.
type snapshot struct {
	db   map[string][]byte
	lock sync.RWMutex
}

var _ ethdb.Snapshot = (*snapshot)(nil)

// Has retrieves if a key is present in the snapshot backing by a key-value
// data store.
func (snap *snapshot) Has(key []byte) (bool, error) {
	snap.lock.RLock()
	defer snap.lock.RUnlock()

	if snap.db == nil {
		return false, errSnapshotReleased
	}
	_, ok := snap.db[string(key)]
	return ok, nil
}

// Get retrieves the given key if it's present in the snapshot backing by
// key-value data store.
func (snap *snapshot) Get(key []byte) ([]byte, error) {
	snap.lock.RLock()
	defer snap.lock.RUnlock()

	if snap.db == nil {
		return nil, errSnapshotReleased
	}
	if entry, ok := snap.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, errNotFound
}

// Release releases associated resources. Release should always succeed and can
// be called multiple times without causing error.
func (snap *snapshot) Release() {
	snap.lock.Lock()
	defer snap.lock.Unlock()

	snap.db = nil
}

func newSnapshot(db *Database) (ethdb.Snapshot, error) {
	keys, err := db.client.Keys("*").Result()
	if err != nil {
		return nil, err
	}
	snap := &snapshot{db: make(map[string][]byte)}
	if len(keys) == 0 {
		return snap, nil
	}

	values, err2 := db.client.MGet(keys...).Result()
	if err2 != nil {
		return nil, err2
	}
	for i, key := range keys {
		v := values[i]
		if val, ok := v.([]byte); ok {
			snap.db[key] = val
			continue
		}
		if val, ok := v.(string); ok {
			snap.db[key] = []byte(val)
			continue
		}
	}
	return snap, nil
}
