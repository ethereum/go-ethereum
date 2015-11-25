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

package state

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/access"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"golang.org/x/net/context"
)

type Code []byte

func (self Code) String() string {
	return string(self) //strings.Join(Disassemble(self), " ")
}

type Storage map[string]common.Hash

func (self Storage) String() (str string) {
	for key, value := range self {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

func (self Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range self {
		cpy[key] = value
	}

	return cpy
}

type StateObject struct {
	// State database for storing state changes
	ca   *access.ChainAccess
	ctx  context.Context
	trie *trie.SecureTrie

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
	// Temporarily initialisation code
	initCode Code
	// Cached storage (flushed when updated)
	storage Storage

	// Mark for deletion
	// When an object is marked for deletion it will be delete from the trie
	// during the "update" phase of the state transition
	remove  bool
	deleted bool
	dirty   bool
}

func NewStateObject(ctx context.Context, address common.Address, ca *access.ChainAccess) *StateObject {
	object := &StateObject{ca: ca, ctx: ctx, address: address, balance: new(big.Int), dirty: true}
	object.trie, _ = trie.NewSecure(common.Hash{}, ca.Db())
	object.storage = make(Storage)
	return object
}

func NewStateObjectFromBytes(ctx context.Context, address common.Address, data []byte, ca *access.ChainAccess) *StateObject {
	var extobject struct {
		Nonce    uint64
		Balance  *big.Int
		Root     common.Hash
		CodeHash []byte
	}
	err := rlp.Decode(bytes.NewReader(data), &extobject)
	if err != nil {
		glog.Errorf("can't decode state object %x: %v", address, err)
		return nil
	}
	trie, err := trie.NewSecureOdr(ctx, extobject.Root, ca.Db(), NewTrieAccess(ca, extobject.Root, ca.Db()))
	if err != nil {
		// TODO: bubble this up or panic
		glog.Errorf("can't create account trie with root %x: %v", extobject.Root[:], err)
		return nil
	}

	object := &StateObject{address: address, ca: ca, ctx: ctx}
	object.nonce = extobject.Nonce
	object.balance = extobject.Balance
	object.codeHash = extobject.CodeHash
	object.trie = trie
	object.storage = make(map[string]common.Hash)
	object.code = RetrieveNodeData(ctx, ca, common.BytesToHash(extobject.CodeHash))
	return object
}

func (self *StateObject) MarkForDeletion() {
	self.remove = true
	self.dirty = true

	if glog.V(logger.Core) {
		glog.Infof("%x: #%d %v X\n", self.Address(), self.nonce, self.balance)
	}
}

func (c *StateObject) getAddr(addr common.Hash) common.Hash {
	var ret []byte
	value := c.trie.Get(addr[:])
	rlp.DecodeBytes(value, &ret)
	return common.BytesToHash(ret)
}

func (c *StateObject) setAddr(addr []byte, value common.Hash) {
	v, err := rlp.EncodeToBytes(bytes.TrimLeft(value[:], "\x00"))
	if err != nil {
		// if RLPing failed we better panic and not fail silently. This would be considered a consensus issue
		panic(err)
	}
	c.trie.Update(addr, v)
}

func (self *StateObject) Storage() Storage {
	return self.storage
}

func (self *StateObject) GetState(key common.Hash) common.Hash {
	strkey := key.Str()
	value, exists := self.storage[strkey]
	if !exists {
		value = self.getAddr(key)
		if (value != common.Hash{}) {
			self.storage[strkey] = value
		}
	}

	return value
}

func (self *StateObject) SetState(k, value common.Hash) {
	self.storage[k.Str()] = value
	self.dirty = true
}

// Update updates the current cached storage to the trie
func (self *StateObject) Update() {
	for key, value := range self.storage {
		if (value == common.Hash{}) {
			self.trie.Delete([]byte(key))
			continue
		}

		self.setAddr([]byte(key), value)
	}
}

func (c *StateObject) AddBalance(amount *big.Int) {
	c.SetBalance(new(big.Int).Add(c.balance, amount))

	if glog.V(logger.Core) {
		glog.Infof("%x: #%d %v (+ %v)\n", c.Address(), c.nonce, c.balance, amount)
	}
}

func (c *StateObject) SubBalance(amount *big.Int) {
	c.SetBalance(new(big.Int).Sub(c.balance, amount))

	if glog.V(logger.Core) {
		glog.Infof("%x: #%d %v (- %v)\n", c.Address(), c.nonce, c.balance, amount)
	}
}

func (c *StateObject) SetBalance(amount *big.Int) {
	c.balance = amount
	c.dirty = true
}

func (c *StateObject) St() Storage {
	return c.storage
}

// Return the gas back to the origin. Used by the Virtual machine or Closures
func (c *StateObject) ReturnGas(gas, price *big.Int) {}

// Copy creates a copy of the state object
func (self *StateObject) Copy() *StateObject {
	return self.CopyOdr(access.NoOdr)
}

// CopyOdr creates a copy of the state object with ODR option
func (self *StateObject) CopyOdr(ctx context.Context) *StateObject {
	stateObject := NewStateObject(ctx, self.Address(), self.ca)
	stateObject.balance.Set(self.balance)
	stateObject.codeHash = common.CopyBytes(self.codeHash)
	stateObject.nonce = self.nonce
	if access.IsOdrContext(ctx) {
		stateObject.trie = self.trie.CopySecureWithOdr(ctx, NewTrieAccess(self.ca, common.BytesToHash(self.trie.Root()), self.ca.Db()))
	} else {
		stateObject.trie = self.trie
	}
	stateObject.code = common.CopyBytes(self.code)
	stateObject.initCode = common.CopyBytes(self.initCode)
	stateObject.storage = self.storage.Copy()
	stateObject.remove = self.remove
	stateObject.dirty = self.dirty
	stateObject.deleted = self.deleted

	return stateObject
}

//
// Attribute accessors
//

func (self *StateObject) Balance() *big.Int {
	return self.balance
}

// Returns the address of the contract/account
func (c *StateObject) Address() common.Address {
	return c.address
}

func (self *StateObject) Trie() *trie.SecureTrie {
	return self.trie
}

func (self *StateObject) Root() []byte {
	return self.trie.Root()
}

func (self *StateObject) Code() []byte {
	return self.code
}

func (self *StateObject) SetCode(code []byte) {
	self.code = code
	self.dirty = true
}

func (self *StateObject) SetNonce(nonce uint64) {
	self.nonce = nonce
	self.dirty = true
}

func (self *StateObject) Nonce() uint64 {
	return self.nonce
}

func (self *StateObject) EachStorage(cb func(key, value []byte)) {
	// When iterating over the storage check the cache first
	for h, v := range self.storage {
		cb([]byte(h), v.Bytes())
	}

	it := self.trie.Iterator()
	for it.Next() {
		// ignore cached values
		key := self.trie.GetKey(it.Key)
		if _, ok := self.storage[string(key)]; !ok {
			cb(key, it.Value)
		}
	}
}

//
// Encoding
//

// State object encoding methods
func (c *StateObject) RlpEncode() []byte {
	return common.Encode([]interface{}{c.nonce, c.balance, c.Root(), c.CodeHash()})
}

func (c *StateObject) CodeHash() common.Bytes {
	return crypto.Sha3(c.code)
}

// Storage change object. Used by the manifest for notifying changes to
// the sub channels.
type StorageState struct {
	StateAddress []byte
	Address      []byte
	Value        *big.Int
}
