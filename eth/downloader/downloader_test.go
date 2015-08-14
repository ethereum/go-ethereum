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

package downloader

import (
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
)

var (
	testdb, _ = ethdb.NewMemDatabase()
	genesis   = core.GenesisBlockForTesting(testdb, common.Address{}, big.NewInt(0))
)

// makeChain creates a chain of n blocks starting at but not including
// parent. the returned hash chain is ordered head->parent.
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

// makeChainFork creates two chains of length n, such that h1[:f] and
// h2[:f] are different but have a common suffix of length n-f.
func makeChainFork(n, f int, parent *types.Block) (h1, h2 []common.Hash, b1, b2 map[common.Hash]*types.Block) {
	// Create the common suffix.
	h, b := makeChain(n-f, 0, parent)
	// Create the forks.
	h1, b1 = makeChain(f, 1, b[h[0]])
	h1 = append(h1, h[1:]...)
	h2, b2 = makeChain(f, 2, b[h[0]])
	h2 = append(h2, h[1:]...)
	for hash, block := range b {
		b1[hash] = block
		b2[hash] = block
	}
	return h1, h2, b1, b2
}

// downloadTester is a test simulator for mocking out local block chain.
type downloadTester struct {
	downloader *Downloader

	ownHashes  []common.Hash                           // Hash chain belonging to the tester
	ownBlocks  map[common.Hash]*types.Block            // Blocks belonging to the tester
	peerHashes map[string][]common.Hash                // Hash chain belonging to different test peers
	peerBlocks map[string]map[common.Hash]*types.Block // Blocks belonging to different test peers

	maxHashFetch int // Overrides the maximum number of retrieved hashes
}

// newTester creates a new downloader test mocker.
func newTester() *downloadTester {
	tester := &downloadTester{
		ownHashes:  []common.Hash{genesis.Hash()},
		ownBlocks:  map[common.Hash]*types.Block{genesis.Hash(): genesis},
		peerHashes: make(map[string][]common.Hash),
		peerBlocks: make(map[string]map[common.Hash]*types.Block),
	}
	tester.downloader = New(new(event.TypeMux), tester.hasBlock, tester.getBlock, tester.headBlock, tester.insertChain, tester.dropPeer)

	return tester
}

// sync starts synchronizing with a remote peer, blocking until it completes.
func (dl *downloadTester) sync(id string, td *big.Int) error {
	hash := dl.peerHashes[id][0]

	// If no particular TD was requested, load from the peer's blockchain
	if td == nil {
		td = big.NewInt(1)
		if block, ok := dl.peerBlocks[id][hash]; ok {
			td = block.Td
		}
	}
	err := dl.downloader.synchronise(id, hash, td)

	for {
		// If the queue is empty and processing stopped, break
		hashes, blocks := dl.downloader.queue.Size()
		if hashes+blocks == 0 && atomic.LoadInt32(&dl.downloader.processing) == 0 {
			break
		}
		// Otherwise sleep a bit and retry
		time.Sleep(time.Millisecond)
	}
	return err
}

// hasBlock checks if a block is pres	ent in the testers canonical chain.
func (dl *downloadTester) hasBlock(hash common.Hash) bool {
	return dl.getBlock(hash) != nil
}

// getBlock retrieves a block from the testers canonical chain.
func (dl *downloadTester) getBlock(hash common.Hash) *types.Block {
	return dl.ownBlocks[hash]
}

// headBlock retrieves the current head block from the canonical chain.
func (dl *downloadTester) headBlock() *types.Block {
	return dl.getBlock(dl.ownHashes[len(dl.ownHashes)-1])
}

// insertChain injects a new batch of blocks into the simulated chain.
func (dl *downloadTester) insertChain(blocks types.Blocks) (int, error) {
	for i, block := range blocks {
		if _, ok := dl.ownBlocks[block.ParentHash()]; !ok {
			return i, errors.New("unknown parent")
		}
		dl.ownHashes = append(dl.ownHashes, block.Hash())
		dl.ownBlocks[block.Hash()] = block
	}
	return len(blocks), nil
}

// newPeer registers a new block download source into the downloader.
func (dl *downloadTester) newPeer(id string, version int, hashes []common.Hash, blocks map[common.Hash]*types.Block) error {
	return dl.newSlowPeer(id, version, hashes, blocks, 0)
}

// newSlowPeer registers a new block download source into the downloader, with a
// specific delay time on processing the network packets sent to it, simulating
// potentially slow network IO.
func (dl *downloadTester) newSlowPeer(id string, version int, hashes []common.Hash, blocks map[common.Hash]*types.Block, delay time.Duration) error {
	err := dl.downloader.RegisterPeer(id, version, hashes[0], dl.peerGetRelHashesFn(id, delay), dl.peerGetAbsHashesFn(id, version, delay), dl.peerGetBlocksFn(id, delay))
	if err == nil {
		// Assign the owned hashes and blocks to the peer (deep copy)
		dl.peerHashes[id] = make([]common.Hash, len(hashes))
		copy(dl.peerHashes[id], hashes)
		dl.peerBlocks[id] = make(map[common.Hash]*types.Block)
		for hash, block := range blocks {
			dl.peerBlocks[id][hash] = block
		}
	}
	return err
}

// dropPeer simulates a hard peer removal from the connection pool.
func (dl *downloadTester) dropPeer(id string) {
	delete(dl.peerHashes, id)
	delete(dl.peerBlocks, id)

	dl.downloader.UnregisterPeer(id)
}

// peerGetRelHashesFn constructs a GetHashes function associated with a specific
// peer in the download tester. The returned function can be used to retrieve
// batches of hashes from the particularly requested peer.
func (dl *downloadTester) peerGetRelHashesFn(id string, delay time.Duration) func(head common.Hash) error {
	return func(head common.Hash) error {
		time.Sleep(delay)

		limit := MaxHashFetch
		if dl.maxHashFetch > 0 {
			limit = dl.maxHashFetch
		}
		// Gather the next batch of hashes
		hashes := dl.peerHashes[id]
		result := make([]common.Hash, 0, limit)
		for i, hash := range hashes {
			if hash == head {
				i++
				for len(result) < cap(result) && i < len(hashes) {
					result = append(result, hashes[i])
					i++
				}
				break
			}
		}
		// Delay delivery a bit to allow attacks to unfold
		go func() {
			time.Sleep(time.Millisecond)
			dl.downloader.DeliverHashes(id, result)
		}()
		return nil
	}
}

// peerGetAbsHashesFn constructs a GetHashesFromNumber function associated with
// a particular peer in the download tester. The returned function can be used to
// retrieve batches of hashes from the particularly requested peer.
func (dl *downloadTester) peerGetAbsHashesFn(id string, version int, delay time.Duration) func(uint64, int) error {
	return func(head uint64, count int) error {
		time.Sleep(delay)

		limit := count
		if dl.maxHashFetch > 0 {
			limit = dl.maxHashFetch
		}
		// Gather the next batch of hashes
		hashes := dl.peerHashes[id]
		result := make([]common.Hash, 0, limit)
		for i := 0; i < limit && len(hashes)-int(head)-1-i >= 0; i++ {
			result = append(result, hashes[len(hashes)-int(head)-1-i])
		}
		// Delay delivery a bit to allow attacks to unfold
		go func() {
			time.Sleep(time.Millisecond)
			dl.downloader.DeliverHashes(id, result)
		}()
		return nil
	}
}

// peerGetBlocksFn constructs a getBlocks function associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of blocks from the particularly requested peer.
func (dl *downloadTester) peerGetBlocksFn(id string, delay time.Duration) func([]common.Hash) error {
	return func(hashes []common.Hash) error {
		time.Sleep(delay)
		blocks := dl.peerBlocks[id]
		result := make([]*types.Block, 0, len(hashes))
		for _, hash := range hashes {
			if block, ok := blocks[hash]; ok {
				result = append(result, block)
			}
		}
		go dl.downloader.DeliverBlocks(id, result)

		return nil
	}
}

// Tests that simple synchronization against a canonical chain works correctly.
// In this test common ancestor lookup should be short circuited and not require
// binary searching.
func TestCanonicalSynchronisation61(t *testing.T) {
	// Create a small enough block chain to download
	targetBlocks := blockCacheLimit - 15
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	tester := newTester()
	tester.newPeer("peer", eth61, hashes, blocks)

	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.sync("peer", nil); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if imported := len(tester.ownBlocks); imported != targetBlocks+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, targetBlocks+1)
	}
}

// Tests that if a large batch of blocks are being downloaded, it is throttled
// until the cached blocks are retrieved.
func TestThrottling61(t *testing.T) { testThrottling(t, eth61) }

func testThrottling(t *testing.T, protocol int) {
	// Create a long block chain to download and the tester
	targetBlocks := 8 * blockCacheLimit
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	tester := newTester()
	tester.newPeer("peer", protocol, hashes, blocks)

	// Wrap the importer to allow stepping
	done := make(chan int)
	tester.downloader.insertChain = func(blocks types.Blocks) (int, error) {
		n, err := tester.insertChain(blocks)
		done <- n
		return n, err
	}
	// Start a synchronisation concurrently
	errc := make(chan error)
	go func() {
		errc <- tester.sync("peer", nil)
	}()
	// Iteratively take some blocks, always checking the retrieval count
	for len(tester.ownBlocks) < targetBlocks+1 {
		// Wait a bit for sync to throttle itself
		var cached int
		for start := time.Now(); time.Since(start) < 3*time.Second; {
			time.Sleep(25 * time.Millisecond)

			cached = len(tester.downloader.queue.blockPool)
			if cached == blockCacheLimit || len(tester.ownBlocks)+cached == targetBlocks+1 {
				break
			}
		}
		// Make sure we filled up the cache, then exhaust it
		time.Sleep(25 * time.Millisecond) // give it a chance to screw up
		if cached != blockCacheLimit && len(tester.ownBlocks)+cached < targetBlocks+1 {
			t.Fatalf("block count mismatch: have %v, want %v", cached, blockCacheLimit)
		}
		<-done // finish previous blocking import
		for cached > maxBlockProcess {
			cached -= <-done
		}
		time.Sleep(25 * time.Millisecond) // yield to the insertion
	}
	<-done // finish the last blocking import

	// Check that we haven't pulled more blocks than available
	if len(tester.ownBlocks) > targetBlocks+1 {
		t.Fatalf("target block count mismatch: have %v, want %v", len(tester.ownBlocks), targetBlocks+1)
	}
	if err := <-errc; err != nil {
		t.Fatalf("block synchronization failed: %v", err)
	}
}

// Tests that simple synchronization against a forked chain works correctly. In
// this test common ancestor lookup should *not* be short circuited, and a full
// binary search should be executed.
func TestForkedSynchronisation61(t *testing.T) {
	// Create a long enough forked chain
	common, fork := MaxHashFetch, 2*MaxHashFetch
	hashesA, hashesB, blocksA, blocksB := makeChainFork(common+fork, fork, genesis)

	tester := newTester()
	tester.newPeer("fork A", eth61, hashesA, blocksA)
	tester.newPeer("fork B", eth61, hashesB, blocksB)

	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.sync("fork A", nil); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if imported := len(tester.ownBlocks); imported != common+fork+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, common+fork+1)
	}
	// Synchronise with the second peer and make sure that fork is pulled too
	if err := tester.sync("fork B", nil); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if imported := len(tester.ownBlocks); imported != common+2*fork+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, common+2*fork+1)
	}
}

// Tests that an inactive downloader will not accept incoming hashes and blocks.
func TestInactiveDownloader(t *testing.T) {
	tester := newTester()

	// Check that neither hashes nor blocks are accepted
	if err := tester.downloader.DeliverHashes("bad peer", []common.Hash{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
	if err := tester.downloader.DeliverBlocks("bad peer", []*types.Block{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
}

// Tests that a canceled download wipes all previously accumulated state.
func TestCancel61(t *testing.T) { testCancel(t, eth61) }

func testCancel(t *testing.T, protocol int) {
	// Create a small enough block chain to download and the tester
	targetBlocks := blockCacheLimit - 15
	if targetBlocks >= MaxHashFetch {
		targetBlocks = MaxHashFetch - 15
	}
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	tester := newTester()
	tester.newPeer("peer", protocol, hashes, blocks)

	// Make sure canceling works with a pristine downloader
	tester.downloader.cancel()
	hashCount, blockCount := tester.downloader.queue.Size()
	if hashCount > 0 || blockCount > 0 {
		t.Errorf("block or hash count mismatch: %d hashes, %d blocks, want 0", hashCount, blockCount)
	}
	// Synchronise with the peer, but cancel afterwards
	if err := tester.sync("peer", nil); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	tester.downloader.cancel()
	hashCount, blockCount = tester.downloader.queue.Size()
	if hashCount > 0 || blockCount > 0 {
		t.Errorf("block or hash count mismatch: %d hashes, %d blocks, want 0", hashCount, blockCount)
	}
}

// Tests that synchronisation from multiple peers works as intended (multi thread sanity test).
func TestMultiSynchronisation61(t *testing.T) { testMultiSynchronisation(t, eth61) }

func testMultiSynchronisation(t *testing.T, protocol int) {
	// Create various peers with various parts of the chain
	targetPeers := 16
	targetBlocks := targetPeers*blockCacheLimit - 15
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	tester := newTester()
	for i := 0; i < targetPeers; i++ {
		id := fmt.Sprintf("peer #%d", i)
		tester.newPeer(id, protocol, hashes[i*blockCacheLimit:], blocks)
	}
	// Synchronise with the middle peer and make sure half of the blocks were retrieved
	id := fmt.Sprintf("peer #%d", targetPeers/2)
	if err := tester.sync(id, nil); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if imported := len(tester.ownBlocks); imported != len(tester.peerHashes[id]) {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, len(tester.peerHashes[id]))
	}
	// Synchronise with the best peer and make sure everything is retrieved
	if err := tester.sync("peer #0", nil); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if imported := len(tester.ownBlocks); imported != targetBlocks+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, targetBlocks+1)
	}
}

// Tests that a peer advertising an high TD doesn't get to stall the downloader
// afterwards by not sending any useful hashes.
func TestHighTDStarvationAttack61(t *testing.T) {
	tester := newTester()
	tester.newPeer("attack", eth61, []common.Hash{genesis.Hash()}, nil)
	if err := tester.sync("attack", big.NewInt(1000000)); err != errStallingPeer {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errStallingPeer)
	}
}

// Tests that misbehaving peers are disconnected, whilst behaving ones are not.
func TestHashAttackerDropping(t *testing.T) {
	// Define the disconnection requirement for individual hash fetch errors
	tests := []struct {
		result error
		drop   bool
	}{
		{nil, false},                 // Sync succeeded, all is well
		{errBusy, false},             // Sync is already in progress, no problem
		{errUnknownPeer, false},      // Peer is unknown, was already dropped, don't double drop
		{errBadPeer, true},           // Peer was deemed bad for some reason, drop it
		{errStallingPeer, true},      // Peer was detected to be stalling, drop it
		{errBannedHead, true},        // Peer's head hash is a known bad hash, drop it
		{errNoPeers, false},          // No peers to download from, soft race, no issue
		{errPendingQueue, false},     // There are blocks still cached, wait to exhaust, no issue
		{errTimeout, true},           // No hashes received in due time, drop the peer
		{errEmptyHashSet, true},      // No hashes were returned as a response, drop as it's a dead end
		{errPeersUnavailable, true},  // Nobody had the advertised blocks, drop the advertiser
		{errInvalidChain, true},      // Hash chain was detected as invalid, definitely drop
		{errCrossCheckFailed, true},  // Hash-origin failed to pass a block cross check, drop
		{errCancelHashFetch, false},  // Synchronisation was canceled, origin may be innocent, don't drop
		{errCancelBlockFetch, false}, // Synchronisation was canceled, origin may be innocent, don't drop
	}
	// Run the tests and check disconnection status
	tester := newTester()
	for i, tt := range tests {
		// Register a new peer and ensure it's presence
		id := fmt.Sprintf("test %d", i)
		if err := tester.newPeer(id, eth61, []common.Hash{genesis.Hash()}, nil); err != nil {
			t.Fatalf("test %d: failed to register new peer: %v", i, err)
		}
		if _, ok := tester.peerHashes[id]; !ok {
			t.Fatalf("test %d: registered peer not found", i)
		}
		// Simulate a synchronisation and check the required result
		tester.downloader.synchroniseMock = func(string, common.Hash) error { return tt.result }

		tester.downloader.Synchronise(id, genesis.Hash(), big.NewInt(1000))
		if _, ok := tester.peerHashes[id]; !ok != tt.drop {
			t.Errorf("test %d: peer drop mismatch for %v: have %v, want %v", i, tt.result, !ok, tt.drop)
		}
	}
}

// Tests that feeding bad blocks will result in a peer drop.
func TestBlockAttackerDropping(t *testing.T) {
	// Define the disconnection requirement for individual block import errors
	tests := []struct {
		failure bool
		drop    bool
	}{
		{true, true},
		{false, false},
	}

	// Run the tests and check disconnection status
	tester := newTester()
	for i, tt := range tests {
		// Register a new peer and ensure it's presence
		id := fmt.Sprintf("test %d", i)
		if err := tester.newPeer(id, eth61, []common.Hash{common.Hash{}}, nil); err != nil {
			t.Fatalf("test %d: failed to register new peer: %v", i, err)
		}
		if _, ok := tester.peerHashes[id]; !ok {
			t.Fatalf("test %d: registered peer not found", i)
		}
		// Assemble a good or bad block, depending of the test
		raw := core.GenerateChain(genesis, testdb, 1, nil)[0]
		if tt.failure {
			parent := types.NewBlock(&types.Header{}, nil, nil, nil)
			raw = core.GenerateChain(parent, testdb, 1, nil)[0]
		}
		block := &Block{OriginPeer: id, RawBlock: raw}

		// Simulate block processing and check the result
		tester.downloader.queue.blockCache[0] = block
		tester.downloader.process()
		if _, ok := tester.peerHashes[id]; !ok != tt.drop {
			t.Errorf("test %d: peer drop mismatch for %v: have %v, want %v", i, tt.failure, !ok, tt.drop)
		}
	}
}
