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

// Package state provides a caching layer atop the Ethereum state trie.
package lendingstate

import (
	"fmt"
	"math/big"
	"sort"
	"sync"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/rlp"
)

type revision struct {
	id           int
	journalIndex int
}

type LendingStateDB struct {
	db   Database
	trie Trie

	// This map holds 'live' objects, which will get modified while processing a state transition.
	lendingExchangeStates      map[common.Hash]*lendingExchangeState
	lendingExchangeStatesDirty map[common.Hash]struct{}

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by LendingStateDB.Commit.
	dbErr error

	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	journal        journal
	validRevisions []revision
	nextRevisionId int

	lock sync.Mutex
}

// Create a new state from a given trie.
func New(root common.Hash, db Database) (*LendingStateDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		log.Error("Error when init new lending state trie ", "root", root.Hex(), "err", err)
		return nil, err
	}
	return &LendingStateDB{
		db:                         db,
		trie:                       tr,
		lendingExchangeStates:      make(map[common.Hash]*lendingExchangeState),
		lendingExchangeStatesDirty: make(map[common.Hash]struct{}),
	}, nil
}

// setError remembers the first non-nil error it is called with.
func (ls *LendingStateDB) setError(err error) {
	if ls.dbErr == nil {
		ls.dbErr = err
	}
}

func (ls *LendingStateDB) Error() error {
	return ls.dbErr
}

// Exist reports whether the given tradeId address exists in the state.
// Notably this also returns true for suicided lenddinges.
func (ls *LendingStateDB) Exist(addr common.Hash) bool {
	return ls.getLendingExchange(addr) != nil
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (ls *LendingStateDB) Empty(addr common.Hash) bool {
	so := ls.getLendingExchange(addr)
	return so == nil || so.empty()
}

func (ls *LendingStateDB) GetNonce(addr common.Hash) uint64 {
	stateObject := ls.getLendingExchange(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}
	return 0
}

func (ls *LendingStateDB) GetTradeNonce(addr common.Hash) uint64 {
	stateObject := ls.getLendingExchange(addr)
	if stateObject != nil {
		return stateObject.TradeNonce()
	}
	return 0
}

// Database retrieves the low level database supporting the lower level trie ops.
func (ls *LendingStateDB) Database() Database {
	return ls.db
}

func (ls *LendingStateDB) SetNonce(addr common.Hash, nonce uint64) {
	stateObject := ls.GetOrNewLendingExchangeObject(addr)
	if stateObject != nil {
		ls.journal = append(ls.journal, nonceChange{
			hash: addr,
			prev: stateObject.Nonce(),
		})
		stateObject.setNonce(nonce)
	}
}

func (ls *LendingStateDB) SetTradeNonce(addr common.Hash, nonce uint64) {
	stateObject := ls.GetOrNewLendingExchangeObject(addr)
	if stateObject != nil {
		ls.journal = append(ls.journal, tradeNonceChange{
			hash: addr,
			prev: stateObject.TradeNonce(),
		})
		stateObject.setTradeNonce(nonce)
	}
}

func (ls *LendingStateDB) InsertLendingItem(orderBook common.Hash, orderId common.Hash, order LendingItem) {
	interestHash := common.BigToHash(order.Interest)
	stateExchange := ls.getLendingExchange(orderBook)
	if stateExchange == nil {
		stateExchange = ls.createLendingExchangeObject(orderBook)
	}
	var stateOrderList *itemListState
	switch order.Side {
	case Investing:
		stateOrderList = stateExchange.getInvestingOrderList(ls.db, interestHash)
		if stateOrderList == nil {
			stateOrderList = stateExchange.createInvestingOrderList(ls.db, interestHash)
		}
	case Borrowing:
		stateOrderList = stateExchange.getBorrowingOrderList(ls.db, interestHash)
		if stateOrderList == nil {
			stateOrderList = stateExchange.createBorrowingOrderList(ls.db, interestHash)
		}
	default:
		return
	}
	ls.journal = append(ls.journal, insertOrder{
		orderBook: orderBook,
		orderId:   orderId,
		order:     &order,
	})
	stateExchange.createLendingItem(ls.db, orderId, order)
	stateOrderList.insertLendingItem(ls.db, orderId, common.BigToHash(order.Quantity))
	stateOrderList.AddVolume(order.Quantity)
}

func (ls *LendingStateDB) InsertTradingItem(orderBook common.Hash, tradeId uint64, order LendingTrade) {
	tradeIdHash := common.Uint64ToHash(tradeId)
	stateExchange := ls.getLendingExchange(orderBook)
	if stateExchange == nil {
		stateExchange = ls.createLendingExchangeObject(orderBook)
	}
	prvTrade := ls.GetLendingTrade(orderBook, tradeIdHash)
	ls.journal = append(ls.journal, insertTrading{
		orderBook: orderBook,
		tradeId:   tradeId,
		prvTrade:  &prvTrade,
	})
	stateExchange.insertLendingTrade(tradeIdHash, order)
}

func (ls *LendingStateDB) UpdateLiquidationPrice(orderBook common.Hash, tradeId uint64, price *big.Int) {
	tradeIdHash := common.Uint64ToHash(tradeId)
	stateExchange := ls.getLendingExchange(orderBook)
	if stateExchange == nil {
		stateExchange = ls.createLendingExchangeObject(orderBook)
	}
	stateLendingTrade := stateExchange.getLendingTrade(ls.db, tradeIdHash)
	ls.journal = append(ls.journal, liquidationPriceChange{
		orderBook: orderBook,
		tradeId:   tradeIdHash,
		prev:      stateLendingTrade.data.LiquidationPrice,
	})
	stateLendingTrade.SetLiquidationPrice(price)
}

func (ls *LendingStateDB) UpdateCollateralLockedAmount(orderBook common.Hash, tradeId uint64, amount *big.Int) {
	tradeIdHash := common.Uint64ToHash(tradeId)
	stateExchange := ls.getLendingExchange(orderBook)
	if stateExchange == nil {
		stateExchange = ls.createLendingExchangeObject(orderBook)
	}
	stateLendingTrade := stateExchange.getLendingTrade(ls.db, tradeIdHash)
	ls.journal = append(ls.journal, collateralLockedAmount{
		orderBook: orderBook,
		tradeId:   tradeIdHash,
		prev:      stateLendingTrade.data.CollateralLockedAmount,
	})
	stateLendingTrade.SetCollateralLockedAmount(amount)
}

func (ls *LendingStateDB) GetLendingOrder(orderBook common.Hash, orderId common.Hash) LendingItem {
	stateObject := ls.GetOrNewLendingExchangeObject(orderBook)
	if stateObject == nil {
		return EmptyLendingOrder
	}
	stateOrderItem := stateObject.getLendingItem(ls.db, orderId)
	if stateOrderItem == nil {
		return EmptyLendingOrder
	}
	return stateOrderItem.data
}

func (ls *LendingStateDB) GetLendingTrade(orderBook common.Hash, tradeId common.Hash) LendingTrade {
	stateObject := ls.GetOrNewLendingExchangeObject(orderBook)
	if stateObject == nil {
		return EmptyLendingTrade
	}
	stateOrderItem := stateObject.getLendingTrade(ls.db, tradeId)
	if stateOrderItem == nil || stateOrderItem.empty() {
		return EmptyLendingTrade
	}
	return stateOrderItem.data
}

func (ls *LendingStateDB) SubAmountLendingItem(orderBook common.Hash, orderId common.Hash, price *big.Int, amount *big.Int, side string) error {
	priceHash := common.BigToHash(price)
	lendingExchange := ls.GetOrNewLendingExchangeObject(orderBook)
	if lendingExchange == nil {
		return fmt.Errorf("not found order book: %s", orderBook.Hex())
	}
	var orderList *itemListState
	switch side {
	case Investing:
		orderList = lendingExchange.getInvestingOrderList(ls.db, priceHash)
	case Borrowing:
		orderList = lendingExchange.getBorrowingOrderList(ls.db, priceHash)
	default:
		return fmt.Errorf("not found order type: %s", side)
	}
	if orderList == nil || orderList.empty() {
		return fmt.Errorf("empty orderList: order book : %s , order id : %s , key : %s", orderBook, orderId.Hex(), priceHash.Hex())
	}
	lendingItem := lendingExchange.getLendingItem(ls.db, orderId)
	if lendingItem == nil || lendingItem.empty() {
		return fmt.Errorf("empty order item: order book : %s , order id : %s , key : %s", orderBook, orderId.Hex(), priceHash.Hex())
	}
	currentAmount := new(big.Int).SetBytes(orderList.GetOrderAmount(ls.db, orderId).Bytes()[:])
	if currentAmount.Cmp(amount) < 0 {
		return fmt.Errorf("not enough order amount %s: have : %d , want : %d", orderId.Hex(), currentAmount, amount)
	}
	ls.journal = append(ls.journal, subAmountOrder{
		orderBook: orderBook,
		orderId:   orderId,
		order:     ls.GetLendingOrder(orderBook, orderId),
		amount:    amount,
	})
	newAmount := new(big.Int).Sub(currentAmount, amount)
	lendingItem.setVolume(newAmount)
	log.Debug("SubAmountOrderItem", "tradeId", orderId.Hex(), "side", side, "key", price.Uint64(), "amount", amount.Uint64(), "new amount", newAmount.Uint64())
	orderList.subVolume(amount)
	if newAmount.Sign() == 0 {
		orderList.removeOrderItem(ls.db, orderId)
	} else {
		orderList.setOrderItem(orderId, common.BigToHash(newAmount))
	}
	if orderList.empty() {
		switch side {
		case Investing:
			lendingExchange.removeInvestingOrderList(ls.db, orderList)
		case Borrowing:
			lendingExchange.removeBorrowingOrderList(ls.db, orderList)
		default:
		}
	}
	return nil
}

func (ls *LendingStateDB) CancelLendingOrder(orderBook common.Hash, order *LendingItem) error {
	interestHash := common.BigToHash(order.Interest)
	orderIdHash := common.BigToHash(new(big.Int).SetUint64(order.LendingId))
	stateObject := ls.GetOrNewLendingExchangeObject(orderBook)
	if stateObject == nil {
		return fmt.Errorf("not found order book: %s", orderBook.Hex())
	}
	lendingItem := stateObject.getLendingItem(ls.db, orderIdHash)
	var orderList *itemListState
	switch lendingItem.data.Side {
	case Investing:
		orderList = stateObject.getInvestingOrderList(ls.db, interestHash)
	case Borrowing:
		orderList = stateObject.getBorrowingOrderList(ls.db, interestHash)
	default:
		return fmt.Errorf("not found order side: %s", order.Side)
	}
	if orderList == nil || orderList.empty() {
		return fmt.Errorf("empty OrderList: order book : %s , order id : %s , key : %s", orderBook, orderIdHash.Hex(), interestHash.Hex())
	}
	if lendingItem == nil || lendingItem.empty() {
		return fmt.Errorf("empty order item: order book : %s , order id : %s , key : %s", orderBook, orderIdHash.Hex(), interestHash.Hex())
	}
	if lendingItem.data.UserAddress != order.UserAddress {
		return fmt.Errorf("error Order UserAddress mismatch when cancel order book: %s , order id : %s , got : %s , expect : %s", orderBook, orderIdHash.Hex(), lendingItem.data.UserAddress.Hex(), order.UserAddress.Hex())
	}
	ls.journal = append(ls.journal, cancelOrder{
		orderBook: orderBook,
		orderId:   orderIdHash,
		order:     ls.GetLendingOrder(orderBook, orderIdHash),
	})
	lendingItem.setVolume(big.NewInt(0))
	currentAmount := new(big.Int).SetBytes(orderList.GetOrderAmount(ls.db, orderIdHash).Bytes()[:])
	orderList.subVolume(currentAmount)
	orderList.removeOrderItem(ls.db, orderIdHash)
	if orderList.empty() {
		switch order.Side {
		case Investing:
			stateObject.removeInvestingOrderList(ls.db, orderList)
		case Borrowing:
			stateObject.removeBorrowingOrderList(ls.db, orderList)
		default:
		}
	}
	return nil
}

func (ls *LendingStateDB) GetBestInvestingRate(orderBook common.Hash) (*big.Int, *big.Int) {
	stateObject := ls.getLendingExchange(orderBook)
	if stateObject != nil {
		investingHash := stateObject.getBestInvestingInterest(ls.db)
		if investingHash.IsZero() {
			return Zero, Zero
		}
		orderList := stateObject.getInvestingOrderList(ls.db, investingHash)
		if orderList == nil {
			log.Error("order list investing not found", "key", investingHash.Hex())
			return Zero, Zero
		}
		return new(big.Int).SetBytes(investingHash.Bytes()), orderList.Volume()
	}
	return Zero, Zero
}

func (ls *LendingStateDB) GetBestBorrowRate(orderBook common.Hash) (*big.Int, *big.Int) {
	stateObject := ls.getLendingExchange(orderBook)
	if stateObject != nil {
		priceHash := stateObject.getBestBorrowingInterest(ls.db)
		if priceHash.IsZero() {
			return Zero, Zero
		}
		orderList := stateObject.getBorrowingOrderList(ls.db, priceHash)
		if orderList == nil {
			log.Error("order list ask not found", "key", priceHash.Hex())
			return Zero, Zero
		}
		return new(big.Int).SetBytes(priceHash.Bytes()), orderList.Volume()
	}
	return Zero, Zero
}

func (ls *LendingStateDB) GetBestLendingIdAndAmount(orderBook common.Hash, price *big.Int, side string) (common.Hash, *big.Int, error) {
	stateObject := ls.GetOrNewLendingExchangeObject(orderBook)
	if stateObject != nil {
		var stateOrderList *itemListState
		switch side {
		case Investing:
			stateOrderList = stateObject.getInvestingOrderList(ls.db, common.BigToHash(price))
		case Borrowing:
			stateOrderList = stateObject.getBorrowingOrderList(ls.db, common.BigToHash(price))
		default:
			return EmptyHash, Zero, fmt.Errorf("not found side: %s", side)
		}
		if stateOrderList != nil {
			key, _, err := stateOrderList.getTrie(ls.db).TryGetBestLeftKeyAndValue()
			if err != nil {
				return EmptyHash, Zero, err
			}
			orderId := common.BytesToHash(key)
			amount := stateOrderList.GetOrderAmount(ls.db, orderId)
			return orderId, new(big.Int).SetBytes(amount.Bytes()), nil
		}
		return EmptyHash, Zero, fmt.Errorf("not found order list with orderBook: %s , key : %d , side : %s", orderBook.Hex(), price, side)
	}
	return EmptyHash, Zero, fmt.Errorf("not found orderBook: %s", orderBook.Hex())
}

// updateLendingExchange writes the given object to the trie.
func (ls *LendingStateDB) updateLendingExchange(stateObject *lendingExchangeState) {
	addr := stateObject.Hash()
	data, err := rlp.EncodeToBytes(stateObject)
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
	}
	ls.setError(ls.trie.TryUpdate(addr[:], data))
}

// Retrieve a state object given my the address. Returns nil if not found.
func (ls *LendingStateDB) getLendingExchange(addr common.Hash) (stateObject *lendingExchangeState) {
	// Prefer 'live' objects.
	if obj := ls.lendingExchangeStates[addr]; obj != nil {
		return obj
	}
	// Load the object from the database.
	enc, err := ls.trie.TryGet(addr[:])
	if len(enc) == 0 {
		ls.setError(err)
		return nil
	}
	var data lendingObject
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state object", "addr", addr, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newStateExchanges(ls, addr, data, ls.MarkLendingExchangeObjectDirty)
	ls.lendingExchangeStates[addr] = obj
	return obj
}

func (ls *LendingStateDB) setLendingExchangeObject(object *lendingExchangeState) {
	ls.lendingExchangeStates[object.Hash()] = object
	ls.lendingExchangeStatesDirty[object.Hash()] = struct{}{}
}

// Retrieve a state object or create a new state object if nil.
func (ls *LendingStateDB) GetOrNewLendingExchangeObject(addr common.Hash) *lendingExchangeState {
	stateExchangeObject := ls.getLendingExchange(addr)
	if stateExchangeObject == nil {
		stateExchangeObject = ls.createLendingExchangeObject(addr)
	}
	return stateExchangeObject
}

// MarkStateLendObjectDirty adds the specified object to the dirty map to avoid costly
// state object cache iteration to find a handful of modified ones.
func (ls *LendingStateDB) MarkLendingExchangeObjectDirty(addr common.Hash) {
	ls.lendingExchangeStatesDirty[addr] = struct{}{}
}

// createStateOrderListObject creates a new state object. If there is an existing tradeId with
// the given address, it is overwritten and returned as the second return value.
func (ls *LendingStateDB) createLendingExchangeObject(hash common.Hash) (newobj *lendingExchangeState) {
	newobj = newStateExchanges(ls, hash, lendingObject{}, ls.MarkLendingExchangeObjectDirty)
	newobj.setNonce(0) // sets the object to dirty
	ls.setLendingExchangeObject(newobj)
	return newobj
}

// Copy creates a deep, independent copy of the state.
// Snapshots of the copied state cannot be applied to the copy.
func (ls *LendingStateDB) Copy() *LendingStateDB {
	ls.lock.Lock()
	defer ls.lock.Unlock()

	// Copy all the basic fields, initialize the memory ones
	state := &LendingStateDB{
		db:                         ls.db,
		trie:                       ls.db.CopyTrie(ls.trie),
		lendingExchangeStates:      make(map[common.Hash]*lendingExchangeState, len(ls.lendingExchangeStatesDirty)),
		lendingExchangeStatesDirty: make(map[common.Hash]struct{}, len(ls.lendingExchangeStatesDirty)),
	}
	// Copy the dirty states, logs, and preimages
	for addr := range ls.lendingExchangeStatesDirty {
		state.lendingExchangeStatesDirty[addr] = struct{}{}
	}
	for addr, exchangeObject := range ls.lendingExchangeStates {
		state.lendingExchangeStates[addr] = exchangeObject.deepCopy(state, state.MarkLendingExchangeObjectDirty)
	}

	return state
}

func (ls *LendingStateDB) clearJournalAndRefund() {
	ls.journal = nil
	ls.validRevisions = ls.validRevisions[:0]
}

// Snapshot returns an identifier for the current revision of the state.
func (ls *LendingStateDB) Snapshot() int {
	id := ls.nextRevisionId
	ls.nextRevisionId++
	ls.validRevisions = append(ls.validRevisions, revision{id, len(ls.journal)})
	return id
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (ls *LendingStateDB) RevertToSnapshot(revid int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(ls.validRevisions), func(i int) bool {
		return ls.validRevisions[i].id >= revid
	})
	if idx == len(ls.validRevisions) || ls.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := ls.validRevisions[idx].journalIndex

	// Replay the journal to undo changes.
	for i := len(ls.journal) - 1; i >= snapshot; i-- {
		ls.journal[i].undo(ls)
	}
	ls.journal = ls.journal[:snapshot]

	// Remove invalidated snapshots from the stack.
	ls.validRevisions = ls.validRevisions[:idx]
}

// Finalise finalises the state by removing the self destructed objects
// and clears the journal as well as the refunds.
func (ls *LendingStateDB) Finalise() {
	// Commit objects to the trie.
	for addr, stateObject := range ls.lendingExchangeStates {
		if _, isDirty := ls.lendingExchangeStatesDirty[addr]; isDirty {
			// Write any storage changes in the state object to its storage trie.
			err := stateObject.updateInvestingRoot(ls.db)
			if err != nil {
				log.Warn("Finalise updateInvestingRoot", "err", err, "addr", addr, "stateObject", *stateObject)
			}
			stateObject.updateBorrowingRoot(ls.db)
			stateObject.updateOrderRoot(ls.db)
			stateObject.updateLendingTradeRoot(ls.db)
			stateObject.updateLiquidationTimeRoot(ls.db)
			// Update the object in the main tradeId trie.
			ls.updateLendingExchange(stateObject)
			//delete(s.investingStatesDirty, addr)
		}
	}
	ls.clearJournalAndRefund()
}

// IntermediateRoot computes the current root orderBook of the state trie.
// It is called in between transactions to get the root orderBook that
// goes into transaction receipts.
func (ls *LendingStateDB) IntermediateRoot() common.Hash {
	ls.Finalise()
	return ls.trie.Hash()
}

// Commit writes the state to the underlying in-memory trie database.
func (ls *LendingStateDB) Commit() (root common.Hash, err error) {
	defer ls.clearJournalAndRefund()
	// Commit objects to the trie.
	for addr, stateObject := range ls.lendingExchangeStates {
		if _, isDirty := ls.lendingExchangeStatesDirty[addr]; isDirty {
			// Write any storage changes in the state object to its storage trie.
			if err := stateObject.CommitInvestingTrie(ls.db); err != nil {
				return EmptyHash, err
			}
			if err := stateObject.CommitBorrowingTrie(ls.db); err != nil {
				return EmptyHash, err
			}
			if err := stateObject.CommitLendingItemTrie(ls.db); err != nil {
				return EmptyHash, err
			}
			if err := stateObject.CommitLendingTradeTrie(ls.db); err != nil {
				return EmptyHash, err
			}
			if err := stateObject.CommitLiquidationTimeTrie(ls.db); err != nil {
				return EmptyHash, err
			}
			// Update the object in the main tradeId trie.
			ls.updateLendingExchange(stateObject)
			delete(ls.lendingExchangeStatesDirty, addr)
		}
	}
	// Write trie changes.
	root, err = ls.trie.Commit(func(leaf []byte, parent common.Hash) error {
		var exchange lendingObject
		if err := rlp.DecodeBytes(leaf, &exchange); err != nil {
			return nil
		}
		if exchange.InvestingRoot != EmptyRoot {
			ls.db.TrieDB().Reference(exchange.InvestingRoot, parent)
		}
		if exchange.BorrowingRoot != EmptyRoot {
			ls.db.TrieDB().Reference(exchange.BorrowingRoot, parent)
		}
		if exchange.LendingItemRoot != EmptyRoot {
			ls.db.TrieDB().Reference(exchange.LendingItemRoot, parent)
		}
		if exchange.LendingTradeRoot != EmptyRoot {
			ls.db.TrieDB().Reference(exchange.LendingTradeRoot, parent)
		}
		if exchange.LiquidationTimeRoot != EmptyRoot {
			ls.db.TrieDB().Reference(exchange.LiquidationTimeRoot, parent)
		}
		return nil
	})
	log.Debug("Lending State Trie cache stats after commit", "root", root.Hex())
	return root, err
}

func (ls *LendingStateDB) InsertLiquidationTime(lendingBook common.Hash, time *big.Int, tradeId uint64) {
	timeHash := common.BigToHash(time)
	lendingExchangeState := ls.getLendingExchange(lendingBook)
	if lendingExchangeState == nil {
		lendingExchangeState = ls.createLendingExchangeObject(lendingBook)
	}
	liquidationTime := lendingExchangeState.getLiquidationTimeOrderList(ls.db, timeHash)
	if liquidationTime == nil {
		liquidationTime = lendingExchangeState.createLiquidationTime(ls.db, timeHash)
	}
	liquidationTime.insertTradeId(ls.db, common.Uint64ToHash(tradeId))
	liquidationTime.AddVolume(One)
}

func (ls *LendingStateDB) RemoveLiquidationTime(lendingBook common.Hash, tradeId uint64, time uint64) error {
	timeHash := common.Uint64ToHash(time)
	tradeIdHash := common.Uint64ToHash(tradeId)
	lendingExchangeState := ls.getLendingExchange(lendingBook)
	if lendingExchangeState == nil {
		return fmt.Errorf("lending book not found: %s", lendingBook.Hex())
	}
	liquidationTime := lendingExchangeState.getLiquidationTimeOrderList(ls.db, timeHash)
	if liquidationTime == nil {
		return fmt.Errorf("not found liquidation time: %s , %d", lendingBook.Hex(), time)
	}
	if !liquidationTime.Exist(ls.db, tradeIdHash) {
		return fmt.Errorf("not exist tradeId: %s, %d, %d", lendingBook.Hex(), time, tradeId)
	}
	liquidationTime.removeTradeId(ls.db, tradeIdHash)
	liquidationTime.subVolume(One)
	if liquidationTime.Volume().Sign() == 0 {
		err := lendingExchangeState.getLiquidationTimeTrie(ls.db).TryDelete(timeHash[:])
		if err != nil {
			log.Warn("RemoveLiquidationTime getLiquidationTimeTrie.TryDelete", "err", err, "timeHash[:]", timeHash[:])
		}
	}
	return nil
}

func (ls *LendingStateDB) GetLowestLiquidationTime(lendingBook common.Hash, time *big.Int) (*big.Int, []common.Hash) {
	liquidationData := []common.Hash{}
	lendingExchangeState := ls.getLendingExchange(lendingBook)
	if lendingExchangeState == nil {
		return common.Big0, liquidationData
	}
	lowestPriceHash, liquidationState := lendingExchangeState.getLowestLiquidationTime(ls.db)
	lowestTime := new(big.Int).SetBytes(lowestPriceHash[:])
	if liquidationState != nil && lowestTime.Sign() > 0 && lowestTime.Cmp(time) <= 0 {
		liquidationData = liquidationState.getAllTradeIds(ls.db)
	}
	return lowestTime, liquidationData
}

func (ls *LendingStateDB) CancelLendingTrade(orderBook common.Hash, tradeId uint64) error {
	tradeIdHash := common.Uint64ToHash(tradeId)
	stateObject := ls.GetOrNewLendingExchangeObject(orderBook)
	if stateObject == nil {
		return fmt.Errorf("not found order book: %s", orderBook.Hex())
	}
	lendingTrade := stateObject.getLendingTrade(ls.db, tradeIdHash)
	if lendingTrade == nil || lendingTrade.empty() {
		return fmt.Errorf("lending trade empty order book: %s , trade id : %s , trade id hash : %s", orderBook, tradeIdHash.Hex(), tradeIdHash.Hex())
	}
	ls.journal = append(ls.journal, cancelTrading{
		orderBook: orderBook,
		order:     ls.GetLendingTrade(orderBook, tradeIdHash),
	})
	lendingTrade.SetAmount(Zero)
	return nil
}
