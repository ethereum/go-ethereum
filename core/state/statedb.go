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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/trie"
)

// StateDBs within the ethereum protocol are used to store anything
// within the merkle trie. StateDBs take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Contracts
// * Accounts
type StateDB struct {
	db   ethdb.Database
	trie *trie.SecureTrie

	stateObjects map[string]*StateObject

	refund *big.Int

	thash, bhash common.Hash
	txIndex      int
	logs         map[common.Hash]Logs
	logSize      uint
}

// Create a new state from a given trie
func New(root common.Hash, db ethdb.Database) *StateDB {
	tr, err := trie.NewSecure(root, db)
	if err != nil {
		// TODO: bubble this up
		tr, _ = trie.NewSecure(common.Hash{}, db)
		glog.Errorf("can't create state trie with root %x: %v", root[:], err)
	}
	return &StateDB{
		db:           db,
		trie:         tr,
		stateObjects: make(map[string]*StateObject),
		refund:       new(big.Int),
		logs:         make(map[common.Hash]Logs),
	}
}

func (self *StateDB) StartRecord(thash, bhash common.Hash, ti int) {
	self.thash = thash
	self.bhash = bhash
	self.txIndex = ti
}

func (self *StateDB) AddLog(log *Log) {
	log.TxHash = self.thash
	log.BlockHash = self.bhash
	log.TxIndex = uint(self.txIndex)
	log.Index = self.logSize
	self.logs[self.thash] = append(self.logs[self.thash], log)
	self.logSize++
}

func (self *StateDB) GetLogs(hash common.Hash) Logs {
	return self.logs[hash]
}

func (self *StateDB) Logs() Logs {
	var logs Logs
	for _, lgs := range self.logs {
		logs = append(logs, lgs...)
	}
	return logs
}

func (self *StateDB) Refund(gas *big.Int) {
	self.refund.Add(self.refund, gas)
}

/*
 * GETTERS
 */

func (self *StateDB) HasAccount(addr common.Address) bool {
	return self.GetStateObject(addr) != nil
}

// Retrieve the balance from the given address or 0 if object not found
func (self *StateDB) GetBalance(addr common.Address) *big.Int {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.balance
	}

	return common.Big0
}

func (self *StateDB) GetNonce(addr common.Address) uint64 {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.nonce
	}

	return 0
}

func (self *StateDB) GetCode(addr common.Address) []byte {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.code
	}

	return nil
}

func (self *StateDB) GetState(a common.Address, b common.Hash) common.Hash {
	stateObject := self.GetStateObject(a)
	if stateObject != nil {
		return stateObject.GetState(b)
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

func (self *StateDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (self *StateDB) SetCode(addr common.Address, code []byte) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(code)
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
		stateObject.balance = new(big.Int)

		return true
	}

	return false
}

//
// Setting, updating & deleting state object methods
//

// Update the given state object and apply it to state trie
func (self *StateDB) UpdateStateObject(stateObject *StateObject) {
	//addr := stateObject.Address()

	if len(stateObject.CodeHash()) > 0 {
		self.db.Put(stateObject.CodeHash(), stateObject.code)
	}
	addr := stateObject.Address()
	self.trie.Update(addr[:], stateObject.RlpEncode())
}

// Delete the given state object and delete it from the state trie
func (self *StateDB) DeleteStateObject(stateObject *StateObject) {
	stateObject.deleted = true

	addr := stateObject.Address()
	self.trie.Delete(addr[:])
	//delete(self.stateObjects, addr.Str())
}

// Retrieve a state object given my the address. Nil if not found
func (self *StateDB) GetStateObject(addr common.Address) (stateObject *StateObject) {
	stateObject = self.stateObjects[addr.Str()]
	if stateObject != nil {
		if stateObject.deleted {
			stateObject = nil
		}

		return stateObject
	}

	data := self.trie.Get(addr[:])
	if len(data) == 0 {
		return nil
	}

	stateObject = NewStateObjectFromBytes(addr, []byte(data), self.db)
	self.SetStateObject(stateObject)

	return stateObject
}

func (self *StateDB) SetStateObject(object *StateObject) {
	self.stateObjects[object.Address().Str()] = object
}

// Retrieve a state object or create a new state object if nil
func (self *StateDB) GetOrNewStateObject(addr common.Address) *StateObject {
	stateObject := self.GetStateObject(addr)
	if stateObject == nil || stateObject.deleted {
		stateObject = self.CreateAccount(addr)
	}

	return stateObject
}

// NewStateObject create a state object whether it exist in the trie or not
func (self *StateDB) newStateObject(addr common.Address) *StateObject {
	if glog.V(logger.Core) {
		glog.Infof("(+) %x\n", addr)
	}

	stateObject := NewStateObject(addr, self.db)
	self.stateObjects[addr.Str()] = stateObject

	return stateObject
}

// Creates creates a new state object and takes ownership. This is different from "NewStateObject"
func (self *StateDB) CreateAccount(addr common.Address) *StateObject {
	// Get previous (if any)
	so := self.GetStateObject(addr)
	// Create a new one
	newSo := self.newStateObject(addr)

	// If it existed set the balance to the new account
	if so != nil {
		newSo.balance = so.balance
	}

	return newSo
}

//
// Setting, copying of the state methods
//

func (self *StateDB) Copy() *StateDB {
	state := New(common.Hash{}, self.db)
	state.trie = self.trie
	for k, stateObject := range self.stateObjects {
		state.stateObjects[k] = stateObject.Copy()
	}

	state.refund.Set(self.refund)

	for hash, logs := range self.logs {
		state.logs[hash] = make(Logs, len(logs))
		copy(state.logs[hash], logs)
	}
	state.logSize = self.logSize

	return state
}

func (self *StateDB) Set(state *StateDB) {
	self.trie = state.trie
	self.stateObjects = state.stateObjects

	self.refund = state.refund
	self.logs = state.logs
	self.logSize = state.logSize
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (s *StateDB) IntermediateRoot() common.Hash {
	s.refund = new(big.Int)
	for _, stateObject := range s.stateObjects {
		if stateObject.dirty {
			if stateObject.remove {
				s.DeleteStateObject(stateObject)
			} else {
				stateObject.Update()
				s.UpdateStateObject(stateObject)
			}
			stateObject.dirty = false
		}
	}
	return s.trie.Hash()
}

// Commit commits all state changes to the database.
func (s *StateDB) Commit() (root common.Hash, err error) {
	return s.commit(s.db)
}

// CommitBatch commits all state changes to a write batch but does not
// execute the batch. It is used to validate state changes against
// the root hash stored in a block.
func (s *StateDB) CommitBatch() (root common.Hash, batch ethdb.Batch) {
	batch = s.db.NewBatch()
	root, _ = s.commit(batch)
	return root, batch
}

func (s *StateDB) commit(db trie.DatabaseWriter) (common.Hash, error) {
	s.refund = new(big.Int)

	for _, stateObject := range s.stateObjects {
		if stateObject.remove {
			// If the object has been removed, don't bother syncing it
			// and just mark it for deletion in the trie.
			s.DeleteStateObject(stateObject)
		} else {
			// Write any storage changes in the state object to its trie.
			stateObject.Update()
			// Commit the trie of the object to the batch.
			// This updates the trie root internally, so
			// getting the root hash of the storage trie
			// through UpdateStateObject is fast.
			if _, err := stateObject.trie.CommitTo(db); err != nil {
				return common.Hash{}, err
			}
			// Update the object in the account trie.
			s.UpdateStateObject(stateObject)
		}
		stateObject.dirty = false
	}
	return s.trie.CommitTo(db)
}

func (self *StateDB) Refunds() *big.Int {
	return self.refund
}

// Debug stuff
func (self *StateDB) CreateOutputForDiff() {
	for _, stateObject := range self.stateObjects {
		stateObject.CreateOutputForDiff()
	}
}
