// Copyright 2015 The go-ethereum Authors
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

package light

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/net/context"
)

var emptyCodeHash = crypto.Keccak256(nil)

// Code represents a contract code in binary form
type Code []byte

// String returns a string representation of the code
func (self Code) String() string {
	return string(self) //strings.Join(Disassemble(self), " ")
}

// Storage is a memory map cache of a contract storage
type Storage map[common.Hash]common.Hash

// String returns a string representation of the storage cache
func (self Storage) String() (str string) {
	for key, value := range self {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

// Copy copies the contents of a storage cache
func (self Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range self {
		cpy[key] = value
	}

	return cpy
}

// StateObject is a memory representation of an account or contract and its storage.
// This version is ODR capable, caching only the already accessed part of the
// storage, retrieving unknown parts on-demand from the ODR backend. Changes are
// never stored in the local database, only in the memory objects.
type StateObject struct {
	odr  OdrBackend
	trie *LightTrie

	// Address belonging to this account
	address common.Address
	// The balance of the account
	balance *big.Int
	// The nonce of the account
	nonce uint64
	// The code hash if code is present (i.e. a contract)
	codeHash []byte
	// The code for this account
	code Code
	// Cached storage (flushed when updated)
	storage Storage

	// Mark for deletion
	// When an object is marked for deletion it will be delete from the trie
	// during the "update" phase of the state transition
	remove  bool
	deleted bool
	dirty   bool
}

// NewStateObject creates a new StateObject of the specified account address
func NewStateObject(address common.Address, odr OdrBackend) *StateObject {
	object := &StateObject{
		odr:      odr,
		address:  address,
		balance:  new(big.Int),
		dirty:    true,
		codeHash: emptyCodeHash,
		storage:  make(Storage),
	}
	object.trie = NewLightTrie(&TrieID{}, odr, true)
	return object
}

// MarkForDeletion marks an account to be removed
func (self *StateObject) MarkForDeletion() {
	self.remove = true
	self.dirty = true

	log.Debug("", "msg", log.Lazy{Fn: func() string {
		return fmt.Sprintf("%x: #%d %v X\n", self.Address(), self.nonce, self.balance)
	}})
}

// getAddr gets the storage value at the given address from the trie
func (c *StateObject) getAddr(ctx context.Context, addr common.Hash) (common.Hash, error) {
	var ret []byte
	val, err := c.trie.Get(ctx, addr[:])
	if err != nil {
		return common.Hash{}, err
	}
	rlp.DecodeBytes(val, &ret)
	return common.BytesToHash(ret), nil
}

// Storage returns the storage cache object of the account
func (self *StateObject) Storage() Storage {
	return self.storage
}

// GetState returns the storage value at the given address from either the cache
// or the trie
func (self *StateObject) GetState(ctx context.Context, key common.Hash) (common.Hash, error) {
	value, exists := self.storage[key]
	if !exists {
		var err error
		value, err = self.getAddr(ctx, key)
		if err != nil {
			return common.Hash{}, err
		}
		if (value != common.Hash{}) {
			self.storage[key] = value
		}
	}

	return value, nil
}

// SetState sets the storage value at the given address
func (self *StateObject) SetState(k, value common.Hash) {
	self.storage[k] = value
	self.dirty = true
}

// AddBalance adds the given amount to the account balance
func (c *StateObject) AddBalance(amount *big.Int) {
	c.SetBalance(new(big.Int).Add(c.balance, amount))

	log.Debug("", "msg", log.Lazy{Fn: func() string {
		return fmt.Sprintf("%x: #%d %v (+ %v)\n", c.Address(), c.nonce, c.balance, amount)
	}})
}

// SubBalance subtracts the given amount from the account balance
func (c *StateObject) SubBalance(amount *big.Int) {
	c.SetBalance(new(big.Int).Sub(c.balance, amount))

	log.Debug("", "msg", log.Lazy{Fn: func() string {
		return fmt.Sprintf("%x: #%d %v (- %v)\n", c.Address(), c.nonce, c.balance, amount)
	}})
}

// SetBalance sets the account balance to the given amount
func (c *StateObject) SetBalance(amount *big.Int) {
	c.balance = amount
	c.dirty = true
}

// ReturnGas returns the gas back to the origin. Used by the Virtual machine or Closures
func (c *StateObject) ReturnGas(gas *big.Int) {}

// Copy creates a copy of the state object
func (self *StateObject) Copy() *StateObject {
	stateObject := NewStateObject(self.Address(), self.odr)
	stateObject.balance.Set(self.balance)
	stateObject.codeHash = common.CopyBytes(self.codeHash)
	stateObject.nonce = self.nonce
	stateObject.trie = self.trie
	stateObject.code = self.code
	stateObject.storage = self.storage.Copy()
	stateObject.remove = self.remove
	stateObject.dirty = self.dirty
	stateObject.deleted = self.deleted

	return stateObject
}

//
// Attribute accessors
//

// empty returns whether the account is considered empty.
func (self *StateObject) empty() bool {
	return self.nonce == 0 && self.balance.Sign() == 0 && bytes.Equal(self.codeHash, emptyCodeHash)
}

// Balance returns the account balance
func (self *StateObject) Balance() *big.Int {
	return self.balance
}

// Address returns the address of the contract/account
func (self *StateObject) Address() common.Address {
	return self.address
}

// Code returns the contract code
func (self *StateObject) Code() []byte {
	return self.code
}

// SetCode sets the contract code
func (self *StateObject) SetCode(hash common.Hash, code []byte) {
	self.code = code
	self.codeHash = hash[:]
	self.dirty = true
}

// SetNonce sets the account nonce
func (self *StateObject) SetNonce(nonce uint64) {
	self.nonce = nonce
	self.dirty = true
}

// Nonce returns the account nonce
func (self *StateObject) Nonce() uint64 {
	return self.nonce
}

// ForEachStorage calls a callback function for every key/value pair found
// in the local storage cache. Note that unlike core/state.StateObject,
// light.StateObject only returns cached values and doesn't download the
// entire storage tree.
func (self *StateObject) ForEachStorage(cb func(key, value common.Hash) bool) {
	for h, v := range self.storage {
		cb(h, v)
	}
}

// Never called, but must be present to allow StateObject to be used
// as a vm.Account interface that also satisfies the vm.ContractRef
// interface. Interfaces are awesome.
func (self *StateObject) Value() *big.Int {
	panic("Value on StateObject should never be called")
}

// Encoding

type extStateObject struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash
	CodeHash []byte
}

// DecodeObject decodes an RLP-encoded state object.
func DecodeObject(ctx context.Context, stateID *TrieID, address common.Address, odr OdrBackend, data []byte) (*StateObject, error) {
	var (
		obj = &StateObject{address: address, odr: odr, storage: make(Storage)}
		ext extStateObject
		err error
	)
	if err = rlp.DecodeBytes(data, &ext); err != nil {
		return nil, err
	}
	trieID := StorageTrieID(stateID, address, ext.Root)
	obj.trie = NewLightTrie(trieID, odr, true)
	if !bytes.Equal(ext.CodeHash, emptyCodeHash) {
		if obj.code, err = retrieveContractCode(ctx, obj.odr, trieID, common.BytesToHash(ext.CodeHash)); err != nil {
			return nil, fmt.Errorf("can't find code for hash %x: %v", ext.CodeHash, err)
		}
	}
	obj.nonce = ext.Nonce
	obj.balance = ext.Balance
	obj.codeHash = ext.CodeHash
	return obj, nil
}
