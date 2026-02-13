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
	"fmt"
	"io"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/rlp"
)

type lendingExchangeState struct {
	lendingBook common.Hash
	data        lendingObject
	db          *LendingStateDB

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by LendingStateDB.Commit.
	dbErr error

	investingTrie       Trie
	borrowingTrie       Trie
	lendingItemTrie     Trie
	lendingTradeTrie    Trie
	liquidationTimeTrie Trie

	liquidationTimeStates      map[common.Hash]*liquidationTimeState
	liquidationTimestatesDirty map[common.Hash]struct{}

	investingStates      map[common.Hash]*itemListState
	investingStatesDirty map[common.Hash]struct{}

	borrowingStates      map[common.Hash]*itemListState
	borrowingStatesDirty map[common.Hash]struct{}

	lendingItemStates      map[common.Hash]*lendingItemState
	lendingItemStatesDirty map[common.Hash]struct{}

	lendingTradeStates      map[common.Hash]*lendingTradeState
	lendingTradeStatesDirty map[common.Hash]struct{}

	onDirty func(hash common.Hash) // Callback method to mark a state object newly dirty
}

// empty returns whether the tradeId is considered empty.
func (s *lendingExchangeState) empty() bool {
	if s.data.Nonce != 0 {
		return false
	}
	if s.data.TradeNonce != 0 {
		return false
	}
	if !s.data.InvestingRoot.IsZero() {
		return false
	}
	if !s.data.BorrowingRoot.IsZero() {
		return false
	}
	if !s.data.LendingItemRoot.IsZero() {
		return false
	}
	if !s.data.LendingTradeRoot.IsZero() {
		return false
	}
	if !s.data.LiquidationTimeRoot.IsZero() {
		return false
	}
	return true
}

func newStateExchanges(db *LendingStateDB, hash common.Hash, data lendingObject, onDirty func(addr common.Hash)) *lendingExchangeState {
	return &lendingExchangeState{
		db:                         db,
		lendingBook:                hash,
		data:                       data,
		investingStates:            make(map[common.Hash]*itemListState),
		borrowingStates:            make(map[common.Hash]*itemListState),
		lendingItemStates:          make(map[common.Hash]*lendingItemState),
		lendingTradeStates:         make(map[common.Hash]*lendingTradeState),
		liquidationTimeStates:      make(map[common.Hash]*liquidationTimeState),
		investingStatesDirty:       make(map[common.Hash]struct{}),
		borrowingStatesDirty:       make(map[common.Hash]struct{}),
		lendingItemStatesDirty:     make(map[common.Hash]struct{}),
		lendingTradeStatesDirty:    make(map[common.Hash]struct{}),
		liquidationTimestatesDirty: make(map[common.Hash]struct{}),
		onDirty:                    onDirty,
	}
}

// EncodeRLP implements rlp.Encoder.
func (s *lendingExchangeState) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, s.data)
}

// setError remembers the first non-nil error it is called with.
func (s *lendingExchangeState) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

/**
  Get Trie
*/

func (s *lendingExchangeState) getLendingItemTrie(db Database) Trie {
	if s.lendingItemTrie == nil {
		var err error
		s.lendingItemTrie, err = db.OpenStorageTrie(s.lendingBook, common.Address{}, s.data.LendingItemRoot)
		if err != nil {
			s.lendingItemTrie, _ = db.OpenStorageTrie(s.lendingBook, common.Address{}, types.EmptyRootHash)
			s.setError(fmt.Errorf("can't create Lendings trie: %v", err))
		}
	}
	return s.lendingItemTrie
}

func (s *lendingExchangeState) getLendingTradeTrie(db Database) Trie {
	if s.lendingTradeTrie == nil {
		var err error
		s.lendingTradeTrie, err = db.OpenStorageTrie(s.lendingBook, common.Address{}, s.data.LendingTradeRoot)
		if err != nil {
			s.lendingTradeTrie, _ = db.OpenStorageTrie(s.lendingBook, common.Address{}, types.EmptyRootHash)
			s.setError(fmt.Errorf("can't create Lendings trie: %v", err))
		}
	}
	return s.lendingTradeTrie
}

func (s *lendingExchangeState) getInvestingTrie(db Database) Trie {
	if s.investingTrie == nil {
		var err error
		s.investingTrie, err = db.OpenStorageTrie(s.lendingBook, common.Address{}, s.data.InvestingRoot)
		if err != nil {
			s.investingTrie, _ = db.OpenStorageTrie(s.lendingBook, common.Address{}, types.EmptyRootHash)
			s.setError(fmt.Errorf("can't create Lendings trie: %v", err))
		}
	}
	return s.investingTrie
}

func (s *lendingExchangeState) getBorrowingTrie(db Database) Trie {
	if s.borrowingTrie == nil {
		var err error
		s.borrowingTrie, err = db.OpenStorageTrie(s.lendingBook, common.Address{}, s.data.BorrowingRoot)
		if err != nil {
			s.borrowingTrie, _ = db.OpenStorageTrie(s.lendingBook, common.Address{}, types.EmptyRootHash)
			s.setError(fmt.Errorf("can't create bids trie: %v", err))
		}
	}
	return s.borrowingTrie
}

func (s *lendingExchangeState) getLiquidationTimeTrie(db Database) Trie {
	if s.liquidationTimeTrie == nil {
		var err error
		s.liquidationTimeTrie, err = db.OpenStorageTrie(s.lendingBook, common.Address{}, s.data.LiquidationTimeRoot)
		if err != nil {
			s.liquidationTimeTrie, _ = db.OpenStorageTrie(s.lendingBook, common.Address{}, types.EmptyRootHash)
			s.setError(fmt.Errorf("can't create bids trie: %v", err))
		}
	}
	return s.liquidationTimeTrie
}

/*
*

	Get State
*/
func (s *lendingExchangeState) getBorrowingOrderList(db Database, rate common.Hash) (stateOrderList *itemListState) {
	// Prefer 'live' objects.
	if obj := s.borrowingStates[rate]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := s.getBorrowingTrie(db).TryGet(rate[:])
	if len(enc) == 0 {
		s.setError(err)
		return nil
	}
	var data itemList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state order list object", "rate", rate, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newItemListState(s.lendingBook, rate, data, s.MarkBorrowingDirty)
	s.borrowingStates[rate] = obj
	return obj
}

func (s *lendingExchangeState) getInvestingOrderList(db Database, rate common.Hash) (stateOrderList *itemListState) {
	// Prefer 'live' objects.
	if obj := s.investingStates[rate]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := s.getInvestingTrie(db).TryGet(rate[:])
	if len(enc) == 0 {
		s.setError(err)
		return nil
	}
	var data itemList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state order list object", "rate", rate, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newItemListState(s.lendingBook, rate, data, s.MarkInvestingDirty)
	s.investingStates[rate] = obj
	return obj
}

func (s *lendingExchangeState) getLiquidationTimeOrderList(db Database, time common.Hash) (stateObject *liquidationTimeState) {
	// Prefer 'live' objects.
	if obj := s.liquidationTimeStates[time]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := s.getLiquidationTimeTrie(db).TryGet(time[:])
	if len(enc) == 0 {
		s.setError(err)
		return nil
	}
	var data itemList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state liquidation time", "time", time, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newLiquidationTimeState(s.lendingBook, time, data, s.MarkLiquidationTimeDirty)
	s.liquidationTimeStates[time] = obj
	return obj
}

func (s *lendingExchangeState) getLendingItem(db Database, lendingId common.Hash) (stateObject *lendingItemState) {
	// Prefer 'live' objects.
	if obj := s.lendingItemStates[lendingId]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := s.getLendingItemTrie(db).TryGet(lendingId[:])
	if len(enc) == 0 {
		s.setError(err)
		return nil
	}
	var data LendingItem
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state lending item", "tradeId", lendingId, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newLendinItemState(s.lendingBook, lendingId, data, s.MarkLendingItemDirty)
	s.lendingItemStates[lendingId] = obj
	return obj
}

func (s *lendingExchangeState) getLendingTrade(db Database, tradeId common.Hash) (stateObject *lendingTradeState) {
	// Prefer 'live' objects.
	if obj := s.lendingTradeStates[tradeId]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := s.getLendingTradeTrie(db).TryGet(tradeId[:])
	if len(enc) == 0 {
		s.setError(err)
		return nil
	}
	var data LendingTrade
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state lending trade", "tradeId", tradeId, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newLendingTradeState(s.lendingBook, tradeId, data, s.MarkLendingTradeDirty)
	s.lendingTradeStates[tradeId] = obj
	return obj
}

/*
*

	Update Trie
*/
func (s *lendingExchangeState) updateLendingTimeTrie(db Database) Trie {
	tr := s.getLendingItemTrie(db)
	for lendingId, lendingItem := range s.lendingItemStates {
		if _, isDirty := s.lendingItemStatesDirty[lendingId]; isDirty {
			delete(s.lendingItemStatesDirty, lendingId)
			if lendingItem.empty() {
				s.setError(tr.TryDelete(lendingId[:]))
				continue
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(lendingItem)
			s.setError(tr.TryUpdate(lendingId[:], v))
		}
	}
	return tr
}

func (s *lendingExchangeState) updateLendingTradeTrie(db Database) Trie {
	tr := s.getLendingTradeTrie(db)
	for tradeId, lendingTradeItem := range s.lendingTradeStates {
		if _, isDirty := s.lendingTradeStatesDirty[tradeId]; isDirty {
			delete(s.lendingTradeStatesDirty, tradeId)
			if lendingTradeItem.empty() {
				s.setError(tr.TryDelete(tradeId[:]))
				continue
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(lendingTradeItem)
			s.setError(tr.TryUpdate(tradeId[:], v))
		}
	}
	return tr
}

func (s *lendingExchangeState) updateBorrowingTrie(db Database) Trie {
	tr := s.getBorrowingTrie(db)
	for rate, orderList := range s.borrowingStates {
		if _, isDirty := s.borrowingStatesDirty[rate]; isDirty {
			delete(s.borrowingStatesDirty, rate)
			if orderList.empty() {
				s.setError(tr.TryDelete(rate[:]))
				continue
			}
			err := orderList.updateRoot(db)
			if err != nil {
				log.Warn("updateBorrowingTrie updateRoot", "err", err, "rate", rate, "orderList", *orderList)
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(orderList)
			s.setError(tr.TryUpdate(rate[:], v))
		}
	}
	return tr
}

func (s *lendingExchangeState) updateInvestingTrie(db Database) Trie {
	tr := s.getInvestingTrie(db)
	for rate, orderList := range s.investingStates {
		if _, isDirty := s.investingStatesDirty[rate]; isDirty {
			delete(s.investingStatesDirty, rate)
			if orderList.empty() {
				s.setError(tr.TryDelete(rate[:]))
				continue
			}
			err := orderList.updateRoot(db)
			if err != nil {
				log.Warn("updateInvestingTrie updateRoot", "err", err, "rate", rate, "orderList", *orderList)
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(orderList)
			s.setError(tr.TryUpdate(rate[:], v))
		}
	}
	return tr
}

func (s *lendingExchangeState) updateLiquidationTimeTrie(db Database) Trie {
	tr := s.getLiquidationTimeTrie(db)
	for time, itemList := range s.liquidationTimeStates {
		if _, isDirty := s.liquidationTimestatesDirty[time]; isDirty {
			delete(s.liquidationTimestatesDirty, time)
			if itemList.empty() {
				s.setError(tr.TryDelete(time[:]))
				continue
			}
			err := itemList.updateRoot(db)
			if err != nil {
				log.Warn("updateLiquidationTimeTrie updateRoot", "err", err, "time", time, "itemList", *itemList)
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(itemList)
			s.setError(tr.TryUpdate(time[:], v))
		}
	}
	return tr
}

/**
  Update Root
*/

func (s *lendingExchangeState) updateOrderRoot(db Database) {
	s.updateLendingTimeTrie(db)
	s.data.LendingItemRoot = s.lendingItemTrie.Hash()
}

func (s *lendingExchangeState) updateInvestingRoot(db Database) error {
	s.updateInvestingTrie(db)
	if s.dbErr != nil {
		return s.dbErr
	}
	s.data.InvestingRoot = s.investingTrie.Hash()
	return nil
}

func (s *lendingExchangeState) updateBorrowingRoot(db Database) {
	s.updateBorrowingTrie(db)
	s.data.BorrowingRoot = s.borrowingTrie.Hash()
}

func (s *lendingExchangeState) updateLiquidationTimeRoot(db Database) {
	s.updateLiquidationTimeTrie(db)
	s.data.LiquidationTimeRoot = s.liquidationTimeTrie.Hash()
}

func (s *lendingExchangeState) updateLendingTradeRoot(db Database) {
	s.updateLendingTradeTrie(db)
	s.data.LendingTradeRoot = s.lendingTradeTrie.Hash()
}

/**
  Commit Trie
*/

func (s *lendingExchangeState) CommitLendingItemTrie(db Database) error {
	s.updateLendingTimeTrie(db)
	if s.dbErr != nil {
		return s.dbErr
	}
	root, err := s.lendingItemTrie.Commit(nil)
	if err == nil {
		s.data.LendingItemRoot = root
	}
	return err
}

func (s *lendingExchangeState) CommitLendingTradeTrie(db Database) error {
	s.updateLendingTradeTrie(db)
	if s.dbErr != nil {
		return s.dbErr
	}
	root, err := s.lendingTradeTrie.Commit(nil)
	if err == nil {
		s.data.LendingTradeRoot = root
	}
	return err
}

func (s *lendingExchangeState) CommitInvestingTrie(db Database) error {
	s.updateInvestingTrie(db)
	if s.dbErr != nil {
		return s.dbErr
	}
	root, err := s.investingTrie.Commit(func(_ [][]byte, _ []byte, leaf []byte, parent common.Hash, _ []byte) error {
		var orderList itemList
		if err := rlp.DecodeBytes(leaf, &orderList); err != nil {
			return nil
		}
		if orderList.Root != EmptyRoot {
			db.TrieDB().Reference(orderList.Root, parent)
		}
		return nil
	})
	if err == nil {
		s.data.InvestingRoot = root
	}
	return err
}

func (s *lendingExchangeState) CommitBorrowingTrie(db Database) error {
	s.updateBorrowingTrie(db)
	if s.dbErr != nil {
		return s.dbErr
	}
	root, err := s.borrowingTrie.Commit(func(_ [][]byte, _ []byte, leaf []byte, parent common.Hash, _ []byte) error {
		var orderList itemList
		if err := rlp.DecodeBytes(leaf, &orderList); err != nil {
			return nil
		}
		if orderList.Root != EmptyRoot {
			db.TrieDB().Reference(orderList.Root, parent)
		}
		return nil
	})
	if err == nil {
		s.data.BorrowingRoot = root
	}
	return err
}

func (s *lendingExchangeState) CommitLiquidationTimeTrie(db Database) error {
	s.updateLiquidationTimeTrie(db)
	if s.dbErr != nil {
		return s.dbErr
	}
	root, err := s.liquidationTimeTrie.Commit(func(_ [][]byte, _ []byte, leaf []byte, parent common.Hash, _ []byte) error {
		var orderList itemList
		if err := rlp.DecodeBytes(leaf, &orderList); err != nil {
			return nil
		}
		if orderList.Root != EmptyRoot {
			db.TrieDB().Reference(orderList.Root, parent)
		}
		return nil
	})
	if err == nil {
		s.data.LiquidationTimeRoot = root
	}
	return err
}

/*
*

	Get Trie Data
*/
func (s *lendingExchangeState) getBestInvestingInterest(db Database) common.Hash {
	trie := s.getInvestingTrie(db)
	encKey, encValue, err := trie.TryGetBestLeftKeyAndValue()
	if err != nil {
		log.Error("Failed find best investing rate", "orderbook", s.lendingBook.Hex())
		return EmptyHash
	}
	if len(encKey) == 0 || len(encValue) == 0 {
		log.Debug("Not found get best investing rate", "encKey", encKey, "encValue", encValue)
		return EmptyHash
	}
	// Insert into the live set.
	interest := common.BytesToHash(encKey)
	if _, exist := s.investingStates[interest]; !exist {
		var data itemList
		if err := rlp.DecodeBytes(encValue, &data); err != nil {
			log.Error("Failed to decode state get best investing rate", "err", err)
			return EmptyHash
		}
		obj := newItemListState(s.lendingBook, interest, data, s.MarkInvestingDirty)
		s.investingStates[interest] = obj
	}
	return interest
}

func (s *lendingExchangeState) getBestBorrowingInterest(db Database) common.Hash {
	trie := s.getBorrowingTrie(db)
	encKey, encValue, err := trie.TryGetBestRightKeyAndValue()
	if err != nil {
		log.Error("Failed find best key bid trie ", "orderbook", s.lendingBook.Hex())
		return EmptyHash
	}
	if len(encKey) == 0 || len(encValue) == 0 {
		log.Debug("Not found get best bid trie", "encKey", encKey, "encValue", encValue)
		return EmptyHash
	}
	// Insert into the live set.
	interest := common.BytesToHash(encKey)
	if _, exist := s.borrowingStates[interest]; !exist {
		var data itemList
		if err := rlp.DecodeBytes(encValue, &data); err != nil {
			log.Error("Failed to decode state get best bid trie", "err", err)
			return EmptyHash
		}
		obj := newItemListState(s.lendingBook, interest, data, s.MarkBorrowingDirty)
		s.borrowingStates[interest] = obj
	}
	return interest
}

func (s *lendingExchangeState) getLowestLiquidationTime(db Database) (common.Hash, *liquidationTimeState) {
	trie := s.getLiquidationTimeTrie(db)
	encKey, encValue, err := trie.TryGetBestLeftKeyAndValue()
	if err != nil {
		log.Error("Failed find best liquidation time trie ", "orderBook", s.lendingBook.Hex())
		return EmptyHash, nil
	}
	if len(encKey) == 0 || len(encValue) == 0 {
		log.Debug("Not found get liquidation time trie", "encKey", encKey, "encValue", encValue)
		return EmptyHash, nil
	}
	price := common.BytesToHash(encKey)
	obj, exist := s.liquidationTimeStates[price]
	if !exist {
		var data itemList
		if err := rlp.DecodeBytes(encValue, &data); err != nil {
			log.Error("Failed to decode state get liquidation time trie", "err", err)
			return EmptyHash, nil
		}
		obj = newLiquidationTimeState(s.lendingBook, price, data, s.MarkLiquidationTimeDirty)
		s.liquidationTimeStates[price] = obj
	}
	if obj.empty() {
		return EmptyHash, nil
	}
	return price, obj
}

func (s *lendingExchangeState) deepCopy(db *LendingStateDB, onDirty func(hash common.Hash)) *lendingExchangeState {
	stateExchanges := newStateExchanges(db, s.lendingBook, s.data, onDirty)
	if s.investingTrie != nil {
		stateExchanges.investingTrie = db.db.CopyTrie(s.investingTrie)
	}
	if s.borrowingTrie != nil {
		stateExchanges.borrowingTrie = db.db.CopyTrie(s.borrowingTrie)
	}
	if s.lendingItemTrie != nil {
		stateExchanges.lendingItemTrie = db.db.CopyTrie(s.lendingItemTrie)
	}
	for key, value := range s.borrowingStates {
		stateExchanges.borrowingStates[key] = value.deepCopy(db, s.MarkBorrowingDirty)
	}
	for key := range s.borrowingStatesDirty {
		stateExchanges.borrowingStatesDirty[key] = struct{}{}
	}
	for key, value := range s.investingStates {
		stateExchanges.investingStates[key] = value.deepCopy(db, s.MarkInvestingDirty)
	}
	for key := range s.investingStatesDirty {
		stateExchanges.investingStatesDirty[key] = struct{}{}
	}
	for key, value := range s.lendingItemStates {
		stateExchanges.lendingItemStates[key] = value.deepCopy(s.MarkLendingItemDirty)
	}
	for orderId := range s.lendingItemStatesDirty {
		stateExchanges.lendingItemStatesDirty[orderId] = struct{}{}
	}
	for key, value := range s.lendingTradeStates {
		stateExchanges.lendingTradeStates[key] = value.deepCopy(s.MarkLendingTradeDirty)
	}
	for orderId := range s.lendingTradeStatesDirty {
		stateExchanges.lendingTradeStatesDirty[orderId] = struct{}{}
	}
	for time, orderList := range s.liquidationTimeStates {
		stateExchanges.liquidationTimeStates[time] = orderList.deepCopy(db, s.MarkLiquidationTimeDirty)
	}
	for time := range s.liquidationTimestatesDirty {
		stateExchanges.liquidationTimestatesDirty[time] = struct{}{}
	}
	return stateExchanges
}

// Returns the address of the contract/tradeId
func (s *lendingExchangeState) Hash() common.Hash {
	return s.lendingBook
}

func (s *lendingExchangeState) setNonce(nonce uint64) {
	s.data.Nonce = nonce
	if s.onDirty != nil {
		s.onDirty(s.Hash())
		s.onDirty = nil
	}
}

func (s *lendingExchangeState) Nonce() uint64 {
	return s.data.Nonce
}

func (s *lendingExchangeState) setTradeNonce(nonce uint64) {
	s.data.TradeNonce = nonce
	if s.onDirty != nil {
		s.onDirty(s.Hash())
		s.onDirty = nil
	}
}

func (s *lendingExchangeState) TradeNonce() uint64 {
	return s.data.TradeNonce
}

func (s *lendingExchangeState) removeInvestingOrderList(db Database, stateOrderList *itemListState) {
	s.setError(s.investingTrie.TryDelete(stateOrderList.key[:]))
}

func (s *lendingExchangeState) removeBorrowingOrderList(db Database, stateOrderList *itemListState) {
	s.setError(s.borrowingTrie.TryDelete(stateOrderList.key[:]))
}

func (s *lendingExchangeState) createInvestingOrderList(db Database, price common.Hash) (newobj *itemListState) {
	newobj = newItemListState(s.lendingBook, price, itemList{Volume: Zero}, s.MarkInvestingDirty)
	s.investingStates[price] = newobj
	s.investingStatesDirty[price] = struct{}{}
	data, err := rlp.EncodeToBytes(newobj)
	if err != nil {
		panic(fmt.Errorf("can't encode order list object at %x: %v", price[:], err))
	}
	s.setError(s.getInvestingTrie(db).TryUpdate(price[:], data))
	if s.onDirty != nil {
		s.onDirty(s.Hash())
		s.onDirty = nil
	}
	return newobj
}

func (s *lendingExchangeState) MarkBorrowingDirty(price common.Hash) {
	s.borrowingStatesDirty[price] = struct{}{}
	if s.onDirty != nil {
		s.onDirty(s.Hash())
		s.onDirty = nil
	}
}

func (s *lendingExchangeState) MarkInvestingDirty(price common.Hash) {
	s.investingStatesDirty[price] = struct{}{}
	if s.onDirty != nil {
		s.onDirty(s.Hash())
		s.onDirty = nil
	}
}

func (s *lendingExchangeState) MarkLendingItemDirty(lending common.Hash) {
	s.lendingItemStatesDirty[lending] = struct{}{}
	if s.onDirty != nil {
		s.onDirty(s.Hash())
		s.onDirty = nil
	}
}

func (s *lendingExchangeState) MarkLendingTradeDirty(tradeId common.Hash) {
	s.lendingTradeStatesDirty[tradeId] = struct{}{}
	if s.onDirty != nil {
		s.onDirty(s.Hash())
		s.onDirty = nil
	}
}

func (s *lendingExchangeState) MarkLiquidationTimeDirty(orderId common.Hash) {
	s.liquidationTimestatesDirty[orderId] = struct{}{}
	if s.onDirty != nil {
		s.onDirty(s.Hash())
		s.onDirty = nil
	}
}

func (s *lendingExchangeState) createBorrowingOrderList(db Database, price common.Hash) (newobj *itemListState) {
	newobj = newItemListState(s.lendingBook, price, itemList{Volume: Zero}, s.MarkBorrowingDirty)
	s.borrowingStates[price] = newobj
	s.borrowingStatesDirty[price] = struct{}{}
	data, err := rlp.EncodeToBytes(newobj)
	if err != nil {
		panic(fmt.Errorf("can't encode order list object at %x: %v", price[:], err))
	}
	s.setError(s.getBorrowingTrie(db).TryUpdate(price[:], data))
	if s.onDirty != nil {
		s.onDirty(s.Hash())
		s.onDirty = nil
	}
	return newobj
}

func (s *lendingExchangeState) createLendingItem(db Database, orderId common.Hash, order LendingItem) (newobj *lendingItemState) {
	newobj = newLendinItemState(s.lendingBook, orderId, order, s.MarkLendingItemDirty)
	orderIdHash := common.BigToHash(new(big.Int).SetUint64(order.LendingId))
	s.lendingItemStates[orderIdHash] = newobj
	s.lendingItemStatesDirty[orderIdHash] = struct{}{}
	if s.onDirty != nil {
		s.onDirty(s.lendingBook)
		s.onDirty = nil
	}
	return newobj
}

func (s *lendingExchangeState) createLiquidationTime(db Database, time common.Hash) (newobj *liquidationTimeState) {
	newobj = newLiquidationTimeState(time, s.lendingBook, itemList{Volume: Zero}, s.MarkLiquidationTimeDirty)
	s.liquidationTimeStates[time] = newobj
	s.liquidationTimestatesDirty[time] = struct{}{}
	data, err := rlp.EncodeToBytes(newobj)
	if err != nil {
		panic(fmt.Errorf("can't encode liquidation time at %x: %v", time[:], err))
	}
	s.setError(s.getLiquidationTimeTrie(db).TryUpdate(time[:], data))
	if s.onDirty != nil {
		s.onDirty(s.lendingBook)
		s.onDirty = nil
	}
	return newobj
}

func (s *lendingExchangeState) insertLendingTrade(tradeId common.Hash, order LendingTrade) (newobj *lendingTradeState) {
	newobj = newLendingTradeState(s.lendingBook, tradeId, order, s.MarkLendingTradeDirty)
	s.lendingTradeStates[tradeId] = newobj
	s.lendingTradeStatesDirty[tradeId] = struct{}{}
	if s.onDirty != nil {
		s.onDirty(s.lendingBook)
		s.onDirty = nil
	}
	return newobj
}
