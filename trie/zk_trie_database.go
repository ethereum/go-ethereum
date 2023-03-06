package trie

import (
	"math/big"

	"github.com/syndtr/goleveldb/leveldb"

	zktrie "github.com/scroll-tech/zktrie/trie"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethdb"
)

// ZktrieDatabase Database adaptor imple zktrie.ZktrieDatbase
type ZktrieDatabase struct {
	db     *Database
	prefix []byte
}

func NewZktrieDatabase(diskdb ethdb.KeyValueStore) *ZktrieDatabase {
	return &ZktrieDatabase{db: NewDatabase(diskdb), prefix: []byte{}}
}

// adhoc wrapper...
func NewZktrieDatabaseFromTriedb(db *Database) *ZktrieDatabase {
	db.Zktrie = true
	return &ZktrieDatabase{db: db, prefix: []byte{}}
}

// Put saves a key:value into the Storage
func (l *ZktrieDatabase) Put(k, v []byte) error {
	l.db.lock.Lock()
	l.db.rawDirties.Put(Concat(l.prefix, k[:]), v)
	l.db.lock.Unlock()
	return nil
}

// Get retrieves a value from a key in the Storage
func (l *ZktrieDatabase) Get(key []byte) ([]byte, error) {
	concatKey := Concat(l.prefix, key[:])
	l.db.lock.RLock()
	value, ok := l.db.rawDirties.Get(concatKey)
	l.db.lock.RUnlock()
	if ok {
		return value, nil
	}

	if l.db.cleans != nil {
		if enc := l.db.cleans.Get(nil, concatKey); enc != nil {
			memcacheCleanHitMeter.Mark(1)
			memcacheCleanReadMeter.Mark(int64(len(enc)))
			return enc, nil
		}
	}

	v, err := l.db.diskdb.Get(concatKey)
	if err == leveldb.ErrNotFound {
		return nil, zktrie.ErrKeyNotFound
	}
	if l.db.cleans != nil {
		l.db.cleans.Set(concatKey[:], v)
		memcacheCleanMissMeter.Mark(1)
		memcacheCleanWriteMeter.Mark(int64(len(v)))
	}
	return v, err
}

func (l *ZktrieDatabase) UpdatePreimage(preimage []byte, hashField *big.Int) {
	db := l.db
	if db.preimages != nil { // Ugly direct check but avoids the below write lock
		// we must copy the input key
		db.preimages.insertPreimage(map[common.Hash][]byte{common.BytesToHash(hashField.Bytes()): common.CopyBytes(preimage)})
	}
}

// Iterate implements the method Iterate of the interface Storage
func (l *ZktrieDatabase) Iterate(f func([]byte, []byte) (bool, error)) error {
	iter := l.db.diskdb.NewIterator(l.prefix, nil)
	defer iter.Release()
	for iter.Next() {
		localKey := iter.Key()[len(l.prefix):]
		if cont, err := f(localKey, iter.Value()); err != nil {
			return err
		} else if !cont {
			break
		}
	}
	iter.Release()
	return iter.Error()
}

// Close implements the method Close of the interface Storage
func (l *ZktrieDatabase) Close() {
	// FIXME: is this correct?
	if err := l.db.diskdb.Close(); err != nil {
		panic(err)
	}
}

// List implements the method List of the interface Storage
func (l *ZktrieDatabase) List(limit int) ([]KV, error) {
	ret := []KV{}
	err := l.Iterate(func(key []byte, value []byte) (bool, error) {
		ret = append(ret, KV{K: Clone(key), V: Clone(value)})
		if len(ret) == limit {
			return false, nil
		}
		return true, nil
	})
	return ret, err
}
