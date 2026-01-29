// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package miner

import (
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	// ErrNotAuthorized is returned when the signer is not authorized
	ErrNotAuthorized = errors.New("signer not authorized")
	
	// ErrWrongDifficulty is returned when difficulty check fails
	ErrWrongDifficulty = errors.New("wrong difficulty")
)

// XDCWorkerConfig contains XDPoS-specific worker configuration
type XDCWorkerConfig struct {
	// Recommit interval for block sealing
	Recommit time.Duration
	
	// GasFloor is the target gas floor for blocks
	GasFloor uint64
	
	// GasCeil is the target gas ceiling for blocks
	GasCeil uint64
	
	// Extra data for blocks
	ExtraData []byte
}

// XDCWorker extends the base worker with XDPoS-specific functionality
type XDCWorker struct {
	config      *params.ChainConfig
	chainConfig *params.ChainConfig
	engine      consensus.Engine
	eth         Backend
	chain       *core.BlockChain

	// Feeds
	pendingLogsFeed event.Feed

	// Channels
	taskCh        chan *types.Block
	startCh       chan struct{}
	exitCh        chan struct{}
	resubmitIntervalCh chan time.Duration

	// State
	running int32 // atomic
	syncing int32 // atomic
	
	// Current work
	mu           sync.RWMutex
	coinbase     common.Address
	extra        []byte
	
	// Pending block
	pendingMu    sync.RWMutex
	pendingBlock *types.Block
	pendingState *state.StateDB
	
	// XDPoS specific
	orderpool   OrderPool
	lendingpool LendingPool
}

// OrderPool interface for XDCx order pool integration
type OrderPool interface {
	Pending() (map[common.Address]types.OrderTransactions, error)
}

// LendingPool interface for XDCx lending pool integration  
type LendingPool interface {
	Pending() (map[common.Address]types.LendingTransactions, error)
}

// Backend wraps all required backend methods for mining
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() TxPool
}

// TxPool interface for transaction pool
type TxPool interface {
	Pending(enforceTips bool) map[common.Address][]*types.Transaction
}

// NewXDCWorker creates a new XDPoS worker
func NewXDCWorker(config *params.ChainConfig, engine consensus.Engine, eth Backend, mux *event.TypeMux, isLocalBlock func(header *types.Header) bool, init bool) *XDCWorker {
	worker := &XDCWorker{
		config:      config,
		chainConfig: config,
		engine:      engine,
		eth:         eth,
		chain:       eth.BlockChain(),
		taskCh:      make(chan *types.Block),
		startCh:     make(chan struct{}, 1),
		exitCh:      make(chan struct{}),
		resubmitIntervalCh: make(chan time.Duration),
	}
	
	if init {
		go worker.mainLoop()
	}
	
	return worker
}

// mainLoop is the main event loop for the worker
func (w *XDCWorker) mainLoop() {
	for {
		select {
		case <-w.startCh:
			w.commitWork()
		case <-w.exitCh:
			return
		}
	}
}

// start begins the mining process
func (w *XDCWorker) start() {
	atomic.StoreInt32(&w.running, 1)
	w.startCh <- struct{}{}
}

// stop halts the mining process
func (w *XDCWorker) stop() {
	atomic.StoreInt32(&w.running, 0)
}

// close terminates all internal goroutines
func (w *XDCWorker) close() {
	close(w.exitCh)
}

// isRunning returns whether the worker is running
func (w *XDCWorker) isRunning() bool {
	return atomic.LoadInt32(&w.running) == 1
}

// setEtherbase sets the etherbase for mining
func (w *XDCWorker) setEtherbase(addr common.Address) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.coinbase = addr
}

// setExtra sets extra data for blocks
func (w *XDCWorker) setExtra(extra []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.extra = extra
}

// pending returns the pending block and state
func (w *XDCWorker) pending() (*types.Block, *state.StateDB) {
	w.pendingMu.RLock()
	defer w.pendingMu.RUnlock()
	
	if w.pendingBlock != nil {
		return w.pendingBlock, w.pendingState.Copy()
	}
	return nil, nil
}

// pendingBlock returns the pending block
func (w *XDCWorker) pendingBlockAndReceipts() (*types.Block, types.Receipts) {
	w.pendingMu.RLock()
	defer w.pendingMu.RUnlock()
	return w.pendingBlock, nil
}

// commitWork generates a new work based on the parent block
func (w *XDCWorker) commitWork() {
	parent := w.chain.CurrentBlock()
	if parent == nil {
		return
	}
	
	w.mu.RLock()
	coinbase := w.coinbase
	extra := w.extra
	w.mu.RUnlock()
	
	if coinbase == (common.Address{}) {
		log.Error("Refusing to mine without etherbase")
		return
	}
	
	// Create new work
	num := parent.Number()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(num, common.Big1),
		GasLimit:   core.CalcGasLimit(parent.GasLimit(), w.config.XDPoS.GasLimitBoundDivisor),
		Extra:      extra,
		Time:       uint64(time.Now().Unix()),
	}
	
	// Set coinbase for XDPoS
	if w.config.XDPoS != nil {
		header.Coinbase = coinbase
	}
	
	// Prepare header with consensus engine
	if err := w.engine.Prepare(w.chain, header); err != nil {
		log.Error("Failed to prepare header for sealing", "err", err)
		return
	}
	
	// Create state
	statedb, err := w.chain.StateAt(parent.Root())
	if err != nil {
		log.Error("Failed to create state", "err", err)
		return
	}
	
	// Fill transactions
	pending := w.eth.TxPool().Pending(true)
	txs := make([]*types.Transaction, 0)
	for _, list := range pending {
		txs = append(txs, list...)
	}
	
	// Apply transactions
	receipts, logs := w.applyTransactions(txs, statedb, header)
	
	// Finalize block with consensus engine
	block, err := w.engine.FinalizeAndAssemble(w.chain, header, statedb, &types.Body{Transactions: txs}, receipts)
	if err != nil {
		log.Error("Failed to finalize block", "err", err)
		return
	}
	
	// Store pending work
	w.pendingMu.Lock()
	w.pendingBlock = block
	w.pendingState = statedb
	w.pendingMu.Unlock()
	
	// Submit to seal
	w.taskCh <- block
	
	log.Info("Commit new sealing work", "number", block.Number(), "txs", len(txs), "logs", len(logs))
}

// applyTransactions applies transactions to the state
func (w *XDCWorker) applyTransactions(txs []*types.Transaction, statedb *state.StateDB, header *types.Header) ([]*types.Receipt, []*types.Log) {
	var (
		receipts []*types.Receipt
		logs     []*types.Log
		gasPool  = new(core.GasPool).AddGas(header.GasLimit)
	)
	
	for _, tx := range txs {
		statedb.Prepare(tx.Hash(), len(receipts))
		
		receipt, err := core.ApplyTransaction(w.config, w.chain, nil, gasPool, statedb, header, tx, &header.GasUsed, *w.chain.GetVMConfig())
		if err != nil {
			continue
		}
		
		receipts = append(receipts, receipt)
		logs = append(logs, receipt.Logs...)
	}
	
	return receipts, logs
}

// setOrderPool sets the order pool for XDCx integration
func (w *XDCWorker) setOrderPool(orderpool OrderPool) {
	w.orderpool = orderpool
}

// setLendingPool sets the lending pool for XDCx integration
func (w *XDCWorker) setLendingPool(lendingpool LendingPool) {
	w.lendingpool = lendingpool
}

// getOrderTransactions gets pending order transactions
func (w *XDCWorker) getOrderTransactions() (types.OrderTransactions, error) {
	if w.orderpool == nil {
		return nil, nil
	}
	
	pending, err := w.orderpool.Pending()
	if err != nil {
		return nil, err
	}
	
	var txs types.OrderTransactions
	for _, list := range pending {
		txs = append(txs, list...)
	}
	return txs, nil
}

// getLendingTransactions gets pending lending transactions
func (w *XDCWorker) getLendingTransactions() (types.LendingTransactions, error) {
	if w.lendingpool == nil {
		return nil, nil
	}
	
	pending, err := w.lendingpool.Pending()
	if err != nil {
		return nil, err
	}
	
	var txs types.LendingTransactions
	for _, list := range pending {
		txs = append(txs, list...)
	}
	return txs, nil
}
