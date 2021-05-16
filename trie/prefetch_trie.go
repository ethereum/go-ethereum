package trie

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// PrefetchTrie has additional methods and data fields over
// Trie that allow for futures of data to be returned
// and allows for async IO within a Trie.

// GetReturnType is the return type for Get
type GetReturnType struct {
	value []byte
	err   error
}

type trieGetJob struct {
	key []byte
	ret chan GetReturnType
}

type trieKillJob struct {
	synced chan struct{}
}

type asyncTrieJob interface {
	isTrieJob()
}

func (j trieGetJob) isTrieJob()  {}
func (j trieKillJob) isTrieJob() {}

type prefetchJobWrapper struct {
	job             asyncTrieJob
	prefetchSuccess chan error
}

// PrefetchTrie allows you to fetch nodes into the DB layer
// Or to start to build a Trie async as the nodes are fetched
type PrefetchTrie struct {
	trie     SecureTrie
	jobs     chan prefetchJobWrapper
	rootLock *sync.RWMutex
}

func (t *PrefetchTrie) getRoot() node {
	t.rootLock.RLock()
	node := t.trie.trie.root
	t.rootLock.RUnlock()
	return node
}

func (t *PrefetchTrie) setRoot(newroot node) {
	t.rootLock.Lock()
	t.trie.trie.root = newroot
	t.rootLock.Unlock()
}

// NewPrefetch intializes an PrefetchTrie
func NewPrefetch(root common.Hash, db *Database) (*PrefetchTrie, error) {
	if db == nil {
		panic("trie.NewAsync called without a database")
	}

	trie, err := NewSecure(root, db)
	if err != nil {
		return nil, err
	}

	pt := PrefetchTrie{
		trie: *trie,

		jobs: make(chan prefetchJobWrapper, 2048),

		rootLock: &sync.RWMutex{},
	}

	if root != (common.Hash{}) && root != emptyRoot {
		rootnode, err := pt.trie.trie.resolveHash(root[:], nil)
		if err != nil {
			return nil, err
		}
		pt.setRoot(rootnode)
	}

	go pt.loop()
	return &pt, nil
}

// Sync blocks until all prefetch jobs for a PrefetchTries
// have completed
func (t *PrefetchTrie) Sync(clearCache bool) {
	prefetchSuccess := make(chan error, 1)
	synced := make(chan struct{})
	t.jobs <- prefetchJobWrapper{
		trieKillJob{
			synced: synced,
		},
		prefetchSuccess,
	}
	prefetchSuccess <- nil

	<-synced
}

func (t *PrefetchTrie) loop() {
	for prefetchJob := range t.jobs {
		<-prefetchJob.prefetchSuccess

		close(prefetchJob.prefetchSuccess)

		// If prefetch was a success
		switch j := prefetchJob.job.(type) {
		case trieKillJob:
			j.synced <- struct{}{}
		case trieGetJob:
			// This op should return immediately
			val, newroot, didResolve, err := t.trie.trie.tryGet(t.getRoot(), keybytesToHex(j.key), 0)
			if err == nil && didResolve {
				t.setRoot(newroot)
			}
			j.ret <- GetReturnType{val, err}
		default:
			panic("Invalid Job")
		}

	}
}

func (t *PrefetchTrie) prefetchAsyncIO(prefetchSuccess chan error, origNode node, key []byte) {
	prefetchSuccess <- t.prefetchConcurrent(origNode, keybytesToHex(key), 0)
}

// TryGetAsync submits a job to the main loop
// to build the Trie in memory, first letting async fetchers
// fetch nodes from disk into the Database layer.
func (t *PrefetchTrie) TryGetAsync(key []byte) chan GetReturnType {
	key = t.trie.hashKey(key)
	prefetchSuccess := make(chan error, 1)

	go t.prefetchAsyncIO(prefetchSuccess, t.getRoot(), copyKey(key))

	ret := make(chan GetReturnType, 1)

	t.jobs <- prefetchJobWrapper{
		job:             trieGetJob{key: copyKey(key), ret: ret},
		prefetchSuccess: prefetchSuccess,
	}

	return ret
}

func copyKey(key []byte) []byte {
	tmp := make([]byte, len(key))
	copy(tmp, key)
	return tmp
}

func (t *PrefetchTrie) prefetchConcurrent(origNode node, key []byte, pos int) error {
	switch n := (origNode).(type) {
	case nil:
		return nil
	case valueNode:
		return nil
	case *shortNode:
		if len(key)-pos < len(n.Key) || !bytes.Equal(n.Key, key[pos:pos+len(n.Key)]) {

			// key not found in trie
			return nil
		}
		return t.prefetchConcurrent(n.Val, key, pos+len(n.Key))
	case *fullNode:
		return t.prefetchConcurrent(n.Children[key[pos]], key, pos+1)
	case hashNode:
		child, err := t.trie.trie.resolveHash(n, key[:pos])
		if err != nil {
			return err
		}
		return t.prefetchConcurrent(child, key, pos)
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", origNode, origNode))
	}
}

func (t *PrefetchTrie) resolveHash(hash common.Hash, prefix []byte) (node, error) {
	if node := t.trie.trie.db.node(hash); node != nil {
		return node, nil
	}
	return nil, &MissingNodeError{NodeHash: hash, Path: prefix}
}

// Copy returns a copy of PrefetchTrie.
func (t *PrefetchTrie) Copy() *PrefetchTrie {
	cpy := *t
	cpy.rootLock = &sync.RWMutex{}
	return &cpy
}

// AsTrie returns the trie the prefetch Trie wraps
func (t *PrefetchTrie) AsTrie() *SecureTrie {
	return t.trie.Copy()
}
