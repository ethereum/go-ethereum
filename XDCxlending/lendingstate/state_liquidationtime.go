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
	"github.com/XinFinOrg/XDPoSChain/trie"
	"io"
	"math/big"
)

type liquidationTimeState struct {
	time        common.Hash
	lendingBook common.Hash
	data        itemList

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by TradingStateDB.Commit.
	dbErr error

	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	cachedStorage map[common.Hash]common.Hash
	dirtyStorage  map[common.Hash]common.Hash

	onDirty func(time common.Hash) // Callback method to mark a state object newly dirty
}

func (s *liquidationTimeState) empty() bool {
	return s.data.Volume == nil || s.data.Volume.Sign() == 0
}

func newLiquidationTimeState(time common.Hash, lendingBook common.Hash, data itemList, onDirty func(time common.Hash)) *liquidationTimeState {
	return &liquidationTimeState{
		lendingBook:   lendingBook,
		time:          time,
		data:          data,
		cachedStorage: make(map[common.Hash]common.Hash),
		dirtyStorage:  make(map[common.Hash]common.Hash),
		onDirty:       onDirty,
	}
}

func (self *liquidationTimeState) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, self.data)
}

func (self *liquidationTimeState) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (self *liquidationTimeState) getTrie(db Database) Trie {
	if self.trie == nil {
		var err error
		self.trie, err = db.OpenStorageTrie(self.lendingBook, self.data.Root)
		if err != nil {
			self.trie, _ = db.OpenStorageTrie(self.time, EmptyHash)
			self.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return self.trie
}

func (self *liquidationTimeState) Exist(db Database, tradeId common.Hash) bool {
	amount, exists := self.cachedStorage[tradeId]
	if exists {
		return true
	}
	// Load from DB in case it is missing.
	enc, err := self.getTrie(db).TryGet(tradeId[:])
	if err != nil {
		self.setError(err)
		return false
	}
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			self.setError(err)
		}
		amount.SetBytes(content)
	}
	if (amount != common.Hash{}) {
		self.cachedStorage[tradeId] = amount
	}
	return true
}

func (self *liquidationTimeState) getAllTradeIds(db Database) []common.Hash {
	tradeIds := []common.Hash{}
	lendingBookTrie := self.getTrie(db)
	if lendingBookTrie == nil {
		return tradeIds
	}
	for id, value := range self.cachedStorage {
		if !common.EmptyHash(value) {
			tradeIds = append(tradeIds, id)
		}
	}
	orderListIt := trie.NewIterator(lendingBookTrie.NodeIterator(nil))
	for orderListIt.Next() {
		id := common.BytesToHash(orderListIt.Key)
		if _, exist := self.cachedStorage[id]; exist {
			continue
		}
		tradeIds = append(tradeIds, id)
	}
	return tradeIds
}

func (self *liquidationTimeState) insertTradeId(db Database, tradeId common.Hash) {
	self.setTradeId(tradeId, tradeId)
	self.setError(self.getTrie(db).TryUpdate(tradeId[:], tradeId[:]))
}

func (self *liquidationTimeState) removeTradeId(db Database, tradeId common.Hash) {
	tr := self.getTrie(db)
	self.setError(tr.TryDelete(tradeId[:]))
	self.setTradeId(tradeId, EmptyHash)
}

func (self *liquidationTimeState) setTradeId(tradeId common.Hash, value common.Hash) {
	self.cachedStorage[tradeId] = value
	self.dirtyStorage[tradeId] = value

	if self.onDirty != nil {
		self.onDirty(self.lendingBook)
		self.onDirty = nil
	}
}

func (self *liquidationTimeState) updateTrie(db Database) Trie {
	tr := self.getTrie(db)
	for key, value := range self.dirtyStorage {
		delete(self.dirtyStorage, key)
		if value == EmptyHash {
			self.setError(tr.TryDelete(key[:]))
			continue
		}
		v, _ := rlp.EncodeToBytes(bytes.TrimLeft(value[:], "\x00"))
		self.setError(tr.TryUpdate(key[:], v))
	}
	return tr
}

func (self *liquidationTimeState) updateRoot(db Database) error {
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

func (self *liquidationTimeState) deepCopy(db *LendingStateDB, onDirty func(time common.Hash)) *liquidationTimeState {
	stateLendingBook := newLiquidationTimeState(self.lendingBook, self.time, self.data, onDirty)
	if self.trie != nil {
		stateLendingBook.trie = db.db.CopyTrie(self.trie)
	}
	for key, value := range self.dirtyStorage {
		stateLendingBook.dirtyStorage[key] = value
	}
	for key, value := range self.cachedStorage {
		stateLendingBook.cachedStorage[key] = value
	}
	return stateLendingBook
}

func (c *liquidationTimeState) AddVolume(amount *big.Int) {
	c.setVolume(new(big.Int).Add(c.data.Volume, amount))
}

func (c *liquidationTimeState) subVolume(amount *big.Int) {
	c.setVolume(new(big.Int).Sub(c.data.Volume, amount))
}

func (self *liquidationTimeState) setVolume(volume *big.Int) {
	self.data.Volume = volume
	if self.onDirty != nil {
		self.onDirty(self.lendingBook)
		self.onDirty = nil
	}
}

func (self *liquidationTimeState) Volume() *big.Int {
	return self.data.Volume
}
