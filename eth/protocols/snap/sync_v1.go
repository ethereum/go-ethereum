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
	"errors"
	"fmt"
	gomath "math"
	"math/rand"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
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

// assignTrienodeHealTasks attempts to match idle peers to trie node requests to
// heal any trie errors caused by the snap sync's chunked retrieval model.
func (s *Syncer) assignTrienodeHealTasks(success chan *trienodeHealResponse, fail chan *trienodeHealRequest, cancel chan struct{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Sort the peers by download capacity to use faster ones if many available
	idlers := &capacitySort{
		ids:  make([]string, 0, len(s.trienodeHealIdlers)),
		caps: make([]int, 0, len(s.trienodeHealIdlers)),
	}
	targetTTL := s.rates.TargetTimeout()
	for id := range s.trienodeHealIdlers {
		if _, ok := s.statelessPeers[id]; ok {
			continue
		}
		idlers.ids = append(idlers.ids, id)
		idlers.caps = append(idlers.caps, s.rates.Capacity(id, TrieNodesMsg, targetTTL))
	}
	if len(idlers.ids) == 0 {
		return
	}
	sort.Sort(sort.Reverse(idlers))

	// Iterate over pending tasks and try to find a peer to retrieve with
	for len(s.healer.trieTasks) > 0 || s.healer.scheduler.Pending() > 0 {
		// If there are not enough trie tasks queued to fully assign, fill the
		// queue from the state sync scheduler. The trie synced schedules these
		// together with bytecodes, so we need to queue them combined.
		var (
			have = len(s.healer.trieTasks) + len(s.healer.codeTasks)
			want = maxTrieRequestCount + maxCodeRequestCount
		)
		if have < want {
			paths, hashes, codes := s.healer.scheduler.Missing(want - have)
			for i, path := range paths {
				s.healer.trieTasks[path] = hashes[i]
			}
			for _, hash := range codes {
				s.healer.codeTasks[hash] = struct{}{}
			}
		}
		// If all the heal tasks are bytecodes or already downloading, bail
		if len(s.healer.trieTasks) == 0 {
			return
		}
		// Task pending retrieval, try to find an idle peer. If no such peer
		// exists, we probably assigned tasks for all (or they are stateless).
		// Abort the entire assignment mechanism.
		if len(idlers.ids) == 0 {
			return
		}
		var (
			idle = idlers.ids[0]
			peer = s.peers[idle]
			cap  = idlers.caps[0]
		)
		idlers.ids, idlers.caps = idlers.ids[1:], idlers.caps[1:]

		// Matched a pending task to an idle peer, allocate a unique request id
		var reqid uint64
		for {
			reqid = uint64(rand.Int63())
			if reqid == 0 {
				continue
			}
			if _, ok := s.trienodeHealReqs[reqid]; ok {
				continue
			}
			break
		}
		// Generate the network query and send it to the peer
		if cap > maxTrieRequestCount {
			cap = maxTrieRequestCount
		}
		cap = int(float64(cap) / s.trienodeHealThrottle)
		if cap <= 0 {
			cap = 1
		}
		var (
			hashes   = make([]common.Hash, 0, cap)
			paths    = make([]string, 0, cap)
			pathsets = make([]TrieNodePathSet, 0, cap)
		)
		for path, hash := range s.healer.trieTasks {
			delete(s.healer.trieTasks, path)

			paths = append(paths, path)
			hashes = append(hashes, hash)
			if len(paths) >= cap {
				break
			}
		}
		// Group requests by account hash
		paths, hashes, _, pathsets = sortByAccountPath(paths, hashes)
		req := &trienodeHealRequest{
			peer:    idle,
			id:      reqid,
			time:    time.Now(),
			deliver: success,
			revert:  fail,
			cancel:  cancel,
			stale:   make(chan struct{}),
			paths:   paths,
			hashes:  hashes,
			task:    s.healer,
		}
		req.timeout = time.AfterFunc(s.rates.TargetTimeout(), func() {
			peer.Log().Debug("Trienode heal request timed out", "reqid", reqid)
			s.rates.Update(idle, TrieNodesMsg, 0, 0)
			s.scheduleRevertTrienodeHealRequest(req)
		})
		s.trienodeHealReqs[reqid] = req
		delete(s.trienodeHealIdlers, idle)

		s.pend.Add(1)
		go func(root common.Hash) {
			defer s.pend.Done()

			// Attempt to send the remote request and revert if it fails
			if err := peer.RequestTrieNodes(reqid, root, len(paths), pathsets, maxRequestSize); err != nil {
				log.Debug("Failed to request trienode healers", "err", err)
				s.scheduleRevertTrienodeHealRequest(req)
			}
		}(s.root)
	}
}

// assignBytecodeHealTasks attempts to match idle peers to bytecode requests to
// heal any trie errors caused by the snap sync's chunked retrieval model.
func (s *Syncer) assignBytecodeHealTasks(success chan *bytecodeHealResponse, fail chan *bytecodeHealRequest, cancel chan struct{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Sort the peers by download capacity to use faster ones if many available
	idlers := &capacitySort{
		ids:  make([]string, 0, len(s.bytecodeHealIdlers)),
		caps: make([]int, 0, len(s.bytecodeHealIdlers)),
	}
	targetTTL := s.rates.TargetTimeout()
	for id := range s.bytecodeHealIdlers {
		if _, ok := s.statelessPeers[id]; ok {
			continue
		}
		idlers.ids = append(idlers.ids, id)
		idlers.caps = append(idlers.caps, s.rates.Capacity(id, ByteCodesMsg, targetTTL))
	}
	if len(idlers.ids) == 0 {
		return
	}
	sort.Sort(sort.Reverse(idlers))

	// Iterate over pending tasks and try to find a peer to retrieve with
	for len(s.healer.codeTasks) > 0 || s.healer.scheduler.Pending() > 0 {
		// If there are not enough trie tasks queued to fully assign, fill the
		// queue from the state sync scheduler. The trie synced schedules these
		// together with trie nodes, so we need to queue them combined.
		var (
			have = len(s.healer.trieTasks) + len(s.healer.codeTasks)
			want = maxTrieRequestCount + maxCodeRequestCount
		)
		if have < want {
			paths, hashes, codes := s.healer.scheduler.Missing(want - have)
			for i, path := range paths {
				s.healer.trieTasks[path] = hashes[i]
			}
			for _, hash := range codes {
				s.healer.codeTasks[hash] = struct{}{}
			}
		}
		// If all the heal tasks are trienodes or already downloading, bail
		if len(s.healer.codeTasks) == 0 {
			return
		}
		// Task pending retrieval, try to find an idle peer. If no such peer
		// exists, we probably assigned tasks for all (or they are stateless).
		// Abort the entire assignment mechanism.
		if len(idlers.ids) == 0 {
			return
		}
		var (
			idle = idlers.ids[0]
			peer = s.peers[idle]
			cap  = idlers.caps[0]
		)
		idlers.ids, idlers.caps = idlers.ids[1:], idlers.caps[1:]

		// Matched a pending task to an idle peer, allocate a unique request id
		var reqid uint64
		for {
			reqid = uint64(rand.Int63())
			if reqid == 0 {
				continue
			}
			if _, ok := s.bytecodeHealReqs[reqid]; ok {
				continue
			}
			break
		}
		// Generate the network query and send it to the peer
		if cap > maxCodeRequestCount {
			cap = maxCodeRequestCount
		}
		hashes := make([]common.Hash, 0, cap)
		for hash := range s.healer.codeTasks {
			delete(s.healer.codeTasks, hash)

			hashes = append(hashes, hash)
			if len(hashes) >= cap {
				break
			}
		}
		req := &bytecodeHealRequest{
			peer:    idle,
			id:      reqid,
			time:    time.Now(),
			deliver: success,
			revert:  fail,
			cancel:  cancel,
			stale:   make(chan struct{}),
			hashes:  hashes,
			task:    s.healer,
		}
		req.timeout = time.AfterFunc(s.rates.TargetTimeout(), func() {
			peer.Log().Debug("Bytecode heal request timed out", "reqid", reqid)
			s.rates.Update(idle, ByteCodesMsg, 0, 0)
			s.scheduleRevertBytecodeHealRequest(req)
		})
		s.bytecodeHealReqs[reqid] = req
		delete(s.bytecodeHealIdlers, idle)

		s.pend.Add(1)
		go func() {
			defer s.pend.Done()

			// Attempt to send the remote request and revert if it fails
			if err := peer.RequestByteCodes(reqid, hashes, maxRequestSize); err != nil {
				log.Debug("Failed to request bytecode healers", "err", err)
				s.scheduleRevertBytecodeHealRequest(req)
			}
		}()
	}
}

// scheduleRevertTrienodeHealRequest asks the event loop to clean up a trienode heal
// request and return all failed retrieval tasks to the scheduler for reassignment.
func (s *Syncer) scheduleRevertTrienodeHealRequest(req *trienodeHealRequest) {
	select {
	case req.revert <- req:
		// Sync event loop notified
	case <-req.cancel:
		// Sync cycle got cancelled
	case <-req.stale:
		// Request already reverted
	}
}

// revertTrienodeHealRequest cleans up a trienode heal request and returns all
// failed retrieval tasks to the scheduler for reassignment.
//
// Note, this needs to run on the event runloop thread to reschedule to idle peers.
// On peer threads, use scheduleRevertTrienodeHealRequest.
func (s *Syncer) revertTrienodeHealRequest(req *trienodeHealRequest) {
	log.Debug("Reverting trienode heal request", "peer", req.peer)
	select {
	case <-req.stale:
		log.Trace("Trienode heal request already reverted", "peer", req.peer, "reqid", req.id)
		return
	default:
	}
	close(req.stale)

	// Remove the request from the tracked set and restore the peer to the
	// idle pool so it can be reassigned work (skip if peer already left).
	s.lock.Lock()
	delete(s.trienodeHealReqs, req.id)
	if _, ok := s.peers[req.peer]; ok {
		s.trienodeHealIdlers[req.peer] = struct{}{}
	}
	s.lock.Unlock()

	// If there's a timeout timer still running, abort it and mark the trie node
	// retrievals as not-pending, ready for rescheduling
	req.timeout.Stop()
	for i, path := range req.paths {
		req.task.trieTasks[path] = req.hashes[i]
	}
}

// scheduleRevertBytecodeHealRequest asks the event loop to clean up a bytecode heal
// request and return all failed retrieval tasks to the scheduler for reassignment.
func (s *Syncer) scheduleRevertBytecodeHealRequest(req *bytecodeHealRequest) {
	select {
	case req.revert <- req:
		// Sync event loop notified
	case <-req.cancel:
		// Sync cycle got cancelled
	case <-req.stale:
		// Request already reverted
	}
}

// revertBytecodeHealRequest cleans up a bytecode heal request and returns all
// failed retrieval tasks to the scheduler for reassignment.
//
// Note, this needs to run on the event runloop thread to reschedule to idle peers.
// On peer threads, use scheduleRevertBytecodeHealRequest.
func (s *Syncer) revertBytecodeHealRequest(req *bytecodeHealRequest) {
	log.Debug("Reverting bytecode heal request", "peer", req.peer)
	select {
	case <-req.stale:
		log.Trace("Bytecode heal request already reverted", "peer", req.peer, "reqid", req.id)
		return
	default:
	}
	close(req.stale)

	// Remove the request from the tracked set and restore the peer to the
	// idle pool so it can be reassigned work (skip if peer already left).
	s.lock.Lock()
	delete(s.bytecodeHealReqs, req.id)
	if _, ok := s.peers[req.peer]; ok {
		s.bytecodeHealIdlers[req.peer] = struct{}{}
	}
	s.lock.Unlock()

	// If there's a timeout timer still running, abort it and mark the code
	// retrievals as not-pending, ready for rescheduling
	req.timeout.Stop()
	for _, hash := range req.hashes {
		req.task.codeTasks[hash] = struct{}{}
	}
}

// processTrienodeHealResponse integrates an already validated trienode response
// into the healer tasks.
func (s *Syncer) processTrienodeHealResponse(res *trienodeHealResponse) {
	var (
		start = time.Now()
		fills int
	)
	for i, hash := range res.hashes {
		node := res.nodes[i]

		// If the trie node was not delivered, reschedule it
		if node == nil {
			res.task.trieTasks[res.paths[i]] = res.hashes[i]
			continue
		}
		fills++

		// Push the trie node into the state syncer
		s.trienodeHealSynced++
		s.trienodeHealBytes += common.StorageSize(len(node))

		err := s.healer.scheduler.ProcessNode(trie.NodeSyncResult{Path: res.paths[i], Data: node})
		switch err {
		case nil:
		case trie.ErrAlreadyProcessed:
			s.trienodeHealDups++
		case trie.ErrNotRequested:
			s.trienodeHealNops++
		default:
			log.Error("Invalid trienode processed", "hash", hash, "err", err)
		}
	}
	s.commitHealer(false)

	// Calculate the processing rate of one filled trie node
	rate := float64(fills) / (float64(time.Since(start)) / float64(time.Second))

	// Update the currently measured trienode queueing and processing throughput.
	//
	// The processing rate needs to be updated uniformly independent if we've
	// processed 1x100 trie nodes or 100x1 to keep the rate consistent even in
	// the face of varying network packets. As such, we cannot just measure the
	// time it took to process N trie nodes and update once, we need one update
	// per trie node.
	//
	// Naively, that would be:
	//
	//   for i:=0; i<fills; i++ {
	//     healRate = (1-measurementImpact)*oldRate + measurementImpact*newRate
	//   }
	//
	// Essentially, a recursive expansion of HR = (1-MI)*HR + MI*NR.
	//
	// We can expand that formula for the Nth item as:
	//   HR(N) = (1-MI)^N*OR + (1-MI)^(N-1)*MI*NR + (1-MI)^(N-2)*MI*NR + ... + (1-MI)^0*MI*NR
	//
	// The above is a geometric sequence that can be summed to:
	//   HR(N) = (1-MI)^N*(OR-NR) + NR
	s.trienodeHealRate = gomath.Pow(1-trienodeHealRateMeasurementImpact, float64(fills))*(s.trienodeHealRate-rate) + rate

	pending := s.trienodeHealPend.Load()
	if time.Since(s.trienodeHealThrottled) > time.Second {
		// Periodically adjust the trie node throttler
		if float64(pending) > 2*s.trienodeHealRate {
			s.trienodeHealThrottle *= trienodeHealThrottleIncrease
		} else {
			s.trienodeHealThrottle /= trienodeHealThrottleDecrease
		}
		if s.trienodeHealThrottle > maxTrienodeHealThrottle {
			s.trienodeHealThrottle = maxTrienodeHealThrottle
		} else if s.trienodeHealThrottle < minTrienodeHealThrottle {
			s.trienodeHealThrottle = minTrienodeHealThrottle
		}
		s.trienodeHealThrottled = time.Now()

		log.Debug("Updated trie node heal throttler", "rate", s.trienodeHealRate, "pending", pending, "throttle", s.trienodeHealThrottle)
	}
}

func (s *Syncer) commitHealer(force bool) {
	if !force && s.healer.scheduler.MemSize() < ethdb.IdealBatchSize {
		return
	}
	batch := s.db.NewBatch()
	if err := s.healer.scheduler.Commit(batch); err != nil {
		log.Crit("Failed to commit healing data", "err", err)
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to persist healing data", "err", err)
	}
	log.Debug("Persisted set of healing data", "type", "trienodes", "bytes", common.StorageSize(batch.ValueSize()))
}

// processBytecodeHealResponse integrates an already validated bytecode response
// into the healer tasks.
func (s *Syncer) processBytecodeHealResponse(res *bytecodeHealResponse) {
	for i, hash := range res.hashes {
		node := res.codes[i]

		// If the trie node was not delivered, reschedule it
		if node == nil {
			res.task.codeTasks[hash] = struct{}{}
			continue
		}
		// Push the trie node into the state syncer
		s.bytecodeHealSynced++
		s.bytecodeHealBytes += common.StorageSize(len(node))

		err := s.healer.scheduler.ProcessCode(trie.CodeSyncResult{Hash: hash, Data: node})
		switch err {
		case nil:
		case trie.ErrAlreadyProcessed:
			s.bytecodeHealDups++
		case trie.ErrNotRequested:
			s.bytecodeHealNops++
		default:
			log.Error("Invalid bytecode processed", "hash", hash, "err", err)
		}
	}
	s.commitHealer(false)
}

// OnTrieNodes is a callback method to invoke when a batch of trie nodes
// are received from a remote peer.
func (s *Syncer) OnTrieNodes(peer SyncPeer, id uint64, trienodes [][]byte) error {
	var size common.StorageSize
	for _, node := range trienodes {
		size += common.StorageSize(len(node))
	}
	logger := peer.Log().New("reqid", id)
	logger.Trace("Delivering set of healing trienodes", "trienodes", len(trienodes), "bytes", size)

	// Whether or not the response is valid, we can mark the peer as idle and
	// notify the scheduler to assign a new task. If the response is invalid,
	// we'll drop the peer in a bit.
	defer func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		if _, ok := s.peers[peer.ID()]; ok {
			s.trienodeHealIdlers[peer.ID()] = struct{}{}
		}
		select {
		case s.update <- struct{}{}:
		default:
		}
	}()
	s.lock.Lock()
	// Ensure the response is for a valid request
	req, ok := s.trienodeHealReqs[id]
	if !ok {
		// Request stale, perhaps the peer timed out but came through in the end
		logger.Warn("Unexpected trienode heal packet")
		s.lock.Unlock()
		return nil
	}
	delete(s.trienodeHealReqs, id)
	s.rates.Update(peer.ID(), TrieNodesMsg, time.Since(req.time), len(trienodes))

	// Clean up the request timeout timer, we'll see how to proceed further based
	// on the actual delivered content
	if !req.timeout.Stop() {
		// The timeout is already triggered, and this request will be reverted+rescheduled
		s.lock.Unlock()
		return nil
	}

	// Response is valid, but check if peer is signalling that it does not have
	// the requested data. For bytecode range queries that means the peer is not
	// yet synced.
	if len(trienodes) == 0 {
		logger.Debug("Peer rejected trienode heal request")
		s.statelessPeers[peer.ID()] = struct{}{}
		s.lock.Unlock()

		// Signal this request as failed, and ready for rescheduling
		s.scheduleRevertTrienodeHealRequest(req)
		return nil
	}
	s.lock.Unlock()

	// Cross reference the requested trienodes with the response to find gaps
	// that the serving node is missing
	var (
		hasher = crypto.NewKeccakState()
		hash   = make([]byte, 32)
		nodes  = make([][]byte, len(req.hashes))
		fills  uint64
	)
	for i, j := 0, 0; i < len(trienodes); i++ {
		// Find the next hash that we've been served, leaving misses with nils
		hasher.Reset()
		hasher.Write(trienodes[i])
		hasher.Read(hash)

		for j < len(req.hashes) && !bytes.Equal(hash, req.hashes[j][:]) {
			j++
		}
		if j < len(req.hashes) {
			nodes[j] = trienodes[i]
			fills++
			j++
			continue
		}
		// We've either ran out of hashes, or got unrequested data
		logger.Warn("Unexpected healing trienodes", "count", len(trienodes)-i)

		// Signal this request as failed, and ready for rescheduling
		s.scheduleRevertTrienodeHealRequest(req)
		return errors.New("unexpected healing trienode")
	}
	// Response validated, send it to the scheduler for filling
	s.trienodeHealPend.Add(fills)
	defer func() {
		s.trienodeHealPend.Add(^(fills - 1))
	}()
	response := &trienodeHealResponse{
		paths:  req.paths,
		task:   req.task,
		hashes: req.hashes,
		nodes:  nodes,
	}
	select {
	case req.deliver <- response:
	case <-req.cancel:
	case <-req.stale:
	}
	return nil
}

// onHealByteCodes is a callback method to invoke when a batch of contract
// bytes codes are received from a remote peer in the healing phase.
func (s *Syncer) onHealByteCodes(peer SyncPeer, id uint64, bytecodes [][]byte) error {
	var size common.StorageSize
	for _, code := range bytecodes {
		size += common.StorageSize(len(code))
	}
	logger := peer.Log().New("reqid", id)
	logger.Trace("Delivering set of healing bytecodes", "bytecodes", len(bytecodes), "bytes", size)

	// Whether or not the response is valid, we can mark the peer as idle and
	// notify the scheduler to assign a new task. If the response is invalid,
	// we'll drop the peer in a bit.
	defer func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		if _, ok := s.peers[peer.ID()]; ok {
			s.bytecodeHealIdlers[peer.ID()] = struct{}{}
		}
		select {
		case s.update <- struct{}{}:
		default:
		}
	}()
	s.lock.Lock()
	// Ensure the response is for a valid request
	req, ok := s.bytecodeHealReqs[id]
	if !ok {
		// Request stale, perhaps the peer timed out but came through in the end
		logger.Warn("Unexpected bytecode heal packet")
		s.lock.Unlock()
		return nil
	}
	delete(s.bytecodeHealReqs, id)
	s.rates.Update(peer.ID(), ByteCodesMsg, time.Since(req.time), len(bytecodes))

	// Clean up the request timeout timer, we'll see how to proceed further based
	// on the actual delivered content
	if !req.timeout.Stop() {
		// The timeout is already triggered, and this request will be reverted+rescheduled
		s.lock.Unlock()
		return nil
	}

	// Response is valid, but check if peer is signalling that it does not have
	// the requested data. For bytecode range queries that means the peer is not
	// yet synced.
	if len(bytecodes) == 0 {
		logger.Debug("Peer rejected bytecode heal request")
		s.statelessPeers[peer.ID()] = struct{}{}
		s.lock.Unlock()

		// Signal this request as failed, and ready for rescheduling
		s.scheduleRevertBytecodeHealRequest(req)
		return nil
	}
	s.lock.Unlock()

	// Cross reference the requested bytecodes with the response to find gaps
	// that the serving node is missing
	hasher := crypto.NewKeccakState()
	hash := make([]byte, 32)

	codes := make([][]byte, len(req.hashes))
	for i, j := 0, 0; i < len(bytecodes); i++ {
		// Find the next hash that we've been served, leaving misses with nils
		hasher.Reset()
		hasher.Write(bytecodes[i])
		hasher.Read(hash)

		for j < len(req.hashes) && !bytes.Equal(hash, req.hashes[j][:]) {
			j++
		}
		if j < len(req.hashes) {
			codes[j] = bytecodes[i]
			j++
			continue
		}
		// We've either ran out of hashes, or got unrequested data
		logger.Warn("Unexpected healing bytecodes", "count", len(bytecodes)-i)
		// Signal this request as failed, and ready for rescheduling
		s.scheduleRevertBytecodeHealRequest(req)
		return errors.New("unexpected healing bytecode")
	}
	// Response validated, send it to the scheduler for filling
	response := &bytecodeHealResponse{
		task:   req.task,
		hashes: req.hashes,
		codes:  codes,
	}
	select {
	case req.deliver <- response:
	case <-req.cancel:
	case <-req.stale:
	}
	return nil
}

// onHealState is a callback method to invoke when a flat state(account
// or storage slot) is downloaded during the healing stage. The flat states
// can be persisted blindly and can be fixed later in the generation stage.
// Note it's not concurrent safe, please handle the concurrent issue outside.
func (s *Syncer) onHealState(paths [][]byte, value []byte) error {
	if len(paths) == 1 {
		var account types.StateAccount
		if err := rlp.DecodeBytes(value, &account); err != nil {
			return nil // Returning the error here would drop the remote peer
		}
		blob := types.SlimAccountRLP(account)
		rawdb.WriteAccountSnapshot(s.stateWriter, common.BytesToHash(paths[0]), blob)
		s.accountHealed += 1
		s.accountHealedBytes += common.StorageSize(1 + common.HashLength + len(blob))
	}
	if len(paths) == 2 {
		rawdb.WriteStorageSnapshot(s.stateWriter, common.BytesToHash(paths[0]), common.BytesToHash(paths[1]), value)
		s.storageHealed += 1
		s.storageHealedBytes += common.StorageSize(1 + 2*common.HashLength + len(value))
	}
	if s.stateWriter.ValueSize() > ethdb.IdealBatchSize {
		s.stateWriter.Write() // It's fine to ignore the error here
		s.stateWriter.Reset()
	}
	return nil
}

// reportHealProgress calculates various status reports and provides it to the user.
func (s *Syncer) reportHealProgress(force bool) {
	// Don't report all the events, just occasionally
	if !force && time.Since(s.logTime) < 8*time.Second {
		return
	}
	s.logTime = time.Now()

	// Create a mega progress report
	var (
		trienode = fmt.Sprintf("%v@%v", log.FormatLogfmtUint64(s.trienodeHealSynced), s.trienodeHealBytes.TerminalString())
		bytecode = fmt.Sprintf("%v@%v", log.FormatLogfmtUint64(s.bytecodeHealSynced), s.bytecodeHealBytes.TerminalString())
		accounts = fmt.Sprintf("%v@%v", log.FormatLogfmtUint64(s.accountHealed), s.accountHealedBytes.TerminalString())
		storage  = fmt.Sprintf("%v@%v", log.FormatLogfmtUint64(s.storageHealed), s.storageHealedBytes.TerminalString())
	)
	log.Info("Syncing: state healing in progress", "accounts", accounts, "slots", storage,
		"codes", bytecode, "nodes", trienode, "pending", s.healer.scheduler.Pending())
}
