// Copyright 2014 The go-ethereum Authors
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

// Package state provides a caching layer atop the Ethereum state trie.
package state

import (
	"errors"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/triestate"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
	"golang.org/x/sync/errgroup"
)

// TriesInMemory represents the number of layers that are kept in RAM.
const TriesInMemory = 128

type revision struct {
	id           int
	journalIndex int
}

type mutationType int

const (
	update mutationType = iota
	deletion
)

type mutation struct {
	typ     mutationType
	applied bool
}

func (m *mutation) copy() *mutation {
	return &mutation{typ: m.typ, applied: m.applied}
}

func (m *mutation) isDelete() bool {
	return m.typ == deletion
}

// StateDB structs within the ethereum protocol are used to store anything
// within the merkle trie. StateDBs take care of caching and storing
// nested states. It's the general query interface to retrieve:
//
// * Contracts
// * Accounts
//
// Once the state is committed, tries cached in stateDB (including account
// trie, storage tries) will no longer be functional. A new state instance
// must be created with new root and updated database for accessing post-
// commit states.
type StateDB struct {
	db         Database
	prefetcher *triePrefetcher
	trie       Trie
	hasher     crypto.KeccakState
	logger     *tracing.Hooks
	snaps      *snapshot.Tree    // Nil if snapshot is not available
	snap       snapshot.Snapshot // Nil if snapshot is not available

	// originalRoot is the pre-state root, before any changes were made.
	// It will be updated when the Commit is called.
	originalRoot common.Hash

	// This map holds 'live' objects, which will get modified while
	// processing a state transition.
	stateObjects map[common.Address]*stateObject

	// This map holds 'deleted' objects. An object with the same address
	// might also occur in the 'stateObjects' map due to account
	// resurrection. The account value is tracked as the original value
	// before the transition. This map is populated at the transaction
	// boundaries.
	stateObjectsDestruct map[common.Address]*types.StateAccount

	// This map tracks the account mutations that occurred during the
	// transition. Uncommitted mutations belonging to the same account
	// can be merged into a single one which is equivalent from database's
	// perspective. This map is populated at the transaction boundaries.
	mutations map[common.Address]*mutation

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be
	// returned by StateDB.Commit. Notably, this error is also shared
	// by all cached state objects in case the database failure occurs
	// when accessing state of accounts.
	dbErr error

	// The refund counter, also used by state transitioning.
	refund uint64

	// The tx context and all occurred logs in the scope of transaction.
	thash   common.Hash
	txIndex int
	logs    map[common.Hash][]*types.Log
	logSize uint

	// Preimages occurred seen by VM in the scope of block.
	preimages map[common.Hash][]byte

	// Per-transaction access list
	accessList *accessList

	// Transient storage
	transientStorage transientStorage

	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	journal        *journal
	validRevisions []revision
	nextRevisionId int

	// Measurements gathered during execution for debugging purposes
	AccountReads         time.Duration
	AccountHashes        time.Duration
	AccountUpdates       time.Duration
	AccountCommits       time.Duration
	StorageReads         time.Duration
	StorageUpdates       time.Duration
	StorageCommits       time.Duration
	SnapshotAccountReads time.Duration
	SnapshotStorageReads time.Duration
	SnapshotCommits      time.Duration
	TrieDBCommits        time.Duration

	AccountUpdated int
	StorageUpdated atomic.Int64
	AccountDeleted int
	StorageDeleted atomic.Int64
}

// New creates a new state from a given trie.
func New(root common.Hash, db Database, snaps *snapshot.Tree) (*StateDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}
	sdb := &StateDB{
		db:                   db,
		trie:                 tr,
		originalRoot:         root,
		snaps:                snaps,
		stateObjects:         make(map[common.Address]*stateObject),
		stateObjectsDestruct: make(map[common.Address]*types.StateAccount),
		mutations:            make(map[common.Address]*mutation),
		logs:                 make(map[common.Hash][]*types.Log),
		preimages:            make(map[common.Hash][]byte),
		journal:              newJournal(),
		accessList:           newAccessList(),
		transientStorage:     newTransientStorage(),
		hasher:               crypto.NewKeccakState(),
	}
	if sdb.snaps != nil {
		sdb.snap = sdb.snaps.Snapshot(root)
	}
	return sdb, nil
}

// SetLogger sets the logger for account update hooks.
func (s *StateDB) SetLogger(l *tracing.Hooks) {
	s.logger = l
}

// StartPrefetcher initializes a new trie prefetcher to pull in nodes from the
// state trie concurrently while the state is mutated so that when we reach the
// commit phase, most of the needed data is already hot.
func (s *StateDB) StartPrefetcher(namespace string, noreads bool) {
	if s.prefetcher != nil {
		s.prefetcher.terminate(false)
		s.prefetcher.report()
		s.prefetcher = nil
	}
	if s.snap != nil {
		s.prefetcher = newTriePrefetcher(s.db, s.originalRoot, namespace, noreads)

		// With the switch to the Proof-of-Stake consensus algorithm, block production
		// rewards are now handled at the consensus layer. Consequently, a block may
		// have no state transitions if it contains no transactions and no withdrawals.
		// In such cases, the account trie won't be scheduled for prefetching, leading
		// to unnecessary error logs.
		//
		// To prevent this, the account trie is always scheduled for prefetching once
		// the prefetcher is constructed. For more details, see:
		// https://github.com/ethereum/go-ethereum/issues/29880
		if err := s.prefetcher.prefetch(common.Hash{}, s.originalRoot, common.Address{}, nil, false); err != nil {
			log.Error("Failed to prefetch account trie", "root", s.originalRoot, "err", err)
		}
	}
}

// StopPrefetcher terminates a running prefetcher and reports any leftover stats
// from the gathered metrics.
func (s *StateDB) StopPrefetcher() {
	if s.prefetcher != nil {
		s.prefetcher.terminate(false)
		s.prefetcher.report()
		s.prefetcher = nil
	}
}

// setError remembers the first non-nil error it is called with.
func (s *StateDB) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

// Error returns the memorized database failure occurred earlier.
func (s *StateDB) Error() error {
	return s.dbErr
}

func (s *StateDB) AddLog(log *types.Log) {
	s.journal.append(addLogChange{txhash: s.thash})

	log.TxHash = s.thash
	log.TxIndex = uint(s.txIndex)
	log.Index = s.logSize
	if s.logger != nil && s.logger.OnLog != nil {
		s.logger.OnLog(log)
	}
	s.logs[s.thash] = append(s.logs[s.thash], log)
	s.logSize++
}

// GetLogs returns the logs matching the specified transaction hash, and annotates
// them with the given blockNumber and blockHash.
func (s *StateDB) GetLogs(hash common.Hash, blockNumber uint64, blockHash common.Hash) []*types.Log {
	logs := s.logs[hash]
	for _, l := range logs {
		l.BlockNumber = blockNumber
		l.BlockHash = blockHash
	}
	return logs
}

func (s *StateDB) Logs() []*types.Log {
	var logs []*types.Log
	for _, lgs := range s.logs {
		logs = append(logs, lgs...)
	}
	return logs
}

// AddPreimage records a SHA3 preimage seen by the VM.
func (s *StateDB) AddPreimage(hash common.Hash, preimage []byte) {
	if _, ok := s.preimages[hash]; !ok {
		s.journal.append(addPreimageChange{hash: hash})
		s.preimages[hash] = slices.Clone(preimage)
	}
}

// Preimages returns a list of SHA3 preimages that have been submitted.
func (s *StateDB) Preimages() map[common.Hash][]byte {
	return s.preimages
}

// AddRefund adds gas to the refund counter
func (s *StateDB) AddRefund(gas uint64) {
	s.journal.append(refundChange{prev: s.refund})
	s.refund += gas
}

// SubRefund removes gas from the refund counter.
// This method will panic if the refund counter goes below zero
func (s *StateDB) SubRefund(gas uint64) {
	s.journal.append(refundChange{prev: s.refund})
	if gas > s.refund {
		panic(fmt.Sprintf("Refund counter below zero (gas: %d > refund: %d)", gas, s.refund))
	}
	s.refund -= gas
}

// Exist reports whether the given account address exists in the state.
// Notably this also returns true for self-destructed accounts.
func (s *StateDB) Exist(addr common.Address) bool {
	return s.getStateObject(addr) != nil
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (s *StateDB) Empty(addr common.Address) bool {
	so := s.getStateObject(addr)
	return so == nil || so.empty()
}

// GetBalance retrieves the balance from the given address or 0 if object not found
func (s *StateDB) GetBalance(addr common.Address) *uint256.Int {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Balance()
	}
	return common.U2560
}

// GetNonce retrieves the nonce from the given address or 0 if object not found
func (s *StateDB) GetNonce(addr common.Address) uint64 {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}

	return 0
}

// GetStorageRoot retrieves the storage root from the given address or empty
// if object not found.
func (s *StateDB) GetStorageRoot(addr common.Address) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Root()
	}
	return common.Hash{}
}

// TxIndex returns the current transaction index set by SetTxContext.
func (s *StateDB) TxIndex() int {
	return s.txIndex
}

func (s *StateDB) GetCode(addr common.Address) []byte {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Code()
	}
	return nil
}

func (s *StateDB) GetCodeSize(addr common.Address) int {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.CodeSize()
	}
	return 0
}

func (s *StateDB) GetCodeHash(addr common.Address) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return common.BytesToHash(stateObject.CodeHash())
	}
	return common.Hash{}
}

// GetState retrieves the value associated with the specific key.
func (s *StateDB) GetState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetState(hash)
	}
	return common.Hash{}
}

// GetCommittedState retrieves the value associated with the specific key
// without any mutations caused in the current execution.
func (s *StateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetCommittedState(hash)
	}
	return common.Hash{}
}

// Database retrieves the low level database supporting the lower level trie ops.
func (s *StateDB) Database() Database {
	return s.db
}

func (s *StateDB) HasSelfDestructed(addr common.Address) bool {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.selfDestructed
	}
	return false
}

/*
 * SETTERS
 */

// AddBalance adds amount to the account associated with addr.
func (s *StateDB) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(amount, reason)
	}
}

// SubBalance subtracts amount from the account associated with addr.
func (s *StateDB) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(amount, reason)
	}
}

func (s *StateDB) SetBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetBalance(amount, reason)
	}
}

func (s *StateDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (s *StateDB) SetCode(addr common.Address, code []byte) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(crypto.Keccak256Hash(code), code)
	}
}

func (s *StateDB) SetState(addr common.Address, key, value common.Hash) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(key, value)
	}
}

// SetStorage replaces the entire storage for the specified account with given
// storage. This function should only be used for debugging and the mutations
// must be discarded afterwards.
func (s *StateDB) SetStorage(addr common.Address, storage map[common.Hash]common.Hash) {
	// SetStorage needs to wipe existing storage. We achieve this by pretending
	// that the account self-destructed earlier in this block, by flagging
	// it in stateObjectsDestruct. The effect of doing so is that storage lookups
	// will not hit disk, since it is assumed that the disk-data is belonging
	// to a previous incarnation of the object.
	//
	// TODO(rjl493456442) this function should only be supported by 'unwritable'
	// state and all mutations made should all be discarded afterwards.
	if _, ok := s.stateObjectsDestruct[addr]; !ok {
		s.stateObjectsDestruct[addr] = nil
	}
	stateObject := s.getOrNewStateObject(addr)
	for k, v := range storage {
		stateObject.SetState(k, v)
	}
}

// SelfDestruct marks the given account as selfdestructed.
// This clears the account balance.
//
// The account's state object is still available until the state is committed,
// getStateObject will return a non-nil account after SelfDestruct.
func (s *StateDB) SelfDestruct(addr common.Address) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		return
	}
	var (
		prev = new(uint256.Int).Set(stateObject.Balance())
		n    = new(uint256.Int)
	)
	s.journal.append(selfDestructChange{
		account:     &addr,
		prev:        stateObject.selfDestructed,
		prevbalance: prev,
	})
	if s.logger != nil && s.logger.OnBalanceChange != nil && prev.Sign() > 0 {
		s.logger.OnBalanceChange(addr, prev.ToBig(), n.ToBig(), tracing.BalanceDecreaseSelfdestruct)
	}
	stateObject.markSelfdestructed()
	stateObject.data.Balance = n
}

func (s *StateDB) Selfdestruct6780(addr common.Address) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		return
	}
	if stateObject.newContract {
		s.SelfDestruct(addr)
	}
}

// SetTransientState sets transient storage for a given account. It
// adds the change to the journal so that it can be rolled back
// to its previous value if there is a revert.
func (s *StateDB) SetTransientState(addr common.Address, key, value common.Hash) {
	prev := s.GetTransientState(addr, key)
	if prev == value {
		return
	}
	s.journal.append(transientStorageChange{
		account:  &addr,
		key:      key,
		prevalue: prev,
	})
	s.setTransientState(addr, key, value)
}

// setTransientState is a lower level setter for transient storage. It
// is called during a revert to prevent modifications to the journal.
func (s *StateDB) setTransientState(addr common.Address, key, value common.Hash) {
	s.transientStorage.Set(addr, key, value)
}

// GetTransientState gets transient storage for a given account.
func (s *StateDB) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return s.transientStorage.Get(addr, key)
}

//
// Setting, updating & deleting state object methods.
//

// updateStateObject writes the given object to the trie.
func (s *StateDB) updateStateObject(obj *stateObject) {
	// Encode the account and update the account trie
	addr := obj.Address()
	if err := s.trie.UpdateAccount(addr, &obj.data); err != nil {
		s.setError(fmt.Errorf("updateStateObject (%x) error: %v", addr[:], err))
	}
	if obj.dirtyCode {
		s.trie.UpdateContractCode(obj.Address(), common.BytesToHash(obj.CodeHash()), obj.code)
	}
}

// deleteStateObject removes the given object from the state trie.
func (s *StateDB) deleteStateObject(addr common.Address) {
	if err := s.trie.DeleteAccount(addr); err != nil {
		s.setError(fmt.Errorf("deleteStateObject (%x) error: %v", addr[:], err))
	}
}

// getStateObject retrieves a state object given by the address, returning nil if
// the object is not found or was deleted in this execution context.
func (s *StateDB) getStateObject(addr common.Address) *stateObject {
	// Prefer live objects if any is available
	if obj := s.stateObjects[addr]; obj != nil {
		return obj
	}
	// Short circuit if the account is already destructed in this block.
	if _, ok := s.stateObjectsDestruct[addr]; ok {
		return nil
	}
	// If no live objects are available, attempt to use snapshots
	var data *types.StateAccount
	if s.snap != nil {
		start := time.Now()
		acc, err := s.snap.Account(crypto.HashData(s.hasher, addr.Bytes()))
		s.SnapshotAccountReads += time.Since(start)

		if err == nil {
			if acc == nil {
				return nil
			}
			data = &types.StateAccount{
				Nonce:    acc.Nonce,
				Balance:  acc.Balance,
				CodeHash: acc.CodeHash,
				Root:     common.BytesToHash(acc.Root),
			}
			if len(data.CodeHash) == 0 {
				data.CodeHash = types.EmptyCodeHash.Bytes()
			}
			if data.Root == (common.Hash{}) {
				data.Root = types.EmptyRootHash
			}
		}
	}
	// If snapshot unavailable or reading from it failed, load from the database
	if data == nil {
		start := time.Now()
		var err error
		data, err = s.trie.GetAccount(addr)
		s.AccountReads += time.Since(start)

		if err != nil {
			s.setError(fmt.Errorf("getDeleteStateObject (%x) error: %w", addr.Bytes(), err))
			return nil
		}
		if data == nil {
			return nil
		}
	}
	// Independent of where we loaded the data from, add it to the prefetcher.
	// Whilst this would be a bit weird if snapshots are disabled, but we still
	// want the trie nodes to end up in the prefetcher too, so just push through.
	if s.prefetcher != nil {
		if err := s.prefetcher.prefetch(common.Hash{}, s.originalRoot, common.Address{}, [][]byte{addr[:]}, true); err != nil {
			log.Error("Failed to prefetch account", "addr", addr, "err", err)
		}
	}
	// Insert into the live set
	obj := newObject(s, addr, data)
	s.setStateObject(obj)
	return obj
}

func (s *StateDB) setStateObject(object *stateObject) {
	s.stateObjects[object.Address()] = object
}

// getOrNewStateObject retrieves a state object or create a new state object if nil.
func (s *StateDB) getOrNewStateObject(addr common.Address) *stateObject {
	obj := s.getStateObject(addr)
	if obj == nil {
		obj = s.createObject(addr)
	}
	return obj
}

// createObject creates a new state object. The assumption is held there is no
// existing account with the given address, otherwise it will be silently overwritten.
func (s *StateDB) createObject(addr common.Address) *stateObject {
	obj := newObject(s, addr, nil)
	s.journal.append(createObjectChange{account: &addr})
	s.setStateObject(obj)
	return obj
}

// CreateAccount explicitly creates a new state object, assuming that the
// account did not previously exist in the state. If the account already
// exists, this function will silently overwrite it which might lead to a
// consensus bug eventually.
func (s *StateDB) CreateAccount(addr common.Address) {
	s.createObject(addr)
}

// CreateContract is used whenever a contract is created. This may be preceded
// by CreateAccount, but that is not required if it already existed in the
// state due to funds sent beforehand.
// This operation sets the 'newContract'-flag, which is required in order to
// correctly handle EIP-6780 'delete-in-same-transaction' logic.
func (s *StateDB) CreateContract(addr common.Address) {
	obj := s.getStateObject(addr)
	if !obj.newContract {
		obj.newContract = true
		s.journal.append(createContractChange{account: addr})
	}
}

// Copy creates a deep, independent copy of the state.
// Snapshots of the copied state cannot be applied to the copy.
func (s *StateDB) Copy() *StateDB {
	// Copy all the basic fields, initialize the memory ones
	state := &StateDB{
		db:                   s.db,
		trie:                 s.db.CopyTrie(s.trie),
		hasher:               crypto.NewKeccakState(),
		originalRoot:         s.originalRoot,
		stateObjects:         make(map[common.Address]*stateObject, len(s.stateObjects)),
		stateObjectsDestruct: maps.Clone(s.stateObjectsDestruct),
		mutations:            make(map[common.Address]*mutation, len(s.mutations)),
		dbErr:                s.dbErr,
		refund:               s.refund,
		thash:                s.thash,
		txIndex:              s.txIndex,
		logs:                 make(map[common.Hash][]*types.Log, len(s.logs)),
		logSize:              s.logSize,
		preimages:            maps.Clone(s.preimages),
		journal:              s.journal.copy(),
		validRevisions:       slices.Clone(s.validRevisions),
		nextRevisionId:       s.nextRevisionId,

		// In order for the block producer to be able to use and make additions
		// to the snapshot tree, we need to copy that as well. Otherwise, any
		// block mined by ourselves will cause gaps in the tree, and force the
		// miner to operate trie-backed only.
		snaps: s.snaps,
		snap:  s.snap,
	}
	// Deep copy cached state objects.
	for addr, obj := range s.stateObjects {
		state.stateObjects[addr] = obj.deepCopy(state)
	}
	// Deep copy the object state markers.
	for addr, op := range s.mutations {
		state.mutations[addr] = op.copy()
	}
	// Deep copy the logs occurred in the scope of block
	for hash, logs := range s.logs {
		cpy := make([]*types.Log, len(logs))
		for i, l := range logs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		state.logs[hash] = cpy
	}
	// Do we need to copy the access list and transient storage?
	// In practice: No. At the start of a transaction, these two lists are empty.
	// In practice, we only ever copy state _between_ transactions/blocks, never
	// in the middle of a transaction. However, it doesn't cost us much to copy
	// empty lists, so we do it anyway to not blow up if we ever decide copy them
	// in the middle of a transaction.
	state.accessList = s.accessList.Copy()
	state.transientStorage = s.transientStorage.Copy()
	return state
}

// Snapshot returns an identifier for the current revision of the state.
func (s *StateDB) Snapshot() int {
	id := s.nextRevisionId
	s.nextRevisionId++
	s.validRevisions = append(s.validRevisions, revision{id, s.journal.length()})
	return id
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (s *StateDB) RevertToSnapshot(revid int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(s.validRevisions), func(i int) bool {
		return s.validRevisions[i].id >= revid
	})
	if idx == len(s.validRevisions) || s.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := s.validRevisions[idx].journalIndex

	// Replay the journal to undo changes and remove invalidated snapshots
	s.journal.revert(s, snapshot)
	s.validRevisions = s.validRevisions[:idx]
}

// GetRefund returns the current value of the refund counter.
func (s *StateDB) GetRefund() uint64 {
	return s.refund
}

// Finalise finalises the state by removing the destructed objects and clears
// the journal as well as the refunds. Finalise, however, will not push any updates
// into the tries just yet. Only IntermediateRoot or Commit will do that.
func (s *StateDB) Finalise(deleteEmptyObjects bool) {
	addressesToPrefetch := make([][]byte, 0, len(s.journal.dirties))
	for addr := range s.journal.dirties {
		obj, exist := s.stateObjects[addr]
		if !exist {
			// ripeMD is 'touched' at block 1714175, in tx 0x1237f737031e40bcde4a8b7e717b2d15e3ecadfe49bb1bbc71ee9deb09c6fcf2
			// That tx goes out of gas, and although the notion of 'touched' does not exist there, the
			// touch-event will still be recorded in the journal. Since ripeMD is a special snowflake,
			// it will persist in the journal even though the journal is reverted. In this special circumstance,
			// it may exist in `s.journal.dirties` but not in `s.stateObjects`.
			// Thus, we can safely ignore it here
			continue
		}
		if obj.selfDestructed || (deleteEmptyObjects && obj.empty()) {
			delete(s.stateObjects, obj.address)
			s.markDelete(addr)

			// If ether was sent to account post-selfdestruct it is burnt.
			if bal := obj.Balance(); s.logger != nil && s.logger.OnBalanceChange != nil && obj.selfDestructed && bal.Sign() != 0 {
				s.logger.OnBalanceChange(obj.address, bal.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestructBurn)
			}
			// We need to maintain account deletions explicitly (will remain
			// set indefinitely). Note only the first occurred self-destruct
			// event is tracked.
			if _, ok := s.stateObjectsDestruct[obj.address]; !ok {
				s.stateObjectsDestruct[obj.address] = obj.origin
			}
		} else {
			obj.finalise()
			s.markUpdate(addr)
		}
		// At this point, also ship the address off to the precacher. The precacher
		// will start loading tries, and when the change is eventually committed,
		// the commit-phase will be a lot faster
		addressesToPrefetch = append(addressesToPrefetch, common.CopyBytes(addr[:])) // Copy needed for closure
	}
	if s.prefetcher != nil && len(addressesToPrefetch) > 0 {
		if err := s.prefetcher.prefetch(common.Hash{}, s.originalRoot, common.Address{}, addressesToPrefetch, false); err != nil {
			log.Error("Failed to prefetch addresses", "addresses", len(addressesToPrefetch), "err", err)
		}
	}
	// Invalidate journal because reverting across transactions is not allowed.
	s.clearJournalAndRefund()
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (s *StateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	// Finalise all the dirty storage states and write them into the tries
	s.Finalise(deleteEmptyObjects)

	// If there was a trie prefetcher operating, terminate it async so that the
	// individual storage tries can be updated as soon as the disk load finishes.
	if s.prefetcher != nil {
		s.prefetcher.terminate(true)
		defer func() {
			s.prefetcher.report()
			s.prefetcher = nil // Pre-byzantium, unset any used up prefetcher
		}()
	}
	// Process all storage updates concurrently. The state object update root
	// method will internally call a blocking trie fetch from the prefetcher,
	// so there's no need to explicitly wait for the prefetchers to finish.
	var (
		start   = time.Now()
		workers errgroup.Group
	)
	if s.db.TrieDB().IsVerkle() {
		// Whilst MPT storage tries are independent, Verkle has one single trie
		// for all the accounts and all the storage slots merged together. The
		// former can thus be simply parallelized, but updating the latter will
		// need concurrency support within the trie itself. That's a TODO for a
		// later time.
		workers.SetLimit(1)
	}
	for addr, op := range s.mutations {
		if op.applied || op.isDelete() {
			continue
		}
		obj := s.stateObjects[addr] // closure for the task runner below
		workers.Go(func() error {
			obj.updateRoot()
			return nil
		})
	}
	workers.Wait()
	s.StorageUpdates += time.Since(start)

	// Now we're about to start to write changes to the trie. The trie is so far
	// _untouched_. We can check with the prefetcher, if it can give us a trie
	// which has the same root, but also has some content loaded into it.
	start = time.Now()

	if s.prefetcher != nil {
		if trie := s.prefetcher.trie(common.Hash{}, s.originalRoot); trie == nil {
			log.Error("Failed to retrieve account pre-fetcher trie")
		} else {
			s.trie = trie
		}
	}
	// Perform updates before deletions.  This prevents resolution of unnecessary trie nodes
	// in circumstances similar to the following:
	//
	// Consider nodes `A` and `B` who share the same full node parent `P` and have no other siblings.
	// During the execution of a block:
	// - `A` self-destructs,
	// - `C` is created, and also shares the parent `P`.
	// If the self-destruct is handled first, then `P` would be left with only one child, thus collapsed
	// into a shortnode. This requires `B` to be resolved from disk.
	// Whereas if the created node is handled first, then the collapse is avoided, and `B` is not resolved.
	var (
		usedAddrs    [][]byte
		deletedAddrs []common.Address
	)
	for addr, op := range s.mutations {
		if op.applied {
			continue
		}
		op.applied = true

		if op.isDelete() {
			deletedAddrs = append(deletedAddrs, addr)
		} else {
			s.updateStateObject(s.stateObjects[addr])
			s.AccountUpdated += 1
		}
		usedAddrs = append(usedAddrs, common.CopyBytes(addr[:])) // Copy needed for closure
	}
	for _, deletedAddr := range deletedAddrs {
		s.deleteStateObject(deletedAddr)
		s.AccountDeleted += 1
	}
	s.AccountUpdates += time.Since(start)

	if s.prefetcher != nil {
		s.prefetcher.used(common.Hash{}, s.originalRoot, usedAddrs)
	}
	// Track the amount of time wasted on hashing the account trie
	defer func(start time.Time) { s.AccountHashes += time.Since(start) }(time.Now())

	return s.trie.Hash()
}

// SetTxContext sets the current transaction hash and index which are
// used when the EVM emits new state logs. It should be invoked before
// transaction execution.
func (s *StateDB) SetTxContext(thash common.Hash, ti int) {
	s.thash = thash
	s.txIndex = ti
}

func (s *StateDB) clearJournalAndRefund() {
	if len(s.journal.entries) > 0 {
		s.journal = newJournal()
		s.refund = 0
	}
	s.validRevisions = s.validRevisions[:0] // Snapshots can be created without journal entries
}

// fastDeleteStorage is the function that efficiently deletes the storage trie
// of a specific account. It leverages the associated state snapshot for fast
// storage iteration and constructs trie node deletion markers by creating
// stack trie with iterated slots.
func (s *StateDB) fastDeleteStorage(addrHash common.Hash, root common.Hash) (common.StorageSize, map[common.Hash][]byte, *trienode.NodeSet, error) {
	iter, err := s.snaps.StorageIterator(s.originalRoot, addrHash, common.Hash{})
	if err != nil {
		return 0, nil, nil, err
	}
	defer iter.Release()

	var (
		size  common.StorageSize
		nodes = trienode.NewNodeSet(addrHash)
		slots = make(map[common.Hash][]byte)
	)
	stack := trie.NewStackTrie(func(path []byte, hash common.Hash, blob []byte) {
		nodes.AddNode(path, trienode.NewDeleted())
		size += common.StorageSize(len(path))
	})
	for iter.Next() {
		slot := common.CopyBytes(iter.Slot())
		if err := iter.Error(); err != nil { // error might occur after Slot function
			return 0, nil, nil, err
		}
		size += common.StorageSize(common.HashLength + len(slot))
		slots[iter.Hash()] = slot

		if err := stack.Update(iter.Hash().Bytes(), slot); err != nil {
			return 0, nil, nil, err
		}
	}
	if err := iter.Error(); err != nil { // error might occur during iteration
		return 0, nil, nil, err
	}
	if stack.Hash() != root {
		return 0, nil, nil, fmt.Errorf("snapshot is not matched, exp %x, got %x", root, stack.Hash())
	}
	return size, slots, nodes, nil
}

// slowDeleteStorage serves as a less-efficient alternative to "fastDeleteStorage,"
// employed when the associated state snapshot is not available. It iterates the
// storage slots along with all internal trie nodes via trie directly.
func (s *StateDB) slowDeleteStorage(addr common.Address, addrHash common.Hash, root common.Hash) (common.StorageSize, map[common.Hash][]byte, *trienode.NodeSet, error) {
	tr, err := s.db.OpenStorageTrie(s.originalRoot, addr, root, s.trie)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("failed to open storage trie, err: %w", err)
	}
	it, err := tr.NodeIterator(nil)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("failed to open storage iterator, err: %w", err)
	}
	var (
		size  common.StorageSize
		nodes = trienode.NewNodeSet(addrHash)
		slots = make(map[common.Hash][]byte)
	)
	for it.Next(true) {
		if it.Leaf() {
			slots[common.BytesToHash(it.LeafKey())] = common.CopyBytes(it.LeafBlob())
			size += common.StorageSize(common.HashLength + len(it.LeafBlob()))
			continue
		}
		if it.Hash() == (common.Hash{}) {
			continue
		}
		size += common.StorageSize(len(it.Path()))
		nodes.AddNode(it.Path(), trienode.NewDeleted())
	}
	if err := it.Error(); err != nil {
		return 0, nil, nil, err
	}
	return size, slots, nodes, nil
}

// deleteStorage is designed to delete the storage trie of a designated account.
// The function will make an attempt to utilize an efficient strategy if the
// associated state snapshot is reachable; otherwise, it will resort to a less
// efficient approach.
func (s *StateDB) deleteStorage(addr common.Address, addrHash common.Hash, root common.Hash) (map[common.Hash][]byte, *trienode.NodeSet, error) {
	var (
		start = time.Now()
		err   error
		size  common.StorageSize
		slots map[common.Hash][]byte
		nodes *trienode.NodeSet
	)
	// The fast approach can be failed if the snapshot is not fully
	// generated, or it's internally corrupted. Fallback to the slow
	// one just in case.
	if s.snap != nil {
		size, slots, nodes, err = s.fastDeleteStorage(addrHash, root)
	}
	if s.snap == nil || err != nil {
		size, slots, nodes, err = s.slowDeleteStorage(addr, addrHash, root)
	}
	if err != nil {
		return nil, nil, err
	}
	// Report the metrics
	n := int64(len(slots))

	slotDeletionMaxCount.UpdateIfGt(int64(len(slots)))
	slotDeletionMaxSize.UpdateIfGt(int64(size))

	slotDeletionTimer.UpdateSince(start)
	slotDeletionCount.Mark(n)
	slotDeletionSize.Mark(int64(size))

	return slots, nodes, nil
}

// handleDestruction processes all destruction markers and deletes the account
// and associated storage slots if necessary. There are four potential scenarios
// as following:
//
//	(a) the account was not existent and be marked as destructed
//	(b) the account was not existent and be marked as destructed,
//	    however, it's resurrected later in the same block.
//	(c) the account was existent and be marked as destructed
//	(d) the account was existent and be marked as destructed,
//	    however it's resurrected later in the same block.
//
// In case (a), nothing needs be deleted, nil to nil transition can be ignored.
// In case (b), nothing needs be deleted, nil is used as the original value for
// newly created account and storages
// In case (c), **original** account along with its storages should be deleted,
// with their values be tracked as original value.
// In case (d), **original** account along with its storages should be deleted,
// with their values be tracked as original value.
func (s *StateDB) handleDestruction() (map[common.Hash]*accountDelete, []*trienode.NodeSet, error) {
	var (
		nodes   []*trienode.NodeSet
		buf     = crypto.NewKeccakState()
		deletes = make(map[common.Hash]*accountDelete)
	)
	for addr, prev := range s.stateObjectsDestruct {
		// The account was non-existent, and it's marked as destructed in the scope
		// of block. It can be either case (a) or (b) and will be interpreted as
		// null->null state transition.
		// - for (a), skip it without doing anything
		// - for (b), the resurrected account with nil as original will be handled afterwards
		if prev == nil {
			continue
		}
		// The account was existent, it can be either case (c) or (d).
		addrHash := crypto.HashData(buf, addr.Bytes())
		op := &accountDelete{
			address: addr,
			origin:  types.SlimAccountRLP(*prev),
		}
		deletes[addrHash] = op

		// Short circuit if the origin storage was empty.
		if prev.Root == types.EmptyRootHash {
			continue
		}
		// Remove storage slots belonging to the account.
		slots, set, err := s.deleteStorage(addr, addrHash, prev.Root)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete storage, err: %w", err)
		}
		op.storagesOrigin = slots

		// Aggregate the associated trie node changes.
		nodes = append(nodes, set)
	}
	return deletes, nodes, nil
}

// GetTrie returns the account trie.
func (s *StateDB) GetTrie() Trie {
	return s.trie
}

// commit gathers the state mutations accumulated along with the associated
// trie changes, resetting all internal flags with the new state as the base.
func (s *StateDB) commit(deleteEmptyObjects bool) (*stateUpdate, error) {
	// Short circuit in case any database failure occurred earlier.
	if s.dbErr != nil {
		return nil, fmt.Errorf("commit aborted due to earlier error: %v", s.dbErr)
	}
	// Finalize any pending changes and merge everything into the tries
	s.IntermediateRoot(deleteEmptyObjects)

	// Commit objects to the trie, measuring the elapsed time
	var (
		accountTrieNodesUpdated int
		accountTrieNodesDeleted int
		storageTrieNodesUpdated int
		storageTrieNodesDeleted int

		lock    sync.Mutex                                               // protect two maps below
		nodes   = trienode.NewMergedNodeSet()                            // aggregated trie nodes
		updates = make(map[common.Hash]*accountUpdate, len(s.mutations)) // aggregated account updates

		// merge aggregates the dirty trie nodes into the global set.
		//
		// Given that some accounts may be destroyed and then recreated within
		// the same block, it's possible that a node set with the same owner
		// may already exists. In such cases, these two sets are combined, with
		// the later one overwriting the previous one if any nodes are modified
		// or deleted in both sets.
		//
		// merge run concurrently across  all the state objects and account trie.
		merge = func(set *trienode.NodeSet) error {
			if set == nil {
				return nil
			}
			lock.Lock()
			defer lock.Unlock()

			updates, deletes := set.Size()
			if set.Owner == (common.Hash{}) {
				accountTrieNodesUpdated += updates
				accountTrieNodesDeleted += deletes
			} else {
				storageTrieNodesUpdated += updates
				storageTrieNodesDeleted += deletes
			}
			return nodes.Merge(set)
		}
	)
	// Given that some accounts could be destroyed and then recreated within
	// the same block, account deletions must be processed first. This ensures
	// that the storage trie nodes deleted during destruction and recreated
	// during subsequent resurrection can be combined correctly.
	deletes, delNodes, err := s.handleDestruction()
	if err != nil {
		return nil, err
	}
	for _, set := range delNodes {
		if err := merge(set); err != nil {
			return nil, err
		}
	}
	// Handle all state updates afterwards, concurrently to one another to shave
	// off some milliseconds from the commit operation. Also accumulate the code
	// writes to run in parallel with the computations.
	var (
		start   = time.Now()
		root    common.Hash
		workers errgroup.Group
	)
	// Schedule the account trie first since that will be the biggest, so give
	// it the most time to crunch.
	//
	// TODO(karalabe): This account trie commit is *very* heavy. 5-6ms at chain
	// heads, which seems excessive given that it doesn't do hashing, it just
	// shuffles some data. For comparison, the *hashing* at chain head is 2-3ms.
	// We need to investigate what's happening as it seems something's wonky.
	// Obviously it's not an end of the world issue, just something the original
	// code didn't anticipate for.
	workers.Go(func() error {
		// Write the account trie changes, measuring the amount of wasted time
		newroot, set := s.trie.Commit(true)
		root = newroot

		if err := merge(set); err != nil {
			return err
		}
		s.AccountCommits = time.Since(start)
		return nil
	})
	// Schedule each of the storage tries that need to be updated, so they can
	// run concurrently to one another.
	//
	// TODO(karalabe): Experimentally, the account commit takes approximately the
	// same time as all the storage commits combined, so we could maybe only have
	// 2 threads in total. But that kind of depends on the account commit being
	// more expensive than it should be, so let's fix that and revisit this todo.
	for addr, op := range s.mutations {
		if op.isDelete() {
			continue
		}
		// Write any contract code associated with the state object
		obj := s.stateObjects[addr]
		if obj == nil {
			return nil, errors.New("missing state object")
		}
		// Run the storage updates concurrently to one another
		workers.Go(func() error {
			// Write any storage changes in the state object to its storage trie
			update, set, err := obj.commit()
			if err != nil {
				return err
			}
			if err := merge(set); err != nil {
				return err
			}
			lock.Lock()
			updates[obj.addrHash] = update
			s.StorageCommits = time.Since(start) // overwrite with the longest storage commit runtime
			lock.Unlock()
			return nil
		})
	}
	// Wait for everything to finish and update the metrics
	if err := workers.Wait(); err != nil {
		return nil, err
	}
	accountUpdatedMeter.Mark(int64(s.AccountUpdated))
	storageUpdatedMeter.Mark(s.StorageUpdated.Load())
	accountDeletedMeter.Mark(int64(s.AccountDeleted))
	storageDeletedMeter.Mark(s.StorageDeleted.Load())
	accountTrieUpdatedMeter.Mark(int64(accountTrieNodesUpdated))
	accountTrieDeletedMeter.Mark(int64(accountTrieNodesDeleted))
	storageTriesUpdatedMeter.Mark(int64(storageTrieNodesUpdated))
	storageTriesDeletedMeter.Mark(int64(storageTrieNodesDeleted))
	s.AccountUpdated, s.AccountDeleted = 0, 0
	s.StorageUpdated.Store(0)
	s.StorageDeleted.Store(0)

	// Clear all internal flags and update state root at the end.
	s.mutations = make(map[common.Address]*mutation)
	s.stateObjectsDestruct = make(map[common.Address]*types.StateAccount)

	origin := s.originalRoot
	s.originalRoot = root
	return newStateUpdate(origin, root, deletes, updates, nodes), nil
}

// commitAndFlush is a wrapper of commit which also commits the state mutations
// to the configured data stores.
func (s *StateDB) commitAndFlush(block uint64, deleteEmptyObjects bool) (*stateUpdate, error) {
	ret, err := s.commit(deleteEmptyObjects)
	if err != nil {
		return nil, err
	}
	// Commit dirty contract code if any exists
	if db := s.db.DiskDB(); db != nil && len(ret.codes) > 0 {
		batch := db.NewBatch()
		for _, code := range ret.codes {
			rawdb.WriteCode(batch, code.hash, code.blob)
		}
		if err := batch.Write(); err != nil {
			return nil, err
		}
	}
	if !ret.empty() {
		// If snapshotting is enabled, update the snapshot tree with this new version
		if s.snap != nil {
			s.snap = nil

			start := time.Now()
			if err := s.snaps.Update(ret.root, ret.originRoot, ret.destructs, ret.accounts, ret.storages); err != nil {
				log.Warn("Failed to update snapshot tree", "from", ret.originRoot, "to", ret.root, "err", err)
			}
			// Keep 128 diff layers in the memory, persistent layer is 129th.
			// - head layer is paired with HEAD state
			// - head-1 layer is paired with HEAD-1 state
			// - head-127 layer(bottom-most diff layer) is paired with HEAD-127 state
			if err := s.snaps.Cap(ret.root, TriesInMemory); err != nil {
				log.Warn("Failed to cap snapshot tree", "root", ret.root, "layers", TriesInMemory, "err", err)
			}
			s.SnapshotCommits += time.Since(start)
		}
		// If trie database is enabled, commit the state update as a new layer
		if db := s.db.TrieDB(); db != nil {
			start := time.Now()
			set := triestate.New(ret.accountsOrigin, ret.storagesOrigin)
			if err := db.Update(ret.root, ret.originRoot, block, ret.nodes, set); err != nil {
				return nil, err
			}
			s.TrieDBCommits += time.Since(start)
		}
	}
	return ret, err
}

// Commit writes the state mutations into the configured data stores.
//
// Once the state is committed, tries cached in stateDB (including account
// trie, storage tries) will no longer be functional. A new state instance
// must be created with new root and updated database for accessing post-
// commit states.
//
// The associated block number of the state transition is also provided
// for more chain context.
func (s *StateDB) Commit(block uint64, deleteEmptyObjects bool) (common.Hash, error) {
	ret, err := s.commitAndFlush(block, deleteEmptyObjects)
	if err != nil {
		return common.Hash{}, err
	}
	return ret.root, nil
}

// Prepare handles the preparatory steps for executing a state transition with.
// This method must be invoked before state transition.
//
// Berlin fork:
// - Add sender to access list (2929)
// - Add destination to access list (2929)
// - Add precompiles to access list (2929)
// - Add the contents of the optional tx access list (2930)
//
// Potential EIPs:
// - Reset access list (Berlin)
// - Add coinbase to access list (EIP-3651)
// - Reset transient storage (EIP-1153)
func (s *StateDB) Prepare(rules params.Rules, sender, coinbase common.Address, dst *common.Address, precompiles []common.Address, list types.AccessList) {
	if rules.IsEIP2929 && rules.IsEIP4762 {
		panic("eip2929 and eip4762 are both activated")
	}
	if rules.IsEIP2929 {
		// Clear out any leftover from previous executions
		al := newAccessList()
		s.accessList = al

		al.AddAddress(sender)
		if dst != nil {
			al.AddAddress(*dst)
			// If it's a create-tx, the destination will be added inside evm.create
		}
		for _, addr := range precompiles {
			al.AddAddress(addr)
		}
		for _, el := range list {
			al.AddAddress(el.Address)
			for _, key := range el.StorageKeys {
				al.AddSlot(el.Address, key)
			}
		}
		if rules.IsShanghai { // EIP-3651: warm coinbase
			al.AddAddress(coinbase)
		}
	}
	// Reset transient storage at the beginning of transaction execution
	s.transientStorage = newTransientStorage()
}

// AddAddressToAccessList adds the given address to the access list
func (s *StateDB) AddAddressToAccessList(addr common.Address) {
	if s.accessList.AddAddress(addr) {
		s.journal.append(accessListAddAccountChange{&addr})
	}
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (s *StateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	addrMod, slotMod := s.accessList.AddSlot(addr, slot)
	if addrMod {
		// In practice, this should not happen, since there is no way to enter the
		// scope of 'address' without having the 'address' become already added
		// to the access list (via call-variant, create, etc).
		// Better safe than sorry, though
		s.journal.append(accessListAddAccountChange{&addr})
	}
	if slotMod {
		s.journal.append(accessListAddSlotChange{
			address: &addr,
			slot:    &slot,
		})
	}
}

// AddressInAccessList returns true if the given address is in the access list.
func (s *StateDB) AddressInAccessList(addr common.Address) bool {
	return s.accessList.ContainsAddress(addr)
}

// SlotInAccessList returns true if the given (address, slot)-tuple is in the access list.
func (s *StateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	return s.accessList.Contains(addr, slot)
}

// markDelete is invoked when an account is deleted but the deletion is
// not yet committed. The pending mutation is cached and will be applied
// all together
func (s *StateDB) markDelete(addr common.Address) {
	if _, ok := s.mutations[addr]; !ok {
		s.mutations[addr] = &mutation{}
	}
	s.mutations[addr].applied = false
	s.mutations[addr].typ = deletion
}

func (s *StateDB) markUpdate(addr common.Address) {
	if _, ok := s.mutations[addr]; !ok {
		s.mutations[addr] = &mutation{}
	}
	s.mutations[addr].applied = false
	s.mutations[addr].typ = update
}

func (s *StateDB) PointCache() *utils.PointCache {
	return s.db.PointCache()
}
