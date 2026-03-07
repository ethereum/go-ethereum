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

package filters

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestUnmarshalJSONNewFilterArgs(t *testing.T) {
	var (
		fromBlock rpc.BlockNumber = 0x123435
		toBlock   rpc.BlockNumber = 0xabcdef
		address0                  = common.HexToAddress("70c87d191324e6712a591f304b4eedef6ad9bb9d")
		address1                  = common.HexToAddress("9b2055d370f73ec7d8a03e965129118dc8f5bf83")
		topic0                    = common.HexToHash("3ac225168df54212a25c1c01fd35bebfea408fdac2e31ddd6f80a4bbf9a5f1ca")
		topic1                    = common.HexToHash("9084a792d2f8b16a62b882fd56f7860c07bf5fa91dd8a2ae7e809e5180fef0b3")
		topic2                    = common.HexToHash("6ccae1c4af4152f460ff510e573399795dfab5dcf1fa60d1f33ac8fdc1e480ce")
	)

	// default values
	var test0 FilterCriteria
	if err := json.Unmarshal([]byte("{}"), &test0); err != nil {
		t.Fatal(err)
	}
	if test0.FromBlock != nil {
		t.Fatalf("expected nil, got %d", test0.FromBlock)
	}
	if test0.ToBlock != nil {
		t.Fatalf("expected nil, got %d", test0.ToBlock)
	}
	if len(test0.Addresses) != 0 {
		t.Fatalf("expected 0 addresses, got %d", len(test0.Addresses))
	}
	if len(test0.Topics) != 0 {
		t.Fatalf("expected 0 topics, got %d topics", len(test0.Topics))
	}

	// from, to block number
	var test1 FilterCriteria
	vector := fmt.Sprintf(`{"fromBlock":"%v","toBlock":"%v"}`, fromBlock, toBlock)
	if err := json.Unmarshal([]byte(vector), &test1); err != nil {
		t.Fatal(err)
	}
	if test1.FromBlock.Int64() != fromBlock.Int64() {
		t.Fatalf("expected FromBlock %d, got %d", fromBlock, test1.FromBlock)
	}
	if test1.ToBlock.Int64() != toBlock.Int64() {
		t.Fatalf("expected ToBlock %d, got %d", toBlock, test1.ToBlock)
	}

	// single address
	var test2 FilterCriteria
	vector = fmt.Sprintf(`{"address": "%s"}`, address0.Hex())
	if err := json.Unmarshal([]byte(vector), &test2); err != nil {
		t.Fatal(err)
	}
	if len(test2.Addresses) != 1 {
		t.Fatalf("expected 1 address, got %d address(es)", len(test2.Addresses))
	}
	if test2.Addresses[0] != address0 {
		t.Fatalf("expected address %x, got %x", address0, test2.Addresses[0])
	}

	// multiple address
	var test3 FilterCriteria
	vector = fmt.Sprintf(`{"address": ["%s", "%s"]}`, address0.Hex(), address1.Hex())
	if err := json.Unmarshal([]byte(vector), &test3); err != nil {
		t.Fatal(err)
	}
	if len(test3.Addresses) != 2 {
		t.Fatalf("expected 2 addresses, got %d address(es)", len(test3.Addresses))
	}
	if test3.Addresses[0] != address0 {
		t.Fatalf("expected address %x, got %x", address0, test3.Addresses[0])
	}
	if test3.Addresses[1] != address1 {
		t.Fatalf("expected address %x, got %x", address1, test3.Addresses[1])
	}

	// single topic
	var test4 FilterCriteria
	vector = fmt.Sprintf(`{"topics": ["%s"]}`, topic0.Hex())
	if err := json.Unmarshal([]byte(vector), &test4); err != nil {
		t.Fatal(err)
	}
	if len(test4.Topics) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(test4.Topics))
	}
	if len(test4.Topics[0]) != 1 {
		t.Fatalf("expected len(topics[0]) to be 1, got %d", len(test4.Topics[0]))
	}
	if test4.Topics[0][0] != topic0 {
		t.Fatalf("got %x, expected %x", test4.Topics[0][0], topic0)
	}

	// test multiple "AND" topics
	var test5 FilterCriteria
	vector = fmt.Sprintf(`{"topics": ["%s", "%s"]}`, topic0.Hex(), topic1.Hex())
	if err := json.Unmarshal([]byte(vector), &test5); err != nil {
		t.Fatal(err)
	}
	if len(test5.Topics) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(test5.Topics))
	}
	if len(test5.Topics[0]) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(test5.Topics[0]))
	}
	if test5.Topics[0][0] != topic0 {
		t.Fatalf("got %x, expected %x", test5.Topics[0][0], topic0)
	}
	if len(test5.Topics[1]) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(test5.Topics[1]))
	}
	if test5.Topics[1][0] != topic1 {
		t.Fatalf("got %x, expected %x", test5.Topics[1][0], topic1)
	}

	// test optional topic
	var test6 FilterCriteria
	vector = fmt.Sprintf(`{"topics": ["%s", null, "%s"]}`, topic0.Hex(), topic2.Hex())
	if err := json.Unmarshal([]byte(vector), &test6); err != nil {
		t.Fatal(err)
	}
	if len(test6.Topics) != 3 {
		t.Fatalf("expected 3 topics, got %d", len(test6.Topics))
	}
	if len(test6.Topics[0]) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(test6.Topics[0]))
	}
	if test6.Topics[0][0] != topic0 {
		t.Fatalf("got %x, expected %x", test6.Topics[0][0], topic0)
	}
	if len(test6.Topics[1]) != 0 {
		t.Fatalf("expected 0 topic, got %d", len(test6.Topics[1]))
	}
	if len(test6.Topics[2]) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(test6.Topics[2]))
	}
	if test6.Topics[2][0] != topic2 {
		t.Fatalf("got %x, expected %x", test6.Topics[2][0], topic2)
	}

	// test OR topics
	var test7 FilterCriteria
	vector = fmt.Sprintf(`{"topics": [["%s", "%s"], null, ["%s", null]]}`, topic0.Hex(), topic1.Hex(), topic2.Hex())
	if err := json.Unmarshal([]byte(vector), &test7); err != nil {
		t.Fatal(err)
	}
	if len(test7.Topics) != 3 {
		t.Fatalf("expected 3 topics, got %d topics", len(test7.Topics))
	}
	if len(test7.Topics[0]) != 2 {
		t.Fatalf("expected 2 topics, got %d topics", len(test7.Topics[0]))
	}
	if test7.Topics[0][0] != topic0 || test7.Topics[0][1] != topic1 {
		t.Fatalf("invalid topics expected [%x,%x], got [%x,%x]",
			topic0, topic1, test7.Topics[0][0], test7.Topics[0][1],
		)
	}
	if len(test7.Topics[1]) != 0 {
		t.Fatalf("expected 0 topic, got %d topics", len(test7.Topics[1]))
	}
	if len(test7.Topics[2]) != 0 {
		t.Fatalf("expected 0 topics, got %d topics", len(test7.Topics[2]))
	}
}

func TestGetFilterChangesDrainSemanticsLogs(t *testing.T) {
	t.Parallel()

	var (
		db           = rawdb.NewMemoryDatabase()
		backend, sys = newTestFilterSystem(db, Config{})
		api          = NewFilterAPI(sys)
	)
	id, err := api.NewFilter(FilterCriteria{})
	if err != nil {
		t.Fatalf("failed to create filter: %v", err)
	}

	want := &types.Log{
		Address:     common.HexToAddress("0x1000000000000000000000000000000000000001"),
		Topics:      []common.Hash{common.HexToHash("0x01")},
		BlockNumber: 7,
	}
	if nsend := backend.logsFeed.Send([]*types.Log{want}); nsend == 0 {
		t.Fatal("logs event not delivered")
	}

	var first []*types.Log
	timeout := time.Now().Add(time.Second)
	for {
		changes, err := api.GetFilterChanges(id)
		if err != nil {
			t.Fatalf("failed to fetch filter changes: %v", err)
		}
		first = changes.([]*types.Log)
		if len(first) > 0 || time.Now().After(timeout) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(first) != 1 {
		t.Fatalf("expected first poll to return 1 log, got %d", len(first))
	}
	if first[0].Address != want.Address || first[0].BlockNumber != want.BlockNumber {
		t.Fatalf("unexpected log returned: got address=%s block=%d", first[0].Address.Hex(), first[0].BlockNumber)
	}

	changes, err := api.GetFilterChanges(id)
	if err != nil {
		t.Fatalf("failed to fetch drained changes: %v", err)
	}
	second := changes.([]*types.Log)
	if len(second) != 0 {
		t.Fatalf("expected second poll to be empty, got %d logs", len(second))
	}
}

func TestGetFilterChangesHeadBoundaryIncludesHeadBlock(t *testing.T) {
	t.Parallel()

	genesis := &core.Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	db, chain, _ := core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), 2, func(i int, gen *core.BlockGen) {})
	blockchain, err := core.NewBlockChain(db, genesis, ethash.NewFaker(), nil)
	if err != nil {
		t.Fatalf("failed to create blockchain: %v", err)
	}
	if n, err := blockchain.InsertChain(chain[:1]); err != nil {
		t.Fatalf("failed to insert block %d: %v", n, err)
	}

	backend, sys := newTestFilterSystem(db, Config{})
	api := NewFilterAPI(sys)
	id := api.NewBlockFilter()

	head := backend.CurrentHeader()
	if head.Number.Uint64() != chain[0].NumberU64() {
		t.Fatalf("expected filter creation head %d, got %d", chain[0].NumberU64(), head.Number.Uint64())
	}
	if nsend := backend.chainFeed.Send(core.ChainEvent{Header: chain[1].Header()}); nsend == 0 {
		t.Fatal("chain event not delivered")
	}

	var hashes []common.Hash
	timeout := time.Now().Add(time.Second)
	for {
		changes, err := api.GetFilterChanges(id)
		if err != nil {
			t.Fatalf("failed to fetch filter changes: %v", err)
		}
		hashes = changes.([]common.Hash)
		if len(hashes) > 0 || time.Now().After(timeout) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(hashes) != 1 {
		t.Fatalf("expected 1 new head hash, got %d", len(hashes))
	}
	if hashes[0] != chain[1].Hash() {
		t.Fatalf("expected head hash %x, got %x", chain[1].Hash(), hashes[0])
	}
}

func TestGetFilterChangesDrainSemanticsPendingTx(t *testing.T) {
	t.Parallel()

	var (
		db           = rawdb.NewMemoryDatabase()
		backend, sys = newTestFilterSystem(db, Config{})
		api          = NewFilterAPI(sys)
	)

	fullTx := true
	id := api.NewPendingTransactionFilter(&fullTx)

	tx := types.NewTransaction(0, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil)
	backend.txFeed.Send(core.NewTxsEvent{Txs: []*types.Transaction{tx}})

	// Poll until the tx arrives.
	var first []*ethapi.RPCTransaction
	timeout := time.Now().Add(time.Second)
	for {
		changes, err := api.GetFilterChanges(id)
		if err != nil {
			t.Fatalf("failed to fetch filter changes: %v", err)
		}
		first = changes.([]*ethapi.RPCTransaction)
		if len(first) > 0 || time.Now().After(timeout) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(first) != 1 {
		t.Fatalf("expected 1 pending tx, got %d", len(first))
	}
	if first[0].Hash != tx.Hash() {
		t.Fatalf("expected tx hash %x, got %x", tx.Hash(), first[0].Hash)
	}

	// Second poll must be empty (drain semantics).
	changes, err := api.GetFilterChanges(id)
	if err != nil {
		t.Fatalf("failed to fetch drained changes: %v", err)
	}
	second := changes.([]*ethapi.RPCTransaction)
	if len(second) != 0 {
		t.Fatalf("expected second poll to be empty, got %d txs", len(second))
	}
}
