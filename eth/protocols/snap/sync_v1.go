// Copyright 2026 The go-ethereum Authors
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

package snap

import (
	"bytes"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
)

// trienodeHealRequest tracks a pending state trie request to ensure responses
// are to actual requests and to validate any security constraints.
//
// Concurrency note: trie node requests and responses are handled concurrently from
// the main runloop to allow Keccak256 hash verifications on the peer's thread and
// to drop on invalid response. The request struct must contain all the data to
// construct the response without accessing runloop internals (i.e. task). That
// is only included to allow the runloop to match a response to the task being
// synced without having yet another set of maps.
type trienodeHealRequest struct {
	peer string    // Peer to which this request is assigned
	id   uint64    // Request ID of this request
	time time.Time // Timestamp when the request was sent

	deliver chan *trienodeHealResponse // Channel to deliver successful response on
	revert  chan *trienodeHealRequest  // Channel to deliver request failure on
	cancel  chan struct{}              // Channel to track sync cancellation
	timeout *time.Timer                // Timer to track delivery timeout
	stale   chan struct{}              // Channel to signal the request was dropped

	paths  []string      // Trie node paths for identifying trie node
	hashes []common.Hash // Trie node hashes to validate responses

	task *healTask // Task which this request is filling (only access fields through the runloop!!)
}

// trienodeHealResponse is an already verified remote response to a trie node request.
type trienodeHealResponse struct {
	task *healTask // Task which this request is filling

	paths  []string      // Paths of the trie nodes
	hashes []common.Hash // Hashes of the trie nodes to avoid double hashing
	nodes  [][]byte      // Actual trie nodes to store into the database (nil = missing)
}

// bytecodeHealRequest tracks a pending bytecode request to ensure responses are to
// actual requests and to validate any security constraints.
//
// Concurrency note: bytecode requests and responses are handled concurrently from
// the main runloop to allow Keccak256 hash verifications on the peer's thread and
// to drop on invalid response. The request struct must contain all the data to
// construct the response without accessing runloop internals (i.e. task). That
// is only included to allow the runloop to match a response to the task being
// synced without having yet another set of maps.
type bytecodeHealRequest struct {
	peer string    // Peer to which this request is assigned
	id   uint64    // Request ID of this request
	time time.Time // Timestamp when the request was sent

	deliver chan *bytecodeHealResponse // Channel to deliver successful response on
	revert  chan *bytecodeHealRequest  // Channel to deliver request failure on
	cancel  chan struct{}              // Channel to track sync cancellation
	timeout *time.Timer                // Timer to track delivery timeout
	stale   chan struct{}              // Channel to signal the request was dropped

	hashes []common.Hash // Bytecode hashes to validate responses
	task   *healTask     // Task which this request is filling (only access fields through the runloop!!)
}

// bytecodeHealResponse is an already verified remote response to a bytecode request.
type bytecodeHealResponse struct {
	task *healTask // Task which this request is filling

	hashes []common.Hash // Hashes of the bytecode to avoid double hashing
	codes  [][]byte      // Actual bytecodes to store into the database (nil = missing)
}

// healTask represents the sync task for healing the snap-synced chunk boundaries.
type healTask struct {
	scheduler *trie.Sync // State trie sync scheduler defining the tasks

	trieTasks map[string]common.Hash   // Set of trie node tasks currently queued for retrieval, indexed by node path
	codeTasks map[common.Hash]struct{} // Set of byte code tasks currently queued for retrieval, indexed by code hash
}

// SyncPending is analogous to SyncProgress, but it's used to report on pending
// ephemeral sync progress that doesn't get persisted into the database.
type SyncPending struct {
	TrienodeHeal uint64 // Number of state trie nodes pending
	BytecodeHeal uint64 // Number of bytecodes pending
}

// healRequestSort implements the Sort interface, allowing sorting trienode
// heal requests, which is a prerequisite for merging storage-requests.
type healRequestSort struct {
	paths     []string
	hashes    []common.Hash
	syncPaths []trie.SyncPath
}

func (t *healRequestSort) Len() int {
	return len(t.hashes)
}

func (t *healRequestSort) Less(i, j int) bool {
	a := t.syncPaths[i]
	b := t.syncPaths[j]
	switch bytes.Compare(a[0], b[0]) {
	case -1:
		return true
	case 1:
		return false
	}
	// identical first part
	if len(a) < len(b) {
		return true
	}
	if len(b) < len(a) {
		return false
	}
	if len(a) == 2 {
		return bytes.Compare(a[1], b[1]) < 0
	}
	return false
}

func (t *healRequestSort) Swap(i, j int) {
	t.paths[i], t.paths[j] = t.paths[j], t.paths[i]
	t.hashes[i], t.hashes[j] = t.hashes[j], t.hashes[i]
	t.syncPaths[i], t.syncPaths[j] = t.syncPaths[j], t.syncPaths[i]
}

// Merge merges the pathsets, so that several storage requests concerning the
// same account are merged into one, to reduce bandwidth.
// OBS: This operation is moot if t has not first been sorted.
func (t *healRequestSort) Merge() []TrieNodePathSet {
	var result []TrieNodePathSet
	for _, path := range t.syncPaths {
		pathset := TrieNodePathSet(path)
		if len(path) == 1 {
			// It's an account reference.
			result = append(result, pathset)
		} else {
			// It's a storage reference.
			end := len(result) - 1
			if len(result) == 0 || !bytes.Equal(pathset[0], result[end][0]) {
				// The account doesn't match last, create a new entry.
				result = append(result, pathset)
			} else {
				// It's the same account as the previous one, add to the storage
				// paths of that request.
				result[end] = append(result[end], pathset[1])
			}
		}
	}
	return result
}
