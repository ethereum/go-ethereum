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

package lendingstate

import (
	"bytes"
	"fmt"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"io"
	"math/big"
)

type itemListState struct {
	lendingBook common.Hash
	key         common.Hash
	data        itemList

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by LendingStateDB.Commit.
	dbErr error

	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	cachedStorage map[common.Hash]common.Hash // Storage entry cache to avoid duplicate reads
	dirtyStorage  map[common.Hash]common.Hash // Storage entries that need to be flushed to disk

	onDirty func(price common.Hash) // Callback method to mark a state object newly dirty
}

func (s *itemListState) empty() bool {
	return s.data.Volume == nil || s.data.Volume.Sign() == 0
}

func newItemListState(lendingBook common.Hash, key common.Hash, data itemList, onDirty func(price common.Hash)) *itemListState {
	return &itemListState{
		lendingBook:   lendingBook,
		key:           key,
		data:          data,
		cachedStorage: make(map[common.Hash]common.Hash),
		dirtyStorage:  make(map[common.Hash]common.Hash),
		onDirty:       onDirty,
	}
}

// EncodeRLP implements rlp.Encoder.
func (c *itemListState) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

// setError remembers the first non-nil error it is called with.
func (self *itemListState) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (c *itemListState) getTrie(db Database) Trie {
	if c.trie == nil {
		var err error
		c.trie, err = db.OpenStorageTrie(c.key, c.data.Root)
		if err != nil {
			c.trie, _ = db.OpenStorageTrie(c.key, EmptyHash)
			c.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return c.trie
}

func (self *itemListState) GetOrderAmount(db Database, orderId common.Hash) common.Hash {
	amount, exists := self.cachedStorage[orderId]
	if exists {
		return amount
	}
	// Load from DB in case it is missing.
	enc, err := self.getTrie(db).TryGet(orderId[:])
	if err != nil {
		self.setError(err)
		return EmptyHash
	}
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			self.setError(err)
		}
		amount.SetBytes(content)
	}
	if (amount != common.Hash{}) {
		self.cachedStorage[orderId] = amount
	}
	return amount
}

func (self *itemListState) insertLendingItem(db Database, orderId common.Hash, amount common.Hash) {
	self.setOrderItem(orderId, amount)
	self.setError(self.getTrie(db).TryUpdate(orderId[:], amount[:]))
}

func (self *itemListState) removeOrderItem(db Database, orderId common.Hash) {
	tr := self.getTrie(db)
	self.setError(tr.TryDelete(orderId[:]))
	self.setOrderItem(orderId, EmptyHash)
}

func (self *itemListState) setOrderItem(orderId common.Hash, amount common.Hash) {
	self.cachedStorage[orderId] = amount
	self.dirtyStorage[orderId] = amount

	if self.onDirty != nil {
		self.onDirty(self.key)
		self.onDirty = nil
	}
}

// updateAskTrie writes cached storage modifications into the object's storage trie.
func (self *itemListState) updateTrie(db Database) Trie {
	tr := self.getTrie(db)
	for orderId, amount := range self.dirtyStorage {
		delete(self.dirtyStorage, orderId)
		if amount == EmptyHash {
			self.setError(tr.TryDelete(orderId[:]))
			continue
		}
		v, _ := rlp.EncodeToBytes(bytes.TrimLeft(amount[:], "\x00"))
		self.setError(tr.TryUpdate(orderId[:], v))
	}
	return tr
}

// UpdateRoot sets the trie root to the current root tradeId of
func (self *itemListState) updateRoot(db Database) error {
	self.updateTrie(db)
	if self.dbErr != nil {
		return self.dbErr
	}
	root, err := self.trie.Commit(nil)
	if err == nil {
		self.data.Root = root
	}
	return err
}

func (self *itemListState) deepCopy(db *LendingStateDB, onDirty func(price common.Hash)) *itemListState {
	stateOrderList := newItemListState(self.lendingBook, self.key, self.data, onDirty)
	if self.trie != nil {
		stateOrderList.trie = db.db.CopyTrie(self.trie)
	}
	for orderId, amount := range self.dirtyStorage {
		stateOrderList.dirtyStorage[orderId] = amount
	}
	for orderId, amount := range self.cachedStorage {
		stateOrderList.cachedStorage[orderId] = amount
	}
	return stateOrderList
}

func (c *itemListState) AddVolume(amount *big.Int) {
	c.setVolume(new(big.Int).Add(c.data.Volume, amount))
}

func (c *itemListState) subVolume(amount *big.Int) {
	c.setVolume(new(big.Int).Sub(c.data.Volume, amount))
}

func (self *itemListState) setVolume(volume *big.Int) {
	self.data.Volume = volume
	if self.onDirty != nil {
		self.onDirty(self.key)
		self.onDirty = nil
	}
}

func (self *itemListState) Volume() *big.Int {
	return self.data.Volume
}
