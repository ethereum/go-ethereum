// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package miner

import (
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/log"
)

type CpuAgent struct {
	mu sync.Mutex

	taskCh        chan *Package
	returnCh      chan<- *Package
	stop          chan struct{}
	quitCurrentOp chan struct{}

	chain  consensus.ChainReader
	engine consensus.Engine

	started int32 // started indicates whether the agent is currently started
}

func NewCpuAgent(chain consensus.ChainReader, engine consensus.Engine) *CpuAgent {
	agent := &CpuAgent{
		chain:  chain,
		engine: engine,
		stop:   make(chan struct{}, 1),
		taskCh: make(chan *Package, 1),
	}
	return agent
}

func (self *CpuAgent) AssignTask(p *Package) {
	if atomic.LoadInt32(&self.started) == 1 {
		self.taskCh <- p
	}
}
func (self *CpuAgent) DeliverTo(ch chan<- *Package) { self.returnCh = ch }

func (self *CpuAgent) Start() {
	if !atomic.CompareAndSwapInt32(&self.started, 0, 1) {
		return // agent already started
	}
	go self.update()
}

func (self *CpuAgent) Stop() {
	if !atomic.CompareAndSwapInt32(&self.started, 1, 0) {
		return // agent already stopped
	}
	self.stop <- struct{}{}
done:
	// Empty work channel
	for {
		select {
		case <-self.taskCh:
		default:
			break done
		}
	}
}

func (self *CpuAgent) update() {
out:
	for {
		select {
		case p := <-self.taskCh:
			self.mu.Lock()
			if self.quitCurrentOp != nil {
				close(self.quitCurrentOp)
			}
			self.quitCurrentOp = make(chan struct{})
			go self.mine(p, self.quitCurrentOp)
			self.mu.Unlock()
		case <-self.stop:
			self.mu.Lock()
			if self.quitCurrentOp != nil {
				close(self.quitCurrentOp)
				self.quitCurrentOp = nil
			}
			self.mu.Unlock()
			break out
		}
	}
}

func (self *CpuAgent) mine(p *Package, stop <-chan struct{}) {
	var err error
	if p.Block, err = self.engine.Seal(self.chain, p.Block, stop); p.Block != nil {
		log.Info("Successfully sealed new block", "number", p.Block.Number(), "hash", p.Block.Hash())
		self.returnCh <- p
	} else {
		if err != nil {
			log.Warn("Block sealing failed", "err", err)
		}
		self.returnCh <- nil
	}
}
