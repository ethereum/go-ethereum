package blockpool

import (
	"flag"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

var (
	_ = flag.Set("alsologtostderr", "true")
	// _ = flag.Set("log_dir", ".")
	_ = flag.Set("v", "5")
)

// the actual tests
func TestAddPeer(t *testing.T) {
	glog.V(logger.Error).Infoln("logging...")
	hashPool, blockPool, blockPoolTester := newTestBlockPool(t)
	peer0 := blockPoolTester.newPeer("peer0", 2, 2)
	peer1 := blockPoolTester.newPeer("peer1", 4, 4)
	peer2 := blockPoolTester.newPeer("peer2", 6, 6)
	var bestpeer *peer

	blockPool.Start()

	// pool
	best := peer0.AddPeer()
	if !best {
		t.Errorf("peer0 (TD=2) not accepted as best")
		return
	}
	if blockPool.peers.best.id != "peer0" {
		t.Errorf("peer0 (TD=2) not set as best")
		return
	}
	peer0.serveBlocks(1, 2)

	best = peer2.AddPeer()
	if !best {
		t.Errorf("peer2 (TD=6) not accepted as best")
		return
	}
	if blockPool.peers.best.id != "peer2" {
		t.Errorf("peer2 (TD=6) not set as best")
		return
	}
	peer2.serveBlocks(5, 6)

	best = peer1.AddPeer()
	if best {
		t.Errorf("peer1 (TD=4) accepted as best")
		return
	}
	if blockPool.peers.best.id != "peer2" {
		t.Errorf("peer2 (TD=6) not set any more as best")
		return
	}
	if blockPool.peers.best.td.Cmp(big.NewInt(int64(6))) != 0 {
		t.Errorf("peer2 TD=6 not set")
		return
	}

	peer2.td = 8
	peer2.currentBlock = 8
	best = peer2.AddPeer()
	if !best {
		t.Errorf("peer2 (TD=8) not accepted as best")
		return
	}
	if blockPool.peers.best.id != "peer2" {
		t.Errorf("peer2 (TD=8) not set as best")
		return
	}
	if blockPool.peers.best.td.Cmp(big.NewInt(int64(8))) != 0 {
		t.Errorf("peer2 TD = 8 not updated")
		return
	}

	peer1.td = 6
	peer1.currentBlock = 6
	best = peer1.AddPeer()
	if best {
		t.Errorf("peer1 (TD=6) should not be set as best")
		return
	}
	if blockPool.peers.best.id == "peer1" {
		t.Errorf("peer1 (TD=6) should not be set as best")
		return
	}
	bestpeer, best = blockPool.peers.getPeer("peer1")
	if bestpeer.td.Cmp(big.NewInt(int64(6))) != 0 {
		t.Errorf("peer1 TD=6 should be updated")
		return
	}

	blockPool.RemovePeer("peer2")
	bestpeer, best = blockPool.peers.getPeer("peer2")
	if bestpeer != nil {
		t.Errorf("peer2 not removed")
		return
	}

	if blockPool.peers.best.id != "peer1" {
		t.Errorf("existing peer1 (TD=6) should be set as best peer")
		return
	}

	blockPool.RemovePeer("peer1")
	bestpeer, best = blockPool.peers.getPeer("peer1")
	if bestpeer != nil {
		t.Errorf("peer1 not removed")
		return
	}

	if blockPool.peers.best.id != "peer0" {
		t.Errorf("existing peer0 (TD=2) should be set as best peer")
		return
	}

	blockPool.RemovePeer("peer0")
	bestpeer, best = blockPool.peers.getPeer("peer0")
	if bestpeer != nil {
		t.Errorf("peer0 not removed")
		return
	}

	// adding back earlier peer ok
	peer0.currentBlock = 5
	peer0.td = 5
	best = peer0.AddPeer()
	if !best {
		t.Errorf("peer0 (TD=5) should be set as best")
		return
	}

	if blockPool.peers.best.id != "peer0" {
		t.Errorf("peer0 (TD=5) should be set as best")
		return
	}
	peer0.serveBlocks(4, 5)

	hash := hashPool.IndexesToHashes([]int{6})[0]
	newblock := &types.Block{Td: big.NewInt(int64(6)), HeaderHash: hash}
	blockPool.chainEvents.Post(core.ChainHeadEvent{newblock})
	time.Sleep(100 * time.Millisecond)
	if blockPool.peers.best != nil {
		t.Errorf("no peer should be ahead of self")
		return
	}
	best = peer1.AddPeer()
	if blockPool.peers.best != nil {
		t.Errorf("after peer1 (TD=6) still no peer should be ahead of self")
		return
	}

	best = peer2.AddPeer()
	if !best {
		t.Errorf("peer2 (TD=8) not accepted as best")
		return
	}

	blockPool.RemovePeer("peer2")
	if blockPool.peers.best != nil {
		t.Errorf("no peer should be ahead of self")
		return
	}

	blockPool.Stop()
}

func TestPeerPromotionByTdOnBlock(t *testing.T) {
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
