package blockpool

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/blockpool/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/pow"
)

func TestInvalidBlock(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(2)
	blockPoolTester.refBlockChain[2] = []int{}

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 3)
	peer1.AddPeer()
	go peer1.serveBlocks(2, 3)
	go peer1.serveBlockHashes(3, 2, 1, 0)
	peer1.serveBlocks(0, 1, 2)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[2] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
	if len(peer1.peerErrors) == 1 {
		if peer1.peerErrors[0] != ErrInvalidBlock {
			t.Errorf("wrong error, got %v, expected %v", peer1.peerErrors[0], ErrInvalidBlock)
		}
	} else {
		t.Errorf("expected %v error, got %v", ErrInvalidBlock, peer1.peerErrors)
	}
}

func TestVerifyPoW(t *testing.T) {
	t.Skip() // :FIXME:

	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(3)
	first := false
	blockPoolTester.blockPool.verifyPoW = func(b pow.Block) bool {
		bb, _ := b.(*types.Block)
		indexes := blockPoolTester.hashPool.HashesToIndexes([]common.Hash{bb.Hash()})
		if indexes[0] == 2 && !first {
			first = true
			return false
		} else {
			return true
		}

	}

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 3)
	peer2 := blockPoolTester.newPeer("peer2", 1, 3)
	peer1.AddPeer()
	peer2.AddPeer()
	go peer1.serveBlocks(2, 3)
	go peer1.serveBlockHashes(3, 2, 1, 0)
	peer1.serveBlocks(0, 1, 2, 3)
	blockPoolTester.blockPool.verifyPoW = func(b pow.Block) bool {
		return true
	}
	peer2.serveBlocks(1, 2)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[3] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
	if len(peer1.peerErrors) == 1 {
		if peer1.peerErrors[0] != ErrInvalidPoW {
			t.Errorf("wrong error, expected %v, got %v", ErrInvalidPoW, peer1.peerErrors[0])
		}
	} else {
		t.Errorf("expected %v error, got %v", ErrInvalidPoW, peer1.peerErrors)
	}
}

func TestUnrequestedBlock(t *testing.T) {
	t.Skip() // :FIXME:

	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 3)
	peer1.AddPeer()
	peer1.sendBlocks(1, 2)

	blockPool.Stop()
	if len(peer1.peerErrors) == 1 {
		if peer1.peerErrors[0] != ErrUnrequestedBlock {
			t.Errorf("wrong error, got %v, expected %v", peer1.peerErrors[0], ErrUnrequestedBlock)
		}
	} else {
		t.Errorf("expected %v error, got %v", ErrUnrequestedBlock, peer1.peerErrors)
	}
}

func TestErrInsufficientChainInfo(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPool.Config.BlockHashesTimeout = 100 * time.Millisecond
	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 3)
	peer1.AddPeer()

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	if len(peer1.peerErrors) == 1 {
		if peer1.peerErrors[0] != ErrInsufficientChainInfo {
			t.Errorf("wrong error, got %v, expected %v", peer1.peerErrors[0], ErrInsufficientChainInfo)
		}
	} else {
		t.Errorf("expected %v error, got %v", ErrInsufficientChainInfo, peer1.peerErrors)
	}
}

func TestIncorrectTD(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(3)

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 3)
	peer1.AddPeer()
	go peer1.serveBlocks(2, 3)
	go peer1.serveBlockHashes(3, 2, 1, 0)
	peer1.serveBlocks(0, 1, 2)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[3] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
	if len(peer1.peerErrors) == 1 {
		if peer1.peerErrors[0] != ErrIncorrectTD {
			t.Errorf("wrong error, got %v, expected %v", peer1.peerErrors[0], ErrIncorrectTD)
		}
	} else {
		t.Errorf("expected %v error, got %v", ErrIncorrectTD, peer1.peerErrors)
	}
}

func TestSkipIncorrectTDonFutureBlocks(t *testing.T) {
	// t.Skip() // @zelig this one requires fixing for the TD

	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(3)

	blockPool.insertChain = func(blocks types.Blocks) error {
		err := blockPoolTester.insertChain(blocks)
		if err == nil {
			for _, block := range blocks {
				if block.Td.Cmp(common.Big3) == 0 {
					block.Td = common.Big3
					block.SetQueued(true)
					break
				}
			}
		}
		return err
	}

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 3, 3)
	peer1.AddPeer()
	go peer1.serveBlocks(2, 3)
	go peer1.serveBlockHashes(3, 2, 1, 0)
	peer1.serveBlocks(0, 1, 2)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[3] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
	if len(peer1.peerErrors) > 0 {
		t.Errorf("expected no error, got %v (1 of %v)", peer1.peerErrors[0], len(peer1.peerErrors))
	}
}

func TestPeerSuspension(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPool.Config.PeerSuspensionInterval = 100 * time.Millisecond

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 3)
	peer1.AddPeer()
	blockPool.peers.peerError("peer1", 0, "")
	bestpeer, _ := blockPool.peers.getPeer("peer1")
	if bestpeer != nil {
		t.Errorf("peer1 not removed on error")
	}
	peer1.AddPeer()
	bestpeer, _ = blockPool.peers.getPeer("peer1")
	if bestpeer != nil {
		t.Errorf("peer1 not removed on reconnect")
	}
	time.Sleep(100 * time.Millisecond)
	peer1.AddPeer()
	bestpeer, _ = blockPool.peers.getPeer("peer1")
	if bestpeer == nil {
		t.Errorf("peer1 not connected after PeerSuspensionInterval")
	}
	// blockPool.Wait(waitTimeout)
	blockPool.Stop()

}
