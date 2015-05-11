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
	chMu          sync.Mutex
	c             chan *types.Block
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

func (self *CpuAgent) Work() chan<- *types.Block          { return self.c }
func (self *CpuAgent) Pow() pow.PoW                       { return self.pow }
func (self *CpuAgent) SetReturnCh(ch chan<- *types.Block) { self.returnCh = ch }

func (self *CpuAgent) Stop() {
	close(self.quit)
	close(self.quitCurrentOp)
}

func (self *CpuAgent) Start() {
	self.quit = make(chan struct{})
	self.quitCurrentOp = make(chan struct{}, 1)
	self.c = make(chan *types.Block, 1)

	go self.update()
}

func (self *CpuAgent) update() {
out:
	for {
		select {
		case block := <-self.c:
			self.chMu.Lock()
			self.quitCurrentOp <- struct{}{}
			self.chMu.Unlock()

			go self.mine(block)
		case <-self.quit:
			break out
		}
	}

	//close(self.quitCurrentOp)
done:
	// Empty channel
	for {
		select {
		case <-self.c:
		default:
			close(self.c)

			break done
		}
	}
}

func (self *CpuAgent) mine(block *types.Block) {
	glog.V(logger.Debug).Infof("(re)started agent[%d]. mining...\n", self.index)

	// Reset the channel
	self.chMu.Lock()
	self.quitCurrentOp = make(chan struct{}, 1)
	self.chMu.Unlock()

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
