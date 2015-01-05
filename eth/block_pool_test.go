package eth

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	ethlogger "github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/pow"
)

const waitTimeout = 60 // seconds

var logsys = ethlogger.NewStdLogSystem(os.Stdout, log.LstdFlags, ethlogger.LogLevel(ethlogger.DebugLevel))

var ini = false

func logInit() {
	if !ini {
		ethlogger.AddLogSystem(logsys)
		ini = true
	}
}

// test helpers
func arrayEq(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type intToHash map[int][]byte

type hashToInt map[string]int

// hashPool is a test helper, that allows random hashes to be referred to by integers
type testHashPool struct {
	intToHash
	hashToInt
	lock sync.Mutex
}

func newHash(i int) []byte {
	return crypto.Sha3([]byte(string(i)))
}

func (self *testHashPool) indexesToHashes(indexes []int) (hashes [][]byte) {
	self.lock.Lock()
	defer self.lock.Unlock()
	for _, i := range indexes {
		hash, found := self.intToHash[i]
		if !found {
			hash = newHash(i)
			self.intToHash[i] = hash
			self.hashToInt[string(hash)] = i
		}
		hashes = append(hashes, hash)
	}
	return
}

func (self *testHashPool) hashesToIndexes(hashes [][]byte) (indexes []int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	for _, hash := range hashes {
		i, found := self.hashToInt[string(hash)]
		if !found {
			i = -1
		}
		indexes = append(indexes, i)
	}
	return
}

// test blockChain is an integer trie
type blockChain map[int][]int

// blockPoolTester provides the interface between tests and a blockPool
//
// refBlockChain is used to guide which blocks will be accepted as valid
// blockChain gives the current state of the blockchain and
// accumulates inserts so that we can check the resulting chain
type blockPoolTester struct {
	hashPool      *testHashPool
	lock          sync.RWMutex
	refBlockChain blockChain
	blockChain    blockChain
	blockPool     *BlockPool
	t             *testing.T
}

func newTestBlockPool(t *testing.T) (hashPool *testHashPool, blockPool *BlockPool, b *blockPoolTester) {
	hashPool = &testHashPool{intToHash: make(intToHash), hashToInt: make(hashToInt)}
	b = &blockPoolTester{
		t:             t,
		hashPool:      hashPool,
		blockChain:    make(blockChain),
		refBlockChain: make(blockChain),
	}
	b.blockPool = NewBlockPool(b.hasBlock, b.insertChain, b.verifyPoW)
	blockPool = b.blockPool
	return
}

func (self *blockPoolTester) Errorf(format string, params ...interface{}) {
	fmt.Printf(format+"\n", params...)
	self.t.Errorf(format, params...)
}

// blockPoolTester implements the 3 callbacks needed by the blockPool:
// hasBlock, insetChain, verifyPoW
func (self *blockPoolTester) hasBlock(block []byte) (ok bool) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	indexes := self.hashPool.hashesToIndexes([][]byte{block})
	i := indexes[0]
	_, ok = self.blockChain[i]
	fmt.Printf("has block %v (%x...): %v\n", i, block[0:4], ok)
	return
}

func (self *blockPoolTester) insertChain(blocks types.Blocks) error {
	self.lock.RLock()
	defer self.lock.RUnlock()
	var parent, child int
	var children, refChildren []int
	var ok bool
	for _, block := range blocks {
		child = self.hashPool.hashesToIndexes([][]byte{block.Hash()})[0]
		_, ok = self.blockChain[child]
		if ok {
			fmt.Printf("block %v already in blockchain\n", child)
			continue // already in chain
		}
		parent = self.hashPool.hashesToIndexes([][]byte{block.ParentHeaderHash})[0]
		children, ok = self.blockChain[parent]
		if !ok {
			return fmt.Errorf("parent %v not in blockchain ", parent)
		}
		ok = false
		var found bool
		refChildren, found = self.refBlockChain[parent]
		if found {
			for _, c := range refChildren {
				if c == child {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("invalid block %v", child)
			}
		} else {
			ok = true
		}
		if ok {
			// accept any blocks if parent not in refBlockChain
			fmt.Errorf("blockchain insert %v -> %v\n", parent, child)
			self.blockChain[parent] = append(children, child)
			self.blockChain[child] = nil
		}
	}
	return nil
}

func (self *blockPoolTester) verifyPoW(pblock pow.Block) bool {
	return true
}

// test helper that compares the resulting blockChain to the desired blockChain
func (self *blockPoolTester) checkBlockChain(blockChain map[int][]int) {
	for k, v := range self.blockChain {
		fmt.Printf("got: %v -> %v\n", k, v)
	}
	for k, v := range blockChain {
		fmt.Printf("expected: %v -> %v\n", k, v)
	}
	if len(blockChain) != len(self.blockChain) {
		self.Errorf("blockchain incorrect (zlength differ)")
	}
	for k, v := range blockChain {
		vv, ok := self.blockChain[k]
		if !ok || !arrayEq(v, vv) {
			self.Errorf("blockchain incorrect on %v -> %v (!= %v)", k, vv, v)
		}
	}
}

//

// peerTester provides the peer callbacks for the blockPool
// it registers actual callbacks so that result can be compared to desired behaviour
// provides helper functions to mock the protocol calls to the blockPool
type peerTester struct {
	blockHashesRequests []int
	blocksRequests      [][]int
	blocksRequestsMap   map[int]bool
	peerErrors          []int
	blockPool           *BlockPool
	hashPool            *testHashPool
	lock                sync.RWMutex
	id                  string
	td                  int
	currentBlock        int
	t                   *testing.T
}

// peerTester constructor takes hashPool and blockPool from the blockPoolTester
func (self *blockPoolTester) newPeer(id string, td int, cb int) *peerTester {
	return &peerTester{
		id:                id,
		td:                td,
		currentBlock:      cb,
		hashPool:          self.hashPool,
		blockPool:         self.blockPool,
		t:                 self.t,
		blocksRequestsMap: make(map[int]bool),
	}
}

func (self *peerTester) Errorf(format string, params ...interface{}) {
	fmt.Printf(format+"\n", params...)
	self.t.Errorf(format, params...)
}

// helper to compare actual and expected block requests
func (self *peerTester) checkBlocksRequests(blocksRequests ...[]int) {
	if len(blocksRequests) > len(self.blocksRequests) {
		self.Errorf("blocks requests incorrect (length differ)\ngot %v\nexpected %v", self.blocksRequests, blocksRequests)
	} else {
		for i, rr := range blocksRequests {
			r := self.blocksRequests[i]
			if !arrayEq(r, rr) {
				self.Errorf("blocks requests incorrect\ngot %v\nexpected %v", self.blocksRequests, blocksRequests)
			}
		}
	}
}

// helper to compare actual and expected block hash requests
func (self *peerTester) checkBlockHashesRequests(blocksHashesRequests ...int) {
	rr := blocksHashesRequests
	self.lock.RLock()
	r := self.blockHashesRequests
	self.lock.RUnlock()
	if len(r) != len(rr) {
		self.Errorf("block hashes requests incorrect (length differ)\ngot %v\nexpected %v", r, rr)
	} else {
		if !arrayEq(r, rr) {
			self.Errorf("block hashes requests incorrect\ngot %v\nexpected %v", r, rr)
		}
	}
}

// waiter function used by peer.AddBlocks
// blocking until requests appear
// since block requests are sent to any random peers
// block request map is shared between peers
// times out after a period
func (self *peerTester) waitBlocksRequests(blocksRequest ...int) {
	timeout := time.After(waitTimeout * time.Second)
	rr := blocksRequest
	for {
		self.lock.RLock()
		r := self.blocksRequestsMap
		fmt.Printf("[%s] blocks request check %v (%v)\n", self.id, rr, r)
		i := 0
		for i = 0; i < len(rr); i++ {
			_, ok := r[rr[i]]
			if !ok {
				break
			}
		}
		self.lock.RUnlock()

		if i == len(rr) {
			return
		}
		time.Sleep(100 * time.Millisecond)
		select {
		case <-timeout:
		default:
		}
	}
}

// waiter function used by peer.AddBlockHashes
// blocking until requests appear
// times out after a period
func (self *peerTester) waitBlockHashesRequests(blocksHashesRequest int) {
	timeout := time.After(waitTimeout * time.Second)
	rr := blocksHashesRequest
	for i := 0; ; {
		self.lock.RLock()
		r := self.blockHashesRequests
		self.lock.RUnlock()
		fmt.Printf("[%s] block hash request check %v (%v)\n", self.id, rr, r)
		for ; i < len(r); i++ {
			if rr == r[i] {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
		select {
		case <-timeout:
		default:
		}
	}
}

// mocks a simple blockchain 0 (genesis) ... n (head)
func (self *blockPoolTester) initRefBlockChain(n int) {
	for i := 0; i < n; i++ {
		self.refBlockChain[i] = []int{i + 1}
	}
}

// peerTester functions that mimic protocol calls to the blockpool
//  registers the peer with the blockPool
func (self *peerTester) AddPeer() bool {
	hash := self.hashPool.indexesToHashes([]int{self.currentBlock})[0]
	return self.blockPool.AddPeer(big.NewInt(int64(self.td)), hash, self.id, self.requestBlockHashes, self.requestBlocks, self.peerError)
}

// peer sends blockhashes if and when gets a request
func (self *peerTester) AddBlockHashes(indexes ...int) {
	i := 0
	fmt.Printf("ready to add block hashes %v\n", indexes)

	self.waitBlockHashesRequests(indexes[0])
	fmt.Printf("adding block hashes %v\n", indexes)
	hashes := self.hashPool.indexesToHashes(indexes)
	next := func() (hash []byte, ok bool) {
		if i < len(hashes) {
			hash = hashes[i]
			ok = true
			i++
		}
		return
	}
	self.blockPool.AddBlockHashes(next, self.id)
}

// peer sends blocks if and when there is a request
// (in the shared request store, not necessarily to a person)
func (self *peerTester) AddBlocks(indexes ...int) {
	hashes := self.hashPool.indexesToHashes(indexes)
	fmt.Printf("ready to add blocks %v\n", indexes[1:])
	self.waitBlocksRequests(indexes[1:]...)
	fmt.Printf("adding blocks %v \n", indexes[1:])
	for i := 1; i < len(hashes); i++ {
		fmt.Printf("adding block %v %x\n", indexes[i], hashes[i][:4])
		self.blockPool.AddBlock(&types.Block{HeaderHash: ethutil.Bytes(hashes[i]), ParentHeaderHash: ethutil.Bytes(hashes[i-1])}, self.id)
	}
}

// peer callbacks
// -1 is special: not found (a hash never seen)
// records block hashes requests by the blockPool
func (self *peerTester) requestBlockHashes(hash []byte) error {
	indexes := self.hashPool.hashesToIndexes([][]byte{hash})
	fmt.Printf("[%s] blocks hash request %v %x\n", self.id, indexes[0], hash[:4])
	self.lock.Lock()
	defer self.lock.Unlock()
	self.blockHashesRequests = append(self.blockHashesRequests, indexes[0])
	return nil
}

// records block requests by the blockPool
func (self *peerTester) requestBlocks(hashes [][]byte) error {
	indexes := self.hashPool.hashesToIndexes(hashes)
	fmt.Printf("blocks request %v %x...\n", indexes, hashes[0][:4])
	self.lock.Lock()
	defer self.lock.Unlock()
	self.blocksRequests = append(self.blocksRequests, indexes)
	for _, i := range indexes {
		self.blocksRequestsMap[i] = true
	}
	return nil
}

// records the error codes of all the peerErrors found the blockPool
func (self *peerTester) peerError(code int, format string, params ...interface{}) {
	self.peerErrors = append(self.peerErrors, code)
}

// the actual tests
func TestAddPeer(t *testing.T) {
	logInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	peer0 := blockPoolTester.newPeer("peer0", 1, 0)
	peer1 := blockPoolTester.newPeer("peer1", 2, 1)
	peer2 := blockPoolTester.newPeer("peer2", 3, 2)
	var peer *peerInfo

	blockPool.Start()

	// pool
	best := peer0.AddPeer()
	if !best {
		t.Errorf("peer0 (TD=1) not accepted as best")
	}
	if blockPool.peer.id != "peer0" {
		t.Errorf("peer0 (TD=1) not set as best")
	}
	peer0.checkBlockHashesRequests(0)

	best = peer2.AddPeer()
	if !best {
		t.Errorf("peer2 (TD=3) not accepted as best")
	}
	if blockPool.peer.id != "peer2" {
		t.Errorf("peer2 (TD=3) not set as best")
	}
	peer2.checkBlockHashesRequests(2)

	best = peer1.AddPeer()
	if best {
		t.Errorf("peer1 (TD=2) accepted as best")
	}
	if blockPool.peer.id != "peer2" {
		t.Errorf("peer2 (TD=3) not set any more as best")
	}
	if blockPool.peer.td.Cmp(big.NewInt(int64(3))) != 0 {
		t.Errorf("peer1 TD not set")
	}

	peer2.td = 4
	peer2.currentBlock = 3
	best = peer2.AddPeer()
	if !best {
		t.Errorf("peer2 (TD=4) not accepted as best")
	}
	if blockPool.peer.id != "peer2" {
		t.Errorf("peer2 (TD=4) not set as best")
	}
	if blockPool.peer.td.Cmp(big.NewInt(int64(4))) != 0 {
		t.Errorf("peer2 TD not updated")
	}
	peer2.checkBlockHashesRequests(2, 3)

	peer1.td = 3
	peer1.currentBlock = 2
	best = peer1.AddPeer()
	if best {
		t.Errorf("peer1 (TD=3) should not be set as best")
	}
	if blockPool.peer.id == "peer1" {
		t.Errorf("peer1 (TD=3) should not be set as best")
	}
	peer, best = blockPool.getPeer("peer1")
	if peer.td.Cmp(big.NewInt(int64(3))) != 0 {
		t.Errorf("peer1 TD should be updated")
	}

	blockPool.RemovePeer("peer2")
	peer, best = blockPool.getPeer("peer2")
	if peer != nil {
		t.Errorf("peer2 not removed")
	}

	if blockPool.peer.id != "peer1" {
		t.Errorf("existing peer1 (TD=3) should be set as best peer")
	}
	peer1.checkBlockHashesRequests(2)

	blockPool.RemovePeer("peer1")
	peer, best = blockPool.getPeer("peer1")
	if peer != nil {
		t.Errorf("peer1 not removed")
	}

	if blockPool.peer.id != "peer0" {
		t.Errorf("existing peer0 (TD=1) should be set as best peer")
	}

	blockPool.RemovePeer("peer0")
	peer, best = blockPool.getPeer("peer0")
	if peer != nil {
		t.Errorf("peer1 not removed")
	}

	// adding back earlier peer ok
	peer0.currentBlock = 3
	best = peer0.AddPeer()
	if !best {
		t.Errorf("peer0 (TD=1) should be set as best")
	}

	if blockPool.peer.id != "peer0" {
		t.Errorf("peer0 (TD=1) should be set as best")
	}
	peer0.checkBlockHashesRequests(0, 0, 3)

	blockPool.Stop()

}

func TestPeerWithKnownBlock(t *testing.T) {
	logInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.refBlockChain[0] = nil
	blockPoolTester.blockChain[0] = nil
	// hashPool, blockPool, blockPoolTester := newTestBlockPool()
	blockPool.Start()

	peer0 := blockPoolTester.newPeer("0", 1, 0)
	peer0.AddPeer()

	blockPool.Stop()
	// no request on known block
	peer0.checkBlockHashesRequests()
}

func TestSimpleChain(t *testing.T) {
	logInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(2)

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 2)
	peer1.AddPeer()
	go peer1.AddBlockHashes(2, 1, 0)
	peer1.AddBlocks(0, 1, 2)

	blockPool.Wait(waitTimeout * time.Second)
	blockPool.Stop()
	blockPoolTester.refBlockChain[2] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestInvalidBlock(t *testing.T) {
	logInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(2)
	blockPoolTester.refBlockChain[2] = []int{}

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 3)
	peer1.AddPeer()
	go peer1.AddBlockHashes(3, 2, 1, 0)
	peer1.AddBlocks(0, 1, 2, 3)

	blockPool.Wait(waitTimeout * time.Second)
	blockPool.Stop()
	blockPoolTester.refBlockChain[2] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
	if len(peer1.peerErrors) == 1 {
		if peer1.peerErrors[0] != ErrInvalidBlock {
			t.Errorf("wrong error, got %v, expected %v", peer1.peerErrors[0], ErrInvalidBlock)
		}
	} else {
		t.Errorf("expected invalid block error, got nothing")
	}
}

func TestVerifyPoW(t *testing.T) {
	logInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(3)
	first := false
	blockPoolTester.blockPool.verifyPoW = func(b pow.Block) bool {
		bb, _ := b.(*types.Block)
		indexes := blockPoolTester.hashPool.hashesToIndexes([][]byte{bb.Hash()})
		if indexes[0] == 1 && !first {
			first = true
			return false
		} else {
			return true
		}

	}

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 2)
	peer1.AddPeer()
	go peer1.AddBlockHashes(2, 1, 0)
	peer1.AddBlocks(0, 1, 2)
	peer1.AddBlocks(0, 1)

	blockPool.Wait(waitTimeout * time.Second)
	blockPool.Stop()
	blockPoolTester.refBlockChain[2] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
	if len(peer1.peerErrors) == 1 {
		if peer1.peerErrors[0] != ErrInvalidPoW {
			t.Errorf("wrong error, got %v, expected %v", peer1.peerErrors[0], ErrInvalidPoW)
		}
	} else {
		t.Errorf("expected invalid pow error, got nothing")
	}
}

func TestMultiSectionChain(t *testing.T) {
	logInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(5)

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 5)

	peer1.AddPeer()
	go peer1.AddBlockHashes(5, 4, 3)
	go peer1.AddBlocks(2, 3, 4, 5)
	go peer1.AddBlockHashes(3, 2, 1, 0)
	peer1.AddBlocks(0, 1, 2)

	blockPool.Wait(waitTimeout * time.Second)
	blockPool.Stop()
	blockPoolTester.refBlockChain[5] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestNewBlocksOnPartialChain(t *testing.T) {
	logInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(7)
	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 5)

	peer1.AddPeer()
	go peer1.AddBlockHashes(5, 4, 3)
	peer1.AddBlocks(2, 3) // partially complete section
	// peer1 found new blocks
	peer1.td = 2
	peer1.currentBlock = 7
	peer1.AddPeer()
	go peer1.AddBlockHashes(7, 6, 5)
	go peer1.AddBlocks(3, 4, 5, 6, 7)
	go peer1.AddBlockHashes(3, 2, 1, 0) // tests that hash request from known chain root is remembered
	peer1.AddBlocks(0, 1, 2)

	blockPool.Wait(waitTimeout * time.Second)
	blockPool.Stop()
	blockPoolTester.refBlockChain[7] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestPeerSwitch(t *testing.T) {
	logInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(6)

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 5)
	peer2 := blockPoolTester.newPeer("peer2", 2, 6)
	peer2.blocksRequestsMap = peer1.blocksRequestsMap

	peer1.AddPeer()
	go peer1.AddBlockHashes(5, 4, 3)
	peer1.AddBlocks(2, 3)               // section partially complete, block 3 will be preserved after peer demoted
	peer2.AddPeer()                     // peer2 is promoted as best peer, peer1 is demoted
	go peer2.AddBlockHashes(6, 5)       //
	go peer2.AddBlocks(4, 5, 6)         // tests that block request for earlier section is remembered
	go peer1.AddBlocks(3, 4)            // tests that connecting section by demoted peer is remembered and blocks are accepted from demoted peer
	go peer2.AddBlockHashes(3, 2, 1, 0) // tests that known chain section is activated, hash requests from 3 is remembered
	peer2.AddBlocks(0, 1, 2)            // final blocks linking to blockchain sent

	blockPool.Wait(waitTimeout * time.Second)
	blockPool.Stop()
	blockPoolTester.refBlockChain[6] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestPeerDownSwitch(t *testing.T) {
	logInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(6)
	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 4)
	peer2 := blockPoolTester.newPeer("peer2", 2, 6)
	peer2.blocksRequestsMap = peer1.blocksRequestsMap

	peer2.AddPeer()
	go peer2.AddBlockHashes(6, 5, 4)
	peer2.AddBlocks(5, 6)                  // partially complete, section will be preserved
	blockPool.RemovePeer("peer2")          // peer2 disconnects
	peer1.AddPeer()                        // inferior peer1 is promoted as best peer
	go peer1.AddBlockHashes(4, 3, 2, 1, 0) //
	go peer1.AddBlocks(3, 4, 5)            // tests that section set by demoted peer is remembered and blocks are accepted
	peer1.AddBlocks(0, 1, 2, 3)

	blockPool.Wait(waitTimeout * time.Second)
	blockPool.Stop()
	blockPoolTester.refBlockChain[6] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestPeerSwitchBack(t *testing.T) {
	logInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(8)

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 2, 11)
	peer2 := blockPoolTester.newPeer("peer2", 1, 8)
	peer2.blocksRequestsMap = peer1.blocksRequestsMap

	peer2.AddPeer()
	go peer2.AddBlockHashes(8, 7, 6)
	go peer2.AddBlockHashes(6, 5, 4)
	peer2.AddBlocks(4, 5)                  // section partially complete
	peer1.AddPeer()                        // peer1 is promoted as best peer
	go peer1.AddBlockHashes(11, 10)        // only gives useless results
	blockPool.RemovePeer("peer1")          // peer1 disconnects
	go peer2.AddBlockHashes(4, 3, 2, 1, 0) // tests that asking for hashes from 4 is remembered
	go peer2.AddBlocks(3, 4, 5, 6, 7, 8)   // tests that section 4, 5, 6 and 7, 8 are remembered for missing blocks
	peer2.AddBlocks(0, 1, 2, 3)

	blockPool.Wait(waitTimeout * time.Second)
	blockPool.Stop()
	blockPoolTester.refBlockChain[8] = []int{}
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)
}

func TestForkSimple(t *testing.T) {
	logInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(9)
	blockPoolTester.refBlockChain[3] = []int{4, 7}
	delete(blockPoolTester.refBlockChain, 6)

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 9)
	peer2 := blockPoolTester.newPeer("peer2", 2, 6)
	peer2.blocksRequestsMap = peer1.blocksRequestsMap

	peer1.AddPeer()
	go peer1.AddBlockHashes(9, 8, 7, 3, 2)
	peer1.AddBlocks(1, 2, 3, 7, 8, 9)
	peer2.AddPeer()                        // peer2 is promoted as best peer
	go peer2.AddBlockHashes(6, 5, 4, 3, 2) // fork on 3 -> 4 (earlier child: 7)
	go peer2.AddBlocks(1, 2, 3, 4, 5, 6)
	go peer2.AddBlockHashes(2, 1, 0)
	peer2.AddBlocks(0, 1, 2)

	blockPool.Wait(waitTimeout * time.Second)
	blockPool.Stop()
	blockPoolTester.refBlockChain[6] = []int{}
	blockPoolTester.refBlockChain[3] = []int{4}
	delete(blockPoolTester.refBlockChain, 7)
	delete(blockPoolTester.refBlockChain, 8)
	delete(blockPoolTester.refBlockChain, 9)
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)

}

func TestForkSwitchBackByNewBlocks(t *testing.T) {
	logInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(11)
	blockPoolTester.refBlockChain[3] = []int{4, 7}
	delete(blockPoolTester.refBlockChain, 6)

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 9)
	peer2 := blockPoolTester.newPeer("peer2", 2, 6)
	peer2.blocksRequestsMap = peer1.blocksRequestsMap

	peer1.AddPeer()
	go peer1.AddBlockHashes(9, 8, 7, 3, 2)
	peer1.AddBlocks(8, 9)                  // partial section
	peer2.AddPeer()                        //
	go peer2.AddBlockHashes(6, 5, 4, 3, 2) // peer2 forks on block 3
	peer2.AddBlocks(1, 2, 3, 4, 5, 6)      //

	// peer1 finds new blocks
	peer1.td = 3
	peer1.currentBlock = 11
	peer1.AddPeer()
	go peer1.AddBlockHashes(11, 10, 9)
	peer1.AddBlocks(7, 8, 9, 10, 11)
	go peer1.AddBlockHashes(7, 3) // tests that hash request from fork root is remembered
	go peer1.AddBlocks(3, 7)      // tests that block requests on earlier fork are remembered
	// go peer1.AddBlockHashes(1, 0) // tests that hash request from root of connecting chain section (added by demoted peer) is remembered
	go peer1.AddBlockHashes(2, 1, 0) // tests that hash request from root of connecting chain section (added by demoted peer) is remembered
	peer1.AddBlocks(0, 1, 2, 3)

	blockPool.Wait(waitTimeout * time.Second)
	blockPool.Stop()
	blockPoolTester.refBlockChain[11] = []int{}
	blockPoolTester.refBlockChain[3] = []int{7}
	delete(blockPoolTester.refBlockChain, 6)
	delete(blockPoolTester.refBlockChain, 5)
	delete(blockPoolTester.refBlockChain, 4)
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)

}

func TestForkSwitchBackByPeerSwitchBack(t *testing.T) {
	logInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(9)
	blockPoolTester.refBlockChain[3] = []int{4, 7}
	delete(blockPoolTester.refBlockChain, 6)

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 9)
	peer2 := blockPoolTester.newPeer("peer2", 2, 6)
	peer2.blocksRequestsMap = peer1.blocksRequestsMap

	peer1.AddPeer()
	go peer1.AddBlockHashes(9, 8, 7, 3, 2)
	peer1.AddBlocks(8, 9)
	peer2.AddPeer()                        //
	go peer2.AddBlockHashes(6, 5, 4, 3, 2) // peer2 forks on block 3
	peer2.AddBlocks(2, 3, 4, 5, 6)         //
	blockPool.RemovePeer("peer2")          // peer2 disconnects, peer1 is promoted again as best peer
	peer1.AddBlockHashes(7, 3)             // tests that hash request from fork root is remembered
	go peer1.AddBlocks(3, 7, 8)            // tests that block requests on earlier fork are remembered
	go peer1.AddBlockHashes(2, 1, 0)       //
	peer1.AddBlocks(0, 1, 2, 3)

	blockPool.Wait(waitTimeout * time.Second)
	blockPool.Stop()
	blockPoolTester.refBlockChain[9] = []int{}
	blockPoolTester.refBlockChain[3] = []int{7}
	delete(blockPoolTester.refBlockChain, 6)
	delete(blockPoolTester.refBlockChain, 5)
	delete(blockPoolTester.refBlockChain, 4)
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)

}

func TestForkCompleteSectionSwitchBackByPeerSwitchBack(t *testing.T) {
	logInit()
	_, blockPool, blockPoolTester := newTestBlockPool(t)
	blockPoolTester.blockChain[0] = nil
	blockPoolTester.initRefBlockChain(9)
	blockPoolTester.refBlockChain[3] = []int{4, 7}
	delete(blockPoolTester.refBlockChain, 6)

	blockPool.Start()

	peer1 := blockPoolTester.newPeer("peer1", 1, 9)
	peer2 := blockPoolTester.newPeer("peer2", 2, 6)
	peer2.blocksRequestsMap = peer1.blocksRequestsMap

	peer1.AddPeer()
	go peer1.AddBlockHashes(9, 8, 7)
	peer1.AddBlocks(3, 7, 8, 9) // make sure this section is complete
	time.Sleep(1 * time.Second)
	go peer1.AddBlockHashes(7, 3, 2)       // block 3/7 is section boundary
	peer1.AddBlocks(2, 3)                  // partially complete sections
	peer2.AddPeer()                        //
	go peer2.AddBlockHashes(6, 5, 4, 3, 2) // peer2 forks on block 3
	peer2.AddBlocks(2, 3, 4, 5, 6)         // block 2 still missing.
	blockPool.RemovePeer("peer2")          // peer2 disconnects, peer1 is promoted again as best peer
	peer1.AddBlockHashes(7, 3)             // tests that hash request from fork root is remembered even though section process completed
	go peer1.AddBlockHashes(2, 1, 0)       //
	peer1.AddBlocks(0, 1, 2)

	blockPool.Wait(waitTimeout * time.Second)
	blockPool.Stop()
	blockPoolTester.refBlockChain[9] = []int{}
	blockPoolTester.refBlockChain[3] = []int{7}
	delete(blockPoolTester.refBlockChain, 6)
	delete(blockPoolTester.refBlockChain, 5)
	delete(blockPoolTester.refBlockChain, 4)
	blockPoolTester.checkBlockChain(blockPoolTester.refBlockChain)

}
