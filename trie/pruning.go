// Copyright 2019 The go-ethereum Authors
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

package trie

import (
	"bytes"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/karalabe/cookiejar/collections/deque"
)

var (
	memcachePruneTimeTimer  = metrics.NewRegisteredResettingTimer("trie/memcache/prune/time", nil)
	memcachePruneNodesMeter = metrics.NewRegisteredMeter("trie/memcache/prune/nodes", nil)
	memcachePruneSizeMeter  = metrics.NewRegisteredMeter("trie/memcache/prune/size", nil)

	memcachePruneAssignHistogram    = metrics.NewRegisteredHistogram("trie/memcache/prune/assign", nil, metrics.NewExpDecaySample(1028, 0.015))
	memcachePruneAssignDupHistogram = metrics.NewRegisteredHistogram("trie/memcache/prune/assigndup", nil, metrics.NewExpDecaySample(1028, 0.015))
	memcachePruneRemainHistogram    = metrics.NewRegisteredHistogram("trie/memcache/prune/remain", nil, metrics.NewExpDecaySample(1028, 0.015))
	memcachePruneRemainDupHistogram = metrics.NewRegisteredHistogram("trie/memcache/prune/remaindup", nil, metrics.NewExpDecaySample(1028, 0.015))
	memcachePruneQueueHistogram     = metrics.NewRegisteredHistogram("trie/memcache/prune/queue", nil, metrics.NewExpDecaySample(1028, 0.015))
)

// pruner is responsible for pruning the state trie based on liveness checks
// whenever the in-memory garbage collector attempt to dereference a node from
// disk.
//
// Note, the pruner is not a standalone construct, rather an extension to the
// trie database. No attempt was made to separate the API surface and make one
// a disjoint client of the other.
type pruner struct {
	db *Database // Trie database for accessing dirty and clean data

	taskCh   chan []*prunerTarget // Task queue receiving the pruning targets to delete
	abortCh  chan chan struct{}   // Notification channel to terminate the pruner
	resumeCh chan chan struct{}   // Notification channel to resume the pruner

	interrupt uint32 // Signals to a running deep pruning to suspend itself
}

// prunerTarget represents a single marked target for potential pruning.
type prunerTarget struct {
	owner common.Hash // Owner account hash of the node to delete
	path  []byte      // Patricia path leading to this node
	hash  common.Hash // Hash of the node to delete
}

// newPruner creates a new background trie pruner to delete unreferenced nodes
// whenever the tries are not being actively written.
func newPruner(db *Database) *pruner {
	p := &pruner{
		db:       db,
		taskCh:   make(chan []*prunerTarget),
		abortCh:  make(chan chan struct{}),
		resumeCh: make(chan chan struct{}),
	}
	go p.loop()
	return p
}

// enqueue adds a batch of potential prune targets to the removal queue to be
// inspected and removed from the database if deemed unreferenced by recentMeter.
// and snapshot tries.
//
// It's important to queue in batches as a single block might enque hundreds or
// thousands of targets. Queueing individually entails a huge performance hit.
func (p *pruner) enqueue(targets []*prunerTarget) {
	p.taskCh <- targets
}

// resume (re)starts the pruning, locking the dirty caches for reads to prevent
// trie nodes going missing due to concurrent pruning/referencing.
//
// Note, calling resume on an already running pruner will deadlock!
func (p *pruner) resume() {
	// The prumer might have been interrupted previously, so we need to ensure the
	// interrut is cleared before requesting a resumption. This could be done by the
	// pruner ron loop too, but figured it might be cleaner to set the interrupt at
	// the same scope as with `pause`,
	atomic.StoreUint32(&p.interrupt, 0)

	// We *must* wait for the pruner to obtain the lock, otherwise the caller might
	// race forward and lock the database for writing, messing up the state machine.
	ch := make(chan struct{})
	p.resumeCh <- ch
	<-ch
}

// pause signals the pruner to interrupt its operation and release its held lock.
// This is needed for the block processor to obtain a write lock on the dirty
// caches, which are otherwise held hostage by the pruner.
//
// Note, calling pause on a non-running pruner will panic!
func (p *pruner) pause() {
	// Notify the pruner to abort right now.
	atomic.StoreUint32(&p.interrupt, 1)
}

// terminate signals the pruner to finish all remaining tasks and permanently
// release all locks and clean itself up.
//
// Note, calling terminate on a non-running pruner will panic!
func (p *pruner) terminate() {
	// Signal to the pruner that it should terminate itself gracefully and wait for
	// it to confirm before pulling the rug from underneath.
	ch := make(chan struct{})
	p.abortCh <- ch
	<-ch
}

// loop is the pruner background gorutineo that waits for pruning targets the be
// added, causing liveness checks and potentially database deletions in response.
func (p *pruner) loop() {
	var (
		tasks   = deque.New()               // Queue of trie nodes queued for potential pruning
		taskset = make(map[string]struct{}) // Set of trie nodes queued to prevent duplication

		tries []*traverser  // Individual trie traversers for liveness checks
		quit  chan struct{} // Quit signal channel when termination is requested

		batch = p.db.diskdb.NewBatch() // Create a write batch to minimize thrashing
	)
	// Wait for different events and process them accordingly
	for {
		select {
		case targets := <-p.taskCh:
			// New task received, queue it up. We will not start immediately processing
			// this as the enqueueing is done whilst doing in-memory garbage collection,
			// so the dirty caches are locked for writing.
			duplicates := 0
			for _, task := range targets {
				key := makeNodeKey(task.owner, task.hash)
				if _, exists := taskset[key]; exists {
					duplicates++
					continue
				}
				tasks.PushRight(task)
				taskset[key] = struct{}{}
			}
			memcachePruneQueueHistogram.Update(int64(tasks.Size()))
			memcachePruneAssignHistogram.Update(int64(len(targets)))
			memcachePruneAssignDupHistogram.Update(int64(duplicates))

		case ch := <-p.resumeCh:
			// Pruner was requested to resume operation. Obtain the necessary locks to
			// prevent the block processor for modifying the dirty caches, but allow any
			// goroutines to still read the data.
			if tasks.Size() == 0 {
				ch <- struct{}{} // signal back, but nothing to do really
				continue
			}
			p.db.lock.RLock()
			ch <- struct{}{} // signal back that the lock was obtained

			// Ensure the traversers are pointing to the currently live tries. Usually
			// after each pause/resume cycle, one (new block) or two (new snapshot) tries
			// get swapped out.
			tries = nil // cheat a bit for now and just reconstruct them

			for key := range p.db.dirties[metaRoot].children {
				_, root := splitNodeKey(key)
				tries = append(tries, &traverser{
					db:    p.db,
					state: &traverserState{hash: root, node: hashNode(root[:])},
				})
			}
			for hash := range p.db.noprune {
				tries = append(tries, &traverser{
					db:    p.db,
					state: &traverserState{hash: hash, node: hashNode(common.CopyBytes(hash[:]))}, // need closure!
				})
			}
			// Process the tasks until an interrupt arrives
			start, nodes, size := time.Now(), p.db.prunenodes, p.db.prunesize

			for !tasks.Empty() {
				// Delete this particular task from the deduplication set
				task := tasks.PopLeft().(*prunerTarget)
				delete(taskset, makeNodeKey(task.owner, task.hash))

				// Prune the target and reschedule any interrupted sub-tasks
				remain := p.prune(task.owner, task.hash, task.path, taskset, tries, batch)
				if atomic.LoadUint32(&p.interrupt) == 1 {
					duplicates := 0
					for j := len(remain) - 1; j >= 0; j-- { // reverse to keep the depth priority
						// Dedup already scheduled tasks, no need to prune twice
						key := makeNodeKey(remain[j].owner, remain[j].hash)
						if _, exist := taskset[key]; exist {
							duplicates++
							continue
						}
						// Reschedule (high priority) anything that's not a duplicate
						tasks.PushLeft(remain[j])
						taskset[key] = struct{}{}
					}
					memcachePruneQueueHistogram.Update(int64(tasks.Size()))
					memcachePruneRemainHistogram.Update(int64(len(remain)))
					memcachePruneRemainDupHistogram.Update(int64(duplicates))

					break
				}
			}
			// If all tasks have been procesed, get rid of any allocated task slice and
			// terminate the runner pathway.
			if tasks.Empty() {
				tasks.Reset()

				memcachePruneQueueHistogram.Update(0)
				memcachePruneRemainHistogram.Update(0)
				memcachePruneRemainDupHistogram.Update(0)
			}
			// Update all the stats with the results until now
			memcachePruneNodesMeter.Mark(int64(p.db.prunenodes - nodes))
			memcachePruneSizeMeter.Mark(int64(p.db.prunesize - size))
			memcachePruneTimeTimer.Update(time.Since(start))

			p.db.prunetime += time.Since(start)

			// Push any change to disk
			if err := batch.Write(); err != nil {
				log.Crit("Failed to flush pruned nodes", "err", err)
			}
			batch.Reset()

			// Relinquish the lock to any writer. We can't do this earlier to avoid data
			// races between this goroutine deleting the same node that some other one is
			// attempting to put back.
			p.db.lock.RUnlock()

			// If we're actually shutting down, clean up everything
			if quit != nil {
				quit <- struct{}{}
				return
			}

		case quit = <-p.abortCh:
			// Pruner was requetsed to terminate. Since termination doesn't interrupt, we
			// can at this point safely assume everything was pruned.
			quit <- struct{}{}
			return
		}
	}
}

var faultyOwner = common.HexToHash("0x9f13f88230a70de90ed5fa41ba35a5fb78bc55d11cc9406f17d314fb67047ac7")
var faultyHash = common.HexToHash("0x5610d8d5e4056edad0db0743616df01c0911675c5fc5604c8967312a4961cf72")

// prune deletes a trie node from disk if there are no more live references to
// it, cascading until all dangling nodes are removed. If the pruner's interrupt
// has been triggered (block processing pending), the remaining nodes are bubbled
// up to the caller to reschedule later.
func (p *pruner) prune(owner common.Hash, hash common.Hash, path []byte, taskset map[string]struct{}, tries []*traverser, batch ethdb.Batch) []*prunerTarget {
	if owner == faultyOwner && hash == faultyHash {
		log.Error("Evaluating sensitive dex trie node", "owner", owner, "hash", hash, "path", hexutil.Encode(path))
	}
	// If the node is already queued for pruning, don't duplicate any effort on it
	key := makeNodeKey(owner, hash)
	if _, ok := taskset[key]; ok {
		return nil
	}
	// If the node is still live in the memory cache, it's still referenced so we
	// can abort. This case is important when and old trie being pruned references
	// a new node (maybe that node was recreated since), since currently live nodes
	// are stored expanded, not as hashes.
	if p.db.dirties[key] != nil {
		return nil
	}
	// Iterate over all the live tries and check node liveliness
	crosspath := path
	if owner != (common.Hash{}) {
		crosspath = append(append(keybytesToHex(owner[:]), 0xff), crosspath...)
	}
	unrefs := make(map[common.Hash]bool)
	for _, trie := range tries {
		// If the node is still live, abort
		if trie.live(owner, hash, crosspath, unrefs) {
			return nil
		}
		// Node dead in this trie, cache the result for subsequent traversals
		trie.unref(2, unrefs)
	}
	// Dead node found, delete it from the database
	dead := []byte(key)
	blob, err := p.db.diskdb.Get(dead)
	if blob == nil || err != nil {
		// Node already deleted by something else, happens with delayed pruning
		//log.Error("Missing prune target", "owner", owner, "hash", hash, "path", fmt.Sprintf("%x", path))
		return nil
	}
	node := mustDecodeNode(hash[:], blob, 0)

	// Prune the node and its children if it's not a bytecode blob
	if owner == faultyOwner && hash == faultyHash {
		log.Error("Deleting sensitive dex trie node", "owner", owner, "hash", hash, "path", hexutil.Encode(path))
	}
	p.db.cleans.Delete(string(hash[:]))
	batch.Delete(dead)
	p.db.prunenodes++
	p.db.prunesize += common.StorageSize(len(blob))

	var remain []*prunerTarget
	iterateRefs(node, path, func(path []byte, hash common.Hash) error {
		// If the pruner was interrupted, accumulate the remaining targets
		if atomic.LoadUint32(&p.interrupt) == 1 {
			remain = append(remain, &prunerTarget{owner: owner, hash: hash, path: common.CopyBytes(path)})
			return nil
		}
		// Pruning not interrupted until now, attempt to process children too. It's
		// fine to assign the result directly to the `remain` slice because it's nil
		// anyway until the interrupt triggers.
		remain = p.prune(owner, hash, path, taskset, tries, batch)
		return nil
	})
	return remain
}

// traverser is a stateful trie traversal data structure used by the pruner to
// verify the liveness of a node within a specific trie. The reason for having
// a separate data structure is to allow reusing previous traversals to check
// the liveness of nested nodes (i.e. entire subtried during pruning).
type traverser struct {
	db    *Database       // Trie database for accessing dirty and clean data
	state *traverserState // Leftover state from the previous traversals
}

// traverserState is the internal state of a trie traverser.
type traverserState struct {
	parent *traverserState // Parent traverser to allow backtracking
	prefix []byte          // Path leading up to the root of this traverser
	node   node            // Trie node where this traverser is currently at
	hash   common.Hash     // Hash of the trie node at the traversed position
}

// live checks whether the trie iterated by this traverser contains the hashnode
// at the given path, minimizing data access and processing by reusing previous
// state instead of starting fresh.
//
// The path is a full canonical path from the account trie root down to the node
// potentially crossing over into a storage trie. The account and storage trie
// paths are separated by a 0xff byte (nibbles range from 0x00-0x10). This byte
// is needed to differentiate between the leaf of the account trie and the root
// of a storage trie (which otherwise would have the same traversal path).
func (t *traverser) live(owner common.Hash, hash common.Hash, path []byte, unrefs map[common.Hash]bool) bool {
	// Rewind the traverser until it's prefix is actually a prefix of the path
	for !bytes.HasPrefix(path, t.state.prefix) {
		t.state = t.state.parent
	}
	// Short circuit the liveness check if we already covered this prefix (if this
	// prefix path was not yet seen in previous tries, no parent could have been
	// seen either, so no point in checkin upwards further than the first hash).
	for state := t.state; state != nil; state = state.parent {
		if state.hash != (common.Hash{}) {
			if unrefs[state.hash] {
				return false
			}
			break
		}
	}
	// Traverse downward until the prefix matches the path completely
	path = path[len(t.state.prefix):]
	for len(path) > 0 {
		// If we're at a hash node, expand before continuing
		if n, ok := t.state.node.(hashNode); ok {
			// Short circuit if we already encountered this node
			t.state.hash = common.BytesToHash(n)
			if unrefs[t.state.hash] {
				return false
			}
			// Generate the database key for this hash node
			var key string
			if len(t.state.prefix) < 2*common.HashLength {
				key = makeNodeKey(common.Hash{}, t.state.hash)
			} else {
				key = makeNodeKey(owner, t.state.hash)
			}
			// Replace the node in the traverser with the expanded one
			if enc, err := t.db.cleans.Get(string(t.state.hash[:])); err == nil && enc != nil {
				t.state.node = mustDecodeNode(t.state.hash[:], enc, 0)
			} else if node := t.db.dirties[key]; node != nil {
				t.state.node = node.node
			} else {
				blob, err := t.db.diskdb.Get([]byte(key))
				if blob == nil || err != nil {
					log.Error("Missing referenced node", "owner", owner, "hash", t.state.hash.Hex(), "path", fmt.Sprintf("%x%x", t.state.prefix, path))
					return false
					//panic(fmt.Sprintf("missing referenced node %x (searching for %x:%x at %x%x)", key, owner, t.state.hash, t.state.prefix, path))
				}
				t.state.node = mustDecodeNode(t.state.hash[:], blob, 0)
				t.db.cleans.Set(string(t.state.hash[:]), blob)
			}
		}
		// If we reached an account node, extract the storage trie root to continue on
		if path[0] == 0xff {
			// Retrieve the storage trie root and abort if empty
			if have, ok := t.state.node.(valueNode); ok {
				var account struct {
					Nonce    uint64
					Balance  *big.Int
					Root     common.Hash
					CodeHash []byte
				}
				if err := rlp.DecodeBytes(have, &account); err != nil {
					panic(err)
				}
				if account.Root == emptyRoot {
					return false
				}
				// Create a new nesting in the traversal and continue on that depth
				t.state, path = &traverserState{
					parent: t.state,
					prefix: append(t.state.prefix, 0xff),
					node:   hashNode(account.Root[:]),
				}, path[1:]
				continue
			}
			panic(fmt.Sprintf("liveness check path swap terminated on non value node: %T", t.state.node))
		}
		// Descend into the trie following the specified path. This code segment must
		// be able to handle both simplified raw nodes kept in this cache as well as
		// cold nodes loaded directly from disk.
		switch n := t.state.node.(type) {
		case *rawShortNode:
			if prefixLen(n.Key, path) == len(n.Key) {
				t.state, path = &traverserState{
					parent: t.state,
					prefix: append(t.state.prefix, path[:len(n.Key)]...),
					node:   n.Val,
				}, path[len(n.Key):]
				continue
			}
			return false

		case *shortNode:
			if prefixLen(n.Key, path) == len(n.Key) {
				t.state, path = &traverserState{
					parent: t.state,
					prefix: append(t.state.prefix, path[:len(n.Key)]...),
					node:   n.Val,
				}, path[len(n.Key):]
				continue
			}
			return false

		case rawFullNode:
			if child := n[path[0]]; child != nil {
				t.state, path = &traverserState{
					parent: t.state,
					prefix: append(t.state.prefix, path[0]),
					node:   child,
				}, path[1:]
				continue
			}
			return false

		case *fullNode:
			if child := n.Children[path[0]]; child != nil {
				t.state, path = &traverserState{
					parent: t.state,
					prefix: append(t.state.prefix, path[0]),
					node:   child,
				}, path[1:]
				continue
			}
			return false

		default:
			panic(fmt.Sprintf("unknown node type: %T", n))
		}
	}
	// The prefix should match perfectly here, check if the hashes matches
	if t.state.hash != (common.Hash{}) { // expanded/cached hash node
		return t.state.hash == hash
	}
	if have, ok := t.state.node.(hashNode); ok { // collapsed hash node
		t.state.hash = common.BytesToHash(have)
		return t.state.hash == hash
	}
	return false
}

// unref marks the current traversal nodes as *not* containing the specific trie
// node having been searched for. It is used by searches in subsequent tries to
// avoid reiterating the exact same sub-tries.
func (t *traverser) unref(count int, unrefs map[common.Hash]bool) {
	state := t.state
	for state != nil && count > 0 {
		// If we've found a hash node, store it as a subresult
		if state.hash != (common.Hash{}) {
			unrefs[state.hash] = true
			count--
		}
		// Traverse further up to the next hash node
		state = state.parent
	}
}
