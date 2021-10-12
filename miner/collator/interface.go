package collator

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	// errors which indicate that a given transaction cannot be
	// added at a given block or chain configuration.
	ErrGasLimitReached    = errors.New("gas limit reached")
	ErrNonceTooLow        = errors.New("tx nonce too low")
	ErrNonceTooHigh       = errors.New("tx nonce too high")
	ErrTxTypeNotSupported = errors.New("tx type not supported")
	ErrGasFeeCapTooLow    = errors.New("gas fee cap too low")
	ErrZeroTxs            = errors.New("zero transactions")
	ErrTooManyTxs         = errors.New("applying txs to block would go over the block gas limit")
	// error which encompasses all other reasons a given transaction
	// could not be added to the block.
	ErrStrange = errors.New("strange error")
)

type MinerState interface {
	IsRunning() bool
	ChainConfig() *params.ChainConfig
}

/*
	BlockState represents an under-construction block.  An instance of
	BlockState is passed to CollateBlock where it can be filled with transactions
	via BlockState.AddTransaction() and submitted for sealing via
	BlockState.Commit().
	Operations on a single BlockState instance are not threadsafe.  However,
	instances can be copied with BlockState.Copy().
*/
type BlockState interface {
	/*
		adds a single transaction to the block state
	*/
	AddTransaction(tx *types.Transaction) (*types.Receipt, error)

	/*
		Commits the block to the sealer, return true if the block was successfully committed
		and false if not (block became stale)
	*/
	Commit() bool
	/*
		deep-copy a BlockState
	*/
	Copy() BlockState
	/*
		read-only view of the Ethereum state
	*/
	State() vm.StateReader
	Signer() types.Signer
	/*
		returns a copy of the block header
	*/
	Header() *types.Header
	/*
		the account which will receive the block reward.
	*/
	Etherbase() common.Address
	/*
		Remaining gas in a block
	*/
	GasPool() core.GasPool
}

// Pool is an interface to the transaction pool
type Pool interface {
	Pending(bool) (map[common.Address]types.Transactions, error)
	Locals() []common.Address
}

type BlockCollatorWork struct {
	Ctx   context.Context
	Block BlockState
}

type Collator interface {
	// long-running function executed in it's own go-routine which handles eth1-style block collation.
	// an empty pending block is read from blockCh for each work-cycle and filled with transactions
	// from the pool or elswhere.  exitCh signals client exit.
	// a work-cycle can be interrupted by the arrival of a new canonical chain head block or the client
	// changing the miner's etherbase.
	CollateBlocks(miner MinerState, pool Pool, blockCh <-chan BlockCollatorWork, exitCh <-chan struct{})
	// post-merge block collation which expects the implementation to finish after choosing a single block.
	// the block chosen for proposal is the final blockState that was committed.
	CollateBlock(bs BlockState, pool Pool)
}

// fill a block with as many transactions as possible from the pool
// preferencing locally-originating txs and prioritizing the highest-paying
// txs.
// will stop adding transactions to the block if the context is cancelled, or the timer elapses.
func FillTransactions(ctx context.Context, bs BlockState, timer *time.Timer, pendingTxs map[common.Address]types.Transactions, locals []common.Address) uint {
	header := bs.Header()
	if len(pendingTxs) == 0 {
		return 0
	}
	// Split the pending transactions into locals and remotes
	localTxs, remoteTxs := make(map[common.Address]types.Transactions), pendingTxs
	for _, account := range locals {
		if accountTxs := remoteTxs[account]; len(accountTxs) > 0 {
			delete(remoteTxs, account)
			localTxs[account] = accountTxs
		}
	}
	var txCount uint = 0
	if len(localTxs) > 0 {
		txCount += submitTransactions(ctx, bs, types.NewTransactionsByPriceAndNonce(bs.Signer(), localTxs, header.BaseFee), timer)
	}
	if len(remoteTxs) > 0 {
		txCount += submitTransactions(ctx, bs, types.NewTransactionsByPriceAndNonce(bs.Signer(), remoteTxs, header.BaseFee), timer)
	}

	return txCount
}

func submitTransactions(ctx context.Context, bs BlockState, txs *types.TransactionsByPriceAndNonce, timer *time.Timer) uint {
	header := bs.Header()
	availableGas := header.GasLimit
	var numTxsAdded uint = 0

	for {
		select {
		case <-ctx.Done():
			return numTxsAdded
		default:
		}

		if timer != nil {
			select {
			case <-timer.C:
				return numTxsAdded
			default:
			}
		}

		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()
		if tx == nil {
			break
		}
		// Enough space for this tx?
		if availableGas < tx.Gas() {
			txs.Pop()
			continue
		}
		from, _ := types.Sender(bs.Signer(), tx)

		receipt, err := bs.AddTransaction(tx)
		switch {
		case errors.Is(err, ErrGasLimitReached):
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Trace("Gas limit exceeded for current block", "sender", from)
			txs.Pop()

		case errors.Is(err, ErrNonceTooLow):
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case errors.Is(err, ErrNonceTooHigh):
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Trace("Skipping account with high nonce", "sender", from, "nonce", tx.Nonce())
			txs.Pop()

		case errors.Is(err, nil):
			availableGas = header.GasLimit - receipt.CumulativeGasUsed

			numTxsAdded++
			// Everything ok, collect the logs and shift in the next transaction from the same account
			txs.Shift()

		case errors.Is(err, ErrTxTypeNotSupported):
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

	return numTxsAdded
}
