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
	"slices"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/holiman/uint256"
	"golang.org/x/sync/errgroup"
)

// TriesInMemory represents the number of layers that are kept in RAM.
const TriesInMemory = 128

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
	db     Database
	reader Reader
	hasher Hasher

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
	stateObjectsDestruct map[common.Address]*stateObject

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
	accessList   *accessList
	accessEvents *AccessEvents

	// Transient storage
	transientStorage transientStorage

	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	journal *journal

	// State witness if cross validation is needed
	witness *stateless.Witness

	// Measurements gathered during execution for debugging purposes
	Stats
}

// New creates a new state from a given trie.
func New(root common.Hash, db Database) (*StateDB, error) {
	reader, err := db.Reader(root)
	if err != nil {
		return nil, err
	}
	return NewWithReader(root, db, reader)
}

// NewWithReader creates a new state for the specified state root. Unlike New,
// this function accepts an additional Reader which is bound to the given root.
func NewWithReader(root common.Hash, db Database, reader Reader) (*StateDB, error) {
	hasher, err := db.Hasher(root)
	if err != nil {
		return nil, err
	}
	sdb := &StateDB{
		db:                   db,
		originalRoot:         root,
		reader:               reader,
		hasher:               hasher,
		stateObjects:         make(map[common.Address]*stateObject),
		stateObjectsDestruct: make(map[common.Address]*stateObject),
		mutations:            make(map[common.Address]*mutation),
		logs:                 make(map[common.Hash][]*types.Log),
		preimages:            make(map[common.Hash][]byte),
		journal:              newJournal(),
		accessList:           newAccessList(),
		transientStorage:     newTransientStorage(),
	}
	if db.TrieDB().IsVerkle() {
		sdb.accessEvents = NewAccessEvents()
	}
	return sdb, nil
}

// TraceWitness enables execution witness gathering.
func (s *StateDB) TraceWitness(witness *stateless.Witness) {
	s.witness = witness
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
	s.journal.logChange(s.thash)

	log.TxHash = s.thash
	log.TxIndex = uint(s.txIndex)
	log.Index = s.logSize
	s.logs[s.thash] = append(s.logs[s.thash], log)
	s.logSize++
}

// GetLogs returns the logs matching the specified transaction hash, and annotates
// them with the given block attributes.
func (s *StateDB) GetLogs(hash common.Hash, blockNumber uint64, blockHash common.Hash, blockTime uint64) []*types.Log {
	logs := s.logs[hash]
	for _, l := range logs {
		l.BlockNumber = blockNumber
		l.BlockHash = blockHash
		l.BlockTimestamp = blockTime
	}
	return logs
}

// Logs returns the un-annotated logs in order.
func (s *StateDB) Logs() []*types.Log {
	logs := make([]*types.Log, 0, s.logSize)
	for _, lgs := range s.logs {
		logs = append(logs, lgs...)
	}
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Index < logs[j].Index
	})
	return logs
}

// AddPreimage records a SHA3 preimage seen by the VM.
func (s *StateDB) AddPreimage(hash common.Hash, preimage []byte) {
	if _, ok := s.preimages[hash]; !ok {
		s.preimages[hash] = slices.Clone(preimage)
	}
}

// Preimages returns a list of SHA3 preimages that have been submitted.
func (s *StateDB) Preimages() map[common.Hash][]byte {
	return s.preimages
}

// AddRefund adds gas to the refund counter
func (s *StateDB) AddRefund(gas uint64) {
	s.journal.refundChange(s.refund)
	s.refund += gas
}

// SubRefund removes gas from the refund counter.
// This method will panic if the refund counter goes below zero
func (s *StateDB) SubRefund(gas uint64) {
	s.journal.refundChange(s.refund)
	if gas > s.refund {
		panic(fmt.Sprintf("Refund counter below zero (gas: %d > refund: %d)", gas, s.refund))
	}
	s.refund -= gas
}

// Exist reports whether the given account address exists in the state.
// Notably this also returns true for self-destructed accounts within the current transaction.
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

// TxIndex returns the current transaction index set by SetTxContext.
func (s *StateDB) TxIndex() int {
	return s.txIndex
}

func (s *StateDB) GetCode(addr common.Address) []byte {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		if s.witness != nil {
			s.witness.AddCode(stateObject.Code())
		}
		return stateObject.Code()
	}
	return nil
}

func (s *StateDB) GetCodeSize(addr common.Address) int {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		if s.witness != nil {
			s.witness.AddCode(stateObject.Code())
		}
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

// GetStateAndCommittedState returns the current value and the original value.
func (s *StateDB) GetStateAndCommittedState(addr common.Address, hash common.Hash) (common.Hash, common.Hash) {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.getState(hash)
	}
	return common.Hash{}, common.Hash{}
}

// Database retrieves the low level database supporting the lower level trie ops.
func (s *StateDB) Database() Database {
	return s.db
}

// Reader retrieves the low level database reader supporting the
// lower level operations.
func (s *StateDB) Reader() Reader {
	return s.reader
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
func (s *StateDB) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject == nil {
		return uint256.Int{}
	}
	return stateObject.AddBalance(amount)
}

// SubBalance subtracts amount from the account associated with addr.
func (s *StateDB) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject == nil {
		return uint256.Int{}
	}
	if amount.IsZero() {
		return *(stateObject.Balance())
	}
	return stateObject.SetBalance(new(uint256.Int).Sub(stateObject.Balance(), amount))
}

func (s *StateDB) SetBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetBalance(amount)
	}
}

func (s *StateDB) SetNonce(addr common.Address, nonce uint64, reason tracing.NonceChangeReason) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (s *StateDB) SetCode(addr common.Address, code []byte, reason tracing.CodeChangeReason) (prev []byte) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		return stateObject.SetCode(crypto.Keccak256Hash(code), code)
	}
	return nil
}

func (s *StateDB) SetState(addr common.Address, key, value common.Hash) common.Hash {
	if stateObject := s.getOrNewStateObject(addr); stateObject != nil {
		return stateObject.SetState(key, value)
	}
	return common.Hash{}
}

// SetStorage replaces the entire storage for the specified account with given
// storage. This function should only be used for debugging and the mutations
// must be discarded afterwards.
func (s *StateDB) SetStorage(addr common.Address, storage map[common.Hash]common.Hash) {
	// SetStorage needs to wipe the existing storage. We achieve this by marking
	// the account as self-destructed in this block. The effect is that storage
	// lookups will not hit the disk, as it is assumed that the disk data belongs
	// to a previous incarnation of the object.
	//
	// TODO (rjl493456442): This function should only be supported by 'unwritable'
	// state, and all mutations made should be discarded afterward.
	obj := s.getStateObject(addr)
	if obj != nil {
		if _, ok := s.stateObjectsDestruct[addr]; !ok {
			s.stateObjectsDestruct[addr] = obj
		}
	}
	newObj := s.createObject(addr)
	for k, v := range storage {
		newObj.SetState(k, v)
	}
	// Inherit the metadata of original object if it was existent
	if obj != nil {
		newObj.SetCode(common.BytesToHash(obj.CodeHash()), obj.code)
		newObj.SetNonce(obj.Nonce())
		newObj.SetBalance(obj.Balance())
	}
}

// SelfDestruct marks the given account as selfdestructed.
//
// The account's state object is still available until the state is committed,
// getStateObject will return a non-nil account after SelfDestruct.
func (s *StateDB) SelfDestruct(addr common.Address) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		return
	}
	// If it is already marked as self-destructed, we do not need to add it
	// for journalling a second time.
	if !stateObject.selfDestructed {
		s.journal.destruct(addr)
		stateObject.markSelfdestructed()
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
	s.journal.transientStateChange(addr, key, prev)
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
	start := time.Now()
	acct, err := s.reader.Account(addr)
	if err != nil {
		s.setError(fmt.Errorf("getStateObject (%x) error: %w", addr.Bytes(), err))
		return nil
	}
	s.AccountLoaded++
	s.AccountReads += time.Since(start)

	// Short circuit if the account is not found
	if acct == nil {
		return nil
	}
	// Insert into the live set
	obj := newObject(s, addr, acct)
	s.setStateObject(obj)

	// Schedule the resolved account for prefetching if it's enabled.
	prefetcher, ok := s.hasher.(Prefetcher)
	if ok {
		prefetcher.PrefetchAccount([]common.Address{addr}, true)
	}
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
	s.journal.createObject(addr)
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
		s.journal.createContract(addr)
	}
}

// IsNewContract reports whether the contract at the given address was deployed
// during the current transaction.
func (s *StateDB) IsNewContract(addr common.Address) bool {
	obj := s.getStateObject(addr)
	if obj == nil {
		return false
	}
	return obj.newContract
}

// Copy creates a deep, independent copy of the state.
// Snapshots of the copied state cannot be applied to the copy.
func (s *StateDB) Copy() *StateDB {
	// Copy all the basic fields, initialize the memory ones
	state := &StateDB{
		db:                   s.db,
		reader:               s.reader,
		hasher:               s.hasher.Copy(),
		originalRoot:         s.originalRoot,
		stateObjects:         make(map[common.Address]*stateObject, len(s.stateObjects)),
		stateObjectsDestruct: make(map[common.Address]*stateObject, len(s.stateObjectsDestruct)),
		mutations:            make(map[common.Address]*mutation, len(s.mutations)),
		dbErr:                s.dbErr,
		refund:               s.refund,
		thash:                s.thash,
		txIndex:              s.txIndex,
		logs:                 make(map[common.Hash][]*types.Log, len(s.logs)),
		logSize:              s.logSize,
		preimages:            maps.Clone(s.preimages),

		// Do we need to copy the access list and transient storage?
		// In practice: No. At the start of a transaction, these two lists are empty.
		// In practice, we only ever copy state _between_ transactions/blocks, never
		// in the middle of a transaction. However, it doesn't cost us much to copy
		// empty lists, so we do it anyway to not blow up if we ever decide copy them
		// in the middle of a transaction.
		accessList:       s.accessList.Copy(),
		transientStorage: s.transientStorage.Copy(),
		journal:          s.journal.copy(),
	}
	if s.witness != nil {
		state.witness = s.witness.Copy()
	}
	if s.accessEvents != nil {
		state.accessEvents = s.accessEvents.Copy()
	}
	// Deep copy cached state objects.
	for addr, obj := range s.stateObjects {
		state.stateObjects[addr] = obj.deepCopy(state)
	}
	// Deep copy destructed state objects.
	for addr, obj := range s.stateObjectsDestruct {
		state.stateObjectsDestruct[addr] = obj.deepCopy(state)
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
	return state
}

// Snapshot returns an identifier for the current revision of the state.
func (s *StateDB) Snapshot() int {
	return s.journal.snapshot()
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (s *StateDB) RevertToSnapshot(revid int) {
	s.journal.revertToSnapshot(revid, s)
}

// GetRefund returns the current value of the refund counter.
func (s *StateDB) GetRefund() uint64 {
	return s.refund
}

type removedAccountWithBalance struct {
	address common.Address
	balance *uint256.Int
}

// LogsForBurnAccounts returns the eth burn logs for accounts scheduled for
// removal which still have positive balance. The purpose of this function is
// to handle a corner case of EIP-7708 where a self-destructed account might
// still receive funds between sending/burning its previous balance and actual
// removal. In this case the burning of these remaining balances still need to
// be logged.
// Specification EIP-7708: https://eips.ethereum.org/EIPS/eip-7708
//
// This function should only be invoked at the transaction boundary, specifically
// before the Finalise.
func (s *StateDB) LogsForBurnAccounts() []*types.Log {
	var list []removedAccountWithBalance
	for addr := range s.journal.dirties {
		if obj, exist := s.stateObjects[addr]; exist && obj.selfDestructed && !obj.Balance().IsZero() {
			list = append(list, removedAccountWithBalance{
				address: obj.address,
				balance: obj.Balance(),
			})
		}
	}
	if list == nil {
		return nil
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].address.Cmp(list[j].address) < 0
	})
	logs := make([]*types.Log, len(list))
	for i, acct := range list {
		logs[i] = types.EthBurnLog(acct.address, acct.balance)
	}
	return logs
}

// Finalise finalises the state by removing the destructed objects and clears
// the journal as well as the refunds. Finalise, however, will not push any updates
// into the tries just yet. Only IntermediateRoot or Commit will do that.
func (s *StateDB) Finalise(deleteEmptyObjects bool) {
	addressesToPrefetch := make([]common.Address, 0, len(s.journal.dirties))
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
			// We need to maintain account deletions explicitly (will remain
			// set indefinitely). Note only the first occurred self-destruct
			// event is tracked.
			if _, ok := s.stateObjectsDestruct[obj.address]; !ok {
				s.stateObjectsDestruct[obj.address] = obj
			}
		} else {
			obj.finalise()
			s.markUpdate(addr)
		}
		// At this point, also ship the address off to the prefetcher. The prefetcher
		// will start loading tries, and when the change is eventually committed,
		// the commit-phase will be a lot faster
		addressesToPrefetch = append(addressesToPrefetch, addr)
	}
	// Invalidate journal because reverting across transactions is not allowed.
	s.clearJournalAndRefund()

	prefetcher, ok := s.hasher.(Prefetcher)
	if ok {
		prefetcher.PrefetchAccount(addressesToPrefetch, false)
	}
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (s *StateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	// Finalise all the dirty storage states and write them into the tries
	s.Finalise(deleteEmptyObjects)

	// Pre-process mutations whose preceding deletion has not yet been
	// applied. This happens when an account is deleted and then re-created
	// within the same block and the deletion was overwritten by the update.
	// Notify the hasher of the deletion first so that any cached storage
	// trie is evicted and the re-created account starts with a fresh trie.
	var (
		delAddrs []common.Address
		delAccts []AccountMut
		start    = time.Now()
	)
	for addr, op := range s.mutations {
		if !op.precedingDelete {
			continue
		}
		op.precedingDelete = false

		delAddrs = append(delAddrs, addr)
		delAccts = append(delAccts, AccountMut{Account: nil})
	}
	if len(delAddrs) > 0 {
		if err := s.hasher.UpdateAccount(delAddrs, delAccts); err != nil {
			s.setError(err)
			return common.Hash{}
		}
		s.AccountDeleted += len(delAddrs)
	}
	s.AccountUpdates += time.Since(start)

	// Process all storage updates concurrently, flushing them to hasher.
	start = time.Now()
	var workers errgroup.Group
	for addr, op := range s.mutations {
		if op.applied || op.isDelete() {
			continue
		}
		obj := s.stateObjects[addr]
		workers.Go(obj.updateTrie)
	}
	if err := workers.Wait(); err != nil {
		s.setError(err)
	}
	s.StorageUpdates += time.Since(start)

	// Process all account updates
	var (
		addresses []common.Address
		accounts  []AccountMut
	)
	start = time.Now()
	for addr, op := range s.mutations {
		if op.applied {
			continue
		}
		op.applied = true
		addresses = append(addresses, addr)

		if op.isDelete() {
			accounts = append(accounts, AccountMut{Account: nil})
			s.AccountDeleted += 1
			continue
		}
		obj := s.stateObjects[addr]
		mut := AccountMut{Account: &obj.data}
		if obj.dirtyCode {
			mut.Code = &CodeMut{Code: obj.code}

			// Count code writes post-Finalise so reverted CREATEs are excluded.
			s.CodeUpdated += 1
			s.CodeUpdateBytes += len(obj.code)
		}
		accounts = append(accounts, mut)
		s.AccountUpdated += 1
	}
	if err := s.hasher.UpdateAccount(addresses, accounts); err != nil {
		s.setError(err)
		return common.Hash{}
	}
	s.AccountUpdates += time.Since(start)

	// Track the amount of time wasted on hashing the account trie
	defer func(start time.Time) { s.AccountHashes += time.Since(start) }(time.Now())

	return s.hasher.Hash()
}

// SetTxContext sets the current transaction hash and index which are
// used when the EVM emits new state logs. It should be invoked before
// transaction execution.
func (s *StateDB) SetTxContext(thash common.Hash, ti int) {
	s.thash = thash
	s.txIndex = ti
}

func (s *StateDB) clearJournalAndRefund() {
	s.journal.reset()
	s.refund = 0
}

// deleteStorage is designed to delete the storage trie of a designated account.
func (s *StateDB) deleteStorage(addrHash common.Hash) (map[common.Hash]common.Hash, map[common.Hash]common.Hash, *trienode.NodeSet, error) {
	var (
		nodes          = trienode.NewNodeSet(addrHash)     // the set for trie node mutations (value is nil)
		storages       = make(map[common.Hash]common.Hash) // the set for storage mutations (value is nil)
		storageOrigins = make(map[common.Hash]common.Hash) // the set for tracking the original value of slot
	)
	iteratee, err := s.db.Iteratee(s.originalRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	it, err := iteratee.NewStorageIterator(addrHash, common.Hash{})
	if err != nil {
		return nil, nil, nil, err
	}
	defer it.Release()

	stack := trie.NewStackTrie(func(path []byte, hash common.Hash, blob []byte) {
		nodes.AddNode(path, trienode.NewDeletedWithPrev(blob))
	})
	for it.Next() {
		slot := common.CopyBytes(it.Slot())
		if err := it.Error(); err != nil { // error might occur after Slot function
			return nil, nil, nil, err
		}
		key := it.Hash()
		storages[key] = common.Hash{}

		_, content, _, err := rlp.Split(it.Slot())
		if err != nil {
			return nil, nil, nil, err
		}
		var value common.Hash
		value.SetBytes(content)
		storageOrigins[key] = value

		if err := stack.Update(key.Bytes(), slot); err != nil {
			return nil, nil, nil, err
		}
	}
	if err := it.Error(); err != nil { // error might occur during iteration
		return nil, nil, nil, err
	}
	stack.Hash() // Commit the right boundary
	return storages, storageOrigins, nodes, nil
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
func (s *StateDB) handleDestruction(noStorageWiping bool) (map[common.Hash]*accountDelete, *trienode.MergedNodeSet, error) {
	var (
		nodes   = trienode.NewMergedNodeSet()
		deletes = make(map[common.Hash]*accountDelete)
	)
	for addr, prevObj := range s.stateObjectsDestruct {
		prev := prevObj.origin

		// The account was non-existent, and it's marked as destructed in the scope
		// of block. It can be either case (a) or (b) and will be interpreted as
		// null->null state transition.
		// - for (a), skip it without doing anything
		// - for (b), the resurrected account with nil as original will be handled afterwards
		if prev == nil {
			continue
		}
		// The account was existent, it can be either case (c) or (d).
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		op := &accountDelete{
			address: addr,
			origin:  *prev,
		}
		deletes[addrHash] = op

		// Short circuit if the origin storage was empty.
		if s.db.TrieDB().IsVerkle() {
			continue
		}
		if noStorageWiping {
			return nil, nil, fmt.Errorf("unexpected storage wiping, %x", addr)
		}
		// Remove storage slots belonging to the account.
		storages, storagesOrigin, set, err := s.deleteStorage(addrHash)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete storage, err: %w", err)
		}
		op.storages, op.storagesOrigin = storages, storagesOrigin

		// Aggregate the associated trie node changes.
		if err := nodes.Merge(set); err != nil {
			return nil, nil, err
		}
	}
	return deletes, nodes, nil
}

// commit gathers the state mutations accumulated along with the associated
// trie changes, resetting all internal flags with the new state as the base.
func (s *StateDB) commit(deleteEmptyObjects bool, noStorageWiping bool, blockNumber uint64) (*stateUpdate, error) {
	// Short circuit in case any database failure occurred earlier.
	if s.dbErr != nil {
		return nil, fmt.Errorf("commit aborted due to earlier error: %v", s.dbErr)
	}
	// Finalize any pending changes and merge everything into the tries
	root := s.IntermediateRoot(deleteEmptyObjects)

	// Short circuit if any error occurs within the IntermediateRoot.
	if s.dbErr != nil {
		return nil, fmt.Errorf("commit aborted due to database error: %v", s.dbErr)
	}
	// Given that some accounts could be destroyed and then recreated within
	// the same block, account deletions must be processed first. This ensures
	// that the storage trie nodes deleted during destruction and recreated
	// during subsequent resurrection can be combined correctly.
	deletes, nodes, err := s.handleDestruction(noStorageWiping)
	if err != nil {
		return nil, err
	}
	// Aggregated account updates
	updates := make(map[common.Hash]*accountUpdate, len(s.mutations))
	for addr, op := range s.mutations {
		if op.isDelete() {
			continue
		}
		// Write any contract code associated with the state object
		obj := s.stateObjects[addr]
		if obj == nil {
			return nil, errors.New("missing state object")
		}
		update, err := obj.commit()
		if err != nil {
			return nil, err
		}
		updates[obj.addrHash()] = update
	}
	// Handle all state updates afterwards, concurrently to one another to shave
	// off some milliseconds from the commit operation. Also accumulate the code
	// writes to run in parallel with the computations.
	start := time.Now()
	root, set, secondaryHashes, err := s.hasher.Commit()
	if err != nil {
		return nil, err
	}
	s.HasherCommits = time.Since(start)

	if err := nodes.MergeSet(set); err != nil {
		return nil, err
	}
	// Clear all internal flags and update state root at the end.
	s.mutations = make(map[common.Address]*mutation)
	s.stateObjectsDestruct = make(map[common.Address]*stateObject)

	origin := s.originalRoot
	s.originalRoot = root

	if s.witness != nil {
		builder, ok := s.hasher.(WitnessCollector)
		if ok {
			builder.CollectWitness(s.witness)
		}
	}
	return newStateUpdate(noStorageWiping, origin, root, blockNumber, deletes, updates, nodes, secondaryHashes), nil
}

// commitAndFlush is a wrapper of commit which also commits the state mutations
// to the configured data stores.
func (s *StateDB) commitAndFlush(block uint64, deleteEmptyObjects bool, noStorageWiping bool, deriveCodeFields bool) (*stateUpdate, error) {
	ret, err := s.commit(deleteEmptyObjects, noStorageWiping, block)
	if err != nil {
		return nil, err
	}
	if deriveCodeFields {
		if err := ret.deriveCodeFields(s.reader); err != nil {
			return nil, err
		}
	}
	start := time.Now()
	if err := s.db.Commit(ret); err != nil {
		return nil, err
	}
	s.DatabaseCommits = time.Since(start)

	// The reader update must be performed as the final step, otherwise,
	// the new state would not be visible before db.commit.
	s.reader, _ = s.db.Reader(s.originalRoot)
	s.hasher, _ = s.db.Hasher(s.originalRoot)
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
//
// noStorageWiping is a flag indicating whether storage wiping is permitted.
// Since self-destruction was deprecated with the Cancun fork and there are
// no empty accounts left that could be deleted by EIP-158, storage wiping
// should not occur.
func (s *StateDB) Commit(block uint64, deleteEmptyObjects bool, noStorageWiping bool) (common.Hash, error) {
	ret, err := s.commitAndFlush(block, deleteEmptyObjects, noStorageWiping, false)
	if err != nil {
		return common.Hash{}, err
	}
	return ret.root, nil
}

// CommitWithUpdate writes the state mutations and returns the state update for
// external processing (e.g., live tracing hooks or size tracker).
func (s *StateDB) CommitWithUpdate(block uint64, deleteEmptyObjects bool, noStorageWiping bool) (common.Hash, *stateUpdate, error) {
	ret, err := s.commitAndFlush(block, deleteEmptyObjects, noStorageWiping, true)
	if err != nil {
		return common.Hash{}, nil, err
	}
	return ret.root, ret, nil
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
		s.journal.accessListAddAccount(addr)
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
		s.journal.accessListAddAccount(addr)
	}
	if slotMod {
		s.journal.accessListAddSlot(addr, slot)
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

// Witness retrieves the current state witness being collected.
func (s *StateDB) Witness() *stateless.Witness {
	return s.witness
}

func (s *StateDB) AccessEvents() *AccessEvents {
	return s.accessEvents
}
