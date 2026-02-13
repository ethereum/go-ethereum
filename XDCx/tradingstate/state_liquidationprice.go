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
	"github.com/XinFinOrg/XDPoSChain/core/types"
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
func (s *liquidationPriceState) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, s.data)
}

// setError remembers the first non-nil error it is called with.
func (s *liquidationPriceState) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

func (s *liquidationPriceState) MarkStateLendingBookDirty(price common.Hash) {
	s.stateLendingBooksDirty[price] = struct{}{}
	if s.onDirty != nil {
		s.onDirty(s.liquidationPrice)
		s.onDirty = nil
	}
}

func (s *liquidationPriceState) createLendingBook(db Database, lendingBook common.Hash) (newobj *stateLendingBook) {
	newobj = newStateLendingBook(s.orderBook, s.liquidationPrice, lendingBook, orderList{Volume: Zero}, s.MarkStateLendingBookDirty)
	s.stateLendingBooks[lendingBook] = newobj
	s.stateLendingBooksDirty[lendingBook] = struct{}{}
	if s.onDirty != nil {
		s.onDirty(s.liquidationPrice)
		s.onDirty = nil
	}
	return newobj
}

func (s *liquidationPriceState) getTrie(db Database) Trie {
	if s.trie == nil {
		var err error
		s.trie, err = db.OpenStorageTrie(s.liquidationPrice, common.Address{}, s.data.Root)
		if err != nil {
			s.trie, _ = db.OpenStorageTrie(s.liquidationPrice, common.Address{}, types.EmptyRootHash)
			s.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return s.trie
}

func (s *liquidationPriceState) updateTrie(db Database) Trie {
	tr := s.getTrie(db)
	for lendingId, stateObject := range s.stateLendingBooks {
		delete(s.stateLendingBooksDirty, lendingId)
		if stateObject.empty() {
			s.setError(tr.TryDelete(lendingId[:]))
			continue
		}
		err := stateObject.updateRoot(db)
		if err != nil {
			log.Warn("updateTrie updateRoot", "err", err)
		}

		// Encoding []byte cannot fail, ok to ignore the error.
		v, _ := rlp.EncodeToBytes(stateObject)
		s.setError(tr.TryUpdate(lendingId[:], v))
	}
	return tr
}

func (s *liquidationPriceState) updateRoot(db Database) error {
	s.updateTrie(db)
	if s.dbErr != nil {
		return s.dbErr
	}
	root, err := s.trie.Commit(func(_ [][]byte, _ []byte, leaf []byte, parent common.Hash, _ []byte) error {
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
		s.data.Root = root
	}
	return err
}

func (s *liquidationPriceState) deepCopy(db *TradingStateDB, onDirty func(liquidationPrice common.Hash)) *liquidationPriceState {
	stateOrderList := newLiquidationPriceState(db, s.orderBook, s.liquidationPrice, s.data, onDirty)
	if s.trie != nil {
		stateOrderList.trie = db.db.CopyTrie(s.trie)
	}
	for key, value := range s.stateLendingBooks {
		stateOrderList.stateLendingBooks[key] = value.deepCopy(db, s.MarkStateLendingBookDirty)
	}
	for key, value := range s.stateLendingBooksDirty {
		stateOrderList.stateLendingBooksDirty[key] = value
	}
	return stateOrderList
}

// Retrieve a state object given by the address. Returns nil if not found.
func (s *liquidationPriceState) getStateLendingBook(db Database, lendingBook common.Hash) (stateObject *stateLendingBook) {
	// Prefer 'live' objects.
	if obj := s.stateLendingBooks[lendingBook]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := s.getTrie(db).TryGet(lendingBook[:])
	if len(enc) == 0 {
		s.setError(err)
		return nil
	}
	var data orderList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state lending book ", "orderbook", s.orderBook, "liquidation price", s.liquidationPrice, "lendingBook", lendingBook, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newStateLendingBook(s.orderBook, s.liquidationPrice, lendingBook, data, s.MarkStateLendingBookDirty)
	s.stateLendingBooks[lendingBook] = obj
	return obj
}

func (s *liquidationPriceState) getAllLiquidationData(db Database) map[common.Hash][]common.Hash {
	liquidationData := map[common.Hash][]common.Hash{}
	lendingBookTrie := s.getTrie(db)
	if lendingBookTrie == nil {
		return liquidationData
	}
	lendingBooks := []common.Hash{}
	for id, stateLendingBook := range s.stateLendingBooks {
		if !stateLendingBook.empty() {
			lendingBooks = append(lendingBooks, id)
		}
	}
	lendingBookListIt := trie.NewIterator(lendingBookTrie.NodeIterator(nil))
	for lendingBookListIt.Next() {
		id := common.BytesToHash(lendingBookListIt.Key)
		if _, exist := s.stateLendingBooks[id]; exist {
			continue
		}
		lendingBooks = append(lendingBooks, id)
	}
	for _, lendingBook := range lendingBooks {
		stateLendingBook := s.getStateLendingBook(db, lendingBook)
		if stateLendingBook != nil {
			liquidationData[lendingBook] = stateLendingBook.getAllTradeIds(db)
		}
	}
	return liquidationData
}

func (s *liquidationPriceState) AddVolume(amount *big.Int) {
	s.setVolume(new(big.Int).Add(s.data.Volume, amount))
}

func (s *liquidationPriceState) subVolume(amount *big.Int) {
	s.setVolume(new(big.Int).Sub(s.data.Volume, amount))
}

func (s *liquidationPriceState) setVolume(volume *big.Int) {
	s.data.Volume = volume
	if s.onDirty != nil {
		s.onDirty(s.liquidationPrice)
		s.onDirty = nil
	}
}

func (s *liquidationPriceState) Volume() *big.Int {
	return s.data.Volume
}
