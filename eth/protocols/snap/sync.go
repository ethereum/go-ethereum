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
	"encoding/json"
	"errors"
	"fmt"
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

	// maxStorageSetRequestCountis th maximum number of contracts to request the
	// storage of in a single query. If this number is too low, we're not filling
	// responses fully and waste round trip times. If it's too high, we're capping
	// responses and waste bandwidth.
	maxStorageSetRequestCount = maxRequestSize / 1024

	// maxCodeRequestCount is the maximum number of bytecode blobs to request in a
	// single query. If this number is too low, we're not filling responses fully
	// and waste round trip times. If it's too high, we're capping responses and
	// waste bandwidth.
	//
	// Depoyed bytecodes are currently capped at 24KB, so the minimum request
	// size should be maxRequestSize / 24K. Assuming that most contracts do not
	// come close to that, requesting 4x should be a good approximation.
	maxCodeRequestCount = maxRequestSize / (24 * 1024) * 4

	// maxTrieRequestCount is the maximum number of trie node blobs to request in
	// a single query. If this number is too low, we're not filling responses fully
	// and waste round trip times. If it's too high, we're capping responses and
	// waste bandwidth.
	maxTrieRequestCount = 512

	// requestTimeout is the maximum time a peer is allowed to spend on serving
	// a single network request.
	requestTimeout = 10 * time.Second // TODO(karalabe): Make it dynamic ala fast-sync?

	// accountConcurrency is the number of chunks to split the account trie into
	// to allow concurrent retrievals.
	accountConcurrency = 16

	// storageConcurrency is the number of chunks to split the a large contract
	// storage trie into to allow concurrent retrievals.
	storageConcurrency = 16
)

// accountRequest tracks a pending account range request to ensure responses are
// to actual requests and to validate any security constraints.
//
// Concurrency note: account requests and responses are handled concurrently from
// the main runloop to allow Merkle proof verifications on the peer's thread and
// to drop on invalid response. The request struct must contain all the data to
// construct the response without accessing runloop internals (i.e. task). That
// is only included to allow the runloop to match a response to the task being
// synced without having yet another set of maps.
type accountRequest struct {
	peer string // Peer to which this request is assigned
	id   uint64 // Request ID of this request

	cancel  chan struct{} // Channel to track sync cancellation
	timeout *time.Timer   // Timer to track delivery timeout
	stale   chan struct{} // Channel to signal the request was dropped

	origin common.Hash // First account requested to allow continuation checks
	limit  common.Hash // Last account requested to allow non-overlapping chunking

	task *accountTask // Task which this request is filling (only access fields through the runloop!!)
}

// accountResponse is an already Merkle-verified remote response to an account
// range request. It contains the subtrie for the requested account range and
// the database that's going to be filled with the internal nodes on commit.
type accountResponse struct {
	task *accountTask // Task which this request is filling

	hashes   []common.Hash    // Account hashes in the returned range
	accounts []*state.Account // Expanded accounts in the returned range

	nodes ethdb.KeyValueStore // Database containing the reconstructed trie nodes
	trie  *trie.Trie          // Reconstructed trie to reject incomplete account paths

	bounds   map[common.Hash]struct{} // Boundary nodes to avoid persisting incomplete accounts
	overflow *light.NodeSet           // Overflow nodes to avoid persisting across chunk boundaries

	cont bool // Whether the account range has a continuation
}

// bytecodeRequest tracks a pending bytecode request to ensure responses are to
// actual requests and to validate any security constraints.
//
// Concurrency note: bytecode requests and responses are handled concurrently from
// the main runloop to allow Keccak256 hash verifications on the peer's thread and
// to drop on invalid response. The request struct must contain all the data to
// construct the response without accessing runloop internals (i.e. task). That
// is only included to allow the runloop to match a response to the task being
// synced without having yet another set of maps.
type bytecodeRequest struct {
	peer string // Peer to which this request is assigned
	id   uint64 // Request ID of this request

	cancel  chan struct{} // Channel to track sync cancellation
	timeout *time.Timer   // Timer to track delivery timeout
	stale   chan struct{} // Channel to signal the request was dropped

	hashes []common.Hash // Bytecode hashes to validate responses
	task   *accountTask  // Task which this request is filling (only access fields through the runloop!!)
}

// bytecodeResponse is an already verified remote response to a bytecode request.
type bytecodeResponse struct {
	task *accountTask // Task which this request is filling

	hashes []common.Hash // Hashes of the bytecode to avoid double hashing
	codes  [][]byte      // Actual bytecodes to store into the database (nil = missing)
}

// storageRequest tracks a pending storage ranges request to ensure responses are
// to actual requests and to validate any security constraints.
//
// Concurrency note: storage requests and responses are handled concurrently from
// the main runloop to allow Merkel proof verifications on the peer's thread and
// to drop on invalid response. The request struct must contain all the data to
// construct the response without accessing runloop internals (i.e. tasks). That
// is only included to allow the runloop to match a response to the task being
// synced without having yet another set of maps.
type storageRequest struct {
	peer string // Peer to which this request is assigned
	id   uint64 // Request ID of this request

	cancel  chan struct{} // Channel to track sync cancellation
	timeout *time.Timer   // Timer to track delivery timeout
	stale   chan struct{} // Channel to signal the request was dropped

	accounts []common.Hash // Account hashes to validate responses
	roots    []common.Hash // Storage roots to validate responses

	origin common.Hash // First storage slot requested to allow continuation checks
	limit  common.Hash // Last storage slot requested to allow non-overlapping chunking

	mainTask *accountTask // Task which this response belongs to (only access fields through the runloop!!)
	subTask  *storageTask // Task which this response is filling (only access fields through the runloop!!)
}

// storageResponse is an already Merkle-verified remote response to a storage
// range request. It contains the subtries for the requested storage ranges and
// the databases that's going to be filled with the internal nodes on commit.
type storageResponse struct {
	mainTask *accountTask // Task which this response belongs to
	subTask  *storageTask // Task which this response is filling

	accounts []common.Hash // Account hashes requested, may be only partially filled
	roots    []common.Hash // Storage roots requested, may be only partially filled

	hashes [][]common.Hash       // Storage slot hashes in the returned range
	slots  [][][]byte            // Storage slot values in the returned range
	nodes  []ethdb.KeyValueStore // Database containing the reconstructed trie nodes
	tries  []*trie.Trie          // Reconstructed tries to reject overflown slots

	// Fields relevant for the last account only
	bounds   map[common.Hash]struct{} // Boundary nodes to avoid persisting (incomplete)
	overflow *light.NodeSet           // Overflow nodes to avoid persisting across chunk boundaries
	cont     bool                     // Whether the last storage range has a continuation
}

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
	peer string // Peer to which this request is assigned
	id   uint64 // Request ID of this request

	cancel  chan struct{} // Channel to track sync cancellation
	timeout *time.Timer   // Timer to track delivery timeout
	stale   chan struct{} // Channel to signal the request was dropped

	hashes []common.Hash   // Trie node hashes to validate responses
	paths  []trie.SyncPath // Trie node paths requested for rescheduling

	task *healTask // Task which this request is filling (only access fields through the runloop!!)
}

// trienodeHealResponse is an already verified remote response to a trie node request.
type trienodeHealResponse struct {
	task *healTask // Task which this request is filling

	hashes []common.Hash   // Hashes of the trie nodes to avoid double hashing
	paths  []trie.SyncPath // Trie node paths requested for rescheduling missing ones
	nodes  [][]byte        // Actual trie nodes to store into the database (nil = missing)
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
	peer string // Peer to which this request is assigned
	id   uint64 // Request ID of this request

	cancel  chan struct{} // Channel to track sync cancellation
	timeout *time.Timer   // Timer to track delivery timeout
	stale   chan struct{} // Channel to signal the request was dropped

	hashes []common.Hash // Bytecode hashes to validate responses
	task   *healTask     // Task which this request is filling (only access fields through the runloop!!)
}

// bytecodeHealResponse is an already verified remote response to a bytecode request.
type bytecodeHealResponse struct {
	task *healTask // Task which this request is filling

	hashes []common.Hash // Hashes of the bytecode to avoid double hashing
	codes  [][]byte      // Actual bytecodes to store into the database (nil = missing)
}

// accountTask represents the sync task for a chunk of the account snapshot.
type accountTask struct {
	// These fields get serialized to leveldb on shutdown
	Next     common.Hash                    // Next account to sync in this interval
	Last     common.Hash                    // Last account to sync in this interval
	SubTasks map[common.Hash][]*storageTask // Storage intervals needing fetching for large contracts

	// These fields are internals used during runtime
	req  *accountRequest  // Pending request to fill this task
	res  *accountResponse // Validate response filling this task
	pend int              // Number of pending subtasks for this round

	needCode  []bool // Flags whether the filling accounts need code retrieval
	needState []bool // Flags whether the filling accounts need storage retrieval
	needHeal  []bool // Flags whether the filling accounts's state was chunked and need healing

	codeTasks  map[common.Hash]struct{}    // Code hashes that need retrieval
	stateTasks map[common.Hash]common.Hash // Account hashes->roots that need full state retrieval

	done bool // Flag whether the task can be removed
}

// storageTask represents the sync task for a chunk of the storage snapshot.
type storageTask struct {
	Next common.Hash // Next account to sync in this interval
	Last common.Hash // Last account to sync in this interval

	// These fields are internals used during runtime
	root common.Hash     // Storage root hash for this instance
	req  *storageRequest // Pending request to fill this task
	done bool            // Flag whether the task can be removed
}

// healTask represents the sync task for healing the snap-synced chunk boundaries.
type healTask struct {
	scheduler *trie.Sync // State trie sync scheduler defining the tasks

	trieTasks map[common.Hash]trie.SyncPath // Set of trie node tasks currently queued for retrieval
	codeTasks map[common.Hash]struct{}      // Set of byte code tasks currently queued for retrieval
}

// syncProgress is a database entry to allow suspending and resuming a snapshot state
// sync. Opposed to full and fast sync, there is no way to restart a suspended
// snap sync without prior knowledge of the suspension point.
type syncProgress struct {
	Tasks []*accountTask // The suspended account tasks (contract tasks within)

	// Status report during syncing phase
	AccountSynced  uint64             // Number of accounts downloaded
	AccountBytes   common.StorageSize // Number of account trie bytes persisted to disk
	BytecodeSynced uint64             // Number of bytecodes downloaded
	BytecodeBytes  common.StorageSize // Number of bytecode bytes downloaded
	StorageSynced  uint64             // Number of storage slots downloaded
	StorageBytes   common.StorageSize // Number of storage trie bytes persisted to disk

	// Status report during healing phase
	TrienodeHealSynced uint64             // Number of state trie nodes downloaded
	TrienodeHealBytes  common.StorageSize // Number of state trie bytes persisted to disk
	TrienodeHealDups   uint64             // Number of state trie nodes already processed
	TrienodeHealNops   uint64             // Number of state trie nodes not requested
	BytecodeHealSynced uint64             // Number of bytecodes downloaded
	BytecodeHealBytes  common.StorageSize // Number of bytecodes persisted to disk
	BytecodeHealDups   uint64             // Number of bytecodes already processed
	BytecodeHealNops   uint64             // Number of bytecodes not requested
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

	root   common.Hash    // Current state trie root being synced
	tasks  []*accountTask // Current account task set being synced
	healer *healTask      // Current state healing task being executed
	update chan struct{}  // Notification channel for possible sync progression

	peers    map[string]*Peer // Currently active peers to download from
	peerJoin *event.Feed      // Event feed to react to peers joining
	peerDrop *event.Feed      // Event feed to react to peers dropping

	// Request tracking during syncing phase
	statelessPeers map[string]struct{} // Peers that failed to deliver state data
	accountIdlers  map[string]struct{} // Peers that aren't serving account requests
	bytecodeIdlers map[string]struct{} // Peers that aren't serving bytecode requests
	storageIdlers  map[string]struct{} // Peers that aren't serving storage requests

	accountReqs  map[uint64]*accountRequest  // Account requests currently running
	bytecodeReqs map[uint64]*bytecodeRequest // Bytecode requests currently running
	storageReqs  map[uint64]*storageRequest  // Storage requests currently running

	accountReqFails  chan *accountRequest  // Failed account range requests to revert
	bytecodeReqFails chan *bytecodeRequest // Failed bytecode requests to revert
	storageReqFails  chan *storageRequest  // Failed storage requests to revert

	accountResps  chan *accountResponse  // Account sub-tries to integrate into the database
	bytecodeResps chan *bytecodeResponse // Bytecodes to integrate into the database
	storageResps  chan *storageResponse  // Storage sub-tries to integrate into the database

	accountSynced  uint64             // Number of accounts downloaded
	accountBytes   common.StorageSize // Number of account trie bytes persisted to disk
	bytecodeSynced uint64             // Number of bytecodes downloaded
	bytecodeBytes  common.StorageSize // Number of bytecode bytes downloaded
	storageSynced  uint64             // Number of storage slots downloaded
	storageBytes   common.StorageSize // Number of storage trie bytes persisted to disk

	// Request tracking during healing phase
	trienodeHealIdlers map[string]struct{} // Peers that aren't serving trie node requests
	bytecodeHealIdlers map[string]struct{} // Peers that aren't serving bytecode requests

	trienodeHealReqs map[uint64]*trienodeHealRequest // Trie node requests currently running
	bytecodeHealReqs map[uint64]*bytecodeHealRequest // Bytecode requests currently running

	trienodeHealReqFails chan *trienodeHealRequest // Failed trienode requests to revert
	bytecodeHealReqFails chan *bytecodeHealRequest // Failed bytecode requests to revert

	trienodeHealResps chan *trienodeHealResponse // Trie nodes to integrate into the database
	bytecodeHealResps chan *bytecodeHealResponse // Bytecodes to integrate into the database

	trienodeHealSynced uint64             // Number of state trie nodes downloaded
	trienodeHealBytes  common.StorageSize // Number of state trie bytes persisted to disk
	trienodeHealDups   uint64             // Number of state trie nodes already processed
	trienodeHealNops   uint64             // Number of state trie nodes not requested
	bytecodeHealSynced uint64             // Number of bytecodes downloaded
	bytecodeHealBytes  common.StorageSize // Number of bytecodes persisted to disk
	bytecodeHealDups   uint64             // Number of bytecodes already processed
	bytecodeHealNops   uint64             // Number of bytecodes not requested

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

		accountReqs:      make(map[uint64]*accountRequest),
		storageReqs:      make(map[uint64]*storageRequest),
		bytecodeReqs:     make(map[uint64]*bytecodeRequest),
		accountReqFails:  make(chan *accountRequest),
		storageReqFails:  make(chan *storageRequest),
		bytecodeReqFails: make(chan *bytecodeRequest),
		accountResps:     make(chan *accountResponse),
		storageResps:     make(chan *storageResponse),
		bytecodeResps:    make(chan *bytecodeResponse),

		trienodeHealIdlers: make(map[string]struct{}),
		bytecodeHealIdlers: make(map[string]struct{}),

		trienodeHealReqs:     make(map[uint64]*trienodeHealRequest),
		bytecodeHealReqs:     make(map[uint64]*bytecodeHealRequest),
		trienodeHealReqFails: make(chan *trienodeHealRequest),
		bytecodeHealReqFails: make(chan *bytecodeHealRequest),
		trienodeHealResps:    make(chan *trienodeHealResponse),
		bytecodeHealResps:    make(chan *bytecodeHealResponse),
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
	s.trienodeHealIdlers[peer.id] = struct{}{}
	s.bytecodeHealIdlers[peer.id] = struct{}{}
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

		s.lock.Unlock()
		return errors.New("not registered")
	}
	delete(s.peers, id)

	// Remove status markers, even if no sync is running
	delete(s.statelessPeers, id)

	delete(s.accountIdlers, id)
	delete(s.storageIdlers, id)
	delete(s.bytecodeIdlers, id)
	delete(s.trienodeHealIdlers, id)
	delete(s.bytecodeHealIdlers, id)
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
	s.healer = &healTask{
		scheduler: state.NewStateSync(root, s.db, s.bloom),
		trieTasks: make(map[common.Hash]trie.SyncPath),
		codeTasks: make(map[common.Hash]struct{}),
	}
	s.statelessPeers = make(map[string]struct{})
	s.lock.Unlock()

	if s.startTime == (time.Time{}) {
		s.startTime = time.Now()
	}
	// Retrieve the previous sync status from LevelDB and abort if already synced
	s.loadSyncStatus()
	if len(s.tasks) == 0 && s.healer.scheduler.Pending() == 0 {
		log.Debug("Snapshot sync already completed")
		return nil
	}
	defer func() { // Persist any progress, independent of failure
		for _, task := range s.tasks {
			s.forwardAccountTask(task)
		}
		s.cleanAccountTasks()
		s.saveSyncStatus()
	}()

	log.Debug("Starting snapshot sync cycle", "root", root)
	defer s.report(true)

	// Whether sync completed or not, disregard any future packets
	defer func() {
		log.Debug("Terminating snapshot sync cycle", "root", root)
		s.lock.Lock()
		s.accountReqs = make(map[uint64]*accountRequest)
		s.storageReqs = make(map[uint64]*storageRequest)
		s.bytecodeReqs = make(map[uint64]*bytecodeRequest)
		s.trienodeHealReqs = make(map[uint64]*trienodeHealRequest)
		s.bytecodeHealReqs = make(map[uint64]*bytecodeHealRequest)
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
		// Remove all completed tasks and terminate sync if everything's done
		s.cleanStorageTasks()
		s.cleanAccountTasks()
		if len(s.tasks) == 0 && s.healer.scheduler.Pending() == 0 {
			return nil
		}
		// Assign all the data retrieval tasks to any free peers
		s.assignAccountTasks(cancel)
		s.assignBytecodeTasks(cancel)
		s.assignStorageTasks(cancel)
		if len(s.tasks) == 0 {
			// Sync phase done, run heal phase
			s.assignTrienodeHealTasks(cancel)
			s.assignBytecodeHealTasks(cancel)
		}
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
		case req := <-s.storageReqFails:
			s.revertStorageRequest(req)
		case req := <-s.trienodeHealReqFails:
			s.revertTrienodeHealRequest(req)
		case req := <-s.bytecodeHealReqFails:
			s.revertBytecodeHealRequest(req)

		case res := <-s.accountResps:
			s.processAccountResponse(res)
		case res := <-s.bytecodeResps:
			s.processBytecodeResponse(res)
		case res := <-s.storageResps:
			s.processStorageResponse(res)
		case res := <-s.trienodeHealResps:
			s.processTrienodeHealResponse(res)
		case res := <-s.bytecodeHealResps:
			s.processBytecodeHealResponse(res)
		}
		// Report stats if something meaningful happened
		s.report(false)
	}
}

// loadSyncStatus retrieves a previously aborted sync status from the database,
// or generates a fresh one if none is available.
func (s *Syncer) loadSyncStatus() {
	var progress syncProgress

	if status := rawdb.ReadSanpshotSyncStatus(s.db); status != nil {
		if err := json.Unmarshal(status, &progress); err != nil {
			log.Error("Failed to decode snap sync status", "err", err)
		} else {
			for _, task := range progress.Tasks {
				log.Debug("Scheduled account sync task", "from", task.Next, "last", task.Last)
			}
			s.tasks = progress.Tasks

			s.accountSynced = progress.AccountSynced
			s.accountBytes = progress.AccountBytes
			s.bytecodeSynced = progress.BytecodeSynced
			s.bytecodeBytes = progress.BytecodeBytes
			s.storageSynced = progress.StorageSynced
			s.storageBytes = progress.StorageBytes

			s.trienodeHealSynced = progress.TrienodeHealSynced
			s.trienodeHealBytes = progress.TrienodeHealBytes
			s.bytecodeHealSynced = progress.BytecodeHealSynced
			s.bytecodeHealBytes = progress.BytecodeHealBytes
			return
		}
	}
	// Either we've failed to decode the previus state, or there was none.
	// Start a fresh sync by chunking up the account range and scheduling
	// them for retrieval.
	s.tasks = nil
	s.accountSynced, s.accountBytes = 0, 0
	s.bytecodeSynced, s.bytecodeBytes = 0, 0
	s.storageSynced, s.storageBytes = 0, 0
	s.trienodeHealSynced, s.trienodeHealBytes = 0, 0
	s.bytecodeHealSynced, s.bytecodeHealBytes = 0, 0

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
		s.tasks = append(s.tasks, &accountTask{
			Next:     next,
			Last:     last,
			SubTasks: make(map[common.Hash][]*storageTask),
		})
		log.Debug("Created account sync task", "from", next, "last", last)
		next = common.BigToHash(new(big.Int).Add(last.Big(), common.Big1))
	}
}

// saveSyncStatus marshals the remaining sync tasks into leveldb.
func (s *Syncer) saveSyncStatus() {
	progress := &syncProgress{
		Tasks:              s.tasks,
		AccountSynced:      s.accountSynced,
		AccountBytes:       s.accountBytes,
		BytecodeSynced:     s.bytecodeSynced,
		BytecodeBytes:      s.bytecodeBytes,
		StorageSynced:      s.storageSynced,
		StorageBytes:       s.storageBytes,
		TrienodeHealSynced: s.trienodeHealSynced,
		TrienodeHealBytes:  s.trienodeHealBytes,
		BytecodeHealSynced: s.bytecodeHealSynced,
		BytecodeHealBytes:  s.bytecodeHealBytes,
	}
	status, err := json.Marshal(progress)
	if err != nil {
		panic(err) // This can only fail during implementation
	}
	rawdb.WriteSnapshotSyncStatus(s.db, status)
}

// cleanAccountTasks removes account range retrieval tasks that have already been
// completed.
func (s *Syncer) cleanAccountTasks() {
	for i := 0; i < len(s.tasks); i++ {
		if s.tasks[i].done {
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			i--
		}
	}
}

// cleanStorageTasks iterates over all the account tasks and storage sub-tasks
// within, cleaning any that have been completed.
func (s *Syncer) cleanStorageTasks() {
	for _, task := range s.tasks {
		for account, subtasks := range task.SubTasks {
			// Remove storage range retrieval tasks that completed
			for j := 0; j < len(subtasks); j++ {
				if subtasks[j].done {
					subtasks = append(subtasks[:j], subtasks[j+1:]...)
					j--
				}
			}
			if len(subtasks) > 0 {
				task.SubTasks[account] = subtasks
				continue
			}
			// If all storage chunks are done, mark the account as done too
			for j, hash := range task.res.hashes {
				if hash == account {
					task.needState[j] = false
				}
			}
			delete(task.SubTasks, account)
			task.pend--

			// If this was the last pending task, forward the account task
			if task.pend == 0 {
				s.forwardAccountTask(task)
			}
		}
	}
}

// assignAccountTasks attempts to match idle peers to pending account range
// retrievals.
func (s *Syncer) assignAccountTasks(cancel chan struct{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// If there are no idle peers, short circuit assignment
	if len(s.accountIdlers) == 0 {
		return
	}
	// Iterate over all the tasks and try to find a pending one
	for _, task := range s.tasks {
		// Skip any tasks already filling
		if task.req != nil || task.res != nil {
			continue
		}
		// Task pending retrieval, try to find an idle peer. If no such peer
		// exists, we probably assigned tasks for all (or they are stateless).
		// Abort the entire assignment mechanism.
		var idle string
		for id := range s.accountIdlers {
			// If the peer rejected a query in this sync cycle, don't bother asking
			// again for anything, it's either out of sync or already pruned
			if _, ok := s.statelessPeers[id]; ok {
				continue
			}
			idle = id
			break
		}
		if idle == "" {
			return
		}
		// Matched a pending task to an idle peer, allocate a unique request id
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
		req := &accountRequest{
			peer:   idle,
			id:     reqid,
			cancel: cancel,
			stale:  make(chan struct{}),
			origin: task.Next,
			limit:  task.Last,
			task:   task,
		}
		req.timeout = time.AfterFunc(requestTimeout, func() {
			log.Debug("Account range request timed out")
			select {
			case s.accountReqFails <- req:
			default:
			}
		})
		s.accountReqs[reqid] = req
		delete(s.accountIdlers, idle)

		s.pend.Add(1)
		go func(peer *Peer, root common.Hash) {
			defer s.pend.Done()

			// Attempt to send the remote request and revert if it fails
			if err := peer.RequestAccountRange(reqid, root, req.origin, req.limit, maxRequestSize); err != nil {
				peer.Log().Debug("Failed to request account range", "err", err)
				select {
				case s.accountReqFails <- req:
				default:
				}
			}
			// Request successfully sent, start a
		}(s.peers[idle], s.root) // We're in the lock, peers[id] surely exists

		// Inject the request into the task to block further assignments
		task.req = req
	}
}

// assignBytecodeTasks attempts to match idle peers to pending code retrievals.
func (s *Syncer) assignBytecodeTasks(cancel chan struct{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// If there are no idle peers, short circuit assignment
	if len(s.bytecodeIdlers) == 0 {
		return
	}
	// Iterate over all the tasks and try to find a pending one
	for _, task := range s.tasks {
		// Skip any tasks not in the bytecode retrieval phase
		if task.res == nil {
			continue
		}
		// Skip tasks that are already retrieving (or done with) all codes
		if len(task.codeTasks) == 0 {
			continue
		}
		// Task pending retrieval, try to find an idle peer. If no such peer
		// exists, we probably assigned tasks for all (or they are stateless).
		// Abort the entire assignment mechanism.
		var idle string
		for id := range s.bytecodeIdlers {
			// If the peer rejected a query in this sync cycle, don't bother asking
			// again for anything, it's either out of sync or already pruned
			if _, ok := s.statelessPeers[id]; ok {
				continue
			}
			idle = id
			break
		}
		if idle == "" {
			return
		}
		// Matched a pending task to an idle peer, allocate a unique request id
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
			peer:   idle,
			id:     reqid,
			cancel: cancel,
			stale:  make(chan struct{}),
			hashes: hashes,
			task:   task,
		}
		req.timeout = time.AfterFunc(requestTimeout, func() {
			log.Debug("Bytecode request timed out")
			select {
			case s.bytecodeReqFails <- req:
			default:
			}
		})
		s.bytecodeReqs[reqid] = req
		delete(s.bytecodeIdlers, idle)

		s.pend.Add(1)
		go func(peer *Peer) {
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
		}(s.peers[idle]) // We're in the lock, peers[id] surely exists
	}
}

// assignStorageTasks attempts to match idle peers to pending storage range
// retrievals.
func (s *Syncer) assignStorageTasks(cancel chan struct{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// If there are no idle peers, short circuit assignment
	if len(s.storageIdlers) == 0 {
		return
	}
	// Iterate over all the tasks and try to find a pending one
	for _, task := range s.tasks {
		// Skip any tasks not in the storage retrieval phase
		if task.res == nil {
			continue
		}
		// Skip tasks that are already retrieving (or done with) all small states
		if len(task.SubTasks) == 0 && len(task.stateTasks) == 0 {
			continue
		}
		// Task pending retrieval, try to find an idle peer. If no such peer
		// exists, we probably assigned tasks for all (or they are stateless).
		// Abort the entire assignment mechanism.
		var idle string
		for id := range s.storageIdlers {
			// If the peer rejected a query in this sync cycle, don't bother asking
			// again for anything, it's either out of sync or already pruned
			if _, ok := s.statelessPeers[id]; ok {
				continue
			}
			idle = id
			break
		}
		if idle == "" {
			return
		}
		// Matched a pending task to an idle peer, allocate a unique request id
		var reqid uint64
		for {
			reqid = uint64(rand.Int63())
			if reqid == 0 {
				continue
			}
			if _, ok := s.storageReqs[reqid]; ok {
				continue
			}
			break
		}
		// Generate the network query and send it to the peer. If there are
		// large contract tasks pending, complete those before diving into
		// even more new contracts.
		var (
			accounts = make([]common.Hash, 0, maxStorageSetRequestCount)
			roots    = make([]common.Hash, 0, maxStorageSetRequestCount)
			subtask  *storageTask
		)
		for account, subtasks := range task.SubTasks {
			for _, st := range subtasks {
				// Skip any subtasks already filling
				if st.req != nil {
					continue
				}
				// Found an incomplete storage chunk, schedule it
				accounts = append(accounts, account)
				roots = append(roots, st.root)

				subtask = st
				break // Large contract chunks are downloaded individually
			}
			if subtask != nil {
				break // Large contract chunks are downloaded individually
			}
		}
		if subtask == nil {
			// No large contract required retrieval, but small ones available
			for acccount, root := range task.stateTasks {
				delete(task.stateTasks, acccount)

				accounts = append(accounts, acccount)
				roots = append(roots, root)

				if len(accounts) >= maxStorageSetRequestCount {
					break
				}
			}
		}
		// If nothing was found, it means this task is actually already fully
		// retrieving, but large contracts are hard to detect. Skip to the next.
		if len(accounts) == 0 {
			continue
		}
		req := &storageRequest{
			peer:     idle,
			id:       reqid,
			cancel:   cancel,
			stale:    make(chan struct{}),
			accounts: accounts,
			roots:    roots,
			mainTask: task,
			subTask:  subtask,
		}
		if subtask != nil {
			req.origin = subtask.Next
			req.limit = subtask.Last
		}
		req.timeout = time.AfterFunc(requestTimeout, func() {
			log.Debug("Storage request timed out")
			select {
			case s.storageReqFails <- req:
			default:
			}
		})
		s.storageReqs[reqid] = req
		delete(s.storageIdlers, idle)

		s.pend.Add(1)
		go func(peer *Peer, root common.Hash) {
			defer s.pend.Done()

			// Attempt to send the remote request and revert if it fails
			var origin, limit []byte
			if subtask != nil {
				origin, limit = req.origin[:], req.limit[:]
			}
			if err := peer.RequestStorageRanges(reqid, root, accounts, origin, limit, maxRequestSize); err != nil {
				log.Debug("Failed to request storage", "err", err)
				select {
				case s.storageReqFails <- req:
				default:
				}
			}
			// Request successfully sent, start a
		}(s.peers[idle], s.root) // We're in the lock, peers[id] surely exists

		// Inject the request into the subtask to block further assignments
		if subtask != nil {
			subtask.req = req
		}
	}
}

// assignTrienodeHealTasks attempts to match idle peers to trie node requests to
// heal any trie errors caused by the snap sync's chunked retrieval model.
func (s *Syncer) assignTrienodeHealTasks(cancel chan struct{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// If there are no idle peers, short circuit assignment
	if len(s.trienodeHealIdlers) == 0 {
		return
	}
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
			nodes, paths, codes := s.healer.scheduler.Missing(want - have)
			for i, hash := range nodes {
				s.healer.trieTasks[hash] = paths[i]
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
		var idle string
		for id := range s.trienodeHealIdlers {
			// If the peer rejected a query in this sync cycle, don't bother asking
			// again for anything, it's either out of sync or already pruned
			if _, ok := s.statelessPeers[id]; ok {
				continue
			}
			idle = id
			break
		}
		if idle == "" {
			return
		}
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
		var (
			hashes   = make([]common.Hash, 0, maxTrieRequestCount)
			paths    = make([]trie.SyncPath, 0, maxTrieRequestCount)
			pathsets = make([]TrieNodePathSet, 0, maxTrieRequestCount)
		)
		for hash, pathset := range s.healer.trieTasks {
			delete(s.healer.trieTasks, hash)

			hashes = append(hashes, hash)
			paths = append(paths, pathset)
			pathsets = append(pathsets, [][]byte(pathset)) // TODO(karalabe): group requests by account hash

			if len(hashes) >= maxTrieRequestCount {
				break
			}
		}
		req := &trienodeHealRequest{
			peer:   idle,
			id:     reqid,
			cancel: cancel,
			stale:  make(chan struct{}),
			hashes: hashes,
			paths:  paths,
			task:   s.healer,
		}
		req.timeout = time.AfterFunc(requestTimeout, func() {
			log.Debug("Trienode heal request timed out")
			select {
			case s.trienodeHealReqFails <- req:
			default:
			}
		})
		s.trienodeHealReqs[reqid] = req
		delete(s.trienodeHealIdlers, idle)

		s.pend.Add(1)
		go func(peer *Peer, root common.Hash) {
			defer s.pend.Done()

			// Attempt to send the remote request and revert if it fails
			if err := peer.RequestTrieNodes(reqid, root, pathsets, maxRequestSize); err != nil {
				log.Debug("Failed to request trienode healers", "err", err)
				select {
				case s.trienodeHealReqFails <- req:
				default:
				}
			}
			// Request successfully sent, start a
		}(s.peers[idle], s.root) // We're in the lock, peers[id] surely exists
	}
}

// assignBytecodeHealTasks attempts to match idle peers to bytecode requests to
// heal any trie errors caused by the snap sync's chunked retrieval model.
func (s *Syncer) assignBytecodeHealTasks(cancel chan struct{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// If there are no idle peers, short circuit assignment
	if len(s.bytecodeHealIdlers) == 0 {
		return
	}
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
			nodes, paths, codes := s.healer.scheduler.Missing(want - have)
			for i, hash := range nodes {
				s.healer.trieTasks[hash] = paths[i]
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
		var idle string
		for id := range s.bytecodeHealIdlers {
			// If the peer rejected a query in this sync cycle, don't bother asking
			// again for anything, it's either out of sync or already pruned
			if _, ok := s.statelessPeers[id]; ok {
				continue
			}
			idle = id
			break
		}
		if idle == "" {
			return
		}
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
		hashes := make([]common.Hash, 0, maxCodeRequestCount)
		for hash := range s.healer.codeTasks {
			delete(s.healer.codeTasks, hash)

			hashes = append(hashes, hash)
			if len(hashes) >= maxCodeRequestCount {
				break
			}
		}
		req := &bytecodeHealRequest{
			peer:   idle,
			id:     reqid,
			cancel: cancel,
			stale:  make(chan struct{}),
			hashes: hashes,
			task:   s.healer,
		}
		req.timeout = time.AfterFunc(requestTimeout, func() {
			log.Debug("Bytecode heal request timed out")
			select {
			case s.bytecodeHealReqFails <- req:
			default:
			}
		})
		s.bytecodeHealReqs[reqid] = req
		delete(s.bytecodeHealIdlers, idle)

		s.pend.Add(1)
		go func(peer *Peer) {
			defer s.pend.Done()

			// Attempt to send the remote request and revert if it fails
			if err := peer.RequestByteCodes(reqid, hashes, maxRequestSize); err != nil {
				log.Debug("Failed to request bytecode healers", "err", err)
				select {
				case s.bytecodeHealReqFails <- req:
				default:
				}
			}
			// Request successfully sent, start a
		}(s.peers[idle]) // We're in the lock, peers[id] surely exists
	}
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
	var trienodeHealReqs []*trienodeHealRequest
	for _, req := range s.trienodeHealReqs {
		if req.peer == peer {
			trienodeHealReqs = append(trienodeHealReqs, req)
		}
	}
	var bytecodeHealReqs []*bytecodeHealRequest
	for _, req := range s.bytecodeHealReqs {
		if req.peer == peer {
			bytecodeHealReqs = append(bytecodeHealReqs, req)
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
	for _, req := range trienodeHealReqs {
		s.revertTrienodeHealRequest(req)
	}
	for _, req := range bytecodeHealReqs {
		s.revertBytecodeHealRequest(req)
	}
}

// revertAccountRequest cleans up an account range request and returns all failed
// retrieval tasks to the scheduler for reassignment.
func (s *Syncer) revertAccountRequest(req *accountRequest) {
	log.Trace("Reverting account request", "peer", req.peer, "reqid", req.id)
	select {
	case <-req.stale:
		log.Trace("Account request already reverted", "peer", req.peer, "reqid", req.id)
		return
	default:
	}
	close(req.stale)

	// Remove the request from the tracked set
	s.lock.Lock()
	delete(s.accountReqs, req.id)
	s.lock.Unlock()

	// If there's a timeout timer still running, abort it and mark the account
	// task as not-pending, ready for resheduling
	req.timeout.Stop()
	if req.task.req == req {
		req.task.req = nil
	}
}

// revertBytecodeRequest cleans up an bytecode request and returns all failed
// retrieval tasks to the scheduler for reassignment.
func (s *Syncer) revertBytecodeRequest(req *bytecodeRequest) {
	log.Trace("Reverting bytecode request", "peer", req.peer)
	select {
	case <-req.stale:
		log.Trace("Bytecode request already reverted", "peer", req.peer, "reqid", req.id)
		return
	default:
	}
	close(req.stale)

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
	log.Trace("Reverting storage request", "peer", req.peer)
	select {
	case <-req.stale:
		log.Trace("Storage request already reverted", "peer", req.peer, "reqid", req.id)
		return
	default:
	}
	close(req.stale)

	// Remove the request from the tracked set
	s.lock.Lock()
	delete(s.storageReqs, req.id)
	s.lock.Unlock()

	// If there's a timeout timer still running, abort it and mark the storage
	// task as not-pending, ready for resheduling
	req.timeout.Stop()
	if req.subTask != nil {
		req.subTask.req = nil
	} else {
		for i, account := range req.accounts {
			req.mainTask.stateTasks[account] = req.roots[i]
		}
	}
}

// revertTrienodeHealRequest cleans up an trienode heal request and returns all
// failed retrieval tasks to the scheduler for reassignment.
func (s *Syncer) revertTrienodeHealRequest(req *trienodeHealRequest) {
	log.Trace("Reverting trienode heal request", "peer", req.peer)
	select {
	case <-req.stale:
		log.Trace("Trienode heal request already reverted", "peer", req.peer, "reqid", req.id)
		return
	default:
	}
	close(req.stale)

	// Remove the request from the tracked set
	s.lock.Lock()
	delete(s.trienodeHealReqs, req.id)
	s.lock.Unlock()

	// If there's a timeout timer still running, abort it and mark the trie node
	// retrievals as not-pending, ready for resheduling
	req.timeout.Stop()
	for i, hash := range req.hashes {
		req.task.trieTasks[hash] = [][]byte(req.paths[i])
	}
}

// revertBytecodeHealRequest cleans up an bytecode request and returns all failed
// retrieval tasks to the scheduler for reassignment.
func (s *Syncer) revertBytecodeHealRequest(req *bytecodeHealRequest) {
	log.Trace("Reverting bytecode heal request", "peer", req.peer)
	select {
	case <-req.stale:
		log.Trace("Bytecode heal request already reverted", "peer", req.peer, "reqid", req.id)
		return
	default:
	}
	close(req.stale)

	// Remove the request from the tracked set
	s.lock.Lock()
	delete(s.bytecodeHealReqs, req.id)
	s.lock.Unlock()

	// If there's a timeout timer still running, abort it and mark the code
	// retrievals as not-pending, ready for resheduling
	req.timeout.Stop()
	for _, hash := range req.hashes {
		req.task.codeTasks[hash] = struct{}{}
	}
}

// processAccountResponse integrates an already validated account range response
// into the account tasks.
func (s *Syncer) processAccountResponse(res *accountResponse) {
	// Switch the task from pending to filling
	res.task.req = nil
	res.task.res = res

	// Ensure that the response doesn't overflow into the subsequent task
	last := res.task.Last.Big()
	for i, hash := range res.hashes {
		if hash.Big().Cmp(last) > 0 {
			// Chunk overflown, cut off excess, but also update the boundary nodes
			for j := i; j < len(res.hashes); j++ {
				if err := res.trie.Prove(res.hashes[j][:], 0, res.overflow); err != nil {
					panic(err) // Account range was already proven, what happened
				}
			}
			res.hashes = res.hashes[:i]
			res.accounts = res.accounts[:i]
			res.cont = false // Mark range completed
			break
		}
	}
	// Itereate over all the accounts and assemble which ones need further sub-
	// filling before the entire account range can be persisted.
	res.task.needCode = make([]bool, len(res.accounts))
	res.task.needState = make([]bool, len(res.accounts))
	res.task.needHeal = make([]bool, len(res.accounts))

	res.task.codeTasks = make(map[common.Hash]struct{})
	res.task.stateTasks = make(map[common.Hash]common.Hash)

	resumed := make(map[common.Hash]struct{})

	res.task.pend = 0
	for i, account := range res.accounts {
		// Check if the account is a contract with an unknown code
		if !bytes.Equal(account.CodeHash, emptyCode[:]) {
			if code := rawdb.ReadCodeWithPrefix(s.db, common.BytesToHash(account.CodeHash)); code == nil {
				res.task.codeTasks[common.BytesToHash(account.CodeHash)] = struct{}{}
				res.task.needCode[i] = true
				res.task.pend++
			}
		}
		// Check if the account is a contract with an unknown storage trie
		if account.Root != emptyRoot {
			if node, err := s.db.Get(account.Root[:]); err != nil || node == nil {
				// If there was a previous large state retrieval in progress,
				// don't restart it from scratch. This happens if a sync cycle
				// is interrupted and resumed later. However, *do* update the
				// previous root hash.
				if subtasks, ok := res.task.SubTasks[res.hashes[i]]; ok {
					log.Error("Resuming large storage retrieval", "account", res.hashes[i], "root", account.Root)
					for _, subtask := range subtasks {
						subtask.root = account.Root
					}
					res.task.needHeal[i] = true
					resumed[res.hashes[i]] = struct{}{}
				} else {
					res.task.stateTasks[res.hashes[i]] = account.Root
				}
				res.task.needState[i] = true
				res.task.pend++
			}
		}
	}
	// Delete any subtasks that have been aborted but not resumed. This may undo
	// some progress if a newpeer gives us less accounts than an old one, but for
	// now we have to live with that.
	for hash := range res.task.SubTasks {
		if _, ok := resumed[hash]; !ok {
			log.Error("Aborting suspended storage retrieval", "account", hash)
			delete(res.task.SubTasks, hash)
		}
	}
	// If the account range contained no contracts, or all have been fully filled
	// beforehand, short circuit storage filling and forward to the next task
	if res.task.pend == 0 {
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
			if res.task.needCode[j] && hash == common.BytesToHash(account.CodeHash) {
				res.task.needCode[j] = false
				res.task.pend--
			}
		}
		// Push the bytecode into a database batch
		s.bytecodeSynced++
		s.bytecodeBytes += common.StorageSize(len(code))

		codes++
		bytes += common.StorageSize(len(code))

		rawdb.WriteCode(batch, hash, code)
		s.bloom.Add(hash[:])
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to persist bytecodes", "err", err)
	}
	log.Debug("Persisted set of bytecodes", "count", codes, "bytes", bytes)

	// If this delivery completed the last pending task, forward the account task
	// to the next chunk
	if res.task.pend == 0 {
		s.forwardAccountTask(res.task)
		return
	}
	// Some accounts are still incomplete, leave as is for the storage and contract
	// task assigners to pick up and fill.
}

// processStorageResponse integrates an already validated storage response
// into the account tasks.
func (s *Syncer) processStorageResponse(res *storageResponse) {
	// Switch the suntask from pending to idle
	if res.subTask != nil {
		res.subTask.req = nil
	}
	batch := s.db.NewBatch()

	var (
		slots   int
		nodes   int
		skipped int
		bytes   common.StorageSize
	)
	// Iterate over all the accounts and reconstruct their storage tries from the
	// delivered slots
	delivered := make(map[common.Hash]bool)
	for i := 0; i < len(res.hashes); i++ {
		delivered[res.roots[i]] = true
	}
	for i, account := range res.accounts {
		// If the account was not delivered, reschedule it
		if i >= len(res.hashes) {
			if !delivered[res.roots[i]] {
				res.mainTask.stateTasks[account] = res.roots[i]
			}
			continue
		}
		// State was delivered, if complete mark as not needed any more, otherwise
		// mark the account as needing healing
		for j, acc := range res.mainTask.res.accounts {
			if res.roots[i] == acc.Root {
				// If the packet contains multiple contract storage slots, all
				// but the last are surely complete. The last contract may be
				// chunked, so check it's continuation flag.
				if res.subTask == nil && res.mainTask.needState[j] && (i < len(res.hashes)-1 || !res.cont) {
					res.mainTask.needState[j] = false
					res.mainTask.pend--
				}
				// If the last contract was chunked, mark it as needing healing
				// to avoid writing it out to disk prematurely.
				if res.subTask == nil && !res.mainTask.needHeal[j] && i == len(res.hashes)-1 && res.cont {
					res.mainTask.needHeal[j] = true
				}
				// If the last contract was chunked, we need to switch to large
				// contract handling mode
				if res.subTask == nil && i == len(res.hashes)-1 && res.cont {
					// If we haven't yet started a large-contract retrieval, create
					// the subtasks for it within the main account task
					if tasks, ok := res.mainTask.SubTasks[account]; !ok {
						var (
							next common.Hash
						)
						step := new(big.Int).Sub(
							new(big.Int).Div(
								new(big.Int).Exp(common.Big2, common.Big256, nil),
								big.NewInt(storageConcurrency),
							), common.Big1,
						)
						for k := 0; k < storageConcurrency; k++ {
							last := common.BigToHash(new(big.Int).Add(next.Big(), step))
							if k == storageConcurrency-1 {
								// Make sure we don't overflow if the step is not a proper divisor
								last = common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
							}
							tasks = append(tasks, &storageTask{
								Next: next,
								Last: last,
								root: acc.Root,
							})
							log.Debug("Created storage sync task", "account", account, "root", acc.Root, "from", next, "last", last)
							next = common.BigToHash(new(big.Int).Add(last.Big(), common.Big1))
						}
						res.mainTask.SubTasks[account] = tasks

						// Since we've just created the sub-tasks, this response
						// is surely for the first one (zero origin)
						res.subTask = tasks[0]
					}
				}
				// If we're in large contract delivery mode, forward the subtask
				if res.subTask != nil {
					// Ensure the response doesn't overflow into the subsequent task
					last := res.subTask.Last.Big()
					for k, hash := range res.hashes[i] {
						if hash.Big().Cmp(last) > 0 {
							// Chunk overflown, cut off excess, but also update the boundary
							for l := k; l < len(res.hashes[i]); l++ {
								if err := res.tries[i].Prove(res.hashes[i][l][:], 0, res.overflow); err != nil {
									panic(err) // Account range was already proven, what happened
								}
							}
							res.hashes[i] = res.hashes[i][:k]
							res.slots[i] = res.slots[i][:k]
							res.cont = false // Mark range completed
							break
						}
					}
					// Forward the relevant storage chunk (even if created just now)
					if res.cont {
						res.subTask.Next = common.BigToHash(new(big.Int).Add(res.hashes[i][len(res.hashes[i])-1].Big(), big.NewInt(1)))
					} else {
						res.subTask.done = true
					}
				}
			}
		}
		// Iterate over all the reconstructed trie nodes and push them to disk
		slots += len(res.hashes[i])

		it := res.nodes[i].NewIterator(nil, nil)
		for it.Next() {
			// Boundary nodes are not written for the last result, since they are incomplete
			if i == len(res.hashes)-1 {
				if _, ok := res.bounds[common.BytesToHash(it.Key())]; ok {
					skipped++
					continue
				}
			}
			// Node is not a boundary, persist to disk
			batch.Put(it.Key(), it.Value())
			s.bloom.Add(it.Key())

			bytes += common.StorageSize(common.HashLength + len(it.Value()))
			nodes++
		}
		it.Release()
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to persist storage slots", "err", err)
	}
	s.storageSynced += uint64(slots)
	s.storageBytes += bytes

	log.Debug("Persisted set of storage slots", "accounts", len(res.hashes), "slots", slots, "nodes", nodes, "skipped", skipped, "bytes", bytes)

	// If this delivery completed the last pending task, forward the account task
	// to the next chunk
	if res.mainTask.pend == 0 {
		s.forwardAccountTask(res.mainTask)
		return
	}
	// Some accounts are still incomplete, leave as is for the storage and contract
	// task assigners to pick up and fill.
}

// processTrienodeHealResponse integrates an already validated trienode response
// into the healer tasks.
func (s *Syncer) processTrienodeHealResponse(res *trienodeHealResponse) {
	for i, hash := range res.hashes {
		node := res.nodes[i]

		// If the trie node was not delivered, reschedule it
		if node == nil {
			res.task.trieTasks[hash] = res.paths[i]
			continue
		}
		// Push the trie node into the state syncer
		s.trienodeHealSynced++
		s.trienodeHealBytes += common.StorageSize(len(node))

		err := s.healer.scheduler.Process(trie.SyncResult{Hash: hash, Data: node})
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
	batch := s.db.NewBatch()
	if err := s.healer.scheduler.Commit(batch); err != nil {
		log.Error("Failed to commit healing data", "err", err)
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to persist healing data", "err", err)
	}
	log.Debug("Persisted set of healing data", "bytes", common.StorageSize(batch.ValueSize()))
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

		err := s.healer.scheduler.Process(trie.SyncResult{Hash: hash, Data: node})
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
	batch := s.db.NewBatch()
	if err := s.healer.scheduler.Commit(batch); err != nil {
		log.Error("Failed to commit healing data", "err", err)
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to persist healing data", "err", err)
	}
	log.Debug("Persisted set of healing data", "bytes", common.StorageSize(batch.ValueSize()))
}

// forwardAccountTask takes a filled account task and persists anything available
// into the database, after which it forwards the next account marker so that the
// task's next chunk may be filled.
func (s *Syncer) forwardAccountTask(task *accountTask) {
	// Remove any pending delivery
	res := task.res
	if res == nil {
		return // nothing to forward
	}
	task.res = nil

	// Iterate over all the accounts and gather all the incomplete trie nodes. A
	// node is incomplete if we haven't yet filled it (sync was interrupted), or
	// if we filled it in multiple chunks (storage trie), in which case the few
	// nodes on the chunk boundaries are missing.
	incompletes := light.NewNodeSet()
	for i := range res.accounts {
		// If the filling was interrupted, mark everything after as incomplete
		if task.needCode[i] || task.needState[i] {
			for j := i; j < len(res.accounts); j++ {
				if err := res.trie.Prove(res.hashes[j][:], 0, incompletes); err != nil {
					panic(err) // Account range was already proven, what happened
				}
			}
			break
		}
		// Filling not interrupted until this point, mark incomplete if needs healing
		if task.needHeal[i] {
			if err := res.trie.Prove(res.hashes[i][:], 0, incompletes); err != nil {
				panic(err) // Account range was already proven, what happened
			}
		}
	}
	// Persist every finalized trie node that's not on the boundary
	batch := s.db.NewBatch()

	var (
		nodes   int
		skipped int
		bytes   common.StorageSize
	)
	it := res.nodes.NewIterator(nil, nil)
	for it.Next() {
		// Boundary nodes are not written, since they are incomplete
		if _, ok := res.bounds[common.BytesToHash(it.Key())]; ok {
			skipped++
			continue
		}
		// Overflow nodes are not written, since they mess with another task
		if _, err := res.overflow.Get(it.Key()); err == nil {
			skipped++
			continue
		}
		// Accounts with split storage requests are incomplete
		if _, err := incompletes.Get(it.Key()); err == nil {
			skipped++
			continue
		}
		// Node is neither a boundary, not an incomplete account, persist to disk
		batch.Put(it.Key(), it.Value())
		s.bloom.Add(it.Key())

		bytes += common.StorageSize(common.HashLength + len(it.Value()))
		nodes++
	}
	it.Release()

	if err := batch.Write(); err != nil {
		log.Crit("Failed to persist accounts", "err", err)
	}
	s.accountBytes += bytes
	s.accountSynced += uint64(len(res.accounts))

	log.Debug("Persisted range of accounts", "accounts", len(res.accounts), "nodes", nodes, "skipped", skipped, "bytes", bytes)

	// Task filling persisted, push it the chunk marker forward to the first
	// account still missing data.
	for i, hash := range res.hashes {
		if task.needCode[i] || task.needState[i] {
			return
		}
		task.Next = common.BigToHash(new(big.Int).Add(hash.Big(), big.NewInt(1)))
	}
	// All accounts marked as complete, track if the entire task is done
	task.done = !res.cont
}

// OnAccounts is a callback method to invoke when a range of accounts are
// received from a remote peer.
func (s *Syncer) OnAccounts(peer *Peer, id uint64, hashes []common.Hash, accounts [][]byte, proof [][]byte) error {
	size := common.StorageSize(len(hashes) * common.HashLength)
	for _, account := range accounts {
		size += common.StorageSize(len(account))
	}
	for _, node := range proof {
		size += common.StorageSize(len(node))
	}
	logger := peer.logger.New("reqid", id)
	logger.Trace("Delivering range of accounts", "hashes", len(hashes), "accounts", len(accounts), "proofs", len(proof), "bytes", size)

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
		logger.Warn("Unexpected account range packet")
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
		logger.Debug("Peer rejected account range request", "root", s.root)
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

	var end []byte
	if len(keys) > 0 {
		end = keys[len(keys)-1]
	}
	db, tr, notary, cont, err := trie.VerifyRangeProof(root, req.origin[:], end, keys, accounts, proofdb)
	if err != nil {
		logger.Warn("Account range failed proof", "err", err)
		return err
	}
	// Partial trie reconstructed, send it to the scheduler for storage filling
	bounds := make(map[common.Hash]struct{})

	it := notary.Accessed().NewIterator(nil, nil)
	for it.Next() {
		bounds[common.BytesToHash(it.Key())] = struct{}{}
	}
	it.Release()

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
		overflow: light.NewNodeSet(),
		cont:     cont,
	}
	select {
	case s.accountResps <- response:
	case <-req.cancel:
	case <-req.stale:
	}
	return nil
}

// OnByteCodes is a callback method to invoke when a batch of contract
// bytes codes are received from a remote peer.
func (s *Syncer) OnByteCodes(peer *Peer, id uint64, bytecodes [][]byte) error {
	s.lock.RLock()
	syncing := len(s.tasks) > 0
	s.lock.RUnlock()

	if syncing {
		return s.onByteCodes(peer, id, bytecodes)
	}
	return s.onHealByteCodes(peer, id, bytecodes)
}

// onByteCodes is a callback method to invoke when a batch of contract
// bytes codes are received from a remote peer in the syncing phase.
func (s *Syncer) onByteCodes(peer *Peer, id uint64, bytecodes [][]byte) error {
	var size common.StorageSize
	for _, code := range bytecodes {
		size += common.StorageSize(len(code))
	}
	logger := peer.logger.New("reqid", id)
	logger.Trace("Delivering set of bytecodes", "bytecodes", len(bytecodes), "bytes", size)

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
		logger.Warn("Unexpected bytecode packet")
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
		logger.Debug("Peer rejected bytecode request")
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
		logger.Warn("Unexpected bytecodes", "count", len(bytecodes)-i)
		return errors.New("unexpected bytecode")
	}
	// Response validated, send it to the scheduler for filling
	response := &bytecodeResponse{
		task:   req.task,
		hashes: req.hashes,
		codes:  codes,
	}
	select {
	case s.bytecodeResps <- response:
	case <-req.cancel:
	case <-req.stale:
	}
	return nil
}

// OnStorage is a callback method to invoke when ranges of storage slots
// are received from a remote peer.
func (s *Syncer) OnStorage(peer *Peer, id uint64, hashes [][]common.Hash, slots [][][]byte, proof [][]byte) error {
	// Gather some trace stats to aid in debugging issues
	var (
		hashCount int
		slotCount int
		size      common.StorageSize
	)
	for _, hashset := range hashes {
		size += common.StorageSize(common.HashLength * len(hashset))
		hashCount += len(hashset)
	}
	for _, slotset := range slots {
		for _, slot := range slotset {
			size += common.StorageSize(len(slot))
		}
		slotCount += len(slotset)
	}
	for _, node := range proof {
		size += common.StorageSize(len(node))
	}
	logger := peer.logger.New("reqid", id)
	logger.Trace("Delivering ranges of storage slots", "accounts", len(hashes), "hashes", hashCount, "slots", slotCount, "proofs", len(proof), "size", size)

	// Whether or not the response is valid, we can mark the peer as idle and
	// notify the scheduler to assign a new task. If the response is invalid,
	// we'll drop the peer in a bit.
	s.lock.Lock()
	if _, ok := s.peers[peer.id]; ok {
		s.storageIdlers[peer.id] = struct{}{}
	}
	select {
	case s.update <- struct{}{}:
	default:
	}
	// Ensure the response is for a valid request
	req, ok := s.storageReqs[id]
	if !ok {
		// Request stale, perhaps the peer timed out but came through in the end
		logger.Warn("Unexpected storage ranges packet")
		s.lock.Unlock()
		return nil
	}
	delete(s.storageReqs, id)

	// Clean up the request timeout timer, we'll see how to proceed further based
	// on the actual delivered content
	req.timeout.Stop()

	// Reject the response if the hash sets and slot sets don't match, or if the
	// peer sent more data than requested.
	if len(hashes) != len(slots) {
		s.lock.Unlock()
		logger.Warn("Hash and slot set size mismatch", "hashset", len(hashes), "slotset", len(slots))
		return errors.New("hash and slot set size mismatch")
	}
	if len(hashes) > len(req.accounts) {
		s.lock.Unlock()
		logger.Warn("Hash set larger than requested", "hashset", len(hashes), "requested", len(req.accounts))
		return errors.New("hash set larger than requested")
	}
	// Response is valid, but check if peer is signalling that it does not have
	// the requested data. For storage range queries that means the state being
	// retrieved was either already pruned remotely, or the peer is not yet
	// synced to our head.
	if len(hashes) == 0 {
		logger.Debug("Peer rejected storage request")
		s.statelessPeers[peer.id] = struct{}{}
		s.lock.Unlock()
		return nil
	}
	s.lock.Unlock()

	// Reconstruct the partial tries from the response and verify them
	var (
		dbs    = make([]ethdb.KeyValueStore, len(hashes))
		tries  = make([]*trie.Trie, len(hashes))
		notary *trie.KeyValueNotary
		cont   bool
	)
	for i := 0; i < len(hashes); i++ {
		// Convert the keys and proofs into an internal format
		keys := make([][]byte, len(hashes[i]))
		for j, key := range hashes[i] {
			keys[j] = common.CopyBytes(key[:])
		}
		nodes := make(light.NodeList, 0, len(proof))
		if i == len(hashes)-1 {
			for _, node := range proof {
				nodes = append(nodes, node)
			}
		}
		var err error
		if len(nodes) == 0 {
			// No proof has been attached, the response must cover the entire key
			// space and hash to the origin root.
			dbs[i], tries[i], _, _, err = trie.VerifyRangeProof(req.roots[i], nil, nil, keys, slots[i], nil)
			if err != nil {
				logger.Warn("Storage slots failed proof", "err", err)
				return err
			}
		} else {
			// A proof was attached, the response is only partial, check that the
			// returned data is indeed part of the storage trie
			proofdb := nodes.NodeSet()

			var end []byte
			if len(keys) > 0 {
				end = keys[len(keys)-1]
			}
			dbs[i], tries[i], notary, cont, err = trie.VerifyRangeProof(req.roots[i], req.origin[:], end, keys, slots[i], proofdb)
			if err != nil {
				logger.Warn("Storage range failed proof", "err", err)
				return err
			}
		}
	}
	// Partial tries reconstructed, send them to the scheduler for storage filling
	bounds := make(map[common.Hash]struct{})

	if notary != nil { // if all contract storages are delivered in full, no notary will be created
		it := notary.Accessed().NewIterator(nil, nil)
		for it.Next() {
			bounds[common.BytesToHash(it.Key())] = struct{}{}
		}
		it.Release()
	}
	response := &storageResponse{
		mainTask: req.mainTask,
		subTask:  req.subTask,
		accounts: req.accounts,
		roots:    req.roots,
		hashes:   hashes,
		slots:    slots,
		nodes:    dbs,
		tries:    tries,
		bounds:   bounds,
		overflow: light.NewNodeSet(),
		cont:     cont,
	}
	select {
	case s.storageResps <- response:
	case <-req.cancel:
	case <-req.stale:
	}
	return nil
}

// OnTrieNodes is a callback method to invoke when a batch of trie nodes
// are received from a remote peer.
func (s *Syncer) OnTrieNodes(peer *Peer, id uint64, trienodes [][]byte) error {
	var size common.StorageSize
	for _, node := range trienodes {
		size += common.StorageSize(len(node))
	}
	logger := peer.logger.New("reqid", id)
	logger.Trace("Delivering set of healing trienodes", "trienodes", len(trienodes), "bytes", size)

	// Whether or not the response is valid, we can mark the peer as idle and
	// notify the scheduler to assign a new task. If the response is invalid,
	// we'll drop the peer in a bit.
	s.lock.Lock()
	if _, ok := s.peers[peer.id]; ok {
		s.trienodeHealIdlers[peer.id] = struct{}{}
	}
	select {
	case s.update <- struct{}{}:
	default:
	}
	// Ensure the response is for a valid request
	req, ok := s.trienodeHealReqs[id]
	if !ok {
		// Request stale, perhaps the peer timed out but came through in the end
		logger.Warn("Unexpected trienode heal packet")
		s.lock.Unlock()
		return nil
	}
	delete(s.trienodeHealReqs, id)

	// Clean up the request timeout timer, we'll see how to proceed further based
	// on the actual delivered content
	req.timeout.Stop()

	// Response is valid, but check if peer is signalling that it does not have
	// the requested data. For bytecode range queries that means the peer is not
	// yet synced.
	if len(trienodes) == 0 {
		logger.Debug("Peer rejected trienode heal request")
		s.statelessPeers[peer.id] = struct{}{}
		s.lock.Unlock()
		return nil
	}
	s.lock.Unlock()

	// Cross reference the requested trienodes with the response to find gaps
	// that the serving node is missing
	hasher := sha3.NewLegacyKeccak256()

	nodes := make([][]byte, len(req.hashes))
	for i, j := 0, 0; i < len(trienodes); i++ {
		// Find the next hash that we've been served, leaving misses with nils
		hasher.Reset()
		hasher.Write(trienodes[i])
		hash := hasher.Sum(nil)

		for j < len(req.hashes) && !bytes.Equal(hash, req.hashes[j][:]) {
			j++
		}
		if j < len(req.hashes) {
			nodes[j] = trienodes[i]
			j++
			continue
		}
		// We've either ran out of hashes, or got unrequested data
		logger.Warn("Unexpected healing trienodes", "count", len(trienodes)-i)
		return errors.New("unexpected healing trienode")
	}
	// Response validated, send it to the scheduler for filling
	response := &trienodeHealResponse{
		task:   req.task,
		hashes: req.hashes,
		paths:  req.paths,
		nodes:  nodes,
	}
	select {
	case s.trienodeHealResps <- response:
	case <-req.cancel:
	case <-req.stale:
	}
	return nil
}

// onHealByteCodes is a callback method to invoke when a batch of contract
// bytes codes are received from a remote peer in the healing phase.
func (s *Syncer) onHealByteCodes(peer *Peer, id uint64, bytecodes [][]byte) error {
	var size common.StorageSize
	for _, code := range bytecodes {
		size += common.StorageSize(len(code))
	}
	logger := peer.logger.New("reqid", id)
	logger.Trace("Delivering set of healing bytecodes", "bytecodes", len(bytecodes), "bytes", size)

	// Whether or not the response is valid, we can mark the peer as idle and
	// notify the scheduler to assign a new task. If the response is invalid,
	// we'll drop the peer in a bit.
	s.lock.Lock()
	if _, ok := s.peers[peer.id]; ok {
		s.bytecodeHealIdlers[peer.id] = struct{}{}
	}
	select {
	case s.update <- struct{}{}:
	default:
	}
	// Ensure the response is for a valid request
	req, ok := s.bytecodeHealReqs[id]
	if !ok {
		// Request stale, perhaps the peer timed out but came through in the end
		logger.Warn("Unexpected bytecode heal packet")
		s.lock.Unlock()
		return nil
	}
	delete(s.bytecodeHealReqs, id)

	// Clean up the request timeout timer, we'll see how to proceed further based
	// on the actual delivered content
	req.timeout.Stop()

	// Response is valid, but check if peer is signalling that it does not have
	// the requested data. For bytecode range queries that means the peer is not
	// yet synced.
	if len(bytecodes) == 0 {
		logger.Debug("Peer rejected bytecode heal request")
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
		logger.Warn("Unexpected healing bytecodes", "count", len(bytecodes)-i)
		return errors.New("unexpected healing bytecode")
	}
	// Response validated, send it to the scheduler for filling
	response := &bytecodeHealResponse{
		task:   req.task,
		hashes: req.hashes,
		codes:  codes,
	}
	select {
	case s.bytecodeHealResps <- response:
	case <-req.cancel:
	case <-req.stale:
	}
	return nil
}

// hashSpace is the total size of the 256 bit hash space for accounts.
var hashSpace = new(big.Int).Exp(common.Big2, common.Big256, nil)

// report calculates various status reports and provides it to the user.
func (s *Syncer) report(force bool) {
	if len(s.tasks) > 0 {
		s.reportSyncProgress(force)
		return
	}
	s.reportHealProgress(force)
}

// reportSyncProgress calculates various status reports and provides it to the user.
func (s *Syncer) reportSyncProgress(force bool) {
	// Don't report all the events, just occasionally
	if !force && time.Since(s.logTime) < 3*time.Second {
		return
	}
	// Don't report anything until we have a meaningful progress
	synced := s.accountBytes + s.bytecodeBytes + s.storageBytes
	if synced == 0 {
		return
	}
	accountGaps := new(big.Int)
	for _, task := range s.tasks {
		accountGaps.Add(accountGaps, new(big.Int).Sub(task.Last.Big(), task.Next.Big()))
	}
	accountFills := new(big.Int).Sub(hashSpace, accountGaps)
	if accountFills.BitLen() == 0 {
		return
	}
	s.logTime = time.Now()
	estBytes := float64(new(big.Int).Div(
		new(big.Int).Mul(new(big.Int).SetUint64(uint64(synced)), hashSpace),
		accountFills,
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
		"accounts", accounts, "slots", storage, "codes", bytecode, "eta", common.PrettyDuration(estTime-elapsed))
}

// reportHealProgress calculates various status reports and provides it to the user.
func (s *Syncer) reportHealProgress(force bool) {
	// Don't report all the events, just occasionally
	if !force && time.Since(s.logTime) < 3*time.Second {
		return
	}
	s.logTime = time.Now()

	// Create a mega progress report
	var (
		trienode = fmt.Sprintf("%d@%v", s.trienodeHealSynced, s.trienodeHealBytes.TerminalString())
		bytecode = fmt.Sprintf("%d@%v", s.bytecodeHealSynced, s.bytecodeHealBytes.TerminalString())
	)
	log.Info("State heal in progress", "nodes", trienode, "codes", bytecode,
		"pending", s.healer.scheduler.Pending())
}
