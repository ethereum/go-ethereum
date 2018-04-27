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

	workCh        chan *Work
	stop          chan struct{}
	quitCurrentOp chan struct{}
	returnCh      chan<- *Result

	chain  consensus.ChainReader
	engine consensus.Engine

	isMining int32 // isMining indicates whether the agent is currently mining
}

func NewCpuAgent(chain consensus.ChainReader, engine consensus.Engine) *CpuAgent {
	miner := &CpuAgent{
		chain:  chain,
		engine: engine,
		stop:   make(chan struct{}, 1),
		workCh: make(chan *Work, 1),
	}
	return miner
}

func (a *CpuAgent) Work() chan<- *Work            { return a.workCh }
func (a *CpuAgent) SetReturnCh(ch chan<- *Result) { a.returnCh = ch }

func (a *CpuAgent) Stop() {
	if !atomic.CompareAndSwapInt32(&a.isMining, 1, 0) {
		return // agent already stopped
	}
	a.stop <- struct{}{}
done:
	// Empty work channel
	for {
		select {
		case <-a.workCh:
		default:
			break done
		}
	}
}

func (a *CpuAgent) Start() {
	if !atomic.CompareAndSwapInt32(&a.isMining, 0, 1) {
		return // agent already started
	}
	go a.update()
}

func (a *CpuAgent) update() {
out:
	for {
		select {
		case work := <-a.workCh:
			a.mu.Lock()
			if a.quitCurrentOp != nil {
				close(a.quitCurrentOp)
			}
			a.quitCurrentOp = make(chan struct{})
			go a.mine(work, a.quitCurrentOp)
			a.mu.Unlock()
		case <-a.stop:
			a.mu.Lock()
			if a.quitCurrentOp != nil {
				close(a.quitCurrentOp)
				a.quitCurrentOp = nil
			}
			a.mu.Unlock()
			break out
		}
	}
}

func (a *CpuAgent) mine(work *Work, stop <-chan struct{}) {
	if result, err := a.engine.Seal(a.chain, work.Block, stop); result != nil {
		log.Info("Successfully sealed new block", "number", result.Number(), "hash", result.Hash())
		a.returnCh <- &Result{work, result}
	} else {
		if err != nil {
			log.Warn("Block sealing failed", "err", err)
		}
		a.returnCh <- nil
	}
}

func (a *CpuAgent) GetHashRate() int64 {
	if pow, ok := a.engine.(consensus.PoW); ok {
		return int64(pow.Hashrate())
	}
	return 0
}
