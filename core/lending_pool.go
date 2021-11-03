// Copyright 2014 The go-ethereum Authors
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

package core

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"

	"github.com/XinFinOrg/XDPoSChain/XDCxlending/lendingstate"
	"github.com/XinFinOrg/XDPoSChain/consensus"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

var (
	ErrInvalidLendingSide        = errors.New("invalid lending side")
	ErrInvalidLendingType        = errors.New("invalid lending type")
	ErrInvalidLendingStatus      = errors.New("invalid lending status")
	ErrInvalidLendingUserAddress = errors.New("invalid lending user address")
	ErrInvalidLendingQuantity    = errors.New("invalid lending quantity")
	ErrInvalidLendingInterest    = errors.New("invalid lending interest")
	ErrInvalidLendingRelayer     = errors.New("invalid lending relayer address")
	ErrInvalidLendingHash        = errors.New("invalid lending hash")
	ErrInvalidCancelledLending   = errors.New("invalid cancel lending id")
	ErrInvalidLendingTradeID     = errors.New("invalid lending trade ID")
	ErrInvalidLendingCollateral  = errors.New("invalid collateral")
)

var (
	LendingTypeLimit  = "LO"
	LendingTypeMarket = "MO"
)

// LendingPoolConfig are the configuration parameters of the order transaction pool.
type LendingPoolConfig struct {
	NoLocals  bool          // Whether local transaction handling should be disabled
	Journal   string        // Journal of local transactions to survive node restarts
	Rejournal time.Duration // Time interval to regenerate the local transaction journal

	AccountSlots uint64 // Minimum number of executable transaction slots guaranteed per account
	GlobalSlots  uint64 // Maximum number of executable transaction slots for all accounts
	AccountQueue uint64 // Maximum number of non-executable transaction slots permitted per account
	GlobalQueue  uint64 // Maximum number of non-executable transaction slots for all accounts

	Lifetime time.Duration // Maximum amount of time non-executable transaction are queued
}

// blockChain_XDCx add order state
type blockChainLending interface {
	CurrentBlock() *types.Block
	GetBlock(hash common.Hash, number uint64) *types.Block
	LendingStateAt(block *types.Block) (*lendingstate.LendingStateDB, error)
	StateAt(root common.Hash) (*state.StateDB, error)
	SubscribeChainHeadEvent(ch chan<- ChainHeadEvent) event.Subscription
	Engine() consensus.Engine
	// GetHeader returns the hash corresponding to their hash.
	GetHeader(common.Hash, uint64) *types.Header
	// CurrentHeader retrieves the current header from the local chain.
	CurrentHeader() *types.Header
	// Config retrieves the blockchain's chain configuration.
	Config() *params.ChainConfig
}

// DefaultLendingPoolConfig contains the default configurations for the transaction
// pool.
var DefaultLendingPoolConfig = LendingPoolConfig{
	Journal:   "",
	Rejournal: time.Hour,

	AccountSlots: 16,
	GlobalSlots:  4096,
	AccountQueue: 64,
	GlobalQueue:  1024,

	Lifetime: 3 * time.Hour,
}

// sanitize checks the provided user configurations and changes anything that's
// unreasonable or unworkable.
func (config *LendingPoolConfig) sanitize() LendingPoolConfig {
	conf := *config
	if conf.Rejournal < time.Second {
		log.Warn("Sanitizing invalid LendingPool journal time", "provided", conf.Rejournal, "updated", time.Second)
		conf.Rejournal = time.Second
	}
	return conf
}

// LendingPool contains all currently known transactions. Transactions
// enter the pool when they are received from the network or submitted
// locally. They exit the pool when they are included in the blockchain.
//
// The pool separates processable transactions (which can be applied to the
// current state) and future transactions. Transactions move between those
// two states over time as they are received and processed.
type LendingPool struct {
	config      LendingPoolConfig
	chainconfig *params.ChainConfig
	chain       blockChainLending

	txFeed       event.Feed
	scope        event.SubscriptionScope
	chainHeadCh  chan ChainHeadEvent
	chainHeadSub event.Subscription
	signer       types.LendingSigner
	mu           sync.RWMutex

	currentRootState    *state.StateDB
	currentLendingState *lendingstate.LendingStateDB      // Current order state in the blockchain head
	pendingState        *lendingstate.LendingManagedState // Pending state tracking virtual nonces

	locals  *lendingAccountSet // Set of local transaction to exempt from eviction rules
	journal *lendingtxJournal  // Journal of local transaction to back up to disk

	pending   map[common.Address]*lendingtxList         // All currently processable transactions
	queue     map[common.Address]*lendingtxList         // Queued but non-processable transactions
	beats     map[common.Address]time.Time              // Last heartbeat from each known account
	all       map[common.Hash]*types.LendingTransaction // All transactions to allow lookups
	wg        sync.WaitGroup                            // for shutdown sync
	homestead bool
	IsSigner  func(address common.Address) bool
}

// NewLendingPool creates a new transaction pool to gather, sort and filter inbound
// transactions from the network.
func NewLendingPool(chainconfig *params.ChainConfig, chain blockChainLending) *LendingPool {
	// Sanitize the input to ensure no vulnerable gas prices are set
	config := (&DefaultLendingPoolConfig).sanitize()
	log.Debug("NewLendingPool start...", "current block", chain.CurrentBlock().Header().Number)
	// Create the transaction pool with its initial settings
	pool := &LendingPool{
		config:      config,
		chainconfig: chainconfig,
		chain:       chain,
		signer:      types.LendingTxSigner{},
		pending:     make(map[common.Address]*lendingtxList),
		queue:       make(map[common.Address]*lendingtxList),
		beats:       make(map[common.Address]time.Time),
		all:         make(map[common.Hash]*types.LendingTransaction),
		chainHeadCh: make(chan ChainHeadEvent, chainHeadChanSize),
	}
	pool.locals = newLendingAccountSet(pool.signer)
	pool.reset(nil, chain.CurrentBlock())

	// If local transactions and journaling is enabled, load from disk
	if !config.NoLocals && config.Journal != "" {
		pool.journal = newLendingTxJournal(config.Journal)

		if err := pool.journal.load(pool.AddLocal); err != nil {
			log.Warn("Failed to load transaction journal", "err", err)
		}
		if err := pool.journal.rotate(pool.local()); err != nil {
			log.Warn("Failed to rotate transaction journal", "err", err)
		}
	}
	// Subscribe events from blockchain
	pool.chainHeadSub = pool.chain.SubscribeChainHeadEvent(pool.chainHeadCh)

	// Start the event loop and return
	pool.wg.Add(1)
	go pool.loop()

	return pool
}

// loop is the transaction pool's main event loop, waiting for and reacting to
// outside blockchain events as well as for various reporting and transaction
// eviction events.
func (pool *LendingPool) loop() {
	defer pool.wg.Done()

	// Start the stats reporting and transaction eviction tickers
	var prevPending, prevQueued int

	report := time.NewTicker(statsReportInterval)
	defer report.Stop()

	evict := time.NewTicker(evictionInterval)
	defer evict.Stop()

	journal := time.NewTicker(pool.config.Rejournal)
	defer journal.Stop()

	// Track the previous head headers for transaction reorgs
	head := pool.chain.CurrentBlock()

	// Keep waiting for and reacting to the various events
	for {
		select {
		// Handle ChainHeadEvent
		case ev := <-pool.chainHeadCh:
			if ev.Block != nil {
				pool.mu.Lock()
				if pool.chainconfig.IsHomestead(ev.Block.Number()) {
					pool.homestead = true
				}
				log.Debug("LendingPool new chain header reset pool", "old", head.Header().Number, "new", ev.Block.Header().Number)
				pool.reset(head, ev.Block)
				head = ev.Block

				pool.mu.Unlock()
			}
			// Be unsubscribed due to system stopped
		case <-pool.chainHeadSub.Err():
			return

			// Handle stats reporting ticks
		case <-report.C:
			pool.mu.RLock()
			pending, queued := pool.stats()
			pool.mu.RUnlock()
			if pending != prevPending || queued != prevQueued {
				log.Debug("Lending pool status report", "executable", pending, "queued", queued)
				prevPending, prevQueued = pending, queued
			}

			// Handle inactive account transaction eviction
		case <-evict.C:
			pool.mu.Lock()
			for addr := range pool.queue {
				// Skip local transactions from the eviction mechanism
				if pool.locals.contains(addr) {
					continue
				}
				// Any non-locals old enough should be removed
				if time.Since(pool.beats[addr]) > pool.config.Lifetime {
					for _, tx := range pool.queue[addr].Flatten() {
						pool.removeTx(tx.Hash())
					}
				}
			}
			pool.mu.Unlock()

			// Handle local transaction journal rotation
		case <-journal.C:
			if pool.journal != nil {
				pool.mu.Lock()
				if err := pool.journal.rotate(pool.local()); err != nil {
					log.Warn("Failed to rotate local tx journal", "err", err)
				}
				pool.mu.Unlock()
			}
		}
	}
}

// reset retrieves the current state of the blockchain and ensures the content
// of the transaction pool is valid with regard to the chain state.
func (pool *LendingPool) reset(oldHead, newblock *types.Block) {
	if !pool.chainconfig.IsTIPXDCX(pool.chain.CurrentBlock().Number()) || pool.chain.Config().XDPoS == nil || pool.chain.CurrentBlock().NumberU64() <= pool.chain.Config().XDPoS.Epoch {
		return
	}
	// If we're reorging an old state, reinject all dropped transactions
	var reinject types.LendingTransactions

	// Initialize the internal state to the current head
	if newblock == nil {
		newblock = pool.chain.CurrentBlock()
	}
	newHead := newblock.Header()
	lendingState, err := pool.chain.LendingStateAt(newblock)
	if err != nil {
		log.Error("Failed to reset LendingPool state", "err", err)
		return
	}
	pool.currentLendingState = lendingState
	pool.pendingState = lendingstate.ManageState(lendingState)

	state, err := pool.chain.StateAt(newHead.Root)
	if err != nil {
		log.Error("Failed to reset pool state", "err", err)
		return
	}
	pool.currentRootState = state

	// Inject any transactions discarded due to reorgs
	log.Debug("Reinjecting stale transactions", "count", len(reinject))
	pool.addTxsLocked(reinject, false)

	// validate the pool of pending transactions, this will remove
	// any transactions that have been included in the block or
	// have been invalidated because of another transaction (e.g.
	// higher gas price)
	pool.demoteUnexecutables()

	// Update all accounts to the latest known pending nonce
	for addr, list := range pool.pending {
		txs := list.Flatten() // Heavy but will be cached and is needed by the miner anyway
		pool.pendingState.SetNonce(addr.Hash(), txs[len(txs)-1].Nonce()+1)
	}
	// Check the queue and move transactions over to the pending if possible
	// or remove those that have become invalid
	pool.promoteExecutables(nil)
}

// Stop terminates the transaction pool.
func (pool *LendingPool) Stop() {
	// Unsubscribe all subscriptions registered from LendingPool
	pool.scope.Close()

	// Unsubscribe subscriptions registered from blockchain
	pool.chainHeadSub.Unsubscribe()
	pool.wg.Wait()

	if pool.journal != nil {
		pool.journal.close()
	}
	log.Info("Transaction pool stopped")
}

// SubscribeTxPreEvent registers a subscription of TxPreEvent and
// starts sending event to the given channel.
func (pool *LendingPool) SubscribeTxPreEvent(ch chan<- LendingTxPreEvent) event.Subscription {
	return pool.scope.Track(pool.txFeed.Subscribe(ch))
}

// State returns the virtual managed state of the transaction pool.
func (pool *LendingPool) State() *lendingstate.LendingManagedState {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.pendingState
}

// Stats retrieves the current pool stats, namely the number of pending and the
// number of queued (non-executable) transactions.
func (pool *LendingPool) Stats() (int, int) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.stats()
}

// stats retrieves the current pool stats, namely the number of pending and the
// number of queued (non-executable) transactions.
func (pool *LendingPool) stats() (int, int) {
	pending := 0
	for _, list := range pool.pending {
		pending += list.Len()
	}
	queued := 0
	for _, list := range pool.queue {
		queued += list.Len()
	}
	return pending, queued
}

// Pending retrieves all currently processable transactions, groupped by origin
// account and sorted by nonce. The returned transaction set is a copy and can be
// freely modified by calling code.
func (pool *LendingPool) Pending() (map[common.Address]types.LendingTransactions, error) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pending := make(map[common.Address]types.LendingTransactions)
	for addr, list := range pool.pending {
		pending[addr] = list.Flatten()
	}
	return pending, nil
}

// local retrieves all currently known local transactions, groupped by origin
// account and sorted by nonce. The returned transaction set is a copy and can be
// freely modified by calling code.
func (pool *LendingPool) local() map[common.Address]types.LendingTransactions {
	txs := make(map[common.Address]types.LendingTransactions)
	for addr := range pool.locals.accounts {
		if pending := pool.pending[addr]; pending != nil {
			txs[addr] = append(txs[addr], pending.Flatten()...)
		}
		if queued := pool.queue[addr]; queued != nil {
			txs[addr] = append(txs[addr], queued.Flatten()...)
		}
	}
	return txs
}

// GetSender get sender from transaction
func (pool *LendingPool) GetSender(tx *types.LendingTransaction) (common.Address, error) {
	from, err := types.LendingSender(pool.signer, tx)
	if err != nil {
		return common.Address{}, ErrInvalidSender
	}
	return from, nil
}

func (pool *LendingPool) validateNewLending(cloneStateDb *state.StateDB, cloneLendingStateDb *lendingstate.LendingStateDB, tx *types.LendingTransaction) error {
	lendingSide := tx.Side()
	lendingType := tx.Type()
	interest := tx.Interest()
	quantity := tx.Quantity()
	if quantity == nil || quantity.Sign() <= 0 {
		return ErrInvalidLendingQuantity
	}
	if lendingType != LendingTypeMarket {
		if interest <= 0 {
			return ErrInvalidLendingInterest
		}
	}

	if lendingSide != lendingstate.Investing && lendingSide != lendingstate.Borrowing {
		return ErrInvalidLendingSide
	}
	if lendingType != LendingTypeLimit && lendingType != LendingTypeMarket {
		return ErrInvalidLendingType
	}
	if tx.Side() == lendingstate.Borrowing {
		if tx.CollateralToken().String() == lendingstate.EmptyAddress || tx.CollateralToken().String() == tx.LendingToken().String() {
			return ErrInvalidLendingCollateral
		}
		validCollateral := false
		collateralList, _ := lendingstate.GetCollaterals(cloneStateDb, tx.RelayerAddress(), tx.LendingToken(), tx.Term())
		for _, collateral := range collateralList {
			if tx.CollateralToken().String() == collateral.String() {
				validCollateral = true
				break
			}
		}
		if !validCollateral {
			return ErrInvalidLendingCollateral
		}
	}
	if lendingType == LendingTypeLimit {
		if err := pool.validateBalance(cloneStateDb, cloneLendingStateDb, tx, tx.CollateralToken()); err != nil {
			return err
		}
	}
	return nil
}

func (pool *LendingPool) validateCancelledLending(cloneLendingStateDb *lendingstate.LendingStateDB, tx *types.LendingTransaction) error {
	if tx.LendingId() == 0 {
		return ErrInvalidCancelledLending
	}
	item := cloneLendingStateDb.GetLendingOrder(lendingstate.GetLendingOrderBookHash(tx.LendingToken(), tx.Term()), common.Uint64ToHash(tx.LendingId()))
	if item == lendingstate.EmptyLendingOrder {
		log.Debug("LendingOrder not found ", "LendingId", tx.LendingId(), "LendToken", tx.LendingToken().Hex(), "CollateralToken", tx.CollateralToken().Hex(), "Term", tx.Term())
		return ErrInvalidCancelledLending
	}
	if item.Hash != tx.LendingHash() {
		log.Debug("Invalid lending hash", "expected", item.Hash.Hex(), "got", tx.LendingHash().Hex())
		return ErrInvalidLendingHash
	}
	return nil
}
func (pool *LendingPool) validateRepayLending(cloneStateDb *state.StateDB, cloneLendingStateDb *lendingstate.LendingStateDB, tx *types.LendingTransaction) error {
	if tx.LendingTradeId() == 0 {
		return ErrInvalidLendingTradeID
	}
	lendingBook := lendingstate.GetLendingOrderBookHash(tx.LendingToken(), tx.Term())
	lendingTrade := cloneLendingStateDb.GetLendingTrade(lendingBook, common.Uint64ToHash(tx.LendingTradeId()))
	if lendingTrade == lendingstate.EmptyLendingTrade {
		return ErrInvalidLendingTradeID
	}
	if tx.UserAddress().String() != lendingTrade.Borrower.String() {
		return ErrInvalidLendingUserAddress
	}
	if tx.RelayerAddress().String() != lendingTrade.BorrowingRelayer.String() {
		return ErrInvalidLendingRelayer
	}
	if err := pool.validateBalance(cloneStateDb, cloneLendingStateDb, tx, tx.CollateralToken()); err != nil {
		return err
	}
	return nil
}
func (pool *LendingPool) validateTopupLending(cloneStateDb *state.StateDB, cloneLendingStateDb *lendingstate.LendingStateDB, tx *types.LendingTransaction) error {
	if tx.LendingTradeId() == 0 {
		return ErrInvalidLendingTradeID
	}
	if tx.Quantity() == nil || tx.Quantity().Sign() <= 0 {
		return ErrInvalidLendingQuantity
	}
	lendingBook := lendingstate.GetLendingOrderBookHash(tx.LendingToken(), tx.Term())
	lendingTrade := cloneLendingStateDb.GetLendingTrade(lendingBook, common.Uint64ToHash(tx.LendingTradeId()))
	if lendingTrade == lendingstate.EmptyLendingTrade {
		return ErrInvalidLendingTradeID
	}
	if tx.UserAddress().String() != lendingTrade.Borrower.String() {
		return ErrInvalidLendingUserAddress
	}
	if tx.RelayerAddress().String() != lendingTrade.BorrowingRelayer.String() {
		return ErrInvalidLendingRelayer
	}
	if err := pool.validateBalance(cloneStateDb, cloneLendingStateDb, tx, lendingTrade.CollateralToken); err != nil {
		return err
	}
	return nil
}

func (pool *LendingPool) validateBalance(cloneStateDb *state.StateDB, cloneLendingStateDb *lendingstate.LendingStateDB, tx *types.LendingTransaction, collateralToken common.Address) error {
	XDPoSEngine, ok := pool.chain.Engine().(*XDPoS.XDPoS)
	if !ok {
		return ErrNotXDPoS
	}
	XDCXServ := XDPoSEngine.GetXDCXService()
	lendingServ := XDPoSEngine.GetLendingService()
	if XDCXServ == nil {
		return fmt.Errorf("XDCx not found in order validation")
	}
	lendingTokenDecimal, err := XDCXServ.GetTokenDecimal(pool.chain, cloneStateDb, tx.LendingToken())
	if err != nil {
		return fmt.Errorf("validateOrder: failed to get lendingTokenDecimal. err: %v", err)
	}
	author, err := pool.chain.Engine().Author(pool.chain.CurrentHeader())
	if err != nil {
		return err
	}
	tradingStateDb, err := XDCXServ.GetTradingState(pool.chain.CurrentBlock(), author)
	if err != nil {
		return fmt.Errorf("validateLending: failed to get tradingStateDb. Error: %v", err)
	}
	cloneTradingStateDb := tradingStateDb.Copy()
	// collateralPrice: price of collateral by LendingToken
	// Eg: LendingToken: USD, CollateralToken: BTC
	// collateralPrice = BTC/USD (eg: 8000 USD)
	// lendTokenXDCPrice: price of lendingToken in XDC quote
	var lendTokenXDCPrice, collateralPrice, collateralTokenDecimal *big.Int
	if collateralToken.String() != lendingstate.EmptyAddress {
		collateralTokenDecimal, err = XDCXServ.GetTokenDecimal(pool.chain, cloneStateDb, collateralToken)
		if err != nil {
			return fmt.Errorf("validateOrder: failed to get collateralTokenDecimal. err: %v", err)
		}
		lendTokenXDCPrice, collateralPrice, err = lendingServ.GetCollateralPrices(pool.chain.CurrentHeader(), pool.chain, cloneStateDb, cloneTradingStateDb, collateralToken, tx.LendingToken())
		if err != nil {
			return err
		}
		if lendTokenXDCPrice == nil || lendTokenXDCPrice.Sign() <= 0 || collateralPrice == nil || collateralPrice.Sign() <= 0 {
			log.Debug("ValidateLending: ErrInvalidCollateralPrice", "lendTokenXDCPrice", lendTokenXDCPrice, "collateralPrice", collateralPrice)
			return lendingstate.ErrInvalidCollateralPrice
		}
	}
	if lendTokenXDCPrice == nil || lendTokenXDCPrice.Sign() == 0 {
		if tx.LendingToken().String() == common.XDCNativeAddress {
			lendTokenXDCPrice = common.BasePrice
		} else {
			lendTokenXDCPrice, err = lendingServ.GetMediumTradePriceBeforeEpoch(pool.chain, cloneStateDb, cloneTradingStateDb, tx.LendingToken(), common.HexToAddress(common.XDCNativeAddress))
			if err != nil {
				return err
			}
		}
	}
	isXDCXLendingFork := pool.chain.Config().IsTIPXDCXLending(pool.chain.CurrentHeader().Number)
	if err := lendingstate.VerifyBalance(isXDCXLendingFork,
		cloneStateDb,
		cloneLendingStateDb,
		tx.Type(),
		tx.Side(),
		tx.Status(),
		tx.UserAddress(),
		tx.RelayerAddress(),
		tx.LendingToken(),
		tx.CollateralToken(),
		tx.Quantity(),
		lendingTokenDecimal,
		collateralTokenDecimal,
		lendTokenXDCPrice,
		collateralPrice,
		tx.Term(),
		tx.LendingId(),
		tx.LendingTradeId(),
	); err != nil {
		return err
	}
	return nil
}

func (pool *LendingPool) validateLending(tx *types.LendingTransaction) error {
	cloneStateDb := pool.currentRootState.Copy()
	cloneLendingStateDb := pool.currentLendingState.Copy()
	from, _ := types.LendingSender(pool.signer, tx)
	if from != tx.UserAddress() {
		return ErrInvalidLendingUserAddress
	}
	if !lendingstate.IsValidRelayer(cloneStateDb, tx.RelayerAddress()) {
		return fmt.Errorf("invalid lending relayer. ExchangeAddress: %s", tx.RelayerAddress().Hex())
	}
	if valid, _ := lendingstate.IsValidPair(cloneStateDb, tx.RelayerAddress(), tx.LendingToken(), tx.Term()); valid == false {
		return fmt.Errorf("invalid pair. Relayer: %s. LendingToken: %s. Term: %d", tx.RelayerAddress().Hex(), tx.LendingToken().Hex(), tx.Term())
	}
	if tx.IsCreatedLending() {
		return pool.validateNewLending(cloneStateDb, cloneLendingStateDb, tx)
	}
	if tx.IsCancelledLending() {
		return pool.validateCancelledLending(cloneLendingStateDb, tx)
	}
	if tx.IsTopupLending() {
		return pool.validateTopupLending(cloneStateDb, cloneLendingStateDb, tx)
	}
	if tx.IsRepayLending() {
		return pool.validateRepayLending(cloneStateDb, cloneLendingStateDb, tx)
	}

	return ErrInvalidLendingStatus
}

// validateTx checks whether a transaction is valid according to the consensus
// rules and adheres to some heuristic limits of the local node (price and size).
func (pool *LendingPool) validateTx(tx *types.LendingTransaction, local bool) error {

	// check if sender is in black list
	if tx.From() != nil && common.Blacklist[*tx.From()] {
		return fmt.Errorf("Reject transaction with sender in black-list: %v", tx.From().Hex())
	}
	// Heuristic limit, reject transactions over 32KB to prevent DOS attacks
	if tx.Size() > 32*1024 {
		return ErrOversizedData
	}

	// Make sure the transaction is signed properly
	from, err := types.LendingSender(pool.signer, tx)
	if err != nil {
		return ErrInvalidSender
	}
	err = pool.validateLending(tx)
	if err != nil {
		return err
	}
	// Ensure the transaction adheres to nonce lending
	if pool.currentLendingState.GetNonce(from.Hash()) > tx.Nonce() {
		return ErrNonceTooLow
	}
	if pool.pendingState.GetNonce(from.Hash())+common.LimitThresholdNonceInQueue < tx.Nonce() {
		return ErrNonceTooHigh
	}

	return nil
}

// add validates a transaction and inserts it into the non-executable queue for
// later pending promotion and execution. If the transaction is a replacement for
// an already pending or queued one, it overwrites the previous and returns this
// so outer code doesn't uselessly call promote.
//
// If a newly added transaction is marked as local, its sending account will be
// whitelisted, preventing any associated transaction from being dropped out of
// the pool due to pricing constraints.
func (pool *LendingPool) add(tx *types.LendingTransaction, local bool) (bool, error) {
	// If the transaction is already known, discard it
	hash := tx.Hash()
	if pool.all[hash] != nil {
		log.Debug("Discarding already known transaction", "hash", hash)
		return false, fmt.Errorf("known transaction: %x", hash)
	}

	// If the transaction fails basic validation, discard it
	if err := pool.validateTx(tx, local); err != nil {
		log.Debug("Discarding invalid lending transaction", "hash", hash, "userAddress", tx.UserAddress, "status", tx.Status, "err", err)
		invalidTxCounter.Inc(1)
		return false, err
	}
	from, _ := types.LendingSender(pool.signer, tx) // already validated

	// If the transaction pool is full, discard underpriced transactions
	if uint64(len(pool.all)) >= pool.config.GlobalSlots+pool.config.GlobalQueue {
		log.Debug("Add lending transaction to pool full", "hash", hash, "nonce", tx.Nonce())
		return false, ErrPoolOverflow
	}
	// If the transaction is replacing an already pending one, do directly
	if list := pool.pending[from]; list != nil && list.Overlaps(tx) {
		inserted, old := list.Add(tx)
		if !inserted {
			pendingDiscardCounter.Inc(1)
			return false, ErrPendingNonceTooLow
		}
		if old != nil {
			delete(pool.all, old.Hash())
			pendingReplaceCounter.Inc(1)
		}
		pool.all[tx.Hash()] = tx
		pool.journalTx(from, tx)

		log.Debug("Lending Pooled new executable transaction", "hash", hash, "useraddress", tx.UserAddress(), "nonce", tx.Nonce(), "status", tx.Status(), "lendingid", tx.LendingId())
		return old != nil, nil

	}
	// New transaction isn't replacing a pending one, push into queue
	replace, err := pool.enqueueTx(hash, tx)
	if err != nil {
		return false, err
	}
	// Mark local addresses and journal local transactions
	if local {
		pool.locals.add(from)
	}
	pool.journalTx(from, tx)

	log.Debug("Pooled new future transaction", "hash", hash, "from", from)
	return replace, nil
}

// enqueueTx inserts a new transaction into the non-executable transaction queue.
//
// Note, this method assumes the pool lock is held!
func (pool *LendingPool) enqueueTx(hash common.Hash, tx *types.LendingTransaction) (bool, error) {
	// Try to insert the transaction into the future queue
	from, _ := types.LendingSender(pool.signer, tx) // already validated
	if pool.queue[from] == nil {
		pool.queue[from] = newLendingTxList(false)
	}
	inserted, old := pool.queue[from].Add(tx)
	if !inserted {
		// An older transaction was better, discard this
		queuedDiscardCounter.Inc(1)
		return false, ErrPendingNonceTooLow
	}
	// Discard any previous transaction and mark this
	if old != nil {
		delete(pool.all, old.Hash())
		queuedReplaceCounter.Inc(1)
	}
	pool.all[hash] = tx
	return old != nil, nil
}

// journalTx adds the specified transaction to the local disk journal if it is
// deemed to have been sent from a local account.
func (pool *LendingPool) journalTx(from common.Address, tx *types.LendingTransaction) {
	// Only journal if it's enabled and the transaction is local
	if pool.journal == nil || !pool.locals.contains(from) {
		return
	}
	if err := pool.journal.insert(tx); err != nil {
		log.Warn("Failed to journal local transaction", "err", err)
	}
}

// promoteTx adds a transaction to the pending (processable) list of transactions.
//
// Note, this method assumes the pool lock is held!
func (pool *LendingPool) promoteTx(addr common.Address, hash common.Hash, tx *types.LendingTransaction) {
	// Try to insert the transaction into the pending queue
	if pool.pending[addr] == nil {
		pool.pending[addr] = newLendingTxList(true)
	}
	list := pool.pending[addr]

	inserted, old := list.Add(tx)
	if !inserted {
		// An older transaction was better, discard this
		delete(pool.all, hash)
		pendingDiscardCounter.Inc(1)
		return
	}
	// Otherwise discard any previous transaction and mark this
	if old != nil {
		delete(pool.all, old.Hash())
		pendingReplaceCounter.Inc(1)
	}
	// Failsafe to work around direct pending inserts (tests)
	if pool.all[hash] == nil {
		pool.all[hash] = tx
	}
	// Set the potentially new pending nonce and notify any subsystems of the new tx
	pool.beats[addr] = time.Now()
	pool.pendingState.SetNonce(addr.Hash(), tx.Nonce()+1)

	go pool.txFeed.Send(LendingTxPreEvent{tx})
}

// AddLocal enqueues a single transaction into the pool if it is valid, marking
// the sender as a local one in the mean time, ensuring it goes around the local
// pricing constraints.
func (pool *LendingPool) AddLocal(tx *types.LendingTransaction) error {
	log.Debug("Lending add local tx", "relayeraddress", tx.RelayerAddress().Hex(), "addr", tx.UserAddress(), "nonce", tx.Nonce(), "ohash", tx.LendingHash().Hex(), "status", tx.Status(), "lendingid", tx.LendingId(), "lendingtradeid", tx.LendingTradeId())
	return pool.addTx(tx, !pool.config.NoLocals)
}

// AddRemote enqueues a single transaction into the pool if it is valid. If the
// sender is not among the locally tracked ones, full pricing constraints will
// apply.
func (pool *LendingPool) AddRemote(tx *types.LendingTransaction) error {
	return pool.addTx(tx, false)
}

// AddLocals enqueues a batch of transactions into the pool if they are valid,
// marking the senders as a local ones in the mean time, ensuring they go around
// the local pricing constraints.
func (pool *LendingPool) AddLocals(txs []*types.LendingTransaction) []error {
	return pool.addTxs(txs, !pool.config.NoLocals)
}

// AddRemotes enqueues a batch of transactions into the pool if they are valid.
// If the senders are not among the locally tracked ones, full pricing constraints
// will apply.
func (pool *LendingPool) AddRemotes(txs []*types.LendingTransaction) []error {
	return pool.addTxs(txs, false)
}

// addTx enqueues a single transaction into the pool if it is valid.
func (pool *LendingPool) addTx(tx *types.LendingTransaction, local bool) error {
	if !pool.chainconfig.IsTIPXDCX(pool.chain.CurrentBlock().Number()) {
		return nil
	}
	tx.CacheHash()
	types.CacheLendingSigner(pool.signer, tx)
	pool.mu.Lock()
	defer pool.mu.Unlock()

	// Try to inject the transaction and update any state
	replace, err := pool.add(tx, local)
	if err != nil {
		return err
	}
	// If we added a new transaction, run promotion checks and return
	if !replace {
		from, _ := types.LendingSender(pool.signer, tx) // already validated
		pool.promoteExecutables([]common.Address{from})
	}
	return nil
}

// addTxs attempts to queue a batch of transactions if they are valid.
func (pool *LendingPool) addTxs(txs []*types.LendingTransaction, local bool) []error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	return pool.addTxsLocked(txs, local)
}

// addTxsLocked attempts to queue a batch of transactions if they are valid,
// whilst assuming the transaction pool lock is already held.
func (pool *LendingPool) addTxsLocked(txs []*types.LendingTransaction, local bool) []error {
	// Add the batch of transaction, tracking the accepted ones
	dirty := make(map[common.Address]struct{})
	errs := make([]error, len(txs))

	for i, tx := range txs {
		var replace bool
		if replace, errs[i] = pool.add(tx, local); errs[i] == nil {
			if !replace {
				from, _ := types.LendingSender(pool.signer, tx) // already validated
				dirty[from] = struct{}{}
			}
		}
	}
	// Only reprocess the internal state if something was actually added
	if len(dirty) > 0 {
		addrs := make([]common.Address, 0, len(dirty))
		for addr := range dirty {
			addrs = append(addrs, addr)
		}
		pool.promoteExecutables(addrs)
	}
	return errs
}

// Status returns the status (unknown/pending/queued) of a batch of transactions
// identified by their hashes.
func (pool *LendingPool) Status(hashes []common.Hash) []TxStatus {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	status := make([]TxStatus, len(hashes))
	for i, hash := range hashes {
		if tx := pool.all[hash]; tx != nil {
			from, _ := types.LendingSender(pool.signer, tx) // already validated
			if pool.pending[from] != nil && pool.pending[from].txs.items[tx.Nonce()] != nil {
				status[i] = TxStatusPending
			} else {
				status[i] = TxStatusQueued
			}
		}
	}
	return status
}

// Get returns a transaction if it is contained in the pool
// and nil otherwise.
func (pool *LendingPool) Get(hash common.Hash) *types.LendingTransaction {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.all[hash]
}

// removeTx removes a single transaction from the queue, moving all subsequent
// transactions back to the future queue.
func (pool *LendingPool) removeTx(hash common.Hash) {
	// Fetch the transaction we wish to delete
	tx, ok := pool.all[hash]
	if !ok {
		return
	}
	addr, _ := types.LendingSender(pool.signer, tx) // already validated during insertion

	// Remove it from the list of known transactions
	delete(pool.all, hash)

	// Remove the transaction from the pending lists and reset the account nonce
	if pending := pool.pending[addr]; pending != nil {
		if removed, invalids := pending.Remove(tx); removed {
			// If no more pending transactions are left, remove the list
			if pending.Empty() {
				delete(pool.pending, addr)
				delete(pool.beats, addr)
			}
			// Postpone any invalidated transactions
			for _, tx := range invalids {
				pool.enqueueTx(tx.Hash(), tx)
			}
			// Update the account nonce if needed
			if nonce := tx.Nonce(); pool.pendingState.GetNonce(addr.Hash()) > nonce {
				pool.pendingState.SetNonce(addr.Hash(), nonce)
			}
			return
		}
	}
	// Transaction is in the future queue
	if future := pool.queue[addr]; future != nil {
		future.Remove(tx)
		if future.Empty() {
			delete(pool.queue, addr)
		}
	}
}

// promoteExecutables moves transactions that have become processable from the
// future queue to the set of pending transactions. During this process, all
// invalidated transactions (low nonce, low balance) are deleted.
func (pool *LendingPool) promoteExecutables(accounts []common.Address) {
	start := time.Now()
	log.Debug("start promoteExecutables")
	defer log.Debug("end promoteExecutables", "time", common.PrettyDuration(time.Since(start)))
	// Gather all the accounts potentially needing updates
	if accounts == nil {
		accounts = make([]common.Address, 0, len(pool.queue))
		for addr := range pool.queue {
			accounts = append(accounts, addr)
		}
	}
	// Iterate over all accounts and promote any executable transactions
	for _, addr := range accounts {
		list := pool.queue[addr]
		if list == nil {
			continue // Just in case someone calls with a non existing account
		}
		// Drop all transactions that are deemed too old (low nonce)
		for _, tx := range list.Forward(pool.currentLendingState.GetNonce(addr.Hash())) {
			hash := tx.Hash()
			log.Trace("Removed old queued transaction", "hash", hash)
			delete(pool.all, hash)

		}

		// Gather all executable transactions and promote them
		for _, tx := range list.Ready(pool.pendingState.GetNonce(addr.Hash())) {
			hash := tx.Hash()
			log.Trace("Promoting queued transaction", "hash", hash)
			pool.promoteTx(addr, hash, tx)
		}
		// Drop all transactions over the allowed limit
		if !pool.locals.contains(addr) {
			for _, tx := range list.Cap(int(pool.config.AccountQueue)) {
				hash := tx.Hash()
				delete(pool.all, hash)

				queuedRateLimitCounter.Inc(1)
				log.Trace("Removed cap-exceeding queued transaction", "hash", hash)
			}
		}
		// Delete the entire queue entry if it became empty.
		if list.Empty() {
			delete(pool.queue, addr)
		}
	}
	// If the pending limit is overflown, start equalizing allowances
	pending := uint64(0)
	for _, list := range pool.pending {
		pending += uint64(list.Len())
	}
	if pending > pool.config.GlobalSlots {
		pendingBeforeCap := pending
		// Assemble a spam order to penalize large transactors first
		spammers := prque.New()
		for addr, list := range pool.pending {
			// Only evict transactions from high rollers
			if !pool.locals.contains(addr) && uint64(list.Len()) > pool.config.AccountSlots {
				spammers.Push(addr, float32(list.Len()))
			}
		}
		// Gradually drop transactions from offenders
		offenders := []common.Address{}
		for pending > pool.config.GlobalSlots && !spammers.Empty() {
			// Retrieve the next offender if not local address
			offender, _ := spammers.Pop()
			offenders = append(offenders, offender.(common.Address))

			// Equalize balances until all the same or below threshold
			if len(offenders) > 1 {
				// Calculate the equalization threshold for all current offenders
				threshold := pool.pending[offender.(common.Address)].Len()

				// Iteratively reduce all offenders until below limit or threshold reached
				for pending > pool.config.GlobalSlots && pool.pending[offenders[len(offenders)-2]].Len() > threshold {
					for i := 0; i < len(offenders)-1; i++ {
						list := pool.pending[offenders[i]]
						for _, tx := range list.Cap(list.Len() - 1) {
							// Drop the transaction from the global pools too
							hash := tx.Hash()
							delete(pool.all, hash)

							// Update the account nonce to the dropped transaction
							if nonce := tx.Nonce(); pool.pendingState.GetNonce(offenders[i].Hash()) > nonce {
								pool.pendingState.SetNonce(offenders[i].Hash(), nonce)
							}
							log.Trace("Removed fairness-exceeding pending transaction", "hash", hash)
						}
						pending--
					}
				}
			}
		}
		// If still above threshold, reduce to limit or min allowance
		if pending > pool.config.GlobalSlots && len(offenders) > 0 {
			for pending > pool.config.GlobalSlots && uint64(pool.pending[offenders[len(offenders)-1]].Len()) > pool.config.AccountSlots {
				for _, addr := range offenders {
					list := pool.pending[addr]
					for _, tx := range list.Cap(list.Len() - 1) {
						// Drop the transaction from the global pools too
						hash := tx.Hash()
						delete(pool.all, hash)

						// Update the account nonce to the dropped transaction
						if nonce := tx.Nonce(); pool.pendingState.GetNonce(addr.Hash()) > nonce {
							pool.pendingState.SetNonce(addr.Hash(), nonce)
						}
						log.Trace("Removed fairness-exceeding pending transaction", "hash", hash)
					}
					pending--
				}
			}
		}
		pendingRateLimitCounter.Inc(int64(pendingBeforeCap - pending))
	}
	// If we've queued more transactions than the hard limit, drop oldest ones
	queued := uint64(0)
	for _, list := range pool.queue {
		queued += uint64(list.Len())
	}
	if queued > pool.config.GlobalQueue {
		// Sort all accounts with queued transactions by heartbeat
		addresses := make(addresssByHeartbeat, 0, len(pool.queue))
		for addr := range pool.queue {
			if !pool.locals.contains(addr) { // don't drop locals
				addresses = append(addresses, addressByHeartbeat{addr, pool.beats[addr]})
			}
		}
		sort.Sort(addresses)

		// Drop transactions until the total is below the limit or only locals remain
		for drop := queued - pool.config.GlobalQueue; drop > 0 && len(addresses) > 0; {
			addr := addresses[len(addresses)-1]
			list := pool.queue[addr.address]

			addresses = addresses[:len(addresses)-1]

			// Drop all transactions if they are less than the overflow
			if size := uint64(list.Len()); size <= drop {
				for _, tx := range list.Flatten() {
					pool.removeTx(tx.Hash())
				}
				drop -= size
				queuedRateLimitCounter.Inc(int64(size))
				continue
			}
			// Otherwise drop only last few transactions
			txs := list.Flatten()
			for i := len(txs) - 1; i >= 0 && drop > 0; i-- {
				pool.removeTx(txs[i].Hash())
				drop--
				queuedRateLimitCounter.Inc(1)
			}
		}
	}
}

// demoteUnexecutables removes invalid and processed transactions from the pools
// executable/pending queue and any subsequent transactions that become unexecutable
// are moved back into the future queue.
func (pool *LendingPool) demoteUnexecutables() {
	// Iterate over all accounts and demote any non-executable transactions
	for addr, list := range pool.pending {
		nonce := pool.currentLendingState.GetNonce(addr.Hash())
		log.Debug("demoteUnexecutables", "addr", addr.Hex(), "nonce", nonce)
		// Drop all transactions that are deemed too old (low nonce)
		for _, tx := range list.Forward(nonce) {
			hash := tx.Hash()
			log.Debug("Removed old pending transaction", "hash", hash)
			delete(pool.all, hash)
		}

		// If there's a gap in front, warn (should never happen) and postpone all transactions
		if list.Len() > 0 && list.txs.Get(nonce) == nil {
			for _, tx := range list.Cap(0) {
				hash := tx.Hash()
				log.Warn("Demoting invalidated transaction", "hash", hash)
				pool.enqueueTx(hash, tx)
			}
		}
		// Delete the entire queue entry if it became empty.
		if list.Empty() {
			delete(pool.pending, addr)
			delete(pool.beats, addr)
		}
	}
}

type lendingAccountSet struct {
	accounts map[common.Address]struct{}
	signer   types.LendingSigner
}

// newAccountSet creates a new address set with an associated signer for sender
// derivations.
func newLendingAccountSet(signer types.LendingSigner) *lendingAccountSet {
	return &lendingAccountSet{
		accounts: make(map[common.Address]struct{}),
		signer:   signer,
	}
}

// contains checks if a given address is contained within the set.
func (as *lendingAccountSet) contains(addr common.Address) bool {
	_, exist := as.accounts[addr]
	return exist
}

// containsTx checks if the sender of a given tx is within the set. If the sender
// cannot be derived, this method returns false.
func (as *lendingAccountSet) containsTx(tx *types.LendingTransaction) bool {
	if addr, err := types.LendingSender(as.signer, tx); err == nil {
		return as.contains(addr)
	}
	return false
}

// add inserts a new address into the set to track.
func (as *lendingAccountSet) add(addr common.Address) {
	as.accounts[addr] = struct{}{}
}
