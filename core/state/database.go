// Copyright 2017 The go-ethereum Authors
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

package state

import (
	"container/heap"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	lru "github.com/hashicorp/golang-lru"
)

// Trie cache generation limit after which to evic trie nodes from memory.
var MaxTrieCacheGen = uint16(120)

var StateNotInCache = errors.New("State trie not in cache")

const (
	// Number of codehash->size associations to keep.
	codeSizeCacheSize = 100000
)

// Database wraps access to tries and contract code.
type Database interface {
	// Accessing tries:
	// OpenTrie opens the main account trie.
	// OpenStorageTrie opens the storage trie of an account.
	OpenTrie(root common.Hash) (Trie, error)
	OpenStorageTrie(addrHash, root common.Hash) (Trie, error)
	// Accessing contract code:
	ContractCode(addrHash, codeHash common.Hash) ([]byte, error)
	ContractCodeSize(addrHash, codeHash common.Hash) (int, error)
	// CopyTrie returns an independent copy of the given trie.
	CopyTrie(Trie) Trie
	SetBlockNumber(blockNumber uint64)
	WriteState(trie.DatabaseWriter, common.Hash) error
}

// Trie is a Ethereum Merkle Trie.
type Trie interface {
	TryGet(key []byte) ([]byte, error)
	TryUpdate(key, value []byte) error
	TryDelete(key []byte) error
	CommitTo(trie.DatabaseWriter) (common.Hash, error)
	Hash() common.Hash
	NodeIterator(startKey []byte) trie.NodeIterator
	GetKey([]byte) []byte // TODO(fjl): remove this when SecureTrie is removed
}

// NewDatabase creates a backing store for state. The returned database is safe for
// concurrent use and retains cached trie nodes in memory.
func NewDatabase(db ethdb.Database, cacheDuration uint64) Database {
	csc, _ := lru.New(codeSizeCacheSize)
	return &cachingDB{db: db, pastTries: make(map[common.Hash]pastTrie), pastHeap: &pastTrieHeap{}, codeSizeCache: csc, cacheDuration: cacheDuration}
}

type pastTrie struct {
	trie        *trie.SecureTrie
	blockNumber uint64
	hash        common.Hash
}

type pastTrieHeap []pastTrie

func (pt pastTrieHeap) Len() int { return len(pt) }

func (pt pastTrieHeap) Less(i, j int) bool { return pt[i].blockNumber < pt[j].blockNumber }

func (pt pastTrieHeap) Swap(i, j int) { pt[i], pt[j] = pt[j], pt[i] }

func (pt *pastTrieHeap) Push(x interface{}) {
	*pt = append(*pt, x.(pastTrie))
}

func (pt *pastTrieHeap) Pop() interface{} {
	old := *pt
	n := len(old)
	x := old[n-1]
	*pt = old[0 : n-1]
	return x
}

type cachingDB struct {
	db            ethdb.Database
	mu            sync.Mutex
	pastTries     map[common.Hash]pastTrie
	pastHeap      *pastTrieHeap
	codeSizeCache *lru.Cache
	blockNumber   uint64
	cacheDuration uint64
}

// OpenTrie returns a trie with the specified root hash
func (db *cachingDB) OpenTrie(root common.Hash) (Trie, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if pt, ok := db.pastTries[root]; ok {
		return cachedTrie{pt.trie.Copy(), db}, nil
	}
	tr, err := trie.NewSecure(root, db.db, MaxTrieCacheGen)
	if err != nil {
		return nil, err
	}
	return cachedTrie{tr, db}, nil
}

func (db *cachingDB) SetBlockNumber(blockNumber uint64) {
	db.blockNumber = blockNumber
}

// pushTrie adds the trie to the list of past tries with the specified block number
func (db *cachingDB) pushTrie(t *trie.SecureTrie) common.Hash {
	db.mu.Lock()
	defer db.mu.Unlock()

	for uint64(db.pastHeap.Len()) > db.cacheDuration {
		it := heap.Pop(db.pastHeap).(pastTrie)
		delete(db.pastTries, it.hash)
	}

	hash := t.Hash()
	db.pastTries[hash] = pastTrie{t, db.blockNumber, hash}
	heap.Push(db.pastHeap, db.pastTries[hash])
	return hash
}

// WriteState commits state to disk as of a specific block
func (db *cachingDB) WriteState(dbw trie.DatabaseWriter, hash common.Hash) error {
	trie, ok := db.pastTries[hash]
	if !ok {
		return StateNotInCache
	}
	_, err := trie.trie.CommitTo(dbw)
	return err
}

func (db *cachingDB) OpenStorageTrie(addrHash, root common.Hash) (Trie, error) {
	return trie.NewSecure(root, db.db, 0)
}

func (db *cachingDB) CopyTrie(t Trie) Trie {
	switch t := t.(type) {
	case *trie.SecureTrie:
		return t.Copy()
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}

func (db *cachingDB) ContractCode(addrHash, codeHash common.Hash) ([]byte, error) {
	code, err := db.db.Get(codeHash[:])
	if err == nil {
		db.codeSizeCache.Add(codeHash, len(code))
	}
	return code, err
}

func (db *cachingDB) ContractCodeSize(addrHash, codeHash common.Hash) (int, error) {
	if cached, ok := db.codeSizeCache.Get(codeHash); ok {
		return cached.(int), nil
	}
	code, err := db.ContractCode(addrHash, codeHash)
	if err == nil {
		db.codeSizeCache.Add(codeHash, len(code))
	}
	return len(code), err
}

// cachedTrie inserts its trie into a cachingDB on commit.
type cachedTrie struct {
	*trie.SecureTrie
	db *cachingDB
}

func (m cachedTrie) CommitTo(dbw trie.DatabaseWriter) (common.Hash, error) {
	return m.db.pushTrie(m.SecureTrie), nil
}
