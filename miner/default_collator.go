package miner

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	// minRecommitInterval is the minimal time interval to recreate the mining block with
	// any newly arrived transactions.
	minRecommitInterval = 1 * time.Second

	// maxRecommitInterval is the maximum time interval to recreate the mining block with
	// any newly arrived transactions.
	maxRecommitInterval = 15 * time.Second

	// intervalAdjustRatio is the impact a single interval adjustment has on sealing work
	// resubmitting interval.
	intervalAdjustRatio = 0.1

	// intervalAdjustBias is applied during the new resubmit interval calculation in favor of
	// increasing upper limit or decreasing lower limit so that the limit can be reachable.
	intervalAdjustBias = 200 * 1000.0 * 1000.0
)

type DefaultCollator struct {
	recommitMu   sync.Mutex
	recommit     time.Duration
	minRecommit  time.Duration
	miner        MinerState
	exitCh       <-chan struct{}
	pool         Pool
	resubmitHook func(time.Duration, time.Duration) // Method to call upon updating resubmitting interval.
}

// recalcRecommit recalculates the resubmitting interval upon feedback.
func recalcRecommit(minRecommit, prev time.Duration, target float64, inc bool) time.Duration {
	var (
		prevF = float64(prev.Nanoseconds())
		next  float64
	)
	if inc {
		next = prevF*(1-intervalAdjustRatio) + intervalAdjustRatio*(target+intervalAdjustBias)
		max := float64(maxRecommitInterval.Nanoseconds())
		if next > max {
			next = max
		}
	} else {
		next = prevF*(1-intervalAdjustRatio) + intervalAdjustRatio*(target-intervalAdjustBias)
		min := float64(minRecommit.Nanoseconds())
		if next < min {
			next = min
		}
	}
	return time.Duration(int64(next))
}

func (c *DefaultCollator) adjustRecommit(ratio float64, inc bool) {
	c.recommitMu.Lock()
	defer c.recommitMu.Unlock()
	if inc {
		before := c.recommit
		if ratio < 0.1 {
			ratio = 0.1
		}

		target := float64(c.recommit.Nanoseconds()) / ratio
		c.recommit = recalcRecommit(c.minRecommit, c.recommit, target, true)
		log.Trace("Increase miner recommit interval", "from", before, "to", c.recommit)
	} else {
		before := c.recommit
		c.recommit = recalcRecommit(c.minRecommit, c.recommit, float64(c.minRecommit.Nanoseconds()), false)
		log.Trace("Decrease miner recommit interval", "from", before, "to", c.recommit)
	}

	if c.resubmitHook != nil {
		c.resubmitHook(c.minRecommit, c.recommit)
	}
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

		err, receipts := bs.AddTransactions(types.Transactions{tx})
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
			availableGas = header.GasLimit - receipts[0].CumulativeGasUsed

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

func (c *DefaultCollator) workCycle(work BlockCollatorWork) {
	ctx := work.Ctx
	emptyBs := work.Block

	for {
		c.recommitMu.Lock()
		curRecommit := c.recommit
		c.recommitMu.Unlock()
		timer := time.NewTimer(curRecommit)

		bs := emptyBs.Copy()
		pendingTxs, _ := c.pool.Pending(true)
		locals := c.pool.Locals()
		FillTransactions(ctx, bs, timer, pendingTxs, locals)
		bs.Commit()
		shouldContinue := false

		select {
		case <-timer.C:
			select {
			case <-ctx.Done():
				return
			case <-c.exitCh:
				return
			default:
			}

			header := bs.Header()
			gasLimit := header.GasLimit
			gasPool := bs.GasPool()
			ratio := float64(gasLimit-gasPool.Gas()) / float64(gasLimit)

			c.adjustRecommit(ratio, true)
			shouldContinue = true
		default:
		}

		if shouldContinue {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			// If mining is running resubmit a new work cycle periodically to pull in
			// higher priced transactions. Disable this overhead for pending blocks.
			chainConfig := c.miner.ChainConfig()
			if c.miner.IsRunning() && (chainConfig.Clique == nil || chainConfig.Clique.Period > 0) {
				c.adjustRecommit(0.0, false)
			} else {
				return
			}
		case <-c.exitCh:
			return
		}
	}
}

func (c *DefaultCollator) SetRecommit(interval time.Duration) {
	if interval < minRecommitInterval {
		log.Warn("Sanitizing miner recommit interval", "provided", interval, "updated", minRecommitInterval)
		interval = minRecommitInterval
	}

	c.recommitMu.Lock()
	defer c.recommitMu.Unlock()

	c.recommit, c.minRecommit = interval, interval

	if c.resubmitHook != nil {
		c.resubmitHook(c.minRecommit, c.recommit)
	}
}

func (c *DefaultCollator) CollateBlocks(miner MinerState, pool Pool, blockCh <-chan BlockCollatorWork, exitCh <-chan struct{}) {
	c.miner = miner
	c.exitCh = exitCh
	c.pool = pool
	// TODO move this to constructor
	c.recommitMu = sync.Mutex{}

	for {
		select {
		case <-c.exitCh:
			return
		case cycleWork := <-blockCh:
			c.workCycle(cycleWork)
		}
	}
}

func (c *DefaultCollator) CollateBlock(bs BlockState, pool Pool) {
	c.pool = pool // TODO weird to set this here
	pendingTxs, _ := pool.Pending(true)
	locals := pool.Locals()
	FillTransactions(context.Background(), bs, nil, pendingTxs, locals)
	bs.Commit()
}

func NewDefaultCollator(recommit time.Duration) *DefaultCollator {
	if recommit < minRecommitInterval {
		log.Warn("Sanitizing miner recommit interval", "provided", recommit, "updated", minRecommitInterval)
		recommit = minRecommitInterval
	}

	return &DefaultCollator{recommit: recommit, minRecommit: recommit}
}
