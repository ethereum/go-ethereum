// Copyright 2016 The go-ethereum Authors
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
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"math/big"
	"testing"
)

func TestEchangeStates(t *testing.T) {
	t.SkipNow()
	orderBook := common.StringToHash("BTC/XDC")
	numberOrder := 20000
	orderItems := []LendingItem{}
	relayers := []common.Hash{}
	for i := 0; i < numberOrder; i++ {
		relayers = append(relayers, common.BigToHash(big.NewInt(int64(i))))
		id := new(big.Int).SetUint64(uint64(i) + 1)
		orderItems = append(orderItems, LendingItem{LendingId: id.Uint64(), Quantity: big.NewInt(int64(2*i + 1)), Interest: big.NewInt(int64(2*i + 1)), Side: Investing, Signature: &Signature{V: 1, R: common.HexToHash("111111"), S: common.HexToHash("222222222222")}})
		orderItems = append(orderItems, LendingItem{LendingId: id.Uint64(), Quantity: big.NewInt(int64(2*i + 1)), Interest: big.NewInt(int64(2*i + 1)), Side: Borrowing, Signature: &Signature{V: 1, R: common.HexToHash("3333333333"), S: common.HexToHash("22222222222222222")}})
	}
	// Create an empty statedb database
	db := rawdb.NewMemoryDatabase()
	stateCache := NewDatabase(db)
	statedb, _ := New(common.Hash{}, stateCache)

	// Update it with some lenddinges
	for i := 0; i < numberOrder; i++ {
		statedb.SetNonce(relayers[i], uint64(1))
	}
	mapPriceSell := map[uint64]uint64{}
	mapPriceBuy := map[uint64]uint64{}

	for i := 0; i < len(orderItems); i++ {
		amount := orderItems[i].Quantity.Uint64()
		orderIdHash := common.BigToHash(new(big.Int).SetUint64(orderItems[i].LendingId))
		statedb.InsertLendingItem(orderBook, orderIdHash, orderItems[i])

		switch orderItems[i].Side {
		case Investing:
			old := mapPriceSell[amount]
			mapPriceSell[amount] = old + amount
		case Borrowing:
			old := mapPriceBuy[amount]
			mapPriceBuy[amount] = old + amount
		default:
		}
		statedb.InsertLiquidationTime(orderBook, big.NewInt(int64(i)), uint64(i))
		order := LendingTrade{TradeId: uint64(i), Amount: big.NewInt(int64(i))}
		statedb.InsertTradingItem(orderBook, order.TradeId, order)
		root := statedb.IntermediateRoot()
		size, _ := stateCache.TrieDB().Size()
		fmt.Println(i, "size", size)
		statedb.Commit()
		size, _ = stateCache.TrieDB().Size()
		fmt.Println(i, "size", size)
		statedb, _ = New(root, stateCache)
	}
	statedb.InsertLiquidationTime(orderBook, big.NewInt(1), 1)
	order := LendingTrade{TradeId: 1, Amount: big.NewInt(2)}
	statedb.InsertTradingItem(orderBook, 1, order)
	root := statedb.IntermediateRoot()
	size, _ := stateCache.TrieDB().Size()
	fmt.Println("size", size)
	statedb.Commit()
	size, _ = stateCache.TrieDB().Size()
	fmt.Println("size", size)
	err := stateCache.TrieDB().Commit(root, false)
	if err != nil {
		t.Errorf("Error when commit into database: %v", err)
	}
	stateCache.TrieDB().Reference(root, common.Hash{})
	statedb, err = New(root, stateCache)
	if err != nil {
		t.Fatalf("Error when get trie in database: %s , err: %v", root.Hex(), err)
	}
	_, liquidationData := statedb.GetLowestLiquidationTime(orderBook, big.NewInt(2))
	if len(liquidationData) == 0 {
		t.Fatalf("Error when get liquidation data save in database: got : %s  ", liquidationData)
	}
	for i := 0; i < numberOrder; i++ {
		nonce := statedb.GetNonce(relayers[i])
		if nonce != uint64(1) {
			t.Fatalf("Error when get nonce save in database: got : %d , wanted : %d ", nonce, i)
		}
	}

	for i := 0; i < len(orderItems); i++ {
		amount := statedb.GetLendingOrder(orderBook, common.BigToHash(new(big.Int).SetUint64(orderItems[i].LendingId))).Quantity
		if orderItems[i].Quantity.Cmp(amount) != 0 {
			t.Fatalf("Error when get amount save in database: tradeId %d , lendingType %s,got : %d , wanted : %d ", orderItems[i].LendingId, orderItems[i].Side, amount.Uint64(), orderItems[i].Quantity.Uint64())
		}
	}
	fmt.Println(statedb.GetLendingTrade(orderBook, common.BigToHash(big.NewInt(1))).Amount)
	db.Close()
}

func TestRevertStates(t *testing.T) {
	orderBook := common.StringToHash("BTC/XDC")
	numberOrder := 20
	orderItems := []LendingItem{}
	relayers := []common.Hash{}
	for i := 0; i < numberOrder; i++ {
		relayers = append(relayers, common.BigToHash(big.NewInt(int64(i))))
		id := new(big.Int).SetUint64(uint64(i) + 1)
		orderItems = append(orderItems, LendingItem{LendingId: id.Uint64(), Quantity: big.NewInt(int64(2*i + 1)), Interest: big.NewInt(int64(2*i + 1)), Side: Investing, Signature: &Signature{V: 1, R: common.HexToHash("111111"), S: common.HexToHash("222222222222")}})
		orderItems = append(orderItems, LendingItem{LendingId: id.Uint64(), Quantity: big.NewInt(int64(2*i + 1)), Interest: big.NewInt(int64(2*i + 1)), Side: Borrowing, Signature: &Signature{V: 1, R: common.HexToHash("3333333333"), S: common.HexToHash("22222222222222222")}})
	}
	// Create an empty statedb database
	db := rawdb.NewMemoryDatabase()
	stateCache := NewDatabase(db)
	statedb, _ := New(common.Hash{}, stateCache)

	// Update it with some lenddinges
	for i := 0; i < numberOrder; i++ {
		statedb.SetNonce(relayers[i], uint64(1))
	}
	mapPriceSell := map[uint64]uint64{}
	mapPriceBuy := map[uint64]uint64{}

	for i := 0; i < len(orderItems); i++ {
		amount := orderItems[i].Quantity.Uint64()
		orderIdHash := common.BigToHash(new(big.Int).SetUint64(orderItems[i].LendingId))
		statedb.InsertLendingItem(orderBook, orderIdHash, orderItems[i])

		switch orderItems[i].Side {
		case Investing:
			old := mapPriceSell[amount]
			mapPriceSell[amount] = old + amount
		case Borrowing:
			old := mapPriceBuy[amount]
			mapPriceBuy[amount] = old + amount
		default:
		}

	}
	statedb.InsertLiquidationTime(orderBook, big.NewInt(1), 1)
	root := statedb.IntermediateRoot()
	statedb.Commit()
	//err := stateCache.TrieDB().Commit(root, false)
	//if err != nil {
	//	t.Errorf("Error when commit into database: %v", err)
	//}
	stateCache.TrieDB().Reference(root, common.Hash{})
	statedb, err := New(root, stateCache)
	if err != nil {
		t.Fatalf("Error when get trie in database: %s , err: %v", root.Hex(), err)
	}

	orderIdHash := common.BigToHash(new(big.Int).SetUint64(orderItems[0].LendingId))
	// set nonce
	wantedNonce := statedb.GetNonce(relayers[1])
	snap := statedb.Snapshot()
	statedb.SetNonce(relayers[1], 0)
	statedb.RevertToSnapshot(snap)
	gotNonce := statedb.GetNonce(relayers[1])
	if wantedNonce != gotNonce {
		t.Fatalf(" err get nonce addr: %v after try revert snap shot , got : %d ,want : %d", relayers[1].Hex(), gotNonce, wantedNonce)
	}

	// cancel order
	wantedOrder := statedb.GetLendingOrder(orderBook, orderIdHash)
	snap = statedb.Snapshot()
	statedb.CancelLendingOrder(orderBook, &wantedOrder)
	statedb.RevertToSnapshot(snap)
	gotOrder := statedb.GetLendingOrder(orderBook, orderIdHash)
	if gotOrder.Quantity.Cmp(wantedOrder.Quantity) != 0 {
		t.Fatalf(" err cancel order info : %v after try revert snap shot , got : %v ,want : %v", orderIdHash.Hex(), gotOrder, wantedOrder)
	}

	// insert order
	i := 2*numberOrder + 1
	id := new(big.Int).SetUint64(uint64(i) + 1)
	testOrder := LendingItem{LendingId: id.Uint64(), Quantity: big.NewInt(int64(2*i + 1)), Interest: big.NewInt(int64(2*i + 1)), Side: Investing, Signature: &Signature{V: 1, R: common.HexToHash("111111"), S: common.HexToHash("222222222222")}}
	orderIdHash = common.BigToHash(new(big.Int).SetUint64(testOrder.LendingId))
	snap = statedb.Snapshot()
	statedb.InsertLendingItem(orderBook, orderIdHash, testOrder)
	statedb.RevertToSnapshot(snap)
	gotOrder = statedb.GetLendingOrder(orderBook, orderIdHash)
	if gotOrder.Quantity.Cmp(EmptyLendingOrder.Quantity) != 0 {
		t.Fatalf(" err insert order info : %v after try revert snap shot , got : %v ,want Empty Order", orderIdHash.Hex(), gotOrder)
	}

	// insert trade order
	i = 2*numberOrder + 1
	id = new(big.Int).SetUint64(uint64(i) + 1)
	order := LendingTrade{TradeId: id.Uint64(), Amount: big.NewInt(int64(2*i + 1))}
	orderIdHash = common.BigToHash(new(big.Int).SetUint64(order.TradeId))
	snap = statedb.Snapshot()
	statedb.InsertTradingItem(orderBook, order.TradeId, order)
	statedb.RevertToSnapshot(snap)
	gotLendingTrade := statedb.GetLendingTrade(orderBook, orderIdHash)
	if gotLendingTrade.Amount.Cmp(EmptyLendingTrade.Amount) != 0 {
		t.Fatalf(" err insert lending trade : %s after try revert snap shot , got : %v ,want Empty Order", orderIdHash.Hex(), gotLendingTrade.Amount)
	}

	// insert trade order
	time, data := statedb.GetLowestLiquidationTime(orderBook, big.NewInt(1))
	fmt.Println(time, data)
	statedb.RemoveLiquidationTime(orderBook, 1, 1)
	time, data = statedb.GetLowestLiquidationTime(orderBook, big.NewInt(1))
	fmt.Println(time, data)
	// change key
	db.Close()
}

func TestDumpStates(t *testing.T) {
	orderBook := common.StringToHash("BTC/XDC")
	numberOrder := 20
	orderItems := []LendingItem{}
	relayers := []common.Hash{}
	for i := 0; i < numberOrder; i++ {
		relayers = append(relayers, common.BigToHash(big.NewInt(int64(i))))
		id := new(big.Int).SetUint64(uint64(i) + 1)
		orderItems = append(orderItems, LendingItem{LendingId: id.Uint64(), Quantity: big.NewInt(int64(2*i + 1)), Interest: big.NewInt(1), Side: Investing, Signature: &Signature{V: 1, R: common.HexToHash("111111"), S: common.HexToHash("222222222222")}})
		orderItems = append(orderItems, LendingItem{LendingId: id.Uint64(), Quantity: big.NewInt(int64(2*i + 1)), Interest: big.NewInt(1), Side: Borrowing, Signature: &Signature{V: 1, R: common.HexToHash("3333333333"), S: common.HexToHash("22222222222222222")}})
	}
	// Create an empty statedb database
	db := rawdb.NewMemoryDatabase()
	stateCache := NewDatabase(db)
	statedb, _ := New(common.Hash{}, stateCache)
	for i := 0; i < len(orderItems); i++ {
		orderIdHash := common.BigToHash(new(big.Int).SetUint64(orderItems[i].LendingId))
		statedb.InsertLendingItem(orderBook, orderIdHash, orderItems[i])
	}
	statedb.InsertLiquidationTime(orderBook, big.NewInt(1), 1)
	order := LendingTrade{TradeId: 1, Amount: big.NewInt(2)}
	statedb.InsertTradingItem(orderBook, 1, order)
	root := statedb.IntermediateRoot()
	statedb.Commit()
	//err := stateCache.TrieDB().Commit(root, false)
	//if err != nil {
	//	t.Errorf("Error when commit into database: %v", err)
	//}
	stateCache.TrieDB().Reference(root, common.Hash{})
	statedb, err := New(root, stateCache)
	if err != nil {
		t.Fatalf("Error when get trie in database: %s , err: %v", root.Hex(), err)
	}

	fmt.Println(statedb.DumpBorrowingTrie(orderBook))
	db.Close()
}
