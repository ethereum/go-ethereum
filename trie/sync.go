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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

// ErrNotRequested is returned by the trie sync when it's requested to process a
// node it did not request.
var ErrNotRequested = errors.New("not requested")

// request represents a scheduled or already in-flight state retrieval request.
type request struct {
	hash   common.Hash // Hash of the node data content to retrieve
	data   []byte      // Data content of the node, cached until all subtrees complete
	object *node       // Target node to populate with retrieved data (hashnode originally)

	parents []*request // Parent state nodes referencing this entry (notify all upon completion)
	depth   int        // Depth level within the trie the node is located to prioritise DFS
	deps    int        // Number of dependencies before allowed to commit this node

	callback TrieSyncLeafCallback // Callback to invoke if a leaf node it reached on this branch
}

// SyncResult is a simple list to return missing nodes along with their request
// hashes.
type SyncResult struct {
	Hash common.Hash // Hash of the originally unknown trie node
	Data []byte      // Data content of the retrieved node
}

// TrieSyncLeafCallback is a callback type invoked when a trie sync reaches a
// leaf node. It's used by state syncing to check if the leaf node requires some
// further data syncing.
type TrieSyncLeafCallback func(leaf []byte, parent common.Hash) error

// TrieSync is the main state trie synchronisation scheduler, which provides yet
// unknown trie hashes to retrieve, accepts node data associated with said hashes
// and reconstructs the trie step by step until all is done.
type TrieSync struct {
	database ethdb.Database           // State database for storing all the assembled node data
	requests map[common.Hash]*request // Pending requests pertaining to a key hash
	queue    *prque.Prque             // Priority queue with the pending requests
}

// NewTrieSync creates a new trie data download scheduler.
func NewTrieSync(root common.Hash, database ethdb.Database, callback TrieSyncLeafCallback) *TrieSync {
	ts := &TrieSync{
		database: database,
		requests: make(map[common.Hash]*request),
		queue:    prque.New(),
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
	key := root.Bytes()
	blob, _ := s.database.Get(key)
	if local, err := decodeNode(key, blob); local != nil && err == nil {
		return
	}
	// Assemble the new sub-trie sync request
	node := node(hashNode(root.Bytes()))
	req := &request{
		object:   &node,
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
	s.schedule(req)
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
	if blob, _ := s.database.Get(hash.Bytes()); blob != nil {
		return
	}
	// Assemble the new sub-trie sync request
	req := &request{
		hash:  hash,
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
	s.schedule(req)
}

// Missing retrieves the known missing nodes from the trie for retrieval.
func (s *TrieSync) Missing(max int) []common.Hash {
	requests := []common.Hash{}
	for !s.queue.Empty() && (max == 0 || len(requests) < max) {
		requests = append(requests, s.queue.PopItem().(common.Hash))
	}
	return requests
}

// Process injects a batch of retrieved trie nodes data.
func (s *TrieSync) Process(results []SyncResult) (int, error) {
	for i, item := range results {
		// If the item was not requested, bail out
		request := s.requests[item.Hash]
		if request == nil {
			return i, ErrNotRequested
		}
		// If the item is a raw entry request, commit directly
		if request.object == nil {
			request.data = item.Data
			s.commit(request, nil)
			continue
		}
		// Decode the node data content and update the request
		node, err := decodeNode(item.Hash[:], item.Data)
		if err != nil {
			return i, err
		}
		*request.object = node
		request.data = item.Data

		// Create and schedule a request for all the children nodes
		requests, err := s.children(request)
		if err != nil {
			return i, err
		}
		if len(requests) == 0 && request.deps == 0 {
			s.commit(request, nil)
			continue
		}
		request.deps += len(requests)
		for _, child := range requests {
			s.schedule(child)
		}
	}
	return 0, nil
}

// Pending returns the number of state entries currently pending for download.
func (s *TrieSync) Pending() int {
	return len(s.requests)
}

// schedule inserts a new state retrieval request into the fetch queue. If there
// is already a pending request for this node, the new request will be discarded
// and only a parent reference added to the old one.
func (s *TrieSync) schedule(req *request) {
	// If we're already requesting this node, add a new reference and stop
	if old, ok := s.requests[req.hash]; ok {
		old.parents = append(old.parents, req.parents...)
		return
	}
	// Schedule the request for future retrieval
	s.queue.Push(req.hash, float32(req.depth))
	s.requests[req.hash] = req
}

// children retrieves all the missing children of a state trie entry for future
// retrieval scheduling.
func (s *TrieSync) children(req *request) ([]*request, error) {
	// Gather all the children of the node, irrelevant whether known or not
	type child struct {
		node  *node
		depth int
	}
	children := []child{}

	switch node := (*req.object).(type) {
	case *shortNode:
		node = node.copy() // Prevents linking all downloaded nodes together.
		children = []child{{
			node:  &node.Val,
			depth: req.depth + len(node.Key),
		}}
	case *fullNode:
		node = node.copy()
		for i := 0; i < 17; i++ {
			if node.Children[i] != nil {
				children = append(children, child{
					node:  &node.Children[i],
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
			if node, ok := (*child.node).(valueNode); ok {
				if err := req.callback(node, req.hash); err != nil {
					return nil, err
				}
			}
		}
		// If the child references another node, resolve or schedule
		if node, ok := (*child.node).(hashNode); ok {
			// Try to resolve the node from the local database
			blob, _ := s.database.Get(node)
			if local, err := decodeNode(node[:], blob); local != nil && err == nil {
				*child.node = local
				continue
			}
			// Locally unknown node, schedule for retrieval
			requests = append(requests, &request{
				object:   child.node,
				hash:     common.BytesToHash(node),
				parents:  []*request{req},
				depth:    child.depth,
				callback: req.callback,
			})
		}
	}
	return requests, nil
}

// commit finalizes a retrieval request and stores it into the database. If any
// of the referencing parent requests complete due to this commit, they are also
// committed themselves.
func (s *TrieSync) commit(req *request, batch ethdb.Batch) (err error) {
	// Create a new batch if none was specified
	if batch == nil {
		batch = s.database.NewBatch()
		defer func() {
			err = batch.Write()
		}()
	}
	// Write the node content to disk
	if err := batch.Put(req.hash[:], req.data); err != nil {
		return err
	}
	delete(s.requests, req.hash)

	// Check all parents for completion
	for _, parent := range req.parents {
		parent.deps--
		if parent.deps == 0 {
			if err := s.commit(parent, batch); err != nil {
				return err
			}
		}
	}
	return nil
}
