package miner

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethutil"
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
	ancestors    *set.Set
	uncles       *set.Set
}

func env(block *types.Block, eth core.Backend) *environment {
	state := state.New(block.Root(), eth.StateDb())
	env := &environment{
		totalUsedGas: new(big.Int),
		state:        state,
		block:        block,
		ancestors:    set.New(),
		uncles:       set.New(),
		coinbase:     state.GetOrNewStateObject(block.Coinbase()),
	}
	for _, ancestor := range eth.ChainManager().GetAncestors(block, 7) {
		env.ancestors.Add(string(ancestor.Hash()))
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
	Pow() pow.PoW
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
	coinbase []byte

	current *environment

	mining bool
}

func newWorker(coinbase []byte, eth core.Backend) *worker {
	return &worker{
		eth:      eth,
		mux:      eth.EventMux(),
		recv:     make(chan Work),
		chain:    eth.ChainManager(),
		proc:     eth.BlockProcessor(),
		coinbase: coinbase,
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
	events := self.mux.Subscribe(core.ChainHeadEvent{}, core.NewMinedBlockEvent{})

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

func (self *worker) wait() {
	for {
		for work := range self.recv {
			// Someone Successfully Mined!
			block := self.current.block
			if block.Number().Uint64() == work.Number && block.Nonce() == 0 {
				self.current.block.SetNonce(work.Nonce)
				self.current.block.Header().MixDigest = work.MixDigest
				self.current.block.Header().SeedHash = work.SeedHash

				jsonlogger.LogJson(&logger.EthMinerNewBlock{
					BlockHash:     ethutil.Bytes2Hex(block.Hash()),
					BlockNumber:   block.Number(),
					ChainHeadHash: ethutil.Bytes2Hex(block.ParentHeaderHash),
					BlockPrevHash: ethutil.Bytes2Hex(block.ParentHeaderHash),
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
	seednum := ethash.GetSeedBlockNum(block.NumberU64())
	block.Header().SeedHash = self.chain.GetBlockByNumber(seednum).SeedHash()

	self.current = env(block, self.eth)
	parent := self.chain.GetBlock(self.current.block.ParentHash())
	self.current.coinbase.SetGasPool(core.CalcGasLimit(parent, self.current.block))

	transactions := self.eth.TxPool().GetTransactions()
	sort.Sort(types.TxByNonce{transactions})

	minerlogger.Infof("committing new work with %d txs\n", len(transactions))
	// Keep track of transactions which return errors so they can be removed
	var remove types.Transactions
gasLimit:
	for _, tx := range transactions {
		err := self.commitTransaction(tx)
		switch {
		case core.IsNonceErr(err):
			fallthrough
		case core.IsInvalidTxErr(err):
			// Remove invalid transactions
			self.chain.TxState().RemoveNonce(tx.From(), tx.Nonce())
			remove = append(remove, tx)
		case state.IsGasLimitErr(err):
			// Break on gas limit
			break gasLimit
		}

		if err != nil {
			minerlogger.Infoln(err)
		}
	}
	self.eth.TxPool().RemoveSet(remove)

	self.current.state.AddBalance(self.coinbase, core.BlockReward)

	self.current.state.Update(ethutil.Big0)
	self.push()
}

var (
	inclusionReward = new(big.Int).Div(core.BlockReward, big.NewInt(32))
	_uncleReward    = new(big.Int).Mul(core.BlockReward, big.NewInt(15))
	uncleReward     = new(big.Int).Div(_uncleReward, big.NewInt(16))
)

func (self *worker) commitUncle(uncle *types.Header) error {
	if self.current.uncles.Has(string(uncle.Hash())) {
		// Error not unique
		return core.UncleError("Uncle not unique")
	}
	self.current.uncles.Add(string(uncle.Hash()))

	if !self.current.ancestors.Has(string(uncle.ParentHash)) {
		return core.UncleError(fmt.Sprintf("Uncle's parent unknown (%x)", uncle.ParentHash[0:4]))
	}

	if !self.pow.Verify(types.NewBlockWithHeader(uncle)) {
		return core.ValidationError("Uncle's nonce is invalid (= %x)", uncle.Nonce)
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
		tot += agent.Pow().GetHashrate()
	}

	return tot
}
