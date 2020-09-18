// Copyright 2017 The go-ethereum Authors
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

package downloader

import (
	"fmt"
	"hash"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"golang.org/x/crypto/sha3"
)

// stateReq represents a batch of state fetch requests grouped together into
// a single data retrieval network packet.
type stateReq struct {
	nItems    uint16                    // Number of items requested for download (max is 384, so uint16 is sufficient)
	trieTasks map[common.Hash]*trieTask // Trie node download tasks to track previous attempts
	codeTasks map[common.Hash]*codeTask // Byte code download tasks to track previous attempts
	timeout   time.Duration             // Maximum round trip time for this to complete
	timer     *time.Timer               // Timer to fire when the RTT timeout expires
	peer      *peerConnection           // Peer that we're requesting from
	delivered time.Time                 // Time when the packet was delivered (independent when we process it)
	response  [][]byte                  // Response data of the peer (nil for timeouts)
	dropped   bool                      // Flag whether the peer dropped off early
}

// timedOut returns if this request timed out.
func (req *stateReq) timedOut() bool {
	return req.response == nil
}

// stateSyncStats is a collection of progress stats to report during a state trie
// sync to RPC requests as well as to display in user logs.
type stateSyncStats struct {
	processed  uint64 // Number of state entries processed
	duplicate  uint64 // Number of state entries downloaded twice
	unexpected uint64 // Number of non-requested state entries received
	pending    uint64 // Number of still pending state entries
}

// syncState starts downloading state with the given root hash.
func (d *Downloader) syncState(root common.Hash) *stateSync {
	// Create the state sync
	s := newStateSync(d, root)
	select {
	case d.stateSyncStart <- s:
		// If we tell the statesync to restart with a new root, we also need
		// to wait for it to actually also start -- when old requests have timed
		// out or been delivered
		<-s.started
	case <-d.quitCh:
		s.err = errCancelStateFetch
		close(s.done)
	}
	return s
}

// stateFetcher manages the active state sync and accepts requests
// on its behalf.
func (d *Downloader) stateFetcher() {
	for {
		select {
		case s := <-d.stateSyncStart:
			for next := s; next != nil; {
				next = d.runStateSync(next)
			}
		case <-d.stateCh:
			// Ignore state responses while no sync is running.
		case <-d.quitCh:
			return
		}
	}
}

// runStateSync runs a state synchronisation until it completes or another root
// hash is requested to be switched over to.
func (d *Downloader) runStateSync(s *stateSync) *stateSync {
	var (
		active   = make(map[string]*stateReq) // Currently in-flight requests
		finished []*stateReq                  // Completed or failed requests
		timeout  = make(chan *stateReq)       // Timed out active requests
	)
	// Run the state sync.
	log.Trace("State sync starting", "root", s.root)
	go s.run()
	defer s.Cancel()

	// Listen for peer departure events to cancel assigned tasks
	peerDrop := make(chan *peerConnection, 1024)
	peerSub := s.d.peers.SubscribePeerDrops(peerDrop)
	defer peerSub.Unsubscribe()

	for {
		// Enable sending of the first buffered element if there is one.
		var (
			deliverReq   *stateReq
			deliverReqCh chan *stateReq
		)
		if len(finished) > 0 {
			deliverReq = finished[0]
			deliverReqCh = s.deliver
		}

		select {
		// The stateSync lifecycle:
		case next := <-d.stateSyncStart:
			d.spindownStateSync(active, finished, timeout, peerDrop)
			return next

		case <-s.done:
			d.spindownStateSync(active, finished, timeout, peerDrop)
			return nil

		// Send the next finished request to the current sync:
		case deliverReqCh <- deliverReq:
			// Shift out the first request, but also set the emptied slot to nil for GC
			copy(finished, finished[1:])
			finished[len(finished)-1] = nil
			finished = finished[:len(finished)-1]

		// Handle incoming state packs:
		case pack := <-d.stateCh:
			// Discard any data not requested (or previously timed out)
			req := active[pack.PeerId()]
			if req == nil {
				log.Debug("Unrequested node data", "peer", pack.PeerId(), "len", pack.Items())
				continue
			}
			// Finalize the request and queue up for processing
			req.timer.Stop()
			req.response = pack.(*statePack).states
			req.delivered = time.Now()

			finished = append(finished, req)
			delete(active, pack.PeerId())

		// Handle dropped peer connections:
		case p := <-peerDrop:
			// Skip if no request is currently pending
			req := active[p.id]
			if req == nil {
				continue
			}
			// Finalize the request and queue up for processing
			req.timer.Stop()
			req.dropped = true
			req.delivered = time.Now()

			finished = append(finished, req)
			delete(active, p.id)

		// Handle timed-out requests:
		case req := <-timeout:
			// If the peer is already requesting something else, ignore the stale timeout.
			// This can happen when the timeout and the delivery happens simultaneously,
			// causing both pathways to trigger.
			if active[req.peer.id] != req {
				continue
			}
			req.delivered = time.Now()
			// Move the timed out data back into the download queue
			finished = append(finished, req)
			delete(active, req.peer.id)

		// Track outgoing state requests:
		case req := <-d.trackStateReq:
			// If an active request already exists for this peer, we have a problem. In
			// theory the trie node schedule must never assign two requests to the same
			// peer. In practice however, a peer might receive a request, disconnect and
			// immediately reconnect before the previous times out. In this case the first
			// request is never honored, alas we must not silently overwrite it, as that
			// causes valid requests to go missing and sync to get stuck.
			if old := active[req.peer.id]; old != nil {
				log.Warn("Busy peer assigned new state fetch", "peer", old.peer.id)
				// Move the previous request to the finished set
				old.timer.Stop()
				old.dropped = true
				old.delivered = time.Now()
				finished = append(finished, old)
			}
			// Start a timer to notify the sync loop if the peer stalled.
			req.timer = time.AfterFunc(req.timeout, func() {
				timeout <- req
			})
			active[req.peer.id] = req
		}
	}
}

// spindownStateSync 'drains' the outstanding requests; some will be delivered and other
// will time out. This is to ensure that when the next stateSync starts working, all peers
// are marked as idle and de facto _are_ idle.
func (d *Downloader) spindownStateSync(active map[string]*stateReq, finished []*stateReq, timeout chan *stateReq, peerDrop chan *peerConnection) {
	log.Trace("State sync spinning down", "active", len(active), "finished", len(finished))
	for len(active) > 0 {
		var (
			req    *stateReq
			reason string
		)
		select {
		// Handle (drop) incoming state packs:
		case pack := <-d.stateCh:
			req = active[pack.PeerId()]
			reason = "delivered"
		// Handle dropped peer connections:
		case p := <-peerDrop:
			req = active[p.id]
			reason = "peerdrop"
		// Handle timed-out requests:
		case req = <-timeout:
			reason = "timeout"
		}
		if req == nil {
			continue
		}
		req.peer.log.Trace("State peer marked idle (spindown)", "req.items", int(req.nItems), "reason", reason)
		req.timer.Stop()
		delete(active, req.peer.id)
		req.peer.SetNodeDataIdle(int(req.nItems), time.Now())
	}
	// The 'finished' set contains deliveries that we were going to pass to processing.
	// Those are now moot, but we still need to set those peers as idle, which would
	// otherwise have been done after processing
	for _, req := range finished {
		req.peer.SetNodeDataIdle(int(req.nItems), time.Now())
	}
}

// stateSync schedules requests for downloading a particular state trie defined
// by a given state root.
type stateSync struct {
	d *Downloader // Downloader instance to access and manage current peerset

	sched  *trie.Sync // State trie sync scheduler defining the tasks
	keccak hash.Hash  // Keccak256 hasher to verify deliveries with

	trieTasks map[common.Hash]*trieTask // Set of trie node tasks currently queued for retrieval
	codeTasks map[common.Hash]*codeTask // Set of byte code tasks currently queued for retrieval

	numUncommitted   int
	bytesUncommitted int

	started chan struct{} // Started is signalled once the sync loop starts

	deliver    chan *stateReq // Delivery channel multiplexing peer responses
	cancel     chan struct{}  // Channel to signal a termination request
	cancelOnce sync.Once      // Ensures cancel only ever gets called once
	done       chan struct{}  // Channel to signal termination completion
	err        error          // Any error hit during sync (set before completion)

	root common.Hash
}

// trieTask represents a single trie node download task, containing a set of
// peers already attempted retrieval from to detect stalled syncs and abort.
type trieTask struct {
	path     [][]byte
	attempts map[string]struct{}
}

// codeTask represents a single byte code download task, containing a set of
// peers already attempted retrieval from to detect stalled syncs and abort.
type codeTask struct {
	attempts map[string]struct{}
}

// newStateSync creates a new state trie download scheduler. This method does not
// yet start the sync. The user needs to call run to initiate.
func newStateSync(d *Downloader, root common.Hash) *stateSync {
	return &stateSync{
		d:         d,
		sched:     state.NewStateSync(root, d.stateDB, d.stateBloom),
		keccak:    sha3.NewLegacyKeccak256(),
		trieTasks: make(map[common.Hash]*trieTask),
		codeTasks: make(map[common.Hash]*codeTask),
		deliver:   make(chan *stateReq),
		cancel:    make(chan struct{}),
		done:      make(chan struct{}),
		started:   make(chan struct{}),
		root:      root,
	}
}

// run starts the task assignment and response processing loop, blocking until
// it finishes, and finally notifying any goroutines waiting for the loop to
// finish.
func (s *stateSync) run() {
	s.err = s.loop()
	close(s.done)
}

// Wait blocks until the sync is done or canceled.
func (s *stateSync) Wait() error {
	<-s.done
	return s.err
}

// Cancel cancels the sync and waits until it has shut down.
func (s *stateSync) Cancel() error {
	s.cancelOnce.Do(func() { close(s.cancel) })
	return s.Wait()
}

// loop is the main event loop of a state trie sync. It it responsible for the
// assignment of new tasks to peers (including sending it to them) as well as
// for the processing of inbound data. Note, that the loop does not directly
// receive data from peers, rather those are buffered up in the downloader and
// pushed here async. The reason is to decouple processing from data receipt
// and timeouts.
func (s *stateSync) loop() (err error) {
	close(s.started)
	// Listen for new peer events to assign tasks to them
	newPeer := make(chan *peerConnection, 1024)
	peerSub := s.d.peers.SubscribeNewPeers(newPeer)
	defer peerSub.Unsubscribe()
	defer func() {
		cerr := s.commit(true)
		if err == nil {
			err = cerr
		}
	}()

	// Keep assigning new tasks until the sync completes or aborts
	for s.sched.Pending() > 0 {
		if err = s.commit(false); err != nil {
			return err
		}
		s.assignTasks()
		// Tasks assigned, wait for something to happen
		select {
		case <-newPeer:
			// New peer arrived, try to assign it download tasks

		case <-s.cancel:
			return errCancelStateFetch

		case <-s.d.cancelCh:
			return errCanceled

		case req := <-s.deliver:
			// Response, disconnect or timeout triggered, drop the peer if stalling
			log.Trace("Received node data response", "peer", req.peer.id, "count", len(req.response), "dropped", req.dropped, "timeout", !req.dropped && req.timedOut())
			if req.nItems <= 2 && !req.dropped && req.timedOut() {
				// 2 items are the minimum requested, if even that times out, we've no use of
				// this peer at the moment.
				log.Warn("Stalling state sync, dropping peer", "peer", req.peer.id)
				if s.d.dropPeer == nil {
					// The dropPeer method is nil when `--copydb` is used for a local copy.
					// Timeouts can occur if e.g. compaction hits at the wrong time, and can be ignored
					req.peer.log.Warn("Downloader wants to drop peer, but peerdrop-function is not set", "peer", req.peer.id)
				} else {
					s.d.dropPeer(req.peer.id)

					// If this peer was the master peer, abort sync immediately
					s.d.cancelLock.RLock()
					master := req.peer.id == s.d.cancelPeer
					s.d.cancelLock.RUnlock()

					if master {
						s.d.cancel()
						return errTimeout
					}
				}
			}
			// Process all the received blobs and check for stale delivery
			delivered, err := s.process(req)
			req.peer.SetNodeDataIdle(delivered, req.delivered)
			if err != nil {
				log.Warn("Node data write error", "err", err)
				return err
			}
		}
	}
	return nil
}

func (s *stateSync) commit(force bool) error {
	if !force && s.bytesUncommitted < ethdb.IdealBatchSize {
		return nil
	}
	start := time.Now()
	b := s.d.stateDB.NewBatch()
	if err := s.sched.Commit(b); err != nil {
		return err
	}
	if err := b.Write(); err != nil {
		return fmt.Errorf("DB write error: %v", err)
	}
	s.updateStats(s.numUncommitted, 0, 0, time.Since(start))
	s.numUncommitted = 0
	s.bytesUncommitted = 0
	return nil
}

// assignTasks attempts to assign new tasks to all idle peers, either from the
// batch currently being retried, or fetching new data from the trie sync itself.
func (s *stateSync) assignTasks() {
	// Iterate over all idle peers and try to assign them state fetches
	peers, _ := s.d.peers.NodeDataIdlePeers()
	for _, p := range peers {
		// Assign a batch of fetches proportional to the estimated latency/bandwidth
		cap := p.NodeDataCapacity(s.d.requestRTT())
		req := &stateReq{peer: p, timeout: s.d.requestTTL()}

		nodes, _, codes := s.fillTasks(cap, req)

		// If the peer was assigned tasks to fetch, send the network request
		if len(nodes)+len(codes) > 0 {
			req.peer.log.Trace("Requesting batch of state data", "nodes", len(nodes), "codes", len(codes), "root", s.root)
			select {
			case s.d.trackStateReq <- req:
				req.peer.FetchNodeData(append(nodes, codes...)) // Unified retrieval under eth/6x
			case <-s.cancel:
			case <-s.d.cancelCh:
			}
		}
	}
}

// fillTasks fills the given request object with a maximum of n state download
// tasks to send to the remote peer.
func (s *stateSync) fillTasks(n int, req *stateReq) (nodes []common.Hash, paths []trie.SyncPath, codes []common.Hash) {
	// Refill available tasks from the scheduler.
	if fill := n - (len(s.trieTasks) + len(s.codeTasks)); fill > 0 {
		nodes, paths, codes := s.sched.Missing(fill)
		for i, hash := range nodes {
			s.trieTasks[hash] = &trieTask{
				path:     paths[i],
				attempts: make(map[string]struct{}),
			}
		}
		for _, hash := range codes {
			s.codeTasks[hash] = &codeTask{
				attempts: make(map[string]struct{}),
			}
		}
	}
	// Find tasks that haven't been tried with the request's peer. Prefer code
	// over trie nodes as those can be written to disk and forgotten about.
	nodes = make([]common.Hash, 0, n)
	paths = make([]trie.SyncPath, 0, n)
	codes = make([]common.Hash, 0, n)

	req.trieTasks = make(map[common.Hash]*trieTask, n)
	req.codeTasks = make(map[common.Hash]*codeTask, n)

	for hash, t := range s.codeTasks {
		// Stop when we've gathered enough requests
		if len(nodes)+len(codes) == n {
			break
		}
		// Skip any requests we've already tried from this peer
		if _, ok := t.attempts[req.peer.id]; ok {
			continue
		}
		// Assign the request to this peer
		t.attempts[req.peer.id] = struct{}{}
		codes = append(codes, hash)
		req.codeTasks[hash] = t
		delete(s.codeTasks, hash)
	}
	for hash, t := range s.trieTasks {
		// Stop when we've gathered enough requests
		if len(nodes)+len(codes) == n {
			break
		}
		// Skip any requests we've already tried from this peer
		if _, ok := t.attempts[req.peer.id]; ok {
			continue
		}
		// Assign the request to this peer
		t.attempts[req.peer.id] = struct{}{}

		nodes = append(nodes, hash)
		paths = append(paths, t.path)

		req.trieTasks[hash] = t
		delete(s.trieTasks, hash)
	}
	req.nItems = uint16(len(nodes) + len(codes))
	return nodes, paths, codes
}

// process iterates over a batch of delivered state data, injecting each item
// into a running state sync, re-queuing any items that were requested but not
// delivered. Returns whether the peer actually managed to deliver anything of
// value, and any error that occurred.
func (s *stateSync) process(req *stateReq) (int, error) {
	// Collect processing stats and update progress if valid data was received
	duplicate, unexpected, successful := 0, 0, 0

	defer func(start time.Time) {
		if duplicate > 0 || unexpected > 0 {
			s.updateStats(0, duplicate, unexpected, time.Since(start))
		}
	}(time.Now())

	// Iterate over all the delivered data and inject one-by-one into the trie
	for _, blob := range req.response {
		hash, err := s.processNodeData(blob)
		switch err {
		case nil:
			s.numUncommitted++
			s.bytesUncommitted += len(blob)
			successful++
		case trie.ErrNotRequested:
			unexpected++
		case trie.ErrAlreadyProcessed:
			duplicate++
		default:
			return successful, fmt.Errorf("invalid state node %s: %v", hash.TerminalString(), err)
		}
		// Delete from both queues (one delivery is enough for the syncer)
		delete(req.trieTasks, hash)
		delete(req.codeTasks, hash)
	}
	// Put unfulfilled tasks back into the retry queue
	npeers := s.d.peers.Len()
	for hash, task := range req.trieTasks {
		// If the node did deliver something, missing items may be due to a protocol
		// limit or a previous timeout + delayed delivery. Both cases should permit
		// the node to retry the missing items (to avoid single-peer stalls).
		if len(req.response) > 0 || req.timedOut() {
			delete(task.attempts, req.peer.id)
		}
		// If we've requested the node too many times already, it may be a malicious
		// sync where nobody has the right data. Abort.
		if len(task.attempts) >= npeers {
			return successful, fmt.Errorf("trie node %s failed with all peers (%d tries, %d peers)", hash.TerminalString(), len(task.attempts), npeers)
		}
		// Missing item, place into the retry queue.
		s.trieTasks[hash] = task
	}
	for hash, task := range req.codeTasks {
		// If the node did deliver something, missing items may be due to a protocol
		// limit or a previous timeout + delayed delivery. Both cases should permit
		// the node to retry the missing items (to avoid single-peer stalls).
		if len(req.response) > 0 || req.timedOut() {
			delete(task.attempts, req.peer.id)
		}
		// If we've requested the node too many times already, it may be a malicious
		// sync where nobody has the right data. Abort.
		if len(task.attempts) >= npeers {
			return successful, fmt.Errorf("byte code %s failed with all peers (%d tries, %d peers)", hash.TerminalString(), len(task.attempts), npeers)
		}
		// Missing item, place into the retry queue.
		s.codeTasks[hash] = task
	}
	return successful, nil
}

// processNodeData tries to inject a trie node data blob delivered from a remote
// peer into the state trie, returning whether anything useful was written or any
// error occurred.
func (s *stateSync) processNodeData(blob []byte) (common.Hash, error) {
	res := trie.SyncResult{Data: blob}
	s.keccak.Reset()
	s.keccak.Write(blob)
	s.keccak.Sum(res.Hash[:0])
	err := s.sched.Process(res)
	return res.Hash, err
}

// updateStats bumps the various state sync progress counters and displays a log
// message for the user to see.
func (s *stateSync) updateStats(written, duplicate, unexpected int, duration time.Duration) {
	s.d.syncStatsLock.Lock()
	defer s.d.syncStatsLock.Unlock()

	s.d.syncStatsState.pending = uint64(s.sched.Pending())
	s.d.syncStatsState.processed += uint64(written)
	s.d.syncStatsState.duplicate += uint64(duplicate)
	s.d.syncStatsState.unexpected += uint64(unexpected)

	if written > 0 || duplicate > 0 || unexpected > 0 {
		log.Info("Imported new state entries", "count", written, "elapsed", common.PrettyDuration(duration), "processed", s.d.syncStatsState.processed, "pending", s.d.syncStatsState.pending, "trieretry", len(s.trieTasks), "coderetry", len(s.codeTasks), "duplicate", s.d.syncStatsState.duplicate, "unexpected", s.d.syncStatsState.unexpected)
	}
	if written > 0 {
		rawdb.WriteFastTrieProgress(s.d.stateDB, s.d.syncStatsState.processed)
	}
}
