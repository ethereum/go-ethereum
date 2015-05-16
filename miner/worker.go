package miner

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"

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

// Work holds the current work
type Work struct {
	Number    uint64
	Nonce     uint64
	MixDigest []byte
	SeedHash  []byte
}

// Agent can register themself with the worker
type Agent interface {
	Work() chan<- *types.Block
	SetReturnCh(chan<- *types.Block)
	Stop()
	Start()
	GetHashRate() int64
}

// environment is the workers current environment and holds
// all of the current state information
type environment struct {
	totalUsedGas       *big.Int           // total gas usage in the cycle
	state              *state.StateDB     // apply state changes here
	coinbase           *state.StateObject // the miner's account
	block              *types.Block       // the new block
	ancestors          *set.Set           // ancestor set (used for checking uncle parent validity)
	family             *set.Set           // family set (used for checking uncle invalidity)
	uncles             *set.Set           // uncle set
	remove             *set.Set           // tx which will be removed
	tcount             int                // tx count in cycle
	ignoredTransactors *set.Set
	lowGasTransactors  *set.Set
	ownedAccounts      *set.Set
	lowGasTxs          types.Transactions
}

// env returns a new environment for the current cycle
func env(block *types.Block, eth core.Backend) *environment {
	state := state.New(block.Root(), eth.StateDb())
	env := &environment{
		totalUsedGas: new(big.Int),
		state:        state,
		block:        block,
		ancestors:    set.New(),
		family:       set.New(),
		uncles:       set.New(),
		coinbase:     state.GetOrNewStateObject(block.Coinbase()),
	}

	return env
}

// worker is the main object which takes care of applying messages to the new state
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
	gasPrice *big.Int
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
		gasPrice:       new(big.Int),
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
					self.mu.Lock()
					self.commitTransactions(types.Transactions{ev.Tx})
					self.mu.Unlock()
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

			if _, err := self.chain.InsertChain(types.Blocks{block}); err == nil {
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

	current := env(block, self.eth)
	for _, ancestor := range self.chain.GetAncestors(block, 7) {
		for _, uncle := range ancestor.Uncles() {
			current.family.Add(uncle.Hash())
		}
		current.family.Add(ancestor.Hash())
		current.ancestors.Add(ancestor.Hash())
	}
	accounts, _ := self.eth.AccountManager().Accounts()
	// Keep track of transactions which return errors so they can be removed
	current.remove = set.New()
	current.tcount = 0
	current.ignoredTransactors = set.New()
	current.lowGasTransactors = set.New()
	current.ownedAccounts = accountAddressesSet(accounts)

	parent := self.chain.GetBlock(current.block.ParentHash())
	current.coinbase.SetGasPool(core.CalcGasLimit(parent))

	self.current = current
}

func (w *worker) setGasPrice(p *big.Int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// calculate the minimal gas price the miner accepts when sorting out transactions.
	const pct = int64(90)
	w.gasPrice = gasprice(p, pct)

	w.mux.Post(core.GasPriceChanged{w.gasPrice})
}

func (self *worker) commitNewWork() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.uncleMu.Lock()
	defer self.uncleMu.Unlock()
	self.currentMu.Lock()
	defer self.currentMu.Unlock()

	self.makeCurrent()
	current := self.current

	transactions := self.eth.TxPool().GetTransactions()
	sort.Sort(types.TxByNonce{transactions})

	// commit transactions for this run
	self.commitTransactions(transactions)
	self.eth.TxPool().RemoveTransactions(current.lowGasTxs)

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
		glog.V(logger.Info).Infof("commit new work on block %v with %d txs & %d uncles\n", current.block.Number(), current.tcount, len(uncles))
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

	if !self.current.ancestors.Has(uncle.ParentHash) {
		return core.UncleError(fmt.Sprintf("Uncle's parent unknown (%x)", uncle.ParentHash[0:4]))
	}

	if self.current.family.Has(uncle.Hash()) {
		return core.UncleError(fmt.Sprintf("Uncle already in family (%x)", uncle.Hash()))
	}

	return nil
}

func (self *worker) commitTransactions(transactions types.Transactions) {
	current := self.current

	for _, tx := range transactions {
		// We can skip err. It has already been validated in the tx pool
		from, _ := tx.From()

		// Check if it falls within margin. Txs from owned accounts are always processed.
		if tx.GasPrice().Cmp(self.gasPrice) < 0 && !current.ownedAccounts.Has(from) {
			// ignore the transaction and transactor. We ignore the transactor
			// because nonce will fail after ignoring this transaction so there's
			// no point
			current.lowGasTransactors.Add(from)

			glog.V(logger.Info).Infof("transaction(%x) below gas price (tx=%v ask=%v). All sequential txs from this address(%x) will be ignored\n", tx.Hash().Bytes()[:4], common.CurrencyToString(tx.GasPrice()), common.CurrencyToString(self.gasPrice), from[:4])
		}

		// Continue with the next transaction if the transaction sender is included in
		// the low gas tx set. This will also remove the tx and all sequential transaction
		// from this transactor
		if current.lowGasTransactors.Has(from) {
			// add tx to the low gas set. This will be removed at the end of the run
			// owned accounts are ignored
			if !current.ownedAccounts.Has(from) {
				current.lowGasTxs = append(current.lowGasTxs, tx)
			}
			continue
		}

		// Move on to the next transaction when the transactor is in ignored transactions set
		// This may occur when a transaction hits the gas limit. When a gas limit is hit and
		// the transaction is processed (that could potentially be included in the block) it
		// will throw a nonce error because the previous transaction hasn't been processed.
		// Therefor we need to ignore any transaction after the ignored one.
		if current.ignoredTransactors.Has(from) {
			continue
		}

		self.current.state.StartRecord(tx.Hash(), common.Hash{}, 0)

		err := self.commitTransaction(tx)
		switch {
		case core.IsNonceErr(err) || core.IsInvalidTxErr(err):
			// Remove invalid transactions
			from, _ := tx.From()

			self.chain.TxState().RemoveNonce(from, tx.Nonce())
			current.remove.Add(tx.Hash())

			if glog.V(logger.Detail) {
				glog.Infof("TX (%x) failed, will be removed: %v\n", tx.Hash().Bytes()[:4], err)
			}
		case state.IsGasLimitErr(err):
			from, _ := tx.From()
			// ignore the transactor so no nonce errors will be thrown for this account
			// next time the worker is run, they'll be picked up again.
			current.ignoredTransactors.Add(from)

			glog.V(logger.Detail).Infof("Gas limit reached for (%x) in this block. Continue to try smaller txs\n", from[:4])
		default:
			current.tcount++
		}
	}
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
