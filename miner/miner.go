package miner

import (
	"math/big"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/pow"
)

type Miner struct {
	worker *worker

	MinAcceptedGasPrice *big.Int

	mining bool
	eth    core.Backend
	pow    pow.PoW
}

func New(eth core.Backend, pow pow.PoW, minerThreads int) *Miner {
	// note: minerThreads is currently ignored because
	// ethash is not thread safe.
	miner := &Miner{eth: eth, pow: pow, worker: newWorker(common.Address{}, eth)}
	for i := 0; i < minerThreads; i++ {
		miner.worker.register(NewCpuMiner(i, pow))
	}
	return miner
}

func (self *Miner) Mining() bool {
	return self.mining
}

func (self *Miner) Start(coinbase common.Address) {
	self.mining = true
	self.worker.coinbase = coinbase

	self.pow.(*ethash.Ethash).UpdateDAG()

	self.worker.start()
	self.worker.commitNewWork()
}

func (self *Miner) Register(agent Agent) {
	self.worker.register(agent)
}

func (self *Miner) Stop() {
	self.mining = false
	self.worker.stop()

	//self.pow.(*ethash.Ethash).Stop()
}

func (self *Miner) HashRate() int64 {
	return self.worker.HashRate()
}

func (self *Miner) SetExtra(extra []byte) {
	self.worker.extra = extra
}
