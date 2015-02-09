package miner

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/pow"
)

type CpuMiner struct {
	c             chan *types.Block
	quit          chan struct{}
	quitCurrentOp chan struct{}
	returnCh      chan<- []byte

	index int
	pow   pow.PoW
}

func NewCpuMiner(index int, pow pow.PoW) *CpuMiner {
	miner := &CpuMiner{
		c:             make(chan *types.Block, 1),
		quit:          make(chan struct{}),
		quitCurrentOp: make(chan struct{}, 1),
		pow:           pow,
		index:         index,
	}
	go miner.update()

	return miner
}

func (self *CpuMiner) Work() chan<- *types.Block   { return self.c }
func (self *CpuMiner) Pow() pow.PoW                { return self.pow }
func (self *CpuMiner) SetNonceCh(ch chan<- []byte) { self.returnCh = ch }

func (self *CpuMiner) Stop() {
	close(self.quit)
	close(self.quitCurrentOp)
}

func (self *CpuMiner) update() {
out:
	for {
		select {
		case block := <-self.c:
			minerlogger.Infof("miner[%d] got block\n", self.index)
			// make sure it's open
			self.quitCurrentOp <- struct{}{}

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
	minerlogger.Infof("started agent[%d]. mining...\n", self.index)
	nonce := self.pow.Search(block, self.quitCurrentOp)
	if nonce != nil {
		self.returnCh <- nonce
	}
}
