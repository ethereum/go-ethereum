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

func (agent *CpuAgent) Work() chan<- *Work            { return agent.workCh }
func (agent *CpuAgent) SetReturnCh(ch chan<- *Result) { agent.returnCh = ch }

func (agent *CpuAgent) Stop() {
	if !atomic.CompareAndSwapInt32(&agent.isMining, 1, 0) {
		return // agent already stopped
	}
	agent.stop <- struct{}{}
done:
	// Empty work channel
	for {
		select {
		case <-agent.workCh:
		default:
			break done
		}
	}
}

func (agent *CpuAgent) Start() {
	if !atomic.CompareAndSwapInt32(&agent.isMining, 0, 1) {
		return // agent already started
	}
	go agent.update()
}

func (agent *CpuAgent) update() {
out:
	for {
		select {
		case work := <-agent.workCh:
			agent.mu.Lock()
			if agent.quitCurrentOp != nil {
				close(agent.quitCurrentOp)
			}
			agent.quitCurrentOp = make(chan struct{})
			go agent.mine(work, agent.quitCurrentOp)
			agent.mu.Unlock()
		case <-agent.stop:
			agent.mu.Lock()
			if agent.quitCurrentOp != nil {
				close(agent.quitCurrentOp)
				agent.quitCurrentOp = nil
			}
			agent.mu.Unlock()
			break out
		}
	}
}

func (agent *CpuAgent) mine(work *Work, stop <-chan struct{}) {
	if result, err := agent.engine.Seal(agent.chain, work.Block, stop); result != nil {
		log.Info("Successfully sealed new block", "number", result.Number(), "hash", result.Hash())
		agent.returnCh <- &Result{work, result}
	} else {
		if err != nil {
			log.Warn("Block sealing failed", "err", err)
		}
		agent.returnCh <- nil
	}
}

func (agent *CpuAgent) GetHashRate() int64 {
	if pow, ok := agent.engine.(consensus.PoW); ok {
		return int64(pow.Hashrate())
	}
	return 0
}
