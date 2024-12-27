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

package tradingstate

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/rlp"
)

// stateObject represents an Ethereum orderId which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// tradingExchangeObject values can be accessed and modified through the object.
// Finally, call CommitAskTrie to write the modified storage trie into a database.
type stateOrderList struct {
	price     common.Hash
	orderBook common.Hash
	orderType string
	data      orderList
	db        *TradingStateDB

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by TradingStateDB.Commit.
	dbErr error

	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	cachedStorage map[common.Hash]common.Hash // Storage entry cache to avoid duplicate reads
	dirtyStorage  map[common.Hash]common.Hash // Storage entries that need to be flushed to disk

	onDirty func(price common.Hash) // Callback method to mark a state object newly dirty
}

// empty returns whether the orderId is considered empty.
func (s *stateOrderList) empty() bool {
	return s.data.Volume == nil || s.data.Volume.Cmp(Zero) == 0
}

// newObject creates a state object.
func newStateOrderList(db *TradingStateDB, orderType string, orderBook common.Hash, price common.Hash, data orderList, onDirty func(price common.Hash)) *stateOrderList {
	return &stateOrderList{
		db:            db,
		orderType:     orderType,
		orderBook:     orderBook,
		price:         price,
		data:          data,
		cachedStorage: make(map[common.Hash]common.Hash),
		dirtyStorage:  make(map[common.Hash]common.Hash),
		onDirty:       onDirty,
	}
}

// EncodeRLP implements rlp.Encoder.
func (s *stateOrderList) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, s.data)
}

// setError remembers the first non-nil error it is called with.
func (s *stateOrderList) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

func (c *stateOrderList) getTrie(db Database) Trie {
	if c.trie == nil {
		var err error
		c.trie, err = db.OpenStorageTrie(c.price, c.data.Root)
		if err != nil {
			c.trie, _ = db.OpenStorageTrie(c.price, EmptyHash)
			c.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return c.trie
}

// GetState returns a value in orderId storage.
func (s *stateOrderList) GetOrderAmount(db Database, orderId common.Hash) common.Hash {
	amount, exists := s.cachedStorage[orderId]
	if exists {
		return amount
	}
	// Load from DB in case it is missing.
	enc, err := s.getTrie(db).TryGet(orderId[:])
	if err != nil {
		s.setError(err)
		return EmptyHash
	}
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			s.setError(err)
		}
		amount.SetBytes(content)
	}
	if (amount != common.Hash{}) {
		s.cachedStorage[orderId] = amount
	}
	return amount
}

// SetState updates a value in orderId storage.
func (s *stateOrderList) insertOrderItem(db Database, orderId common.Hash, amount common.Hash) {
	s.setOrderItem(orderId, amount)
	s.setError(s.getTrie(db).TryUpdate(orderId[:], amount[:]))
}

// SetState updates a value in orderId storage.
func (s *stateOrderList) removeOrderItem(db Database, orderId common.Hash) {
	tr := s.getTrie(db)
	s.setError(tr.TryDelete(orderId[:]))
	s.setOrderItem(orderId, EmptyHash)
}

func (s *stateOrderList) setOrderItem(orderId common.Hash, amount common.Hash) {
	s.cachedStorage[orderId] = amount
	s.dirtyStorage[orderId] = amount

	if s.onDirty != nil {
		s.onDirty(s.Price())
		s.onDirty = nil
	}
}

// updateAskTrie writes cached storage modifications into the object's storage trie.
func (s *stateOrderList) updateTrie(db Database) Trie {
	tr := s.getTrie(db)
	for orderId, amount := range s.dirtyStorage {
		delete(s.dirtyStorage, orderId)
		if amount == EmptyHash {
			s.setError(tr.TryDelete(orderId[:]))
			continue
		}
		v, _ := rlp.EncodeToBytes(bytes.TrimLeft(amount[:], "\x00"))
		s.setError(tr.TryUpdate(orderId[:], v))
	}
	return tr
}

// UpdateRoot sets the trie root to the current root orderId of
func (s *stateOrderList) updateRoot(db Database) error {
	s.updateTrie(db)
	if s.dbErr != nil {
		return s.dbErr
	}
	root, err := s.trie.Commit(nil)
	if err == nil {
		s.data.Root = root
	}
	return err
}

func (s *stateOrderList) deepCopy(db *TradingStateDB, onDirty func(price common.Hash)) *stateOrderList {
	stateOrderList := newStateOrderList(db, s.orderType, s.orderBook, s.price, s.data, onDirty)
	if s.trie != nil {
		stateOrderList.trie = db.db.CopyTrie(s.trie)
	}
	for orderId, amount := range s.dirtyStorage {
		stateOrderList.dirtyStorage[orderId] = amount
	}
	for orderId, amount := range s.cachedStorage {
		stateOrderList.cachedStorage[orderId] = amount
	}
	return stateOrderList
}

// AddVolume removes amount from c's balance.
// It is used to add funds to the destination exchanges of a transfer.
func (s *stateOrderList) AddVolume(amount *big.Int) {
	s.setVolume(new(big.Int).Add(s.data.Volume, amount))
}

// AddVolume removes amount from c's balance.
// It is used to add funds to the destination exchanges of a transfer.
func (s *stateOrderList) subVolume(amount *big.Int) {
	s.setVolume(new(big.Int).Sub(s.data.Volume, amount))
}

func (s *stateOrderList) setVolume(volume *big.Int) {
	s.data.Volume = volume
	if s.onDirty != nil {
		s.onDirty(s.price)
		s.onDirty = nil
	}
}

// Returns the address of the contract/orderId
func (s *stateOrderList) Price() common.Hash {
	return s.price
}

func (s *stateOrderList) Volume() *big.Int {
	return s.data.Volume
}
