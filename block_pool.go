package eth

import (
	"math"
	"math/big"
	"sync"

	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
)

type block struct {
	peer  *Peer
	block *ethchain.Block
}

type BlockPool struct {
	mut sync.Mutex

	eth *Ethereum

	hashPool [][]byte
	pool     map[string]*block

	td *big.Int
}

func NewBlockPool(eth *Ethereum) *BlockPool {
	return &BlockPool{
		eth:  eth,
		pool: make(map[string]*block),
		td:   ethutil.Big0,
	}
}

func (self *BlockPool) HasLatestHash() bool {
	return self.pool[string(self.eth.BlockChain().CurrentBlock.Hash())] != nil
}

func (self *BlockPool) HasCommonHash(hash []byte) bool {
	return self.eth.BlockChain().GetBlock(hash) != nil
}

func (self *BlockPool) AddHash(hash []byte) {
	if self.pool[string(hash)] == nil {
		self.pool[string(hash)] = &block{nil, nil}

		self.hashPool = append([][]byte{hash}, self.hashPool...)
	}
}

func (self *BlockPool) SetBlock(b *ethchain.Block, peer *Peer) {
	hash := string(b.Hash())

	if self.pool[hash] == nil && !self.eth.BlockChain().HasBlock(b.Hash()) {
		self.hashPool = append(self.hashPool, b.Hash())
		self.pool[hash] = &block{peer, b}
	} else if self.pool[hash] != nil {
		self.pool[hash].block = b
	}
}

func (self *BlockPool) CheckLinkAndProcess(f func(block *ethchain.Block)) {

	var blocks ethchain.Blocks
	for _, item := range self.pool {
		if item.block != nil {
			blocks = append(blocks, item.block)
		}
	}

	ethchain.BlockBy(ethchain.Number).Sort(blocks)
	for _, block := range blocks {
		if self.eth.BlockChain().HasBlock(block.PrevHash) {
			f(block)

			hash := block.Hash()
			self.hashPool = ethutil.DeleteFromByteSlice(self.hashPool, hash)
			delete(self.pool, string(hash))
		}

	}
}

func (self *BlockPool) Take(amount int, peer *Peer) (hashes [][]byte) {
	self.mut.Lock()
	defer self.mut.Unlock()

	num := int(math.Min(float64(amount), float64(len(self.pool))))
	j := 0
	for i := 0; i < len(self.hashPool) && j < num; i++ {
		hash := string(self.hashPool[i])
		if self.pool[hash] != nil && (self.pool[hash].peer == nil || self.pool[hash].peer == peer) && self.pool[hash].block == nil {
			self.pool[hash].peer = peer

			hashes = append(hashes, self.hashPool[i])
			j++
		}
	}

	return
}
