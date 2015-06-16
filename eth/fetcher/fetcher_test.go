package fetcher

import (
	"encoding/binary"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	knownHash   = common.Hash{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	unknownHash = common.Hash{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	bannedHash  = common.Hash{3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3}

	genesis = createBlock(1, common.Hash{}, knownHash)
)

// idCounter is used by the createHashes method the generate deterministic but unique hashes
var idCounter = int64(2) // #1 is the genesis block

// createHashes generates a batch of hashes rooted at a specific point in the chain.
func createHashes(amount int, root common.Hash) (hashes []common.Hash) {
	hashes = make([]common.Hash, amount+1)
	hashes[len(hashes)-1] = root

	for i := 0; i < len(hashes)-1; i++ {
		binary.BigEndian.PutUint64(hashes[i][:8], uint64(idCounter))
		idCounter++
	}
	return
}

// createBlock assembles a new block at the given chain height.
func createBlock(i int, parent, hash common.Hash) *types.Block {
	header := &types.Header{Number: big.NewInt(int64(i))}
	block := types.NewBlockWithHeader(header)
	block.HeaderHash = hash
	block.ParentHeaderHash = parent
	return block
}

// copyBlock makes a deep copy of a block suitable for local modifications.
func copyBlock(block *types.Block) *types.Block {
	return createBlock(int(block.Number().Int64()), block.ParentHeaderHash, block.HeaderHash)
}

// createBlocksFromHashes assembles a collection of blocks, each having a correct
// place in the given hash chain.
func createBlocksFromHashes(hashes []common.Hash) map[common.Hash]*types.Block {
	blocks := make(map[common.Hash]*types.Block)
	for i := 0; i < len(hashes); i++ {
		parent := knownHash
		if i < len(hashes)-1 {
			parent = hashes[i+1]
		}
		blocks[hashes[i]] = createBlock(len(hashes)-i, parent, hashes[i])
	}
	return blocks
}

// fetcherTester is a test simulator for mocking out local block chain.
type fetcherTester struct {
	fetcher *Fetcher

	ownHashes []common.Hash                // Hash chain belonging to the tester
	ownBlocks map[common.Hash]*types.Block // Blocks belonging to the tester
}

// newTester creates a new fetcher test mocker.
func newTester() *fetcherTester {
	tester := &fetcherTester{
		ownHashes: []common.Hash{knownHash},
		ownBlocks: map[common.Hash]*types.Block{knownHash: genesis},
	}
	tester.fetcher = New(tester.hasBlock, tester.importBlock, tester.chainHeight)
	tester.fetcher.Start()

	return tester
}

// hasBlock checks if a block is pres	ent in the testers canonical chain.
func (f *fetcherTester) hasBlock(hash common.Hash) bool {
	_, ok := f.ownBlocks[hash]
	return ok
}

// importBlock injects a new blocks into the simulated chain.
func (f *fetcherTester) importBlock(peer string, block *types.Block) error {
	if _, ok := f.ownBlocks[block.ParentHash()]; !ok {
		return errors.New("unknown parent")
	}
	f.ownHashes = append(f.ownHashes, block.Hash())
	f.ownBlocks[block.Hash()] = block
	return nil
}

// chainHeight retrieves the current height (block number) of the chain.
func (f *fetcherTester) chainHeight() uint64 {
	return f.ownBlocks[f.ownHashes[len(f.ownHashes)-1]].NumberU64()
}

// peerFetcher retrieves a fetcher associated with a simulated peer.
func (f *fetcherTester) makeFetcher(blocks map[common.Hash]*types.Block) blockRequesterFn {
	// Copy all the blocks to ensure they are not tampered with
	closure := make(map[common.Hash]*types.Block)
	for hash, block := range blocks {
		closure[hash] = copyBlock(block)
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

// Tests that a fetcher accepts block announcements and initiates retrievals for
// them, successfully importing into the local chain.
func TestSequentialAnnouncements(t *testing.T) {
	// Create a chain of blocks to import
	targetBlocks := 24
	hashes := createHashes(targetBlocks, knownHash)
	blocks := createBlocksFromHashes(hashes)

	tester := newTester()
	fetcher := tester.makeFetcher(blocks)

	// Iteratively announce blocks until all are imported
	for i := len(hashes) - 1; i >= 0; i-- {
		tester.fetcher.Notify("valid", hashes[i], time.Now().Add(-arriveTimeout), fetcher)
		time.Sleep(50 * time.Millisecond)
	}
	if imported := len(tester.ownBlocks); imported != targetBlocks+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, targetBlocks+1)
	}
}

// Tests that if blocks are announced by multiple peers (or even the same buggy
// peer), they will only get downloaded at most once.
func TestConcurrentAnnouncements(t *testing.T) {
	// Create a chain of blocks to import
	targetBlocks := 24
	hashes := createHashes(targetBlocks, knownHash)
	blocks := createBlocksFromHashes(hashes)

	// Assemble a tester with a built in counter for the requests
	tester := newTester()
	fetcher := tester.makeFetcher(blocks)

	counter := uint32(0)
	wrapper := func(hashes []common.Hash) error {
		atomic.AddUint32(&counter, uint32(len(hashes)))
		return fetcher(hashes)
	}
	// Iteratively announce blocks until all are imported
	for i := len(hashes) - 1; i >= 0; i-- {
		tester.fetcher.Notify("first", hashes[i], time.Now().Add(-arriveTimeout), wrapper)
		tester.fetcher.Notify("second", hashes[i], time.Now().Add(-arriveTimeout+time.Millisecond), wrapper)
		tester.fetcher.Notify("second", hashes[i], time.Now().Add(-arriveTimeout-time.Millisecond), wrapper)

		time.Sleep(50 * time.Millisecond)
	}
	if imported := len(tester.ownBlocks); imported != targetBlocks+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, targetBlocks+1)
	}
	// Make sure no blocks were retrieved twice
	if int(counter) != targetBlocks {
		t.Fatalf("retrieval count mismatch: have %v, want %v", counter, targetBlocks)
	}
}

// Tests that announcements arriving while a previous is being fetched still
// results in a valid import.
func TestOverlappingAnnouncements(t *testing.T) {
	// Create a chain of blocks to import
	targetBlocks := 24
	hashes := createHashes(targetBlocks, knownHash)
	blocks := createBlocksFromHashes(hashes)

	tester := newTester()
	fetcher := tester.makeFetcher(blocks)

	// Iteratively announce blocks, but overlap them continuously
	delay, overlap := 50*time.Millisecond, time.Duration(5)
	for i := len(hashes) - 1; i >= 0; i-- {
		tester.fetcher.Notify("valid", hashes[i], time.Now().Add(-arriveTimeout+overlap*delay), fetcher)
		time.Sleep(delay)
	}
	time.Sleep(overlap * delay)

	if imported := len(tester.ownBlocks); imported != targetBlocks+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, targetBlocks+1)
	}
}

// Tests that announces already being retrieved will not be duplicated.
func TestPendingDeduplication(t *testing.T) {
	// Create a hash and corresponding block
	hashes := createHashes(1, knownHash)
	blocks := createBlocksFromHashes(hashes)

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
	for !tester.hasBlock(hashes[0]) {
		tester.fetcher.Notify("repeater", hashes[0], time.Now().Add(-arriveTimeout), wrapper)
		time.Sleep(time.Millisecond)
	}
	time.Sleep(delay)

	// Check that all blocks were imported and none fetched twice
	if imported := len(tester.ownBlocks); imported != 2 {
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
	targetBlocks := 24
	hashes := createHashes(targetBlocks, knownHash)
	blocks := createBlocksFromHashes(hashes)
	skip := targetBlocks / 2

	tester := newTester()
	fetcher := tester.makeFetcher(blocks)

	// Iteratively announce blocks, skipping one entry
	for i := len(hashes) - 1; i >= 0; i-- {
		if i != skip {
			tester.fetcher.Notify("valid", hashes[i], time.Now().Add(-arriveTimeout), fetcher)
			time.Sleep(50 * time.Millisecond)
		}
	}
	// Finally announce the skipped entry and check full import
	tester.fetcher.Notify("valid", hashes[skip], time.Now().Add(-arriveTimeout), fetcher)
	time.Sleep(50 * time.Millisecond)

	if imported := len(tester.ownBlocks); imported != targetBlocks+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, targetBlocks+1)
	}
}

// Tests that direct block enqueues (due to block propagation vs. hash announce)
// are correctly schedule, filling and import queue gaps.
func TestQueueGapFill(t *testing.T) {
	// Create a chain of blocks to import, and choose one to not announce at all
	targetBlocks := 24
	hashes := createHashes(targetBlocks, knownHash)
	blocks := createBlocksFromHashes(hashes)
	skip := targetBlocks / 2

	tester := newTester()
	fetcher := tester.makeFetcher(blocks)

	// Iteratively announce blocks, skipping one entry
	for i := len(hashes) - 1; i >= 0; i-- {
		if i != skip {
			tester.fetcher.Notify("valid", hashes[i], time.Now().Add(-arriveTimeout), fetcher)
			time.Sleep(50 * time.Millisecond)
		}
	}
	// Fill the missing block directly as if propagated
	tester.fetcher.Enqueue("valid", blocks[hashes[skip]])
	time.Sleep(50 * time.Millisecond)

	if imported := len(tester.ownBlocks); imported != targetBlocks+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, targetBlocks+1)
	}
}
