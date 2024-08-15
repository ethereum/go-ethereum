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
	"bytes"
	"encoding/binary"
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/XinFinOrg/XDPoSChain/XDCx/tradingstate"
	"github.com/XinFinOrg/XDPoSChain/XDCxlending/lendingstate"
	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/misc"
	"github.com/XinFinOrg/XDPoSChain/contracts"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	mapset "github.com/deckarep/golang-set"
)

const (
	resultQueueSize  = 10
	miningLogAtDepth = 5

	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096
	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10
	// chainSideChanSize is the size of channel listening to ChainSideEvent.
	chainSideChanSize = 10

	txMatchGasLimit = 40000000
)

// Agent can register themself with the worker
type Agent interface {
	Work() chan<- *Work
	SetReturnCh(chan<- *Result)
	Stop()
	Start()
	GetHashRate() int64
}

// Work is the workers current environment and holds
// all of the current state information
type Work struct {
	config *params.ChainConfig
	signer types.Signer

	state        *state.StateDB // apply state changes here
	parentState  *state.StateDB
	tradingState *tradingstate.TradingStateDB
	lendingState *lendingstate.LendingStateDB
	ancestors    mapset.Set // ancestor set (used for checking uncle parent validity)
	family       mapset.Set // family set (used for checking uncle invalidity)
	uncles       mapset.Set // uncle set
	tcount       int        // tx count in cycle

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
	config *params.ChainConfig
	engine consensus.Engine

	mu sync.Mutex

	// Feeds
	pendingLogsFeed event.Feed

	// update loop
	mux          *event.TypeMux
	txsCh        chan core.NewTxsEvent
	txsSub       event.Subscription
	chainHeadCh  chan core.ChainHeadEvent
	chainHeadSub event.Subscription
	chainSideCh  chan core.ChainSideEvent
	chainSideSub event.Subscription
	wg           sync.WaitGroup

	agents map[Agent]struct{}
	recv   chan *Result

	eth     Backend
	chain   *core.BlockChain
	proc    core.Validator
	chainDb ethdb.Database

	coinbase common.Address
	extra    []byte

	snapshotMu       sync.RWMutex // The lock used to protect the block snapshot and state snapshot
	snapshotBlock    *types.Block
	snapshotReceipts types.Receipts
	snapshotState    *state.StateDB

	currentMu sync.Mutex
	current   *Work

	uncleMu        sync.Mutex
	possibleUncles map[common.Hash]*types.Block

	unconfirmed *unconfirmedBlocks // set of locally mined blocks pending canonicalness confirmations

	// atomic status counters
	mining                int32
	atWork                int32
	announceTxs           bool
	lastParentBlockCommit string
}

func newWorker(config *params.ChainConfig, engine consensus.Engine, coinbase common.Address, eth Backend, mux *event.TypeMux, announceTxs bool) *worker {
	worker := &worker{
		config:         config,
		engine:         engine,
		eth:            eth,
		mux:            mux,
		txsCh:          make(chan core.NewTxsEvent, txChanSize),
		chainHeadCh:    make(chan core.ChainHeadEvent, chainHeadChanSize),
		chainSideCh:    make(chan core.ChainSideEvent, chainSideChanSize),
		chainDb:        eth.ChainDb(),
		recv:           make(chan *Result, resultQueueSize),
		chain:          eth.BlockChain(),
		proc:           eth.BlockChain().Validator(),
		possibleUncles: make(map[common.Hash]*types.Block),
		coinbase:       coinbase,
		agents:         make(map[Agent]struct{}),
		unconfirmed:    newUnconfirmedBlocks(eth.BlockChain(), miningLogAtDepth),
		announceTxs:    announceTxs,
	}
	if worker.announceTxs {
		// Subscribe NewTxsEvent for tx pool
		worker.txsSub = eth.TxPool().SubscribeNewTxsEvent(worker.txsCh)
	}
	// Subscribe events for blockchain
	worker.chainHeadSub = eth.BlockChain().SubscribeChainHeadEvent(worker.chainHeadCh)
	worker.chainSideSub = eth.BlockChain().SubscribeChainSideEvent(worker.chainSideCh)
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

func (self *worker) setExtra(extra []byte) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.extra = extra
}

// pending returns the pending state and corresponding block. The returned
// values can be nil in case the pending block is not initialized.
func (w *worker) pending() (*types.Block, *state.StateDB) {
	w.snapshotMu.RLock()
	defer w.snapshotMu.RUnlock()
	if w.snapshotState == nil {
		return nil, nil
	}
	return w.snapshotBlock, w.snapshotState.Copy()
}

// pendingBlock returns pending block. The returned block can be nil in case the
// pending block is not initialized.
func (w *worker) pendingBlock() *types.Block {
	w.snapshotMu.RLock()
	defer w.snapshotMu.RUnlock()
	return w.snapshotBlock
}

// pendingBlockAndReceipts returns pending block and corresponding receipts.
func (w *worker) pendingBlockAndReceipts() (*types.Block, types.Receipts) {
	// return a snapshot to avoid contention on currentMu mutex
	w.snapshotMu.RLock()
	defer w.snapshotMu.RUnlock()
	return w.snapshotBlock, w.snapshotReceipts
}

func (self *worker) start() {
	self.mu.Lock()
	defer self.mu.Unlock()

	atomic.StoreInt32(&self.mining, 1)

	// spin up agents
	for agent := range self.agents {
		agent.Start()
	}
}

func (self *worker) stop() {
	self.wg.Wait()

	self.mu.Lock()
	defer self.mu.Unlock()
	if atomic.LoadInt32(&self.mining) == 1 {
		for agent := range self.agents {
			agent.Stop()
		}
	}
	atomic.StoreInt32(&self.mining, 0)
	atomic.StoreInt32(&self.atWork, 0)
}

func (self *worker) register(agent Agent) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.agents[agent] = struct{}{}
	agent.SetReturnCh(self.recv)
}

func (self *worker) unregister(agent Agent) {
	self.mu.Lock()
	defer self.mu.Unlock()
	delete(self.agents, agent)
	agent.Stop()
}

func (self *worker) update() {
	if self.announceTxs {
		defer self.txsSub.Unsubscribe()
	}
	defer self.chainHeadSub.Unsubscribe()
	defer self.chainSideSub.Unsubscribe()

	// timeout waiting for v1 inital value
	minePeriod := 2
	MinePeriodCh := self.engine.(*XDPoS.XDPoS).MinePeriodCh
	defer close(MinePeriodCh)

	timeout := time.NewTimer(time.Duration(minePeriod) * time.Second)
	c := make(chan struct{})
	finish := make(chan struct{})
	defer close(finish)
	defer timeout.Stop()
	go func() {
		for {
			// A real event arrived, process interesting content
			select {
			case <-timeout.C:
				c <- struct{}{}
			case <-finish:
				return
			}
		}
	}()
	for {
		// A real event arrived, process interesting content
		select {
		case v := <-MinePeriodCh:
			log.Info("[worker] update wait period", "period", v)
			minePeriod = v
			timeout.Reset(time.Duration(minePeriod) * time.Second)

		case <-c:
			if atomic.LoadInt32(&self.mining) == 1 {
				self.commitNewWork()
			}
			timeout.Reset(time.Duration(minePeriod) * time.Second)

		// Handle ChainHeadEvent
		case <-self.chainHeadCh:
			self.commitNewWork()
			timeout.Reset(time.Duration(minePeriod) * time.Second)

		// Handle ChainSideEvent
		case <-self.chainSideCh:

		// Handle NewTxsEvent
		case ev := <-self.txsCh:
			// Apply transactions to the pending state if we're not mining.
			//
			// Note all transactions received may not be continuous with transactions
			// already included in the current mining block. These transactions will
			// be automatically eliminated.
			if atomic.LoadInt32(&self.mining) == 0 {
				self.currentMu.Lock()
				txs := make(map[common.Address]types.Transactions)
				for _, tx := range ev.Txs {
					acc, _ := types.Sender(self.current.signer, tx)
					txs[acc] = append(txs[acc], tx)
				}
				feeCapacity := state.GetTRC21FeeCapacityFromState(self.current.state)
				txset, specialTxs := types.NewTransactionsByPriceAndNonce(self.current.signer, txs, nil, feeCapacity)

				tcount := self.current.tcount
				self.current.commitTransactions(self.mux, feeCapacity, txset, specialTxs, self.chain, self.coinbase, &self.pendingLogsFeed)

				// Only update the snapshot if any new transactions were added
				// to the pending block
				if tcount != self.current.tcount {
					self.updateSnapshot()
				}
				self.currentMu.Unlock()
			} else {
				// If we're mining, but nothing is being processed, wake on new transactions
				if self.config.XDPoS != nil && self.config.XDPoS.Period == 0 {
					self.commitNewWork()
				}
			}

		case <-self.chainHeadSub.Err():
			return

		case <-self.chainSideSub.Err():
			return
		}
	}
}

func (self *worker) wait() {
	for {
		mustCommitNewWork := true
		for result := range self.recv {
			atomic.AddInt32(&self.atWork, -1)

			if result == nil {
				continue
			}
			block := result.Block
			if self.config.XDPoS != nil && block.NumberU64() >= self.config.XDPoS.Epoch && len(block.Validator()) == 0 {
				self.mux.Post(core.NewMinedBlockEvent{Block: block})
				continue
			}
			work := result.Work

			// Different block could share same sealhash, deep copy here to prevent write-write conflict.
			hash := block.Hash()
			receipts := make([]*types.Receipt, len(work.receipts))
			for i, receipt := range work.receipts {
				// add block location fields
				receipt.BlockHash = hash
				receipt.BlockNumber = block.Number()
				receipt.TransactionIndex = uint(i)

				receipts[i] = new(types.Receipt)
				*receipts[i] = *receipt
			}
			// Update the block hash in all logs since it is now available and not when the
			// receipt/log of individual transactions were created.
			for _, log := range work.state.Logs() {
				log.BlockHash = hash
			}
			// Commit block and state to database.
			self.currentMu.Lock()
			stat, err := self.chain.WriteBlockWithState(block, receipts, work.state, work.tradingState, work.lendingState)
			self.currentMu.Unlock()
			if err != nil {
				log.Error("Failed writing block to chain", "err", err)
				continue
			}
			// check if canon block and write transactions
			if stat == core.CanonStatTy {
				// implicit by posting ChainHeadEvent
				mustCommitNewWork = false
			}
			// Broadcast the block and announce chain insertion event
			self.mux.Post(core.NewMinedBlockEvent{Block: block})
			var (
				events []interface{}
				logs   = work.state.Logs()
			)
			events = append(events, core.ChainEvent{Block: block, Hash: block.Hash(), Logs: logs})
			if stat == core.CanonStatTy {
				events = append(events, core.ChainHeadEvent{Block: block})
			}
			if work.config.XDPoS != nil {
				// epoch block
				isEpochSwitchBlock, _, err := self.engine.(*XDPoS.XDPoS).IsEpochSwitch(block.Header())
				if err != nil {
					log.Error("[wait] fail to check if block is epoch switch block when worker waiting", "BlockNum", block.Number(), "Hash", block.Hash())
				}
				if isEpochSwitchBlock {
					core.CheckpointCh <- 1
				}
			}
			self.chain.UpdateBlocksHashCache(block)
			self.chain.PostChainEvents(events, logs)

			// Insert the block into the set of pending ones to wait for confirmations
			self.unconfirmed.Insert(block.NumberU64(), block.Hash())

			if mustCommitNewWork {
				self.commitNewWork()
			}

			if self.config.XDPoS != nil {
				c := self.engine.(*XDPoS.XDPoS)
				err = c.HandleProposedBlock(self.chain, block.Header())
				if err != nil {
					log.Warn("[wait] Unable to handle new proposed block", "err", err, "number", block.Number(), "hash", block.Hash())
				}

				authorized := c.IsAuthorisedAddress(self.chain, block.Header(), self.coinbase)
				if !authorized {
					valid := false
					masternodes := c.GetMasternodes(self.chain, block.Header())
					for _, m := range masternodes {
						if m == self.coinbase {
							valid = true
							break
						}
					}
					if !valid {
						log.Error("Coinbase address not in snapshot signers.")
						return
					}
				}
				// Send tx sign to smart contract blockSigners.
				if block.NumberU64()%common.MergeSignRange == 0 || !self.config.IsTIP2019(block.Number()) {
					if err := contracts.CreateTransactionSign(self.config, self.eth.TxPool(), self.eth.AccountManager(), block, self.chainDb, self.coinbase); err != nil {
						log.Error("Fail to create tx sign for signer", "error", err)
					}
				}
			}
		}
	}
}

// push sends a new work task to currently live miner agents.
func (self *worker) push(work *Work) {
	if atomic.LoadInt32(&self.mining) != 1 {
		return
	}
	for agent := range self.agents {
		atomic.AddInt32(&self.atWork, 1)
		if ch := agent.Work(); ch != nil {
			ch <- work
		}
	}
}

// copyReceipts makes a deep copy of the given receipts.
func copyReceipts(receipts []*types.Receipt) []*types.Receipt {
	result := make([]*types.Receipt, len(receipts))
	for i, l := range receipts {
		cpy := *l
		result[i] = &cpy
	}
	return result
}

// updateSnapshot updates pending snapshot block and state.
// Note this function assumes the current variable is thread safe.
func (w *worker) updateSnapshot() {
	w.snapshotMu.Lock()
	defer w.snapshotMu.Unlock()

	w.snapshotBlock = types.NewBlock(
		w.current.header,
		w.current.txs,
		nil,
		w.current.receipts,
	)
	w.snapshotReceipts = copyReceipts(w.current.receipts)
	w.snapshotState = w.current.state.Copy()
}

// makeCurrent creates a new environment for the current cycle.
func (self *worker) makeCurrent(parent *types.Block, header *types.Header) error {
	// Retrieve the parent state to execute on top and start a prefetcher for
	// the miner to speed block sealing up a bit
	state, err := self.chain.StateAt(parent.Root())
	if err != nil {
		return err
	}

	author, _ := self.chain.Engine().Author(parent.Header())
	var XDCxState *tradingstate.TradingStateDB
	var lendingState *lendingstate.LendingStateDB
	if self.config.XDPoS != nil {
		XDCX := self.eth.GetXDCX()
		XDCxState, err = XDCX.GetTradingState(parent, author)
		if err != nil {
			log.Error("Failed to get XDCx state ", "number", parent.Number(), "err", err)
			return err
		}
		lending := self.eth.GetXDCXLending()
		lendingState, err = lending.GetLendingState(parent, author)
		if err != nil {
			log.Error("Failed to get lending state ", "number", parent.Number(), "err", err)
			return err
		}
	}

	work := &Work{
		config:       self.config,
		signer:       types.MakeSigner(self.config, header.Number),
		state:        state,
		parentState:  state.Copy(),
		tradingState: XDCxState,
		lendingState: lendingState,
		ancestors:    mapset.NewSet(),
		family:       mapset.NewSet(),
		uncles:       mapset.NewSet(),
		header:       header,
		createdAt:    time.Now(),
	}

	// Keep track of transactions which return errors so they can be removed
	work.tcount = 0
	self.current = work
	return nil
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func (self *worker) commitNewWork() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.uncleMu.Lock()
	defer self.uncleMu.Unlock()
	self.currentMu.Lock()
	defer self.currentMu.Unlock()

	tstart := time.Now()

	c := self.engine.(*XDPoS.XDPoS)
	var parent *types.Block
	if c != nil {
		parent = c.FindParentBlockToAssign(self.chain, self.chain.CurrentBlock())
	} else {
		parent = self.chain.CurrentBlock()
	}

	var signers map[common.Address]struct{}
	if parent.Hash().Hex() == self.lastParentBlockCommit {
		return
	}
	if !self.announceTxs && atomic.LoadInt32(&self.mining) == 0 {
		return
	}

	// Only try to commit new work if we are mining
	if atomic.LoadInt32(&self.mining) == 1 {
		// check if we are right after parent's coinbase in the list
		if self.config.XDPoS != nil {
			ok, err := c.YourTurn(self.chain, parent.Header(), self.coinbase)
			if err != nil {
				log.Warn("Failed when trying to commit new work", "err", err)
				return
			}
			if !ok {
				log.Info("Not my turn to commit block. Waiting...")
				return
			}
		}
	}
	tstamp := tstart.Unix()
	if parent.Time().Cmp(new(big.Int).SetInt64(tstamp)) >= 0 {
		tstamp = parent.Time().Int64() + 1
	}
	// this will ensure we're not going off too far in the future
	if now := time.Now().Unix(); tstamp > now {
		wait := time.Duration(tstamp-now) * time.Second
		log.Info("Mining too far in the future", "wait", common.PrettyDuration(wait))
		time.Sleep(wait)
	}

	num := parent.Number()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     num.Add(num, common.Big1),
		GasLimit:   params.TargetGasLimit,
		Extra:      self.extra,
		Time:       big.NewInt(tstamp),
	}
	// Only set the coinbase if we are mining (avoid spurious block rewards)
	if atomic.LoadInt32(&self.mining) == 1 {
		header.Coinbase = self.coinbase
	}

	if err := self.engine.Prepare(self.chain, header); err != nil {
		if err == consensus.ErrNotReadyToPropose {
			log.Info("Waiting...", "err", err)
			return
		}
		log.Error("Failed to prepare header for new block", "err", err)
		return
	}
	// If we are care about TheDAO hard-fork check whether to override the extra-data or not
	if daoBlock := self.config.DAOForkBlock; daoBlock != nil {
		// Check whether the block is among the fork extra-override range
		limit := new(big.Int).Add(daoBlock, params.DAOForkExtraRange)
		if header.Number.Cmp(daoBlock) >= 0 && header.Number.Cmp(limit) < 0 {
			// Depending whether we support or oppose the fork, override differently
			if self.config.DAOForkSupport {
				header.Extra = common.CopyBytes(params.DAOForkBlockExtra)
			} else if bytes.Equal(header.Extra, params.DAOForkBlockExtra) {
				header.Extra = []byte{} // If miner opposes, don't let it use the reserved extra-data
			}
		}
	}
	// Could potentially happen if starting to mine in an odd state.
	err := self.makeCurrent(parent, header)
	if err != nil {
		log.Error("Failed to create mining context", "err", err)
		return
	}
	// Create the current work task and check any fork transitions needed
	work := self.current
	if self.config.DAOForkSupport && self.config.DAOForkBlock != nil && self.config.DAOForkBlock.Cmp(header.Number) == 0 {
		misc.ApplyDAOHardFork(work.state)
	}
	if common.TIPSigning.Cmp(header.Number) == 0 {
		work.state.DeleteAddress(common.BlockSignersBinary)
	}
	// won't grasp txs at checkpoint
	var (
		txs                                                                  *types.TransactionsByPriceAndNonce
		specialTxs                                                           types.Transactions
		tradingTransaction                                                   *types.Transaction
		lendingTransaction                                                   *types.Transaction
		tradingTxMatches                                                     []tradingstate.TxDataMatch
		tradingMatchingResults                                               map[common.Hash]tradingstate.MatchingResult
		lendingMatchingResults                                               map[common.Hash]lendingstate.MatchingResult
		lendingInput                                                         []*lendingstate.LendingItem
		updatedTrades                                                        map[common.Hash]*lendingstate.LendingTrade
		liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades []*lendingstate.LendingTrade
		lendingFinalizedTradeTransaction                                     *types.Transaction
	)
	feeCapacity := state.GetTRC21FeeCapacityFromStateWithCache(parent.Root(), work.state)
	if self.config.XDPoS != nil {
		isEpochSwitchBlock, _, err := self.engine.(*XDPoS.XDPoS).IsEpochSwitch(header)
		if err != nil {
			log.Error("[commitNewWork] fail to check if block is epoch switch block when fetching pending transactions", "BlockNum", header.Number, "Hash", header.Hash())
		}
		if !isEpochSwitchBlock {
			pending, err := self.eth.TxPool().Pending()
			if err != nil {
				log.Error("Failed to fetch pending transactions", "err", err)
				return
			}
			txs, specialTxs = types.NewTransactionsByPriceAndNonce(self.current.signer, pending, signers, feeCapacity)
		}
	}
	if atomic.LoadInt32(&self.mining) == 1 {
		wallet, err := self.eth.AccountManager().Find(accounts.Account{Address: self.coinbase})
		if err != nil {
			log.Warn("Can't find coinbase account wallet", "coinbase", self.coinbase, "err", err)
			return
		}
		if self.config.XDPoS != nil && self.chain.Config().IsTIPXDCXMiner(header.Number) {
			XDCX := self.eth.GetXDCX()
			XDCXLending := self.eth.GetXDCXLending()
			if XDCX != nil && header.Number.Uint64() > self.config.XDPoS.Epoch {
				isEpochSwitchBlock, epochNumber, err := self.engine.(*XDPoS.XDPoS).IsEpochSwitch(header)
				if err != nil {
					log.Error("[commitNewWork] fail to check if block is epoch switch block when performing XDCX and XDCXLending operations", "BlockNum", header.Number, "Hash", header.Hash())
				}

				if isEpochSwitchBlock {
					err := XDCX.UpdateMediumPriceBeforeEpoch(epochNumber, work.tradingState, work.state)
					if err != nil {
						log.Error("Fail when update medium price last epoch", "error", err)
						return
					}
				} else {
					// won't grasp tx at checkpoint
					//https://github.com/XinFinOrg/XDPoSChain-v1/pull/416
					log.Debug("Start processing order pending")
					tradingOrderPending, _ := self.eth.OrderPool().Pending()
					log.Debug("Start processing order pending", "len", len(tradingOrderPending))
					tradingTxMatches, tradingMatchingResults = XDCX.ProcessOrderPending(header, self.coinbase, self.chain, tradingOrderPending, work.state, work.tradingState)
					log.Debug("trading transaction matches found", "tradingTxMatches", len(tradingTxMatches))

					lendingOrderPending, _ := self.eth.LendingPool().Pending()
					lendingInput, lendingMatchingResults = XDCXLending.ProcessOrderPending(header, self.coinbase, self.chain, lendingOrderPending, work.state, work.lendingState, work.tradingState)
					log.Debug("lending transaction matches found", "lendingInput", len(lendingInput), "lendingMatchingResults", len(lendingMatchingResults))
					if header.Number.Uint64()%self.config.XDPoS.Epoch == common.LiquidateLendingTradeBlock {
						updatedTrades, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades, err = XDCXLending.ProcessLiquidationData(header, self.chain, work.state, work.tradingState, work.lendingState)
						if err != nil {
							log.Error("Fail when process lending liquidation data ", "error", err)
							return
						}
					}
				}

				if len(tradingTxMatches) > 0 {
					txMatchBatch := &tradingstate.TxMatchBatch{
						Data:      tradingTxMatches,
						Timestamp: time.Now().UnixNano(),
						TxHash:    common.Hash{},
					}
					txMatchBytes, err := tradingstate.EncodeTxMatchesBatch(*txMatchBatch)
					if err != nil {
						log.Error("Fail to marshal txMatch", "error", err)
						return
					}
					nonce := work.state.GetNonce(self.coinbase)
					tx := types.NewTransaction(nonce, common.XDCXAddrBinary, big.NewInt(0), txMatchGasLimit, big.NewInt(0), txMatchBytes)
					txM, err := wallet.SignTx(accounts.Account{Address: self.coinbase}, tx, self.config.ChainId)
					if err != nil {
						log.Error("Fail to create tx matches", "error", err)
						return
					} else {
						tradingTransaction = txM
						if XDCX.IsSDKNode() {
							self.chain.AddMatchingResult(tradingTransaction.Hash(), tradingMatchingResults)
						}
						// force adding trading, lending transaction to this block
						if tradingTransaction != nil {
							specialTxs = append(specialTxs, tradingTransaction)
						}
					}
				}

				if len(lendingInput) > 0 {
					// lending transaction
					lendingBatch := &lendingstate.TxLendingBatch{
						Data:      lendingInput,
						Timestamp: time.Now().UnixNano(),
						TxHash:    common.Hash{},
					}
					lendingDataBytes, err := lendingstate.EncodeTxLendingBatch(*lendingBatch)
					if err != nil {
						log.Error("Fail to marshal lendingData", "error", err)
						return
					}
					nonce := work.state.GetNonce(self.coinbase)
					lendingTx := types.NewTransaction(nonce, common.XDCXLendingAddressBinary, big.NewInt(0), txMatchGasLimit, big.NewInt(0), lendingDataBytes)
					signedLendingTx, err := wallet.SignTx(accounts.Account{Address: self.coinbase}, lendingTx, self.config.ChainId)
					if err != nil {
						log.Error("Fail to create lending tx", "error", err)
						return
					} else {
						lendingTransaction = signedLendingTx
						if XDCX.IsSDKNode() {
							self.chain.AddLendingResult(lendingTransaction.Hash(), lendingMatchingResults)
						}
						if lendingTransaction != nil {
							specialTxs = append(specialTxs, lendingTransaction)
						}
					}
				}

				if len(updatedTrades) > 0 {
					log.Debug("M1 finalized trades")
					finalizedTradeData, err := lendingstate.EncodeFinalizedResult(liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades)
					if err != nil {
						log.Error("Fail to marshal lendingData", "error", err)
						return
					}
					nonce := work.state.GetNonce(self.coinbase)
					finalizedTx := types.NewTransaction(nonce, common.XDCXLendingFinalizedTradeAddressBinary, big.NewInt(0), txMatchGasLimit, big.NewInt(0), finalizedTradeData)
					signedFinalizedTx, err := wallet.SignTx(accounts.Account{Address: self.coinbase}, finalizedTx, self.config.ChainId)
					if err != nil {
						log.Error("Fail to create lending tx", "error", err)
						return
					} else {
						lendingFinalizedTradeTransaction = signedFinalizedTx
						if XDCX.IsSDKNode() {
							self.chain.AddFinalizedTrades(lendingFinalizedTradeTransaction.Hash(), updatedTrades)
						}
						if lendingFinalizedTradeTransaction != nil {
							specialTxs = append(specialTxs, lendingFinalizedTradeTransaction)
						}
					}
				}
			}
			XDCxStateRoot := work.tradingState.IntermediateRoot()
			LendingStateRoot := work.lendingState.IntermediateRoot()
			txData := append(XDCxStateRoot.Bytes(), LendingStateRoot.Bytes()...)
			tx := types.NewTransaction(work.state.GetNonce(self.coinbase), common.TradingStateAddrBinary, big.NewInt(0), txMatchGasLimit, big.NewInt(0), txData)
			txStateRoot, err := wallet.SignTx(accounts.Account{Address: self.coinbase}, tx, self.config.ChainId)
			if err != nil {
				log.Error("Fail to create tx state root", "error", err)
				return
			}
			specialTxs = append(specialTxs, txStateRoot)
		}
	}
	work.commitTransactions(self.mux, feeCapacity, txs, specialTxs, self.chain, self.coinbase, &self.pendingLogsFeed)
	// compute uncles for the new block.
	var (
		uncles []*types.Header
	)

	// Create the new block to seal with the consensus engine
	if work.Block, err = self.engine.Finalize(self.chain, header, work.state, work.parentState, work.txs, uncles, work.receipts); err != nil {
		log.Error("Failed to finalize block for sealing", "err", err)
		return
	}

	if atomic.LoadInt32(&self.mining) == 1 {
		log.Info("Committing new block", "number", work.Block.Number(), "txs", work.tcount, "special-txs", len(specialTxs), "uncles", len(uncles), "elapsed", common.PrettyDuration(time.Since(tstart)))
		self.unconfirmed.Shift(work.Block.NumberU64() - 1)
		self.lastParentBlockCommit = parent.Hash().Hex()
	}
	self.push(work)
	self.updateSnapshot()
}

func (env *Work) commitTransactions(mux *event.TypeMux, balanceFee map[common.Address]*big.Int, txs *types.TransactionsByPriceAndNonce, specialTxs types.Transactions, bc *core.BlockChain, coinbase common.Address, pendingLogsFeed *event.Feed) {
	gp := new(core.GasPool).AddGas(env.header.GasLimit)
	balanceUpdated := map[common.Address]*big.Int{}
	totalFeeUsed := big.NewInt(0)
	var coalescedLogs []*types.Log
	// first priority for special Txs
	for _, tx := range specialTxs {
		to := tx.To()
		//HF number for black-list
		if (env.header.Number.Uint64() >= common.BlackListHFNumber) && !common.IsTestnet {
			from := tx.From()
			// check if sender is in black list
			if from != nil && common.Blacklist[*from] {
				log.Debug("Skipping transaction with sender in black-list", "sender", from.Hex())
				continue
			}
			// check if receiver is in black list
			if to != nil && common.Blacklist[*to] {
				log.Debug("Skipping transaction with receiver in black-list", "receiver", to.Hex())
				continue
			}
		}
		data := tx.Data()
		// validate minFee slot for XDCZ
		if tx.IsXDCZApplyTransaction() {
			copyState, _ := bc.State()
			if err := core.ValidateXDCZApplyTransaction(bc, nil, copyState, common.BytesToAddress(data[4:])); err != nil {
				log.Debug("XDCZApply: invalid token", "token", common.BytesToAddress(data[4:]).Hex())
				txs.Pop()
				continue
			}
		}
		// validate balance slot, token decimal for XDCX
		if tx.IsXDCXApplyTransaction() {
			copyState, _ := bc.State()
			if err := core.ValidateXDCXApplyTransaction(bc, nil, copyState, common.BytesToAddress(data[4:])); err != nil {
				log.Debug("XDCXApply: invalid token", "token", common.BytesToAddress(data[4:]).Hex())
				txs.Pop()
				continue
			}
		}

		if gp.Gas() < params.TxGas && tx.Gas() > 0 {
			log.Trace("Not enough gas for further transactions", "gp", gp)
			break
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		//
		// We use the eip155 signer regardless of the current hf.
		from, _ := types.Sender(env.signer, tx)
		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		hash := tx.Hash()
		if tx.Protected() && !env.config.IsEIP155(env.header.Number) {
			log.Trace("Ignoring reply protected special transaction", "hash", hash, "eip155", env.config.EIP155Block)
			continue
		}
		if *to == common.BlockSignersBinary {
			if len(data) < 68 {
				log.Trace("Data special transaction invalid length", "hash", hash, "data", len(data))
				continue
			}
			blkNumber := binary.BigEndian.Uint64(data[8:40])
			if blkNumber >= env.header.Number.Uint64() || blkNumber <= env.header.Number.Uint64()-env.config.XDPoS.Epoch*2 {
				log.Trace("Data special transaction invalid number", "hash", hash, "blkNumber", blkNumber, "miner", env.header.Number)
				continue
			}
		}
		// Start executing the transaction
		env.state.Prepare(hash, common.Hash{}, env.tcount)

		nonce := env.state.GetNonce(from)
		if nonce != tx.Nonce() && !tx.IsSkipNonceTransaction() {
			log.Trace("Skipping account with special transaction invalid nonce", "sender", from, "nonce", nonce, "tx nonce ", tx.Nonce(), "to", to)
			continue
		}
		err, logs, tokenFeeUsed, gas := env.commitTransaction(balanceFee, tx, bc, coinbase, gp)
		switch err {
		case core.ErrNonceTooLow:
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping special transaction with low nonce", "sender", from, "nonce", tx.Nonce(), "to", to)

		case core.ErrNonceTooHigh:
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Trace("Skipping account with special transaction hight nonce", "sender", from, "nonce", tx.Nonce(), "to", to)
		case nil:
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, logs...)
			env.tcount++

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Add Special Transaction failed, account skipped", "hash", hash, "sender", from, "nonce", tx.Nonce(), "to", to, "err", err)
		}
		if tokenFeeUsed {
			fee := common.GetGasFee(env.header.Number.Uint64(), gas)
			balanceFee[*to] = new(big.Int).Sub(balanceFee[*to], fee)
			balanceUpdated[*to] = balanceFee[*to]
			totalFeeUsed = totalFeeUsed.Add(totalFeeUsed, fee)
		}
	}
	for {
		// If we don't have enough gas for any further transactions then we're done
		if gp.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "gp", gp)
			break
		}
		if txs == nil {
			log.Info("this block has no transaction")
			break
		}
		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()

		if tx == nil {
			break
		}

		//HF number for black-list
		to := tx.To()
		if (env.header.Number.Uint64() >= common.BlackListHFNumber) && !common.IsTestnet {
			from := tx.From()
			// check if sender is in black list
			if from != nil && common.Blacklist[*from] {
				log.Debug("Skipping transaction with sender in black-list", "sender", from.Hex())
				txs.Pop()
				continue
			}
			// check if receiver is in black list
			if to != nil && common.Blacklist[*to] {
				log.Debug("Skipping transaction with receiver in black-list", "receiver", to.Hex())
				txs.Shift()
				continue
			}
		}
		data := tx.Data()
		// validate minFee slot for XDCZ
		if tx.IsXDCZApplyTransaction() {
			copyState, _ := bc.State()
			if err := core.ValidateXDCZApplyTransaction(bc, nil, copyState, common.BytesToAddress(data[4:])); err != nil {
				log.Debug("XDCZApply: invalid token", "token", common.BytesToAddress(data[4:]).Hex())
				txs.Pop()
				continue
			}
		}
		// validate balance slot, token decimal for XDCX
		if tx.IsXDCXApplyTransaction() {
			copyState, _ := bc.State()
			if err := core.ValidateXDCXApplyTransaction(bc, nil, copyState, common.BytesToAddress(data[4:])); err != nil {
				log.Debug("XDCXApply: invalid token", "token", common.BytesToAddress(data[4:]).Hex())
				txs.Pop()
				continue
			}
		}

		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		//
		// We use the eip155 signer regardless of the current hf.
		from, _ := types.Sender(env.signer, tx)
		hash := tx.Hash()
		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !env.config.IsEIP155(env.header.Number) {
			log.Trace("Ignoring reply protected transaction", "hash", hash, "eip155", env.config.EIP155Block)
			txs.Pop()
			continue
		}
		// Start executing the transaction
		env.state.Prepare(hash, common.Hash{}, env.tcount)
		nonce := env.state.GetNonce(from)
		if nonce > tx.Nonce() {
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txs.Shift()
			continue
		}
		if nonce < tx.Nonce() {
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Trace("Skipping account with hight nonce", "sender", from, "nonce", tx.Nonce())
			txs.Pop()
			continue
		}
		err, logs, tokenFeeUsed, gas := env.commitTransaction(balanceFee, tx, bc, coinbase, gp)
		switch {
		case errors.Is(err, core.ErrGasLimitReached):
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Trace("Gas limit exceeded for current block", "sender", from)
			txs.Pop()

		case errors.Is(err, core.ErrNonceTooLow):
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case errors.Is(err, core.ErrNonceTooHigh):
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Trace("Skipping account with high nonce", "sender", from, "nonce", tx.Nonce())
			txs.Pop()

		case errors.Is(err, nil):
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, logs...)
			env.tcount++
			txs.Shift()

		case errors.Is(err, core.ErrTxTypeNotSupported):
			// Pop the unsupported transaction without shifting in the next from the account
			log.Trace("Skipping unsupported transaction type", "sender", from, "type", tx.Type())
			txs.Pop()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Transaction failed, account skipped", "hash", hash, "err", err)
			txs.Shift()
		}
		if tokenFeeUsed {
			fee := common.GetGasFee(env.header.Number.Uint64(), gas)
			balanceFee[*to] = new(big.Int).Sub(balanceFee[*to], fee)
			balanceUpdated[*to] = balanceFee[*to]
			totalFeeUsed = totalFeeUsed.Add(totalFeeUsed, fee)
		}
	}
	state.UpdateTRC21Fee(env.state, balanceUpdated, totalFeeUsed)
	// make a copy, the state caches the logs and these logs get "upgraded" from pending to mined
	// logs by filling in the block hash when the block was mined by the local miner. This can
	// cause a race condition if a log was "upgraded" before the PendingLogsEvent is processed.
	if len(coalescedLogs) > 0 {
		cpy := make([]*types.Log, len(coalescedLogs))
		for i, l := range coalescedLogs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		pendingLogsFeed.Send(cpy)
	}
	if env.tcount > 0 {
		go func(tcount int) {
			err := mux.Post(core.PendingStateEvent{})
			if err != nil {
				log.Warn("[commitTransactions] Error when sending PendingStateEvent", "tcount", tcount)
			}
		}(env.tcount)

	}
}

func (env *Work) commitTransaction(balanceFee map[common.Address]*big.Int, tx *types.Transaction, bc *core.BlockChain, coinbase common.Address, gp *core.GasPool) (error, []*types.Log, bool, uint64) {
	snap := env.state.Snapshot()

	receipt, gas, err, tokenFeeUsed := core.ApplyTransaction(env.config, balanceFee, bc, &coinbase, gp, env.state, env.tradingState, env.header, tx, &env.header.GasUsed, vm.Config{})
	if err != nil {
		env.state.RevertToSnapshot(snap)
		return err, nil, false, 0
	}
	env.txs = append(env.txs, tx)
	env.receipts = append(env.receipts, receipt)

	return nil, receipt.Logs, tokenFeeUsed, gas
}
