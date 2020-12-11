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
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/trie"
)

// Reduce some of the parameters to make the tester faster.
func init() {
	fullMaxForkAncestry = 10000
	lightMaxForkAncestry = 10000
	blockCacheMaxItems = 1024
	fsHeaderContCheck = 500 * time.Millisecond
}

// downloadTester is a test simulator for mocking out local block chain.
type downloadTester struct {
	downloader *Downloader

	genesis *types.Block   // Genesis blocks used by the tester and peers
	stateDb ethdb.Database // Database used by the tester for syncing from peers
	peerDb  ethdb.Database // Database of the peers containing all data
	peers   map[string]*downloadTesterPeer

	ownHashes   []common.Hash                  // Hash chain belonging to the tester
	ownHeaders  map[common.Hash]*types.Header  // Headers belonging to the tester
	ownBlocks   map[common.Hash]*types.Block   // Blocks belonging to the tester
	ownReceipts map[common.Hash]types.Receipts // Receipts belonging to the tester
	ownChainTd  map[common.Hash]*big.Int       // Total difficulties of the blocks in the local chain

	ancientHeaders  map[common.Hash]*types.Header  // Ancient headers belonging to the tester
	ancientBlocks   map[common.Hash]*types.Block   // Ancient blocks belonging to the tester
	ancientReceipts map[common.Hash]types.Receipts // Ancient receipts belonging to the tester
	ancientChainTd  map[common.Hash]*big.Int       // Ancient total difficulties of the blocks in the local chain

	lock sync.RWMutex
}

// newTester creates a new downloader test mocker.
func newTester() *downloadTester {
	tester := &downloadTester{
		genesis:     testGenesis,
		peerDb:      testDB,
		peers:       make(map[string]*downloadTesterPeer),
		ownHashes:   []common.Hash{testGenesis.Hash()},
		ownHeaders:  map[common.Hash]*types.Header{testGenesis.Hash(): testGenesis.Header()},
		ownBlocks:   map[common.Hash]*types.Block{testGenesis.Hash(): testGenesis},
		ownReceipts: map[common.Hash]types.Receipts{testGenesis.Hash(): nil},
		ownChainTd:  map[common.Hash]*big.Int{testGenesis.Hash(): testGenesis.Difficulty()},

		// Initialize ancient store with test genesis block
		ancientHeaders:  map[common.Hash]*types.Header{testGenesis.Hash(): testGenesis.Header()},
		ancientBlocks:   map[common.Hash]*types.Block{testGenesis.Hash(): testGenesis},
		ancientReceipts: map[common.Hash]types.Receipts{testGenesis.Hash(): nil},
		ancientChainTd:  map[common.Hash]*big.Int{testGenesis.Hash(): testGenesis.Difficulty()},
	}
	tester.stateDb = rawdb.NewMemoryDatabase()
	tester.stateDb.Put(testGenesis.Root().Bytes(), []byte{0x00})

	tester.downloader = New(0, tester.stateDb, trie.NewSyncBloom(1, tester.stateDb), new(event.TypeMux), tester, nil, tester.dropPeer)
	return tester
}

// terminate aborts any operations on the embedded downloader and releases all
// held resources.
func (dl *downloadTester) terminate() {
	dl.downloader.Terminate()
}

// sync starts synchronizing with a remote peer, blocking until it completes.
func (dl *downloadTester) sync(id string, td *big.Int, mode SyncMode) error {
	dl.lock.RLock()
	hash := dl.peers[id].chain.headBlock().Hash()
	// If no particular TD was requested, load from the peer's blockchain
	if td == nil {
		td = dl.peers[id].chain.td(hash)
	}
	dl.lock.RUnlock()

	// Synchronise with the chosen peer and ensure proper cleanup afterwards
	err := dl.downloader.synchronise(id, hash, td, mode)
	select {
	case <-dl.downloader.cancelCh:
		// Ok, downloader fully cancelled after sync cycle
	default:
		// Downloader is still accepting packets, can block a peer up
		panic("downloader active post sync cycle") // panic will be caught by tester
	}
	return err
}

// HasHeader checks if a header is present in the testers canonical chain.
func (dl *downloadTester) HasHeader(hash common.Hash, number uint64) bool {
	return dl.GetHeaderByHash(hash) != nil
}

// HasBlock checks if a block is present in the testers canonical chain.
func (dl *downloadTester) HasBlock(hash common.Hash, number uint64) bool {
	return dl.GetBlockByHash(hash) != nil
}

// HasFastBlock checks if a block is present in the testers canonical chain.
func (dl *downloadTester) HasFastBlock(hash common.Hash, number uint64) bool {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if _, ok := dl.ancientReceipts[hash]; ok {
		return true
	}
	_, ok := dl.ownReceipts[hash]
	return ok
}

// GetHeader retrieves a header from the testers canonical chain.
func (dl *downloadTester) GetHeaderByHash(hash common.Hash) *types.Header {
	dl.lock.RLock()
	defer dl.lock.RUnlock()
	return dl.getHeaderByHash(hash)
}

// getHeaderByHash returns the header if found either within ancients or own blocks)
// This method assumes that the caller holds at least the read-lock (dl.lock)
func (dl *downloadTester) getHeaderByHash(hash common.Hash) *types.Header {
	header := dl.ancientHeaders[hash]
	if header != nil {
		return header
	}
	return dl.ownHeaders[hash]
}

// GetBlock retrieves a block from the testers canonical chain.
func (dl *downloadTester) GetBlockByHash(hash common.Hash) *types.Block {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	block := dl.ancientBlocks[hash]
	if block != nil {
		return block
	}
	return dl.ownBlocks[hash]
}

// CurrentHeader retrieves the current head header from the canonical chain.
func (dl *downloadTester) CurrentHeader() *types.Header {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	for i := len(dl.ownHashes) - 1; i >= 0; i-- {
		if header := dl.ancientHeaders[dl.ownHashes[i]]; header != nil {
			return header
		}
		if header := dl.ownHeaders[dl.ownHashes[i]]; header != nil {
			return header
		}
	}
	return dl.genesis.Header()
}

// CurrentBlock retrieves the current head block from the canonical chain.
func (dl *downloadTester) CurrentBlock() *types.Block {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	for i := len(dl.ownHashes) - 1; i >= 0; i-- {
		if block := dl.ancientBlocks[dl.ownHashes[i]]; block != nil {
			if _, err := dl.stateDb.Get(block.Root().Bytes()); err == nil {
				return block
			}
			return block
		}
		if block := dl.ownBlocks[dl.ownHashes[i]]; block != nil {
			if _, err := dl.stateDb.Get(block.Root().Bytes()); err == nil {
				return block
			}
		}
	}
	return dl.genesis
}

// CurrentFastBlock retrieves the current head fast-sync block from the canonical chain.
func (dl *downloadTester) CurrentFastBlock() *types.Block {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	for i := len(dl.ownHashes) - 1; i >= 0; i-- {
		if block := dl.ancientBlocks[dl.ownHashes[i]]; block != nil {
			return block
		}
		if block := dl.ownBlocks[dl.ownHashes[i]]; block != nil {
			return block
		}
	}
	return dl.genesis
}

// FastSyncCommitHead manually sets the head block to a given hash.
func (dl *downloadTester) FastSyncCommitHead(hash common.Hash) error {
	// For now only check that the state trie is correct
	if block := dl.GetBlockByHash(hash); block != nil {
		_, err := trie.NewSecure(block.Root(), trie.NewDatabase(dl.stateDb))
		return err
	}
	return fmt.Errorf("non existent block: %x", hash[:4])
}

// GetTd retrieves the block's total difficulty from the canonical chain.
func (dl *downloadTester) GetTd(hash common.Hash, number uint64) *big.Int {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return dl.getTd(hash)
}

// getTd retrieves the block's total difficulty if found either within
// ancients or own blocks).
// This method assumes that the caller holds at least the read-lock (dl.lock)
func (dl *downloadTester) getTd(hash common.Hash) *big.Int {
	if td := dl.ancientChainTd[hash]; td != nil {
		return td
	}
	return dl.ownChainTd[hash]
}

// InsertHeaderChain injects a new batch of headers into the simulated chain.
func (dl *downloadTester) InsertHeaderChain(headers []*types.Header, checkFreq int) (i int, err error) {
	dl.lock.Lock()
	defer dl.lock.Unlock()
	// Do a quick check, as the blockchain.InsertHeaderChain doesn't insert anything in case of errors
	if dl.getHeaderByHash(headers[0].ParentHash) == nil {
		return 0, fmt.Errorf("InsertHeaderChain: unknown parent at first position, parent of number %d", headers[0].Number)
	}
	var hashes []common.Hash
	for i := 1; i < len(headers); i++ {
		hash := headers[i-1].Hash()
		if headers[i].ParentHash != headers[i-1].Hash() {
			return i, fmt.Errorf("non-contiguous import at position %d", i)
		}
		hashes = append(hashes, hash)
	}
	hashes = append(hashes, headers[len(headers)-1].Hash())
	// Do a full insert if pre-checks passed
	for i, header := range headers {
		hash := hashes[i]
		if dl.getHeaderByHash(hash) != nil {
			continue
		}
		if dl.getHeaderByHash(header.ParentHash) == nil {
			// This _should_ be impossible, due to precheck and induction
			return i, fmt.Errorf("InsertHeaderChain: unknown parent at position %d", i)
		}
		dl.ownHashes = append(dl.ownHashes, hash)
		dl.ownHeaders[hash] = header

		td := dl.getTd(header.ParentHash)
		dl.ownChainTd[hash] = new(big.Int).Add(td, header.Difficulty)
	}
	return len(headers), nil
}

// InsertChain injects a new batch of blocks into the simulated chain.
func (dl *downloadTester) InsertChain(blocks types.Blocks) (i int, err error) {
	dl.lock.Lock()
	defer dl.lock.Unlock()
	for i, block := range blocks {
		if parent, ok := dl.ownBlocks[block.ParentHash()]; !ok {
			return i, fmt.Errorf("InsertChain: unknown parent at position %d / %d", i, len(blocks))
		} else if _, err := dl.stateDb.Get(parent.Root().Bytes()); err != nil {
			return i, fmt.Errorf("InsertChain: unknown parent state %x: %v", parent.Root(), err)
		}
		if hdr := dl.getHeaderByHash(block.Hash()); hdr == nil {
			dl.ownHashes = append(dl.ownHashes, block.Hash())
			dl.ownHeaders[block.Hash()] = block.Header()
		}
		dl.ownBlocks[block.Hash()] = block
		dl.ownReceipts[block.Hash()] = make(types.Receipts, 0)
		dl.stateDb.Put(block.Root().Bytes(), []byte{0x00})
		td := dl.getTd(block.ParentHash())
		dl.ownChainTd[block.Hash()] = new(big.Int).Add(td, block.Difficulty())
	}
	return len(blocks), nil
}

// InsertReceiptChain injects a new batch of receipts into the simulated chain.
func (dl *downloadTester) InsertReceiptChain(blocks types.Blocks, receipts []types.Receipts, ancientLimit uint64) (i int, err error) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	for i := 0; i < len(blocks) && i < len(receipts); i++ {
		if _, ok := dl.ownHeaders[blocks[i].Hash()]; !ok {
			return i, errors.New("unknown owner")
		}
		if _, ok := dl.ancientBlocks[blocks[i].ParentHash()]; !ok {
			if _, ok := dl.ownBlocks[blocks[i].ParentHash()]; !ok {
				return i, errors.New("InsertReceiptChain: unknown parent")
			}
		}
		if blocks[i].NumberU64() <= ancientLimit {
			dl.ancientBlocks[blocks[i].Hash()] = blocks[i]
			dl.ancientReceipts[blocks[i].Hash()] = receipts[i]

			// Migrate from active db to ancient db
			dl.ancientHeaders[blocks[i].Hash()] = blocks[i].Header()
			dl.ancientChainTd[blocks[i].Hash()] = new(big.Int).Add(dl.ancientChainTd[blocks[i].ParentHash()], blocks[i].Difficulty())
			delete(dl.ownHeaders, blocks[i].Hash())
			delete(dl.ownChainTd, blocks[i].Hash())
		} else {
			dl.ownBlocks[blocks[i].Hash()] = blocks[i]
			dl.ownReceipts[blocks[i].Hash()] = receipts[i]
		}
	}
	return len(blocks), nil
}

// SetHead rewinds the local chain to a new head.
func (dl *downloadTester) SetHead(head uint64) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	// Find the hash of the head to reset to
	var hash common.Hash
	for h, header := range dl.ownHeaders {
		if header.Number.Uint64() == head {
			hash = h
		}
	}
	for h, header := range dl.ancientHeaders {
		if header.Number.Uint64() == head {
			hash = h
		}
	}
	if hash == (common.Hash{}) {
		return fmt.Errorf("unknown head to set: %d", head)
	}
	// Find the offset in the header chain
	var offset int
	for o, h := range dl.ownHashes {
		if h == hash {
			offset = o
			break
		}
	}
	// Remove all the hashes and associated data afterwards
	for i := offset + 1; i < len(dl.ownHashes); i++ {
		delete(dl.ownChainTd, dl.ownHashes[i])
		delete(dl.ownHeaders, dl.ownHashes[i])
		delete(dl.ownReceipts, dl.ownHashes[i])
		delete(dl.ownBlocks, dl.ownHashes[i])

		delete(dl.ancientChainTd, dl.ownHashes[i])
		delete(dl.ancientHeaders, dl.ownHashes[i])
		delete(dl.ancientReceipts, dl.ownHashes[i])
		delete(dl.ancientBlocks, dl.ownHashes[i])
	}
	dl.ownHashes = dl.ownHashes[:offset+1]
	return nil
}

// Rollback removes some recently added elements from the chain.
func (dl *downloadTester) Rollback(hashes []common.Hash) {
}

// newPeer registers a new block download source into the downloader.
func (dl *downloadTester) newPeer(id string, version uint, chain *testChain) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	peer := &downloadTesterPeer{dl: dl, id: id, chain: chain}
	dl.peers[id] = peer
	return dl.downloader.RegisterPeer(id, version, peer)
}

// dropPeer simulates a hard peer removal from the connection pool.
func (dl *downloadTester) dropPeer(id string) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	delete(dl.peers, id)
	dl.downloader.UnregisterPeer(id)
}

type downloadTesterPeer struct {
	dl            *downloadTester
	id            string
	chain         *testChain
	missingStates map[common.Hash]bool // State entries that fast sync should not return
}

// Head constructs a function to retrieve a peer's current head hash
// and total difficulty.
func (dlp *downloadTesterPeer) Head() (common.Hash, *big.Int) {
	b := dlp.chain.headBlock()
	return b.Hash(), dlp.chain.td(b.Hash())
}

// RequestHeadersByHash constructs a GetBlockHeaders function based on a hashed
// origin; associated with a particular peer in the download tester. The returned
// function can be used to retrieve batches of headers from the particular peer.
func (dlp *downloadTesterPeer) RequestHeadersByHash(origin common.Hash, amount int, skip int, reverse bool) error {
	result := dlp.chain.headersByHash(origin, amount, skip, reverse)
	go dlp.dl.downloader.DeliverHeaders(dlp.id, result)
	return nil
}

// RequestHeadersByNumber constructs a GetBlockHeaders function based on a numbered
// origin; associated with a particular peer in the download tester. The returned
// function can be used to retrieve batches of headers from the particular peer.
func (dlp *downloadTesterPeer) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool) error {
	result := dlp.chain.headersByNumber(origin, amount, skip, reverse)
	go dlp.dl.downloader.DeliverHeaders(dlp.id, result)
	return nil
}

// RequestBodies constructs a getBlockBodies method associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of block bodies from the particularly requested peer.
func (dlp *downloadTesterPeer) RequestBodies(hashes []common.Hash) error {
	txs, uncles := dlp.chain.bodies(hashes)
	go dlp.dl.downloader.DeliverBodies(dlp.id, txs, uncles)
	return nil
}

// RequestReceipts constructs a getReceipts method associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of block receipts from the particularly requested peer.
func (dlp *downloadTesterPeer) RequestReceipts(hashes []common.Hash) error {
	receipts := dlp.chain.receipts(hashes)
	go dlp.dl.downloader.DeliverReceipts(dlp.id, receipts)
	return nil
}

// RequestNodeData constructs a getNodeData method associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of node state data from the particularly requested peer.
func (dlp *downloadTesterPeer) RequestNodeData(hashes []common.Hash) error {
	dlp.dl.lock.RLock()
	defer dlp.dl.lock.RUnlock()

	results := make([][]byte, 0, len(hashes))
	for _, hash := range hashes {
		if data, err := dlp.dl.peerDb.Get(hash.Bytes()); err == nil {
			if !dlp.missingStates[hash] {
				results = append(results, data)
			}
		}
	}
	go dlp.dl.downloader.DeliverNodeData(dlp.id, results)
	return nil
}

// assertOwnChain checks if the local chain contains the correct number of items
// of the various chain components.
func assertOwnChain(t *testing.T, tester *downloadTester, length int) {
	// Mark this method as a helper to report errors at callsite, not in here
	t.Helper()

	assertOwnForkedChain(t, tester, 1, []int{length})
}

// assertOwnForkedChain checks if the local forked chain contains the correct
// number of items of the various chain components.
func assertOwnForkedChain(t *testing.T, tester *downloadTester, common int, lengths []int) {
	// Mark this method as a helper to report errors at callsite, not in here
	t.Helper()

	// Initialize the counters for the first fork
	headers, blocks, receipts := lengths[0], lengths[0], lengths[0]

	// Update the counters for each subsequent fork
	for _, length := range lengths[1:] {
		headers += length - common
		blocks += length - common
		receipts += length - common
	}
	if tester.downloader.getMode() == LightSync {
		blocks, receipts = 1, 1
	}
	if hs := len(tester.ownHeaders) + len(tester.ancientHeaders) - 1; hs != headers {
		t.Fatalf("synchronised headers mismatch: have %v, want %v", hs, headers)
	}
	if bs := len(tester.ownBlocks) + len(tester.ancientBlocks) - 1; bs != blocks {
		t.Fatalf("synchronised blocks mismatch: have %v, want %v", bs, blocks)
	}
	if rs := len(tester.ownReceipts) + len(tester.ancientReceipts) - 1; rs != receipts {
		t.Fatalf("synchronised receipts mismatch: have %v, want %v", rs, receipts)
	}
}

// Tests that simple synchronization against a canonical chain works correctly.
// In this test common ancestor lookup should be short circuited and not require
// binary searching.
func TestCanonicalSynchronisation64Full(t *testing.T) { testCanonicalSynchronisation(t, 64, FullSync) }
func TestCanonicalSynchronisation64Fast(t *testing.T) { testCanonicalSynchronisation(t, 64, FastSync) }
func TestCanonicalSynchronisation65Full(t *testing.T) { testCanonicalSynchronisation(t, 65, FullSync) }
func TestCanonicalSynchronisation65Fast(t *testing.T) { testCanonicalSynchronisation(t, 65, FastSync) }
func TestCanonicalSynchronisation65Light(t *testing.T) {
	testCanonicalSynchronisation(t, 65, LightSync)
}

func testCanonicalSynchronisation(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()

	// Create a small enough block chain to download
	chain := testChainBase.shorten(blockCacheMaxItems - 15)
	tester.newPeer("peer", protocol, chain)

	// Synchronise with the peer and make sure all relevant data was retrieved
	if err := tester.sync("peer", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, chain.len())
}

// Tests that if a large batch of blocks are being downloaded, it is throttled
// until the cached blocks are retrieved.
func TestThrottling64Full(t *testing.T) { testThrottling(t, 64, FullSync) }
func TestThrottling64Fast(t *testing.T) { testThrottling(t, 64, FastSync) }
func TestThrottling65Full(t *testing.T) { testThrottling(t, 65, FullSync) }
func TestThrottling65Fast(t *testing.T) { testThrottling(t, 65, FastSync) }

func testThrottling(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()
	tester := newTester()

	// Create a long block chain to download and the tester
	targetBlocks := testChainBase.len() - 1
	tester.newPeer("peer", protocol, testChainBase)

	// Wrap the importer to allow stepping
	blocked, proceed := uint32(0), make(chan struct{})
	tester.downloader.chainInsertHook = func(results []*fetchResult) {
		atomic.StoreUint32(&blocked, uint32(len(results)))
		<-proceed
	}
	// Start a synchronisation concurrently
	errc := make(chan error, 1)
	go func() {
		errc <- tester.sync("peer", nil, mode)
	}()
	// Iteratively take some blocks, always checking the retrieval count
	for {
		// Check the retrieval count synchronously (! reason for this ugly block)
		tester.lock.RLock()
		retrieved := len(tester.ownBlocks)
		tester.lock.RUnlock()
		if retrieved >= targetBlocks+1 {
			break
		}
		// Wait a bit for sync to throttle itself
		var cached, frozen int
		for start := time.Now(); time.Since(start) < 3*time.Second; {
			time.Sleep(25 * time.Millisecond)

			tester.lock.Lock()
			{
				tester.downloader.queue.resultCache.lock.Lock()
				cached = tester.downloader.queue.resultCache.countCompleted()
				tester.downloader.queue.resultCache.lock.Unlock()
				frozen = int(atomic.LoadUint32(&blocked))
				retrieved = len(tester.ownBlocks)

			}
			tester.lock.Unlock()

			if cached == blockCacheMaxItems ||
				cached == blockCacheMaxItems-reorgProtHeaderDelay ||
				retrieved+cached+frozen == targetBlocks+1 ||
				retrieved+cached+frozen == targetBlocks+1-reorgProtHeaderDelay {
				break
			}
		}
		// Make sure we filled up the cache, then exhaust it
		time.Sleep(25 * time.Millisecond) // give it a chance to screw up
		tester.lock.RLock()
		retrieved = len(tester.ownBlocks)
		tester.lock.RUnlock()
		if cached != blockCacheMaxItems && cached != blockCacheMaxItems-reorgProtHeaderDelay && retrieved+cached+frozen != targetBlocks+1 && retrieved+cached+frozen != targetBlocks+1-reorgProtHeaderDelay {
			t.Fatalf("block count mismatch: have %v, want %v (owned %v, blocked %v, target %v)", cached, blockCacheMaxItems, retrieved, frozen, targetBlocks+1)
		}

		// Permit the blocked blocks to import
		if atomic.LoadUint32(&blocked) > 0 {
			atomic.StoreUint32(&blocked, uint32(0))
			proceed <- struct{}{}
		}
	}
	// Check that we haven't pulled more blocks than available
	assertOwnChain(t, tester, targetBlocks+1)
	if err := <-errc; err != nil {
		t.Fatalf("block synchronization failed: %v", err)
	}
	tester.terminate()

}

// Tests that simple synchronization against a forked chain works correctly. In
// this test common ancestor lookup should *not* be short circuited, and a full
// binary search should be executed.
func TestForkedSync64Full(t *testing.T)  { testForkedSync(t, 64, FullSync) }
func TestForkedSync64Fast(t *testing.T)  { testForkedSync(t, 64, FastSync) }
func TestForkedSync65Full(t *testing.T)  { testForkedSync(t, 65, FullSync) }
func TestForkedSync65Fast(t *testing.T)  { testForkedSync(t, 65, FastSync) }
func TestForkedSync65Light(t *testing.T) { testForkedSync(t, 65, LightSync) }

func testForkedSync(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()

	chainA := testChainForkLightA.shorten(testChainBase.len() + 80)
	chainB := testChainForkLightB.shorten(testChainBase.len() + 80)
	tester.newPeer("fork A", protocol, chainA)
	tester.newPeer("fork B", protocol, chainB)
	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.sync("fork A", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, chainA.len())

	// Synchronise with the second peer and make sure that fork is pulled too
	if err := tester.sync("fork B", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnForkedChain(t, tester, testChainBase.len(), []int{chainA.len(), chainB.len()})
}

// Tests that synchronising against a much shorter but much heavyer fork works
// corrently and is not dropped.
func TestHeavyForkedSync64Full(t *testing.T)  { testHeavyForkedSync(t, 64, FullSync) }
func TestHeavyForkedSync64Fast(t *testing.T)  { testHeavyForkedSync(t, 64, FastSync) }
func TestHeavyForkedSync65Full(t *testing.T)  { testHeavyForkedSync(t, 65, FullSync) }
func TestHeavyForkedSync65Fast(t *testing.T)  { testHeavyForkedSync(t, 65, FastSync) }
func TestHeavyForkedSync65Light(t *testing.T) { testHeavyForkedSync(t, 65, LightSync) }

func testHeavyForkedSync(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()

	chainA := testChainForkLightA.shorten(testChainBase.len() + 80)
	chainB := testChainForkHeavy.shorten(testChainBase.len() + 80)
	tester.newPeer("light", protocol, chainA)
	tester.newPeer("heavy", protocol, chainB)

	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.sync("light", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, chainA.len())

	// Synchronise with the second peer and make sure that fork is pulled too
	if err := tester.sync("heavy", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnForkedChain(t, tester, testChainBase.len(), []int{chainA.len(), chainB.len()})
}

// Tests that chain forks are contained within a certain interval of the current
// chain head, ensuring that malicious peers cannot waste resources by feeding
// long dead chains.
func TestBoundedForkedSync64Full(t *testing.T)  { testBoundedForkedSync(t, 64, FullSync) }
func TestBoundedForkedSync64Fast(t *testing.T)  { testBoundedForkedSync(t, 64, FastSync) }
func TestBoundedForkedSync65Full(t *testing.T)  { testBoundedForkedSync(t, 65, FullSync) }
func TestBoundedForkedSync65Fast(t *testing.T)  { testBoundedForkedSync(t, 65, FastSync) }
func TestBoundedForkedSync65Light(t *testing.T) { testBoundedForkedSync(t, 65, LightSync) }

func testBoundedForkedSync(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()

	chainA := testChainForkLightA
	chainB := testChainForkLightB
	tester.newPeer("original", protocol, chainA)
	tester.newPeer("rewriter", protocol, chainB)

	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.sync("original", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, chainA.len())

	// Synchronise with the second peer and ensure that the fork is rejected to being too old
	if err := tester.sync("rewriter", nil, mode); err != errInvalidAncestor {
		t.Fatalf("sync failure mismatch: have %v, want %v", err, errInvalidAncestor)
	}
}

// Tests that chain forks are contained within a certain interval of the current
// chain head for short but heavy forks too. These are a bit special because they
// take different ancestor lookup paths.
func TestBoundedHeavyForkedSync64Full(t *testing.T)  { testBoundedHeavyForkedSync(t, 64, FullSync) }
func TestBoundedHeavyForkedSync64Fast(t *testing.T)  { testBoundedHeavyForkedSync(t, 64, FastSync) }
func TestBoundedHeavyForkedSync65Full(t *testing.T)  { testBoundedHeavyForkedSync(t, 65, FullSync) }
func TestBoundedHeavyForkedSync65Fast(t *testing.T)  { testBoundedHeavyForkedSync(t, 65, FastSync) }
func TestBoundedHeavyForkedSync65Light(t *testing.T) { testBoundedHeavyForkedSync(t, 65, LightSync) }

func testBoundedHeavyForkedSync(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()
	tester := newTester()

	// Create a long enough forked chain
	chainA := testChainForkLightA
	chainB := testChainForkHeavy
	tester.newPeer("original", protocol, chainA)

	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.sync("original", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, chainA.len())

	tester.newPeer("heavy-rewriter", protocol, chainB)
	// Synchronise with the second peer and ensure that the fork is rejected to being too old
	if err := tester.sync("heavy-rewriter", nil, mode); err != errInvalidAncestor {
		t.Fatalf("sync failure mismatch: have %v, want %v", err, errInvalidAncestor)
	}
	tester.terminate()
}

// Tests that an inactive downloader will not accept incoming block headers,
// bodies and receipts.
func TestInactiveDownloader63(t *testing.T) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()

	// Check that neither block headers nor bodies are accepted
	if err := tester.downloader.DeliverHeaders("bad peer", []*types.Header{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
	if err := tester.downloader.DeliverBodies("bad peer", [][]*types.Transaction{}, [][]*types.Header{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
	if err := tester.downloader.DeliverReceipts("bad peer", [][]*types.Receipt{}); err != errNoSyncActive {
		t.Errorf("error mismatch: have %v, want %v", err, errNoSyncActive)
	}
}

// Tests that a canceled download wipes all previously accumulated state.
func TestCancel64Full(t *testing.T)  { testCancel(t, 64, FullSync) }
func TestCancel64Fast(t *testing.T)  { testCancel(t, 64, FastSync) }
func TestCancel65Full(t *testing.T)  { testCancel(t, 65, FullSync) }
func TestCancel65Fast(t *testing.T)  { testCancel(t, 65, FastSync) }
func TestCancel65Light(t *testing.T) { testCancel(t, 65, LightSync) }

func testCancel(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()

	chain := testChainBase.shorten(MaxHeaderFetch)
	tester.newPeer("peer", protocol, chain)

	// Make sure canceling works with a pristine downloader
	tester.downloader.Cancel()
	if !tester.downloader.queue.Idle() {
		t.Errorf("download queue not idle")
	}
	// Synchronise with the peer, but cancel afterwards
	if err := tester.sync("peer", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	tester.downloader.Cancel()
	if !tester.downloader.queue.Idle() {
		t.Errorf("download queue not idle")
	}
}

// Tests that synchronisation from multiple peers works as intended (multi thread sanity test).
func TestMultiSynchronisation64Full(t *testing.T)  { testMultiSynchronisation(t, 64, FullSync) }
func TestMultiSynchronisation64Fast(t *testing.T)  { testMultiSynchronisation(t, 64, FastSync) }
func TestMultiSynchronisation65Full(t *testing.T)  { testMultiSynchronisation(t, 65, FullSync) }
func TestMultiSynchronisation65Fast(t *testing.T)  { testMultiSynchronisation(t, 65, FastSync) }
func TestMultiSynchronisation65Light(t *testing.T) { testMultiSynchronisation(t, 65, LightSync) }

func testMultiSynchronisation(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()

	// Create various peers with various parts of the chain
	targetPeers := 8
	chain := testChainBase.shorten(targetPeers * 100)

	for i := 0; i < targetPeers; i++ {
		id := fmt.Sprintf("peer #%d", i)
		tester.newPeer(id, protocol, chain.shorten(chain.len()/(i+1)))
	}
	if err := tester.sync("peer #0", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, chain.len())
}

// Tests that synchronisations behave well in multi-version protocol environments
// and not wreak havoc on other nodes in the network.
func TestMultiProtoSynchronisation64Full(t *testing.T)  { testMultiProtoSync(t, 64, FullSync) }
func TestMultiProtoSynchronisation64Fast(t *testing.T)  { testMultiProtoSync(t, 64, FastSync) }
func TestMultiProtoSynchronisation65Full(t *testing.T)  { testMultiProtoSync(t, 65, FullSync) }
func TestMultiProtoSynchronisation65Fast(t *testing.T)  { testMultiProtoSync(t, 65, FastSync) }
func TestMultiProtoSynchronisation65Light(t *testing.T) { testMultiProtoSync(t, 65, LightSync) }

func testMultiProtoSync(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()

	// Create a small enough block chain to download
	chain := testChainBase.shorten(blockCacheMaxItems - 15)

	// Create peers of every type
	tester.newPeer("peer 63", 63, chain)
	tester.newPeer("peer 64", 64, chain)
	tester.newPeer("peer 65", 65, chain)

	// Synchronise with the requested peer and make sure all blocks were retrieved
	if err := tester.sync(fmt.Sprintf("peer %d", protocol), nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, chain.len())

	// Check that no peers have been dropped off
	for _, version := range []int{63, 64, 65} {
		peer := fmt.Sprintf("peer %d", version)
		if _, ok := tester.peers[peer]; !ok {
			t.Errorf("%s dropped", peer)
		}
	}
}

// Tests that if a block is empty (e.g. header only), no body request should be
// made, and instead the header should be assembled into a whole block in itself.
func TestEmptyShortCircuit64Full(t *testing.T)  { testEmptyShortCircuit(t, 64, FullSync) }
func TestEmptyShortCircuit64Fast(t *testing.T)  { testEmptyShortCircuit(t, 64, FastSync) }
func TestEmptyShortCircuit65Full(t *testing.T)  { testEmptyShortCircuit(t, 65, FullSync) }
func TestEmptyShortCircuit65Fast(t *testing.T)  { testEmptyShortCircuit(t, 65, FastSync) }
func TestEmptyShortCircuit65Light(t *testing.T) { testEmptyShortCircuit(t, 65, LightSync) }

func testEmptyShortCircuit(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()

	// Create a block chain to download
	chain := testChainBase
	tester.newPeer("peer", protocol, chain)

	// Instrument the downloader to signal body requests
	bodiesHave, receiptsHave := int32(0), int32(0)
	tester.downloader.bodyFetchHook = func(headers []*types.Header) {
		atomic.AddInt32(&bodiesHave, int32(len(headers)))
	}
	tester.downloader.receiptFetchHook = func(headers []*types.Header) {
		atomic.AddInt32(&receiptsHave, int32(len(headers)))
	}
	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.sync("peer", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, chain.len())

	// Validate the number of block bodies that should have been requested
	bodiesNeeded, receiptsNeeded := 0, 0
	for _, block := range chain.blockm {
		if mode != LightSync && block != tester.genesis && (len(block.Transactions()) > 0 || len(block.Uncles()) > 0) {
			bodiesNeeded++
		}
	}
	for _, receipt := range chain.receiptm {
		if mode == FastSync && len(receipt) > 0 {
			receiptsNeeded++
		}
	}
	if int(bodiesHave) != bodiesNeeded {
		t.Errorf("body retrieval count mismatch: have %v, want %v", bodiesHave, bodiesNeeded)
	}
	if int(receiptsHave) != receiptsNeeded {
		t.Errorf("receipt retrieval count mismatch: have %v, want %v", receiptsHave, receiptsNeeded)
	}
}

// Tests that headers are enqueued continuously, preventing malicious nodes from
// stalling the downloader by feeding gapped header chains.
func TestMissingHeaderAttack64Full(t *testing.T)  { testMissingHeaderAttack(t, 64, FullSync) }
func TestMissingHeaderAttack64Fast(t *testing.T)  { testMissingHeaderAttack(t, 64, FastSync) }
func TestMissingHeaderAttack65Full(t *testing.T)  { testMissingHeaderAttack(t, 65, FullSync) }
func TestMissingHeaderAttack65Fast(t *testing.T)  { testMissingHeaderAttack(t, 65, FastSync) }
func TestMissingHeaderAttack65Light(t *testing.T) { testMissingHeaderAttack(t, 65, LightSync) }

func testMissingHeaderAttack(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()

	chain := testChainBase.shorten(blockCacheMaxItems - 15)
	brokenChain := chain.shorten(chain.len())
	delete(brokenChain.headerm, brokenChain.chain[brokenChain.len()/2])
	tester.newPeer("attack", protocol, brokenChain)

	if err := tester.sync("attack", nil, mode); err == nil {
		t.Fatalf("succeeded attacker synchronisation")
	}
	// Synchronise with the valid peer and make sure sync succeeds
	tester.newPeer("valid", protocol, chain)
	if err := tester.sync("valid", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, chain.len())
}

// Tests that if requested headers are shifted (i.e. first is missing), the queue
// detects the invalid numbering.
func TestShiftedHeaderAttack64Full(t *testing.T)  { testShiftedHeaderAttack(t, 64, FullSync) }
func TestShiftedHeaderAttack64Fast(t *testing.T)  { testShiftedHeaderAttack(t, 64, FastSync) }
func TestShiftedHeaderAttack65Full(t *testing.T)  { testShiftedHeaderAttack(t, 65, FullSync) }
func TestShiftedHeaderAttack65Fast(t *testing.T)  { testShiftedHeaderAttack(t, 65, FastSync) }
func TestShiftedHeaderAttack65Light(t *testing.T) { testShiftedHeaderAttack(t, 65, LightSync) }

func testShiftedHeaderAttack(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()

	chain := testChainBase.shorten(blockCacheMaxItems - 15)

	// Attempt a full sync with an attacker feeding shifted headers
	brokenChain := chain.shorten(chain.len())
	delete(brokenChain.headerm, brokenChain.chain[1])
	delete(brokenChain.blockm, brokenChain.chain[1])
	delete(brokenChain.receiptm, brokenChain.chain[1])
	tester.newPeer("attack", protocol, brokenChain)
	if err := tester.sync("attack", nil, mode); err == nil {
		t.Fatalf("succeeded attacker synchronisation")
	}

	// Synchronise with the valid peer and make sure sync succeeds
	tester.newPeer("valid", protocol, chain)
	if err := tester.sync("valid", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	assertOwnChain(t, tester, chain.len())
}

// Tests that upon detecting an invalid header, the recent ones are rolled back
// for various failure scenarios. Afterwards a full sync is attempted to make
// sure no state was corrupted.
func TestInvalidHeaderRollback64Fast(t *testing.T) { testInvalidHeaderRollback(t, 64, FastSync) }
func TestInvalidHeaderRollback65Fast(t *testing.T) { testInvalidHeaderRollback(t, 65, FastSync) }

func testInvalidHeaderRollback(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()

	// Create a small enough block chain to download
	targetBlocks := 3*fsHeaderSafetyNet + 256 + fsMinFullBlocks
	chain := testChainBase.shorten(targetBlocks)

	// Attempt to sync with an attacker that feeds junk during the fast sync phase.
	// This should result in the last fsHeaderSafetyNet headers being rolled back.
	missing := fsHeaderSafetyNet + MaxHeaderFetch + 1
	fastAttackChain := chain.shorten(chain.len())
	delete(fastAttackChain.headerm, fastAttackChain.chain[missing])
	tester.newPeer("fast-attack", protocol, fastAttackChain)

	if err := tester.sync("fast-attack", nil, mode); err == nil {
		t.Fatalf("succeeded fast attacker synchronisation")
	}
	if head := tester.CurrentHeader().Number.Int64(); int(head) > MaxHeaderFetch {
		t.Errorf("rollback head mismatch: have %v, want at most %v", head, MaxHeaderFetch)
	}

	// Attempt to sync with an attacker that feeds junk during the block import phase.
	// This should result in both the last fsHeaderSafetyNet number of headers being
	// rolled back, and also the pivot point being reverted to a non-block status.
	missing = 3*fsHeaderSafetyNet + MaxHeaderFetch + 1
	blockAttackChain := chain.shorten(chain.len())
	delete(fastAttackChain.headerm, fastAttackChain.chain[missing]) // Make sure the fast-attacker doesn't fill in
	delete(blockAttackChain.headerm, blockAttackChain.chain[missing])
	tester.newPeer("block-attack", protocol, blockAttackChain)

	if err := tester.sync("block-attack", nil, mode); err == nil {
		t.Fatalf("succeeded block attacker synchronisation")
	}
	if head := tester.CurrentHeader().Number.Int64(); int(head) > 2*fsHeaderSafetyNet+MaxHeaderFetch {
		t.Errorf("rollback head mismatch: have %v, want at most %v", head, 2*fsHeaderSafetyNet+MaxHeaderFetch)
	}
	if mode == FastSync {
		if head := tester.CurrentBlock().NumberU64(); head != 0 {
			t.Errorf("fast sync pivot block #%d not rolled back", head)
		}
	}

	// Attempt to sync with an attacker that withholds promised blocks after the
	// fast sync pivot point. This could be a trial to leave the node with a bad
	// but already imported pivot block.
	withholdAttackChain := chain.shorten(chain.len())
	tester.newPeer("withhold-attack", protocol, withholdAttackChain)
	tester.downloader.syncInitHook = func(uint64, uint64) {
		for i := missing; i < withholdAttackChain.len(); i++ {
			delete(withholdAttackChain.headerm, withholdAttackChain.chain[i])
		}
		tester.downloader.syncInitHook = nil
	}
	if err := tester.sync("withhold-attack", nil, mode); err == nil {
		t.Fatalf("succeeded withholding attacker synchronisation")
	}
	if head := tester.CurrentHeader().Number.Int64(); int(head) > 2*fsHeaderSafetyNet+MaxHeaderFetch {
		t.Errorf("rollback head mismatch: have %v, want at most %v", head, 2*fsHeaderSafetyNet+MaxHeaderFetch)
	}
	if mode == FastSync {
		if head := tester.CurrentBlock().NumberU64(); head != 0 {
			t.Errorf("fast sync pivot block #%d not rolled back", head)
		}
	}

	// synchronise with the valid peer and make sure sync succeeds. Since the last rollback
	// should also disable fast syncing for this process, verify that we did a fresh full
	// sync. Note, we can't assert anything about the receipts since we won't purge the
	// database of them, hence we can't use assertOwnChain.
	tester.newPeer("valid", protocol, chain)
	if err := tester.sync("valid", nil, mode); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if hs := len(tester.ownHeaders); hs != chain.len() {
		t.Fatalf("synchronised headers mismatch: have %v, want %v", hs, chain.len())
	}
	if mode != LightSync {
		if bs := len(tester.ownBlocks); bs != chain.len() {
			t.Fatalf("synchronised blocks mismatch: have %v, want %v", bs, chain.len())
		}
	}
	tester.terminate()
}

// Tests that a peer advertising a high TD doesn't get to stall the downloader
// afterwards by not sending any useful hashes.
func TestHighTDStarvationAttack64Full(t *testing.T)  { testHighTDStarvationAttack(t, 64, FullSync) }
func TestHighTDStarvationAttack64Fast(t *testing.T)  { testHighTDStarvationAttack(t, 64, FastSync) }
func TestHighTDStarvationAttack65Full(t *testing.T)  { testHighTDStarvationAttack(t, 65, FullSync) }
func TestHighTDStarvationAttack65Fast(t *testing.T)  { testHighTDStarvationAttack(t, 65, FastSync) }
func TestHighTDStarvationAttack65Light(t *testing.T) { testHighTDStarvationAttack(t, 65, LightSync) }

func testHighTDStarvationAttack(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()

	chain := testChainBase.shorten(1)
	tester.newPeer("attack", protocol, chain)
	if err := tester.sync("attack", big.NewInt(1000000), mode); err != errStallingPeer {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errStallingPeer)
	}
	tester.terminate()
}

// Tests that misbehaving peers are disconnected, whilst behaving ones are not.
func TestBlockHeaderAttackerDropping64(t *testing.T) { testBlockHeaderAttackerDropping(t, 64) }
func TestBlockHeaderAttackerDropping65(t *testing.T) { testBlockHeaderAttackerDropping(t, 65) }

func testBlockHeaderAttackerDropping(t *testing.T, protocol uint) {
	t.Parallel()

	// Define the disconnection requirement for individual hash fetch errors
	tests := []struct {
		result error
		drop   bool
	}{
		{nil, false},                        // Sync succeeded, all is well
		{errBusy, false},                    // Sync is already in progress, no problem
		{errUnknownPeer, false},             // Peer is unknown, was already dropped, don't double drop
		{errBadPeer, true},                  // Peer was deemed bad for some reason, drop it
		{errStallingPeer, true},             // Peer was detected to be stalling, drop it
		{errUnsyncedPeer, true},             // Peer was detected to be unsynced, drop it
		{errNoPeers, false},                 // No peers to download from, soft race, no issue
		{errTimeout, true},                  // No hashes received in due time, drop the peer
		{errEmptyHeaderSet, true},           // No headers were returned as a response, drop as it's a dead end
		{errPeersUnavailable, true},         // Nobody had the advertised blocks, drop the advertiser
		{errInvalidAncestor, true},          // Agreed upon ancestor is not acceptable, drop the chain rewriter
		{errInvalidChain, true},             // Hash chain was detected as invalid, definitely drop
		{errInvalidBody, false},             // A bad peer was detected, but not the sync origin
		{errInvalidReceipt, false},          // A bad peer was detected, but not the sync origin
		{errCancelContentProcessing, false}, // Synchronisation was canceled, origin may be innocent, don't drop
	}
	// Run the tests and check disconnection status
	tester := newTester()
	defer tester.terminate()
	chain := testChainBase.shorten(1)

	for i, tt := range tests {
		// Register a new peer and ensure its presence
		id := fmt.Sprintf("test %d", i)
		if err := tester.newPeer(id, protocol, chain); err != nil {
			t.Fatalf("test %d: failed to register new peer: %v", i, err)
		}
		if _, ok := tester.peers[id]; !ok {
			t.Fatalf("test %d: registered peer not found", i)
		}
		// Simulate a synchronisation and check the required result
		tester.downloader.synchroniseMock = func(string, common.Hash) error { return tt.result }

		tester.downloader.Synchronise(id, tester.genesis.Hash(), big.NewInt(1000), FullSync)
		if _, ok := tester.peers[id]; !ok != tt.drop {
			t.Errorf("test %d: peer drop mismatch for %v: have %v, want %v", i, tt.result, !ok, tt.drop)
		}
	}
}

// Tests that synchronisation progress (origin block number, current block number
// and highest block number) is tracked and updated correctly.
func TestSyncProgress64Full(t *testing.T)  { testSyncProgress(t, 64, FullSync) }
func TestSyncProgress64Fast(t *testing.T)  { testSyncProgress(t, 64, FastSync) }
func TestSyncProgress65Full(t *testing.T)  { testSyncProgress(t, 65, FullSync) }
func TestSyncProgress65Fast(t *testing.T)  { testSyncProgress(t, 65, FastSync) }
func TestSyncProgress65Light(t *testing.T) { testSyncProgress(t, 65, LightSync) }

func testSyncProgress(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()
	chain := testChainBase.shorten(blockCacheMaxItems - 15)

	// Set a sync init hook to catch progress changes
	starting := make(chan struct{})
	progress := make(chan struct{})

	tester.downloader.syncInitHook = func(origin, latest uint64) {
		starting <- struct{}{}
		<-progress
	}
	checkProgress(t, tester.downloader, "pristine", ethereum.SyncProgress{})

	// Synchronise half the blocks and check initial progress
	tester.newPeer("peer-half", protocol, chain.shorten(chain.len()/2))
	pending := new(sync.WaitGroup)
	pending.Add(1)

	go func() {
		defer pending.Done()
		if err := tester.sync("peer-half", nil, mode); err != nil {
			panic(fmt.Sprintf("failed to synchronise blocks: %v", err))
		}
	}()
	<-starting
	checkProgress(t, tester.downloader, "initial", ethereum.SyncProgress{
		HighestBlock: uint64(chain.len()/2 - 1),
	})
	progress <- struct{}{}
	pending.Wait()

	// Synchronise all the blocks and check continuation progress
	tester.newPeer("peer-full", protocol, chain)
	pending.Add(1)
	go func() {
		defer pending.Done()
		if err := tester.sync("peer-full", nil, mode); err != nil {
			panic(fmt.Sprintf("failed to synchronise blocks: %v", err))
		}
	}()
	<-starting
	checkProgress(t, tester.downloader, "completing", ethereum.SyncProgress{
		StartingBlock: uint64(chain.len()/2 - 1),
		CurrentBlock:  uint64(chain.len()/2 - 1),
		HighestBlock:  uint64(chain.len() - 1),
	})

	// Check final progress after successful sync
	progress <- struct{}{}
	pending.Wait()
	checkProgress(t, tester.downloader, "final", ethereum.SyncProgress{
		StartingBlock: uint64(chain.len()/2 - 1),
		CurrentBlock:  uint64(chain.len() - 1),
		HighestBlock:  uint64(chain.len() - 1),
	})
}

func checkProgress(t *testing.T, d *Downloader, stage string, want ethereum.SyncProgress) {
	// Mark this method as a helper to report errors at callsite, not in here
	t.Helper()

	p := d.Progress()
	p.KnownStates, p.PulledStates = 0, 0
	want.KnownStates, want.PulledStates = 0, 0
	if p != want {
		t.Fatalf("%s progress mismatch:\nhave %+v\nwant %+v", stage, p, want)
	}
}

// Tests that synchronisation progress (origin block number and highest block
// number) is tracked and updated correctly in case of a fork (or manual head
// revertal).
func TestForkedSyncProgress64Full(t *testing.T)  { testForkedSyncProgress(t, 64, FullSync) }
func TestForkedSyncProgress64Fast(t *testing.T)  { testForkedSyncProgress(t, 64, FastSync) }
func TestForkedSyncProgress65Full(t *testing.T)  { testForkedSyncProgress(t, 65, FullSync) }
func TestForkedSyncProgress65Fast(t *testing.T)  { testForkedSyncProgress(t, 65, FastSync) }
func TestForkedSyncProgress65Light(t *testing.T) { testForkedSyncProgress(t, 65, LightSync) }

func testForkedSyncProgress(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()
	chainA := testChainForkLightA.shorten(testChainBase.len() + MaxHeaderFetch)
	chainB := testChainForkLightB.shorten(testChainBase.len() + MaxHeaderFetch)

	// Set a sync init hook to catch progress changes
	starting := make(chan struct{})
	progress := make(chan struct{})

	tester.downloader.syncInitHook = func(origin, latest uint64) {
		starting <- struct{}{}
		<-progress
	}
	checkProgress(t, tester.downloader, "pristine", ethereum.SyncProgress{})

	// Synchronise with one of the forks and check progress
	tester.newPeer("fork A", protocol, chainA)
	pending := new(sync.WaitGroup)
	pending.Add(1)
	go func() {
		defer pending.Done()
		if err := tester.sync("fork A", nil, mode); err != nil {
			panic(fmt.Sprintf("failed to synchronise blocks: %v", err))
		}
	}()
	<-starting

	checkProgress(t, tester.downloader, "initial", ethereum.SyncProgress{
		HighestBlock: uint64(chainA.len() - 1),
	})
	progress <- struct{}{}
	pending.Wait()

	// Simulate a successful sync above the fork
	tester.downloader.syncStatsChainOrigin = tester.downloader.syncStatsChainHeight

	// Synchronise with the second fork and check progress resets
	tester.newPeer("fork B", protocol, chainB)
	pending.Add(1)
	go func() {
		defer pending.Done()
		if err := tester.sync("fork B", nil, mode); err != nil {
			panic(fmt.Sprintf("failed to synchronise blocks: %v", err))
		}
	}()
	<-starting
	checkProgress(t, tester.downloader, "forking", ethereum.SyncProgress{
		StartingBlock: uint64(testChainBase.len()) - 1,
		CurrentBlock:  uint64(chainA.len() - 1),
		HighestBlock:  uint64(chainB.len() - 1),
	})

	// Check final progress after successful sync
	progress <- struct{}{}
	pending.Wait()
	checkProgress(t, tester.downloader, "final", ethereum.SyncProgress{
		StartingBlock: uint64(testChainBase.len()) - 1,
		CurrentBlock:  uint64(chainB.len() - 1),
		HighestBlock:  uint64(chainB.len() - 1),
	})
}

// Tests that if synchronisation is aborted due to some failure, then the progress
// origin is not updated in the next sync cycle, as it should be considered the
// continuation of the previous sync and not a new instance.
func TestFailedSyncProgress64Full(t *testing.T)  { testFailedSyncProgress(t, 64, FullSync) }
func TestFailedSyncProgress64Fast(t *testing.T)  { testFailedSyncProgress(t, 64, FastSync) }
func TestFailedSyncProgress65Full(t *testing.T)  { testFailedSyncProgress(t, 65, FullSync) }
func TestFailedSyncProgress65Fast(t *testing.T)  { testFailedSyncProgress(t, 65, FastSync) }
func TestFailedSyncProgress65Light(t *testing.T) { testFailedSyncProgress(t, 65, LightSync) }

func testFailedSyncProgress(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()
	chain := testChainBase.shorten(blockCacheMaxItems - 15)

	// Set a sync init hook to catch progress changes
	starting := make(chan struct{})
	progress := make(chan struct{})

	tester.downloader.syncInitHook = func(origin, latest uint64) {
		starting <- struct{}{}
		<-progress
	}
	checkProgress(t, tester.downloader, "pristine", ethereum.SyncProgress{})

	// Attempt a full sync with a faulty peer
	brokenChain := chain.shorten(chain.len())
	missing := brokenChain.len() / 2
	delete(brokenChain.headerm, brokenChain.chain[missing])
	delete(brokenChain.blockm, brokenChain.chain[missing])
	delete(brokenChain.receiptm, brokenChain.chain[missing])
	tester.newPeer("faulty", protocol, brokenChain)

	pending := new(sync.WaitGroup)
	pending.Add(1)
	go func() {
		defer pending.Done()
		if err := tester.sync("faulty", nil, mode); err == nil {
			panic("succeeded faulty synchronisation")
		}
	}()
	<-starting
	checkProgress(t, tester.downloader, "initial", ethereum.SyncProgress{
		HighestBlock: uint64(brokenChain.len() - 1),
	})
	progress <- struct{}{}
	pending.Wait()
	afterFailedSync := tester.downloader.Progress()

	// Synchronise with a good peer and check that the progress origin remind the same
	// after a failure
	tester.newPeer("valid", protocol, chain)
	pending.Add(1)
	go func() {
		defer pending.Done()
		if err := tester.sync("valid", nil, mode); err != nil {
			panic(fmt.Sprintf("failed to synchronise blocks: %v", err))
		}
	}()
	<-starting
	checkProgress(t, tester.downloader, "completing", afterFailedSync)

	// Check final progress after successful sync
	progress <- struct{}{}
	pending.Wait()
	checkProgress(t, tester.downloader, "final", ethereum.SyncProgress{
		CurrentBlock: uint64(chain.len() - 1),
		HighestBlock: uint64(chain.len() - 1),
	})
}

// Tests that if an attacker fakes a chain height, after the attack is detected,
// the progress height is successfully reduced at the next sync invocation.
func TestFakedSyncProgress64Full(t *testing.T)  { testFakedSyncProgress(t, 64, FullSync) }
func TestFakedSyncProgress64Fast(t *testing.T)  { testFakedSyncProgress(t, 64, FastSync) }
func TestFakedSyncProgress65Full(t *testing.T)  { testFakedSyncProgress(t, 65, FullSync) }
func TestFakedSyncProgress65Fast(t *testing.T)  { testFakedSyncProgress(t, 65, FastSync) }
func TestFakedSyncProgress65Light(t *testing.T) { testFakedSyncProgress(t, 65, LightSync) }

func testFakedSyncProgress(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	tester := newTester()
	defer tester.terminate()
	chain := testChainBase.shorten(blockCacheMaxItems - 15)

	// Set a sync init hook to catch progress changes
	starting := make(chan struct{})
	progress := make(chan struct{})
	tester.downloader.syncInitHook = func(origin, latest uint64) {
		starting <- struct{}{}
		<-progress
	}
	checkProgress(t, tester.downloader, "pristine", ethereum.SyncProgress{})

	// Create and sync with an attacker that promises a higher chain than available.
	brokenChain := chain.shorten(chain.len())
	numMissing := 5
	for i := brokenChain.len() - 2; i > brokenChain.len()-numMissing; i-- {
		delete(brokenChain.headerm, brokenChain.chain[i])
	}
	tester.newPeer("attack", protocol, brokenChain)

	pending := new(sync.WaitGroup)
	pending.Add(1)
	go func() {
		defer pending.Done()
		if err := tester.sync("attack", nil, mode); err == nil {
			panic("succeeded attacker synchronisation")
		}
	}()
	<-starting
	checkProgress(t, tester.downloader, "initial", ethereum.SyncProgress{
		HighestBlock: uint64(brokenChain.len() - 1),
	})
	progress <- struct{}{}
	pending.Wait()
	afterFailedSync := tester.downloader.Progress()

	// Synchronise with a good peer and check that the progress height has been reduced to
	// the true value.
	validChain := chain.shorten(chain.len() - numMissing)
	tester.newPeer("valid", protocol, validChain)
	pending.Add(1)

	go func() {
		defer pending.Done()
		if err := tester.sync("valid", nil, mode); err != nil {
			panic(fmt.Sprintf("failed to synchronise blocks: %v", err))
		}
	}()
	<-starting
	checkProgress(t, tester.downloader, "completing", ethereum.SyncProgress{
		CurrentBlock: afterFailedSync.CurrentBlock,
		HighestBlock: uint64(validChain.len() - 1),
	})

	// Check final progress after successful sync.
	progress <- struct{}{}
	pending.Wait()
	checkProgress(t, tester.downloader, "final", ethereum.SyncProgress{
		CurrentBlock: uint64(validChain.len() - 1),
		HighestBlock: uint64(validChain.len() - 1),
	})
}

// This test reproduces an issue where unexpected deliveries would
// block indefinitely if they arrived at the right time.
func TestDeliverHeadersHang64Full(t *testing.T)  { testDeliverHeadersHang(t, 64, FullSync) }
func TestDeliverHeadersHang64Fast(t *testing.T)  { testDeliverHeadersHang(t, 64, FastSync) }
func TestDeliverHeadersHang65Full(t *testing.T)  { testDeliverHeadersHang(t, 65, FullSync) }
func TestDeliverHeadersHang65Fast(t *testing.T)  { testDeliverHeadersHang(t, 65, FastSync) }
func TestDeliverHeadersHang65Light(t *testing.T) { testDeliverHeadersHang(t, 65, LightSync) }

func testDeliverHeadersHang(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	master := newTester()
	defer master.terminate()
	chain := testChainBase.shorten(15)

	for i := 0; i < 200; i++ {
		tester := newTester()
		tester.peerDb = master.peerDb
		tester.newPeer("peer", protocol, chain)

		// Whenever the downloader requests headers, flood it with
		// a lot of unrequested header deliveries.
		tester.downloader.peers.peers["peer"].peer = &floodingTestPeer{
			peer:   tester.downloader.peers.peers["peer"].peer,
			tester: tester,
		}
		if err := tester.sync("peer", nil, mode); err != nil {
			t.Errorf("test %d: sync failed: %v", i, err)
		}
		tester.terminate()
	}
}

type floodingTestPeer struct {
	peer   Peer
	tester *downloadTester
}

func (ftp *floodingTestPeer) Head() (common.Hash, *big.Int) { return ftp.peer.Head() }
func (ftp *floodingTestPeer) RequestHeadersByHash(hash common.Hash, count int, skip int, reverse bool) error {
	return ftp.peer.RequestHeadersByHash(hash, count, skip, reverse)
}
func (ftp *floodingTestPeer) RequestBodies(hashes []common.Hash) error {
	return ftp.peer.RequestBodies(hashes)
}
func (ftp *floodingTestPeer) RequestReceipts(hashes []common.Hash) error {
	return ftp.peer.RequestReceipts(hashes)
}
func (ftp *floodingTestPeer) RequestNodeData(hashes []common.Hash) error {
	return ftp.peer.RequestNodeData(hashes)
}

func (ftp *floodingTestPeer) RequestHeadersByNumber(from uint64, count, skip int, reverse bool) error {
	deliveriesDone := make(chan struct{}, 500)
	for i := 0; i < cap(deliveriesDone)-1; i++ {
		peer := fmt.Sprintf("fake-peer%d", i)
		go func() {
			ftp.tester.downloader.DeliverHeaders(peer, []*types.Header{{}, {}, {}, {}})
			deliveriesDone <- struct{}{}
		}()
	}

	// None of the extra deliveries should block.
	timeout := time.After(60 * time.Second)
	launched := false
	for i := 0; i < cap(deliveriesDone); i++ {
		select {
		case <-deliveriesDone:
			if !launched {
				// Start delivering the requested headers
				// after one of the flooding responses has arrived.
				go func() {
					ftp.peer.RequestHeadersByNumber(from, count, skip, reverse)
					deliveriesDone <- struct{}{}
				}()
				launched = true
			}
		case <-timeout:
			panic("blocked")
		}
	}
	return nil
}

func TestRemoteHeaderRequestSpan(t *testing.T) {
	testCases := []struct {
		remoteHeight uint64
		localHeight  uint64
		expected     []int
	}{
		// Remote is way higher. We should ask for the remote head and go backwards
		{1500, 1000,
			[]int{1323, 1339, 1355, 1371, 1387, 1403, 1419, 1435, 1451, 1467, 1483, 1499},
		},
		{15000, 13006,
			[]int{14823, 14839, 14855, 14871, 14887, 14903, 14919, 14935, 14951, 14967, 14983, 14999},
		},
		// Remote is pretty close to us. We don't have to fetch as many
		{1200, 1150,
			[]int{1149, 1154, 1159, 1164, 1169, 1174, 1179, 1184, 1189, 1194, 1199},
		},
		// Remote is equal to us (so on a fork with higher td)
		// We should get the closest couple of ancestors
		{1500, 1500,
			[]int{1497, 1499},
		},
		// We're higher than the remote! Odd
		{1000, 1500,
			[]int{997, 999},
		},
		// Check some weird edgecases that it behaves somewhat rationally
		{0, 1500,
			[]int{0, 2},
		},
		{6000000, 0,
			[]int{5999823, 5999839, 5999855, 5999871, 5999887, 5999903, 5999919, 5999935, 5999951, 5999967, 5999983, 5999999},
		},
		{0, 0,
			[]int{0, 2},
		},
	}
	reqs := func(from, count, span int) []int {
		var r []int
		num := from
		for len(r) < count {
			r = append(r, num)
			num += span + 1
		}
		return r
	}
	for i, tt := range testCases {
		from, count, span, max := calculateRequestSpan(tt.remoteHeight, tt.localHeight)
		data := reqs(int(from), count, span)

		if max != uint64(data[len(data)-1]) {
			t.Errorf("test %d: wrong last value %d != %d", i, data[len(data)-1], max)
		}
		failed := false
		if len(data) != len(tt.expected) {
			failed = true
			t.Errorf("test %d: length wrong, expected %d got %d", i, len(tt.expected), len(data))
		} else {
			for j, n := range data {
				if n != tt.expected[j] {
					failed = true
					break
				}
			}
		}
		if failed {
			res := strings.Replace(fmt.Sprint(data), " ", ",", -1)
			exp := strings.Replace(fmt.Sprint(tt.expected), " ", ",", -1)
			t.Logf("got: %v\n", res)
			t.Logf("exp: %v\n", exp)
			t.Errorf("test %d: wrong values", i)
		}
	}
}

// Tests that peers below a pre-configured checkpoint block are prevented from
// being fast-synced from, avoiding potential cheap eclipse attacks.
func TestCheckpointEnforcement64Full(t *testing.T)  { testCheckpointEnforcement(t, 64, FullSync) }
func TestCheckpointEnforcement64Fast(t *testing.T)  { testCheckpointEnforcement(t, 64, FastSync) }
func TestCheckpointEnforcement65Full(t *testing.T)  { testCheckpointEnforcement(t, 65, FullSync) }
func TestCheckpointEnforcement65Fast(t *testing.T)  { testCheckpointEnforcement(t, 65, FastSync) }
func TestCheckpointEnforcement65Light(t *testing.T) { testCheckpointEnforcement(t, 65, LightSync) }

func testCheckpointEnforcement(t *testing.T, protocol uint, mode SyncMode) {
	t.Parallel()

	// Create a new tester with a particular hard coded checkpoint block
	tester := newTester()
	defer tester.terminate()

	tester.downloader.checkpoint = uint64(fsMinFullBlocks) + 256
	chain := testChainBase.shorten(int(tester.downloader.checkpoint) - 1)

	// Attempt to sync with the peer and validate the result
	tester.newPeer("peer", protocol, chain)

	var expect error
	if mode == FastSync || mode == LightSync {
		expect = errUnsyncedPeer
	}
	if err := tester.sync("peer", nil, mode); !errors.Is(err, expect) {
		t.Fatalf("block sync error mismatch: have %v, want %v", err, expect)
	}
	if mode == FastSync || mode == LightSync {
		assertOwnChain(t, tester, 1)
	} else {
		assertOwnChain(t, tester, chain.len())
	}
}
