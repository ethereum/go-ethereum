package miner

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/pow"
)

type Miner struct {
	worker *worker

	MinAcceptedGasPrice *big.Int

	threads int
	mining  bool
	eth     core.Backend
	pow     pow.PoW
}

func New(eth core.Backend, pow pow.PoW) *Miner {
	return &Miner{eth: eth, pow: pow, worker: newWorker(common.Address{}, eth)}
}

func (self *Miner) Mining() bool {
	return self.mining
}

func (m *Miner) SetGasPrice(price *big.Int) {
	// FIXME block tests set a nil gas price. Quick dirty fix
	if price == nil {
		return
	}

	m.worker.gasPrice = price
}

func (self *Miner) Start(coinbase common.Address, threads int) {

	self.mining = true

	for i := 0; i < threads; i++ {
		self.worker.register(NewCpuAgent(i, self.pow))
	}
	self.threads = threads

	glog.V(logger.Info).Infof("Starting mining operation (CPU=%d TOT=%d)\n", threads, len(self.worker.agents))

	self.worker.coinbase = coinbase
	self.worker.start()
	self.worker.commitNewWork()
}

func (self *Miner) Stop() {
	self.worker.stop()
	self.mining = false
}

func (self *Miner) Register(agent Agent) {
	if self.mining {
		agent.Start()
	}

	self.worker.register(agent)
}

func (self *Miner) HashRate() int64 {
	return self.pow.GetHashrate()
}

func (self *Miner) SetExtra(extra []byte) {
	self.worker.extra = extra
}

func (self *Miner) PendingState() *state.StateDB {
	return self.worker.pendingState()
}

func (self *Miner) PendingBlock() *types.Block {
	return self.worker.pendingBlock()
}
