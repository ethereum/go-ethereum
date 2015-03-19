package blockpool

import (
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/blockpool/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/errs"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/pow"
)

var (
	waitTimeout                    = 60 * time.Second
	testBlockHashesRequestInterval = 10 * time.Millisecond
	testBlocksRequestInterval      = 10 * time.Millisecond
	requestWatchInterval           = 10 * time.Millisecond
)

// test blockChain is an integer trie
type blockChain map[int][]int

// blockPoolTester provides the interface between tests and a blockPool
//
// refBlockChain is used to guide which blocks will be accepted as valid
// blockChain gives the current state of the blockchain and
// accumulates inserts so that we can check the resulting chain
type blockPoolTester struct {
	hashPool          *test.TestHashPool
	lock              sync.RWMutex
	reqlock           sync.RWMutex
	blocksRequestsMap map[int]bool
	refBlockChain     blockChain
	blockChain        blockChain
	blockPool         *BlockPool
	t                 *testing.T
	chainEvents       *event.TypeMux
	tds               map[int]int
}

func newTestBlockPool(t *testing.T) (hashPool *test.TestHashPool, blockPool *BlockPool, b *blockPoolTester) {
	hashPool = test.NewHashPool()
	b = &blockPoolTester{
		t:                 t,
		hashPool:          hashPool,
		blockChain:        make(blockChain),
		refBlockChain:     make(blockChain),
		blocksRequestsMap: make(map[int]bool),
		chainEvents:       &event.TypeMux{},
	}
	b.blockPool = New(b.hasBlock, b.insertChain, b.verifyPoW, b.chainEvents, common.Big0)
	blockPool = b.blockPool
	blockPool.Config.BlockHashesRequestInterval = testBlockHashesRequestInterval
	blockPool.Config.BlocksRequestInterval = testBlocksRequestInterval
	return
}

func (self *blockPoolTester) Errorf(format string, params ...interface{}) {
	// fmt.Printf(format+"\n", params...)
	self.t.Errorf(format, params...)
}

// blockPoolTester implements the 3 callbacks needed by the blockPool:
// hasBlock, insetChain, verifyPoW
func (self *blockPoolTester) hasBlock(block common.Hash) (ok bool) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	indexes := self.hashPool.HashesToIndexes([]common.Hash{block})
	i := indexes[0]
	_, ok = self.blockChain[i]
	// fmt.Printf("has block %v (%x...): %v\n", i, block[0:4], ok)
	return
}

func (self *blockPoolTester) insertChain(blocks types.Blocks) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	var parent, child int
	var children, refChildren []int
	var ok bool
	for _, block := range blocks {
		child = self.hashPool.HashesToIndexes([]common.Hash{block.Hash()})[0]
		var td int
		if self.tds != nil {
			td, ok = self.tds[child]
		}
		if !ok {
			td = child
		}
		block.Td = big.NewInt(int64(td))
		_, ok = self.blockChain[child]
		if ok {
			// fmt.Printf("block %v already in blockchain\n", child)
			continue // already in chain
		}
		parent = self.hashPool.HashesToIndexes([]common.Hash{block.ParentHeaderHash})[0]
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
	self.lock.RLock()
	defer self.lock.RUnlock()
	// for k, v := range self.blockChain {
	// 	fmt.Printf("got: %v -> %v\n", k, v)
	// }
	// for k, v := range blockChain {
	// 	fmt.Printf("expected: %v -> %v\n", k, v)
	// }
	if len(blockChain) != len(self.blockChain) {
		self.Errorf("blockchain incorrect (zlength differ)")
	}
	for k, v := range blockChain {
		vv, ok := self.blockChain[k]
		if !ok || !test.ArrayEq(v, vv) {
			self.Errorf("blockchain incorrect on %v -> %v (!= %v)", k, vv, v)
		}
	}
}

//

// peerTester provides the peer callbacks for the blockPool
// it registers actual callbacks so that the result can be compared to desired behaviour
// provides helper functions to mock the protocol calls to the blockPool
type peerTester struct {
	blockHashesRequests []int
	blocksRequests      [][]int
	blocksRequestsMap   map[int]bool
	peerErrors          []int
	blockPool           *BlockPool
	hashPool            *test.TestHashPool
	lock                sync.RWMutex
	bt                  *blockPoolTester
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
		bt:                self,
		blocksRequestsMap: self.blocksRequestsMap,
	}
}

func (self *peerTester) Errorf(format string, params ...interface{}) {
	// fmt.Printf(format+"\n", params...)
	self.t.Errorf(format, params...)
}

// helper to compare actual and expected block requests
func (self *peerTester) checkBlocksRequests(blocksRequests ...[]int) {
	if len(blocksRequests) > len(self.blocksRequests) {
		self.Errorf("blocks requests incorrect (length differ)\ngot %v\nexpected %v", self.blocksRequests, blocksRequests)
	} else {
		for i, rr := range blocksRequests {
			r := self.blocksRequests[i]
			if !test.ArrayEq(r, rr) {
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
		if !test.ArrayEq(r, rr) {
			self.Errorf("block hashes requests incorrect\ngot %v\nexpected %v", r, rr)
		}
	}
}

// waiter function used by peer.serveBlocks
// blocking until requests appear
// since block requests are sent to any random peers
// block request map is shared between peers
// times out after waitTimeout
func (self *peerTester) waitBlocksRequests(blocksRequest ...int) {
	timeout := time.After(waitTimeout)
	rr := blocksRequest
	for {
		self.lock.RLock()
		r := self.blocksRequestsMap
		// fmt.Printf("[%s] blocks request check %v (%v)\n", self.id, rr, r)
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
		time.Sleep(requestWatchInterval)
		select {
		case <-timeout:
		default:
		}
	}
}

// waiter function used by peer.serveBlockHashes
// blocking until requests appear
// times out after a period
func (self *peerTester) waitBlockHashesRequests(blocksHashesRequest int) {
	timeout := time.After(waitTimeout)
	rr := blocksHashesRequest
	for i := 0; ; {
		self.lock.RLock()
		r := self.blockHashesRequests
		self.lock.RUnlock()
		// fmt.Printf("[%s] block hash request check %v (%v)\n", self.id, rr, r)
		for ; i < len(r); i++ {
			if rr == r[i] {
				return
			}
		}
		time.Sleep(requestWatchInterval)
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
func (self *peerTester) AddPeer() (best bool) {
	hash := self.hashPool.IndexesToHashes([]int{self.currentBlock})[0]
	best, _ = self.blockPool.AddPeer(big.NewInt(int64(self.td)), hash, self.id, self.requestBlockHashes, self.requestBlocks, self.peerError)
	return
}

// peer sends blockhashes if and when gets a request
func (self *peerTester) serveBlockHashes(indexes ...int) {
	// fmt.Printf("ready to serve block hashes %v\n", indexes)

	self.waitBlockHashesRequests(indexes[0])
	self.sendBlockHashes(indexes...)
}

func (self *peerTester) sendBlockHashes(indexes ...int) {
	// fmt.Printf("adding block hashes %v\n", indexes)
	hashes := self.hashPool.IndexesToHashes(indexes)
	i := 1
	next := func() (hash common.Hash, ok bool) {
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
func (self *peerTester) serveBlocks(indexes ...int) {
	// fmt.Printf("ready to serve blocks %v\n", indexes[1:])
	self.waitBlocksRequests(indexes[1:]...)
	self.sendBlocks(indexes...)
}

func (self *peerTester) sendBlocks(indexes ...int) {
	// fmt.Printf("adding blocks %v \n", indexes)
	hashes := self.hashPool.IndexesToHashes(indexes)
	for i := 1; i < len(hashes); i++ {
		// fmt.Printf("adding block %v %x\n", indexes[i], hashes[i][:4])
		self.blockPool.AddBlock(&types.Block{HeaderHash: hashes[i], ParentHeaderHash: hashes[i-1]}, self.id)
	}
}

// peer callbacks
// -1 is special: not found (a hash never seen)
// records block hashes requests by the blockPool
func (self *peerTester) requestBlockHashes(hash common.Hash) error {
	indexes := self.hashPool.HashesToIndexes([]common.Hash{hash})
	// fmt.Printf("[%s] block hash request %v %x\n", self.id, indexes[0], hash[:4])
	self.lock.Lock()
	defer self.lock.Unlock()
	self.blockHashesRequests = append(self.blockHashesRequests, indexes[0])
	return nil
}

// records block requests by the blockPool
func (self *peerTester) requestBlocks(hashes []common.Hash) error {
	indexes := self.hashPool.HashesToIndexes(hashes)
	// fmt.Printf("blocks request %v %x...\n", indexes, hashes[0][:4])
	self.bt.reqlock.Lock()
	defer self.bt.reqlock.Unlock()
	self.blocksRequests = append(self.blocksRequests, indexes)
	for _, i := range indexes {
		self.blocksRequestsMap[i] = true
	}
	return nil
}

// records the error codes of all the peerErrors found the blockPool
func (self *peerTester) peerError(err *errs.Error) {
	self.peerErrors = append(self.peerErrors, err.Code)
	if err.Fatal() {
		self.blockPool.RemovePeer(self.id)
	}
}
