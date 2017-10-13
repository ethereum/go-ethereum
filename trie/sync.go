// Copyright 2015 The go-ethereum Authors
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
	"errors"
	"fmt"
	"hash"
	"math"
	"reflect"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/ethdb"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

// ErrNotRequested is returned by the trie sync when it's requested to process a
// node it did not request.
var ErrNotRequested = errors.New("not requested")

// ErrAlreadyProcessed is returned by the trie sync when it's requested to process a
// node it already processed previously.
var ErrAlreadyProcessed = errors.New("already processed")

// request represents a scheduled or already in-flight state retrieval request.
type request struct {
	hash common.Hash // Hash of the node data content to retrieve
	data []byte      // Data content of the node, cached until all subtrees complete
	raw  bool        // Whether this is a raw entry (code) or a trie node

	parents []*request // Parent state nodes referencing this entry (notify all upon completion)
	depth   int        // Depth level within the trie the node is located to prioritise DFS
	deps    int        // Number of dependencies before allowed to commit this node

	callback TrieSyncLeafCallback // Callback to invoke if a leaf node it reached on this branch
}

// SyncResult represents a response to a trie node retrieval request. The result
// data might be a simple binary blob if returning only a single node, or it may
// be a batch of trie leaves (with associated merkle proofs) if returning batched
// results.
type SyncResult struct {
	Data   []byte   // Data content of the retrieved node, in node-sync mode
	Keys   [][]byte // Trie keys rooted under the specified hash, in leaf-sync mode
	Values [][]byte // Trie values rooted under the specified hash, in leaf-sync mode
	Proof  [][]byte // Proofs to validate the leaves, in leaf-sync mode, if leaves are partial
}

// syncMemBatch is an in-memory buffer of successfully downloaded but not yet
// persisted data items.
type syncMemBatch struct {
	batch map[common.Hash][]byte // In-memory membatch of recently completed items
	order []common.Hash          // Order of completion to prevent out-of-order data loss
}

// newSyncMemBatch allocates a new memory-buffer for not-yet persisted trie nodes.
func newSyncMemBatch() *syncMemBatch {
	return &syncMemBatch{
		batch: make(map[common.Hash][]byte),
		order: make([]common.Hash, 0, 256),
	}
}

// TrieSyncLeafCallback is a callback type invoked when a trie sync reaches a
// leaf node. It's used by state syncing to check if the leaf node requires some
// further data syncing.
type TrieSyncLeafCallback func(leaf []byte, parent common.Hash) error

// TrieSync is the main state trie synchronisation scheduler, which provides yet
// unknown trie hashes to retrieve, accepts node data associated with said hashes
// and reconstructs the trie step by step until all is done.
type TrieSync struct {
	database DatabaseReader           // Persistent database to check for existing entries
	membatch *syncMemBatch            // Memory buffer to avoid frequest database writes
	requests map[common.Hash]*request // Pending requests pertaining to a key hash
	queue    *prque.Prque             // Priority queue with the pending requests
	keccak   hash.Hash                // Keccak256 hasher to verify deliveries with

	nextId uint64 // Identifier component for the priority queue to split between same depths
}

// NewTrieSync creates a new trie data download scheduler.
func NewTrieSync(root common.Hash, database DatabaseReader, callback TrieSyncLeafCallback) *TrieSync {
	ts := &TrieSync{
		database: database,
		membatch: newSyncMemBatch(),
		requests: make(map[common.Hash]*request),
		queue:    prque.New(),
		keccak:   sha3.NewKeccak256(),
	}
	ts.AddSubTrie(root, 0, common.Hash{}, callback)
	return ts
}

// AddSubTrie registers a new trie to the sync code, rooted at the designated parent.
func (s *TrieSync) AddSubTrie(root common.Hash, depth int, parent common.Hash, callback TrieSyncLeafCallback) {
	// Short circuit if the trie is empty or already known
	if root == emptyRoot {
		return
	}
	if _, ok := s.membatch.batch[root]; ok {
		return
	}
	key := root.Bytes()
	blob, _ := s.database.Get(key)
	if local, err := decodeNode(key, blob, 0); local != nil && err == nil {
		return
	}
	// Assemble the new sub-trie sync request
	req := &request{
		hash:     root,
		depth:    depth,
		callback: callback,
	}
	// If this sub-trie has a designated parent, link them together
	if parent != (common.Hash{}) {
		ancestor := s.requests[parent]
		if ancestor == nil {
			panic(fmt.Sprintf("sub-trie ancestor not found: %x", parent))
		}
		ancestor.deps++
		req.parents = append(req.parents, ancestor)
	}
	s.schedule(req, false)
}

// AddRawEntry schedules the direct retrieval of a state entry that should not be
// interpreted as a trie node, but rather accepted and stored into the database
// as is. This method's goal is to support misc state metadata retrievals (e.g.
// contract code).
func (s *TrieSync) AddRawEntry(hash common.Hash, depth int, parent common.Hash) {
	// Short circuit if the entry is empty or already known
	if hash == emptyState {
		return
	}
	if _, ok := s.membatch.batch[hash]; ok {
		return
	}
	if ok, _ := s.database.Has(hash.Bytes()); ok {
		return
	}
	// Assemble the new sub-trie sync request
	req := &request{
		hash:  hash,
		raw:   true,
		depth: depth,
	}
	// If this sub-trie has a designated parent, link them together
	if parent != (common.Hash{}) {
		ancestor := s.requests[parent]
		if ancestor == nil {
			panic(fmt.Sprintf("raw-entry ancestor not found: %x", parent))
		}
		ancestor.deps++
		req.parents = append(req.parents, ancestor)
	}
	s.schedule(req, false)
}

// Missing retrieves the known missing nodes from the trie for retrieval.
func (s *TrieSync) Missing(max int) []common.Hash {
	requests := []common.Hash{}
	for !s.queue.Empty() && (max == 0 || len(requests) < max) {
		hash := s.queue.PopItem().(common.Hash)
		if req := s.requests[hash]; req != nil && req.data == nil {
			requests = append(requests, hash)
		} else {
			fmt.Printf(".")
		}
	}
	return requests
}

// Process injects a batch of retrieved trie data, returning the number of nodes
// and bytes written, along with the hash of the node or sub-trie just processed.
func (s *TrieSync) Process(result *SyncResult) (int, common.StorageSize, common.Hash, error) {
	// If it's a plain or full sub-trie delivery, inject and return
	if len(result.Keys) == 0 && len(result.Proof) == 0 {
		return s.processNode(common.Hash{}, result.Data, false)
	}
	if len(result.Proof) == 0 {
		return s.processLeaves(result.Keys, result.Values)
	}
	// For partial depliveries, expand the keys and iteratively fulfil the sub-trie
	for i, key := range result.Keys {
		result.Keys[i] = keybytesToHex(key)
	}
	return s.processPartialLeaves(result.Keys, result.Values, result.Proof)
}

// processNode verifies and processes a trie node, returning if anything was
// committed and the hash of the node injected.
func (s *TrieSync) processNode(hash common.Hash, blob []byte, ready bool) (int, common.StorageSize, common.Hash, error) {
	// Derive the hash of the result based on its content
	if hash == (common.Hash{}) {
		s.keccak.Reset()
		s.keccak.Write(blob)
		s.keccak.Sum(hash[:0])
	}
	// If the item was not requested, bail out
	request := s.requests[hash]
	if request == nil {
		return 0, 0, hash, nil //ErrNotRequested
	}
	if request.data != nil {
		return 0, 0, hash, ErrAlreadyProcessed
	}
	// If the item is a raw entry request, commit directly
	if request.raw {
		request.data = blob
		items, bytes := s.commit(request)
		return items, bytes, hash, nil
	}
	// Decode and inject into the trie
	node, err := decodeNode(hash[:], blob, 0)
	if err != nil {
		return 0, 0, hash, err
	}
	request.data = blob

	// Create and schedule a request for all the children nodes
	requests, err := s.children(request, node)
	if err != nil {
		return 0, 0, hash, err
	}
	if len(requests) == 0 && request.deps == 0 {
		items, bytes := s.commit(request)
		return items, bytes, hash, nil
	}
	request.deps += len(requests)
	for _, child := range requests {
		s.schedule(child, ready)
	}
	return 0, 0, hash, nil
}

// processLeaves reconstructs a sub-trie from the given key-value pairs, returning
// the number of nodes and bytes written, along with the hash of the sub-trie just
// processed.
func (s *TrieSync) processLeaves(keys [][]byte, values [][]byte) (int, common.StorageSize, common.Hash, error) {
	// Inject all the leaves into a fresh trie and derive it's root hash
	db := ethdb.NewMemDatabase()
	trie, err := New(common.Hash{}, db)
	if err != nil {
		return 0, 0, common.Hash{}, err
	}
	for j := 0; j < len(keys); j++ {
		trie.Update(keys[j], values[j])
	}
	root, err := trie.Commit()
	if err != nil {
		return 0, 0, common.Hash{}, err
	}
	// If the item was not requested, bail out
	request := s.requests[root]
	if request == nil {
		return 0, 0, root, ErrNotRequested
	}
	if request.data != nil {
		return 0, 0, root, ErrAlreadyProcessed
	}
	// Inject all key-values as is and complete the root
	var (
		items int
		bytes common.StorageSize
	)
	it := trie.NodeIterator(nil)
	for it.Next(true) {
		if hash := it.Hash(); hash != (common.Hash{}) {
			blob, _ := db.Get(hash[:])
			count, size, _, err := s.processNode(hash, blob, true)

			items += count
			bytes += size

			if err != nil {
				return items, bytes, root, err
			}
		}
	}
	return items, bytes, root, nil
}

// processPartialLeaves reconstructs a sub-trie from the Merkle proof and the
// available key-value pairs, commiting the available parts and scheduling the
// missing items for future retrival.
func (s *TrieSync) processPartialLeaves(keys [][]byte, values [][]byte, proof [][]byte) (int, common.StorageSize, common.Hash, error) {
	// Derive the hash of the topmost proof
	var root common.Hash

	s.keccak.Reset()
	s.keccak.Write(proof[0])
	s.keccak.Sum(root[:0])

	// If the item was not requested, bail out
	request := s.requests[root]
	if request == nil {
		return 0, 0, root, ErrNotRequested
	}
	if request.data != nil {
		return 0, 0, root, ErrAlreadyProcessed
	}
	// Decode the root node and schedule missing children
	node, err := decodeNode(root[:], proof[0], 0)
	if err != nil {
		return 0, 0, root, err
	}
	request.data = proof[0]

	requests, err := s.children(request, node)
	if err != nil {
		return 0, 0, root, err
	}
	if len(requests) == 0 && request.deps == 0 {
		items, bytes := s.commit(request)
		return items, bytes, root, nil
	}
	request.deps += len(requests)
	for _, child := range requests {
		s.schedule(child, false)
	}
	// Fulfill any children satisfied by the key-value pairs
	switch node := (node).(type) {
	case *shortNode:
		// All keys must have the short node's path as a prefix
		for i, key := range keys {
			if !bytes.HasPrefix(key, node.Key) {
				return 0, 0, root, fmt.Errorf("key mismatch at proof %x", proof[0])
			}
			keys[i] = key[len(node.Key):]
		}
		// Recurse into the subtrie of the short node
		items, bytes, _, err := s.processPartialLeaves(keys, values, proof[1:])
		return items, bytes, root, err

	case *fullNode:
		// Track the number of items and bytes written
		var (
			items int
			bytes common.StorageSize
		)
		// Split up the keyspace between the full node's children
		for i := 0; i < 17; i++ {
			if node.Children[i] != nil {
				// Split off the keyspace for this child
				var split int
				for split < len(keys) && keys[split][0] == byte(i) {
					keys[split] = keys[split][1:]
					split++
				}
				// Only process this child if it's not fully embedded
				if _, ok := node.Children[i].(hashNode); !ok {
					// If we're at the last node, process it as a partial trie
					if split == len(keys) && len(proof) != 1 {
						count, size, _, err := s.processPartialLeaves(keys[:split], values[:split], proof[1:])
						return items + count, bytes + size, root, err
					}
					// Otherwise we have a full sub-trie, parse in its entirety (if not already contained within the full node)
					count, size, _, err := s.processLeaves(keys[:split], values[:split])

					items += count
					bytes += size

					if err != nil {
						return items, bytes, root, err
					}
				}
				keys = keys[split:]
				values = values[split:]
			}
		}
		return items, bytes, root, nil
	}
	return 0, 0, root, fmt.Errorf("unexpected node type: %v", reflect.TypeOf(node))
}

// Commit flushes the data stored in the internal membatch out to persistent
// storage, returning th enumber of items written and any occurred error.
func (s *TrieSync) Commit(dbw DatabaseWriter) (int, error) {
	// Dump the membatch into a database dbw
	for i, key := range s.membatch.order {
		if err := dbw.Put(key[:], s.membatch.batch[key]); err != nil {
			return i, err
		}
	}
	written := len(s.membatch.order)

	// Drop the membatch data and return
	s.membatch = newSyncMemBatch()
	return written, nil
}

// Pending returns the number of state entries currently pending for download.
func (s *TrieSync) Pending() int {
	return len(s.requests)
}

// schedule inserts a new state retrieval request into the fetch queue. If there
// is already a pending request for this node, the new request will be discarded
// and only a parent reference added to the old one.
func (s *TrieSync) schedule(req *request, ready bool) {
	// If we're already requesting this node, add a new reference and stop
	if old, ok := s.requests[req.hash]; ok {
		old.parents = append(old.parents, req.parents...)
		return
	}
	// Schedule the request for future retrieval
	if !ready {
		s.queue.Push(req.hash, float32(req.depth)*math.MaxUint64+float32(math.MaxUint64-atomic.AddUint64(&s.nextId, 1)))
	}
	s.requests[req.hash] = req
}

// children retrieves all the missing children of a state trie entry for future
// retrieval scheduling.
func (s *TrieSync) children(req *request, object node) ([]*request, error) {
	// Gather all the children of the node, irrelevant whether known or not
	type child struct {
		node  node
		depth int
	}
	children := []child{}

	switch node := (object).(type) {
	case *shortNode:
		children = []child{{
			node:  node.Val,
			depth: req.depth + len(node.Key),
		}}
	case *fullNode:
		for i := 0; i < 17; i++ {
			if node.Children[i] != nil {
				children = append(children, child{
					node:  node.Children[i],
					depth: req.depth + 1,
				})
			}
		}
	default:
		panic(fmt.Sprintf("unknown node: %+v", node))
	}
	// Iterate over the children, and request all unknown ones
	requests := make([]*request, 0, len(children))
	for _, child := range children {
		// Notify any external watcher of a new key/value node
		if req.callback != nil {
			if node, ok := (child.node).(valueNode); ok {
				if err := req.callback(node, req.hash); err != nil {
					return nil, err
				}
			}
		}
		// If the child references another node, resolve or schedule
		if node, ok := (child.node).(hashNode); ok {
			// Try to resolve the node from the local database
			hash := common.BytesToHash(node)
			if _, ok := s.membatch.batch[hash]; ok {
				continue
			}
			if ok, _ := s.database.Has(node); ok {
				continue
			}
			// Locally unknown node, schedule for retrieval
			requests = append(requests, &request{
				hash:     hash,
				parents:  []*request{req},
				depth:    child.depth,
				callback: req.callback,
			})
		}
	}
	return requests, nil
}

// commit finalizes a retrieval request and stores it into the membatch. If any
// of the referencing parent requests complete due to this commit, they are also
// committed themselves. The method returns the number of state items written to
// the membatch as well as their total data size.
func (s *TrieSync) commit(req *request) (int, common.StorageSize) {
	var (
		items = 1
		bytes = common.StorageSize(len(req.data))
	)
	// Write the node content to the membatch
	s.commitEntry(req.hash, req.data)
	delete(s.requests, req.hash)

	// Check all parents for completion
	for _, parent := range req.parents {
		parent.deps--
		if parent.deps == 0 {
			count, size := s.commit(parent)

			items += count
			bytes += size
		}
	}
	return items, bytes
}

// commitEntry injects a raw database entry into the memory batch to be flushed
// out at a later point into the real database.
func (s *TrieSync) commitEntry(key common.Hash, blob []byte) {
	s.membatch.batch[key] = blob
	s.membatch.order = append(s.membatch.order, key)
}
