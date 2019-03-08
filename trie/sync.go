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
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/ethdb"
)

// ErrNotRequested is returned by the trie sync when it's requested to process a
// node it did not request.
var ErrNotRequested = errors.New("not requested")

// ErrAlreadyProcessed is returned by the trie sync when it's requested to process a
// node it already processed previously.
var ErrAlreadyProcessed = errors.New("already processed")

// codePrefix is the database key prefix used to store raw trie entries (code).
var codePrefix = []byte("c")

// request represents a scheduled or already in-flight state retrieval request.
type request struct {
	key  string // Key of the node data content to retrieve
	path []byte // Merkle-Patricia path to track sub-trie ownership
	data []byte // Data content of the node, cached until all subtrees complete
	raw  bool   // Whether this is a raw entry (code) or a trie node

	parents []*request // Parent state nodes referencing this entry (notify all upon completion)
	depth   int        // Depth level within the trie the node is located to prioritise DFS
	deps    int        // Number of dependencies before allowed to commit this node

	callback LeafCallback // Callback to invoke if a leaf node it reached on this branch
}

// SplitNodeKey interprets the specified key, splitting it into an owner:hash
// tuple, also specifying whether the key represents a bytecode.
func SplitNodeKey(key string) (common.Hash, common.Hash, bool) {
	// Figure out if this key represents a byte code or not
	var code bool

	if len(key)%2 == 1 {
		if key[0] != codePrefix[0] {
			panic(fmt.Sprintf("invalid node prefix: %x", key[0]))
		}
		code, key = true, key[1:]
	}
	// Split the key into an [owner]:hash tuple and return
	owner, hash := splitNodeKey(key)
	return owner, hash, code
}

// SyncResult is a simple struct to return missing nodes along with their request
// keys.
type SyncResult struct {
	Key  string // Key of the originally unknown trie node
	Data []byte // Data content of the retrieved node
}

// syncMemBatch is an in-memory buffer of successfully downloaded but not yet
// persisted data items.
type syncMemBatch struct {
	batch map[string][]byte // In-memory membatch of recently completed items
	order []string          // Order of completion to prevent out-of-order data loss
}

// newSyncMemBatch allocates a new memory-buffer for not-yet persisted trie nodes.
func newSyncMemBatch() *syncMemBatch {
	return &syncMemBatch{
		batch: make(map[string][]byte),
		order: make([]string, 0, 256),
	}
}

// Sync is the main state trie synchronisation scheduler, which provides yet
// unknown trie hashes to retrieve, accepts node data associated with said hashes
// and reconstructs the trie step by step until all is done.
type Sync struct {
	database ethdb.Reader        // Persistent database to check for existing entries
	membatch *syncMemBatch       // Memory buffer to avoid frequent database writes
	requests map[string]*request // Pending requests pertaining to a key hash
	queue    *prque.Prque        // Priority queue with the pending requests
}

// NewSync creates a new trie data download scheduler.
func NewSync(root common.Hash, database ethdb.Reader, callback LeafCallback) *Sync {
	ts := &Sync{
		database: database,
		membatch: newSyncMemBatch(),
		requests: make(map[string]*request),
		queue:    prque.New(nil),
	}
	ts.AddSubTrie(common.Hash{}, root, 0, common.Hash{}, callback)
	return ts
}

// AddSubTrie registers a new trie to the sync code, rooted at the designated
// parent for completion tracking.
//
// Note, the root has an owner field for tracking which account (hash/path) it
// belongs to whereas parent does not. The reason is that Ethereum only ever
// supports 2 layers of tries (account -> storage), so a sub-trie will never
// ever have a parent who's owner is not the nil hash.
func (s *Sync) AddSubTrie(owner, root common.Hash, depth int, parent common.Hash, callback LeafCallback) {
	// Short circuit if the trie is empty or already known	_, hash := splitNodeKey(root)
	if root == emptyRoot {
		return
	}
	key := makeNodeKey(owner, root)
	if _, ok := s.membatch.batch[key]; ok {
		return
	}
	blob, _ := s.database.Get([]byte(key))
	if local, err := decodeNode(root[:], blob, 0); local != nil && err == nil {
		return
	}
	// Assemble the new sub-trie sync request
	req := &request{
		key:      key,
		depth:    depth,
		callback: callback,
	}
	// If this sub-trie has a designated parent, link them together
	if (parent != common.Hash{}) {
		ancestor := s.requests[makeNodeKey(common.Hash{}, parent)]
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
//
// Note, neither the hash, nor the parent has an owner specified. The reason is
// that in Ethereum, only bytecode is stored as a raw-entry referenced by the
// trie, but that is deduplicated to prevent attacks. The parent is always an
// account, so we know it's owner is the nil hash.
func (s *Sync) AddRawEntry(hash common.Hash, depth int, parent common.Hash) {
	// Short circuit if the entry is empty or already known
	if hash == emptyState {
		return
	}
	var (
		keyRaw = append(codePrefix, hash[:]...)
		keyStr = string(keyRaw)
	)
	if _, ok := s.membatch.batch[keyStr]; ok {
		return
	}
	if ok, _ := s.database.Has(keyRaw); ok {
		return
	}
	// Assemble the new sub-trie sync request
	req := &request{
		key:   keyStr,
		raw:   true,
		depth: depth,
	}
	// If this sub-trie has a designated parent, link them together
	if (parent != common.Hash{}) {
		ancestor := s.requests[makeNodeKey(common.Hash{}, parent)]
		if ancestor == nil {
			panic(fmt.Sprintf("raw-entry ancestor not found: %x", parent))
		}
		ancestor.deps++
		req.parents = append(req.parents, ancestor)
	}
	s.schedule(req)
}

// Missing retrieves the known missing nodes from the trie for retrieval.
//
// The returned strings can represent three different things:
//   - `hash` if it's an account trie node
//   - `owner + hash` if it's a storage trie node
//   - `'c' + hash` if it's an account bydecode node
//
// Use trie.SplitNodeKey to get an accurate interpretation of what exactly a
// returned key means.
func (s *Sync) Missing(max int) []string {
	var requests []string
	for !s.queue.Empty() && (max == 0 || len(requests) < max) {
		requests = append(requests, s.queue.PopItem().(string))
	}
	return requests
}

// Process injects a batch of retrieved trie nodes data, returning if something
// was committed to the database and also the index of an entry if processing of
// it failed.
func (s *Sync) Process(results []SyncResult) (bool, int, error) {
	committed := false

	for i, item := range results {
		// If the item was not requested, bail out
		request := s.requests[item.Key]
		if request == nil {
			return committed, i, ErrNotRequested
		}
		if request.data != nil {
			return committed, i, ErrAlreadyProcessed
		}
		// If the item is a raw entry request, commit directly
		if request.raw {
			request.data = item.Data
			s.commit(request)
			committed = true
			continue
		}
		// Decode the node data content and update the request
		_, hash := splitNodeKey(item.Key)
		node, err := decodeNode(hash[:], item.Data, 0)
		if err != nil {
			return committed, i, err
		}
		request.data = item.Data

		// Create and schedule a request for all the children nodes
		requests, err := s.children(request, node)
		if err != nil {
			return committed, i, err
		}
		if len(requests) == 0 && request.deps == 0 {
			s.commit(request)
			committed = true
			continue
		}
		request.deps += len(requests)
		for _, child := range requests {
			s.schedule(child)
		}
	}
	return committed, 0, nil
}

// Commit flushes the data stored in the internal membatch out to persistent
// storage, returning the number of items written and any occurred error.
func (s *Sync) Commit(dbw ethdb.Writer) (int, error) {
	// Dump the membatch into a database dbw
	for i, key := range s.membatch.order {
		if err := dbw.Put([]byte(key), s.membatch.batch[key]); err != nil {
			return i, err
		}
	}
	written := len(s.membatch.order)

	// Drop the membatch data and return
	s.membatch = newSyncMemBatch()
	return written, nil
}

// Pending returns the number of state entries currently pending for download.
func (s *Sync) Pending() int {
	return len(s.requests)
}

// schedule inserts a new state retrieval request into the fetch queue. If there
// is already a pending request for this node, the new request will be discarded
// and only a parent reference added to the old one.
func (s *Sync) schedule(req *request) {
	// If we're already requesting this node, add a new reference and stop
	if old, ok := s.requests[req.key]; ok {
		old.parents = append(old.parents, req.parents...)
		return
	}
	// Schedule the request for future retrieval
	s.queue.Push(req.key, int64(req.depth))
	s.requests[req.key] = req
}

// children retrieves all the missing children of a state trie entry for future
// retrieval scheduling.
func (s *Sync) children(req *request, object node) ([]*request, error) {
	// Gather all the children of the node, irrelevant whether known or not
	type child struct {
		path  []byte
		node  node
		depth int
	}
	var children []child

	switch node := (object).(type) {
	case *shortNode:
		children = []child{{
			path:  append(common.CopyBytes(req.path), node.Key...),
			node:  node.Val,
			depth: req.depth + len(node.Key),
		}}
	case *fullNode:
		for i := 0; i < 17; i++ {
			if node.Children[i] != nil {
				children = append(children, child{
					path:  append(common.CopyBytes(req.path), byte(i)),
					node:  node.Children[i],
					depth: req.depth + 1,
				})
			}
		}
	default:
		panic(fmt.Sprintf("unknown node: %+v", node))
	}
	// Iterate over the children, and request all unknown ones
	owner, hash := splitNodeKey(req.key)

	requests := make([]*request, 0, len(children))
	for _, child := range children {
		// Notify any external watcher of a new key/value node
		if req.callback != nil {
			if node, ok := (child.node).(valueNode); ok {
				if len(child.path) != 65 {
					panic(fmt.Sprintf("invalid child path (len %d): %x", len(child.path), child.path))
				}
				owner := common.BytesToHash(hexToKeybytes(child.path[:64]))
				if err := req.callback(owner, node, hash); err != nil {
					return nil, err
				}
			}
		}
		// If the child references another node, resolve or schedule
		if node, ok := (child.node).(hashNode); ok {
			// Try to resolve the node from the local database
			key := makeNodeKey(owner, common.BytesToHash(node))
			if _, ok := s.membatch.batch[key]; ok {
				continue
			}
			if ok, _ := s.database.Has([]byte(key)); ok {
				continue
			}
			// Locally unknown node, schedule for retrieval
			requests = append(requests, &request{
				key:      key,
				path:     child.path,
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
// committed themselves.
func (s *Sync) commit(req *request) (err error) {
	// Write the node content to the membatch
	s.membatch.batch[req.key] = req.data
	s.membatch.order = append(s.membatch.order, req.key)

	delete(s.requests, req.key)

	// Check all parents for completion
	for _, parent := range req.parents {
		parent.deps--
		if parent.deps == 0 {
			if err := s.commit(parent); err != nil {
				return err
			}
		}
	}
	return nil
}
