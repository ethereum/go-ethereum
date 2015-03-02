package miner

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/pow/ezp"
)

var minerlogger = logger.NewLogger("MINER")

type Miner struct {
	worker *worker

	MinAcceptedGasPrice *big.Int
	Extra               string

	Coinbase []byte
	mining   bool
}

func New(coinbase []byte, eth core.Backend, minerThreads int) *Miner {
	miner := &Miner{
		Coinbase: coinbase,
		worker:   newWorker(coinbase, eth),
	}

	for i := 0; i < minerThreads; i++ {
		miner.worker.register(NewCpuMiner(i, ezp.New()))
	}

	return miner
}

func (self *Miner) Mining() bool {
	return self.mining
}

func (self *Miner) Start() {
	self.mining = true

	self.worker.start()

	self.worker.commitNewWork()
}

func (self *Miner) Stop() {
	self.mining = false

	self.worker.stop()
}

func (self *Miner) HashRate() int64 {
	return self.worker.HashRate()
}
