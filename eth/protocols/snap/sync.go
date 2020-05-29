// Copyright 2020 The go-ethereum Authors
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
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"golang.org/x/crypto/sha3"
)

var (
	// emptyRoot is the known root hash of an empty trie.
	emptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	// emptyCode is the known hash of the empty EVM bytecode.
	emptyCode = crypto.Keccak256Hash(nil)
)

const (
	// maxRequestSize is the maximum number of bytes to request from a remote peer.
	maxRequestSize = 512 * 1024

	// maxCodeRequestCount is the maximum number of bytecode blobs to request in a
	// single query. If this number is too low, we're not filling responses fully
	// and waste round trip times. If it's too high, we're capping responses and
	// waste bandwidth.
	//
	// Depoyed bytecodes are currently capped at 24KB, so the minimum request
	// size should be maxRequestSize / 24K. Assuming that most contracts do not
	// come close to that, requesting 4x should be a good approximation.
	maxCodeRequestCount = maxRequestSize / (24 * 1024) * 4

	// requestTimeout is the maximum time a peer is allowed to spend on serving
	// a single network request.
	requestTimeout = 10 * time.Second // TODO(karalabe): Make it dynamic ala fast-sync?

	// accountConcurrency is the number of
	accountConcurrency = 16
)

// accountRequest tracks a pending account range request to ensure responses are
// to actual requests and to validate any security constraints.
type accountRequest struct {
	peer string // Peer to which this request is assigned
	id   uint64 // Request ID of this request

	cancel  chan struct{} // Channel to track sync cancellation
	timeout *time.Timer   // Timer to track delivery timeout

	task *accountTask // Task which this request is filling
}

// accountResponse is an already Merkle-verified remote response to an account
// range request. It contains the subtrie for the requested account range and
// the database that's going to be filled with the internal nodes on commit.
type accountResponse struct {
	task *accountTask // Task which this response is filling

	hashes   []common.Hash    // Account hashes in the returned range
	accounts []*state.Account // Expanded accounts in the returned range

	nodes  ethdb.KeyValueStore      // Database containing the reconstructed trie nodes
	trie   *trie.Trie               // Reconstructed trie to reject incomplete account paths
	bounds map[common.Hash]struct{} // Boundary nodes to avoid persisting (incomplete)
}

// byteCodesRequest tracks a pending bytecode request to ensure responses are to
// actual requests and to validate any security constraints.
type bytecodeRequest struct {
	peer string // Peer to which this request is assigned
	id   uint64 // Request ID of this request

	cancel  chan struct{} // Channel to track sync cancellation
	timeout *time.Timer   // Timer to track delivery timeout

	task   *accountTask  // Task which this request is filling
	hashes []common.Hash // Bytecode hashes to validate responses
}

// bytecodeResponse is an already verified remote response to a bytecode request.
type bytecodeResponse struct {
	task *accountTask // Task which this response is filling

	hashes []common.Hash // Hashes of the bytecode to avoid double hashing
	codes  [][]byte      // Actual bytecodes to store into the database (nil = missing)
}

// storageRequest tracks a pending storage range request to ensure responses are
// to actual requests and to validate any security constraints.
type storageRequest struct {
	peer string // Peer to which this request is assigned
	id   uint64 // Request ID of this request

	cancel  chan struct{} // Channel to track sync cancellation
	timeout *time.Timer   // Timer to track delivery timeout

	root   common.Hash // Storage trie root hash to prove
	origin common.Hash // Origin slot to guarantee overlaps
}

// storageResponse is an already Merkle-verified remote response to a storage
// range request. It contains the subtrie for the requested storage range and
// the database that's going to be filled with the internal nodes on commit.
type storageResponse struct {
	task *accountTask // Task which this response is filling

	hashes []common.Hash // Storage slot hashes in the returned range
	slots  [][]byte      // Storage slot values in the returned range

	nodes  ethdb.KeyValueStore      // Database containing the reconstructed trie nodes
	bounds map[common.Hash]struct{} // Boundary nodes to avoid persisting (incomplete)
	last   common.Hash              // Last returned slot, acts as the next query origin
}

// accountTask represents the sync task for a chunk of the account snapshot.
type accountTask struct {
	// These fields get serialized to leveldb on shutdown
	Next     common.Hash    // Next account to sync in this interval
	Last     common.Hash    // Last account to sync in this interval
	SubTasks []*storageTask // Storage intervals needing fetching for the origin account

	// These fields are internals used during runtime
	req *accountRequest  // Pending request to fill this task
	res *accountResponse // Validate response filling this task

	needCode  []bool // Flags whether the filling accounts need code retrieval
	needState []bool // Flags whether the filling accounts need storage retrieval
	needHeal  []bool // Flags whether the filling accounts's state was chunked and need healing

	codeTasks map[common.Hash]struct{} // Code hashes that need retrieval
}

// accountTask represents the sync task for a chunk of the storage snapshot.
type storageTask struct {
	Next common.Hash // Next account to sync in this interval
	Last common.Hash // Last account to sync in this interval
}

// Syncer is an Ethereum account and storage trie syncer based on snapshots and
// the  snap protocol. It's purpose is to download all the accounts and storage
// slots from remote peers and reassemble chunks of the state trie, on top of
// which a state sync can be run to fix any gaps / overlaps.
//
// Every network request has a variety of failure events:
//   - The peer disconnects after task assignment, failing to send the request
//   - The peer disconnects after sending the request, before delivering on it
//   - The peer remains connected, but does not deliver a response in time
//   - The peer delivers a stale response after a previous timeout
//   - The peer delivers a refusal to serve the requested state
type Syncer struct {
	db    ethdb.KeyValueStore // Database to store the trie nodes into (and dedup)
	bloom *trie.SyncBloom     // Bloom filter to deduplicate nodes for state fixup

	root   common.Hash   // Current state trie root being synced
	update chan struct{} // Notification channel for possible sync progression

	peers    map[string]*Peer // Currently active peers to download from
	peerJoin *event.Feed      // Event feed to react to peers joining
	peerDrop *event.Feed      // Event feed to react to peers dropping

	statelessPeers map[string]struct{} // Peers that failed to deliver state data
	accountIdlers  map[string]struct{} // Peers that aren't serving account requests
	bytecodeIdlers map[string]struct{} // Peers that aren't serving bytecode requests
	storageIdlers  map[string]struct{} // Peers that aren't serving storage requests
	trienodeIdlers map[string]struct{} // Peers that aren't serving trie node requests

	accountReqs  map[uint64]*accountRequest  // Account requests currently running
	bytecodeReqs map[uint64]*bytecodeRequest // Bytecode requests currently running
	storageReqs  map[uint64]*storageRequest  // Storage requests currently running

	accountReqFails  chan *accountRequest   // Failed account range requests to revert
	bytecodeReqFails chan *bytecodeRequest  // Failed bytecode requests to revert
	storageReqFails  chan *storageRequest   // Failed storage requests to revert
	accountResps     chan *accountResponse  // Account sub-tries to integrate into the database
	bytecodeResps    chan *bytecodeResponse // Bytecodes to integrate into the database
	storageResps     chan *storageResponse  // Storage sub-tries to integrate into the database

	accountSynced  uint64             // Number of accounts downloaded
	accountBytes   common.StorageSize // Number of account trie bytes persisted to disk
	bytecodeSynced uint64             // Number of bytecodes downloaded
	bytecodeBytes  common.StorageSize // Number of bytecode bytes downloaded
	storageSynced  uint64             // Number of storage slots downloaded
	storageBytes   common.StorageSize // Number of storage trie bytes persisted to disk

	startTime time.Time   // Time instance when snapshot sync started
	startAcc  common.Hash // Account hash where sync started from
	logTime   time.Time   // Time instance when status was last reported

	pend sync.WaitGroup // Tracks network request goroutines for graceful shutdown
	lock sync.RWMutex   // Protects fields that can change outside of sync (peers, reqs, root)
}

func NewSyncer(db ethdb.KeyValueStore, bloom *trie.SyncBloom) *Syncer {
	return &Syncer{
		db:    db,
		bloom: bloom,

		peers:    make(map[string]*Peer),
		peerJoin: new(event.Feed),
		peerDrop: new(event.Feed),
		update:   make(chan struct{}, 1),

		accountIdlers:  make(map[string]struct{}),
		storageIdlers:  make(map[string]struct{}),
		bytecodeIdlers: make(map[string]struct{}),
		trienodeIdlers: make(map[string]struct{}),

		accountReqs:  make(map[uint64]*accountRequest),
		storageReqs:  make(map[uint64]*storageRequest),
		bytecodeReqs: make(map[uint64]*bytecodeRequest),

		accountReqFails:  make(chan *accountRequest),
		storageReqFails:  make(chan *storageRequest),
		bytecodeReqFails: make(chan *bytecodeRequest),

		accountResps:  make(chan *accountResponse),
		storageResps:  make(chan *storageResponse),
		bytecodeResps: make(chan *bytecodeResponse),
	}
}

// Register injects a new data source into the syncer's peerset.
func (s *Syncer) Register(peer *Peer) error {
	// Make sure the peer is not registered yet
	s.lock.Lock()
	if _, ok := s.peers[peer.id]; ok {
		log.Error("Snap peer already registered", "id", peer.id)

		s.lock.Unlock()
		return errors.New("already registered")
	}
	s.peers[peer.id] = peer

	// Mark the peer as idle, even if no sync is running
	s.accountIdlers[peer.id] = struct{}{}
	s.storageIdlers[peer.id] = struct{}{}
	s.bytecodeIdlers[peer.id] = struct{}{}
	s.trienodeIdlers[peer.id] = struct{}{}
	s.lock.Unlock()

	// Notify any active syncs that a new peer can be assigned data
	s.peerJoin.Send(peer.id)
	return nil
}

// Unregister injects a new data source into the syncer's peerset.
func (s *Syncer) Unregister(id string) error {
	// Remove all traces of the peer from the registry
	s.lock.Lock()
	if _, ok := s.peers[id]; !ok {
		log.Error("Snap peer not registered", "id", id)
		return errors.New("not registered")
	}
	delete(s.peers, id)

	// Remove status markers, even if no sync is running
	delete(s.statelessPeers, id)

	delete(s.accountIdlers, id)
	delete(s.storageIdlers, id)
	delete(s.bytecodeIdlers, id)
	delete(s.trienodeIdlers, id)
	s.lock.Unlock()

	// Notify any active syncs that pending requests need to be reverted
	s.peerDrop.Send(id)
	return nil
}

// Sync starts (or resumes a previous) sync cycle to iterate over an state trie
// with the given root and reconstruct the nodes based on the snapshot leaves.
// Previously downloaded segments will not be redownloaded of fixed, rather any
// errors will be healed after the leaves are fully accumulated.
func (s *Syncer) Sync(root common.Hash, cancel chan struct{}) error {
	// Move the trie root from any previous value, revert stateless markers for
	// any peers and initialize the syncer if it was not yet run
	s.lock.Lock()
	s.root = root
	s.statelessPeers = make(map[string]struct{})
	s.lock.Unlock()

	if s.startTime == (time.Time{}) {
		s.startTime = time.Now()
	}
	// Retrieve the previous sync status from LevelDB and abort if already synced
	tasks := s.loadSyncStatus()
	if len(tasks) == 0 {
		log.Debug("Snapshot sync already completed")
		return nil
	}
	defer func() { // Persist any progress, independent of failure
		s.saveSyncStatus(tasks)
	}()

	log.Debug("Starting snapshot sync cycle", "root", root)
	defer s.report(true)

	// Whether sync completed or not, disregard any future packets
	defer func() {
		s.lock.Lock()
		s.accountReqs = make(map[uint64]*accountRequest)
		s.storageReqs = make(map[uint64]*storageRequest)
		s.bytecodeReqs = make(map[uint64]*bytecodeRequest)
		s.lock.Unlock()
	}()
	// Keep scheduling sync tasks
	peerJoin := make(chan string, 16)
	peerJoinSub := s.peerJoin.Subscribe(peerJoin)
	defer peerJoinSub.Unsubscribe()

	peerDrop := make(chan string, 16)
	peerDropSub := s.peerDrop.Subscribe(peerDrop)
	defer peerDropSub.Unsubscribe()

	for {
		// Assign all the data retrieval tasks to any free peers
		s.assignAccountTasks(tasks, cancel)
		s.assignStorageTasks(tasks, cancel)
		s.assignBytecodeTasks(tasks, cancel)

		// Wait for something to happen
		select {
		case <-s.update:
			// Something happened (new peer, delivery, timeout), recheck tasks
		case <-peerJoin:
			// A new peer joined, try to schedule it new tasks
		case id := <-peerDrop:
			s.revertRequests(id)
		case <-cancel:
			return nil

		case req := <-s.accountReqFails:
			s.revertAccountRequest(req)
		case req := <-s.bytecodeReqFails:
			s.revertBytecodeRequest(req)

		case res := <-s.accountResps:
			s.processAccountResponse(res)
		case res := <-s.bytecodeResps:
			s.processBytecodeResponse(res)
		}
	}
}

// loadSyncStatus retrieves a previously aborted sync status from the database,
// or generates a fresh one if none is available.
func (s *Syncer) loadSyncStatus() []*accountTask {
	var tasks []*accountTask

	if status := rawdb.ReadSanpshotSyncStatus(s.db); status != nil {
		if err := rlp.DecodeBytes(status, &tasks); err != nil {
			log.Error("Failed to decode snap sync status", "err", err)
		} else {
			for _, task := range tasks {
				log.Debug("Scheduled account sync task", "from", task.Next, "last", task.Last)
			}
			return tasks
		}
	}
	// Either we've failed to decode the previus state, or there was none.
	// Start a fresh sync by chunking up the account range and scheduling
	// them for retrieval.
	var next common.Hash
	step := new(big.Int).Sub(
		new(big.Int).Div(
			new(big.Int).Exp(common.Big2, common.Big256, nil),
			big.NewInt(accountConcurrency),
		), common.Big1,
	)
	for i := 0; i < accountConcurrency; i++ {
		last := common.BigToHash(new(big.Int).Add(next.Big(), step))
		if i == accountConcurrency-1 {
			// Make sure we don't overflow if the step is not a proper divisor
			last = common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
		}
		tasks = append(tasks, &accountTask{
			Next: next,
			Last: last,
		})
		log.Debug("Created account sync task", "from", next, "last", last)
		next = common.BigToHash(new(big.Int).Add(last.Big(), common.Big1))
	}
	return tasks
}

// saveSyncStatus marshals the remaining sync tasks into leveldb.
func (s *Syncer) saveSyncStatus(tasks []*accountTask) {
	status, err := rlp.EncodeToBytes(tasks)
	if err != nil {
		panic(err) // This can only fail during implementation
	}
	rawdb.WriteSnapshotSyncStatus(s.db, status)
}

// assignAccountTasks attempts to match idle peers to pending account range
// retrievals.
func (s *Syncer) assignAccountTasks(tasks []*accountTask, cancel chan struct{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for id := range s.accountIdlers {
		// If the peer rejected a query in this sync cycle, don't bother asking
		// again for anything, it's either out of sync or already pruned
		if _, ok := s.statelessPeers[id]; ok {
			continue
		}
		for _, task := range tasks {
			// Skip any tasks already filling
			if task.req != nil || task.res != nil {
				continue
			}
			// Task not yet done, allocate a unique request id
			var reqid uint64
			for {
				reqid = uint64(rand.Int63())
				if reqid == 0 {
					continue
				}
				if _, ok := s.accountReqs[reqid]; ok {
					continue
				}
				break
			}
			// Generate the network query and send it to the peer
			task.req = &accountRequest{
				peer:   id,
				id:     reqid,
				task:   task,
				cancel: cancel,
			}
			task.req.timeout = time.AfterFunc(requestTimeout, func() {
				log.Debug("Account range request timed out")
				select {
				case s.accountReqFails <- task.req:
				default:
				}
			})
			s.accountReqs[reqid] = task.req
			delete(s.accountIdlers, id)

			s.pend.Add(1)
			go func(peer *Peer, task *accountTask) {
				defer s.pend.Done()

				// Attempt to send the remote request and revert if it fails
				if err := peer.RequestAccountRange(reqid, s.root, task.Next, maxRequestSize); err != nil {
					peer.Log().Debug("Failed to request account range", "err", err)
					select {
					case s.accountReqFails <- task.req:
					default:
					}
				}
				// Request successfully sent, start a
			}(s.peers[id], task) // We're in the lock, peers[id] surely exists

			// Task assigned, abort scanning for new tasks
			break
		}
	}
}

// assignBytecodeTasks attempts to match idle peers to pending code retrievals.
func (s *Syncer) assignBytecodeTasks(tasks []*accountTask, cancel chan struct{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for id := range s.bytecodeIdlers {
		// If the peer rejected a query in this sync cycle, don't bother asking
		// again for anything, it's either out of sync or already pruned
		if _, ok := s.statelessPeers[id]; ok {
			continue
		}
		for _, task := range tasks {
			// Skip any tasks not in the bytecode retrieval phase
			if task.res == nil {
				continue
			}
			// Skip tasks that are already retrieving (or done with) all codes
			if len(task.codeTasks) == 0 {
				continue
			}
			// Task not yet done, allocate a unique request id
			var reqid uint64
			for {
				reqid = uint64(rand.Int63())
				if reqid == 0 {
					continue
				}
				if _, ok := s.bytecodeReqs[reqid]; ok {
					continue
				}
				break
			}
			// Generate the network query and send it to the peer
			hashes := make([]common.Hash, 0, maxCodeRequestCount)
			for hash := range task.codeTasks {
				delete(task.codeTasks, hash)
				hashes = append(hashes, hash)
				if len(hashes) >= maxCodeRequestCount {
					break
				}
			}
			req := &bytecodeRequest{
				peer:   id,
				task:   task,
				hashes: hashes,
				cancel: cancel,
			}
			req.timeout = time.AfterFunc(requestTimeout, func() {
				log.Debug("Bytecode request timed out")
				select {
				case s.bytecodeReqFails <- req:
				default:
				}
			})
			s.bytecodeReqs[reqid] = req
			delete(s.bytecodeIdlers, id)

			s.pend.Add(1)
			go func(peer *Peer, task *accountTask) {
				defer s.pend.Done()

				// Attempt to send the remote request and revert if it fails
				if err := peer.RequestByteCodes(reqid, hashes, maxRequestSize); err != nil {
					log.Debug("Failed to request bytecodes", "err", err)
					select {
					case s.bytecodeReqFails <- req:
					default:
					}
				}
				// Request successfully sent, start a
			}(s.peers[id], task) // We're in the lock, peers[id] surely exists

			// Task assigned, abort scanning for new tasks
			break
		}
	}
}

// assignStorageTasks attempts to match idle peers to pending storage range
// retrievals.
func (s *Syncer) assignStorageTasks(tasks []*accountTask, cancel chan struct{}) {
	/*s.lock.Lock()
	defer s.lock.Unlock()

	for id := range s.storageIdlers {
		// If the peer rejected a query in this sync cycle, don't bother asking
		// again for anything, it's either out of sync or already pruned
		if _, ok := s.statelessPeers[id]; ok {
			continue
		}
		for _, task := range tasks {
			// Skip any tasks not in the storage retrieval phase
			if task.res == nil {
				continue
			}
			// Task not yet done, allocate a unique request id
			var reqid uint64
			for {
				reqid = uint64(rand.Int63())
				if reqid == 0 {
					continue
				}
				if _, ok := s.accountReqs[reqid]; ok {
					continue
				}
				break
			}
			// Generate the network query and send it to the peer
			task.req = &accountRequest{
				task:   task,
				cancel: cancel,
				timeout: time.AfterFunc(requestTimeout, func() {
					// Request timed out, stop tracking it and notify the scheduler.
					// There's no need to mark the peer idle, if it's overloaded
					// it will eventually come around with the stale reply.
					s.lock.Lock()
					delete(s.accountReqs, reqid)
					task.req = nil
					s.lock.Unlock()

					// Notify the scheduler that something went wrong. There is
					// nothing it can do about the failed peer, but there might
					// be other free peers to reassing the task to.
					select {
					case s.update <- struct{}{}:
					default:
					}
				}),
			}
			s.accountReqs[reqid] = task.req
			delete(s.accountIdlers, id)

			s.pend.Add(1)
			go func(peer *Peer, task *accountTask) {
				defer s.pend.Done()

				// Attempt to send the remote request and revert if it fails
				if err := peer.RequestAccountRange(reqid, s.root, task.Next, maxRequestSize); err != nil {
					log.Debug("Failed to request account range", "err", err)

					// Request failed, stop tracking it and notify the scheduler.
					// There's no need to mark the peer idle, since at this point
					// the only failure possibility is disconnection.
					s.lock.Lock()
					delete(s.accountReqs, reqid)
					task.req.timeout.Stop()
					task.req = nil
					s.lock.Unlock()

					// Notify the scheduler that something went wrong. There is
					// nothing it can do about the failed peer, but there might
					// be other free peers to reassing the task to.
					select {
					case s.update <- struct{}{}:
					default:
					}
				}
				// Request successfully sent, start a
			}(s.peers[id], task) // We're in the lock, peers[id] surely exists
		}
	}*/
}

// revertRequests locates all the currently pending reuqests from a particular
// peer and reverts them, rescheduling for others to fulfill.
func (s *Syncer) revertRequests(peer string) {
	// Gather the requests first, revertals need the lock too
	s.lock.Lock()
	var accountReqs []*accountRequest
	for _, req := range s.accountReqs {
		if req.peer == peer {
			accountReqs = append(accountReqs, req)
		}
	}
	var bytecodeReqs []*bytecodeRequest
	for _, req := range s.bytecodeReqs {
		if req.peer == peer {
			bytecodeReqs = append(bytecodeReqs, req)
		}
	}
	var storageReqs []*storageRequest
	for _, req := range s.storageReqs {
		if req.peer == peer {
			storageReqs = append(storageReqs, req)
		}
	}
	s.lock.Unlock()

	// Revert all the requests matching the peer
	for _, req := range accountReqs {
		s.revertAccountRequest(req)
	}
	for _, req := range bytecodeReqs {
		s.revertBytecodeRequest(req)
	}
	for _, req := range storageReqs {
		s.revertStorageRequest(req)
	}
}

// revertAccountRequest cleans up an account range request and returns all failed
// retrieval tasks to the scheduler for reassignment.
func (s *Syncer) revertAccountRequest(req *accountRequest) {
	// Remove the request from the tracked set
	s.lock.Lock()
	delete(s.accountReqs, req.id)
	s.lock.Unlock()

	// If there's a timeout timer still running, abort it and mark the account
	// task as not-pending, ready for resheduling
	req.timeout.Stop()
	req.task.req = nil
}

// revertBytecodeRequest cleans up an bytecode request and returns all failed
// retrieval tasks to the scheduler for reassignment.
func (s *Syncer) revertBytecodeRequest(req *bytecodeRequest) {
	// Remove the request from the tracked set
	s.lock.Lock()
	delete(s.bytecodeReqs, req.id)
	s.lock.Unlock()

	// If there's a timeout timer still running, abort it and mark the code
	// retrievals as not-pending, ready for resheduling
	req.timeout.Stop()
	for _, hash := range req.hashes {
		req.task.codeTasks[hash] = struct{}{}
	}
}

// revertStorageRequest cleans up a storage range request and returns all failed
// retrieval tasks to the scheduler for reassignment.
func (s *Syncer) revertStorageRequest(req *storageRequest) {
	// Remove the request from the tracked set
	s.lock.Lock()
	delete(s.storageReqs, req.id)
	s.lock.Unlock()

	// If there's a timeout timer still running, abort it and mark the storage
	// task as not-pending, ready for resheduling
	req.timeout.Stop()
}

// processAccountResponse integrates an already validated account range response
// into the account tasks.
func (s *Syncer) processAccountResponse(res *accountResponse) {
	s.accountSynced += uint64(len(res.accounts))

	// Switch the task from pending to filling
	res.task.req = nil
	res.task.res = res

	// Ensure that the response doesn't overflow into the subsequent task, otherwise
	// we run the risk of downloading a huge contract twice.
	last := res.task.Last.Big()
	for i, hash := range res.hashes {
		if hash.Big().Cmp(last) > 0 {
			res.hashes = res.hashes[:i]
			res.accounts = res.accounts[:i]
			break
		}
	}
	// Itereate over all the accounts and assemble which ones need further sub-
	// filling before the entire account range can be persisted.
	res.task.needCode = make([]bool, len(res.accounts))
	res.task.needState = make([]bool, len(res.accounts))
	res.task.needHeal = make([]bool, len(res.accounts))

	res.task.codeTasks = make(map[common.Hash]struct{})

	var incomplete bool
	for i, account := range res.accounts {
		// Check if the account is a contract with an unknown code
		if !bytes.Equal(account.CodeHash, emptyCode[:]) {
			if code, err := s.db.Get(account.CodeHash); err != nil || code == nil {
				res.task.codeTasks[common.BytesToHash(account.CodeHash)] = struct{}{}
				res.task.needCode[i] = true
				incomplete = true
			}
		}
		// Check if the account is a contract with an unknown storage trie
		if account.Root != emptyRoot {
			if node, err := s.db.Get(account.Root[:]); err != nil || node == nil {
				res.task.needState[i] = true
				incomplete = true
			}
		}
	}
	// If the account range contained no contracts, or all have been fully filled
	// beforehand, short circuit storage filling and forward to the next task
	if !incomplete {
		s.forwardAccountTask(res.task)
		return
	}
	// Some accounts are incomplete, leave as is for the storage and contract
	// task assigners to pick up and fill.
}

// processBytecodeResponse integrates an already validated bytecode response
// into the account tasks.
func (s *Syncer) processBytecodeResponse(res *bytecodeResponse) {
	batch := s.db.NewBatch()

	var (
		codes uint64
		bytes common.StorageSize
	)
	for i, hash := range res.hashes {
		code := res.codes[i]

		// If the bytecode was not delivered, reschedule it
		if code == nil {
			res.task.codeTasks[hash] = struct{}{}
			continue
		}
		// Code was delivered, mark it not needed any more
		for j, account := range res.task.res.accounts {
			if hash == common.BytesToHash(account.CodeHash) {
				res.task.needCode[j] = false
			}
		}
		// Push the bytecode into a database batch
		s.bytecodeSynced++
		s.bytecodeBytes += common.StorageSize(len(code))

		codes++
		bytes += common.StorageSize(len(code))

		batch.Put(hash[:], code)
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to persist bytecodes", "err", err)
	}
	log.Debug("Persisted set of bytecodes", "count", codes, "bytes", bytes)
}

// forwardAccountTask takes a filled account task and persists anything available
// into the database, after which it forwards the next account marker so that the
// task's next chunk may be filled.
func (s *Syncer) forwardAccountTask(task *accountTask) {
	// Iterate over all the accounts and gather all the incomplete trie nodes. A
	// node is incomplete if we haven't yet filled it (sync was interrupted), or
	// if we filled it in multiple chunks (storage trie), in which case the few
	// nodes on the chunk boundaries are missing.
	incompletes := light.NewNodeSet()
	for i := range task.res.accounts {
		// If the filling was interrupted, mark everything after as incomplete
		if task.needCode[i] || task.needState[i] {
			for j := i; j < len(task.res.accounts); j++ {
				if err := task.res.trie.Prove(task.res.hashes[j][:], 0, incompletes); err != nil {
					panic(err) // Account range was already proven, what happened
				}
			}
			break
		}
		// Filling not interrupted until this point, mark incomplete if needs healing
		if task.needHeal[i] {
			if err := task.res.trie.Prove(task.res.hashes[i][:], 0, incompletes); err != nil {
				panic(err) // Account range was already proven, what happened
			}
		}
	}
	// Persist every finalized trie node that's not on the boundary
	batch := s.db.NewBatch()

	it := task.res.nodes.NewIterator(nil, nil)
	for it.Next() {
		// Boundary nodes are not written, since they are incomplete
		if _, ok := task.res.bounds[common.BytesToHash(it.Key())]; ok {
			continue
		}
		// Accounts with split storage requests are incomplete
		if _, err := incompletes.Get(it.Key()); err == nil {
			continue
		}
		// Node is neither a boundary, not an incomplete account, persist to disk
		batch.Put(it.Key(), it.Value())
		s.bloom.Add(it.Key())

		s.accountBytes += common.StorageSize(common.HashLength + len(it.Value()))
	}
	it.Release()

	if err := batch.Write(); err != nil {
		log.Crit("Failed to persist accounts", "err", err)
	}
	// Task filling persisted, push it the chunk marker forward to the first
	// account still missing data.
	for i, hash := range task.res.hashes {
		if task.needCode[i] || task.needState[i] {
			break
		}
		task.Next = common.BigToHash(new(big.Int).Add(hash.Big(), big.NewInt(1)))
	}
}

/*
// syncAccounts starts (or resumes a previous) sync cycle to iterate over an state trie
// with the given root and reconstruct the nodes based on the snapshot leaves.
// Previously downloaded segments will not be redownloaded of fixed, rather any
// errors will be healed after the leaves are fully accumulated.
func (s *Syncer) syncAccounts(task *accountTask, cancel chan struct{}, failed chan struct{}) error {
	for {
		// Iterate over all the accounts and retrieve any missing storage tries.
		// Any storage tries that we can't sync fully in one go (proofs == missing
		// boundary nodes) will be marked incomplete to heal later.
		for i, blob := range res.accounts {
			// Retrieve any associated bytecode, if not yet downloaded
			if !bytes.Equal(acc.CodeHash, emptyCode[:]) {
				if code, err := s.db.Get(acc.CodeHash); err != nil || code == nil {


				}
			}
			// Retrieve any associated storage trie, if not yet downloaded
			if acc.Root != emptyRoot {
				if node, err := s.db.Get(acc.Root[:]); err != nil || node == nil {
					// Sync the contract's storage trie
					complete, err := s.syncStorage(root, res.hashes[i], acc.Root, peer, cancel)
					if err != nil {
						return err
					}
					// If the storage sync is incomplete (missing boundary nodes
					// across multiple requests), mark the account as incomplete
					// to force self healing at the end,
					if !complete {
						if err := res.trie.Prove(res.hashes[i][:], 0, incompletes); err != nil {
							panic(err) // Account range was already proven, what happened
						}
					}
				}
			}
			// If the snapshot moved during contract sync, nuke out all remaining accounts
			var interrupted bool
			select {
			case <-cancel:
				for j := i + 1; j < len(res.hashes); j++ {
					if err := res.trie.Prove(res.hashes[j][:], 0, incompletes); err != nil {
						panic(err) // Account range was already proven, what happened
					}
				}
				interrupted = true
			default:
			}
			if interrupted {
				// Sync was interrupted, restart next cycle at the current account,
				// but leave next slot at wherever we were.
				//
				// TODO(karalabe): Special case account deletion in the next cycle or proof-lessness, musn't write
				s.nextAcc = res.hashes[i]
				break
			}
			// Account processed fully (may still be incomplete, but that's for
			// trie node sync to complete), push the next account marker
			s.nextAcc = common.BigToHash(new(big.Int).Add(res.hashes[i].Big(), big.NewInt(1)))
			s.nextSlot = common.Hash{}
		}

		// Account range processed, step to the next chunk
		log.Debug("Persisted range of accounts", "next", s.nextAcc)
		s.report(false)
	}
	return nil
}

// syncStorage starts (or resumes a previous) sync cycle to iterate over a storage
// trie with the given root and reconstruct the nodes based on the snapshot leaves.
// Previously downloaded segments will not be redownloaded of fixed, rather any
// errors will be healed after the leaves are fully accumulated.
func (s *Syncer) syncStorage(root common.Hash, account common.Hash, stroot common.Hash, peer *Peer, cancel chan struct{}) (bool, error) {
	// For now simply iterate over the state iteratively
	batch := s.db.NewBatch()

	for {
		// If the sync was cancelled, abort
		select {
		case <-cancel:
			return false, nil
		default:
		}
		// Generate a random request ID (clash probability is insignificant)
		id := uint64(rand.Int63())

		// Track the request and send it to the peer
		s.lock.Lock()
		s.storageReqs[peer.id] = &storageRequest{
			id:     id,
			root:   stroot,
			origin: s.nextSlot,
			cancel: cancel,
		}
		s.lock.Unlock()

		if err := peer.RequestStorageRange(id, root, account, s.nextSlot, maxRequestSize); err != nil {
			return false, err
		}
		// Wait for the reply to arrive
		res := <-s.storageResps
		if res == nil {
			return false, errors.New("unfulfilled request")
		}
		s.storageSynced += uint64(len(res.slots))

		// Hack
		if res.nodes == nil {
			break
		}
		// Persist every finalized trie node that's not on the boundary
		it := res.nodes.NewIterator(nil, nil)
		for it.Next() {
			// Boundary nodes are not written, since they are incomplete
			if _, ok := res.bounds[common.BytesToHash(it.Key())]; ok {
				continue
			}
			// Node not a boundary, persist to disk
			batch.Put(it.Key(), it.Value())
			s.bloom.Add(it.Key())

			s.storageBytes += common.StorageSize(common.HashLength + len(it.Value()))
		}
		it.Release()

		if err := batch.Write(); err != nil {
			return false, err
		}
		batch.Reset()

		// Storage range processed, step to the next chunk
		log.Debug("Persisted range of storage slots", "account", account, "slot", res.last)
		s.report(false)

		// If the response contained all the data in one shot (no proofs), there
		// is no reason to continue the sync, report immediate success.
		if len(res.bounds) == 0 {
			return true, nil
		}
		s.nextSlot = common.BigToHash(new(big.Int).Add(res.last.Big(), big.NewInt(1)))
	}
	return false, nil
}*/

// OnAccounts is a callback method to invoke when a range of accounts are
// received from a remote peer.
func (s *Syncer) OnAccounts(peer *Peer, id uint64, hashes []common.Hash, accounts [][]byte, proof [][]byte) error {
	peer.Log().Trace("Delivering range of accounts", "hashes", len(hashes), "accounts", len(accounts), "proofs", len(proof))

	// Whether or not the response is valid, we can mark the peer as idle and
	// notify the scheduler to assign a new task. If the response is invalid,
	// we'll drop the peer in a bit.
	s.lock.Lock()
	if _, ok := s.peers[peer.id]; ok {
		s.accountIdlers[peer.id] = struct{}{}
	}
	select {
	case s.update <- struct{}{}:
	default:
	}
	// Ensure the response is for a valid request
	req, ok := s.accountReqs[id]
	if !ok {
		// Request stale, perhaps the peer timed out but came through in the end
		peer.Log().Warn("Unexpected account range packet")
		s.lock.Unlock()
		return nil
	}
	delete(s.accountReqs, id)

	// Clean up the request timeout timer, we'll see how to proceed further based
	// on the actual delivered content
	req.timeout.Stop()

	// Response is valid, but check if peer is signalling that it does not have
	// the requested data. For account range queries that means the state being
	// retrieved was either already pruned remotely, or the peer is not yet
	// synced to our head.
	if len(hashes) == 0 && len(accounts) == 0 && len(proof) == 0 {
		peer.Log().Debug("Peer rejected account range request", "root", s.root)
		s.statelessPeers[peer.id] = struct{}{}
		s.lock.Unlock()
		return nil
	}
	root := s.root
	s.lock.Unlock()

	// Reconstruct a partial trie from the response and verify it
	keys := make([][]byte, len(hashes))
	for i, key := range hashes {
		keys[i] = common.CopyBytes(key[:])
	}
	nodes := make(light.NodeList, len(proof))
	for i, node := range proof {
		nodes[i] = node
	}
	proofdb := nodes.NodeSet()

	db, tr, err := trie.VerifyRangeProof(root, req.task.Next[:], keys, accounts, proofdb, proofdb)
	if err != nil {
		return err
	}
	// Partial trie reconstructed, send it to the scheduler for storage filling
	bounds := make(map[common.Hash]struct{})
	hasher := sha3.NewLegacyKeccak256()
	for _, node := range proof {
		hasher.Reset()
		hasher.Write(node)
		bounds[common.BytesToHash(hasher.Sum(nil))] = struct{}{}
	}
	accs := make([]*state.Account, len(accounts))
	for i, account := range accounts {
		acc := new(state.Account)
		if err := rlp.DecodeBytes(account, acc); err != nil {
			panic(err) // We created these blobs, we must be able to decode them
		}
		accs[i] = acc
	}
	response := &accountResponse{
		task:     req.task,
		hashes:   hashes,
		accounts: accs,
		nodes:    db,
		trie:     tr,
		bounds:   bounds,
	}
	select {
	case <-req.cancel:
	case s.accountResps <- response:
	}
	return nil
}

// OnByteCodes is a callback method to invoke when a batch of contract
// bytes codes are received from a remote peer.
func (s *Syncer) OnByteCodes(peer *Peer, id uint64, bytecodes [][]byte) error {
	peer.Log().Trace("Delivering set of bytecodes", "bytecodes", len(bytecodes))

	// Whether or not the response is valid, we can mark the peer as idle and
	// notify the scheduler to assign a new task. If the response is invalid,
	// we'll drop the peer in a bit.
	s.lock.Lock()
	if _, ok := s.peers[peer.id]; ok {
		s.bytecodeIdlers[peer.id] = struct{}{}
	}
	select {
	case s.update <- struct{}{}:
	default:
	}
	// Ensure the response is for a valid request
	req, ok := s.bytecodeReqs[id]
	if !ok {
		// Request stale, perhaps the peer timed out but came through in the end
		peer.Log().Warn("Unexpected bytecode packet")
		s.lock.Unlock()
		return nil
	}
	delete(s.bytecodeReqs, id)

	// Clean up the request timeout timer, we'll see how to proceed further based
	// on the actual delivered content
	req.timeout.Stop()

	// Response is valid, but check if peer is signalling that it does not have
	// the requested data. For bytecode range queries that means the peer is not
	// yet synced.
	if len(bytecodes) == 0 {
		peer.Log().Debug("Peer rejected bytecode request")
		s.statelessPeers[peer.id] = struct{}{}
		s.lock.Unlock()
		return nil
	}
	s.lock.Unlock()

	// Cross reference the requested bytecodes with the response to find gaps
	// that the serving node is missing
	hasher := sha3.NewLegacyKeccak256()

	codes := make([][]byte, len(req.hashes))
	for i, j := 0, 0; i < len(bytecodes); i++ {
		// Find the next hash that we've been served, leaving misses with nils
		hasher.Reset()
		hasher.Write(bytecodes[i])
		hash := hasher.Sum(nil)

		for j < len(req.hashes) && !bytes.Equal(hash, req.hashes[j][:]) {
			j++
		}
		if j < len(req.hashes) {
			codes[j] = bytecodes[i]
			j++
			continue
		}
		// We've either ran out of hashes, or got unrequested data
		peer.Log().Warn("Unexpected bytecodes", "count", len(bytecodes)-i)
		return errors.New("unexpected bytecode")
	}
	// Response validated, send it to the scheduler for filling
	response := &bytecodeResponse{
		task:   req.task,
		hashes: req.hashes,
		codes:  codes,
	}
	select {
	case <-req.cancel:
	case s.bytecodeResps <- response:
	}
	return nil
}

// OnStorage is a callback method to invoke when a range of storage slots
// are received from a remote peer.
func (s *Syncer) OnStorage(peer *Peer, id uint64, hashes []common.Hash, slots [][]byte, proof [][]byte) error {
	peer.Log().Trace("Delivering range of storage slots", "hashes", len(hashes), "slots", len(slots), "proofs", len(proof))
	/*
		// If the request is stale, discard it
		s.lock.Lock()
		req, ok := s.storageReqs[peer.id]
		if !ok || req.id != id {
			peer.Log().Warn("Unexpected storage range packet")
			s.lock.Unlock()
			return nil
		}
		delete(s.storageReqs, peer.id)
		s.lock.Unlock()

		// If the response is unavailable snapshot, forward to the requester
		if len(hashes) == 0 && len(slots) == 0 && len(proof) == 0 {
			select {
			//case <-req.cancel:
			case s.storageResps <- nil:
			}
			return nil
		}
		// Reconstruct a partial trie from the response and verify it
		keys := make([][]byte, len(hashes))
		for i, key := range hashes {
			keys[i] = common.CopyBytes(key[:])
		}
		nodes := make(light.NodeList, len(proof))
		for i, node := range proof {
			nodes[i] = node
		}
		var (
			db  ethdb.KeyValueStore
			err error
		)
		if len(nodes) == 0 {
			// No proof has been attached, the response must cover the entire key
			// space and hash to the origin root.
			db, _, err = trie.VerifyRangeProof(req.root, req.origin[:], keys, slots, nil, nil)
			if err != nil {
				return err
			}
		} else {
			// A proof was attached, the response is only partial, check that the
			// returned data is indeed part of the storage trie
			proofdb := nodes.NodeSet()

			db, _, err = trie.VerifyRangeProof(req.root, req.origin[:], keys, slots, proofdb, proofdb)
			if err != nil {
				return err
			}
		}
		// Partial trie reconstructed, send it to the scheduler for storage filling
		bounds := make(map[common.Hash]struct{})

		hasher := sha3.NewLegacyKeccak256()
		for _, node := range proof {
			hasher.Reset()
			hasher.Write(node)
			bounds[common.BytesToHash(hasher.Sum(nil))] = struct{}{}
		}
		last := req.origin
		if len(hashes) > 0 {
			last = hashes[len(hashes)-1]
		}
		response := &storageResponse{
			hashes: hashes,
			slots:  slots,
			nodes:  db,
			bounds: bounds,
			last:   last,
		}
		select {
		//case <-req.cancel:
		case s.storageResps <- response:
		}*/
	return nil
}

// OnTrieNodes is a callback method to invoke when a batch of trie nodes
// are received from a remote peer.
func (s *Syncer) OnTrieNodes(peer *Peer, id uint64, nodes [][]byte) error {
	return errors.New("not implemented")
}

// report calculates various status reports and provides it to the user.
func (s *Syncer) report(force bool) {
	/*// Don't report all the events, just occasionally
	if !force && time.Since(s.logTime) < 3*time.Second {
		return
	}
	// Don't report anything until we have a meaningful progress
	synced := s.accountBytes + s.bytecodeBytes + s.storageBytes
	if synced == 0 || bytes.Compare(s.nextAcc[:], s.startAcc[:]) <= 0 {
		return
	}
	s.logTime = time.Now()

	estBytes := float64(new(big.Int).Div(
		new(big.Int).Exp(common.Big2, common.Big256, nil),
		new(big.Int).Div(
			new(big.Int).Sub(s.nextAcc.Big(), s.startAcc.Big()),
			new(big.Int).SetUint64(uint64(synced)),
		),
	).Uint64())

	elapsed := time.Since(s.startTime)
	estTime := elapsed / time.Duration(synced) * time.Duration(estBytes)

	// Create a mega progress report
	var (
		progress = fmt.Sprintf("%.2f%%", float64(synced)*100/estBytes)
		accounts = fmt.Sprintf("%d@%v", s.accountSynced, s.accountBytes.TerminalString())
		storage  = fmt.Sprintf("%d@%v", s.storageSynced, s.storageBytes.TerminalString())
		bytecode = fmt.Sprintf("%d@%v", s.bytecodeSynced, s.bytecodeBytes.TerminalString())
	)
	log.Info("State sync in progress", "synced", progress, "state", synced,
		"accounts", accounts, "slots", storage, "codes", bytecode, "eta", common.PrettyDuration(estTime-elapsed))*/
}
