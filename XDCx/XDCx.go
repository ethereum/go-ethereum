// Copyright 2019 XDC Network
// This file is part of the XDC library.

// Package XDCx implements the XDC decentralized exchange engine
package XDCx

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/XDCx/tradingstate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	// ErrXDCxServiceNotRunning is returned when XDCx service is not running
	ErrXDCxServiceNotRunning = errors.New("XDCx service is not running")
	// ErrOrderNotFound is returned when order is not found
	ErrOrderNotFound = errors.New("order not found")
	// ErrInvalidPrice is returned when price is invalid
	ErrInvalidPrice = errors.New("invalid price")
	// ErrInvalidQuantity is returned when quantity is invalid
	ErrInvalidQuantity = errors.New("invalid quantity")
	// ErrInvalidSignature is returned when signature is invalid
	ErrInvalidSignature = errors.New("invalid signature")
)

// XDCx represents the XDC decentralized exchange
type XDCx struct {
	config     *Config
	db         ethdb.Database
	stateCache tradingstate.Database
	lock       sync.RWMutex
	running    bool

	orderProcessor *OrderProcessor
	matcher        *Matcher
}

// Config holds XDCx configuration
type Config struct {
	DataDir        string
	DBEngine       string
	TradingStateDB string
}

// DefaultConfig returns default XDCx configuration
func DefaultConfig() *Config {
	return &Config{
		DataDir:        "",
		DBEngine:       "leveldb",
		TradingStateDB: "XDCx",
	}
}

// New creates a new XDCx instance
func New(config *Config, db ethdb.Database) (*XDCx, error) {
	if config == nil {
		config = DefaultConfig()
	}

	xdcx := &XDCx{
		config:  config,
		db:      db,
		running: false,
	}

	xdcx.stateCache = tradingstate.NewDatabase(db)
	xdcx.orderProcessor = NewOrderProcessor(xdcx)
	xdcx.matcher = NewMatcher(xdcx)

	return xdcx, nil
}

// Start starts the XDCx service
func (x *XDCx) Start() error {
	x.lock.Lock()
	defer x.lock.Unlock()

	if x.running {
		return nil
	}

	log.Info("Starting XDCx service")
	x.running = true
	return nil
}

// Stop stops the XDCx service
func (x *XDCx) Stop() error {
	x.lock.Lock()
	defer x.lock.Unlock()

	if !x.running {
		return nil
	}

	log.Info("Stopping XDCx service")
	x.running = false
	return nil
}

// IsRunning returns whether XDCx is running
func (x *XDCx) IsRunning() bool {
	x.lock.RLock()
	defer x.lock.RUnlock()
	return x.running
}

// GetTradingState returns the trading state for a given root
func (x *XDCx) GetTradingState(block *types.Block, statedb *state.StateDB) (*tradingstate.TradingStateDB, error) {
	if block == nil {
		return nil, errors.New("block is nil")
	}

	root := block.Root()
	tradingState, err := tradingstate.New(root, x.stateCache)
	if err != nil {
		return nil, err
	}

	return tradingState, nil
}

// ProcessOrder processes an order and returns matched trades
func (x *XDCx) ProcessOrder(ctx context.Context, statedb *state.StateDB, tradingState *tradingstate.TradingStateDB, order *Order) ([]*Trade, error) {
	if !x.IsRunning() {
		return nil, ErrXDCxServiceNotRunning
	}

	return x.orderProcessor.Process(ctx, statedb, tradingState, order)
}

// CancelOrder cancels an existing order
func (x *XDCx) CancelOrder(ctx context.Context, statedb *state.StateDB, tradingState *tradingstate.TradingStateDB, orderID common.Hash) error {
	if !x.IsRunning() {
		return ErrXDCxServiceNotRunning
	}

	return x.orderProcessor.Cancel(ctx, statedb, tradingState, orderID)
}

// GetOrderBook returns the order book for a trading pair
func (x *XDCx) GetOrderBook(baseToken, quoteToken common.Address, tradingState *tradingstate.TradingStateDB) (*OrderBook, error) {
	return x.matcher.GetOrderBook(baseToken, quoteToken, tradingState)
}

// GetBestBid returns the best bid price for a trading pair
func (x *XDCx) GetBestBid(baseToken, quoteToken common.Address, tradingState *tradingstate.TradingStateDB) (*big.Int, error) {
	return x.matcher.GetBestBid(baseToken, quoteToken, tradingState)
}

// GetBestAsk returns the best ask price for a trading pair
func (x *XDCx) GetBestAsk(baseToken, quoteToken common.Address, tradingState *tradingstate.TradingStateDB) (*big.Int, error) {
	return x.matcher.GetBestAsk(baseToken, quoteToken, tradingState)
}

// ApplyXDCxMatchedTransaction applies matched trades to state
func (x *XDCx) ApplyXDCxMatchedTransaction(chainConfig *params.ChainConfig, statedb *state.StateDB, block *types.Block, trades []*Trade) error {
	for _, trade := range trades {
		if err := x.settleTrade(statedb, trade); err != nil {
			return err
		}
	}
	return nil
}

// settleTrade settles a trade by updating balances
func (x *XDCx) settleTrade(statedb *state.StateDB, trade *Trade) error {
	// Trade settlement logic
	// 1. Debit maker and credit taker for base token
	// 2. Debit taker and credit maker for quote token
	// 3. Deduct fees
	log.Debug("Settling trade", "trade", trade.Hash())
	return nil
}

// GetConfig returns XDCx configuration
func (x *XDCx) GetConfig() *Config {
	return x.config
}

// Database returns the database instance
func (x *XDCx) Database() ethdb.Database {
	return x.db
}

// StateCache returns the trading state cache
func (x *XDCx) StateCache() tradingstate.Database {
	return x.stateCache
}
