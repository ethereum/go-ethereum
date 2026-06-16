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
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/msgrate"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
)

const (
	// maxAccessListRequestCount is the maximum number of block BALs to
	// request in a single query. BALs average ~72 KiB compressed (per EIP-7928),
	// and EIP-8189 recommends a 2 MiB response soft limit, so we target ~28
	// blocks per request to avoid server-side truncation.
	//
	// NOTE: If the gas limit is raised significantly, this number may need to be adjusted
	// to avoid server-side truncation and re-requesting. It is currently based on
	// the assumption that the gas limit is 60M.
	maxAccessListRequestCount = 28

	// maxCatchUpBlocks is the maximum gap (in blocks) that BAL catch-up is
	// allowed to span. BALs are only retained by peers for a limited window
	// (roughly two weeks, ~100k blocks at 12s block time). If the pivot has
	// moved further than this conservative bound, the BALs needed to roll the
	// flat state forward are likely no longer available, so we discard the
	// stale progress and restart the sync from scratch rather than attempting
	// a catch-up that is bound to fail partway through.
	maxCatchUpBlocks = params.FullImmutabilityThreshold

	// catchUpWindow is the number of blocks BAL catch-up fetches and applies at
	// a time. The whole gap can span up to maxCatchUpBlocks, so fetching it in
	// one shot would buffer every block's BAL (~100 KiB each) in memory before
	// applying any. Processing the gap in bounded windows caps peak memory to a
	// single window's worth of BALs.
	catchUpWindow = 512

	// syncProgressVersion is the version byte prepended to the JSON-encoded
	// syncProgressV2 when persisted. On load, a mismatching version byte causes
	// the persisted progress to be discarded and sync to start fresh.
	syncProgressVersion byte = 2
)

// minRequestSize, maxRequestSize, maxCodeRequestCount, accountConcurrency and
// storageConcurrency are shared with the snap/1 syncer; see sync.go.

// errAccessListPeersExhausted is returned from fetchAccessLists when every
// connected peer has been marked stateless for BAL requests and there
// are still hashes left to fetch.
var errAccessListPeersExhausted = errors.New("all peers exhausted for BAL requests")

// errAccessListUnavailable is returned from the BAL catch-up when some gap
// block's access list cannot be retrieved against the current peerset.
var errAccessListUnavailable = errors.New("block access lists unavailable")

// accountRequestV2 tracks a pending account range request to ensure responses are
// to actual requests and to validate any security constraints.
//
// Concurrency note: account requests and responses are handled concurrently from
// the main runloop to allow Merkle proof verifications on the peer's thread and
// to drop on invalid response. The request struct must contain all the data to
// construct the response without accessing runloop internals (i.e. task). That
// is only included to allow the runloop to match a response to the task being
// synced without having yet another set of maps.
type accountRequestV2 struct {
	peer string    // Peer to which this request is assigned
	id   uint64    // Request ID of this request
	time time.Time // Timestamp when the request was sent

	deliver chan *accountResponseV2 // Channel to deliver successful response on
	revert  chan *accountRequestV2  // Channel to deliver request failure on
	cancel  chan struct{}           // Channel to track sync cancellation
	timeout *time.Timer             // Timer to track delivery timeout
	stale   chan struct{}           // Channel to signal the request was dropped

	origin common.Hash // First account requested to allow continuation checks
	limit  common.Hash // Last account requested to allow non-overlapping chunking

	task *accountTaskV2 // Task which this request is filling (only access fields through the runloop!!)
}

// accountResponseV2 is an already Merkle-verified remote response to an account
// range request.
type accountResponseV2 struct {
	task *accountTaskV2 // Task which this request is filling

	hashes   []common.Hash         // Account hashes in the returned range
	accounts []*types.StateAccount // Expanded accounts in the returned range

	cont bool // Whether the account range has a continuation
}

// bytecodeRequestV2 tracks a pending bytecode request to ensure responses are to
// actual requests and to validate any security constraints.
//
// Concurrency note: bytecode requests and responses are handled concurrently from
// the main runloop to allow Keccak256 hash verifications on the peer's thread and
// to drop on invalid response. The request struct must contain all the data to
// construct the response without accessing runloop internals (i.e. task). That
// is only included to allow the runloop to match a response to the task being
// synced without having yet another set of maps.
type bytecodeRequestV2 struct {
	peer string    // Peer to which this request is assigned
	id   uint64    // Request ID of this request
	time time.Time // Timestamp when the request was sent

	deliver chan *bytecodeResponseV2 // Channel to deliver successful response on
	revert  chan *bytecodeRequestV2  // Channel to deliver request failure on
	cancel  chan struct{}            // Channel to track sync cancellation
	timeout *time.Timer              // Timer to track delivery timeout
	stale   chan struct{}            // Channel to signal the request was dropped

	hashes []common.Hash  // Bytecode hashes to validate responses
	task   *accountTaskV2 // Task which this request is filling (only access fields through the runloop!!)
}

// bytecodeResponseV2 is an already verified remote response to a bytecode request.
type bytecodeResponseV2 struct {
	task *accountTaskV2 // Task which this request is filling

	hashes []common.Hash // Hashes of the bytecode to avoid double hashing
	codes  [][]byte      // Actual bytecodes to store into the database (nil = missing)
}

// storageRequestV2 tracks a pending storage ranges request to ensure responses are
// to actual requests and to validate any security constraints.
//
// Concurrency note: storage requests and responses are handled concurrently from
// the main runloop to allow Merkle proof verifications on the peer's thread and
// to drop on invalid response. The request struct must contain all the data to
// construct the response without accessing runloop internals (i.e. tasks). That
// is only included to allow the runloop to match a response to the task being
// synced without having yet another set of maps.
type storageRequestV2 struct {
	peer string    // Peer to which this request is assigned
	id   uint64    // Request ID of this request
	time time.Time // Timestamp when the request was sent

	deliver chan *storageResponseV2 // Channel to deliver successful response on
	revert  chan *storageRequestV2  // Channel to deliver request failure on
	cancel  chan struct{}           // Channel to track sync cancellation
	timeout *time.Timer             // Timer to track delivery timeout
	stale   chan struct{}           // Channel to signal the request was dropped

	accounts []common.Hash // Account hashes to validate responses
	roots    []common.Hash // Storage roots to validate responses

	origin common.Hash // First storage slot requested to allow continuation checks
	limit  common.Hash // Last storage slot requested to allow non-overlapping chunking

	mainTask *accountTaskV2 // Task which this response belongs to (only access fields through the runloop!!)
	subTask  *storageTaskV2 // Task which this response is filling (only access fields through the runloop!!)
}

// storageResponseV2 is an already Merkle-verified remote response to a storage
// range request.
type storageResponseV2 struct {
	mainTask *accountTaskV2 // Task which this response belongs to
	subTask  *storageTaskV2 // Task which this response is filling

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

// accountTaskV2 represents the sync task for a chunk of the account snapshot.
type accountTaskV2 struct {
	// These fields get serialized to key-value store on shutdown
	Next     common.Hash                      // Next account to sync in this interval
	Last     common.Hash                      // Last account to sync in this interval
	SubTasks map[common.Hash][]*storageTaskV2 // Storage intervals needing fetching for large contracts

	// Pending accounts whose storage has already been fully committed in
	// this cycle, but which cannot advance Next yet because account commits
	// must be sequential. Persisting them across cycle switches avoids
	// refetching their storage.
	StorageCompleted []common.Hash

	// These fields are internals used during runtime
	req  *accountRequestV2  // Pending request to fill this task
	res  *accountResponseV2 // Validate response filling this task
	pend int                // Number of pending subtasks for this round

	needCode  []bool // Flags whether the filling accounts need code retrieval
	needState []bool // Flags whether the filling accounts need storage retrieval

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
func (task *accountTaskV2) activeSubTasks() map[common.Hash][]*storageTaskV2 {
	if len(task.res.hashes) == 0 {
		return nil
	}
	var (
		tasks = make(map[common.Hash][]*storageTaskV2)
		last  = task.res.hashes[len(task.res.hashes)-1]
	)
	for hash, subTasks := range task.SubTasks {
		if hash.Cmp(last) <= 0 {
			tasks[hash] = subTasks
		}
	}
	return tasks
}

// storageTaskV2 represents the sync task for a chunk of the storage snapshot.
type storageTaskV2 struct {
	Next common.Hash // Next account to sync in this interval
	Last common.Hash // Last account to sync in this interval

	// These fields are internals used during runtime
	root common.Hash       // Storage root hash for this instance
	req  *storageRequestV2 // Pending request to fill this task
	done bool              // Flag whether the task can be removed
}

// syncPhase tracks how far a snap/2 sync has progressed for the journaled
// pivot. The phases are strictly ordered: each one implies all previous
// ones have finished.
type syncPhase uint8

const (
	// phaseDownload covers the flat state (account, storage, bytecode)
	// download. The requests target the pivot root, which remote peers
	// only serve while it is recent, so the pivot must keep tracking the
	// chain head (see FrozenPivot).
	phaseDownload syncPhase = iota

	// phaseGenerate covers the local trie generation after the download
	// has completed. It targets the exact pivot root it was started with,
	// so pivot updates are refused from here on.
	phaseGenerate

	// phaseComplete means the sync ran to completion for the pivot.
	phaseComplete
)

// syncProgressV2 is a database entry to allow suspending and resuming a snapshot state
// sync. Opposed to full and fast sync, there is no way to restart a suspended
// snap sync without prior knowledge of the suspension point.
type syncProgressV2 struct {
	Pivot *types.Header    // Pivot header being synced (for pivot move and reorg detection)
	Tasks []*accountTaskV2 // The suspended account tasks (contract tasks within)
	Phase syncPhase        // Phase is how far the sync has progressed for Pivot

	// Status report during syncing phase
	AccountSynced  uint64             // Number of accounts downloaded
	AccountBytes   common.StorageSize // Number of account trie bytes persisted to disk
	BytecodeSynced uint64             // Number of bytecodes downloaded
	BytecodeBytes  common.StorageSize // Number of bytecode bytes downloaded
	StorageSynced  uint64             // Number of storage slots downloaded
	StorageBytes   common.StorageSize // Number of storage trie bytes persisted to disk
}

// SyncPeerV2 abstracts out the methods required for a peer to be synced against
// with the goal of allowing the construction of mock peers without the full
// blown networking.
type SyncPeerV2 interface {
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

	// RequestTrieNodes fetches a batch of account or storage trie nodes rooted in
	// a specific state trie. snap/2 never issues these requests itself, but the
	// method is retained so a single peer type can serve both the snap/1 and
	// snap/2 syncers (e.g. via the downloader's syncer abstraction).
	RequestTrieNodes(id uint64, root common.Hash, count int, paths []TrieNodePathSet, bytes int) error

	// RequestAccessLists fetches a batch of BALs by block hash.
	RequestAccessLists(id uint64, hashes []common.Hash, bytes int) error

	// Log retrieves the peer's own contextual logger.
	Log() log.Logger
}

// syncerV2 is an Ethereum account and storage trie syncer based on the snap
// protocol. It downloads all accounts, storage slots, and bytecodes from
// remote peers as flat state, applies BAL diffs on pivot moves,
// and triggers a final trie generation once flat state is consistent.
//
// Every network request has a variety of failure events:
//   - The peer disconnects after task assignment, failing to send the request
//   - The peer disconnects after sending the request, before delivering on it
//   - The peer remains connected, but does not deliver a response in time
//   - The peer delivers a stale response after a previous timeout
//   - The peer delivers a refusal to serve the requested state
type syncerV2 struct {
	db     ethdb.Database   // Database to store the trie nodes into (and dedup)
	scheme string           // Node scheme used in node database
	pivot  *types.Header    // Current pivot header being synced (lock needed)
	phase  atomic.Uint32    // Current syncPhase; atomic so phase transitions are visible across goroutines
	tasks  []*accountTaskV2 // Current account task set being synced
	update chan struct{}    // Notification channel for possible sync progression

	peers    map[string]SyncPeerV2 // Currently active peers to download from
	peerJoin *event.Feed           // Event feed to react to peers joining
	peerDrop *event.Feed           // Event feed to react to peers dropping
	rates    *msgrate.Trackers     // Message throughput rates for peers

	// Request tracking during syncing phase.
	//
	// These fields should be protected by lock.
	statelessPeers   map[string]struct{} // Peers that failed to deliver state data
	accountIdlers    map[string]struct{} // Peers that aren't serving account requests
	bytecodeIdlers   map[string]struct{} // Peers that aren't serving bytecode requests
	storageIdlers    map[string]struct{} // Peers that aren't serving storage requests
	accessListIdlers map[string]struct{} // Peers that aren't serving BAL requests

	// These fields should be protected by lock.
	accountReqs    map[uint64]*accountRequestV2  // Account requests currently running
	bytecodeReqs   map[uint64]*bytecodeRequestV2 // Bytecode requests currently running
	storageReqs    map[uint64]*storageRequestV2  // Storage requests currently running
	accessListReqs map[uint64]*accessListRequest // BAL requests currently running

	accountSynced  uint64             // Number of accounts downloaded
	accountBytes   common.StorageSize // Number of account trie bytes persisted to disk
	bytecodeSynced uint64             // Number of bytecodes downloaded
	bytecodeBytes  common.StorageSize // Number of bytecode bytes downloaded
	storageSynced  uint64             // Number of storage slots downloaded
	storageBytes   common.StorageSize // Number of storage trie bytes persisted to disk

	extProgress *syncProgressV2 // progress that can be exposed to external caller.

	startTime time.Time // Time instance when snapshot sync started
	logTime   time.Time // Time instance when status was last reported

	catchUpWindow uint64 // Number of blocks fetched/applied per BAL catch-up window (overridable in tests)

	pend sync.WaitGroup // Tracks network request goroutines for graceful shutdown
	lock sync.RWMutex   // Protects fields that can change outside of sync (peers, reqs, pivot)
}

// newSyncerV2 creates a new snapshot syncer to download the Ethereum state over the
// snap protocol.
func newSyncerV2(db ethdb.Database, scheme string) *syncerV2 {
	s := &syncerV2{
		db:     db,
		scheme: scheme,

		peers:    make(map[string]SyncPeerV2),
		peerJoin: new(event.Feed),
		peerDrop: new(event.Feed),
		rates:    msgrate.NewTrackers(log.New("proto", "snap")),
		update:   make(chan struct{}, 1),

		statelessPeers:   make(map[string]struct{}),
		accountIdlers:    make(map[string]struct{}),
		storageIdlers:    make(map[string]struct{}),
		bytecodeIdlers:   make(map[string]struct{}),
		accessListIdlers: make(map[string]struct{}),

		accountReqs:    make(map[uint64]*accountRequestV2),
		storageReqs:    make(map[uint64]*storageRequestV2),
		bytecodeReqs:   make(map[uint64]*bytecodeRequestV2),
		accessListReqs: make(map[uint64]*accessListRequest),

		extProgress:   new(syncProgressV2),
		catchUpWindow: catchUpWindow,
	}
	if raw := rawdb.ReadSnapshotSyncStatus(db); len(raw) > 0 && raw[0] == syncProgressVersion {
		var progress syncProgressV2
		if err := json.Unmarshal(raw[1:], &progress); err == nil {
			s.pivot = progress.Pivot
			s.setPhase(progress.Phase)
		}
	}
	return s
}

// getPhase returns the current sync phase.
func (s *syncerV2) getPhase() syncPhase {
	return syncPhase(s.phase.Load())
}

// setPhase moves the sync to the given phase.
func (s *syncerV2) setPhase(phase syncPhase) {
	s.phase.Store(uint32(phase))
}

// Register injects a new data source into the syncer's peerset.
func (s *syncerV2) Register(peer SyncPeerV2) error {
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
func (s *syncerV2) Unregister(id string) error {
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

// Sync starts (or resumes a previous) sync cycle to iterate over a state trie
// with the given pivot header and reconstruct the nodes based on the snapshot
// leaves.
func (s *syncerV2) Sync(target *types.Header, cancel chan struct{}) error {
	if target == nil {
		return errors.New("snap sync: pivot header is nil")
	}
	s.lock.Lock()
	s.statelessPeers = make(map[string]struct{})
	s.lock.Unlock()
	if s.startTime.IsZero() {
		s.startTime = time.Now()
	}
	root := target.Root

	// Retrieve the previous sync status from DB. If there's no persisted
	// status, sync is either fresh or already complete.
	s.loadSyncStatus()

	// isPivotChanged is true when we have prior progress against a different
	// pivot. That means we need to roll forward via catchUp, or wipe and
	// restart if the prior pivot was reorged out.
	s.lock.RLock()
	prevPivot := s.pivot
	s.lock.RUnlock()
	isPivotChanged := prevPivot != nil && prevPivot.Hash() != target.Hash()

	// Skip if we've already finished syncing this pivot.
	if !isPivotChanged && s.getPhase() == phaseComplete {
		log.Info("Snap sync already complete for this pivot", "root", root)
		return nil
	}

	// We're committing to running this sync. Demote a completed phase so a
	// mid-run save (on cancel or error) doesn't persist a stale complete
	// status from a prior pivot. The download remains done, only the trie
	// generation must be redone against the new pivot.
	if s.getPhase() == phaseComplete {
		s.setPhase(phaseGenerate)
	}

	defer func() {
		// Whether sync completed or not, disregard any future packets
		log.Debug("Terminating snapshot sync cycle", "root", root)
		s.lock.Lock()
		s.accountReqs = make(map[uint64]*accountRequestV2)
		s.storageReqs = make(map[uint64]*storageRequestV2)
		s.bytecodeReqs = make(map[uint64]*bytecodeRequestV2)
		s.accessListReqs = make(map[uint64]*accessListRequest)
		s.lock.Unlock()

		// Persist final task state.
		for _, task := range s.tasks {
			s.forwardAccountTask(task)
		}
		s.cleanAccountTasks()
		s.saveSyncStatus()

		// Log final progress.
		s.report(true)
	}()

	log.Debug("Starting snapshot sync cycle", "root", root)

	// If we resumed against a different pivot, decide whether the persisted
	// progress is still usable. If yes, roll forward via BAL catch-up. If not,
	// wipe everything and restart fresh.
	if isPivotChanged {
		switch {
		case isPivotReorged(s.db, prevPivot, target):
			log.Warn("Restarting snap sync from scratch", "oldnumber", prevPivot.Number, "oldHash", prevPivot.Hash())
			s.resetSyncState()
		case catchUpExceedsRetention(prevPivot, target):
			// The pivot moved further than the BAL retention window. The access
			// lists required for catch-up are almost certainly unavailable from
			// peers, so discard the stale progress and resync from scratch
			// instead of starting a catch-up doomed to stall.
			log.Warn("Catch-up gap exceeds BAL retention, restarting snap sync from scratch", "oldnumber", prevPivot.Number, "newnumber", target.Number, "gap", new(big.Int).Sub(target.Number, prevPivot.Number), "limit", maxCatchUpBlocks)
			s.resetSyncState()
		default:
			// A canonical pivot move past a frozen pivot should be impossible:
			// the downloader both refuses moves (FrozenPivot) and resumes new
			// cycles against the frozen header itself. Reaching this branch
			// frozen indicates a bug on the downloader side; roll the flat
			// state forward defensively and regenerate.
			if s.getPhase() >= phaseGenerate {
				log.Warn("Frozen pivot moved unexpectedly, rolling forward", "frozen", prevPivot.Number, "new", target.Number)
			}
			if err := s.catchUp(target, cancel); err != nil {
				return err
			}
		}
	}
	s.lock.Lock()
	s.pivot = target
	s.lock.Unlock()

	log.Info("Starting state download", "root", root)
	if err := s.downloadState(cancel); err != nil {
		return err
	}
	log.Info("State download complete", "root", root)

	// Entering the generation phase makes the downloader stop moving the
	// pivot (see FrozenPivot) until the pivot block is committed. The phase
	// is persisted right away so the freeze also holds across a restart,
	// before the generation has had a chance to finish.
	if s.getPhase() < phaseGenerate {
		s.setPhase(phaseGenerate)
		s.saveSyncStatus()
	}

	log.Info("Starting trie generation", "root", root)
	batch := s.db.NewBatch()
	s.resetTrienodes(batch)
	if err := batch.Write(); err != nil {
		return err
	}
	if _, err := triedb.GenerateTrie(s.db, s.scheme, root, cancel); err != nil {
		return err
	}
	log.Info("Trie generation complete", "root", root)

	// Mark sync complete. The deferred saveSyncStatus persists this so a
	// follow-up Sync call for the same pivot can skip the work entirely.
	s.setPhase(phaseComplete)
	return nil
}

// FrozenPivot returns the pivot header the sync is bound to, or nil while
// the pivot may still move freely. The pivot freezes once the state
// download completes. The remaining work (trie generation) and the pivot
// commit is purely local and targets the exact pivot root the download
// finished with, so from that point on the downloader must neither move the
// pivot nor start a new cycle against a different one.
func (s *syncerV2) FrozenPivot() *types.Header {
	if s.getPhase() < phaseGenerate {
		return nil
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.pivot
}

// download runs the bulk flat-state download. It fetches
// account ranges, storage slots, and bytecodes, writing flat state to disk.
func (s *syncerV2) downloadState(cancel chan struct{}) error {
	// Subscribe to peer events
	peerJoin := make(chan string, 16)
	peerJoinSub := s.peerJoin.Subscribe(peerJoin)
	defer peerJoinSub.Unsubscribe()
	peerDrop := make(chan string, 16)
	peerDropSub := s.peerDrop.Subscribe(peerDrop)
	defer peerDropSub.Unsubscribe()

	// Create ephemeral channels for this download cycle
	var (
		accountReqFails  = make(chan *accountRequestV2)
		storageReqFails  = make(chan *storageRequestV2)
		bytecodeReqFails = make(chan *bytecodeRequestV2)
		accountResps     = make(chan *accountResponseV2)
		storageResps     = make(chan *storageResponseV2)
		bytecodeResps    = make(chan *bytecodeResponseV2)
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
		s.refreshProgressLocked()
		s.lock.Unlock()

		// Wait for something to happen
		select {
		case <-s.update:
			// Something happened (new peer, delivery, timeout), recheck tasks
		case <-peerJoin:
			// A new peer joined, try to schedule it new tasks
		case id := <-peerDrop:
			s.revertStateRequests(id)
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

// isPivotReorged reports whether the previous pivot is no longer usable
// as a starting point for forward catch-up. Either it was reorged out
// of the canonical chain, or the new pivot doesn't advance past it.
func isPivotReorged(db ethdb.Database, prev, curr *types.Header) bool {
	// If the new pivot is at or below the old one, there's nothing for
	// catchUp to roll forward.
	if curr.Number.Cmp(prev.Number) <= 0 {
		return true
	}
	// If there's no canonical hash at the old pivot's height, something
	// is wrong. Headers up to the new pivot should already be indexed,
	// so a missing entry at an earlier block means the chain state is
	// broken. The most common cause is a chain rewind across the
	// snap-synced pivot, which resets head to genesis and deletes
	// canonical entries above it (see rewindPathHead in core/blockchain.go).
	// Bail and let the fresh sync recover.
	canonical := rawdb.ReadCanonicalHash(db, prev.Number.Uint64())
	if canonical == (common.Hash{}) {
		return true
	}

	// If canonical at the old pivot's height has a different hash, the
	// old pivot was reorged out.
	return canonical != prev.Hash()
}

// catchUpExceedsRetention reports whether rolling the flat state forward from
// prev to curr would span more blocks than peers are expected to retain BALs
// for. Beyond this bound the access lists needed for catch-up are likely gone,
// so the caller should wipe and resync from scratch instead.
func catchUpExceedsRetention(prev, curr *types.Header) bool {
	gap := new(big.Int).Sub(curr.Number, prev.Number)
	return gap.Cmp(big.NewInt(maxCatchUpBlocks)) > 0
}

// catchUp runs the BAL catch-up. When the pivot has moved, it fetches BALs
// for the gap blocks, verifies them against block headers, and applies the
// diffs to roll flat state forward.
func (s *syncerV2) catchUp(target *types.Header, cancel chan struct{}) error {
	s.lock.RLock()
	from := s.pivot.Number.Uint64() + 1
	to := target.Number.Uint64()
	s.lock.RUnlock()
	log.Info("Starting BAL catch-up", "from", from, "to", to, "blocks", to-from+1)

	for start := from; start <= to; start += s.catchUpWindow {
		select {
		case <-cancel:
			return ErrCancelled
		default:
		}
		end := start + s.catchUpWindow - 1
		if end > to {
			end = to
		}
		// Collect block hashes and headers for this window.
		var (
			hashes  = make([]common.Hash, 0, end-start+1)
			headers = make(map[common.Hash]*types.Header, end-start+1)
		)
		for num := start; num <= end; num++ {
			hash := rawdb.ReadCanonicalHash(s.db, num)
			if hash == (common.Hash{}) {
				return fmt.Errorf("missing canonical hash for block %d during catch-up", num)
			}
			header := rawdb.ReadHeader(s.db, hash, num)
			if header == nil {
				return fmt.Errorf("missing header for block %d (hash %v) during catch-up", num, hash)
			}
			hashes = append(hashes, hash)
			headers[hash] = header
		}

		// Fetch this window's BALs from peers.
		rawBALs, err := s.fetchAccessLists(hashes, headers, cancel)
		if err != nil {
			return err
		}

		// Apply each BAL in block order. BALs are already verified by fetchAccessLists.
		for i, raw := range rawBALs {
			select {
			case <-cancel:
				return ErrCancelled
			default:
			}
			num := start + uint64(i)
			hash := hashes[i]

			// Decode the raw RLP into a BAL.
			var (
				b     bal.BlockAccessList
				batch = s.db.NewBatch()
			)
			if err := rlp.DecodeBytes(raw, &b); err != nil {
				return fmt.Errorf("failed to decode BAL for block %d: %v", num, err)
			}
			if err := s.applyAccessList(&b, batch); err != nil {
				return fmt.Errorf("BAL application failed for block %d: %v", num, err)
			}

			// Persist incremental progress so a crash mid-catchUp can resume
			// from the next unapplied block.
			s.lock.Lock()
			s.pivot = headers[hash]
			s.lock.Unlock()
			s.saveSyncStatusWithDB(batch)

			// Commit the state transition alongside the sync progress atomically.
			if err := batch.Write(); err != nil {
				return err
			}
		}
		log.Info("BAL catch-up progress", "applied", end, "target", to, "remaining", to-end)
	}
	log.Info("BAL catch-up complete", "from", from, "to", to)
	return nil
}

// fetchAccessLists fetches BALs for the given block hashes from
// remote peers. It runs its own event loop to assign requests
// to idle peers and process responses asynchronously. Each BAL is verified
// against its header before being accepted. Results are returned in the
// same order as the input hashes.
func (s *syncerV2) fetchAccessLists(hashes []common.Hash, headers map[common.Hash]*types.Header, cancel chan struct{}) ([]rlp.RawValue, error) {
	log.Debug("Fetching BALs for catch-up", "blocks", len(hashes))

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

	// refused tracks the mapping between BAL and peerset which doesn't have
	// it available.
	refused := make(map[common.Hash]map[string]struct{})

	var (
		accessListReqFails = make(chan *accessListRequest)
		accessListResps    = make(chan *accessListResponse)
		lastStallLog       = time.Now()
	)
	for len(fetched) < len(hashes) {
		if err := s.checkAccessListProgress(pending, refused); err != nil {
			log.Warn("BAL fetch cannot progress", "err", err, "fetched", len(fetched), "remaining", len(pending))
			return nil, err
		}
		// Assign BAL retrieval tasks to idle peers
		s.assignAccessListTasks(pending, refused, accessListResps, accessListReqFails, cancel)

		// Periodic visibility while stalled with peers connected but idle.
		if len(pending) > 0 && time.Since(lastStallLog) > 30*time.Second {
			lastStallLog = time.Now()
			log.Warn("BAL catch-up stalled, awaiting peers", "fetched", len(fetched), "remaining", len(pending))
		}

		// Wait for something to happen
		select {
		case <-s.update:
			// Something happened (new peer, delivery, timeout), recheck
		case <-peerJoin:
			// A new peer joined, try to assign it work
		case id := <-peerDrop:
			s.revertBALRequests(id, pending)
			for h, set := range refused {
				delete(set, id)
				if len(set) == 0 {
					delete(refused, h)
				}
			}
		case <-cancel:
			return nil, ErrCancelled
		case req := <-accessListReqFails:
			s.revertAccessListRequest(req, pending)
		case res := <-accessListResps:
			s.processAccessListResponse(res, headers, pending, fetched, refused)
		}
	}
	// Assemble results in input order
	results := make([]rlp.RawValue, len(hashes))
	for i, h := range hashes {
		results[i] = fetched[h]
	}
	return results, nil
}

// assignAccessListTasks attempts to assign BAL fetch requests to idle
// peers for any hashes still in pending. Hashes a peer has already refused
// (recorded in refused) are not assigned back to that same peer.
func (s *syncerV2) assignAccessListTasks(pending map[common.Hash]struct{}, refused map[common.Hash]map[string]struct{}, success chan *accessListResponse, fail chan *accessListRequest, cancel chan struct{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Iterate over pending hashes and assign to idle peers
	idlers := s.sortIdlePeers(s.accessListIdlers, AccessListsMsg)
	for len(idlers.ids) > 0 && len(pending) > 0 {
		var (
			idle = idlers.ids[0]
			peer = s.peers[idle]
			cap  = idlers.caps[0]
		)
		idlers.ids, idlers.caps = idlers.ids[1:], idlers.caps[1:]

		// Collect hashes to fetch, capped by peer capacity and the
		// EIP-8189 2 MiB response soft limit (~72 KiB/BAL -> 28 blocks).
		if cap > maxAccessListRequestCount {
			cap = maxAccessListRequestCount
		}
		batch := make([]common.Hash, 0, cap)
		for h := range pending {
			// Skip hashes this peer has already refused; another peer
			// must serve them.
			if set := refused[h]; set != nil {
				if _, ok := set[idle]; ok {
					continue
				}
			}
			delete(pending, h)

			batch = append(batch, h)
			if len(batch) >= cap {
				break
			}
		}
		// The peer has already refused every pending hash; leave them in
		// pending for another peer and move on without a wasted request.
		if len(batch) == 0 {
			continue
		}

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
			peer.Log().Debug("BAL request timed out", "reqid", reqid)
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
				log.Debug("Failed to request BALs", "err", err)
				s.scheduleRevertAccessListRequest(req)
			}
		}()
	}
}

// processAccessListResponse handles a successful BAL response. It
// verifies each non-empty BAL against the corresponding block header and
// stores the verified ones in fetched.
func (s *syncerV2) processAccessListResponse(res *accessListResponse, headers map[common.Hash]*types.Header, pending map[common.Hash]struct{}, fetched map[common.Hash]rlp.RawValue, refused map[common.Hash]map[string]struct{}) {
	var (
		stateless bool
		valid     = make(map[common.Hash]rlp.RawValue)
	)
	// Each response entry corresponds to the requested hash at the same index.
	for i, raw := range res.accessLists {
		h := res.req.hashes[i]

		// Peer doesn't have this BAL (a legitimate reply, e.g. the block is
		// outside its retention window). Record the refusal and add the hash
		// back to pending for a retry against other peers.
		if bytes.Equal(raw, rlp.EmptyString) {
			if refused[h] == nil {
				refused[h] = make(map[string]struct{})
			}
			refused[h][res.req.peer] = struct{}{}
			continue
		}
		var b bal.BlockAccessList
		if err := rlp.DecodeBytes(raw, &b); err != nil {
			log.Warn("Peer sent unparseable BAL", "peer", res.req.peer, "block", h, "err", err)
			stateless = true
			continue
		}
		if err := verifyAccessList(&b, headers[h]); err != nil {
			log.Warn("Peer sent invalid BAL", "peer", res.req.peer, "block", h, "err", err)
			stateless = true
			continue
		}
		valid[h] = raw
	}
	if stateless {
		s.lock.Lock()
		s.statelessPeers[res.req.peer] = struct{}{}
		s.lock.Unlock()
	}
	// Re-add hashes that were not served back or invalid to pending
	for i := 0; i < len(res.req.hashes); i++ {
		if _, ok := valid[res.req.hashes[i]]; ok {
			delete(refused, res.req.hashes[i])
			continue
		}
		pending[res.req.hashes[i]] = struct{}{}
	}
	maps.Copy(fetched, valid)
}

// loadSyncStatus retrieves a previously aborted sync status from the database,
// or generates a fresh one if none is available. The persisted blob is framed
// as `[version byte | JSON payload]`; a missing or mismatching version byte
// causes the progress to be discarded and sync to start fresh.
func (s *syncerV2) loadSyncStatus() {
	var progress syncProgressV2

	if raw := rawdb.ReadSnapshotSyncStatus(s.db); len(raw) > 0 {
		if raw[0] != syncProgressVersion {
			log.Info("Discarding old-format sync progress", "version", raw[0], "expected", syncProgressVersion)
		} else if err := json.Unmarshal(raw[1:], &progress); err != nil {
			log.Error("Failed to decode snap sync status", "err", err)
		} else {
			s.lock.Lock()
			defer s.lock.Unlock()

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
			s.pivot = progress.Pivot
			s.setPhase(progress.Phase)
			s.accountSynced = progress.AccountSynced
			s.accountBytes = progress.AccountBytes
			s.bytecodeSynced = progress.BytecodeSynced
			s.bytecodeBytes = progress.BytecodeBytes
			s.storageSynced = progress.StorageSynced
			s.storageBytes = progress.StorageBytes

			// Seed the externally-exposed snapshot from the restored counters so
			// eth_syncing reports real stats during catch-up and trie generation
			// after a resume, instead of the zero-valued initial snapshot.
			s.refreshProgressLocked()
			return
		}
	}
	// Either we've failed to decode the previous state, or there was none.
	s.resetSyncState()
}

// increaseKey increase the input key by one bit. Return nil if the entire
// addition operation overflows.
func increaseKey(key []byte) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		key[i]++
		if key[i] != 0x0 {
			return key
		}
	}
	return nil
}

// DeleteHistoryByRange completely removes all database entries with the specific
// prefix. Note, this method assumes the space with the given prefix is exclusively
// occupied!
func deleteRange(batch ethdb.Batch, prefix []byte) {
	start := prefix
	limit := increaseKey(bytes.Clone(prefix))

	// Try to remove the data in the range by a loop, as the leveldb
	// doesn't support the native range deletion.
	for {
		err := batch.DeleteRange(start, limit)
		if err == nil {
			return
		}
		// An unclean shutdown may leave the on-disk state partially wiped and
		// therefore inconsistent. This is a tradeoff of the current LevelDB-based
		// approach.
		if errors.Is(err, ethdb.ErrTooManyKeys) {
			batch.Write()
			batch.Reset()
			continue
		}
		log.Crit("Failed to delete state entries", "err", err)
	}
}

// resetTrienodes wipes all persisted trienodes if the path scheme is used.
// It's a defensive operation, ensuring all the leftover trie nodes are cleared
// before the new generation cycle.
func (s *syncerV2) resetTrienodes(batch ethdb.Batch) {
	if s.scheme == rawdb.PathScheme {
		deleteRange(batch, rawdb.TrieNodeAccountPrefix)
		deleteRange(batch, rawdb.TrieNodeStoragePrefix)
	}
}

// resetSyncState wipes all persisted snap-sync data (sync status, account
// and storage snapshots) and re-initializes in-memory state with a fresh
// chunking of the account hash range.
func (s *syncerV2) resetSyncState() {
	batch := s.db.NewBatch()
	rawdb.DeleteSnapshotSyncStatus(batch)
	deleteRange(batch, rawdb.SnapshotAccountPrefix)
	deleteRange(batch, rawdb.SnapshotStoragePrefix)
	s.resetTrienodes(batch)
	batch.Write()

	s.lock.Lock()
	defer s.lock.Unlock()

	s.tasks = nil
	s.pivot = nil
	s.setPhase(phaseDownload)
	s.accountSynced, s.accountBytes = 0, 0
	s.bytecodeSynced, s.bytecodeBytes = 0, 0
	s.storageSynced, s.storageBytes = 0, 0
	s.refreshProgressLocked()

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
		s.tasks = append(s.tasks, &accountTaskV2{
			Next:           next,
			Last:           last,
			SubTasks:       make(map[common.Hash][]*storageTaskV2),
			stateCompleted: make(map[common.Hash]struct{}),
		})
		log.Debug("Created account sync task", "from", next, "last", last)
		next = common.BigToHash(new(big.Int).Add(last.Big(), common.Big1))
	}
}

// saveSyncStatus marshals the remaining sync tasks into db.
func (s *syncerV2) saveSyncStatus() {
	s.saveSyncStatusWithDB(s.db)
}

// saveSyncStatusWithDB marshals the remaining sync tasks into the given database.
func (s *syncerV2) saveSyncStatusWithDB(db ethdb.KeyValueWriter) {
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
	// Store the actual progress markers.
	progress := &syncProgressV2{
		Pivot:          s.pivot,
		Tasks:          s.tasks,
		Phase:          s.getPhase(),
		AccountSynced:  s.accountSynced,
		AccountBytes:   s.accountBytes,
		BytecodeSynced: s.bytecodeSynced,
		BytecodeBytes:  s.bytecodeBytes,
		StorageSynced:  s.storageSynced,
		StorageBytes:   s.storageBytes,
	}
	blob, err := json.Marshal(progress)
	if err != nil {
		panic(err) // This can only fail during implementation
	}
	// Prepend the version byte so future format changes can be detected on load.
	status := append([]byte{syncProgressVersion}, blob...)
	rawdb.WriteSnapshotSyncStatus(db, status)
}

// refreshProgressLocked rebuilds the externally-exposed progress snapshot from
// the live counters. The caller must hold s.lock.
func (s *syncerV2) refreshProgressLocked() {
	s.extProgress = &syncProgressV2{
		AccountSynced:  s.accountSynced,
		AccountBytes:   s.accountBytes,
		BytecodeSynced: s.bytecodeSynced,
		BytecodeBytes:  s.bytecodeBytes,
		StorageSynced:  s.storageSynced,
		StorageBytes:   s.storageBytes,
	}
}

// Progress returns the snap sync status statistics.
func (s *syncerV2) Progress() *syncProgressV2 {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.extProgress
}

// cleanAccountTasks removes account range retrieval tasks that have already been
// completed.
func (s *syncerV2) cleanAccountTasks() {
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
		s.reportSyncProgressV2(true)
	}
}

// cleanStorageTasks iterates over all the account tasks and storage sub-tasks
// within, cleaning any that have been completed.
func (s *syncerV2) cleanStorageTasks() {
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
func (s *syncerV2) assignAccountTasks(success chan *accountResponseV2, fail chan *accountRequestV2, cancel chan struct{}) {
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
		req := &accountRequestV2{
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
		}(s.pivot.Root)

		// Inject the request into the task to block further assignments
		task.req = req
	}
}

// assignBytecodeTasks attempts to match idle peers to pending code retrievals.
func (s *syncerV2) assignBytecodeTasks(success chan *bytecodeResponseV2, fail chan *bytecodeRequestV2, cancel chan struct{}) {
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
		req := &bytecodeRequestV2{
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
func (s *syncerV2) assignStorageTasks(success chan *storageResponseV2, fail chan *storageRequestV2, cancel chan struct{}) {
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
		storageTaskV2s := task.activeSubTasks()
		if len(storageTaskV2s) == 0 && len(task.stateTasks) == 0 {
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
			subtask  *storageTaskV2
		)
		for account, subtasks := range storageTaskV2s {
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
		req := &storageRequestV2{
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
		}(s.pivot.Root)

		// Inject the request into the subtask to block further assignments
		if subtask != nil {
			subtask.req = req
		}
	}
}

// revertStateRequests locates all the currently pending state requests from a
// particular peer and reverts them, rescheduling for others to fulfill.
func (s *syncerV2) revertStateRequests(peer string) {
	// Gather the requests first, revertals need the lock too
	s.lock.Lock()
	var accountReqs []*accountRequestV2
	for _, req := range s.accountReqs {
		if req.peer == peer {
			accountReqs = append(accountReqs, req)
		}
	}
	var bytecodeReqs []*bytecodeRequestV2
	for _, req := range s.bytecodeReqs {
		if req.peer == peer {
			bytecodeReqs = append(bytecodeReqs, req)
		}
	}
	var storageReqs []*storageRequestV2
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

// revertBALRequests locates all the currently pending bal requests from a
// particular peer and reverts them, rescheduling for others to fulfill.
func (s *syncerV2) revertBALRequests(peer string, pending map[common.Hash]struct{}) {
	// Gather the requests first, revertals need the lock too
	s.lock.Lock()
	var accessListReqs []*accessListRequest
	for _, req := range s.accessListReqs {
		if req.peer == peer {
			accessListReqs = append(accessListReqs, req)
		}
	}
	s.lock.Unlock()

	// Revert all the requests matching the peer
	for _, req := range accessListReqs {
		s.revertAccessListRequest(req, pending)
	}
}

// scheduleRevertAccountRequest asks the event loop to clean up an account range
// request and return all failed retrieval tasks to the scheduler for reassignment.
func (s *syncerV2) scheduleRevertAccountRequest(req *accountRequestV2) {
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
func (s *syncerV2) revertAccountRequest(req *accountRequestV2) {
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
func (s *syncerV2) scheduleRevertBytecodeRequest(req *bytecodeRequestV2) {
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
func (s *syncerV2) revertBytecodeRequest(req *bytecodeRequestV2) {
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
func (s *syncerV2) scheduleRevertStorageRequest(req *storageRequestV2) {
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
func (s *syncerV2) revertStorageRequest(req *storageRequestV2) {
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
func (s *syncerV2) scheduleRevertAccessListRequest(req *accessListRequest) {
	select {
	case req.revert <- req:
		// Sync event loop notified
	case <-req.cancel:
		// Sync cycle got cancelled
	case <-req.stale:
		// Request already reverted
	}
}

// revertAccessListRequest cleans up an BAL request and returns all
// failed retrieval tasks to the scheduler for reassignment.
func (s *syncerV2) revertAccessListRequest(req *accessListRequest, pending map[common.Hash]struct{}) {
	log.Debug("Reverting BAL request", "peer", req.peer)
	select {
	case <-req.stale:
		log.Trace("BAL request already reverted", "peer", req.peer, "reqid", req.id)
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
	for _, h := range req.hashes {
		pending[h] = struct{}{}
	}
}

// processAccountResponse integrates an already validated account range response
// into the account tasks.
func (s *syncerV2) processAccountResponse(res *accountResponseV2) {
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
			if hash.Cmp(last) > 0 {
				log.Debug("Keeping suspended storage retrieval", "account", hash)
				continue
			}
			if _, ok := resumed[hash]; !ok {
				log.Warn("Aborting suspended storage retrieval", "account", hash)
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
func (s *syncerV2) processBytecodeResponse(res *bytecodeResponseV2) {
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
func (s *syncerV2) processStorageResponse(res *storageResponseV2) {
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
					tasks = append(tasks, &storageTaskV2{
						Next: common.Hash{},
						Last: r.End(),
						root: acc.Root,
					})
					for r.Next() {
						tasks = append(tasks, &storageTaskV2{
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

		// Persist the received storage segments. These flat state may be outdated
		// during the sync, but it will be fixed by the BAL-healing.
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
func (s *syncerV2) forwardAccountTask(task *accountTaskV2) {
	// Remove any pending delivery
	res := task.res
	if res == nil {
		return // nothing to forward
	}
	task.res = nil

	// Persist the received account segments. These flat state maybe
	// outdated during the sync, but it can be fixed later during the
	// trie generation.
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
func (s *syncerV2) OnAccounts(peer SyncPeerV2, id uint64, hashes []common.Hash, accounts [][]byte, proof [][]byte) error {
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
		logger.Debug("Peer rejected account range request", "root", s.pivot.Root)
		s.statelessPeers[peer.ID()] = struct{}{}
		s.lock.Unlock()

		// Signal this request as failed, and ready for rescheduling
		s.scheduleRevertAccountRequest(req)
		return nil
	}
	root := s.pivot.Root
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
	response := &accountResponseV2{
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
// bytes codes are received from a remote peer in the syncing phase.
func (s *syncerV2) OnByteCodes(peer SyncPeerV2, id uint64, bytecodes [][]byte) error {
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
	response := &bytecodeResponseV2{
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
func (s *syncerV2) OnStorage(peer SyncPeerV2, id uint64, hashes [][]common.Hash, slots [][][]byte, proof [][]byte) error {
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
	response := &storageResponseV2{
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

// OnAccessLists is a callback method to invoke when a batch of BALs
// are received from a remote peer.
func (s *syncerV2) OnAccessLists(peer SyncPeerV2, id uint64, accessLists rlp.RawList[rlp.RawValue]) error {
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
		logger.Warn("Unexpected BAL packet")
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
		logger.Debug("Peer rejected BAL request")
		s.statelessPeers[peer.ID()] = struct{}{}
		s.lock.Unlock()

		// Signal this request as failed, and ready for rescheduling
		s.scheduleRevertAccessListRequest(req)
		return nil
	}
	if len(bals) > len(req.hashes) {
		s.lock.Unlock()
		s.scheduleRevertAccessListRequest(req)
		logger.Warn("Peer sent more BALs than requested", "count", len(bals), "requested", len(req.hashes))
		return errors.New("more BALs than requested")
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

// report calculates various status reports and provides it to the user.
func (s *syncerV2) report(force bool) {
	s.reportSyncProgressV2(force)
}

// reportSyncProgressV2 calculates various status reports and provides it to the user.
func (s *syncerV2) reportSyncProgressV2(force bool) {
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

// checkAccessListProgress reports whether the BAL fetch can still make
// forward progress against the current peer set.
func (s *syncerV2) checkAccessListProgress(pending map[common.Hash]struct{}, refused map[common.Hash]map[string]struct{}) error {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if len(s.peers) == 0 {
		return nil
	}
	if len(s.accessListReqs) > 0 {
		return nil
	}
	serviceable := make(map[string]struct{}, len(s.peers))
	for id := range s.peers {
		if _, ok := s.statelessPeers[id]; !ok {
			serviceable[id] = struct{}{}
		}
	}
	if len(serviceable) == 0 {
		return errAccessListPeersExhausted
	}
	for h, set := range refused {
		// Delivered by some other peer after all
		if _, ok := pending[h]; !ok {
			continue
		}
		unobtainable := true
		for id := range serviceable {
			if _, ok := set[id]; !ok {
				unobtainable = false
				break
			}
		}
		if unobtainable {
			log.Warn("Access list unavailable from all peers", "hash", h)
			return errAccessListUnavailable
		}
	}
	return nil
}

// sortIdlePeers builds a list of idle peers sorted by download capacity
// (highest first), filtering out stateless peers. Must be called with s.lock held.
func (s *syncerV2) sortIdlePeers(idlerSet map[string]struct{}, msgCode uint64) *capacitySort {
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
