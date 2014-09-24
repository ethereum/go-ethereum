package eth

import (
	"bytes"
	"container/list"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
)

type block struct {
	peer      *Peer
	block     *ethchain.Block
	reqAt     time.Time
	requested int
}

type BlockPool struct {
	mut sync.Mutex

	eth *Ethereum

	hashPool [][]byte
	pool     map[string]*block

	td   *big.Int
	quit chan bool
}

func NewBlockPool(eth *Ethereum) *BlockPool {
	return &BlockPool{
		eth:  eth,
		pool: make(map[string]*block),
		td:   ethutil.Big0,
		quit: make(chan bool),
	}
}

func (self *BlockPool) Len() int {
	return len(self.hashPool)
}

func (self *BlockPool) HasLatestHash() bool {
	return self.pool[string(self.eth.BlockChain().CurrentBlock.Hash())] != nil
}

func (self *BlockPool) HasCommonHash(hash []byte) bool {
	return self.eth.BlockChain().GetBlock(hash) != nil
}

func (self *BlockPool) AddHash(hash []byte) {
	if self.pool[string(hash)] == nil {
		self.pool[string(hash)] = &block{nil, nil, time.Now(), 0}

		self.hashPool = append([][]byte{hash}, self.hashPool...)
	}
}

func (self *BlockPool) SetBlock(b *ethchain.Block, peer *Peer) {
	hash := string(b.Hash())

	if self.pool[hash] == nil && !self.eth.BlockChain().HasBlock(b.Hash()) {
		self.hashPool = append(self.hashPool, b.Hash())
		self.pool[hash] = &block{peer, b, time.Now(), 0}
	} else if self.pool[hash] != nil {
		self.pool[hash].block = b
	}
}

func (self *BlockPool) getParent(block *ethchain.Block) *ethchain.Block {
	for _, item := range self.pool {
		if item.block != nil {
			if bytes.Compare(item.block.Hash(), block.PrevHash) == 0 {
				return item.block
			}
		}
	}

	return nil
}

func (self *BlockPool) GetChainFromBlock(block *ethchain.Block) ethchain.Blocks {
	var blocks ethchain.Blocks

	for b := block; b != nil; b = self.getParent(b) {
		blocks = append(ethchain.Blocks{b}, blocks...)
	}

	return blocks
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
		item := self.pool[hash]
		if item != nil && item.block == nil &&
			(item.peer == nil ||
				((time.Since(item.reqAt) > 5*time.Second && item.peer != peer) && self.eth.peers.Len() > 1) || // multiple peers
				(time.Since(item.reqAt) > 5*time.Second && self.eth.peers.Len() == 1) /* single peer*/) {
			self.pool[hash].peer = peer
			self.pool[hash].reqAt = time.Now()
			self.pool[hash].requested++

			hashes = append(hashes, self.hashPool[i])
			j++
		}
	}

	return
}

func (self *BlockPool) Start() {
	go self.update()
}

func (self *BlockPool) Stop() {
	close(self.quit)
}

func (self *BlockPool) update() {
	serviceTimer := time.NewTicker(100 * time.Millisecond)
	procTimer := time.NewTicker(500 * time.Millisecond)
out:
	for {
		select {
		case <-self.quit:
			break out
		case <-serviceTimer.C:
			// Clean up hashes that can't be fetched
			done := true
			eachPeer(self.eth.peers, func(p *Peer, v *list.Element) {
				if p.statusKnown && p.FetchingHashes() {
					done = false
				}
			})

			if done {
				eachPeer(self.eth.peers, func(p *Peer, v *list.Element) {
					if p.statusKnown {
						hashes := self.Take(100, p)
						if len(hashes) > 0 {
							p.FetchBlocks(hashes)
							if len(hashes) == 1 {
								fmt.Printf("last hash = %x\n", hashes[0])
							} else {
								fmt.Println("Requesting", len(hashes), "of", p)
							}
						}
					}
				})
			}
		case <-procTimer.C:
			var err error
			self.CheckLinkAndProcess(func(block *ethchain.Block) {
				err = self.eth.StateManager().Process(block, false)
			})

			if err != nil {
				peerlogger.Infoln(err)
			}
		}
	}
}
