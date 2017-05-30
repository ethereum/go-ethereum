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
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
)

type stateReq struct {
	items    []common.Hash
	tasks    map[common.Hash]*stateTask
	timeout  time.Duration
	timer    *time.Timer // fires when timeout has elapsed
	peer     *peer       // the peer that we're requesting from
	response [][]byte    // the response. this is nil for timed-out requests.
}

func (req *stateReq) timedOut() bool {
	return req.response == nil
}

type stateSyncStats struct {
	done    uint64 // number of entries pulled
	pending uint64 // number of pending entries
}

// syncPivotState starts downloading state with the given root hash.
func (d *Downloader) syncState(root common.Hash) *stateSync {
	s := newStateSync(d, root)
	select {
	case d.stateSyncStart <- s:
	case <-d.cancelCh:
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

// runStateSync runs s until it completes or another stateSync is requested.
func (d *Downloader) runStateSync(s *stateSync) *stateSync {
	var (
		activeReqs   = make(map[string]*stateReq)
		finishedReqs []*stateReq
		timeout      = make(chan *stateReq)
	)
	// Cancel active request timers on exit.
	defer func() {
		for _, req := range activeReqs {
			req.timer.Stop()
		}
	}()
	// Run the state sync.
	go s.run()
	defer s.Cancel()

	for {
		// Enable sending of the first buffered element if there is one.
		var deliverReq *stateReq
		var deliverReqCh chan *stateReq
		if len(finishedReqs) > 0 {
			deliverReq = finishedReqs[0]
			deliverReqCh = s.deliver
		}

		select {
		// The stateSync lifecycle:
		case next := <-d.stateSyncStart:
			return next
		case <-s.done:
			return nil

		// Send the next finished request to the current sync:
		case deliverReqCh <- deliverReq:
			finishedReqs = finishedReqs[1:]

		// Handle incoming state packs:
		case pack := <-d.stateCh:
			req := activeReqs[pack.PeerId()]
			if req == nil {
				log.Debug("Unrequested node data", "peer", pack.PeerId(), "len", pack.Items())
				continue
			}
			delete(activeReqs, pack.PeerId())
			req.timer.Stop()
			req.response = pack.(*statePack).states
			finishedReqs = append(finishedReqs, req)

		// Handle timed-out requests:
		case req := <-timeout:
			if activeReqs[req.peer.id] != req {
				continue // ignore old timeout
			}
			finishedReqs = append(finishedReqs, req)
			delete(activeReqs, req.peer.id)

		// Track outgoing state requests:
		case req := <-d.trackStateReq:
			activeReqs[req.peer.id] = req
			req.timer = time.AfterFunc(req.timeout, func() {
				select {
				case timeout <- req:
				case <-s.done:
					// Prevent leaking of timer goroutines in the unlikely case where a
					// timer is fired just before exiting runStateSync.
				}
			})
		}
	}
}

// stateSync schedules requests for downloading a particular state root.
type stateSync struct {
	d              *Downloader
	sched          *state.StateSync
	keccak         hash.Hash
	tasksAvailable map[common.Hash]*stateTask

	deliver    chan *stateReq
	cancel     chan struct{}
	cancelOnce sync.Once
	done       chan struct{}
	err        error // set after done is closed
}

type stateTask struct {
	triedPeers map[string]struct{}
}

func newStateSync(d *Downloader, root common.Hash) *stateSync {
	return &stateSync{
		d:              d,
		sched:          state.NewStateSync(root, d.stateDB),
		keccak:         sha3.NewKeccak256(),
		tasksAvailable: make(map[common.Hash]*stateTask),
		deliver:        make(chan *stateReq),
		cancel:         make(chan struct{}),
		done:           make(chan struct{}, 1),
	}
}

// wait blocks until the sync is done or canceled.
func (s *stateSync) wait() error {
	<-s.done
	return s.err
}

// wait blocks until the sync is done or canceled.
func (s *stateSync) checkDone() (bool, error) {
	select {
	case <-s.done:
		return true, s.err
	default:
		return false, nil
	}
}

func (s *stateSync) Cancel() error {
	s.cancelOnce.Do(func() { close(s.cancel) })
	return s.wait()
}

func (s *stateSync) run() {
	s.err = s.loop()
	close(s.done)
}

func (s *stateSync) loop() error {
	newPeer := make(chan *peer, 200)
	peerSub := s.d.peers.SubscribeNewPeers(newPeer)
	defer peerSub.Unsubscribe()

	for s.sched.Pending() > 0 {
		if err := s.assignTasks(); err != nil {
			return err
		}

		select {
		case <-newPeer:
			// assign new tasks
		case <-s.cancel:
			return errCancelStateFetch
		case req := <-s.deliver:
			// response or timeout
			log.Trace("Received node data response", "peer", req.peer.id, "count", len(req.items), "timeout", req.timedOut())
			req.peer.SetNodeDataIdle(len(req.response))
			procStart := time.Now()
			n, err := s.process(req)
			if err != nil {
				log.Warn("Node data write error", "err", err)
				return err
			}
			if len(req.items) == 1 && req.timedOut() {
				log.Warn("Node data timeout, dropping peer", "peer", req.peer.id)
				s.d.dropPeer(req.peer.id)
			}
			s.updateStats(n, time.Since(procStart))
		}
	}
	return nil
}

func (s *stateSync) sendReq(req *stateReq) {
	req.peer.log.Trace("Requesting new batch of data", "type", "state", "count", len(req.items))
	select {
	case s.d.trackStateReq <- req:
		req.peer.FetchNodeData(req.items)
	case <-s.cancel:
	}
}

func (s *stateSync) assignTasks() error {
	peers, _ := s.d.peers.NodeDataIdlePeers()
	for _, p := range peers {
		cap := p.NodeDataCapacity(s.d.requestRTT())
		req := &stateReq{peer: p, timeout: s.d.requestTTL()}
		if err := s.popTasks(cap, req); err != nil {
			return err
		}
		if len(req.items) > 0 {
			s.sendReq(req)
		}
	}
	return nil
}

func (s *stateSync) popTasks(n int, req *stateReq) error {
	// Refill available tasks from the scheduler.
	if len(s.tasksAvailable) < n {
		new := s.sched.Missing(n - len(s.tasksAvailable))
		for _, hash := range new {
			s.tasksAvailable[hash] = &stateTask{make(map[string]struct{})}
		}
	}
	// Find tasks that haven't been tried with the request's peer.
	req.items = make([]common.Hash, 0, n)
	req.tasks = make(map[common.Hash]*stateTask, n)
	for hash, t := range s.tasksAvailable {
		if len(req.items) == n {
			break
		}
		if _, ok := t.triedPeers[req.peer.id]; ok {
			continue
		}
		t.triedPeers[req.peer.id] = struct{}{}
		req.items = append(req.items, hash)
		req.tasks[hash] = t
		delete(s.tasksAvailable, hash)
	}
	return nil
}

func (s *stateSync) process(req *stateReq) (nproc int, err error) {
	batch := s.d.stateDB.NewBatch()
	for _, blob := range req.response {
		if hash, ok := s.processNodeData(blob, batch); ok {
			nproc++
			delete(req.tasks, hash)
		}
	}
	if err := batch.Write(); err != nil {
		return 0, err
	}
	if nproc > 0 && atomic.LoadUint32(&s.d.fsPivotFails) > 1 {
		log.Trace("Fast-sync progressed, resetting fail counter", "previous", atomic.LoadUint32(&s.d.fsPivotFails))
		atomic.StoreUint32(&s.d.fsPivotFails, 1) // Don't ever reset to 0, as that will unlock the pivot block
	}
	// Put unfulfilled tasks back.
	for hash, task := range req.tasks {
		if len(req.response) > 0 || req.timedOut() {
			// Ensure that the item will be retried if the response contained some data or
			// timed out.
			delete(task.triedPeers, req.peer.id)
		}
		if npeers := s.d.peers.Len(); len(task.triedPeers) >= npeers {
			return nproc, fmt.Errorf("state node %s failed with all peers (%d tries, %d peers)", hash.TerminalString(), len(task.triedPeers), npeers)
		}
		s.tasksAvailable[hash] = task
	}
	return nproc, nil
}

func (s *stateSync) processNodeData(blob []byte, batch ethdb.Batch) (common.Hash, bool) {
	res := trie.SyncResult{Data: blob}
	s.keccak.Reset()
	s.keccak.Write(blob)
	s.keccak.Sum(res.Hash[:0])
	_, _, err := s.sched.Process([]trie.SyncResult{res}, batch)
	return res.Hash, err == nil
}

func (s *stateSync) updateStats(processed int, duration time.Duration) {
	s.d.syncStatsLock.Lock()
	defer s.d.syncStatsLock.Unlock()
	s.d.syncStatsState.pending = uint64(s.sched.Pending())
	s.d.syncStatsState.done += uint64(processed)
	log.Info("Imported new state entries", "count", processed, "elapsed", common.PrettyDuration(duration), "processed", s.d.syncStatsState.done, "pending", s.d.syncStatsState.pending, "retry", len(s.tasksAvailable))
}
