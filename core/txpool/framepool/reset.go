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

// Reset revalidates prefixes affected by a linear head advance, falling back to
// all pending prefixes when dependencies cannot safely describe the transition.
// Admission commits are generation-gated until revalidation and reorg recovery
// finish, while all prefix execution runs outside the pool lock.
func (p *FramePool) Reset(oldHead, newHead *types.Header) {
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
	if oldHead == nil {
		oldHead = p.head.Load()
	}
	snapshot := make([]*frameTx, 0, len(p.txs))
	for _, entry := range p.txs {
		snapshot = append(snapshot, entry)
	}
	p.lock.Unlock()
	defer func() {
		p.lock.Lock()
		close(p.resetDone)
		p.resetDone = nil
		p.lock.Unlock()
	}()
	affected, reinject := p.headChanges(oldHead, newHead)
	path, affectedCount := "selective", len(affected)
	if affected == nil {
		path, affectedCount = "full", len(snapshot)
		resetfullMeter.Mark(1)
	} else {
		resetselectiveMeter.Mark(1)
	}
	var resimulated, removed, recovered int64
	defer func() {
		resetresimulatedMeter.Mark(resimulated)
		resetreinjectedMeter.Mark(recovered)
		var oldNumber any
		if oldHead != nil {
			oldNumber = oldHead.Number
		}
		log.Debug("Reset framepool", "old", oldNumber, "new", newHead.Number, "path", path,
			"affected", affectedCount, "resimulated", resimulated, "removed", removed)
	}()

	// This pristine handle becomes the pool's read state only after all of its
	// disposable simulation copies have been consumed. Never copy p.state.
	statedb, err := p.chain.StateAt(newHead)
	if err != nil {
		log.Error("Failed to reset framepool state", "err", err)
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
		if _, changed := affected[entry.tx.Hash()]; affected != nil && !changed {
			results[i] = entry.simResult
			continue
		}
		// The nil result marks a rejected prefix. Immutable signatures have
		// already been checked at admission and are not verified again.
		resimulated++
		results[i], _ = p.simulate(newHead, statedb.Copy(), entry.tx, entry.prefix)
	}
	baseFee := eip1559.CalcBaseFee(p.chain.Config(), newHead)

	p.lock.Lock()
	for i, entry := range snapshot {
		// Clear, Close or SetGasTip may have removed a snapshot entry while
		// simulations ran. Never resurrect it or release its hold twice.
		if p.all[entry.tx.Hash()] != entry {
			continue
		}
		if results[i] == nil {
			p.remove(entry, true)
			removed++
			evictedMeter.Mark(1)
		}
	}
	groups := make(map[common.Address][]*frameTx)
	for i, entry := range snapshot {
		if p.all[entry.tx.Hash()] != entry {
			continue
		}
		if entry.simResult != results[i] {
			p.unindexDependencies(entry)
			entry.simResult = results[i]
			p.indexDependencies(entry)
		}
		entry.effectiveTip = entry.tx.EffectiveGasTipValue(baseFee)
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
					removed++
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
	p.lock.Unlock()

	// These transactions came from displaced blocks, not an eviction history.
	// Reset still owns the generation, so recovery uses the shared admission
	// path without waiting on its own resetDone channel.
	for _, tx := range reinject {
		if err := p.add(tx, true); err != nil {
			rejectedMeter.Mark(1)
		} else {
			recovered++
		}
	}
}

// headChanges returns nil for the full fallback, or a (possibly empty) affected
// set for a linear advance. Historical traversal is bounded independently from
// both heads; missing history or an excessive depth disables recovery entirely.
func (p *FramePool) headChanges(oldHead, newHead *types.Header) (map[common.Hash]struct{}, []*types.Transaction) {
	if oldHead == nil || newHead == nil {
		log.Debug("Full framepool reset", "reason", "missing head")
		return nil, nil
	}
	old, next := oldHead, newHead
	var discarded, included []*types.Block
	var oldBlock, newBlock *types.Block
	step := func(head, other *types.Header, branch *[]*types.Block, cached **types.Block) *types.Header {
		if len(*branch) == 64 {
			log.Debug("Full framepool reset", "reason", "depth bound", "head", head.Number)
			return nil
		}
		if head.Number.Sign() == 0 {
			log.Debug("Full framepool reset", "reason", "missing common ancestor")
			return nil
		}
		block := *cached
		if block == nil {
			block = p.chain.GetBlock(head.Hash(), head.Number.Uint64())
		}
		if block == nil {
			log.Debug("Full framepool reset", "reason", "missing block", "hash", head.Hash(), "number", head.Number)
			return nil
		}
		*branch = append(*branch, block)
		if block.ParentHash() == other.Hash() {
			return other
		}
		parent := p.chain.GetBlock(block.ParentHash(), block.NumberU64()-1)
		*cached = parent
		if parent == nil {
			log.Debug("Full framepool reset", "reason", "missing block", "hash", block.ParentHash(), "number", block.NumberU64()-1)
			return nil
		}
		return parent.Header()
	}
	for old.Hash() != next.Hash() {
		if next.Number.Cmp(old.Number) >= 0 {
			next = step(next, old, &included, &newBlock)
			if next == nil {
				return nil, nil
			}
		} else {
			old = step(old, next, &discarded, &oldBlock)
			if old == nil {
				return nil, nil
			}
		}
	}
	if len(discarded) != 0 {
		log.Debug("Full framepool reset", "reason", "reorg")
		inclusions := make(map[common.Hash]struct{})
		for _, block := range included {
			for _, tx := range block.Transactions() {
				inclusions[tx.Hash()] = struct{}{}
			}
		}
		var reinject []*types.Transaction
		// Recover in block order. Every candidate still needs public validation,
		// including signatures; block inclusion is not a mempool endorsement.
		for _, block := range slices.Backward(discarded) {
			for _, tx := range block.Transactions() {
				if _, ok := inclusions[tx.Hash()]; !ok && tx.Type() == types.FrameTxType {
					reinject = append(reinject, tx)
				}
			}
		}
		return nil, reinject
	}
	config := p.chain.Config()
	merged := func(head *types.Header) bool { return head.Difficulty == nil || head.Difficulty.Sign() == 0 }
	if config.Rules(oldHead.Number, merged(oldHead), oldHead.Time) != config.Rules(newHead.Number, merged(newHead), newHead.Time) {
		log.Debug("Full framepool reset", "reason", "fork-rule change")
		return nil, nil
	}
	for _, block := range included {
		if block.AccessList() == nil {
			log.Debug("Full framepool reset", "reason", "missing access list", "hash", block.Hash(), "number", block.NumberU64())
			return nil, nil
		}
	}
	affected := make(map[common.Hash]struct{})
	p.lock.RLock()
	defer p.lock.RUnlock()
	for _, block := range included {
		for _, account := range *block.AccessList() {
			var fields accountFields
			if len(account.BalanceChanges) != 0 {
				fields |= dependencyBalance
			}
			if len(account.NonceChanges) != 0 {
				fields |= dependencyNonce
			}
			if len(account.CodeChanges) != 0 {
				fields |= dependencyCode
			}
			for hash, deps := range p.byAccount[account.Address] {
				if deps&fields != 0 {
					affected[hash] = struct{}{}
				}
			}
			for _, change := range account.StorageChanges {
				for hash := range p.bySlot[account.Address][common.Hash(change.Slot.Bytes32())] {
					affected[hash] = struct{}{}
				}
			}
		}
	}
	return affected, nil
}
