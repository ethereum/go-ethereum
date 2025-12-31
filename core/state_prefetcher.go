// Copyright 2019 The go-ethereum Authors
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
	"bytes"
	"math/rand"
	"runtime"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"golang.org/x/sync/errgroup"
)

const (
	// heavyTransactionThreshold defines the threshold for classifying a
	// transaction as heavy. As defined, the transaction consumes more than
	// 20% of the block's GasUsed is regarded as heavy.
	//
	// Heavy transactions are prioritized for prefetching to allow additional
	// preparation time.
	heavyTransactionThreshold = 20

	// heavyTransactionPriority defines the probability with which the heavy
	// transactions will be scheduled first for prefetching.
	heavyTransactionPriority = 40
)

// isHeavyTransaction returns an indicator whether the transaction is regarded
// as heavy or not.
func isHeavyTransaction(txGasLimit uint64, blockGasUsed uint64) bool {
	threshold := min(blockGasUsed*heavyTransactionThreshold/100, params.MaxTxGas/2)
	return txGasLimit >= threshold
}

// statePrefetcher is a basic Prefetcher that executes transactions from a block
// on top of the parent state, aiming to prefetch potentially useful state data
// from disk. Transactions are executed in parallel to fully leverage the
// SSD's read performance.
type statePrefetcher struct {
	config *params.ChainConfig // Chain configuration options
	chain  *HeaderChain        // Canonical block chain
}

// newStatePrefetcher initialises a new statePrefetcher.
func newStatePrefetcher(config *params.ChainConfig, chain *HeaderChain) *statePrefetcher {
	return &statePrefetcher{
		config: config,
		chain:  chain,
	}
}

// Prefetch processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb, but any changes are discarded. The
// only goal is to warm the state caches.
func (p *statePrefetcher) Prefetch(block *types.Block, statedb *state.StateDB, cfg vm.Config, interrupt *atomic.Bool) {
	var (
		fails   atomic.Int64
		header  = block.Header()
		signer  = types.MakeSigner(p.config, header.Number, header.Time)
		workers errgroup.Group
		reader  = statedb.Reader()
	)
	workers.SetLimit(max(1, 4*runtime.NumCPU()/5)) // Aggressively run the prefetching

	var (
		processed = make(map[common.Hash]struct{}, len(block.Transactions()))
		heavyTxs  = make(chan *types.Transaction, len(block.Transactions()))
		normalTxs = make(chan *types.Transaction, len(block.Transactions()))
	)
	for _, tx := range block.Transactions() {
		// Note: gasLimit is not equivalent to gasUsed. Ideally, transaction heaviness
		// should be measured using gasUsed. However, gasUsed is unknown prior to
		// execution, so gasLimit is used as the indicator instead. This allows transaction
		// senders to inflate gasLimit to gain higher prefetch priority, but this
		// trade-off is unavoidable.
		if isHeavyTransaction(tx.Gas(), block.GasUsed()) {
			heavyTxs <- tx
		}
		// The heavy transaction will also be emitted with the normal prefetching
		// ordering, depends on in which track it will be selected first.
		normalTxs <- tx
	}
	blockPrefetchHeavyTxsMeter.Mark(int64(len(heavyTxs)))

	fetchTx := func() (*types.Transaction, bool) {
		// Pick the heavy transaction first based on the pre-defined probability
		if rand.Intn(100) < heavyTransactionPriority {
			select {
			case tx := <-heavyTxs:
				return tx, false
			default:
			}
		}
		// No more heavy transaction, or no priority for them, pick the transaction
		// with normal order.
		select {
		case tx := <-normalTxs:
			return tx, false
		default:
			return nil, true
		}
	}
	for {
		next, done := fetchTx()
		if done {
			break
		}
		if _, exists := processed[next.Hash()]; exists {
			continue
		}
		processed[next.Hash()] = struct{}{}

		var (
			stateCpy = statedb.Copy() // closure
			tx       = next           // closure
		)
		workers.Go(func() error {
			// If block precaching was interrupted, abort
			if interrupt != nil && interrupt.Load() {
				return nil
			}
			// Preload the touched accounts and storage slots in advance
			sender, err := types.Sender(signer, tx)
			if err != nil {
				fails.Add(1)
				return nil
			}
			reader.Account(sender)

			if tx.To() != nil {
				account, _ := reader.Account(*tx.To())

				// Preload the contract code if the destination has non-empty code
				if account != nil && !bytes.Equal(account.CodeHash, types.EmptyCodeHash.Bytes()) {
					reader.Code(*tx.To(), common.BytesToHash(account.CodeHash))
				}
			}
			for _, list := range tx.AccessList() {
				reader.Account(list.Address)
				if len(list.StorageKeys) > 0 {
					for _, slot := range list.StorageKeys {
						reader.Storage(list.Address, slot)
					}
				}
			}
			// Execute the message to preload the implicit touched states
			evm := vm.NewEVM(NewEVMBlockContext(header, p.chain, nil), stateCpy, p.config, cfg)

			// Convert the transaction into an executable message and pre-cache its sender
			msg, err := TransactionToMessage(tx, signer, header.BaseFee)
			if err != nil {
				fails.Add(1)
				return nil // Also invalid block, bail out
			}
			// Disable the nonce check
			msg.SkipNonceChecks = true

			// The transaction index is assigned blindly with zero, it's fine
			// for prefetching only.
			stateCpy.SetTxContext(tx.Hash(), 0)

			// We attempt to apply a transaction. The goal is not to execute
			// the transaction successfully, rather to warm up touched data slots.
			if _, err := ApplyMessage(evm, msg, new(GasPool).AddGas(block.GasLimit())); err != nil {
				fails.Add(1)
				return nil // Ugh, something went horribly wrong, bail out
			}
			return nil
		})
	}
	workers.Wait()

	blockPrefetchTxsValidMeter.Mark(int64(len(block.Transactions())) - fails.Load())
	blockPrefetchTxsInvalidMeter.Mark(fails.Load())
	return
}
