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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
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
}

// CacheableDatabase extends the Database by adding the functions for lazy state commit.
type CacheableDatabase interface {
	Database

	// Commit enqueues the given commit task and schedules for background committing.
	Commit(root common.Hash, number uint64, stateTrie Trie, storageTries map[common.Hash]Trie, codes map[common.Hash][]byte, postCommit func()) error

	// WaitCommits waits until the cached commit tasks equal or less than n.
	// This function can act as the barrier between task generator and commiter.
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

var (
	stateTrieHits      = metrics.NewRegisteredMeter("chain/committer/state/hits", nil)
	stateTrieMiss      = metrics.NewRegisteredMeter("chain/committer/state/miss", nil)
	accountCommitTimer = metrics.NewRegisteredTimer("chain/committer/state/commits", nil)
	storageTrieHits    = metrics.NewRegisteredMeter("chain/committer/storage/hits", nil)
	storageTrieMiss    = metrics.NewRegisteredMeter("chain/committer/storage/miss", nil)
	storageCommitTimer = metrics.NewRegisteredTimer("chain/committer/storage/commits", nil)
	codeHits           = metrics.NewRegisteredMeter("chain/committer/code/hits", nil)
	codeMiss           = metrics.NewRegisteredMeter("chain/committer/code/miss", nil)
	memoryGCTimer      = metrics.NewRegisteredTimer("chain/committer/gc/duration", nil)
)

// NewDatabase creates a backing store for state. The returned database is safe for
// concurrent use, but does not retain any recent trie nodes in memory. To keep some
// historical state in memory, use the NewDatabaseWithCache constructor.
func NewDatabase(db ethdb.Database) CacheableDatabase {
	return NewDatabaseWithCache(db, 0, "")
}

// NewDatabaseWithCache creates a backing store for state. The returned database
// is safe for concurrent use and retains a lot of collapsed RLP trie nodes in a
// large memory cache.
func NewDatabaseWithCache(db ethdb.Database, cache int, journal string) CacheableDatabase {
	csc, _ := lru.New(codeSizeCacheSize)
	cdb := &cachingDB{
		db:            trie.NewDatabaseWithCache(db, cache, journal),
		codeSizeCache: csc,
		tasks:         make(map[common.Hash]*commitTask),
		close:         make(chan struct{}),
		task:          make(chan *commitTask),
	}
	cdb.wg.Add(1)
	go cdb.run()
	return cdb
}

// commitTask contains all commitTask tries as well as a postcommit callback.
// usually the callback is necessary in order to run the memory gc algorithm.
type commitTask struct {
	root       common.Hash
	number     uint64
	state      Trie
	storage    map[common.Hash]Trie
	codes      map[common.Hash][]byte
	postCommit func()

	// ACK fields.
	waitN int
	ack   chan struct{}
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

	lock  sync.Mutex
	fifo  []common.Hash
	tasks map[common.Hash]*commitTask
	wg    sync.WaitGroup
	close chan struct{}
	task  chan *commitTask
}

// OpenTrie opens the main account trie at a specific root hash.
func (db *cachingDB) OpenTrie(root common.Hash) (Trie, error) {
	db.lock.Lock()
	if task, ok := db.tasks[root]; ok {
		db.lock.Unlock()
		stateTrieHits.Mark(1)
		return task.state.(*trie.SecureTrie).HashAndCopy(), nil
	}
	db.lock.Unlock()
	stateTrieMiss.Mark(1)
	return trie.NewSecure(root, db.db)
}

// OpenStorageTrie opens the storage trie of an account.
func (db *cachingDB) OpenStorageTrie(addrHash, root common.Hash) (Trie, error) {
	db.lock.Lock()
	for _, task := range db.tasks {
		if t, ok := task.storage[addrHash]; ok && t.Hash() == root {
			db.lock.Unlock()
			storageTrieHits.Mark(1)
			return t.(*trie.SecureTrie).HashAndCopy(), nil
		}
	}
	db.lock.Unlock()
	storageTrieMiss.Mark(1)
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
	db.lock.Lock()
	for _, task := range db.tasks {
		if code, ok := task.codes[codeHash]; ok {
			db.lock.Unlock()
			codeHits.Mark(1)
			return code, nil
		}
	}
	db.lock.Unlock()
	codeMiss.Mark(1)
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
func (db *cachingDB) Commit(root common.Hash, number uint64, stateTrie Trie, storageTries map[common.Hash]Trie, codes map[common.Hash][]byte, postCommit func()) error {
	// Short circult if nothing to commit
	if root == emptyRoot {
		return nil
	}
	db.lock.Lock()
	// Reject duplicated commit task. It can happen in several cases:
	// - in the clique networks where empty blocks don't modify the state
	// 	 (0 block subsidy). In this case we should keep the original one.
	// - in the clique networks the reorg happens that two different blocks
	//   have the same root. In this case it's meaningless to commit twice.
	if _, ok := db.tasks[root]; ok {
		db.lock.Unlock()
		return nil
	}
	left := len(db.fifo)
	db.lock.Unlock()

	// If there are too many cached commit tasks, suspend the
	// commit for a while before enough tasks are finished.
	var (
		ack   = make(chan struct{})
		waitN = -1
	)
	if left+1 > maxCachedState {
		waitN = maxCachedState
	}
	select {
	case db.task <- &commitTask{
		root:       root,
		number:     number,
		state:      stateTrie,
		storage:    storageTries,
		codes:      codes,
		postCommit: postCommit,
		waitN:      waitN,
		ack:        ack,
	}:
		log.Debug("Enqueue commit task", "root", root, "number", number)
		<-ack
	case <-db.close:
		return nil // DB closed
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
	ack := make(chan struct{})
	select {
	case db.task <- &commitTask{
		waitN: n,
		ack:   ack,
	}:
		<-ack
	case <-db.close:
	}
}

// pick retrieves the task in the head, return nil if there is nothing left.
func (db *cachingDB) pick() *commitTask {
	db.lock.Lock()
	if len(db.fifo) == 0 {
		db.lock.Unlock()
		return nil
	}
	root := db.fifo[0]
	task := db.tasks[root]
	if task == nil {
		panic(fmt.Sprintf("missing commit task %x", root))
	}
	db.lock.Unlock()
	return task
}

// del deletes the head task from the queue and returns the remaining tasks as well.
func (db *cachingDB) del() int {
	db.lock.Lock()
	if len(db.fifo) == 0 {
		db.lock.Unlock()
		return 0
	}
	root := db.fifo[0]
	db.fifo = db.fifo[1:]
	delete(db.tasks, root)
	left := len(db.fifo)
	db.lock.Unlock()
	return left
}

// run is the main loop of cachingDB, all background tasks will be executed here
// with the strict order of enqueue.
func (db *cachingDB) run() {
	defer db.wg.Done()

	var (
		// Non-nil if background commit routine is active.
		done  chan struct{}
		waits []*waitAck
	)
	// commitState flushes dirty state in the given tries into the underlying database.
	commitState := func(stateTrie Trie, storageTries map[common.Hash]Trie, codes map[common.Hash][]byte) {
		triedb := db.TrieDB()
		start := time.Now()
		for _, storage := range storageTries {
			storage.Commit(nil)
		}
		storageCommitTimer.UpdateSince(start)

		for h, code := range codes {
			triedb.InsertBlob(h, code)
		}
		// The onleaf func is called _serially_, so we can reuse the same account
		// for unmarshalling every time.
		var account Account
		start = time.Now()
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
		accountCommitTimer.UpdateSince(start)
	}
	// release sends the "resume" signal to the waiting channels.
	release := func(n int) {
		for i := 0; i < len(waits); i++ {
			if n <= waits[i].n {
				close(waits[i].ack)
				waits[i] = waits[len(waits)-1]
				waits[len(waits)-1] = nil
				waits = waits[:len(waits)-1]
				i--
			}
		}
	}
	// commit tries to commit the given task in the single thread.
	commit := func(done chan struct{}, task *commitTask) {
		defer close(done)

		start := time.Now()
		commitState(task.state, task.storage, task.codes)

		callstart := time.Now()
		if task.postCommit != nil {
			task.postCommit()
		}
		memoryGCTimer.UpdateSince(callstart)
		log.Debug("State committed", "root", task.root, "number", task.number, "elapsed", common.PrettyDuration(time.Since(start)))
	}
	for {
		if done == nil {
			task := db.pick()
			if task != nil {
				done = make(chan struct{})
				go commit(done, task)
			}
		}
		select {
		case t := <-db.task:
			db.lock.Lock()
			left := len(db.fifo)
			if t.root != (common.Hash{}) {
				db.fifo = append(db.fifo, t.root)
				db.tasks[t.root] = t
				left += 1
			}
			db.lock.Unlock()
			if t.waitN == -1 || left == 0 {
				close(t.ack)
			} else {
				waits = append(waits, &waitAck{
					n:   t.waitN,
					ack: t.ack,
				})
			}

		case <-done:
			done = nil
			// Cleanup the processed task and resume the
			// waiting acks as soon as possible.
			release(db.del())

		case <-db.close:
			for {
				if done != nil {
					<-done
					done = nil
					release(db.del())
				}
				task := db.pick()
				if task == nil {
					return
				}
				done = make(chan struct{})
				go commit(done, task)
			}
		}
	}
}
