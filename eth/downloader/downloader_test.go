package downloader

import (
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	knownHash   = common.Hash{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	unknownHash = common.Hash{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9}
)

func createHashes(start, amount int) (hashes []common.Hash) {
	hashes = make([]common.Hash, amount+1)
	hashes[len(hashes)-1] = knownHash

	for i := range hashes[:len(hashes)-1] {
		binary.BigEndian.PutUint64(hashes[i][:8], uint64(i+2))
	}

	return
}

func createBlock(i int, prevHash, hash common.Hash) *types.Block {
	header := &types.Header{Number: big.NewInt(int64(i))}
	block := types.NewBlockWithHeader(header)
	block.HeaderHash = hash
	block.ParentHeaderHash = prevHash
	return block
}

func createBlocksFromHashes(hashes []common.Hash) map[common.Hash]*types.Block {
	blocks := make(map[common.Hash]*types.Block)

	for i, hash := range hashes {
		blocks[hash] = createBlock(len(hashes)-i, knownHash, hash)
	}

	return blocks
}

type downloadTester struct {
	downloader *Downloader

	hashes []common.Hash                // Chain of hashes simulating
	blocks map[common.Hash]*types.Block // Blocks associated with the hashes
	chain  []common.Hash                // Block-chain being constructed

	t            *testing.T
	pcount       int
	done         chan bool
	activePeerId string
}

func newTester(t *testing.T, hashes []common.Hash, blocks map[common.Hash]*types.Block) *downloadTester {
	tester := &downloadTester{
		t: t,

		hashes: hashes,
		blocks: blocks,
		chain:  []common.Hash{knownHash},

		done: make(chan bool),
	}
	downloader := New(tester.hasBlock, tester.getBlock)
	tester.downloader = downloader

	return tester
}

func (dl *downloadTester) sync(peerId string, hash common.Hash) error {
	dl.activePeerId = peerId
	return dl.downloader.Synchronise(peerId, hash)
}

func (dl *downloadTester) insertBlocks(blocks types.Blocks) {
	for _, block := range blocks {
		dl.chain = append(dl.chain, block.Hash())
	}
}

func (dl *downloadTester) hasBlock(hash common.Hash) bool {
	for _, h := range dl.chain {
		if h == hash {
			return true
		}
	}
	return false
}

func (dl *downloadTester) getBlock(hash common.Hash) *types.Block {
	return dl.blocks[knownHash]
}

func (dl *downloadTester) getHashes(hash common.Hash) error {
	dl.downloader.DeliverHashes(dl.activePeerId, dl.hashes)
	return nil
}

func (dl *downloadTester) getBlocks(id string) func([]common.Hash) error {
	return func(hashes []common.Hash) error {
		blocks := make([]*types.Block, len(hashes))
		for i, hash := range hashes {
			blocks[i] = dl.blocks[hash]
		}

		go dl.downloader.DeliverBlocks(id, blocks)

		return nil
	}
}

func (dl *downloadTester) newPeer(id string, td *big.Int, hash common.Hash) {
	dl.pcount++

	dl.downloader.RegisterPeer(id, hash, dl.getHashes, dl.getBlocks(id))
}

func (dl *downloadTester) badBlocksPeer(id string, td *big.Int, hash common.Hash) {
	dl.pcount++

	// This bad peer never returns any blocks
	dl.downloader.RegisterPeer(id, hash, dl.getHashes, func([]common.Hash) error {
		return nil
	})
}

func TestDownload(t *testing.T) {
	minDesiredPeerCount = 4
	blockTtl = 1 * time.Second

	targetBlocks := 1000
	hashes := createHashes(0, targetBlocks)
	blocks := createBlocksFromHashes(hashes)
	tester := newTester(t, hashes, blocks)

	tester.newPeer("peer1", big.NewInt(10000), hashes[0])
	tester.newPeer("peer2", big.NewInt(0), common.Hash{})
	tester.badBlocksPeer("peer3", big.NewInt(0), common.Hash{})
	tester.badBlocksPeer("peer4", big.NewInt(0), common.Hash{})
	tester.activePeerId = "peer1"

	err := tester.sync("peer1", hashes[0])
	if err != nil {
		t.Error("download error", err)
	}

	inqueue := len(tester.downloader.queue.blockCache)
	if inqueue != targetBlocks {
		t.Error("expected", targetBlocks, "have", inqueue)
	}
}

func TestMissing(t *testing.T) {
	targetBlocks := 1000
	hashes := createHashes(0, 1000)
	extraHashes := createHashes(1001, 1003)
	blocks := createBlocksFromHashes(append(extraHashes, hashes...))
	tester := newTester(t, hashes, blocks)

	tester.newPeer("peer1", big.NewInt(10000), hashes[len(hashes)-1])

	hashes = append(extraHashes, hashes[:len(hashes)-1]...)
	tester.newPeer("peer2", big.NewInt(0), common.Hash{})

	err := tester.sync("peer1", hashes[0])
	if err != nil {
		t.Error("download error", err)
	}

	inqueue := len(tester.downloader.queue.blockCache)
	if inqueue != targetBlocks {
		t.Error("expected", targetBlocks, "have", inqueue)
	}
}

func TestTaking(t *testing.T) {
	minDesiredPeerCount = 4
	blockTtl = 1 * time.Second

	targetBlocks := 1000
	hashes := createHashes(0, targetBlocks)
	blocks := createBlocksFromHashes(hashes)
	tester := newTester(t, hashes, blocks)

	tester.newPeer("peer1", big.NewInt(10000), hashes[0])
	tester.newPeer("peer2", big.NewInt(0), common.Hash{})
	tester.badBlocksPeer("peer3", big.NewInt(0), common.Hash{})
	tester.badBlocksPeer("peer4", big.NewInt(0), common.Hash{})

	err := tester.sync("peer1", hashes[0])
	if err != nil {
		t.Error("download error", err)
	}
	bs, err := tester.downloader.TakeBlocks()
	if err != nil {
		t.Fatalf("failed to take blocks: %v", err)
	}
	if len(bs) != targetBlocks {
		t.Error("retrieved block mismatch: have %v, want %v", len(bs), targetBlocks)
	}
}

func TestInactiveDownloader(t *testing.T) {
	targetBlocks := 1000
	hashes := createHashes(0, targetBlocks)
	blocks := createBlocksFromHashSet(createHashSet(hashes))
	tester := newTester(t, hashes, nil)

	err := tester.downloader.DeliverHashes("bad peer 001", hashes)
	if err != errNoSyncActive {
		t.Error("expected no sync error, got", err)
	}

	err = tester.downloader.DeliverBlocks("bad peer 001", blocks)
	if err != errNoSyncActive {
		t.Error("expected no sync error, got", err)
	}
}

func TestCancel(t *testing.T) {
	minDesiredPeerCount = 4
	blockTtl = 1 * time.Second

	targetBlocks := 1000
	hashes := createHashes(0, targetBlocks)
	blocks := createBlocksFromHashes(hashes)
	tester := newTester(t, hashes, blocks)

	tester.newPeer("peer1", big.NewInt(10000), hashes[0])

	err := tester.sync("peer1", hashes[0])
	if err != nil {
		t.Error("download error", err)
	}

	if !tester.downloader.Cancel() {
		t.Error("cancel operation unsuccessfull")
	}

	hashSize, blockSize := tester.downloader.queue.Size()
	if hashSize > 0 || blockSize > 0 {
		t.Error("block (", blockSize, ") or hash (", hashSize, ") not 0")
	}
}

func TestThrottling(t *testing.T) {
	minDesiredPeerCount = 4
	blockTtl = 1 * time.Second

	targetBlocks := 16 * blockCacheLimit
	hashes := createHashes(0, targetBlocks)
	blocks := createBlocksFromHashes(hashes)
	tester := newTester(t, hashes, blocks)

	tester.newPeer("peer1", big.NewInt(10000), hashes[0])
	tester.newPeer("peer2", big.NewInt(0), common.Hash{})
	tester.badBlocksPeer("peer3", big.NewInt(0), common.Hash{})
	tester.badBlocksPeer("peer4", big.NewInt(0), common.Hash{})

	// Concurrently download and take the blocks
	errc := make(chan error, 1)
	go func() {
		errc <- tester.sync("peer1", hashes[0])
	}()

	done := make(chan struct{})
	took := []*types.Block{}
	go func() {
		for running := true; running; {
			select {
			case <-done:
				running = false
			default:
				time.Sleep(time.Millisecond)
			}
			// Take a batch of blocks and accumulate
			blocks, _ := tester.downloader.TakeBlocks()
			took = append(took, blocks...)
		}
		done <- struct{}{}
	}()

	// Synchronise the two threads and verify
	err := <-errc
	done <- struct{}{}
	<-done

	if err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	if len(took) != targetBlocks {
		t.Fatalf("downloaded block mismatch: have %v, want %v", len(took), targetBlocks)
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
	tester := newTester(t, hashes, blocks)
	tester.newPeer("attack", big.NewInt(10000), hashes[0])
	if err := tester.sync("attack", hashes[0]); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	bs, err := tester.downloader.TakeBlocks()
	if err != ErrUnknownParent {
		t.Fatalf("take error mismatch: have %v, want %v", err, ErrUnknownParent)
	}
	if len(bs) != 0 {
		t.Error("retrieved block mismatch: have %v, want %v", len(bs), 0)
	}
	// Cancel the download due to the parent attack
	tester.downloader.Cancel()

	// Reconstruct a valid chain, and try to synchronize with it
	forged.ParentHeaderHash = knownHash
	tester.newPeer("valid", big.NewInt(20000), hashes[0])
	if err := tester.sync("valid", hashes[0]); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	bs, err = tester.downloader.TakeBlocks()
	if err != nil {
		t.Fatalf("failed to retrieve blocks: %v", err)
	}
	if len(bs) != 1 {
		t.Error("retrieved block mismatch: have %v, want %v", len(bs), 1)
	}
}
