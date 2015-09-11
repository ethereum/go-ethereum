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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
)

var (
	testdb, _   = ethdb.NewMemDatabase()
	testKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddress = crypto.PubkeyToAddress(testKey.PublicKey)
	genesis     = core.GenesisBlockForTesting(testdb, testAddress, big.NewInt(1000000000))
)

// makeChain creates a chain of n blocks starting at and including parent.
// the returned hash chain is ordered head->parent. In addition, every 3rd block
// contains a transaction and every 5th an uncle to allow testing correct block
// reassembly.
func makeChain(n int, seed byte, parent *types.Block) ([]common.Hash, map[common.Hash]*types.Block) {
	blocks := core.GenerateChain(parent, testdb, n, func(i int, block *core.BlockGen) {
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

	ownHashes    []common.Hash                           // Hash chain belonging to the tester
	ownBlocks    map[common.Hash]*types.Block            // Blocks belonging to the tester
	ownChainTd   map[common.Hash]*big.Int                // Total difficulties of the blocks in the local chain
	peerHashes   map[string][]common.Hash                // Hash chain belonging to different test peers
	peerBlocks   map[string]map[common.Hash]*types.Block // Blocks belonging to different test peers
	peerChainTds map[string]map[common.Hash]*big.Int     // Total difficulties of the blocks in the peer chains
}

// newTester creates a new downloader test mocker.
func newTester() *downloadTester {
	tester := &downloadTester{
		ownHashes:    []common.Hash{genesis.Hash()},
		ownBlocks:    map[common.Hash]*types.Block{genesis.Hash(): genesis},
		ownChainTd:   map[common.Hash]*big.Int{genesis.Hash(): genesis.Difficulty()},
		peerHashes:   make(map[string][]common.Hash),
		peerBlocks:   make(map[string]map[common.Hash]*types.Block),
		peerChainTds: make(map[string]map[common.Hash]*big.Int),
	}
	tester.downloader = New(new(event.TypeMux), tester.hasBlock, tester.getBlock, tester.headBlock, tester.getTd, tester.insertChain, tester.dropPeer)

	return tester
}

// sync starts synchronizing with a remote peer, blocking until it completes.
func (dl *downloadTester) sync(id string, td *big.Int) error {
	hash := dl.peerHashes[id][0]

	// If no particular TD was requested, load from the peer's blockchain
	if td == nil {
		td = big.NewInt(1)
		if diff, ok := dl.peerChainTds[id][hash]; ok {
			td = diff
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

// getTd retrieves the block's total difficulty from the canonical chain.
func (dl *downloadTester) getTd(hash common.Hash) *big.Int {
	return dl.ownChainTd[hash]
}

// insertChain injects a new batch of blocks into the simulated chain.
func (dl *downloadTester) insertChain(blocks types.Blocks) (int, error) {
	for i, block := range blocks {
		if _, ok := dl.ownBlocks[block.ParentHash()]; !ok {
			return i, errors.New("unknown parent")
		}
		dl.ownHashes = append(dl.ownHashes, block.Hash())
		dl.ownBlocks[block.Hash()] = block
		dl.ownChainTd[block.Hash()] = dl.ownChainTd[block.ParentHash()]
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
	err := dl.downloader.RegisterPeer(id, version, hashes[0],
		dl.peerGetRelHashesFn(id, delay), dl.peerGetAbsHashesFn(id, delay), dl.peerGetBlocksFn(id, delay),
		nil, dl.peerGetAbsHeadersFn(id, delay), dl.peerGetBodiesFn(id, delay))
	if err == nil {
		// Assign the owned hashes and blocks to the peer (deep copy)
		dl.peerHashes[id] = make([]common.Hash, len(hashes))
		copy(dl.peerHashes[id], hashes)

		dl.peerBlocks[id] = make(map[common.Hash]*types.Block)
		dl.peerChainTds[id] = make(map[common.Hash]*big.Int)
		for _, hash := range hashes {
			if block, ok := blocks[hash]; ok {
				dl.peerBlocks[id][hash] = block
				if parent, ok := dl.peerBlocks[id][block.ParentHash()]; ok {
					dl.peerChainTds[id][hash] = new(big.Int).Add(block.Difficulty(), dl.peerChainTds[id][parent.Hash()])
				}
			}
		}
	}
	return err
}

// dropPeer simulates a hard peer removal from the connection pool.
func (dl *downloadTester) dropPeer(id string) {
	delete(dl.peerHashes, id)
	delete(dl.peerBlocks, id)
	delete(dl.peerChainTds, id)

	dl.downloader.UnregisterPeer(id)
}

// peerGetRelHashesFn constructs a GetHashes function associated with a specific
// peer in the download tester. The returned function can be used to retrieve
// batches of hashes from the particularly requested peer.
func (dl *downloadTester) peerGetRelHashesFn(id string, delay time.Duration) func(head common.Hash) error {
	return func(head common.Hash) error {
		time.Sleep(delay)

		// Gather the next batch of hashes
		hashes := dl.peerHashes[id]
		result := make([]common.Hash, 0, MaxHashFetch)
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
			dl.downloader.DeliverHashes61(id, result)
		}()
		return nil
	}
}

// peerGetAbsHashesFn constructs a GetHashesFromNumber function associated with
// a particular peer in the download tester. The returned function can be used to
// retrieve batches of hashes from the particularly requested peer.
func (dl *downloadTester) peerGetAbsHashesFn(id string, delay time.Duration) func(uint64, int) error {
	return func(head uint64, count int) error {
		time.Sleep(delay)

		// Gather the next batch of hashes
		hashes := dl.peerHashes[id]
		result := make([]common.Hash, 0, count)
		for i := 0; i < count && len(hashes)-int(head)-1-i >= 0; i++ {
			result = append(result, hashes[len(hashes)-int(head)-1-i])
		}
		// Delay delivery a bit to allow attacks to unfold
		go func() {
			time.Sleep(time.Millisecond)
			dl.downloader.DeliverHashes61(id, result)
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
		go dl.downloader.DeliverBlocks61(id, result)

		return nil
	}
}

// peerGetAbsHeadersFn constructs a GetBlockHeaders function based on a numbered
// origin; associated with a particular peer in the download tester. The returned
// function can be used to retrieve batches of headers from the particular peer.
func (dl *downloadTester) peerGetAbsHeadersFn(id string, delay time.Duration) func(uint64, int, int, bool) error {
	return func(origin uint64, amount int, skip int, reverse bool) error {
		time.Sleep(delay)

		// Gather the next batch of hashes
		hashes := dl.peerHashes[id]
		blocks := dl.peerBlocks[id]
		result := make([]*types.Header, 0, amount)
		for i := 0; i < amount && len(hashes)-int(origin)-1-i >= 0; i++ {
			if block, ok := blocks[hashes[len(hashes)-int(origin)-1-i]]; ok {
				result = append(result, block.Header())
			}
		}
		// Delay delivery a bit to allow attacks to unfold
		go func() {
			time.Sleep(time.Millisecond)
			dl.downloader.DeliverHeaders(id, result)
		}()
		return nil
	}
}

// peerGetBodiesFn constructs a getBlockBodies method associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of block bodies from the particularly requested peer.
func (dl *downloadTester) peerGetBodiesFn(id string, delay time.Duration) func([]common.Hash) error {
	return func(hashes []common.Hash) error {
		time.Sleep(delay)
		blocks := dl.peerBlocks[id]

		transactions := make([][]*types.Transaction, 0, len(hashes))
		uncles := make([][]*types.Header, 0, len(hashes))

		for _, hash := range hashes {
			if block, ok := blocks[hash]; ok {
				transactions = append(transactions, block.Transactions())
				uncles = append(uncles, block.Uncles())
			}
		}
		go dl.downloader.DeliverBodies(id, transactions, uncles)

		return nil
	}
}

// Tests that simple synchronization against a canonical chain works correctly.
// In this test common ancestor lookup should be short circuited and not require
// binary searching.
func TestCanonicalSynchronisation61(t *testing.T) { testCanonicalSynchronisation(t, 61) }
func TestCanonicalSynchronisation62(t *testing.T) { testCanonicalSynchronisation(t, 62) }
func TestCanonicalSynchronisation63(t *testing.T) { testCanonicalSynchronisation(t, 63) }
func TestCanonicalSynchronisation64(t *testing.T) { testCanonicalSynchronisation(t, 64) }

func testCanonicalSynchronisation(t *testing.T, protocol int) {
	// Create a small enough block chain to download
	targetBlocks := blockCacheLimit - 15
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	tester := newTester()
	tester.newPeer("peer", protocol, hashes, blocks)

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
func TestThrottling61(t *testing.T) { testThrottling(t, 61) }
func TestThrottling62(t *testing.T) { testThrottling(t, 62) }
func TestThrottling63(t *testing.T) { testThrottling(t, 63) }
func TestThrottling64(t *testing.T) { testThrottling(t, 64) }

func testThrottling(t *testing.T, protocol int) {
	// Create a long block chain to download and the tester
	targetBlocks := 8 * blockCacheLimit
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	tester := newTester()
	tester.newPeer("peer", protocol, hashes, blocks)

	// Wrap the importer to allow stepping
	blocked, proceed := uint32(0), make(chan struct{})
	tester.downloader.chainInsertHook = func(blocks []*Block) {
		atomic.StoreUint32(&blocked, uint32(len(blocks)))
		<-proceed
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
		for start := time.Now(); time.Since(start) < time.Second; {
			time.Sleep(25 * time.Millisecond)

			cached = len(tester.downloader.queue.blockPool)
			if cached == blockCacheLimit || len(tester.ownBlocks)+cached+int(atomic.LoadUint32(&blocked)) == targetBlocks+1 {
				break
			}
		}
		// Make sure we filled up the cache, then exhaust it
		time.Sleep(25 * time.Millisecond) // give it a chance to screw up
		if cached != blockCacheLimit && len(tester.ownBlocks)+cached+int(atomic.LoadUint32(&blocked)) != targetBlocks+1 {
			t.Fatalf("block count mismatch: have %v, want %v (owned %v, target %v)", cached, blockCacheLimit, len(tester.ownBlocks), targetBlocks+1)
		}
		// Permit the blocked blocks to import
		if atomic.LoadUint32(&blocked) > 0 {
			atomic.StoreUint32(&blocked, uint32(0))
			proceed <- struct{}{}
		}
	}
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
func TestForkedSynchronisation61(t *testing.T) { testForkedSynchronisation(t, 61) }
func TestForkedSynchronisation62(t *testing.T) { testForkedSynchronisation(t, 62) }
func TestForkedSynchronisation63(t *testing.T) { testForkedSynchronisation(t, 63) }
func TestForkedSynchronisation64(t *testing.T) { testForkedSynchronisation(t, 64) }

func testForkedSynchronisation(t *testing.T, protocol int) {
	// Create a long enough forked chain
	common, fork := MaxHashFetch, 2*MaxHashFetch
	hashesA, hashesB, blocksA, blocksB := makeChainFork(common+fork, fork, genesis)

	tester := newTester()
	tester.newPeer("fork A", protocol, hashesA, blocksA)
	tester.newPeer("fork B", protocol, hashesB, blocksB)

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
func TestInactiveDownloader61(t *testing.T) {
	tester := newTester()

	// Check that neither hashes nor blocks are accepted
	if err := tester.downloader.DeliverHashes61("bad peer", []common.Hash{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
	if err := tester.downloader.DeliverBlocks61("bad peer", []*types.Block{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
}

// Tests that an inactive downloader will not accept incoming block headers and bodies.
func TestInactiveDownloader62(t *testing.T) {
	tester := newTester()

	// Check that neither block headers nor bodies are accepted
	if err := tester.downloader.DeliverHeaders("bad peer", []*types.Header{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
	if err := tester.downloader.DeliverBodies("bad peer", [][]*types.Transaction{}, [][]*types.Header{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
}

// Tests that a canceled download wipes all previously accumulated state.
func TestCancel61(t *testing.T) { testCancel(t, 61) }
func TestCancel62(t *testing.T) { testCancel(t, 62) }
func TestCancel63(t *testing.T) { testCancel(t, 63) }
func TestCancel64(t *testing.T) { testCancel(t, 64) }

func testCancel(t *testing.T, protocol int) {
	// Create a small enough block chain to download and the tester
	targetBlocks := blockCacheLimit - 15
	if targetBlocks >= MaxHashFetch {
		targetBlocks = MaxHashFetch - 15
	}
	if targetBlocks >= MaxHeaderFetch {
		targetBlocks = MaxHeaderFetch - 15
	}
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	tester := newTester()
	tester.newPeer("peer", protocol, hashes, blocks)

	// Make sure canceling works with a pristine downloader
	tester.downloader.cancel()
	downloading, importing := tester.downloader.queue.Size()
	if downloading > 0 || importing > 0 {
		t.Errorf("download or import count mismatch: %d downloading, %d importing, want 0", downloading, importing)
	}
	// Synchronise with the peer, but cancel afterwards
	if err := tester.sync("peer", nil); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	tester.downloader.cancel()
	downloading, importing = tester.downloader.queue.Size()
	if downloading > 0 || importing > 0 {
		t.Errorf("download or import count mismatch: %d downloading, %d importing, want 0", downloading, importing)
	}
}

// Tests that synchronisation from multiple peers works as intended (multi thread sanity test).
func TestMultiSynchronisation61(t *testing.T) { testMultiSynchronisation(t, 61) }
func TestMultiSynchronisation62(t *testing.T) { testMultiSynchronisation(t, 62) }
func TestMultiSynchronisation63(t *testing.T) { testMultiSynchronisation(t, 63) }
func TestMultiSynchronisation64(t *testing.T) { testMultiSynchronisation(t, 64) }

func testMultiSynchronisation(t *testing.T, protocol int) {
	// Create various peers with various parts of the chain
	targetPeers := 8
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

// Tests that if a block is empty (i.e. header only), no body request should be
// made, and instead the header should be assembled into a whole block in itself.
func TestEmptyBlockShortCircuit62(t *testing.T) { testEmptyBlockShortCircuit(t, 62) }
func TestEmptyBlockShortCircuit63(t *testing.T) { testEmptyBlockShortCircuit(t, 63) }
func TestEmptyBlockShortCircuit64(t *testing.T) { testEmptyBlockShortCircuit(t, 64) }

func testEmptyBlockShortCircuit(t *testing.T, protocol int) {
	// Create a small enough block chain to download
	targetBlocks := blockCacheLimit - 15
	hashes, blocks := makeChain(targetBlocks, 0, genesis)

	tester := newTester()
	tester.newPeer("peer", protocol, hashes, blocks)

	// Instrument the downloader to signal body requests
	requested := int32(0)
	tester.downloader.bodyFetchHook = func(headers []*types.Header) {
		atomic.AddInt32(&requested, int32(len(headers)))
	}
	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.sync("peer", nil); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if imported := len(tester.ownBlocks); imported != targetBlocks+1 {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, targetBlocks+1)
	}
	// Validate the number of block bodies that should have been requested
	needed := 0
	for _, block := range blocks {
		if block != genesis && (len(block.Transactions()) > 0 || len(block.Uncles()) > 0) {
			needed++
		}
	}
	if int(requested) != needed {
		t.Fatalf("block body retrieval count mismatch: have %v, want %v", requested, needed)
	}
}

// Tests that if a peer sends an invalid body for a requested block, it gets
// dropped immediately by the downloader.
func TestInvalidBlockBodyAttack62(t *testing.T) { testInvalidBlockBodyAttack(t, 62) }
func TestInvalidBlockBodyAttack63(t *testing.T) { testInvalidBlockBodyAttack(t, 63) }
func TestInvalidBlockBodyAttack64(t *testing.T) { testInvalidBlockBodyAttack(t, 64) }

func testInvalidBlockBodyAttack(t *testing.T, protocol int) {
	// Create two peers, one feeding invalid block bodies
	targetBlocks := 4*blockCacheLimit - 15
	hashes, validBlocks := makeChain(targetBlocks, 0, genesis)

	invalidBlocks := make(map[common.Hash]*types.Block)
	for hash, block := range validBlocks {
		invalidBlocks[hash] = types.NewBlockWithHeader(block.Header())
	}

	tester := newTester()
	tester.newPeer("valid", protocol, hashes, validBlocks)
	tester.newPeer("attack", protocol, hashes, invalidBlocks)

	// Synchronise with the valid peer (will pull contents from the attacker too)
	if err := tester.sync("valid", nil); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if imported := len(tester.ownBlocks); imported != len(hashes) {
		t.Fatalf("synchronised block mismatch: have %v, want %v", imported, len(hashes))
	}
	// Make sure the attacker was detected and dropped in the mean time
	if _, ok := tester.peerHashes["attack"]; ok {
		t.Fatalf("block body attacker not detected/dropped")
	}
}

// Tests that a peer advertising an high TD doesn't get to stall the downloader
// afterwards by not sending any useful hashes.
func TestHighTDStarvationAttack61(t *testing.T) { testHighTDStarvationAttack(t, 61) }
func TestHighTDStarvationAttack62(t *testing.T) { testHighTDStarvationAttack(t, 62) }
func TestHighTDStarvationAttack63(t *testing.T) { testHighTDStarvationAttack(t, 63) }
func TestHighTDStarvationAttack64(t *testing.T) { testHighTDStarvationAttack(t, 64) }

func testHighTDStarvationAttack(t *testing.T, protocol int) {
	tester := newTester()
	hashes, blocks := makeChain(0, 0, genesis)

	tester.newPeer("attack", protocol, []common.Hash{hashes[0]}, blocks)
	if err := tester.sync("attack", big.NewInt(1000000)); err != errStallingPeer {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errStallingPeer)
	}
}

// Tests that misbehaving peers are disconnected, whilst behaving ones are not.
func TestBlockHeaderAttackerDropping61(t *testing.T) { testBlockHeaderAttackerDropping(t, 61) }
func TestBlockHeaderAttackerDropping62(t *testing.T) { testBlockHeaderAttackerDropping(t, 62) }
func TestBlockHeaderAttackerDropping63(t *testing.T) { testBlockHeaderAttackerDropping(t, 63) }
func TestBlockHeaderAttackerDropping64(t *testing.T) { testBlockHeaderAttackerDropping(t, 64) }

func testBlockHeaderAttackerDropping(t *testing.T, protocol int) {
	// Define the disconnection requirement for individual hash fetch errors
	tests := []struct {
		result error
		drop   bool
	}{
		{nil, false},                  // Sync succeeded, all is well
		{errBusy, false},              // Sync is already in progress, no problem
		{errUnknownPeer, false},       // Peer is unknown, was already dropped, don't double drop
		{errBadPeer, true},            // Peer was deemed bad for some reason, drop it
		{errStallingPeer, true},       // Peer was detected to be stalling, drop it
		{errNoPeers, false},           // No peers to download from, soft race, no issue
		{errPendingQueue, false},      // There are blocks still cached, wait to exhaust, no issue
		{errTimeout, true},            // No hashes received in due time, drop the peer
		{errEmptyHashSet, true},       // No hashes were returned as a response, drop as it's a dead end
		{errEmptyHeaderSet, true},     // No headers were returned as a response, drop as it's a dead end
		{errPeersUnavailable, true},   // Nobody had the advertised blocks, drop the advertiser
		{errInvalidChain, true},       // Hash chain was detected as invalid, definitely drop
		{errInvalidBody, false},       // A bad peer was detected, but not the sync origin
		{errCancelHashFetch, false},   // Synchronisation was canceled, origin may be innocent, don't drop
		{errCancelBlockFetch, false},  // Synchronisation was canceled, origin may be innocent, don't drop
		{errCancelHeaderFetch, false}, // Synchronisation was canceled, origin may be innocent, don't drop
		{errCancelBodyFetch, false},   // Synchronisation was canceled, origin may be innocent, don't drop
	}
	// Run the tests and check disconnection status
	tester := newTester()
	for i, tt := range tests {
		// Register a new peer and ensure it's presence
		id := fmt.Sprintf("test %d", i)
		if err := tester.newPeer(id, protocol, []common.Hash{genesis.Hash()}, nil); err != nil {
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
func TestBlockBodyAttackerDropping61(t *testing.T) { testBlockBodyAttackerDropping(t, 61) }
func TestBlockBodyAttackerDropping62(t *testing.T) { testBlockBodyAttackerDropping(t, 62) }
func TestBlockBodyAttackerDropping63(t *testing.T) { testBlockBodyAttackerDropping(t, 63) }
func TestBlockBodyAttackerDropping64(t *testing.T) { testBlockBodyAttackerDropping(t, 64) }

func testBlockBodyAttackerDropping(t *testing.T, protocol int) {
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
		if err := tester.newPeer(id, protocol, []common.Hash{common.Hash{}}, nil); err != nil {
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
