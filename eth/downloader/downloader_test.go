package downloader

import (
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

var (
	knownHash   = common.Hash{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	unknownHash = common.Hash{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9}
	bannedHash  = common.Hash{5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}

	genesis = createBlock(1, common.Hash{}, knownHash)
)

func createHashes(start, amount int) (hashes []common.Hash) {
	hashes = make([]common.Hash, amount+1)
	hashes[len(hashes)-1] = knownHash

	for i := range hashes[:len(hashes)-1] {
		binary.BigEndian.PutUint64(hashes[i][:8], uint64(start+i+2))
	}
	return
}

func createBlock(i int, parent, hash common.Hash) *types.Block {
	header := &types.Header{Number: big.NewInt(int64(i))}
	block := types.NewBlockWithHeader(header)
	block.HeaderHash = hash
	block.ParentHeaderHash = parent
	return block
}

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

type downloadTester struct {
	downloader *Downloader

	ownHashes  []common.Hash                           // Hash chain belonging to the tester
	ownBlocks  map[common.Hash]*types.Block            // Blocks belonging to the tester
	peerHashes map[string][]common.Hash                // Hash chain belonging to different test peers
	peerBlocks map[string]map[common.Hash]*types.Block // Blocks belonging to different test peers

	maxHashFetch int // Overrides the maximum number of retrieved hashes
}

func newTester() *downloadTester {
	tester := &downloadTester{
		ownHashes:  []common.Hash{knownHash},
		ownBlocks:  map[common.Hash]*types.Block{knownHash: genesis},
		peerHashes: make(map[string][]common.Hash),
		peerBlocks: make(map[string]map[common.Hash]*types.Block),
	}
	var mux event.TypeMux
	downloader := New(&mux, tester.hasBlock, tester.getBlock, nil)
	tester.downloader = downloader

	return tester
}

// syncTake is starts synchronising with a remote peer, but concurrently it also
// starts fetching blocks that the downloader retrieved. IT blocks until both go
// routines terminate.
func (dl *downloadTester) syncTake(peerId string, head common.Hash) ([]*Block, error) {
	// Start a block collector to take blocks as they become available
	done := make(chan struct{})
	took := []*Block{}
	go func() {
		for running := true; running; {
			select {
			case <-done:
				running = false
			default:
				time.Sleep(time.Millisecond)
			}
			// Take a batch of blocks and accumulate
			blocks := dl.downloader.TakeBlocks()
			for _, block := range blocks {
				dl.ownHashes = append(dl.ownHashes, block.RawBlock.Hash())
				dl.ownBlocks[block.RawBlock.Hash()] = block.RawBlock
			}
			took = append(took, blocks...)
		}
		done <- struct{}{}
	}()
	// Start the downloading, sync the taker and return
	err := dl.downloader.synchronise(peerId, head)

	done <- struct{}{}
	<-done

	return took, err
}

// hasBlock checks if a block is present in the testers canonical chain.
func (dl *downloadTester) hasBlock(hash common.Hash) bool {
	return dl.getBlock(hash) != nil
}

// getBlock retrieves a block from the testers canonical chain.
func (dl *downloadTester) getBlock(hash common.Hash) *types.Block {
	return dl.ownBlocks[hash]
}

// newPeer registers a new block download source into the downloader.
func (dl *downloadTester) newPeer(id string, hashes []common.Hash, blocks map[common.Hash]*types.Block) error {
	err := dl.downloader.RegisterPeer(id, hashes[0], dl.peerGetHashesFn(id), dl.peerGetBlocksFn(id))
	if err == nil {
		// Assign the owned hashes and blocks to the peer
		dl.peerHashes[id] = hashes
		dl.peerBlocks[id] = blocks
	}
	return err
}

// peerGetBlocksFn constructs a getHashes function associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of hashes from the particularly requested peer.
func (dl *downloadTester) peerGetHashesFn(id string) func(head common.Hash) error {
	return func(head common.Hash) error {
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

// peerGetBlocksFn constructs a getBlocks function associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of blocks from the particularly requested peer.
func (dl *downloadTester) peerGetBlocksFn(id string) func([]common.Hash) error {
	return func(hashes []common.Hash) error {
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
func TestSynchronisation(t *testing.T) {
	// Create a small enough block chain to download and the tester
	targetBlocks := blockCacheLimit - 15
	hashes := createHashes(0, targetBlocks)
	blocks := createBlocksFromHashes(hashes)

	tester := newTester()
	tester.newPeer("peer", hashes, blocks)

	// Synchronise with the peer and make sure all blocks were retrieved
	if err := tester.downloader.synchronise("peer", hashes[0]); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if queued := len(tester.downloader.queue.blockPool); queued != targetBlocks {
		t.Fatalf("synchronised block mismatch: have %v, want %v", queued, targetBlocks)
	}
}

// Tests that the synchronized blocks can be correctly retrieved.
func TestBlockTaking(t *testing.T) {
	// Create a small enough block chain to download and the tester
	targetBlocks := blockCacheLimit - 15
	hashes := createHashes(0, targetBlocks)
	blocks := createBlocksFromHashes(hashes)

	tester := newTester()
	tester.newPeer("peer", hashes, blocks)

	// Synchronise with the peer and test block retrieval
	if err := tester.downloader.synchronise("peer", hashes[0]); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if took := tester.downloader.TakeBlocks(); len(took) != targetBlocks {
		t.Fatalf("took block mismatch: have %v, want %v", len(took), targetBlocks)
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
func TestCancel(t *testing.T) {
	// Create a small enough block chain to download and the tester
	targetBlocks := blockCacheLimit - 15
	hashes := createHashes(0, targetBlocks)
	blocks := createBlocksFromHashes(hashes)

	tester := newTester()
	tester.newPeer("peer", hashes, blocks)

	// Synchronise with the peer, but cancel afterwards
	if err := tester.downloader.synchronise("peer", hashes[0]); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if !tester.downloader.Cancel() {
		t.Fatalf("cancel operation failed")
	}
	// Make sure the queue reports empty and no blocks can be taken
	hashCount, blockCount := tester.downloader.queue.Size()
	if hashCount > 0 || blockCount > 0 {
		t.Errorf("block or hash count mismatch: %d hashes, %d blocks, want 0", hashCount, blockCount)
	}
	if took := tester.downloader.TakeBlocks(); len(took) != 0 {
		t.Errorf("taken blocks mismatch: have %d, want %d", len(took), 0)
	}
}

// Tests that if a large batch of blocks are being downloaded, it is throttled
// until the cached blocks are retrieved.
func TestThrottling(t *testing.T) {
	// Create a long block chain to download and the tester
	targetBlocks := 8 * blockCacheLimit
	hashes := createHashes(0, targetBlocks)
	blocks := createBlocksFromHashes(hashes)

	tester := newTester()
	tester.newPeer("peer", hashes, blocks)

	// Start a synchronisation concurrently
	errc := make(chan error)
	go func() {
		errc <- tester.downloader.synchronise("peer", hashes[0])
	}()
	// Iteratively take some blocks, always checking the retrieval count
	for total := 0; total < targetBlocks; {
		// Wait a bit for sync to complete
		for start := time.Now(); time.Since(start) < 3*time.Second; {
			time.Sleep(25 * time.Millisecond)
			if len(tester.downloader.queue.blockPool) == blockCacheLimit {
				break
			}
		}
		// Fetch the next batch of blocks
		took := tester.downloader.TakeBlocks()
		if len(took) != blockCacheLimit {
			t.Fatalf("block count mismatch: have %v, want %v", len(took), blockCacheLimit)
		}
		total += len(took)
		if total > targetBlocks {
			t.Fatalf("target block count mismatch: have %v, want %v", total, targetBlocks)
		}
	}
	if err := <-errc; err != nil {
		t.Fatalf("block synchronization failed: %v", err)
	}
}

// Tests that if a peer returns an invalid chain with a block pointing to a non-
// existing parent, it is correctly detected and handled.
func TestNonExistingParentAttack(t *testing.T) {
	// Forge a single-link chain with a forged header
	hashes := createHashes(0, 1)
	blocks := createBlocksFromHashes(hashes)

	forged := blocks[hashes[0]]
	forged.ParentHeaderHash = unknownHash

	// Try and sync with the malicious node and check that it fails
	tester := newTester()
	tester.newPeer("attack", hashes, blocks)
	if err := tester.downloader.synchronise("attack", hashes[0]); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	bs := tester.downloader.TakeBlocks()
	if len(bs) != 1 {
		t.Fatalf("retrieved block mismatch: have %v, want %v", len(bs), 1)
	}
	if tester.hasBlock(bs[0].RawBlock.ParentHash()) {
		t.Fatalf("tester knows about the unknown hash")
	}
	tester.downloader.Cancel()

	// Reconstruct a valid chain, and try to synchronize with it
	forged.ParentHeaderHash = knownHash
	tester.newPeer("valid", hashes, blocks)
	if err := tester.downloader.synchronise("valid", hashes[0]); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	bs = tester.downloader.TakeBlocks()
	if len(bs) != 1 {
		t.Fatalf("retrieved block mismatch: have %v, want %v", len(bs), 1)
	}
	if !tester.hasBlock(bs[0].RawBlock.ParentHash()) {
		t.Fatalf("tester doesn't know about the origin hash")
	}
}

// Tests that if a malicious peers keeps sending us repeating hashes, we don't
// loop indefinitely.
func TestRepeatingHashAttack(t *testing.T) {
	// Create a valid chain, but drop the last link
	hashes := createHashes(0, blockCacheLimit)
	blocks := createBlocksFromHashes(hashes)
	forged := hashes[:len(hashes)-1]

	// Try and sync with the malicious node
	tester := newTester()
	tester.newPeer("attack", forged, blocks)

	errc := make(chan error)
	go func() {
		errc <- tester.downloader.synchronise("attack", hashes[0])
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
	tester.newPeer("valid", hashes, blocks)
	if err := tester.downloader.synchronise("valid", hashes[0]); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
}

// Tests that if a malicious peers returns a non-existent block hash, it should
// eventually time out and the sync reattempted.
func TestNonExistingBlockAttack(t *testing.T) {
	// Create a valid chain, but forge the last link
	hashes := createHashes(0, blockCacheLimit)
	blocks := createBlocksFromHashes(hashes)
	origin := hashes[len(hashes)/2]

	hashes[len(hashes)/2] = unknownHash

	// Try and sync with the malicious node and check that it fails
	tester := newTester()
	tester.newPeer("attack", hashes, blocks)
	if err := tester.downloader.synchronise("attack", hashes[0]); err != errPeersUnavailable {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errPeersUnavailable)
	}
	// Ensure that a valid chain can still pass sync
	hashes[len(hashes)/2] = origin
	tester.newPeer("valid", hashes, blocks)
	if err := tester.downloader.synchronise("valid", hashes[0]); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
}

// Tests that if a malicious peer is returning hashes in a weird order, that the
// sync throttler doesn't choke on them waiting for the valid blocks.
func TestInvalidHashOrderAttack(t *testing.T) {
	// Create a valid long chain, but reverse some hashes within
	hashes := createHashes(0, 4*blockCacheLimit)
	blocks := createBlocksFromHashes(hashes)

	chunk1 := make([]common.Hash, blockCacheLimit)
	chunk2 := make([]common.Hash, blockCacheLimit)
	copy(chunk1, hashes[blockCacheLimit:2*blockCacheLimit])
	copy(chunk2, hashes[2*blockCacheLimit:3*blockCacheLimit])

	reverse := make([]common.Hash, len(hashes))
	copy(reverse, hashes)
	copy(reverse[2*blockCacheLimit:], chunk1)
	copy(reverse[blockCacheLimit:], chunk2)

	// Try and sync with the malicious node and check that it fails
	tester := newTester()
	tester.newPeer("attack", reverse, blocks)
	if _, err := tester.syncTake("attack", reverse[0]); err != errInvalidChain {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errInvalidChain)
	}
	// Ensure that a valid chain can still pass sync
	tester.newPeer("valid", hashes, blocks)
	if _, err := tester.syncTake("valid", hashes[0]); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
}

// Tests that if a malicious peer makes up a random hash chain and tries to push
// indefinitely, it actually gets caught with it.
func TestMadeupHashChainAttack(t *testing.T) {
	blockSoftTTL = 100 * time.Millisecond
	crossCheckCycle = 25 * time.Millisecond

	// Create a long chain of hashes without backing blocks
	hashes := createHashes(0, 1024*blockCacheLimit)

	// Try and sync with the malicious node and check that it fails
	tester := newTester()
	tester.newPeer("attack", hashes, nil)
	if _, err := tester.syncTake("attack", hashes[0]); err != errCrossCheckFailed {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errCrossCheckFailed)
	}
}

// Tests that if a malicious peer makes up a random hash chain, and tries to push
// indefinitely, one hash at a time, it actually gets caught with it. The reason
// this is separate from the classical made up chain attack is that sending hashes
// one by one prevents reliable block/parent verification.
func TestMadeupHashChainDrippingAttack(t *testing.T) {
	// Create a random chain of hashes to drip
	hashes := createHashes(0, 16*blockCacheLimit)
	tester := newTester()

	// Try and sync with the attacker, one hash at a time
	tester.maxHashFetch = 1
	tester.newPeer("attack", hashes, nil)
	if _, err := tester.syncTake("attack", hashes[0]); err != errStallingPeer {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errStallingPeer)
	}
}

// Tests that if a malicious peer makes up a random block chain, and tried to
// push indefinitely, it actually gets caught with it.
func TestMadeupBlockChainAttack(t *testing.T) {
	defaultBlockTTL := blockSoftTTL
	defaultCrossCheckCycle := crossCheckCycle

	blockSoftTTL = 100 * time.Millisecond
	crossCheckCycle = 25 * time.Millisecond

	// Create a long chain of blocks and simulate an invalid chain by dropping every second
	hashes := createHashes(0, 16*blockCacheLimit)
	blocks := createBlocksFromHashes(hashes)

	gapped := make([]common.Hash, len(hashes)/2)
	for i := 0; i < len(gapped); i++ {
		gapped[i] = hashes[2*i]
	}
	// Try and sync with the malicious node and check that it fails
	tester := newTester()
	tester.newPeer("attack", gapped, blocks)
	if _, err := tester.syncTake("attack", gapped[0]); err != errCrossCheckFailed {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errCrossCheckFailed)
	}
	// Ensure that a valid chain can still pass sync
	blockSoftTTL = defaultBlockTTL
	crossCheckCycle = defaultCrossCheckCycle

	tester.newPeer("valid", hashes, blocks)
	if _, err := tester.syncTake("valid", hashes[0]); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
}

// Advanced form of the above forged blockchain attack, where not only does the
// attacker make up a valid hashes for random blocks, but also forges the block
// parents to point to existing hashes.
func TestMadeupParentBlockChainAttack(t *testing.T) {
	defaultBlockTTL := blockSoftTTL
	defaultCrossCheckCycle := crossCheckCycle

	blockSoftTTL = 100 * time.Millisecond
	crossCheckCycle = 25 * time.Millisecond

	// Create a long chain of blocks and simulate an invalid chain by dropping every second
	hashes := createHashes(0, 16*blockCacheLimit)
	blocks := createBlocksFromHashes(hashes)
	forges := createBlocksFromHashes(hashes)
	for hash, block := range forges {
		block.ParentHeaderHash = hash // Simulate pointing to already known hash
	}
	// Try and sync with the malicious node and check that it fails
	tester := newTester()
	tester.newPeer("attack", hashes, forges)
	if _, err := tester.syncTake("attack", hashes[0]); err != errCrossCheckFailed {
		t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errCrossCheckFailed)
	}
	// Ensure that a valid chain can still pass sync
	blockSoftTTL = defaultBlockTTL
	crossCheckCycle = defaultCrossCheckCycle

	tester.newPeer("valid", hashes, blocks)
	if _, err := tester.syncTake("valid", hashes[0]); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
}

// Tests that if one/multiple malicious peers try to feed a banned blockchain to
// the downloader, it will not keep refetching the same chain indefinitely, but
// gradually block pieces of it, until it's head is also blocked.
func TestBannedChainStarvationAttack(t *testing.T) {
	// Construct a valid chain, but ban one of the hashes in it
	hashes := createHashes(0, 8*blockCacheLimit)
	hashes[len(hashes)/2+23] = bannedHash // weird index to have non multiple of ban chunk size

	blocks := createBlocksFromHashes(hashes)

	// Create the tester and ban the selected hash
	tester := newTester()
	tester.downloader.banned.Add(bannedHash)

	// Iteratively try to sync, and verify that the banned hash list grows until
	// the head of the invalid chain is blocked too.
	tester.newPeer("attack", hashes, blocks)
	for banned := tester.downloader.banned.Size(); ; {
		// Try to sync with the attacker, check hash chain failure
		if _, err := tester.syncTake("attack", hashes[0]); err != errInvalidChain {
			t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errInvalidChain)
		}
		// Check that the ban list grew with at least 1 new item, or all banned
		bans := tester.downloader.banned.Size()
		if bans < banned+1 {
			if tester.downloader.banned.Has(hashes[0]) {
				break
			}
			t.Fatalf("ban count mismatch: have %v, want %v+", bans, banned+1)
		}
		banned = bans
	}
	// Check that after banning an entire chain, bad peers get dropped
	if err := tester.newPeer("new attacker", hashes, blocks); err != errBannedHead {
		t.Fatalf("peer registration mismatch: have %v, want %v", err, errBannedHead)
	}
	if peer := tester.downloader.peers.Peer("net attacker"); peer != nil {
		t.Fatalf("banned attacker registered: %v", peer)
	}
}

// Tests that if a peer sends excessively many/large invalid chains that are
// gradually banned, it will have an upper limit on the consumed memory and also
// the origin bad hashes will not be evacuated.
func TestBannedChainMemoryExhaustionAttack(t *testing.T) {
	// Reduce the test size a bit
	MaxBlockFetch = 4
	maxBannedHashes = 256

	// Construct a banned chain with more chunks than the ban limit
	hashes := createHashes(0, maxBannedHashes*MaxBlockFetch)
	hashes[len(hashes)-1] = bannedHash // weird index to have non multiple of ban chunk size

	blocks := createBlocksFromHashes(hashes)

	// Create the tester and ban the selected hash
	tester := newTester()
	tester.downloader.banned.Add(bannedHash)

	// Iteratively try to sync, and verify that the banned hash list grows until
	// the head of the invalid chain is blocked too.
	tester.newPeer("attack", hashes, blocks)
	for {
		// Try to sync with the attacker, check hash chain failure
		if _, err := tester.syncTake("attack", hashes[0]); err != errInvalidChain {
			t.Fatalf("synchronisation error mismatch: have %v, want %v", err, errInvalidChain)
		}
		// Short circuit if the entire chain was banned
		if tester.downloader.banned.Has(hashes[0]) {
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
}
