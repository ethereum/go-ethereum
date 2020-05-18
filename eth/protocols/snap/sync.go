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
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"

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
	ctx    context.Context // Context to track cancellations
	id     uint64          // Request ID to drop stale replies
	origin common.Hash     // Origin account to guarantee overlaps
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
	last     common.Hash              // Last returned account, acts as the next query origin
}

// storageRequest tracks a pending storage range request to ensure responses are
// to actual requests and to validate any security constraints.
type storageRequest struct {
	ctx    context.Context // Context to track cancellations
	id     uint64          // Request ID to drop stale replies
	root   common.Hash     // Storage trie root hash to prove
	origin common.Hash     // Origin slot to guarantee overlaps
}

// storageResponse is an already Merkle-verified remote response to a storage
// range request. It contains the subtrie for the requested storage range and
// the database that's going to be filled with the internal nodes on commit.
type storageResponse struct {
	nodes  ethdb.KeyValueStore      // Database containing the reconstructed trie nodes
	bounds map[common.Hash]struct{} // Boundary nodes to avoid persisting (incomplete)
	last   common.Hash              // Last returned slot, acts as the next query origin
}

// byteCodesRequest tracks a pending bytecode request to ensure responses are to
// actual requests and to validate any security constraints.
type bytecodeRequest struct {
	ctx    context.Context // Context to track cancellations
	id     uint64          // Request ID to drop stale replies
	hashes []common.Hash   // Bytecode hashes to validate responses
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

	root  common.Hash      // Current state trie root being synced
	done  bool             // Flag whether sync has already completed
	peers map[string]*Peer // Currently active peers to download from

	accountReqs   map[string]*accountRequest  // Account requests currently running for a peer
	storageReqs   map[string]*storageRequest  // Storage requests currently running for a peer
	bytecodeReqs  map[string]*bytecodeRequest // Bytecode requests currently running for a peer
	accountResps  chan *accountResponse       // Account sub-tries to integrate into the database
	storageResps  chan *storageResponse       // Storage sub-tries to integrate into the database
	bytecodeResps chan *bytecodeResponse      // Bytecodes to integrate into the database

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
func (s *Syncer) Sync(ctx context.Context, root common.Hash) error {

	// Move the trie root from any previous value
	s.lock.Lock()
	s.root = root
	done := s.done
	s.lock.Unlock()

	if done {
		return nil
	}
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

	if err := s.syncAccounts(ctx, root, peer); err != nil {
		return err
	}
	s.lock.Lock()
	s.done = true
	s.lock.Unlock()

	return nil
}

// syncAccounts starts (or resumes a previous) sync cycle to iterate over an state trie
// with the given root and reconstruct the nodes based on the snapshot leaves.
// Previously downloaded segments will not be redownloaded of fixed, rather any
// errors will be healed after the leaves are fully accumulated.
func (s *Syncer) syncAccounts(ctx context.Context, root common.Hash, peer *Peer) error {
	// For now simply iterate over the state iteratively
	next := common.Hash{}

	var (
		batch = s.db.NewBatch()

		nodes = uint64(0)
		size  = uint64(0)
	)
	for {
		// Generate a random request ID (clash probability is insignificant)
		id := uint64(rand.Int63())

		// Track the request and send it to the peer
		s.lock.Lock()
		s.accountReqs[peer.id] = &accountRequest{
			ctx:    ctx,
			id:     id,
			origin: next,
		}
		s.lock.Unlock()

		if err := peer.RequestAccountRange(id, root, next, maxRequestSize); err != nil {
			return err
		}
		// Wait for the reply to arrive
		res := <-s.accountResps

		// HAck
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
						ctx:    ctx,
						id:     id,
						hashes: []common.Hash{common.BytesToHash(acc.CodeHash)},
					}
					s.lock.Unlock()

					if err := peer.RequestByteCodes(id, []common.Hash{common.BytesToHash(acc.CodeHash)}, maxRequestSize); err != nil {
						return err
					}
					// Wait for the reply to arrive
					res := <-s.bytecodeResps

					if len(res.codes) != 1 || res.codes[0] == nil {
						return errors.New("protocol violation")
					}
					s.db.Put(acc.CodeHash, res.codes[0])
				}
			}
			// Retrieve any associated storage trie, if not yet downloaded
			if acc.Root != emptyRoot {
				if node, err := s.db.Get(acc.Root[:]); err != nil || node == nil {
					// Sync the contract's storage trie
					snodes, ssize, complete, err := s.syncStorage(ctx, root, res.hashes[i], acc.Root, peer)
					if err != nil {
						return err
					}
					nodes += snodes
					size += ssize

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
			// TODO(karalabe): if the snapshot moved during contact sync, nuke the account path
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

			size += uint64(common.HashLength + len(it.Value()))
			nodes++
		}
		it.Release()

		if err := batch.Write(); err != nil {
			return err
		}
		batch.Reset()

		// Account range processed, step to the next chunk
		log.Info("Persisted range of accounts", "at", res.last, "nodes", nodes, "bytes", common.StorageSize(size))

		/*nextAccount = common.BigToHash(new(big.Int).Add(res.last.Big(), big.NewInt(1)))
		if nextAccount == (common.Hash{}) {
			break // Overflow, someone created 0xff...f, oh well
		}*/
		next = res.last
	}
	return nil
}

// syncStorage starts (or resumes a previous) sync cycle to iterate over a storage
// trie with the given root and reconstruct the nodes based on the snapshot leaves.
// Previously downloaded segments will not be redownloaded of fixed, rather any
// errors will be healed after the leaves are fully accumulated.
func (s *Syncer) syncStorage(ctx context.Context, root common.Hash, account common.Hash, stroot common.Hash, peer *Peer) (uint64, uint64, bool, error) {
	// For now simply iterate over the state iteratively
	next := common.Hash{}

	var (
		batch = s.db.NewBatch()

		nodes = uint64(0)
		size  = uint64(0)
	)
	for {
		// Generate a random request ID (clash probability is insignificant)
		id := uint64(rand.Int63())

		// Track the request and send it to the peer
		s.lock.Lock()
		s.storageReqs[peer.id] = &storageRequest{
			ctx:    ctx,
			id:     id,
			root:   stroot,
			origin: next,
		}
		s.lock.Unlock()

		if err := peer.RequestStorageRange(id, root, account, next, maxRequestSize); err != nil {
			return 0, 0, false, err
		}
		// Wait for the reply to arrive
		res := <-s.storageResps

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

			size += uint64(common.HashLength + len(it.Value()))
			nodes++
		}
		it.Release()

		if err := batch.Write(); err != nil {
			return 0, 0, false, err
		}
		batch.Reset()

		// Storage range processed, step to the next chunk
		log.Debug("Persisted range of storage slots", "account", account, "at", res.last, "nodes", nodes, "bytes", common.StorageSize(size))

		/*nextAccount = common.BigToHash(new(big.Int).Add(res.last.Big(), big.NewInt(1)))
		if nextAccount == (common.Hash{}) {
			break // Overflow, someone created 0xff...f, oh well
		}*/
		// If the response contained all the data in one shot (no proofs), there
		// is no reason to continue the sync, report immediate success.
		if len(res.bounds) == 0 {
			return nodes, size, true, nil
		}
		next = res.last
	}
	return nodes, size, false, nil
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

	db, tr, err := trie.VerifyRangeProof(root, keys, accounts, proofdb, proofdb)
	if err != nil {
		return err
	}
	// The returned data checks out, ensure there are no malicious prefix gaps
	if len(hashes) == 0 || hashes[0] != req.origin {
		// Prefix looks funky, make sure the response is what we asked for
		if val, err := trie.VerifyProof(root, req.origin[:], proofdb); err != nil {
			return err
		} else if val != nil {
			peer.Log().Warn("Skipped origin account proof", "req", id)
			return fmt.Errorf("skipped origin proof: %x -> %x", req.origin, val)
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
	response := &accountResponse{
		hashes:   hashes,
		accounts: accounts,
		nodes:    db,
		trie:     tr,
		bounds:   bounds,
		last:     hashes[len(hashes)-1], // TODO(karalabe): bounds check
	}
	select {
	case <-req.ctx.Done():
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
		db, _, err = trie.VerifyRange(req.root, keys, slots)
		if err != nil {
			return err
		}
	} else {
		// A proof was attached, the response is only partial, check that the
		// returned data is indeed part of the storage trie
		proofdb := nodes.NodeSet()

		db, _, err = trie.VerifyRangeProof(req.root, keys, slots, proofdb, proofdb)
		if err != nil {
			return err
		}
		// The returned data checks out, ensure there are no malicious prefix gaps
		if len(hashes) == 0 || hashes[0] != req.origin {
			// Prefix looks funky, make sure the response is what we asked for
			if val, err := trie.VerifyProof(req.root, req.origin[:], proofdb); err != nil {
				return err
			} else if val != nil {
				peer.Log().Warn("Skipped origin slot proof")
				return fmt.Errorf("skipped origin proof: %x -> %x", req.origin, val)
			}
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
	response := &storageResponse{
		nodes:  db,
		bounds: bounds,
		last:   hashes[len(hashes)-1], // TODO(karalabe): bounds check
	}
	select {
	case <-req.ctx.Done():
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
	case <-req.ctx.Done():
	case s.bytecodeResps <- response:
	}
	return nil
}

// OnTrieNodes is a callback method to invoke when a batch of trie nodes
// are received from a remote peer.
func (s *Syncer) OnTrieNodes(peer *Peer, id uint64, nodes [][]byte) error {
	return errors.New("not implemented")
}
