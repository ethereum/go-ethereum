package eth

import (
	"bytes"
	"container/list"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
)

var poollogger = ethlog.NewLogger("[BPOOL]")

type block struct {
	from      *Peer
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

	ChainLength, BlocksProcessed int
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

func (self *BlockPool) AddHash(hash []byte, peer *Peer) {
	if self.pool[string(hash)] == nil {
		self.pool[string(hash)] = &block{peer, nil, nil, time.Now(), 0}

		self.hashPool = append([][]byte{hash}, self.hashPool...)
	}
}

func (self *BlockPool) SetBlock(b *ethchain.Block, peer *Peer) {
	hash := string(b.Hash())

	if self.pool[hash] == nil && !self.eth.BlockChain().HasBlock(b.Hash()) {
		self.hashPool = append(self.hashPool, b.Hash())
		self.pool[hash] = &block{peer, peer, b, time.Now(), 0}

		if !self.eth.BlockChain().HasBlock(b.PrevHash) && self.pool[string(b.PrevHash)] == nil {
			peer.QueueMessage(ethwire.NewMessage(ethwire.MsgGetBlockHashesTy, []interface{}{b.PrevHash, uint32(256)}))
		}
	} else if self.pool[hash] != nil {
		self.pool[hash].block = b
	}

	self.BlocksProcessed++
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

func (self *BlockPool) Blocks() (blocks ethchain.Blocks) {
	for _, item := range self.pool {
		if item.block != nil {
			blocks = append(blocks, item.block)
		}
	}

	return
}

func (self *BlockPool) ProcessCanonical(f func(block *ethchain.Block)) (procAmount int) {
	blocks := self.Blocks()

	ethchain.BlockBy(ethchain.Number).Sort(blocks)
	for _, block := range blocks {
		if self.eth.BlockChain().HasBlock(block.PrevHash) {
			procAmount++

			f(block)

			hash := block.Hash()
			self.hashPool = ethutil.DeleteFromByteSlice(self.hashPool, hash)
			delete(self.pool, string(hash))
		}

	}

	return
}

func (self *BlockPool) DistributeHashes() {
	var (
		peerLen = self.eth.peers.Len()
		amount  = 200 * peerLen
		dist    = make(map[*Peer][][]byte)
	)

	num := int(math.Min(float64(amount), float64(len(self.pool))))
	for i, j := 0, 0; i < len(self.hashPool) && j < num; i++ {
		hash := self.hashPool[i]
		item := self.pool[string(hash)]

		if item != nil && item.block == nil {
			var peer *Peer
			lastFetchFailed := time.Since(item.reqAt) > 5*time.Second

			// Handle failed requests
			if lastFetchFailed && item.requested > 0 && item.peer != nil {
				if item.requested < 100 {
					// Select peer the hash was retrieved off
					peer = item.from
				} else {
					// Remove it
					self.hashPool = ethutil.DeleteFromByteSlice(self.hashPool, hash)
					delete(self.pool, string(hash))
				}
			} else if lastFetchFailed || item.peer == nil {
				// Find a suitable, available peer
				eachPeer(self.eth.peers, func(p *Peer, v *list.Element) {
					if peer == nil && len(dist[p]) < amount/peerLen {
						peer = p
					}
				})
			}

			if peer != nil {
				item.reqAt = time.Now()
				item.peer = peer
				item.requested++

				dist[peer] = append(dist[peer], hash)
			}
		}
	}

	for peer, hashes := range dist {
		peer.FetchBlocks(hashes)
	}
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
			// Check if we're catching up. If not distribute the hashes to
			// the peers and download the blockchain
			done := true
			eachPeer(self.eth.peers, func(p *Peer, v *list.Element) {
				if p.statusKnown && p.FetchingHashes() {
					done = false
				}
			})

			if done && len(self.hashPool) > 0 {
				self.DistributeHashes()
			}

			if self.ChainLength < len(self.hashPool) {
				self.ChainLength = len(self.hashPool)
			}
		case <-procTimer.C:
			// XXX We can optimize this lifting this on to a new goroutine.
			// We'd need to make sure that the pools are properly protected by a mutex
			// XXX This should moved in The Great Refactor(TM)
			amount := self.ProcessCanonical(func(block *ethchain.Block) {
				err := self.eth.StateManager().Process(block, false)
				if err != nil {
					poollogger.Infoln(err)
				}
			})

			// Do not propagate to the network on catchups
			if amount == 1 {
				block := self.eth.BlockChain().CurrentBlock
				self.eth.Broadcast(ethwire.MsgBlockTy, []interface{}{block.Value().Val})
			}
		}
	}
}
