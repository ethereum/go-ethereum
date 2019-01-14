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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
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

	taskCh      chan *prunerTarget // Task queue receiving the pruning targets to delete
	pauseCh     chan chan struct{} // Notification channel to pause the pruner
	resumeCh    chan chan struct{} // Notification channel to resume the pruner
	terminateCh chan chan struct{} // Notification channel to terminate the pruner
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
		db:          db,
		taskCh:      make(chan *prunerTarget, 128),
		pauseCh:     make(chan chan struct{}),
		resumeCh:    make(chan chan struct{}),
		terminateCh: make(chan chan struct{}),
	}
	go p.loop()
	return p
}

// enqueue adds a potential prune target to the removal queue to be inspected and
// removed from the database if deemed unreferenced by recent and snapshot tries.
func (p *pruner) enqueue(owner common.Hash, hash common.Hash, path []byte) {
	p.taskCh <- &prunerTarget{
		owner: owner,
		hash:  hash,
		path:  common.CopyBytes(path),
	}
}

// resume (re)starts the pruning, locking the dirty caches for reads to prevent
// trie nodes going missing due to concurrent pruning/referencing.
//
// Note, calling resume on an already running pruner will deadlock! The pruner is
// initially paused.
func (p *pruner) resume() {
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
// Note, calling pause on a non-running pruner will panic! The pruner is initially
// paused.
func (p *pruner) pause() {
	// We don't really need to wait for the pause to complete here as we're unable
	// to obtain a write-lock sooner anyway, but it's perhaps nicer code to make it
	// symmetrical to `resume`.
	ch := make(chan struct{})
	p.pauseCh <- ch
	<-ch
}

// terminate signals the pruner to finish all remaining tasks and permanently
// release all locks and clean itself up.
func (p *pruner) terminate() {
	ch := make(chan struct{})
	p.terminateCh <- ch
	<-ch
}

// loop is the pruner background gorutineo that waits for pruning targets the be
// added, causing liveness checks and potentially database deletions in response.
func (p *pruner) loop() {
	var (
		runner chan struct{}   // Runner channel acting as a boolean 'running' flag
		tasks  []*prunerTarget // Batch of trie nodes queued for potential pruning
		tries  []*traverser    // Individual trie traversers for liveness checks
		done   int             // Number of pruning tasks done, for smarter CG

		batch = p.db.diskdb.NewBatch() // Create a write batch to minimize thrashing

		start time.Time          // Time instance when the pruner was resumed
		nodes uint64             // Number of nodes pruned when the pruner was resumed
		size  common.StorageSize // Number of bytes pruned when the pruner was resumed

		quit     chan struct{}    // Quit signal channel when termination is requested
		quitting <-chan time.Time // Ticker to periodically log termination progress
	)
	// Wait for different events and process them accordingly
	for {
		select {
		case task := <-p.taskCh:
			// New task received, queue it up. We will not start immediately processing
			// this as the enqueueing is done whilst doing in-memory garbage collection,
			// so the dirty caches are locked for writing.
			tasks = append(tasks, task)

		case ch := <-p.resumeCh:
			// Pruner was requested to resume operation. Obtain the necessary locks to
			// prevent the block processor for modifying the dirty caches, but allow any
			// goroutines to still read the data.
			p.db.lock.RLock()
			ch <- struct{}{} // signal back that the lock was obtained

			// Only proceed with task processing if there's something available
			if len(tasks) > 0 {
				// Create a runner channel that will allow running whenever checked
				runner = make(chan struct{})
				close(runner)

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
			}
			// Mark the resumption to track the pruning time
			start, nodes, size = time.Now(), p.db.prunenodes, p.db.prunesize

		case ch := <-p.pauseCh:
			// Pruner was requestd to pause operation. We can just release the read lock
			// and stop processing the queued tasks.

			// Destroy the runner, disabling the deletion part of the event loop.
			if runner != nil {
				memcachePruneNodesMeter.Mark(int64(p.db.prunenodes - nodes))
				memcachePruneSizeMeter.Mark(int64(p.db.prunesize - size))
				memcachePruneTimeTimer.Update(time.Since(start))
				p.db.prunetime += time.Since(start)
				runner = nil
			}
			// Signal back that the lock was released and nothing touches the database
			// filds any more.
			p.db.lock.RUnlock()
			ch <- struct{}{}

			// If we have anything queued up for writing, might as well push it out now
			if batch.ValueSize() > 0 {
				if err := batch.Write(); err != nil {
					log.Crit("Failed to flush pruned nodes", "err", err)
				}
			}
			batch.Reset()

		case quit = <-p.terminateCh:
			// Pruner was requetsed to terminate. If everything was already processed, we
			// can exit cleanly. Otherwise we must schedule a cleanup.
			if len(tasks) == 0 {
				p.db.lock.RUnlock()
				quit <- struct{}{}
				return
			}
			// Still some tasks left, create a progress ticker to not hang the user
			log.Info("Pruner finishing pending jobs", "count", len(tasks))

			quitter := time.NewTicker(8 * time.Second)
			defer quitter.Stop()
			quitting = quitter.C

		case <-quitting:
			// A bit of time passed since the last info log, print our progress
			log.Info("Pruner finishing pending jobs", "count", len(tasks))

		case <-runner:
			// No interesting events available, but pruner is permitted to delete queued
			// up tasks. Process the next one.
			p.prune(tasks[0].owner, tasks[0].hash, tasks[0].path, tries, batch)

			// Delete the task from the queue. Here let's be a bit smarter to prevent the
			// task slice growing indefinitely.
			if done++; done%1024 == 0 {
				tasks = append([]*prunerTarget{}, tasks[1:]...)
			} else {
				tasks = tasks[1:]
			}
			// If we're out of pruning tasks, stop looping the runner (but don't release
			// the lock, that's up to higher layer code to request).
			if len(tasks) == 0 {
				// Update all the stats and disable the runner
				memcachePruneNodesMeter.Mark(int64(p.db.prunenodes - nodes))
				memcachePruneSizeMeter.Mark(int64(p.db.prunesize - size))
				memcachePruneTimeTimer.Update(time.Since(start))
				p.db.prunetime += time.Since(start)

				runner = nil

				// If we're actually shutting down, clean up everything
				if quit != nil {
					if err := batch.Write(); err != nil {
						log.Crit("Failed to flush pruned nodes", "err", err)
					}
					batch.Reset()

					p.db.lock.RUnlock()
					quit <- struct{}{}
					return
				}
			}
		}
	}
}

// prune deletes a trie node from disk if there are no more live references to
// it, cascading until all dangling nodes are removed.
func (p *pruner) prune(owner common.Hash, hash common.Hash, path []byte, tries []*traverser, batch ethdb.Batch) {
	// If the node is still live in the memory cache, it's still referenced so we
	// can abort. This case is important when and old trie being pruned references
	// a new node (maybe that node was recreted since), since currently live nodes
	// are stored expanded, not as hashes.
	key := makeNodeKey(owner, hash)
	if p.db.dirties[key] != nil {
		return
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
			return
		}
		// Node dead in this trie, cache the result for subsequent traversals
		trie.unref(2, unrefs)
	}
	// Dead node found, delete it from the database
	dead := []byte(makeNodeKey(owner, hash))
	blob, err := p.db.diskdb.Get(dead)
	if blob == nil || err != nil {
		log.Error("Missing prune target", "owner", owner, "hash", hash, "path", fmt.Sprintf("%x", path))
		return
	}
	node := mustDecodeNode(hash[:], blob, 0)

	// Prune the node and its children if it's not a bytecode blob
	p.db.cleans.Delete(string(hash[:]))
	batch.Delete(dead)
	p.db.prunenodes++
	p.db.prunesize += common.StorageSize(len(blob))

	iterateRefs(node, path, func(path []byte, hash common.Hash) error {
		p.prune(owner, hash, path, tries, batch)
		return nil
	})
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
