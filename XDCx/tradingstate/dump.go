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
	"math/big"
	"sort"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/XinFinOrg/XDPoSChain/trie"
)

type DumpOrderList struct {
	Volume *big.Int
	Orders map[*big.Int]*big.Int
}
type DumpLendingBook struct {
	Volume       *big.Int
	LendingBooks map[common.Hash]DumpOrderList
}

type DumpOrderBookInfo struct {
	LastPrice              *big.Int
	LendingCount           *big.Int
	MediumPrice            *big.Int
	MediumPriceBeforeEpoch *big.Int
	Nonce                  uint64
	TotalQuantity          *big.Int
	BestAsk                *big.Int
	BestBid                *big.Int
	LowestLiquidationPrice *big.Int
}

func (self *TradingStateDB) DumpAskTrie(orderBook common.Hash) (map[*big.Int]DumpOrderList, error) {
	exhangeObject := self.getStateExchangeObject(orderBook)
	if exhangeObject == nil {
		return nil, fmt.Errorf("Order book not found orderBook : %v ", orderBook.Hex())
	}
	mapResult := map[*big.Int]DumpOrderList{}
	it := trie.NewIterator(exhangeObject.getAsksTrie(self.db).NodeIterator(nil))
	for it.Next() {
		priceHash := common.BytesToHash(it.Key)
		if common.EmptyHash(priceHash) {
			continue
		}
		price := new(big.Int).SetBytes(priceHash.Bytes())
		if _, exist := exhangeObject.stateAskObjects[priceHash]; exist {
			continue
		} else {
			var data orderList
			if err := rlp.DecodeBytes(it.Value, &data); err != nil {
				return nil, fmt.Errorf("Fail when decode order iist orderBook : %v ,price :%v ", orderBook.Hex(), price)
			}
			stateOrderList := newStateOrderList(self, Ask, orderBook, priceHash, data, nil)
			mapResult[price] = stateOrderList.DumpOrderList(self.db)
		}
	}
	for priceHash, stateOrderList := range exhangeObject.stateAskObjects {
		if stateOrderList.Volume().Sign() > 0 {
			mapResult[new(big.Int).SetBytes(priceHash.Bytes())] = stateOrderList.DumpOrderList(self.db)
		}
	}
	listPrice := []*big.Int{}
	for price := range mapResult {
		listPrice = append(listPrice, price)
	}
	sort.Slice(listPrice, func(i, j int) bool {
		return listPrice[i].Cmp(listPrice[j]) < 0
	})
	result := map[*big.Int]DumpOrderList{}
	for _, price := range listPrice {
		result[price] = mapResult[price]
	}
	return result, nil
}

func (self *TradingStateDB) DumpBidTrie(orderBook common.Hash) (map[*big.Int]DumpOrderList, error) {
	exhangeObject := self.getStateExchangeObject(orderBook)
	if exhangeObject == nil {
		return nil, fmt.Errorf("Order book not found orderBook : %v ", orderBook.Hex())
	}
	mapResult := map[*big.Int]DumpOrderList{}
	it := trie.NewIterator(exhangeObject.getBidsTrie(self.db).NodeIterator(nil))
	for it.Next() {
		priceHash := common.BytesToHash(it.Key)
		if common.EmptyHash(priceHash) {
			continue
		}
		price := new(big.Int).SetBytes(priceHash.Bytes())
		if _, exist := exhangeObject.stateBidObjects[priceHash]; exist {
			continue
		} else {
			var data orderList
			if err := rlp.DecodeBytes(it.Value, &data); err != nil {
				return nil, fmt.Errorf("Fail when decode order iist orderBook : %v ,price :%v ", orderBook.Hex(), price)
			}
			stateOrderList := newStateOrderList(self, Bid, orderBook, priceHash, data, nil)
			mapResult[price] = stateOrderList.DumpOrderList(self.db)
		}
	}
	for priceHash, stateOrderList := range exhangeObject.stateBidObjects {
		if stateOrderList.Volume().Sign() > 0 {
			mapResult[new(big.Int).SetBytes(priceHash.Bytes())] = stateOrderList.DumpOrderList(self.db)
		}
	}
	listPrice := []*big.Int{}
	for price := range mapResult {
		listPrice = append(listPrice, price)
	}
	sort.Slice(listPrice, func(i, j int) bool {
		return listPrice[i].Cmp(listPrice[j]) < 0
	})
	result := map[*big.Int]DumpOrderList{}
	for _, price := range listPrice {
		result[price] = mapResult[price]
	}
	return mapResult, nil
}

func (self *TradingStateDB) GetBids(orderBook common.Hash) (map[*big.Int]*big.Int, error) {
	exhangeObject := self.getStateExchangeObject(orderBook)
	if exhangeObject == nil {
		return nil, fmt.Errorf("Order book not found orderBook : %v ", orderBook.Hex())
	}
	mapResult := map[*big.Int]*big.Int{}
	it := trie.NewIterator(exhangeObject.getBidsTrie(self.db).NodeIterator(nil))
	for it.Next() {
		priceHash := common.BytesToHash(it.Key)
		if common.EmptyHash(priceHash) {
			continue
		}
		price := new(big.Int).SetBytes(priceHash.Bytes())
		if _, exist := exhangeObject.stateBidObjects[priceHash]; exist {
			continue
		} else {
			var data orderList
			if err := rlp.DecodeBytes(it.Value, &data); err != nil {
				return nil, fmt.Errorf("Fail when decode order iist orderBook : %v ,price :%v ", orderBook.Hex(), price)
			}
			stateOrderList := newStateOrderList(self, Bid, orderBook, priceHash, data, nil)
			mapResult[price] = stateOrderList.data.Volume
		}
	}
	for priceHash, stateOrderList := range exhangeObject.stateBidObjects {
		if stateOrderList.Volume().Sign() > 0 {
			mapResult[new(big.Int).SetBytes(priceHash.Bytes())] = stateOrderList.data.Volume
		}
	}
	listPrice := []*big.Int{}
	for price := range mapResult {
		listPrice = append(listPrice, price)
	}
	sort.Slice(listPrice, func(i, j int) bool {
		return listPrice[i].Cmp(listPrice[j]) < 0
	})
	result := map[*big.Int]*big.Int{}
	for _, price := range listPrice {
		result[price] = mapResult[price]
	}
	return mapResult, nil
}

func (self *TradingStateDB) GetAsks(orderBook common.Hash) (map[*big.Int]*big.Int, error) {
	exhangeObject := self.getStateExchangeObject(orderBook)
	if exhangeObject == nil {
		return nil, fmt.Errorf("Order book not found orderBook : %v ", orderBook.Hex())
	}
	mapResult := map[*big.Int]*big.Int{}
	it := trie.NewIterator(exhangeObject.getAsksTrie(self.db).NodeIterator(nil))
	for it.Next() {
		priceHash := common.BytesToHash(it.Key)
		if common.EmptyHash(priceHash) {
			continue
		}
		price := new(big.Int).SetBytes(priceHash.Bytes())
		if _, exist := exhangeObject.stateAskObjects[priceHash]; exist {
			continue
		} else {
			var data orderList
			if err := rlp.DecodeBytes(it.Value, &data); err != nil {
				return nil, fmt.Errorf("Fail when decode order iist orderBook : %v ,price :%v ", orderBook.Hex(), price)
			}
			stateOrderList := newStateOrderList(self, Ask, orderBook, priceHash, data, nil)
			mapResult[price] = stateOrderList.data.Volume
		}
	}
	for priceHash, stateOrderList := range exhangeObject.stateAskObjects {
		if stateOrderList.Volume().Sign() > 0 {
			mapResult[new(big.Int).SetBytes(priceHash.Bytes())] = stateOrderList.data.Volume
		}
	}
	listPrice := []*big.Int{}
	for price := range mapResult {
		listPrice = append(listPrice, price)
	}
	sort.Slice(listPrice, func(i, j int) bool {
		return listPrice[i].Cmp(listPrice[j]) < 0
	})
	result := map[*big.Int]*big.Int{}
	for _, price := range listPrice {
		result[price] = mapResult[price]
	}
	return result, nil
}
func (self *stateOrderList) DumpOrderList(db Database) DumpOrderList {
	mapResult := DumpOrderList{Volume: self.Volume(), Orders: map[*big.Int]*big.Int{}}
	orderListIt := trie.NewIterator(self.getTrie(db).NodeIterator(nil))
	for orderListIt.Next() {
		keyHash := common.BytesToHash(orderListIt.Key)
		if common.EmptyHash(keyHash) {
			continue
		}
		if _, exist := self.cachedStorage[keyHash]; exist {
			continue
		} else {
			_, content, _, _ := rlp.Split(orderListIt.Value)
			mapResult.Orders[new(big.Int).SetBytes(keyHash.Bytes())] = new(big.Int).SetBytes(content)
		}
	}
	for key, value := range self.cachedStorage {
		if !common.EmptyHash(value) {
			mapResult.Orders[new(big.Int).SetBytes(key.Bytes())] = new(big.Int).SetBytes(value.Bytes())
		}
	}
	listIds := []*big.Int{}
	for id := range mapResult.Orders {
		listIds = append(listIds, id)
	}
	sort.Slice(listIds, func(i, j int) bool {
		return listIds[i].Cmp(listIds[j]) < 0
	})
	result := DumpOrderList{Volume: self.Volume(), Orders: map[*big.Int]*big.Int{}}
	for _, id := range listIds {
		result.Orders[id] = mapResult.Orders[id]
	}
	return mapResult
}

func (self *TradingStateDB) DumpOrderBookInfo(orderBook common.Hash) (*DumpOrderBookInfo, error) {
	exhangeObject := self.getStateExchangeObject(orderBook)
	if exhangeObject == nil {
		return nil, fmt.Errorf("Order book not found orderBook : %v ", orderBook.Hex())
	}
	result := &DumpOrderBookInfo{}
	result.LastPrice = exhangeObject.data.LastPrice
	result.LendingCount = exhangeObject.data.LendingCount
	result.MediumPrice = exhangeObject.data.MediumPrice
	result.MediumPriceBeforeEpoch = exhangeObject.data.MediumPriceBeforeEpoch
	result.Nonce = exhangeObject.data.Nonce
	result.TotalQuantity = exhangeObject.data.TotalQuantity
	result.BestAsk = new(big.Int).SetBytes(exhangeObject.getBestPriceAsksTrie(self.db).Bytes())
	result.BestBid = new(big.Int).SetBytes(exhangeObject.getBestBidsTrie(self.db).Bytes())
	lowestPrice, _ := exhangeObject.getLowestLiquidationPrice(self.db)
	result.LowestLiquidationPrice = new(big.Int).SetBytes(lowestPrice.Bytes())
	return result, nil
}

func (self *stateLendingBook) DumpOrderList(db Database) DumpOrderList {
	mapResult := DumpOrderList{Volume: self.Volume(), Orders: map[*big.Int]*big.Int{}}
	orderListIt := trie.NewIterator(self.getTrie(db).NodeIterator(nil))
	for orderListIt.Next() {
		keyHash := common.BytesToHash(orderListIt.Key)
		if common.EmptyHash(keyHash) {
			continue
		}
		if _, exist := self.cachedStorage[keyHash]; exist {
			continue
		} else {
			_, content, _, _ := rlp.Split(orderListIt.Value)
			mapResult.Orders[new(big.Int).SetBytes(keyHash.Bytes())] = new(big.Int).SetBytes(content)
		}
	}
	for key, value := range self.cachedStorage {
		if !common.EmptyHash(value) {
			mapResult.Orders[new(big.Int).SetBytes(key.Bytes())] = new(big.Int).SetBytes(value.Bytes())
		}
	}
	listIds := []*big.Int{}
	for id := range mapResult.Orders {
		listIds = append(listIds, id)
	}
	sort.Slice(listIds, func(i, j int) bool {
		return listIds[i].Cmp(listIds[j]) < 0
	})
	result := DumpOrderList{Volume: self.Volume(), Orders: map[*big.Int]*big.Int{}}
	for _, id := range listIds {
		result.Orders[id] = mapResult.Orders[id]
	}
	return mapResult
}

func (self *liquidationPriceState) DumpLendingBook(db Database) (DumpLendingBook, error) {
	result := DumpLendingBook{Volume: self.Volume(), LendingBooks: map[common.Hash]DumpOrderList{}}
	it := trie.NewIterator(self.getTrie(db).NodeIterator(nil))
	for it.Next() {
		lendingBook := common.BytesToHash(it.Key)
		if common.EmptyHash(lendingBook) {
			continue
		}
		if _, exist := self.stateLendingBooks[lendingBook]; exist {
			continue
		} else {
			var data orderList
			if err := rlp.DecodeBytes(it.Value, &data); err != nil {
				return result, fmt.Errorf("Failed to decode state lending book orderbook : %s ,liquidation price :%s , lendingBook : %s ,err : %v", self.orderBook, self.liquidationPrice, lendingBook, err)
			}
			stateLendingBook := newStateLendingBook(self.orderBook, self.liquidationPrice, lendingBook, data, nil)
			result.LendingBooks[lendingBook] = stateLendingBook.DumpOrderList(db)
		}
	}
	for lendingBook, stateLendingBook := range self.stateLendingBooks {
		if !common.EmptyHash(lendingBook) {
			result.LendingBooks[lendingBook] = stateLendingBook.DumpOrderList(db)
		}
	}
	return result, nil
}

func (self *TradingStateDB) DumpLiquidationPriceTrie(orderBook common.Hash) (map[*big.Int]DumpLendingBook, error) {
	exhangeObject := self.getStateExchangeObject(orderBook)
	if exhangeObject == nil {
		return nil, fmt.Errorf("Order book not found orderBook : %v ", orderBook.Hex())
	}
	mapResult := map[*big.Int]DumpLendingBook{}
	it := trie.NewIterator(exhangeObject.getLiquidationPriceTrie(self.db).NodeIterator(nil))
	for it.Next() {
		priceHash := common.BytesToHash(it.Key)
		if common.EmptyHash(priceHash) {
			continue
		}
		price := new(big.Int).SetBytes(priceHash.Bytes())
		if _, exist := exhangeObject.liquidationPriceStates[priceHash]; exist {
			continue
		} else {
			var data orderList
			if err := rlp.DecodeBytes(it.Value, &data); err != nil {
				return nil, fmt.Errorf("Fail when decode order iist orderBook : %v ,price :%v ", orderBook.Hex(), price)
			}
			liquidationPriceState := newLiquidationPriceState(self, orderBook, priceHash, data, nil)
			dumpLendingBook, err := liquidationPriceState.DumpLendingBook(self.db)
			if err != nil {
				return nil, err
			}
			mapResult[price] = dumpLendingBook
		}
	}
	for priceHash, liquidationPriceState := range exhangeObject.liquidationPriceStates {
		if liquidationPriceState.Volume().Sign() > 0 {
			dumpLendingBook, err := liquidationPriceState.DumpLendingBook(self.db)
			if err != nil {
				return nil, err
			}
			mapResult[new(big.Int).SetBytes(priceHash.Bytes())] = dumpLendingBook
		}
	}
	listPrice := []*big.Int{}
	for price := range mapResult {
		listPrice = append(listPrice, price)
	}
	sort.Slice(listPrice, func(i, j int) bool {
		return listPrice[i].Cmp(listPrice[j]) < 0
	})
	result := map[*big.Int]DumpLendingBook{}
	for _, price := range listPrice {
		result[price] = mapResult[price]
	}
	return mapResult, nil
}
