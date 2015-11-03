// Copyright 2015 The go-ethereum Authors
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

package fetcher

import (
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
)

var (
	testdb, _    = ethdb.NewMemDatabase()
	testKey, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddress  = crypto.PubkeyToAddress(testKey.PublicKey)
	genesis      = core.GenesisBlockForTesting(testdb, testAddress, big.NewInt(1000000000))
	unknownBlock = types.NewBlock(&types.Header{GasLimit: params.GenesisGasLimit}, nil, nil, nil)
)

// makeChain creates a chain of n blocks starting at and including parent.
// the returned hash chain is ordered head->parent. In addition, every 3rd block
// contains a transaction and every 5th an uncle to allow testing correct block
// reassembly.
func makeChain(n int, seed byte, parent *types.Block) ([]common.Hash, map[common.Hash]*types.Block) {
	blocks, _ := core.GenerateChain(parent, testdb, n, func(i int, block *core.BlockGen) {
		block.SetCoinbase(common.Address{seed})

		// If the block number is multiple of 3, send a bonus transaction to the miner
		if parent == genesis && i%3 == 0 {
			tx, err := types.NewTransaction(block.TxNonce(testAddress), common.Address{seed}, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(testKey)
			if err != nil {
				panic(err)
			}
			block.AddTx(tx)
		}
		// If the block number is a multiple of 5, add a bonus uncle to the block
		if i%5 == 0 {
			block.AddUncle(&types.Header{ParentHash: block.PrevBlock(i - 1).Hash(), Number: big.NewInt(int64(i - 1))})
		}
	})
	hashes := make([]common.Hash, n+1)
	hashes[len(hashes)-1] = parent.Hash()
	blockm := make(map[common.Hash]*types.Block, n+1)
	blockm[parent.Hash()] = parent
	for i, b := range blocks {
		hashes[len(hashes)-i-2] = b.Hash()
		blockm[b.Hash()] = b
	}
	return hashes, blockm
}

// fetcherTester is a test simulator for mocking out local block chain.
type fetcherTester struct {
	fetcher *Fetcher

	hashes []common.Hash                // Hash chain belonging to the tester
	blocks map[common.Hash]*types.Block // Blocks belonging to the tester
	drops  map[string]bool              // Map of peers dropped by the fetcher

	lock sync.RWMutex
}

// newTester creates a new fetcher test mocker.
func newTester() *fetcherTester {
	tester := &fetcherTester{
		hashes: []common.Hash{genesis.Hash()},
		blocks: map[common.Hash]*types.Block{genesis.Hash(): genesis},
		drops:  make(map[string]bool),
	}
	tester.fetcher = New(tester.getBlock, tester.verifyBlock, tester.broadcastBlock, tester.chainHeight, tester.insertChain, tester.dropPeer)
	tester.fetcher.Start()

	return tester
}

// getBlock retrieves a block from the tester's block chain.
func (f *fetcherTester) getBlock(hash common.Hash) *types.Block {
	f.lock.RLock()
	defer f.lock.RUnlock()

	return f.blocks[hash]
}

// verifyBlock is a nop placeholder for the block header verification.
func (f *fetcherTester) verifyBlock(block *types.Block, parent *types.Block) error {
	return nil
}

// broadcastBlock is a nop placeholder for the block broadcasting.
func (f *fetcherTester) broadcastBlock(block *types.Block, propagate bool) {
}

// chainHeight retrieves the current height (block number) of the chain.
func (f *fetcherTester) chainHeight() uint64 {
	f.lock.RLock()
	defer f.lock.RUnlock()

	return f.blocks[f.hashes[len(f.hashes)-1]].NumberU64()
}

// insertChain injects a new blocks into the simulated chain.
func (f *fetcherTester) insertChain(blocks types.Blocks) (int, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	for i, block := range blocks {
		// Make sure the parent in known
		if _, ok := f.blocks[block.ParentHash()]; !ok {
			return i, errors.New("unknown parent")
		}
		// Discard any new blocks if the same height already exists
		if block.NumberU64() <= f.blocks[f.hashes[len(f.hashes)-1]].NumberU64() {
			return i, nil
		}
		// Otherwise build our current chain
		f.hashes = append(f.hashes, block.Hash())
		f.blocks[block.Hash()] = block
	}
	return 0, nil
}

// dropPeer is an emulator for the peer removal, simply accumulating the various
// peers dropped by the fetcher.
func (f *fetcherTester) dropPeer(peer string) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.drops[peer] = true
}

// makeBlockFetcher retrieves a block fetcher associated with a simulated peer.
func (f *fetcherTester) makeBlockFetcher(blocks map[common.Hash]*types.Block) blockRequesterFn {
	closure := make(map[common.Hash]*types.Block)
	for hash, block := range blocks {
		closure[hash] = block
	}
	// Create a function that returns blocks from the closure
	return func(hashes []common.Hash) error {
		// Gather the blocks to return
		blocks := make([]*types.Block, 0, len(hashes))
		for _, hash := range hashes {
			if block, ok := closure[hash]; ok {
				blocks = append(blocks, block)
			}
		}
		// Return on a new thread
		go f.fetcher.FilterBlocks(blocks)

		return nil
	}
}

// makeHeaderFetcher retrieves a block header fetcher associated with a simulated peer.
func (f *fetcherTester) makeHeaderFetcher(blocks map[common.Hash]*types.Block, drift time.Duration) headerRequesterFn {
	closure := make(map[common.Hash]*types.Block)
	for hash, block := range blocks {
		closure[hash] = block
	}
	// Create a function that return a header from the closure
	return func(hash common.Hash) error {
		// Gather the blocks to return
		headers := make([]*types.Header, 0, 1)
		if block, ok := closure[hash]; ok {
			headers = append(headers, block.Header())
		}
		// Return on a new thread
		go f.fetcher.FilterHeaders(headers, time.Now().Add(drift))

		return nil
	}
}

// makeBodyFetcher retrieves a block body fetcher associated with a simulated peer.
func (f *fetcherTester) makeBodyFetcher(blocks map[common.Hash]*types.Block, drift time.Duration) bodyRequesterFn {
	closure := make(map[common.Hash]*types.Block)
	for hash, block := range blocks {
		closure[hash] = block
	}
	// Create a function that returns blocks from the closure
	return func(hashes []common.Hash) error {
		// Gather the block bodies to return
		transactions := make([][]*types.Transaction, 0, len(hashes))
		uncles := make([][]*types.Header, 0, len(hashes))

		for _, hash := range hashes {
			if block, ok := closure[hash]; ok {
				transactions = append(transactions, block.Transactions())
				uncles = append(uncles, block.Uncles())
			}
		}
		// Return on a new thread
		go f.fetcher.FilterBodies(transactions, uncles, time.Now().Add(drift))

		return nil
	}
}

// verifyFetchingEvent verifies that one single event arrive on an fetching channel.
func verifyFetchingEvent(t *testing.T, fetching chan []common.Hash, arrive bool) {
	if arrive {
		select {
		case <-fetching:
		case <-time.After(time.Second):
			t.Fatalf("fetching timeout")
		}
	} else {
		select {
		case <-fetching:
			t.Fatalf("fetching invoked")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// verifyCompletingEvent verifies that one single event arrive on an completing channel.
func verifyCompletingEvent(t *testing.T, completing chan []common.Hash, arrive bool) {
	if arrive {
		select {
		case <-completing:
		case <-time.After(time.Second):
			t.Fatalf("completing timeout")
		}
	} else {
		select {
		case <-completing:
			t.Fatalf("completing invoked")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// verifyImportEvent verifies that one single event arrive on an import channel.
func verifyImportEvent(t *testing.T, imported chan *types.Block, arrive bool) {
	if arrive {
		select {
		case <-imported:
		case <-time.After(time.Second):
			t.Fatalf("import timeout")
		}
	} else {
		select {
		case <-imported:
			t.Fatalf("import invoked")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// verifyImportCount verifies that exactly count number of events arrive on an
// import hook channel.
func verifyImportCount(t *testing.T, imported chan *types.Block, count int) {
	for i := 0; i < count; i++ {
		select {
		case <-imported:
		case <-time.After(time.Second):
			t.Fatalf("block %d: import timeout", i+1)
		}
	}
	verifyImportDone(t, imported)
}

// verifyImportDone verifies that no more events are arriving on an import channel.
func verifyImportDone(t *testing.T, imported chan *types.Block) {
	select {
	case <-imported:
		t.Fatalf("extra block imported")
	case <-time.After(50 * time.Millisecond):
	}
}

// Tests that a fetcher accepts block announcements and initiates retrievals for
// them, successfully importing into the local chain.
func TestSequentialAnnouncements61(t *testing.T) { testSequentialAnnouncements(t, 61) }
func TestSequentialAnnouncements62(t *testing.T) { testSequentialAnnouncements(t, 62) }
func TestSequentialAnnouncements63(t *testing.T) { testSequentialAnnouncements(t, 63) }
func TestSequentialAnnouncements64(t *testing.T) { testSequentialAnnouncements(t, 64) }

func testSequentialAnnouncements(t *testing.T, protocol int) {
	// Create a chain of blocks to import
	targetBlocks := 4 * hashLimit
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	tester := newTester()
	blockFetcher := tester.makeBlockFetcher(blocks)
	headerFetcher := tester.makeHeaderFetcher(blocks, -gatherSlack)
	bodyFetcher := tester.makeBodyFetcher(blocks, 0)

	// Iteratively announce blocks until all are imported
	imported := make(chan *types.Block)
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

	for i := len(hashes) - 2; i >= 0; i-- {
		if protocol < 62 {
			tester.fetcher.Notify("valid", hashes[i], 0, time.Now().Add(-arriveTimeout), blockFetcher, nil, nil)
		} else {
			tester.fetcher.Notify("valid", hashes[i], uint64(len(hashes)-i-1), time.Now().Add(-arriveTimeout), nil, headerFetcher, bodyFetcher)
		}
		verifyImportEvent(t, imported, true)
	}
	verifyImportDone(t, imported)
}

// Tests that if blocks are announced by multiple peers (or even the same buggy
// peer), they will only get downloaded at most once.
func TestConcurrentAnnouncements61(t *testing.T) { testConcurrentAnnouncements(t, 61) }
func TestConcurrentAnnouncements62(t *testing.T) { testConcurrentAnnouncements(t, 62) }
func TestConcurrentAnnouncements63(t *testing.T) { testConcurrentAnnouncements(t, 63) }
func TestConcurrentAnnouncements64(t *testing.T) { testConcurrentAnnouncements(t, 64) }

func testConcurrentAnnouncements(t *testing.T, protocol int) {
	// Create a chain of blocks to import
	targetBlocks := 4 * hashLimit
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	// Assemble a tester with a built in counter for the requests
	tester := newTester()
	blockFetcher := tester.makeBlockFetcher(blocks)
	headerFetcher := tester.makeHeaderFetcher(blocks, -gatherSlack)
	bodyFetcher := tester.makeBodyFetcher(blocks, 0)

	counter := uint32(0)
	blockWrapper := func(hashes []common.Hash) error {
		atomic.AddUint32(&counter, uint32(len(hashes)))
		return blockFetcher(hashes)
	}
	headerWrapper := func(hash common.Hash) error {
		atomic.AddUint32(&counter, 1)
		return headerFetcher(hash)
	}
	// Iteratively announce blocks until all are imported
	imported := make(chan *types.Block)
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

	for i := len(hashes) - 2; i >= 0; i-- {
		if protocol < 62 {
			tester.fetcher.Notify("first", hashes[i], 0, time.Now().Add(-arriveTimeout), blockWrapper, nil, nil)
			tester.fetcher.Notify("second", hashes[i], 0, time.Now().Add(-arriveTimeout+time.Millisecond), blockWrapper, nil, nil)
			tester.fetcher.Notify("second", hashes[i], 0, time.Now().Add(-arriveTimeout-time.Millisecond), blockWrapper, nil, nil)
		} else {
			tester.fetcher.Notify("first", hashes[i], uint64(len(hashes)-i-1), time.Now().Add(-arriveTimeout), nil, headerWrapper, bodyFetcher)
			tester.fetcher.Notify("second", hashes[i], uint64(len(hashes)-i-1), time.Now().Add(-arriveTimeout+time.Millisecond), nil, headerWrapper, bodyFetcher)
			tester.fetcher.Notify("second", hashes[i], uint64(len(hashes)-i-1), time.Now().Add(-arriveTimeout-time.Millisecond), nil, headerWrapper, bodyFetcher)
		}
		verifyImportEvent(t, imported, true)
	}
	verifyImportDone(t, imported)

	// Make sure no blocks were retrieved twice
	if int(counter) != targetBlocks {
		t.Fatalf("retrieval count mismatch: have %v, want %v", counter, targetBlocks)
	}
}

// Tests that announcements arriving while a previous is being fetched still
// results in a valid import.
func TestOverlappingAnnouncements61(t *testing.T) { testOverlappingAnnouncements(t, 61) }
func TestOverlappingAnnouncements62(t *testing.T) { testOverlappingAnnouncements(t, 62) }
func TestOverlappingAnnouncements63(t *testing.T) { testOverlappingAnnouncements(t, 63) }
func TestOverlappingAnnouncements64(t *testing.T) { testOverlappingAnnouncements(t, 64) }

func testOverlappingAnnouncements(t *testing.T, protocol int) {
	// Create a chain of blocks to import
	targetBlocks := 4 * hashLimit
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	tester := newTester()
	blockFetcher := tester.makeBlockFetcher(blocks)
	headerFetcher := tester.makeHeaderFetcher(blocks, -gatherSlack)
	bodyFetcher := tester.makeBodyFetcher(blocks, 0)

	// Iteratively announce blocks, but overlap them continuously
	overlap := 16
	imported := make(chan *types.Block, len(hashes)-1)
	for i := 0; i < overlap; i++ {
		imported <- nil
	}
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

	for i := len(hashes) - 2; i >= 0; i-- {
		if protocol < 62 {
			tester.fetcher.Notify("valid", hashes[i], 0, time.Now().Add(-arriveTimeout), blockFetcher, nil, nil)
		} else {
			tester.fetcher.Notify("valid", hashes[i], uint64(len(hashes)-i-1), time.Now().Add(-arriveTimeout), nil, headerFetcher, bodyFetcher)
		}
		select {
		case <-imported:
		case <-time.After(time.Second):
			t.Fatalf("block %d: import timeout", len(hashes)-i)
		}
	}
	// Wait for all the imports to complete and check count
	verifyImportCount(t, imported, overlap)
}

// Tests that announces already being retrieved will not be duplicated.
func TestPendingDeduplication61(t *testing.T) { testPendingDeduplication(t, 61) }
func TestPendingDeduplication62(t *testing.T) { testPendingDeduplication(t, 62) }
func TestPendingDeduplication63(t *testing.T) { testPendingDeduplication(t, 63) }
func TestPendingDeduplication64(t *testing.T) { testPendingDeduplication(t, 64) }

func testPendingDeduplication(t *testing.T, protocol int) {
	// Create a hash and corresponding block
	hashes, blocks := makeChain(1, 0, genesis)

	// Assemble a tester with a built in counter and delayed fetcher
	tester := newTester()
	blockFetcher := tester.makeBlockFetcher(blocks)
	headerFetcher := tester.makeHeaderFetcher(blocks, -gatherSlack)
	bodyFetcher := tester.makeBodyFetcher(blocks, 0)

	delay := 50 * time.Millisecond
	counter := uint32(0)
	blockWrapper := func(hashes []common.Hash) error {
		atomic.AddUint32(&counter, uint32(len(hashes)))

		// Simulate a long running fetch
		go func() {
			time.Sleep(delay)
			blockFetcher(hashes)
		}()
		return nil
	}
	headerWrapper := func(hash common.Hash) error {
		atomic.AddUint32(&counter, 1)

		// Simulate a long running fetch
		go func() {
			time.Sleep(delay)
			headerFetcher(hash)
		}()
		return nil
	}
	// Announce the same block many times until it's fetched (wait for any pending ops)
	for tester.getBlock(hashes[0]) == nil {
		if protocol < 62 {
			tester.fetcher.Notify("repeater", hashes[0], 0, time.Now().Add(-arriveTimeout), blockWrapper, nil, nil)
		} else {
			tester.fetcher.Notify("repeater", hashes[0], 1, time.Now().Add(-arriveTimeout), nil, headerWrapper, bodyFetcher)
		}
		time.Sleep(time.Millisecond)
	}
	time.Sleep(delay)

	// Check that all blocks were imported and none fetched twice
	if imported := len(tester.blocks); imported != 2 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, 2)
	}
	if int(counter) != 1 {
		t.Fatalf("retrieval count mismatch: have %v, want %v", counter, 1)
	}
}

// Tests that announcements retrieved in a random order are cached and eventually
// imported when all the gaps are filled in.
func TestRandomArrivalImport61(t *testing.T) { testRandomArrivalImport(t, 61) }
func TestRandomArrivalImport62(t *testing.T) { testRandomArrivalImport(t, 62) }
func TestRandomArrivalImport63(t *testing.T) { testRandomArrivalImport(t, 63) }
func TestRandomArrivalImport64(t *testing.T) { testRandomArrivalImport(t, 64) }

func testRandomArrivalImport(t *testing.T, protocol int) {
	// Create a chain of blocks to import, and choose one to delay
	targetBlocks := maxQueueDist
	hashes, blocks := makeChain(targetBlocks, 0, genesis)
	skip := targetBlocks / 2

	tester := newTester()
	blockFetcher := tester.makeBlockFetcher(blocks)
	headerFetcher := tester.makeHeaderFetcher(blocks, -gatherSlack)
	bodyFetcher := tester.makeBodyFetcher(blocks, 0)

	// Iteratively announce blocks, skipping one entry
	imported := make(chan *types.Block, len(hashes)-1)
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

	for i := len(hashes) - 1; i >= 0; i-- {
		if i != skip {
			if protocol < 62 {
				tester.fetcher.Notify("valid", hashes[i], 0, time.Now().Add(-arriveTimeout), blockFetcher, nil, nil)
			} else {
				tester.fetcher.Notify("valid", hashes[i], uint64(len(hashes)-i-1), time.Now().Add(-arriveTimeout), nil, headerFetcher, bodyFetcher)
			}
			time.Sleep(time.Millisecond)
		}
	}
	// Finally announce the skipped entry and check full import
	if protocol < 62 {
		tester.fetcher.Notify("valid", hashes[skip], 0, time.Now().Add(-arriveTimeout), blockFetcher, nil, nil)
	} else {
		tester.fetcher.Notify("valid", hashes[skip], uint64(len(hashes)-skip-1), time.Now().Add(-arriveTimeout), nil, headerFetcher, bodyFetcher)
	}
	verifyImportCount(t, imported, len(hashes)-1)
}

// Tests that direct block enqueues (due to block propagation vs. hash announce)
// are correctly schedule, filling and import queue gaps.
func TestQueueGapFill61(t *testing.T) { testQueueGapFill(t, 61) }
func TestQueueGapFill62(t *testing.T) { testQueueGapFill(t, 62) }
func TestQueueGapFill63(t *testing.T) { testQueueGapFill(t, 63) }
func TestQueueGapFill64(t *testing.T) { testQueueGapFill(t, 64) }

func testQueueGapFill(t *testing.T, protocol int) {
	// Create a chain of blocks to import, and choose one to not announce at all
	targetBlocks := maxQueueDist
	hashes, blocks := makeChain(targetBlocks, 0, genesis)
	skip := targetBlocks / 2

	tester := newTester()
	blockFetcher := tester.makeBlockFetcher(blocks)
	headerFetcher := tester.makeHeaderFetcher(blocks, -gatherSlack)
	bodyFetcher := tester.makeBodyFetcher(blocks, 0)

	// Iteratively announce blocks, skipping one entry
	imported := make(chan *types.Block, len(hashes)-1)
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

	for i := len(hashes) - 1; i >= 0; i-- {
		if i != skip {
			if protocol < 62 {
				tester.fetcher.Notify("valid", hashes[i], 0, time.Now().Add(-arriveTimeout), blockFetcher, nil, nil)
			} else {
				tester.fetcher.Notify("valid", hashes[i], uint64(len(hashes)-i-1), time.Now().Add(-arriveTimeout), nil, headerFetcher, bodyFetcher)
			}
			time.Sleep(time.Millisecond)
		}
	}
	// Fill the missing block directly as if propagated
	tester.fetcher.Enqueue("valid", blocks[hashes[skip]])
	verifyImportCount(t, imported, len(hashes)-1)
}

// Tests that blocks arriving from various sources (multiple propagations, hash
// announces, etc) do not get scheduled for import multiple times.
func TestImportDeduplication61(t *testing.T) { testImportDeduplication(t, 61) }
func TestImportDeduplication62(t *testing.T) { testImportDeduplication(t, 62) }
func TestImportDeduplication63(t *testing.T) { testImportDeduplication(t, 63) }
func TestImportDeduplication64(t *testing.T) { testImportDeduplication(t, 64) }

func testImportDeduplication(t *testing.T, protocol int) {
	// Create two blocks to import (one for duplication, the other for stalling)
	hashes, blocks := makeChain(2, 0, genesis)

	// Create the tester and wrap the importer with a counter
	tester := newTester()
	blockFetcher := tester.makeBlockFetcher(blocks)
	headerFetcher := tester.makeHeaderFetcher(blocks, -gatherSlack)
	bodyFetcher := tester.makeBodyFetcher(blocks, 0)

	counter := uint32(0)
	tester.fetcher.insertChain = func(blocks types.Blocks) (int, error) {
		atomic.AddUint32(&counter, uint32(len(blocks)))
		return tester.insertChain(blocks)
	}
	// Instrument the fetching and imported events
	fetching := make(chan []common.Hash)
	imported := make(chan *types.Block, len(hashes)-1)
	tester.fetcher.fetchingHook = func(hashes []common.Hash) { fetching <- hashes }
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

	// Announce the duplicating block, wait for retrieval, and also propagate directly
	if protocol < 62 {
		tester.fetcher.Notify("valid", hashes[0], 0, time.Now().Add(-arriveTimeout), blockFetcher, nil, nil)
	} else {
		tester.fetcher.Notify("valid", hashes[0], 1, time.Now().Add(-arriveTimeout), nil, headerFetcher, bodyFetcher)
	}
	<-fetching

	tester.fetcher.Enqueue("valid", blocks[hashes[0]])
	tester.fetcher.Enqueue("valid", blocks[hashes[0]])
	tester.fetcher.Enqueue("valid", blocks[hashes[0]])

	// Fill the missing block directly as if propagated, and check import uniqueness
	tester.fetcher.Enqueue("valid", blocks[hashes[1]])
	verifyImportCount(t, imported, 2)

	if counter != 2 {
		t.Fatalf("import invocation count mismatch: have %v, want %v", counter, 2)
	}
}

// Tests that blocks with numbers much lower or higher than out current head get
// discarded to prevent wasting resources on useless blocks from faulty peers.
func TestDistantPropagationDiscarding(t *testing.T) {
	// Create a long chain to import and define the discard boundaries
	hashes, blocks := makeChain(3*maxQueueDist, 0, genesis)
	head := hashes[len(hashes)/2]

	low, high := len(hashes)/2+maxUncleDist+1, len(hashes)/2-maxQueueDist-1

	// Create a tester and simulate a head block being the middle of the above chain
	tester := newTester()

	tester.lock.Lock()
	tester.hashes = []common.Hash{head}
	tester.blocks = map[common.Hash]*types.Block{head: blocks[head]}
	tester.lock.Unlock()

	// Ensure that a block with a lower number than the threshold is discarded
	tester.fetcher.Enqueue("lower", blocks[hashes[low]])
	time.Sleep(10 * time.Millisecond)
	if !tester.fetcher.queue.Empty() {
		t.Fatalf("fetcher queued stale block")
	}
	// Ensure that a block with a higher number than the threshold is discarded
	tester.fetcher.Enqueue("higher", blocks[hashes[high]])
	time.Sleep(10 * time.Millisecond)
	if !tester.fetcher.queue.Empty() {
		t.Fatalf("fetcher queued future block")
	}
}

// Tests that announcements with numbers much lower or higher than out current
// head get discarded to prevent wasting resources on useless blocks from faulty
// peers.
func TestDistantAnnouncementDiscarding62(t *testing.T) { testDistantAnnouncementDiscarding(t, 62) }
func TestDistantAnnouncementDiscarding63(t *testing.T) { testDistantAnnouncementDiscarding(t, 63) }
func TestDistantAnnouncementDiscarding64(t *testing.T) { testDistantAnnouncementDiscarding(t, 64) }

func testDistantAnnouncementDiscarding(t *testing.T, protocol int) {
	// Create a long chain to import and define the discard boundaries
	hashes, blocks := makeChain(3*maxQueueDist, 0, genesis)
	head := hashes[len(hashes)/2]

	low, high := len(hashes)/2+maxUncleDist+1, len(hashes)/2-maxQueueDist-1

	// Create a tester and simulate a head block being the middle of the above chain
	tester := newTester()

	tester.lock.Lock()
	tester.hashes = []common.Hash{head}
	tester.blocks = map[common.Hash]*types.Block{head: blocks[head]}
	tester.lock.Unlock()

	headerFetcher := tester.makeHeaderFetcher(blocks, -gatherSlack)
	bodyFetcher := tester.makeBodyFetcher(blocks, 0)

	fetching := make(chan struct{}, 2)
	tester.fetcher.fetchingHook = func(hashes []common.Hash) { fetching <- struct{}{} }

	// Ensure that a block with a lower number than the threshold is discarded
	tester.fetcher.Notify("lower", hashes[low], blocks[hashes[low]].NumberU64(), time.Now().Add(-arriveTimeout), nil, headerFetcher, bodyFetcher)
	select {
	case <-time.After(50 * time.Millisecond):
	case <-fetching:
		t.Fatalf("fetcher requested stale header")
	}
	// Ensure that a block with a higher number than the threshold is discarded
	tester.fetcher.Notify("higher", hashes[high], blocks[hashes[high]].NumberU64(), time.Now().Add(-arriveTimeout), nil, headerFetcher, bodyFetcher)
	select {
	case <-time.After(50 * time.Millisecond):
	case <-fetching:
		t.Fatalf("fetcher requested future header")
	}
}

// Tests that peers announcing blocks with invalid numbers (i.e. not matching
// the headers provided afterwards) get dropped as malicious.
func TestInvalidNumberAnnouncement62(t *testing.T) { testInvalidNumberAnnouncement(t, 62) }
func TestInvalidNumberAnnouncement63(t *testing.T) { testInvalidNumberAnnouncement(t, 63) }
func TestInvalidNumberAnnouncement64(t *testing.T) { testInvalidNumberAnnouncement(t, 64) }

func testInvalidNumberAnnouncement(t *testing.T, protocol int) {
	// Create a single block to import and check numbers against
	hashes, blocks := makeChain(1, 0, genesis)

	tester := newTester()
	headerFetcher := tester.makeHeaderFetcher(blocks, -gatherSlack)
	bodyFetcher := tester.makeBodyFetcher(blocks, 0)

	imported := make(chan *types.Block)
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

	// Announce a block with a bad number, check for immediate drop
	tester.fetcher.Notify("bad", hashes[0], 2, time.Now().Add(-arriveTimeout), nil, headerFetcher, bodyFetcher)
	verifyImportEvent(t, imported, false)

	tester.lock.RLock()
	dropped := tester.drops["bad"]
	tester.lock.RUnlock()

	if !dropped {
		t.Fatalf("peer with invalid numbered announcement not dropped")
	}
	// Make sure a good announcement passes without a drop
	tester.fetcher.Notify("good", hashes[0], 1, time.Now().Add(-arriveTimeout), nil, headerFetcher, bodyFetcher)
	verifyImportEvent(t, imported, true)

	tester.lock.RLock()
	dropped = tester.drops["good"]
	tester.lock.RUnlock()

	if dropped {
		t.Fatalf("peer with valid numbered announcement dropped")
	}
	verifyImportDone(t, imported)
}

// Tests that if a block is empty (i.e. header only), no body request should be
// made, and instead the header should be assembled into a whole block in itself.
func TestEmptyBlockShortCircuit62(t *testing.T) { testEmptyBlockShortCircuit(t, 62) }
func TestEmptyBlockShortCircuit63(t *testing.T) { testEmptyBlockShortCircuit(t, 63) }
func TestEmptyBlockShortCircuit64(t *testing.T) { testEmptyBlockShortCircuit(t, 64) }

func testEmptyBlockShortCircuit(t *testing.T, protocol int) {
	// Create a chain of blocks to import
	hashes, blocks := makeChain(32, 0, genesis)

	tester := newTester()
	headerFetcher := tester.makeHeaderFetcher(blocks, -gatherSlack)
	bodyFetcher := tester.makeBodyFetcher(blocks, 0)

	// Add a monitoring hook for all internal events
	fetching := make(chan []common.Hash)
	tester.fetcher.fetchingHook = func(hashes []common.Hash) { fetching <- hashes }

	completing := make(chan []common.Hash)
	tester.fetcher.completingHook = func(hashes []common.Hash) { completing <- hashes }

	imported := make(chan *types.Block)
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

	// Iteratively announce blocks until all are imported
	for i := len(hashes) - 2; i >= 0; i-- {
		tester.fetcher.Notify("valid", hashes[i], uint64(len(hashes)-i-1), time.Now().Add(-arriveTimeout), nil, headerFetcher, bodyFetcher)

		// All announces should fetch the header
		verifyFetchingEvent(t, fetching, true)

		// Only blocks with data contents should request bodies
		verifyCompletingEvent(t, completing, len(blocks[hashes[i]].Transactions()) > 0 || len(blocks[hashes[i]].Uncles()) > 0)

		// Irrelevant of the construct, import should succeed
		verifyImportEvent(t, imported, true)
	}
	verifyImportDone(t, imported)
}

// Tests that a peer is unable to use unbounded memory with sending infinite
// block announcements to a node, but that even in the face of such an attack,
// the fetcher remains operational.
func TestHashMemoryExhaustionAttack61(t *testing.T) { testHashMemoryExhaustionAttack(t, 61) }
func TestHashMemoryExhaustionAttack62(t *testing.T) { testHashMemoryExhaustionAttack(t, 62) }
func TestHashMemoryExhaustionAttack63(t *testing.T) { testHashMemoryExhaustionAttack(t, 63) }
func TestHashMemoryExhaustionAttack64(t *testing.T) { testHashMemoryExhaustionAttack(t, 64) }

func testHashMemoryExhaustionAttack(t *testing.T, protocol int) {
	// Create a tester with instrumented import hooks
	tester := newTester()

	imported, announces := make(chan *types.Block), int32(0)
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }
	tester.fetcher.announceChangeHook = func(hash common.Hash, added bool) {
		if added {
			atomic.AddInt32(&announces, 1)
		} else {
			atomic.AddInt32(&announces, -1)
		}
	}
	// Create a valid chain and an infinite junk chain
	targetBlocks := hashLimit + 2*maxQueueDist
	hashes, blocks := makeChain(targetBlocks, 0, genesis)
	validBlockFetcher := tester.makeBlockFetcher(blocks)
	validHeaderFetcher := tester.makeHeaderFetcher(blocks, -gatherSlack)
	validBodyFetcher := tester.makeBodyFetcher(blocks, 0)

	attack, _ := makeChain(targetBlocks, 0, unknownBlock)
	attackerBlockFetcher := tester.makeBlockFetcher(nil)
	attackerHeaderFetcher := tester.makeHeaderFetcher(nil, -gatherSlack)
	attackerBodyFetcher := tester.makeBodyFetcher(nil, 0)

	// Feed the tester a huge hashset from the attacker, and a limited from the valid peer
	for i := 0; i < len(attack); i++ {
		if i < maxQueueDist {
			if protocol < 62 {
				tester.fetcher.Notify("valid", hashes[len(hashes)-2-i], 0, time.Now(), validBlockFetcher, nil, nil)
			} else {
				tester.fetcher.Notify("valid", hashes[len(hashes)-2-i], uint64(i+1), time.Now(), nil, validHeaderFetcher, validBodyFetcher)
			}
		}
		if protocol < 62 {
			tester.fetcher.Notify("attacker", attack[i], 0, time.Now(), attackerBlockFetcher, nil, nil)
		} else {
			tester.fetcher.Notify("attacker", attack[i], 1 /* don't distance drop */, time.Now(), nil, attackerHeaderFetcher, attackerBodyFetcher)
		}
	}
	if count := atomic.LoadInt32(&announces); count != hashLimit+maxQueueDist {
		t.Fatalf("queued announce count mismatch: have %d, want %d", count, hashLimit+maxQueueDist)
	}
	// Wait for fetches to complete
	verifyImportCount(t, imported, maxQueueDist)

	// Feed the remaining valid hashes to ensure DOS protection state remains clean
	for i := len(hashes) - maxQueueDist - 2; i >= 0; i-- {
		if protocol < 62 {
			tester.fetcher.Notify("valid", hashes[i], 0, time.Now().Add(-arriveTimeout), validBlockFetcher, nil, nil)
		} else {
			tester.fetcher.Notify("valid", hashes[i], uint64(len(hashes)-i-1), time.Now().Add(-arriveTimeout), nil, validHeaderFetcher, validBodyFetcher)
		}
		verifyImportEvent(t, imported, true)
	}
	verifyImportDone(t, imported)
}

// Tests that blocks sent to the fetcher (either through propagation or via hash
// announces and retrievals) don't pile up indefinitely, exhausting available
// system memory.
func TestBlockMemoryExhaustionAttack(t *testing.T) {
	// Create a tester with instrumented import hooks
	tester := newTester()

	imported, enqueued := make(chan *types.Block), int32(0)
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }
	tester.fetcher.queueChangeHook = func(hash common.Hash, added bool) {
		if added {
			atomic.AddInt32(&enqueued, 1)
		} else {
			atomic.AddInt32(&enqueued, -1)
		}
	}
	// Create a valid chain and a batch of dangling (but in range) blocks
	targetBlocks := hashLimit + 2*maxQueueDist
	hashes, blocks := makeChain(targetBlocks, 0, genesis)
	attack := make(map[common.Hash]*types.Block)
	for i := byte(0); len(attack) < blockLimit+2*maxQueueDist; i++ {
		hashes, blocks := makeChain(maxQueueDist-1, i, unknownBlock)
		for _, hash := range hashes[:maxQueueDist-2] {
			attack[hash] = blocks[hash]
		}
	}
	// Try to feed all the attacker blocks make sure only a limited batch is accepted
	for _, block := range attack {
		tester.fetcher.Enqueue("attacker", block)
	}
	time.Sleep(200 * time.Millisecond)
	if queued := atomic.LoadInt32(&enqueued); queued != blockLimit {
		t.Fatalf("queued block count mismatch: have %d, want %d", queued, blockLimit)
	}
	// Queue up a batch of valid blocks, and check that a new peer is allowed to do so
	for i := 0; i < maxQueueDist-1; i++ {
		tester.fetcher.Enqueue("valid", blocks[hashes[len(hashes)-3-i]])
	}
	time.Sleep(100 * time.Millisecond)
	if queued := atomic.LoadInt32(&enqueued); queued != blockLimit+maxQueueDist-1 {
		t.Fatalf("queued block count mismatch: have %d, want %d", queued, blockLimit+maxQueueDist-1)
	}
	// Insert the missing piece (and sanity check the import)
	tester.fetcher.Enqueue("valid", blocks[hashes[len(hashes)-2]])
	verifyImportCount(t, imported, maxQueueDist)

	// Insert the remaining blocks in chunks to ensure clean DOS protection
	for i := maxQueueDist; i < len(hashes)-1; i++ {
		tester.fetcher.Enqueue("valid", blocks[hashes[len(hashes)-2-i]])
		verifyImportEvent(t, imported, true)
	}
	verifyImportDone(t, imported)
}
