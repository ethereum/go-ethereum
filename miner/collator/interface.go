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
	ErrAlreadyCommitted = errors.New("can't mutate BlockState after calling Commit()")

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
		adds a single transaction to the blockState.  Returned errors include ..,..,..
		which signify that the transaction was invalid for the current EVM/chain state.
		ErrRecommit signals that the recommit timer has elapsed.
		ErrNewHead signals that the client has received a new canonical chain head.
		All subsequent calls to AddTransaction fail if either newHead or the recommit timer
		have occured.
		If the recommit interval has elapsed, the BlockState can still be committed to the sealer.
	*/
	AddTransaction(tx *types.Transaction) (*types.Receipt, error)

	/*
		returns true if the Block has been made the current sealing block.
		returns false if the newHead interrupt has been triggered.
		can also return false if the BlockState is no longer valid (the call to CollateBlock
		which the original instance was passed has returned).
	*/
	Commit() bool
	Copy() BlockState
	State() vm.StateReader
	Signer() types.Signer
	Header() *types.Header
	/*
		the account which will receive the block reward.
	*/
	Etherbase() common.Address
	GasPool() core.GasPool
}

type CollatorAPI interface {
	Version() string
	Service() interface{}
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
	CollateBlocks(miner MinerState, pool Pool, blockCh <-chan BlockCollatorWork, exitCh <-chan struct{})
	CollateBlock(bs BlockState, pool Pool)
}

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
