package miner

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner/collator"
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
	exitCh       <-chan struct{}
	pool         collator.Pool
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

func (c *DefaultCollator) workCycle(work collator.BlockCollatorWork) {
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
		collator.FillTransactions(ctx, bs, timer, pendingTxs, locals)
		bs.Commit()
		shouldContinue := false

		select {
		case <-timer.C:
			// timer (likely) elapsed while the block was being filled in FillTransactions
			// if sealing, adjust thre recommit interval upwards.
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
			// waited after filling the block which means recommit interval is too long.
			// if sealing, adjust it down (unless it would go below minRecommit).

			// If mining is running resubmit a new work cycle periodically to pull in
			// higher priced transactions. Disable this overhead for pending blocks.
			c.adjustRecommit(0.0, false)
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

func (c *DefaultCollator) CollateBlocks(pool collator.Pool, blockCh <-chan collator.BlockCollatorWork, exitCh <-chan struct{}) {
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

func (c *DefaultCollator) CollateBlock(bs collator.BlockState, pool collator.Pool) {
	c.pool = pool // TODO weird to set this here
	pendingTxs, _ := pool.Pending(true)
	locals := pool.Locals()
	collator.FillTransactions(context.Background(), bs, nil, pendingTxs, locals)
	bs.Commit()
}

func NewDefaultCollator(recommit time.Duration) *DefaultCollator {
	if recommit < minRecommitInterval {
		log.Warn("Sanitizing miner recommit interval", "provided", recommit, "updated", minRecommitInterval)
		recommit = minRecommitInterval
	}

	return &DefaultCollator{recommit: recommit, minRecommit: recommit}
}
