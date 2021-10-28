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

// nodeRequest represents a scheduled or already in-flight trie node retrieval request.
type nodeRequest struct {
	key  string // Key of the node data content to retrieve
	path []byte // Merkle path leading to this node for prioritization
	data []byte // Data content of the node, cached until all subtrees complete

	parents  []*nodeRequest // Parent state nodes referencing this entry (notify all upon completion)
	deps     int            // Number of dependencies before allowed to commit this node
	callback LeafCallback   // Callback to invoke if a leaf node it reached on this branch
}

// codeRequest represents a scheduled or already in-flight bytecode retrieval request.
type codeRequest struct {
	hash    common.Hash    // Hash of the contract bytecode to retrieve
	path    []byte         // Merkle path leading to this node for prioritization
	data    []byte         // Data content of the node, cached until all subtrees complete
	parents []*nodeRequest // Parent state nodes referencing this entry (notify all upon completion)
}

// NodeSyncResult is a response with requested trie node along with it's database key.
type NodeSyncResult struct {
	Key  string // String format key of the originally unknown trie node
	Data []byte // Data content of the retrieved trie node
}

// CodeSyncResult is a response with requested bytecode along with it's hash
type CodeSyncResult struct {
	Hash common.Hash // Hash the originally unknown bytecode
	Data []byte      // Data content of the retrieved bytecode
}

// syncMemBatch is an in-memory buffer of successfully downloaded but not yet
// persisted data items.
type syncMemBatch struct {
	nodes map[string][]byte      // In-memory membatch of recently completed nodes
	codes map[common.Hash][]byte // In-memory membatch of recently completed codes
}

// newSyncMemBatch allocates a new memory-buffer for not-yet persisted trie nodes.
func newSyncMemBatch() *syncMemBatch {
	return &syncMemBatch{
		nodes: make(map[string][]byte),
		codes: make(map[common.Hash][]byte),
	}
}

// hasNode reports the trie node with specific hash is already cached.
func (batch *syncMemBatch) hasNode(key string) bool {
	_, ok := batch.nodes[key]
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
	nodeReqs map[string]*nodeRequest      // Pending requests pertaining to a trie node hash
	codeReqs map[common.Hash]*codeRequest // Pending requests pertaining to a code hash
	queue    *prque.Prque                 // Priority queue with the pending requests
	fetches  map[int]int                  // Number of active fetches per trie node depth
	bloom    *SyncBloom                   // Bloom filter for fast state existence checks
}

// NewSync creates a new trie data download scheduler.
func NewSync(root common.Hash, database ethdb.KeyValueReader, callback LeafCallback, bloom *SyncBloom) *Sync {
	ts := &Sync{
		database: database,
		membatch: newSyncMemBatch(),
		nodeReqs: make(map[string]*nodeRequest),
		codeReqs: make(map[common.Hash]*codeRequest),
		queue:    prque.New(nil),
		fetches:  make(map[int]int),
		bloom:    bloom,
	}
	ts.AddSubTrie(root, nil, common.Hash{}, nil, callback)
	return ts
}

// AddSubTrie registers a new trie to the sync code, rooted at the designated
// parent for completion tracking. The given path is in hexary format and should
// contains all the parent path if it's layered trie node.
func (s *Sync) AddSubTrie(root common.Hash, path []byte, parent common.Hash, parentPath []byte, callback LeafCallback) {
	// Short circuit if the trie is empty or already known
	if root == emptyRoot {
		return
	}
	var owner common.Hash
	if len(path) == 2*common.HashLength {
		owner = common.BytesToHash(hexToKeybytes(path))
	}
	var (
		storageKey  = EncodeStorageKey(owner, nil)
		internalKey = EncodeInternalKey(storageKey, root)
	)
	if s.membatch.hasNode(string(internalKey)) {
		return
	}
	if s.bloom == nil || s.bloom.Contains(root.Bytes()) {
		// Bloom filter says this might be a duplicate, double check.
		blob, nodeHash := rawdb.ReadTrieNode(s.database, storageKey)
		if len(blob) == 0 {
			bloomFaultMeter.Mark(1)
		} else if nodeHash == root {
			return
		}
	}
	// Assemble the new sub-trie sync request
	req := &nodeRequest{
		key:      string(internalKey),
		path:     path,
		callback: callback,
	}
	// If this sub-trie has a designated parent, link them together
	if parent != (common.Hash{}) {
		ancestor := s.nodeReqs[string(EncodeInternalKeyWithPath(common.Hash{}, parentPath, parent))]
		if ancestor == nil {
			panic(fmt.Sprintf("sub-trie ancestor not found: %x", parent))
		}
		ancestor.deps++
		req.parents = append(req.parents, ancestor)
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
	req := &codeRequest{
		path: path,
		hash: hash,
	}
	// If this sub-trie has a designated parent, link them together
	if parent != (common.Hash{}) {
		ancestor := s.nodeReqs[string(EncodeInternalKeyWithPath(common.Hash{}, parentPath, parent))] // the parent of codereq can ONLY be nodereq
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
func (s *Sync) Missing(max int) ([]string, []common.Hash, []NodePath, []common.Hash) {
	var (
		nodeKeys   []string
		nodeHashes []common.Hash
		codeHashes []common.Hash
		nodePaths  []NodePath
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

		switch item.(type) {
		case common.Hash:
			codeHashes = append(codeHashes, item.(common.Hash))
		case string:
			key := item.(string)
			req, ok := s.nodeReqs[key]
			if !ok {
				log.Warn("Missing node request", "key", key)
				continue // System very wrong, shouldn't happen
			}
			_, hash := DecodeInternalKey([]byte(key))
			nodeKeys = append(nodeKeys, key)
			nodeHashes = append(nodeHashes, hash)
			nodePaths = append(nodePaths, NewNodePath(req.path))
		}
	}
	return nodeKeys, nodeHashes, nodePaths, codeHashes
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
// happpen that the single response commits two pending requests(e.g.
// there are two requests one for code and one for node but the hash
// is same). In this case the second response for the same hash will
// be treated as "non-requested" item or "already-processed" item but
// there is no downside.
func (s *Sync) ProcessNode(result NodeSyncResult) error {
	// If the trie node was not requested or it's already processed, bail out
	req := s.nodeReqs[result.Key]
	if req == nil {
		return ErrNotRequested
	}
	if req.data != nil {
		return ErrAlreadyProcessed
	}
	// Decode the node data content and update the request
	_, hash := DecodeInternalKey([]byte(req.key))
	node, err := decodeNode(hash.Bytes(), result.Data)
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
	for key, value := range s.membatch.nodes {
		storageKey, hash := DecodeInternalKey([]byte(key))
		rawdb.WriteTrieNode(dbw, storageKey, value)
		if s.bloom != nil {
			s.bloom.Add(hash.Bytes())
		}
	}
	for key, value := range s.membatch.codes {
		rawdb.WriteCode(dbw, key, value)
		if s.bloom != nil {
			s.bloom.Add(key[:])
		}
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
func (s *Sync) scheduleNodeRequest(req *nodeRequest) {
	// If we're already requesting this node, add a new reference and stop
	// TODO(rjl493456442) it seems impossible now since all the trie node
	// has the unique key. Remove it.
	if old, ok := s.nodeReqs[req.key]; ok {
		old.parents = append(old.parents, req.parents...)
		return
	}
	s.nodeReqs[req.key] = req

	// Schedule the request for future retrieval. This queue is shared
	// by both node requests and code requests. It can happen that there
	// is a trie node and code has same hash. In this case two elements
	// with same hash and same or different depth will be pushed. But it's
	// ok the worst case is the second response will be treated as duplicated.
	prio := int64(len(req.path)) << 56 // depth >= 128 will never happen, storage leaves will be included in their parents
	for i := 0; i < 14 && i < len(req.path); i++ {
		prio |= int64(15-req.path[i]) << (52 - i*4) // 15-nibble => lexicographic order
	}
	s.queue.Push(req.key, prio)
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
	// by both node requests and code requests. It can happen that there
	// is a trie node and code has same hash. In this case two elements
	// with same hash and same or different depth will be pushed. But it's
	// ok the worst case is the second response will be treated as duplicated.
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
	type child struct {
		path []byte
		node node
	}
	var children []child

	switch node := (object).(type) {
	case *shortNode:
		key := node.Key
		if hasTerm(key) {
			key = key[:len(key)-1]
		}
		children = []child{{
			node: node.Val,
			path: append(append([]byte(nil), req.path...), key...),
		}}
	case *fullNode:
		for i := 0; i < 17; i++ {
			if node.Children[i] != nil {
				children = append(children, child{
					node: node.Children[i],
					path: append(append([]byte(nil), req.path...), byte(i)),
				})
			}
		}
	default:
		panic(fmt.Sprintf("unknown node: %+v", node))
	}
	// Iterate over the children, and request all unknown ones
	requests := make([]*nodeRequest, 0, len(children))
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
				_, hash := DecodeInternalKey([]byte(req.key))
				if err := req.callback(paths, child.path, node, hash, req.path); err != nil {
					return nil, err
				}
			}
		}
		// If the child references another node, resolve or schedule
		if node, ok := (child.node).(hashNode); ok {
			var (
				inner []byte
				owner common.Hash
			)
			if len(child.path) >= 2*common.HashLength {
				owner = common.BytesToHash(hexToKeybytes(child.path[:2*common.HashLength]))
				inner = child.path[2*common.HashLength:]
			} else {
				inner = child.path
			}
			var (
				chash      = common.BytesToHash(node)
				storageKey = EncodeStorageKey(owner, inner)
				childKey   = EncodeInternalKey(storageKey, chash)
			)
			// Try to resolve the node from the local database
			if s.membatch.hasNode(string(childKey)) {
				continue
			}
			if s.bloom == nil || s.bloom.Contains(chash.Bytes()) {
				// Bloom filter says this might be a duplicate, double check.
				blob, hash := rawdb.ReadTrieNode(s.database, storageKey)
				if len(blob) == 0 {
					bloomFaultMeter.Mark(1)
				} else if hash == chash {
					continue
				}
			}
			// Locally unknown node, schedule for retrieval
			requests = append(requests, &nodeRequest{
				key:      string(childKey),
				path:     child.path,
				parents:  []*nodeRequest{req},
				callback: req.callback,
			})
		}
	}
	return requests, nil
}

// commit finalizes a retrieval request and stores it into the membatch. If any
// of the referencing parent requests complete due to this commit, they are also
// committed themselves.
func (s *Sync) commitNodeRequest(req *nodeRequest) error {
	// Write the node content to the membatch
	s.membatch.nodes[req.key] = req.data
	delete(s.nodeReqs, req.key)
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

// commit finalizes a retrieval request and stores it into the membatch. If any
// of the referencing parent requests complete due to this commit, they are also
// committed themselves.
func (s *Sync) commitCodeRequest(req *codeRequest) error {
	// Write the node content to the membatch
	s.membatch.codes[req.hash] = req.data
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
