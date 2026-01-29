// Copyright 2019 XDC Network
// This file is part of the XDC library.

// Package XDCxlending implements the XDC decentralized lending protocol
package XDCxlending

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/XDCxlending/lendingstate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	// ErrLendingServiceNotRunning is returned when lending service is not running
	ErrLendingServiceNotRunning = errors.New("lending service is not running")
	// ErrLoanNotFound is returned when loan is not found
	ErrLoanNotFound = errors.New("loan not found")
	// ErrInvalidInterestRate is returned when interest rate is invalid
	ErrInvalidInterestRate = errors.New("invalid interest rate")
	// ErrInvalidTerm is returned when term is invalid
	ErrInvalidTerm = errors.New("invalid term")
	// ErrInsufficientCollateral is returned when collateral is insufficient
	ErrInsufficientCollateral = errors.New("insufficient collateral")
)

// XDCxLending represents the XDC decentralized lending protocol
type XDCxLending struct {
	config     *Config
	db         ethdb.Database
	stateCache lendingstate.Database
	lock       sync.RWMutex
	running    bool

	orderProcessor *OrderProcessor
	liquidator     *Liquidator
}

// Config holds XDCxLending configuration
type Config struct {
	DataDir         string
	DBEngine        string
	LendingStateDB  string
	DefaultTerm     uint64
	MinCollateral   *big.Int
	LiquidationRate *big.Int
}

// DefaultConfig returns default XDCxLending configuration
func DefaultConfig() *Config {
	return &Config{
		DataDir:         "",
		DBEngine:        "leveldb",
		LendingStateDB:  "XDCxlending",
		DefaultTerm:     86400 * 30, // 30 days in seconds
		MinCollateral:   big.NewInt(150),  // 150%
		LiquidationRate: big.NewInt(110),  // 110%
	}
}

// New creates a new XDCxLending instance
func New(config *Config, db ethdb.Database) (*XDCxLending, error) {
	if config == nil {
		config = DefaultConfig()
	}

	lending := &XDCxLending{
		config:  config,
		db:      db,
		running: false,
	}

	lending.stateCache = lendingstate.NewDatabase(db)
	lending.orderProcessor = NewOrderProcessor(lending)
	lending.liquidator = NewLiquidator(lending)

	return lending, nil
}

// Start starts the XDCxLending service
func (l *XDCxLending) Start() error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.running {
		return nil
	}

	log.Info("Starting XDCxLending service")
	l.running = true
	return nil
}

// Stop stops the XDCxLending service
func (l *XDCxLending) Stop() error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if !l.running {
		return nil
	}

	log.Info("Stopping XDCxLending service")
	l.running = false
	return nil
}

// IsRunning returns whether XDCxLending is running
func (l *XDCxLending) IsRunning() bool {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.running
}

// GetLendingState returns the lending state for a given root
func (l *XDCxLending) GetLendingState(block *types.Block, statedb *state.StateDB) (*lendingstate.LendingStateDB, error) {
	if block == nil {
		return nil, errors.New("block is nil")
	}

	root := block.Root()
	lendingState, err := lendingstate.New(root, l.stateCache)
	if err != nil {
		return nil, err
	}

	return lendingState, nil
}

// ProcessLendingOrder processes a lending order
func (l *XDCxLending) ProcessLendingOrder(ctx context.Context, statedb *state.StateDB, lendingState *lendingstate.LendingStateDB, order *LendingOrder) ([]*LendingTrade, error) {
	if !l.IsRunning() {
		return nil, ErrLendingServiceNotRunning
	}

	return l.orderProcessor.Process(ctx, statedb, lendingState, order)
}

// CancelLendingOrder cancels an existing lending order
func (l *XDCxLending) CancelLendingOrder(ctx context.Context, statedb *state.StateDB, lendingState *lendingstate.LendingStateDB, orderID common.Hash) error {
	if !l.IsRunning() {
		return ErrLendingServiceNotRunning
	}

	return l.orderProcessor.Cancel(ctx, statedb, lendingState, orderID)
}

// Topup adds collateral to a loan
func (l *XDCxLending) Topup(ctx context.Context, statedb *state.StateDB, lendingState *lendingstate.LendingStateDB, loanID common.Hash, amount *big.Int) error {
	if !l.IsRunning() {
		return ErrLendingServiceNotRunning
	}

	loan := lendingState.GetLoan(loanID)
	if loan == nil {
		return ErrLoanNotFound
	}

	// Add collateral
	loan.CollateralAmount = new(big.Int).Add(loan.CollateralAmount, amount)
	lendingState.UpdateLoan(loan)

	log.Debug("Loan topped up", "loanID", loanID.Hex(), "amount", amount)
	return nil
}

// Repay repays a loan
func (l *XDCxLending) Repay(ctx context.Context, statedb *state.StateDB, lendingState *lendingstate.LendingStateDB, loanID common.Hash) error {
	if !l.IsRunning() {
		return ErrLendingServiceNotRunning
	}

	loan := lendingState.GetLoan(loanID)
	if loan == nil {
		return ErrLoanNotFound
	}

	// Calculate total repayment amount
	interest := l.calculateInterest(loan)
	totalRepayment := new(big.Int).Add(loan.Principal, interest)

	// Update loan status
	loan.Status = LoanStatusRepaid
	lendingState.UpdateLoan(loan)

	log.Debug("Loan repaid", "loanID", loanID.Hex(), "total", totalRepayment)
	return nil
}

// Liquidate liquidates an undercollateralized loan
func (l *XDCxLending) Liquidate(ctx context.Context, statedb *state.StateDB, lendingState *lendingstate.LendingStateDB, loanID common.Hash) error {
	if !l.IsRunning() {
		return ErrLendingServiceNotRunning
	}

	return l.liquidator.Liquidate(ctx, statedb, lendingState, loanID)
}

// CheckLiquidation checks if loans need liquidation
func (l *XDCxLending) CheckLiquidation(ctx context.Context, statedb *state.StateDB, lendingState *lendingstate.LendingStateDB) ([]common.Hash, error) {
	if !l.IsRunning() {
		return nil, ErrLendingServiceNotRunning
	}

	return l.liquidator.CheckLoansForLiquidation(lendingState)
}

// ApplyXDCxLendingMatchedTransaction applies matched lending trades to state
func (l *XDCxLending) ApplyXDCxLendingMatchedTransaction(chainConfig *params.ChainConfig, statedb *state.StateDB, block *types.Block, trades []*LendingTrade) error {
	for _, trade := range trades {
		if err := l.settleLendingTrade(statedb, trade); err != nil {
			return err
		}
	}
	return nil
}

// settleLendingTrade settles a lending trade
func (l *XDCxLending) settleLendingTrade(statedb *state.StateDB, trade *LendingTrade) error {
	log.Debug("Settling lending trade", "trade", trade.Hash())
	return nil
}

// calculateInterest calculates the interest for a loan
func (l *XDCxLending) calculateInterest(loan *Loan) *big.Int {
	// Simple interest calculation: Principal * Rate * Time / (365 * 100)
	interest := new(big.Int).Mul(loan.Principal, loan.InterestRate)
	interest = interest.Mul(interest, big.NewInt(int64(loan.Term)))
	interest = interest.Div(interest, big.NewInt(365*24*3600*100))
	return interest
}

// GetConfig returns XDCxLending configuration
func (l *XDCxLending) GetConfig() *Config {
	return l.config
}

// Database returns the database instance
func (l *XDCxLending) Database() ethdb.Database {
	return l.db
}

// StateCache returns the lending state cache
func (l *XDCxLending) StateCache() lendingstate.Database {
	return l.stateCache
}
