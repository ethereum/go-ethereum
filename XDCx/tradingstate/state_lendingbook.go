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
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/XinFinOrg/XDPoSChain/trie"
)

type stateLendingBook struct {
	price       common.Hash
	orderBook   common.Hash
	lendingBook common.Hash
	data        orderList

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

	onDirty func(price common.Hash) // Callback method to mark a state object newly dirty
}

func (s *stateLendingBook) empty() bool {
	return s.data.Volume == nil || s.data.Volume.Sign() == 0
}

func newStateLendingBook(orderBook common.Hash, price common.Hash, lendingBook common.Hash, data orderList, onDirty func(price common.Hash)) *stateLendingBook {
	return &stateLendingBook{
		lendingBook:   lendingBook,
		orderBook:     orderBook,
		price:         price,
		data:          data,
		cachedStorage: make(map[common.Hash]common.Hash),
		dirtyStorage:  make(map[common.Hash]common.Hash),
		onDirty:       onDirty,
	}
}

func (s *stateLendingBook) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, s.data)
}

func (s *stateLendingBook) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

func (s *stateLendingBook) getTrie(db Database) Trie {
	if s.trie == nil {
		var err error
		s.trie, err = db.OpenStorageTrie(s.lendingBook, s.data.Root)
		if err != nil {
			s.trie, _ = db.OpenStorageTrie(s.price, types.EmptyRootHash)
			s.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return s.trie
}

func (s *stateLendingBook) Exist(db Database, lendingId common.Hash) bool {
	amount, exists := s.cachedStorage[lendingId]
	if exists {
		return true
	}
	// Load from DB in case it is missing.
	enc, err := s.getTrie(db).TryGet(lendingId[:])
	if err != nil {
		s.setError(err)
		return false
	}
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			s.setError(err)
		}
		amount.SetBytes(content)
	}
	if (amount != common.Hash{}) {
		s.cachedStorage[lendingId] = amount
	}
	return true
}

func (s *stateLendingBook) getAllTradeIds(db Database) []common.Hash {
	tradeIds := []common.Hash{}
	lendingBookTrie := s.getTrie(db)
	if lendingBookTrie == nil {
		return tradeIds
	}
	for id, value := range s.cachedStorage {
		if !value.IsZero() {
			tradeIds = append(tradeIds, id)
		}
	}
	orderListIt := trie.NewIterator(lendingBookTrie.NodeIterator(nil))
	for orderListIt.Next() {
		id := common.BytesToHash(orderListIt.Key)
		if _, exist := s.cachedStorage[id]; exist {
			continue
		}
		tradeIds = append(tradeIds, id)
	}
	return tradeIds
}

func (s *stateLendingBook) insertTradingId(db Database, tradeId common.Hash) {
	s.setTradingId(tradeId, tradeId)
	s.setError(s.getTrie(db).TryUpdate(tradeId[:], tradeId[:]))
}

func (s *stateLendingBook) removeTradingId(db Database, tradeId common.Hash) {
	tr := s.getTrie(db)
	s.setError(tr.TryDelete(tradeId[:]))
	s.setTradingId(tradeId, EmptyHash)
}

func (s *stateLendingBook) setTradingId(tradeId common.Hash, value common.Hash) {
	s.cachedStorage[tradeId] = value
	s.dirtyStorage[tradeId] = value

	if s.onDirty != nil {
		s.onDirty(s.lendingBook)
		s.onDirty = nil
	}
}

func (s *stateLendingBook) updateTrie(db Database) Trie {
	tr := s.getTrie(db)
	for key, value := range s.dirtyStorage {
		delete(s.dirtyStorage, key)
		if value == EmptyHash {
			s.setError(tr.TryDelete(key[:]))
			continue
		}
		v, _ := rlp.EncodeToBytes(bytes.TrimLeft(value[:], "\x00"))
		s.setError(tr.TryUpdate(key[:], v))
	}
	return tr
}

func (s *stateLendingBook) updateRoot(db Database) error {
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

func (s *stateLendingBook) deepCopy(db *TradingStateDB, onDirty func(price common.Hash)) *stateLendingBook {
	stateLendingBook := newStateLendingBook(s.lendingBook, s.orderBook, s.price, s.data, onDirty)
	if s.trie != nil {
		stateLendingBook.trie = db.db.CopyTrie(s.trie)
	}
	for key, value := range s.dirtyStorage {
		stateLendingBook.dirtyStorage[key] = value
	}
	for key, value := range s.cachedStorage {
		stateLendingBook.cachedStorage[key] = value
	}
	return stateLendingBook
}

func (s *stateLendingBook) AddVolume(amount *big.Int) {
	s.setVolume(new(big.Int).Add(s.data.Volume, amount))
}

func (s *stateLendingBook) subVolume(amount *big.Int) {
	s.setVolume(new(big.Int).Sub(s.data.Volume, amount))
}

func (s *stateLendingBook) setVolume(volume *big.Int) {
	s.data.Volume = volume
	if s.onDirty != nil {
		s.onDirty(s.lendingBook)
		s.onDirty = nil
	}
}

func (s *stateLendingBook) Volume() *big.Int {
	return s.data.Volume
}
