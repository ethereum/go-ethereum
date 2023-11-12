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

// Package localpool implements a transaction pool only for local transactions.
package localpool

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

var (
	errNotLocal = errors.New("non-local transaction added to local txpool")
)

var _ txpool.SubPool = new(LocalPool)

type LocalPool struct {
	allTxs      map[common.Hash]*types.Transaction
	allAccounts map[common.Address]nonceOrderedList

	signer       types.Signer
	currentState *state.StateDB // Current state in the blockchain head
	reserver     txpool.AddressReserver
	chain        BlockChain
}

func NewLocalPool(chain BlockChain, signer types.Signer) (*LocalPool, error) {
	head := chain.CurrentBlock()
	currentState, err := chain.StateAt(head.Root)
	if err != nil {
		return nil, err
	}
	return &LocalPool{
		chain:        chain,
		currentState: currentState,
		signer:       signer,
	}, nil
}

// Filter is a selector used to decide whether a transaction whould be added
// to this particular subpool.
func (l *LocalPool) Filter(tx *types.Transaction) bool {
	// Only disallow blob txs, all other txs should be allowed
	return tx.Type() != types.BlobTxType
}

// Init sets the base parameters of the subpool, allowing it to load any saved
// transactions from disk and also permitting internal maintenance routines to
// start up.
//
// These should not be passed as a constructor argument - nor should the pools
// start by themselves - in order to keep multiple subpools in lockstep with
// one another.
func (l *LocalPool) Init(gasTip *big.Int, head *types.Header, reserve txpool.AddressReserver) error {
	l.reserver = reserve
	// TODO load transactions.rlp
	// todo run reorg procedure
	l.Reset(nil, head)
	return errors.New("not implemented")
}

func (l *LocalPool) Close() error {
	// todo shut down subscription
	return nil
}

// Reset retrieves the current state of the blockchain and ensures the content
// of the transaction pool is valid with regard to the chain state.
func (l *LocalPool) Reset(oldHead, newHead *types.Header) {
	newState, err := l.chain.StateAt(newHead.Root)
	if err != nil {
		log.Error("Could not get new state in LocalPool", "head", newHead.Hash())
	}
	l.currentState = newState
	l.runReorg()
	// todo should I reinsert local transactions that have been mined?
}

func (l *LocalPool) runReorg() {
	for addr, list := range l.allAccounts {
		reorged := list.reorg(l.currentState.GetNonce(addr))
		for _, hash := range reorged {
			delete(l.allTxs, hash)
		}
	}
}

// SetGasTip updates the minimum price, since all transactions should
// be retained, we do nothing here.
func (l *LocalPool) SetGasTip(tip *big.Int) {}

func (l *LocalPool) Has(hash common.Hash) bool {
	_, ok := l.allTxs[hash]
	return ok
}

func (l *LocalPool) Get(hash common.Hash) *types.Transaction {
	return l.allTxs[hash]
}

func (l *LocalPool) Add(txs []*types.Transaction, local bool, sync bool) []error {
	errs := make([]error, len(txs))
	if !local {
		// If the transactions are not local, reject all
		for i := 0; i < len(txs); i++ {
			errs[i] = errNotLocal
		}
		return errs
	}
	for i, tx := range txs {
		errs[i] = l.add(tx)
	}
	return nil
}

func (l *LocalPool) add(tx *types.Transaction) error {
	// Ignore already known transactions
	if _, ok := l.allTxs[tx.Hash()]; ok {
		log.Info("Ignoring already known transaction", "hash", tx.Hash())
		return nil
	}
	sender, err := l.signer.Sender(tx)
	if err != nil {
		return err
	}
	if _, ok := l.allAccounts[sender]; !ok {
		if err := l.reserver(sender, true); err != nil {
			log.Warn("Could not reserve account", "account", sender)
			return err
		}
		l.allAccounts[sender] = make(nonceOrderedList)
	}
	l.allTxs[tx.Hash()] = tx
	l.allAccounts[sender].enqueue(tx)
	return nil
}

// Pending retrieves all currently processable transactions, grouped by origin
// account and sorted by nonce.
func (l *LocalPool) Pending(enforceTips bool) map[common.Address][]*txpool.LazyTransaction {
	pending := make(map[common.Address][]*txpool.LazyTransaction)
	for addr, list := range l.allAccounts {
		var txs []*txpool.LazyTransaction
		list.forAllPending(l.currentState.GetNonce(addr), func(tx *types.Transaction) {
			txs = append(txs, &txpool.LazyTransaction{
				Pool:      l,
				Hash:      tx.Hash(),
				Tx:        tx,
				Time:      tx.Time(),
				GasFeeCap: tx.GasFeeCap(),
				GasTipCap: tx.GasTipCap(),
				Gas:       tx.Gas(),
				BlobGas:   tx.BlobGas(),
			})
		})
		pending[addr] = txs
	}
	return nil
}

// SubscribeTransactions subscribes to new transaction events. The subscriber
// can decide whether to receive notifications only for newly seen transactions
// or also for reorged out ones.
func (l *LocalPool) SubscribeTransactions(ch chan<- core.NewTxsEvent, reorgs bool) event.Subscription {
	// todo impl
	return nil
}

// Nonce returns the next nonce of an account, with all transactions executable
// by the pool already applied on top.
func (l *LocalPool) Nonce(addr common.Address) uint64 {
	nextNonce := l.currentState.GetNonce(addr) + 1
	l.allAccounts[addr].forAllPending(l.currentState.GetNonce(addr), func(tx *types.Transaction) {
		nextNonce++
	})
	return nextNonce
}

// Stats retrieves the current pool stats, namely the number of pending and the
// number of queued (non-executable) transactions.
func (l *LocalPool) Stats() (int, int) {
	var pending, queued = 0, 0
	for addr, list := range l.allAccounts {
		var pCount int
		list.forAllPending(l.currentState.GetNonce(addr), func(tx *types.Transaction) {
			pCount++
		})
		pending += pCount
		queued += len(list) - pCount
	}
	return pending, queued
}

// Content retrieves the data content of the transaction pool, returning all the
// pending as well as queued transactions, grouped by account and sorted by nonce.
func (l *LocalPool) Content() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
	var (
		queued  = make(map[common.Address][]*types.Transaction)
		pending = make(map[common.Address][]*types.Transaction)
	)
	for addr, list := range l.allAccounts {
		p, q := list.content(l.currentState.GetNonce(addr))
		queued[addr] = q
		pending[addr] = p
	}
	return pending, queued
}

// ContentFrom retrieves the data content of the transaction pool, returning the
// pending as well as queued transactions of this address, grouped by nonce.
func (l *LocalPool) ContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
	return l.allAccounts[addr].content(l.currentState.GetNonce(addr))
}

// Locals retrieves the accounts currently considered local by the pool.
func (l *LocalPool) Locals() []common.Address {
	var addresses []common.Address
	for addr := range l.allAccounts {
		addresses = append(addresses, addr)
	}
	return addresses
}

// Status returns the known status (unknown/pending/queued) of a transaction
// identified by their hashes.
func (l *LocalPool) Status(hash common.Hash) txpool.TxStatus {
	return txpool.TxStatusUnknown
}
