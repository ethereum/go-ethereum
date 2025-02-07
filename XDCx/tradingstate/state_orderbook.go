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
func (te *tradingExchanges) empty() bool {
	if te.data.Nonce != 0 {
		return false
	}
	if te.data.LendingCount != nil && te.data.LendingCount.Sign() > 0 {
		return false
	}
	if te.data.LastPrice != nil && te.data.LastPrice.Sign() > 0 {
		return false
	}
	if te.data.MediumPrice != nil && te.data.MediumPrice.Sign() > 0 {
		return false
	}
	if te.data.MediumPriceBeforeEpoch != nil && te.data.MediumPriceBeforeEpoch.Sign() > 0 {
		return false
	}
	if te.data.TotalQuantity != nil && te.data.TotalQuantity.Sign() > 0 {
		return false
	}
	if !te.data.AskRoot.IsZero() {
		return false
	}
	if !te.data.BidRoot.IsZero() {
		return false
	}
	if !te.data.OrderRoot.IsZero() {
		return false
	}
	if !te.data.LiquidationPriceRoot.IsZero() {
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
func (te *tradingExchanges) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, te.data)
}

// setError remembers the first non-nil error it is called with.
func (te *tradingExchanges) setError(err error) {
	if te.dbErr == nil {
		te.dbErr = err
	}
}

func (te *tradingExchanges) getAsksTrie(db Database) Trie {
	if te.asksTrie == nil {
		var err error
		te.asksTrie, err = db.OpenStorageTrie(te.orderBookHash, te.data.AskRoot)
		if err != nil {
			te.asksTrie, _ = db.OpenStorageTrie(te.orderBookHash, types.EmptyRootHash)
			te.setError(fmt.Errorf("can't create asks trie: %v", err))
		}
	}
	return te.asksTrie
}

func (te *tradingExchanges) getOrdersTrie(db Database) Trie {
	if te.ordersTrie == nil {
		var err error
		te.ordersTrie, err = db.OpenStorageTrie(te.orderBookHash, te.data.OrderRoot)
		if err != nil {
			te.ordersTrie, _ = db.OpenStorageTrie(te.orderBookHash, types.EmptyRootHash)
			te.setError(fmt.Errorf("can't create asks trie: %v", err))
		}
	}
	return te.ordersTrie
}

func (te *tradingExchanges) getBestPriceAsksTrie(db Database) common.Hash {
	trie := te.getAsksTrie(db)
	encKey, encValue, err := trie.TryGetBestLeftKeyAndValue()
	if err != nil {
		log.Error("Failed find best price ask trie ", "orderbook", te.orderBookHash.Hex())
		return EmptyHash
	}
	if len(encKey) == 0 || len(encValue) == 0 {
		log.Debug("Not found get best ask trie", "encKey", encKey, "encValue", encValue)
		return EmptyHash
	}
	price := common.BytesToHash(encKey)
	if _, exit := te.stateAskObjects[price]; !exit {
		var data orderList
		if err := rlp.DecodeBytes(encValue, &data); err != nil {
			log.Error("Failed to decode state get best ask trie", "err", err)
			return EmptyHash
		}
		obj := newStateOrderList(te.db, Bid, te.orderBookHash, price, data, te.MarkStateAskObjectDirty)
		te.stateAskObjects[price] = obj
	}
	return common.BytesToHash(encKey)
}

func (te *tradingExchanges) getBestBidsTrie(db Database) common.Hash {
	trie := te.getBidsTrie(db)
	encKey, encValue, err := trie.TryGetBestRightKeyAndValue()
	if err != nil {
		log.Error("Failed find best price bid trie ", "orderbook", te.orderBookHash.Hex())
		return EmptyHash
	}
	if len(encKey) == 0 || len(encValue) == 0 {
		log.Debug("Not found get best bid trie", "encKey", encKey, "encValue", encValue)
		return EmptyHash
	}
	price := common.BytesToHash(encKey)
	if _, exit := te.stateBidObjects[price]; !exit {
		var data orderList
		if err := rlp.DecodeBytes(encValue, &data); err != nil {
			log.Error("Failed to decode state get best bid trie", "err", err)
			return EmptyHash
		}
		// Insert into the live set.
		obj := newStateOrderList(te.db, Bid, te.orderBookHash, price, data, te.MarkStateBidObjectDirty)
		te.stateBidObjects[price] = obj
	}
	return common.BytesToHash(encKey)
}

// updateAskTrie writes cached storage modifications into the object's storage trie.
func (te *tradingExchanges) updateAsksTrie(db Database) Trie {
	tr := te.getAsksTrie(db)
	for price, orderList := range te.stateAskObjects {
		if _, isDirty := te.stateAskObjectsDirty[price]; isDirty {
			delete(te.stateAskObjectsDirty, price)
			if orderList.empty() {
				te.setError(tr.TryDelete(price[:]))
				continue
			}
			err := orderList.updateRoot(db)
			if err != nil {
				log.Warn("updateAsksTrie updateRoot", "err", err, "price", price, "orderList", *orderList)
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(orderList)
			te.setError(tr.TryUpdate(price[:], v))
		}
	}

	return tr
}

// CommitAskTrie the storage trie of the object to dwb.
// This updates the trie root.
func (te *tradingExchanges) updateAsksRoot(db Database) error {
	te.updateAsksTrie(db)
	if te.dbErr != nil {
		return te.dbErr
	}
	te.data.AskRoot = te.asksTrie.Hash()
	return nil
}

// CommitAskTrie the storage trie of the object to dwb.
// This updates the trie root.
func (te *tradingExchanges) CommitAsksTrie(db Database) error {
	te.updateAsksTrie(db)
	if te.dbErr != nil {
		return te.dbErr
	}
	root, err := te.asksTrie.Commit(func(leaf []byte, parent common.Hash) error {
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
		te.data.AskRoot = root
	}
	return err
}

func (te *tradingExchanges) getBidsTrie(db Database) Trie {
	if te.bidsTrie == nil {
		var err error
		te.bidsTrie, err = db.OpenStorageTrie(te.orderBookHash, te.data.BidRoot)
		if err != nil {
			te.bidsTrie, _ = db.OpenStorageTrie(te.orderBookHash, types.EmptyRootHash)
			te.setError(fmt.Errorf("can't create bids trie: %v", err))
		}
	}
	return te.bidsTrie
}

// updateAskTrie writes cached storage modifications into the object's storage trie.
func (te *tradingExchanges) updateBidsTrie(db Database) Trie {
	tr := te.getBidsTrie(db)
	for price, orderList := range te.stateBidObjects {
		if _, isDirty := te.stateBidObjectsDirty[price]; isDirty {
			delete(te.stateBidObjectsDirty, price)
			if orderList.empty() {
				te.setError(tr.TryDelete(price[:]))
				continue
			}
			err := orderList.updateRoot(db)
			if err != nil {
				log.Warn("updateBidsTrie updateRoot", "err", err, "price", price, "orderList", *orderList)
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(orderList)
			te.setError(tr.TryUpdate(price[:], v))
		}
	}
	return tr
}

func (te *tradingExchanges) updateBidsRoot(db Database) {
	te.updateBidsTrie(db)
	te.data.BidRoot = te.bidsTrie.Hash()
}

// CommitAskTrie the storage trie of the object to dwb.
// This updates the trie root.
func (te *tradingExchanges) CommitBidsTrie(db Database) error {
	te.updateBidsTrie(db)
	if te.dbErr != nil {
		return te.dbErr
	}
	root, err := te.bidsTrie.Commit(func(leaf []byte, parent common.Hash) error {
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
		te.data.BidRoot = root
	}
	return err
}

func (te *tradingExchanges) deepCopy(db *TradingStateDB, onDirty func(hash common.Hash)) *tradingExchanges {
	stateExchanges := newStateExchanges(db, te.orderBookHash, te.data, onDirty)
	if te.asksTrie != nil {
		stateExchanges.asksTrie = db.db.CopyTrie(te.asksTrie)
	}
	if te.bidsTrie != nil {
		stateExchanges.bidsTrie = db.db.CopyTrie(te.bidsTrie)
	}
	if te.ordersTrie != nil {
		stateExchanges.ordersTrie = db.db.CopyTrie(te.ordersTrie)
	}
	for price, bidObject := range te.stateBidObjects {
		stateExchanges.stateBidObjects[price] = bidObject.deepCopy(db, te.MarkStateBidObjectDirty)
	}
	for price := range te.stateBidObjectsDirty {
		stateExchanges.stateBidObjectsDirty[price] = struct{}{}
	}
	for price, askObject := range te.stateAskObjects {
		stateExchanges.stateAskObjects[price] = askObject.deepCopy(db, te.MarkStateAskObjectDirty)
	}
	for price := range te.stateAskObjectsDirty {
		stateExchanges.stateAskObjectsDirty[price] = struct{}{}
	}
	for orderId, orderItem := range te.stateOrderObjects {
		stateExchanges.stateOrderObjects[orderId] = orderItem.deepCopy(te.MarkStateOrderObjectDirty)
	}
	for orderId := range te.stateOrderObjectsDirty {
		stateExchanges.stateOrderObjectsDirty[orderId] = struct{}{}
	}
	for price, liquidationPrice := range te.liquidationPriceStates {
		stateExchanges.liquidationPriceStates[price] = liquidationPrice.deepCopy(db, te.MarkStateLiquidationPriceDirty)
	}
	for price := range te.liquidationPriceStatesDirty {
		stateExchanges.liquidationPriceStatesDirty[price] = struct{}{}
	}
	return stateExchanges
}

// Returns the address of the contract/orderId
func (te *tradingExchanges) Hash() common.Hash {
	return te.orderBookHash
}

func (te *tradingExchanges) SetNonce(nonce uint64) {
	te.setNonce(nonce)
}

func (te *tradingExchanges) setNonce(nonce uint64) {
	te.data.Nonce = nonce
	if te.onDirty != nil {
		te.onDirty(te.Hash())
		te.onDirty = nil
	}
}

func (te *tradingExchanges) Nonce() uint64 {
	return te.data.Nonce
}

func (te *tradingExchanges) setLastPrice(price *big.Int) {
	te.data.LastPrice = price
	if te.onDirty != nil {
		te.onDirty(te.Hash())
		te.onDirty = nil
	}
}

func (te *tradingExchanges) setMediumPriceBeforeEpoch(price *big.Int) {
	te.data.MediumPriceBeforeEpoch = price
	if te.onDirty != nil {
		te.onDirty(te.Hash())
		te.onDirty = nil
	}
}

func (te *tradingExchanges) setMediumPrice(price *big.Int, quantity *big.Int) {
	te.data.MediumPrice = price
	te.data.TotalQuantity = quantity
	if te.onDirty != nil {
		te.onDirty(te.Hash())
		te.onDirty = nil
	}
}

// updateStateExchangeObject writes the given object to the trie.
func (te *tradingExchanges) removeStateOrderListAskObject(db Database, stateOrderList *stateOrderList) {
	te.setError(te.asksTrie.TryDelete(stateOrderList.price[:]))
}

// updateStateExchangeObject writes the given object to the trie.
func (te *tradingExchanges) removeStateOrderListBidObject(db Database, stateOrderList *stateOrderList) {
	te.setError(te.bidsTrie.TryDelete(stateOrderList.price[:]))
}

// Retrieve a state object given my the address. Returns nil if not found.
func (te *tradingExchanges) getStateOrderListAskObject(db Database, price common.Hash) (stateOrderList *stateOrderList) {
	// Prefer 'live' objects.
	if obj := te.stateAskObjects[price]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := te.getAsksTrie(db).TryGet(price[:])
	if len(enc) == 0 {
		te.setError(err)
		return nil
	}
	var data orderList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state order list object", "orderId", price, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newStateOrderList(te.db, Bid, te.orderBookHash, price, data, te.MarkStateAskObjectDirty)
	te.stateAskObjects[price] = obj
	return obj
}

// MarkStateAskObjectDirty adds the specified object to the dirty map to avoid costly
// state object cache iteration to find a handful of modified ones.
func (te *tradingExchanges) MarkStateAskObjectDirty(price common.Hash) {
	te.stateAskObjectsDirty[price] = struct{}{}
	if te.onDirty != nil {
		te.onDirty(te.Hash())
		te.onDirty = nil
	}
}

// createStateOrderListObject creates a new state object. If there is an existing orderId with
// the given address, it is overwritten and returned as the second return value.
func (te *tradingExchanges) createStateOrderListAskObject(db Database, price common.Hash) (newobj *stateOrderList) {
	newobj = newStateOrderList(te.db, Ask, te.orderBookHash, price, orderList{Volume: Zero}, te.MarkStateAskObjectDirty)
	te.stateAskObjects[price] = newobj
	te.stateAskObjectsDirty[price] = struct{}{}
	data, err := rlp.EncodeToBytes(newobj)
	if err != nil {
		panic(fmt.Errorf("can't encode order list object at %x: %v", price[:], err))
	}
	te.setError(te.asksTrie.TryUpdate(price[:], data))
	if te.onDirty != nil {
		te.onDirty(te.Hash())
		te.onDirty = nil
	}
	return newobj
}

// Retrieve a state object given my the address. Returns nil if not found.
func (te *tradingExchanges) getStateBidOrderListObject(db Database, price common.Hash) (stateOrderList *stateOrderList) {
	// Prefer 'live' objects.
	if obj := te.stateBidObjects[price]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := te.getBidsTrie(db).TryGet(price[:])
	if len(enc) == 0 {
		te.setError(err)
		return nil
	}
	var data orderList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state order list object", "orderId", price, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newStateOrderList(te.db, Bid, te.orderBookHash, price, data, te.MarkStateBidObjectDirty)
	te.stateBidObjects[price] = obj
	return obj
}

// MarkStateAskObjectDirty adds the specified object to the dirty map to avoid costly
// state object cache iteration to find a handful of modified ones.
func (te *tradingExchanges) MarkStateBidObjectDirty(price common.Hash) {
	te.stateBidObjectsDirty[price] = struct{}{}
	if te.onDirty != nil {
		te.onDirty(te.Hash())
		te.onDirty = nil
	}
}

// createStateOrderListObject creates a new state object. If there is an existing orderId with
// the given address, it is overwritten and returned as the second return value.
func (te *tradingExchanges) createStateBidOrderListObject(db Database, price common.Hash) (newobj *stateOrderList) {
	newobj = newStateOrderList(te.db, Bid, te.orderBookHash, price, orderList{Volume: Zero}, te.MarkStateBidObjectDirty)
	te.stateBidObjects[price] = newobj
	te.stateBidObjectsDirty[price] = struct{}{}
	data, err := rlp.EncodeToBytes(newobj)
	if err != nil {
		panic(fmt.Errorf("can't encode order list object at %x: %v", price[:], err))
	}
	te.setError(te.bidsTrie.TryUpdate(price[:], data))
	if te.onDirty != nil {
		te.onDirty(te.Hash())
		te.onDirty = nil
	}
	return newobj
}

// Retrieve a state object given my the address. Returns nil if not found.
func (te *tradingExchanges) getStateOrderObject(db Database, orderId common.Hash) (stateOrderItem *stateOrderItem) {
	// Prefer 'live' objects.
	if obj := te.stateOrderObjects[orderId]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := te.getOrdersTrie(db).TryGet(orderId[:])
	if len(enc) == 0 {
		te.setError(err)
		return nil
	}
	var data OrderItem
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state order object", "orderId", orderId, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newStateOrderItem(te.orderBookHash, orderId, data, te.MarkStateOrderObjectDirty)
	te.stateOrderObjects[orderId] = obj
	return obj
}

// MarkStateAskObjectDirty adds the specified object to the dirty map to avoid costly
// state object cache iteration to find a handful of modified ones.
func (te *tradingExchanges) MarkStateOrderObjectDirty(orderId common.Hash) {
	te.stateOrderObjectsDirty[orderId] = struct{}{}
	if te.onDirty != nil {
		te.onDirty(te.Hash())
		te.onDirty = nil
	}
}

// createStateOrderListObject creates a new state object. If there is an existing orderId with
// the given address, it is overwritten and returned as the second return value.
func (t *tradingExchanges) createStateOrderObject(db Database, orderId common.Hash, order OrderItem) (newobj *stateOrderItem) {
	newobj = newStateOrderItem(t.orderBookHash, orderId, order, t.MarkStateOrderObjectDirty)
	orderIdHash := common.BigToHash(new(big.Int).SetUint64(order.OrderID))
	t.stateOrderObjects[orderIdHash] = newobj
	t.stateOrderObjectsDirty[orderIdHash] = struct{}{}
	if t.onDirty != nil {
		t.onDirty(t.orderBookHash)
		t.onDirty = nil
	}
	return newobj
}

// updateAskTrie writes cached storage modifications into the object's storage trie.
func (t *tradingExchanges) updateOrdersTrie(db Database) Trie {
	tr := t.getOrdersTrie(db)
	for orderId, orderItem := range t.stateOrderObjects {
		if _, isDirty := t.stateOrderObjectsDirty[orderId]; isDirty {
			delete(t.stateOrderObjectsDirty, orderId)
			if orderItem.empty() {
				t.setError(tr.TryDelete(orderId[:]))
				continue
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(orderItem)
			t.setError(tr.TryUpdate(orderId[:], v))
		}
	}
	return tr
}

// CommitAskTrie the storage trie of the object to dwb.
// This updates the trie root.
func (t *tradingExchanges) updateOrdersRoot(db Database) {
	t.updateOrdersTrie(db)
	t.data.OrderRoot = t.ordersTrie.Hash()
}

// CommitAskTrie the storage trie of the object to dwb.
// This updates the trie root.
func (t *tradingExchanges) CommitOrdersTrie(db Database) error {
	t.updateOrdersTrie(db)
	if t.dbErr != nil {
		return t.dbErr
	}
	root, err := t.ordersTrie.Commit(nil)
	if err == nil {
		t.data.OrderRoot = root
	}
	return err
}

func (t *tradingExchanges) MarkStateLiquidationPriceDirty(price common.Hash) {
	t.liquidationPriceStatesDirty[price] = struct{}{}
	if t.onDirty != nil {
		t.onDirty(t.Hash())
		t.onDirty = nil
	}
}

func (t *tradingExchanges) createStateLiquidationPrice(db Database, liquidationPrice common.Hash) (newobj *liquidationPriceState) {
	newobj = newLiquidationPriceState(t.db, t.orderBookHash, liquidationPrice, orderList{Volume: Zero}, t.MarkStateLiquidationPriceDirty)
	t.liquidationPriceStates[liquidationPrice] = newobj
	t.liquidationPriceStatesDirty[liquidationPrice] = struct{}{}
	data, err := rlp.EncodeToBytes(newobj)
	if err != nil {
		panic(fmt.Errorf("can't encode liquidation price object at %x: %v", liquidationPrice[:], err))
	}
	t.setError(t.getLiquidationPriceTrie(db).TryUpdate(liquidationPrice[:], data))
	if t.onDirty != nil {
		t.onDirty(t.Hash())
		t.onDirty = nil
	}
	return newobj
}

func (t *tradingExchanges) getLiquidationPriceTrie(db Database) Trie {
	if t.liquidationPriceTrie == nil {
		var err error
		t.liquidationPriceTrie, err = db.OpenStorageTrie(t.orderBookHash, t.data.LiquidationPriceRoot)
		if err != nil {
			t.liquidationPriceTrie, _ = db.OpenStorageTrie(t.orderBookHash, types.EmptyRootHash)
			t.setError(fmt.Errorf("can't create liquidation liquidationPrice trie: %v", err))
		}
	}
	return t.liquidationPriceTrie
}

func (t *tradingExchanges) getStateLiquidationPrice(db Database, price common.Hash) (stateObject *liquidationPriceState) {
	// Prefer 'live' objects.
	if obj := t.liquidationPriceStates[price]; obj != nil {
		return obj
	}

	// Load the object from the database.
	enc, err := t.getLiquidationPriceTrie(db).TryGet(price[:])
	if len(enc) == 0 {
		t.setError(err)
		return nil
	}
	var data orderList
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state liquidation liquidationPrice", "liquidationPrice", price, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newLiquidationPriceState(t.db, t.orderBookHash, price, data, t.MarkStateLiquidationPriceDirty)
	t.liquidationPriceStates[price] = obj
	return obj
}

func (t *tradingExchanges) getLowestLiquidationPrice(db Database) (common.Hash, *liquidationPriceState) {
	trie := t.getLiquidationPriceTrie(db)
	encKey, encValue, err := trie.TryGetBestLeftKeyAndValue()
	if err != nil {
		log.Error("Failed find best liquidationPrice ask trie ", "orderbook", t.orderBookHash.Hex())
		return EmptyHash, nil
	}
	if len(encKey) == 0 || len(encValue) == 0 {
		log.Debug("Not found get best ask trie", "encKey", encKey, "encValue", encValue)
		return EmptyHash, nil
	}
	price := common.BytesToHash(encKey)
	obj := t.liquidationPriceStates[price]
	if obj == nil {
		var data orderList
		if err := rlp.DecodeBytes(encValue, &data); err != nil {
			log.Error("Failed to decode state get best ask trie", "err", err)
			return EmptyHash, nil
		}
		obj = newLiquidationPriceState(t.db, t.orderBookHash, price, data, t.MarkStateLiquidationPriceDirty)
		t.liquidationPriceStates[price] = obj
	}
	return price, obj
}

func (t *tradingExchanges) getAllLowerLiquidationPrice(db Database, limit common.Hash) map[common.Hash]*liquidationPriceState {
	trie := t.getLiquidationPriceTrie(db)
	encKeys, encValues, err := trie.TryGetAllLeftKeyAndValue(limit.Bytes())
	result := map[common.Hash]*liquidationPriceState{}
	if err != nil || len(encKeys) != len(encValues) {
		log.Error("Failed get lower liquidation price trie ", "orderbook", t.orderBookHash.Hex(), "encKeys", len(encKeys), "encValues", len(encValues))
		return result
	}
	if len(encKeys) == 0 || len(encValues) == 0 {
		log.Debug("Not found get lower liquidation price trie ", "limit", limit)
		return result
	}
	for i := range encKeys {
		price := common.BytesToHash(encKeys[i])
		obj := t.liquidationPriceStates[price]
		if obj == nil {
			var data orderList
			if err := rlp.DecodeBytes(encValues[i], &data); err != nil {
				log.Error("Failed to decode state get all lower liquidation price trie", "price", price, "encValues", encValues[i], "err", err)
				return result
			}
			obj = newLiquidationPriceState(t.db, t.orderBookHash, price, data, t.MarkStateLiquidationPriceDirty)
			t.liquidationPriceStates[price] = obj
		}
		if obj.empty() {
			continue
		}
		result[price] = obj
	}
	return result
}

func (t *tradingExchanges) getHighestLiquidationPrice(db Database) (common.Hash, *liquidationPriceState) {
	trie := t.getLiquidationPriceTrie(db)
	encKey, encValue, err := trie.TryGetBestRightKeyAndValue()
	if err != nil {
		log.Error("Failed find best liquidationPrice ask trie ", "orderbook", t.orderBookHash.Hex())
		return EmptyHash, nil
	}
	if len(encKey) == 0 || len(encValue) == 0 {
		log.Debug("Not found get best ask trie", "encKey", encKey, "encValue", encValue)
		return EmptyHash, nil
	}
	price := common.BytesToHash(encKey)
	obj := t.liquidationPriceStates[price]
	if obj == nil {
		var data orderList
		if err := rlp.DecodeBytes(encValue, &data); err != nil {
			log.Error("Failed to decode state get best ask trie", "err", err)
			return EmptyHash, nil
		}
		obj = newLiquidationPriceState(t.db, t.orderBookHash, price, data, t.MarkStateLiquidationPriceDirty)
		t.liquidationPriceStates[price] = obj
	}
	if obj.empty() {
		return EmptyHash, nil
	}
	return price, obj
}

func (t *tradingExchanges) updateLiquidationPriceTrie(db Database) Trie {
	tr := t.getLiquidationPriceTrie(db)
	for price, stateObject := range t.liquidationPriceStates {
		if _, isDirty := t.liquidationPriceStatesDirty[price]; isDirty {
			delete(t.liquidationPriceStatesDirty, price)
			if stateObject.empty() {
				t.setError(tr.TryDelete(price[:]))
				continue
			}
			err := stateObject.updateRoot(db)
			if err != nil {
				log.Warn("updateLiquidationPriceTrie updateRoot", "err", err, "price", price, "stateObject", *stateObject)
			}
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ := rlp.EncodeToBytes(stateObject)
			t.setError(tr.TryUpdate(price[:], v))
		}
	}
	return tr
}

func (t *tradingExchanges) updateLiquidationPriceRoot(db Database) {
	t.updateLiquidationPriceTrie(db)
	t.data.LiquidationPriceRoot = t.liquidationPriceTrie.Hash()
}

func (t *tradingExchanges) CommitLiquidationPriceTrie(db Database) error {
	t.updateLiquidationPriceTrie(db)
	if t.dbErr != nil {
		return t.dbErr
	}
	root, err := t.liquidationPriceTrie.Commit(func(leaf []byte, parent common.Hash) error {
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
		t.data.LiquidationPriceRoot = root
	}
	return err
}

func (t *tradingExchanges) addLendingCount(amount *big.Int) {
	t.setLendingCount(new(big.Int).Add(t.data.LendingCount, amount))
}

func (t *tradingExchanges) subLendingCount(amount *big.Int) {
	t.setLendingCount(new(big.Int).Sub(t.data.LendingCount, amount))
}

func (t *tradingExchanges) setLendingCount(volume *big.Int) {
	t.data.LendingCount = volume
	if t.onDirty != nil {
		t.onDirty(t.orderBookHash)
		t.onDirty = nil
	}
}
