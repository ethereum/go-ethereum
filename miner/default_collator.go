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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/log"
)

type DefaultCollator struct {}

func submitTransactions(bs BlockState, txs *types.TransactionsByPriceAndNonce) bool {
    var shouldAbort bool
	cb := func(err error, receipts []*types.Receipt) bool {
		switch {
		case errors.Is(err, core.ErrGasLimitReached):
			fallthrough
		case errors.Is(err, core.ErrTxTypeNotSupported):
			fallthrough
		case errors.Is(err, core.ErrNonceTooHigh):
			txs.Pop()
		case errors.Is(err, core.ErrNonceTooLow):
			fallthrough
        case errors.Is(err, ErrAbort):
            shouldAbort = true
            return false // don't need to waste time rolling back these transactions when this work will be thrown away anyways.
		case errors.Is(err, nil):
			fallthrough
		default:
			txs.Shift()
		}
		return false
	}

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
		bs.AddTransactions(types.Transactions{tx}, cb)
        if shouldAbort {
            return true
		}
	}
	return false
}

// CollateBlock fills a block based on the highest paying transactions from the
// transaction pool, giving precedence over local transactions.
func (w *DefaultCollator) CollateBlock(bs BlockState, pool Pool) {
	txs, err := pool.Pending(true)
	if err != nil {
		log.Error("could not get pending transactions from the pool", "err", err)
		return
	}
	if len(txs) == 0 {
        return
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
			return
		}
	}
	if len(remoteTxs) > 0 {
		if submitTransactions(bs, types.NewTransactionsByPriceAndNonce(bs.Signer(), remoteTxs, bs.BaseFee())) {
            return
        }
	}

    bs.Commit()

	return
}
