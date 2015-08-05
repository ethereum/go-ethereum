// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package miner

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
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

const (
	resultQueueSize  = 10
	miningLogAtDepth = 5
)

// Agent can register themself with the worker
type Agent interface {
	Work() chan<- *Work
	SetReturnCh(chan<- *Result)
	Stop()
	Start()
	GetHashRate() int64
}

type uint64RingBuffer struct {
	ints []uint64 //array of all integers in buffer
	next int      //where is the next insertion? assert 0 <= next < len(ints)
}

// environment is the workers current environment and holds
// all of the current state information
type Work struct {
	state              *state.StateDB     // apply state changes here
	coinbase           *state.StateObject // the miner's account
	ancestors          *set.Set           // ancestor set (used for checking uncle parent validity)
	family             *set.Set           // family set (used for checking uncle invalidity)
	uncles             *set.Set           // uncle set
	remove             *set.Set           // tx which will be removed
	tcount             int                // tx count in cycle
	ignoredTransactors *set.Set
	lowGasTransactors  *set.Set
	ownedAccounts      *set.Set
	lowGasTxs          types.Transactions
	localMinedBlocks   *uint64RingBuffer // the most recent block numbers that were mined locally (used to check block inclusion)

	Block *types.Block // the new block

	header   *types.Header
	txs      []*types.Transaction
	receipts []*types.Receipt

	createdAt time.Time
}

type Result struct {
	Work  *Work
	Block *types.Block
}

// worker is the main object which takes care of applying messages to the new state
type worker struct {
	mu sync.Mutex

	agents []Agent
	recv   chan *Result
	mux    *event.TypeMux
	quit   chan struct{}
	pow    pow.PoW

	eth     core.Backend
	chain   *core.ChainManager
	proc    *core.BlockProcessor
	extraDb common.Database

	coinbase common.Address
	gasPrice *big.Int
	extra    []byte

	currentMu sync.Mutex
	current   *Work

	uncleMu        sync.Mutex
	possibleUncles map[common.Hash]*types.Block

	txQueueMu sync.Mutex
	txQueue   map[common.Hash]*types.Transaction

	// atomic status counters
	mining int32
	atWork int32

	fullValidation bool
}

func newWorker(coinbase common.Address, eth core.Backend) *worker {
	worker := &worker{
		eth:            eth,
		mux:            eth.EventMux(),
		extraDb:        eth.ExtraDb(),
		recv:           make(chan *Result, resultQueueSize),
		gasPrice:       new(big.Int),
		chain:          eth.ChainManager(),
		proc:           eth.BlockProcessor(),
		possibleUncles: make(map[common.Hash]*types.Block),
		coinbase:       coinbase,
		txQueue:        make(map[common.Hash]*types.Transaction),
		quit:           make(chan struct{}),
		fullValidation: false,
	}
	go worker.update()
	go worker.wait()

	worker.commitNewWork()

	return worker
}

func (self *worker) setEtherbase(addr common.Address) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.coinbase = addr
}

func (self *worker) pendingState() *state.StateDB {
	self.currentMu.Lock()
	defer self.currentMu.Unlock()
	return self.current.state
}

func (self *worker) pendingBlock() *types.Block {
	self.currentMu.Lock()
	defer self.currentMu.Unlock()

	if atomic.LoadInt32(&self.mining) == 0 {
		return types.NewBlock(
			self.current.header,
			self.current.txs,
			nil,
			self.current.receipts,
		)
	}
	return self.current.Block
}

func (self *worker) start() {
	self.mu.Lock()
	defer self.mu.Unlock()

	atomic.StoreInt32(&self.mining, 1)

	// spin up agents
	for _, agent := range self.agents {
		agent.Start()
	}
}

func (self *worker) stop() {
	self.mu.Lock()
	defer self.mu.Unlock()

	if atomic.LoadInt32(&self.mining) == 1 {
		var keep []Agent
		// stop all agents
		for _, agent := range self.agents {
			agent.Stop()
			// keep all that's not a cpu agent
			if _, ok := agent.(*CpuAgent); !ok {
				keep = append(keep, agent)
			}
		}
		self.agents = keep
	}

	atomic.StoreInt32(&self.mining, 0)
	atomic.StoreInt32(&self.atWork, 0)
}

func (self *worker) register(agent Agent) {
	self.mu.Lock()
	defer self.mu.Unlock()
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
				// Apply transaction to the pending state if we're not mining
				if atomic.LoadInt32(&self.mining) == 0 {
					self.currentMu.Lock()
					self.current.commitTransactions(types.Transactions{ev.Tx}, self.gasPrice, self.proc)
					self.currentMu.Unlock()
				}
			}
		case <-self.quit:
			break out
		}
	}

	events.Unsubscribe()
}

func newLocalMinedBlock(blockNumber uint64, prevMinedBlocks *uint64RingBuffer) (minedBlocks *uint64RingBuffer) {
	if prevMinedBlocks == nil {
		minedBlocks = &uint64RingBuffer{next: 0, ints: make([]uint64, miningLogAtDepth+1)}
	} else {
		minedBlocks = prevMinedBlocks
	}

	minedBlocks.ints[minedBlocks.next] = blockNumber
	minedBlocks.next = (minedBlocks.next + 1) % len(minedBlocks.ints)
	return minedBlocks
}

func (self *worker) wait() {
	for {
		for result := range self.recv {
			atomic.AddInt32(&self.atWork, -1)

			if result == nil {
				continue
			}
			block := result.Block
			work := result.Work

			work.state.Sync()
			if self.fullValidation {
				if _, err := self.chain.InsertChain(types.Blocks{block}); err != nil {
					glog.V(logger.Error).Infoln("mining err", err)
					continue
				}
				go self.mux.Post(core.NewMinedBlockEvent{block})
			} else {
				parent := self.chain.GetBlock(block.ParentHash())
				if parent == nil {
					glog.V(logger.Error).Infoln("Invalid block found during mining")
					continue
				}
				if err := core.ValidateHeader(self.eth.BlockProcessor().Pow, block.Header(), parent, true); err != nil && err != core.BlockFutureErr {
					glog.V(logger.Error).Infoln("Invalid header on mined block:", err)
					continue
				}

				stat, err := self.chain.WriteBlock(block, false)
				if err != nil {
					glog.V(logger.Error).Infoln("error writing block to chain", err)
					continue
				}
				// check if canon block and write transactions
				if stat == core.CanonStatTy {
					// This puts transactions in a extra db for rpc
					core.PutTransactions(self.extraDb, block, block.Transactions())
					// store the receipts
					core.PutReceipts(self.extraDb, work.receipts)
				}

				// broadcast before waiting for validation
				go func(block *types.Block, logs state.Logs) {
					self.mux.Post(core.NewMinedBlockEvent{block})
					self.mux.Post(core.ChainEvent{block, block.Hash(), logs})
					if stat == core.CanonStatTy {
						self.mux.Post(core.ChainHeadEvent{block})
						self.mux.Post(logs)
					}
				}(block, work.state.Logs())
			}

			// check staleness and display confirmation
			var stale, confirm string
			canonBlock := self.chain.GetBlockByNumber(block.NumberU64())
			if canonBlock != nil && canonBlock.Hash() != block.Hash() {
				stale = "stale "
			} else {
				confirm = "Wait 5 blocks for confirmation"
				work.localMinedBlocks = newLocalMinedBlock(block.Number().Uint64(), work.localMinedBlocks)
			}
			glog.V(logger.Info).Infof("🔨  Mined %sblock (#%v / %x). %s", stale, block.Number(), block.Hash().Bytes()[:4], confirm)

			self.commitNewWork()
		}
	}
}

func (self *worker) push(work *Work) {
	if atomic.LoadInt32(&self.mining) == 1 {
		if core.Canary(work.state) {
			glog.Infoln("Toxicity levels rising to deadly levels. Your canary has died. You can go back or continue down the mineshaft --more--")
			glog.Infoln("You turn back and abort mining")
			return
		}

		// push new work to agents
		for _, agent := range self.agents {
			atomic.AddInt32(&self.atWork, 1)

			if agent.Work() != nil {
				agent.Work() <- work
			}
		}
	}
}

// makeCurrent creates a new environment for the current cycle.
func (self *worker) makeCurrent(parent *types.Block, header *types.Header) {
	state := state.New(parent.Root(), self.eth.StateDb())
	work := &Work{
		state:     state,
		ancestors: set.New(),
		family:    set.New(),
		uncles:    set.New(),
		header:    header,
		coinbase:  state.GetOrNewStateObject(self.coinbase),
		createdAt: time.Now(),
	}

	// when 08 is processed ancestors contain 07 (quick block)
	for _, ancestor := range self.chain.GetBlocksFromHash(parent.Hash(), 7) {
		for _, uncle := range ancestor.Uncles() {
			work.family.Add(uncle.Hash())
		}
		work.family.Add(ancestor.Hash())
		work.ancestors.Add(ancestor.Hash())
	}
	accounts, _ := self.eth.AccountManager().Accounts()

	// Keep track of transactions which return errors so they can be removed
	work.remove = set.New()
	work.tcount = 0
	work.ignoredTransactors = set.New()
	work.lowGasTransactors = set.New()
	work.ownedAccounts = accountAddressesSet(accounts)
	if self.current != nil {
		work.localMinedBlocks = self.current.localMinedBlocks
	}
	self.current = work
}

func (w *worker) setGasPrice(p *big.Int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// calculate the minimal gas price the miner accepts when sorting out transactions.
	const pct = int64(90)
	w.gasPrice = gasprice(p, pct)

	w.mux.Post(core.GasPriceChanged{w.gasPrice})
}

func (self *worker) isBlockLocallyMined(current *Work, deepBlockNum uint64) bool {
	//Did this instance mine a block at {deepBlockNum} ?
	var isLocal = false
	for idx, blockNum := range current.localMinedBlocks.ints {
		if deepBlockNum == blockNum {
			isLocal = true
			current.localMinedBlocks.ints[idx] = 0 //prevent showing duplicate logs
			break
		}
	}
	//Short-circuit on false, because the previous and following tests must both be true
	if !isLocal {
		return false
	}

	//Does the block at {deepBlockNum} send earnings to my coinbase?
	var block = self.chain.GetBlockByNumber(deepBlockNum)
	return block != nil && block.Coinbase() == self.coinbase
}

func (self *worker) logLocalMinedBlocks(current, previous *Work) {
	if previous != nil && current.localMinedBlocks != nil {
		nextBlockNum := current.Block.NumberU64()
		for checkBlockNum := previous.Block.NumberU64(); checkBlockNum < nextBlockNum; checkBlockNum++ {
			inspectBlockNum := checkBlockNum - miningLogAtDepth
			if self.isBlockLocallyMined(current, inspectBlockNum) {
				glog.V(logger.Info).Infof("🔨 🔗  Mined %d blocks back: block #%v", miningLogAtDepth, inspectBlockNum)
			}
		}
	}
}

func (self *worker) commitNewWork() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.uncleMu.Lock()
	defer self.uncleMu.Unlock()
	self.currentMu.Lock()
	defer self.currentMu.Unlock()

	tstart := time.Now()
	parent := self.chain.CurrentBlock()
	tstamp := tstart.Unix()
	if tstamp <= int64(parent.Time()) {
		tstamp = int64(parent.Time()) + 1
	}
	// this will ensure we're not going off too far in the future
	if now := time.Now().Unix(); tstamp > now+4 {
		wait := time.Duration(tstamp-now) * time.Second
		glog.V(logger.Info).Infoln("We are too far in the future. Waiting for", wait)
		time.Sleep(wait)
	}

	num := parent.Number()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     num.Add(num, common.Big1),
		Difficulty: core.CalcDifficulty(uint64(tstamp), parent.Time(), parent.Number(), parent.Difficulty()),
		GasLimit:   core.CalcGasLimit(parent),
		GasUsed:    new(big.Int),
		Coinbase:   self.coinbase,
		Extra:      self.extra,
		Time:       uint64(tstamp),
	}

	previous := self.current
	self.makeCurrent(parent, header)
	work := self.current

	// commit transactions for this run.
	transactions := self.eth.TxPool().GetTransactions()
	sort.Sort(types.TxByNonce{transactions})
	work.coinbase.SetGasLimit(header.GasLimit)
	work.commitTransactions(transactions, self.gasPrice, self.proc)
	self.eth.TxPool().RemoveTransactions(work.lowGasTxs)

	// compute uncles for the new block.
	var (
		uncles    []*types.Header
		badUncles []common.Hash
	)
	for hash, uncle := range self.possibleUncles {
		if len(uncles) == 2 {
			break
		}
		if err := self.commitUncle(work, uncle.Header()); err != nil {
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
	for _, hash := range badUncles {
		delete(self.possibleUncles, hash)
	}

	if atomic.LoadInt32(&self.mining) == 1 {
		// commit state root after all state transitions.
		core.AccumulateRewards(work.state, header, uncles)
		work.state.SyncObjects()
		header.Root = work.state.Root()
	}

	// create the new block whose nonce will be mined.
	work.Block = types.NewBlock(header, work.txs, uncles, work.receipts)
	work.Block.Td = new(big.Int).Set(core.CalcTD(work.Block, self.chain.GetBlock(work.Block.ParentHash())))

	// We only care about logging if we're actually mining.
	if atomic.LoadInt32(&self.mining) == 1 {
		glog.V(logger.Info).Infof("commit new work on block %v with %d txs & %d uncles. Took %v\n", work.Block.Number(), work.tcount, len(uncles), time.Since(tstart))
		self.logLocalMinedBlocks(work, previous)
	}

	self.push(work)
}

func (self *worker) commitUncle(work *Work, uncle *types.Header) error {
	hash := uncle.Hash()
	if work.uncles.Has(hash) {
		return core.UncleError("Uncle not unique")
	}
	if !work.ancestors.Has(uncle.ParentHash) {
		return core.UncleError(fmt.Sprintf("Uncle's parent unknown (%x)", uncle.ParentHash[0:4]))
	}
	if work.family.Has(hash) {
		return core.UncleError(fmt.Sprintf("Uncle already in family (%x)", hash))
	}
	work.uncles.Add(uncle.Hash())
	return nil
}

func (env *Work) commitTransactions(transactions types.Transactions, gasPrice *big.Int, proc *core.BlockProcessor) {
	for _, tx := range transactions {
		// We can skip err. It has already been validated in the tx pool
		from, _ := tx.From()

		// Check if it falls within margin. Txs from owned accounts are always processed.
		if tx.GasPrice().Cmp(gasPrice) < 0 && !env.ownedAccounts.Has(from) {
			// ignore the transaction and transactor. We ignore the transactor
			// because nonce will fail after ignoring this transaction so there's
			// no point
			env.lowGasTransactors.Add(from)

			glog.V(logger.Info).Infof("transaction(%x) below gas price (tx=%v ask=%v). All sequential txs from this address(%x) will be ignored\n", tx.Hash().Bytes()[:4], common.CurrencyToString(tx.GasPrice()), common.CurrencyToString(gasPrice), from[:4])
		}

		// Continue with the next transaction if the transaction sender is included in
		// the low gas tx set. This will also remove the tx and all sequential transaction
		// from this transactor
		if env.lowGasTransactors.Has(from) {
			// add tx to the low gas set. This will be removed at the end of the run
			// owned accounts are ignored
			if !env.ownedAccounts.Has(from) {
				env.lowGasTxs = append(env.lowGasTxs, tx)
			}
			continue
		}

		// Move on to the next transaction when the transactor is in ignored transactions set
		// This may occur when a transaction hits the gas limit. When a gas limit is hit and
		// the transaction is processed (that could potentially be included in the block) it
		// will throw a nonce error because the previous transaction hasn't been processed.
		// Therefor we need to ignore any transaction after the ignored one.
		if env.ignoredTransactors.Has(from) {
			continue
		}

		env.state.StartRecord(tx.Hash(), common.Hash{}, 0)

		err := env.commitTransaction(tx, proc)
		switch {
		case state.IsGasLimitErr(err):
			// ignore the transactor so no nonce errors will be thrown for this account
			// next time the worker is run, they'll be picked up again.
			env.ignoredTransactors.Add(from)

			glog.V(logger.Detail).Infof("Gas limit reached for (%x) in this block. Continue to try smaller txs\n", from[:4])
		case err != nil:
			env.remove.Add(tx.Hash())

			if glog.V(logger.Detail) {
				glog.Infof("TX (%x) failed, will be removed: %v\n", tx.Hash().Bytes()[:4], err)
			}
		default:
			env.tcount++
		}
	}
}

func (env *Work) commitTransaction(tx *types.Transaction, proc *core.BlockProcessor) error {
	snap := env.state.Copy()
	receipt, _, err := proc.ApplyTransaction(env.coinbase, env.state, env.header, tx, env.header.GasUsed, true)
	if err != nil {
		env.state.Set(snap)
		return err
	}
	env.txs = append(env.txs, tx)
	env.receipts = append(env.receipts, receipt)
	return nil
}

// TODO: remove or use
func (self *worker) HashRate() int64 {
	return 0
}

// gasprice calculates a reduced gas price based on the pct
// XXX Use big.Rat?
func gasprice(price *big.Int, pct int64) *big.Int {
	p := new(big.Int).Set(price)
	p.Div(p, big.NewInt(100))
	p.Mul(p, big.NewInt(pct))
	return p
}

func accountAddressesSet(accounts []accounts.Account) *set.Set {
	accountSet := set.New()
	for _, account := range accounts {
		accountSet.Add(account.Address)
	}
	return accountSet
}
