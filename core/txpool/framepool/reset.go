// Copyright 2026 The go-ethereum Authors
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

package framepool

import (
	"container/heap"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
)

// Reset fully re-simulates pending prefixes at the new head. It intentionally
// does not selectively invalidate dependencies or reinject displaced block txs.
// Admission commits are generation-gated while simulation runs without the lock.
func (p *FramePool) Reset(_, newHead *types.Header) {
	p.resetLock.Lock()
	defer p.resetLock.Unlock()
	start := time.Now()
	defer func() { resettimeHist.Update(time.Since(start).Nanoseconds()) }()
	if newHead == nil {
		newHead = p.chain.CurrentBlock()
	}
	p.lock.Lock()
	if p.closed {
		p.lock.Unlock()
		return
	}
	p.generation++
	p.resetDone = make(chan struct{})
	snapshot := make([]*frameTx, 0, len(p.txs))
	for _, entry := range p.txs {
		snapshot = append(snapshot, entry)
	}
	p.lock.Unlock()

	// This pristine handle becomes the pool's read state only after all of its
	// disposable simulation copies have been consumed. Never copy p.state.
	statedb, err := p.chain.StateAt(newHead)
	if err != nil {
		log.Error("Failed to reset framepool state", "err", err)
		p.lock.Lock()
		close(p.resetDone)
		p.resetDone = nil
		p.lock.Unlock()
		return
	}
	results := make([]*simResult, len(snapshot))
	for i, entry := range snapshot {
		if entry.expiryDeadline != nil && *entry.expiryDeadline < newHead.Time {
			continue
		}
		if p.validatePolicy(entry.tx, newHead) != nil {
			continue
		}
		// The nil result marks a rejected prefix. Immutable signatures have
		// already been checked at admission and are not verified again.
		results[i], _ = simulate(p.chain.Config(), newHead, statedb.Copy(), entry.tx, entry.prefix)
	}
	baseFee := eip1559.CalcBaseFee(p.chain.Config(), newHead)

	p.lock.Lock()
	defer p.lock.Unlock()
	defer func() {
		close(p.resetDone)
		p.resetDone = nil
	}()
	for i, entry := range snapshot {
		// Clear, Close or SetGasTip may have removed a snapshot entry while
		// simulations ran. Never resurrect it or release its hold twice.
		if p.all[entry.tx.Hash()] != entry {
			continue
		}
		if results[i] == nil {
			p.remove(entry, true)
			evictedMeter.Mark(1)
		}
	}
	groups := make(map[common.Address][]*frameTx)
	for i, entry := range snapshot {
		if p.all[entry.tx.Hash()] != entry {
			continue
		}
		p.unindexDependencies(entry)
		entry.simResult = results[i]
		entry.effectiveTip = entry.tx.EffectiveGasTipValue(baseFee)
		p.indexDependencies(entry)
		groups[entry.payer] = append(groups[entry.payer], entry)
	}
	heap.Init(&p.evict)
	p.payers = make(map[common.Address]*payerUsage, len(groups))
	for payer, txs := range groups {
		// Keeping the highest-priority prefix is equivalent to removing the
		// lowest-priority txs until both limits fit. Accumulating in this order
		// avoids ever storing an overflowing aggregate reservation.
		slices.SortFunc(txs, func(a, b *frameTx) int { return compareEviction(b, a) })
		usage := &payerUsage{txs: make(map[common.Hash]*frameTx)}
		for i, entry := range txs {
			var total uint256.Int
			_, overflow := total.AddOverflow(&usage.reserved, &entry.maxCost)
			coded := usage.coded
			if entry.codedPaymaster {
				coded++
			}
			if overflow || total.Gt(entry.payerBalance) || coded > MaxPendingTxsUsingNonCanonicalPaymaster {
				for _, victim := range txs[i:] {
					p.remove(victim, true)
					evictedMeter.Mark(1)
				}
				break
			}
			usage.reserved.Set(&total)
			usage.coded = coded
			usage.txs[entry.tx.Hash()] = entry
		}
		if len(usage.txs) != 0 {
			p.payers[payer] = usage
		}
	}
	p.state, p.baseFee = statedb, baseFee
	p.head.Store(newHead)
	p.updateMetrics()
}
