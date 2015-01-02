package eth

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	ethlogger "github.com/ethereum/go-ethereum/logger"
)

var sys = ethlogger.NewStdLogSystem(os.Stdout, log.LstdFlags, ethlogger.LogLevel(ethlogger.DebugDetailLevel))

type testChainManager struct {
	knownBlock func(hash []byte) bool
	addBlock   func(*types.Block) error
	checkPoW   func(*types.Block) bool
}

func (self *testChainManager) KnownBlock(hash []byte) bool {
	if self.knownBlock != nil {
		return self.knownBlock(hash)
	}
	return false
}

func (self *testChainManager) AddBlock(block *types.Block) error {
	if self.addBlock != nil {
		return self.addBlock(block)
	}
	return nil
}

func (self *testChainManager) CheckPoW(block *types.Block) bool {
	if self.checkPoW != nil {
		return self.checkPoW(block)
	}
	return false
}

func knownBlock(hashes ...[]byte) (f func([]byte) bool) {
	f = func(block []byte) bool {
		for _, hash := range hashes {
			if bytes.Compare(block, hash) == 0 {
				return true
			}
		}
		return false
	}
	return
}

func addBlock(hashes ...[]byte) (f func(*types.Block) error) {
	f = func(block *types.Block) error {
		for _, hash := range hashes {
			if bytes.Compare(block.Hash(), hash) == 0 {
				return fmt.Errorf("invalid by test")
			}
		}
		return nil
	}
	return
}

func checkPoW(hashes ...[]byte) (f func(*types.Block) bool) {
	f = func(block *types.Block) bool {
		for _, hash := range hashes {
			if bytes.Compare(block.Hash(), hash) == 0 {
				return false
			}
		}
		return true
	}
	return
}

func newTestChainManager(knownBlocks [][]byte, invalidBlocks [][]byte, invalidPoW [][]byte) *testChainManager {
	return &testChainManager{
		knownBlock: knownBlock(knownBlocks...),
		addBlock:   addBlock(invalidBlocks...),
		checkPoW:   checkPoW(invalidPoW...),
	}
}

type intToHash map[int][]byte

type hashToInt map[string]int

type testHashPool struct {
	intToHash
	hashToInt
}

func newHash(i int) []byte {
	return crypto.Sha3([]byte(string(i)))
}

func newTestBlockPool(knownBlockIndexes []int, invalidBlockIndexes []int, invalidPoWIndexes []int) (hashPool *testHashPool, blockPool *BlockPool) {
	hashPool = &testHashPool{make(intToHash), make(hashToInt)}
	knownBlocks := hashPool.indexesToHashes(knownBlockIndexes)
	invalidBlocks := hashPool.indexesToHashes(invalidBlockIndexes)
	invalidPoW := hashPool.indexesToHashes(invalidPoWIndexes)
	blockPool = NewBlockPool(newTestChainManager(knownBlocks, invalidBlocks, invalidPoW))
	return
}

func (self *testHashPool) indexesToHashes(indexes []int) (hashes [][]byte) {
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
	for _, hash := range hashes {
		i, found := self.hashToInt[string(hash)]
		if !found {
			i = -1
		}
		indexes = append(indexes, i)
	}
	return
}

type protocolChecker struct {
	blockHashesRequests []int
	blocksRequests      [][]int
	invalidBlocks       []error
	hashPool            *testHashPool
	lock                sync.Mutex
}

// -1 is special: not found (a hash never seen)
func (self *protocolChecker) requestBlockHashesCallBack() (requestBlockHashesCallBack func([]byte) error) {
	requestBlockHashesCallBack = func(hash []byte) error {
		indexes := self.hashPool.hashesToIndexes([][]byte{hash})
		self.lock.Lock()
		defer self.lock.Unlock()
		self.blockHashesRequests = append(self.blockHashesRequests, indexes[0])
		return nil
	}
	return
}

func (self *protocolChecker) requestBlocksCallBack() (requestBlocksCallBack func([][]byte) error) {
	requestBlocksCallBack = func(hashes [][]byte) error {
		indexes := self.hashPool.hashesToIndexes(hashes)
		self.lock.Lock()
		defer self.lock.Unlock()
		self.blocksRequests = append(self.blocksRequests, indexes)
		return nil
	}
	return
}

func (self *protocolChecker) invalidBlockCallBack() (invalidBlockCallBack func(error)) {
	invalidBlockCallBack = func(err error) {
		self.invalidBlocks = append(self.invalidBlocks, err)
	}
	return
}

func TestAddPeer(t *testing.T) {
	ethlogger.AddLogSystem(sys)
	knownBlockIndexes := []int{0, 1}
	invalidBlockIndexes := []int{2, 3}
	invalidPoWIndexes := []int{4, 5}
	hashPool, blockPool := newTestBlockPool(knownBlockIndexes, invalidBlockIndexes, invalidPoWIndexes)
	// TODO:
	// hashPool, blockPool, blockChainChecker = newTestBlockPool(knownBlockIndexes, invalidBlockIndexes, invalidPoWIndexes)
	peer0 := &protocolChecker{
		// blockHashesRequests: make([]int),
		// blocksRequests:      make([][]int),
		// invalidBlocks:       make([]error),
		hashPool: hashPool,
	}
	best := blockPool.AddPeer(ethutil.Big1, newHash(100), "0",
		peer0.requestBlockHashesCallBack(),
		peer0.requestBlocksCallBack(),
		peer0.invalidBlockCallBack(),
	)
	if !best {
		t.Errorf("peer not accepted as best")
	}
	blockPool.Stop()

}
