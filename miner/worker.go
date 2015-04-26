package miner

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/pow"
	"gopkg.in/fatih/set.v0"
)

var jsonlogger = logger.NewJsonLogger()

type environment struct {
	totalUsedGas *big.Int
	state        *state.StateDB
	coinbase     *state.StateObject
	block        *types.Block
	family       *set.Set
	uncles       *set.Set
}

func env(block *types.Block, eth core.Backend) *environment {
	state := state.New(block.Root(), eth.StateDb())
	env := &environment{
		totalUsedGas: new(big.Int),
		state:        state,
		block:        block,
		family:       set.New(),
		uncles:       set.New(),
		coinbase:     state.GetOrNewStateObject(block.Coinbase()),
	}

	return env
}

type Work struct {
	Number    uint64
	Nonce     uint64
	MixDigest []byte
	SeedHash  []byte
}

type Agent interface {
	Work() chan<- *types.Block
	SetReturnCh(chan<- *types.Block)
	Stop()
	Start()
	GetHashRate() int64
}

type worker struct {
	mu sync.Mutex

	agents []Agent
	recv   chan *types.Block
	mux    *event.TypeMux
	quit   chan struct{}
	pow    pow.PoW

	eth   core.Backend
	chain *core.ChainManager
	proc  *core.BlockProcessor

	coinbase common.Address
	extra    []byte

	currentMu sync.Mutex
	current   *environment

	uncleMu        sync.Mutex
	possibleUncles map[common.Hash]*types.Block

	txQueueMu sync.Mutex
	txQueue   map[common.Hash]*types.Transaction

	// atomic status counters
	mining int32
	atWork int32
}

func newWorker(coinbase common.Address, eth core.Backend) *worker {
	worker := &worker{
		eth:            eth,
		mux:            eth.EventMux(),
		recv:           make(chan *types.Block),
		chain:          eth.ChainManager(),
		proc:           eth.BlockProcessor(),
		possibleUncles: make(map[common.Hash]*types.Block),
		coinbase:       coinbase,
		txQueue:        make(map[common.Hash]*types.Transaction),
		quit:           make(chan struct{}),
	}
	go worker.update()
	go worker.wait()

	worker.commitNewWork()

	return worker
}

func (self *worker) pendingState() *state.StateDB {
	self.currentMu.Lock()
	defer self.currentMu.Unlock()

	return self.current.state
}

func (self *worker) pendingBlock() *types.Block {
	self.currentMu.Lock()
	defer self.currentMu.Unlock()

	return self.current.block
}

func (self *worker) start() {
	// spin up agents
	for _, agent := range self.agents {
		agent.Start()
	}

	atomic.StoreInt32(&self.mining, 1)
}

func (self *worker) stop() {
	if atomic.LoadInt32(&self.mining) == 1 {
		// stop all agents
		for _, agent := range self.agents {
			agent.Stop()
		}
	}

	atomic.StoreInt32(&self.mining, 0)
	atomic.StoreInt32(&self.atWork, 0)
}

func (self *worker) register(agent Agent) {
	self.agents = append(self.agents, agent)
	agent.SetReturnCh(self.recv)
}

func (self *worker) update() {
	events := self.mux.Subscribe(core.ChainHeadEvent{}, core.ChainSideEvent{}, core.TxPreEvent{})

out:
	for {
		select {
		case event := <-events.Chan():
			switch ev := event.(type) {
			case core.ChainHeadEvent:
				self.commitNewWork()
			case core.ChainSideEvent:
				self.uncleMu.Lock()
				self.possibleUncles[ev.Block.Hash()] = ev.Block
				self.uncleMu.Unlock()
			case core.TxPreEvent:
				if atomic.LoadInt32(&self.mining) == 0 {
					self.commitNewWork()
				}
			}
		case <-self.quit:
			break out
		}
	}

	events.Unsubscribe()
}

func (self *worker) wait() {
	for {
		for block := range self.recv {
			atomic.AddInt32(&self.atWork, -1)

			if block == nil {
				continue
			}

			if err := self.chain.InsertChain(types.Blocks{block}); err == nil {
				for _, uncle := range block.Uncles() {
					delete(self.possibleUncles, uncle.Hash())
				}
				self.mux.Post(core.NewMinedBlockEvent{block})

				glog.V(logger.Info).Infof("🔨  Mined block #%v", block.Number())

				jsonlogger.LogJson(&logger.EthMinerNewBlock{
					BlockHash:     block.Hash().Hex(),
					BlockNumber:   block.Number(),
					ChainHeadHash: block.ParentHeaderHash.Hex(),
					BlockPrevHash: block.ParentHeaderHash.Hex(),
				})
			} else {
				self.commitNewWork()
			}
		}
	}
}

func (self *worker) push() {
	if atomic.LoadInt32(&self.mining) == 1 {
		self.current.block.Header().GasUsed = self.current.totalUsedGas
		self.current.block.SetRoot(self.current.state.Root())

		// push new work to agents
		for _, agent := range self.agents {
			atomic.AddInt32(&self.atWork, 1)

			if agent.Work() != nil {
				agent.Work() <- self.current.block.Copy()
			} else {
				common.Report(fmt.Sprintf("%v %T\n", agent, agent))
			}
		}
	}
}

func (self *worker) makeCurrent() {
	block := self.chain.NewBlock(self.coinbase)
	if block.Time() == self.chain.CurrentBlock().Time() {
		block.Header().Time++
	}
	block.Header().Extra = self.extra

	self.current = env(block, self.eth)
	for _, ancestor := range self.chain.GetAncestors(block, 7) {
		self.current.family.Add(ancestor.Hash())
	}

	parent := self.chain.GetBlock(self.current.block.ParentHash())
	self.current.coinbase.SetGasPool(core.CalcGasLimit(parent))
}

func (self *worker) commitNewWork() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.uncleMu.Lock()
	defer self.uncleMu.Unlock()
	self.currentMu.Lock()
	defer self.currentMu.Unlock()

	self.makeCurrent()

	transactions := self.eth.TxPool().GetTransactions()
	sort.Sort(types.TxByNonce{transactions})

	// Keep track of transactions which return errors so they can be removed
	var (
		remove             = set.New()
		tcount             = 0
		ignoredTransactors = set.New()
	)

	for _, tx := range transactions {
		// We can skip err. It has already been validated in the tx pool
		from, _ := tx.From()
		// Move on to the next transaction when the transactor is in ignored transactions set
		// This may occur when a transaction hits the gas limit. When a gas limit is hit and
		// the transaction is processed (that could potentially be included in the block) it
		// will throw a nonce error because the previous transaction hasn't been processed.
		// Therefor we need to ignore any transaction after the ignored one.
		if ignoredTransactors.Has(from) {
			continue
		}

		self.current.state.StartRecord(tx.Hash(), common.Hash{}, 0)

		err := self.commitTransaction(tx)
		switch {
		case core.IsNonceErr(err) || core.IsInvalidTxErr(err):
			// Remove invalid transactions
			from, _ := tx.From()

			self.chain.TxState().RemoveNonce(from, tx.Nonce())
			remove.Add(tx.Hash())

			if glog.V(logger.Detail) {
				glog.Infof("TX (%x) failed, will be removed: %v\n", tx.Hash().Bytes()[:4], err)
			}
		case state.IsGasLimitErr(err):
			from, _ := tx.From()
			// ignore the transactor so no nonce errors will be thrown for this account
			// next time the worker is run, they'll be picked up again.
			ignoredTransactors.Add(from)

			glog.V(logger.Detail).Infof("Gas limit reached for (%x) in this block. Continue to try smaller txs\n", from[:4])
		default:
			tcount++
		}
	}

	var (
		uncles    []*types.Header
		badUncles []common.Hash
	)
	for hash, uncle := range self.possibleUncles {
		if len(uncles) == 2 {
			break
		}

		if err := self.commitUncle(uncle.Header()); err != nil {
			if glog.V(logger.Ridiculousness) {
				glog.V(logger.Detail).Infof("Bad uncle found and will be removed (%x)\n", hash[:4])
				glog.V(logger.Detail).Infoln(uncle)
			}

			badUncles = append(badUncles, hash)
		} else {
			glog.V(logger.Debug).Infof("commiting %x as uncle\n", hash[:4])
			uncles = append(uncles, uncle.Header())
		}
	}

	// We only care about logging if we're actually mining
	if atomic.LoadInt32(&self.mining) == 1 {
		glog.V(logger.Info).Infof("commit new work on block %v with %d txs & %d uncles\n", self.current.block.Number(), tcount, len(uncles))
	}

	for _, hash := range badUncles {
		delete(self.possibleUncles, hash)
	}

	self.current.block.SetUncles(uncles)

	core.AccumulateRewards(self.current.state, self.current.block)

	self.current.state.Update()

	self.push()
}

var (
	inclusionReward = new(big.Int).Div(core.BlockReward, big.NewInt(32))
	_uncleReward    = new(big.Int).Mul(core.BlockReward, big.NewInt(15))
	uncleReward     = new(big.Int).Div(_uncleReward, big.NewInt(16))
)

func (self *worker) commitUncle(uncle *types.Header) error {
	if self.current.uncles.Has(uncle.Hash()) {
		// Error not unique
		return core.UncleError("Uncle not unique")
	}
	self.current.uncles.Add(uncle.Hash())

	if !self.current.family.Has(uncle.ParentHash) {
		return core.UncleError(fmt.Sprintf("Uncle's parent unknown (%x)", uncle.ParentHash[0:4]))
	}

	if self.current.family.Has(uncle.Hash()) {
		return core.UncleError(fmt.Sprintf("Uncle already in family (%x)", uncle.Hash()))
	}

	return nil
}

func (self *worker) commitTransaction(tx *types.Transaction) error {
	snap := self.current.state.Copy()
	receipt, _, err := self.proc.ApplyTransaction(self.current.coinbase, self.current.state, self.current.block, tx, self.current.totalUsedGas, true)
	if err != nil && (core.IsNonceErr(err) || state.IsGasLimitErr(err) || core.IsInvalidTxErr(err)) {
		self.current.state.Set(snap)
		return err
	}

	self.current.block.AddTransaction(tx)
	self.current.block.AddReceipt(receipt)

	return nil
}

func (self *worker) HashRate() int64 {
	var tot int64
	for _, agent := range self.agents {
		tot += agent.GetHashRate()
	}

	return tot
}
