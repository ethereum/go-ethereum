package downloader

import (
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

var knownHash = common.Hash{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

func createHashes(start, amount int) (hashes []common.Hash) {
	hashes = make([]common.Hash, amount+1)
	hashes[len(hashes)-1] = knownHash

	for i := range hashes[:len(hashes)-1] {
		binary.BigEndian.PutUint64(hashes[i][:8], uint64(i+2))
	}

	return
}

func createBlocksFromHashes(hashes []common.Hash) map[common.Hash]*types.Block {
	blocks := make(map[common.Hash]*types.Block)
	for i, hash := range hashes {
		header := &types.Header{Number: big.NewInt(int64(i))}
		blocks[hash] = types.NewBlockWithHeader(header)
		blocks[hash].HeaderHash = hash
	}

	return blocks
}

type downloadTester struct {
	downloader *Downloader
	hashes     []common.Hash
	blocks     map[common.Hash]*types.Block
	t          *testing.T
	pcount     int
	done       chan bool

	insertedBlocks int
}

func newTester(t *testing.T, hashes []common.Hash, blocks map[common.Hash]*types.Block) *downloadTester {
	tester := &downloadTester{t: t, hashes: hashes, blocks: blocks, done: make(chan bool)}
	downloader := New(tester.hasBlock, tester.insertChain, func() *big.Int { return new(big.Int) })
	tester.downloader = downloader

	return tester
}

func (dl *downloadTester) hasBlock(hash common.Hash) bool {
	if knownHash == hash {
		return true
	}
	return false
}

func (dl *downloadTester) insertChain(blocks types.Blocks) error {
	dl.insertedBlocks += len(blocks)

	if len(dl.blocks)-1 <= dl.insertedBlocks {
		dl.done <- true
	}

	return nil
}

func (dl *downloadTester) getHashes(hash common.Hash) error {
	dl.downloader.hashCh <- dl.hashes
	return nil
}

func (dl *downloadTester) getBlocks(id string) func([]common.Hash) error {
	return func(hashes []common.Hash) error {
		blocks := make([]*types.Block, len(hashes))
		for i, hash := range hashes {
			blocks[i] = dl.blocks[hash]
		}

		go dl.downloader.DeliverChunk(id, blocks)

		return nil
	}
}

func (dl *downloadTester) newPeer(id string, td *big.Int, hash common.Hash) {
	dl.pcount++

	dl.downloader.RegisterPeer(id, td, hash, dl.getHashes, dl.getBlocks(id))
}

func (dl *downloadTester) badBlocksPeer(id string, td *big.Int, hash common.Hash) {
	dl.pcount++

	// This bad peer never returns any blocks
	dl.downloader.RegisterPeer(id, td, hash, dl.getHashes, func([]common.Hash) error {
		return nil
	})
}

func TestDownload(t *testing.T) {
	glog.SetV(logger.Detail)
	glog.SetToStderr(true)

	minDesiredPeerCount = 4

	hashes := createHashes(0, 1000)
	blocks := createBlocksFromHashes(hashes)
	tester := newTester(t, hashes, blocks)

	tester.newPeer("peer1", big.NewInt(10000), hashes[0])
	tester.newPeer("peer2", big.NewInt(0), common.Hash{})
	tester.badBlocksPeer("peer3", big.NewInt(0), common.Hash{})
	tester.badBlocksPeer("peer4", big.NewInt(0), common.Hash{})

success:
	select {
	case <-tester.done:
		break success
	case <-time.After(10 * time.Second): // XXX this could actually fail on a slow computer
		t.Error("timeout")
	}
}

func TestMissing(t *testing.T) {
	t.Skip()

	glog.SetV(logger.Detail)
	glog.SetToStderr(true)

	hashes := createHashes(0, 1000)
	extraHashes := createHashes(1001, 1003)
	blocks := createBlocksFromHashes(append(extraHashes, hashes...))
	tester := newTester(t, hashes, blocks)

	tester.newPeer("peer1", big.NewInt(10000), hashes[len(hashes)-1])

	hashes = append(extraHashes, hashes[:len(hashes)-1]...)
	tester.newPeer("peer2", big.NewInt(0), common.Hash{})

success1:
	select {
	case <-tester.done:
		break success1
	case <-time.After(10 * time.Second): // XXX this could actually fail on a slow computer
		t.Error("timout")
	}

	tester.downloader.AddBlock("peer2", blocks[hashes[len(hashes)-1]], big.NewInt(10001))
}
