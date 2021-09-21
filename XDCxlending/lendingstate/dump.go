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
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"math/big"
	"sort"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/trie"
)

type DumpOrderList struct {
	Volume *big.Int
	Orders map[*big.Int]*big.Int
}

type DumpOrderBookInfo struct {
	Nonce                 uint64
	TradeNonce            uint64
	BestInvesting         *big.Int
	BestBorrowing         *big.Int
	LowestLiquidationTime *big.Int
}

func (self *LendingStateDB) DumpInvestingTrie(orderBook common.Hash) (map[*big.Int]DumpOrderList, error) {
	exhangeObject := self.getLendingExchange(orderBook)
	if exhangeObject == nil {
		return nil, fmt.Errorf("Order book not found orderBook : %v ", orderBook.Hex())
	}
	mapResult := map[*big.Int]DumpOrderList{}
	it := trie.NewIterator(exhangeObject.getInvestingTrie(self.db).NodeIterator(nil))
	for it.Next() {
		interestHash := common.BytesToHash(it.Key)
		if common.EmptyHash(interestHash) {
			continue
		}
		interest := new(big.Int).SetBytes(interestHash.Bytes())
		if _, exist := exhangeObject.investingStates[interestHash]; exist {
			continue
		} else {
			var data itemList
			if err := rlp.DecodeBytes(it.Value, &data); err != nil {
				return nil, fmt.Errorf("Fail when decode order iist orderBook : %v ,interest :%v ", orderBook.Hex(), interest)
			}
			stateOrderList := newItemListState(orderBook, interestHash, data, nil)
			mapResult[interest] = stateOrderList.DumpItemList(self.db)
		}
	}
	for interestHash, itemList := range exhangeObject.investingStates {
		if itemList.Volume().Sign() > 0 {
			mapResult[new(big.Int).SetBytes(interestHash.Bytes())] = itemList.DumpItemList(self.db)
		}
	}
	listInterest := []*big.Int{}
	for interest := range mapResult {
		listInterest = append(listInterest, interest)
	}
	sort.Slice(listInterest, func(i, j int) bool {
		return listInterest[i].Cmp(listInterest[j]) < 0
	})
	result := map[*big.Int]DumpOrderList{}
	for _, interest := range listInterest {
		result[interest] = mapResult[interest]
	}
	return result, nil
}

func (self *LendingStateDB) DumpBorrowingTrie(orderBook common.Hash) (map[*big.Int]DumpOrderList, error) {
	exhangeObject := self.getLendingExchange(orderBook)
	if exhangeObject == nil {
		return nil, fmt.Errorf("Order book not found orderBook : %v ", orderBook.Hex())
	}
	mapResult := map[*big.Int]DumpOrderList{}
	it := trie.NewIterator(exhangeObject.getBorrowingTrie(self.db).NodeIterator(nil))
	for it.Next() {
		interestHash := common.BytesToHash(it.Key)
		if common.EmptyHash(interestHash) {
			continue
		}
		interest := new(big.Int).SetBytes(interestHash.Bytes())
		if _, exist := exhangeObject.borrowingStates[interestHash]; exist {
			continue
		} else {
			var data itemList
			if err := rlp.DecodeBytes(it.Value, &data); err != nil {
				return nil, fmt.Errorf("Fail when decode order iist orderBook : %v ,interest :%v ", orderBook.Hex(), interest)
			}
			stateOrderList := newItemListState(orderBook, interestHash, data, nil)
			mapResult[interest] = stateOrderList.DumpItemList(self.db)
		}
	}
	for interestHash, itemList := range exhangeObject.borrowingStates {
		if itemList.Volume().Sign() > 0 {
			mapResult[new(big.Int).SetBytes(interestHash.Bytes())] = itemList.DumpItemList(self.db)
		}
	}
	listInterest := []*big.Int{}
	for interest := range mapResult {
		listInterest = append(listInterest, interest)
	}
	sort.Slice(listInterest, func(i, j int) bool {
		return listInterest[i].Cmp(listInterest[j]) < 0
	})
	result := map[*big.Int]DumpOrderList{}
	for _, interest := range listInterest {
		result[interest] = mapResult[interest]
	}
	return result, nil
}

func (self *LendingStateDB) GetInvestings(orderBook common.Hash) (map[*big.Int]*big.Int, error) {
	exhangeObject := self.getLendingExchange(orderBook)
	if exhangeObject == nil {
		return nil, fmt.Errorf("Order book not found orderBook : %v ", orderBook.Hex())
	}
	mapResult := map[*big.Int]*big.Int{}
	it := trie.NewIterator(exhangeObject.getInvestingTrie(self.db).NodeIterator(nil))
	for it.Next() {
		interestHash := common.BytesToHash(it.Key)
		if common.EmptyHash(interestHash) {
			continue
		}
		interest := new(big.Int).SetBytes(interestHash.Bytes())
		if _, exist := exhangeObject.investingStates[interestHash]; exist {
			continue
		} else {
			var data itemList
			if err := rlp.DecodeBytes(it.Value, &data); err != nil {
				return nil, fmt.Errorf("Fail when decode order iist orderBook : %v ,interest :%v ", orderBook.Hex(), interest)
			}
			stateOrderList := newItemListState(orderBook, interestHash, data, nil)
			mapResult[interest] = stateOrderList.data.Volume
		}
	}
	for interestHash, itemList := range exhangeObject.investingStates {
		if itemList.Volume().Sign() > 0 {
			mapResult[new(big.Int).SetBytes(interestHash.Bytes())] = itemList.data.Volume
		}
	}
	listInterest := []*big.Int{}
	for interest := range mapResult {
		listInterest = append(listInterest, interest)
	}
	sort.Slice(listInterest, func(i, j int) bool {
		return listInterest[i].Cmp(listInterest[j]) < 0
	})
	result := map[*big.Int]*big.Int{}
	for _, interest := range listInterest {
		result[interest] = mapResult[interest]
	}
	return result, nil
}

func (self *LendingStateDB) GetBorrowings(orderBook common.Hash) (map[*big.Int]*big.Int, error) {
	exhangeObject := self.getLendingExchange(orderBook)
	if exhangeObject == nil {
		return nil, fmt.Errorf("Order book not found orderBook : %v ", orderBook.Hex())
	}
	mapResult := map[*big.Int]*big.Int{}
	it := trie.NewIterator(exhangeObject.getBorrowingTrie(self.db).NodeIterator(nil))
	for it.Next() {
		interestHash := common.BytesToHash(it.Key)
		if common.EmptyHash(interestHash) {
			continue
		}
		interest := new(big.Int).SetBytes(interestHash.Bytes())
		if _, exist := exhangeObject.borrowingStates[interestHash]; exist {
			continue
		} else {
			var data itemList
			if err := rlp.DecodeBytes(it.Value, &data); err != nil {
				return nil, fmt.Errorf("Fail when decode order iist orderBook : %v ,interest :%v ", orderBook.Hex(), interest)
			}
			stateOrderList := newItemListState(orderBook, interestHash, data, nil)
			mapResult[interest] = stateOrderList.data.Volume
		}
	}
	for interestHash, itemList := range exhangeObject.borrowingStates {
		if itemList.Volume().Sign() > 0 {
			mapResult[new(big.Int).SetBytes(interestHash.Bytes())] = itemList.data.Volume
		}
	}
	listInterest := []*big.Int{}
	for interest := range mapResult {
		listInterest = append(listInterest, interest)
	}
	sort.Slice(listInterest, func(i, j int) bool {
		return listInterest[i].Cmp(listInterest[j]) < 0
	})
	result := map[*big.Int]*big.Int{}
	for _, interest := range listInterest {
		result[interest] = mapResult[interest]
	}
	return result, nil
}

func (self *itemListState) DumpItemList(db Database) DumpOrderList {
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
	return result
}

func (self *LendingStateDB) DumpOrderBookInfo(orderBook common.Hash) (*DumpOrderBookInfo, error) {
	exhangeObject := self.getLendingExchange(orderBook)
	if exhangeObject == nil {
		return nil, fmt.Errorf("Order book not found orderBook : %v ", orderBook.Hex())
	}
	result := &DumpOrderBookInfo{}
	result.Nonce = exhangeObject.data.Nonce
	result.TradeNonce = exhangeObject.data.TradeNonce
	result.BestInvesting = new(big.Int).SetBytes(exhangeObject.getBestInvestingInterest(self.db).Bytes())
	result.BestBorrowing = new(big.Int).SetBytes(exhangeObject.getBestBorrowingInterest(self.db).Bytes())
	lowestLiquidationTime, _ := exhangeObject.getLowestLiquidationTime(self.db)
	result.LowestLiquidationTime = new(big.Int).SetBytes(lowestLiquidationTime.Bytes())
	return result, nil
}

func (self *liquidationTimeState) DumpItemList(db Database) DumpOrderList {
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
func (self *LendingStateDB) DumpLiquidationTimeTrie(orderBook common.Hash) (map[*big.Int]DumpOrderList, error) {
	exhangeObject := self.getLendingExchange(orderBook)
	if exhangeObject == nil {
		return nil, fmt.Errorf("Order book not found orderBook : %v ", orderBook.Hex())
	}
	mapResult := map[*big.Int]DumpOrderList{}
	it := trie.NewIterator(exhangeObject.getLiquidationTimeTrie(self.db).NodeIterator(nil))
	for it.Next() {
		unixTimeHash := common.BytesToHash(it.Key)
		if common.EmptyHash(unixTimeHash) {
			continue
		}
		unixTime := new(big.Int).SetBytes(unixTimeHash.Bytes())
		if _, exist := exhangeObject.liquidationTimeStates[unixTimeHash]; exist {
			continue
		} else {
			var data itemList
			if err := rlp.DecodeBytes(it.Value, &data); err != nil {
				return nil, fmt.Errorf("Fail when decode order iist orderBook : %v ,unixTime :%v ", orderBook.Hex(), unixTime)
			}
			stateOrderList := newLiquidationTimeState(orderBook, unixTimeHash, data, nil)
			mapResult[unixTime] = stateOrderList.DumpItemList(self.db)
		}
	}
	for unixTimeHash, itemList := range exhangeObject.liquidationTimeStates {
		if itemList.Volume().Sign() > 0 {
			mapResult[new(big.Int).SetBytes(unixTimeHash.Bytes())] = itemList.DumpItemList(self.db)
		}
	}
	listUnixTime := []*big.Int{}
	for unixTime := range mapResult {
		listUnixTime = append(listUnixTime, unixTime)
	}
	sort.Slice(listUnixTime, func(i, j int) bool {
		return listUnixTime[i].Cmp(listUnixTime[j]) < 0
	})
	result := map[*big.Int]DumpOrderList{}
	for _, unixTime := range listUnixTime {
		result[unixTime] = mapResult[unixTime]
	}
	return result, nil
}

func (self *LendingStateDB) DumpLendingOrderTrie(orderBook common.Hash) (map[*big.Int]LendingItem, error) {
	exhangeObject := self.getLendingExchange(orderBook)
	if exhangeObject == nil {
		return nil, fmt.Errorf("Order book not found orderBook : %v ", orderBook.Hex())
	}
	mapResult := map[*big.Int]LendingItem{}
	it := trie.NewIterator(exhangeObject.getLendingItemTrie(self.db).NodeIterator(nil))
	for it.Next() {
		orderIdHash := common.BytesToHash(it.Key)
		if common.EmptyHash(orderIdHash) {
			continue
		}
		orderId := new(big.Int).SetBytes(orderIdHash.Bytes())
		if _, exist := exhangeObject.lendingItemStates[orderIdHash]; exist {
			continue
		} else {
			var data LendingItem
			if err := rlp.DecodeBytes(it.Value, &data); err != nil {
				return nil, fmt.Errorf("Fail when decode order iist orderBook : %v ,orderId :%v ", orderBook.Hex(), orderId)
			}
			mapResult[orderId] = data
		}
	}
	for orderIdHash, lendingOrder := range exhangeObject.lendingItemStates {
		mapResult[new(big.Int).SetBytes(orderIdHash.Bytes())] = lendingOrder.data
	}
	listOrderId := []*big.Int{}
	for orderId := range mapResult {
		listOrderId = append(listOrderId, orderId)
	}
	sort.Slice(listOrderId, func(i, j int) bool {
		return listOrderId[i].Cmp(listOrderId[j]) < 0
	})
	result := map[*big.Int]LendingItem{}
	for _, orderId := range listOrderId {
		result[orderId] = mapResult[orderId]
	}
	return result, nil
}

func (self *LendingStateDB) DumpLendingTradeTrie(orderBook common.Hash) (map[*big.Int]LendingTrade, error) {
	exhangeObject := self.getLendingExchange(orderBook)
	if exhangeObject == nil {
		return nil, fmt.Errorf("Order book not found orderBook : %v ", orderBook.Hex())
	}
	mapResult := map[*big.Int]LendingTrade{}
	it := trie.NewIterator(exhangeObject.getLendingTradeTrie(self.db).NodeIterator(nil))
	for it.Next() {
		tradeIdHash := common.BytesToHash(it.Key)
		if common.EmptyHash(tradeIdHash) {
			continue
		}
		tradeId := new(big.Int).SetBytes(tradeIdHash.Bytes())
		if _, exist := exhangeObject.lendingTradeStates[tradeIdHash]; exist {
			continue
		} else {
			var data LendingTrade
			if err := rlp.DecodeBytes(it.Value, &data); err != nil {
				return nil, fmt.Errorf("Fail when decode order iist orderBook : %v ,tradeId :%v ", orderBook.Hex(), tradeId)
			}
			mapResult[tradeId] = data
		}
	}
	for tradeIdHash, lendingTrade := range exhangeObject.lendingTradeStates {
		mapResult[new(big.Int).SetBytes(tradeIdHash.Bytes())] = lendingTrade.data
	}
	listTradeId := []*big.Int{}
	for tradeId := range mapResult {
		listTradeId = append(listTradeId, tradeId)
	}
	sort.Slice(listTradeId, func(i, j int) bool {
		return listTradeId[i].Cmp(listTradeId[j]) < 0
	})
	result := map[*big.Int]LendingTrade{}
	for _, tradeId := range listTradeId {
		result[tradeId] = mapResult[tradeId]
	}
	return result, nil
}
