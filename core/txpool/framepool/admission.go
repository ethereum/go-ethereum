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
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// ValidateTxBasics checks stateless consensus and public-mempool policy before
// any sender-state access. Prefix shape and budgets precede cryptography.
func (p *FramePool) ValidateTxBasics(tx *types.Transaction) error {
	if _, err := p.classify(tx, p.head.Load()); err != nil {
		return err
	}
	return p.verifySignatures(tx)
}

func (p *FramePool) validatePolicy(tx *types.Transaction, head *types.Header) error {
	if head == nil {
		return ErrHeadChanged
	}
	return txpool.ValidateTransaction(tx, head, p.signer, &txpool.ValidationOptions{
		Config: p.chain.Config(), Accept: 1 << types.FrameTxType,
		MaxSize: txMaxSize, MinTip: p.gasTip.Load().ToBig(),
	})
}

func (p *FramePool) classify(tx *types.Transaction, head *types.Header) (Prefix, error) {
	if err := p.validatePolicy(tx, head); err != nil {
		return Prefix{}, err
	}
	sender := *tx.FrameSender() // Shared validation checked the type and sender.
	return ClassifyPrefix(tx.Frames(), sender, tx.FrameSignatures())
}

func (p *FramePool) verifySignatures(tx *types.Transaction) error {
	// The signer hash is the same canonical FrameSigHash used by
	// core.TransactionToMessage. types.Sender alone does not verify these.
	return types.ValidateFrameTxSignatures(tx.FrameSignatures(), *tx.FrameSender(), p.signer.Hash(tx))
}

// Add validates and immediately integrates the batch. Signatures are verified
// once per admission; prefix simulations always run outside the pool lock.
func (p *FramePool) Add(txs []*types.Transaction, _ bool) []error {
	errs := make([]error, len(txs))
	for i, tx := range txs {
		errs[i] = p.add(tx)
		if errs[i] != nil {
			rejectedMeter.Mark(1)
		}
	}
	return errs
}

func (p *FramePool) add(tx *types.Transaction) error {
	validatedHead := p.head.Load()
	prefix, err := p.classify(tx, validatedHead)
	if err != nil {
		return err
	}
	entry := &frameTx{tx: tx, sender: *tx.FrameSender(), prefix: prefix, index: -1, simResult: new(simResult)}
	if prefix.ExpiryFrame >= 0 {
		entry.expiryDeadline = &entry.prefix.ExpiryDeadline
	}
	if err := p.precheck(entry); err != nil {
		return err
	}
	if _, overflow := entry.maxCost.MulOverflow(uint256.NewInt(tx.Gas()), &entry.feeCap); overflow {
		return core.ErrInsufficientFunds
	}
	if err := p.verifySignatures(tx); err != nil {
		return err
	}
	if p.hasPendingAuth != nil && p.hasPendingAuth(entry.sender) {
		return txpool.ErrInflightTxLimitReached
	}
	for range 2 {
		// A Reset owns the generation until its fresh results are committed.
		// Wait without the pool lock before taking a reconciled head snapshot.
		p.lock.RLock()
		done := p.resetDone
		p.lock.RUnlock()
		if done != nil {
			<-done
		}
		p.lock.RLock()
		if p.closed {
			p.lock.RUnlock()
			return ErrClosed
		}
		if p.resetDone != nil {
			p.lock.RUnlock()
			continue
		}
		head, generation := p.head.Load(), p.generation
		p.lock.RUnlock()

		// Fork rules and the block gas limit may have changed since the basics
		// check. Recheck only on a head change, without verifying signatures.
		err = nil
		if head != validatedHead {
			err = p.validatePolicy(tx, head)
			if err == nil {
				validatedHead = head
			}
		}
		if err == nil {
			var result *simResult
			statedb, stateErr := p.chain.StateAt(head)
			if stateErr != nil {
				err = stateErr
			} else {
				result, err = simulate(p.chain.Config(), head, statedb, tx, prefix)
				entry.simResult = result
			}
		}
		p.lock.Lock()
		if generation != p.generation || p.resetDone != nil {
			p.lock.Unlock()
			continue
		}
		if p.closed {
			p.lock.Unlock()
			return ErrClosed
		}
		if err != nil {
			p.lock.Unlock()
			return err
		}
		err = p.commit(entry)
		p.lock.Unlock()
		return err
	}
	return ErrHeadChanged
}

// precheck cheaply rejects gossip duplicates and uncompetitive transactions
// before signature verification or prefix simulation. These decisions use only
// static transaction fields and the pool's current contents and pricing keys.
// commit repeats them authoritatively after the expensive work.
func (p *FramePool) precheck(entry *frameTx) error {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if p.closed {
		return ErrClosed
	}
	if p.all[entry.tx.Hash()] != nil {
		return txpool.ErrAlreadyKnown
	}
	entry.feeCap.Set(uint256.MustFromBig(entry.tx.GasFeeCap()))
	entry.tipCap.Set(uint256.MustFromBig(entry.tx.GasTipCap()))
	old, err := p.checkReplacement(entry)
	if err != nil {
		return err
	}
	entry.effectiveTip = entry.tx.EffectiveGasTipValue(p.baseFee)
	_, err = p.capacityVictims(entry, old, false)
	return err
}

// checkReplacement is shared by the read-only precheck and the atomic commit.
// The caller holds at least a read lock.
func (p *FramePool) checkReplacement(entry *frameTx) (*frameTx, error) {
	old := p.txs[entry.sender]
	if old != nil {
		if old.tx.Nonce() != entry.tx.Nonce() {
			return nil, ErrSenderPending
		}
		if !p.bumped(&old.feeCap, &entry.feeCap) || !p.bumped(&old.tipCap, &entry.tipCap) {
			return nil, txpool.ErrReplaceUnderpriced
		}
	}
	return old, nil
}

// commit computes all policy decisions before taking a new sender hold, then
// performs only infallible mutations. A replacement retains its existing hold.
func (p *FramePool) commit(entry *frameTx) error {
	if p.all[entry.tx.Hash()] != nil {
		return txpool.ErrAlreadyKnown
	}
	if entry.tipCap.Lt(p.gasTip.Load()) {
		return txpool.ErrTxGasPriceTooLow
	}
	if p.hasPendingAuth != nil && p.hasPendingAuth(entry.sender) {
		return txpool.ErrInflightTxLimitReached
	}
	old, err := p.checkReplacement(entry)
	if err != nil {
		return err
	}
	var exposure uint256.Int
	var coded int
	if usage := p.payers[entry.payer]; usage != nil {
		exposure.Set(&usage.reserved)
		coded = usage.coded
	}
	if old != nil && old.payer == entry.payer {
		exposure.Sub(&exposure, &old.maxCost)
		if old.codedPaymaster {
			coded--
		}
	}
	if _, overflow := exposure.AddOverflow(&exposure, &entry.maxCost); overflow || exposure.Gt(entry.payerBalance) {
		return core.ErrInsufficientFunds
	}
	if entry.codedPaymaster && coded >= MaxPendingTxsUsingNonCanonicalPaymaster {
		return txpool.ErrInflightTxLimitReached
	}
	entry.effectiveTip = entry.tx.EffectiveGasTipValue(p.baseFee)
	victims, err := p.capacityVictims(entry, old, true)
	if err != nil {
		return err
	}
	if old == nil {
		if err := p.reserver.Hold(entry.sender); err != nil {
			return err
		}
	}
	// No fallible operations remain after the hold. In particular capacity
	// selection did not pop the eviction heap or change payer accounting.
	for _, victim := range victims {
		p.remove(victim, true)
		evictedMeter.Mark(1)
	}
	if old != nil {
		p.remove(old, false)
		replacedMeter.Mark(1)
	}
	p.insert(entry)
	p.updateMetrics()
	acceptedMeter.Mark(1)
	accepted := core.NewTxsEvent{Txs: []*types.Transaction{entry.tx}}
	p.discoverFeed.Send(accepted)
	p.insertFeed.Send(accepted)
	return nil
}

func (p *FramePool) bumped(old, next *uint256.Int) bool {
	if next.Cmp(old) <= 0 {
		return false
	}
	// Cross-multiply to avoid rounding a fractional percentage bump down and
	// to avoid wrapping even for a uint64 configuration or 256-bit fees.
	factor := new(big.Int).Add(new(big.Int).SetUint64(p.config.PriceBump), big.NewInt(100))
	threshold := new(big.Int).Mul(old.ToBig(), factor)
	return new(big.Int).Mul(next.ToBig(), big.NewInt(100)).Cmp(threshold) >= 0
}

// capacityVictims checks that entry outranks every required eviction candidate.
// collect is false for the early check, avoiding allocation of a victim list.
func (p *FramePool) capacityVictims(entry, old *frameTx, collect bool) ([]*frameTx, error) {
	used := p.slots
	if old != nil {
		used -= numSlots(old.tx)
	}
	needed := numSlots(entry.tx)
	if needed <= p.config.GlobalSlots-used {
		return nil, nil
	}
	needed -= p.config.GlobalSlots - used
	if len(p.evict) == 0 {
		return nil, txpool.ErrUnderpriced
	}
	cursor := evictionCursor{heap: p.evict, front: []int{0}}
	var victims []*frameTx
	for needed > 0 {
		victim := cursor.next()
		if victim == nil {
			return nil, txpool.ErrUnderpriced
		}
		if victim == old {
			continue
		}
		if comparePriority(entry, victim) <= 0 {
			return nil, txpool.ErrUnderpriced
		}
		if collect {
			victims = append(victims, victim)
		}
		needed -= min(needed, numSlots(victim.tx))
	}
	return victims, nil
}
