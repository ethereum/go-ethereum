// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type RemoteAgent struct {
	mu sync.Mutex

	quit     chan struct{}
	workCh   chan *Work
	returnCh chan<- *Result

	currentWork *Work
	work        map[common.Hash]*Work
}

func NewRemoteAgent() *RemoteAgent {
	agent := &RemoteAgent{work: make(map[common.Hash]*Work)}

	return agent
}

func (a *RemoteAgent) Work() chan<- *Work {
	return a.workCh
}

func (a *RemoteAgent) SetReturnCh(returnCh chan<- *Result) {
	a.returnCh = returnCh
}

func (a *RemoteAgent) Start() {
	a.quit = make(chan struct{})
	a.workCh = make(chan *Work, 1)
	go a.maintainLoop()
}

func (a *RemoteAgent) Stop() {
	close(a.quit)
	close(a.workCh)
}

func (a *RemoteAgent) GetHashRate() int64 { return 0 }

func (a *RemoteAgent) GetWork() [3]string {
	a.mu.Lock()
	defer a.mu.Unlock()

	var res [3]string

	if a.currentWork != nil {
		block := a.currentWork.Block

		res[0] = block.HashNoNonce().Hex()
		seedHash, _ := ethash.GetSeedHash(block.NumberU64())
		res[1] = common.BytesToHash(seedHash).Hex()
		// Calculate the "target" to be returned to the external miner
		n := big.NewInt(1)
		n.Lsh(n, 255)
		n.Div(n, block.Difficulty())
		n.Lsh(n, 1)
		res[2] = common.BytesToHash(n.Bytes()).Hex()

		a.work[block.HashNoNonce()] = a.currentWork
	}

	return res
}

// Returns true or false, but does not indicate if the PoW was correct
func (a *RemoteAgent) SubmitWork(nonce uint64, mixDigest, hash common.Hash) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Make sure the work submitted is present
	if a.work[hash] != nil {
		block := a.work[hash].Block.WithMiningResult(nonce, mixDigest)
		a.returnCh <- &Result{a.work[hash], block}

		delete(a.work, hash)

		return true
	} else {
		glog.V(logger.Info).Infof("Work was submitted for %x but no pending work found\n", hash)
	}

	return false
}

func (a *RemoteAgent) maintainLoop() {
	ticker := time.Tick(5 * time.Second)

out:
	for {
		select {
		case <-a.quit:
			break out
		case work := <-a.workCh:
			a.mu.Lock()
			a.currentWork = work
			a.mu.Unlock()
		case <-ticker:
			// cleanup
			a.mu.Lock()
			for hash, work := range a.work {
				if time.Since(work.createdAt) > 7*(12*time.Second) {
					delete(a.work, hash)
				}
			}
			a.mu.Unlock()
		}
	}
}
