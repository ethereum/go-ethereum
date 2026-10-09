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
	"errors"
	"fmt"
	"math/big"
	"slices"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// extend creates either branch of a synthetic chain. Every snapshot has a unique
// root, including competing blocks at the same height. A nil access list models
// unavailable BAL data; an empty constructed list models an unrelated block.
func (c *testBlockChain) extend(parent *types.Header, timestamp uint64, txs []*types.Transaction, accesses *bal.ConstructionBlockAccessList, change func(*state.StateDB)) *types.Header {
	c.lock.Lock()
	defer c.lock.Unlock()
	next := types.CopyHeader(parent)
	next.ParentHash = parent.Hash()
	next.Number.Add(next.Number, common.Big1)
	next.Root = common.BigToHash(new(big.Int).SetUint64(uint64(len(c.states) + 1)))
	next.Time = timestamp
	if next.SlotNumber != nil {
		*next.SlotNumber++
	}
	statedb := c.states[parent.Root].Copy()
	if change != nil {
		change(statedb)
	}
	block := types.NewBlockWithHeader(next).WithBody(types.Body{Transactions: txs})
	if accesses != nil {
		list := accesses.ToEncodingObj()
		hash := list.Hash()
		next.BlockAccessListHash = &hash
		block = types.NewBlockWithHeader(next).WithBody(types.Body{Transactions: txs}).WithAccessList(list)
	}
	c.states[next.Root], c.blocks[next.Hash()], c.head = statedb, block, next
	return next
}

func assertSimulations(t *testing.T, p *FramePool, want uint64) {
	t.Helper()
	if got := p.simulations.Swap(0); got != want {
		t.Fatalf("prefix simulations: got %d, want %d", got, want)
	}
}

func TestSelectiveResetSenderSlots(t *testing.T) {
	slot, otherSlot := common.HexToHash("0x42"), common.HexToHash("0x43")
	pool, chain, other := setupFramePool(t, 10, func(s *state.StateDB) {
		// A nonzero slot makes the verifier take an invalid jump; zero approves.
		code := approveCode(program.New().Push(slot).Op(vm.SLOAD).Push(0).Op(vm.JUMPI), types.ApproveExecutionAndPayment)
		s.SetCode(poolAddress(t, 1), code, tracing.CodeChangeUnspecified)
	})
	dependent := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
	unrelated := signedPoolTx(t, 2, 0, 0, 20, 2, nil, nil)
	addPoolTx(t, pool, dependent, nil)
	addPoolTx(t, pool, unrelated, nil)
	assertSimulations(t, pool, 2)
	old := chain.CurrentBlock()
	accesses := bal.NewConstructionBlockAccessList()
	accesses.StorageWrite(1, poolAddress(t, 1), otherSlot, common.HexToHash("0x01"))
	accesses.StorageRead(poolAddress(t, 1), slot)
	next := chain.extend(old, 101, nil, accesses, func(s *state.StateDB) {
		s.SetState(poolAddress(t, 1), otherSlot, common.HexToHash("0x01"))
	})
	pool.Reset(old, next)
	assertSimulations(t, pool, 0)
	assertLive(t, pool, dependent, true)
	accesses = bal.NewConstructionBlockAccessList()
	accesses.StorageWrite(1, poolAddress(t, 1), slot, common.HexToHash("0x01"))
	last := chain.extend(next, 102, nil, accesses, func(s *state.StateDB) {
		s.SetState(poolAddress(t, 1), slot, common.HexToHash("0x01"))
	})
	pool.Reset(next, last)
	assertSimulations(t, pool, 1)
	assertLive(t, pool, dependent, false)
	assertLive(t, pool, unrelated, true)
	assertReleased(t, other, poolAddress(t, 1))
	assertFramePoolConsistent(t, pool)
}

func TestSelectiveResetPayerChanges(t *testing.T) {
	for _, coded := range []bool{false, true} {
		t.Run(fmt.Sprintf("code=%v", coded), func(t *testing.T) {
			pool, chain, other := setupFramePool(t, 10, nil)
			var sponsored []*types.Transaction
			for id := 1; id <= 3; id++ {
				tx := signedPoolTx(t, id, 20, 0, 20, uint64(id), nil, nil)
				sponsored = append(sponsored, tx)
				addPoolTx(t, pool, tx, nil)
			}
			unrelated := signedPoolTx(t, 4, 0, 0, 20, 2, nil, nil)
			addPoolTx(t, pool, unrelated, nil)
			assertSimulations(t, pool, 4)
			payer := poolAddress(t, 20)
			balance := new(uint256.Int).Add(txCost(sponsored[1]), txCost(sponsored[2]))
			code := approveCode(program.New(), types.ApprovePayment)
			accesses := bal.NewConstructionBlockAccessList()
			if coded {
				accesses.CodeChange(payer, 1, code)
			} else {
				accesses.BalanceChange(1, payer, balance)
			}
			old := chain.CurrentBlock()
			next := chain.extend(old, 101, nil, accesses, func(s *state.StateDB) {
				if coded {
					s.SetCode(payer, code, tracing.CodeChangeUnspecified)
				} else {
					s.SetBalance(payer, balance, tracing.BalanceChangeUnspecified)
				}
			})
			pool.Reset(old, next)
			// A sponsor balance change is revalidated by payer accounting alone.
			sims := uint64(0)
			if coded {
				sims = 3
			}
			assertSimulations(t, pool, sims)
			for i, tx := range sponsored {
				want := i >= 1
				if coded {
					want = i == 2
				}
				assertLive(t, pool, tx, want)
				if !want {
					assertReleased(t, other, *tx.FrameSender())
				}
			}
			assertLive(t, pool, unrelated, true)
			assertFramePoolConsistent(t, pool)
		})
	}
}

// Balance changes to a codeless sponsor shared by many pending transactions are
// revalidated by payer accounting alone, never by re-executing their prefixes.
func TestSelectiveResetSharedSponsorBalance(t *testing.T) {
	pool, chain, other := setupFramePool(t, 32, nil)
	sponsor := poolAddress(t, 20)
	var sponsored []*types.Transaction
	for id := 1; id <= 16; id++ {
		tx := signedPoolTx(t, id, 20, 0, 20, uint64(id), nil, nil)
		sponsored = append(sponsored, tx)
		addPoolTx(t, pool, tx, nil)
	}
	assertSimulations(t, pool, 16)
	setBalance := func(parent *types.Header, timestamp uint64, balance *uint256.Int) *types.Header {
		accesses := bal.NewConstructionBlockAccessList()
		accesses.BalanceChange(1, sponsor, balance)
		return chain.extend(parent, timestamp, nil, accesses, func(s *state.StateDB) {
			s.SetBalance(sponsor, balance, tracing.BalanceChangeUnspecified)
		})
	}
	old := chain.CurrentBlock()
	next := setBalance(old, 101, uint256.NewInt(1e18+1))
	pool.Reset(old, next)
	assertSimulations(t, pool, 0)
	for _, tx := range sponsored {
		assertLive(t, pool, tx, true)
	}
	assertFramePoolConsistent(t, pool)

	// The sponsor now covers only the ten highest priorities.
	balance := new(uint256.Int)
	for _, tx := range sponsored[6:] {
		balance.Add(balance, txCost(tx))
	}
	last := setBalance(next, 102, balance)
	pool.Reset(next, last)
	assertSimulations(t, pool, 0)
	for i, tx := range sponsored {
		assertLive(t, pool, tx, i >= 6)
		if i < 6 {
			assertReleased(t, other, *tx.FrameSender())
		}
	}
	assertFramePoolConsistent(t, pool)
}

// A balance-only change that flips a dependency account's EIP-161 emptiness can
// change account-creation charges and EXTCODEHASH, so it requires simulation.
func TestSelectiveResetBalanceEmptiness(t *testing.T) {
	for _, fund := range []bool{false, true} {
		t.Run(fmt.Sprintf("fund=%v", fund), func(t *testing.T) {
			pool, chain, _ := setupFramePool(t, 10, nil)
			// Sender 40 is unfunded and empty; sender 1 holds a balance.
			flipping, balance := poolAddress(t, 1), new(uint256.Int)
			flipped := signedPoolTx(t, 1, 20, 0, 20, 2, nil, nil)
			if fund {
				flipping, balance = poolAddress(t, 40), uint256.NewInt(1)
				flipped = signedPoolTx(t, 40, 20, 0, 20, 2, nil, nil)
			}
			steady := signedPoolTx(t, 2, 20, 0, 20, 2, nil, nil)
			addPoolTx(t, pool, flipped, nil)
			addPoolTx(t, pool, steady, nil)
			assertSimulations(t, pool, 2)
			accesses := bal.NewConstructionBlockAccessList()
			accesses.BalanceChange(1, flipping, balance)
			accesses.BalanceChange(1, poolAddress(t, 2), uint256.NewInt(1))
			old := chain.CurrentBlock()
			next := chain.extend(old, 101, nil, accesses, func(s *state.StateDB) {
				s.SetBalance(flipping, balance, tracing.BalanceChangeUnspecified)
				s.SetBalance(poolAddress(t, 2), uint256.NewInt(1), tracing.BalanceChangeUnspecified)
			})
			pool.Reset(old, next)
			assertSimulations(t, pool, 1)
			assertLive(t, pool, steady, true)
			assertFramePoolConsistent(t, pool)
		})
	}
}

func TestSelectiveResetUnrelatedAndFieldMasks(t *testing.T) {
	helper := poolAddress(t, 20)
	pool, chain, _ := setupFramePool(t, 10, func(s *state.StateDB) {
		s.SetCode(helper, []byte{byte(vm.STOP)}, tracing.CodeChangeUnspecified)
		code := approveCode(program.New().Call(nil, helper, 0, 0, 0, 0, 0).Op(vm.POP), types.ApproveExecutionAndPayment)
		s.SetCode(poolAddress(t, 1), code, tracing.CodeChangeUnspecified)
	})
	tx := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
	addPoolTx(t, pool, tx, nil)
	assertSimulations(t, pool, 1)
	old := chain.CurrentBlock()
	accesses := bal.NewConstructionBlockAccessList()
	accesses.BalanceChange(1, poolAddress(t, 30), uint256.NewInt(1))
	accesses.NonceChange(poolAddress(t, 30), 1, 1)
	accesses.CodeChange(poolAddress(t, 30), 1, []byte{byte(vm.STOP)})
	accesses.StorageWrite(1, poolAddress(t, 30), common.Hash{}, common.HexToHash("0x01"))
	// The helper is a code-only dependency. Its balance and nonce are irrelevant.
	accesses.BalanceChange(1, helper, uint256.NewInt(1))
	accesses.NonceChange(helper, 1, 1)
	next := chain.extend(old, 101, nil, accesses, func(s *state.StateDB) {
		s.SetBalance(helper, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
		s.SetNonce(helper, 1, tracing.NonceChangeUnspecified)
	})
	pool.Reset(old, next)
	assertSimulations(t, pool, 0)
	assertLive(t, pool, tx, true)
	accesses = bal.NewConstructionBlockAccessList()
	accesses.CodeChange(helper, 1, []byte{byte(vm.INVALID)})
	last := chain.extend(next, 102, nil, accesses, func(s *state.StateDB) {
		s.SetCode(helper, []byte{byte(vm.INVALID)}, tracing.CodeChangeUnspecified)
	})
	pool.Reset(next, last)
	assertSimulations(t, pool, 1)
	assertLive(t, pool, tx, false)
	assertFramePoolConsistent(t, pool)
}

func TestSelectiveResetCoalescedHeads(t *testing.T) {
	pool, chain, _ := setupFramePool(t, 10, nil)
	var txs []*types.Transaction
	for id := 1; id <= 3; id++ {
		tx := signedPoolTx(t, id, 0, 0, 20, 2, nil, nil)
		txs = append(txs, tx)
		addPoolTx(t, pool, tx, nil)
	}
	assertSimulations(t, pool, 3)
	old, next := chain.CurrentBlock(), chain.CurrentBlock()
	for id := 1; id <= 2; id++ {
		accesses := bal.NewConstructionBlockAccessList()
		accesses.NonceChange(poolAddress(t, id), 1, 1)
		next = chain.extend(next, 100+uint64(id), nil, accesses, func(s *state.StateDB) {
			s.SetNonce(poolAddress(t, id), 1, tracing.NonceChangeUnspecified)
		})
	}
	pool.Reset(old, next)
	assertSimulations(t, pool, 2)
	for i, tx := range txs {
		assertLive(t, pool, tx, i == 2)
	}
	assertFramePoolConsistent(t, pool)
}

func TestSelectiveResetFallback(t *testing.T) {
	for _, reason := range []string{"missing-list", "missing-intermediate-list", "missing-block", "fork", "block-fork", "linear-depth"} {
		t.Run(reason, func(t *testing.T) {
			pool, chain, _ := setupFramePool(t, 10, nil)
			if reason == "fork" {
				activation := uint64(101)
				chain.config.OsakaTime = &activation
			}
			if reason == "block-fork" {
				chain.config.EIP158Block = big.NewInt(2)
			}
			valid := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
			invalidated := signedPoolTx(t, 2, 0, 0, 20, 2, nil, nil)
			addPoolTx(t, pool, valid, nil)
			addPoolTx(t, pool, invalidated, nil)
			assertSimulations(t, pool, 2)
			old := chain.CurrentBlock()
			accesses := bal.NewConstructionBlockAccessList()
			if reason == "missing-list" || reason == "missing-intermediate-list" {
				accesses = nil
			}
			next := chain.extend(old, 101, nil, accesses, func(s *state.StateDB) {
				s.SetNonce(poolAddress(t, 2), 1, tracing.NonceChangeUnspecified)
			})
			if reason == "missing-intermediate-list" {
				next = chain.extend(next, 102, nil, bal.NewConstructionBlockAccessList(), nil)
			}
			if reason == "missing-block" {
				chain.lock.Lock()
				delete(chain.blocks, next.Hash())
				chain.lock.Unlock()
			}
			if reason == "linear-depth" {
				for range 64 {
					next = chain.extend(next, next.Time+1, nil, bal.NewConstructionBlockAccessList(), nil)
				}
			}
			pool.Reset(old, next)
			assertSimulations(t, pool, 2)
			assertLive(t, pool, valid, true)
			assertLive(t, pool, invalidated, false)
			assertFramePoolConsistent(t, pool)
		})
	}
}

func TestSelectiveResetExpiry(t *testing.T) {
	pool, chain, other := setupFramePool(t, 10, nil)
	deadline := uint64(101)
	tx := signedPoolTx(t, 1, 0, 0, 20, 2, &deadline, nil)
	addPoolTx(t, pool, tx, nil)
	assertSimulations(t, pool, 1)
	old := chain.CurrentBlock()
	// Force equality through an affected dependency, not merely the cached path.
	// The verifier's code is rewritten unchanged, which still requires simulation.
	accesses := bal.NewConstructionBlockAccessList()
	accesses.CodeChange(params.FrameTxExpiryVerifier, 1, params.FrameTxExpiryVerifierCode)
	next := chain.extend(old, deadline, nil, accesses, nil)
	pool.Reset(old, next)
	assertSimulations(t, pool, 1)
	assertLive(t, pool, tx, true)
	last := chain.extend(next, deadline+1, nil, nil, nil) // Full fallback must skip expired prefixes too.
	pool.Reset(next, last)
	assertSimulations(t, pool, 0)
	assertLive(t, pool, tx, false)
	assertReleased(t, other, poolAddress(t, 1))
	assertFramePoolConsistent(t, pool)
}

func TestResetReorgPublicReinjection(t *testing.T) {
	pool, chain, other := setupFramePool(t, 10, func(s *state.StateDB) {
		s.SetCode(poolAddress(t, 2), approveCode(program.New().Op(vm.NUMBER, vm.POP), types.ApproveExecutionAndPayment), tracing.CodeChangeUnspecified)
	})
	good := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
	private := signedPoolTx(t, 2, 0, 0, 20, 2, nil, nil)
	included := signedPoolTx(t, 3, 0, 0, 20, 2, nil, nil)
	forged := signedPoolTx(t, 4, 0, 0, 20, 2, nil, nil)
	inner := poolTxData(forged)
	inner.Signatures[0].Signature = make([]byte, 65)
	forged = types.NewTx(inner)
	overbudget := signedPoolTx(t, 5, 0, 0, 20, 2, nil, func(tx *types.FrameTx) { tx.Frames[0].GasLimits.Execution = MaxVerifyGas + 1 })
	// Neither a replaced transaction nor a cleared transaction is recoverable
	// without a displaced block containing it.
	replaced := signedPoolTx(t, 6, 0, 0, 20, 2, nil, nil)
	addPoolTx(t, pool, replaced, nil)
	addPoolTx(t, pool, signedPoolTx(t, 6, 0, 0, 30, 3, nil, nil), nil)
	cleared := signedPoolTx(t, 7, 0, 0, 20, 2, nil, nil)
	addPoolTx(t, pool, cleared, nil)
	pool.Clear()
	addPoolTx(t, pool, good, nil)
	ancestor := chain.CurrentBlock()
	old := chain.extend(ancestor, 101, []*types.Transaction{good, private, included, forged, overbudget}, nil, func(s *state.StateDB) {
		s.SetNonce(poolAddress(t, 1), 1, tracing.NonceChangeUnspecified)
	})
	pool.Reset(ancestor, old)
	assertLive(t, pool, good, false)
	assertReleased(t, other, poolAddress(t, 1))
	newHead := chain.extend(ancestor, 102, []*types.Transaction{included}, bal.NewConstructionBlockAccessList(), nil)
	waitAnnounced(t, pool)
	discover, insert := make(chan core.NewTxsEvent, 10), make(chan core.NewTxsEvent, 10)
	subDiscover := pool.SubscribeTransactions(discover, false)
	subInsert := pool.SubscribeTransactions(insert, true)
	defer subDiscover.Unsubscribe()
	defer subInsert.Unsubscribe()
	pool.Reset(old, newHead)
	waitAnnounced(t, pool)
	assertLive(t, pool, good, true)
	for _, tx := range []*types.Transaction{private, included, forged, overbudget, replaced, cleared} {
		assertLive(t, pool, tx, false)
	}
	select {
	case event := <-insert:
		if len(event.Txs) != 1 || event.Txs[0].Hash() != good.Hash() {
			t.Fatalf("unexpected reinjection event: %v", event.Txs)
		}
	default:
		t.Fatal("missing reorg-inclusive event")
	}
	select {
	case event := <-insert:
		t.Fatalf("unexpected extra reinjection event: %v", event.Txs)
	default:
	}
	select {
	case event := <-discover:
		t.Fatalf("recovered tx announced as discovery: %v", event.Txs)
	default:
	}
	assertFramePoolConsistent(t, pool)
}

func TestResetReorgRecoveryBound(t *testing.T) {
	for _, depth := range []int{64, 65} {
		for _, longOld := range []bool{false, true} {
			t.Run(fmt.Sprintf("depth=%d/old=%v", depth, longOld), func(t *testing.T) {
				pool, chain, _ := setupFramePool(t, 10, nil)
				ancestor := chain.CurrentBlock()
				recoverable := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
				old := chain.extend(ancestor, 101, []*types.Transaction{recoverable}, nil, nil)
				if longOld {
					for range depth - 1 {
						old = chain.extend(old, old.Time+1, nil, nil, nil)
					}
				}
				pool.Reset(ancestor, old)
				invalidated := signedPoolTx(t, 2, 0, 0, 20, 2, nil, nil)
				valid := signedPoolTx(t, 3, 0, 0, 20, 2, nil, nil)
				addPoolTx(t, pool, invalidated, nil)
				addPoolTx(t, pool, valid, nil)
				assertSimulations(t, pool, 2)
				newHead := chain.extend(ancestor, 102, nil, nil, func(s *state.StateDB) {
					s.SetNonce(poolAddress(t, 2), 1, tracing.NonceChangeUnspecified)
				})
				if !longOld {
					for range depth - 1 {
						newHead = chain.extend(newHead, newHead.Time+1, nil, nil, nil)
					}
				}
				pool.Reset(old, newHead)
				want := uint64(2)
				if depth == 64 {
					want++
				}
				assertSimulations(t, pool, want)
				assertLive(t, pool, recoverable, depth == 64)
				assertLive(t, pool, invalidated, false)
				assertLive(t, pool, valid, true)
				assertFramePoolConsistent(t, pool)
			})
		}
	}
}

// Reorg recovery attempts at most GlobalSlots candidates, oldest blocks first,
// and only one per sender: the one at the sender's new-head nonce. Later nonces
// of one sender cannot starve other senders.
func TestResetReorgRecoveryCap(t *testing.T) {
	pool, chain, _ := setupFramePool(t, 2, nil)
	ancestor := chain.CurrentBlock()
	// Each later candidate outranks the earlier ones and could displace them.
	var saturating, others []*types.Transaction
	for nonce := range uint64(3) {
		saturating = append(saturating, signedPoolTx(t, 1, 0, nonce, 20, nonce+2, nil, nil))
	}
	for id := 2; id <= 3; id++ {
		others = append(others, signedPoolTx(t, id, 0, 0, 20, uint64(id+4), nil, nil))
	}
	old := chain.extend(ancestor, 101, append(slices.Clone(saturating), others[0]), nil, nil)
	old = chain.extend(old, 102, others[1:], nil, nil)
	pool.Reset(ancestor, old)
	assertSimulations(t, pool, 0)
	pool.Reset(old, chain.extend(ancestor, 103, nil, bal.NewConstructionBlockAccessList(), nil))
	assertSimulations(t, pool, 2)
	for nonce, tx := range saturating {
		assertLive(t, pool, tx, nonce == 0)
	}
	assertLive(t, pool, others[0], true)
	assertLive(t, pool, others[1], false)
	assertFramePoolConsistent(t, pool)
}

// A sender's nonce need not advance with each inclusion: an account created
// and destroyed in one transaction (EIP-6780) stays at nonce 0. Its repeated
// displaced transactions spend one recovery attempt, the oldest.
func TestResetReorgRecoverySenderRepeats(t *testing.T) {
	pool, chain, _ := setupFramePool(t, 2, nil)
	ancestor := chain.CurrentBlock()
	var repeated []*types.Transaction
	for i := range uint64(3) {
		repeated = append(repeated, signedPoolTx(t, 1, 0, 0, 20+10*i, 2+i, nil, nil))
	}
	other := signedPoolTx(t, 2, 0, 0, 20, 2, nil, nil)
	old := chain.extend(ancestor, 101, append(slices.Clone(repeated), other), nil, nil)
	pool.Reset(ancestor, old)
	assertSimulations(t, pool, 0)
	pool.Reset(old, chain.extend(ancestor, 102, nil, bal.NewConstructionBlockAccessList(), nil))
	assertSimulations(t, pool, 2)
	for i, tx := range repeated {
		assertLive(t, pool, tx, i == 0)
	}
	assertLive(t, pool, other, true)
	assertFramePoolConsistent(t, pool)
}

// A Reset that cannot open the new head state leaves the pool at its last
// reconciled head. Admissions are refused until a later Reset succeeds, and that
// Reset revalidates the skipped interval although the caller moved past it.
func TestResetAfterFailedState(t *testing.T) {
	pool, chain, other := setupFramePool(t, 10, nil)
	valid := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
	invalidated := signedPoolTx(t, 2, 0, 0, 20, 2, nil, nil)
	addPoolTx(t, pool, valid, nil)
	addPoolTx(t, pool, invalidated, nil)
	assertSimulations(t, pool, 2)
	old := chain.CurrentBlock()
	accesses := bal.NewConstructionBlockAccessList()
	accesses.NonceChange(poolAddress(t, 2), 1, 1)
	skipped := chain.extend(old, 101, nil, accesses, func(s *state.StateDB) {
		s.SetNonce(poolAddress(t, 2), 1, tracing.NonceChangeUnspecified)
	})
	next := chain.extend(skipped, 102, nil, bal.NewConstructionBlockAccessList(), nil)
	chain.lock.Lock()
	delete(chain.states, skipped.Root)
	chain.lock.Unlock()
	pool.Reset(old, skipped)
	assertSimulations(t, pool, 0)
	if head := pool.head.Load(); head != old {
		t.Fatalf("failed reset advanced the pool head to %d", head.Number)
	}
	fresh := signedPoolTx(t, 3, 0, 0, 20, 2, nil, nil)
	addPoolTx(t, pool, fresh, ErrHeadChanged)
	assertSimulations(t, pool, 0)
	assertReleased(t, other, poolAddress(t, 3))

	pool.Reset(skipped, next)
	assertSimulations(t, pool, 1)
	assertLive(t, pool, valid, true)
	assertLive(t, pool, invalidated, false)
	assertReleased(t, other, poolAddress(t, 2))
	addPoolTx(t, pool, fresh, nil)
	assertFramePoolConsistent(t, pool)
}

func TestAddDuringSelectiveReset(t *testing.T) {
	pool, chain, _ := setupFramePool(t, 10, nil)
	existing := signedPoolTx(t, 1, 20, 0, 20, 3, nil, nil)
	incoming := signedPoolTx(t, 2, 20, 0, 20, 2, nil, nil)
	addPoolTx(t, pool, existing, nil)
	old := chain.CurrentBlock()
	balance := new(uint256.Int).Add(txCost(existing), txCost(incoming))
	balance.Sub(balance, uint256.NewInt(1))
	accesses := bal.NewConstructionBlockAccessList()
	accesses.BalanceChange(1, poolAddress(t, 20), balance)
	next := chain.extend(old, 101, nil, accesses, func(s *state.StateDB) {
		s.SetBalance(poolAddress(t, 20), balance, tracing.BalanceChangeUnspecified)
	})
	enteredReset, releaseReset := make(chan struct{}), make(chan struct{})
	enteredAdd, releaseAdd := make(chan struct{}), make(chan struct{})
	var resetOnce, addOnce sync.Once
	chain.setHook(func(head *types.Header) {
		if head == next {
			resetOnce.Do(func() {
				close(enteredReset)
				<-releaseReset
			})
		} else if head == old {
			addOnce.Do(func() {
				close(enteredAdd)
				<-releaseAdd
			})
		}
	})
	result := make(chan error, 1)
	go func() { result <- pool.Add([]*types.Transaction{incoming}, true)[0] }()
	<-enteredAdd // Admission already captured the old generation.
	resetDone := make(chan struct{})
	go func() { pool.Reset(old, next); close(resetDone) }()
	<-enteredReset
	close(releaseAdd) // Stale simulation finishes while Reset is still in flight.
	close(releaseReset)
	<-resetDone
	if err := <-result; !errors.Is(err, core.ErrInsufficientFunds) {
		t.Fatalf("stale payer exposure admitted: %v", err)
	}
	assertLive(t, pool, existing, true)
	assertLive(t, pool, incoming, false)
	assertFramePoolConsistent(t, pool)
}

func TestResetReorgAdmissionLimits(t *testing.T) {
	for _, limit := range []string{"payer-exposure", "coded-cap", "replacement", "capacity", "sender-hold", "missing-history"} {
		t.Run(limit, func(t *testing.T) {
			payerID := 0
			if limit == "payer-exposure" || limit == "coded-cap" {
				payerID = 20
			}
			recovered := signedPoolTx(t, 1, payerID, 0, 20, 2, nil, nil)
			slots := uint64(10)
			if limit == "capacity" {
				slots = 1
			}
			pool, chain, other := setupFramePool(t, slots, func(s *state.StateDB) {
				if limit == "payer-exposure" {
					s.SetBalance(poolAddress(t, 20), txCost(recovered), tracing.BalanceChangeUnspecified)
				}
				if limit == "coded-cap" {
					s.SetCode(poolAddress(t, 20), approveCode(program.New(), types.ApprovePayment), tracing.CodeChangeUnspecified)
				}
			})
			ancestor := chain.CurrentBlock()
			old := chain.extend(ancestor, 101, []*types.Transaction{recovered}, nil, nil)
			pool.Reset(ancestor, old)
			senderID := 2
			if limit == "replacement" {
				senderID = 1
			}
			current := signedPoolTx(t, senderID, payerID, 0, 20, 3, nil, nil)
			addPoolTx(t, pool, current, nil)
			if limit == "sender-hold" {
				if err := other.Hold(poolAddress(t, 1)); err != nil {
					t.Fatal(err)
				}
				defer other.Release(poolAddress(t, 1))
			}
			if limit == "missing-history" {
				chain.lock.Lock()
				delete(chain.blocks, old.Hash())
				chain.lock.Unlock()
			}
			next := chain.extend(ancestor, 102, nil, bal.NewConstructionBlockAccessList(), nil)
			pool.Reset(old, next)
			assertLive(t, pool, recovered, false)
			assertLive(t, pool, current, true)
			assertFramePoolConsistent(t, pool)
		})
	}
}
