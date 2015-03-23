package miner

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/ethereum/go-ethereum/state"
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
	SetWorkCh(chan<- Work)
	Stop()
	Start()
	GetHashRate() int64
}

type worker struct {
	mu     sync.Mutex
	agents []Agent
	recv   chan Work
	mux    *event.TypeMux
	quit   chan struct{}
	pow    pow.PoW

	eth      core.Backend
	chain    *core.ChainManager
	proc     *core.BlockProcessor
	coinbase common.Address

	current *environment

	uncleMu        sync.Mutex
	possibleUncles map[common.Hash]*types.Block

	mining bool
}

func newWorker(coinbase common.Address, eth core.Backend) *worker {
	return &worker{
		eth:            eth,
		mux:            eth.EventMux(),
		recv:           make(chan Work),
		chain:          eth.ChainManager(),
		proc:           eth.BlockProcessor(),
		possibleUncles: make(map[common.Hash]*types.Block),
		coinbase:       coinbase,
	}
}

func (self *worker) start() {
	self.mining = true

	self.quit = make(chan struct{})

	// spin up agents
	for _, agent := range self.agents {
		agent.Start()
	}

	go self.update()
	go self.wait()
}

func (self *worker) stop() {
	self.mining = false

	close(self.quit)
}

func (self *worker) register(agent Agent) {
	self.agents = append(self.agents, agent)
	agent.SetWorkCh(self.recv)
}

func (self *worker) update() {
	events := self.mux.Subscribe(core.ChainHeadEvent{}, core.NewMinedBlockEvent{}, core.ChainSideEvent{})

	timer := time.NewTicker(2 * time.Second)

out:
	for {
		select {
		case event := <-events.Chan():
			switch ev := event.(type) {
			case core.ChainHeadEvent:
				if self.current.block != ev.Block {
					self.commitNewWork()
				}
			case core.NewMinedBlockEvent:
				self.commitNewWork()
			case core.ChainSplitEvent:
				self.uncleMu.Lock()
				self.possibleUncles[ev.Block.Hash()] = ev.Block
				self.uncleMu.Unlock()
			}
		case <-self.quit:
			// stop all agents
			for _, agent := range self.agents {
				agent.Stop()
			}
			break out
		case <-timer.C:
			minerlogger.Infoln("Hash rate:", self.HashRate(), "Khash")
		}
	}

	events.Unsubscribe()
}

func (self *worker) addUncle(uncle *types.Block) {
}

func (self *worker) wait() {
	for {
		for work := range self.recv {
			// Someone Successfully Mined!
			block := self.current.block
			if block.Number().Uint64() == work.Number && block.Nonce() == 0 {
				self.current.block.SetNonce(work.Nonce)
				self.current.block.Header().MixDigest = common.BytesToHash(work.MixDigest)

				jsonlogger.LogJson(&logger.EthMinerNewBlock{
					BlockHash:     block.Hash().Hex(),
					BlockNumber:   block.Number(),
					ChainHeadHash: block.ParentHeaderHash.Hex(),
					BlockPrevHash: block.ParentHeaderHash.Hex(),
				})

				if err := self.chain.InsertChain(types.Blocks{self.current.block}); err == nil {
					self.mux.Post(core.NewMinedBlockEvent{self.current.block})
				} else {
					self.commitNewWork()
				}
			}
			break
		}
	}
}

func (self *worker) push() {
	if self.mining {
		self.current.block.Header().GasUsed = self.current.totalUsedGas
		self.current.block.SetRoot(self.current.state.Root())

		// push new work to agents
		for _, agent := range self.agents {
			agent.Work() <- self.current.block
		}
	}
}

func (self *worker) commitNewWork() {
	self.mu.Lock()
	defer self.mu.Unlock()

	block := self.chain.NewBlock(self.coinbase)

	self.current = env(block, self.eth)
	for _, ancestor := range self.chain.GetAncestors(block, 7) {
		self.current.family.Add(ancestor.Hash())
	}

	parent := self.chain.GetBlock(self.current.block.ParentHash())
	self.current.coinbase.SetGasPool(core.CalcGasLimit(parent, self.current.block))

	transactions := self.eth.TxPool().GetTransactions()
	sort.Sort(types.TxByNonce{transactions})

	minerlogger.Infof("committing new work with %d txs\n", len(transactions))
	// Keep track of transactions which return errors so they can be removed
	var remove types.Transactions
gasLimit:
	for i, tx := range transactions {
		err := self.commitTransaction(tx)
		switch {
		case core.IsNonceErr(err):
			fallthrough
		case core.IsInvalidTxErr(err):
			// Remove invalid transactions
			from, _ := tx.From()
			self.chain.TxState().RemoveNonce(from, tx.Nonce())
			remove = append(remove, tx)
			minerlogger.Infof("TX (%x) failed, will be removed: %v\n", tx.Hash().Bytes()[:4], err)
			minerlogger.Infoln(tx)
		case state.IsGasLimitErr(err):
			minerlogger.Infof("Gas limit reached for block. %d TXs included in this block\n", i)
			// Break on gas limit
			break gasLimit
		}
	}
	self.eth.TxPool().RemoveSet(remove)

	ucount := 0
	for hash, uncle := range self.possibleUncles {
		if ucount == 2 {
			break
		}

		if err := self.commitUncle(uncle.Header()); err != nil {
			minerlogger.Infof("Bad uncle found and will be removed (%x)\n", hash[:4])
			minerlogger.Debugln(uncle)
		} else {
			minerlogger.Infof("Commiting %x as uncle\n", hash[:4])
			ucount++
		}
	}
	minerlogger.Infoln("Included %d uncle(s)")

	self.current.state.AddBalance(self.coinbase, core.BlockReward)

	self.current.state.Update(common.Big0)
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

	uncleAccount := self.current.state.GetAccount(uncle.Coinbase)
	uncleAccount.AddBalance(uncleReward)

	self.current.coinbase.AddBalance(uncleReward)

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
