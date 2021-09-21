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
	"fmt"
	"io"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/XinFinOrg/XDPoSChain/trie"
)

type liquidationPriceState struct {
	liquidationPrice common.Hash
	orderBook        common.Hash
	data             orderList
	db               *TradingStateDB

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by TradingStateDB.Commit.
	dbErr error

	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	stateLendingBooks      map[common.Hash]*stateLendingBook
	stateLendingBooksDirty map[common.Hash]struct{}

	onDirty func(price common.Hash) // Callback method to mark a state object newly dirty
}

// empty returns whether the orderId is considered empty.
func (s *liquidationPriceState) empty() bool {
	return s.data.Volume == nil || s.data.Volume.Sign() == 0
}

// newObject creates a state object.
func newLiquidationPriceState(db *TradingStateDB, orderBook common.Hash, price common.Hash, data orderList, onDirty func(price common.Hash)) *liquidationPriceState {
	return &liquidationPriceState{
		db:                     db,
		orderBook:              orderBook,
		liquidationPrice:       price,
		data:                   data,
		stateLendingBooks:      make(map[common.Hash]*stateLendingBook),
		stateLendingBooksDirty: make(map[common.Hash]struct{}),
		onDirty:                onDirty,
	}
}

// EncodeRLP implements rlp.Encoder.
func (c *liquidationPriceState) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

// setError remembers the first non-nil error it is called with.
func (self *liquidationPriceState) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (self *liquidationPriceState) MarkStateLendingBookDirty(price common.Hash) {
	self.stateLendingBooksDirty[price] = struct{}{}
	if self.onDirty != nil {
		self.onDirty(self.liquidationPrice)
		self.onDirty = nil
	}
}

func (self *liquidationPriceState) createLendingBook(db Database, lendingBook common.Hash) (newobj *stateLendingBook) {
	newobj = newStateLendingBook(self.orderBook, self.liquidationPrice, lendingBook, orderList{Volume: Zero}, self.MarkStateLendingBookDirty)
	self.stateLendingBooks[lendingBook] = newobj
	self.stateLendingBooksDirty[lendingBook] = struct{}{}
	if self.onDirty != nil {
		self.onDirty(self.liquidationPrice)
		self.onDirty = nil
	}
	return newobj
}

func (self *liquidationPriceState) getTrie(db Database) Trie {
	if self.trie == nil {
		var err error
		self.trie, err = db.OpenStorageTrie(self.liquidationPrice, self.data.Root)
		if err != nil {
			self.trie, _ = db.OpenStorageTrie(self.liquidationPrice, EmptyHash)
			self.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return self.trie
}

func (self *liquidationPriceState) updateTrie(db Database) Trie {
	tr := self.getTrie(db)
	for lendingId, stateObject := range self.stateLendingBooks {
		delete(self.stateLendingBooksDirty, lendingId)
		if stateObject.empty() {
			self.setError(tr.TryDelete(lendingId[:]))
			continue
		}
		stateObject.updateRoot(db)
		// Encoding []byte cannot fail, ok to ignore the error.
		v, _ := rlp.EncodeToBytes(stateObject)
		self.setError(tr.TryUpdate(lendingId[:], v))
	}
	return tr
}

func (self *liquidationPriceState) updateRoot(db Database) error {
	self.updateTrie(db)
	if self.dbErr != nil {
		return self.dbErr
	}
	root, err := self.trie.Commit(func(leaf []byte, parent common.Hash) error {
		var orderList orderList
		if err := rlp.DecodeBytes(leaf, &orderList); err != nil {
			return nil
		}
		if orderList.Root != EmptyRoot {
			db.TrieDB().Reference(orderList.Root, parent)
		}
		return nil
	})
	if err == nil {
		self.data.Root = root
	}
	return err
}

func (self *liquidationPriceState) deepCopy(db *TradingStateDB, onDirty func(liquidationPrice common.Hash)) *liquidationPriceState {
	stateOrderList := newLiquidationPriceState(db, self.orderBook, self.liquidationPrice, self.data, onDirty)
	if self.trie != nil {
		stateOrderList.trie = db.db.CopyTrie(self.trie)
	}
	for key, value := range self.stateLendingBooks {
		stateOrderList.stateLendingBooks[key] = value.deepCopy(db, self.MarkStateLendingBookDirty)
	}
	for key, value := range self.stateLendingBooksDirty {
		stateOrderList.stateLendingBooksDirty[key] = value
	}
	return stateOrderList
}

// Retrieve a state object given my the address. Returns nil if not found.
func (self *liquidationPriceState) getStateLendingBook(db Database, lendingBook common.Hash) (stateObject *stateLendingBook) {
	// Prefer 'live' objects.
	if obj := self.stateLendingBooks[lendingBook]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := self.getTrie(db).TryGet(lendingBook[:])
	if len(enc) == 0 {
		self.setError(err)
		return nil
	}
	var data orderList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state lending book ", "orderbook", self.orderBook, "liquidation price", self.liquidationPrice, "lendingBook", lendingBook, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newStateLendingBook(self.orderBook, self.liquidationPrice, lendingBook, data, self.MarkStateLendingBookDirty)
	self.stateLendingBooks[lendingBook] = obj
	return obj
}

func (self *liquidationPriceState) getAllLiquidationData(db Database) map[common.Hash][]common.Hash {
	liquidationData := map[common.Hash][]common.Hash{}
	lendingBookTrie := self.getTrie(db)
	if lendingBookTrie == nil {
		return liquidationData
	}
	lendingBooks := []common.Hash{}
	for id, stateLendingBook := range self.stateLendingBooks {
		if !stateLendingBook.empty() {
			lendingBooks = append(lendingBooks, id)
		}
	}
	lendingBookListIt := trie.NewIterator(lendingBookTrie.NodeIterator(nil))
	for lendingBookListIt.Next() {
		id := common.BytesToHash(lendingBookListIt.Key)
		if _, exist := self.stateLendingBooks[id]; exist {
			continue
		}
		lendingBooks = append(lendingBooks, id)
	}
	for _, lendingBook := range lendingBooks {
		stateLendingBook := self.getStateLendingBook(db, lendingBook)
		if stateLendingBook != nil {
			liquidationData[lendingBook] = stateLendingBook.getAllTradeIds(db)
		}
	}
	return liquidationData
}

func (c *liquidationPriceState) AddVolume(amount *big.Int) {
	c.setVolume(new(big.Int).Add(c.data.Volume, amount))
}

func (c *liquidationPriceState) subVolume(amount *big.Int) {
	c.setVolume(new(big.Int).Sub(c.data.Volume, amount))
}

func (self *liquidationPriceState) setVolume(volume *big.Int) {
	self.data.Volume = volume
	if self.onDirty != nil {
		self.onDirty(self.liquidationPrice)
		self.onDirty = nil
	}
}

func (self *liquidationPriceState) Volume() *big.Int {
	return self.data.Volume
}
