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

	Coinbase []byte
	mining   bool

	pow pow.PoW
}

func New(coinbase []byte, eth core.Backend, pow pow.PoW, minerThreads int) *Miner {
	miner := &Miner{
		Coinbase: coinbase,
		worker:   newWorker(coinbase, eth),
		pow:      pow,
	}

	for i := 0; i < minerThreads; i++ {
		miner.worker.register(NewCpuMiner(i, miner.pow))
	}

	return miner
}

func (self *Miner) Mining() bool {
	return self.mining
}

func (self *Miner) Start() {
	self.mining = true

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
