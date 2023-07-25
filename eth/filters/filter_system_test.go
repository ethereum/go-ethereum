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
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/filtermaps"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	logger "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

type testBackend struct {
	db              ethdb.Database
	fm              *filtermaps.FilterMaps
	txFeed          event.Feed
	logsFeed        event.Feed
	rmLogsFeed      event.Feed
	chainFeed       event.Feed
	pendingBlock    *types.Block
	pendingReceipts types.Receipts
}

func (b *testBackend) ChainConfig() *params.ChainConfig {
	return params.TestChainConfig
}

func (b *testBackend) CurrentHeader() *types.Header {
	hdr, _ := b.HeaderByNumber(context.TODO(), rpc.LatestBlockNumber)
	return hdr
}

func (b *testBackend) CurrentBlock() *types.Header {
	return b.CurrentHeader()
}

func (b *testBackend) ChainDb() ethdb.Database {
	return b.db
}

func (b *testBackend) GetCanonicalHash(number uint64) common.Hash {
	return rawdb.ReadCanonicalHash(b.db, number)
}

func (b *testBackend) GetHeader(hash common.Hash, number uint64) *types.Header {
	hdr, _ := b.HeaderByHash(context.Background(), hash)
	return hdr
}

func (b *testBackend) GetReceiptsByHash(hash common.Hash) types.Receipts {
	r, _ := b.GetReceipts(context.Background(), hash)
	return r
}

func (b *testBackend) GetRawReceipts(hash common.Hash, number uint64) types.Receipts {
	return rawdb.ReadRawReceipts(b.db, hash, number)
}

func (b *testBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	var (
		hash common.Hash
		num  uint64
	)
	switch blockNr {
	case rpc.LatestBlockNumber:
		hash = rawdb.ReadHeadBlockHash(b.db)
		number, ok := rawdb.ReadHeaderNumber(b.db, hash)
		if !ok {
			return nil, nil
		}
		num = number
	case rpc.FinalizedBlockNumber:
		hash = rawdb.ReadFinalizedBlockHash(b.db)
		number, ok := rawdb.ReadHeaderNumber(b.db, hash)
		if !ok {
			return nil, nil
		}
		num = number
	case rpc.SafeBlockNumber:
		return nil, errors.New("safe block not found")
	default:
		num = uint64(blockNr)
		hash = rawdb.ReadCanonicalHash(b.db, num)
	}
	return rawdb.ReadHeader(b.db, hash, num), nil
}

func (b *testBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	number, ok := rawdb.ReadHeaderNumber(b.db, hash)
	if !ok {
		return nil, nil
	}
	return rawdb.ReadHeader(b.db, hash, number), nil
}

func (b *testBackend) GetBody(ctx context.Context, hash common.Hash, number rpc.BlockNumber) (*types.Body, error) {
	if body := rawdb.ReadBody(b.db, hash, uint64(number)); body != nil {
		return body, nil
	}
	return nil, errors.New("block body not found")
}

func (b *testBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	if number, ok := rawdb.ReadHeaderNumber(b.db, hash); ok {
		if header := rawdb.ReadHeader(b.db, hash, number); header != nil {
			return rawdb.ReadReceipts(b.db, hash, number, header.Time, params.TestChainConfig), nil
		}
	}
	return nil, nil
}

func (b *testBackend) GetLogs(ctx context.Context, hash common.Hash, number uint64) ([][]*types.Log, error) {
	logs := rawdb.ReadLogs(b.db, hash, number)
	return logs, nil
}

func (b *testBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.txFeed.Subscribe(ch)
}

func (b *testBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.rmLogsFeed.Subscribe(ch)
}

func (b *testBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.logsFeed.Subscribe(ch)
}

func (b *testBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.chainFeed.Subscribe(ch)
}

func (b *testBackend) CurrentView() *filtermaps.ChainView {
	head := b.CurrentBlock()
	return filtermaps.NewChainView(b, head.Number.Uint64(), head.Hash())
}

func (b *testBackend) NewMatcherBackend() filtermaps.MatcherBackend {
	return b.fm.NewMatcherBackend()
}

func (b *testBackend) forwardLogEvents(logCh chan []*types.Log, removedLogCh chan core.RemovedLogsEvent) {
	go func() {
		for {
			select {
			case logs := <-logCh:
				b.logsFeed.Send(logs)
			case logs := <-removedLogCh:
				b.rmLogsFeed.Send(logs)
			}
		}
	}()
}

func newTestFilterSystem(t testing.TB, db ethdb.Database, cfg Config) (*testBackend, *FilterSystem) {
	backend := &testBackend{db: db}
	sys := NewFilterSystem(backend, cfg)
	return backend, sys
}

// TestBlockSubscription tests if a block subscription returns block hashes for posted chain events.
// It creates multiple subscriptions:
// - one at the start and should receive all posted chain events and a second (blockHashes)
// - one that is created after a cutoff moment and uninstalled after a second cutoff moment (blockHashes[cutoff1:cutoff2])
// - one that is created after the second cutoff moment (blockHashes[cutoff2:])
func TestBlockSubscription(t *testing.T) {
	t.Parallel()

	var (
		db           = rawdb.NewMemoryDatabase()
		backend, sys = newTestFilterSystem(db, Config{})
		api          = NewFilterAPI(sys)
		genesis      = &core.Genesis{
			Config:  params.TestChainConfig,
			BaseFee: big.NewInt(params.InitialBaseFee),
		}
		_, chain, _ = core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), 10, func(i int, gen *core.BlockGen) {})
		chainEvents []core.ChainEvent
	)

	for _, blk := range chain {
		chainEvents = append(chainEvents, core.ChainEvent{Header: blk.Header()})
	}

	chan0 := make(chan *types.Header)
	sub0 := api.events.SubscribeNewHeads(chan0)
	chan1 := make(chan *types.Header)
	sub1 := api.events.SubscribeNewHeads(chan1)

	go func() { // simulate client
		i1, i2 := 0, 0
		for i1 != len(chainEvents) || i2 != len(chainEvents) {
			select {
			case header := <-chan0:
				if chainEvents[i1].Header.Hash() != header.Hash() {
					t.Errorf("sub0 received invalid hash on index %d, want %x, got %x", i1, chainEvents[i1].Header.Hash(), header.Hash())
				}
				i1++
			case header := <-chan1:
				if chainEvents[i2].Header.Hash() != header.Hash() {
					t.Errorf("sub1 received invalid hash on index %d, want %x, got %x", i2, chainEvents[i2].Header.Hash(), header.Hash())
				}
				i2++
			}
		}

		sub0.Unsubscribe()
		sub1.Unsubscribe()
	}()

	time.Sleep(1 * time.Second)
	for _, e := range chainEvents {
		backend.chainFeed.Send(e)
	}

	<-sub0.Err()
	<-sub1.Err()
}

// TestPendingTxFilter tests whether pending tx filters retrieve all pending transactions that are posted to the event mux.
func TestPendingTxFilter(t *testing.T) {
	t.Parallel()

	var (
		db           = rawdb.NewMemoryDatabase()
		backend, sys = newTestFilterSystem(db, Config{})
		api          = NewFilterAPI(sys)

		transactions = []*types.Transaction{
			types.NewTransaction(0, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
			types.NewTransaction(1, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
			types.NewTransaction(2, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
			types.NewTransaction(3, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
			types.NewTransaction(4, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
		}

		hashes []common.Hash
	)

	fid0 := api.NewPendingTransactionFilter(nil)

	time.Sleep(1 * time.Second)
	backend.txFeed.Send(core.NewTxsEvent{Txs: transactions})

	timeout := time.Now().Add(1 * time.Second)
	for {
		results, err := api.GetFilterChanges(fid0)
		if err != nil {
			t.Fatalf("Unable to retrieve logs: %v", err)
		}

		h := results.([]common.Hash)
		hashes = append(hashes, h...)
		if len(hashes) >= len(transactions) {
			break
		}
		// check timeout
		if time.Now().After(timeout) {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	if len(hashes) != len(transactions) {
		t.Errorf("invalid number of transactions, want %d transactions(s), got %d", len(transactions), len(hashes))
		return
	}
	for i := range hashes {
		if hashes[i] != transactions[i].Hash() {
			t.Errorf("hashes[%d] invalid, want %x, got %x", i, transactions[i].Hash(), hashes[i])
		}
	}
}

// TestPendingTxFilterFullTx tests whether pending tx filters retrieve all pending transactions that are posted to the event mux.
func TestPendingTxFilterFullTx(t *testing.T) {
	t.Parallel()

	var (
		db           = rawdb.NewMemoryDatabase()
		backend, sys = newTestFilterSystem(db, Config{})
		api          = NewFilterAPI(sys)

		transactions = []*types.Transaction{
			types.NewTransaction(0, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
			types.NewTransaction(1, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
			types.NewTransaction(2, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
			types.NewTransaction(3, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
			types.NewTransaction(4, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
		}

		txs []*ethapi.RPCTransaction
	)

	fullTx := true
	fid0 := api.NewPendingTransactionFilter(&fullTx)

	time.Sleep(1 * time.Second)
	backend.txFeed.Send(core.NewTxsEvent{Txs: transactions})

	timeout := time.Now().Add(1 * time.Second)
	for {
		results, err := api.GetFilterChanges(fid0)
		if err != nil {
			t.Fatalf("Unable to retrieve logs: %v", err)
		}

		tx := results.([]*ethapi.RPCTransaction)
		txs = append(txs, tx...)
		if len(txs) >= len(transactions) {
			break
		}
		// check timeout
		if time.Now().After(timeout) {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	if len(txs) != len(transactions) {
		t.Errorf("invalid number of transactions, want %d transactions(s), got %d", len(transactions), len(txs))
		return
	}
	for i := range txs {
		if txs[i].Hash != transactions[i].Hash() {
			t.Errorf("hashes[%d] invalid, want %x, got %x", i, transactions[i].Hash(), txs[i].Hash)
		}
	}
}

// TestLogFilterCreation test whether a given filter criteria makes sense.
// If not it must return an error.
func TestLogFilterCreation(t *testing.T) {
	var (
		db     = rawdb.NewMemoryDatabase()
		_, sys = newTestFilterSystem(db, Config{})
		api    = NewFilterAPI(sys)

		testCases = []struct {
			crit    FilterCriteria
			success bool
		}{
			// defaults
			{FilterCriteria{}, true},
			// valid block number range
			{FilterCriteria{FromBlock: big.NewInt(1), ToBlock: big.NewInt(2)}, true},
			// "mined" block range to pending
			{FilterCriteria{FromBlock: big.NewInt(1), ToBlock: big.NewInt(rpc.LatestBlockNumber.Int64())}, true},
			// from block "higher" than to block
			{FilterCriteria{FromBlock: big.NewInt(2), ToBlock: big.NewInt(1)}, false},
			// from block "higher" than to block
			{FilterCriteria{FromBlock: big.NewInt(rpc.LatestBlockNumber.Int64()), ToBlock: big.NewInt(100)}, false},
			// from block "higher" than to block
			{FilterCriteria{FromBlock: big.NewInt(rpc.PendingBlockNumber.Int64()), ToBlock: big.NewInt(100)}, false},
			// from block "higher" than to block
			{FilterCriteria{FromBlock: big.NewInt(rpc.PendingBlockNumber.Int64()), ToBlock: big.NewInt(rpc.LatestBlockNumber.Int64())}, false},
			// topics more than 4
			{FilterCriteria{Topics: [][]common.Hash{{}, {}, {}, {}, {}}}, false},
		}
	)

	for i, test := range testCases {
		id, err := api.NewFilter(test.crit)
		if err != nil && test.success {
			t.Errorf("expected filter creation for case %d to success, got %v", i, err)
		}
		if err == nil {
			api.UninstallFilter(id)
			if !test.success {
				t.Errorf("expected testcase %d to fail with an error", i)
			}
		}
	}
}

// TestInvalidLogFilterCreation tests whether invalid filter log criteria results in an error
// when the filter is created.
func TestInvalidLogFilterCreation(t *testing.T) {
	t.Parallel()

	var (
		db     = rawdb.NewMemoryDatabase()
		_, sys = newTestFilterSystem(db, Config{})
		api    = NewFilterAPI(sys)
	)

	// different situations where log filter creation should fail.
	// Reason: fromBlock > toBlock
	testCases := []FilterCriteria{
		0: {FromBlock: big.NewInt(rpc.PendingBlockNumber.Int64()), ToBlock: big.NewInt(rpc.LatestBlockNumber.Int64())},
		1: {FromBlock: big.NewInt(rpc.PendingBlockNumber.Int64()), ToBlock: big.NewInt(100)},
		2: {FromBlock: big.NewInt(rpc.LatestBlockNumber.Int64()), ToBlock: big.NewInt(100)},
		3: {Topics: [][]common.Hash{{}, {}, {}, {}, {}}},
		4: {Addresses: make([]common.Address, maxAddresses+1)},
	}

	for i, test := range testCases {
		if _, err := api.NewFilter(test); err == nil {
			t.Errorf("Expected NewFilter for case #%d to fail", i)
		}
	}
}

// TestInvalidGetLogsRequest tests invalid getLogs requests
func TestInvalidGetLogsRequest(t *testing.T) {
	t.Parallel()

	var (
		genesis = &core.Genesis{
			Config:  params.TestChainConfig,
			BaseFee: big.NewInt(params.InitialBaseFee),
		}
		db, blocks, _    = core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), 10, func(i int, gen *core.BlockGen) {})
		_, sys           = newTestFilterSystem(db, Config{})
		api              = NewFilterAPI(sys)
		blockHash        = blocks[0].Hash()
		unknownBlockHash = common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	)

	// Insert the blocks into the chain so filter can look them up
	blockchain, err := core.NewBlockChain(db, genesis, ethash.NewFaker(), nil)
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	if n, err := blockchain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}

	type testcase struct {
		f   FilterCriteria
		err error
	}
	testCases := []testcase{
		{
			f:   FilterCriteria{BlockHash: &blockHash, FromBlock: big.NewInt(100)},
			err: errBlockHashWithRange,
		},
		{
			f:   FilterCriteria{BlockHash: &blockHash, ToBlock: big.NewInt(500)},
			err: errBlockHashWithRange,
		},
		{
			f:   FilterCriteria{BlockHash: &blockHash, FromBlock: big.NewInt(rpc.LatestBlockNumber.Int64())},
			err: errBlockHashWithRange,
		},
		{
			f:   FilterCriteria{BlockHash: &unknownBlockHash},
			err: errUnknownBlock,
		},
		{
			f:   FilterCriteria{BlockHash: &blockHash, Topics: [][]common.Hash{{}, {}, {}, {}, {}}},
			err: errExceedMaxTopics,
		},
		{
			f:   FilterCriteria{BlockHash: &blockHash, Topics: [][]common.Hash{{}, {}, {}, {}, {}}},
			err: errExceedMaxTopics,
		},
		{
			f:   FilterCriteria{BlockHash: &blockHash, Addresses: make([]common.Address, maxAddresses+1)},
			err: errExceedMaxAddresses,
		},
	}

	for i, test := range testCases {
		_, err := api.GetLogs(context.Background(), test.f)
		if !errors.Is(err, test.err) {
			t.Errorf("case %d: wrong error: %q\nwant: %q", i, err, test.err)
		}
	}
}

// TestInvalidGetRangeLogsRequest tests getLogs with invalid block range
func TestInvalidGetRangeLogsRequest(t *testing.T) {
	t.Parallel()

	var (
		db     = rawdb.NewMemoryDatabase()
		_, sys = newTestFilterSystem(db, Config{})
		api    = NewFilterAPI(sys)
	)

	if _, err := api.GetLogs(context.Background(), FilterCriteria{FromBlock: big.NewInt(2), ToBlock: big.NewInt(1)}); err != errInvalidBlockRange {
		t.Errorf("Expected Logs for invalid range return error, but got: %v", err)
	}
}

// TestLogFilter tests whether log filters match the correct logs that are posted to the event feed.
func TestLogFilter(t *testing.T) {
	t.Parallel()

	var (
		db           = rawdb.NewMemoryDatabase()
		backend, sys = newTestFilterSystem(db, Config{})
		api          = NewFilterAPI(sys)

		firstAddr      = common.HexToAddress("0x1111111111111111111111111111111111111111")
		secondAddr     = common.HexToAddress("0x2222222222222222222222222222222222222222")
		thirdAddress   = common.HexToAddress("0x3333333333333333333333333333333333333333")
		notUsedAddress = common.HexToAddress("0x9999999999999999999999999999999999999999")
		firstTopic     = common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
		secondTopic    = common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
		notUsedTopic   = common.HexToHash("0x9999999999999999999999999999999999999999999999999999999999999999")

		// posted twice, once as regular logs and once as pending logs.
		allLogs = []*types.Log{
			{Address: firstAddr},
			{Address: firstAddr, Topics: []common.Hash{firstTopic}, BlockNumber: 1},
			{Address: secondAddr, Topics: []common.Hash{firstTopic}, BlockNumber: 1},
			{Address: thirdAddress, Topics: []common.Hash{secondTopic}, BlockNumber: 2},
			{Address: thirdAddress, Topics: []common.Hash{secondTopic}, BlockNumber: 3},
		}

		testCases = []struct {
			crit     FilterCriteria
			expected []*types.Log
			id       rpc.ID
		}{
			// match all
			0: {FilterCriteria{}, allLogs, ""},
			// match none due to no matching addresses
			1: {FilterCriteria{Addresses: []common.Address{{}, notUsedAddress}, Topics: [][]common.Hash{nil}}, []*types.Log{}, ""},
			// match logs based on addresses, ignore topics
			2: {FilterCriteria{Addresses: []common.Address{firstAddr}}, allLogs[:2], ""},
			// match none due to no matching topics (match with address)
			3: {FilterCriteria{Addresses: []common.Address{secondAddr}, Topics: [][]common.Hash{{notUsedTopic}}}, []*types.Log{}, ""},
			// match logs based on addresses and topics
			4: {FilterCriteria{Addresses: []common.Address{thirdAddress}, Topics: [][]common.Hash{{firstTopic, secondTopic}}}, allLogs[3:5], ""},
			// match logs based on multiple addresses and "or" topics
			5: {FilterCriteria{Addresses: []common.Address{secondAddr, thirdAddress}, Topics: [][]common.Hash{{firstTopic, secondTopic}}}, allLogs[2:5], ""},
			// all "mined" logs with block num >= 2
			6: {FilterCriteria{FromBlock: big.NewInt(2), ToBlock: big.NewInt(rpc.LatestBlockNumber.Int64())}, allLogs[3:], ""},
			// all "mined" logs
			7: {FilterCriteria{ToBlock: big.NewInt(rpc.LatestBlockNumber.Int64())}, allLogs, ""},
			// all "mined" logs with 1>= block num <=2 and topic secondTopic
			8: {FilterCriteria{FromBlock: big.NewInt(1), ToBlock: big.NewInt(2), Topics: [][]common.Hash{{secondTopic}}}, allLogs[3:4], ""},
			// match all logs due to wildcard topic
			9: {FilterCriteria{Topics: [][]common.Hash{nil}}, allLogs[1:], ""},
		}
	)

	// create all filters
	for i := range testCases {
		testCases[i].id, _ = api.NewFilter(testCases[i].crit)
	}

	// raise events
	time.Sleep(1 * time.Second)
	if nsend := backend.logsFeed.Send(allLogs); nsend == 0 {
		t.Fatal("Logs event not delivered")
	}

	for i, tt := range testCases {
		var fetched []*types.Log
		timeout := time.Now().Add(1 * time.Second)
		for { // fetch all expected logs
			results, err := api.GetFilterChanges(tt.id)
			if err != nil {
				t.Fatalf("test %d: unable to fetch logs: %v", i, err)
			}

			fetched = append(fetched, results.([]*types.Log)...)
			if len(fetched) >= len(tt.expected) {
				break
			}
			// check timeout
			if time.Now().After(timeout) {
				break
			}

			time.Sleep(100 * time.Millisecond)
		}

		if len(fetched) != len(tt.expected) {
			t.Errorf("invalid number of logs for case %d, want %d log(s), got %d", i, len(tt.expected), len(fetched))
			return
		}

		for l := range fetched {
			if fetched[l].Removed {
				t.Errorf("expected log not to be removed for log %d in case %d", l, i)
			}
			if !reflect.DeepEqual(fetched[l], tt.expected[l]) {
				t.Errorf("invalid log on index %d for case %d", l, i)
			}
		}
	}
}

// TestPendingTxFilterDeadlock tests if the event loop hangs when pending
// txes arrive at the same time that one of multiple filters is timing out.
// Please refer to #22131 for more details.
func TestPendingTxFilterDeadlock(t *testing.T) {
	t.Parallel()
	timeout := 100 * time.Millisecond

	var (
		db           = rawdb.NewMemoryDatabase()
		backend, sys = newTestFilterSystem(db, Config{Timeout: timeout})
		api          = NewFilterAPI(sys)
		done         = make(chan struct{})
	)

	go func() {
		// Bombard feed with txes until signal was received to stop
		i := uint64(0)
		for {
			select {
			case <-done:
				return
			default:
			}

			tx := types.NewTransaction(i, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil)
			backend.txFeed.Send(core.NewTxsEvent{Txs: []*types.Transaction{tx}})
			i++
		}
	}()

	// Create a bunch of filters that will
	// timeout either in 100ms or 200ms
	subs := make([]*Subscription, 20)
	for i := range subs {
		fid := api.NewPendingTransactionFilter(nil)
		api.filtersMu.Lock()
		f, ok := api.filters[fid]
		api.filtersMu.Unlock()
		if !ok {
			t.Fatalf("Filter %s should exist", fid)
		}
		subs[i] = f.s
		// Wait for at least one tx to arrive in filter
		for {
			hashes, err := api.GetFilterChanges(fid)
			if err != nil {
				t.Fatalf("Filter should exist: %v\n", err)
			}
			if len(hashes.([]common.Hash)) > 0 {
				break
			}
			runtime.Gosched()
		}
	}

	// Wait until filters have timed out and have been uninstalled.
	for _, sub := range subs {
		select {
		case <-sub.Err():
		case <-time.After(1 * time.Second):
			t.Fatalf("Filter timeout is hanging")
		}
	}
}

func flattenLogs(pl [][]*types.Log) []*types.Log {
	var logs []*types.Log
	for _, l := range pl {
		logs = append(logs, l...)
	}
	return logs
}

type mockNotifier struct {
	c chan interface{}
}

func newMockNotifier() *mockNotifier {
	return &mockNotifier{c: make(chan interface{})}
}

func (n *mockNotifier) Notify(id rpc.ID, data interface{}) error {
	n.c <- data
	return nil
}

func (n *mockNotifier) Closed() <-chan interface{} {
	return nil
}

// TestLogsSubscription tests if a rpc subscription receives the correct logs
func TestLogsSubscription(t *testing.T) {
	t.Parallel()

	var (
		db       = rawdb.NewMemoryDatabase()
		signer   = types.HomesteadSigner{}
		key, _   = crypto.GenerateKey()
		addr     = crypto.PubkeyToAddress(key.PublicKey)
		contract = common.HexToAddress("0000000000000000000000000000000000031ec7")
		genesis  = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc: core.GenesisAlloc{
				// // SPDX-License-Identifier: GPL-3.0
				// pragma solidity >=0.7.0 <0.9.0;
				//
				// contract Token {
				//     event Transfer(address indexed from, address indexed to, uint256 value);
				//     function transfer(address to, uint256 value) public returns (bool) {
				//         emit Transfer(msg.sender, to, value);
				//         return true;
				//     }
				// }
				contract: {Balance: big.NewInt(params.Ether), Code: common.FromHex("0x608060405234801561001057600080fd5b506004361061002b5760003560e01c8063a9059cbb14610030575b600080fd5b61004a6004803603810190610045919061016a565b610060565b60405161005791906101c5565b60405180910390f35b60008273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040516100bf91906101ef565b60405180910390a36001905092915050565b600080fd5b60073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000610101826100d6565b9050919050565b610111816100f6565b811461011c57600080fd5b50565b60008135905061012e81610108565b92915050565b6000819050919050565b61014781610134565b811461015257600080fd5b50565b6000813590506101648161013e565b92915050565b60008060408385031215610181576101806100d1565b5b600061018f8582860161011f565b92505060206101a085828601610155565b9150509250929050565b60008115159050919050565b6101bf816101aa565b82525050565b60006020820190506101da60008301846101b6565b92915050565b6101e981610134565b82525050565b600060208201905061020460008301846101e0565b9291505056fea2646970667358221220b469033f4b77b9565ee84e0a2f04d496b18160d26034d54f9487e57788fd36d564736f6c63430008120033")},
				addr:     {Balance: big.NewInt(params.Ether)},
			},
		}
	)

	// Hack: GenerateChainWithGenesis creates a new db.
	// Commit the genesis manually and use GenerateChain.
	_, err := genesis.Commit(db, trie.NewDatabase(db))
	if err != nil {
		t.Fatal(err)
	}
	blocks, _ := core.GenerateChain(genesis.Config, genesis.ToBlock(), ethash.NewFaker(), db, 4, func(i int, b *core.BlockGen) {
		// transfer(address to, uint256 value)
		data := fmt.Sprintf("0xa9059cbb%s%s", common.HexToHash(common.BigToAddress(big.NewInt(int64(i + 1))).Hex()).String()[2:], common.BytesToHash([]byte{byte(i + 11)}).String()[2:])
		tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{Nonce: uint64(i), To: &contract, Value: big.NewInt(0), Gas: 46000, GasPrice: b.BaseFee(), Data: common.FromHex(data)}), signer, key)
		b.AddTx(tx)
	})
	bc, err := core.NewBlockChain(db, nil, genesis, nil, ethash.NewFaker(), vm.Config{}, nil, new(uint64))
	if err != nil {
		t.Fatal(err)
	}
	_, err = bc.InsertChain(blocks)
	if err != nil {
		t.Fatal(err)
	}

	// Generate pending block, logs for which
	// will be sent to subscription feed.
	_, preceipts := core.GenerateChain(genesis.Config, blocks[len(blocks)-1], ethash.NewFaker(), db, 1, func(i int, gen *core.BlockGen) {
		// transfer(address to, uint256 value)
		data := fmt.Sprintf("0xa9059cbb%s%s", common.HexToHash(common.BigToAddress(big.NewInt(int64(i + 1))).Hex()).String()[2:], common.BytesToHash([]byte{byte(i + 11)}).String()[2:])
		tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{Nonce: uint64(len(blocks) + i), To: &contract, Value: big.NewInt(0), Gas: 46000, GasPrice: gen.BaseFee(), Data: common.FromHex(data)}), signer, key)
		gen.AddTx(tx)
	})
	liveLogs := preceipts[0][0].Logs
	var (
		backend, sys = newTestFilterSystem(t, db, Config{})
		api          = NewFilterAPI(sys, false)
		// Transfer(address indexed from, address indexed to, uint256 value);
		topic = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	)

	i2h := func(i int) common.Hash { return common.BigToHash(big.NewInt(int64(i))) }

	allLogs := []*types.Log{
		{Address: contract, Topics: []common.Hash{topic, common.HexToHash(addr.Hex()), i2h(1)}, Data: i2h(11).Bytes(), BlockNumber: 1, BlockHash: blocks[0].Hash(), TxHash: blocks[0].Transactions()[0].Hash()},
		{Address: contract, Topics: []common.Hash{topic, common.HexToHash(addr.Hex()), i2h(2)}, Data: i2h(12).Bytes(), BlockNumber: 2, BlockHash: blocks[1].Hash(), TxHash: blocks[1].Transactions()[0].Hash()},
		{Address: contract, Topics: []common.Hash{topic, common.HexToHash(addr.Hex()), i2h(3)}, Data: i2h(13).Bytes(), BlockNumber: 3, BlockHash: blocks[2].Hash(), TxHash: blocks[2].Transactions()[0].Hash()},
		{Address: contract, Topics: []common.Hash{topic, common.HexToHash(addr.Hex()), i2h(4)}, Data: i2h(14).Bytes(), BlockNumber: 4, BlockHash: blocks[3].Hash(), TxHash: blocks[3].Transactions()[0].Hash()},
	}
	testCases := []struct {
		crit      FilterCriteria
		expected  []*types.Log
		notifier  *mockNotifier
		sub       *rpc.Subscription
		expectErr error
		err       chan error
	}{
		// from 0 to latest
		{
			FilterCriteria{FromBlock: big.NewInt(0)},
			append(allLogs, liveLogs...), newMockNotifier(), &rpc.Subscription{ID: rpc.NewID()}, nil, nil,
		},
		// from 1 to latest
		{
			FilterCriteria{FromBlock: big.NewInt(1), ToBlock: big.NewInt(rpc.LatestBlockNumber.Int64())},
			nil, newMockNotifier(), &rpc.Subscription{ID: rpc.NewID()}, errInvalidToBlock, nil,
		},
		// from 2 to latest
		{
			FilterCriteria{FromBlock: big.NewInt(2)},
			append(allLogs[1:], liveLogs...), newMockNotifier(), &rpc.Subscription{ID: rpc.NewID()}, nil, nil,
		},
		// from latest to latest
		{
			FilterCriteria{},
			liveLogs, newMockNotifier(), &rpc.Subscription{ID: rpc.NewID()}, nil, nil,
		},
		// from 1 to 3
		{
			FilterCriteria{FromBlock: big.NewInt(1), ToBlock: big.NewInt(3)},
			nil, newMockNotifier(), &rpc.Subscription{ID: rpc.NewID()}, errInvalidToBlock, nil,
		},
		// from -101 to latest
		{
			FilterCriteria{FromBlock: big.NewInt(-101)},
			nil, newMockNotifier(), &rpc.Subscription{ID: rpc.NewID()}, errInvalidFromBlock, nil,
		},
	}

	// subscribe logs
	for i, tc := range testCases {
		testCases[i].err = make(chan error)
		err := api.logs(context.Background(), tc.notifier, tc.sub, tc.crit)
		if tc.expectErr != nil {
			if err == nil {
				t.Errorf("test %d: want error %v, have nothing", i, tc.expectErr)
				continue
			}
			if !errors.Is(err, tc.expectErr) {
				t.Errorf("test %d: error mismatch, want %v, have %v", i, tc.expectErr, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("SubscribeLogs %d failed: %v\n", i, err)
		}
	}

	// receive logs
	for n, test := range testCases {
		i := n
		tt := test
		go func() {
			var fetched []*types.Log

			timeout := time.After(3 * time.Second)
		fetchLoop:
			for {
				select {
				case log := <-tt.notifier.c:
					fetched = append(fetched, *log.(**types.Log))
				case <-timeout:
					break fetchLoop
				}
			}

			if len(fetched) != len(tt.expected) {
				tt.err <- fmt.Errorf("invalid number of logs for case %d, want %d log(s), got %d", i, len(tt.expected), len(fetched))
				return
			}

			for l := range fetched {
				have, want := fetched[l], tt.expected[l]
				if !reflect.DeepEqual(have, want) {
					tt.err <- fmt.Errorf("invalid log on index %d for case %d have: %+v want: %+v\n", l, i, have, want)
					return
				}
			}
			tt.err <- nil
		}()
	}

	// Wait for historical logs to be processed.
	// The reason we need to wait is this test is artificial
	// and no new block containing the logs is mined.
	time.Sleep(1 * time.Second)
	// Send live logs
	backend.logsFeed.Send(liveLogs)

	for i := range testCases {
		err := <-testCases[i].err
		if err != nil {
			t.Fatalf("test %d failed: %v", i, err)
		}
	}
}

// TestLogsSubscriptionReorg tests the behavior of the filter system when a reorganization occurs.
func TestLogsSubscriptionReorg(t *testing.T) {
	t.Parallel()

	var (
		db           = rawdb.NewMemoryDatabase()
		backend, sys = newTestFilterSystem(t, db, Config{})
		api          = NewFilterAPI(sys, false)
		signer       = types.HomesteadSigner{}
		key, _       = crypto.GenerateKey()
		addr         = crypto.PubkeyToAddress(key.PublicKey)
		contract     = common.HexToAddress("0000000000000000000000000000000000031ec7")
		// Transfer(address indexed from, address indexed to, uint256 value);
		topic   = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
		genesis = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc: core.GenesisAlloc{
				// // SPDX-License-Identifier: GPL-3.0
				// pragma solidity >=0.7.0 <0.9.0;
				//
				// contract Token {
				//     event Transfer(address indexed from, address indexed to, uint256 value);
				//     function transfer(address to, uint256 value) public returns (bool) {
				//         emit Transfer(msg.sender, to, value);
				//         return true;
				//     }
				// }
				contract: {Balance: big.NewInt(params.Ether), Code: common.FromHex("0x608060405234801561001057600080fd5b506004361061002b5760003560e01c8063a9059cbb14610030575b600080fd5b61004a6004803603810190610045919061016a565b610060565b60405161005791906101c5565b60405180910390f35b60008273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040516100bf91906101ef565b60405180910390a36001905092915050565b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000610101826100d6565b9050919050565b610111816100f6565b811461011c57600080fd5b50565b60008135905061012e81610108565b92915050565b6000819050919050565b61014781610134565b811461015257600080fd5b50565b6000813590506101648161013e565b92915050565b60008060408385031215610181576101806100d1565b5b600061018f8582860161011f565b92505060206101a085828601610155565b9150509250929050565b60008115159050919050565b6101bf816101aa565b82525050565b60006020820190506101da60008301846101b6565b92915050565b6101e981610134565b82525050565b600060208201905061020460008301846101e0565b9291505056fea2646970667358221220b469033f4b77b9565ee84e0a2f04d496b18160d26034d54f9487e57788fd36d564736f6c63430008120033")},
				addr:     {Balance: big.NewInt(params.Ether)},
			},
		}
	)

	// Hack: GenerateChainWithGenesis creates a new db.
	// Commit the genesis manually and use GenerateChain.
	_, err := genesis.Commit(db, trie.NewDatabase(db))
	if err != nil {
		t.Fatal(err)
	}
	oldChain, _ := core.GenerateChain(genesis.Config, genesis.ToBlock(), ethash.NewFaker(), db, 5, func(i int, b *core.BlockGen) {
		// transfer(address to, uint256 value)
		data := fmt.Sprintf("0xa9059cbb%s%s", common.HexToHash(common.BigToAddress(big.NewInt(int64(i + 1))).Hex()).String()[2:], common.BytesToHash([]byte{byte(i + 11)}).String()[2:])
		tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{Nonce: uint64(i), To: &contract, Value: big.NewInt(0), Gas: 46000, GasPrice: b.BaseFee(), Data: common.FromHex(data)}), signer, key)
		b.AddTx(tx)
	})
	bc, err := core.NewBlockChain(db, nil, genesis, nil, ethash.NewFaker(), vm.Config{}, nil, new(uint64))
	if err != nil {
		t.Fatal(err)
	}
	// Hack: FilterSystem is using mock backend, i.e. blockchain events will be not received
	// by it. Forward them manually.
	var (
		bcLogsFeed   = make(chan []*types.Log)
		bcRmLogsFeed = make(chan core.RemovedLogsEvent)
	)
	backend.forwardLogEvents(bcLogsFeed, bcRmLogsFeed)
	bc.SubscribeLogsEvent(bcLogsFeed)
	bc.SubscribeRemovedLogsEvent(bcRmLogsFeed)

	_, err = bc.InsertChain(oldChain)
	if err != nil {
		t.Fatal(err)
	}
	newChain, _ := core.GenerateChain(genesis.Config, oldChain[1], ethash.NewFaker(), db, 4, func(i int, b *core.BlockGen) {
		// transfer(address to, uint256 value)
		data := fmt.Sprintf("0xa9059cbb%s%s", common.HexToHash(common.BigToAddress(big.NewInt(int64(i + 1))).Hex()).String()[2:], common.BytesToHash([]byte{byte(i + 103)}).String()[2:])
		tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{Nonce: uint64(2 + i), To: &contract, Value: big.NewInt(0), Gas: 46000, GasPrice: b.BaseFee(), Data: common.FromHex(data)}), signer, key)
		b.AddTx(tx)
	})

	// Generate pending block, logs for which
	// will be sent to subscription feed.
	_, preceipts := core.GenerateChain(genesis.Config, newChain[len(newChain)-1], ethash.NewFaker(), db, 1, func(i int, gen *core.BlockGen) {
		// transfer(address to, uint256 value)
		data := fmt.Sprintf("0xa9059cbb%s%s", common.HexToHash(common.BigToAddress(big.NewInt(int64(i + 1))).Hex()).String()[2:], common.BytesToHash([]byte{byte(i + 21)}).String()[2:])
		tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{Nonce: uint64(6 + i), To: &contract, Value: big.NewInt(0), Gas: 46000, GasPrice: gen.BaseFee(), Data: common.FromHex(data)}), signer, key)
		gen.AddTx(tx)
	})
	liveLogs := preceipts[0][0].Logs

	i2h := func(i int) common.Hash { return common.BigToHash(big.NewInt(int64(i))) }
	a2h := func(a common.Address) common.Hash { return common.HexToHash(a.Hex()) }
	c2h := func(bs []*types.Block, i int) common.Hash { return bs[i].Transactions()[0].Hash() }
	expected := []*types.Log{
		// Original chain until block 3
		{Address: contract, Topics: []common.Hash{topic, a2h(addr), i2h(1)}, Data: i2h(11).Bytes(), BlockNumber: 1, BlockHash: oldChain[0].Hash(), TxHash: c2h(oldChain, 0)},
		{Address: contract, Topics: []common.Hash{topic, a2h(addr), i2h(2)}, Data: i2h(12).Bytes(), BlockNumber: 2, BlockHash: oldChain[1].Hash(), TxHash: c2h(oldChain, 1)},
		{Address: contract, Topics: []common.Hash{topic, a2h(addr), i2h(3)}, Data: i2h(13).Bytes(), BlockNumber: 3, BlockHash: oldChain[2].Hash(), TxHash: c2h(oldChain, 2)},
		// Removed log for block 3
		{Address: contract, Topics: []common.Hash{topic, a2h(addr), i2h(3)}, Data: i2h(13).Bytes(), BlockNumber: 3, BlockHash: oldChain[2].Hash(), TxHash: c2h(oldChain, 2), Removed: true},
		// New logs for 4 onwards
		{Address: contract, Topics: []common.Hash{topic, a2h(addr), i2h(1)}, Data: i2h(103).Bytes(), BlockNumber: 3, BlockHash: newChain[0].Hash(), TxHash: c2h(newChain, 0)},
		{Address: contract, Topics: []common.Hash{topic, a2h(addr), i2h(2)}, Data: i2h(104).Bytes(), BlockNumber: 4, BlockHash: newChain[1].Hash(), TxHash: c2h(newChain, 1)},
		{Address: contract, Topics: []common.Hash{topic, a2h(addr), i2h(3)}, Data: i2h(105).Bytes(), BlockNumber: 5, BlockHash: newChain[2].Hash(), TxHash: c2h(newChain, 2)},
		{Address: contract, Topics: []common.Hash{topic, a2h(addr), i2h(4)}, Data: i2h(106).Bytes(), BlockNumber: 6, BlockHash: newChain[3].Hash(), TxHash: c2h(newChain, 3)},
	}
	expected = append(expected, liveLogs...)

	// Calculate address balances
	balanceDiffer := func(logs []*types.Log) map[common.Address]uint64 {
		balances := make(map[common.Address]uint64)
		for _, log := range logs {
			log := log
			from := common.BytesToAddress(log.Topics[1].Bytes())
			to := common.BytesToAddress(log.Topics[2].Bytes())
			amount := common.BytesToHash(log.Data).Big().Uint64()

			if _, ok := balances[from]; !ok {
				balances[from] = 0
			}
			if _, ok := balances[to]; !ok {
				balances[to] = 0
			}

			if log.Removed { // revert
				balances[from] += amount
				balances[to] -= amount
			} else {
				balances[from] -= amount
				balances[to] += amount
			}
		}
		// Remove zero balances
		for addr, balance := range balances {
			if balance == 0 {
				delete(balances, addr)
			}
		}
		return balances
	}
	expectedBalance := balanceDiffer(expected)

	// Subscribe to logs
	var (
		errc        = make(chan error)
		notifier    = newMockNotifier()
		sub         = &rpc.Subscription{ID: rpc.NewID()}
		crit        = FilterCriteria{FromBlock: big.NewInt(1)}
		reorgSignal = make(chan struct{})
	)
	err = api.logs(context.Background(), notifier, sub, crit)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		var fetched []*types.Log

		timeout := time.After(3 * time.Second)
	fetchLoop:
		for {
			select {
			case log := <-notifier.c:
				l := *log.(**types.Log)
				fetched = append(fetched, l)
				// We halt the sender by blocking Notify(). However sender will already prepare
				// logs for next block and send as soon as Notify is released. So we do reorg
				// one block earlier than we intend.
				if l.BlockNumber == 2 {
					// signal reorg
					reorgSignal <- struct{}{}
					// wait for reorg to happen
					<-reorgSignal
				}
			case <-timeout:
				break fetchLoop
			}
		}

		fetchedBalance := balanceDiffer(fetched)

		for i, log := range fetched {
			logger.Info("Flog", "i", i, "blknum", log.BlockNumber, "index", log.Index, "removed", log.Removed, "to", common.BytesToAddress(log.Topics[2].Bytes()), "amount", common.BytesToHash(log.Data).Big().Uint64())
		}

		for i, log := range expected {
			logger.Info("Elog", "i", i, "blknum", log.BlockNumber, "index", log.Index, "removed", log.Removed, "to", common.BytesToAddress(log.Topics[2].Bytes()), "amount", common.BytesToHash(log.Data).Big().Uint64())
		}
		if len(fetchedBalance) != len(expectedBalance) {
			errc <- fmt.Errorf("invalid number of balances, have %d, want %d", len(fetchedBalance), len(expectedBalance))
			logger.Info("balance diff", "fetched", fetchedBalance, "expected", expectedBalance)
			return
		}

		diffBalance := make(map[common.Address]struct{ have, want uint64 })
		for addr, balance := range expectedBalance {
			if fetchedBalance[addr] != balance {
				diffBalance[addr] = struct{ have, want uint64 }{fetchedBalance[addr], balance}
			}
		}
		if len(diffBalance) > 0 {
			errc <- fmt.Errorf("invalid balance detected: %+v\n", diffBalance)
			return
		}

		errc <- nil
	}()
	<-reorgSignal
	if n, err := bc.InsertChain(newChain); err != nil {
		t.Fatalf("failed to insert forked chain at %d: %v", n, err)
	}
	reorgSignal <- struct{}{}

	// Wait for historical logs to be processed.
	// The reason we need to wait is this test is artificial
	// and no new block containing the logs is mined.
	time.Sleep(2 * time.Second)
	// Send live logs
	backend.logsFeed.Send(liveLogs)

	err = <-errc
	if err != nil {
		t.Fatalf("test failed: %v", err)
	}
}
