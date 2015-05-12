package miner

import (
	"math/big"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type RemoteAgent struct {
	work        *types.Block
	currentWork *types.Block

	quit     chan struct{}
	workCh   chan *types.Block
	returnCh chan<- *types.Block
}

func NewRemoteAgent() *RemoteAgent {
	agent := &RemoteAgent{}

	return agent
}

func (a *RemoteAgent) Work() chan<- *types.Block {
	return a.workCh
}

func (a *RemoteAgent) SetReturnCh(returnCh chan<- *types.Block) {
	a.returnCh = returnCh
}

func (a *RemoteAgent) Start() {
	a.quit = make(chan struct{})
	a.workCh = make(chan *types.Block, 1)
	go a.run()
}

func (a *RemoteAgent) Stop() {
	close(a.quit)
	close(a.workCh)
}

func (a *RemoteAgent) GetHashRate() int64 { return 0 }

func (a *RemoteAgent) run() {
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

func (a *RemoteAgent) GetWork() [3]string {
	var res [3]string

	if a.work != nil {
		a.currentWork = a.work

		res[0] = a.work.HashNoNonce().Hex()
		seedHash, _ := ethash.GetSeedHash(a.currentWork.NumberU64())
		res[1] = common.BytesToHash(seedHash).Hex()
		// Calculate the "target" to be returned to the external miner
		n := big.NewInt(1)
		n.Lsh(n, 255)
		n.Div(n, a.work.Difficulty())
		n.Lsh(n, 1)
		res[2] = common.BytesToHash(n.Bytes()).Hex()
	}

	return res
}

func (a *RemoteAgent) SubmitWork(nonce uint64, mixDigest, seedHash common.Hash) bool {
	// Return true or false, but does not indicate if the PoW was correct

	// Make sure the external miner was working on the right hash
	if a.currentWork != nil && a.work != nil {
		a.currentWork.SetNonce(nonce)
		a.currentWork.Header().MixDigest = mixDigest
		a.returnCh <- a.currentWork
		//a.returnCh <- Work{a.currentWork.Number().Uint64(), nonce, mixDigest.Bytes(), seedHash.Bytes()}
		return true
	}

	return false
}
