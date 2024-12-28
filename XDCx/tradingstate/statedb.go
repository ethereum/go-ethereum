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
package tradingstate

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

// StateDBs within the ethereum protocol are used to store anything
// within the merkle trie. StateDBs take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Contracts
// * Accounts
type TradingStateDB struct {
	db   Database
	trie Trie

	// This map holds 'live' objects, which will get modified while processing a state transition.
	stateExhangeObjects      map[common.Hash]*tradingExchanges
	stateExhangeObjectsDirty map[common.Hash]struct{}

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by TradingStateDB.Commit.
	dbErr error

	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	journal        journal
	validRevisions []revision
	nextRevisionId int

	lock sync.Mutex
}

// Create a new state from a given trie.
func New(root common.Hash, db Database) (*TradingStateDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		log.Error("Error when init new trading state trie ", "root", root.Hex(), "err", err)
		return nil, err
	}
	return &TradingStateDB{
		db:                       db,
		trie:                     tr,
		stateExhangeObjects:      make(map[common.Hash]*tradingExchanges),
		stateExhangeObjectsDirty: make(map[common.Hash]struct{}),
	}, nil
}

// setError remembers the first non-nil error it is called with.
func (t *TradingStateDB) setError(err error) {
	if t.dbErr == nil {
		t.dbErr = err
	}
}

func (t *TradingStateDB) Error() error {
	return t.dbErr
}

// Exist reports whether the given orderId address exists in the state.
// Notably this also returns true for suicided exchanges.
func (t *TradingStateDB) Exist(addr common.Hash) bool {
	return t.getStateExchangeObject(addr) != nil
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (t *TradingStateDB) Empty(addr common.Hash) bool {
	so := t.getStateExchangeObject(addr)
	return so == nil || so.empty()
}

func (t *TradingStateDB) GetNonce(addr common.Hash) uint64 {
	stateObject := t.getStateExchangeObject(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}
	return 0
}

func (t *TradingStateDB) GetLastPrice(addr common.Hash) *big.Int {
	stateObject := t.getStateExchangeObject(addr)
	if stateObject != nil {
		return stateObject.data.LastPrice
	}
	return nil
}

func (t *TradingStateDB) GetMediumPriceBeforeEpoch(addr common.Hash) *big.Int {
	stateObject := t.getStateExchangeObject(addr)
	if stateObject != nil {
		return stateObject.data.MediumPriceBeforeEpoch
	}
	return Zero
}

func (t *TradingStateDB) GetMediumPriceAndTotalAmount(addr common.Hash) (*big.Int, *big.Int) {
	stateObject := t.getStateExchangeObject(addr)
	if stateObject != nil {
		return stateObject.data.MediumPrice, stateObject.data.TotalQuantity
	}
	return Zero, Zero
}

// Database retrieves the low level database supporting the lower level trie ops.
func (t *TradingStateDB) Database() Database {
	return t.db
}

func (t *TradingStateDB) SetNonce(addr common.Hash, nonce uint64) {
	stateObject := t.GetOrNewStateExchangeObject(addr)
	if stateObject != nil {
		t.journal = append(t.journal, nonceChange{
			hash: addr,
			prev: t.GetNonce(addr),
		})
		stateObject.SetNonce(nonce)
	}
}

func (t *TradingStateDB) SetLastPrice(addr common.Hash, price *big.Int) {
	stateObject := t.GetOrNewStateExchangeObject(addr)
	if stateObject != nil {
		t.journal = append(t.journal, lastPriceChange{
			hash: addr,
			prev: stateObject.data.LastPrice,
		})
		stateObject.setLastPrice(price)
	}
}

func (t *TradingStateDB) SetMediumPrice(addr common.Hash, price *big.Int, quantity *big.Int) {
	stateObject := t.GetOrNewStateExchangeObject(addr)
	if stateObject != nil {
		t.journal = append(t.journal, mediumPriceChange{
			hash:         addr,
			prevPrice:    stateObject.data.MediumPrice,
			prevQuantity: stateObject.data.TotalQuantity,
		})
		stateObject.setMediumPrice(price, quantity)
	}
}

func (t *TradingStateDB) SetMediumPriceBeforeEpoch(addr common.Hash, price *big.Int) {
	stateObject := t.GetOrNewStateExchangeObject(addr)
	if stateObject != nil {
		t.journal = append(t.journal, mediumPriceBeforeEpochChange{
			hash:      addr,
			prevPrice: stateObject.data.MediumPriceBeforeEpoch,
		})
		stateObject.setMediumPriceBeforeEpoch(price)
	}
}

func (t *TradingStateDB) InsertOrderItem(orderBook common.Hash, orderId common.Hash, order OrderItem) {
	priceHash := common.BigToHash(order.Price)
	stateExchange := t.getStateExchangeObject(orderBook)
	if stateExchange == nil {
		stateExchange = t.createExchangeObject(orderBook)
	}
	var stateOrderList *stateOrderList
	switch order.Side {
	case Ask:
		stateOrderList = stateExchange.getStateOrderListAskObject(t.db, priceHash)
		if stateOrderList == nil {
			stateOrderList = stateExchange.createStateOrderListAskObject(t.db, priceHash)
		}
	case Bid:
		stateOrderList = stateExchange.getStateBidOrderListObject(t.db, priceHash)
		if stateOrderList == nil {
			stateOrderList = stateExchange.createStateBidOrderListObject(t.db, priceHash)
		}
	default:
		return
	}
	t.journal = append(t.journal, insertOrder{
		orderBook: orderBook,
		orderId:   orderId,
		order:     &order,
	})
	stateExchange.createStateOrderObject(t.db, orderId, order)
	stateOrderList.insertOrderItem(t.db, orderId, common.BigToHash(order.Quantity))
	stateOrderList.AddVolume(order.Quantity)
}

func (t *TradingStateDB) GetOrder(orderBook common.Hash, orderId common.Hash) OrderItem {
	stateObject := t.GetOrNewStateExchangeObject(orderBook)
	if stateObject == nil {
		return EmptyOrder
	}
	stateOrderItem := stateObject.getStateOrderObject(t.db, orderId)
	if stateOrderItem == nil {
		return EmptyOrder
	}
	return stateOrderItem.data
}

func (t *TradingStateDB) SubAmountOrderItem(orderBook common.Hash, orderId common.Hash, price *big.Int, amount *big.Int, side string) error {
	priceHash := common.BigToHash(price)
	stateObject := t.GetOrNewStateExchangeObject(orderBook)
	if stateObject == nil {
		return fmt.Errorf("not found orderBook: %s", orderBook.Hex())
	}
	var stateOrderList *stateOrderList
	switch side {
	case Ask:
		stateOrderList = stateObject.getStateOrderListAskObject(t.db, priceHash)
	case Bid:
		stateOrderList = stateObject.getStateBidOrderListObject(t.db, priceHash)
	default:
		return fmt.Errorf("not found order type: %s", side)
	}
	if stateOrderList == nil || stateOrderList.empty() {
		return fmt.Errorf("empty Orderlist: order book: %s , order id : %s , price : %s", orderBook, orderId.Hex(), priceHash.Hex())
	}
	stateOrderItem := stateObject.getStateOrderObject(t.db, orderId)
	if stateOrderItem == nil || stateOrderItem.empty() {
		return fmt.Errorf("empty OrderItem: order book: %s , order id : %s , price : %s", orderBook, orderId.Hex(), priceHash.Hex())
	}
	currentAmount := new(big.Int).SetBytes(stateOrderList.GetOrderAmount(t.db, orderId).Bytes()[:])
	if currentAmount.Cmp(amount) < 0 {
		return fmt.Errorf("not enough order amount: %s , have : %d , want : %d ", orderId.Hex(), currentAmount, amount)
	}
	t.journal = append(t.journal, subAmountOrder{
		orderBook: orderBook,
		orderId:   orderId,
		order:     t.GetOrder(orderBook, orderId),
		amount:    amount,
	})
	newAmount := new(big.Int).Sub(currentAmount, amount)
	log.Debug("SubAmountOrderItem", "orderId", orderId.Hex(), "side", side, "price", price.Uint64(), "amount", amount.Uint64(), "new amount", newAmount.Uint64())
	stateOrderList.subVolume(amount)
	stateOrderItem.setVolume(newAmount)
	if newAmount.Sign() == 0 {
		stateOrderList.removeOrderItem(t.db, orderId)
	} else {
		stateOrderList.setOrderItem(orderId, common.BigToHash(newAmount))
	}
	if stateOrderList.empty() {
		switch side {
		case Ask:
			stateObject.removeStateOrderListAskObject(t.db, stateOrderList)
		case Bid:
			stateObject.removeStateOrderListBidObject(t.db, stateOrderList)
		default:
		}
	}
	return nil
}

func (t *TradingStateDB) CancelOrder(orderBook common.Hash, order *OrderItem) error {
	orderIdHash := common.BigToHash(new(big.Int).SetUint64(order.OrderID))
	stateObject := t.GetOrNewStateExchangeObject(orderBook)
	if stateObject == nil {
		return fmt.Errorf("not found orderBook: %s", orderBook.Hex())
	}
	stateOrderItem := stateObject.getStateOrderObject(t.db, orderIdHash)
	if stateOrderItem == nil || stateOrderItem.empty() {
		return fmt.Errorf("empty OrderItem: order book: %s , order id : %s", orderBook, orderIdHash.Hex())
	}
	priceHash := common.BigToHash(stateOrderItem.data.Price)
	var stateOrderList *stateOrderList
	switch stateOrderItem.data.Side {
	case Ask:
		stateOrderList = stateObject.getStateOrderListAskObject(t.db, priceHash)
	case Bid:
		stateOrderList = stateObject.getStateBidOrderListObject(t.db, priceHash)
	default:
		return fmt.Errorf("not found order.Side: %s", order.Side)
	}
	if stateOrderList == nil || stateOrderList.empty() {
		return fmt.Errorf("empty OrderList: order book: %s , order id : %s , price : %s", orderBook, orderIdHash.Hex(), priceHash.Hex())
	}

	if stateOrderItem.data.UserAddress != order.UserAddress {
		return fmt.Errorf("error Order UserAddress mismatch when cancel: order book: %s , order id : %s , got : %s , expect : %s", orderBook, orderIdHash.Hex(), stateOrderItem.data.UserAddress.Hex(), order.UserAddress.Hex())
	}
	if stateOrderItem.data.Hash != order.Hash {
		return fmt.Errorf("invalid order hash: got : %s , expect : %s", order.Hash.Hex(), stateOrderItem.data.Hash.Hex())
	}
	if stateOrderItem.data.ExchangeAddress != order.ExchangeAddress {
		return fmt.Errorf("mismatch ExchangeAddress when cancel: order book : %s , order id : %s , got : %s , expect : %s", orderBook, orderIdHash.Hex(), order.ExchangeAddress.Hex(), stateOrderItem.data.ExchangeAddress.Hex())
	}
	t.journal = append(t.journal, cancelOrder{
		orderBook: orderBook,
		orderId:   orderIdHash,
		order:     stateOrderItem.data,
	})
	currentAmount := new(big.Int).SetBytes(stateOrderList.GetOrderAmount(t.db, orderIdHash).Bytes()[:])
	stateOrderItem.setVolume(big.NewInt(0))
	stateOrderList.subVolume(currentAmount)
	stateOrderList.removeOrderItem(t.db, orderIdHash)
	if stateOrderList.empty() {
		switch stateOrderItem.data.Side {
		case Ask:
			stateObject.removeStateOrderListAskObject(t.db, stateOrderList)
		case Bid:
			stateObject.removeStateOrderListBidObject(t.db, stateOrderList)
		default:
		}
	}
	return nil
}

func (t *TradingStateDB) GetVolume(orderBook common.Hash, price *big.Int, orderType string) *big.Int {
	stateObject := t.GetOrNewStateExchangeObject(orderBook)
	var volume *big.Int = nil
	if stateObject != nil {
		var stateOrderList *stateOrderList
		switch orderType {
		case Ask:
			stateOrderList = stateObject.getStateOrderListAskObject(t.db, common.BigToHash(price))
		case Bid:
			stateOrderList = stateObject.getStateBidOrderListObject(t.db, common.BigToHash(price))
		default:
			return Zero
		}
		if stateOrderList == nil || stateOrderList.empty() {
			return Zero
		}
		volume = stateOrderList.Volume()
	}
	return volume
}

func (t *TradingStateDB) GetBestAskPrice(orderBook common.Hash) (*big.Int, *big.Int) {
	stateObject := t.getStateExchangeObject(orderBook)
	if stateObject != nil {
		priceHash := stateObject.getBestPriceAsksTrie(t.db)
		if priceHash.IsZero() {
			return Zero, Zero
		}
		orderList := stateObject.getStateOrderListAskObject(t.db, priceHash)
		if orderList == nil {
			log.Error("order list ask not found", "price", priceHash.Hex())
			return Zero, Zero
		}
		return new(big.Int).SetBytes(priceHash.Bytes()), orderList.Volume()
	}
	return Zero, Zero
}

func (t *TradingStateDB) GetBestBidPrice(orderBook common.Hash) (*big.Int, *big.Int) {
	stateObject := t.getStateExchangeObject(orderBook)
	if stateObject != nil {
		priceHash := stateObject.getBestBidsTrie(t.db)
		if priceHash.IsZero() {
			return Zero, Zero
		}
		orderList := stateObject.getStateBidOrderListObject(t.db, priceHash)
		if orderList == nil {
			log.Error("order list bid not found", "price", priceHash.Hex())
			return Zero, Zero
		}
		return new(big.Int).SetBytes(priceHash.Bytes()), orderList.Volume()
	}
	return Zero, Zero
}

func (t *TradingStateDB) GetBestOrderIdAndAmount(orderBook common.Hash, price *big.Int, side string) (common.Hash, *big.Int, error) {
	stateObject := t.GetOrNewStateExchangeObject(orderBook)
	if stateObject != nil {
		var stateOrderList *stateOrderList
		switch side {
		case Ask:
			stateOrderList = stateObject.getStateOrderListAskObject(t.db, common.BigToHash(price))
		case Bid:
			stateOrderList = stateObject.getStateBidOrderListObject(t.db, common.BigToHash(price))
		default:
			return EmptyHash, Zero, fmt.Errorf("not found side: %s", side)
		}
		if stateOrderList != nil {
			key, _, err := stateOrderList.getTrie(t.db).TryGetBestLeftKeyAndValue()
			if err != nil {
				return EmptyHash, Zero, err
			}
			orderId := common.BytesToHash(key)
			amount := stateOrderList.GetOrderAmount(t.db, orderId)
			return orderId, new(big.Int).SetBytes(amount.Bytes()), nil
		}
		return EmptyHash, Zero, fmt.Errorf("not found order list with orderBook: %s , price : %d , side : %s", orderBook.Hex(), price, side)
	}
	return EmptyHash, Zero, fmt.Errorf("not found orderBook: %s", orderBook.Hex())
}

// updateStateExchangeObject writes the given object to the trie.
func (t *TradingStateDB) updateStateExchangeObject(stateObject *tradingExchanges) {
	addr := stateObject.Hash()
	data, err := rlp.EncodeToBytes(stateObject)
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
	}
	t.setError(t.trie.TryUpdate(addr[:], data))
}

// Retrieve a state object given my the address. Returns nil if not found.
func (t *TradingStateDB) getStateExchangeObject(addr common.Hash) (stateObject *tradingExchanges) {
	// Prefer 'live' objects.
	if obj := t.stateExhangeObjects[addr]; obj != nil {
		return obj
	}
	// Load the object from the database.
	enc, err := t.trie.TryGet(addr[:])
	if len(enc) == 0 {
		t.setError(err)
		return nil
	}
	var data tradingExchangeObject
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state object", "addr", addr, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newStateExchanges(t, addr, data, t.MarkStateExchangeObjectDirty)
	t.stateExhangeObjects[addr] = obj
	return obj
}

func (t *TradingStateDB) setStateExchangeObject(object *tradingExchanges) {
	t.stateExhangeObjects[object.Hash()] = object
	t.stateExhangeObjectsDirty[object.Hash()] = struct{}{}
}

// Retrieve a state object or create a new state object if nil.
func (t *TradingStateDB) GetOrNewStateExchangeObject(addr common.Hash) *tradingExchanges {
	stateExchangeObject := t.getStateExchangeObject(addr)
	if stateExchangeObject == nil {
		stateExchangeObject = t.createExchangeObject(addr)
	}
	return stateExchangeObject
}

// MarkStateAskObjectDirty adds the specified object to the dirty map to avoid costly
// state object cache iteration to find a handful of modified ones.
func (t *TradingStateDB) MarkStateExchangeObjectDirty(addr common.Hash) {
	t.stateExhangeObjectsDirty[addr] = struct{}{}
}

// createStateOrderListObject creates a new state object. If there is an existing orderId with
// the given address, it is overwritten and returned as the second return value.
func (t *TradingStateDB) createExchangeObject(hash common.Hash) (newobj *tradingExchanges) {
	newobj = newStateExchanges(t, hash, tradingExchangeObject{LendingCount: Zero, MediumPrice: Zero, MediumPriceBeforeEpoch: Zero, TotalQuantity: Zero}, t.MarkStateExchangeObjectDirty)
	newobj.setNonce(0) // sets the object to dirty
	t.setStateExchangeObject(newobj)
	return newobj
}

// Copy creates a deep, independent copy of the state.
// Snapshots of the copied state cannot be applied to the copy.
func (t *TradingStateDB) Copy() *TradingStateDB {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Copy all the basic fields, initialize the memory ones
	state := &TradingStateDB{
		db:                       t.db,
		trie:                     t.db.CopyTrie(t.trie),
		stateExhangeObjects:      make(map[common.Hash]*tradingExchanges, len(t.stateExhangeObjectsDirty)),
		stateExhangeObjectsDirty: make(map[common.Hash]struct{}, len(t.stateExhangeObjectsDirty)),
	}
	// Copy the dirty states, logs, and preimages
	for addr := range t.stateExhangeObjectsDirty {
		state.stateExhangeObjectsDirty[addr] = struct{}{}
	}
	for addr, exchangeObject := range t.stateExhangeObjects {
		state.stateExhangeObjects[addr] = exchangeObject.deepCopy(state, state.MarkStateExchangeObjectDirty)
	}

	return state
}

func (t *TradingStateDB) clearJournalAndRefund() {
	t.journal = nil
	t.validRevisions = t.validRevisions[:0]
}

// Snapshot returns an identifier for the current revision of the state.
func (t *TradingStateDB) Snapshot() int {
	id := t.nextRevisionId
	t.nextRevisionId++
	t.validRevisions = append(t.validRevisions, revision{id, len(t.journal)})
	return id
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (t *TradingStateDB) RevertToSnapshot(revid int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(t.validRevisions), func(i int) bool {
		return t.validRevisions[i].id >= revid
	})
	if idx == len(t.validRevisions) || t.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := t.validRevisions[idx].journalIndex

	// Replay the journal to undo changes.
	for i := len(t.journal) - 1; i >= snapshot; i-- {
		t.journal[i].undo(t)
	}
	t.journal = t.journal[:snapshot]

	// Remove invalidated snapshots from the stack.
	t.validRevisions = t.validRevisions[:idx]
}

// Finalise finalises the state by removing the self destructed objects
// and clears the journal as well as the refunds.
func (t *TradingStateDB) Finalise() {
	// Commit objects to the trie.
	for addr, stateObject := range t.stateExhangeObjects {
		if _, isDirty := t.stateExhangeObjectsDirty[addr]; isDirty {
			// Write any storage changes in the state object to its storage trie.
			err := stateObject.updateAsksRoot(t.db)
			if err != nil {
				log.Warn("Finalise updateAsksRoot", "err", err, "addr", addr, "stateObject", *stateObject)
			}
			stateObject.updateBidsRoot(t.db)
			stateObject.updateOrdersRoot(t.db)
			stateObject.updateLiquidationPriceRoot(t.db)
			// Update the object in the main orderId trie.
			t.updateStateExchangeObject(stateObject)
			//delete(s.stateExhangeObjectsDirty, addr)
		}
	}
	t.clearJournalAndRefund()
}

// IntermediateRoot computes the current root orderBookHash of the state trie.
// It is called in between transactions to get the root orderBookHash that
// goes into transaction receipts.
func (t *TradingStateDB) IntermediateRoot() common.Hash {
	t.Finalise()
	return t.trie.Hash()
}

// Commit writes the state to the underlying in-memory trie database.
func (t *TradingStateDB) Commit() (root common.Hash, err error) {
	defer t.clearJournalAndRefund()
	// Commit objects to the trie.
	for addr, stateObject := range t.stateExhangeObjects {
		if _, isDirty := t.stateExhangeObjectsDirty[addr]; isDirty {
			// Write any storage changes in the state object to its storage trie.
			if err := stateObject.CommitAsksTrie(t.db); err != nil {
				return EmptyHash, err
			}
			if err := stateObject.CommitBidsTrie(t.db); err != nil {
				return EmptyHash, err
			}
			if err := stateObject.CommitOrdersTrie(t.db); err != nil {
				return EmptyHash, err
			}
			if err := stateObject.CommitLiquidationPriceTrie(t.db); err != nil {
				return EmptyHash, err
			}
			// Update the object in the main orderId trie.
			t.updateStateExchangeObject(stateObject)
			delete(t.stateExhangeObjectsDirty, addr)
		}
	}
	// Write trie changes.
	root, err = t.trie.Commit(func(leaf []byte, parent common.Hash) error {
		var exchange tradingExchangeObject
		if err := rlp.DecodeBytes(leaf, &exchange); err != nil {
			return nil
		}
		if exchange.AskRoot != EmptyRoot {
			t.db.TrieDB().Reference(exchange.AskRoot, parent)
		}
		if exchange.BidRoot != EmptyRoot {
			t.db.TrieDB().Reference(exchange.BidRoot, parent)
		}
		if exchange.OrderRoot != EmptyRoot {
			t.db.TrieDB().Reference(exchange.OrderRoot, parent)
		}
		if exchange.LiquidationPriceRoot != EmptyRoot {
			t.db.TrieDB().Reference(exchange.LiquidationPriceRoot, parent)
		}
		return nil
	})
	log.Debug("Trading State Trie cache stats after commit", "root", root.Hex())
	return root, err
}

func (t *TradingStateDB) GetAllLowerLiquidationPriceData(orderBook common.Hash, limit *big.Int) map[*big.Int]map[common.Hash][]common.Hash {
	result := map[*big.Int]map[common.Hash][]common.Hash{}
	orderbookState := t.getStateExchangeObject(orderBook)
	if orderbookState == nil {
		return result
	}
	mapPrices := orderbookState.getAllLowerLiquidationPrice(t.db, common.BigToHash(limit))
	for priceHash, liquidationState := range mapPrices {
		price := new(big.Int).SetBytes(priceHash[:])
		log.Debug("GetAllLowerLiquidationPriceData", "price", price, "limit", limit)
		if liquidationState != nil && price.Sign() > 0 && price.Cmp(limit) < 0 {
			liquidationData := map[common.Hash][]common.Hash{}
			priceLiquidationData := liquidationState.getAllLiquidationData(t.db)
			for lendingBook, data := range priceLiquidationData {
				if len(data) == 0 {
					continue
				}
				oldData := liquidationData[lendingBook]
				if len(oldData) == 0 {
					oldData = data
				} else {
					oldData = append(oldData, data...)
				}
				liquidationData[lendingBook] = oldData
			}
			result[price] = liquidationData
		}
	}
	return result
}

func (t *TradingStateDB) GetHighestLiquidationPriceData(orderBook common.Hash, price *big.Int) (*big.Int, map[common.Hash][]common.Hash) {
	liquidationData := map[common.Hash][]common.Hash{}
	orderbookState := t.getStateExchangeObject(orderBook)
	if orderbookState == nil {
		return common.Big0, liquidationData
	}
	highestPriceHash, liquidationState := orderbookState.getHighestLiquidationPrice(t.db)
	highestPrice := new(big.Int).SetBytes(highestPriceHash[:])
	if liquidationState != nil && highestPrice.Sign() > 0 && price.Cmp(highestPrice) < 0 {
		priceLiquidationData := liquidationState.getAllLiquidationData(t.db)
		for lendingBook, data := range priceLiquidationData {
			if len(data) == 0 {
				continue
			}
			oldData := liquidationData[lendingBook]
			if len(oldData) == 0 {
				oldData = data
			} else {
				oldData = append(oldData, data...)
			}
			liquidationData[lendingBook] = oldData
		}
	}
	return highestPrice, liquidationData
}

func (t *TradingStateDB) InsertLiquidationPrice(orderBook common.Hash, price *big.Int, lendingBook common.Hash, tradeId uint64) {
	tradIdHash := common.Uint64ToHash(tradeId)
	priceHash := common.BigToHash(price)
	orderBookState := t.getStateExchangeObject(orderBook)
	if orderBookState == nil {
		orderBookState = t.createExchangeObject(orderBook)
	}
	liquidationPriceState := orderBookState.getStateLiquidationPrice(t.db, priceHash)
	if liquidationPriceState == nil {
		liquidationPriceState = orderBookState.createStateLiquidationPrice(t.db, priceHash)
	}
	lendingBookState := liquidationPriceState.getStateLendingBook(t.db, lendingBook)
	if lendingBookState == nil {
		lendingBookState = liquidationPriceState.createLendingBook(t.db, lendingBook)
	}
	lendingBookState.insertTradingId(t.db, tradIdHash)
	lendingBookState.AddVolume(One)
	liquidationPriceState.AddVolume(One)
	orderBookState.addLendingCount(One)
	t.journal = append(t.journal, insertLiquidationPrice{
		orderBook:   orderBook,
		price:       price,
		lendingBook: lendingBook,
		tradeId:     tradeId,
	})
}

func (t *TradingStateDB) RemoveLiquidationPrice(orderBook common.Hash, price *big.Int, lendingBook common.Hash, tradeId uint64) error {
	tradeIdHash := common.Uint64ToHash(tradeId)
	priceHash := common.BigToHash(price)
	orderbookState := t.getStateExchangeObject(orderBook)
	if orderbookState == nil {
		return fmt.Errorf("not found order book: %s", orderBook.Hex())
	}
	liquidationPriceState := orderbookState.getStateLiquidationPrice(t.db, priceHash)
	if liquidationPriceState == nil {
		return fmt.Errorf("not found liquidation price: %s , %s", orderBook.Hex(), priceHash.Hex())
	}
	lendingBookState := liquidationPriceState.getStateLendingBook(t.db, lendingBook)
	if lendingBookState == nil {
		return fmt.Errorf("not found lending book: %s , %s ,%s", orderBook.Hex(), priceHash.Hex(), lendingBook.Hex())
	}
	if !lendingBookState.Exist(t.db, tradeIdHash) {
		return fmt.Errorf("not found trade id: %s, %s ,%s , %d", orderBook.Hex(), priceHash.Hex(), lendingBook.Hex(), tradeId)
	}
	lendingBookState.removeTradingId(t.db, tradeIdHash)
	lendingBookState.subVolume(One)
	liquidationPriceState.subVolume(One)
	if liquidationPriceState.Volume().Sign() == 0 {
		err := orderbookState.getLiquidationPriceTrie(t.db).TryDelete(priceHash[:])
		if err != nil {
			log.Warn("RemoveLiquidationPrice getLiquidationPriceTrie.TryDelete", "err", err, "priceHash", priceHash[:])
		}
	}
	orderbookState.subLendingCount(One)
	t.journal = append(t.journal, removeLiquidationPrice{
		orderBook:   orderBook,
		price:       price,
		lendingBook: lendingBook,
		tradeId:     tradeId,
	})
	return nil
}
