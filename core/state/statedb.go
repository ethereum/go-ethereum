// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

// Package state provides a caching layer atop the Ethereum state trie.
package state

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
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
	db   common.Database
	trie *trie.SecureTrie
	root common.Hash

	stateObjects map[string]*StateObject

	refund *big.Int

	thash, bhash common.Hash
	txIndex      int
	logs         map[common.Hash]Logs
}

// Create a new state from a given trie
func New(root common.Hash, db common.Database) *StateDB {
	trie := trie.NewSecure(root[:], db)
	return &StateDB{root: root, db: db, trie: trie, stateObjects: make(map[string]*StateObject), refund: new(big.Int), logs: make(map[common.Hash]Logs)}
}

func (self *StateDB) PrintRoot() {
	self.trie.Trie.PrintRoot()
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
	self.logs[self.thash] = append(self.logs[self.thash], log)
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
	addr := stateObject.Address()
	self.trie.Delete(addr[:])

	//delete(self.stateObjects, addr.Str())
}

// Retrieve a state object given my the address. Nil if not found
func (self *StateDB) GetStateObject(addr common.Address) *StateObject {
	//addr = common.Address(addr)

	stateObject := self.stateObjects[addr.Str()]
	if stateObject != nil {
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
	if stateObject == nil {
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

func (s *StateDB) Cmp(other *StateDB) bool {
	return bytes.Equal(s.trie.Root(), other.trie.Root())
}

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

	return state
}

func (self *StateDB) Set(state *StateDB) {
	self.trie = state.trie
	self.stateObjects = state.stateObjects

	self.refund = state.refund
	self.logs = state.logs
}

func (s *StateDB) Root() common.Hash {
	return common.BytesToHash(s.trie.Root())
}

func (s *StateDB) Trie() *trie.SecureTrie {
	return s.trie
}

// Resets the trie and all siblings
func (s *StateDB) Reset() {
	s.trie.Reset()

	// Reset all nested states
	for _, stateObject := range s.stateObjects {
		stateObject.Reset()
	}

	s.Empty()
}

// Syncs the trie and all siblings
func (s *StateDB) Sync() {
	// Sync all nested states
	for _, stateObject := range s.stateObjects {
		stateObject.trie.Commit()
	}

	s.trie.Commit()

	s.Empty()
}

func (self *StateDB) Empty() {
	self.stateObjects = make(map[string]*StateObject)
	self.refund = new(big.Int)
}

func (self *StateDB) Refunds() *big.Int {
	return self.refund
}

// SyncIntermediate updates the intermediate state and all mid steps
func (self *StateDB) SyncIntermediate() {
	self.refund = new(big.Int)

	for _, stateObject := range self.stateObjects {
		if stateObject.dirty {
			if stateObject.remove {
				self.DeleteStateObject(stateObject)
			} else {
				stateObject.Update()

				self.UpdateStateObject(stateObject)
			}
			stateObject.dirty = false
		}
	}
}

// SyncObjects syncs the changed objects to the trie
func (self *StateDB) SyncObjects() {
	self.trie = trie.NewSecure(self.root[:], self.db)

	self.refund = new(big.Int)

	for _, stateObject := range self.stateObjects {
		if stateObject.remove {
			self.DeleteStateObject(stateObject)
		} else {
			stateObject.Update()

			self.UpdateStateObject(stateObject)
		}
		stateObject.dirty = false
	}
}

// Debug stuff
func (self *StateDB) CreateOutputForDiff() {
	for _, stateObject := range self.stateObjects {
		stateObject.CreateOutputForDiff()
	}
}
