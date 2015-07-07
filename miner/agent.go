// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

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
}

func (self *CpuAgent) Start() {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.quit = make(chan struct{})
	// creating current op ch makes sure we're not closing a nil ch
	// later on
	self.workCh = make(chan *types.Block, 1)

	go self.update()
}

func (self *CpuAgent) update() {
out:
	for {
		select {
		case block := <-self.workCh:
			self.mu.Lock()
			if self.quitCurrentOp != nil {
				close(self.quitCurrentOp)
			}
			self.quitCurrentOp = make(chan struct{})
			go self.mine(block, self.quitCurrentOp)
			self.mu.Unlock()
		case <-self.quit:
			self.mu.Lock()
			if self.quitCurrentOp != nil {
				close(self.quitCurrentOp)
				self.quitCurrentOp = nil
			}
			self.mu.Unlock()
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

func (self *CpuAgent) mine(block *types.Block, stop <-chan struct{}) {
	glog.V(logger.Debug).Infof("(re)started agent[%d]. mining...\n", self.index)

	// Mine
	nonce, mixDigest := self.pow.Search(block, stop)
	if nonce != 0 {
		self.returnCh <- block.WithMiningResult(nonce, common.BytesToHash(mixDigest))
	} else {
		self.returnCh <- nil
	}
}

func (self *CpuAgent) GetHashRate() int64 {
	return self.pow.GetHashrate()
}
