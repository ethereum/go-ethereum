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
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
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
type Storage map[string]common.Hash

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
	object.trie = NewLightTrie(common.Hash{}, odr, true)
	return object
}

// MarkForDeletion marks an account to be removed
func (self *StateObject) MarkForDeletion() {
	self.remove = true
	self.dirty = true

	if glog.V(logger.Core) {
		glog.Infof("%x: #%d %v X\n", self.Address(), self.nonce, self.balance)
	}
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
	strkey := key.Str()
	value, exists := self.storage[strkey]
	if !exists {
		var err error
		value, err = self.getAddr(ctx, key)
		if err != nil {
			return common.Hash{}, err
		}
		if (value != common.Hash{}) {
			self.storage[strkey] = value
		}
	}

	return value, nil
}

// SetState sets the storage value at the given address
func (self *StateObject) SetState(k, value common.Hash) {
	self.storage[k.Str()] = value
	self.dirty = true
}

// AddBalance adds the given amount to the account balance
func (c *StateObject) AddBalance(amount *big.Int) {
	c.SetBalance(new(big.Int).Add(c.balance, amount))

	if glog.V(logger.Core) {
		glog.Infof("%x: #%d %v (+ %v)\n", c.Address(), c.nonce, c.balance, amount)
	}
}

// SubBalance subtracts the given amount from the account balance
func (c *StateObject) SubBalance(amount *big.Int) {
	c.SetBalance(new(big.Int).Sub(c.balance, amount))

	if glog.V(logger.Core) {
		glog.Infof("%x: #%d %v (- %v)\n", c.Address(), c.nonce, c.balance, amount)
	}
}

// SetBalance sets the account balance to the given amount
func (c *StateObject) SetBalance(amount *big.Int) {
	c.balance = amount
	c.dirty = true
}

// Copy creates a copy of the state object
func (self *StateObject) Copy() *StateObject {
	stateObject := NewStateObject(self.Address(), self.odr)
	stateObject.balance.Set(self.balance)
	stateObject.codeHash = common.CopyBytes(self.codeHash)
	stateObject.nonce = self.nonce
	stateObject.trie = self.trie
	stateObject.code = common.CopyBytes(self.code)
	stateObject.storage = self.storage.Copy()
	stateObject.remove = self.remove
	stateObject.dirty = self.dirty
	stateObject.deleted = self.deleted

	return stateObject
}

//
// Attribute accessors
//

// Balance returns the account balance
func (self *StateObject) Balance() *big.Int {
	return self.balance
}

// Address returns the address of the contract/account
func (c *StateObject) Address() common.Address {
	return c.address
}

// Code returns the contract code
func (self *StateObject) Code() []byte {
	return self.code
}

// SetCode sets the contract code
func (self *StateObject) SetCode(code []byte) {
	self.code = code
	self.codeHash = crypto.Keccak256(code)
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

// Encoding

type extStateObject struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash
	CodeHash []byte
}

// DecodeObject decodes an RLP-encoded state object.
func DecodeObject(ctx context.Context, address common.Address, odr OdrBackend, data []byte) (*StateObject, error) {
	var (
		obj = &StateObject{address: address, odr: odr, storage: make(Storage)}
		ext extStateObject
		err error
	)
	if err = rlp.DecodeBytes(data, &ext); err != nil {
		return nil, err
	}
	obj.trie = NewLightTrie(ext.Root, odr, true)
	if !bytes.Equal(ext.CodeHash, emptyCodeHash) {
		if obj.code, err = retrieveNodeData(ctx, obj.odr, common.BytesToHash(ext.CodeHash)); err != nil {
			return nil, fmt.Errorf("can't find code for hash %x: %v", ext.CodeHash, err)
		}
	}
	obj.nonce = ext.Nonce
	obj.balance = ext.Balance
	obj.codeHash = ext.CodeHash
	return obj, nil
}
