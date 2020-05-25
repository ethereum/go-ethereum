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
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
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
)

// accountRequest tracks a pending account range request to ensure responses are
// to actual requests and to validate any security constraints.
type accountRequest struct {
	id     uint64        // Request ID to drop stale replies
	origin common.Hash   // Origin account to guarantee overlaps
	cancel chan struct{} // Channel to track sync cancellation
}

// accountResponse is an already Merkle-verified remote response to an account
// range request. It contains the subtrie for the requested account range and
// the database that's going to be filled with the internal nodes on commit.
type accountResponse struct {
	hashes   []common.Hash            // Account hashes in the returned range
	accounts [][]byte                 // Account values in the returned range
	nodes    ethdb.KeyValueStore      // Database containing the reconstructed trie nodes
	trie     *trie.Trie               // Reconstructed trie to reject incomplete account paths
	bounds   map[common.Hash]struct{} // Boundary nodes to avoid persisting (incomplete)
}

// storageRequest tracks a pending storage range request to ensure responses are
// to actual requests and to validate any security constraints.
type storageRequest struct {
	id     uint64        // Request ID to drop stale replies
	root   common.Hash   // Storage trie root hash to prove
	origin common.Hash   // Origin slot to guarantee overlaps
	cancel chan struct{} // Channel to track sync cancellation
}

// storageResponse is an already Merkle-verified remote response to a storage
// range request. It contains the subtrie for the requested storage range and
// the database that's going to be filled with the internal nodes on commit.
type storageResponse struct {
	hashes []common.Hash            // Storage slot hashes in the returned range
	slots  [][]byte                 // Storage slot values in the returned range
	nodes  ethdb.KeyValueStore      // Database containing the reconstructed trie nodes
	bounds map[common.Hash]struct{} // Boundary nodes to avoid persisting (incomplete)
	last   common.Hash              // Last returned slot, acts as the next query origin
}

// byteCodesRequest tracks a pending bytecode request to ensure responses are to
// actual requests and to validate any security constraints.
type bytecodeRequest struct {
	id     uint64        // Request ID to drop stale replies
	hashes []common.Hash // Bytecode hashes to validate responses
	cancel chan struct{} // Channel to track sync cancellation
}

// bytecodeResponse is an already verified remote response to a bytecode request.
type bytecodeResponse struct {
	hashes []common.Hash // Hashes of the bytecode to avoid double hashing
	codes  [][]byte      // Actual bytecodes to store into the database (nil = missing)
}

// Syncer is an Ethereum account and storage trie syncer based on snapshots and
// the  snap protocol. It's purpose is to download all the accounts and storage
// slots from remote peers and reassemble chunks of the state trie, on top of
// which a state sync can be run to fix any gaps / overlaps.
type Syncer struct {
	db    ethdb.KeyValueStore // Database to store the trie nodes into (and dedup)
	bloom *trie.SyncBloom     // Bloom filter to deduplicate nodes for state fixup

	root     common.Hash      // Current state trie root being synced
	nextAcc  common.Hash      // Next account to sync after a restart
	nextSlot common.Hash      // Next storage slot to sync after a restart
	done     bool             // Flag whether sync has already completed
	peers    map[string]*Peer // Currently active peers to download from

	accountReqs   map[string]*accountRequest  // Account requests currently running for a peer
	storageReqs   map[string]*storageRequest  // Storage requests currently running for a peer
	bytecodeReqs  map[string]*bytecodeRequest // Bytecode requests currently running for a peer
	accountResps  chan *accountResponse       // Account sub-tries to integrate into the database
	storageResps  chan *storageResponse       // Storage sub-tries to integrate into the database
	bytecodeResps chan *bytecodeResponse      // Bytecodes to integrate into the database

	accountRequests  uint64             // Number of account range requests
	accountSynced    uint64             // Number of accounts downloaded
	accountProofs    uint64             // Number of trie nodes received for account proofs
	accountNodes     uint64             // Number of account trie nodes persisted to disk
	accountBytes     common.StorageSize // Number of account trie bytes persisted to disk
	storageRequests  uint64             // Number of storage range requests
	storageSynced    uint64             // Number of storage slots downloaded
	storageProofs    uint64             // Number of trie nodes received for storage proofs
	storageNodes     uint64             // Number of storage trie nodes persisted to disk
	storageBytes     common.StorageSize // Number of storage trie bytes persisted to disk
	bytecodeRequests uint64             // Number of bytecode set requests
	bytecodeSynced   uint64             // Number of bytecodes downloaded
	bytecodeBytes    common.StorageSize // Number of bytecodes downloaded

	startTime time.Time   // Time instance when snapshot sync started
	startAcc  common.Hash // Account hash where sync started from
	logTime   time.Time   // Time instance when status was last reported

	lock sync.RWMutex //
}

func NewSyncer(db ethdb.KeyValueStore, bloom *trie.SyncBloom) *Syncer {
	return &Syncer{
		db:            db,
		bloom:         bloom,
		peers:         make(map[string]*Peer),
		accountReqs:   make(map[string]*accountRequest),
		storageReqs:   make(map[string]*storageRequest),
		bytecodeReqs:  make(map[string]*bytecodeRequest),
		accountResps:  make(chan *accountResponse),
		storageResps:  make(chan *storageResponse),
		bytecodeResps: make(chan *bytecodeResponse),
	}
}

// Register injects a new data source into the syncer's peerset.
func (s *Syncer) Register(peer *Peer) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.peers[peer.id]; ok {
		log.Error("Snap peer already registered", "id", peer.id)
		return errors.New("already registered")
	}
	s.peers[peer.id] = peer
	return nil
}

// Unregister injects a new data source into the syncer's peerset.
func (s *Syncer) Unregister(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.peers[id]; !ok {
		log.Error("Snap peer not registered", "id", id)
		return errors.New("not registered")
	}
	delete(s.peers, id)
	return nil
}

// Sync starts (or resumes a previous) sync cycle to iterate over an state trie
// with the given root and reconstruct the nodes based on the snapshot leaves.
// Previously downloaded segments will not be redownloaded of fixed, rather any
// errors will be healed after the leaves are fully accumulated.
func (s *Syncer) Sync(root common.Hash, cancel chan struct{}) error {
	// Move the trie root from any previous value
	s.lock.Lock()
	s.root = root
	done := s.done

	if s.startTime == (time.Time{}) {
		s.startTime = time.Now()
	}
	s.lock.Unlock()

	if done {
		return nil
	}
	defer s.report(true)
	log.Debug("Starting snapshot sync cycle", "root", root, "account", s.nextAcc, "slot", s.nextSlot)

	// Whether sync completed or not, disregard any future packets
	defer func() {
		s.lock.Lock()
		s.accountReqs = make(map[string]*accountRequest)
		s.storageReqs = make(map[string]*storageRequest)
		s.bytecodeReqs = make(map[string]*bytecodeRequest)
		s.lock.Unlock()
	}()

	// Launch the account trie sync
	var peer *Peer

	s.lock.RLock()
	for _, p := range s.peers {
		peer = p // TODO(karalabe): Myeah, ugly hack
		break
	}
	s.lock.RUnlock()

	if err := s.syncAccounts(root, peer, cancel); err != nil {
		return err
	}
	// If sync completed without errors and interruptions, disable it
	select {
	case <-cancel:
	default:
		s.lock.Lock()
		s.done = true
		s.lock.Unlock()
	}
	return nil
}

// syncAccounts starts (or resumes a previous) sync cycle to iterate over an state trie
// with the given root and reconstruct the nodes based on the snapshot leaves.
// Previously downloaded segments will not be redownloaded of fixed, rather any
// errors will be healed after the leaves are fully accumulated.
func (s *Syncer) syncAccounts(root common.Hash, peer *Peer, cancel chan struct{}) error {
	// For now simply iterate over the state iteratively
	batch := s.db.NewBatch()

	for {
		// If the sync was cancelled, abort
		select {
		case <-cancel:
			return nil
		default:
		}
		// Generate a random request ID (clash probability is insignificant)
		id := uint64(rand.Int63())

		// Track the request and send it to the peer
		s.lock.Lock()
		s.accountReqs[peer.id] = &accountRequest{
			id:     id,
			origin: s.nextAcc,
			cancel: cancel,
		}
		s.lock.Unlock()

		if err := peer.RequestAccountRange(id, root, s.nextAcc, maxRequestSize); err != nil {
			return err
		}
		s.accountRequests++

		// Wait for the reply to arrive
		res := <-s.accountResps
		if res == nil {
			return errors.New("unfulfilled request")
		}
		s.accountSynced += uint64(len(res.accounts))
		s.accountProofs += uint64(len(res.bounds))

		// Hack
		if res.nodes == nil {
			break
		}
		// Iterate over all the accounts and retrieve any missing storage tries.
		// Any storage tries that we can't sync fully in one go (proofs == missing
		// boundary nodes) will be marked incomplete to heal later.
		incompletes := light.NewNodeSet()

		for i, blob := range res.accounts {
			// Decode the retrieved account
			var acc state.Account
			if err := rlp.DecodeBytes(blob, &acc); err != nil {
				log.Error("Failed to decode full account", "err", err)
				return err
			}
			// Retrieve any associated bytecode, if not yet downloaded
			if !bytes.Equal(acc.CodeHash, emptyCode[:]) {
				if code, err := s.db.Get(acc.CodeHash); err != nil || code == nil {
					// Generate a random request ID (clash probability is insignificant)
					id := uint64(rand.Int63())

					// Track the request and send it to the peer
					s.lock.Lock()
					s.bytecodeReqs[peer.id] = &bytecodeRequest{
						id:     id,
						hashes: []common.Hash{common.BytesToHash(acc.CodeHash)},
						cancel: cancel,
					}
					s.lock.Unlock()

					if err := peer.RequestByteCodes(id, []common.Hash{common.BytesToHash(acc.CodeHash)}, maxRequestSize); err != nil {
						return err
					}
					s.bytecodeRequests++

					// Wait for the reply to arrive
					res := <-s.bytecodeResps
					s.bytecodeSynced += uint64(len(res.codes))

					if len(res.codes) != 1 || res.codes[0] == nil {
						return errors.New("protocol violation")
					}
					s.bytecodeBytes += common.StorageSize(len(res.codes[0]))
					s.db.Put(acc.CodeHash, res.codes[0])
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
		// Persist every finalized trie node that's not on the boundary
		it := res.nodes.NewIterator(nil, nil)
		for it.Next() {
			// Boundary nodes are not written, since they are incomplete
			if _, ok := res.bounds[common.BytesToHash(it.Key())]; ok {
				continue
			}
			// Accounts with split storage requests are incomplete
			if _, err := incompletes.Get(it.Key()); err == nil {
				continue
			}
			// Node is neither a boundary, not an incomplete account, persist to disk
			batch.Put(it.Key(), it.Value())
			s.bloom.Add(it.Key())

			s.accountNodes++
			s.accountBytes += common.StorageSize(common.HashLength + len(it.Value()))
		}
		it.Release()

		if err := batch.Write(); err != nil {
			return err
		}
		batch.Reset()

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
		s.storageRequests++

		// Wait for the reply to arrive
		res := <-s.storageResps
		if res == nil {
			return false, errors.New("unfulfilled request")
		}
		s.storageSynced += uint64(len(res.slots))
		s.storageProofs += uint64(len(res.bounds))

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

			s.storageNodes++
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
}

// OnAccounts is a callback method to invoke when a range of accounts are
// received from a remote peer.
func (s *Syncer) OnAccounts(peer *Peer, id uint64, hashes []common.Hash, accounts [][]byte, proof [][]byte) error {
	peer.Log().Trace("Delivering range of accounts", "hashes", len(hashes), "accounts", len(accounts), "proofs", len(proof))

	// If the request is stale, discard it
	s.lock.Lock()
	req, ok := s.accountReqs[peer.id]
	if !ok || req.id != id {
		peer.Log().Warn("Unexpected account range packet")
		s.lock.Unlock()
		return nil
	}
	delete(s.accountReqs, peer.id)
	root := s.root
	s.lock.Unlock()

	// If the response is unavailable snapshot, forward to the requester
	if len(hashes) == 0 && len(accounts) == 0 && len(proof) == 0 {
		select {
		//case <-req.cancel:
		case s.accountResps <- nil:
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
	proofdb := nodes.NodeSet()

	db, tr, err := trie.VerifyRangeProof(root, req.origin[:], keys, accounts, proofdb, proofdb)
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
	response := &accountResponse{
		hashes:   hashes,
		accounts: accounts,
		nodes:    db,
		trie:     tr,
		bounds:   bounds,
	}
	select {
	//case <-req.cancel:
	case s.accountResps <- response:
	}
	return nil
}

// OnStorage is a callback method to invoke when a range of storage slots
// are received from a remote peer.
func (s *Syncer) OnStorage(peer *Peer, id uint64, hashes []common.Hash, slots [][]byte, proof [][]byte) error {
	peer.Log().Trace("Delivering range of storage slots", "hashes", len(hashes), "slots", len(slots), "proofs", len(proof))

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
	}
	return nil
}

// OnByteCodes is a callback method to invoke when a batch of contract
// bytes codes are received from a remote peer.
func (s *Syncer) OnByteCodes(peer *Peer, id uint64, bytecodes [][]byte) error {
	peer.Log().Trace("Delivering set of bytecodes", "bytecodes", len(bytecodes))

	// If the request is stale, discard it
	s.lock.Lock()
	req, ok := s.bytecodeReqs[peer.id]
	if !ok || req.id != id {
		peer.Log().Warn("Unexpected bytecode packet")
		s.lock.Unlock()
		return nil
	}
	delete(s.storageReqs, peer.id)
	s.lock.Unlock()

	// Cross reference the requested bytecodes with the response to find gaps
	// that the serving node is missing
	hasher := sha3.NewLegacyKeccak256()

	codes := make([][]byte, 0, len(req.hashes))
	for i, j := 0, 0; i < len(bytecodes); i++ {
		// Find the next hash that we've been served, add gaps in between
		hasher.Reset()
		hasher.Write(bytecodes[i])
		hash := hasher.Sum(nil)

		for j < len(req.hashes) && !bytes.Equal(hash, req.hashes[j][:]) {
			codes = append(codes, nil)
			j++
		}
		if j < len(req.hashes) {
			codes = append(codes, bytecodes[i])
			j++
			continue
		}
		// We've either ran out of hashes, or got unrequested data
		peer.Log().Warn("Unexpected bytecodes", "count", len(bytecodes)-i)
		return errors.New("unexpected bytecode")
	}
	// Response validated, send it to the scheduler for filling
	response := &bytecodeResponse{
		hashes: req.hashes,
		codes:  codes,
	}
	select {
	//case <-req.cancel:
	case s.bytecodeResps <- response:
	}
	return nil
}

// OnTrieNodes is a callback method to invoke when a batch of trie nodes
// are received from a remote peer.
func (s *Syncer) OnTrieNodes(peer *Peer, id uint64, nodes [][]byte) error {
	return errors.New("not implemented")
}

// report calculates various status reports and provides it to the user.
func (s *Syncer) report(force bool) {
	// Don't report all the events, just occasionally
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
	log.Info("State sync progress report", "synced", progress, "bytes", synced,
		"accounts", accounts, "storage", storage, "code", bytecode, "eta", common.PrettyDuration(estTime-elapsed))
}
