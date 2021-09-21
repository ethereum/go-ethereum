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

package tradingstate

import (
	"fmt"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/math"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"math/big"
	"testing"
)

func TestEchangeStates(t *testing.T) {
	t.SkipNow()
	orderBook := common.StringToHash("BTC/XDC")
	price := big.NewInt(10000)
	numberOrder := 20
	orderItems := []OrderItem{}
	relayers := []common.Hash{}
	for i := 0; i < numberOrder; i++ {
		relayers = append(relayers, common.BigToHash(big.NewInt(int64(i))))
		id := new(big.Int).SetUint64(uint64(i) + 1)
		orderItems = append(orderItems, OrderItem{OrderID: id.Uint64(), Quantity: big.NewInt(int64(2*i + 1)), Price: big.NewInt(int64(2*i + 1)), Side: Ask, Signature: &Signature{V: 1, R: common.HexToHash("111111"), S: common.HexToHash("222222222222")}})
		orderItems = append(orderItems, OrderItem{OrderID: id.Uint64(), Quantity: big.NewInt(int64(2*i + 1)), Price: big.NewInt(int64(2*i + 1)), Side: Bid, Signature: &Signature{V: 1, R: common.HexToHash("3333333333"), S: common.HexToHash("22222222222222222")}})
	}
	// Create an empty statedb database
	db := rawdb.NewMemoryDatabase()
	stateCache := NewDatabase(db)
	statedb, _ := New(common.Hash{}, stateCache)

	// Update it with some exchanges
	for i := 0; i < numberOrder; i++ {
		statedb.SetNonce(relayers[i], uint64(1))
	}
	mapPriceSell := map[uint64]uint64{}
	mapPriceBuy := map[uint64]uint64{}

	for i := 0; i < len(orderItems); i++ {
		amount := orderItems[i].Quantity.Uint64()
		orderIdHash := common.BigToHash(new(big.Int).SetUint64(orderItems[i].OrderID))
		statedb.InsertOrderItem(orderBook, orderIdHash, orderItems[i])

		switch orderItems[i].Side {
		case Ask:
			old := mapPriceSell[amount]
			mapPriceSell[amount] = old + amount
		case Bid:
			old := mapPriceBuy[amount]
			mapPriceBuy[amount] = old + amount
		default:
		}

	}
	statedb.SetLastPrice(orderBook, price)
	statedb.InsertLiquidationPrice(orderBook, big.NewInt(1), orderBook, 2)
	statedb.InsertLiquidationPrice(orderBook, big.NewInt(2), orderBook, 3)
	statedb.InsertLiquidationPrice(orderBook, big.NewInt(3), orderBook, 1)
	root := statedb.IntermediateRoot()
	statedb.Commit()
	err := stateCache.TrieDB().Commit(root, false)
	if err != nil {
		t.Errorf("Error when commit into database: %v", err)
	}
	stateCache.TrieDB().Reference(root, common.Hash{})
	statedb, err = New(root, stateCache)
	if err != nil {
		t.Fatalf("Error when get trie in database: %s , err: %v", root.Hex(), err)
	}
	liquidationPrice, liquidationData := statedb.GetHighestLiquidationPriceData(orderBook, big.NewInt(1))
	if len(liquidationData) == 0 {
		t.Fatalf("Error when get liquidation data save in database: got : %d , %s  ", liquidationPrice, liquidationData)
	}
	fmt.Println("liquidationPrice", liquidationPrice)
	for lendingBook, tradingIds := range liquidationData {
		for _, tradingIdHash := range tradingIds {
			fmt.Println("lendingBook", lendingBook.Hex(), "tradingIdHash", tradingIdHash.Hex())
			err := statedb.RemoveLiquidationPrice(orderBook, liquidationPrice, lendingBook, new(big.Int).SetBytes(tradingIdHash.Bytes()).Uint64())
			if err != nil {
				t.Fatalf("Error when remove liquidation price  in database: %s , err: %v", root.Hex(), err)
			}
		}
	}
	statedb.RemoveLiquidationPrice(orderBook, big.NewInt(2), orderBook, 2)
	mapData := statedb.GetAllLowerLiquidationPriceData(orderBook, big.NewInt(2))
	for price, lendingBooks := range mapData {
		for lendingBook, tradeIds := range lendingBooks {
			for _, tradeId := range tradeIds {
				fmt.Println("price", price, "lendingBook", lendingBook.Hex(), "tradeId", tradeId.Hex())
			}
		}
	}
	fmt.Println("mapData", mapData)
	liquidationPrice, liquidationData = statedb.GetHighestLiquidationPriceData(orderBook, big.NewInt(2))
	for lendingBook, tradingIds := range liquidationData {
		for _, tradingIdHash := range tradingIds {
			fmt.Println("liquidationPrice", liquidationPrice, "lendingBook", lendingBook.Hex(), "tradingIdHash", tradingIdHash.Hex())
		}
	}
	gotPrice := statedb.GetLastPrice(orderBook)
	if gotPrice.Cmp(price) != 0 {
		t.Fatalf("Error when get price save in database: got : %d , wanted : %d ", gotPrice, price)
	}
	fmt.Println(gotPrice)
	for i := 0; i < numberOrder; i++ {
		nonce := statedb.GetNonce(relayers[i])
		if nonce != uint64(1) {
			t.Fatalf("Error when get nonce save in database: got : %d , wanted : %d ", nonce, i)
		}
	}

	minSell := uint64(math.MaxUint64)
	for price, amount := range mapPriceSell {
		data := statedb.GetVolume(orderBook, new(big.Int).SetUint64(price), Ask)
		if data.Uint64() != amount {
			t.Fatalf("Error when get volume save in database: price  %d ,got : %d , wanted : %d ", price, data.Uint64(), amount)
		}
		if price < minSell {
			minSell = price
		}
	}
	maxBuy := uint64(0)
	for price, amount := range mapPriceBuy {
		data := statedb.GetVolume(orderBook, new(big.Int).SetUint64(price), Bid)
		if data.Uint64() != amount {
			t.Fatalf("Error when get volume save in database: price  %d ,got : %d , wanted : %d ", price, data.Uint64(), amount)
		}
		if price > maxBuy {
			maxBuy = price
		}
	}

	for i := 0; i < len(orderItems); i++ {
		amount := statedb.GetOrder(orderBook, common.BigToHash(new(big.Int).SetUint64(orderItems[i].OrderID))).Quantity
		if orderItems[i].Quantity.Cmp(amount) != 0 {
			t.Fatalf("Error when get amount save in database: orderId %d , orderType %s,got : %d , wanted : %d ", orderItems[i].OrderID, orderItems[i].Side, amount.Uint64(), orderItems[i].Quantity.Uint64())
		}
	}
	maxPrice, volumeMax := statedb.GetBestBidPrice(orderBook)
	minPrice, volumeMin := statedb.GetBestAskPrice(orderBook)
	fmt.Println("price", minPrice, volumeMin, maxPrice, volumeMax)

	db.Close()
}

func TestRevertStates(t *testing.T) {
	orderBook := common.StringToHash("BTC/XDC")
	numberOrder := 20
	orderItems := []OrderItem{}
	relayers := []common.Hash{}
	for i := 0; i < numberOrder; i++ {
		relayers = append(relayers, common.BigToHash(big.NewInt(int64(i))))
		id := new(big.Int).SetUint64(uint64(i) + 1)
		orderItems = append(orderItems, OrderItem{OrderID: id.Uint64(), Quantity: big.NewInt(int64(2*i + 1)), Price: big.NewInt(int64(2*i + 1)), Side: Ask, Signature: &Signature{V: 1, R: common.HexToHash("111111"), S: common.HexToHash("222222222222")}})
		orderItems = append(orderItems, OrderItem{OrderID: id.Uint64(), Quantity: big.NewInt(int64(2*i + 1)), Price: big.NewInt(int64(2*i + 1)), Side: Bid, Signature: &Signature{V: 1, R: common.HexToHash("3333333333"), S: common.HexToHash("22222222222222222")}})
	}
	// Create an empty statedb database
	db := rawdb.NewMemoryDatabase()
	stateCache := NewDatabase(db)
	statedb, _ := New(common.Hash{}, stateCache)

	// Update it with some exchanges
	for i := 0; i < numberOrder; i++ {
		statedb.SetNonce(relayers[i], uint64(1))
	}
	mapPriceSell := map[uint64]uint64{}
	mapPriceBuy := map[uint64]uint64{}

	for i := 0; i < len(orderItems); i++ {
		amount := orderItems[i].Quantity.Uint64()
		orderIdHash := common.BigToHash(new(big.Int).SetUint64(orderItems[i].OrderID))
		statedb.InsertOrderItem(orderBook, orderIdHash, orderItems[i])

		switch orderItems[i].Side {
		case Ask:
			old := mapPriceSell[amount]
			mapPriceSell[amount] = old + amount
		case Bid:
			old := mapPriceBuy[amount]
			mapPriceBuy[amount] = old + amount
		default:
		}

	}
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

	orderIdHash := common.BigToHash(new(big.Int).SetUint64(orderItems[0].OrderID))
	order := statedb.GetOrder(orderBook, orderIdHash)
	// sub amount order
	wanted := statedb.GetVolume(orderBook, order.Price, order.Side)
	snap := statedb.Snapshot()
	statedb.SubAmountOrderItem(orderBook, orderIdHash, order.Price, order.Quantity, order.Side)
	statedb.RevertToSnapshot(snap)
	got := statedb.GetVolume(orderBook, order.Price, order.Side)
	if got.Cmp(wanted) != 0 {
		t.Fatalf(" err get volume price : %d after try revert snap shot , got : %d ,want : %d", order.Price, got, wanted)
	}
	// set nonce
	wantedNonce := statedb.GetNonce(relayers[1])
	snap = statedb.Snapshot()
	statedb.SetNonce(relayers[1], 0)
	statedb.RevertToSnapshot(snap)
	gotNonce := statedb.GetNonce(relayers[1])
	if wantedNonce != gotNonce {
		t.Fatalf(" err get nonce addr: %v after try revert snap shot , got : %d ,want : %d", relayers[1].Hex(), gotNonce, wantedNonce)
	}

	// cancel order
	wantedOrder := statedb.GetOrder(orderBook, orderIdHash)
	snap = statedb.Snapshot()
	statedb.CancelOrder(orderBook, &wantedOrder)
	statedb.RevertToSnapshot(snap)
	gotOrder := statedb.GetOrder(orderBook, orderIdHash)
	if gotOrder.Quantity.Cmp(wantedOrder.Quantity) != 0 {
		t.Fatalf(" err cancel order info : %v after try revert snap shot , got : %v ,want : %v", orderIdHash.Hex(), gotOrder, wantedOrder)
	}

	// insert order
	i := 2*numberOrder + 1
	id := new(big.Int).SetUint64(uint64(i) + 1)
	testOrder := OrderItem{OrderID: id.Uint64(), Quantity: big.NewInt(int64(2*i + 1)), Price: big.NewInt(int64(2*i + 1)), Side: Ask, Signature: &Signature{V: 1, R: common.HexToHash("111111"), S: common.HexToHash("222222222222")}}
	orderIdHash = common.BigToHash(new(big.Int).SetUint64(testOrder.OrderID))
	snap = statedb.Snapshot()
	statedb.InsertOrderItem(orderBook, orderIdHash, testOrder)
	statedb.RevertToSnapshot(snap)
	gotOrder = statedb.GetOrder(orderBook, orderIdHash)
	if gotOrder.Quantity.Cmp(EmptyOrder.Quantity) != 0 {
		t.Fatalf(" err insert order info : %v after try revert snap shot , got : %v ,want Empty Order", orderIdHash.Hex(), gotOrder)
	}
	// change price
	price := big.NewInt(10000)
	statedb.SetLastPrice(orderBook, price)
	snap = statedb.Snapshot()
	statedb.SetLastPrice(orderBook, big.NewInt(0))
	statedb.RevertToSnapshot(snap)
	gotPrice := statedb.GetLastPrice(orderBook)
	if gotPrice.Cmp(price) != 0 {
		t.Fatalf("Error when get price save in database: got : %d , wanted : %d ", gotPrice, price)
	}
	db.Close()
}

func TestDumpState(t *testing.T) {
	orderBook := common.StringToHash("BTC/XDC")
	numberOrder := 5
	orderItems := []OrderItem{}
	for i := 0; i < numberOrder; i++ {
		id := new(big.Int).SetUint64(uint64(i) + 1)
		orderItems = append(orderItems, OrderItem{OrderID: id.Uint64(), Quantity: big.NewInt(int64(2*i + 1)), Price: big.NewInt(int64(2*i + 1)), Side: Ask, Signature: &Signature{V: 1, R: common.HexToHash("111111"), S: common.HexToHash("222222222222")}})
		orderItems = append(orderItems, OrderItem{OrderID: id.Uint64(), Quantity: big.NewInt(int64(2*i + 2)), Price: big.NewInt(int64(2*i + 2)), Side: Bid, Signature: &Signature{V: 1, R: common.HexToHash("3333333333"), S: common.HexToHash("22222222222222222")}})
	}
	// Create an empty statedb database
	db := rawdb.NewMemoryDatabase()
	stateCache := NewDatabase(db)
	statedb, _ := New(common.Hash{}, stateCache)

	for i := 0; i < len(orderItems); i++ {
		orderIdHash := common.BigToHash(new(big.Int).SetUint64(orderItems[i].OrderID))
		statedb.InsertOrderItem(orderBook, orderIdHash, orderItems[i])
	}
	bidTrie, _ := statedb.DumpAskTrie(orderBook)
	fmt.Println("bidTrie", bidTrie)
	root := statedb.IntermediateRoot()
	statedb.Commit()
	stateCache.TrieDB().Reference(root, common.Hash{})
	statedb, err := New(root, stateCache)
	if err != nil {
		t.Fatalf("Error when get trie in database: %s , err: %v", root.Hex(), err)
	}
	maxPrice, volumeMax := statedb.GetBestBidPrice(orderBook)
	minPrice, volumeMin := statedb.GetBestAskPrice(orderBook)
	fmt.Println("price", minPrice, volumeMin, maxPrice, volumeMax)

	bidTrie, _ = statedb.DumpBidTrie(orderBook)

	fmt.Println("bidTrie", bidTrie)
	db.Close()
}
