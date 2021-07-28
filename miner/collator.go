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
	// AddTransactions adds the sequence of transactions to the blockstate. Either all
	// transactions are added, or none of them. In the latter case, the error
	// describes the reason why the txs could not be included.
	AddTransactions(sequence types.Transactions) error
	// Commit signals that the collation is finished, and the block is ready
	// to be sealed. After calling Commit, no more transactions may be added.
	Commit()

	Gas() (remaining uint64)
	Coinbase() common.Address
	BaseFee() *big.Int
	Signer() types.Signer
}

// Collator is something that can assemble a block.
type Collator interface {
	CollateBlock(bs BlockState, txs types.Transactions, interrupt *int32, isSealing bool) error
}

// Pool is an interface to the transaction pool
type Pool interface {
	Pending(bool) (map[common.Address]types.Transactions, error)
	Locals() []common.Address
}

// blockState is an implementation of BlockState
type blockState struct {
	state    *state.StateDB
	logs     []*types.Log
	worker   *worker
	coinbase common.Address
	baseFee  *big.Int
	signer   types.Signer
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
	// We don't push the pendingLogsEvent while we are mining. The reason is that
	// when we are mining, the worker will regenerate a mining block every 3 seconds.
	// In order to avoid pushing the repeated pendingLog, we disable the pending log pushing.
	if !w.isRunning() && len(bs.logs) > 0 {
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
}

func (bs *blockState) AddTransactions(sequence types.Transactions) error {
	var (
		w      = bs.worker
		snap   = w.current.state.Snapshot()
		err    error
		logs   []*types.Log
		tcount = w.current.tcount
	)
	for _, tx := range sequence {
		if w.current.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", w.current.gasPool, "want", params.TxGas)
			err = core.ErrGasLimitReached
			break
		}
		from, _ := types.Sender(w.current.signer, tx)

		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !w.chainConfig.IsEIP155(w.current.header.Number) {
			log.Trace("Ignoring reply protected transaction", "hash", tx.Hash(), "eip155", w.chainConfig.EIP155Block)
			err = ErrTxProtectionDisabled
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
type DefaultCollator struct {
	pool Pool
}

var (
	ErrResubmitIntervalElapsed = errors.New("recommit interval elapsed")
	ErrNewHead                 = errors.New("new chain head received")
	ErrNoCurrentEnv            = errors.New("missing env for mining")
	ErrTxProtectionDisabled    = errors.New("eip155-compatible tx provided when chain config does not support it")
)

func (w *DefaultCollator) submit(bs BlockState, txs *types.TransactionsByPriceAndNonce, interrupt *int32) error {
	for {
		if interrupt != nil && atomic.LoadInt32(interrupt) != commitInterruptNone {
			if atomic.LoadInt32(interrupt) == commitInterruptResubmit {
				return ErrResubmitIntervalElapsed
			} else {
				return ErrNewHead
			}
		}
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
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		err := bs.AddTransactions(types.Transactions{tx})
		switch {
		case errors.Is(err, core.ErrGasLimitReached):
			fallthrough
		case errors.Is(err, core.ErrTxTypeNotSupported):
			fallthrough
		case errors.Is(err, core.ErrNonceTooHigh):
			txs.Pop()
		case errors.Is(err, core.ErrNonceTooLow):
			fallthrough
		case errors.Is(err, nil):
			fallthrough
		default:
			txs.Shift()
		}
	}
	return nil
}

// CollateBlock fills a block based on the highest paying transactions from the
// transaction pool, giving precedence over 'local' transactions.
func (w *DefaultCollator) CollateBlock(bs BlockState, txs map[common.Address]types.Transactions, interrupt *int32, isSealing bool) error {
	if isSealing {
		// Split the pending transactions into locals and remotes
		localTxs, remoteTxs := make(map[common.Address]types.Transactions), txs
		for _, account := range w.pool.Locals() {
			if accountTxs := remoteTxs[account]; len(accountTxs) > 0 {
				delete(remoteTxs, account)
				localTxs[account] = accountTxs
			}
		}
		if len(localTxs) > 0 {
			if err := w.submit(bs, types.NewTransactionsByPriceAndNonce(bs.Signer(), localTxs, bs.BaseFee()), interrupt); err != nil {
				return err
			}
		}
		if len(remoteTxs) > 0 {
			if err := w.submit(bs, types.NewTransactionsByPriceAndNonce(bs.Signer(), remoteTxs, bs.BaseFee()), interrupt); err != nil {
				return err
			}
		}
	} else {
		// ignore resubmit interval elapse here (only used when sealing)
		w.submit(bs, types.NewTransactionsByPriceAndNonce(bs.Signer(), txs, bs.BaseFee()), nil)
	}
	bs.Commit()
	return nil
}
