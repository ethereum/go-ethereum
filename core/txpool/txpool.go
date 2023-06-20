// Copyright 2023 The go-ethereum Authors
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

package txpool

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// TxStatus is the current status of a transaction as seen by the pool.
type TxStatus uint

const (
	TxStatusUnknown TxStatus = iota
	TxStatusQueued
	TxStatusPending
	TxStatusIncluded
)

// BlockChain defines the minimal set of methods needed to back a tx pool with
// a chain. Exists to allow mocking the live chain out of tests.
type BlockChain interface {
	// CurrentBlock returns the current head of the chain.
	CurrentBlock() *types.Header

	// SubscribeChainHeadEvent subscribes to new blocks being added to the chain.
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
}

// TxPool is an aggregator for various transaction specific pools, collectively
// tracking all the transactions deemed interesting by the node. Transactions
// enter the pool when they are received from the network or submitted locally.
// They exit the pool when they are included in the blockchain or evicted due to
// resource constraints.
type TxPool struct {
	subpools []SubPool               // List of subpools for specialized transaction handling
	subs     event.SubscriptionScope // Subscription scope to unscubscribe all on shutdown
	quit     chan chan error         // Quit channel to tear down the head updater
}

// New creates a new transaction pool to gather, sort and filter inbound
// transactions from the network.
func New(gasTip *big.Int, chain BlockChain, subpools []SubPool) (*TxPool, error) {
	// Retrieve the current head so that all subpools and this main coordinator
	// pool will have the same starting state, even if the chain moves forward
	// during initialization.
	head := chain.CurrentBlock()

	pool := &TxPool{
		subpools: subpools,
		quit:     make(chan chan error),
	}
	for i, subpool := range subpools {
		if err := subpool.Init(gasTip, head); err != nil {
			for j := i - 1; j >= 0; j-- {
				subpools[j].Close()
			}
			return nil, err
		}
	}
	go pool.loop(head, chain)
	return pool, nil
}

// Close terminates the transaction pool and all its subpools.
func (p *TxPool) Close() error {
	var errs []error

	// Terminate the reset loop and wait for it to finish
	errc := make(chan error)
	p.quit <- errc
	if err := <-errc; err != nil {
		errs = append(errs, err)
	}

	// Terminate each subpool
	for _, subpool := range p.subpools {
		if err := subpool.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("subpool close errors: %v", errs)
	}
	return nil
}

// loop is the transaction pool's main event loop, waiting for and reacting to
// outside blockchain events as well as for various reporting and transaction
// eviction events.
func (p *TxPool) loop(head *types.Header, chain BlockChain) {
	// Subscribe to chain head events to trigger subpool resets
	var (
		newHeadCh  = make(chan core.ChainHeadEvent)
		newHeadSub = chain.SubscribeChainHeadEvent(newHeadCh)
	)
	defer newHeadSub.Unsubscribe()

	// Track the previous and current head to feed to an idle reset
	var (
		oldHead = head
		newHead = oldHead
	)
	// Consume chain head events and start resets when none is running
	var (
		resetBusy = make(chan struct{}, 1) // Allow 1 reset to run concurrently
		resetDone = make(chan *types.Header)
	)
	var errc chan error
	for errc == nil {
		// Something interesting might have happened, run a reset if there is
		// one needed but none is running. The resetter will run on its own
		// goroutine to allow chain head events to be consumed contiguously.
		if newHead != oldHead {
			// Try to inject a busy marker and start a reset if successful
			select {
			case resetBusy <- struct{}{}:
				// Busy marker injected, start a new subpool reset
				go func(oldHead, newHead *types.Header) {
					for _, subpool := range p.subpools {
						subpool.Reset(oldHead, newHead)
					}
					resetDone <- newHead
				}(oldHead, newHead)

			default:
				// Reset already running, wait until it finishes
			}
		}
		// Wait for the next chain head event or a previous reset finish
		select {
		case event := <-newHeadCh:
			// Chain moved forward, store the head for later consumption
			newHead = event.Block.Header()

		case head := <-resetDone:
			// Previous reset finished, update the old head and allow a new reset
			oldHead = head
			<-resetBusy

		case errc = <-p.quit:
			// Termination requested, break out on the next loop round
		}
	}
	// Notify the closer of termination (no error possible for now)
	errc <- nil
}

// SetGasTip updates the minimum gas tip required by the transaction pool for a
// new transaction, and drops all transactions below this threshold.
func (p *TxPool) SetGasTip(tip *big.Int) {
	for _, subpool := range p.subpools {
		subpool.SetGasTip(tip)
	}
}

// Has returns an indicator whether the pool has a transaction cached with the
// given hash.
func (p *TxPool) Has(hash common.Hash) bool {
	for _, subpool := range p.subpools {
		if subpool.Has(hash) {
			return true
		}
	}
	return false
}

// Get returns a transaction if it is contained in the pool, or nil otherwise.
func (p *TxPool) Get(hash common.Hash) *Transaction {
	for _, subpool := range p.subpools {
		if tx := subpool.Get(hash); tx != nil {
			return tx
		}
	}
	return nil
}

// Add enqueues a batch of transactions into the pool if they are valid. Due
// to the large transaction churn, add may postpone fully integrating the tx
// to a later point to batch multiple ones together.
func (p *TxPool) Add(txs []*Transaction, local bool, sync bool) []error {
	// Split the input transactions between the subpools. It shouldn't really
	// happen that we receive merged batches, but better graceful than strange
	// errors.
	//
	// We also need to track how the transactions were split across the subpools,
	// so we can piece back the returned errors into the original order.
	txsets := make([][]*Transaction, len(p.subpools))
	splits := make([]int, len(txs))

	for i, tx := range txs {
		// Mark this transaction belonging to no-subpool
		splits[i] = -1

		// Try to find a subpool that accepts the transaction
		for j, subpool := range p.subpools {
			if subpool.Filter(tx.Tx) {
				txsets[j] = append(txsets[j], tx)
				splits[i] = j
				break
			}
		}
	}
	// Add the transactions split apart to the individual subpools and piece
	// back the errors into the original sort order.
	errsets := make([][]error, len(p.subpools))
	for i := 0; i < len(p.subpools); i++ {
		errsets[i] = p.subpools[i].Add(txsets[i], local, sync)
	}
	errs := make([]error, len(txs))
	for i, split := range splits {
		// If the transaction was rejected by all subpools, mark it unsupported
		if split == -1 {
			errs[i] = core.ErrTxTypeNotSupported
			continue
		}
		// Find which subpool handled it and pull in the corresponding error
		errs[i] = errsets[split][0]
		errsets[split] = errsets[split][1:]
	}
	return errs
}

// Pending retrieves all currently processable transactions, grouped by origin
// account and sorted by nonce.
func (p *TxPool) Pending(enforceTips bool) map[common.Address][]*types.Transaction {
	txs := make(map[common.Address][]*types.Transaction)
	for _, subpool := range p.subpools {
		for addr, set := range subpool.Pending(enforceTips) {
			txs[addr] = set
		}
	}
	return txs
}

// SubscribeNewTxsEvent registers a subscription of NewTxsEvent and starts sending
// events to the given channel.
func (p *TxPool) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	subs := make([]event.Subscription, len(p.subpools))
	for i, subpool := range p.subpools {
		subs[i] = subpool.SubscribeTransactions(ch)
	}
	return p.subs.Track(event.JoinSubscriptions(subs...))
}

// Nonce returns the next nonce of an account, with all transactions executable
// by the pool already applied on top.
func (p *TxPool) Nonce(addr common.Address) uint64 {
	// Since (for now) accounts are unique to subpools, only one pool will have
	// (at max) a non-state nonce. To avoid stateful lookups, just return the
	// highest nonce for now.
	var nonce uint64
	for _, subpool := range p.subpools {
		if next := subpool.Nonce(addr); nonce < next {
			nonce = next
		}
	}
	return nonce
}

// Stats retrieves the current pool stats, namely the number of pending and the
// number of queued (non-executable) transactions.
func (p *TxPool) Stats() (int, int) {
	var runnable, blocked int
	for _, subpool := range p.subpools {
		run, block := subpool.Stats()

		runnable += run
		blocked += block
	}
	return runnable, blocked
}

// Content retrieves the data content of the transaction pool, returning all the
// pending as well as queued transactions, grouped by account and sorted by nonce.
func (p *TxPool) Content() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
	var (
		runnable = make(map[common.Address][]*types.Transaction)
		blocked  = make(map[common.Address][]*types.Transaction)
	)
	for _, subpool := range p.subpools {
		run, block := subpool.Content()

		for addr, txs := range run {
			runnable[addr] = txs
		}
		for addr, txs := range block {
			blocked[addr] = txs
		}
	}
	return runnable, blocked
}

// ContentFrom retrieves the data content of the transaction pool, returning the
// pending as well as queued transactions of this address, grouped by nonce.
func (p *TxPool) ContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
	for _, subpool := range p.subpools {
		run, block := subpool.ContentFrom(addr)
		if len(run) != 0 || len(block) != 0 {
			return run, block
		}
	}
	return []*types.Transaction{}, []*types.Transaction{}
}

// Locals retrieves the accounts currently considered local by the pool.
func (p *TxPool) Locals() []common.Address {
	// Retrieve the locals from each subpool and deduplicate them
	locals := make(map[common.Address]struct{})
	for _, subpool := range p.subpools {
		for _, local := range subpool.Locals() {
			locals[local] = struct{}{}
		}
	}
	// Flatten and return the deduplicated local set
	flat := make([]common.Address, 0, len(locals))
	for local := range locals {
		flat = append(flat, local)
	}
	return flat
}

// Status returns the known status (unknown/pending/queued) of a transaction
// identified by their hashes.
func (p *TxPool) Status(hash common.Hash) TxStatus {
	for _, subpool := range p.subpools {
		if status := subpool.Status(hash); status != TxStatusUnknown {
			return status
		}
	}
	return TxStatusUnknown
}
