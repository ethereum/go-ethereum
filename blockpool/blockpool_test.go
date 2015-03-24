package blockpool

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/blockpool/test"
)

// using the mock framework in blockpool_util_test
// we test various scenarios here

func TestPeerWithKnownBlock(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.refBlockChain[0] = nil
	blockPoolTester.blockChain[0] = nil
	blockPool.Start()

	peer0 := blockPoolTester.newPeer("0", 1, 0)
	peer0.AddPeer()

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	// no request on known block
	peer0.checkBlockHashesRequests()
}

func TestPeerWithKnownParentBlock(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.initRefBlockChain(1)
	blockPoolTester.blockChain[0] = nil
	blockPool.Start()

	peer0 := blockPoolTester.newPeer("0", 1, 1)
	peer0.AddPeer()
	peer0.serveBlocks(0, 1)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	peer0.checkBlocksRequests([]int{1})
	peer0.checkBlockHashesRequests()
	blockPoolTester.refBlockChain[1] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestSimpleChain(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(2)

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 2, 2)
	peer1.AddPeer()
	peer1.serveBlocks(1, 2)
	go peer1.serveBlockHashes(2, 1, 0)
	peer1.serveBlocks(0, 1)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[2] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestChainConnectingWithParentHash(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(3)

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 3, 3)
	peer1.AddPeer()
	go peer1.serveBlocks(2, 3)
	go peer1.serveBlockHashes(3, 2, 1)
	peer1.serveBlocks(0, 1, 2)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[3] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestMultiSectionChain(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(5)

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 5, 5)

	peer1.AddPeer()
	go peer1.serveBlocks(4, 5)
	go peer1.serveBlockHashes(5, 4, 3)
	go peer1.serveBlocks(2, 3, 4)
	go peer1.serveBlockHashes(3, 2, 1, 0)
	peer1.serveBlocks(0, 1, 2)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[5] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestNewBlocksOnPartialChain(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(7)
	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 5, 5)
	blockPoolTester.tds = make(map[int]int)
	blockPoolTester.tds[5] = 5

	peer1.AddPeer()
	go peer1.serveBlocks(4, 5) // partially complete section
	go peer1.serveBlockHashes(5, 4, 3)
	peer1.serveBlocks(3, 4) // partially complete section

	// peer1 found new blocks
	peer1.td = 7
	peer1.currentBlock = 7
	peer1.AddPeer()
	peer1.sendBlocks(6, 7)
	go peer1.serveBlockHashes(7, 6, 5)
	go peer1.serveBlocks(2, 3)
	go peer1.serveBlocks(5, 6)
	go peer1.serveBlockHashes(3, 2, 1) // tests that hash request from known chain root is remembered
	peer1.serveBlocks(0, 1, 2)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[7] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestPeerSwitchUp(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(7)

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 6, 6)
	peer2 := blockPoolTester.newPeer("peer2", 7, 7)

	peer1.AddPeer()
	go peer1.serveBlocks(5, 6)
	go peer1.serveBlockHashes(6, 5, 4, 3) //
	peer1.serveBlocks(2, 3)               // section partially complete, block 3 will be preserved after peer demoted
	peer2.AddPeer()                       // peer2 is promoted as best peer, peer1 is demoted
	go peer2.serveBlocks(6, 7)            //
	go peer2.serveBlocks(4, 5)            // tests that block request for earlier section is remembered
	go peer1.serveBlocks(3, 4)            // tests that connecting section by demoted peer is remembered and blocks are accepted from demoted peer
	go peer2.serveBlockHashes(3, 2, 1, 0) // tests that known chain section is activated, hash requests from 3 is remembered
	peer2.serveBlocks(0, 1, 2)            // final blocks linking to blockchain sent

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[7] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestPeerSwitchDownOverlapSectionWithoutRootBlock(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(6)
	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 4, 4)
	peer2 := blockPoolTester.newPeer("peer2", 6, 6)

	peer2.AddPeer()
	peer2.serveBlocks(5, 6)                  // partially complete, section will be preserved
	peer2.serveBlockHashes(6, 5, 4)          // no go: make sure skeleton is created
	peer1.AddPeer()                          // inferior peer1 is promoted as best peer
	blockPool.RemovePeer("peer2")            // peer2 disconnects
	go peer1.serveBlockHashes(4, 3, 2, 1, 0) //
	go peer1.serveBlocks(3, 4)               //
	go peer1.serveBlocks(4, 5)               // tests that section set by demoted peer is remembered and blocks are accepted from new peer if they have it even if peers original TD is lower
	peer1.serveBlocks(0, 1, 2, 3)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[6] = []int{} // tests that idle sections are not inserted in blockchain
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestPeerSwitchDownOverlapSectionWithRootBlock(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(6)
	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 4, 4)
	peer2 := blockPoolTester.newPeer("peer2", 6, 6)

	peer2.AddPeer()
	peer2.serveBlocks(5, 6)                  // partially complete, section will be preserved
	go peer2.serveBlockHashes(6, 5, 4)       //
	peer2.serveBlocks(3, 4)                  // !incomplete section
	time.Sleep(100 * time.Millisecond)       // make sure block 4 added
	peer1.AddPeer()                          // inferior peer1 is promoted as best peer
	blockPool.RemovePeer("peer2")            // peer2 disconnects
	go peer1.serveBlockHashes(4, 3, 2, 1, 0) // tests that hash request are directly connecting if the head block exists
	go peer1.serveBlocks(4, 5)               // tests that section set by demoted peer is remembered and blocks are accepted from new peer if they have it even if peers original TD is lower
	peer1.serveBlocks(0, 1, 2, 3)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[6] = []int{} // tests that idle sections are not inserted in blockchain
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestPeerSwitchDownDisjointSection(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(3)
	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 3, 3)
	peer2 := blockPoolTester.newPeer("peer2", 6, 6)

	peer2.AddPeer()
	peer2.serveBlocks(5, 6)            // partially complete, section will be preserved
	go peer2.serveBlockHashes(6, 5, 4) //
	peer2.serveBlocks(3, 4, 5)         //
	time.Sleep(100 * time.Millisecond) // make sure blocks are received
	peer1.AddPeer()                    // inferior peer1 is promoted as best peer
	blockPool.RemovePeer("peer2")      // peer2 disconnects
	go peer1.serveBlocks(2, 3)         //
	go peer1.serveBlockHashes(3, 2, 1) //
	peer1.serveBlocks(0, 1, 2)         //

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[3] = []int{} // tests that idle sections are not inserted in blockchain
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestPeerSwitchBack(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(8)

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 11, 11)
	peer2 := blockPoolTester.newPeer("peer2", 8, 8)

	peer2.AddPeer()
	go peer2.serveBlocks(7, 8)
	go peer2.serveBlockHashes(8, 7, 6)
	go peer2.serveBlockHashes(6, 5, 4)
	peer2.serveBlocks(4, 5)                  // section partially complete
	peer1.AddPeer()                          // peer1 is promoted as best peer
	go peer1.serveBlocks(10, 11)             //
	peer1.serveBlockHashes(11, 10)           // only gives useless results
	blockPool.RemovePeer("peer1")            // peer1 disconnects
	go peer2.serveBlockHashes(4, 3, 2, 1, 0) // tests that asking for hashes from 4 is remembered
	go peer2.serveBlocks(3, 4, 5, 6, 7, 8)   // tests that section 4, 5, 6 and 7, 8 are remembered for missing blocks
	peer2.serveBlocks(0, 1, 2, 3)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[8] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestForkSimple(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(9)
	blockPoolTester.refBlockChain[3] = []int{4, 7}
	delete(blockPoolTester.refBlockChain, 6)

	blockPool.Start()
	blockPoolTester.tds = make(map[int]int)
	blockPoolTester.tds[6] = 10
	peer1 := blockPoolTester.newPeer("peer1", 9, 9)
	peer2 := blockPoolTester.newPeer("peer2", 10, 6)

	peer1.AddPeer()
	go peer1.serveBlocks(8, 9)
	go peer1.serveBlockHashes(9, 8, 7, 3, 2)
	peer1.serveBlocks(1, 2, 3, 7, 8)
	peer2.AddPeer()                          // peer2 is promoted as best peer
	go peer2.serveBlocks(5, 6)               //
	go peer2.serveBlockHashes(6, 5, 4, 3, 2) // fork on 3 -> 4 (earlier child: 7)
	go peer2.serveBlocks(1, 2, 3, 4, 5)
	go peer2.serveBlockHashes(2, 1, 0)
	peer2.serveBlocks(0, 1, 2)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[6] = []int{}
	blockPoolTester.refBlockChain[3] = []int{4}
	delete(blockPoolTester.refBlockChain, 7)
	delete(blockPoolTester.refBlockChain, 8)
	delete(blockPoolTester.refBlockChain, 9)
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)

}

func TestForkSwitchBackByNewBlocks(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(11)
	blockPoolTester.refBlockChain[3] = []int{4, 7}
	delete(blockPoolTester.refBlockChain, 6)

	blockPool.Start()
	blockPoolTester.tds = make(map[int]int)
	blockPoolTester.tds[6] = 10
	peer1 := blockPoolTester.newPeer("peer1", 9, 9)
	peer2 := blockPoolTester.newPeer("peer2", 10, 6)

	peer1.AddPeer()
	go peer1.serveBlocks(8, 9)               //
	go peer1.serveBlockHashes(9, 8, 7, 3, 2) //
	peer1.serveBlocks(7, 8)                  // partial section
	// time.Sleep(1 * time.Second)
	peer2.AddPeer()                          //
	go peer2.serveBlocks(5, 6)               //
	go peer2.serveBlockHashes(6, 5, 4, 3, 2) // peer2 forks on block 3
	peer2.serveBlocks(1, 2, 3, 4, 5)         //

	// peer1 finds new blocks
	peer1.td = 11
	peer1.currentBlock = 11
	peer1.AddPeer()
	go peer1.serveBlocks(10, 11)
	go peer1.serveBlockHashes(11, 10, 9)
	go peer1.serveBlocks(9, 10)
	// time.Sleep(1 * time.Second)
	go peer1.serveBlocks(3, 7)      // tests that block requests on earlier fork are remembered
	go peer1.serveBlockHashes(2, 1) // tests that hash request from root of connecting chain section (added by demoted peer) is remembered
	peer1.serveBlocks(0, 1)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[11] = []int{}
	blockPoolTester.refBlockChain[3] = []int{7}
	delete(blockPoolTester.refBlockChain, 6)
	delete(blockPoolTester.refBlockChain, 5)
	delete(blockPoolTester.refBlockChain, 4)
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)

}

func TestForkSwitchBackByPeerSwitchBack(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(9)
	blockPoolTester.refBlockChain[3] = []int{4, 7}
	delete(blockPoolTester.refBlockChain, 6)

	blockPool.Start()

	blockPoolTester.tds = make(map[int]int)
	blockPoolTester.tds[6] = 10

	blockPoolTester.tds = make(map[int]int)
	blockPoolTester.tds[6] = 10

	peer1 := blockPoolTester.newPeer("peer1", 9, 9)
	peer2 := blockPoolTester.newPeer("peer2", 10, 6)

	peer1.AddPeer()
	go peer1.serveBlocks(8, 9)
	go peer1.serveBlockHashes(9, 8, 7, 3, 2)
	peer1.serveBlocks(7, 8)
	peer2.AddPeer()
	go peer2.serveBlocks(5, 6)               //
	go peer2.serveBlockHashes(6, 5, 4, 3, 2) // peer2 forks on block 3
	peer2.serveBlocks(2, 3, 4, 5)            //
	blockPool.RemovePeer("peer2")            // peer2 disconnects, peer1 is promoted again as best peer
	go peer1.serveBlocks(1, 2)               //
	go peer1.serveBlockHashes(2, 1, 0)       //
	go peer1.serveBlocks(3, 7)               // tests that block requests on earlier fork are remembered and orphan section relinks to existing parent block
	peer1.serveBlocks(0, 1)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[9] = []int{}
	blockPoolTester.refBlockChain[3] = []int{7}
	delete(blockPoolTester.refBlockChain, 6)
	delete(blockPoolTester.refBlockChain, 5)
	delete(blockPoolTester.refBlockChain, 4)
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)

}

func TestForkCompleteSectionSwitchBackByPeerSwitchBack(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(9)
	blockPoolTester.refBlockChain[3] = []int{4, 7}
	delete(blockPoolTester.refBlockChain, 6)

	blockPool.Start()

	blockPoolTester.tds = make(map[int]int)
	blockPoolTester.tds[6] = 10

	peer1 := blockPoolTester.newPeer("peer1", 9, 9)
	peer2 := blockPoolTester.newPeer("peer2", 10, 6)

	peer1.AddPeer()
	go peer1.serveBlocks(8, 9)
	go peer1.serveBlockHashes(9, 8, 7)
	peer1.serveBlocks(3, 7, 8)               // make sure this section is complete
	time.Sleep(1 * time.Second)              //
	go peer1.serveBlockHashes(7, 3, 2)       // block 3/7 is section boundary
	peer1.serveBlocks(2, 3)                  // partially complete sections block 2 missing
	peer2.AddPeer()                          //
	go peer2.serveBlocks(5, 6)               //
	go peer2.serveBlockHashes(6, 5, 4, 3, 2) // peer2 forks on block 3
	peer2.serveBlocks(2, 3, 4, 5)            // block 2 still missing.
	blockPool.RemovePeer("peer2")            // peer2 disconnects, peer1 is promoted again as best peer
	go peer1.serveBlockHashes(2, 1, 0)       //
	peer1.serveBlocks(0, 1, 2)

	blockPool.Wait(waitTimeout)
	blockPool.Stop()
	blockPoolTester.refBlockChain[9] = []int{}
	blockPoolTester.refBlockChain[3] = []int{7}
	delete(blockPoolTester.refBlockChain, 6)
	delete(blockPoolTester.refBlockChain, 5)
	delete(blockPoolTester.refBlockChain, 4)
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)

}
