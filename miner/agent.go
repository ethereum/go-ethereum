package miner

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/pow"
)

type CpuAgent struct {
	mu sync.Mutex

	workCh        chan *types.Block
	quit          chan struct{}
	quitCurrentOp chan struct{}
	returnCh      chan<- *types.Block

	index int
	pow   pow.PoW
}

func NewCpuAgent(index int, pow pow.PoW) *CpuAgent {
	miner := &CpuAgent{
		pow:   pow,
		index: index,
	}

	return miner
}

func (self *CpuAgent) Work() chan<- *types.Block          { return self.workCh }
func (self *CpuAgent) Pow() pow.PoW                       { return self.pow }
func (self *CpuAgent) SetReturnCh(ch chan<- *types.Block) { self.returnCh = ch }

func (self *CpuAgent) Stop() {
	self.mu.Lock()
	defer self.mu.Unlock()

	close(self.quit)
	close(self.quitCurrentOp)
}

func (self *CpuAgent) Start() {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.quit = make(chan struct{})
	// creating current op ch makes sure we're not closing a nil ch
	// later on
	self.quitCurrentOp = make(chan struct{})
	self.workCh = make(chan *types.Block, 1)

	go self.update()
}

func (self *CpuAgent) update() {
out:
	for {
		select {
		case block := <-self.workCh:
			self.mu.Lock()
			close(self.quitCurrentOp)
			self.mu.Unlock()

			go self.mine(block)
		case <-self.quit:
			break out
		}
	}

done:
	// Empty work channel
	for {
		select {
		case <-self.workCh:
		default:
			close(self.workCh)

			break done
		}
	}
}

func (self *CpuAgent) mine(block *types.Block) {
	glog.V(logger.Debug).Infof("(re)started agent[%d]. mining...\n", self.index)

	// Reset the channel
	self.mu.Lock()
	self.quitCurrentOp = make(chan struct{})
	self.mu.Unlock()

	// Mine
	nonce, mixDigest := self.pow.Search(block, self.quitCurrentOp)
	if nonce != 0 {
		block.SetNonce(nonce)
		block.Header().MixDigest = common.BytesToHash(mixDigest)
		self.returnCh <- block
	} else {
		self.returnCh <- nil
	}
}

func (self *CpuAgent) GetHashRate() int64 {
	return self.pow.GetHashrate()
}
