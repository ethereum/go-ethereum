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

package downloader

import (
	"crypto/rand"
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
func (dl *downloadTester) sync(id string) error {
	err := dl.downloader.synchronise(id, dl.peerHashes[id][0])
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
	// If the simulated peer runs eth/60, this message is not supported
	if version == eth60 {
		return func(uint64, int) error { return nil }
	}
	// Otherwise create a method to request the blocks by number
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

// Tests that simple synchronization, without throttling from a good peer works.
func TestSynchronisation60(t *testing.T) {
	// Create a small enough block chain to download and the tester
	targetBlocks := blockCacheLimit - 15
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	tester := newTester()
	tester.newPeer("peer", eth60, hashes, blocks)

	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.sync("peer"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if imported := len(tester.ownBlocks); imported != targetBlocks+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, targetBlocks+1)
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
	if err := tester.sync("peer"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if imported := len(tester.ownBlocks); imported != targetBlocks+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, targetBlocks+1)
	}
}

// Tests that if a large batch of blocks are being downloaded, it is throttled
// until the cached blocks are retrieved.
func TestThrottling60(t *testing.T) { testThrottling(t, eth60) }
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
		errc <- tester.sync("peer")
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
	if err := tester.sync("fork A"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if imported := len(tester.ownBlocks); imported != common+fork+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, common+fork+1)
	}
	// Synchronise with the second peer and make sure that fork is pulled too
	if err := tester.sync("fork B"); err != nil {
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
func TestCancel60(t *testing.T) { testCancel(t, eth60) }
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
	if err := tester.sync("peer"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	tester.downloader.cancel()
	hashCount, blockCount = tester.downloader.queue.Size()
	if hashCount > 0 || blockCount > 0 {
		t.Errorf("block or hash count mismatch: %d hashes, %d blocks, want 0", hashCount, blockCount)
	}
}

// Tests that synchronisation from multiple peers works as intended (multi thread sanity test).
func TestMultiSynchronisation60(t *testing.T) { testMultiSynchronisation(t, eth60) }
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
	if err := tester.sync(id); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if imported := len(tester.ownBlocks); imported != len(tester.peerHashes[id]) {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, len(tester.peerHashes[id]))
	}
	// Synchronise with the best peer and make sure everything is retrieved
	if err := tester.sync("peer #0"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if imported := len(tester.ownBlocks); imported != targetBlocks+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, targetBlocks+1)
	}
}

// Tests that synchronising with a peer who's very slow at network IO does not
// stall the other peers in the system.
func TestSlowSynchronisation60(t *testing.T) {
	tester := newTester()

	// Create a batch of blocks, with a slow and a full speed peer
	targetCycles := 2
	targetBlocks := targetCycles*blockCacheLimit - 15
	targetIODelay := time.Second
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	tester.newSlowPeer("fast", eth60, hashes, blocks, 0)
	tester.newSlowPeer("slow", eth60, hashes, blocks, targetIODelay)

	// Try to sync with the peers (pull hashes from fast)
	start := time.Now()
	if err := tester.sync("fast"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if imported := len(tester.ownBlocks); imported != targetBlocks+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, targetBlocks+1)
	}
	// Check that the slow peer got hit at most once per block-cache-size import
	limit := time.Duration(targetCycles+1) * targetIODelay
	if delay := time.Since(start); delay >= limit {
		t.Fatalf("synchronisation exceeded delay limit: have %v, want %v", delay, limit)
	}
}

// Tests that if a peer returns an invalid chain with a block pointing to a non-
// existing parent, it is correctly detected and handled.
func TestNonExistingParentAttack60(t *testing.T) {
	tester := newTester()

	// Forge a single-link chain with a forged header
	hashes, blocks := makeChain(1, 0, genesis)
	tester.newPeer("valid", eth60, hashes, blocks)

	wrongblock := types.NewBlock(&types.Header{}, nil, nil, nil)
	wrongblock.Td = blocks[hashes[0]].Td
	hashes, blocks = makeChain(1, 0, wrongblock)
	tester.newPeer("attack", eth60, hashes, blocks)

	// Try and sync with the malicious node and check that it fails
	if err := tester.sync("attack"); err == nil {
		t.Fatalf("block synchronization succeeded")
	}
	if tester.hasBlock(hashes[0]) {
		t.Fatalf("tester accepted unknown-parent block: %v", blocks[hashes[0]])
	}
	// Try to synchronize with the valid chain and make sure it succeeds
	if err := tester.sync("valid"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if !tester.hasBlock(tester.peerHashes["valid"][0]) {
		t.Fatalf("tester didn't accept known-parent block: %v", tester.peerBlocks["valid"][hashes[0]])
	}
}

// Tests that if a malicious peers keeps sending us repeating hashes, we don't
// loop indefinitely.
func TestRepeatingHashAttack60(t *testing.T) { // TODO: Is this thing valid??
	tester := newTester()

	// Create a valid chain, but drop the last link
	hashes, blocks := makeChain(blockCacheLimit, 0, genesis)
	tester.newPeer("valid", eth60, hashes, blocks)
	tester.newPeer("attack", eth60, hashes[:len(hashes)-1], blocks)

	// Try and sync with the malicious node
	errc := make(chan error)
	go func() {
		errc <- tester.sync("attack")
	}()
	// Make sure that syncing returns and does so with a failure
	select {
	case <-time.After(time.Second):
		t.Fatalf("synchronisation blocked")
	case err := <-errc:
		if err == nil {
			t.Fatalf("synchronisation succeeded")
		}
	}
	// Ensure that a valid chain can still pass sync
	if err := tester.sync("valid"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
}

// Tests that if a malicious peers returns a non-existent block hash, it should
// eventually time out and the sync reattempted.
func TestNonExistingBlockAttack60(t *testing.T) {
	tester := newTester()

	// Create a valid chain, but forge the last link
	hashes, blocks := makeChain(blockCacheLimit, 0, genesis)
	tester.newPeer("valid", eth60, hashes, blocks)

	hashes[len(hashes)/2] = common.Hash{}
	tester.newPeer("attack", eth60, hashes, blocks)

	// Try and sync with the malicious node and check that it fails
	if err := tester.sync("attack"); err != errPeersUnavailable {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errPeersUnavailable)
	}
	// Ensure that a valid chain can still pass sync
	if err := tester.sync("valid"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
}

// Tests that if a malicious peer is returning hashes in a weird order, that the
// sync throttler doesn't choke on them waiting for the valid blocks.
func TestInvalidHashOrderAttack60(t *testing.T) {
	tester := newTester()

	// Create a valid long chain, but reverse some hashes within
	hashes, blocks := makeChain(4*blockCacheLimit, 0, genesis)
	tester.newPeer("valid", eth60, hashes, blocks)

	chunk1 := make([]common.Hash, blockCacheLimit)
	chunk2 := make([]common.Hash, blockCacheLimit)
	copy(chunk1, hashes[blockCacheLimit:2*blockCacheLimit])
	copy(chunk2, hashes[2*blockCacheLimit:3*blockCacheLimit])

	copy(hashes[2*blockCacheLimit:], chunk1)
	copy(hashes[blockCacheLimit:], chunk2)
	tester.newPeer("attack", eth60, hashes, blocks)

	// Try and sync with the malicious node and check that it fails
	if err := tester.sync("attack"); err != errInvalidChain {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errInvalidChain)
	}
	// Ensure that a valid chain can still pass sync
	if err := tester.sync("valid"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
}

// Tests that if a malicious peer makes up a random hash chain and tries to push
// indefinitely, it actually gets caught with it.
func TestMadeupHashChainAttack60(t *testing.T) {
	tester := newTester()
	blockSoftTTL = 100 * time.Millisecond
	crossCheckCycle = 25 * time.Millisecond

	// Create a long chain of hashes without backing blocks
	hashes, blocks := makeChain(4*blockCacheLimit, 0, genesis)

	randomHashes := make([]common.Hash, 1024*blockCacheLimit)
	for i := range randomHashes {
		rand.Read(randomHashes[i][:])
	}

	tester.newPeer("valid", eth60, hashes, blocks)
	tester.newPeer("attack", eth60, randomHashes, nil)

	// Try and sync with the malicious node and check that it fails
	if err := tester.sync("attack"); err != errCrossCheckFailed {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errCrossCheckFailed)
	}
	// Ensure that a valid chain can still pass sync
	if err := tester.sync("valid"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
}

// Tests that if a malicious peer makes up a random hash chain, and tries to push
// indefinitely, one hash at a time, it actually gets caught with it. The reason
// this is separate from the classical made up chain attack is that sending hashes
// one by one prevents reliable block/parent verification.
func TestMadeupHashChainDrippingAttack60(t *testing.T) {
	// Create a random chain of hashes to drip
	randomHashes := make([]common.Hash, 16*blockCacheLimit)
	for i := range randomHashes {
		rand.Read(randomHashes[i][:])
	}
	randomHashes[len(randomHashes)-1] = genesis.Hash()
	tester := newTester()

	// Try and sync with the attacker, one hash at a time
	tester.maxHashFetch = 1
	tester.newPeer("attack", eth60, randomHashes, nil)
	if err := tester.sync("attack"); err != errStallingPeer {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errStallingPeer)
	}
}

// Tests that if a malicious peer makes up a random block chain, and tried to
// push indefinitely, it actually gets caught with it.
func TestMadeupBlockChainAttack60(t *testing.T) {
	defaultBlockTTL := blockSoftTTL
	defaultCrossCheckCycle := crossCheckCycle

	blockSoftTTL = 100 * time.Millisecond
	crossCheckCycle = 25 * time.Millisecond

	// Create a long chain of blocks and simulate an invalid chain by dropping every second
	hashes, blocks := makeChain(16*blockCacheLimit, 0, genesis)
	gapped := make([]common.Hash, len(hashes)/2)
	for i := 0; i < len(gapped); i++ {
		gapped[i] = hashes[2*i]
	}
	// Try and sync with the malicious node and check that it fails
	tester := newTester()
	tester.newPeer("attack", eth60, gapped, blocks)
	if err := tester.sync("attack"); err != errCrossCheckFailed {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errCrossCheckFailed)
	}
	// Ensure that a valid chain can still pass sync
	blockSoftTTL = defaultBlockTTL
	crossCheckCycle = defaultCrossCheckCycle

	tester.newPeer("valid", eth60, hashes, blocks)
	if err := tester.sync("valid"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
}

// Tests that if one/multiple malicious peers try to feed a banned blockchain to
// the downloader, it will not keep refetching the same chain indefinitely, but
// gradually block pieces of it, until its head is also blocked.
func TestBannedChainStarvationAttack60(t *testing.T) {
	n := 8 * blockCacheLimit
	fork := n/2 - 23
	hashes, forkHashes, blocks, forkBlocks := makeChainFork(n, fork, genesis)

	// Create the tester and ban the selected hash.
	tester := newTester()
	tester.downloader.banned.Add(forkHashes[fork-1])
	tester.newPeer("valid", eth60, hashes, blocks)
	tester.newPeer("attack", eth60, forkHashes, forkBlocks)

	// Iteratively try to sync, and verify that the banned hash list grows until
	// the head of the invalid chain is blocked too.
	for banned := tester.downloader.banned.Size(); ; {
		// Try to sync with the attacker, check hash chain failure
		if err := tester.sync("attack"); err != errInvalidChain {
			if tester.downloader.banned.Has(forkHashes[0]) && err == errBannedHead {
				break
			}
			t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errInvalidChain)
		}
		// Check that the ban list grew with at least 1 new item, or all banned
		bans := tester.downloader.banned.Size()
		if bans < banned+1 {
			t.Fatalf("ban count mismatch: have %v, want %v+", bans, banned+1)
		}
		banned = bans
	}
	// Check that after banning an entire chain, bad peers get dropped
	if err := tester.newPeer("new attacker", eth60, forkHashes, forkBlocks); err != errBannedHead {
		t.Fatalf("peer registration mismatch: have %v, want %v", err, errBannedHead)
	}
	if peer := tester.downloader.peers.Peer("new attacker"); peer != nil {
		t.Fatalf("banned attacker registered: %v", peer)
	}
	// Ensure that a valid chain can still pass sync
	if err := tester.sync("valid"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
}

// Tests that if a peer sends excessively many/large invalid chains that are
// gradually banned, it will have an upper limit on the consumed memory and also
// the origin bad hashes will not be evacuated.
func TestBannedChainMemoryExhaustionAttack60(t *testing.T) {
	// Construct a banned chain with more chunks than the ban limit
	n := 8 * blockCacheLimit
	fork := n/2 - 23
	hashes, forkHashes, blocks, forkBlocks := makeChainFork(n, fork, genesis)

	// Create the tester and ban the root hash of the fork.
	tester := newTester()
	tester.downloader.banned.Add(forkHashes[fork-1])

	// Reduce the test size a bit
	defaultMaxBlockFetch := MaxBlockFetch
	defaultMaxBannedHashes := maxBannedHashes

	MaxBlockFetch = 4
	maxBannedHashes = 256

	tester.newPeer("valid", eth60, hashes, blocks)
	tester.newPeer("attack", eth60, forkHashes, forkBlocks)

	// Iteratively try to sync, and verify that the banned hash list grows until
	// the head of the invalid chain is blocked too.
	for {
		// Try to sync with the attacker, check hash chain failure
		if err := tester.sync("attack"); err != errInvalidChain {
			t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errInvalidChain)
		}
		// Short circuit if the entire chain was banned.
		if tester.downloader.banned.Has(forkHashes[0]) {
			break
		}
		// Otherwise ensure we never exceed the memory allowance and the hard coded bans are untouched
		if bans := tester.downloader.banned.Size(); bans > maxBannedHashes {
			t.Fatalf("ban cap exceeded: have %v, want max %v", bans, maxBannedHashes)
		}
		for hash, _ := range core.BadHashes {
			if !tester.downloader.banned.Has(hash) {
				t.Fatalf("hard coded ban evacuated: %x", hash)
			}
		}
	}
	// Ensure that a valid chain can still pass sync
	MaxBlockFetch = defaultMaxBlockFetch
	maxBannedHashes = defaultMaxBannedHashes

	if err := tester.sync("valid"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
}

// Tests a corner case (potential attack) where a peer delivers both good as well
// as unrequested blocks to a hash request. This may trigger a different code
// path than the fully correct or fully invalid delivery, potentially causing
// internal state problems
//
// No, don't delete this test, it actually did happen!
func TestOverlappingDeliveryAttack60(t *testing.T) {
	// Create an arbitrary batch of blocks ( < cache-size not to block)
	targetBlocks := blockCacheLimit - 23
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	// Register an attacker that always returns non-requested blocks too
	tester := newTester()
	tester.newPeer("attack", eth60, hashes, blocks)

	rawGetBlocks := tester.downloader.peers.Peer("attack").getBlocks
	tester.downloader.peers.Peer("attack").getBlocks = func(request []common.Hash) error {
		// Add a non requested hash the screw the delivery (genesis should be fine)
		return rawGetBlocks(append(request, hashes[0]))
	}
	// Test that synchronisation can complete, check for import success
	if err := tester.sync("attack"); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	start := time.Now()
	for len(tester.ownHashes) != len(hashes) && time.Since(start) < time.Second {
		time.Sleep(50 * time.Millisecond)
	}
	if len(tester.ownHashes) != len(hashes) {
		t.Fatalf("chain length mismatch: have %v, want %v", len(tester.ownHashes), len(hashes))
	}
}

// Tests that a peer advertising an high TD doesn't get to stall the downloader
// afterwards by not sending any useful hashes.
func TestHighTDStarvationAttack61(t *testing.T) {
	tester := newTester()
	tester.newPeer("attack", eth61, []common.Hash{genesis.Hash()}, nil)
	if err := tester.sync("attack"); err != errStallingPeer {
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
		if err := tester.newPeer(id, eth60, []common.Hash{genesis.Hash()}, nil); err != nil {
			t.Fatalf("test %d: failed to register new peer: %v", i, err)
		}
		if _, ok := tester.peerHashes[id]; !ok {
			t.Fatalf("test %d: registered peer not found", i)
		}
		// Simulate a synchronisation and check the required result
		tester.downloader.synchroniseMock = func(string, common.Hash) error { return tt.result }

		tester.downloader.Synchronise(id, genesis.Hash())
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
		if err := tester.newPeer(id, eth60, []common.Hash{common.Hash{}}, nil); err != nil {
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
