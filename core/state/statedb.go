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
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	lru "github.com/hashicorp/golang-lru"
)

// The starting nonce determines the default nonce when new accounts are being
// created.
var StartingNonce uint64

const (
	// Number of past tries to keep. The arbitrarily chosen value here
	// is max uncle depth + 1.
	maxJournalLength = 8

	// Number of codehash->size associations to keep.
	codeSizeCacheSize = 100000
)

type intermediateInfo struct {
	txHash, blockHash common.Hash
	txIdx             uint
}

// StateDBs within the ethereum protocol are used to store anything
// within the merkle trie. StateDBs take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Contracts
// * Accounts
type StateDB struct {
	db            ethdb.Database
	trie          *trie.SecureTrie
	pastTries     []*trie.SecureTrie
	codeSizeCache *lru.Cache
	parent        *StateDB

	// This map holds 'live' objects, which will get modified while processing a state transition.
	stateObjects      map[common.Address]*StateObject
	stateObjectsDirty map[common.Address]struct{}

	localStateObjects map[common.Address]bool

	// The refund counter, also used by state transitioning.
	refund *big.Int

	logIdx uint
	logs   []*vm.Log

	interInfo        intermediateInfo
	MarkedTransition bool // marks the transition between transactions

	lock sync.Mutex
}

func (s *StateDB) TransitionState(txHash, blockHash common.Hash, txIdx int) {
	if s.parent != nil {
		s.parent.MarkedTransition = true
	}
	s.interInfo = intermediateInfo{txHash: txHash, blockHash: blockHash, txIdx: uint(txIdx)}
	s.refund = new(big.Int)
}

// Read fetches the state object with the given address by first checking its own cache of state objects. If that didn't result in an object it will attempt to check it's line of ancestors.
func (s *StateDB) Read(address common.Address) (object *StateObject, inCache bool) {
	stateObject := s.stateObjects[address]
	if stateObject == nil && s.parent != nil {
		stateObject, _ = s.parent.Read(address)
		if stateObject != nil {
			s.SetStateObject(stateObject)
		}
	} else if stateObject != nil {
		inCache = true
	}

	if stateObject != nil {
		if stateObject.deleted {
			return nil, false
		}
		return stateObject, inCache
	}

	enc := s.trie.Get(address[:])
	if len(enc) == 0 {
		return nil, false
	}

	var data Account
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		glog.Errorf("can't decode object at %x: %v", address[:], err)
		return nil, false
	}
	// Insert into the live set.
	obj := NewObject(address, data, s.MarkStateObjectDirty)
	s.SetStateObject(obj)
	//s.stateObjects[address] = obj
	return obj, false
}

// Create a new state from a given trie
func New(root common.Hash, db ethdb.Database) (*StateDB, error) {
	tr, err := trie.NewSecure(root, db)
	if err != nil {
		return nil, err
	}
	csc, _ := lru.New(codeSizeCacheSize)
	return &StateDB{
		db:                db,
		trie:              tr,
		codeSizeCache:     csc,
		stateObjects:      make(map[common.Address]*StateObject),
		stateObjectsDirty: make(map[common.Address]struct{}),
		localStateObjects: make(map[common.Address]bool),
		refund:            new(big.Int),
	}, nil
}

// New creates a new statedb by reusing any journalled tries to avoid costly
// disk io.
func (self *StateDB) New(root common.Hash) (*StateDB, error) {
	self.lock.Lock()
	defer self.lock.Unlock()

	tr, err := self.openTrie(root)
	if err != nil {
		return nil, err
	}
	return &StateDB{
		db:                self.db,
		trie:              tr,
		codeSizeCache:     self.codeSizeCache,
		stateObjects:      make(map[common.Address]*StateObject),
		stateObjectsDirty: make(map[common.Address]struct{}),
		localStateObjects: make(map[common.Address]bool),
		refund:            new(big.Int),
	}, nil
}

// Reset clears out all emphemeral state objects from the state db, but keeps
// the underlying state trie to avoid reloading data for the next operations.
func (s *StateDB) Reset(root common.Hash) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	tr, err := s.openTrie(root)
	if err != nil {
		return err
	}
	s.trie = tr
	s.stateObjects = make(map[common.Address]*StateObject)
	s.stateObjectsDirty = make(map[common.Address]struct{})
	s.localStateObjects = make(map[common.Address]bool)
	s.refund = new(big.Int)
	s.logs = nil

	return nil
}

// openTrie creates a trie. It uses an existing trie if one is available
// from the journal if available.
func (self *StateDB) openTrie(root common.Hash) (*trie.SecureTrie, error) {
	for i := len(self.pastTries) - 1; i >= 0; i-- {
		if self.pastTries[i].Hash() == root {
			tr := *self.pastTries[i]
			return &tr, nil
		}
	}
	return trie.NewSecure(root, self.db)
}

func (self *StateDB) pushTrie(t *trie.SecureTrie) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if len(self.pastTries) >= maxJournalLength {
		copy(self.pastTries, self.pastTries[1:])
		self.pastTries[len(self.pastTries)-1] = t
	} else {
		self.pastTries = append(self.pastTries, t)
	}
}

func (s *StateDB) AddLog(log *vm.Log) {
	log.TxIndex = s.interInfo.txIdx
	log.TxHash = s.interInfo.txHash
	log.BlockHash = s.interInfo.blockHash
	log.Index = s.logIdx

	s.logs = append(s.logs, log)

	s.logIdx++
}

// Logs returns the logs of it's entire ancestory chain or until
// the marked transition is found (state between transactions).
func (s *StateDB) Logs() []*vm.Log {
	var logs []*vm.Log
	if s.MarkedTransition {
		return nil
	} else if s.parent != nil {
		logs = s.parent.Logs()
	}
	return append(logs, s.logs...)
}

func (self *StateDB) AddRefund(gas *big.Int) {
	self.refund.Add(self.refund, gas)
}

func (self *StateDB) HasAccount(addr common.Address) bool {
	return self.GetStateObject(addr) != nil
}

func (self *StateDB) Exist(addr common.Address) bool {
	return self.GetStateObject(addr) != nil
}

func (self *StateDB) GetAccount(addr common.Address) vm.Account {
	return self.GetStateObject(addr)
}

// Retrieve the balance from the given address or 0 if object not found
func (self *StateDB) GetBalance(addr common.Address) *big.Int {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.Balance()
	}

	return common.Big0
}

func (self *StateDB) GetNonce(addr common.Address) uint64 {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}

	return StartingNonce
}

func (self *StateDB) GetCode(addr common.Address) []byte {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		code := stateObject.Code(self.db)
		key := common.BytesToHash(stateObject.CodeHash())
		self.codeSizeCache.Add(key, len(code))
		return code
	}
	return nil
}

func (self *StateDB) GetCodeSize(addr common.Address) int {
	stateObject := self.GetStateObject(addr)
	if stateObject == nil {
		return 0
	}
	key := common.BytesToHash(stateObject.CodeHash())
	if cached, ok := self.codeSizeCache.Get(key); ok {
		return cached.(int)
	}
	size := len(stateObject.Code(self.db))
	if stateObject.dbErr == nil {
		self.codeSizeCache.Add(key, size)
	}
	return size
}

func (self *StateDB) GetCodeHash(addr common.Address) common.Hash {
	stateObject := self.GetStateObject(addr)
	if stateObject == nil {
		return common.Hash{}
	}
	return common.BytesToHash(stateObject.CodeHash())
}

func (self *StateDB) GetState(a common.Address, b common.Hash) common.Hash {
	stateObject := self.GetStateObject(a)
	if stateObject != nil {
		return stateObject.GetState(self.db, b)
	}
	return common.Hash{}
}

func (self *StateDB) IsDeleted(addr common.Address) bool {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.remove
	}
	return false
}

/*
 * SETTERS
 */

func (self *StateDB) AddBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(amount)
	}
}

func (self *StateDB) SubBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(amount)
	}
}

func (self *StateDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (self *StateDB) SetCode(addr common.Address, code []byte) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(crypto.Keccak256Hash(code), code)
	}
}

func (self *StateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(key, value)
	}
}

func (self *StateDB) Delete(addr common.Address) bool {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		stateObject.MarkForDeletion()
		stateObject.data.Balance = new(big.Int)
		return true
	}

	return false
}

//
// Setting, updating & deleting state object methods
//

// Update the given state object and apply it to state trie
func (self *StateDB) UpdateStateObject(stateObject *StateObject) {
	addr := stateObject.Address()
	data, err := rlp.EncodeToBytes(stateObject)
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
	}
	self.trie.Update(addr[:], data)
}

// Delete the given state object and delete it from the state trie
func (self *StateDB) DeleteStateObject(stateObject *StateObject) {
	stateObject.deleted = true

	addr := stateObject.Address()
	self.trie.Delete(addr[:])
}

func (self *StateDB) SetStateObject(object *StateObject) {
	if object == nil {
		panic("is nil")
	}
	self.stateObjects[object.Address()] = object
}

func (s *StateDB) SetOwnedStateObject(address common.Address, object *StateObject) {
	if object == nil {
		panic("is nil")
	}
	s.stateObjects[address] = object
	s.localStateObjects[address] = true
}

func (s *StateDB) GetOrNewStateObject(address common.Address) *StateObject {
	stateObject, _ := s.Read(address)
	if stateObject != nil {
		if !s.localStateObjects[address] {
			stateObject = stateObject.Copy(s.db, s.MarkStateObjectDirty)
			s.SetOwnedStateObject(address, stateObject)
			//s.stateObjects[address] = stateObject
			//s.localStateObjects[address] = true
		}
		return stateObject
	}

	if stateObject == nil || stateObject.deleted {
		stateObject = NewObject(address, Account{}, s.MarkStateObjectDirty)
		stateObject.SetNonce(StartingNonce)

		s.SetOwnedStateObject(address, stateObject)
		//s.stateObjects[address] = stateObject
		//s.localStateObjects[address] = true
	}
	return stateObject
}

func (s *StateDB) GetStateObject(address common.Address) *StateObject {
	account, _ := s.Read(address)
	if account != nil {
		if s.stateObjects[address] == nil {
			s.SetStateObject(account)
			//s.stateObjects[address] = account
		}
		return account
	}
	return nil
}

// NewStateObject create a state object whether it exist in the trie or not
func (self *StateDB) newStateObject(addr common.Address) *StateObject {
	if glog.V(logger.Core) {
		glog.Infof("(+) %x\n", addr)
	}
	obj := NewObject(addr, Account{}, self.MarkStateObjectDirty)
	obj.SetNonce(StartingNonce) // sets the object to dirty
	//self.stateObjects[addr] = obj
	self.SetOwnedStateObject(addr, obj)
	return obj
}

// MarkStateObjectDirty adds the specified object to the dirty map to avoid costly
// state object cache iteration to find a handful of modified ones.
func (self *StateDB) MarkStateObjectDirty(addr common.Address) {
	self.stateObjectsDirty[addr] = struct{}{}
}

// Creates creates a new state object and takes ownership.
func (self *StateDB) CreateStateObject(addr common.Address) *StateObject {
	// Get previous (if any)
	so := self.GetStateObject(addr)
	// Create a new one
	newSo := self.newStateObject(addr)

	// If it existed set the balance to the new account
	if so != nil {
		newSo.data.Balance = so.data.Balance
	}

	return newSo
}

func (self *StateDB) CreateAccount(addr common.Address) vm.Account {
	return self.CreateStateObject(addr)
}

func (self *StateDB) Set(state *StateDB) {
	self.lock.Lock()
	defer self.lock.Unlock()

	*self = *state
	/*
		self.db = state.db
		self.trie = state.trie
		self.pastTries = state.pastTries
		self.stateObjects = state.stateObjects
		self.stateObjectsDirty = state.stateObjectsDirty
		self.codeSizeCache = state.codeSizeCache
		self.refund = state.refund
		self.logs = state.logs
	*/
}

func (self *StateDB) GetRefund() *big.Int {
	return self.refund
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (s *StateDB) IntermediateRoot() common.Hash {
	s.refund = new(big.Int)
	for addr, _ := range s.stateObjectsDirty {
		stateObject := s.stateObjects[addr]
		if stateObject.remove {
			s.DeleteStateObject(stateObject)
		} else {
			stateObject.UpdateRoot(s.db)
			s.UpdateStateObject(stateObject)
		}
	}
	return s.trie.Hash()
}

// DeleteSuicides flags the suicided objects for deletion so that it
// won't be referenced again when called / queried up on.
//
// DeleteSuicides should not be used for consensus related updates
// under any circumstances.
func (s *StateDB) DeleteSuicides() {
	// Reset refund so that any used-gas calculations can use
	// this method.
	s.refund = new(big.Int)
	for addr, _ := range s.stateObjectsDirty {
		stateObject := s.stateObjects[addr]

		// If the object has been removed by a suicide
		// flag the object as deleted.
		if stateObject.remove {
			stateObject.deleted = true
		}
		delete(s.stateObjectsDirty, addr)
	}
}

/*
// commit commits all state changes to the database.
func (s *StateDB) Commit() (root common.Hash, err error) {
	root, batch := s.CommitBatch()
	return root, batch.Write()
}

// CommitBatch commits all state changes to a write batch but does not
// execute the batch. It is used to validate state changes against
// the root hash stored in a block.
func (s *StateDB) CommitBatch() (root common.Hash, batch ethdb.Batch) {
	batch = s.db.NewBatch()
	root, _ = s.commit(batch)
	return root, batch
}

func (s *StateDB) commit(dbw trie.DatabaseWriter) (root common.Hash, err error) {
	s.refund = new(big.Int)

	// Commit objects to the trie.
	for addr, stateObject := range s.stateObjects {
		if stateObject.remove {
			// If the object has been removed, don't bother syncing it
			// and just mark it for deletion in the trie.
			s.DeleteStateObject(stateObject)
		} else if _, ok := s.stateObjectsDirty[addr]; ok {
			// Write any contract code associated with the state object
			if stateObject.code != nil && stateObject.dirtyCode {
				if err := dbw.Put(stateObject.CodeHash(), stateObject.code); err != nil {
					return common.Hash{}, err
				}
				stateObject.dirtyCode = false
			}
			// Write any storage changes in the state object to its storage trie.
			if err := stateObject.CommitTrie(s.db, dbw); err != nil {
				return common.Hash{}, err
			}
			// Update the object in the main account trie.
			s.UpdateStateObject(stateObject)
		}
		delete(s.stateObjectsDirty, addr)
	}
	// Write trie changes.
	root, err = s.trie.CommitTo(dbw)
	if err == nil {
		s.pushTrie(s.trie)
	}
	return root, err
}
*/

func (self *StateDB) Refunds() *big.Int {
	return self.refund
}

// Fork preserve the given state and returns a handle to a new modifiable state
// that does not affect the preserved state.
func Fork(parent *StateDB) *StateDB {
	return &StateDB{
		db:            parent.db,
		trie:          parent.trie,
		pastTries:     parent.pastTries,
		codeSizeCache: parent.codeSizeCache,

		parent:            parent,
		stateObjects:      make(map[common.Address]*StateObject),
		localStateObjects: make(map[common.Address]bool),
		stateObjectsDirty: make(map[common.Address]struct{}),
		refund:            new(big.Int),
		logIdx:            parent.logIdx,
	}
}

// Reduce flattens the state in to a single new state, including all changes of all ancestors.
func Reduce(s *StateDB) *StateDB {
	if s.parent == nil {
		return s
	}
	state := Reduce(s.parent)

	for address, object := range s.stateObjects {
		if s.localStateObjects[address] {
			state.SetOwnedStateObject(address, object)
		} else {
			state.SetStateObject(object)
		}
	}

	state.logs = append(state.logs, s.logs...)
	state.refund.Add(state.refund, s.refund)

	return state
}

func IntermediateRoot(state *StateDB) common.Hash {
	for address, _ := range state.localStateObjects {
		stateObject := state.stateObjects[address]
		if _, isdirty := state.stateObjectsDirty[address]; isdirty {
			if stateObject.remove {
				state.DeleteStateObject(stateObject)
			} else {
				stateObject.UpdateRoot(state.db)
				state.UpdateStateObject(stateObject)
			}
		}
	}
	state.stateObjectsDirty = make(map[common.Address]struct{})
	return state.trie.Hash()
}

// Commit commits all state changes to the database.
func Commit(state *StateDB) (common.Hash, error) {
	root, batch := CommitBatch(state)
	return root, batch.Write()
}

// CommitBatch commits all state changes to a write batch but does not
// execute the batch. It is used to validate state changes against
// the root hash stored in a block.
func CommitBatch(state *StateDB) (common.Hash, ethdb.Batch) {
	batch := state.db.NewBatch()
	root, _ := stateCommit(state, batch)
	return root, batch
}

func stateCommit(state *StateDB, dbw trie.DatabaseWriter) (root common.Hash, err error) {
	// make sure the state is flattened before committing
	state = Reduce(state)
	state.refund = new(big.Int)

	for address, _ := range state.localStateObjects {
		stateObject := state.stateObjects[address]
		//for _, stateObject := range state.stateObjects {
		if stateObject.remove {
			// If the object has been removed, don't bother syncing it
			// and just mark it for deletion in the trie.
			stateObject.deleted = true
			state.trie.Delete(stateObject.Address().Bytes()[:])
		} else {
			// Write any contract code associated with the state object
			if stateObject.code != nil && stateObject.dirtyCode {
				if err := dbw.Put(stateObject.CodeHash(), stateObject.code); err != nil {
					return common.Hash{}, err
				}
				stateObject.dirtyCode = false
			}
			// Write any storage changes in the state object to its storage trie.
			if err := stateObject.CommitTrie(state.db, dbw); err != nil {
				return common.Hash{}, err
			}
			// Update the object in the main account trie.
			state.UpdateStateObject(stateObject)
		}
	}
	state.stateObjectsDirty = make(map[common.Address]struct{})
	// Write trie changes.
	root, err = state.trie.CommitTo(dbw)
	if err == nil {
		state.pushTrie(state.trie)
	}
	return root, err
}
