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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"golang.org/x/net/context"
)

// LightState is a memory representation of a state.
// This version is ODR capable, caching only the already accessed part of the
// state, retrieving unknown parts on-demand from the ODR backend. Changes are
// never stored in the local database, only in the memory objects.
type LightState struct {
	odr          OdrBackend
	trie         *LightTrie
	id           *TrieID
	stateObjects map[string]*StateObject
	refund       *big.Int
}

// NewLightState creates a new LightState with the specified root.
// Note that the creation of a light state is always successful, even if the
// root is non-existent. In that case, ODR retrieval will always be unsuccessful
// and every operation will return with an error or wait for the context to be
// cancelled.
func NewLightState(id *TrieID, odr OdrBackend) *LightState {
	var tr *LightTrie
	if id != nil {
		tr = NewLightTrie(id, odr, true)
	}
	return &LightState{
		odr:          odr,
		trie:         tr,
		id:           id,
		stateObjects: make(map[string]*StateObject),
		refund:       new(big.Int),
	}
}

// AddRefund adds an amount to the refund value collected during a vm execution
func (self *LightState) AddRefund(gas *big.Int) {
	self.refund.Add(self.refund, gas)
}

// HasAccount returns true if an account exists at the given address
func (self *LightState) HasAccount(ctx context.Context, addr common.Address) (bool, error) {
	so, err := self.GetStateObject(ctx, addr)
	return so != nil, err
}

// GetBalance retrieves the balance from the given address or 0 if the account does
// not exist
func (self *LightState) GetBalance(ctx context.Context, addr common.Address) (*big.Int, error) {
	stateObject, err := self.GetStateObject(ctx, addr)
	if err != nil {
		return common.Big0, err
	}
	if stateObject != nil {
		return stateObject.balance, nil
	}

	return common.Big0, nil
}

// GetNonce returns the nonce at the given address or 0 if the account does
// not exist
func (self *LightState) GetNonce(ctx context.Context, addr common.Address) (uint64, error) {
	stateObject, err := self.GetStateObject(ctx, addr)
	if err != nil {
		return 0, err
	}
	if stateObject != nil {
		return stateObject.nonce, nil
	}
	return 0, nil
}

// GetCode returns the contract code at the given address or nil if the account
// does not exist
func (self *LightState) GetCode(ctx context.Context, addr common.Address) ([]byte, error) {
	stateObject, err := self.GetStateObject(ctx, addr)
	if err != nil {
		return nil, err
	}
	if stateObject != nil {
		return stateObject.code, nil
	}
	return nil, nil
}

// GetState returns the contract storage value at storage address b from the
// contract address a or common.Hash{} if the account does not exist
func (self *LightState) GetState(ctx context.Context, a common.Address, b common.Hash) (common.Hash, error) {
	stateObject, err := self.GetStateObject(ctx, a)
	if err == nil && stateObject != nil {
		return stateObject.GetState(ctx, b)
	}
	return common.Hash{}, err
}

// HasSuicided returns true if the given account has been marked for deletion
// or false if the account does not exist
func (self *LightState) HasSuicided(ctx context.Context, addr common.Address) (bool, error) {
	stateObject, err := self.GetStateObject(ctx, addr)
	if err == nil && stateObject != nil {
		return stateObject.remove, nil
	}
	return false, err
}

/*
 * SETTERS
 */

// AddBalance adds the given amount to the balance of the specified account
func (self *LightState) AddBalance(ctx context.Context, addr common.Address, amount *big.Int) error {
	stateObject, err := self.GetOrNewStateObject(ctx, addr)
	if err == nil && stateObject != nil {
		stateObject.AddBalance(amount)
	}
	return err
}

// SubBalance adds the given amount to the balance of the specified account
func (self *LightState) SubBalance(ctx context.Context, addr common.Address, amount *big.Int) error {
	stateObject, err := self.GetOrNewStateObject(ctx, addr)
	if err == nil && stateObject != nil {
		stateObject.SubBalance(amount)
	}
	return err
}

// SetNonce sets the nonce of the specified account
func (self *LightState) SetNonce(ctx context.Context, addr common.Address, nonce uint64) error {
	stateObject, err := self.GetOrNewStateObject(ctx, addr)
	if err == nil && stateObject != nil {
		stateObject.SetNonce(nonce)
	}
	return err
}

// SetCode sets the contract code at the specified account
func (self *LightState) SetCode(ctx context.Context, addr common.Address, code []byte) error {
	stateObject, err := self.GetOrNewStateObject(ctx, addr)
	if err == nil && stateObject != nil {
		stateObject.SetCode(crypto.Keccak256Hash(code), code)
	}
	return err
}

// SetState sets the storage value at storage address key of the account addr
func (self *LightState) SetState(ctx context.Context, addr common.Address, key common.Hash, value common.Hash) error {
	stateObject, err := self.GetOrNewStateObject(ctx, addr)
	if err == nil && stateObject != nil {
		stateObject.SetState(key, value)
	}
	return err
}

// Delete marks an account to be removed and clears its balance
func (self *LightState) Suicide(ctx context.Context, addr common.Address) (bool, error) {
	stateObject, err := self.GetOrNewStateObject(ctx, addr)
	if err == nil && stateObject != nil {
		stateObject.MarkForDeletion()
		stateObject.balance = new(big.Int)

		return true, nil
	}

	return false, err
}

//
// Get, set, new state object methods
//

// GetStateObject returns the state object of the given account or nil if the
// account does not exist
func (self *LightState) GetStateObject(ctx context.Context, addr common.Address) (stateObject *StateObject, err error) {
	stateObject = self.stateObjects[addr.Str()]
	if stateObject != nil {
		if stateObject.deleted {
			stateObject = nil
		}
		return stateObject, nil
	}
	data, err := self.trie.Get(ctx, addr[:])
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	stateObject, err = DecodeObject(ctx, self.id, addr, self.odr, []byte(data))
	if err != nil {
		return nil, err
	}

	self.SetStateObject(stateObject)

	return stateObject, nil
}

// SetStateObject sets the state object of the given account
func (self *LightState) SetStateObject(object *StateObject) {
	self.stateObjects[object.Address().Str()] = object
}

// GetOrNewStateObject returns the state object of the given account or creates a
// new one if the account does not exist
func (self *LightState) GetOrNewStateObject(ctx context.Context, addr common.Address) (*StateObject, error) {
	stateObject, err := self.GetStateObject(ctx, addr)
	if err == nil && (stateObject == nil || stateObject.deleted) {
		stateObject, err = self.CreateStateObject(ctx, addr)
	}
	return stateObject, err
}

// newStateObject creates a state object whether it exists in the state or not
func (self *LightState) newStateObject(addr common.Address) *StateObject {
	if glog.V(logger.Debug) {
		glog.Infof("(+) %x\n", addr)
	}

	stateObject := NewStateObject(addr, self.odr)
	self.stateObjects[addr.Str()] = stateObject

	return stateObject
}

// CreateStateObject creates creates a new state object and takes ownership.
// This is different from "NewStateObject"
func (self *LightState) CreateStateObject(ctx context.Context, addr common.Address) (*StateObject, error) {
	// Get previous (if any)
	so, err := self.GetStateObject(ctx, addr)
	if err != nil {
		return nil, err
	}
	// Create a new one
	newSo := self.newStateObject(addr)

	// If it existed set the balance to the new account
	if so != nil {
		newSo.balance = so.balance
	}

	return newSo, nil
}

// ForEachStorage calls a callback function for every key/value pair found
// in the local storage cache. Note that unlike core/state.StateObject,
// light.StateObject only returns cached values and doesn't download the
// entire storage tree.
func (self *LightState) ForEachStorage(ctx context.Context, addr common.Address, cb func(key, value common.Hash) bool) error {
	so, err := self.GetStateObject(ctx, addr)
	if err != nil {
		return err
	}

	if so == nil {
		return nil
	}

	for h, v := range so.storage {
		cb(h, v)
	}
	return nil
}

//
// Setting, copying of the state methods
//

// Copy creates a copy of the state
func (self *LightState) Copy() *LightState {
	// ignore error - we assume state-to-be-copied always exists
	state := NewLightState(nil, self.odr)
	state.trie = self.trie
	state.id = self.id
	for k, stateObject := range self.stateObjects {
		if stateObject.dirty {
			state.stateObjects[k] = stateObject.Copy()
		}
	}

	state.refund.Set(self.refund)
	return state
}

// Set copies the contents of the given state onto this state, overwriting
// its contents
func (self *LightState) Set(state *LightState) {
	self.trie = state.trie
	self.stateObjects = state.stateObjects
	self.refund = state.refund
}

// GetRefund returns the refund value collected during a vm execution
func (self *LightState) GetRefund() *big.Int {
	return self.refund
}
