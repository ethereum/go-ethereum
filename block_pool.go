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

	if self.pool[hash] == nil {
		self.hashPool = append(self.hashPool, b.Hash())
		self.pool[hash] = &block{peer, nil}
	}

	self.pool[hash].block = b
}

func (self *BlockPool) CheckLinkAndProcess(f func(block *ethchain.Block)) bool {
	self.mut.Lock()
	defer self.mut.Unlock()

	if self.IsLinked() {
		for i, hash := range self.hashPool {
			if self.pool[string(hash)] == nil {
				continue
			}

			block := self.pool[string(hash)].block
			if block != nil {
				f(block)

				delete(self.pool, string(hash))
			} else {
				self.hashPool = self.hashPool[i:]

				return false
			}
		}

		return true
	}

	return false
}

func (self *BlockPool) IsLinked() bool {
	if len(self.hashPool) == 0 {
		return false
	}

	for i := 0; i < len(self.hashPool); i++ {
		item := self.pool[string(self.hashPool[i])]
		if item != nil && item.block != nil {
			if self.eth.BlockChain().HasBlock(item.block.PrevHash) {
				self.hashPool = self.hashPool[i:]

				return true
			}
		}
	}

	return false
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
