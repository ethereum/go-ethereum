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
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	lru "github.com/hashicorp/golang-lru"
)

const (
	// Number of codehash->size associations to keep.
	codeSizeCacheSize = 100000
)

// Database wraps access to tries and contract code.
type Database interface {
	// OpenTrie opens the main account trie.
	OpenTrie(root common.Hash) (Trie, error)

	// OpenStorageTrie opens the storage trie of an account.
	OpenStorageTrie(addrHash, root common.Hash) (Trie, error)

	// CopyTrie returns an independent copy of the given trie.
	CopyTrie(Trie) Trie

	// ContractCode retrieves a particular contract's code.
	ContractCode(addrHash, codeHash common.Hash) ([]byte, error)

	// ContractCodeSize retrieves a particular contracts code's size.
	ContractCodeSize(addrHash, codeHash common.Hash) (int, error)

	// TrieDB retrieves the low level trie database used for data storage.
	TrieDB() *trie.Database

	// Commit enqueues the given commit task and schedules for background committing.
	Commit(root common.Hash, stateTrie Trie, storageTries map[common.Hash]Trie, postCommit func()) error

	// WaitCommits waits until the cached commit tasks equal or less than n.
	WaitCommits(n int)

	// Close terminates all background threads and exit.
	Close()
}

// Trie is a Ethereum Merkle Patricia trie.
type Trie interface {
	// GetKey returns the sha3 preimage of a hashed key that was previously used
	// to store a value.
	//
	// TODO(fjl): remove this when SecureTrie is removed
	GetKey([]byte) []byte

	// TryGet returns the value for key stored in the trie. The value bytes must
	// not be modified by the caller. If a node was not found in the database, a
	// trie.MissingNodeError is returned.
	TryGet(key []byte) ([]byte, error)

	// TryUpdate associates key with value in the trie. If value has length zero, any
	// existing value is deleted from the trie. The value bytes must not be modified
	// by the caller while they are stored in the trie. If a node was not found in the
	// database, a trie.MissingNodeError is returned.
	TryUpdate(key, value []byte) error

	// TryDelete removes any existing value for key from the trie. If a node was not
	// found in the database, a trie.MissingNodeError is returned.
	TryDelete(key []byte) error

	// Hash returns the root hash of the trie. It does not write to the database and
	// can be used even if the trie doesn't have one.
	Hash() common.Hash

	// Commit writes all nodes to the trie's memory database, tracking the internal
	// and external (for account tries) references.
	Commit(onleaf trie.LeafCallback) common.Hash

	// NodeIterator returns an iterator that returns nodes of the trie. Iteration
	// starts at the key after the given start key.
	NodeIterator(startKey []byte) trie.NodeIterator

	// Prove constructs a Merkle proof for key. The result contains all encoded nodes
	// on the path to the value at key. The value itself is also included in the last
	// node and can be retrieved by verifying the proof.
	//
	// If the trie does not contain a value for key, the returned proof contains all
	// nodes of the longest existing prefix of the key (at least the root), ending
	// with the node that proves the absence of the key.
	Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error
}

const maxCachedState = 16 // Cache state limit exceeds which to suspend commit.

// NewDatabase creates a backing store for state. The returned database is safe for
// concurrent use, but does not retain any recent trie nodes in memory. To keep some
// historical state in memory, use the NewDatabaseWithCache constructor.
func NewDatabase(db ethdb.Database) Database {
	return NewDatabaseWithCache(db, 0, "")
}

// NewDatabaseWithCache creates a backing store for state. The returned database
// is safe for concurrent use and retains a lot of collapsed RLP trie nodes in a
// large memory cache.
func NewDatabaseWithCache(db ethdb.Database, cache int, journal string) Database {
	csc, _ := lru.New(codeSizeCacheSize)
	cdb := &cachingDB{
		db:            trie.NewDatabaseWithCache(db, cache, journal),
		codeSizeCache: csc,
		tasks:         make(map[common.Hash]*commitTask),
		close:         make(chan struct{}),
		signal:        make(chan *waitAck),
	}
	cdb.wg.Add(1)
	go cdb.run()
	return cdb
}

// commitTask contains all commitTask tries as well as a postcommit callback.
// usually the callback is necessary in order to run the memory gc algorithm.
type commitTask struct {
	root       common.Hash
	state      Trie
	storage    map[common.Hash]Trie
	postCommit func()
}

// waitAck is the response from the background committer. The ack will send
// the signal if the cached commit tasks equal or less than n.
type waitAck struct {
	n   int
	ack chan struct{}
}

// cachingDB is the intermediate layer between the stateDB and underlying memory
// database. It's responsible for caching the uncommitted tries, providing the
// tries for reuse and running the commit tasks in the background.
type cachingDB struct {
	db            *trie.Database
	codeSizeCache *lru.Cache

	lock   sync.Mutex
	fifo   []common.Hash
	tasks  map[common.Hash]*commitTask
	wg     sync.WaitGroup
	close  chan struct{}
	signal chan *waitAck
}

// OpenTrie opens the main account trie at a specific root hash.
func (db *cachingDB) OpenTrie(root common.Hash) (Trie, error) {
	db.lock.Lock()
	if task, ok := db.tasks[root]; ok {
		db.lock.Unlock()
		return task.state.(*trie.SecureTrie).HashAndCopy(), nil
	}
	db.lock.Unlock()
	return trie.NewSecure(root, db.db)
}

// OpenStorageTrie opens the storage trie of an account.
func (db *cachingDB) OpenStorageTrie(addrHash, root common.Hash) (Trie, error) {
	db.lock.Lock()
	for _, task := range db.tasks {
		if t, ok := task.storage[addrHash]; ok && t.Hash() == root {
			db.lock.Unlock()
			return t.(*trie.SecureTrie).HashAndCopy(), nil
		}
	}
	db.lock.Unlock()
	return trie.NewSecure(root, db.db)
}

// CopyTrie returns an independent copy of the given trie.
func (db *cachingDB) CopyTrie(t Trie) Trie {
	switch t := t.(type) {
	case *trie.SecureTrie:
		return t.Copy()
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}

// ContractCode retrieves a particular contract's code.
func (db *cachingDB) ContractCode(addrHash, codeHash common.Hash) ([]byte, error) {
	code, err := db.db.Node(codeHash)
	if err == nil {
		db.codeSizeCache.Add(codeHash, len(code))
	}
	return code, err
}

// ContractCodeSize retrieves a particular contracts code's size.
func (db *cachingDB) ContractCodeSize(addrHash, codeHash common.Hash) (int, error) {
	if cached, ok := db.codeSizeCache.Get(codeHash); ok {
		return cached.(int), nil
	}
	code, err := db.ContractCode(addrHash, codeHash)
	return len(code), err
}

// Commit enqueues the given commit task and schedule for background execution.
// The callback is applied when the given state is committed. It can be used
// to run memory garbage collection algorithm.
func (db *cachingDB) Commit(root common.Hash, stateTrie Trie, storageTries map[common.Hash]Trie, postCommit func()) error {
	db.lock.Lock()

	// Short circult if nothing to commit
	if root == emptyRoot {
		db.lock.Unlock()
		return nil
	}
	// Reject duplicated commit task. It can happen in several cases:
	// - in the clique networks where empty blocks don't modify the state
	// 	 (0 block subsidy). In this case we should keep the original one.
	// - in the clique networks the reorg happens that two different blocks
	//   have the same root. In this case it's meaningless to commit twice.
	if _, ok := db.tasks[root]; ok {
		db.lock.Unlock()
		return nil
	}
	db.fifo = append(db.fifo, root)
	db.tasks[root] = &commitTask{
		root:       root,
		state:      stateTrie,
		storage:    storageTries,
		postCommit: postCommit,
	}
	db.lock.Unlock()

	ack := &waitAck{
		n:   maxCachedState,
		ack: make(chan struct{}),
	}
	select {
	case db.signal <- ack:
		// If there are too many cached commit tasks, suspend the
		// commit for a while before enough tasks are finished.
		<-ack.ack
	case <-db.close:
	}
	return nil
}

// TrieDB retrieves any intermediate trie-node caching layer.
func (db *cachingDB) TrieDB() *trie.Database {
	return db.db
}

// Close terminates all background threads and exit.
func (db *cachingDB) Close() {
	close(db.close)
	db.wg.Wait()
}

// WaitCommits waits until the cached commit tasks equal or less than n.
func (db *cachingDB) WaitCommits(n int) {
	ack := &waitAck{
		n:   0,
		ack: make(chan struct{}),
	}
	select {
	case db.signal <- ack:
		<-ack.ack
	case <-db.close:
	}
}

// run is the main loop of cachingDB, all background tasks will be executed here
// with the strict order of enqueue.
func (db *cachingDB) run() {
	defer db.wg.Done()

	var (
		// Non-nil if background commit routine is active.
		done chan struct{}

		waitLock sync.Mutex
		waits    []*waitAck
	)
	// commitState flushes dirty state in the given tries into the underlying database.
	commitState := func(stateTrie Trie, storageTries map[common.Hash]Trie) {
		triedb := db.TrieDB()
		for _, storage := range storageTries {
			storage.Commit(nil)
		}
		// The onleaf func is called _serially_, so we can reuse the same account
		// for unmarshalling every time.
		var account Account
		stateTrie.Commit(func(leaf []byte, parent common.Hash) error {
			if err := rlp.DecodeBytes(leaf, &account); err != nil {
				return nil
			}
			if account.Root != emptyRoot {
				triedb.Reference(account.Root, parent)
			}
			code := common.BytesToHash(account.CodeHash)
			if code != emptyCode {
				triedb.Reference(code, parent)
			}
			return nil
		})
	}

	// release sends the "resume" signal the waiting channels.
	release := func(n int) {
		waitLock.Lock()
		defer waitLock.Unlock()

		for i := 0; i < len(waits); i++ {
			if n <= waits[i].n {
				close(waits[i].ack)
				waits[i] = waits[len(waits)-1]
				waits[i] = nil
				waits = waits[:len(waits)-1]
				i--
			}
		}
	}
	// commit tries to commit all cumulative tasks in the single thread.
	commit := func(done chan struct{}) {
		defer close(done)

		for {
			db.lock.Lock()
			if len(db.fifo) == 0 {
				db.lock.Unlock()
				return // no more work left
			}
			root := db.fifo[0]
			task := db.tasks[root]
			if task == nil {
				panic(fmt.Sprintf("missing commit task %x", root))
			}
			db.lock.Unlock()

			// Commit the tries without holding the lock. It's totally thread-safe to do it.
			commitState(task.state, task.storage)

			// Run the callback after commit if it's not nil, usually it's in-memory GC algo.
			if task.postCommit != nil {
				task.postCommit()
			}
			// Delete the executed tasks.
			db.lock.Lock()
			db.fifo = db.fifo[1:]
			delete(db.tasks, root)
			remain := len(db.fifo)
			db.lock.Unlock()

			release(remain) // send back ack as soon as possible
		}
	}
	for {
		select {
		case wait := <-db.signal:
			waitLock.Lock()
			waits = append(waits, wait)
			waitLock.Unlock()

			if done == nil {
				done = make(chan struct{})
				go commit(done)
			}
		case <-done:
			done = nil
			release(0)
		case <-db.close:
			if done != nil {
				<-done
				release(0)
			}
			return
		}
	}
}
