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

package state

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-verkle"
)

const (
	// Number of codehash->size associations to keep.
	codeSizeCacheSize = 100000

	// Cache size granted for caching clean code.
	codeCacheSize = 64 * 1024 * 1024

	// Number of address->curve point associations to keep.
	pointCacheSize = 4096
)

// Database wraps access to tries and contract code.
type Database interface {
	// Reader returns a state reader associated with the specified state root.
	Reader(root common.Hash) (Reader, error)

	// OpenTrie opens the main account trie.
	OpenTrie(root common.Hash) (Trie, error)

	// OpenStorageTrie opens the storage trie of an account.
	OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, trie Trie) (Trie, error)

	// PointCache returns the cache holding points used in verkle tree key computation
	PointCache() *utils.PointCache

	// TrieDB returns the underlying trie database for managing trie nodes.
	TrieDB() *triedb.Database

	// Snapshot returns the underlying state snapshot.
	Snapshot() *snapshot.Tree

	// StartVerkleTransition marks the start of the verkle transition
	StartVerkleTransition(originalRoot, translatedRoot common.Hash, chainConfig *params.ChainConfig, verkleTime *uint64, root common.Hash)

	// EndVerkleTransition marks the end of the verkle transition
	EndVerkleTransition()

	// InTransition returns true if the verkle transition is currently ongoing
	InTransition() bool

	// Transitioned returns true if the verkle transition has ended
	Transitioned() bool

	InitTransitionStatus(bool, bool, common.Hash)

	// SetCurrentSlotHash provides the next slot to be translated
	SetCurrentSlotHash(common.Hash)

	// GetCurrentAccountAddress returns the address of the account that is currently being translated
	GetCurrentAccountAddress() *common.Address

	SetCurrentAccountAddress(common.Address)

	GetCurrentAccountHash() common.Hash

	GetCurrentSlotHash() common.Hash

	SetStorageProcessed(bool)

	GetStorageProcessed() bool

	GetCurrentPreimageOffset() int64

	SetCurrentPreimageOffset(int64)

	AddRootTranslation(originalRoot, translatedRoot common.Hash)

	SetLastMerkleRoot(common.Hash)

	SaveTransitionState(common.Hash)

	LoadTransitionState(common.Hash)

	LockCurrentTransitionState()

	UnLockCurrentTransitionState()
}

// Trie is a Ethereum Merkle Patricia trie.
type Trie interface {
	// GetKey returns the sha3 preimage of a hashed key that was previously used
	// to store a value.
	//
	// TODO(fjl): remove this when StateTrie is removed
	GetKey([]byte) []byte

	// GetAccount abstracts an account read from the trie. It retrieves the
	// account blob from the trie with provided account address and decodes it
	// with associated decoding algorithm. If the specified account is not in
	// the trie, nil will be returned. If the trie is corrupted(e.g. some nodes
	// are missing or the account blob is incorrect for decoding), an error will
	// be returned.
	GetAccount(address common.Address) (*types.StateAccount, error)

	// GetStorage returns the value for key stored in the trie. The value bytes
	// must not be modified by the caller. If a node was not found in the database,
	// a trie.MissingNodeError is returned.
	GetStorage(addr common.Address, key []byte) ([]byte, error)

	// UpdateAccount abstracts an account write to the trie. It encodes the
	// provided account object with associated algorithm and then updates it
	// in the trie with provided address.
	UpdateAccount(address common.Address, account *types.StateAccount, codeLen int) error

	// UpdateStorage associates key with value in the trie. If value has length zero,
	// any existing value is deleted from the trie. The value bytes must not be modified
	// by the caller while they are stored in the trie. If a node was not found in the
	// database, a trie.MissingNodeError is returned.
	UpdateStorage(addr common.Address, key, value []byte) error

	// DeleteAccount abstracts an account deletion from the trie.
	DeleteAccount(address common.Address) error

	// DeleteStorage removes any existing value for key from the trie. If a node
	// was not found in the database, a trie.MissingNodeError is returned.
	DeleteStorage(addr common.Address, key []byte) error

	// UpdateContractCode abstracts code write to the trie. It is expected
	// to be moved to the stateWriter interface when the latter is ready.
	UpdateContractCode(address common.Address, codeHash common.Hash, code []byte) error

	// Hash returns the root hash of the trie. It does not write to the database and
	// can be used even if the trie doesn't have one.
	Hash() common.Hash

	// Commit collects all dirty nodes in the trie and replace them with the
	// corresponding node hash. All collected nodes(including dirty leaves if
	// collectLeaf is true) will be encapsulated into a nodeset for return.
	// The returned nodeset can be nil if the trie is clean(nothing to commit).
	// Once the trie is committed, it's not usable anymore. A new trie must
	// be created with new root and updated trie database for following usage
	Commit(collectLeaf bool) (common.Hash, *trienode.NodeSet)

	// Witness returns a set containing all trie nodes that have been accessed.
	// The returned map could be nil if the witness is empty.
	Witness() map[string]struct{}

	// NodeIterator returns an iterator that returns nodes of the trie. Iteration
	// starts at the key after the given start key. And error will be returned
	// if fails to create node iterator.
	NodeIterator(startKey []byte) (trie.NodeIterator, error)

	// Prove constructs a Merkle proof for key. The result contains all encoded nodes
	// on the path to the value at key. The value itself is also included in the last
	// node and can be retrieved by verifying the proof.
	//
	// If the trie does not contain a value for key, the returned proof contains all
	// nodes of the longest existing prefix of the key (at least the root), ending
	// with the node that proves the absence of the key.
	Prove(key []byte, proofDb ethdb.KeyValueWriter) error

	// IsVerkle returns true if the trie is verkle-tree based
	IsVerkle() bool
}

// CachingDB is an implementation of Database interface. It leverages both trie and
// state snapshot to provide functionalities for state access. It's meant to be a
// long-live object and has a few caches inside for sharing between blocks.
type CachingDB struct {
	disk          ethdb.KeyValueStore
	triedb        *triedb.Database
	snap          *snapshot.Tree
	codeCache     *lru.SizeConstrainedCache[common.Hash, []byte]
	codeSizeCache *lru.Cache[common.Hash, int]
	pointCache    *utils.PointCache

	// Transition-specific fields
	CurrentTransitionState *TransitionState
	TransitionStatePerRoot lru.BasicLRU[common.Hash, *TransitionState]
	transitionStateLock    sync.Mutex
	addrToPoint            *utils.PointCache
	baseRoot               common.Hash // hash of last read-only MPT base tree
}

// NewDatabase creates a state database with the provided data sources.
func NewDatabase(triedb *triedb.Database, snap *snapshot.Tree) *CachingDB {
	return &CachingDB{
		disk:          triedb.Disk(),
		triedb:        triedb,
		snap:          snap,
		codeCache:     lru.NewSizeConstrainedCache[common.Hash, []byte](codeCacheSize),
		codeSizeCache: lru.NewCache[common.Hash, int](codeSizeCacheSize),
		pointCache:    utils.NewPointCache(pointCacheSize),
	}
}

func (db *CachingDB) InTransition() bool {
	return db.CurrentTransitionState != nil && db.CurrentTransitionState.Started && !db.CurrentTransitionState.Ended
}

func (db *CachingDB) Transitioned() bool {
	return db.CurrentTransitionState != nil && db.CurrentTransitionState.Ended
}

// StartVerkleTransition marks the start of the verkle transition
func (db *CachingDB) StartVerkleTransition(originalRoot, translatedRoot common.Hash, chainConfig *params.ChainConfig, verkleTime *uint64, root common.Hash) {
	db.CurrentTransitionState = &TransitionState{
		Started: true,
		// initialize so that the first storage-less accounts are processed
		StorageProcessed: true,
	}
	// db.AddTranslation(originalRoot, translatedRoot)
	db.baseRoot = originalRoot

	// Reinitialize values in case of a reorg
	if verkleTime != nil {
		chainConfig.VerkleTime = verkleTime
	}
}

func (db *CachingDB) InitTransitionStatus(started, ended bool, baseRoot common.Hash) {
	db.CurrentTransitionState = &TransitionState{
		Ended:   ended,
		Started: started,
		// TODO add other fields when we handle mid-transition interrupts
	}
	db.baseRoot = baseRoot
}

func (db *CachingDB) EndVerkleTransition() {
	if !db.CurrentTransitionState.Started {
		db.CurrentTransitionState.Started = true
	}

	db.CurrentTransitionState.Ended = true
}

type TransitionState struct {
	CurrentAccountAddress *common.Address // addresss of the last translated account
	CurrentSlotHash       common.Hash     // hash of the last translated storage slot
	CurrentPreimageOffset int64           // next byte to read from the preimage file
	Started, Ended        bool

	// Mark whether the storage for an account has been processed. This is useful if the
	// maximum number of leaves of the conversion is reached before the whole storage is
	// processed.
	StorageProcessed bool
}

func (ts *TransitionState) Copy() *TransitionState {
	ret := &TransitionState{
		Started:               ts.Started,
		Ended:                 ts.Ended,
		CurrentSlotHash:       ts.CurrentSlotHash,
		CurrentPreimageOffset: ts.CurrentPreimageOffset,
		StorageProcessed:      ts.StorageProcessed,
	}

	if ts.CurrentAccountAddress != nil {
		ret.CurrentAccountAddress = &common.Address{}
		copy(ret.CurrentAccountAddress[:], ts.CurrentAccountAddress[:])
	}

	return ret
}

// NewDatabaseForTesting is similar to NewDatabase, but it initializes the caching
// db by using an ephemeral memory db with default config for testing.
func NewDatabaseForTesting() *CachingDB {
	return NewDatabase(triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil), nil)
}

// Reader returns a state reader associated with the specified state root.
func (db *CachingDB) Reader(stateRoot common.Hash) (Reader, error) {
	var readers []StateReader

	// Set up the state snapshot reader if available. This feature
	// is optional and may be partially useful if it's not fully
	// generated.
	if db.snap != nil {
		// If standalone state snapshot is available (hash scheme),
		// then construct the legacy snap reader.
		snap := db.snap.Snapshot(stateRoot)
		if snap != nil {
			readers = append(readers, newFlatReader(snap))
		}
	} else {
		// If standalone state snapshot is not available, try to construct
		// the state reader with database.
		reader, err := db.triedb.StateReader(stateRoot)
		if err == nil {
			readers = append(readers, newFlatReader(reader)) // state reader is optional
		}
	}
	// Set up the trie reader, which is expected to always be available
	// as the gatekeeper unless the state is corrupted.
	tr, err := newTrieReader(stateRoot, db.triedb, db.pointCache, db.InTransition(), db.Transitioned())
	if err != nil {
		return nil, err
	}
	readers = append(readers, tr)

	combined, err := newMultiStateReader(readers...)
	if err != nil {
		return nil, err
	}
	return newReader(newCachingCodeReader(db.disk, db.codeCache, db.codeSizeCache), combined), nil
}

func (db *CachingDB) openMPTTrie(root common.Hash) (Trie, error) {
	tr, err := trie.NewStateTrie(trie.StateTrieID(root), db.triedb)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// OpenTrie opens the main account trie at a specific root hash.
func (db *CachingDB) OpenTrie(root common.Hash) (Trie, error) {
	if db.InTransition() || db.Transitioned() {
		// NOTE this is a kaustinen-only change, it will break replay
		vkt, err := trie.NewVerkleTrie(root, db.triedb, db.addrToPoint)
		if err != nil {
			log.Error("failed to open the vkt", "err", err)
			return nil, err
		}

		// If the verkle conversion has ended, return a single
		// verkle trie.
		if db.CurrentTransitionState.Ended {
			log.Debug("transition ended, returning a simple verkle tree")
			return vkt, nil
		}

		// Otherwise, return a transition trie, with a base MPT
		// trie and an overlay, verkle trie.
		mpt, err := db.openMPTTrie(db.baseRoot)
		if err != nil {
			log.Error("failed to open the mpt", "err", err, "root", db.baseRoot)
			return nil, err
		}

		return trie.NewTransitionTree(mpt.(*trie.SecureTrie), vkt, false), nil
	}

	log.Info("not in transition, opening mpt alone", "root", root)
	return db.openMPTTrie(root)
}

// OpenStorageTrie opens the storage trie of an account.
func (db *CachingDB) OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, self Trie) (Trie, error) {
	// In the verkle case, there is only one tree. But the two-tree structure
	// is hardcoded in the codebase. So we need to return the same trie in this
	// case.
	if db.triedb.IsVerkle() {
		return self, nil
	}
	tr, err := trie.NewStateTrie(trie.StorageTrieID(stateRoot, crypto.Keccak256Hash(address.Bytes()), root), db.triedb)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// ContractCodeWithPrefix retrieves a particular contract's code. If the
// code can't be found in the cache, then check the existence with **new**
// db scheme.
func (db *CachingDB) ContractCodeWithPrefix(address common.Address, codeHash common.Hash) []byte {
	code, _ := db.codeCache.Get(codeHash)
	if len(code) > 0 {
		return code
	}
	code = rawdb.ReadCodeWithPrefix(db.disk, codeHash)
	if len(code) > 0 {
		db.codeCache.Add(codeHash, code)
		db.codeSizeCache.Add(codeHash, len(code))
	}
	return code
}

// TrieDB retrieves any intermediate trie-node caching layer.
func (db *CachingDB) TrieDB() *triedb.Database {
	return db.triedb
}

// PointCache returns the cache of evaluated curve points.
func (db *CachingDB) PointCache() *utils.PointCache {
	return db.pointCache
}

// Snapshot returns the underlying state snapshot.
func (db *CachingDB) Snapshot() *snapshot.Tree {
	return db.snap
}

// mustCopyTrie returns a deep-copied trie.
func mustCopyTrie(t Trie) Trie {
	switch t := t.(type) {
	case *trie.StateTrie:
		return t.Copy()
	case *trie.VerkleTrie:
		return t.Copy()
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}

func (db *CachingDB) GetTreeKeyHeader(addr []byte) *verkle.Point {
	return db.addrToPoint.Get(addr)
}

func (db *CachingDB) SetCurrentAccountAddress(addr common.Address) {
	db.CurrentTransitionState.CurrentAccountAddress = &addr
}

func (db *CachingDB) GetCurrentAccountHash() common.Hash {
	var addrHash common.Hash
	if db.CurrentTransitionState.CurrentAccountAddress != nil {
		addrHash = crypto.Keccak256Hash(db.CurrentTransitionState.CurrentAccountAddress[:])
	}
	return addrHash
}

func (db *CachingDB) GetCurrentAccountAddress() *common.Address {
	return db.CurrentTransitionState.CurrentAccountAddress
}

func (db *CachingDB) GetCurrentPreimageOffset() int64 {
	return db.CurrentTransitionState.CurrentPreimageOffset
}

func (db *CachingDB) SetCurrentPreimageOffset(offset int64) {
	db.CurrentTransitionState.CurrentPreimageOffset = offset
}

func (db *CachingDB) SetCurrentSlotHash(hash common.Hash) {
	db.CurrentTransitionState.CurrentSlotHash = hash
}

func (db *CachingDB) GetCurrentSlotHash() common.Hash {
	return db.CurrentTransitionState.CurrentSlotHash
}

func (db *CachingDB) SetStorageProcessed(processed bool) {
	db.CurrentTransitionState.StorageProcessed = processed
}

func (db *CachingDB) GetStorageProcessed() bool {
	return db.CurrentTransitionState.StorageProcessed
}

func (db *CachingDB) AddRootTranslation(originalRoot, translatedRoot common.Hash) {
}

func (db *CachingDB) SetLastMerkleRoot(merkleRoot common.Hash) {
	db.baseRoot = merkleRoot
}

func (db *CachingDB) SaveTransitionState(root common.Hash) {
	db.transitionStateLock.Lock()
	defer db.transitionStateLock.Unlock()
	if db.CurrentTransitionState != nil {
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		err := enc.Encode(db.CurrentTransitionState)
		if err != nil {
			log.Error("failed to encode transition state", "err", err)
			return
		}

		if !db.TransitionStatePerRoot.Contains(root) {
			// Copy so that the address pointer isn't updated after
			// it has been saved.
			db.TransitionStatePerRoot.Add(root, db.CurrentTransitionState.Copy())

			rawdb.WriteVerkleTransitionState(db.TrieDB().Disk(), root, buf.Bytes())
		}

		log.Debug("saving transition state", "storage processed", db.CurrentTransitionState.StorageProcessed, "addr", db.CurrentTransitionState.CurrentAccountAddress, "slot hash", db.CurrentTransitionState.CurrentSlotHash, "root", root, "ended", db.CurrentTransitionState.Ended, "started", db.CurrentTransitionState.Started)
	}
}

func (db *CachingDB) LoadTransitionState(root common.Hash) {
	db.transitionStateLock.Lock()
	defer db.transitionStateLock.Unlock()
	// Try to get the transition state from the cache and
	// the DB if it's not there.
	ts, ok := db.TransitionStatePerRoot.Get(root)
	if !ok {
		// Not in the cache, try getting it from the DB
		data, err := rawdb.ReadVerkleTransitionState(db.TrieDB().Disk(), root)
		if err != nil {
			log.Error("failed to read transition state", "err", err)
			return
		}

		// if a state could be read from the db, attempt to decode it
		if len(data) > 0 {
			var (
				newts TransitionState
				buf   = bytes.NewBuffer(data[:])
				dec   = gob.NewDecoder(buf)
			)
			// Decode transition state
			err = dec.Decode(&newts)
			if err != nil {
				log.Error("failed to decode transition state", "err", err)
				return
			}
			ts = &newts
		}

		// Fallback that should only happen before the transition
		if ts == nil {
			// Initialize the first transition state, with the "ended"
			// field set to true if the database was created
			// as a verkle database.
			log.Debug("no transition state found, starting fresh", "is verkle", db.triedb.IsVerkle())
			// Start with a fresh state
			ts = &TransitionState{Ended: db.triedb.IsVerkle()}
		}
	}

	// Copy so that the CurrentAddress pointer in the map
	// doesn't get overwritten.
	db.CurrentTransitionState = ts.Copy()

	log.Debug("loaded transition state", "storage processed", db.CurrentTransitionState.StorageProcessed, "addr", db.CurrentTransitionState.CurrentAccountAddress, "slot hash", db.CurrentTransitionState.CurrentSlotHash, "root", root, "ended", db.CurrentTransitionState.Ended, "started", db.CurrentTransitionState.Started)
}

func (db *CachingDB) LockCurrentTransitionState() {
	db.transitionStateLock.Lock()
}

func (db *CachingDB) UnLockCurrentTransitionState() {
	db.transitionStateLock.Unlock()
}
