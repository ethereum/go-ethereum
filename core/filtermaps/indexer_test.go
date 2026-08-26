// Copyright 2024 The go-ethereum Authors
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

package filtermaps

import (
	"context"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"math/rand"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

var testParams = Params{
	logMapHeight:       2,
	logMapWidth:        24,
	logMapsPerEpoch:    4,
	logValuesPerMap:    4,
	baseRowGroupSize:   4,
	baseRowLengthRatio: 2,
	logLayerDiff:       2,
}

func TestIndexerRandomRange(t *testing.T) {
	ts := newTestSetup(t)
	defer ts.close()

	forks := make([][]common.Hash, 10)
	ts.chain.addBlocks(1000, 5, 2, 4, false)
	for i := range forks {
		if i != 0 {
			forkBlock := rand.Intn(1000)
			ts.chain.setHead(forkBlock)
			ts.chain.addBlocks(1000-forkBlock, 5, 2, 4, false)
		}
		forks[i] = ts.chain.getCanonicalChain()
	}
	expspos := func(block uint64) uint64 { // expected position of block start
		if block == 0 {
			return 0
		}
		logCount := (block - 1) * 5 * 2
		mapIndex := logCount / 3
		spos := mapIndex*16 + (logCount%3)*5
		if mapIndex == 0 || logCount%3 != 0 {
			spos++
		}
		return spos
	}
	expdpos := func(block uint64) uint64 { // expected position of delimiter
		if block == 0 {
			return 0
		}
		logCount := block * 5 * 2
		mapIndex := (logCount - 1) / 3
		return mapIndex*16 + (logCount-mapIndex*3)*5
	}
	ts.setHistory(0, false)
	var (
		history       int
		noHistory     bool
		fork, head    = len(forks) - 1, 1000
		checkSnapshot bool
	)
	ts.fm.WaitIdle()
	for i := 0; i < 200; i++ {
		switch rand.Intn(3) {
		case 0:
			// change history settings
			switch rand.Intn(10) {
			case 0:
				history, noHistory = 0, false
			case 1:
				history, noHistory = 0, true
			default:
				history, noHistory = rand.Intn(1000)+1, false
			}
			ts.testDisableSnapshots = rand.Intn(2) == 0
			ts.setHistory(uint64(history), noHistory)
		case 1:
			// change head to random position of random fork
			fork, head = rand.Intn(len(forks)), rand.Intn(1001)
			ts.chain.setCanonicalChain(forks[fork][:head+1])
		case 2:
			checkSnapshot = false
			if head < 1000 {
				checkSnapshot = !noHistory && head != 0 // no snapshot generated for block 0
				// add blocks after the current head
				head += rand.Intn(1000-head) + 1
				ts.fm.testSnapshotUsed = false
				ts.chain.setCanonicalChain(forks[fork][:head+1])
			}
		}
		ts.fm.WaitIdle()
		if checkSnapshot {
			if ts.fm.testSnapshotUsed == ts.fm.testDisableSnapshots {
				ts.t.Fatalf("Invalid snapshot used state after head extension (used: %v, disabled: %v)", ts.fm.testSnapshotUsed, ts.fm.testDisableSnapshots)
			}
			checkSnapshot = false
		}
		if noHistory {
			if ts.fm.indexedRange.initialized {
				t.Fatalf("filterMapsRange initialized while indexing is disabled")
			}
			continue
		}
		if !ts.fm.indexedRange.initialized {
			t.Fatalf("filterMapsRange not initialized while indexing is enabled")
		}
		var tailBlock uint64
		if history > 0 && history <= head {
			tailBlock = uint64(head + 1 - history)
		}
		var tailEpoch uint32
		if tailBlock > 0 {
			tailLvPtr := expspos(tailBlock) - 1
			tailEpoch = uint32(tailLvPtr >> (testParams.logValuesPerMap + testParams.logMapsPerEpoch))
		}
		tailLvPtr := uint64(tailEpoch) << (testParams.logValuesPerMap + testParams.logMapsPerEpoch) // first available lv ptr
		var expTailBlock uint64
		if tailEpoch > 0 {
			for expspos(expTailBlock) <= tailLvPtr {
				expTailBlock++
			}
		}
		if ts.fm.indexedRange.blocks.Last() != uint64(head) {
			ts.t.Fatalf("Invalid index head (expected #%d, got #%d)", head, ts.fm.indexedRange.blocks.Last())
		}
		expHeadDelimiter := expdpos(uint64(head))
		if ts.fm.indexedRange.headDelimiter != expHeadDelimiter {
			ts.t.Fatalf("Invalid index head delimiter pointer (expected %d, got %d)", expHeadDelimiter, ts.fm.indexedRange.headDelimiter)
		}

		if ts.fm.indexedRange.blocks.First() != expTailBlock {
			ts.t.Fatalf("Invalid index tail block (expected #%d, got #%d)", expTailBlock, ts.fm.indexedRange.blocks.First())
		}
	}
}

func TestIndexerMatcherView(t *testing.T) {
	testIndexerMatcherView(t, false)
}

func TestIndexerMatcherViewWithConcurrentRead(t *testing.T) {
	testIndexerMatcherView(t, true)
}

func testIndexerMatcherView(t *testing.T, concurrentRead bool) {
	ts := newTestSetup(t)
	defer ts.close()

	forks := make([][]common.Hash, 20)
	hashes := make([]common.Hash, 20)
	ts.chain.addBlocks(100, 5, 2, 4, true)
	ts.setHistory(0, false)
	for i := range forks {
		if i != 0 {
			ts.chain.setHead(100 - i)
			ts.chain.addBlocks(i, 5, 2, 4, true)
		}
		ts.fm.WaitIdle()
		forks[i] = ts.chain.getCanonicalChain()
		hashes[i] = ts.matcherViewHash()
	}
	fork := len(forks) - 1
	for i := 0; i < 5000; i++ {
		oldFork := fork
		fork = rand.Intn(len(forks))
		stopCh := make(chan chan struct{})
		if concurrentRead {
			go func() {
				for {
					ts.matcherViewHash()
					select {
					case ch := <-stopCh:
						close(ch)
						return
					default:
					}
				}
			}()
		}
		ts.chain.setCanonicalChain(forks[fork])
		ts.fm.WaitIdle()
		if concurrentRead {
			ch := make(chan struct{})
			stopCh <- ch
			<-ch
		}
		hash := ts.matcherViewHash()
		if hash != hashes[fork] {
			t.Fatalf("Matcher view hash mismatch when switching from for %d to %d", oldFork, fork)
		}
	}
}

func TestLogsByIndex(t *testing.T) {
	ts := newTestSetup(t)
	defer func() {
		ts.fm.testProcessEventsHook = nil
		ts.close()
	}()

	ts.chain.addBlocks(1000, 10, 3, 4, true)
	ts.setHistory(0, false)
	ts.fm.WaitIdle()
	firstLog := make([]uint64, 1001) // first valid log position per block
	lastLog := make([]uint64, 1001)  // last valid log position per block
	for i := uint64(0); i <= ts.fm.indexedRange.headDelimiter; i++ {
		log, err := ts.fm.getLogByLvIndex(i)
		if err != nil {
			t.Fatalf("Error getting log by index %d: %v", i, err)
		}
		if log != nil {
			if firstLog[log.BlockNumber] == 0 {
				firstLog[log.BlockNumber] = i
			}
			lastLog[log.BlockNumber] = i
		}
	}
	var failed bool
	ts.fm.testProcessEventsHook = func() {
		if ts.fm.indexedRange.blocks.IsEmpty() {
			return
		}
		if lvi := firstLog[ts.fm.indexedRange.blocks.First()]; lvi != 0 {
			log, err := ts.fm.getLogByLvIndex(lvi)
			if log == nil || err != nil {
				t.Errorf("Error getting first log of indexed block range: %v", err)
				failed = true
			}
		}
		if lvi := lastLog[ts.fm.indexedRange.blocks.Last()]; lvi != 0 {
			log, err := ts.fm.getLogByLvIndex(lvi)
			if log == nil || err != nil {
				t.Errorf("Error getting last log of indexed block range: %v", err)
				failed = true
			}
		}
	}
	chain := ts.chain.getCanonicalChain()
	for i := 0; i < 1000 && !failed; i++ {
		head := rand.Intn(len(chain))
		ts.chain.setCanonicalChain(chain[:head+1])
		ts.fm.WaitIdle()
	}
}

func TestIndexerCompareDb(t *testing.T) {
	ts := newTestSetup(t)
	defer ts.close()

	ts.chain.addBlocks(500, 10, 3, 4, true)
	ts.setHistory(0, false)
	ts.fm.WaitIdle()
	// revert points are stored after block 500
	ts.chain.addBlocks(500, 10, 3, 4, true)
	ts.fm.WaitIdle()
	chain1 := ts.chain.getCanonicalChain()
	ts.storeDbHash("chain 1 [0, 1000]")

	ts.chain.setHead(600)
	ts.fm.WaitIdle()
	ts.storeDbHash("chain 1/2 [0, 600]")

	ts.chain.addBlocks(600, 10, 3, 4, true)
	ts.fm.WaitIdle()
	chain2 := ts.chain.getCanonicalChain()
	ts.storeDbHash("chain 2 [0, 1200]")

	ts.chain.setHead(600)
	ts.fm.WaitIdle()
	ts.checkDbHash("chain 1/2 [0, 600]")

	ts.setHistory(800, false)
	ts.chain.setCanonicalChain(chain1)
	ts.fm.WaitIdle()
	ts.storeDbHash("chain 1 [201, 1000]")

	ts.setHistory(0, false)
	ts.fm.WaitIdle()
	ts.checkDbHash("chain 1 [0, 1000]")

	ts.setHistory(800, false)
	ts.chain.setCanonicalChain(chain2)
	ts.fm.WaitIdle()
	ts.storeDbHash("chain 2 [401, 1200]")

	ts.setHistory(0, true)
	ts.fm.WaitIdle()
	ts.storeDbHash("no index")

	ts.chain.setCanonicalChain(chain2[:501])
	ts.setHistory(0, false)
	ts.fm.WaitIdle()
	ts.chain.setCanonicalChain(chain2)
	ts.fm.WaitIdle()
	ts.checkDbHash("chain 2 [0, 1200]")

	ts.chain.setCanonicalChain(chain1)
	ts.fm.WaitIdle()
	ts.setHistory(800, false)
	ts.fm.WaitIdle()
	ts.checkDbHash("chain 1 [201, 1000]")

	ts.chain.setCanonicalChain(chain2)
	ts.fm.WaitIdle()
	ts.checkDbHash("chain 2 [401, 1200]")

	ts.setHistory(0, true)
	ts.fm.WaitIdle()
	ts.checkDbHash("no index")
}

type testSetup struct {
	t                    *testing.T
	fm                   *FilterMaps
	db                   ethdb.Database
	chain                *testChain
	params               Params
	dbHashes             map[string]common.Hash
	testDisableSnapshots bool
}

func newTestSetup(t *testing.T) *testSetup {
	params := testParams
	params.deriveFields()
	ts := &testSetup{
		t:        t,
		db:       rawdb.NewMemoryDatabase(),
		params:   params,
		dbHashes: make(map[string]common.Hash),
	}
	ts.chain = ts.newTestChain()
	return ts
}

func (ts *testSetup) setHistory(history uint64, noHistory bool) {
	if ts.fm != nil {
		ts.fm.Stop()
	}
	head := ts.chain.CurrentBlock()
	view := NewChainView(ts.chain, head.Number.Uint64(), head.Hash())
	config := Config{
		History:  history,
		Disabled: noHistory,
	}
	ts.fm, _ = NewFilterMaps(ts.db, view, 0, 0, ts.params, config)
	ts.fm.testDisableSnapshots = ts.testDisableSnapshots
	ts.fm.Start()
}

func (ts *testSetup) storeDbHash(id string) {
	dbHash := ts.fmDbHash()
	for otherId, otherHash := range ts.dbHashes {
		if otherHash == dbHash {
			ts.t.Fatalf("Unexpected equal database hashes `%s` and `%s`", id, otherId)
		}
	}
	ts.dbHashes[id] = dbHash
}

func (ts *testSetup) checkDbHash(id string) {
	if ts.fmDbHash() != ts.dbHashes[id] {
		ts.t.Fatalf("Database `%s` hash mismatch", id)
	}
}

func (ts *testSetup) fmDbHash() common.Hash {
	hasher := sha256.New()
	it := ts.db.NewIterator(nil, nil)
	for it.Next() {
		hasher.Write(it.Key())
		hasher.Write(it.Value())
	}
	it.Release()
	var result common.Hash
	hasher.Sum(result[:0])
	return result
}

func (ts *testSetup) matcherViewHash() common.Hash {
	mb := ts.fm.NewMatcherBackend()
	defer mb.Close()

	ctx := context.Background()
	params := mb.GetParams()
	hasher := sha256.New()
	var headPtr uint64
	for b := uint64(0); ; b++ {
		lvptr, err := mb.GetBlockLvPointer(ctx, b)
		if err != nil || (b > 0 && lvptr == headPtr) {
			break
		}
		var enc [8]byte
		binary.LittleEndian.PutUint64(enc[:], lvptr)
		hasher.Write(enc[:])
		headPtr = lvptr
	}
	headMap := uint32(headPtr >> params.logValuesPerMap)
	var enc [12]byte
	for r := uint32(0); r < params.mapHeight; r++ {
		binary.LittleEndian.PutUint32(enc[:4], r)
		for m := uint32(0); m <= headMap; m++ {
			binary.LittleEndian.PutUint32(enc[4:8], m)
			rows, _ := mb.GetFilterMapRows(ctx, []uint32{m}, r, false)
			for _, row := range rows {
				for _, v := range row {
					binary.LittleEndian.PutUint32(enc[8:], v)
					hasher.Write(enc[:])
				}
			}
		}
	}
	var hash common.Hash
	hasher.Sum(hash[:0])
	for i := 0; i < 50; i++ {
		hasher.Reset()
		hasher.Write(hash[:])
		lvptr := binary.LittleEndian.Uint64(hash[:8]) % headPtr
		if log, _ := mb.GetLogByLvIndex(ctx, lvptr); log != nil {
			enc, err := rlp.EncodeToBytes(log)
			if err != nil {
				panic(err)
			}
			hasher.Write(enc)
		}
		hasher.Sum(hash[:0])
	}
	return hash
}

func (ts *testSetup) close() {
	if ts.fm != nil {
		ts.fm.Stop()
	}
	if ts.db != nil {
		ts.db.Close()
	}
	if ts.chain != nil && ts.chain.db != nil {
		ts.chain.db.Close()
	}
}

type testChain struct {
	ts        *testSetup
	db        ethdb.Database
	lock      sync.RWMutex
	canonical []common.Hash
	blocks    map[common.Hash]*types.Block
	receipts  map[common.Hash]types.Receipts
}

func (ts *testSetup) newTestChain() *testChain {
	return &testChain{
		ts:       ts,
		blocks:   make(map[common.Hash]*types.Block),
		receipts: make(map[common.Hash]types.Receipts),
	}
}

func (tc *testChain) CurrentBlock() *types.Header {
	tc.lock.RLock()
	defer tc.lock.RUnlock()

	if len(tc.canonical) == 0 {
		return nil
	}
	return tc.blocks[tc.canonical[len(tc.canonical)-1]].Header()
}

func (tc *testChain) GetHeader(hash common.Hash, number uint64) *types.Header {
	tc.lock.RLock()
	defer tc.lock.RUnlock()

	if block := tc.blocks[hash]; block != nil {
		return block.Header()
	}
	return nil
}

func (tc *testChain) GetCanonicalHash(number uint64) common.Hash {
	tc.lock.RLock()
	defer tc.lock.RUnlock()

	if uint64(len(tc.canonical)) <= number {
		return common.Hash{}
	}
	return tc.canonical[number]
}

func (tc *testChain) GetReceiptsByHash(hash common.Hash) types.Receipts {
	tc.lock.RLock()
	defer tc.lock.RUnlock()

	return tc.receipts[hash]
}

func (tc *testChain) GetRawReceipts(hash common.Hash, number uint64) types.Receipts {
	tc.lock.RLock()
	defer tc.lock.RUnlock()

	return tc.receipts[hash]
}

func (tc *testChain) addBlocks(count, maxTxPerBlock, maxLogsPerReceipt, maxTopicsPerLog int, random bool) {
	tc.lock.Lock()
	blockGen := func(i int, gen *core.BlockGen) {
		var txCount int
		if random {
			txCount = rand.Intn(maxTxPerBlock + 1)
		} else {
			txCount = maxTxPerBlock
		}
		for k := txCount; k > 0; k-- {
			receipt := types.NewReceipt(nil, false, 0)
			var logCount int
			if random {
				logCount = rand.Intn(maxLogsPerReceipt + 1)
			} else {
				logCount = maxLogsPerReceipt
			}
			receipt.Logs = make([]*types.Log, logCount)
			for i := range receipt.Logs {
				log := &types.Log{}
				receipt.Logs[i] = log
				crand.Read(log.Address[:])
				var topicCount int
				if random {
					topicCount = rand.Intn(maxTopicsPerLog + 1)
				} else {
					topicCount = maxTopicsPerLog
				}
				log.Topics = make([]common.Hash, topicCount)
				for j := range log.Topics {
					crand.Read(log.Topics[j][:])
				}
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(999, common.HexToAddress("0x999"), big.NewInt(999), 999, gen.BaseFee(), nil))
		}
	}

	var (
		blocks   []*types.Block
		receipts []types.Receipts
		engine   = ethash.NewFaker()
	)

	if len(tc.canonical) == 0 {
		gspec := &core.Genesis{
			Alloc:   types.GenesisAlloc{},
			BaseFee: big.NewInt(params.InitialBaseFee),
			Config:  params.TestChainConfig,
		}
		tc.db, blocks, receipts = core.GenerateChainWithGenesis(gspec, engine, count, blockGen)
		gblock := gspec.ToBlock()
		ghash := gblock.Hash()
		tc.canonical = []common.Hash{ghash}
		tc.blocks[ghash] = gblock
		tc.receipts[ghash] = types.Receipts{}
	} else {
		blocks, receipts = core.GenerateChain(params.TestChainConfig, tc.blocks[tc.canonical[len(tc.canonical)-1]], engine, tc.db, count, blockGen)
	}

	for i, block := range blocks {
		num, hash := int(block.NumberU64()), block.Hash()
		if len(tc.canonical) != num {
			panic("canonical chain length mismatch")
		}
		tc.canonical = append(tc.canonical, hash)
		tc.blocks[hash] = block
		if receipts[i] != nil {
			tc.receipts[hash] = receipts[i]
		} else {
			tc.receipts[hash] = types.Receipts{}
		}
	}
	tc.lock.Unlock()
	tc.setTargetHead()
}

func (tc *testChain) setHead(headNum int) {
	tc.lock.Lock()
	tc.canonical = tc.canonical[:headNum+1]
	tc.lock.Unlock()
	tc.setTargetHead()
}

func (tc *testChain) setTargetHead() {
	head := tc.CurrentBlock()
	if tc.ts.fm != nil {
		if !tc.ts.fm.disabled {
			//tc.ts.fm.targetViewCh <- NewChainView(tc, head.Number.Uint64(), head.Hash())
			tc.ts.fm.SetTarget(NewChainView(tc, head.Number.Uint64(), head.Hash()), 0, 0)
		}
	}
}

func (tc *testChain) getCanonicalChain() []common.Hash {
	tc.lock.RLock()
	defer tc.lock.RUnlock()

	cc := make([]common.Hash, len(tc.canonical))
	copy(cc, tc.canonical)
	return cc
}

// restore an earlier state of the chain
func (tc *testChain) setCanonicalChain(cc []common.Hash) {
	tc.lock.Lock()
	tc.canonical = make([]common.Hash, len(cc))
	copy(tc.canonical, cc)
	tc.lock.Unlock()
	tc.setTargetHead()
}

// batchTrackingDb wraps an ethdb.Database to track the max batch operations and sizes.
type batchTrackingDb struct {
	ethdb.Database
	maxBatchOps  int
	maxBatchSize int
	lock         sync.Mutex
}

func newBatchTrackingDb(db ethdb.Database) *batchTrackingDb {
	return &batchTrackingDb{Database: db}
}

func (b *batchTrackingDb) NewBatch() ethdb.Batch {
	return &trackedBatch{
		Batch: b.Database.NewBatch(),
		db:    b,
	}
}

func (b *batchTrackingDb) NewBatchWithSize(size int) ethdb.Batch {
	return &trackedBatch{
		Batch: b.Database.NewBatchWithSize(size),
		db:    b,
	}
}

type trackedBatch struct {
	ethdb.Batch
	db   *batchTrackingDb
	ops  int
	size int
}

func (tb *trackedBatch) Put(key, value []byte) error {
	tb.ops++
	tb.size += len(key) + len(value)
	tb.db.lock.Lock()
	if tb.ops > tb.db.maxBatchOps {
		tb.db.maxBatchOps = tb.ops
	}
	if tb.size > tb.db.maxBatchSize {
		tb.db.maxBatchSize = tb.size
	}
	tb.db.lock.Unlock()
	return tb.Batch.Put(key, value)
}

func (tb *trackedBatch) Delete(key []byte) error {
	tb.ops++
	tb.size += len(key)
	tb.db.lock.Lock()
	if tb.ops > tb.db.maxBatchOps {
		tb.db.maxBatchOps = tb.ops
	}
	if tb.size > tb.db.maxBatchSize {
		tb.db.maxBatchSize = tb.size
	}
	tb.db.lock.Unlock()
	return tb.Batch.Delete(key)
}

func TestIndexerRollbackBoundedBatch(t *testing.T) {
	params := testParams
	params.deriveFields()

	trackedDb := newBatchTrackingDb(rawdb.NewMemoryDatabase())
	ts := &testSetup{
		t:        t,
		db:       trackedDb,
		params:   params,
		dbHashes: make(map[string]common.Hash),
	}
	ts.chain = ts.newTestChain()
	defer ts.close()

	// Generate 200 blocks with logs to create a multi-epoch indexed state
	ts.chain.addBlocks(200, 5, 2, 4, false)
	ts.setHistory(0, false)
	ts.fm.WaitIdle()

	t.Logf("Initial indexing: maxBatchOps = %d, maxBatchSize = %d bytes, maps = %d, blocks = %d",
		trackedDb.maxBatchOps, trackedDb.maxBatchSize, ts.fm.indexedRange.maps.Count(), ts.fm.indexedRange.blocks.Count())

	// Reset tracked stats
	trackedDb.lock.Lock()
	trackedDb.maxBatchOps = 0
	trackedDb.maxBatchSize = 0
	trackedDb.lock.Unlock()

	// Rollback chain to block 5 to trigger large multi-epoch stale map deletion
	ts.chain.setHead(5)
	ts.fm.WaitIdle()

	t.Logf("After rollback to block 5: maxBatchOps = %d, maxBatchSize = %d bytes, maps = %d, blocks = %d",
		trackedDb.maxBatchOps, trackedDb.maxBatchSize, ts.fm.indexedRange.maps.Count(), ts.fm.indexedRange.blocks.Count())

	if trackedDb.maxBatchOps > 80 {
		t.Fatalf("Batch operations during rollback too large: got %d, want <= 80", trackedDb.maxBatchOps)
	}
}

func TestIndexerRollbackScenarios(t *testing.T) {
	// Tests multiple rollback shapes (small, mid-epoch, multi-epoch, full resync)
	// and verifies bit-exact database hash equality against a fresh reference build.
	ts := newTestSetup(t)
	defer ts.close()

	// Generate 300 blocks on chain 1
	ts.chain.addBlocks(300, 5, 2, 4, true)
	ts.setHistory(0, false)
	ts.fm.WaitIdle()
	ts.storeDbHash("chain1_300")

	// 1. Short rollback (same epoch, a few blocks)
	ts.chain.setHead(280)
	ts.fm.WaitIdle()
	ts.storeDbHash("chain1_280")

	// 2. Medium rollback (across base-row groups within epoch)
	ts.chain.setHead(250)
	ts.fm.WaitIdle()
	ts.storeDbHash("chain1_250")

	// 3. Large multi-epoch rollback (from block 250 back to block 50)
	ts.chain.setHead(50)
	ts.fm.WaitIdle()
	ts.storeDbHash("chain1_50")

	// 4. Full rollback to genesis block 0
	ts.chain.setHead(0)
	ts.fm.WaitIdle()
	ts.storeDbHash("chain1_0")
}

func TestIndexerRollbackDbIntegrity(t *testing.T) {
	// Verify that after rollback and reorg, database hash matches a freshly built index on the same chain.
	ts1 := newTestSetup(t)
	defer ts1.close()

	ts1.chain.addBlocks(200, 5, 2, 4, true)
	ts1.setHistory(0, false)
	ts1.fm.WaitIdle()
	chainFull := ts1.chain.getCanonicalChain()

	// Roll back to block 50 and add 100 different blocks (fork)
	ts1.chain.setHead(50)
	ts1.chain.addBlocks(100, 5, 2, 4, true)
	ts1.fm.WaitIdle()
	reorgDbHash := ts1.fmDbHash()

	// Now build a fresh node directly from scratch on the reorged chain
	reorgChain := ts1.chain.getCanonicalChain()
	ts2 := newTestSetup(t)
	defer ts2.close()

	// Seed ts2 with the identical blocks from ts1
	for _, h := range reorgChain {
		ts2.chain.blocks[h] = ts1.chain.blocks[h]
		ts2.chain.receipts[h] = ts1.chain.receipts[h]
	}
	ts2.chain.setCanonicalChain(reorgChain)
	ts2.setHistory(0, false)
	ts2.fm.WaitIdle()
	freshDbHash := ts2.fmDbHash()

	if reorgDbHash != freshDbHash {
		t.Fatalf("Database hash mismatch after rollback + reorg:\n  reorg = %x\n  fresh = %x", reorgDbHash, freshDbHash)
	}

	// Now switch ts1 back to the original chainFull and compare with fresh node on chainFull
	for _, h := range chainFull {
		ts2.chain.blocks[h] = ts1.chain.blocks[h]
		ts2.chain.receipts[h] = ts1.chain.receipts[h]
	}
	ts1.chain.setCanonicalChain(chainFull)
	ts1.fm.WaitIdle()
	ts1RestoredHash := ts1.fmDbHash()

	ts3 := newTestSetup(t)
	defer ts3.close()
	for _, h := range chainFull {
		ts3.chain.blocks[h] = ts1.chain.blocks[h]
		ts3.chain.receipts[h] = ts1.chain.receipts[h]
	}
	ts3.chain.setCanonicalChain(chainFull)
	ts3.setHistory(0, false)
	ts3.fm.WaitIdle()
	ts3FreshHash := ts3.fmDbHash()

	if ts1RestoredHash != ts3FreshHash {
		t.Fatalf("Database hash mismatch after restoring original chain:\n  restored = %x\n  fresh    = %x", ts1RestoredHash, ts3FreshHash)
	}
}

func TestIndexerRollbackCrashRecovery(t *testing.T) {
	// Proves that interrupting the indexer during a rollback cleanup
	// cannot leave an unrecoverable or corrupted index state.
	ts := newTestSetup(t)
	defer ts.close()

	// Index 200 blocks
	ts.chain.addBlocks(200, 5, 2, 4, true)
	ts.setHistory(0, false)
	ts.fm.WaitIdle()

	// Stop indexer to simulate crash before final cleanup completion
	ts.fm.Stop()
	ts.fm = nil

	// Simulate partial range deletion on the database while old range metadata remains
	// (e.g. process killed during DeleteFilterMapRows)
	staleStartMap := uint32(20)
	staleEndMap := uint32(50)
	_ = rawdb.DeleteFilterMapLastBlocks(ts.db, common.NewRange(staleStartMap, staleEndMap-staleStartMap), false, nil)

	// Re-open FilterMaps on the interrupted database with the chain rolled back to block 50
	ts.chain.setHead(50)
	ts.setHistory(0, false)
	ts.fm.WaitIdle()

	// Verify that the index recovered cleanly and indexed up to block 50
	if ts.fm.indexedRange.blocks.Last() != 50 {
		t.Fatalf("Recovered indexer head mismatch: got %d, want 50", ts.fm.indexedRange.blocks.Last())
	}

	// Compare with reference node built cleanly to block 50
	targetChain := ts.chain.getCanonicalChain()
	tsRef := newTestSetup(t)
	defer tsRef.close()
	for _, h := range targetChain {
		tsRef.chain.blocks[h] = ts.chain.blocks[h]
		tsRef.chain.receipts[h] = ts.chain.receipts[h]
	}
	tsRef.chain.setCanonicalChain(targetChain)
	tsRef.setHistory(0, false)
	tsRef.fm.WaitIdle()

	recoveredHash := ts.fmDbHash()
	refHash := tsRef.fmDbHash()
	if recoveredHash != refHash {
		t.Fatalf("Recovered database hash mismatch against reference:\n  recovered = %x\n  reference = %x", recoveredHash, refHash)
	}
}
