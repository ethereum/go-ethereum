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
func (l *liquidationPriceState) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, l.data)
}

// setError remembers the first non-nil error it is called with.
func (l *liquidationPriceState) setError(err error) {
	if l.dbErr == nil {
		l.dbErr = err
	}
}

func (l *liquidationPriceState) MarkStateLendingBookDirty(price common.Hash) {
	l.stateLendingBooksDirty[price] = struct{}{}
	if l.onDirty != nil {
		l.onDirty(l.liquidationPrice)
		l.onDirty = nil
	}
}

func (l *liquidationPriceState) createLendingBook(db Database, lendingBook common.Hash) (newobj *stateLendingBook) {
	newobj = newStateLendingBook(l.orderBook, l.liquidationPrice, lendingBook, orderList{Volume: Zero}, l.MarkStateLendingBookDirty)
	l.stateLendingBooks[lendingBook] = newobj
	l.stateLendingBooksDirty[lendingBook] = struct{}{}
	if l.onDirty != nil {
		l.onDirty(l.liquidationPrice)
		l.onDirty = nil
	}
	return newobj
}

func (l *liquidationPriceState) getTrie(db Database) Trie {
	if l.trie == nil {
		var err error
		l.trie, err = db.OpenStorageTrie(l.liquidationPrice, l.data.Root)
		if err != nil {
			l.trie, _ = db.OpenStorageTrie(l.liquidationPrice, EmptyHash)
			l.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return l.trie
}

func (l *liquidationPriceState) updateTrie(db Database) Trie {
	tr := l.getTrie(db)
	for lendingId, stateObject := range l.stateLendingBooks {
		delete(l.stateLendingBooksDirty, lendingId)
		if stateObject.empty() {
			l.setError(tr.TryDelete(lendingId[:]))
			continue
		}
		err := stateObject.updateRoot(db)
		if err != nil {
			log.Warn("updateTrie updateRoot", "err", err)
		}

		// Encoding []byte cannot fail, ok to ignore the error.
		v, _ := rlp.EncodeToBytes(stateObject)
		l.setError(tr.TryUpdate(lendingId[:], v))
	}
	return tr
}

func (l *liquidationPriceState) updateRoot(db Database) error {
	l.updateTrie(db)
	if l.dbErr != nil {
		return l.dbErr
	}
	root, err := l.trie.Commit(func(leaf []byte, parent common.Hash) error {
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
		l.data.Root = root
	}
	return err
}

func (l *liquidationPriceState) deepCopy(db *TradingStateDB, onDirty func(liquidationPrice common.Hash)) *liquidationPriceState {
	stateOrderList := newLiquidationPriceState(db, l.orderBook, l.liquidationPrice, l.data, onDirty)
	if l.trie != nil {
		stateOrderList.trie = db.db.CopyTrie(l.trie)
	}
	for key, value := range l.stateLendingBooks {
		stateOrderList.stateLendingBooks[key] = value.deepCopy(db, l.MarkStateLendingBookDirty)
	}
	for key, value := range l.stateLendingBooksDirty {
		stateOrderList.stateLendingBooksDirty[key] = value
	}
	return stateOrderList
}

// Retrieve a state object given my the address. Returns nil if not found.
func (l *liquidationPriceState) getStateLendingBook(db Database, lendingBook common.Hash) (stateObject *stateLendingBook) {
	// Prefer 'live' objects.
	if obj := l.stateLendingBooks[lendingBook]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := l.getTrie(db).TryGet(lendingBook[:])
	if len(enc) == 0 {
		l.setError(err)
		return nil
	}
	var data orderList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state lending book ", "orderbook", l.orderBook, "liquidation price", l.liquidationPrice, "lendingBook", lendingBook, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newStateLendingBook(l.orderBook, l.liquidationPrice, lendingBook, data, l.MarkStateLendingBookDirty)
	l.stateLendingBooks[lendingBook] = obj
	return obj
}

func (l *liquidationPriceState) getAllLiquidationData(db Database) map[common.Hash][]common.Hash {
	liquidationData := map[common.Hash][]common.Hash{}
	lendingBookTrie := l.getTrie(db)
	if lendingBookTrie == nil {
		return liquidationData
	}
	lendingBooks := []common.Hash{}
	for id, stateLendingBook := range l.stateLendingBooks {
		if !stateLendingBook.empty() {
			lendingBooks = append(lendingBooks, id)
		}
	}
	lendingBookListIt := trie.NewIterator(lendingBookTrie.NodeIterator(nil))
	for lendingBookListIt.Next() {
		id := common.BytesToHash(lendingBookListIt.Key)
		if _, exist := l.stateLendingBooks[id]; exist {
			continue
		}
		lendingBooks = append(lendingBooks, id)
	}
	for _, lendingBook := range lendingBooks {
		stateLendingBook := l.getStateLendingBook(db, lendingBook)
		if stateLendingBook != nil {
			liquidationData[lendingBook] = stateLendingBook.getAllTradeIds(db)
		}
	}
	return liquidationData
}

func (l *liquidationPriceState) AddVolume(amount *big.Int) {
	l.setVolume(new(big.Int).Add(l.data.Volume, amount))
}

func (c *liquidationPriceState) subVolume(amount *big.Int) {
	c.setVolume(new(big.Int).Sub(c.data.Volume, amount))
}

func (l *liquidationPriceState) setVolume(volume *big.Int) {
	l.data.Volume = volume
	if l.onDirty != nil {
		l.onDirty(l.liquidationPrice)
		l.onDirty = nil
	}
}

func (l *liquidationPriceState) Volume() *big.Int {
	return l.data.Volume
}
