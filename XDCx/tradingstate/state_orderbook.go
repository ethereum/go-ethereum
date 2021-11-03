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
)

// stateObject represents an Ethereum orderId which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// tradingExchangeObject values can be accessed and modified through the object.
// Finally, call CommitAskTrie to write the modified storage trie into a database.
type tradingExchanges struct {
	orderBookHash common.Hash
	data          tradingExchangeObject
	db            *TradingStateDB

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by TradingStateDB.Commit.
	dbErr error

	// Write caches.
	asksTrie             Trie // storage trie, which becomes non-nil on first access
	bidsTrie             Trie // storage trie, which becomes non-nil on first access
	ordersTrie           Trie // storage trie, which becomes non-nil on first access
	liquidationPriceTrie Trie

	stateAskObjects      map[common.Hash]*stateOrderList
	stateAskObjectsDirty map[common.Hash]struct{}

	stateBidObjects      map[common.Hash]*stateOrderList
	stateBidObjectsDirty map[common.Hash]struct{}

	stateOrderObjects      map[common.Hash]*stateOrderItem
	stateOrderObjectsDirty map[common.Hash]struct{}

	liquidationPriceStates      map[common.Hash]*liquidationPriceState
	liquidationPriceStatesDirty map[common.Hash]struct{}

	onDirty func(hash common.Hash) // Callback method to mark a state object newly dirty
}

// empty returns whether the orderId is considered empty.
func (s *tradingExchanges) empty() bool {
	if s.data.Nonce != 0 {
		return false
	}
	if s.data.LendingCount != nil && s.data.LendingCount.Sign() > 0 {
		return false
	}
	if s.data.LastPrice != nil && s.data.LastPrice.Sign() > 0 {
		return false
	}
	if s.data.MediumPrice != nil && s.data.MediumPrice.Sign() > 0 {
		return false
	}
	if s.data.MediumPriceBeforeEpoch != nil && s.data.MediumPriceBeforeEpoch.Sign() > 0 {
		return false
	}
	if s.data.TotalQuantity != nil && s.data.TotalQuantity.Sign() > 0 {
		return false
	}
	if !common.EmptyHash(s.data.AskRoot) {
		return false
	}
	if !common.EmptyHash(s.data.BidRoot) {
		return false
	}
	if !common.EmptyHash(s.data.OrderRoot) {
		return false
	}
	if !common.EmptyHash(s.data.LiquidationPriceRoot) {
		return false
	}
	return true
}

// newObject creates a state object.
func newStateExchanges(db *TradingStateDB, hash common.Hash, data tradingExchangeObject, onDirty func(addr common.Hash)) *tradingExchanges {
	return &tradingExchanges{
		db:                          db,
		orderBookHash:               hash,
		data:                        data,
		stateAskObjects:             make(map[common.Hash]*stateOrderList),
		stateBidObjects:             make(map[common.Hash]*stateOrderList),
		stateOrderObjects:           make(map[common.Hash]*stateOrderItem),
		liquidationPriceStates:      make(map[common.Hash]*liquidationPriceState),
		stateAskObjectsDirty:        make(map[common.Hash]struct{}),
		stateBidObjectsDirty:        make(map[common.Hash]struct{}),
		stateOrderObjectsDirty:      make(map[common.Hash]struct{}),
		liquidationPriceStatesDirty: make(map[common.Hash]struct{}),
		onDirty:                     onDirty,
	}
}

// EncodeRLP implements rlp.Encoder.
func (c *tradingExchanges) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

// setError remembers the first non-nil error it is called with.
func (self *tradingExchanges) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (c *tradingExchanges) getAsksTrie(db Database) Trie {
	if c.asksTrie == nil {
		var err error
		c.asksTrie, err = db.OpenStorageTrie(c.orderBookHash, c.data.AskRoot)
		if err != nil {
			c.asksTrie, _ = db.OpenStorageTrie(c.orderBookHash, EmptyHash)
			c.setError(fmt.Errorf("can't create asks trie: %v", err))
		}
	}
	return c.asksTrie
}

func (c *tradingExchanges) getOrdersTrie(db Database) Trie {
	if c.ordersTrie == nil {
		var err error
		c.ordersTrie, err = db.OpenStorageTrie(c.orderBookHash, c.data.OrderRoot)
		if err != nil {
			c.ordersTrie, _ = db.OpenStorageTrie(c.orderBookHash, EmptyHash)
			c.setError(fmt.Errorf("can't create asks trie: %v", err))
		}
	}
	return c.ordersTrie
}

func (c *tradingExchanges) getBestPriceAsksTrie(db Database) common.Hash {
	trie := c.getAsksTrie(db)
	encKey, encValue, err := trie.TryGetBestLeftKeyAndValue()
	if err != nil {
		log.Error("Failed find best price ask trie ", "orderbook", c.orderBookHash.Hex())
		return EmptyHash
	}
	if len(encKey) == 0 || len(encValue) == 0 {
		log.Debug("Not found get best ask trie", "encKey", encKey, "encValue", encValue)
		return EmptyHash
	}
	price := common.BytesToHash(encKey)
	if _, exit := c.stateAskObjects[price]; !exit {
		var data orderList
		if err := rlp.DecodeBytes(encValue, &data); err != nil {
			log.Error("Failed to decode state get best ask trie", "err", err)
			return EmptyHash
		}
		obj := newStateOrderList(c.db, Bid, c.orderBookHash, price, data, c.MarkStateAskObjectDirty)
		c.stateAskObjects[price] = obj
	}
	return common.BytesToHash(encKey)
}

func (c *tradingExchanges) getBestBidsTrie(db Database) common.Hash {
	trie := c.getBidsTrie(db)
	encKey, encValue, err := trie.TryGetBestRightKeyAndValue()
	if err != nil {
		log.Error("Failed find best price bid trie ", "orderbook", c.orderBookHash.Hex())
		return EmptyHash
	}
	if len(encKey) == 0 || len(encValue) == 0 {
		log.Debug("Not found get best bid trie", "encKey", encKey, "encValue", encValue)
		return EmptyHash
	}
	price := common.BytesToHash(encKey)
	if _, exit := c.stateBidObjects[price]; !exit {
		var data orderList
		if err := rlp.DecodeBytes(encValue, &data); err != nil {
			log.Error("Failed to decode state get best bid trie", "err", err)
			return EmptyHash
		}
		// Insert into the live set.
		obj := newStateOrderList(c.db, Bid, c.orderBookHash, price, data, c.MarkStateBidObjectDirty)
		c.stateBidObjects[price] = obj
	}
	return common.BytesToHash(encKey)
}

// updateAskTrie writes cached storage modifications into the object's storage trie.
func (self *tradingExchanges) updateAsksTrie(db Database) Trie {
	tr := self.getAsksTrie(db)
	for price, orderList := range self.stateAskObjects {
		if _, isDirty := self.stateAskObjectsDirty[price]; isDirty {
			delete(self.stateAskObjectsDirty, price)
			if orderList.empty() {
				self.setError(tr.TryDelete(price[:]))
				continue
			}
			orderList.updateRoot(db)
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(orderList)
			self.setError(tr.TryUpdate(price[:], v))
		}
	}

	return tr
}

// CommitAskTrie the storage trie of the object to dwb.
// This updates the trie root.
func (self *tradingExchanges) updateAsksRoot(db Database) error {
	self.updateAsksTrie(db)
	if self.dbErr != nil {
		return self.dbErr
	}
	self.data.AskRoot = self.asksTrie.Hash()
	return nil
}

// CommitAskTrie the storage trie of the object to dwb.
// This updates the trie root.
func (self *tradingExchanges) CommitAsksTrie(db Database) error {
	self.updateAsksTrie(db)
	if self.dbErr != nil {
		return self.dbErr
	}
	root, err := self.asksTrie.Commit(func(leaf []byte, parent common.Hash) error {
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
		self.data.AskRoot = root
	}
	return err
}

func (c *tradingExchanges) getBidsTrie(db Database) Trie {
	if c.bidsTrie == nil {
		var err error
		c.bidsTrie, err = db.OpenStorageTrie(c.orderBookHash, c.data.BidRoot)
		if err != nil {
			c.bidsTrie, _ = db.OpenStorageTrie(c.orderBookHash, EmptyHash)
			c.setError(fmt.Errorf("can't create bids trie: %v", err))
		}
	}
	return c.bidsTrie
}

// updateAskTrie writes cached storage modifications into the object's storage trie.
func (self *tradingExchanges) updateBidsTrie(db Database) Trie {
	tr := self.getBidsTrie(db)
	for price, orderList := range self.stateBidObjects {
		if _, isDirty := self.stateBidObjectsDirty[price]; isDirty {
			delete(self.stateBidObjectsDirty, price)
			if orderList.empty() {
				self.setError(tr.TryDelete(price[:]))
				continue
			}
			orderList.updateRoot(db)
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(orderList)
			self.setError(tr.TryUpdate(price[:], v))
		}
	}
	return tr
}

func (self *tradingExchanges) updateBidsRoot(db Database) {
	self.updateBidsTrie(db)
	self.data.BidRoot = self.bidsTrie.Hash()
}

// CommitAskTrie the storage trie of the object to dwb.
// This updates the trie root.
func (self *tradingExchanges) CommitBidsTrie(db Database) error {
	self.updateBidsTrie(db)
	if self.dbErr != nil {
		return self.dbErr
	}
	root, err := self.bidsTrie.Commit(func(leaf []byte, parent common.Hash) error {
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
		self.data.BidRoot = root
	}
	return err
}

func (self *tradingExchanges) deepCopy(db *TradingStateDB, onDirty func(hash common.Hash)) *tradingExchanges {
	stateExchanges := newStateExchanges(db, self.orderBookHash, self.data, onDirty)
	if self.asksTrie != nil {
		stateExchanges.asksTrie = db.db.CopyTrie(self.asksTrie)
	}
	if self.bidsTrie != nil {
		stateExchanges.bidsTrie = db.db.CopyTrie(self.bidsTrie)
	}
	if self.ordersTrie != nil {
		stateExchanges.ordersTrie = db.db.CopyTrie(self.ordersTrie)
	}
	for price, bidObject := range self.stateBidObjects {
		stateExchanges.stateBidObjects[price] = bidObject.deepCopy(db, self.MarkStateBidObjectDirty)
	}
	for price := range self.stateBidObjectsDirty {
		stateExchanges.stateBidObjectsDirty[price] = struct{}{}
	}
	for price, askObject := range self.stateAskObjects {
		stateExchanges.stateAskObjects[price] = askObject.deepCopy(db, self.MarkStateAskObjectDirty)
	}
	for price := range self.stateAskObjectsDirty {
		stateExchanges.stateAskObjectsDirty[price] = struct{}{}
	}
	for orderId, orderItem := range self.stateOrderObjects {
		stateExchanges.stateOrderObjects[orderId] = orderItem.deepCopy(self.MarkStateOrderObjectDirty)
	}
	for orderId := range self.stateOrderObjectsDirty {
		stateExchanges.stateOrderObjectsDirty[orderId] = struct{}{}
	}
	for price, liquidationPrice := range self.liquidationPriceStates {
		stateExchanges.liquidationPriceStates[price] = liquidationPrice.deepCopy(db, self.MarkStateLiquidationPriceDirty)
	}
	for price := range self.liquidationPriceStatesDirty {
		stateExchanges.liquidationPriceStatesDirty[price] = struct{}{}
	}
	return stateExchanges
}

// Returns the address of the contract/orderId
func (c *tradingExchanges) Hash() common.Hash {
	return c.orderBookHash
}

func (self *tradingExchanges) SetNonce(nonce uint64) {
	self.setNonce(nonce)
}

func (self *tradingExchanges) setNonce(nonce uint64) {
	self.data.Nonce = nonce
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
}

func (self *tradingExchanges) Nonce() uint64 {
	return self.data.Nonce
}

func (self *tradingExchanges) setLastPrice(price *big.Int) {
	self.data.LastPrice = price
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
}

func (self *tradingExchanges) setMediumPriceBeforeEpoch(price *big.Int) {
	self.data.MediumPriceBeforeEpoch = price
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
}

func (self *tradingExchanges) setMediumPrice(price *big.Int, quantity *big.Int) {
	self.data.MediumPrice = price
	self.data.TotalQuantity = quantity
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
}

// updateStateExchangeObject writes the given object to the trie.
func (self *tradingExchanges) removeStateOrderListAskObject(db Database, stateOrderList *stateOrderList) {
	self.setError(self.asksTrie.TryDelete(stateOrderList.price[:]))
}

// updateStateExchangeObject writes the given object to the trie.
func (self *tradingExchanges) removeStateOrderListBidObject(db Database, stateOrderList *stateOrderList) {
	self.setError(self.bidsTrie.TryDelete(stateOrderList.price[:]))
}

// Retrieve a state object given my the address. Returns nil if not found.
func (self *tradingExchanges) getStateOrderListAskObject(db Database, price common.Hash) (stateOrderList *stateOrderList) {
	// Prefer 'live' objects.
	if obj := self.stateAskObjects[price]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := self.getAsksTrie(db).TryGet(price[:])
	if len(enc) == 0 {
		self.setError(err)
		return nil
	}
	var data orderList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state order list object", "orderId", price, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newStateOrderList(self.db, Bid, self.orderBookHash, price, data, self.MarkStateAskObjectDirty)
	self.stateAskObjects[price] = obj
	return obj
}

// MarkStateAskObjectDirty adds the specified object to the dirty map to avoid costly
// state object cache iteration to find a handful of modified ones.
func (self *tradingExchanges) MarkStateAskObjectDirty(price common.Hash) {
	self.stateAskObjectsDirty[price] = struct{}{}
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
}

// createStateOrderListObject creates a new state object. If there is an existing orderId with
// the given address, it is overwritten and returned as the second return value.
func (self *tradingExchanges) createStateOrderListAskObject(db Database, price common.Hash) (newobj *stateOrderList) {
	newobj = newStateOrderList(self.db, Ask, self.orderBookHash, price, orderList{Volume: Zero}, self.MarkStateAskObjectDirty)
	self.stateAskObjects[price] = newobj
	self.stateAskObjectsDirty[price] = struct{}{}
	data, err := rlp.EncodeToBytes(newobj)
	if err != nil {
		panic(fmt.Errorf("can't encode order list object at %x: %v", price[:], err))
	}
	self.setError(self.asksTrie.TryUpdate(price[:], data))
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
	return newobj
}

// Retrieve a state object given my the address. Returns nil if not found.
func (self *tradingExchanges) getStateBidOrderListObject(db Database, price common.Hash) (stateOrderList *stateOrderList) {
	// Prefer 'live' objects.
	if obj := self.stateBidObjects[price]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := self.getBidsTrie(db).TryGet(price[:])
	if len(enc) == 0 {
		self.setError(err)
		return nil
	}
	var data orderList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state order list object", "orderId", price, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newStateOrderList(self.db, Bid, self.orderBookHash, price, data, self.MarkStateBidObjectDirty)
	self.stateBidObjects[price] = obj
	return obj
}

// MarkStateAskObjectDirty adds the specified object to the dirty map to avoid costly
// state object cache iteration to find a handful of modified ones.
func (self *tradingExchanges) MarkStateBidObjectDirty(price common.Hash) {
	self.stateBidObjectsDirty[price] = struct{}{}
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
}

// createStateOrderListObject creates a new state object. If there is an existing orderId with
// the given address, it is overwritten and returned as the second return value.
func (self *tradingExchanges) createStateBidOrderListObject(db Database, price common.Hash) (newobj *stateOrderList) {
	newobj = newStateOrderList(self.db, Bid, self.orderBookHash, price, orderList{Volume: Zero}, self.MarkStateBidObjectDirty)
	self.stateBidObjects[price] = newobj
	self.stateBidObjectsDirty[price] = struct{}{}
	data, err := rlp.EncodeToBytes(newobj)
	if err != nil {
		panic(fmt.Errorf("can't encode order list object at %x: %v", price[:], err))
	}
	self.setError(self.bidsTrie.TryUpdate(price[:], data))
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
	return newobj
}

// Retrieve a state object given my the address. Returns nil if not found.
func (self *tradingExchanges) getStateOrderObject(db Database, orderId common.Hash) (stateOrderItem *stateOrderItem) {
	// Prefer 'live' objects.
	if obj := self.stateOrderObjects[orderId]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := self.getOrdersTrie(db).TryGet(orderId[:])
	if len(enc) == 0 {
		self.setError(err)
		return nil
	}
	var data OrderItem
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state order object", "orderId", orderId, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newStateOrderItem(self.orderBookHash, orderId, data, self.MarkStateOrderObjectDirty)
	self.stateOrderObjects[orderId] = obj
	return obj
}

// MarkStateAskObjectDirty adds the specified object to the dirty map to avoid costly
// state object cache iteration to find a handful of modified ones.
func (self *tradingExchanges) MarkStateOrderObjectDirty(orderId common.Hash) {
	self.stateOrderObjectsDirty[orderId] = struct{}{}
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
}

// createStateOrderListObject creates a new state object. If there is an existing orderId with
// the given address, it is overwritten and returned as the second return value.
func (self *tradingExchanges) createStateOrderObject(db Database, orderId common.Hash, order OrderItem) (newobj *stateOrderItem) {
	newobj = newStateOrderItem(self.orderBookHash, orderId, order, self.MarkStateOrderObjectDirty)
	orderIdHash := common.BigToHash(new(big.Int).SetUint64(order.OrderID))
	self.stateOrderObjects[orderIdHash] = newobj
	self.stateOrderObjectsDirty[orderIdHash] = struct{}{}
	if self.onDirty != nil {
		self.onDirty(self.orderBookHash)
		self.onDirty = nil
	}
	return newobj
}

// updateAskTrie writes cached storage modifications into the object's storage trie.
func (self *tradingExchanges) updateOrdersTrie(db Database) Trie {
	tr := self.getOrdersTrie(db)
	for orderId, orderItem := range self.stateOrderObjects {
		if _, isDirty := self.stateOrderObjectsDirty[orderId]; isDirty {
			delete(self.stateOrderObjectsDirty, orderId)
			if orderItem.empty() {
				self.setError(tr.TryDelete(orderId[:]))
				continue
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(orderItem)
			self.setError(tr.TryUpdate(orderId[:], v))
		}
	}
	return tr
}

// CommitAskTrie the storage trie of the object to dwb.
// This updates the trie root.
func (self *tradingExchanges) updateOrdersRoot(db Database) {
	self.updateOrdersTrie(db)
	self.data.OrderRoot = self.ordersTrie.Hash()
}

// CommitAskTrie the storage trie of the object to dwb.
// This updates the trie root.
func (self *tradingExchanges) CommitOrdersTrie(db Database) error {
	self.updateOrdersTrie(db)
	if self.dbErr != nil {
		return self.dbErr
	}
	root, err := self.ordersTrie.Commit(nil)
	if err == nil {
		self.data.OrderRoot = root
	}
	return err
}

func (self *tradingExchanges) MarkStateLiquidationPriceDirty(price common.Hash) {
	self.liquidationPriceStatesDirty[price] = struct{}{}
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
}

func (self *tradingExchanges) createStateLiquidationPrice(db Database, liquidationPrice common.Hash) (newobj *liquidationPriceState) {
	newobj = newLiquidationPriceState(self.db, self.orderBookHash, liquidationPrice, orderList{Volume: Zero}, self.MarkStateLiquidationPriceDirty)
	self.liquidationPriceStates[liquidationPrice] = newobj
	self.liquidationPriceStatesDirty[liquidationPrice] = struct{}{}
	data, err := rlp.EncodeToBytes(newobj)
	if err != nil {
		panic(fmt.Errorf("can't encode liquidation price object at %x: %v", liquidationPrice[:], err))
	}
	self.setError(self.getLiquidationPriceTrie(db).TryUpdate(liquidationPrice[:], data))
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
	return newobj
}

func (self *tradingExchanges) getLiquidationPriceTrie(db Database) Trie {
	if self.liquidationPriceTrie == nil {
		var err error
		self.liquidationPriceTrie, err = db.OpenStorageTrie(self.orderBookHash, self.data.LiquidationPriceRoot)
		if err != nil {
			self.liquidationPriceTrie, _ = db.OpenStorageTrie(self.orderBookHash, EmptyHash)
			self.setError(fmt.Errorf("can't create liquidation liquidationPrice trie: %v", err))
		}
	}
	return self.liquidationPriceTrie
}

func (self *tradingExchanges) getStateLiquidationPrice(db Database, price common.Hash) (stateObject *liquidationPriceState) {
	// Prefer 'live' objects.
	if obj := self.liquidationPriceStates[price]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := self.getLiquidationPriceTrie(db).TryGet(price[:])
	if len(enc) == 0 {
		self.setError(err)
		return nil
	}
	var data orderList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state liquidation liquidationPrice", "liquidationPrice", price, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newLiquidationPriceState(self.db, self.orderBookHash, price, data, self.MarkStateLiquidationPriceDirty)
	self.liquidationPriceStates[price] = obj
	return obj
}

func (self *tradingExchanges) getLowestLiquidationPrice(db Database) (common.Hash, *liquidationPriceState) {
	trie := self.getLiquidationPriceTrie(db)
	encKey, encValue, err := trie.TryGetBestLeftKeyAndValue()
	if err != nil {
		log.Error("Failed find best liquidationPrice ask trie ", "orderbook", self.orderBookHash.Hex())
		return EmptyHash, nil
	}
	if len(encKey) == 0 || len(encValue) == 0 {
		log.Debug("Not found get best ask trie", "encKey", encKey, "encValue", encValue)
		return EmptyHash, nil
	}
	price := common.BytesToHash(encKey)
	obj := self.liquidationPriceStates[price]
	if obj == nil {
		var data orderList
		if err := rlp.DecodeBytes(encValue, &data); err != nil {
			log.Error("Failed to decode state get best ask trie", "err", err)
			return EmptyHash, nil
		}
		obj = newLiquidationPriceState(self.db, self.orderBookHash, price, data, self.MarkStateLiquidationPriceDirty)
		self.liquidationPriceStates[price] = obj
	}
	return price, obj
}

func (self *tradingExchanges) getAllLowerLiquidationPrice(db Database, limit common.Hash) map[common.Hash]*liquidationPriceState {
	trie := self.getLiquidationPriceTrie(db)
	encKeys, encValues, err := trie.TryGetAllLeftKeyAndValue(limit.Bytes())
	result := map[common.Hash]*liquidationPriceState{}
	if err != nil || len(encKeys) != len(encValues) {
		log.Error("Failed get lower liquidation price trie ", "orderbook", self.orderBookHash.Hex(), "encKeys", len(encKeys), "encValues", len(encValues))
		return result
	}
	if len(encKeys) == 0 || len(encValues) == 0 {
		log.Debug("Not found get lower liquidation price trie ", "limit", limit)
		return result
	}
	for i := range encKeys {
		price := common.BytesToHash(encKeys[i])
		obj := self.liquidationPriceStates[price]
		if obj == nil {
			var data orderList
			if err := rlp.DecodeBytes(encValues[i], &data); err != nil {
				log.Error("Failed to decode state get all lower liquidation price trie", "price", price, "encValues", encValues[i], "err", err)
				return result
			}
			obj = newLiquidationPriceState(self.db, self.orderBookHash, price, data, self.MarkStateLiquidationPriceDirty)
			self.liquidationPriceStates[price] = obj
		}
		if obj.empty() {
			continue
		}
		result[price] = obj
	}
	return result
}

func (self *tradingExchanges) getHighestLiquidationPrice(db Database) (common.Hash, *liquidationPriceState) {
	trie := self.getLiquidationPriceTrie(db)
	encKey, encValue, err := trie.TryGetBestRightKeyAndValue()
	if err != nil {
		log.Error("Failed find best liquidationPrice ask trie ", "orderbook", self.orderBookHash.Hex())
		return EmptyHash, nil
	}
	if len(encKey) == 0 || len(encValue) == 0 {
		log.Debug("Not found get best ask trie", "encKey", encKey, "encValue", encValue)
		return EmptyHash, nil
	}
	price := common.BytesToHash(encKey)
	obj := self.liquidationPriceStates[price]
	if obj == nil {
		var data orderList
		if err := rlp.DecodeBytes(encValue, &data); err != nil {
			log.Error("Failed to decode state get best ask trie", "err", err)
			return EmptyHash, nil
		}
		obj = newLiquidationPriceState(self.db, self.orderBookHash, price, data, self.MarkStateLiquidationPriceDirty)
		self.liquidationPriceStates[price] = obj
	}
	if obj.empty() {
		return EmptyHash, nil
	}
	return price, obj
}
func (self *tradingExchanges) updateLiquidationPriceTrie(db Database) Trie {
	tr := self.getLiquidationPriceTrie(db)
	for price, stateObject := range self.liquidationPriceStates {
		if _, isDirty := self.liquidationPriceStatesDirty[price]; isDirty {
			delete(self.liquidationPriceStatesDirty, price)
			if stateObject.empty() {
				self.setError(tr.TryDelete(price[:]))
				continue
			}
			stateObject.updateRoot(db)
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(stateObject)
			self.setError(tr.TryUpdate(price[:], v))
		}
	}
	return tr
}

func (self *tradingExchanges) updateLiquidationPriceRoot(db Database) {
	self.updateLiquidationPriceTrie(db)
	self.data.LiquidationPriceRoot = self.liquidationPriceTrie.Hash()
}

func (self *tradingExchanges) CommitLiquidationPriceTrie(db Database) error {
	self.updateLiquidationPriceTrie(db)
	if self.dbErr != nil {
		return self.dbErr
	}
	root, err := self.liquidationPriceTrie.Commit(func(leaf []byte, parent common.Hash) error {
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
		self.data.LiquidationPriceRoot = root
	}
	return err
}

func (c *tradingExchanges) addLendingCount(amount *big.Int) {
	c.setLendingCount(new(big.Int).Add(c.data.LendingCount, amount))
}

func (c *tradingExchanges) subLendingCount(amount *big.Int) {
	c.setLendingCount(new(big.Int).Sub(c.data.LendingCount, amount))
}

func (self *tradingExchanges) setLendingCount(volume *big.Int) {
	self.data.LendingCount = volume
	if self.onDirty != nil {
		self.onDirty(self.orderBookHash)
		self.onDirty = nil
	}
}
