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
func (le *lendingExchangeState) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, le.data)
}

// setError remembers the first non-nil error it is called with.
func (le *lendingExchangeState) setError(err error) {
	if le.dbErr == nil {
		le.dbErr = err
	}
}

/**
  Get Trie
*/

func (le *lendingExchangeState) getLendingItemTrie(db Database) Trie {
	if le.lendingItemTrie == nil {
		var err error
		le.lendingItemTrie, err = db.OpenStorageTrie(le.lendingBook, le.data.LendingItemRoot)
		if err != nil {
			le.lendingItemTrie, _ = db.OpenStorageTrie(le.lendingBook, types.EmptyRootHash)
			le.setError(fmt.Errorf("can't create Lendings trie: %v", err))
		}
	}
	return le.lendingItemTrie
}

func (le *lendingExchangeState) getLendingTradeTrie(db Database) Trie {
	if le.lendingTradeTrie == nil {
		var err error
		le.lendingTradeTrie, err = db.OpenStorageTrie(le.lendingBook, le.data.LendingTradeRoot)
		if err != nil {
			le.lendingTradeTrie, _ = db.OpenStorageTrie(le.lendingBook, types.EmptyRootHash)
			le.setError(fmt.Errorf("can't create Lendings trie: %v", err))
		}
	}
	return le.lendingTradeTrie
}

func (le *lendingExchangeState) getInvestingTrie(db Database) Trie {
	if le.investingTrie == nil {
		var err error
		le.investingTrie, err = db.OpenStorageTrie(le.lendingBook, le.data.InvestingRoot)
		if err != nil {
			le.investingTrie, _ = db.OpenStorageTrie(le.lendingBook, types.EmptyRootHash)
			le.setError(fmt.Errorf("can't create Lendings trie: %v", err))
		}
	}
	return le.investingTrie
}

func (le *lendingExchangeState) getBorrowingTrie(db Database) Trie {
	if le.borrowingTrie == nil {
		var err error
		le.borrowingTrie, err = db.OpenStorageTrie(le.lendingBook, le.data.BorrowingRoot)
		if err != nil {
			le.borrowingTrie, _ = db.OpenStorageTrie(le.lendingBook, types.EmptyRootHash)
			le.setError(fmt.Errorf("can't create bids trie: %v", err))
		}
	}
	return le.borrowingTrie
}

func (le *lendingExchangeState) getLiquidationTimeTrie(db Database) Trie {
	if le.liquidationTimeTrie == nil {
		var err error
		le.liquidationTimeTrie, err = db.OpenStorageTrie(le.lendingBook, le.data.LiquidationTimeRoot)
		if err != nil {
			le.liquidationTimeTrie, _ = db.OpenStorageTrie(le.lendingBook, types.EmptyRootHash)
			le.setError(fmt.Errorf("can't create bids trie: %v", err))
		}
	}
	return le.liquidationTimeTrie
}

/*
*

	Get State
*/
func (le *lendingExchangeState) getBorrowingOrderList(db Database, rate common.Hash) (stateOrderList *itemListState) {
	// Prefer 'live' objects.
	if obj := le.borrowingStates[rate]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := le.getBorrowingTrie(db).TryGet(rate[:])
	if len(enc) == 0 {
		le.setError(err)
		return nil
	}
	var data itemList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state order list object", "rate", rate, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newItemListState(le.lendingBook, rate, data, le.MarkBorrowingDirty)
	le.borrowingStates[rate] = obj
	return obj
}

func (le *lendingExchangeState) getInvestingOrderList(db Database, rate common.Hash) (stateOrderList *itemListState) {
	// Prefer 'live' objects.
	if obj := le.investingStates[rate]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := le.getInvestingTrie(db).TryGet(rate[:])
	if len(enc) == 0 {
		le.setError(err)
		return nil
	}
	var data itemList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state order list object", "rate", rate, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newItemListState(le.lendingBook, rate, data, le.MarkInvestingDirty)
	le.investingStates[rate] = obj
	return obj
}

func (le *lendingExchangeState) getLiquidationTimeOrderList(db Database, time common.Hash) (stateObject *liquidationTimeState) {
	// Prefer 'live' objects.
	if obj := le.liquidationTimeStates[time]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := le.getLiquidationTimeTrie(db).TryGet(time[:])
	if len(enc) == 0 {
		le.setError(err)
		return nil
	}
	var data itemList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state liquidation time", "time", time, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newLiquidationTimeState(le.lendingBook, time, data, le.MarkLiquidationTimeDirty)
	le.liquidationTimeStates[time] = obj
	return obj
}

func (le *lendingExchangeState) getLendingItem(db Database, lendingId common.Hash) (stateObject *lendingItemState) {
	// Prefer 'live' objects.
	if obj := le.lendingItemStates[lendingId]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := le.getLendingItemTrie(db).TryGet(lendingId[:])
	if len(enc) == 0 {
		le.setError(err)
		return nil
	}
	var data LendingItem
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state lending item", "tradeId", lendingId, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newLendinItemState(le.lendingBook, lendingId, data, le.MarkLendingItemDirty)
	le.lendingItemStates[lendingId] = obj
	return obj
}

func (le *lendingExchangeState) getLendingTrade(db Database, tradeId common.Hash) (stateObject *lendingTradeState) {
	// Prefer 'live' objects.
	if obj := le.lendingTradeStates[tradeId]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := le.getLendingTradeTrie(db).TryGet(tradeId[:])
	if len(enc) == 0 {
		le.setError(err)
		return nil
	}
	var data LendingTrade
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state lending trade", "tradeId", tradeId, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newLendingTradeState(le.lendingBook, tradeId, data, le.MarkLendingTradeDirty)
	le.lendingTradeStates[tradeId] = obj
	return obj
}

/*
*

	Update Trie
*/
func (le *lendingExchangeState) updateLendingTimeTrie(db Database) Trie {
	tr := le.getLendingItemTrie(db)
	for lendingId, lendingItem := range le.lendingItemStates {
		if _, isDirty := le.lendingItemStatesDirty[lendingId]; isDirty {
			delete(le.lendingItemStatesDirty, lendingId)
			if lendingItem.empty() {
				le.setError(tr.TryDelete(lendingId[:]))
				continue
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(lendingItem)
			le.setError(tr.TryUpdate(lendingId[:], v))
		}
	}
	return tr
}

func (le *lendingExchangeState) updateLendingTradeTrie(db Database) Trie {
	tr := le.getLendingTradeTrie(db)
	for tradeId, lendingTradeItem := range le.lendingTradeStates {
		if _, isDirty := le.lendingTradeStatesDirty[tradeId]; isDirty {
			delete(le.lendingTradeStatesDirty, tradeId)
			if lendingTradeItem.empty() {
				le.setError(tr.TryDelete(tradeId[:]))
				continue
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(lendingTradeItem)
			le.setError(tr.TryUpdate(tradeId[:], v))
		}
	}
	return tr
}

func (le *lendingExchangeState) updateBorrowingTrie(db Database) Trie {
	tr := le.getBorrowingTrie(db)
	for rate, orderList := range le.borrowingStates {
		if _, isDirty := le.borrowingStatesDirty[rate]; isDirty {
			delete(le.borrowingStatesDirty, rate)
			if orderList.empty() {
				le.setError(tr.TryDelete(rate[:]))
				continue
			}
			err := orderList.updateRoot(db)
			if err != nil {
				log.Warn("updateBorrowingTrie updateRoot", "err", err, "rate", rate, "orderList", *orderList)
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(orderList)
			le.setError(tr.TryUpdate(rate[:], v))
		}
	}
	return tr
}

func (le *lendingExchangeState) updateInvestingTrie(db Database) Trie {
	tr := le.getInvestingTrie(db)
	for rate, orderList := range le.investingStates {
		if _, isDirty := le.investingStatesDirty[rate]; isDirty {
			delete(le.investingStatesDirty, rate)
			if orderList.empty() {
				le.setError(tr.TryDelete(rate[:]))
				continue
			}
			err := orderList.updateRoot(db)
			if err != nil {
				log.Warn("updateInvestingTrie updateRoot", "err", err, "rate", rate, "orderList", *orderList)
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(orderList)
			le.setError(tr.TryUpdate(rate[:], v))
		}
	}
	return tr
}

func (le *lendingExchangeState) updateLiquidationTimeTrie(db Database) Trie {
	tr := le.getLiquidationTimeTrie(db)
	for time, itemList := range le.liquidationTimeStates {
		if _, isDirty := le.liquidationTimestatesDirty[time]; isDirty {
			delete(le.liquidationTimestatesDirty, time)
			if itemList.empty() {
				le.setError(tr.TryDelete(time[:]))
				continue
			}
			err := itemList.updateRoot(db)
			if err != nil {
				log.Warn("updateLiquidationTimeTrie updateRoot", "err", err, "time", time, "itemList", *itemList)
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(itemList)
			le.setError(tr.TryUpdate(time[:], v))
		}
	}
	return tr
}

/**
  Update Root
*/

func (le *lendingExchangeState) updateOrderRoot(db Database) {
	le.updateLendingTimeTrie(db)
	le.data.LendingItemRoot = le.lendingItemTrie.Hash()
}

func (le *lendingExchangeState) updateInvestingRoot(db Database) error {
	le.updateInvestingTrie(db)
	if le.dbErr != nil {
		return le.dbErr
	}
	le.data.InvestingRoot = le.investingTrie.Hash()
	return nil
}

func (le *lendingExchangeState) updateBorrowingRoot(db Database) {
	le.updateBorrowingTrie(db)
	le.data.BorrowingRoot = le.borrowingTrie.Hash()
}

func (le *lendingExchangeState) updateLiquidationTimeRoot(db Database) {
	le.updateLiquidationTimeTrie(db)
	le.data.LiquidationTimeRoot = le.liquidationTimeTrie.Hash()
}

func (le *lendingExchangeState) updateLendingTradeRoot(db Database) {
	le.updateLendingTradeTrie(db)
	le.data.LendingTradeRoot = le.lendingTradeTrie.Hash()
}

/**
  Commit Trie
*/

func (le *lendingExchangeState) CommitLendingItemTrie(db Database) error {
	le.updateLendingTimeTrie(db)
	if le.dbErr != nil {
		return le.dbErr
	}
	root, err := le.lendingItemTrie.Commit(nil)
	if err == nil {
		le.data.LendingItemRoot = root
	}
	return err
}

func (le *lendingExchangeState) CommitLendingTradeTrie(db Database) error {
	le.updateLendingTradeTrie(db)
	if le.dbErr != nil {
		return le.dbErr
	}
	root, err := le.lendingTradeTrie.Commit(nil)
	if err == nil {
		le.data.LendingTradeRoot = root
	}
	return err
}

func (le *lendingExchangeState) CommitInvestingTrie(db Database) error {
	le.updateInvestingTrie(db)
	if le.dbErr != nil {
		return le.dbErr
	}
	root, err := le.investingTrie.Commit(func(leaf []byte, parent common.Hash) error {
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
		le.data.InvestingRoot = root
	}
	return err
}

func (le *lendingExchangeState) CommitBorrowingTrie(db Database) error {
	le.updateBorrowingTrie(db)
	if le.dbErr != nil {
		return le.dbErr
	}
	root, err := le.borrowingTrie.Commit(func(leaf []byte, parent common.Hash) error {
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
		le.data.BorrowingRoot = root
	}
	return err
}

func (le *lendingExchangeState) CommitLiquidationTimeTrie(db Database) error {
	le.updateLiquidationTimeTrie(db)
	if le.dbErr != nil {
		return le.dbErr
	}
	root, err := le.liquidationTimeTrie.Commit(func(leaf []byte, parent common.Hash) error {
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
		le.data.LiquidationTimeRoot = root
	}
	return err
}

/*
*

	Get Trie Data
*/
func (le *lendingExchangeState) getBestInvestingInterest(db Database) common.Hash {
	trie := le.getInvestingTrie(db)
	encKey, encValue, err := trie.TryGetBestLeftKeyAndValue()
	if err != nil {
		log.Error("Failed find best investing rate", "orderbook", le.lendingBook.Hex())
		return EmptyHash
	}
	if len(encKey) == 0 || len(encValue) == 0 {
		log.Debug("Not found get best investing rate", "encKey", encKey, "encValue", encValue)
		return EmptyHash
	}
	// Insert into the live set.
	interest := common.BytesToHash(encKey)
	if _, exist := le.investingStates[interest]; !exist {
		var data itemList
		if err := rlp.DecodeBytes(encValue, &data); err != nil {
			log.Error("Failed to decode state get best investing rate", "err", err)
			return EmptyHash
		}
		obj := newItemListState(le.lendingBook, interest, data, le.MarkInvestingDirty)
		le.investingStates[interest] = obj
	}
	return interest
}

func (le *lendingExchangeState) getBestBorrowingInterest(db Database) common.Hash {
	trie := le.getBorrowingTrie(db)
	encKey, encValue, err := trie.TryGetBestRightKeyAndValue()
	if err != nil {
		log.Error("Failed find best key bid trie ", "orderbook", le.lendingBook.Hex())
		return EmptyHash
	}
	if len(encKey) == 0 || len(encValue) == 0 {
		log.Debug("Not found get best bid trie", "encKey", encKey, "encValue", encValue)
		return EmptyHash
	}
	// Insert into the live set.
	interest := common.BytesToHash(encKey)
	if _, exist := le.borrowingStates[interest]; !exist {
		var data itemList
		if err := rlp.DecodeBytes(encValue, &data); err != nil {
			log.Error("Failed to decode state get best bid trie", "err", err)
			return EmptyHash
		}
		obj := newItemListState(le.lendingBook, interest, data, le.MarkBorrowingDirty)
		le.borrowingStates[interest] = obj
	}
	return interest
}

func (le *lendingExchangeState) getLowestLiquidationTime(db Database) (common.Hash, *liquidationTimeState) {
	trie := le.getLiquidationTimeTrie(db)
	encKey, encValue, err := trie.TryGetBestLeftKeyAndValue()
	if err != nil {
		log.Error("Failed find best liquidation time trie ", "orderBook", le.lendingBook.Hex())
		return EmptyHash, nil
	}
	if len(encKey) == 0 || len(encValue) == 0 {
		log.Debug("Not found get liquidation time trie", "encKey", encKey, "encValue", encValue)
		return EmptyHash, nil
	}
	price := common.BytesToHash(encKey)
	obj, exist := le.liquidationTimeStates[price]
	if !exist {
		var data itemList
		if err := rlp.DecodeBytes(encValue, &data); err != nil {
			log.Error("Failed to decode state get liquidation time trie", "err", err)
			return EmptyHash, nil
		}
		obj = newLiquidationTimeState(le.lendingBook, price, data, le.MarkLiquidationTimeDirty)
		le.liquidationTimeStates[price] = obj
	}
	if obj.empty() {
		return EmptyHash, nil
	}
	return price, obj
}

func (le *lendingExchangeState) deepCopy(db *LendingStateDB, onDirty func(hash common.Hash)) *lendingExchangeState {
	stateExchanges := newStateExchanges(db, le.lendingBook, le.data, onDirty)
	if le.investingTrie != nil {
		stateExchanges.investingTrie = db.db.CopyTrie(le.investingTrie)
	}
	if le.borrowingTrie != nil {
		stateExchanges.borrowingTrie = db.db.CopyTrie(le.borrowingTrie)
	}
	if le.lendingItemTrie != nil {
		stateExchanges.lendingItemTrie = db.db.CopyTrie(le.lendingItemTrie)
	}
	for key, value := range le.borrowingStates {
		stateExchanges.borrowingStates[key] = value.deepCopy(db, le.MarkBorrowingDirty)
	}
	for key := range le.borrowingStatesDirty {
		stateExchanges.borrowingStatesDirty[key] = struct{}{}
	}
	for key, value := range le.investingStates {
		stateExchanges.investingStates[key] = value.deepCopy(db, le.MarkInvestingDirty)
	}
	for key := range le.investingStatesDirty {
		stateExchanges.investingStatesDirty[key] = struct{}{}
	}
	for key, value := range le.lendingItemStates {
		stateExchanges.lendingItemStates[key] = value.deepCopy(le.MarkLendingItemDirty)
	}
	for orderId := range le.lendingItemStatesDirty {
		stateExchanges.lendingItemStatesDirty[orderId] = struct{}{}
	}
	for key, value := range le.lendingTradeStates {
		stateExchanges.lendingTradeStates[key] = value.deepCopy(le.MarkLendingTradeDirty)
	}
	for orderId := range le.lendingTradeStatesDirty {
		stateExchanges.lendingTradeStatesDirty[orderId] = struct{}{}
	}
	for time, orderList := range le.liquidationTimeStates {
		stateExchanges.liquidationTimeStates[time] = orderList.deepCopy(db, le.MarkLiquidationTimeDirty)
	}
	for time := range le.liquidationTimestatesDirty {
		stateExchanges.liquidationTimestatesDirty[time] = struct{}{}
	}
	return stateExchanges
}

// Returns the address of the contract/tradeId
func (le *lendingExchangeState) Hash() common.Hash {
	return le.lendingBook
}

func (le *lendingExchangeState) setNonce(nonce uint64) {
	le.data.Nonce = nonce
	if le.onDirty != nil {
		le.onDirty(le.Hash())
		le.onDirty = nil
	}
}

func (le *lendingExchangeState) Nonce() uint64 {
	return le.data.Nonce
}

func (le *lendingExchangeState) setTradeNonce(nonce uint64) {
	le.data.TradeNonce = nonce
	if le.onDirty != nil {
		le.onDirty(le.Hash())
		le.onDirty = nil
	}
}

func (le *lendingExchangeState) TradeNonce() uint64 {
	return le.data.TradeNonce
}

func (le *lendingExchangeState) removeInvestingOrderList(db Database, stateOrderList *itemListState) {
	le.setError(le.investingTrie.TryDelete(stateOrderList.key[:]))
}

func (le *lendingExchangeState) removeBorrowingOrderList(db Database, stateOrderList *itemListState) {
	le.setError(le.borrowingTrie.TryDelete(stateOrderList.key[:]))
}

func (le *lendingExchangeState) createInvestingOrderList(db Database, price common.Hash) (newobj *itemListState) {
	newobj = newItemListState(le.lendingBook, price, itemList{Volume: Zero}, le.MarkInvestingDirty)
	le.investingStates[price] = newobj
	le.investingStatesDirty[price] = struct{}{}
	data, err := rlp.EncodeToBytes(newobj)
	if err != nil {
		panic(fmt.Errorf("can't encode order list object at %x: %v", price[:], err))
	}
	le.setError(le.getInvestingTrie(db).TryUpdate(price[:], data))
	if le.onDirty != nil {
		le.onDirty(le.Hash())
		le.onDirty = nil
	}
	return newobj
}

func (le *lendingExchangeState) MarkBorrowingDirty(price common.Hash) {
	le.borrowingStatesDirty[price] = struct{}{}
	if le.onDirty != nil {
		le.onDirty(le.Hash())
		le.onDirty = nil
	}
}

func (le *lendingExchangeState) MarkInvestingDirty(price common.Hash) {
	le.investingStatesDirty[price] = struct{}{}
	if le.onDirty != nil {
		le.onDirty(le.Hash())
		le.onDirty = nil
	}
}

func (le *lendingExchangeState) MarkLendingItemDirty(lending common.Hash) {
	le.lendingItemStatesDirty[lending] = struct{}{}
	if le.onDirty != nil {
		le.onDirty(le.Hash())
		le.onDirty = nil
	}
}

func (le *lendingExchangeState) MarkLendingTradeDirty(tradeId common.Hash) {
	le.lendingTradeStatesDirty[tradeId] = struct{}{}
	if le.onDirty != nil {
		le.onDirty(le.Hash())
		le.onDirty = nil
	}
}

func (le *lendingExchangeState) MarkLiquidationTimeDirty(orderId common.Hash) {
	le.liquidationTimestatesDirty[orderId] = struct{}{}
	if le.onDirty != nil {
		le.onDirty(le.Hash())
		le.onDirty = nil
	}
}

func (le *lendingExchangeState) createBorrowingOrderList(db Database, price common.Hash) (newobj *itemListState) {
	newobj = newItemListState(le.lendingBook, price, itemList{Volume: Zero}, le.MarkBorrowingDirty)
	le.borrowingStates[price] = newobj
	le.borrowingStatesDirty[price] = struct{}{}
	data, err := rlp.EncodeToBytes(newobj)
	if err != nil {
		panic(fmt.Errorf("can't encode order list object at %x: %v", price[:], err))
	}
	le.setError(le.getBorrowingTrie(db).TryUpdate(price[:], data))
	if le.onDirty != nil {
		le.onDirty(le.Hash())
		le.onDirty = nil
	}
	return newobj
}

func (le *lendingExchangeState) createLendingItem(db Database, orderId common.Hash, order LendingItem) (newobj *lendingItemState) {
	newobj = newLendinItemState(le.lendingBook, orderId, order, le.MarkLendingItemDirty)
	orderIdHash := common.BigToHash(new(big.Int).SetUint64(order.LendingId))
	le.lendingItemStates[orderIdHash] = newobj
	le.lendingItemStatesDirty[orderIdHash] = struct{}{}
	if le.onDirty != nil {
		le.onDirty(le.lendingBook)
		le.onDirty = nil
	}
	return newobj
}

func (le *lendingExchangeState) createLiquidationTime(db Database, time common.Hash) (newobj *liquidationTimeState) {
	newobj = newLiquidationTimeState(time, le.lendingBook, itemList{Volume: Zero}, le.MarkLiquidationTimeDirty)
	le.liquidationTimeStates[time] = newobj
	le.liquidationTimestatesDirty[time] = struct{}{}
	data, err := rlp.EncodeToBytes(newobj)
	if err != nil {
		panic(fmt.Errorf("can't encode liquidation time at %x: %v", time[:], err))
	}
	le.setError(le.getLiquidationTimeTrie(db).TryUpdate(time[:], data))
	if le.onDirty != nil {
		le.onDirty(le.lendingBook)
		le.onDirty = nil
	}
	return newobj
}

func (le *lendingExchangeState) insertLendingTrade(tradeId common.Hash, order LendingTrade) (newobj *lendingTradeState) {
	newobj = newLendingTradeState(le.lendingBook, tradeId, order, le.MarkLendingTradeDirty)
	le.lendingTradeStates[tradeId] = newobj
	le.lendingTradeStatesDirty[tradeId] = struct{}{}
	if le.onDirty != nil {
		le.onDirty(le.lendingBook)
		le.onDirty = nil
	}
	return newobj
}
