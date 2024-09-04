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
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/eth/protocols/snap"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// downloadTester is a test simulator for mocking out local block chain.
type downloadTester struct {
	chain      *core.BlockChain
	downloader *Downloader

	peers map[string]*downloadTesterPeer
	lock  sync.RWMutex
}

// newTester creates a new downloader test mocker.
func newTester(t *testing.T) *downloadTester {
	return newTesterWithNotification(t, nil)
}

// newTesterWithNotification creates a new downloader test mocker.
func newTesterWithNotification(t *testing.T, success func()) *downloadTester {
	db, err := rawdb.NewDatabaseWithFreezer(rawdb.NewMemoryDatabase(), "", "", false)
	if err != nil {
		panic(err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	gspec := &core.Genesis{
		Config:  params.TestChainConfig,
		Alloc:   types.GenesisAlloc{testAddress: {Balance: big.NewInt(1000000000000000)}},
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	chain, err := core.NewBlockChain(db, nil, gspec, nil, ethash.NewFaker(), vm.Config{}, nil)
	if err != nil {
		panic(err)
	}
	tester := &downloadTester{
		chain: chain,
		peers: make(map[string]*downloadTesterPeer),
	}
	tester.downloader = New(db, new(event.TypeMux), tester.chain, tester.dropPeer, success)
	return tester
}

// terminate aborts any operations on the embedded downloader and releases all
// held resources.
func (dl *downloadTester) terminate() {
	dl.downloader.Terminate()
	dl.chain.Stop()
}

// newPeer registers a new block download source into the downloader.
func (dl *downloadTester) newPeer(id string, version uint, blocks []*types.Block) *downloadTesterPeer {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	peer := &downloadTesterPeer{
		dl:             dl,
		id:             id,
		chain:          newTestBlockchain(blocks),
		withholdBodies: make(map[common.Hash]struct{}),
	}
	dl.peers[id] = peer

	if err := dl.downloader.RegisterPeer(id, version, peer); err != nil {
		panic(err)
	}
	if err := dl.downloader.SnapSyncer.Register(peer); err != nil {
		panic(err)
	}
	return peer
}

// dropPeer simulates a hard peer removal from the connection pool.
func (dl *downloadTester) dropPeer(id string) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	delete(dl.peers, id)
	dl.downloader.SnapSyncer.Unregister(id)
	dl.downloader.UnregisterPeer(id)
}

type downloadTesterPeer struct {
	dl             *downloadTester
	withholdBodies map[common.Hash]struct{}
	id             string
	chain          *core.BlockChain
}

// Head constructs a function to retrieve a peer's current head hash
// and total difficulty.
func (dlp *downloadTesterPeer) Head() (common.Hash, *big.Int) {
	head := dlp.chain.CurrentBlock()
	return head.Hash(), dlp.chain.GetTd(head.Hash(), head.Number.Uint64())
}

func unmarshalRlpHeaders(rlpdata []rlp.RawValue) []*types.Header {
	var headers = make([]*types.Header, len(rlpdata))
	for i, data := range rlpdata {
		var h types.Header
		if err := rlp.DecodeBytes(data, &h); err != nil {
			panic(err)
		}
		headers[i] = &h
	}
	return headers
}

// RequestHeadersByHash constructs a GetBlockHeaders function based on a hashed
// origin; associated with a particular peer in the download tester. The returned
// function can be used to retrieve batches of headers from the particular peer.
func (dlp *downloadTesterPeer) RequestHeadersByHash(origin common.Hash, amount int, skip int, reverse bool, sink chan *eth.Response) (*eth.Request, error) {
	// Service the header query via the live handler code
	rlpHeaders := eth.ServiceGetBlockHeadersQuery(dlp.chain, &eth.GetBlockHeadersRequest{
		Origin: eth.HashOrNumber{
			Hash: origin,
		},
		Amount:  uint64(amount),
		Skip:    uint64(skip),
		Reverse: reverse,
	}, nil)
	headers := unmarshalRlpHeaders(rlpHeaders)
	hashes := make([]common.Hash, len(headers))
	for i, header := range headers {
		hashes[i] = header.Hash()
	}
	// Deliver the headers to the downloader
	req := &eth.Request{
		Peer: dlp.id,
	}
	res := &eth.Response{
		Req:  req,
		Res:  (*eth.BlockHeadersRequest)(&headers),
		Meta: hashes,
		Time: 1,
		Done: make(chan error, 1), // Ignore the returned status
	}
	go func() {
		sink <- res
	}()
	return req, nil
}

// RequestHeadersByNumber constructs a GetBlockHeaders function based on a numbered
// origin; associated with a particular peer in the download tester. The returned
// function can be used to retrieve batches of headers from the particular peer.
func (dlp *downloadTesterPeer) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool, sink chan *eth.Response) (*eth.Request, error) {
	// Service the header query via the live handler code
	rlpHeaders := eth.ServiceGetBlockHeadersQuery(dlp.chain, &eth.GetBlockHeadersRequest{
		Origin: eth.HashOrNumber{
			Number: origin,
		},
		Amount:  uint64(amount),
		Skip:    uint64(skip),
		Reverse: reverse,
	}, nil)
	headers := unmarshalRlpHeaders(rlpHeaders)
	hashes := make([]common.Hash, len(headers))
	for i, header := range headers {
		hashes[i] = header.Hash()
	}
	// Deliver the headers to the downloader
	req := &eth.Request{
		Peer: dlp.id,
	}
	res := &eth.Response{
		Req:  req,
		Res:  (*eth.BlockHeadersRequest)(&headers),
		Meta: hashes,
		Time: 1,
		Done: make(chan error, 1), // Ignore the returned status
	}
	go func() {
		sink <- res
	}()
	return req, nil
}

// RequestBodies constructs a getBlockBodies method associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of block bodies from the particularly requested peer.
func (dlp *downloadTesterPeer) RequestBodies(hashes []common.Hash, sink chan *eth.Response) (*eth.Request, error) {
	blobs := eth.ServiceGetBlockBodiesQuery(dlp.chain, hashes)

	bodies := make([]*eth.BlockBody, len(blobs))
	for i, blob := range blobs {
		bodies[i] = new(eth.BlockBody)
		rlp.DecodeBytes(blob, bodies[i])
	}
	var (
		txsHashes        = make([]common.Hash, len(bodies))
		uncleHashes      = make([]common.Hash, len(bodies))
		withdrawalHashes = make([]common.Hash, len(bodies))
		requestsHashes   = make([]common.Hash, len(bodies))
	)
	hasher := trie.NewStackTrie(nil)
	for i, body := range bodies {
		hash := types.DeriveSha(types.Transactions(body.Transactions), hasher)
		if _, ok := dlp.withholdBodies[hash]; ok {
			txsHashes = append(txsHashes[:i], txsHashes[i+1:]...)
			uncleHashes = append(uncleHashes[:i], uncleHashes[i+1:]...)
			continue
		}
		txsHashes[i] = hash
		uncleHashes[i] = types.CalcUncleHash(body.Uncles)
	}
	req := &eth.Request{
		Peer: dlp.id,
	}
	res := &eth.Response{
		Req:  req,
		Res:  (*eth.BlockBodiesResponse)(&bodies),
		Meta: [][]common.Hash{txsHashes, uncleHashes, withdrawalHashes, requestsHashes},
		Time: 1,
		Done: make(chan error, 1), // Ignore the returned status
	}
	go func() {
		sink <- res
	}()
	return req, nil
}

// RequestReceipts constructs a getReceipts method associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of block receipts from the particularly requested peer.
func (dlp *downloadTesterPeer) RequestReceipts(hashes []common.Hash, sink chan *eth.Response) (*eth.Request, error) {
	blobs := eth.ServiceGetReceiptsQuery(dlp.chain, hashes)

	receipts := make([][]*types.Receipt, len(blobs))
	for i, blob := range blobs {
		rlp.DecodeBytes(blob, &receipts[i])
	}
	hasher := trie.NewStackTrie(nil)
	hashes = make([]common.Hash, len(receipts))
	for i, receipt := range receipts {
		hashes[i] = types.DeriveSha(types.Receipts(receipt), hasher)
	}
	req := &eth.Request{
		Peer: dlp.id,
	}
	res := &eth.Response{
		Req:  req,
		Res:  (*eth.ReceiptsResponse)(&receipts),
		Meta: hashes,
		Time: 1,
		Done: make(chan error, 1), // Ignore the returned status
	}
	go func() {
		sink <- res
	}()
	return req, nil
}

// ID retrieves the peer's unique identifier.
func (dlp *downloadTesterPeer) ID() string {
	return dlp.id
}

// RequestAccountRange fetches a batch of accounts rooted in a specific account
// trie, starting with the origin.
func (dlp *downloadTesterPeer) RequestAccountRange(id uint64, root, origin, limit common.Hash, bytes uint64) error {
	// Create the request and service it
	req := &snap.GetAccountRangePacket{
		ID:     id,
		Root:   root,
		Origin: origin,
		Limit:  limit,
		Bytes:  bytes,
	}
	slimaccs, proofs := snap.ServiceGetAccountRangeQuery(dlp.chain, req)

	// We need to convert to non-slim format, delegate to the packet code
	res := &snap.AccountRangePacket{
		ID:       id,
		Accounts: slimaccs,
		Proof:    proofs,
	}
	hashes, accounts, _ := res.Unpack()

	go dlp.dl.downloader.SnapSyncer.OnAccounts(dlp, id, hashes, accounts, proofs)
	return nil
}

// RequestStorageRanges fetches a batch of storage slots belonging to one or
// more accounts. If slots from only one account is requested, an origin marker
// may also be used to retrieve from there.
func (dlp *downloadTesterPeer) RequestStorageRanges(id uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, bytes uint64) error {
	// Create the request and service it
	req := &snap.GetStorageRangesPacket{
		ID:       id,
		Accounts: accounts,
		Root:     root,
		Origin:   origin,
		Limit:    limit,
		Bytes:    bytes,
	}
	storage, proofs := snap.ServiceGetStorageRangesQuery(dlp.chain, req)

	// We need to convert to demultiplex, delegate to the packet code
	res := &snap.StorageRangesPacket{
		ID:    id,
		Slots: storage,
		Proof: proofs,
	}
	hashes, slots := res.Unpack()

	go dlp.dl.downloader.SnapSyncer.OnStorage(dlp, id, hashes, slots, proofs)
	return nil
}

// RequestByteCodes fetches a batch of bytecodes by hash.
func (dlp *downloadTesterPeer) RequestByteCodes(id uint64, hashes []common.Hash, bytes uint64) error {
	req := &snap.GetByteCodesPacket{
		ID:     id,
		Hashes: hashes,
		Bytes:  bytes,
	}
	codes := snap.ServiceGetByteCodesQuery(dlp.chain, req)
	go dlp.dl.downloader.SnapSyncer.OnByteCodes(dlp, id, codes)
	return nil
}

// RequestTrieNodes fetches a batch of account or storage trie nodes rooted in
// a specific state trie.
func (dlp *downloadTesterPeer) RequestTrieNodes(id uint64, root common.Hash, paths []snap.TrieNodePathSet, bytes uint64) error {
	req := &snap.GetTrieNodesPacket{
		ID:    id,
		Root:  root,
		Paths: paths,
		Bytes: bytes,
	}
	nodes, _ := snap.ServiceGetTrieNodesQuery(dlp.chain, req, time.Now())
	go dlp.dl.downloader.SnapSyncer.OnTrieNodes(dlp, id, nodes)
	return nil
}

// Log retrieves the peer's own contextual logger.
func (dlp *downloadTesterPeer) Log() log.Logger {
	return log.New("peer", dlp.id)
}

// assertOwnChain checks if the local chain contains the correct number of items
// of the various chain components.
func assertOwnChain(t *testing.T, tester *downloadTester, length int) {
	// Mark this method as a helper to report errors at callsite, not in here
	t.Helper()

	headers, blocks, receipts := length, length, length
	if hs := int(tester.chain.CurrentHeader().Number.Uint64()) + 1; hs != headers {
		t.Fatalf("synchronised headers mismatch: have %v, want %v", hs, headers)
	}
	if bs := int(tester.chain.CurrentBlock().Number.Uint64()) + 1; bs != blocks {
		t.Fatalf("synchronised blocks mismatch: have %v, want %v", bs, blocks)
	}
	if rs := int(tester.chain.CurrentSnapBlock().Number.Uint64()) + 1; rs != receipts {
		t.Fatalf("synchronised receipts mismatch: have %v, want %v", rs, receipts)
	}
}

func TestCanonicalSynchronisation68Full(t *testing.T) { testCanonSync(t, eth.ETH68, FullSync) }
func TestCanonicalSynchronisation68Snap(t *testing.T) { testCanonSync(t, eth.ETH68, SnapSync) }

func testCanonSync(t *testing.T, protocol uint, mode SyncMode) {
	success := make(chan struct{})
	tester := newTesterWithNotification(t, func() {
		close(success)
	})
	defer tester.terminate()

	// Create a small enough block chain to download
	chain := testChainBase.shorten(blockCacheMaxItems - 15)
	tester.newPeer("peer", protocol, chain.blocks[1:])

	// Synchronise with the peer and make sure all relevant data was retrieved
	if err := tester.downloader.BeaconSync(mode, chain.blocks[len(chain.blocks)-1].Header(), nil); err != nil {
		t.Fatalf("failed to beacon-sync chain: %v", err)
	}
	select {
	case <-success:
		assertOwnChain(t, tester, len(chain.blocks))
	case <-time.NewTimer(time.Second * 3).C:
		t.Fatalf("Failed to sync chain in three seconds")
	}
}

// Tests that if a large batch of blocks are being downloaded, it is throttled
// until the cached blocks are retrieved.
func TestThrottling68Full(t *testing.T) { testThrottling(t, eth.ETH68, FullSync) }
func TestThrottling68Snap(t *testing.T) { testThrottling(t, eth.ETH68, SnapSync) }

func testThrottling(t *testing.T, protocol uint, mode SyncMode) {
	tester := newTester(t)
	defer tester.terminate()

	// Create a long block chain to download and the tester
	targetBlocks := len(testChainBase.blocks) - 1
	tester.newPeer("peer", protocol, testChainBase.blocks[1:])

	// Wrap the importer to allow stepping
	var blocked atomic.Uint32
	proceed := make(chan struct{})
	tester.downloader.chainInsertHook = func(results []*fetchResult) {
		blocked.Store(uint32(len(results)))
		<-proceed
	}
	// Start a synchronisation concurrently
	errc := make(chan error, 1)
	go func() {
		errc <- tester.downloader.BeaconSync(mode, testChainBase.blocks[len(testChainBase.blocks)-1].Header(), nil)
	}()
	// Iteratively take some blocks, always checking the retrieval count
	for {
		// Check the retrieval count synchronously (! reason for this ugly block)
		tester.lock.RLock()
		retrieved := int(tester.chain.CurrentSnapBlock().Number.Uint64()) + 1
		tester.lock.RUnlock()
		if retrieved >= targetBlocks+1 {
			break
		}
		// Wait a bit for sync to throttle itself
		var cached, frozen int
		for start := time.Now(); time.Since(start) < 3*time.Second; {
			time.Sleep(25 * time.Millisecond)

			tester.lock.Lock()
			tester.downloader.queue.lock.Lock()
			tester.downloader.queue.resultCache.lock.Lock()
			{
				cached = tester.downloader.queue.resultCache.countCompleted()
				frozen = int(blocked.Load())
				retrieved = int(tester.chain.CurrentSnapBlock().Number.Uint64()) + 1
			}
			tester.downloader.queue.resultCache.lock.Unlock()
			tester.downloader.queue.lock.Unlock()
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
		retrieved = int(tester.chain.CurrentSnapBlock().Number.Uint64()) + 1
		tester.lock.RUnlock()
		if cached != blockCacheMaxItems && cached != blockCacheMaxItems-reorgProtHeaderDelay && retrieved+cached+frozen != targetBlocks+1 && retrieved+cached+frozen != targetBlocks+1-reorgProtHeaderDelay {
			t.Fatalf("block count mismatch: have %v, want %v (owned %v, blocked %v, target %v)", cached, blockCacheMaxItems, retrieved, frozen, targetBlocks+1)
		}
		// Permit the blocked blocks to import
		if blocked.Load() > 0 {
			blocked.Store(uint32(0))
			proceed <- struct{}{}
		}
	}
	// Check that we haven't pulled more blocks than available
	assertOwnChain(t, tester, targetBlocks+1)
	if err := <-errc; err != nil {
		t.Fatalf("block synchronization failed: %v", err)
	}
}

// Tests that a canceled download wipes all previously accumulated state.
func TestCancel68Full(t *testing.T) { testCancel(t, eth.ETH68, FullSync) }
func TestCancel68Snap(t *testing.T) { testCancel(t, eth.ETH68, SnapSync) }

func testCancel(t *testing.T, protocol uint, mode SyncMode) {
	complete := make(chan struct{})
	success := func() {
		close(complete)
	}
	tester := newTesterWithNotification(t, success)
	defer tester.terminate()

	chain := testChainBase.shorten(MaxHeaderFetch)
	tester.newPeer("peer", protocol, chain.blocks[1:])

	// Make sure canceling works with a pristine downloader
	tester.downloader.Cancel()
	if !tester.downloader.queue.Idle() {
		t.Errorf("download queue not idle")
	}
	// Synchronise with the peer, but cancel afterwards
	if err := tester.downloader.BeaconSync(mode, chain.blocks[len(chain.blocks)-1].Header(), nil); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	<-complete
	tester.downloader.Cancel()
	if !tester.downloader.queue.Idle() {
		t.Errorf("download queue not idle")
	}
}

// Tests that synchronisations behave well in multi-version protocol environments
// and not wreak havoc on other nodes in the network.
func TestMultiProtoSynchronisation68Full(t *testing.T) { testMultiProtoSync(t, eth.ETH68, FullSync) }
func TestMultiProtoSynchronisation68Snap(t *testing.T) { testMultiProtoSync(t, eth.ETH68, SnapSync) }

func testMultiProtoSync(t *testing.T, protocol uint, mode SyncMode) {
	complete := make(chan struct{})
	success := func() {
		close(complete)
	}
	tester := newTesterWithNotification(t, success)
	defer tester.terminate()

	// Create a small enough block chain to download
	chain := testChainBase.shorten(blockCacheMaxItems - 15)

	// Create peers of every type
	tester.newPeer("peer 68", eth.ETH68, chain.blocks[1:])

	if err := tester.downloader.BeaconSync(mode, chain.blocks[len(chain.blocks)-1].Header(), nil); err != nil {
		t.Fatalf("failed to start beacon sync: #{err}")
	}
	select {
	case <-complete:
		break
	case <-time.NewTimer(time.Second * 3).C:
		t.Fatalf("Failed to sync chain in three seconds")
	}
	assertOwnChain(t, tester, len(chain.blocks))

	// Check that no peers have been dropped off
	for _, version := range []int{68} {
		peer := fmt.Sprintf("peer %d", version)
		if _, ok := tester.peers[peer]; !ok {
			t.Errorf("%s dropped", peer)
		}
	}
}

// Tests that if a block is empty (e.g. header only), no body request should be
// made, and instead the header should be assembled into a whole block in itself.
func TestEmptyShortCircuit68Full(t *testing.T) { testEmptyShortCircuit(t, eth.ETH68, FullSync) }
func TestEmptyShortCircuit68Snap(t *testing.T) { testEmptyShortCircuit(t, eth.ETH68, SnapSync) }

func testEmptyShortCircuit(t *testing.T, protocol uint, mode SyncMode) {
	success := make(chan struct{})
	tester := newTesterWithNotification(t, func() {
		close(success)
	})
	defer tester.terminate()

	// Create a block chain to download
	chain := testChainBase
	tester.newPeer("peer", protocol, chain.blocks[1:])

	// Instrument the downloader to signal body requests
	var bodiesHave, receiptsHave atomic.Int32
	tester.downloader.bodyFetchHook = func(headers []*types.Header) {
		bodiesHave.Add(int32(len(headers)))
	}
	tester.downloader.receiptFetchHook = func(headers []*types.Header) {
		receiptsHave.Add(int32(len(headers)))
	}

	if err := tester.downloader.BeaconSync(mode, chain.blocks[len(chain.blocks)-1].Header(), nil); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	select {
	case <-success:
		checkProgress(t, tester.downloader, "initial", ethereum.SyncProgress{
			HighestBlock: uint64(len(chain.blocks) - 1),
			CurrentBlock: uint64(len(chain.blocks) - 1),
		})
	case <-time.NewTimer(time.Second * 3).C:
		t.Fatalf("Failed to sync chain in three seconds")
	}
	assertOwnChain(t, tester, len(chain.blocks))

	// Validate the number of block bodies that should have been requested
	bodiesNeeded, receiptsNeeded := 0, 0
	for _, block := range chain.blocks[1:] {
		if len(block.Transactions()) > 0 || len(block.Uncles()) > 0 {
			bodiesNeeded++
		}
	}
	for _, block := range chain.blocks[1:] {
		if mode == SnapSync && len(block.Transactions()) > 0 {
			receiptsNeeded++
		}
	}
	if int(bodiesHave.Load()) != bodiesNeeded {
		t.Errorf("body retrieval count mismatch: have %v, want %v", bodiesHave.Load(), bodiesNeeded)
	}
	if int(receiptsHave.Load()) != receiptsNeeded {
		t.Errorf("receipt retrieval count mismatch: have %v, want %v", receiptsHave.Load(), receiptsNeeded)
	}
}

func checkProgress(t *testing.T, d *Downloader, stage string, want ethereum.SyncProgress) {
	// Mark this method as a helper to report errors at callsite, not in here
	t.Helper()

	p := d.Progress()
	if p.StartingBlock != want.StartingBlock || p.CurrentBlock != want.CurrentBlock || p.HighestBlock != want.HighestBlock {
		t.Fatalf("%s progress mismatch:\nhave %+v\nwant %+v", stage, p, want)
	}
}

// Tests that peers below a pre-configured checkpoint block are prevented from
// being fast-synced from, avoiding potential cheap eclipse attacks.
func TestBeaconSync68Full(t *testing.T) { testBeaconSync(t, eth.ETH68, FullSync) }
func TestBeaconSync68Snap(t *testing.T) { testBeaconSync(t, eth.ETH68, SnapSync) }

func testBeaconSync(t *testing.T, protocol uint, mode SyncMode) {
	var cases = []struct {
		name  string // The name of testing scenario
		local int    // The length of local chain(canonical chain assumed), 0 means genesis is the head
	}{
		{name: "Beacon sync since genesis", local: 0},
		{name: "Beacon sync with short local chain", local: 1},
		{name: "Beacon sync with long local chain", local: blockCacheMaxItems - 15 - fsMinFullBlocks/2},
		{name: "Beacon sync with full local chain", local: blockCacheMaxItems - 15 - 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			success := make(chan struct{})
			tester := newTesterWithNotification(t, func() {
				close(success)
			})
			defer tester.terminate()

			chain := testChainBase.shorten(blockCacheMaxItems - 15)
			tester.newPeer("peer", protocol, chain.blocks[1:])

			// Build the local chain segment if it's required
			if c.local > 0 {
				tester.chain.InsertChain(chain.blocks[1 : c.local+1])
			}
			if err := tester.downloader.BeaconSync(mode, chain.blocks[len(chain.blocks)-1].Header(), nil); err != nil {
				t.Fatalf("Failed to beacon sync chain %v %v", c.name, err)
			}
			select {
			case <-success:
				// Ok, downloader fully cancelled after sync cycle
				if bs := int(tester.chain.CurrentBlock().Number.Uint64()) + 1; bs != len(chain.blocks) {
					t.Fatalf("synchronised blocks mismatch: have %v, want %v", bs, len(chain.blocks))
				}
			case <-time.NewTimer(time.Second * 3).C:
				t.Fatalf("Failed to sync chain in three seconds")
			}
		})
	}
}

// Tests that synchronisation progress (origin block number, current block number
// and highest block number) is tracked and updated correctly.
func TestSyncProgress68Full(t *testing.T) { testSyncProgress(t, eth.ETH68, FullSync) }
func TestSyncProgress68Snap(t *testing.T) { testSyncProgress(t, eth.ETH68, SnapSync) }

func testSyncProgress(t *testing.T, protocol uint, mode SyncMode) {
	success := make(chan struct{})
	tester := newTesterWithNotification(t, func() {
		success <- struct{}{}
	})
	defer tester.terminate()
	checkProgress(t, tester.downloader, "pristine", ethereum.SyncProgress{})

	chain := testChainBase.shorten(blockCacheMaxItems - 15)
	shortChain := chain.shorten(len(chain.blocks) / 2).blocks[1:]

	// Connect to peer that provides all headers and part of the bodies
	faultyPeer := tester.newPeer("peer-half", protocol, shortChain)
	for _, header := range shortChain {
		faultyPeer.withholdBodies[header.Hash()] = struct{}{}
	}

	if err := tester.downloader.BeaconSync(mode, chain.blocks[len(chain.blocks)/2-1].Header(), nil); err != nil {
		t.Fatalf("failed to beacon-sync chain: %v", err)
	}
	select {
	case <-success:
		// Ok, downloader fully cancelled after sync cycle
		checkProgress(t, tester.downloader, "peer-half", ethereum.SyncProgress{
			CurrentBlock: uint64(len(chain.blocks)/2 - 1),
			HighestBlock: uint64(len(chain.blocks)/2 - 1),
		})
	case <-time.NewTimer(time.Second * 3).C:
		t.Fatalf("Failed to sync chain in three seconds")
	}

	// Synchronise all the blocks and check continuation progress
	tester.newPeer("peer-full", protocol, chain.blocks[1:])
	if err := tester.downloader.BeaconSync(mode, chain.blocks[len(chain.blocks)-1].Header(), nil); err != nil {
		t.Fatalf("failed to beacon-sync chain: %v", err)
	}
	startingBlock := uint64(len(chain.blocks)/2 - 1)

	select {
	case <-success:
		// Ok, downloader fully cancelled after sync cycle
		checkProgress(t, tester.downloader, "peer-full", ethereum.SyncProgress{
			StartingBlock: startingBlock,
			CurrentBlock:  uint64(len(chain.blocks) - 1),
			HighestBlock:  uint64(len(chain.blocks) - 1),
		})
	case <-time.NewTimer(time.Second * 3).C:
		t.Fatalf("Failed to sync chain in three seconds")
	}
}
