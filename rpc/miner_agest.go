package rpc

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/miner"
)

type Agent struct {
	work        *types.Block
	currentWork *types.Block

	quit     chan struct{}
	workCh   chan *types.Block
	returnCh chan<- miner.Work
}

func NewAgent() *Agent {
	agent := &Agent{}
	go agent.run()

	return agent
}

func (a *Agent) Work() chan<- *types.Block {
	return a.workCh
}

func (a *Agent) SetWorkCh(returnCh chan<- miner.Work) {
	a.returnCh = returnCh
}

func (a *Agent) Start() {
	a.quit = make(chan struct{})
	a.workCh = make(chan *types.Block, 1)
}

func (a *Agent) Stop() {
	close(a.quit)
	close(a.workCh)
}

func (a *Agent) GetHashRate() int64 { return 0 }

func (a *Agent) run() {
out:
	for {
		select {
		case <-a.quit:
			break out
		case work := <-a.workCh:
			a.work = work
		}
	}
}

func (a *Agent) GetWork() common.Hash {
	// XXX Wait here untill work != nil ?.
	if a.work != nil {
		return a.work.HashNoNonce()
	}
	return common.Hash{}
}

func (a *Agent) SetResult(nonce uint64, mixDigest, seedHash common.Hash) {
	// Make sure the external miner was working on the right hash
	if a.currentWork != nil && a.work != nil && a.currentWork.Hash() == a.work.Hash() {
		a.returnCh <- miner.Work{a.currentWork.Number().Uint64(), nonce, mixDigest.Bytes(), seedHash.Bytes()}
	}
}
