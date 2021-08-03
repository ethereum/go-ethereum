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

// BlockState represents a block-to-be-mined, which is being assembled.
// A collator can add transactions by calling AddTransactions
type BlockState interface {
	// AddTransactions adds the sequence of transactions to the blockstate. Either all
	// transactions are added, or none of them. In the latter case, the error
	// describes the reason why the txs could not be included.
	// if ErrRecommit, the collator should not attempt to add more transactions to the
	// block and submit the block for sealing.
	// If ErrAbort is returned, the collator should immediately abort and return a
	// value (true) from CollateBlock which indicates to the miner to discard the
	// block
	AddTransactions(sequence types.Transactions) error
	Gas() (remaining uint64)
	Coinbase() common.Address
	BaseFee() *big.Int
	Signer() types.Signer
}

var (
	ErrAbort               = errors.New("miner signalled to abort sealing the current block")
	ErrRecommit            = errors.New("err sealing recommit timer elapsed")
	ErrUnsupportedEIP155Tx = errors.New("encountered eip155 tx when chain doesn't support it")
)

// Collator is something that can assemble a block.
type Collator interface {
	// should add transactions to the pending BlockState.
	// should return true if sealing of the block should be aborted
	CollateBlock(bs BlockState, pool Pool) bool
}

// Pool is an interface to the transaction pool
type Pool interface {
	Pending(bool) (map[common.Address]types.Transactions, error)
	Locals() []common.Address
}

// blockState is an implementation of BlockState
type blockState struct {
	state                 *state.StateDB
	logs                  []*types.Log
	worker                *worker
	coinbase              common.Address
	baseFee               *big.Int
	signer                types.Signer
	interrupt             *int32
	resubmitAdjustHandled bool
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
	if !bs.resubmitAdjustHandled && bs.interrupt != nil {
		w.resubmitAdjustCh <- &intervalAdjust{inc: false}
	}
}

func (bs *blockState) AddTransactions(sequence types.Transactions) error {
	var (
		w           = bs.worker
		snap        = w.current.state.Snapshot()
		err         error
		logs        []*types.Log
		tcount      = w.current.tcount
		startTCount = w.current.tcount
	)
	if bs.resubmitAdjustHandled {
		return ErrRecommit
	}
	for _, tx := range sequence {
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
				err = ErrAbort
			} else {
				err = ErrRecommit
			}
			bs.resubmitAdjustHandled = true
			break
		}
		if w.current.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", w.current.gasPool, "want", params.TxGas)
			err = core.ErrGasLimitReached
			break
		}
		from, _ := types.Sender(w.current.signer, tx)
		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !w.chainConfig.IsEIP155(w.current.header.Number) {
			log.Trace("encountered replay-protected transaction when chain doesn't support replay protection", "hash", tx.Hash(), "eip155", w.chainConfig.EIP155Block)
			err = ErrUnsupportedEIP155Tx
			break
		}
		// Start executing the transaction
		bs.state.Prepare(tx.Hash(), w.current.tcount)

		var txLogs []*types.Log
		txLogs, err = w.commitTransaction(tx, bs.Coinbase())
		if err == nil {
			logs = append(logs, txLogs...)
			tcount++
		} else {
			log.Trace("Tx block inclusion failed", "sender", from, "nonce", tx.Nonce(),
				"type", tx.Type(), "hash", tx.Hash(), "err", err)
			break
		}
	}
	if err != nil {
		bs.state.RevertToSnapshot(snap)

		// remove the txs and receipts that were added
		for i := startTCount; i < tcount; i++ {
			w.current.txs[i] = nil
			w.current.receipts[i] = nil
		}
		w.current.txs = w.current.txs[:startTCount]
		w.current.receipts = w.current.receipts[:startTCount]
	} else {
		bs.logs = append(bs.logs, logs...)
		w.current.tcount = tcount
	}
	return err
}

func (bs *blockState) Gas() (remaining uint64) {
	return bs.worker.current.gasPool.Gas()
}

// The DefaultCollator is the 'normal' block collator. It assembles a block
// based on transaction price ordering.
type DefaultCollator struct{}

func submitTransactions(bs BlockState, txs *types.TransactionsByPriceAndNonce) bool {
	returnVal := false
	for {
		// If we don't have enough gas for any further transactions then we're done
		available := bs.Gas()
		if available < params.TxGas {
			break
		}
		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()
		if tx == nil {
			break
		}
		// Enough space for this tx?
		if available < tx.Gas() {
			txs.Pop()
			continue
		}
		// Error already logged in AddTransactions
		err := bs.AddTransactions(types.Transactions{tx})
		switch {
		case errors.Is(err, core.ErrGasLimitReached):
			fallthrough
		case errors.Is(err, core.ErrTxTypeNotSupported):
			fallthrough
		case errors.Is(err, core.ErrNonceTooHigh):
			txs.Pop()
		case errors.Is(err, ErrAbort):
			returnVal = true
			break
		case errors.Is(err, ErrRecommit):
			break
		case errors.Is(err, core.ErrNonceTooLow):
			fallthrough
		case errors.Is(err, nil):
			fallthrough
		default:
			txs.Shift()
		}
	}

	return returnVal
}

// CollateBlock fills a block based on the highest paying transactions from the
// transaction pool, giving precedence over local transactions.
func (w *DefaultCollator) CollateBlock(bs BlockState, pool Pool) bool {
	txs, err := pool.Pending(true)
	if err != nil {
		log.Error("could not get pending transactions from the pool", "err", err)
		return true
	}
	if len(txs) == 0 {
		return true
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
		if submitTransactions(bs, types.NewTransactionsByPriceAndNonce(bs.Signer(), localTxs, bs.BaseFee())) {
			return true
		}
	}
	if len(remoteTxs) > 0 {
		if submitTransactions(bs, types.NewTransactionsByPriceAndNonce(bs.Signer(), remoteTxs, bs.BaseFee())) {
			return true
		}
	}

	return false
}
