package blockpool

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/blockpool/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

// the actual tests
func TestAddPeer(t *testing.T) {
	test.LogInit()
	hashPool, blockPool, blockPoolTester := newTestBlockPool(t)
	peer0 := blockPoolTester.newPeer("peer0", 1, 1)
	peer1 := blockPoolTester.newPeer("peer1", 2, 2)
	peer2 := blockPoolTester.newPeer("peer2", 3, 3)
	var bestpeer *peer

	blockPool.Start()

	// pool
	best := peer0.AddPeer()
	if !best {
		t.Errorf("peer0 (TD=1) not accepted as best")
	}
	if blockPool.peers.best.id != "peer0" {
		t.Errorf("peer0 (TD=1) not set as best")
	}

	best = peer2.AddPeer()
	if !best {
		t.Errorf("peer2 (TD=3) not accepted as best")
	}
	if blockPool.peers.best.id != "peer2" {
		t.Errorf("peer2 (TD=3) not set as best")
	}
	peer2.waitBlocksRequests(3)

	best = peer1.AddPeer()
	if best {
		t.Errorf("peer1 (TD=2) accepted as best")
	}
	if blockPool.peers.best.id != "peer2" {
		t.Errorf("peer2 (TD=3) not set any more as best")
	}
	if blockPool.peers.best.td.Cmp(big.NewInt(int64(3))) != 0 {
		t.Errorf("peer1 TD not set")
	}

	peer2.td = 4
	peer2.currentBlock = 4
	best = peer2.AddPeer()
	if !best {
		t.Errorf("peer2 (TD=4) not accepted as best")
	}
	if blockPool.peers.best.id != "peer2" {
		t.Errorf("peer2 (TD=4) not set as best")
	}
	if blockPool.peers.best.td.Cmp(big.NewInt(int64(4))) != 0 {
		t.Errorf("peer2 TD not updated")
	}
	peer2.waitBlocksRequests(4)

	peer1.td = 3
	peer1.currentBlock = 3
	best = peer1.AddPeer()
	if best {
		t.Errorf("peer1 (TD=3) should not be set as best")
	}
	if blockPool.peers.best.id == "peer1" {
		t.Errorf("peer1 (TD=3) should not be set as best")
	}
	bestpeer, best = blockPool.peers.getPeer("peer1")
	if bestpeer.td.Cmp(big.NewInt(int64(3))) != 0 {
		t.Errorf("peer1 TD should be updated")
	}

	blockPool.RemovePeer("peer2")
	bestpeer, best = blockPool.peers.getPeer("peer2")
	if bestpeer != nil {
		t.Errorf("peer2 not removed")
	}

	if blockPool.peers.best.id != "peer1" {
		t.Errorf("existing peer1 (TD=3) should be set as best peer")
	}
	peer1.waitBlocksRequests(3)

	blockPool.RemovePeer("peer1")
	bestpeer, best = blockPool.peers.getPeer("peer1")
	if bestpeer != nil {
		t.Errorf("peer1 not removed")
	}

	if blockPool.peers.best.id != "peer0" {
		t.Errorf("existing peer0 (TD=1) should be set as best peer")
	}
	peer0.waitBlocksRequests(1)

	blockPool.RemovePeer("peer0")
	bestpeer, best = blockPool.peers.getPeer("peer0")
	if bestpeer != nil {
		t.Errorf("peer1 not removed")
	}

	// adding back earlier peer ok
	peer0.currentBlock = 3
	best = peer0.AddPeer()
	if !best {
		t.Errorf("peer0 (TD=1) should be set as best")
	}

	if blockPool.peers.best.id != "peer0" {
		t.Errorf("peer0 (TD=1) should be set as best")
	}
	peer0.waitBlocksRequests(3)

	hash := hashPool.IndexesToHashes([]int{0})[0]
	newblock := &types.Block{Td: common.Big3, HeaderHash: hash}
	blockPool.chainEvents.Post(core.ChainHeadEvent{newblock})
	time.Sleep(100 * time.Millisecond)
	if blockPool.peers.best != nil {
		t.Errorf("no peer should be ahead of self")
	}
	best = peer1.AddPeer()
	if blockPool.peers.best != nil {
		t.Errorf("still no peer should be ahead of self")
	}

	best = peer2.AddPeer()
	if !best {
		t.Errorf("peer2 (TD=4) not accepted as best")
	}

	blockPool.RemovePeer("peer2")
	if blockPool.peers.best != nil {
		t.Errorf("no peer should be ahead of self")
	}

	blockPool.Stop()
}

func TestPeerPromotionByTdOnBlock(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(4)
	peer0 := blockPoolTester.newPeer("peer0", 2, 2)
	peer1 := blockPoolTester.newPeer("peer1", 1, 1)
	peer2 := blockPoolTester.newPeer("peer2", 4, 4)

	blockPool.Start()

	peer0.AddPeer()
	peer0.serveBlocks(1, 2)
	best := peer1.AddPeer()
	// this tests that peer1 is not promoted over peer0 yet
	if best {
		t.Errorf("peer1 (TD=1) should not be set as best")
		return
	}
	best = peer2.AddPeer()
	peer2.serveBlocks(3, 4)
	peer2.serveBlockHashes(4, 3, 2, 1)
	peer1.sendBlocks(3, 4)

	blockPool.RemovePeer("peer2")
	if blockPool.peers.best.id != "peer1" {
		t.Errorf("peer1 (TD=3) should be set as best")
		return
	}
	peer1.serveBlocks(0, 1, 2, 3)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[4] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}
