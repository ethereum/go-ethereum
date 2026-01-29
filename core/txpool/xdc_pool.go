// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package txpool contains XDPoS-specific transaction pool functionality.
package txpool

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

// XDCOrderPool manages order transactions for XDCx
type XDCOrderPool struct {
	mu       sync.RWMutex
	pending  map[common.Address]types.OrderTransactions
	queue    map[common.Address]types.OrderTransactions
	all      map[common.Hash]*types.OrderTransaction
	maxSize  int
	
	// Event feeds
	txFeed event.Feed
	scope  event.SubscriptionScope
}

// NewXDCOrderPool creates a new order pool
func NewXDCOrderPool(maxSize int) *XDCOrderPool {
	return &XDCOrderPool{
		pending: make(map[common.Address]types.OrderTransactions),
		queue:   make(map[common.Address]types.OrderTransactions),
		all:     make(map[common.Hash]*types.OrderTransaction),
		maxSize: maxSize,
	}
}

// Add adds an order transaction to the pool
func (p *XDCOrderPool) Add(tx *types.OrderTransaction) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	hash := tx.GetHash()
	if _, exists := p.all[hash]; exists {
		return ErrAlreadyKnown
	}
	
	// Check pool size
	if len(p.all) >= p.maxSize {
		return ErrPoolFull
	}
	
	p.all[hash] = tx
	p.pending[tx.UserAddress] = append(p.pending[tx.UserAddress], tx)
	
	log.Debug("Added order transaction", "hash", hash)
	return nil
}

// AddRemotes adds multiple remote order transactions
func (p *XDCOrderPool) AddRemotes(txs []*types.OrderTransaction) []error {
	errs := make([]error, len(txs))
	for i, tx := range txs {
		errs[i] = p.Add(tx)
	}
	return errs
}

// Get retrieves a transaction by hash
func (p *XDCOrderPool) Get(hash common.Hash) *types.OrderTransaction {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.all[hash]
}

// Pending returns all pending order transactions
func (p *XDCOrderPool) Pending() (map[common.Address]types.OrderTransactions, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	pending := make(map[common.Address]types.OrderTransactions)
	for addr, txs := range p.pending {
		pending[addr] = append(types.OrderTransactions{}, txs...)
	}
	return pending, nil
}

// SubscribeTxPreEvent subscribes to new transaction events
func (p *XDCOrderPool) SubscribeTxPreEvent(ch chan<- OrderTxPreEvent) event.Subscription {
	return p.scope.Track(p.txFeed.Subscribe(ch))
}

// Remove removes a transaction from the pool
func (p *XDCOrderPool) Remove(hash common.Hash) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	tx, exists := p.all[hash]
	if !exists {
		return
	}
	
	delete(p.all, hash)
	
	// Remove from pending
	addr := tx.UserAddress
	txs := p.pending[addr]
	for i, t := range txs {
		if t.GetHash() == hash {
			p.pending[addr] = append(txs[:i], txs[i+1:]...)
			break
		}
	}
}

// Clear removes all transactions
func (p *XDCOrderPool) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.pending = make(map[common.Address]types.OrderTransactions)
	p.queue = make(map[common.Address]types.OrderTransactions)
	p.all = make(map[common.Hash]*types.OrderTransaction)
}

// Count returns the number of transactions
func (p *XDCOrderPool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.all)
}

// XDCLendingPool manages lending transactions
type XDCLendingPool struct {
	mu       sync.RWMutex
	pending  map[common.Address]types.LendingTransactions
	all      map[common.Hash]*types.LendingTransaction
	maxSize  int
	
	// Event feeds
	txFeed event.Feed
	scope  event.SubscriptionScope
}

// NewXDCLendingPool creates a new lending pool
func NewXDCLendingPool(maxSize int) *XDCLendingPool {
	return &XDCLendingPool{
		pending: make(map[common.Address]types.LendingTransactions),
		all:     make(map[common.Hash]*types.LendingTransaction),
		maxSize: maxSize,
	}
}

// Add adds a lending transaction to the pool
func (p *XDCLendingPool) Add(tx *types.LendingTransaction) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	hash := tx.Hash()
	if _, exists := p.all[hash]; exists {
		return ErrAlreadyKnown
	}
	
	if len(p.all) >= p.maxSize {
		return ErrPoolFull
	}
	
	p.all[hash] = tx
	p.pending[tx.UserAddress] = append(p.pending[tx.UserAddress], tx)
	
	log.Debug("Added lending transaction", "hash", hash)
	return nil
}

// AddRemotes adds multiple remote lending transactions
func (p *XDCLendingPool) AddRemotes(txs []*types.LendingTransaction) []error {
	errs := make([]error, len(txs))
	for i, tx := range txs {
		errs[i] = p.Add(tx)
	}
	return errs
}

// Pending returns all pending lending transactions
func (p *XDCLendingPool) Pending() (map[common.Address]types.LendingTransactions, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	pending := make(map[common.Address]types.LendingTransactions)
	for addr, txs := range p.pending {
		pending[addr] = append(types.LendingTransactions{}, txs...)
	}
	return pending, nil
}

// SubscribeTxPreEvent subscribes to new transaction events
func (p *XDCLendingPool) SubscribeTxPreEvent(ch chan<- LendingTxPreEvent) event.Subscription {
	return p.scope.Track(p.txFeed.Subscribe(ch))
}

// OrderTxPreEvent is emitted when order tx enters pool
type OrderTxPreEvent struct {
	Tx *types.OrderTransaction
}

// LendingTxPreEvent is emitted when lending tx enters pool
type LendingTxPreEvent struct {
	Tx *types.LendingTransaction
}

// Errors
var (
	ErrAlreadyKnown = &PoolError{"transaction already known"}
	ErrPoolFull     = &PoolError{"transaction pool full"}
)

type PoolError struct {
	message string
}

func (e *PoolError) Error() string {
	return e.message
}
