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
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
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
	code bool        // Whether this is a code entry

	parents []*request // Parent state nodes referencing this entry (notify all upon completion)
	depth   int        // Depth level within the trie the node is located to prioritise DFS
	deps    int        // Number of dependencies before allowed to commit this node

	callback LeafCallback // Callback to invoke if a leaf node it reached on this branch
}

// SyncResult is a response with requested data along with it's hash.
type SyncResult struct {
	Hash common.Hash // Hash of the originally unknown trie node
	Data []byte      // Data content of the retrieved node
}

// syncMemBatch is an in-memory buffer of successfully downloaded but not yet
// persisted data items.
type syncMemBatch struct {
	nodes map[common.Hash][]byte // In-memory membatch of recently completed nodes
	codes map[common.Hash][]byte // In-memory membatch of recently completed codes
}

// newSyncMemBatch allocates a new memory-buffer for not-yet persisted trie nodes.
func newSyncMemBatch() *syncMemBatch {
	return &syncMemBatch{
		nodes: make(map[common.Hash][]byte),
		codes: make(map[common.Hash][]byte),
	}
}

// hasNode reports the trie node with specific hash is already cached.
func (batch *syncMemBatch) hasNode(hash common.Hash) bool {
	_, ok := batch.nodes[hash]
	return ok
}

// hasCode reports the contract code with specific hash is already cached.
func (batch *syncMemBatch) hasCode(hash common.Hash) bool {
	_, ok := batch.codes[hash]
	return ok
}

// Sync is the main state trie synchronisation scheduler, which provides yet
// unknown trie hashes to retrieve, accepts node data associated with said hashes
// and reconstructs the trie step by step until all is done.
type Sync struct {
	database ethdb.KeyValueReader     // Persistent database to check for existing entries
	membatch *syncMemBatch            // Memory buffer to avoid frequent database writes
	nodeReqs map[common.Hash]*request // Pending requests pertaining to a trie node hash
	codeReqs map[common.Hash]*request // Pending requests pertaining to a code hash
	queue    *prque.Prque             // Priority queue with the pending requests
	bloom    *SyncBloom               // Bloom filter for fast state existence checks
}

// NewSync creates a new trie data download scheduler.
func NewSync(root common.Hash, database ethdb.KeyValueReader, callback LeafCallback, bloom *SyncBloom) *Sync {
	ts := &Sync{
		database: database,
		membatch: newSyncMemBatch(),
		nodeReqs: make(map[common.Hash]*request),
		codeReqs: make(map[common.Hash]*request),
		queue:    prque.New(nil),
		bloom:    bloom,
	}
	ts.AddSubTrie(root, 0, common.Hash{}, callback)
	return ts
}

// AddSubTrie registers a new trie to the sync code, rooted at the designated parent.
func (s *Sync) AddSubTrie(root common.Hash, depth int, parent common.Hash, callback LeafCallback) {
	// Short circuit if the trie is empty or already known
	if root == emptyRoot {
		return
	}
	if s.membatch.hasNode(root) {
		return
	}
	if s.bloom == nil || s.bloom.Contains(root[:]) {
		// Bloom filter says this might be a duplicate, double check.
		// If database says yes, then at least the trie node is present
		// and we hold the assumption that it's NOT legacy contract code.
		blob := rawdb.ReadTrieNode(s.database, root)
		if len(blob) > 0 {
			return
		}
		// False positive, bump fault meter
		bloomFaultMeter.Mark(1)
	}
	// Assemble the new sub-trie sync request
	req := &request{
		hash:     root,
		depth:    depth,
		callback: callback,
	}
	// If this sub-trie has a designated parent, link them together
	if parent != (common.Hash{}) {
		ancestor := s.nodeReqs[parent]
		if ancestor == nil {
			panic(fmt.Sprintf("sub-trie ancestor not found: %x", parent))
		}
		ancestor.deps++
		req.parents = append(req.parents, ancestor)
	}
	s.schedule(req)
}

// AddCodeEntry schedules the direct retrieval of a contract code that should not
// be interpreted as a trie node, but rather accepted and stored into the database
// as is.
func (s *Sync) AddCodeEntry(hash common.Hash, depth int, parent common.Hash) {
	// Short circuit if the entry is empty or already known
	if hash == emptyState {
		return
	}
	if s.membatch.hasCode(hash) {
		return
	}
	if s.bloom == nil || s.bloom.Contains(hash[:]) {
		// Bloom filter says this might be a duplicate, double check.
		// If database says yes, the blob is present for sure.
		// Note we only check the existence with new code scheme, fast
		// sync is expected to run with a fresh new node. Even there
		// exists the code with legacy format, fetch and store with
		// new scheme anyway.
		if blob := rawdb.ReadCodeWithPrefix(s.database, hash); len(blob) > 0 {
			return
		}
		// False positive, bump fault meter
		bloomFaultMeter.Mark(1)
	}
	// Assemble the new sub-trie sync request
	req := &request{
		hash:  hash,
		code:  true,
		depth: depth,
	}
	// If this sub-trie has a designated parent, link them together
	if parent != (common.Hash{}) {
		ancestor := s.nodeReqs[parent] // the parent of codereq can ONLY be nodereq
		if ancestor == nil {
			panic(fmt.Sprintf("raw-entry ancestor not found: %x", parent))
		}
		ancestor.deps++
		req.parents = append(req.parents, ancestor)
	}
	s.schedule(req)
}

// Missing retrieves the known missing nodes from the trie for retrieval.
func (s *Sync) Missing(max int) []common.Hash {
	var requests []common.Hash
	for !s.queue.Empty() && (max == 0 || len(requests) < max) {
		requests = append(requests, s.queue.PopItem().(common.Hash))
	}
	return requests
}

// Process injects the received data for requested item. Note it can
// happpen that the single response commits two pending requests(e.g.
// there are two requests one for code and one for node but the hash
// is same). In this case the second response for the same hash will
// be treated as "non-requested" item or "already-processed" item but
// there is no downside.
func (s *Sync) Process(result SyncResult) error {
	// If the item was not requested either for code or node, bail out
	if s.nodeReqs[result.Hash] == nil && s.codeReqs[result.Hash] == nil {
		return ErrNotRequested
	}
	// There is an pending code request for this data, commit directly
	var filled bool
	if req := s.codeReqs[result.Hash]; req != nil && req.data == nil {
		filled = true
		req.data = result.Data
		s.commit(req)
	}
	// There is an pending node request for this data, fill it.
	if req := s.nodeReqs[result.Hash]; req != nil && req.data == nil {
		filled = true
		// Decode the node data content and update the request
		node, err := decodeNode(result.Hash[:], result.Data)
		if err != nil {
			return err
		}
		req.data = result.Data

		// Create and schedule a request for all the children nodes
		requests, err := s.children(req, node)
		if err != nil {
			return err
		}
		if len(requests) == 0 && req.deps == 0 {
			s.commit(req)
		} else {
			req.deps += len(requests)
			for _, child := range requests {
				s.schedule(child)
			}
		}
	}
	if !filled {
		return ErrAlreadyProcessed
	}
	return nil
}

// Commit flushes the data stored in the internal membatch out to persistent
// storage, returning any occurred error.
func (s *Sync) Commit(dbw ethdb.Batch) error {
	// Dump the membatch into a database dbw
	for key, value := range s.membatch.nodes {
		rawdb.WriteTrieNode(dbw, key, value)
		s.bloom.Add(key[:])
	}
	for key, value := range s.membatch.codes {
		rawdb.WriteCode(dbw, key, value)
		s.bloom.Add(key[:])
	}
	// Drop the membatch data and return
	s.membatch = newSyncMemBatch()
	return nil
}

// Pending returns the number of state entries currently pending for download.
func (s *Sync) Pending() int {
	return len(s.nodeReqs) + len(s.codeReqs)
}

// schedule inserts a new state retrieval request into the fetch queue. If there
// is already a pending request for this node, the new request will be discarded
// and only a parent reference added to the old one.
func (s *Sync) schedule(req *request) {
	var reqset = s.nodeReqs
	if req.code {
		reqset = s.codeReqs
	}
	// If we're already requesting this node, add a new reference and stop
	if old, ok := reqset[req.hash]; ok {
		old.parents = append(old.parents, req.parents...)
		return
	}
	reqset[req.hash] = req

	// Schedule the request for future retrieval. This queue is shared
	// by both node requests and code requests. It can happen that there
	// is a trie node and code has same hash. In this case two elements
	// with same hash and same or different depth will be pushed. But it's
	// ok the worst case is the second response will be treated as duplicated.
	s.queue.Push(req.hash, int64(req.depth))
}

// children retrieves all the missing children of a state trie entry for future
// retrieval scheduling.
func (s *Sync) children(req *request, object node) ([]*request, error) {
	// Gather all the children of the node, irrelevant whether known or not
	type child struct {
		node  node
		depth int
	}
	var children []child

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
			if s.membatch.hasNode(hash) {
				continue
			}
			if s.bloom == nil || s.bloom.Contains(node) {
				// Bloom filter says this might be a duplicate, double check.
				// If database says yes, then at least the trie node is present
				// and we hold the assumption that it's NOT legacy contract code.
				if blob := rawdb.ReadTrieNode(s.database, common.BytesToHash(node)); len(blob) > 0 {
					continue
				}
				// False positive, bump fault meter
				bloomFaultMeter.Mark(1)
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
// committed themselves.
func (s *Sync) commit(req *request) (err error) {
	// Write the node content to the membatch
	if req.code {
		s.membatch.codes[req.hash] = req.data
		delete(s.codeReqs, req.hash)
	} else {
		s.membatch.nodes[req.hash] = req.data
		delete(s.nodeReqs, req.hash)
	}
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
