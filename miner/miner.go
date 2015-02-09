package miner

import (
	"math/big"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/pow/ezp"
)

var minerlogger = logger.NewLogger("MINER")

type Miner struct {
	worker *worker

	MinAcceptedGasPrice *big.Int
	Extra               string

	coinbase []byte
	mining   bool
}

func New(coinbase []byte, eth *eth.Ethereum) *Miner {
	miner := &Miner{
		coinbase: coinbase,
		worker:   newWorker(coinbase, eth),
	}

	for i := 0; i < 4; i++ {
		miner.worker.register(NewCpuMiner(i, ezp.New()))
	}

	return miner
}

func (self *Miner) Mining() bool {
	return self.mining
}

func (self *Miner) Start() {
	self.worker.start()

	self.worker.commitNewWork()

	/*
		timer := time.NewTicker(time.Second)
		for {
			select {
			case <-timer.C:
				fmt.Printf("%d workers. %d/Khash\n", len(self.worker.agents), self.HashRate())
			}
		}
	*/
}

func (self *Miner) Stop() {
	self.worker.stop()
}

func (self *Miner) HashRate() int64 {
	var tot int64
	for _, agent := range self.worker.agents {
		tot += agent.Pow().GetHashrate()
	}

	return tot
}
