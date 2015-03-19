package blockpool

import (
	// "fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/blockpool/test"
)

var statusFields = []string{
	"BlockHashes",
	"BlockHashesInPool",
	"Blocks",
	"BlocksInPool",
	"BlocksInChain",
	"NewBlocks",
	"Forks",
	"LongestChain",
	"Peers",
	"LivePeers",
	"ActivePeers",
	"BestPeers",
	"BadPeers",
}

func getStatusValues(s *Status) []int {
	return []int{
		s.BlockHashes,
		s.BlockHashesInPool,
		s.Blocks,
		s.BlocksInPool,
		s.BlocksInChain,
		s.NewBlocks,
		s.Forks,
		s.LongestChain,
		s.Peers,
		s.LivePeers,
		s.ActivePeers,
		s.BestPeers,
		s.BadPeers,
	}
}

func checkStatus(t *testing.T, bp *BlockPool, syncing bool, expected []int) (err error) {
	s := bp.Status()
	if s.Syncing != syncing {
		t.Errorf("status for Syncing incorrect. expected %v, got %v", syncing, s.Syncing)
	}
	got := getStatusValues(s)
	for i, v := range expected {
		if i == 0 || i == 7 {
			continue //hack
		}
		err = test.CheckInt(statusFields[i], got[i], v, t)
		// fmt.Printf("%v: %v (%v)\n", statusFields[i], got[i], v)
		if err != nil {
			return err
		}
	}
	return
}

func TestBlockPoolStatus(t *testing.T) {
	test.LogInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(12)
	blockPoolTester.refBlockChain[3] = []int{4, 7}
	delete(blockPoolTester.refBlockChain, 6)

	blockPool.Start()
	blockPoolTester.tds = make(map[int]int)
	blockPoolTester.tds[9] = 1
	blockPoolTester.tds[11] = 3
	blockPoolTester.tds[6] = 2

	peer1 := blockPoolTester.newPeer("peer1", 1, 9)
	peer2 := blockPoolTester.newPeer("peer2", 2, 6)
	peer3 := blockPoolTester.newPeer("peer3", 3, 11)
	peer4 := blockPoolTester.newPeer("peer4", 1, 9)
	// peer1 := blockPoolTester.newPeer("peer1", 1, 9)
	// peer2 := blockPoolTester.newPeer("peer2", 2, 6)
	// peer3 := blockPoolTester.newPeer("peer3", 3, 11)
	// peer4 := blockPoolTester.newPeer("peer4", 1, 9)
	peer2.blocksRequestsMap = peer1.blocksRequestsMap

	var expected []int
	var err error
	expected = []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	err = checkStatus(t, blockPool, false, expected)
	if err != nil {
		return
	}

	peer1.AddPeer()
	expected = []int{0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 0, 1, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer1.serveBlocks(8, 9)
	expected = []int{0, 0, 1, 1, 0, 1, 0, 0, 1, 1, 1, 1, 0}
	// err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer1.serveBlockHashes(9, 8, 7, 3, 2)
	expected = []int{6, 5, 1, 1, 0, 1, 0, 0, 1, 1, 1, 1, 0}
	// expected = []int{5, 5, 1, 1, 0, 1, 0, 0, 1, 1, 1, 1, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer1.serveBlocks(3, 7, 8)
	expected = []int{6, 5, 3, 3, 0, 1, 0, 0, 1, 1, 1, 1, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer1.serveBlocks(2, 3)
	expected = []int{6, 5, 4, 4, 0, 1, 0, 0, 1, 1, 1, 1, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer4.AddPeer()
	expected = []int{6, 5, 4, 4, 0, 2, 0, 0, 2, 2, 1, 1, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer4.sendBlockHashes(12, 11)
	expected = []int{6, 5, 4, 4, 0, 2, 0, 0, 2, 2, 1, 1, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer2.AddPeer()
	expected = []int{6, 5, 4, 4, 0, 3, 0, 0, 3, 3, 1, 2, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer2.serveBlocks(5, 6)
	peer2.serveBlockHashes(6, 5, 4, 3, 2)
	expected = []int{10, 8, 5, 5, 0, 3, 1, 0, 3, 3, 2, 2, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer2.serveBlocks(2, 3, 4)
	expected = []int{10, 8, 6, 6, 0, 3, 1, 0, 3, 3, 2, 2, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	blockPool.RemovePeer("peer2")
	expected = []int{10, 8, 6, 6, 0, 3, 1, 0, 3, 2, 2, 2, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer1.serveBlockHashes(2, 1, 0)
	expected = []int{11, 9, 6, 6, 0, 3, 1, 0, 3, 2, 2, 2, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer1.serveBlocks(1, 2)
	expected = []int{11, 9, 7, 7, 0, 3, 1, 0, 3, 2, 2, 2, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer1.serveBlocks(4, 5)
	expected = []int{11, 9, 8, 8, 0, 3, 1, 0, 3, 2, 2, 2, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer3.AddPeer()
	expected = []int{11, 9, 8, 8, 0, 4, 1, 0, 4, 3, 2, 3, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer3.serveBlocks(10, 11)
	expected = []int{12, 9, 9, 9, 0, 4, 1, 0, 4, 3, 3, 3, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer3.serveBlockHashes(11, 10, 9)
	expected = []int{14, 11, 9, 9, 0, 4, 1, 0, 4, 3, 3, 3, 0}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer4.sendBlocks(11, 12)
	expected = []int{14, 11, 9, 9, 0, 4, 1, 0, 4, 3, 4, 3, 1}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}
	peer3.serveBlocks(9, 10)
	expected = []int{14, 11, 10, 10, 0, 4, 1, 0, 4, 3, 4, 3, 1}
	err = checkStatus(t, blockPool, true, expected)
	if err != nil {
		return
	}

	peer3.serveBlocks(0, 1)
	blockPool.Wait(waitTimeout)
	time.Sleep(200 * time.Millisecond)
	expected = []int{14, 3, 11, 3, 8, 4, 1, 8, 4, 3, 4, 3, 1}
	err = checkStatus(t, blockPool, false, expected)
	if err != nil {
		return
	}

	blockPool.Stop()
}
