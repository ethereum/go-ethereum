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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// ErrNotRequested is returned by the trie sync when it's requested to process a
// node it did not request.
var ErrNotRequested = errors.New("not requested")

// ErrAlreadyProcessed is returned by the trie sync when it's requested to process a
// node it already processed previously.
var ErrAlreadyProcessed = errors.New("already processed")

// maxFetchesPerDepth is the maximum number of pending trie nodes per depth. The
// role of this value is to limit the number of trie nodes that get expanded in
// memory if the node was configured with a significant number of peers.
const maxFetchesPerDepth = 16384

// SyncPath is a path tuple identifying a particular trie node either in a single
// trie (account) or a layered trie (account -> storage).
//
// Content wise the tuple either has 1 element if it addresses a node in a single
// trie or 2 elements if it addresses a node in a stacked trie.
//
// To support aiming arbitrary trie nodes, the path needs to support odd nibble
// lengths. To avoid transferring expanded hex form over the network, the last
// part of the tuple (which needs to index into the middle of a trie) is compact
// encoded. In case of a 2-tuple, the first item is always 32 bytes so that is
// simple binary encoded.
//
// Examples:
//   - Path 0x9  -> {0x19}
//   - Path 0x99 -> {0x0099}
//   - Path 0x01234567890123456789012345678901012345678901234567890123456789019  -> {0x0123456789012345678901234567890101234567890123456789012345678901, 0x19}
//   - Path 0x012345678901234567890123456789010123456789012345678901234567890199 -> {0x0123456789012345678901234567890101234567890123456789012345678901, 0x0099}
type SyncPath [][]byte

// NewSyncPath converts an expanded trie path from nibble form into a compact
// version that can be sent over the network.
func NewSyncPath(path []byte) SyncPath {
	// If the hash is from the account trie, append a single item, if it
	// is from the a storage trie, append a tuple. Note, the length 64 is
	// clashing between account leaf and storage root. It's fine though
	// because having a trie node at 64 depth means a hash collision was
	// found and we're long dead.
	if len(path) < 64 {
		return SyncPath{hexToCompact(path)}
	}
	return SyncPath{hexToKeybytes(path[:64]), hexToCompact(path[64:])}
}

// nodeRequest represents a scheduled or already in-flight trie node retrieval request.
type nodeRequest struct {
	hash common.Hash // Hash of the trie node to retrieve
	path []byte      // Merkle path leading to this node for prioritization
	data []byte      // Data content of the node, cached until all subtrees complete

	parent   *nodeRequest // Parent state node referencing this entry
	deps     int          // Number of dependencies before allowed to commit this node
	callback LeafCallback // Callback to invoke if a leaf node it reached on this branch
}

// codeRequest represents a scheduled or already in-flight bytecode retrieval request.
type codeRequest struct {
	hash    common.Hash    // Hash of the contract bytecode to retrieve
	path    []byte         // Merkle path leading to this node for prioritization
	data    []byte         // Data content of the node, cached until all subtrees complete
	parents []*nodeRequest // Parent state nodes referencing this entry (notify all upon completion)
}

// NodeSyncResult is a response with requested trie node along with its node path.
type NodeSyncResult struct {
	Path string // Path of the originally unknown trie node
	Data []byte // Data content of the retrieved trie node
}

// CodeSyncResult is a response with requested bytecode along with its hash.
type CodeSyncResult struct {
	Hash common.Hash // Hash the originally unknown bytecode
	Data []byte      // Data content of the retrieved bytecode
}

// syncMemBatch is an in-memory buffer of successfully downloaded but not yet
// persisted data items.
type syncMemBatch struct {
	nodes  map[string][]byte      // In-memory membatch of recently completed nodes
	hashes map[string]common.Hash // Hashes of recently completed nodes
	codes  map[common.Hash][]byte // In-memory membatch of recently completed codes
	size   uint64                 // Estimated batch-size of in-memory data.
}

// newSyncMemBatch allocates a new memory-buffer for not-yet persisted trie nodes.
func newSyncMemBatch() *syncMemBatch {
	return &syncMemBatch{
		nodes:  make(map[string][]byte),
		hashes: make(map[string]common.Hash),
		codes:  make(map[common.Hash][]byte),
	}
}

// hasNode reports the trie node with specific path is already cached.
func (batch *syncMemBatch) hasNode(path []byte) bool {
	_, ok := batch.nodes[string(path)]
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
	database ethdb.KeyValueReader         // Persistent database to check for existing entries
	membatch *syncMemBatch                // Memory buffer to avoid frequent database writes
	nodeReqs map[string]*nodeRequest      // Pending requests pertaining to a trie node path
	codeReqs map[common.Hash]*codeRequest // Pending requests pertaining to a code hash
	queue    *prque.Prque                 // Priority queue with the pending requests
	fetches  map[int]int                  // Number of active fetches per trie node depth
}

// NewSync creates a new trie data download scheduler.
func NewSync(root common.Hash, database ethdb.KeyValueReader, callback LeafCallback) *Sync {
	ts := &Sync{
		database: database,
		membatch: newSyncMemBatch(),
		nodeReqs: make(map[string]*nodeRequest),
		codeReqs: make(map[common.Hash]*codeRequest),
		queue:    prque.New(nil),
		fetches:  make(map[int]int),
	}
	ts.AddSubTrie(root, nil, common.Hash{}, nil, callback)
	return ts
}

// AddSubTrie registers a new trie to the sync code, rooted at the designated
// parent for completion tracking. The given path is a unique node path in
// hex format and contain all the parent path if it's layered trie node.
func (s *Sync) AddSubTrie(root common.Hash, path []byte, parent common.Hash, parentPath []byte, callback LeafCallback) {
	// Short circuit if the trie is empty or already known
	if root == emptyRoot {
		return
	}
	if s.membatch.hasNode(path) {
		return
	}
	if rawdb.HasTrieNode(s.database, root) {
		return
	}
	// Assemble the new sub-trie sync request
	req := &nodeRequest{
		hash:     root,
		path:     path,
		callback: callback,
	}
	// If this sub-trie has a designated parent, link them together
	if parent != (common.Hash{}) {
		ancestor := s.nodeReqs[string(parentPath)]
		if ancestor == nil {
			panic(fmt.Sprintf("sub-trie ancestor not found: %x", parent))
		}
		ancestor.deps++
		req.parent = ancestor
	}
	s.scheduleNodeRequest(req)
}

// AddCodeEntry schedules the direct retrieval of a contract code that should not
// be interpreted as a trie node, but rather accepted and stored into the database
// as is.
func (s *Sync) AddCodeEntry(hash common.Hash, path []byte, parent common.Hash, parentPath []byte) {
	// Short circuit if the entry is empty or already known
	if hash == emptyState {
		return
	}
	if s.membatch.hasCode(hash) {
		return
	}
	// If database says duplicate, the blob is present for sure.
	// Note we only check the existence with new code scheme, fast
	// sync is expected to run with a fresh new node. Even there
	// exists the code with legacy format, fetch and store with
	// new scheme anyway.
	if rawdb.HasCodeWithPrefix(s.database, hash) {
		return
	}
	// Assemble the new sub-trie sync request
	req := &codeRequest{
		path: path,
		hash: hash,
	}
	// If this sub-trie has a designated parent, link them together
	if parent != (common.Hash{}) {
		ancestor := s.nodeReqs[string(parentPath)] // the parent of codereq can ONLY be nodereq
		if ancestor == nil {
			panic(fmt.Sprintf("raw-entry ancestor not found: %x", parent))
		}
		ancestor.deps++
		req.parents = append(req.parents, ancestor)
	}
	s.scheduleCodeRequest(req)
}

// Missing retrieves the known missing nodes from the trie for retrieval. To aid
// both eth/6x style fast sync and snap/1x style state sync, the paths of trie
// nodes are returned too, as well as separate hash list for codes.
func (s *Sync) Missing(max int) ([]string, []common.Hash, []common.Hash) {
	var (
		nodePaths  []string
		nodeHashes []common.Hash
		codeHashes []common.Hash
	)
	for !s.queue.Empty() && (max == 0 || len(nodeHashes)+len(codeHashes) < max) {
		// Retrieve the next item in line
		item, prio := s.queue.Peek()

		// If we have too many already-pending tasks for this depth, throttle
		depth := int(prio >> 56)
		if s.fetches[depth] > maxFetchesPerDepth {
			break
		}
		// Item is allowed to be scheduled, add it to the task list
		s.queue.Pop()
		s.fetches[depth]++

		switch item := item.(type) {
		case common.Hash:
			codeHashes = append(codeHashes, item)
		case string:
			req, ok := s.nodeReqs[item]
			if !ok {
				log.Error("Missing node request", "path", item)
				continue // System very wrong, shouldn't happen
			}
			nodePaths = append(nodePaths, item)
			nodeHashes = append(nodeHashes, req.hash)
		}
	}
	return nodePaths, nodeHashes, codeHashes
}

// ProcessCode injects the received data for requested item. Note it can
// happpen that the single response commits two pending requests(e.g.
// there are two requests one for code and one for node but the hash
// is same). In this case the second response for the same hash will
// be treated as "non-requested" item or "already-processed" item but
// there is no downside.
func (s *Sync) ProcessCode(result CodeSyncResult) error {
	// If the code was not requested or it's already processed, bail out
	req := s.codeReqs[result.Hash]
	if req == nil {
		return ErrNotRequested
	}
	if req.data != nil {
		return ErrAlreadyProcessed
	}
	req.data = result.Data
	return s.commitCodeRequest(req)
}

// ProcessNode injects the received data for requested item. Note it can
// happen that the single response commits two pending requests(e.g.
// there are two requests one for code and one for node but the hash
// is same). In this case the second response for the same hash will
// be treated as "non-requested" item or "already-processed" item but
// there is no downside.
func (s *Sync) ProcessNode(result NodeSyncResult) error {
	// If the trie node was not requested or it's already processed, bail out
	req := s.nodeReqs[result.Path]
	if req == nil {
		return ErrNotRequested
	}
	if req.data != nil {
		return ErrAlreadyProcessed
	}
	// Decode the node data content and update the request
	node, err := decodeNode(req.hash.Bytes(), result.Data)
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
		s.commitNodeRequest(req)
	} else {
		req.deps += len(requests)
		for _, child := range requests {
			s.scheduleNodeRequest(child)
		}
	}
	return nil
}

// Commit flushes the data stored in the internal membatch out to persistent
// storage, returning any occurred error.
func (s *Sync) Commit(dbw ethdb.Batch) error {
	// Dump the membatch into a database dbw
	for path, value := range s.membatch.nodes {
		rawdb.WriteTrieNode(dbw, s.membatch.hashes[path], value)
	}
	for hash, value := range s.membatch.codes {
		rawdb.WriteCode(dbw, hash, value)
	}
	// Drop the membatch data and return
	s.membatch = newSyncMemBatch()
	return nil
}

// MemSize returns an estimated size (in bytes) of the data held in the membatch.
func (s *Sync) MemSize() uint64 {
	return s.membatch.size
}

// Pending returns the number of state entries currently pending for download.
func (s *Sync) Pending() int {
	return len(s.nodeReqs) + len(s.codeReqs)
}

// schedule inserts a new state retrieval request into the fetch queue. If there
// is already a pending request for this node, the new request will be discarded
// and only a parent reference added to the old one.
func (s *Sync) scheduleNodeRequest(req *nodeRequest) {
	s.nodeReqs[string(req.path)] = req

	// Schedule the request for future retrieval. This queue is shared
	// by both node requests and code requests.
	prio := int64(len(req.path)) << 56 // depth >= 128 will never happen, storage leaves will be included in their parents
	for i := 0; i < 14 && i < len(req.path); i++ {
		prio |= int64(15-req.path[i]) << (52 - i*4) // 15-nibble => lexicographic order
	}
	s.queue.Push(string(req.path), prio)
}

// schedule inserts a new state retrieval request into the fetch queue. If there
// is already a pending request for this node, the new request will be discarded
// and only a parent reference added to the old one.
func (s *Sync) scheduleCodeRequest(req *codeRequest) {
	// If we're already requesting this node, add a new reference and stop
	if old, ok := s.codeReqs[req.hash]; ok {
		old.parents = append(old.parents, req.parents...)
		return
	}
	s.codeReqs[req.hash] = req

	// Schedule the request for future retrieval. This queue is shared
	// by both node requests and code requests.
	prio := int64(len(req.path)) << 56 // depth >= 128 will never happen, storage leaves will be included in their parents
	for i := 0; i < 14 && i < len(req.path); i++ {
		prio |= int64(15-req.path[i]) << (52 - i*4) // 15-nibble => lexicographic order
	}
	s.queue.Push(req.hash, prio)
}

// children retrieves all the missing children of a state trie entry for future
// retrieval scheduling.
func (s *Sync) children(req *nodeRequest, object node) ([]*nodeRequest, error) {
	// Gather all the children of the node, irrelevant whether known or not
	type childNode struct {
		path []byte
		node node
	}
	var children []childNode

	switch node := (object).(type) {
	case *shortNode:
		key := node.Key
		if hasTerm(key) {
			key = key[:len(key)-1]
		}
		children = []childNode{{
			node: node.Val,
			path: append(append([]byte(nil), req.path...), key...),
		}}
	case *fullNode:
		for i := 0; i < 17; i++ {
			if node.Children[i] != nil {
				children = append(children, childNode{
					node: node.Children[i],
					path: append(append([]byte(nil), req.path...), byte(i)),
				})
			}
		}
	default:
		panic(fmt.Sprintf("unknown node: %+v", node))
	}
	// Iterate over the children, and request all unknown ones
	var (
		missing = make(chan *nodeRequest, len(children))
		pending sync.WaitGroup
	)
	for _, child := range children {
		// Notify any external watcher of a new key/value node
		if req.callback != nil {
			if node, ok := (child.node).(valueNode); ok {
				var paths [][]byte
				if len(child.path) == 2*common.HashLength {
					paths = append(paths, hexToKeybytes(child.path))
				} else if len(child.path) == 4*common.HashLength {
					paths = append(paths, hexToKeybytes(child.path[:2*common.HashLength]))
					paths = append(paths, hexToKeybytes(child.path[2*common.HashLength:]))
				}
				if err := req.callback(paths, child.path, node, req.hash, req.path); err != nil {
					return nil, err
				}
			}
		}
		// If the child references another node, resolve or schedule
		if node, ok := (child.node).(hashNode); ok {
			// Try to resolve the node from the local database
			if s.membatch.hasNode(child.path) {
				continue
			}
			// Check the presence of children concurrently
			pending.Add(1)
			go func(child childNode) {
				defer pending.Done()

				// If database says duplicate, then at least the trie node is present
				// and we hold the assumption that it's NOT legacy contract code.
				chash := common.BytesToHash(node)
				if rawdb.HasTrieNode(s.database, chash) {
					return
				}
				// Locally unknown node, schedule for retrieval
				missing <- &nodeRequest{
					path:     child.path,
					hash:     chash,
					parent:   req,
					callback: req.callback,
				}
			}(child)
		}
	}
	pending.Wait()

	requests := make([]*nodeRequest, 0, len(children))
	for done := false; !done; {
		select {
		case miss := <-missing:
			requests = append(requests, miss)
		default:
			done = true
		}
	}
	return requests, nil
}

// commit finalizes a retrieval request and stores it into the membatch. If any
// of the referencing parent requests complete due to this commit, they are also
// committed themselves.
func (s *Sync) commitNodeRequest(req *nodeRequest) error {
	// Write the node content to the membatch
	s.membatch.nodes[string(req.path)] = req.data
	s.membatch.hashes[string(req.path)] = req.hash
	// The size tracking refers to the db-batch, not the in-memory data.
	// Therefore, we ignore the req.path, and account only for the hash+data
	// which eventually is written to db.
	s.membatch.size += common.HashLength + uint64(len(req.data))
	delete(s.nodeReqs, string(req.path))
	s.fetches[len(req.path)]--

	// Check parent for completion
	if req.parent != nil {
		req.parent.deps--
		if req.parent.deps == 0 {
			if err := s.commitNodeRequest(req.parent); err != nil {
				return err
			}
		}
	}
	return nil
}

// commit finalizes a retrieval request and stores it into the membatch. If any
// of the referencing parent requests complete due to this commit, they are also
// committed themselves.
func (s *Sync) commitCodeRequest(req *codeRequest) error {
	// Write the node content to the membatch
	s.membatch.codes[req.hash] = req.data
	s.membatch.size += common.HashLength + uint64(len(req.data))
	delete(s.codeReqs, req.hash)
	s.fetches[len(req.path)]--

	// Check all parents for completion
	for _, parent := range req.parents {
		parent.deps--
		if parent.deps == 0 {
			if err := s.commitNodeRequest(parent); err != nil {
				return err
			}
		}
	}
	return nil
}
