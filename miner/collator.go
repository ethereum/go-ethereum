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
	"errors"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// BlockState represents a block-to-be-mined, which is being assembled. A collator
// can add transactions, and finish by calling Commit.
type BlockState interface {
	// AddTransactions attempts to add transactions to the blockstate.
	// The following errors are returned
	//   1) ErrRecommit- the recommit interval elapses
	//	 2) ErrNewHead- a new chainHead is received
	// For both cases, no more transactions will be added to the block on subsequent
	// calls to AddTransactions
	// If AddTransactions returns an ErrNewHead during sealing, the collator should abort
	// immediately and propogate this error to the caller
	// TODO perhaps this should also return a list of (reason, tx_hash) for failing txs
	AddTransactions(txs *types.TransactionsByPriceAndNonce) error
	// Commit signals that the collation is finished, and the block is ready
	// to be sealed. After calling Commit, no more transactions may be added.
	Commit()

	Gas() (remaining uint64)
	Coinbase() common.Address
	BaseFee() *big.Int
	Signer() types.Signer
}

var (
	ErrNewHead = errors.New("new chain head received")
)

// Collator is something that can assemble a block.
type Collator interface {
	CollateBlock(bs BlockState, pool Pool, interrupt *int32) error
}

// Pool is an interface to the transaction pool
type Pool interface {
	Pending(bool) (map[common.Address]types.Transactions, error)
	Locals() []common.Address
}

// blockState is an implementation of BlockState
type blockState struct {
	state     *state.StateDB
	logs      []*types.Log
	worker    *worker
	coinbase  common.Address
	baseFee   *big.Int
	signer    types.Signer
	interrupt *int32
}

// Coinbase returns the miner-address of the block being mined
func (bs *blockState) Coinbase() common.Address {
	return bs.coinbase
}

// Basefee returns the basefee for the current block
func (bs *blockState) BaseFee() *big.Int {
	return bs.baseFee
}

// Signer returns the block-specific signer
func (bs *blockState) Signer() types.Signer {
	return bs.signer
}

func (bs *blockState) Commit() {
	w := bs.worker

	if !w.isRunning() && len(bs.logs) > 0 {
		// We don't push the pendingLogsEvent while we are mining. The reason is that
		// when we are mining, the worker will regenerate a mining block every 3 seconds.
		// In order to avoid pushing the repeated pendingLog, we disable the pending log pushing.

		// make a copy, the state caches the logs and these logs get "upgraded" from pending to mined
		// logs by filling in the block hash when the block was mined by the local miner. This can
		// cause a race condition if a log was "upgraded" before the PendingLogsEvent is processed.
		cpy := make([]*types.Log, len(bs.logs))
		for i, l := range bs.logs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		w.pendingLogsFeed.Send(cpy)
	}
	// Notify resubmit loop to decrease resubmitting interval if current interval is larger
	// than the user-specified one.
	if bs.interrupt != nil {
		w.resubmitAdjustCh <- &intervalAdjust{inc: false}
	}
}

func (bs *blockState) AddTransactions(txs *types.TransactionsByPriceAndNonce) error {
	var (
		w             = bs.worker
		returnErr     error
		coalescedLogs []*types.Log
	)
	for {
		if bs.interrupt != nil && atomic.LoadInt32(bs.interrupt) != commitInterruptNone {
			// Notify resubmit loop to increase resubmitting interval due to too frequent commits.
			if atomic.LoadInt32(bs.interrupt) == commitInterruptResubmit {
				ratio := float64(w.current.header.GasLimit-w.current.gasPool.Gas()) / float64(w.current.header.GasLimit)
				if ratio < 0.1 {
					ratio = 0.1
				}
				w.resubmitAdjustCh <- &intervalAdjust{
					ratio: ratio,
					inc:   true,
				}
			}
			if atomic.LoadInt32(bs.interrupt) == commitInterruptNewHead {
				returnErr = ErrNewHead
			}
			break
		}

		if w.current.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", w.current.gasPool, "want", params.TxGas)
			break
		}
		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()
		if tx == nil {
			break
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		//
		// We use the eip155 signer regardless of the current hf.
		from, _ := types.Sender(w.current.signer, tx)

		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !w.chainConfig.IsEIP155(w.current.header.Number) {
			log.Trace("Ignoring reply protected transaction", "hash", tx.Hash(), "eip155", w.chainConfig.EIP155Block)
			continue
		}

		// Start executing the transaction
		bs.state.Prepare(tx.Hash(), w.current.tcount)
		logs, err := w.commitTransaction(tx, bs.Coinbase())
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
			log.Trace("Skipping account with hight nonce", "sender", from, "nonce", tx.Nonce())
			txs.Pop()

		case errors.Is(err, nil):
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, logs...)
			w.current.tcount++
			txs.Shift()

		case errors.Is(err, core.ErrTxTypeNotSupported):
			// Pop the unsupported transaction without shifting in the next from the account
			log.Trace("Skipping unsupported transaction type", "sender", from, "type", tx.Type())
			txs.Pop()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txs.Shift()
		}
	}
	bs.logs = coalescedLogs

	return returnErr
}

func (bs *blockState) Gas() (remaining uint64) {
	return bs.worker.current.gasPool.Gas()
}

// The DefaultCollator is the 'normal' block collator. It assembles a block
// based on transaction price ordering.
type DefaultCollator struct{}

// CollateBlock fills a block based on the highest paying transactions from the
// transaction pool, giving precedence over local transactions.
func (w *DefaultCollator) CollateBlock(bs BlockState, pool Pool) error {
	recommitFired := false
	txs, err := pool.Pending(true)
	if err != nil {
		log.Error("could not get pending transactions from the pool", "err", err)
		return err
	}
	if len(txs) == 0 {
		return nil
	}
	// Split the pending transactions into locals and remotes
	localTxs, remoteTxs := make(map[common.Address]types.Transactions), txs
	for _, account := range pool.Locals() {
		if accountTxs := remoteTxs[account]; len(accountTxs) > 0 {
			delete(remoteTxs, account)
			localTxs[account] = accountTxs
		}
	}
	if len(localTxs) > 0 {
		if err := bs.AddTransactions(types.NewTransactionsByPriceAndNonce(bs.Signer(), localTxs, bs.BaseFee())); err != nil {
			if err == ErrNewHead {
				return err
			} else {
				recommitFired = true
			}
		}
	}
	if len(remoteTxs) > 0 {
		if err := bs.AddTransactions(types.NewTransactionsByPriceAndNonce(bs.Signer(), remoteTxs, bs.BaseFee())); err != nil {
			if err == ErrNewHead {
				return err
			} else {
				recommitFired = true
			}
		}
	}
	if !recommitFired {
		bs.Commit()
	}
	return nil
}
