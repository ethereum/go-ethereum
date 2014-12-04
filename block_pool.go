package eth

import (
	"bytes"
	"container/list"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/wire"
)

var poollogger = logger.NewLogger("BPOOL")

type block struct {
	from      *Peer
	peer      *Peer
	block     *types.Block
	reqAt     time.Time
	requested int
}

type BlockPool struct {
	mut sync.Mutex

	eth *Ethereum

	hashes [][]byte
	pool   map[string]*block

	td   *big.Int
	quit chan bool

	fetchingHashes    bool
	downloadStartedAt time.Time

	ChainLength, BlocksProcessed int

	peer *Peer
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
	return len(self.hashes)
}

func (self *BlockPool) Reset() {
	self.pool = make(map[string]*block)
	self.hashes = nil
}

func (self *BlockPool) HasLatestHash() bool {
	self.mut.Lock()
	defer self.mut.Unlock()

	return self.pool[string(self.eth.ChainManager().CurrentBlock.Hash())] != nil
}

func (self *BlockPool) HasCommonHash(hash []byte) bool {
	return self.eth.ChainManager().GetBlock(hash) != nil
}

func (self *BlockPool) Blocks() (blocks types.Blocks) {
	for _, item := range self.pool {
		if item.block != nil {
			blocks = append(blocks, item.block)
		}
	}

	return
}

func (self *BlockPool) FetchHashes(peer *Peer) bool {
	highestTd := self.eth.HighestTDPeer()

	if (self.peer == nil && peer.td.Cmp(highestTd) >= 0) || (self.peer != nil && peer.td.Cmp(self.peer.td) > 0) || self.peer == peer {
		if self.peer != peer {
			poollogger.Infof("Found better suitable peer (%v vs %v)\n", self.td, peer.td)

			if self.peer != nil {
				self.peer.doneFetchingHashes = true
			}
		}

		self.peer = peer
		self.td = peer.td

		if !self.HasLatestHash() {
			peer.doneFetchingHashes = fInfo
			const amount = 256
			peerlogger.Debugf("Fetching hashes (%d) %x...\n", amount, peer.lastReceivedHash[0:4])
			peer.QueueMessage(wire.NewMessage(wire.MsgGetBlockHashesTy, []interface{}{peer.lastReceivedHash, uint32(amount)}))
		}

		return true
	}

	return false
}

func (self *BlockPool) AddHash(hash []byte, peer *Peer) {
	self.mut.Lock()
	defer self.mut.Unlock()

	if self.pool[string(hash)] == nil {
		self.pool[string(hash)] = &block{peer, nil, nil, time.Now(), 0}

		self.hashes = append([][]byte{hash}, self.hashes...)
	}
}

func (self *BlockPool) Add(b *types.Block, peer *Peer) {
	self.addBlock(b, peer, false)
}

func (self *BlockPool) AddNew(b *types.Block, peer *Peer) {
	self.addBlock(b, peer, true)
}

func (self *BlockPool) addBlock(b *types.Block, peer *Peer, newBlock bool) {
	self.mut.Lock()
	defer self.mut.Unlock()

	hash := string(b.Hash())

	if self.pool[hash] == nil && !self.eth.ChainManager().HasBlock(b.Hash()) {
		poollogger.Infof("Got unrequested block (%x...)\n", hash[0:4])

		self.hashes = append(self.hashes, b.Hash())
		self.pool[hash] = &block{peer, peer, b, time.Now(), 0}

		// The following is only performed on an unrequested new block
		if newBlock {
			fmt.Println("1.", !self.eth.ChainManager().HasBlock(b.PrevHash), ethutil.Bytes2Hex(b.Hash()[0:4]), ethutil.Bytes2Hex(b.PrevHash[0:4]))
			fmt.Println("2.", self.pool[string(b.PrevHash)] == nil)
			fmt.Println("3.", !self.fetchingHashes)
			if !self.eth.ChainManager().HasBlock(b.PrevHash) && self.pool[string(b.PrevHash)] == nil && !self.fetchingHashes {
				poollogger.Infof("Unknown chain, requesting (%x...)\n", b.PrevHash[0:4])
				peer.QueueMessage(wire.NewMessage(wire.MsgGetBlockHashesTy, []interface{}{b.Hash(), uint32(256)}))
			}
		}
	} else if self.pool[hash] != nil {
		self.pool[hash].block = b
	}

	self.BlocksProcessed++
}

func (self *BlockPool) Remove(hash []byte) {
	self.mut.Lock()
	defer self.mut.Unlock()

	self.hashes = ethutil.DeleteFromByteSlice(self.hashes, hash)
	delete(self.pool, string(hash))
}

func (self *BlockPool) DistributeHashes() {
	self.mut.Lock()
	defer self.mut.Unlock()

	var (
		peerLen = self.eth.peers.Len()
		amount  = 256 * peerLen
		dist    = make(map[*Peer][][]byte)
	)

	num := int(math.Min(float64(amount), float64(len(self.pool))))
	for i, j := 0, 0; i < len(self.hashes) && j < num; i++ {
		hash := self.hashes[i]
		item := self.pool[string(hash)]

		if item != nil && item.block == nil {
			var peer *Peer
			lastFetchFailed := time.Since(item.reqAt) > 5*time.Second

			// Handle failed requests
			if lastFetchFailed && item.requested > 5 && item.peer != nil {
				if item.requested < 100 {
					// Select peer the hash was retrieved off
					peer = item.from
				} else {
					// Remove it
					self.hashes = ethutil.DeleteFromByteSlice(self.hashes, hash)
					delete(self.pool, string(hash))
				}
			} else if lastFetchFailed || item.peer == nil {
				// Find a suitable, available peer
				eachPeer(self.eth.peers, func(p *Peer, v *list.Element) {
					if peer == nil && len(dist[p]) < amount/peerLen && p.statusKnown {
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

	if len(dist) > 0 {
		self.downloadStartedAt = time.Now()
	}
}

func (self *BlockPool) Start() {
	go self.downloadThread()
	go self.chainThread()
}

func (self *BlockPool) Stop() {
	close(self.quit)
}

func (self *BlockPool) downloadThread() {
	serviceTimer := time.NewTicker(100 * time.Millisecond)
out:
	for {
		select {
		case <-self.quit:
			break out
		case <-serviceTimer.C:
			// Check if we're catching up. If not distribute the hashes to
			// the peers and download the blockchain
			self.fetchingHashes = false
			eachPeer(self.eth.peers, func(p *Peer, v *list.Element) {
				if p.statusKnown && p.FetchingHashes() {
					self.fetchingHashes = true
				}
			})

			if len(self.hashes) > 0 {
				self.DistributeHashes()
			}

			if self.ChainLength < len(self.hashes) {
				self.ChainLength = len(self.hashes)
			}

			/*
				if !self.fetchingHashes {
					blocks := self.Blocks()
					chain.BlockBy(chain.Number).Sort(blocks)

					if len(blocks) > 0 {
						if !self.eth.ChainManager().HasBlock(b.PrevHash) && self.pool[string(b.PrevHash)] == nil && !self.fetchingHashes {
						}
					}
				}
			*/
		}
	}
}

func (self *BlockPool) chainThread() {
	procTimer := time.NewTicker(500 * time.Millisecond)
out:
	for {
		select {
		case <-self.quit:
			break out
		case <-procTimer.C:
			blocks := self.Blocks()
			types.BlockBy(types.Number).Sort(blocks)

			// Find common block
			for i, block := range blocks {
				if self.eth.ChainManager().HasBlock(block.PrevHash) {
					blocks = blocks[i:]
					break
				}
			}

			if len(blocks) > 0 {
				if self.eth.ChainManager().HasBlock(blocks[0].PrevHash) {
					for i, block := range blocks[1:] {
						// NOTE: The Ith element in this loop refers to the previous block in
						// outer "blocks"
						if bytes.Compare(block.PrevHash, blocks[i].Hash()) != 0 {
							blocks = blocks[:i]

							break
						}
					}
				} else {
					blocks = nil
				}
			}

			if len(blocks) > 0 {
				chainman := self.eth.ChainManager()

				err := chainman.InsertChain(blocks)
				if err != nil {
					poollogger.Debugln(err)

					self.Reset()

					if self.peer != nil && self.peer.conn != nil {
						poollogger.Debugf("Punishing peer for supplying bad chain (%v)\n", self.peer.conn.RemoteAddr())
					}

					// This peer gave us bad hashes and made us fetch a bad chain, therefor he shall be punished.
					self.eth.BlacklistPeer(self.peer)
					self.peer.StopWithReason(DiscBadPeer)
					self.td = ethutil.Big0
					self.peer = nil
				}

				for _, block := range blocks {
					self.Remove(block.Hash())
				}
			}
		}
	}
}
