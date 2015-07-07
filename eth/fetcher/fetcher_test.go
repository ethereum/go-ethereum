// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

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
	"github.com/ethereum/go-ethereum/ethdb"
)

var (
	testdb, _    = ethdb.NewMemDatabase()
	genesis      = core.GenesisBlockForTesting(testdb, common.Address{}, big.NewInt(0))
	unknownBlock = types.NewBlock(&types.Header{}, nil, nil, nil)
)

// makeChain creates a chain of n blocks starting at and including parent.
// the returned hash chain is ordered head->parent.
func makeChain(n int, seed byte, parent *types.Block) ([]common.Hash, map[common.Hash]*types.Block) {
	blocks := core.GenerateChain(parent, testdb, n, func(i int, gen *core.BlockGen) {
		gen.SetCoinbase(common.Address{seed})
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

	lock sync.RWMutex
}

// newTester creates a new fetcher test mocker.
func newTester() *fetcherTester {
	tester := &fetcherTester{
		hashes: []common.Hash{genesis.Hash()},
		blocks: map[common.Hash]*types.Block{genesis.Hash(): genesis},
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

// dropPeer is a nop placeholder for the peer removal.
func (f *fetcherTester) dropPeer(peer string) {
}

// peerFetcher retrieves a fetcher associated with a simulated peer.
func (f *fetcherTester) makeFetcher(blocks map[common.Hash]*types.Block) blockRequesterFn {
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
		go f.fetcher.Filter(blocks)

		return nil
	}
}

// verifyImportEvent verifies that one single event arrive on an import channel.
func verifyImportEvent(t *testing.T, imported chan *types.Block) {
	select {
	case <-imported:
	case <-time.After(time.Second):
		t.Fatalf("import timeout")
	}
}

// verifyImportCount verifies that exactly count number of events arrive on an
// import hook channel.
func verifyImportCount(t *testing.T, imported chan *types.Block, count int) {
	for i := 0; i < count; i++ {
		select {
		case <-imported:
		case <-time.After(time.Second):
			t.Fatalf("block %d: import timeout", i)
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
func TestSequentialAnnouncements(t *testing.T) {
	// Create a chain of blocks to import
	targetBlocks := 4 * hashLimit
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	tester := newTester()
	fetcher := tester.makeFetcher(blocks)

	// Iteratively announce blocks until all are imported
	imported := make(chan *types.Block)
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

	for i := len(hashes) - 2; i >= 0; i-- {
		tester.fetcher.Notify("valid", hashes[i], time.Now().Add(-arriveTimeout), fetcher)
		verifyImportEvent(t, imported)
	}
	verifyImportDone(t, imported)
}

// Tests that if blocks are announced by multiple peers (or even the same buggy
// peer), they will only get downloaded at most once.
func TestConcurrentAnnouncements(t *testing.T) {
	// Create a chain of blocks to import
	targetBlocks := 4 * hashLimit
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	// Assemble a tester with a built in counter for the requests
	tester := newTester()
	fetcher := tester.makeFetcher(blocks)

	counter := uint32(0)
	wrapper := func(hashes []common.Hash) error {
		atomic.AddUint32(&counter, uint32(len(hashes)))
		return fetcher(hashes)
	}
	// Iteratively announce blocks until all are imported
	imported := make(chan *types.Block)
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

	for i := len(hashes) - 2; i >= 0; i-- {
		tester.fetcher.Notify("first", hashes[i], time.Now().Add(-arriveTimeout), wrapper)
		tester.fetcher.Notify("second", hashes[i], time.Now().Add(-arriveTimeout+time.Millisecond), wrapper)
		tester.fetcher.Notify("second", hashes[i], time.Now().Add(-arriveTimeout-time.Millisecond), wrapper)

		verifyImportEvent(t, imported)
	}
	verifyImportDone(t, imported)

	// Make sure no blocks were retrieved twice
	if int(counter) != targetBlocks {
		t.Fatalf("retrieval count mismatch: have %v, want %v", counter, targetBlocks)
	}
}

// Tests that announcements arriving while a previous is being fetched still
// results in a valid import.
func TestOverlappingAnnouncements(t *testing.T) {
	// Create a chain of blocks to import
	targetBlocks := 4 * hashLimit
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	tester := newTester()
	fetcher := tester.makeFetcher(blocks)

	// Iteratively announce blocks, but overlap them continuously
	fetching := make(chan []common.Hash)
	imported := make(chan *types.Block, len(hashes)-1)
	tester.fetcher.fetchingHook = func(hashes []common.Hash) { fetching <- hashes }
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

	for i := len(hashes) - 2; i >= 0; i-- {
		tester.fetcher.Notify("valid", hashes[i], time.Now().Add(-arriveTimeout), fetcher)
		select {
		case <-fetching:
		case <-time.After(time.Second):
			t.Fatalf("hash %d: announce timeout", len(hashes)-i)
		}
	}
	// Wait for all the imports to complete and check count
	verifyImportCount(t, imported, len(hashes)-1)
}

// Tests that announces already being retrieved will not be duplicated.
func TestPendingDeduplication(t *testing.T) {
	// Create a hash and corresponding block
	hashes, blocks := makeChain(1, 0, genesis)

	// Assemble a tester with a built in counter and delayed fetcher
	tester := newTester()
	fetcher := tester.makeFetcher(blocks)

	delay := 50 * time.Millisecond
	counter := uint32(0)
	wrapper := func(hashes []common.Hash) error {
		atomic.AddUint32(&counter, uint32(len(hashes)))

		// Simulate a long running fetch
		go func() {
			time.Sleep(delay)
			fetcher(hashes)
		}()
		return nil
	}
	// Announce the same block many times until it's fetched (wait for any pending ops)
	for tester.getBlock(hashes[0]) == nil {
		tester.fetcher.Notify("repeater", hashes[0], time.Now().Add(-arriveTimeout), wrapper)
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
func TestRandomArrivalImport(t *testing.T) {
	// Create a chain of blocks to import, and choose one to delay
	targetBlocks := maxQueueDist
	hashes, blocks := makeChain(targetBlocks, 0, genesis)
	skip := targetBlocks / 2

	tester := newTester()
	fetcher := tester.makeFetcher(blocks)

	// Iteratively announce blocks, skipping one entry
	imported := make(chan *types.Block, len(hashes)-1)
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

	for i := len(hashes) - 1; i >= 0; i-- {
		if i != skip {
			tester.fetcher.Notify("valid", hashes[i], time.Now().Add(-arriveTimeout), fetcher)
			time.Sleep(time.Millisecond)
		}
	}
	// Finally announce the skipped entry and check full import
	tester.fetcher.Notify("valid", hashes[skip], time.Now().Add(-arriveTimeout), fetcher)
	verifyImportCount(t, imported, len(hashes)-1)
}

// Tests that direct block enqueues (due to block propagation vs. hash announce)
// are correctly schedule, filling and import queue gaps.
func TestQueueGapFill(t *testing.T) {
	// Create a chain of blocks to import, and choose one to not announce at all
	targetBlocks := maxQueueDist
	hashes, blocks := makeChain(targetBlocks, 0, genesis)
	skip := targetBlocks / 2

	tester := newTester()
	fetcher := tester.makeFetcher(blocks)

	// Iteratively announce blocks, skipping one entry
	imported := make(chan *types.Block, len(hashes)-1)
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

	for i := len(hashes) - 1; i >= 0; i-- {
		if i != skip {
			tester.fetcher.Notify("valid", hashes[i], time.Now().Add(-arriveTimeout), fetcher)
			time.Sleep(time.Millisecond)
		}
	}
	// Fill the missing block directly as if propagated
	tester.fetcher.Enqueue("valid", blocks[hashes[skip]])
	verifyImportCount(t, imported, len(hashes)-1)
}

// Tests that blocks arriving from various sources (multiple propagations, hash
// announces, etc) do not get scheduled for import multiple times.
func TestImportDeduplication(t *testing.T) {
	// Create two blocks to import (one for duplication, the other for stalling)
	hashes, blocks := makeChain(2, 0, genesis)

	// Create the tester and wrap the importer with a counter
	tester := newTester()
	fetcher := tester.makeFetcher(blocks)

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
	tester.fetcher.Notify("valid", hashes[0], time.Now().Add(-arriveTimeout), fetcher)
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
// discarded no prevent wasting resources on useless blocks from faulty peers.
func TestDistantDiscarding(t *testing.T) {
	// Create a long chain to import
	hashes, blocks := makeChain(3*maxQueueDist, 0, genesis)
	head := hashes[len(hashes)/2]

	// Create a tester and simulate a head block being the middle of the above chain
	tester := newTester()
	tester.hashes = []common.Hash{head}
	tester.blocks = map[common.Hash]*types.Block{head: blocks[head]}

	// Ensure that a block with a lower number than the threshold is discarded
	tester.fetcher.Enqueue("lower", blocks[hashes[0]])
	time.Sleep(10 * time.Millisecond)
	if !tester.fetcher.queue.Empty() {
		t.Fatalf("fetcher queued stale block")
	}
	// Ensure that a block with a higher number than the threshold is discarded
	tester.fetcher.Enqueue("higher", blocks[hashes[len(hashes)-1]])
	time.Sleep(10 * time.Millisecond)
	if !tester.fetcher.queue.Empty() {
		t.Fatalf("fetcher queued future block")
	}
}

// Tests that a peer is unable to use unbounded memory with sending infinite
// block announcements to a node, but that even in the face of such an attack,
// the fetcher remains operational.
func TestHashMemoryExhaustionAttack(t *testing.T) {
	// Create a tester with instrumented import hooks
	tester := newTester()

	imported := make(chan *types.Block)
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

	// Create a valid chain and an infinite junk chain
	targetBlocks := hashLimit + 2*maxQueueDist
	hashes, blocks := makeChain(targetBlocks, 0, genesis)
	valid := tester.makeFetcher(blocks)

	attack, _ := makeChain(targetBlocks, 0, unknownBlock)
	attacker := tester.makeFetcher(nil)

	// Feed the tester a huge hashset from the attacker, and a limited from the valid peer
	for i := 0; i < len(attack); i++ {
		if i < maxQueueDist {
			tester.fetcher.Notify("valid", hashes[len(hashes)-2-i], time.Now(), valid)
		}
		tester.fetcher.Notify("attacker", attack[i], time.Now(), attacker)
	}
	if len(tester.fetcher.announced) != hashLimit+maxQueueDist {
		t.Fatalf("queued announce count mismatch: have %d, want %d", len(tester.fetcher.announced), hashLimit+maxQueueDist)
	}
	// Wait for fetches to complete
	verifyImportCount(t, imported, maxQueueDist)

	// Feed the remaining valid hashes to ensure DOS protection state remains clean
	for i := len(hashes) - maxQueueDist - 2; i >= 0; i-- {
		tester.fetcher.Notify("valid", hashes[i], time.Now().Add(-arriveTimeout), valid)
		verifyImportEvent(t, imported)
	}
	verifyImportDone(t, imported)
}

// Tests that blocks sent to the fetcher (either through propagation or via hash
// announces and retrievals) don't pile up indefinitely, exhausting available
// system memory.
func TestBlockMemoryExhaustionAttack(t *testing.T) {
	// Create a tester with instrumented import hooks
	tester := newTester()

	imported := make(chan *types.Block)
	tester.fetcher.importedHook = func(block *types.Block) { imported <- block }

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
	if queued := tester.fetcher.queue.Size(); queued != blockLimit {
		t.Fatalf("queued block count mismatch: have %d, want %d", queued, blockLimit)
	}
	// Queue up a batch of valid blocks, and check that a new peer is allowed to do so
	for i := 0; i < maxQueueDist-1; i++ {
		tester.fetcher.Enqueue("valid", blocks[hashes[len(hashes)-3-i]])
	}
	time.Sleep(100 * time.Millisecond)
	if queued := tester.fetcher.queue.Size(); queued != blockLimit+maxQueueDist-1 {
		t.Fatalf("queued block count mismatch: have %d, want %d", queued, blockLimit+maxQueueDist-1)
	}
	// Insert the missing piece (and sanity check the import)
	tester.fetcher.Enqueue("valid", blocks[hashes[len(hashes)-2]])
	verifyImportCount(t, imported, maxQueueDist)

	// Insert the remaining blocks in chunks to ensure clean DOS protection
	for i := maxQueueDist; i < len(hashes)-1; i++ {
		tester.fetcher.Enqueue("valid", blocks[hashes[len(hashes)-2-i]])
		verifyImportEvent(t, imported)
	}
	verifyImportDone(t, imported)
}
