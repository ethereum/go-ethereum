package miner

import (
	"math/big"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/pow"
)

var minerlogger = logger.NewLogger("MINER")

type Miner struct {
	worker *worker

	MinAcceptedGasPrice *big.Int
	Extra               string

	mining bool
	eth    core.Backend
	pow    pow.PoW
}

func New(eth core.Backend, pow pow.PoW, minerThreads int) *Miner {
	// note: minerThreads is currently ignored because
	// ethash is not thread safe.
	return &Miner{eth: eth, pow: pow}
}

func (self *Miner) Mining() bool {
	return self.mining
}

func (self *Miner) Start(coinbase []byte) {
	self.mining = true
	self.worker = newWorker(coinbase, self.eth)
	self.worker.register(NewCpuMiner(0, self.pow))

	self.pow.(*ethash.Ethash).UpdateDAG()

	self.worker.start()
	self.worker.commitNewWork()
}

func (self *Miner) Stop() {
	self.mining = false
	self.worker.stop()

	//self.pow.(*ethash.Ethash).Stop()
}

func (self *Miner) HashRate() int64 {
	return self.worker.HashRate()
}
