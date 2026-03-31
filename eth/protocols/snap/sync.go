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
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/msgrate"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
)

const (
	// minRequestSize is the minimum number of bytes to request from a remote peer.
	// This number is used as the low cap for account and storage range requests.
	// Bytecode and trienode are limited inherently by item count (1).
	minRequestSize = 64 * 1024

	// maxRequestSize is the maximum number of bytes to request from a remote peer.
	// This number is used as the high cap for account and storage range requests.
	// Bytecode and trienode are limited more explicitly by the caps below.
	maxRequestSize = 512 * 1024

	// maxCodeRequestCount is the maximum number of bytecode blobs to request in a
	// single query. If this number is too low, we're not filling responses fully
	// and waste round trip times. If it's too high, we're capping responses and
	// waste bandwidth.
	//
	// Deployed bytecodes are currently capped at 24KB, so the minimum request
	// size should be maxRequestSize / 24K. Assuming that most contracts do not
	// come close to that, requesting 4x should be a good approximation.
	maxCodeRequestCount = maxRequestSize / (24 * 1024) * 4

	// maxAccessListRequestCount is the maximum number of block access lists to
	// request in a single query. BALs average ~72 KiB compressed (per EIP-7928),
	// and EIP-8189 recommends a 2 MiB response soft limit, so we target ~28
	// blocks per request to avoid server-side truncation.
	//
	// NOTE: If the gas limit is raised significantly, this number may need to be adjusted
	// to avoid server-side truncation and re-requesting. It is currently based on
	// the assumption that the gas limit is 60M.
	maxAccessListRequestCount = 28
)

var (
	// accountConcurrency is the number of chunks to split the account trie into
	// to allow concurrent retrievals.
	accountConcurrency = 16

	// storageConcurrency is the number of chunks to split a large contract
	// storage trie into to allow concurrent retrievals.
	storageConcurrency = 16
)

// ErrCancelled is returned from snap syncing if the operation was prematurely
// terminated.
var ErrCancelled = errors.New("sync cancelled")

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
	peer string    // Peer to which this request is assigned
	id   uint64    // Request ID of this request
	time time.Time // Timestamp when the request was sent

	deliver chan *accountResponse // Channel to deliver successful response on
	revert  chan *accountRequest  // Channel to deliver request failure on
	cancel  chan struct{}         // Channel to track sync cancellation
	timeout *time.Timer           // Timer to track delivery timeout
	stale   chan struct{}         // Channel to signal the request was dropped

	origin common.Hash // First account requested to allow continuation checks
	limit  common.Hash // Last account requested to allow non-overlapping chunking

	task *accountTask // Task which this request is filling (only access fields through the runloop!!)
}

// accountResponse is an already Merkle-verified remote response to an account
// range request. It contains the subtrie for the requested account range and
// the database that's going to be filled with the internal nodes on commit.
type accountResponse struct {
	task *accountTask // Task which this request is filling

	hashes   []common.Hash         // Account hashes in the returned range
	accounts []*types.StateAccount // Expanded accounts in the returned range

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
	peer string    // Peer to which this request is assigned
	id   uint64    // Request ID of this request
	time time.Time // Timestamp when the request was sent

	deliver chan *bytecodeResponse // Channel to deliver successful response on
	revert  chan *bytecodeRequest  // Channel to deliver request failure on
	cancel  chan struct{}          // Channel to track sync cancellation
	timeout *time.Timer            // Timer to track delivery timeout
	stale   chan struct{}          // Channel to signal the request was dropped

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
// the main runloop to allow Merkle proof verifications on the peer's thread and
// to drop on invalid response. The request struct must contain all the data to
// construct the response without accessing runloop internals (i.e. tasks). That
// is only included to allow the runloop to match a response to the task being
// synced without having yet another set of maps.
type storageRequest struct {
	peer string    // Peer to which this request is assigned
	id   uint64    // Request ID of this request
	time time.Time // Timestamp when the request was sent

	deliver chan *storageResponse // Channel to deliver successful response on
	revert  chan *storageRequest  // Channel to deliver request failure on
	cancel  chan struct{}         // Channel to track sync cancellation
	timeout *time.Timer           // Timer to track delivery timeout
	stale   chan struct{}         // Channel to signal the request was dropped

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

	hashes [][]common.Hash // Storage slot hashes in the returned range
	slots  [][][]byte      // Storage slot values in the returned range

	cont bool // Whether the last storage range has a continuation
}

type accessListRequest struct {
	peer    string                   // Peer to which this request is assigned
	id      uint64                   // Request ID of this request
	hashes  []common.Hash            // Block hashes corresponding to requested BALs
	time    time.Time                // Timestamp when the request was sent
	timeout *time.Timer              // Timer to track the delivery timeout
	deliver chan *accessListResponse // Channel to deliver successful response on
	revert  chan *accessListRequest  // Channel to deliver request failure on
	cancel  chan struct{}            // Channel to track sync cancellation
	stale   chan struct{}            // Channel to signal the request was dropped
}

type accessListResponse struct {
	req         *accessListRequest
	accessLists []rlp.RawValue
}

// accountTask represents the sync task for a chunk of the account snapshot.
type accountTask struct {
	// These fields get serialized to key-value store on shutdown
	Next     common.Hash                    // Next account to sync in this interval
	Last     common.Hash                    // Last account to sync in this interval
	SubTasks map[common.Hash][]*storageTask // Storage intervals needing fetching for large contracts

	// This is a list of account hashes whose storage are already completed
	// in this cycle. This field is newly introduced in v1.14 and will be
	// empty if the task is resolved from legacy progress data. Furthermore,
	// this additional field will be ignored by legacy Geth. The only side
	// effect is that these contracts might be resynced in the new cycle,
	// retaining the legacy behavior.
	StorageCompleted []common.Hash `json:",omitempty"`

	// These fields are internals used during runtime
	req  *accountRequest  // Pending request to fill this task
	res  *accountResponse // Validate response filling this task
	pend int              // Number of pending subtasks for this round

	needCode  []bool // Flags whether the filling accounts need code retrieval
	needState []bool // Flags whether the filling accounts need storage retrieval
	needHeal  []bool // Flags whether the filling accounts's state was chunked and need healing

	codeTasks      map[common.Hash]struct{}    // Code hashes that need retrieval
	stateTasks     map[common.Hash]common.Hash // Account hashes->roots that need full state retrieval
	stateCompleted map[common.Hash]struct{}    // Account hashes whose storage have been completed

	done bool // Flag whether the task can be removed
}

// activeSubTasks returns the set of storage tasks covered by the current account
// range. Normally this would be the entire subTask set, but on a sync interrupt
// and later resume it can happen that a shorter account range is retrieved. This
// method ensures that we only start up the subtasks covered by the latest account
// response.
//
// Nil is returned if the account range is empty.
func (task *accountTask) activeSubTasks() map[common.Hash][]*storageTask {
	if len(task.res.hashes) == 0 {
		return nil
	}
	var (
		tasks = make(map[common.Hash][]*storageTask)
		last  = task.res.hashes[len(task.res.hashes)-1]
	)
	for hash, subTasks := range task.SubTasks {
		if hash.Cmp(last) <= 0 {
			tasks[hash] = subTasks
		}
	}
	return tasks
}

// storageTask represents the sync task for a chunk of the storage snapshot.
type storageTask struct {
	Next common.Hash // Next account to sync in this interval
	Last common.Hash // Last account to sync in this interval

	// These fields are internals used during runtime
	root common.Hash     // Storage root hash for this instance
	req  *storageRequest // Pending request to fill this task

	done bool // Flag whether the task can be removed
}

// SyncProgress is a database entry to allow suspending and resuming a snapshot state
// sync. Opposed to full and fast sync, there is no way to restart a suspended
// snap sync without prior knowledge of the suspension point.
type SyncProgress struct {
	Root        common.Hash    // State root being synced (for pivot move detection)
	BlockNumber uint64         // Block number of the pivot
	Tasks       []*accountTask // The suspended account tasks (contract tasks within)

	// Status report during syncing phase
	AccountSynced  uint64             // Number of accounts downloaded
	AccountBytes   common.StorageSize // Number of account trie bytes persisted to disk
	BytecodeSynced uint64             // Number of bytecodes downloaded
	BytecodeBytes  common.StorageSize // Number of bytecode bytes downloaded
	StorageSynced  uint64             // Number of storage slots downloaded
	StorageBytes   common.StorageSize // Number of storage trie bytes persisted to disk

}

// SyncPeer abstracts out the methods required for a peer to be synced against
// with the goal of allowing the construction of mock peers without the full
// blown networking.
type SyncPeer interface {
	// ID retrieves the peer's unique identifier.
	ID() string

	// RequestAccountRange fetches a batch of accounts rooted in a specific account
	// trie, starting with the origin.
	RequestAccountRange(id uint64, root, origin, limit common.Hash, bytes int) error

	// RequestStorageRanges fetches a batch of storage slots belonging to one or
	// more accounts. If slots from only one account is requested, an origin marker
	// may also be used to retrieve from there.
	RequestStorageRanges(id uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, bytes int) error

	// RequestByteCodes fetches a batch of bytecodes by hash.
	RequestByteCodes(id uint64, hashes []common.Hash, bytes int) error

	// RequestAccessLists fetches a batch of BALs by block hash.
	RequestAccessLists(id uint64, hashes []common.Hash, bytes int) error

	// Log retrieves the peer's own contextual logger.
	Log() log.Logger
}

// Syncer is an Ethereum account and storage trie syncer based on the snap
// protocol. It downloads all accounts, storage slots, and bytecodes from
// remote peers as flat state, applies BAL diffs on pivot moves,
// and triggers a final trie rebuild once flat state is consistent.
//
// Every network request has a variety of failure events:
//   - The peer disconnects after task assignment, failing to send the request
//   - The peer disconnects after sending the request, before delivering on it
//   - The peer remains connected, but does not deliver a response in time
//   - The peer delivers a stale response after a previous timeout
//   - The peer delivers a refusal to serve the requested state
type Syncer struct {
	db     ethdb.Database // Database to store the trie nodes into (and dedup)
	scheme string         // Node scheme used in node database

	root           common.Hash    // Current state trie root being synced
	number         uint64         // Block number of the current pivot
	previousRoot   common.Hash    // Root from previous sync run (for pivot move detection)
	previousNumber uint64         // Block number of the previous pivot
	tasks          []*accountTask // Current account task set being synced
	update         chan struct{}  // Notification channel for possible sync progression

	peers    map[string]SyncPeer // Currently active peers to download from
	peerJoin *event.Feed         // Event feed to react to peers joining
	peerDrop *event.Feed         // Event feed to react to peers dropping
	rates    *msgrate.Trackers   // Message throughput rates for peers

	// Request tracking during syncing phase
	statelessPeers   map[string]struct{} // Peers that failed to deliver state data
	accountIdlers    map[string]struct{} // Peers that aren't serving account requests
	bytecodeIdlers   map[string]struct{} // Peers that aren't serving bytecode requests
	storageIdlers    map[string]struct{} // Peers that aren't serving storage requests
	accessListIdlers map[string]struct{} // Peers that aren't serving access list requests

	accountReqs    map[uint64]*accountRequest    // Account requests currently running
	bytecodeReqs   map[uint64]*bytecodeRequest   // Bytecode requests currently running
	storageReqs    map[uint64]*storageRequest    // Storage requests currently running
	accessListReqs map[uint64]*accessListRequest // Access list requests currently running

	accountSynced  uint64             // Number of accounts downloaded
	accountBytes   common.StorageSize // Number of account trie bytes persisted to disk
	bytecodeSynced uint64             // Number of bytecodes downloaded
	bytecodeBytes  common.StorageSize // Number of bytecode bytes downloaded
	storageSynced  uint64             // Number of storage slots downloaded
	storageBytes   common.StorageSize // Number of storage trie bytes persisted to disk

	extProgress *SyncProgress // progress that can be exposed to external caller.

	startTime time.Time // Time instance when snapshot sync started
	logTime   time.Time // Time instance when status was last reported

	pend sync.WaitGroup // Tracks network request goroutines for graceful shutdown
	lock sync.RWMutex   // Protects fields that can change outside of sync (peers, reqs, root)
}

// NewSyncer creates a new snapshot syncer to download the Ethereum state over the
// snap protocol.
func NewSyncer(db ethdb.Database, scheme string) *Syncer {
	return &Syncer{
		db:     db,
		scheme: scheme,

		peers:    make(map[string]SyncPeer),
		peerJoin: new(event.Feed),
		peerDrop: new(event.Feed),
		rates:    msgrate.NewTrackers(log.New("proto", "snap")),
		update:   make(chan struct{}, 1),

		statelessPeers:   make(map[string]struct{}),
		accountIdlers:    make(map[string]struct{}),
		storageIdlers:    make(map[string]struct{}),
		bytecodeIdlers:   make(map[string]struct{}),
		accessListIdlers: make(map[string]struct{}),

		accountReqs:    make(map[uint64]*accountRequest),
		storageReqs:    make(map[uint64]*storageRequest),
		bytecodeReqs:   make(map[uint64]*bytecodeRequest),
		accessListReqs: make(map[uint64]*accessListRequest),

		extProgress: new(SyncProgress),
	}
}

// Register injects a new data source into the syncer's peerset.
func (s *Syncer) Register(peer SyncPeer) error {
	// Make sure the peer is not registered yet
	id := peer.ID()

	s.lock.Lock()
	if _, ok := s.peers[id]; ok {
		log.Error("Snap peer already registered", "id", id)

		s.lock.Unlock()
		return errors.New("already registered")
	}
	s.peers[id] = peer
	s.rates.Track(id, msgrate.NewTracker(s.rates.MeanCapacities(), s.rates.MedianRoundTrip()))

	// Mark the peer as idle, even if no sync is running
	s.accountIdlers[id] = struct{}{}
	s.storageIdlers[id] = struct{}{}
	s.bytecodeIdlers[id] = struct{}{}
	s.accessListIdlers[id] = struct{}{}
	s.lock.Unlock()

	// Notify any active syncs that a new peer can be assigned data
	s.peerJoin.Send(id)
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
	s.rates.Untrack(id)

	// Remove status markers, even if no sync is running
	delete(s.statelessPeers, id)

	delete(s.accountIdlers, id)
	delete(s.storageIdlers, id)
	delete(s.bytecodeIdlers, id)
	delete(s.accessListIdlers, id)
	s.lock.Unlock()

	// Notify any active syncs that pending requests need to be reverted
	s.peerDrop.Send(id)
	return nil
}

// errPivotStale is returned from download when the pivot has become stale
// and the syncer needs to perform access list catch-up before continuing.
var errPivotStale = errors.New("pivot stale")

// Sync starts (or resumes a previous) sync cycle to iterate over a state trie
// with the given root and reconstruct the nodes based on the snapshot leaves.
// The number parameter is the block number of the pivot block.
func (s *Syncer) Sync(root common.Hash, number uint64, cancel chan struct{}) error {
	s.lock.Lock()
	s.root = root
	s.number = number
	s.previousRoot = root // Default: no pivot move. loadSyncStatus may overwrite.
	s.previousNumber = number
	s.statelessPeers = make(map[string]struct{})
	s.lock.Unlock()
	if s.startTime.IsZero() {
		s.startTime = time.Now()
	}

	// Retrieve the previous sync status from DB. If there's no persisted
	// status, sync is either fresh or already complete.
	s.loadSyncStatus()
	var syncComplete bool
	defer func() {
		if !syncComplete {
			for _, task := range s.tasks {
				s.forwardAccountTask(task)
			}
			s.cleanAccountTasks()
			s.saveSyncStatus()
		}
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
		s.accessListReqs = make(map[uint64]*accessListRequest)
		s.lock.Unlock()
	}()

	// Sync loop
	log.Info("Starting state download", "root", root)
	for {
		// Download: fetch all required state data
		err := s.download(cancel)
		if err == errPivotStale {
			// Pivot moved: catch up to new pivot
			if err := s.catchUp(cancel); err != nil {
				return err
			}
			s.resetDownload(root, number)
			log.Info("Resuming state download", "root", root)
			continue
		}

		// Download error that isn't a stale pivot. This is typically due to
		// the downloader cancelling the sync because the pivot moved. This
		// error propagates to the downloader which will restart the sync with
		// a new root.
		if err != nil {
			return err
		}
		log.Info("State download complete", "root", root)

		// Trie rebuild: build all tries from flat state and verify root
		log.Info("Starting trie rebuild", "root", root)
		if err := triedb.GenerateTrie(s.db, s.scheme, root); err != nil {
			return err
		}
		log.Info("Trie rebuild complete", "root", root)

		// Sync complete: clear persisted status so we don't re-run.
		// Set syncComplete to prevent the deferred saveSyncStatus from
		// overwriting the nil.
		syncComplete = true
		rawdb.WriteSnapshotSyncStatus(s.db, nil)
		return nil
	}
}

// download runs the bulk flat-state download. It fetches
// account ranges, storage slots, and bytecodes, writing flat state to disk.
func (s *Syncer) download(cancel chan struct{}) error {
	// If the pivot moved since the last run (downloader cancelled and restarted
	// us with a new root), signal catch-up before downloading.
	if s.previousRoot != s.root {
		return errPivotStale
	}

	// Subscribe to peer events
	peerJoin := make(chan string, 16)
	peerJoinSub := s.peerJoin.Subscribe(peerJoin)
	defer peerJoinSub.Unsubscribe()
	peerDrop := make(chan string, 16)
	peerDropSub := s.peerDrop.Subscribe(peerDrop)
	defer peerDropSub.Unsubscribe()

	// Create ephemeral channels for this download cycle
	var (
		accountReqFails  = make(chan *accountRequest)
		storageReqFails  = make(chan *storageRequest)
		bytecodeReqFails = make(chan *bytecodeRequest)
		accountResps     = make(chan *accountResponse)
		storageResps     = make(chan *storageResponse)
		bytecodeResps    = make(chan *bytecodeResponse)
	)
	for {
		// Remove all completed tasks and terminate if everything's done
		s.cleanStorageTasks()
		s.cleanAccountTasks()
		if len(s.tasks) == 0 {
			return nil
		}

		// Assign all the data retrieval tasks to any free peers
		s.assignAccountTasks(accountResps, accountReqFails, cancel)
		s.assignBytecodeTasks(bytecodeResps, bytecodeReqFails, cancel)
		s.assignStorageTasks(storageResps, storageReqFails, cancel)

		// Update sync progress
		s.lock.Lock()
		s.extProgress = &SyncProgress{
			AccountSynced:  s.accountSynced,
			AccountBytes:   s.accountBytes,
			BytecodeSynced: s.bytecodeSynced,
			BytecodeBytes:  s.bytecodeBytes,
			StorageSynced:  s.storageSynced,
			StorageBytes:   s.storageBytes,
		}
		s.lock.Unlock()

		// Wait for something to happen
		select {
		case <-s.update:
			// Something happened (new peer, delivery, timeout), recheck tasks
		case <-peerJoin:
			// A new peer joined, try to schedule it new tasks
		case id := <-peerDrop:
			s.revertRequests(id)
		case <-cancel:
			return ErrCancelled

		case req := <-accountReqFails:
			s.revertAccountRequest(req)
		case req := <-bytecodeReqFails:
			s.revertBytecodeRequest(req)
		case req := <-storageReqFails:
			s.revertStorageRequest(req)

		case res := <-accountResps:
			s.processAccountResponse(res)
		case res := <-bytecodeResps:
			s.processBytecodeResponse(res)
		case res := <-storageResps:
			s.processStorageResponse(res)
		}

		// Report stats if something meaningful happened
		s.report(false)
	}
}

// resetDownload resets the download state for a new pivot after catch-up.
// It regenerates the task list for accounts not yet downloaded, clears
// in-flight requests, and updates the root.
func (s *Syncer) resetDownload(root common.Hash, number uint64) {
	s.lock.Lock()
	s.root = root
	s.number = number
	s.previousRoot = root // Prevent download() from returning errPivotStale again
	s.previousNumber = number

	// Clear stateless peers bc they may be able to serve the new pivot
	s.statelessPeers = make(map[string]struct{})
	s.lock.Unlock()
}

// catchUp runs the BAL catch-up. When the pivot has moved (previousRoot !=
// root), it fetches BALs for the gap blocks, verifies them against
// block headers, and applies the diffs to roll flat state forward.
func (s *Syncer) catchUp(cancel chan struct{}) error {
	s.lock.RLock()
	from := s.previousNumber + 1
	to := s.number
	s.lock.RUnlock()

	// The new pivot must be ahead of the old one. This can fail if a reorg
	// replaced the block at the pivot height (same number, different root)
	// or if a deep reorg shortened the chain past the old pivot. In either
	// case, catch-up can't roll forward, so wipe progress and return an
	// error so the caller restarts with a fresh sync.
	//
	// Note: this check lives here rather than in checkDeepReorg because
	// catchUp is reached both when the downloader actively moves the pivot
	// (via restartSnapSync) and when the syncer resumes from persisted
	// progress after a restart. checkDeepReorg only covers the former.
	if from > to {
		log.Warn("Catch-up range inverted, wiping sync progress", "from", from, "to", to)
		rawdb.WriteSnapshotSyncStatus(s.db, nil)
		return fmt.Errorf("catch-up range inverted (from %d > to %d): pivot reorged", from, to)
	}
	log.Info("Starting access list catch-up", "from", from, "to", to, "blocks", to-from+1)

	// Collect block hashes for the gap range
	hashes := make([]common.Hash, 0, to-from+1)
	for num := from; num <= to; num++ {
		hash := rawdb.ReadCanonicalHash(s.db, num)
		if hash == (common.Hash{}) {
			return fmt.Errorf("missing canonical hash for block %d during catch-up", num)
		}
		hashes = append(hashes, hash)
	}

	// Fetch BALs from peers
	rawBALs, err := s.fetchAccessLists(hashes, cancel)
	if err != nil {
		return err
	}

	// Verify and apply each BAL in block order
	for i, raw := range rawBALs {
		num := from + uint64(i)
		hash := hashes[i]

		// Decode the raw RLP into a BlockAccessList
		var bal bal.BlockAccessList
		if err := rlp.DecodeBytes(raw, &bal); err != nil {
			return fmt.Errorf("failed to decode BAL for block %d: %v", num, err)
		}

		// Verify against the block header
		header := rawdb.ReadHeader(s.db, hash, num)
		if header == nil {
			return fmt.Errorf("missing header for block %d (hash %v) during catch-up", num, hash)
		}
		if err := verifyAccessList(&bal, header); err != nil {
			return fmt.Errorf("BAL verification failed for block %d: %v", num, err)
		}

		// Apply the state diffs
		if err := s.applyAccessList(&bal); err != nil {
			return fmt.Errorf("BAL application failed for block %d: %v", num, err)
		}
	}
	log.Info("Access list catch-up complete", "blocks", len(rawBALs))
	return nil
}

// fetchAccessLists fetches BALs for the given block hashes from
// remote peers. It runs its own event loop to assign requests
// to idle peers and process responses asynchronously. Results are returned in
// the same order as the input hashes.
func (s *Syncer) fetchAccessLists(hashes []common.Hash, cancel chan struct{}) ([]rlp.RawValue, error) {
	log.Debug("Fetching access lists for catch-up", "blocks", len(hashes))

	// Subscribe to peer events
	peerJoin := make(chan string, 16)
	peerJoinSub := s.peerJoin.Subscribe(peerJoin)
	defer peerJoinSub.Unsubscribe()
	peerDrop := make(chan string, 16)
	peerDropSub := s.peerDrop.Subscribe(peerDrop)
	defer peerDropSub.Unsubscribe()

	// pending = hashes not yet assigned to a peer, fetched = collected results.
	pending := make(map[common.Hash]struct{}, len(hashes))
	for _, h := range hashes {
		pending[h] = struct{}{}
	}
	fetched := make(map[common.Hash]rlp.RawValue, len(hashes))

	// Create ephemeral channels for this fetch cycle
	var (
		accessListReqFails = make(chan *accessListRequest)
		accessListResps    = make(chan *accessListResponse)
	)
	for len(fetched) < len(hashes) {
		// Assign access list retrieval tasks to idle peers
		s.assignAccessListTasks(pending, accessListResps, accessListReqFails, cancel)

		// Wait for something to happen
		select {
		case <-s.update:
			// Something happened (new peer, delivery, timeout), recheck
		case <-peerJoin:
			// A new peer joined, try to assign it work
		case id := <-peerDrop:
			// Re-add hashes from any requests for this peer
			s.lock.Lock()
			for _, req := range s.accessListReqs {
				if req.peer == id {
					for _, h := range req.hashes {
						pending[h] = struct{}{}
					}
				}
			}
			s.lock.Unlock()
			s.revertRequests(id)
		case <-cancel:
			return nil, ErrCancelled

		case req := <-accessListReqFails:
			s.revertAccessListRequest(req)
			for _, h := range req.hashes {
				pending[h] = struct{}{}
			}
		case res := <-accessListResps:
			s.processAccessListResponse(res, pending, fetched)
		}
	}
	// Assemble results in input order
	results := make([]rlp.RawValue, len(hashes))
	for i, h := range hashes {
		results[i] = fetched[h]
	}
	return results, nil
}

// assignAccessListTasks attempts to assign access list fetch requests to idle
// peers for any hashes still in pending.
func (s *Syncer) assignAccessListTasks(pending map[common.Hash]struct{}, success chan *accessListResponse, fail chan *accessListRequest, cancel chan struct{}) {
	s.lock.Lock()
	defer s.lock.Unlock()
	idlers := s.sortIdlePeers(s.accessListIdlers, AccessListsMsg)

	// Iterate over pending hashes and assign to idle peers
	for len(idlers.ids) > 0 && len(pending) > 0 {
		var (
			idle = idlers.ids[0]
			peer = s.peers[idle]
			cap  = idlers.caps[0]
		)
		idlers.ids, idlers.caps = idlers.ids[1:], idlers.caps[1:]

		// Generate a unique request ID
		var reqid uint64
		for {
			reqid = uint64(rand.Int63())
			if reqid == 0 {
				continue
			}
			if _, ok := s.accessListReqs[reqid]; ok {
				continue
			}
			break
		}

		// Collect hashes to fetch, capped by peer capacity and the
		// EIP-8189 2 MiB response soft limit (~72 KiB/BAL -> 28 blocks).
		if cap > maxAccessListRequestCount {
			cap = maxAccessListRequestCount
		}
		batch := make([]common.Hash, 0, cap)
		for h := range pending {
			delete(pending, h)
			batch = append(batch, h)
			if len(batch) >= cap {
				break
			}
		}
		req := &accessListRequest{
			peer:    idle,
			id:      reqid,
			hashes:  batch,
			time:    time.Now(),
			deliver: success,
			revert:  fail,
			cancel:  cancel,
			stale:   make(chan struct{}),
		}
		req.timeout = time.AfterFunc(s.rates.TargetTimeout(), func() {
			peer.Log().Debug("Access list request timed out", "reqid", reqid)
			s.rates.Update(idle, AccessListsMsg, 0, 0)
			s.scheduleRevertAccessListRequest(req)
		})
		s.accessListReqs[reqid] = req
		delete(s.accessListIdlers, idle)

		s.pend.Add(1)
		go func() {
			defer s.pend.Done()

			// Attempt to send the remote request and revert if it fails
			if err := peer.RequestAccessLists(reqid, batch, softResponseLimit); err != nil {
				log.Debug("Failed to request access lists", "err", err)
				s.scheduleRevertAccessListRequest(req)
			}
		}()
	}
}

// processAccessListResponse handles a successful access list response by
// matching results to pending hashes and storing them.
func (s *Syncer) processAccessListResponse(res *accessListResponse, pending map[common.Hash]struct{}, fetched map[common.Hash]rlp.RawValue) {
	// Each response entry corresponds to the requested hash at the same index
	for i, raw := range res.accessLists {
		if i >= len(res.req.hashes) {
			break
		}
		h := res.req.hashes[i]
		fetched[h] = raw
		delete(pending, h)
	}
	// Re-add hashes that were not served back to pending
	for i := len(res.accessLists); i < len(res.req.hashes); i++ {
		pending[res.req.hashes[i]] = struct{}{}
	}
}

// loadSyncStatus retrieves a previously aborted sync status from the database,
// or generates a fresh one if none is available.
func (s *Syncer) loadSyncStatus() {
	var progress SyncProgress

	if status := rawdb.ReadSnapshotSyncStatus(s.db); status != nil {
		if err := json.Unmarshal(status, &progress); err != nil {
			log.Error("Failed to decode snap sync status", "err", err)
		} else {
			for _, task := range progress.Tasks {
				log.Debug("Scheduled account sync task", "from", task.Next, "last", task.Last)
			}
			s.tasks = progress.Tasks
			for _, task := range s.tasks {
				// Restore the completed storages
				task.stateCompleted = make(map[common.Hash]struct{})
				for _, hash := range task.StorageCompleted {
					task.stateCompleted[hash] = struct{}{}
				}
				task.StorageCompleted = nil
			}
			s.lock.Lock()
			defer s.lock.Unlock()

			s.previousRoot = progress.Root
			s.previousNumber = progress.BlockNumber
			s.accountSynced = progress.AccountSynced
			s.accountBytes = progress.AccountBytes
			s.bytecodeSynced = progress.BytecodeSynced
			s.bytecodeBytes = progress.BytecodeBytes
			s.storageSynced = progress.StorageSynced
			s.storageBytes = progress.StorageBytes
			return
		}
	}
	// Either we've failed to decode the previous state, or there was none.
	// Start a fresh sync by chunking up the account range and scheduling
	// them for retrieval.
	s.tasks = nil
	s.accountSynced, s.accountBytes = 0, 0
	s.bytecodeSynced, s.bytecodeBytes = 0, 0
	s.storageSynced, s.storageBytes = 0, 0

	var next common.Hash
	step := new(big.Int).Sub(
		new(big.Int).Div(
			new(big.Int).Exp(common.Big2, common.Big256, nil),
			big.NewInt(int64(accountConcurrency)),
		), common.Big1,
	)
	for i := 0; i < accountConcurrency; i++ {
		last := common.BigToHash(new(big.Int).Add(next.Big(), step))
		if i == accountConcurrency-1 {
			// Make sure we don't overflow if the step is not a proper divisor
			last = common.MaxHash
		}
		s.tasks = append(s.tasks, &accountTask{
			Next:           next,
			Last:           last,
			SubTasks:       make(map[common.Hash][]*storageTask),
			stateCompleted: make(map[common.Hash]struct{}),
		})
		log.Debug("Created account sync task", "from", next, "last", last)
		next = common.BigToHash(new(big.Int).Add(last.Big(), common.Big1))
	}
}

// saveSyncStatus marshals the remaining sync tasks into leveldb.
func (s *Syncer) saveSyncStatus() {
	// Serialize any partial progress to disk before spinning down
	for _, task := range s.tasks {
		// Save the account hashes of completed storage.
		task.StorageCompleted = make([]common.Hash, 0, len(task.stateCompleted))
		for hash := range task.stateCompleted {
			task.StorageCompleted = append(task.StorageCompleted, hash)
		}
		if len(task.StorageCompleted) > 0 {
			log.Debug("Leftover completed storages", "number", len(task.StorageCompleted), "next", task.Next, "last", task.Last)
		}
	}
	// Store the actual progress markers
	progress := &SyncProgress{
		Root:           s.root,
		BlockNumber:    s.number,
		Tasks:          s.tasks,
		AccountSynced:  s.accountSynced,
		AccountBytes:   s.accountBytes,
		BytecodeSynced: s.bytecodeSynced,
		BytecodeBytes:  s.bytecodeBytes,
		StorageSynced:  s.storageSynced,
		StorageBytes:   s.storageBytes,
	}
	status, err := json.Marshal(progress)
	if err != nil {
		panic(err) // This can only fail during implementation
	}
	rawdb.WriteSnapshotSyncStatus(s.db, status)
}

// Progress returns the snap sync status statistics.
func (s *Syncer) Progress() *SyncProgress {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.extProgress
}

// cleanAccountTasks removes account range retrieval tasks that have already been
// completed.
func (s *Syncer) cleanAccountTasks() {
	// If the sync was already done before, don't even bother
	if len(s.tasks) == 0 {
		return
	}
	// Sync wasn't finished previously, check for any task that can be finalized
	for i := 0; i < len(s.tasks); i++ {
		if s.tasks[i].done {
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			i--
		}
	}
	// If everything was just finalized, push the final sync report
	if len(s.tasks) == 0 {
		s.reportSyncProgress(true)
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

			// Mark the state as complete to prevent resyncing, regardless
			// if state healing is necessary.
			task.stateCompleted[account] = struct{}{}

			// If this was the last pending task, forward the account task
			if task.pend == 0 {
				s.forwardAccountTask(task)
			}
		}
	}
}

// assignAccountTasks attempts to match idle peers to pending account range
// retrievals.
func (s *Syncer) assignAccountTasks(success chan *accountResponse, fail chan *accountRequest, cancel chan struct{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Sort the peers by download capacity to use faster ones if many available
	idlers := s.sortIdlePeers(s.accountIdlers, AccountRangeMsg)
	if len(idlers.ids) == 0 {
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
			if _, ok := s.accountReqs[reqid]; ok {
				continue
			}
			break
		}
		// Generate the network query and send it to the peer
		req := &accountRequest{
			peer:    idle,
			id:      reqid,
			time:    time.Now(),
			deliver: success,
			revert:  fail,
			cancel:  cancel,
			stale:   make(chan struct{}),
			origin:  task.Next,
			limit:   task.Last,
			task:    task,
		}
		req.timeout = time.AfterFunc(s.rates.TargetTimeout(), func() {
			peer.Log().Debug("Account range request timed out", "reqid", reqid)
			s.rates.Update(idle, AccountRangeMsg, 0, 0)
			s.scheduleRevertAccountRequest(req)
		})
		s.accountReqs[reqid] = req
		delete(s.accountIdlers, idle)

		s.pend.Add(1)
		go func(root common.Hash) {
			defer s.pend.Done()

			// Attempt to send the remote request and revert if it fails
			if cap > maxRequestSize {
				cap = maxRequestSize
			}
			if cap < minRequestSize { // Don't bother with peers below a bare minimum performance
				cap = minRequestSize
			}
			if err := peer.RequestAccountRange(reqid, root, req.origin, req.limit, cap); err != nil {
				peer.Log().Debug("Failed to request account range", "err", err)
				s.scheduleRevertAccountRequest(req)
			}
		}(s.root)

		// Inject the request into the task to block further assignments
		task.req = req
	}
}

// assignBytecodeTasks attempts to match idle peers to pending code retrievals.
func (s *Syncer) assignBytecodeTasks(success chan *bytecodeResponse, fail chan *bytecodeRequest, cancel chan struct{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	idlers := s.sortIdlePeers(s.bytecodeIdlers, ByteCodesMsg)
	if len(idlers.ids) == 0 {
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
			if _, ok := s.bytecodeReqs[reqid]; ok {
				continue
			}
			break
		}
		// Generate the network query and send it to the peer
		if cap > maxCodeRequestCount {
			cap = maxCodeRequestCount
		}
		hashes := make([]common.Hash, 0, cap)
		for hash := range task.codeTasks {
			delete(task.codeTasks, hash)
			hashes = append(hashes, hash)
			if len(hashes) >= cap {
				break
			}
		}
		req := &bytecodeRequest{
			peer:    idle,
			id:      reqid,
			time:    time.Now(),
			deliver: success,
			revert:  fail,
			cancel:  cancel,
			stale:   make(chan struct{}),
			hashes:  hashes,
			task:    task,
		}
		req.timeout = time.AfterFunc(s.rates.TargetTimeout(), func() {
			peer.Log().Debug("Bytecode request timed out", "reqid", reqid)
			s.rates.Update(idle, ByteCodesMsg, 0, 0)
			s.scheduleRevertBytecodeRequest(req)
		})
		s.bytecodeReqs[reqid] = req
		delete(s.bytecodeIdlers, idle)

		s.pend.Add(1)
		go func() {
			defer s.pend.Done()

			// Attempt to send the remote request and revert if it fails
			if err := peer.RequestByteCodes(reqid, hashes, maxRequestSize); err != nil {
				log.Debug("Failed to request bytecodes", "err", err)
				s.scheduleRevertBytecodeRequest(req)
			}
		}()
	}
}

// assignStorageTasks attempts to match idle peers to pending storage range
// retrievals.
func (s *Syncer) assignStorageTasks(success chan *storageResponse, fail chan *storageRequest, cancel chan struct{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	idlers := s.sortIdlePeers(s.storageIdlers, StorageRangesMsg)
	if len(idlers.ids) == 0 {
		return
	}
	// Iterate over all the tasks and try to find a pending one
	for _, task := range s.tasks {
		// Skip any tasks not in the storage retrieval phase
		if task.res == nil {
			continue
		}
		// Skip tasks that are already retrieving (or done with) all small states
		storageTasks := task.activeSubTasks()
		if len(storageTasks) == 0 && len(task.stateTasks) == 0 {
			continue
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
			if _, ok := s.storageReqs[reqid]; ok {
				continue
			}
			break
		}
		// Generate the network query and send it to the peer. If there are
		// large contract tasks pending, complete those before diving into
		// even more new contracts.
		if cap > maxRequestSize {
			cap = maxRequestSize
		}
		if cap < minRequestSize { // Don't bother with peers below a bare minimum performance
			cap = minRequestSize
		}
		storageSets := cap / 1024

		var (
			accounts = make([]common.Hash, 0, storageSets)
			roots    = make([]common.Hash, 0, storageSets)
			subtask  *storageTask
		)
		for account, subtasks := range storageTasks {
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
			for account, root := range task.stateTasks {
				delete(task.stateTasks, account)

				accounts = append(accounts, account)
				roots = append(roots, root)

				if len(accounts) >= storageSets {
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
			time:     time.Now(),
			deliver:  success,
			revert:   fail,
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
		req.timeout = time.AfterFunc(s.rates.TargetTimeout(), func() {
			peer.Log().Debug("Storage request timed out", "reqid", reqid)
			s.rates.Update(idle, StorageRangesMsg, 0, 0)
			s.scheduleRevertStorageRequest(req)
		})
		s.storageReqs[reqid] = req
		delete(s.storageIdlers, idle)

		s.pend.Add(1)
		go func(root common.Hash) {
			defer s.pend.Done()

			// Attempt to send the remote request and revert if it fails
			var origin, limit []byte
			if subtask != nil {
				origin, limit = req.origin[:], req.limit[:]
			}
			if err := peer.RequestStorageRanges(reqid, root, accounts, origin, limit, cap); err != nil {
				log.Debug("Failed to request storage", "err", err)
				s.scheduleRevertStorageRequest(req)
			}
		}(s.root)

		// Inject the request into the subtask to block further assignments
		if subtask != nil {
			subtask.req = req
		}
	}
}

// revertRequests locates all the currently pending requests from a particular
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
	var accessListReqs []*accessListRequest
	for _, req := range s.accessListReqs {
		if req.peer == peer {
			accessListReqs = append(accessListReqs, req)
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
	for _, req := range accessListReqs {
		s.revertAccessListRequest(req)
	}
}

// scheduleRevertAccountRequest asks the event loop to clean up an account range
// request and return all failed retrieval tasks to the scheduler for reassignment.
func (s *Syncer) scheduleRevertAccountRequest(req *accountRequest) {
	select {
	case req.revert <- req:
		// Sync event loop notified
	case <-req.cancel:
		// Sync cycle got cancelled
	case <-req.stale:
		// Request already reverted
	}
}

// revertAccountRequest cleans up an account range request and returns all failed
// retrieval tasks to the scheduler for reassignment.
//
// Note, this needs to run on the event runloop thread to reschedule to idle peers.
// On peer threads, use scheduleRevertAccountRequest.
func (s *Syncer) revertAccountRequest(req *accountRequest) {
	log.Debug("Reverting account request", "peer", req.peer, "reqid", req.id)
	select {
	case <-req.stale:
		log.Trace("Account request already reverted", "peer", req.peer, "reqid", req.id)
		return
	default:
	}
	close(req.stale)

	// Remove the request from the tracked set and restore the peer to the
	// idle pool so it can be reassigned work (skip if peer already left).
	s.lock.Lock()
	delete(s.accountReqs, req.id)
	if _, ok := s.peers[req.peer]; ok {
		s.accountIdlers[req.peer] = struct{}{}
	}
	s.lock.Unlock()

	// If there's a timeout timer still running, abort it and mark the account
	// task as not-pending, ready for rescheduling
	req.timeout.Stop()
	if req.task.req == req {
		req.task.req = nil
	}
}

// scheduleRevertBytecodeRequest asks the event loop to clean up a bytecode request
// and return all failed retrieval tasks to the scheduler for reassignment.
func (s *Syncer) scheduleRevertBytecodeRequest(req *bytecodeRequest) {
	select {
	case req.revert <- req:
		// Sync event loop notified
	case <-req.cancel:
		// Sync cycle got cancelled
	case <-req.stale:
		// Request already reverted
	}
}

// revertBytecodeRequest cleans up a bytecode request and returns all failed
// retrieval tasks to the scheduler for reassignment.
//
// Note, this needs to run on the event runloop thread to reschedule to idle peers.
// On peer threads, use scheduleRevertBytecodeRequest.
func (s *Syncer) revertBytecodeRequest(req *bytecodeRequest) {
	log.Debug("Reverting bytecode request", "peer", req.peer)
	select {
	case <-req.stale:
		log.Trace("Bytecode request already reverted", "peer", req.peer, "reqid", req.id)
		return
	default:
	}
	close(req.stale)

	// Remove the request from the tracked set and restore the peer to the
	// idle pool so it can be reassigned work (skip if peer already left).
	s.lock.Lock()
	delete(s.bytecodeReqs, req.id)
	if _, ok := s.peers[req.peer]; ok {
		s.bytecodeIdlers[req.peer] = struct{}{}
	}
	s.lock.Unlock()

	// If there's a timeout timer still running, abort it and mark the code
	// retrievals as not-pending, ready for rescheduling
	req.timeout.Stop()
	for _, hash := range req.hashes {
		req.task.codeTasks[hash] = struct{}{}
	}
}

// scheduleRevertStorageRequest asks the event loop to clean up a storage range
// request and return all failed retrieval tasks to the scheduler for reassignment.
func (s *Syncer) scheduleRevertStorageRequest(req *storageRequest) {
	select {
	case req.revert <- req:
		// Sync event loop notified
	case <-req.cancel:
		// Sync cycle got cancelled
	case <-req.stale:
		// Request already reverted
	}
}

// revertStorageRequest cleans up a storage range request and returns all failed
// retrieval tasks to the scheduler for reassignment.
//
// Note, this needs to run on the event runloop thread to reschedule to idle peers.
// On peer threads, use scheduleRevertStorageRequest.
func (s *Syncer) revertStorageRequest(req *storageRequest) {
	log.Debug("Reverting storage request", "peer", req.peer)
	select {
	case <-req.stale:
		log.Trace("Storage request already reverted", "peer", req.peer, "reqid", req.id)
		return
	default:
	}
	close(req.stale)

	// Remove the request from the tracked set and restore the peer to the
	// idle pool so it can be reassigned work (skip if peer already left).
	s.lock.Lock()
	delete(s.storageReqs, req.id)
	if _, ok := s.peers[req.peer]; ok {
		s.storageIdlers[req.peer] = struct{}{}
	}
	s.lock.Unlock()

	// If there's a timeout timer still running, abort it and mark the storage
	// task as not-pending, ready for rescheduling
	req.timeout.Stop()
	if req.subTask != nil {
		req.subTask.req = nil
	} else {
		for i, account := range req.accounts {
			req.mainTask.stateTasks[account] = req.roots[i]
		}
	}
}

// scheduleRevertAccessListRequest asks the event loop to clean up an access
// list request and return all failed retrieval tasks for reassignment.
//
// Note, this needs to run on the event runloop thread to reschedule to idle
// peers. On peer threads, use scheduleRevertAccessListRequest.
func (s *Syncer) scheduleRevertAccessListRequest(req *accessListRequest) {
	select {
	case req.revert <- req:
		// Sync event loop notified
	case <-req.cancel:
		// Sync cycle got cancelled
	case <-req.stale:
		// Request already reverted
	}
}

// revertAccessListRequest cleans up an access list request and returns all
// failed retrieval tasks to the scheduler for reassignment.
func (s *Syncer) revertAccessListRequest(req *accessListRequest) {
	log.Debug("Reverting access list request", "peer", req.peer)
	select {
	case <-req.stale:
		log.Trace("Access list request already reverted", "peer", req.peer, "reqid", req.id)
		return
	default:
	}
	close(req.stale)

	// Remove the request from the tracked set and restore the peer to the
	// idle pool so it can be reassigned work (skip if peer already left).
	s.lock.Lock()
	delete(s.accessListReqs, req.id)
	if _, ok := s.peers[req.peer]; ok {
		s.accessListIdlers[req.peer] = struct{}{}
	}
	s.lock.Unlock()

	req.timeout.Stop()
	// Hashes remain in the pending map and will be retried on the next loop iteration
}

// processAccountResponse integrates an already validated account range response
// into the account tasks.
func (s *Syncer) processAccountResponse(res *accountResponse) {
	// Switch the task from pending to filling
	res.task.req = nil
	res.task.res = res

	// Ensure that the response doesn't overflow into the subsequent task
	lastBig := res.task.Last.Big()
	for i, hash := range res.hashes {
		// Mark the range complete if the last is already included.
		// Keep iteration to delete the extra states if exists.
		cmp := hash.Big().Cmp(lastBig)
		if cmp == 0 {
			res.cont = false
			continue
		}
		if cmp > 0 {
			// Chunk overflown, cut off excess
			res.hashes = res.hashes[:i]
			res.accounts = res.accounts[:i]
			res.cont = false // Mark range completed
			break
		}
	}
	// Iterate over all the accounts and assemble which ones need further sub-
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
		if !bytes.Equal(account.CodeHash, types.EmptyCodeHash.Bytes()) {
			if !rawdb.HasCodeWithPrefix(s.db, common.BytesToHash(account.CodeHash)) {
				res.task.codeTasks[common.BytesToHash(account.CodeHash)] = struct{}{}
				res.task.needCode[i] = true
				res.task.pend++
			}
		}
		// Check if the account is a contract with an unknown storage trie
		if account.Root != types.EmptyRootHash {
			// If the storage was already retrieved in the last cycle, there's no need
			// to resync it again, regardless of whether the storage root is consistent
			// or not.
			if _, exist := res.task.stateCompleted[res.hashes[i]]; exist {
				// The leftover storage tasks are not expected, unless system is
				// very wrong.
				if _, ok := res.task.SubTasks[res.hashes[i]]; ok {
					panic(fmt.Errorf("unexpected leftover storage tasks, owner: %x", res.hashes[i]))
				}
				// Mark the healing tag if storage root node is inconsistent, or
				// it's non-existent due to storage chunking.
				if !rawdb.HasTrieNode(s.db, res.hashes[i], nil, account.Root, s.scheme) {
					res.task.needHeal[i] = true
				}
			} else {
				// If there was a previous large state retrieval in progress,
				// don't restart it from scratch. This happens if a sync cycle
				// is interrupted and resumed later. However, *do* update the
				// previous root hash.
				if subtasks, ok := res.task.SubTasks[res.hashes[i]]; ok {
					log.Debug("Resuming large storage retrieval", "account", res.hashes[i], "root", account.Root)
					for _, subtask := range subtasks {
						subtask.root = account.Root
					}
					res.task.needHeal[i] = true
					resumed[res.hashes[i]] = struct{}{}
					largeStorageResumedGauge.Inc(1)
				} else {
					// It's possible that in the hash scheme, the storage, along
					// with the trie nodes of the given root, is already present
					// in the database. Schedule the storage task anyway to simplify
					// the logic here.
					res.task.stateTasks[res.hashes[i]] = account.Root
				}
				res.task.needState[i] = true
				res.task.pend++
			}
		}
	}
	// Delete any subtasks that have been aborted but not resumed. It's essential
	// as the corresponding contract might be self-destructed in this cycle(it's
	// no longer possible in ethereum as self-destruction is disabled in Cancun
	// Fork, but the condition is still necessary for other networks).
	//
	// Keep the leftover storage tasks if they are not covered by the responded
	// account range which should be picked up in next account wave.
	if len(res.hashes) > 0 {
		// The hash of last delivered account in the response
		last := res.hashes[len(res.hashes)-1]
		for hash := range res.task.SubTasks {
			// TODO(rjl493456442) degrade the log level before merging.
			if hash.Cmp(last) > 0 {
				log.Info("Keeping suspended storage retrieval", "account", hash)
				continue
			}
			// TODO(rjl493456442) degrade the log level before merging.
			// It should never happen in ethereum.
			if _, ok := resumed[hash]; !ok {
				log.Error("Aborting suspended storage retrieval", "account", hash)
				delete(res.task.SubTasks, hash)
				largeStorageDiscardGauge.Inc(1)
			}
		}
	}
	// If the account range contained no contracts, or all have been fully filled
	// beforehand, short circuit storage filling and forward to the next task
	if res.task.pend == 0 {
		s.forwardAccountTask(res.task)
		return
	}
	// Some accounts are incomplete, leave as is for the storage and contract
	// task assigners to pick up and fill
}

// processBytecodeResponse integrates an already validated bytecode response
// into the account tasks.
func (s *Syncer) processBytecodeResponse(res *bytecodeResponse) {
	batch := s.db.NewBatch()

	var codes uint64
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
		codes++
		rawdb.WriteCode(batch, hash, code)
	}
	bytes := common.StorageSize(batch.ValueSize())
	if err := batch.Write(); err != nil {
		log.Crit("Failed to persist bytecodes", "err", err)
	}
	s.bytecodeSynced += codes
	s.bytecodeBytes += bytes

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
	// Switch the subtask from pending to idle
	if res.subTask != nil {
		res.subTask.req = nil
	}
	batch := ethdb.HookedBatch{
		Batch: s.db.NewBatch(),
		OnPut: func(key []byte, value []byte) {
			s.storageBytes += common.StorageSize(len(key) + len(value))
		},
	}
	var (
		slots           int
		oldStorageBytes = s.storageBytes
	)
	// Iterate over all the accounts and reconstruct their storage tries from the
	// delivered slots
	for i, account := range res.accounts {
		// If the account was not delivered, reschedule it
		if i >= len(res.hashes) {
			res.mainTask.stateTasks[account] = res.roots[i]
			continue
		}
		// State was delivered, if complete mark as not needed any more, otherwise
		// mark the account as needing healing
		for j, hash := range res.mainTask.res.hashes {
			if account != hash {
				continue
			}
			acc := res.mainTask.res.accounts[j]

			// If the packet contains multiple contract storage slots, all
			// but the last are surely complete. The last contract may be
			// chunked, so check it's continuation flag.
			if res.subTask == nil && res.mainTask.needState[j] && (i < len(res.hashes)-1 || !res.cont) {
				res.mainTask.needState[j] = false
				res.mainTask.pend--
				res.mainTask.stateCompleted[account] = struct{}{} // mark it as completed
				smallStorageGauge.Inc(1)
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
						keys    = res.hashes[i]
						chunks  = uint64(storageConcurrency)
						lastKey common.Hash
					)
					if len(keys) > 0 {
						lastKey = keys[len(keys)-1]
					}
					// If the number of slots remaining is low, decrease the
					// number of chunks. Somewhere on the order of 10-15K slots
					// fit into a packet of 500KB. A key/slot pair is maximum 64
					// bytes, so pessimistically maxRequestSize/64 = 8K.
					//
					// Chunk so that at least 2 packets are needed to fill a task.
					if estimate, err := estimateRemainingSlots(len(keys), lastKey); err == nil {
						if n := estimate / (2 * (maxRequestSize / 64)); n+1 < chunks {
							chunks = n + 1
						}
						log.Debug("Chunked large contract", "initiators", len(keys), "tail", lastKey, "remaining", estimate, "chunks", chunks)
					} else {
						log.Debug("Chunked large contract", "initiators", len(keys), "tail", lastKey, "chunks", chunks)
					}
					r := newHashRange(lastKey, chunks)
					if chunks == 1 {
						smallStorageGauge.Inc(1)
					} else {
						largeStorageGauge.Inc(1)
					}
					// Our first task is the one that was just filled by this response.
					tasks = append(tasks, &storageTask{
						Next: common.Hash{},
						Last: r.End(),
						root: acc.Root,
					})
					for r.Next() {
						tasks = append(tasks, &storageTask{
							Next: r.Start(),
							Last: r.End(),
							root: acc.Root,
						})
					}
					for _, task := range tasks {
						log.Debug("Created storage sync task", "account", account, "root", acc.Root, "from", task.Next, "last", task.Last)
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
				// Find the first overflowing key. While at it, mark res as complete
				// if we find the range to include or pass the 'last'
				index := sort.Search(len(res.hashes[i]), func(k int) bool {
					cmp := res.hashes[i][k].Big().Cmp(last)
					if cmp >= 0 {
						res.cont = false
					}
					return cmp > 0
				})
				if index >= 0 {
					// cut off excess
					res.hashes[i] = res.hashes[i][:index]
					res.slots[i] = res.slots[i][:index]
				}
				// Forward the relevant storage chunk (even if created just now)
				if res.cont {
					res.subTask.Next = incHash(res.hashes[i][len(res.hashes[i])-1])
				} else {
					res.subTask.done = true
				}
			}
		}
		// Iterate over all the complete contracts, reconstruct the trie nodes and
		// push them to disk. If the contract is chunked, the trie nodes will be
		// reconstructed later.
		slots += len(res.hashes[i])

		// Persist the received storage segments. These flat state maybe
		// outdated during the sync, but it can be fixed later during the
		// trie rebuild.
		for j := 0; j < len(res.hashes[i]); j++ {
			rawdb.WriteStorageSnapshot(batch, account, res.hashes[i][j], res.slots[i][j])
		}
	}
	// Flush anything written just now and update the stats
	if err := batch.Write(); err != nil {
		log.Crit("Failed to persist storage slots", "err", err)
	}
	s.storageSynced += uint64(slots)

	log.Debug("Persisted set of storage slots", "accounts", len(res.hashes), "slots", slots, "bytes", s.storageBytes-oldStorageBytes)

	// If this delivery completed the last pending task, forward the account task
	// to the next chunk
	if res.mainTask.pend == 0 {
		s.forwardAccountTask(res.mainTask)
		return
	}
	// Some accounts are still incomplete, leave as is for the storage and contract
	// task assigners to pick up and fill.
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

	// Persist the received account segments. These flat state maybe
	// outdated during the sync, but it can be fixed later during the
	// trie rebuild.
	oldAccountBytes := s.accountBytes

	batch := ethdb.HookedBatch{
		Batch: s.db.NewBatch(),
		OnPut: func(key []byte, value []byte) {
			s.accountBytes += common.StorageSize(len(key) + len(value))
		},
	}
	for i, hash := range res.hashes {
		if task.needCode[i] || task.needState[i] {
			break
		}
		slim := types.SlimAccountRLP(*res.accounts[i])
		rawdb.WriteAccountSnapshot(batch, hash, slim)
	}
	// Flush anything written just now and update the stats
	if err := batch.Write(); err != nil {
		log.Crit("Failed to persist accounts", "err", err)
	}
	s.accountSynced += uint64(len(res.accounts))

	// Task filling persisted, push the chunk marker forward to the first
	// account still missing data.
	for i, hash := range res.hashes {
		if task.needCode[i] || task.needState[i] {
			return
		}
		task.Next = incHash(hash)

		// Remove the completion flag once the account range is pushed
		// forward. The leftover accounts will be skipped in the next
		// cycle.
		delete(task.stateCompleted, hash)
	}
	// All accounts marked as complete, track if the entire task is done
	task.done = !res.cont

	// Error out if there is any leftover completion flag.
	if task.done && len(task.stateCompleted) != 0 {
		panic(fmt.Errorf("storage completion flags should be emptied, %d left", len(task.stateCompleted)))
	}
	log.Debug("Persisted range of accounts", "accounts", len(res.accounts), "bytes", s.accountBytes-oldAccountBytes)
}

// OnAccounts is a callback method to invoke when a range of accounts are
// received from a remote peer.
func (s *Syncer) OnAccounts(peer SyncPeer, id uint64, hashes []common.Hash, accounts [][]byte, proof [][]byte) error {
	size := common.StorageSize(len(hashes) * common.HashLength)
	for _, account := range accounts {
		size += common.StorageSize(len(account))
	}
	for _, node := range proof {
		size += common.StorageSize(len(node))
	}
	logger := peer.Log().New("reqid", id)
	logger.Trace("Delivering range of accounts", "hashes", len(hashes), "accounts", len(accounts), "proofs", len(proof), "bytes", size)

	// Whether or not the response is valid, we can mark the peer as idle and
	// notify the scheduler to assign a new task. If the response is invalid,
	// we'll drop the peer in a bit.
	defer func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		if _, ok := s.peers[peer.ID()]; ok {
			s.accountIdlers[peer.ID()] = struct{}{}
		}
		select {
		case s.update <- struct{}{}:
		default:
		}
	}()
	s.lock.Lock()
	// Ensure the response is for a valid request
	req, ok := s.accountReqs[id]
	if !ok {
		// Request stale, perhaps the peer timed out but came through in the end
		logger.Warn("Unexpected account range packet")
		s.lock.Unlock()
		return nil
	}
	delete(s.accountReqs, id)
	s.rates.Update(peer.ID(), AccountRangeMsg, time.Since(req.time), int(size))

	// Clean up the request timeout timer, we'll see how to proceed further based
	// on the actual delivered content
	if !req.timeout.Stop() {
		// The timeout is already triggered, and this request will be reverted+rescheduled
		s.lock.Unlock()
		return nil
	}
	// Response is valid, but check if peer is signalling that it does not have
	// the requested data. For account range queries that means the state being
	// retrieved was either already pruned remotely, or the peer is not yet
	// synced to our head.
	if len(hashes) == 0 && len(accounts) == 0 && len(proof) == 0 {
		logger.Debug("Peer rejected account range request", "root", s.root)
		s.statelessPeers[peer.ID()] = struct{}{}
		s.lock.Unlock()

		// Signal this request as failed, and ready for rescheduling
		s.scheduleRevertAccountRequest(req)
		return nil
	}
	root := s.root
	s.lock.Unlock()

	// Reconstruct a partial trie from the response and verify it
	keys := make([][]byte, len(hashes))
	for i, key := range hashes {
		keys[i] = common.CopyBytes(key[:])
	}
	nodes := make(trienode.ProofList, len(proof))
	for i, node := range proof {
		nodes[i] = node
	}
	cont, err := trie.VerifyRangeProof(root, req.origin[:], keys, accounts, nodes.Set())
	if err != nil {
		logger.Warn("Account range failed proof", "err", err)
		// Signal this request as failed, and ready for rescheduling
		s.scheduleRevertAccountRequest(req)
		return err
	}
	accs := make([]*types.StateAccount, len(accounts))
	for i, account := range accounts {
		acc := new(types.StateAccount)
		if err := rlp.DecodeBytes(account, acc); err != nil {
			panic(err) // We created these blobs, we must be able to decode them
		}
		accs[i] = acc
	}
	response := &accountResponse{
		task:     req.task,
		hashes:   hashes,
		accounts: accs,
		cont:     cont,
	}
	select {
	case req.deliver <- response:
	case <-req.cancel:
	case <-req.stale:
	}
	return nil
}

// OnByteCodes is a callback method to invoke when a batch of contract
// bytes codes are received from a remote peer.
func (s *Syncer) OnByteCodes(peer SyncPeer, id uint64, bytecodes [][]byte) error {
	return s.onByteCodes(peer, id, bytecodes)
}

// onByteCodes is a callback method to invoke when a batch of contract
// bytes codes are received from a remote peer in the syncing phase.
func (s *Syncer) onByteCodes(peer SyncPeer, id uint64, bytecodes [][]byte) error {
	var size common.StorageSize
	for _, code := range bytecodes {
		size += common.StorageSize(len(code))
	}
	logger := peer.Log().New("reqid", id)
	logger.Trace("Delivering set of bytecodes", "bytecodes", len(bytecodes), "bytes", size)

	// Whether or not the response is valid, we can mark the peer as idle and
	// notify the scheduler to assign a new task. If the response is invalid,
	// we'll drop the peer in a bit.
	defer func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		if _, ok := s.peers[peer.ID()]; ok {
			s.bytecodeIdlers[peer.ID()] = struct{}{}
		}
		select {
		case s.update <- struct{}{}:
		default:
		}
	}()
	s.lock.Lock()
	// Ensure the response is for a valid request
	req, ok := s.bytecodeReqs[id]
	if !ok {
		// Request stale, perhaps the peer timed out but came through in the end
		logger.Warn("Unexpected bytecode packet")
		s.lock.Unlock()
		return nil
	}
	delete(s.bytecodeReqs, id)
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
		logger.Debug("Peer rejected bytecode request")
		s.statelessPeers[peer.ID()] = struct{}{}
		s.lock.Unlock()

		// Signal this request as failed, and ready for rescheduling
		s.scheduleRevertBytecodeRequest(req)
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
		logger.Warn("Unexpected bytecodes", "count", len(bytecodes)-i)
		// Signal this request as failed, and ready for rescheduling
		s.scheduleRevertBytecodeRequest(req)
		return errors.New("unexpected bytecode")
	}
	// Response validated, send it to the scheduler for filling
	response := &bytecodeResponse{
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

// OnStorage is a callback method to invoke when ranges of storage slots
// are received from a remote peer.
func (s *Syncer) OnStorage(peer SyncPeer, id uint64, hashes [][]common.Hash, slots [][][]byte, proof [][]byte) error {
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
	logger := peer.Log().New("reqid", id)
	logger.Trace("Delivering ranges of storage slots", "accounts", len(hashes), "hashes", hashCount, "slots", slotCount, "proofs", len(proof), "size", size)

	// Whether or not the response is valid, we can mark the peer as idle and
	// notify the scheduler to assign a new task. If the response is invalid,
	// we'll drop the peer in a bit.
	defer func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		if _, ok := s.peers[peer.ID()]; ok {
			s.storageIdlers[peer.ID()] = struct{}{}
		}
		select {
		case s.update <- struct{}{}:
		default:
		}
	}()
	s.lock.Lock()
	// Ensure the response is for a valid request
	req, ok := s.storageReqs[id]
	if !ok {
		// Request stale, perhaps the peer timed out but came through in the end
		logger.Warn("Unexpected storage ranges packet")
		s.lock.Unlock()
		return nil
	}
	delete(s.storageReqs, id)
	s.rates.Update(peer.ID(), StorageRangesMsg, time.Since(req.time), int(size))

	// Clean up the request timeout timer, we'll see how to proceed further based
	// on the actual delivered content
	if !req.timeout.Stop() {
		// The timeout is already triggered, and this request will be reverted+rescheduled
		s.lock.Unlock()
		return nil
	}

	// Reject the response if the hash sets and slot sets don't match, or if the
	// peer sent more data than requested.
	if len(hashes) != len(slots) {
		s.lock.Unlock()
		s.scheduleRevertStorageRequest(req) // reschedule request
		logger.Warn("Hash and slot set size mismatch", "hashset", len(hashes), "slotset", len(slots))
		return errors.New("hash and slot set size mismatch")
	}
	if len(hashes) > len(req.accounts) {
		s.lock.Unlock()
		s.scheduleRevertStorageRequest(req) // reschedule request
		logger.Warn("Hash set larger than requested", "hashset", len(hashes), "requested", len(req.accounts))
		return errors.New("hash set larger than requested")
	}
	// Response is valid, but check if peer is signalling that it does not have
	// the requested data. For storage range queries that means the state being
	// retrieved was either already pruned remotely, or the peer is not yet
	// synced to our head.
	if len(hashes) == 0 && len(proof) == 0 {
		logger.Debug("Peer rejected storage request")
		s.statelessPeers[peer.ID()] = struct{}{}
		s.lock.Unlock()
		s.scheduleRevertStorageRequest(req) // reschedule request
		return nil
	}
	s.lock.Unlock()

	// Reconstruct the partial tries from the response and verify them
	var cont bool

	// If a proof was attached while the response is empty, it indicates that the
	// requested range specified with 'origin' is empty. Construct an empty state
	// response locally to finalize the range.
	if len(hashes) == 0 && len(proof) > 0 {
		hashes = append(hashes, []common.Hash{})
		slots = append(slots, [][]byte{})
	}
	for i := 0; i < len(hashes); i++ {
		// Convert the keys and proofs into an internal format
		keys := make([][]byte, len(hashes[i]))
		for j, key := range hashes[i] {
			keys[j] = common.CopyBytes(key[:])
		}
		nodes := make(trienode.ProofList, 0, len(proof))
		if i == len(hashes)-1 {
			for _, node := range proof {
				nodes = append(nodes, node)
			}
		}
		var err error
		if len(nodes) == 0 {
			// No proof has been attached, the response must cover the entire key
			// space and hash to the origin root.
			_, err = trie.VerifyRangeProof(req.roots[i], nil, keys, slots[i], nil)
			if err != nil {
				s.scheduleRevertStorageRequest(req) // reschedule request
				logger.Warn("Storage slots failed proof", "err", err)
				return err
			}
		} else {
			// A proof was attached, the response is only partial, check that the
			// returned data is indeed part of the storage trie
			proofdb := nodes.Set()

			cont, err = trie.VerifyRangeProof(req.roots[i], req.origin[:], keys, slots[i], proofdb)
			if err != nil {
				s.scheduleRevertStorageRequest(req) // reschedule request
				logger.Warn("Storage range failed proof", "err", err)
				return err
			}
		}
	}
	// Partial tries reconstructed, send them to the scheduler for storage filling
	response := &storageResponse{
		mainTask: req.mainTask,
		subTask:  req.subTask,
		accounts: req.accounts,
		roots:    req.roots,
		hashes:   hashes,
		slots:    slots,
		cont:     cont,
	}
	select {
	case req.deliver <- response:
	case <-req.cancel:
	case <-req.stale:
	}
	return nil
}

// OnAccessLists is a callback method to invoke when a batch of access lists
// are received from a remote peer.
func (s *Syncer) OnAccessLists(peer SyncPeer, id uint64, accessLists rlp.RawList[rlp.RawValue]) error {
	// Convert RawList to slice of raw values
	bals, err := accessLists.Items()
	if err != nil {
		return err
	}

	// Calculate total size of returned data
	var size common.StorageSize
	for _, bal := range bals {
		size += common.StorageSize(len(bal))
	}
	logger := peer.Log().New("reqid", id)
	logger.Trace("Delivering set of BALs", "count", len(bals), "bytes", size)

	// Whether or not the response is valid, we can mark the peer as idle and
	// notify the scheduler to assign a new task. If the response is invalid,
	// we'll drop the peer in a bit.
	defer func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		if _, ok := s.peers[peer.ID()]; ok {
			s.accessListIdlers[peer.ID()] = struct{}{}
		}
		select {
		case s.update <- struct{}{}:
		default:
		}
	}()
	s.lock.Lock()

	// Ensure the response is for a valid request
	req, ok := s.accessListReqs[id]
	if !ok {
		// Request stale, perhaps the peer timed out but came through in the end
		logger.Warn("Unexpected access list packet")
		s.lock.Unlock()
		return nil
	}
	delete(s.accessListReqs, id)
	s.rates.Update(peer.ID(), AccessListsMsg, time.Since(req.time), len(bals))

	// Clean up the request timeout timer
	if !req.timeout.Stop() {
		// The timeout is already triggered, and this request will be reverted+rescheduled
		s.lock.Unlock()
		return nil
	}

	// Response is valid, but check if peer is signalling that it does not have
	// the requested data.
	if len(bals) == 0 {
		logger.Debug("Peer rejected access list request")
		s.statelessPeers[peer.ID()] = struct{}{}
		s.lock.Unlock()

		// Signal this request as failed, and ready for rescheduling
		s.scheduleRevertAccessListRequest(req)
		return nil
	}
	s.lock.Unlock()

	// Response validated, send it to the scheduler for filling.
	response := &accessListResponse{
		req:         req,
		accessLists: bals,
	}
	select {
	case req.deliver <- response:
	case <-req.cancel:
	case <-req.stale:
	}
	return nil
}

// hashSpace is the total size of the 256 bit hash space for accounts.
var hashSpace = new(big.Int).Exp(common.Big2, common.Big256, nil)

// report calculates various status reports and provides it to the user.
func (s *Syncer) report(force bool) {
	s.reportSyncProgress(force)
}

// reportSyncProgress calculates various status reports and provides it to the user.
func (s *Syncer) reportSyncProgress(force bool) {
	// Don't report all the events, just occasionally
	if !force && time.Since(s.logTime) < 8*time.Second {
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
	// Don't report anything until we have a meaningful progress
	if estBytes < 1.0 {
		return
	}
	// Cap the estimated state size using the synced size to avoid negative values
	if estBytes < float64(synced) {
		estBytes = float64(synced)
	}
	elapsed := time.Since(s.startTime)
	estTime := elapsed / time.Duration(synced) * time.Duration(estBytes)

	// Create a mega progress report
	var (
		progress = fmt.Sprintf("%.2f%%", float64(synced)*100/estBytes)
		accounts = fmt.Sprintf("%v@%v", log.FormatLogfmtUint64(s.accountSynced), s.accountBytes.TerminalString())
		storage  = fmt.Sprintf("%v@%v", log.FormatLogfmtUint64(s.storageSynced), s.storageBytes.TerminalString())
		bytecode = fmt.Sprintf("%v@%v", log.FormatLogfmtUint64(s.bytecodeSynced), s.bytecodeBytes.TerminalString())
	)
	log.Info("Syncing: state download in progress", "synced", progress, "state", synced,
		"accounts", accounts, "slots", storage, "codes", bytecode, "eta", common.PrettyDuration(estTime-elapsed))
}

// estimateRemainingSlots tries to determine roughly how many slots are left in
// a contract storage, based on the number of keys and the last hash. This method
// assumes that the hashes are lexicographically ordered and evenly distributed.
func estimateRemainingSlots(hashes int, last common.Hash) (uint64, error) {
	if last == (common.Hash{}) {
		return 0, errors.New("last hash empty")
	}
	space := new(big.Int).Mul(math.MaxBig256, big.NewInt(int64(hashes)))
	space.Div(space, last.Big())
	if !space.IsUint64() {
		// Gigantic address space probably due to too few or malicious slots
		return 0, errors.New("too few slots for estimation")
	}
	return space.Uint64() - uint64(hashes), nil
}

// sortIdlePeers builds a list of idle peers sorted by download capacity
// (highest first), filtering out stateless peers. Must be called with s.lock held.
func (s *Syncer) sortIdlePeers(idlerSet map[string]struct{}, msgCode uint64) *capacitySort {
	idlers := &capacitySort{
		ids:  make([]string, 0, len(idlerSet)),
		caps: make([]int, 0, len(idlerSet)),
	}
	targetTTL := s.rates.TargetTimeout()
	for id := range idlerSet {
		if _, ok := s.statelessPeers[id]; ok {
			continue
		}
		idlers.ids = append(idlers.ids, id)
		idlers.caps = append(idlers.caps, s.rates.Capacity(id, msgCode, targetTTL))
	}
	if len(idlers.ids) == 0 {
		return idlers
	}
	sort.Sort(sort.Reverse(idlers))
	return idlers
}

// capacitySort implements the Sort interface, allowing sorting by peer message
// throughput. Note, callers should use sort.Reverse to get the desired effect
// of highest capacity being at the front.
type capacitySort struct {
	ids  []string
	caps []int
}

func (s *capacitySort) Len() int {
	return len(s.ids)
}

func (s *capacitySort) Less(i, j int) bool {
	return s.caps[i] < s.caps[j]
}

func (s *capacitySort) Swap(i, j int) {
	s.ids[i], s.ids[j] = s.ids[j], s.ids[i]
	s.caps[i], s.caps[j] = s.caps[j], s.caps[i]
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

// sortByAccountPath takes hashes and paths, and sorts them. After that, it generates
// the TrieNodePaths and merges paths which belongs to the same account path.
func sortByAccountPath(paths []string, hashes []common.Hash) ([]string, []common.Hash, []trie.SyncPath, []TrieNodePathSet) {
	syncPaths := make([]trie.SyncPath, len(paths))
	for i, path := range paths {
		syncPaths[i] = trie.NewSyncPath([]byte(path))
	}
	n := &healRequestSort{paths, hashes, syncPaths}
	sort.Sort(n)
	pathsets := n.Merge()
	return n.paths, n.hashes, n.syncPaths, pathsets
}
