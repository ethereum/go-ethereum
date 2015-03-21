package miner

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/pow"
)

type CpuMiner struct {
	c             chan *types.Block
	quit          chan struct{}
	quitCurrentOp chan struct{}
	returnCh      chan<- Work

	index int
	pow   pow.PoW
}

func NewCpuMiner(index int, pow pow.PoW) *CpuMiner {
	miner := &CpuMiner{
		pow:   pow,
		index: index,
	}

	return miner
}

func (self *CpuMiner) Work() chan<- *types.Block { return self.c }
func (self *CpuMiner) Pow() pow.PoW              { return self.pow }
func (self *CpuMiner) SetWorkCh(ch chan<- Work)  { self.returnCh = ch }

func (self *CpuMiner) Stop() {
	close(self.quit)
	close(self.quitCurrentOp)
}

func (self *CpuMiner) Start() {
	self.quit = make(chan struct{})
	self.quitCurrentOp = make(chan struct{}, 1)
	self.c = make(chan *types.Block, 1)

	go self.update()
}

func (self *CpuMiner) update() {
	justStarted := true
out:
	for {
		select {
		case block := <-self.c:
			if justStarted {
				justStarted = true
			} else {
				self.quitCurrentOp <- struct{}{}
			}

			go self.mine(block)
		case <-self.quit:
			break out
		}
	}

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

func (self *CpuMiner) mine(block *types.Block) {
	minerlogger.Infof("(re)started agent[%d]. mining...\n", self.index)
	nonce, mixDigest, seedHash := self.pow.Search(block, self.quitCurrentOp)
	if nonce != 0 {
		self.returnCh <- Work{block.Number().Uint64(), nonce, mixDigest, seedHash}
	}
}

func (self *CpuMiner) GetHashRate() int64 {
	return self.pow.GetHashrate()
}
