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
func (self *TradingStateDB) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (self *TradingStateDB) Error() error {
	return self.dbErr
}

// Exist reports whether the given orderId address exists in the state.
// Notably this also returns true for suicided exchanges.
func (self *TradingStateDB) Exist(addr common.Hash) bool {
	return self.getStateExchangeObject(addr) != nil
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (self *TradingStateDB) Empty(addr common.Hash) bool {
	so := self.getStateExchangeObject(addr)
	return so == nil || so.empty()
}

func (self *TradingStateDB) GetNonce(addr common.Hash) uint64 {
	stateObject := self.getStateExchangeObject(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}
	return 0
}

func (self *TradingStateDB) GetLastPrice(addr common.Hash) *big.Int {
	stateObject := self.getStateExchangeObject(addr)
	if stateObject != nil {
		return stateObject.data.LastPrice
	}
	return nil
}

func (self *TradingStateDB) GetMediumPriceBeforeEpoch(addr common.Hash) *big.Int {
	stateObject := self.getStateExchangeObject(addr)
	if stateObject != nil {
		return stateObject.data.MediumPriceBeforeEpoch
	}
	return Zero
}

func (self *TradingStateDB) GetMediumPriceAndTotalAmount(addr common.Hash) (*big.Int, *big.Int) {
	stateObject := self.getStateExchangeObject(addr)
	if stateObject != nil {
		return stateObject.data.MediumPrice, stateObject.data.TotalQuantity
	}
	return Zero, Zero
}

// Database retrieves the low level database supporting the lower level trie ops.
func (self *TradingStateDB) Database() Database {
	return self.db
}

func (self *TradingStateDB) SetNonce(addr common.Hash, nonce uint64) {
	stateObject := self.GetOrNewStateExchangeObject(addr)
	if stateObject != nil {
		self.journal = append(self.journal, nonceChange{
			hash: addr,
			prev: self.GetNonce(addr),
		})
		stateObject.SetNonce(nonce)
	}
}

func (self *TradingStateDB) SetLastPrice(addr common.Hash, price *big.Int) {
	stateObject := self.GetOrNewStateExchangeObject(addr)
	if stateObject != nil {
		self.journal = append(self.journal, lastPriceChange{
			hash: addr,
			prev: stateObject.data.LastPrice,
		})
		stateObject.setLastPrice(price)
	}
}

func (self *TradingStateDB) SetMediumPrice(addr common.Hash, price *big.Int, quantity *big.Int) {
	stateObject := self.GetOrNewStateExchangeObject(addr)
	if stateObject != nil {
		self.journal = append(self.journal, mediumPriceChange{
			hash:         addr,
			prevPrice:    stateObject.data.MediumPrice,
			prevQuantity: stateObject.data.TotalQuantity,
		})
		stateObject.setMediumPrice(price, quantity)
	}
}

func (self *TradingStateDB) SetMediumPriceBeforeEpoch(addr common.Hash, price *big.Int) {
	stateObject := self.GetOrNewStateExchangeObject(addr)
	if stateObject != nil {
		self.journal = append(self.journal, mediumPriceBeforeEpochChange{
			hash:      addr,
			prevPrice: stateObject.data.MediumPriceBeforeEpoch,
		})
		stateObject.setMediumPriceBeforeEpoch(price)
	}
}

func (self *TradingStateDB) InsertOrderItem(orderBook common.Hash, orderId common.Hash, order OrderItem) {
	priceHash := common.BigToHash(order.Price)
	stateExchange := self.getStateExchangeObject(orderBook)
	if stateExchange == nil {
		stateExchange = self.createExchangeObject(orderBook)
	}
	var stateOrderList *stateOrderList
	switch order.Side {
	case Ask:
		stateOrderList = stateExchange.getStateOrderListAskObject(self.db, priceHash)
		if stateOrderList == nil {
			stateOrderList = stateExchange.createStateOrderListAskObject(self.db, priceHash)
		}
	case Bid:
		stateOrderList = stateExchange.getStateBidOrderListObject(self.db, priceHash)
		if stateOrderList == nil {
			stateOrderList = stateExchange.createStateBidOrderListObject(self.db, priceHash)
		}
	default:
		return
	}
	self.journal = append(self.journal, insertOrder{
		orderBook: orderBook,
		orderId:   orderId,
		order:     &order,
	})
	stateExchange.createStateOrderObject(self.db, orderId, order)
	stateOrderList.insertOrderItem(self.db, orderId, common.BigToHash(order.Quantity))
	stateOrderList.AddVolume(order.Quantity)
}

func (self *TradingStateDB) GetOrder(orderBook common.Hash, orderId common.Hash) OrderItem {
	stateObject := self.GetOrNewStateExchangeObject(orderBook)
	if stateObject == nil {
		return EmptyOrder
	}
	stateOrderItem := stateObject.getStateOrderObject(self.db, orderId)
	if stateOrderItem == nil {
		return EmptyOrder
	}
	return stateOrderItem.data
}
func (self *TradingStateDB) SubAmountOrderItem(orderBook common.Hash, orderId common.Hash, price *big.Int, amount *big.Int, side string) error {
	priceHash := common.BigToHash(price)
	stateObject := self.GetOrNewStateExchangeObject(orderBook)
	if stateObject == nil {
		return fmt.Errorf("Order book not found : %s ", orderBook.Hex())
	}
	var stateOrderList *stateOrderList
	switch side {
	case Ask:
		stateOrderList = stateObject.getStateOrderListAskObject(self.db, priceHash)
	case Bid:
		stateOrderList = stateObject.getStateBidOrderListObject(self.db, priceHash)
	default:
		return fmt.Errorf("Order type not found : %s ", side)
	}
	if stateOrderList == nil || stateOrderList.empty() {
		return fmt.Errorf("Order list empty  order book : %s , order id  : %s , price  : %s ", orderBook, orderId.Hex(), priceHash.Hex())
	}
	stateOrderItem := stateObject.getStateOrderObject(self.db, orderId)
	if stateOrderItem == nil || stateOrderItem.empty() {
		return fmt.Errorf("Order item empty  order book : %s , order id  : %s , price  : %s ", orderBook, orderId.Hex(), priceHash.Hex())
	}
	currentAmount := new(big.Int).SetBytes(stateOrderList.GetOrderAmount(self.db, orderId).Bytes()[:])
	if currentAmount.Cmp(amount) < 0 {
		return fmt.Errorf("Order amount not enough : %s , have : %d , want : %d ", orderId.Hex(), currentAmount, amount)
	}
	self.journal = append(self.journal, subAmountOrder{
		orderBook: orderBook,
		orderId:   orderId,
		order:     self.GetOrder(orderBook, orderId),
		amount:    amount,
	})
	newAmount := new(big.Int).Sub(currentAmount, amount)
	log.Debug("SubAmountOrderItem", "orderId", orderId.Hex(), "side", side, "price", price.Uint64(), "amount", amount.Uint64(), "new amount", newAmount.Uint64())
	stateOrderList.subVolume(amount)
	stateOrderItem.setVolume(newAmount)
	if newAmount.Sign() == 0 {
		stateOrderList.removeOrderItem(self.db, orderId)
	} else {
		stateOrderList.setOrderItem(orderId, common.BigToHash(newAmount))
	}
	if stateOrderList.empty() {
		switch side {
		case Ask:
			stateObject.removeStateOrderListAskObject(self.db, stateOrderList)
		case Bid:
			stateObject.removeStateOrderListBidObject(self.db, stateOrderList)
		default:
		}
	}
	return nil
}

func (self *TradingStateDB) CancelOrder(orderBook common.Hash, order *OrderItem) error {
	orderIdHash := common.BigToHash(new(big.Int).SetUint64(order.OrderID))
	stateObject := self.GetOrNewStateExchangeObject(orderBook)
	if stateObject == nil {
		return fmt.Errorf("Order book not found : %s ", orderBook.Hex())
	}
	stateOrderItem := stateObject.getStateOrderObject(self.db, orderIdHash)
	if stateOrderItem == nil || stateOrderItem.empty() {
		return fmt.Errorf("Order item empty  order book : %s , order id  : %s ", orderBook, orderIdHash.Hex())
	}
	priceHash := common.BigToHash(stateOrderItem.data.Price)
	var stateOrderList *stateOrderList
	switch stateOrderItem.data.Side {
	case Ask:
		stateOrderList = stateObject.getStateOrderListAskObject(self.db, priceHash)
	case Bid:
		stateOrderList = stateObject.getStateBidOrderListObject(self.db, priceHash)
	default:
		return fmt.Errorf("Order side not found : %s ", order.Side)
	}
	if stateOrderList == nil || stateOrderList.empty() {
		return fmt.Errorf("Order list empty  order book : %s , order id  : %s , price  : %s ", orderBook, orderIdHash.Hex(), priceHash.Hex())
	}

	if stateOrderItem.data.UserAddress != order.UserAddress {
		return fmt.Errorf("Error Order User Address mismatch when cancel order book : %s , order id  : %s , got : %s , expect : %s ", orderBook, orderIdHash.Hex(), stateOrderItem.data.UserAddress.Hex(), order.UserAddress.Hex())
	}
	if stateOrderItem.data.Hash != order.Hash {
		return fmt.Errorf("Invalid order hash :  got : %s , expect : %s ", order.Hash.Hex(), stateOrderItem.data.Hash.Hex())
	}
	if stateOrderItem.data.ExchangeAddress != order.ExchangeAddress {
		return fmt.Errorf("Exchange Address mismatch when cancel. order book : %s , order id  : %s , got : %s , expect : %s ", orderBook, orderIdHash.Hex(), order.ExchangeAddress.Hex(), stateOrderItem.data.ExchangeAddress.Hex())
	}
	self.journal = append(self.journal, cancelOrder{
		orderBook: orderBook,
		orderId:   orderIdHash,
		order:     stateOrderItem.data,
	})
	currentAmount := new(big.Int).SetBytes(stateOrderList.GetOrderAmount(self.db, orderIdHash).Bytes()[:])
	stateOrderItem.setVolume(big.NewInt(0))
	stateOrderList.subVolume(currentAmount)
	stateOrderList.removeOrderItem(self.db, orderIdHash)
	if stateOrderList.empty() {
		switch stateOrderItem.data.Side {
		case Ask:
			stateObject.removeStateOrderListAskObject(self.db, stateOrderList)
		case Bid:
			stateObject.removeStateOrderListBidObject(self.db, stateOrderList)
		default:
		}
	}
	return nil
}

func (self *TradingStateDB) GetVolume(orderBook common.Hash, price *big.Int, orderType string) *big.Int {
	stateObject := self.GetOrNewStateExchangeObject(orderBook)
	var volume *big.Int = nil
	if stateObject != nil {
		var stateOrderList *stateOrderList
		switch orderType {
		case Ask:
			stateOrderList = stateObject.getStateOrderListAskObject(self.db, common.BigToHash(price))
		case Bid:
			stateOrderList = stateObject.getStateBidOrderListObject(self.db, common.BigToHash(price))
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
func (self *TradingStateDB) GetBestAskPrice(orderBook common.Hash) (*big.Int, *big.Int) {
	stateObject := self.getStateExchangeObject(orderBook)
	if stateObject != nil {
		priceHash := stateObject.getBestPriceAsksTrie(self.db)
		if common.EmptyHash(priceHash) {
			return Zero, Zero
		}
		orderList := stateObject.getStateOrderListAskObject(self.db, priceHash)
		if orderList == nil {
			log.Error("order list ask not found", "price", priceHash.Hex())
			return Zero, Zero
		}
		return new(big.Int).SetBytes(priceHash.Bytes()), orderList.Volume()
	}
	return Zero, Zero
}

func (self *TradingStateDB) GetBestBidPrice(orderBook common.Hash) (*big.Int, *big.Int) {
	stateObject := self.getStateExchangeObject(orderBook)
	if stateObject != nil {
		priceHash := stateObject.getBestBidsTrie(self.db)
		if common.EmptyHash(priceHash) {
			return Zero, Zero
		}
		orderList := stateObject.getStateBidOrderListObject(self.db, priceHash)
		if orderList == nil {
			log.Error("order list bid not found", "price", priceHash.Hex())
			return Zero, Zero
		}
		return new(big.Int).SetBytes(priceHash.Bytes()), orderList.Volume()
	}
	return Zero, Zero
}

func (self *TradingStateDB) GetBestOrderIdAndAmount(orderBook common.Hash, price *big.Int, side string) (common.Hash, *big.Int, error) {
	stateObject := self.GetOrNewStateExchangeObject(orderBook)
	if stateObject != nil {
		var stateOrderList *stateOrderList
		switch side {
		case Ask:
			stateOrderList = stateObject.getStateOrderListAskObject(self.db, common.BigToHash(price))
		case Bid:
			stateOrderList = stateObject.getStateBidOrderListObject(self.db, common.BigToHash(price))
		default:
			return EmptyHash, Zero, fmt.Errorf("not found side :%s ", side)
		}
		if stateOrderList != nil {
			key, _, err := stateOrderList.getTrie(self.db).TryGetBestLeftKeyAndValue()
			if err != nil {
				return EmptyHash, Zero, err
			}
			orderId := common.BytesToHash(key)
			amount := stateOrderList.GetOrderAmount(self.db, orderId)
			return orderId, new(big.Int).SetBytes(amount.Bytes()), nil
		}
		return EmptyHash, Zero, fmt.Errorf("not found order list with orderBook : %s , price : %d , side :%s ", orderBook.Hex(), price, side)
	}
	return EmptyHash, Zero, fmt.Errorf("not found orderBook : %s ", orderBook.Hex())
}

// updateStateExchangeObject writes the given object to the trie.
func (self *TradingStateDB) updateStateExchangeObject(stateObject *tradingExchanges) {
	addr := stateObject.Hash()
	data, err := rlp.EncodeToBytes(stateObject)
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
	}
	self.setError(self.trie.TryUpdate(addr[:], data))
}

// Retrieve a state object given my the address. Returns nil if not found.
func (self *TradingStateDB) getStateExchangeObject(addr common.Hash) (stateObject *tradingExchanges) {
	// Prefer 'live' objects.
	if obj := self.stateExhangeObjects[addr]; obj != nil {
		return obj
	}
	// Load the object from the database.
	enc, err := self.trie.TryGet(addr[:])
	if len(enc) == 0 {
		self.setError(err)
		return nil
	}
	var data tradingExchangeObject
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state object", "addr", addr, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newStateExchanges(self, addr, data, self.MarkStateExchangeObjectDirty)
	self.stateExhangeObjects[addr] = obj
	return obj
}

func (self *TradingStateDB) setStateExchangeObject(object *tradingExchanges) {
	self.stateExhangeObjects[object.Hash()] = object
	self.stateExhangeObjectsDirty[object.Hash()] = struct{}{}
}

// Retrieve a state object or create a new state object if nil.
func (self *TradingStateDB) GetOrNewStateExchangeObject(addr common.Hash) *tradingExchanges {
	stateExchangeObject := self.getStateExchangeObject(addr)
	if stateExchangeObject == nil {
		stateExchangeObject = self.createExchangeObject(addr)
	}
	return stateExchangeObject
}

// MarkStateAskObjectDirty adds the specified object to the dirty map to avoid costly
// state object cache iteration to find a handful of modified ones.
func (self *TradingStateDB) MarkStateExchangeObjectDirty(addr common.Hash) {
	self.stateExhangeObjectsDirty[addr] = struct{}{}
}

// createStateOrderListObject creates a new state object. If there is an existing orderId with
// the given address, it is overwritten and returned as the second return value.
func (self *TradingStateDB) createExchangeObject(hash common.Hash) (newobj *tradingExchanges) {
	newobj = newStateExchanges(self, hash, tradingExchangeObject{LendingCount: Zero, MediumPrice: Zero, MediumPriceBeforeEpoch: Zero, TotalQuantity: Zero}, self.MarkStateExchangeObjectDirty)
	newobj.setNonce(0) // sets the object to dirty
	self.setStateExchangeObject(newobj)
	return newobj
}

// Copy creates a deep, independent copy of the state.
// Snapshots of the copied state cannot be applied to the copy.
func (self *TradingStateDB) Copy() *TradingStateDB {
	self.lock.Lock()
	defer self.lock.Unlock()

	// Copy all the basic fields, initialize the memory ones
	state := &TradingStateDB{
		db:                       self.db,
		trie:                     self.db.CopyTrie(self.trie),
		stateExhangeObjects:      make(map[common.Hash]*tradingExchanges, len(self.stateExhangeObjectsDirty)),
		stateExhangeObjectsDirty: make(map[common.Hash]struct{}, len(self.stateExhangeObjectsDirty)),
	}
	// Copy the dirty states, logs, and preimages
	for addr := range self.stateExhangeObjectsDirty {
		state.stateExhangeObjectsDirty[addr] = struct{}{}
	}
	for addr, exchangeObject := range self.stateExhangeObjects {
		state.stateExhangeObjects[addr] = exchangeObject.deepCopy(state, state.MarkStateExchangeObjectDirty)
	}

	return state
}

func (s *TradingStateDB) clearJournalAndRefund() {
	s.journal = nil
	s.validRevisions = s.validRevisions[:0]
}

// Snapshot returns an identifier for the current revision of the state.
func (self *TradingStateDB) Snapshot() int {
	id := self.nextRevisionId
	self.nextRevisionId++
	self.validRevisions = append(self.validRevisions, revision{id, len(self.journal)})
	return id
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (self *TradingStateDB) RevertToSnapshot(revid int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(self.validRevisions), func(i int) bool {
		return self.validRevisions[i].id >= revid
	})
	if idx == len(self.validRevisions) || self.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := self.validRevisions[idx].journalIndex

	// Replay the journal to undo changes.
	for i := len(self.journal) - 1; i >= snapshot; i-- {
		self.journal[i].undo(self)
	}
	self.journal = self.journal[:snapshot]

	// Remove invalidated snapshots from the stack.
	self.validRevisions = self.validRevisions[:idx]
}

// Finalise finalises the state by removing the self destructed objects
// and clears the journal as well as the refunds.
func (s *TradingStateDB) Finalise() {
	// Commit objects to the trie.
	for addr, stateObject := range s.stateExhangeObjects {
		if _, isDirty := s.stateExhangeObjectsDirty[addr]; isDirty {
			// Write any storage changes in the state object to its storage trie.
			stateObject.updateAsksRoot(s.db)
			stateObject.updateBidsRoot(s.db)
			stateObject.updateOrdersRoot(s.db)
			stateObject.updateLiquidationPriceRoot(s.db)
			// Update the object in the main orderId trie.
			s.updateStateExchangeObject(stateObject)
			//delete(s.stateExhangeObjectsDirty, addr)
		}
	}
	s.clearJournalAndRefund()
}

// IntermediateRoot computes the current root orderBookHash of the state trie.
// It is called in between transactions to get the root orderBookHash that
// goes into transaction receipts.
func (s *TradingStateDB) IntermediateRoot() common.Hash {
	s.Finalise()
	return s.trie.Hash()
}

// Commit writes the state to the underlying in-memory trie database.
func (s *TradingStateDB) Commit() (root common.Hash, err error) {
	defer s.clearJournalAndRefund()
	// Commit objects to the trie.
	for addr, stateObject := range s.stateExhangeObjects {
		if _, isDirty := s.stateExhangeObjectsDirty[addr]; isDirty {
			// Write any storage changes in the state object to its storage trie.
			if err := stateObject.CommitAsksTrie(s.db); err != nil {
				return EmptyHash, err
			}
			if err := stateObject.CommitBidsTrie(s.db); err != nil {
				return EmptyHash, err
			}
			if err := stateObject.CommitOrdersTrie(s.db); err != nil {
				return EmptyHash, err
			}
			if err := stateObject.CommitLiquidationPriceTrie(s.db); err != nil {
				return EmptyHash, err
			}
			// Update the object in the main orderId trie.
			s.updateStateExchangeObject(stateObject)
			delete(s.stateExhangeObjectsDirty, addr)
		}
	}
	// Write trie changes.
	root, err = s.trie.Commit(func(leaf []byte, parent common.Hash) error {
		var exchange tradingExchangeObject
		if err := rlp.DecodeBytes(leaf, &exchange); err != nil {
			return nil
		}
		if exchange.AskRoot != EmptyRoot {
			s.db.TrieDB().Reference(exchange.AskRoot, parent)
		}
		if exchange.BidRoot != EmptyRoot {
			s.db.TrieDB().Reference(exchange.BidRoot, parent)
		}
		if exchange.OrderRoot != EmptyRoot {
			s.db.TrieDB().Reference(exchange.OrderRoot, parent)
		}
		if exchange.LiquidationPriceRoot != EmptyRoot {
			s.db.TrieDB().Reference(exchange.LiquidationPriceRoot, parent)
		}
		return nil
	})
	log.Debug("Trading State Trie cache stats after commit", "root", root.Hex())
	return root, err
}

func (self *TradingStateDB) GetAllLowerLiquidationPriceData(orderBook common.Hash, limit *big.Int) map[*big.Int]map[common.Hash][]common.Hash {
	result := map[*big.Int]map[common.Hash][]common.Hash{}
	orderbookState := self.getStateExchangeObject(orderBook)
	if orderbookState == nil {
		return result
	}
	mapPrices := orderbookState.getAllLowerLiquidationPrice(self.db, common.BigToHash(limit))
	for priceHash, liquidationState := range mapPrices {
		price := new(big.Int).SetBytes(priceHash[:])
		log.Debug("GetAllLowerLiquidationPriceData", "price", price, "limit", limit)
		if liquidationState != nil && price.Sign() > 0 && price.Cmp(limit) < 0 {
			liquidationData := map[common.Hash][]common.Hash{}
			priceLiquidationData := liquidationState.getAllLiquidationData(self.db)
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

func (self *TradingStateDB) GetHighestLiquidationPriceData(orderBook common.Hash, price *big.Int) (*big.Int, map[common.Hash][]common.Hash) {
	liquidationData := map[common.Hash][]common.Hash{}
	orderbookState := self.getStateExchangeObject(orderBook)
	if orderbookState == nil {
		return common.Big0, liquidationData
	}
	highestPriceHash, liquidationState := orderbookState.getHighestLiquidationPrice(self.db)
	highestPrice := new(big.Int).SetBytes(highestPriceHash[:])
	if liquidationState != nil && highestPrice.Sign() > 0 && price.Cmp(highestPrice) < 0 {
		priceLiquidationData := liquidationState.getAllLiquidationData(self.db)
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

func (self *TradingStateDB) InsertLiquidationPrice(orderBook common.Hash, price *big.Int, lendingBook common.Hash, tradeId uint64) {
	tradIdHash := common.Uint64ToHash(tradeId)
	priceHash := common.BigToHash(price)
	orderBookState := self.getStateExchangeObject(orderBook)
	if orderBookState == nil {
		orderBookState = self.createExchangeObject(orderBook)
	}
	liquidationPriceState := orderBookState.getStateLiquidationPrice(self.db, priceHash)
	if liquidationPriceState == nil {
		liquidationPriceState = orderBookState.createStateLiquidationPrice(self.db, priceHash)
	}
	lendingBookState := liquidationPriceState.getStateLendingBook(self.db, lendingBook)
	if lendingBookState == nil {
		lendingBookState = liquidationPriceState.createLendingBook(self.db, lendingBook)
	}
	lendingBookState.insertTradingId(self.db, tradIdHash)
	lendingBookState.AddVolume(One)
	liquidationPriceState.AddVolume(One)
	orderBookState.addLendingCount(One)
	self.journal = append(self.journal, insertLiquidationPrice{
		orderBook:   orderBook,
		price:       price,
		lendingBook: lendingBook,
		tradeId:     tradeId,
	})
}

func (self *TradingStateDB) RemoveLiquidationPrice(orderBook common.Hash, price *big.Int, lendingBook common.Hash, tradeId uint64) error {
	tradeIdHash := common.Uint64ToHash(tradeId)
	priceHash := common.BigToHash(price)
	orderbookState := self.getStateExchangeObject(orderBook)
	if orderbookState == nil {
		return fmt.Errorf("order book not found : %s ", orderBook.Hex())
	}
	liquidationPriceState := orderbookState.getStateLiquidationPrice(self.db, priceHash)
	if liquidationPriceState == nil {
		return fmt.Errorf("liquidation price not found : %s , %s ", orderBook.Hex(), priceHash.Hex())
	}
	lendingBookState := liquidationPriceState.getStateLendingBook(self.db, lendingBook)
	if lendingBookState == nil {
		return fmt.Errorf("lending book not found : %s , %s ,%s ", orderBook.Hex(), priceHash.Hex(), lendingBook.Hex())
	}
	if !lendingBookState.Exist(self.db, tradeIdHash) {
		return fmt.Errorf("trade id not found : %s , %s ,%s , %d ", orderBook.Hex(), priceHash.Hex(), lendingBook.Hex(), tradeId)
	}
	lendingBookState.removeTradingId(self.db, tradeIdHash)
	lendingBookState.subVolume(One)
	liquidationPriceState.subVolume(One)
	if liquidationPriceState.Volume().Sign() == 0 {
		orderbookState.getLiquidationPriceTrie(self.db).TryDelete(priceHash[:])
	}
	orderbookState.subLendingCount(One)
	self.journal = append(self.journal, removeLiquidationPrice{
		orderBook:   orderBook,
		price:       price,
		lendingBook: lendingBook,
		tradeId:     tradeId,
	})
	return nil
}
