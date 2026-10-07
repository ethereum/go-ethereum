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
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// testBlockChain returns independent head states just like the real chain. The
// synthetic roots identify snapshots; no simulation state is ever stored back.
type testBlockChain struct {
	config *params.ChainConfig
	lock   sync.Mutex
	head   *types.Header
	states map[common.Hash]*state.StateDB
	blocks map[common.Hash]*types.Block
	hook   func(*types.Header)
}

func (c *testBlockChain) Config() *params.ChainConfig { return c.config }
func (c *testBlockChain) CurrentBlock() *types.Header {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.head
}
func (c *testBlockChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	c.lock.Lock()
	defer c.lock.Unlock()
	if block := c.blocks[hash]; block != nil && block.NumberU64() == number {
		return block
	}
	return nil
}
func (c *testBlockChain) StateAt(head *types.Header) (*state.StateDB, error) {
	c.lock.Lock()
	var statedb *state.StateDB
	if template := c.states[head.Root]; template != nil {
		statedb = template.Copy()
	}
	hook := c.hook
	c.lock.Unlock()
	if hook != nil {
		hook(head)
	}
	if statedb == nil {
		return nil, fmt.Errorf("missing state %v", head.Root)
	}
	return statedb, nil
}
func (c *testBlockChain) setHook(hook func(*types.Header)) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.hook = hook
}
func (c *testBlockChain) advance(timestamp uint64, change func(*state.StateDB)) *types.Header {
	c.lock.Lock()
	defer c.lock.Unlock()
	next := types.CopyHeader(c.head)
	next.ParentHash = c.head.Hash()
	next.Number.Add(next.Number, common.Big1)
	next.Root = common.BigToHash(next.Number)
	next.Time = timestamp
	statedb := c.states[c.head.Root].Copy()
	if change != nil {
		change(statedb)
	}
	c.states[next.Root], c.head = statedb, next
	c.blocks[next.Hash()] = types.NewBlockWithHeader(next)
	return next
}

func poolKey(t *testing.T, id int) *ecdsa.PrivateKey {
	t.Helper()
	key, err := crypto.HexToECDSA(fmt.Sprintf("%064x", id))
	if err != nil {
		t.Fatal(err)
	}
	return key
}
func poolAddress(t *testing.T, id int) common.Address {
	t.Helper()
	return crypto.PubkeyToAddress(poolKey(t, id).PublicKey)
}

func setupFramePool(t *testing.T, slots uint64, change func(*state.StateDB)) (*FramePool, *testBlockChain, txpool.Reserver) {
	t.Helper()
	config := *params.AllDevChainProtocolChanges
	config.AmsterdamTime = new(uint64)
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatal(err)
	}
	for id := range 32 {
		statedb.SetBalance(poolAddress(t, id+1), uint256.NewInt(1e18), tracing.BalanceChangeUnspecified)
	}
	statedb.SetCode(params.FrameTxExpiryVerifier, params.FrameTxExpiryVerifierCode, tracing.CodeChangeUnspecified)
	if change != nil {
		change(statedb)
	}
	head := &types.Header{Root: common.BigToHash(big.NewInt(1)), Number: big.NewInt(1), Time: 100, GasLimit: 30_000_000, GasUsed: 15_000_000, BaseFee: big.NewInt(10), Difficulty: new(big.Int)}
	chain := &testBlockChain{
		config: &config, head: head, states: map[common.Hash]*state.StateDB{head.Root: statedb},
		blocks: map[common.Hash]*types.Block{head.Hash(): types.NewBlockWithHeader(head)},
	}
	tracker := txpool.NewReservationTracker()
	pool := New(Config{GlobalSlots: slots, PriceBump: 10}, chain, nil)
	if err := pool.Init(1, head, tracker.NewHandle(0)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool, chain, tracker.NewHandle(1)
}

func signedPoolTx(t *testing.T, senderID, payerID int, nonce, fee, tip uint64, deadline *uint64, alter func(*types.FrameTx)) *types.Transaction {
	t.Helper()
	sender := poolAddress(t, senderID)
	frames := []types.Frame{verifyFrame(types.ApproveExecutionAndPayment)}
	keys := []*ecdsa.PrivateKey{poolKey(t, senderID)}
	if payerID != 0 {
		payer := poolAddress(t, payerID)
		frames = []types.Frame{
			{Mode: types.ModeVerify, Flags: types.ApproveExecution, GasLimits: types.Limits{Execution: 30_000}, Value: new(uint256.Int)},
			{Mode: types.ModeVerify, Flags: types.ApprovePayment, Target: &payer, GasLimits: types.Limits{Execution: 30_000, State: params.AccountCreationSize * params.CostPerStateByte}, Value: new(uint256.Int)},
		}
		keys = append(keys, poolKey(t, payerID))
	}
	if deadline != nil {
		target := params.FrameTxExpiryVerifier
		data := make([]byte, params.FrameTxExpiryDataLen)
		binary.BigEndian.PutUint64(data, *deadline)
		frames = append([]types.Frame{{Mode: types.ModeVerify, Target: &target, Data: data, GasLimits: types.Limits{Execution: 10_000}, Value: new(uint256.Int)}}, frames...)
	}
	inner := &types.FrameTx{
		ChainID: uint256.MustFromBig(params.AllDevChainProtocolChanges.ChainID), Sender: sender, Nonce: nonce, Frames: frames,
		Fees: types.Fees{MaxFeePerGas: uint256.NewInt(fee), MaxPriorityFeePerGas: uint256.NewInt(tip), MaxFeePerBlobGas: new(uint256.Int)},
	}
	for _, key := range keys {
		inner.Signatures = append(inner.Signatures, types.SignatureEntry{Scheme: types.FrameTxSchemeSecp256k1, Signer: crypto.PubkeyToAddress(key.PublicKey).Bytes()})
	}
	if alter != nil {
		alter(inner)
	}
	hash := types.LatestSigner(params.AllDevChainProtocolChanges).Hash(types.NewTx(inner))
	for i, key := range keys {
		sig, err := crypto.Sign(hash[:], key)
		if err != nil {
			t.Fatal(err)
		}
		inner.Signatures[i].Signature = append([]byte{sig[64]}, sig[:64]...)
	}
	return types.NewTx(inner)
}

func poolTxData(tx *types.Transaction) *types.FrameTx {
	return &types.FrameTx{
		ChainID: uint256.MustFromBig(tx.ChainId()), Nonce: tx.Nonce(), Sender: *tx.FrameSender(),
		Frames: tx.Frames(), Signatures: append(types.SignatureList(nil), tx.FrameSignatures()...),
		Fees: types.Fees{MaxFeePerGas: uint256.MustFromBig(tx.GasFeeCap()), MaxPriorityFeePerGas: uint256.MustFromBig(tx.GasTipCap()), MaxFeePerBlobGas: new(uint256.Int)},
	}
}

func txCost(tx *types.Transaction) *uint256.Int {
	return new(uint256.Int).Mul(uint256.NewInt(tx.Gas()), uint256.MustFromBig(tx.GasFeeCap()))
}
func addPoolTx(t *testing.T, pool *FramePool, tx *types.Transaction, want error) {
	t.Helper()
	if err := pool.Add([]*types.Transaction{tx}, true)[0]; !errors.Is(err, want) {
		t.Fatalf("Add %s: got %v, want %v", tx.Hash(), err, want)
	}
}

// waitAnnounced waits until the dispatcher has published every acceptance
// queued so far, after which buffered subscriptions hold all their events.
func waitAnnounced(t *testing.T, p *FramePool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(time.Millisecond) {
		p.lock.RLock()
		queued := p.enqueued
		p.lock.RUnlock()
		if p.published.Load() == queued {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("dispatcher published %d of %d acceptances", p.published.Load(), queued)
		}
	}
}
func assertLive(t *testing.T, pool *FramePool, tx *types.Transaction, want bool) {
	t.Helper()
	if got := pool.Has(tx.Hash()); got != want {
		t.Fatalf("transaction %s live=%v, want %v", tx.Hash(), got, want)
	}
}
func assertReleased(t *testing.T, other txpool.Reserver, sender common.Address) {
	t.Helper()
	if err := other.Hold(sender); err != nil {
		t.Fatalf("sender reservation leaked: %v", err)
	}
	if err := other.Release(sender); err != nil {
		t.Fatal(err)
	}
}

// Accounting and reverse indexes must refer to exactly the accepted contents,
// including after rejection, replacement, capacity eviction and head changes.
func assertFramePoolConsistent(t *testing.T, p *FramePool) {
	t.Helper()
	p.lock.RLock()
	defer p.lock.RUnlock()
	accounts := make(map[common.Address]map[common.Hash]accountFields)
	slots := make(map[common.Address]map[common.Hash]map[common.Hash]struct{})
	usages := make(map[common.Address]*payerUsage)
	var used uint64
	for sender, entry := range p.txs {
		hash := entry.tx.Hash()
		if p.all[hash] != entry || p.evict[entry.index] != entry {
			t.Fatal("live transaction missing from lookup or eviction heap")
		}
		used += numSlots(entry.tx)
		usage := usages[entry.payer]
		if usage == nil {
			usage = &payerUsage{txs: make(map[common.Hash]*frameTx)}
			usages[entry.payer] = usage
		}
		if _, overflow := usage.reserved.AddOverflow(&usage.reserved, &entry.maxCost); overflow {
			t.Fatal("payer reservation overflow")
		}
		usage.txs[hash] = entry
		if entry.codedPaymaster {
			usage.coded++
		}
		if usage.coded > MaxPendingTxsUsingNonCanonicalPaymaster || usage.reserved.Gt(entry.payerBalance) {
			t.Fatal("payer overcommitted")
		}
		for addr, fields := range entry.dependencies.accounts {
			if accounts[addr] == nil {
				accounts[addr] = make(map[common.Hash]accountFields)
			}
			accounts[addr][hash] = fields
		}
		for slot := range entry.dependencies.slots {
			if slots[sender] == nil {
				slots[sender] = make(map[common.Hash]map[common.Hash]struct{})
			}
			if slots[sender][slot] == nil {
				slots[sender][slot] = make(map[common.Hash]struct{})
			}
			slots[sender][slot][hash] = struct{}{}
		}
	}
	if used != p.slots || len(p.all) != len(p.txs) || len(p.evict) != len(p.txs) || !reflect.DeepEqual(usages, p.payers) || !reflect.DeepEqual(accounts, p.byAccount) || !reflect.DeepEqual(slots, p.bySlot) {
		t.Fatal("pool accounting or dependency indexes inconsistent with live contents")
	}
	for i := 1; i < len(p.evict); i++ {
		if compareEviction(p.evict[i], p.evict[(i-1)/2]) < 0 {
			t.Fatal("eviction heap is out of order")
		}
	}
}

func TestValidateTxBasics(t *testing.T) {
	pool, _, _ := setupFramePool(t, 10, nil)
	valid := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
	for _, tc := range []struct {
		name  string
		alter func(*types.FrameTx)
		want  error
	}{
		{"shape before signature", func(tx *types.FrameTx) { tx.Frames = append(tx.Frames, verifyFrame(types.ApproveExecutionAndPayment)) }, ErrInvalidPrefix},
		{"execution budget before signature", func(tx *types.FrameTx) { tx.Frames[0].GasLimits.Execution = MaxVerifyGas }, ErrPrefixGasLimit},
		{"state budget before signature", func(tx *types.FrameTx) { tx.Frames[0].GasLimits.State = MaxVerifyStateGas + 1 }, ErrPrefixGasLimit},
		{"bad signature", nil, types.ErrFrameTxInvalidSignature},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := signedPoolTx(t, 1, 0, 0, 20, 2, nil, tc.alter)
			inner := poolTxData(tx)
			inner.Signatures[0].Signature = make([]byte, 65)
			if err := pool.ValidateTxBasics(types.NewTx(inner)); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
	if err := pool.ValidateTxBasics(valid); err != nil {
		t.Fatal(err)
	}
	pool.hasPendingAuth = func(addr common.Address) bool { return addr == poolAddress(t, 1) }
	if err := pool.ValidateTxBasics(valid); err != nil {
		t.Fatalf("stateless validation consulted pending authorizations: %v", err)
	}
	addPoolTx(t, pool, valid, txpool.ErrInflightTxLimitReached)
}

func TestForgedAdmissionAndReplacement(t *testing.T) {
	pool, _, other := setupFramePool(t, 10, nil)
	victim := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
	addPoolTx(t, pool, victim, nil)
	for _, tc := range []struct {
		name    string
		corrupt bool
		want    error
	}{
		{"invalid signature", true, types.ErrFrameTxInvalidSignature},
		{"valid attacker signature without sender approval", false, core.ErrFrameTxInvalidExecution},
	} {
		t.Run(tc.name, func(t *testing.T) {
			forged := signedPoolTx(t, 2, 0, 0, 40, 4, nil, func(tx *types.FrameTx) { tx.Sender = poolAddress(t, 1) })
			if tc.corrupt {
				inner := poolTxData(forged)
				inner.Signatures[0].Signature = make([]byte, 65)
				forged = types.NewTx(inner)
			}
			addPoolTx(t, pool, forged, tc.want)
			assertLive(t, pool, victim, true)
			assertFramePoolConsistent(t, pool)

			// The same forgery for an unpooled sender must never acquire a hold.
			fresh := signedPoolTx(t, 2, 0, 0, 40, 4, nil, func(tx *types.FrameTx) { tx.Sender = poolAddress(t, 3) })
			if tc.corrupt {
				inner := poolTxData(fresh)
				inner.Signatures[0].Signature = make([]byte, 65)
				fresh = types.NewTx(inner)
			}
			addPoolTx(t, pool, fresh, tc.want)
			assertReleased(t, other, poolAddress(t, 3))
		})
	}
}

func TestOneSenderAndReplacementFees(t *testing.T) {
	pool, _, other := setupFramePool(t, 10, nil)
	initial := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
	addPoolTx(t, pool, initial, nil)
	addPoolTx(t, pool, initial, txpool.ErrAlreadyKnown)
	addPoolTx(t, pool, signedPoolTx(t, 1, 0, 1, 22, 3, nil, nil), ErrSenderPending)
	for _, fees := range [][2]uint64{{22, 2}, {20, 3}, {21, 3}} {
		addPoolTx(t, pool, signedPoolTx(t, 1, 0, 0, fees[0], fees[1], nil, nil), txpool.ErrReplaceUnderpriced)
		assertLive(t, pool, initial, true)
	}
	replacement := signedPoolTx(t, 1, 0, 0, 22, 3, nil, nil)
	addPoolTx(t, pool, replacement, nil)
	assertLive(t, pool, initial, false)
	assertLive(t, pool, replacement, true)
	if err := other.Hold(poolAddress(t, 1)); !errors.Is(err, txpool.ErrAlreadyReserved) {
		t.Fatalf("replacement lost sender hold: %v", err)
	}
	if pending, queued := pool.Stats(); pending != 1 || queued != 0 || pool.Nonce(poolAddress(t, 1)) != 1 {
		t.Fatalf("one-sender stats: pending=%d queued=%d", pending, queued)
	}
	assertFramePoolConsistent(t, pool)
}

func TestPayerReplacementAccounting(t *testing.T) {
	initial := signedPoolTx(t, 1, 20, 0, 20, 2, nil, nil)
	same := signedPoolTx(t, 1, 20, 0, 22, 3, nil, nil)
	moved := signedPoolTx(t, 1, 21, 0, 25, 4, nil, nil)
	pool, _, _ := setupFramePool(t, 10, func(s *state.StateDB) {
		s.SetBalance(poolAddress(t, 20), txCost(same), tracing.BalanceChangeUnspecified)
		s.SetBalance(poolAddress(t, 21), txCost(moved), tracing.BalanceChangeUnspecified)
		s.SetBalance(poolAddress(t, 22), new(uint256.Int), tracing.BalanceChangeUnspecified)
	})
	addPoolTx(t, pool, initial, nil)
	addPoolTx(t, pool, same, nil) // Fits only after crediting the replaced cost.
	// A payer that cannot cover the cost at head is rejected before execution.
	addPoolTx(t, pool, signedPoolTx(t, 1, 22, 0, 25, 4, nil, nil), core.ErrInsufficientFunds)
	assertLive(t, pool, same, true)
	assertFramePoolConsistent(t, pool)
	addPoolTx(t, pool, moved, nil)
	// Reuse the old payer's released exposure from a different sender.
	addPoolTx(t, pool, signedPoolTx(t, 2, 20, 0, 20, 2, nil, nil), nil)
	assertFramePoolConsistent(t, pool)
}

func TestAggregateExposureAndPaymasterCap(t *testing.T) {
	for _, tc := range []struct {
		name    string
		coded   bool
		fundOne bool
		want    error
	}{
		{"aggregate balance", false, true, core.ErrInsufficientFunds},
		{"coded paymaster", true, false, txpool.ErrInflightTxLimitReached},
		{"default sponsor", false, false, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first := signedPoolTx(t, 1, 20, 0, 20, 2, nil, nil)
			pool, _, other := setupFramePool(t, 10, func(s *state.StateDB) {
				if tc.fundOne {
					s.SetBalance(poolAddress(t, 20), txCost(first), tracing.BalanceChangeUnspecified)
				}
				if tc.coded {
					s.SetCode(poolAddress(t, 20), approveCode(program.New(), types.ApprovePayment), tracing.CodeChangeUnspecified)
				}
			})
			addPoolTx(t, pool, first, nil)
			second := signedPoolTx(t, 2, 20, 0, 20, 2, nil, nil)
			addPoolTx(t, pool, second, tc.want)
			if tc.want != nil {
				assertReleased(t, other, poolAddress(t, 2))
			}
			if tc.coded {
				// A same-payer replacement must credit its coded count, too.
				addPoolTx(t, pool, signedPoolTx(t, 1, 20, 0, 22, 3, nil, nil), nil)
			}
			assertFramePoolConsistent(t, pool)
		})
	}
}

func TestCapacityEvictionOrder(t *testing.T) {
	pool, _, other := setupFramePool(t, 2, nil)
	near, far := uint64(101), uint64(102)
	first := signedPoolTx(t, 1, 0, 0, 100, 30, &near, nil)
	second := signedPoolTx(t, 2, 0, 0, 100, 1, &far, nil)
	addPoolTx(t, pool, first, nil)
	addPoolTx(t, pool, second, nil)
	third := signedPoolTx(t, 3, 0, 0, 100, 2, nil, nil)
	addPoolTx(t, pool, third, nil)
	assertLive(t, pool, first, false)
	assertLive(t, pool, second, true)
	assertReleased(t, other, poolAddress(t, 1))
	fourth := signedPoolTx(t, 4, 0, 0, 100, 3, nil, nil)
	addPoolTx(t, pool, fourth, nil)
	assertLive(t, pool, second, false)
	fifth := signedPoolTx(t, 5, 0, 0, 100, 4, nil, nil)
	addPoolTx(t, pool, fifth, nil)
	assertLive(t, pool, third, false)
	addPoolTx(t, pool, signedPoolTx(t, 6, 0, 0, 100, 3, nil, nil), txpool.ErrUnderpriced)
	assertLive(t, pool, fourth, true)
	assertLive(t, pool, fifth, true)
	assertReleased(t, other, poolAddress(t, 6))
	// A finite deadline, even MaxUint64, cannot displace an unexpiring tx.
	deadline := uint64(math.MaxUint64)
	addPoolTx(t, pool, signedPoolTx(t, 6, 0, 0, 1000, 1000, &deadline, nil), txpool.ErrUnderpriced)
	assertFramePoolConsistent(t, pool)
}

func TestCapacityEffectiveTipAndFailedHold(t *testing.T) {
	pool, _, other := setupFramePool(t, 2, nil)
	lowEffective := signedPoolTx(t, 1, 0, 0, 11, 10, nil, nil) // Effective tip 1, not 10.
	otherTx := signedPoolTx(t, 2, 0, 0, 100, 2, nil, nil)
	addPoolTx(t, pool, lowEffective, nil)
	addPoolTx(t, pool, otherTx, nil)
	waitAnnounced(t, pool)
	discover, insert := make(chan core.NewTxsEvent, 1), make(chan core.NewTxsEvent, 1)
	subDiscover := pool.SubscribeTransactions(discover, false)
	subInsert := pool.SubscribeTransactions(insert, true)
	defer subDiscover.Unsubscribe()
	defer subInsert.Unsubscribe()
	if err := other.Hold(poolAddress(t, 3)); err != nil {
		t.Fatal(err)
	}
	newTx := signedPoolTx(t, 3, 0, 0, 100, 3, nil, nil)
	addPoolTx(t, pool, newTx, txpool.ErrAlreadyReserved)
	assertLive(t, pool, lowEffective, true)
	assertLive(t, pool, otherTx, true)
	assertFramePoolConsistent(t, pool)
	waitAnnounced(t, pool)
	for _, events := range []chan core.NewTxsEvent{discover, insert} {
		select {
		case got := <-events:
			t.Fatalf("failed sender hold emitted an acceptance event: %v", got)
		default:
		}
	}
	if err := other.Release(poolAddress(t, 3)); err != nil {
		t.Fatal(err)
	}
	addPoolTx(t, pool, newTx, nil)
	assertLive(t, pool, lowEffective, false)
	assertLive(t, pool, otherTx, true)
	assertFramePoolConsistent(t, pool)
}

func TestCapacityAllVictimsAndReplacementCredit(t *testing.T) {
	pool, _, _ := setupFramePool(t, 2, nil)
	low := signedPoolTx(t, 1, 0, 0, 100, 1, nil, nil)
	high := signedPoolTx(t, 2, 0, 0, 100, 10, nil, nil)
	addPoolTx(t, pool, low, nil)
	addPoolTx(t, pool, high, nil)
	large := func(tx *types.FrameTx) {
		tx.Frames = append(tx.Frames, types.Frame{Mode: types.ModeSender, GasLimits: types.Limits{Execution: 100_000}, Value: new(uint256.Int), Data: make([]byte, txSlotSize)})
	}
	// The newcomer needs two victims but outranks only the cheapest one.
	addPoolTx(t, pool, signedPoolTx(t, 3, 0, 0, 100, 5, nil, large), txpool.ErrUnderpriced)
	assertLive(t, pool, low, true)
	assertLive(t, pool, high, true)
	// Replacing the cheapest entry excludes its slot and priority from victims.
	replacement := signedPoolTx(t, 1, 0, 0, 110, 11, nil, large)
	addPoolTx(t, pool, replacement, nil)
	assertLive(t, pool, low, false)
	assertLive(t, pool, high, false)
	assertLive(t, pool, replacement, true)
	assertFramePoolConsistent(t, pool)
}

func TestResetNonceAndExpiry(t *testing.T) {
	pool, chain, other := setupFramePool(t, 10, nil)
	deadline := uint64(101)
	expiring := signedPoolTx(t, 1, 0, 0, 20, 2, &deadline, nil)
	included := signedPoolTx(t, 2, 0, 0, 20, 2, nil, nil)
	addPoolTx(t, pool, expiring, nil)
	addPoolTx(t, pool, included, nil)
	old := chain.CurrentBlock()
	equal := chain.advance(101, func(s *state.StateDB) { s.SetNonce(poolAddress(t, 2), 1, tracing.NonceChangeUnspecified) })
	pool.Reset(old, equal)
	assertLive(t, pool, expiring, true)
	assertLive(t, pool, included, false)
	assertReleased(t, other, poolAddress(t, 2))
	if pool.Nonce(poolAddress(t, 2)) != 1 {
		t.Fatal("Nonce did not fall back to new head state")
	}
	pool.Reset(equal, chain.advance(102, nil))
	assertLive(t, pool, expiring, false)
	assertReleased(t, other, poolAddress(t, 1))
	assertFramePoolConsistent(t, pool)
}

func TestResetPayerShortfallAndCode(t *testing.T) {
	for _, coded := range []bool{false, true} {
		t.Run(fmt.Sprintf("coded=%v", coded), func(t *testing.T) {
			pool, chain, other := setupFramePool(t, 10, nil)
			var sponsored []*types.Transaction
			for i := range 3 {
				tx := signedPoolTx(t, i+1, 20, 0, 20, uint64(i+1), nil, nil)
				sponsored = append(sponsored, tx)
				addPoolTx(t, pool, tx, nil)
			}
			unrelated := signedPoolTx(t, 4, 0, 0, 20, 1, nil, nil)
			addPoolTx(t, pool, unrelated, nil)
			old := chain.CurrentBlock()
			next := chain.advance(101, func(s *state.StateDB) {
				if coded {
					s.SetCode(poolAddress(t, 20), approveCode(program.New(), types.ApprovePayment), tracing.CodeChangeUnspecified)
				} else {
					balance := new(uint256.Int).Add(txCost(sponsored[1]), txCost(sponsored[2]))
					s.SetBalance(poolAddress(t, 20), balance, tracing.BalanceChangeUnspecified)
				}
			})
			pool.Reset(old, next)
			for i, tx := range sponsored {
				want := i >= 1
				if coded {
					want = i == 2
				}
				assertLive(t, pool, tx, want)
				if !want {
					assertReleased(t, other, poolAddress(t, i+1))
				}
			}
			assertLive(t, pool, unrelated, true)
			assertFramePoolConsistent(t, pool)
		})
	}
}

func TestResetRefreshesEffectiveTips(t *testing.T) {
	for _, selective := range []bool{false, true} {
		t.Run(fmt.Sprintf("selective=%v", selective), func(t *testing.T) {
			pool, chain, _ := setupFramePool(t, 2, nil)
			capped := signedPoolTx(t, 1, 0, 0, 20, 10, nil, nil)
			uncapped := signedPoolTx(t, 2, 0, 0, 100, 5, nil, nil)
			addPoolTx(t, pool, capped, nil)
			addPoolTx(t, pool, uncapped, nil)
			assertSimulations(t, pool, 2)
			old := chain.CurrentBlock()
			next := chain.advance(101, nil)
			next.BaseFee = big.NewInt(19)
			if selective {
				list := bal.NewConstructionBlockAccessList().ToEncodingObj()
				hash := list.Hash()
				next.BlockAccessListHash = &hash
				chain.lock.Lock()
				chain.blocks[next.Hash()] = types.NewBlockWithHeader(next).WithAccessList(list)
				chain.lock.Unlock()
			}
			pool.Reset(old, next)
			want := uint64(2)
			if selective {
				want = 0
			}
			assertSimulations(t, pool, want)
			addPoolTx(t, pool, signedPoolTx(t, 3, 0, 0, 100, 6, nil, nil), nil)
			assertLive(t, pool, capped, false)
			assertLive(t, pool, uncapped, true)
			assertFramePoolConsistent(t, pool)
		})
	}
}

func TestPendingFiltersAndLiveRetrieval(t *testing.T) {
	pool, _, _ := setupFramePool(t, 10, nil)
	tx := signedPoolTx(t, 1, 0, 0, 20, 5, nil, nil)
	addPoolTx(t, pool, tx, nil)
	for _, tc := range []struct {
		name   string
		filter txpool.PendingFilter
		want   int
	}{
		{"plain", txpool.PendingFilter{}, 1},
		{"blob only", txpool.PendingFilter{BlobTxs: true}, 0},
		{"tip cap", txpool.PendingFilter{MinTip: uint256.NewInt(6)}, 0},
		{"base fee alone", txpool.PendingFilter{BaseFee: uint256.NewInt(21)}, 0},
		{"effective tip", txpool.PendingFilter{BaseFee: uint256.NewInt(18), MinTip: uint256.NewInt(3)}, 0},
		{"effective tip equality", txpool.PendingFilter{BaseFee: uint256.NewInt(18), MinTip: uint256.NewInt(2)}, 1},
		{"gas cap", txpool.PendingFilter{GasLimitCap: tx.Gas() - 1}, 0},
		{"gas cap equality", txpool.PendingFilter{GasLimitCap: tx.Gas()}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pending, count := pool.Pending(tc.filter)
			if count != tc.want || len(pending) != tc.want {
				t.Fatalf("got %d pending, want %d", count, tc.want)
			}
			if count == 1 && pending[poolAddress(t, 1)][0].Resolve() != tx {
				t.Fatal("lazy transaction did not resolve the accepted transaction")
			}
		})
	}
	var decoded types.Transaction
	if err := rlp.DecodeBytes(pool.GetRLP(tx.Hash(), 72), &decoded); err != nil || decoded.Hash() != tx.Hash() {
		t.Fatalf("network transaction retrieval failed: %v", err)
	}
	if pool.GetMetadata(tx.Hash()).Size != tx.Size() || pool.Status(tx.Hash()) != txpool.TxStatusPending {
		t.Fatal("live metadata or status incorrect")
	}
	pending, queued := pool.Content()
	from, queue := pool.ContentFrom(poolAddress(t, 1))
	if len(queued) != 0 || len(queue) != 0 || len(pending[poolAddress(t, 1)]) != 1 || len(from) != 1 || from[0] != tx {
		t.Fatal("pending content incorrect")
	}
	pool.Clear()
	if pool.Get(tx.Hash()) != nil || pool.GetRLP(tx.Hash(), 72) != nil || pool.GetMetadata(tx.Hash()) != nil || pool.Status(tx.Hash()) != txpool.TxStatusUnknown {
		t.Fatal("removed transaction remains exposed to retrieval or announcements")
	}
}

func TestAcceptedEventsAndRemoval(t *testing.T) {
	pool, _, other := setupFramePool(t, 1, nil)
	discover, insert := make(chan core.NewTxsEvent, 10), make(chan core.NewTxsEvent, 10)
	subDiscover := pool.SubscribeTransactions(discover, false)
	subInsert := pool.SubscribeTransactions(insert, true)
	defer subDiscover.Unsubscribe()
	defer subInsert.Unsubscribe()
	first := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
	addPoolTx(t, pool, first, nil)
	addPoolTx(t, pool, signedPoolTx(t, 2, 0, 0, 20, 1, nil, nil), txpool.ErrUnderpriced)
	addPoolTx(t, pool, signedPoolTx(t, 1, 0, 0, 22, 2, nil, nil), txpool.ErrReplaceUnderpriced)
	forged := poolTxData(signedPoolTx(t, 1, 0, 0, 40, 4, nil, nil))
	forged.Signatures[0].Signature = make([]byte, 65)
	addPoolTx(t, pool, types.NewTx(forged), types.ErrFrameTxInvalidSignature)
	replacement := signedPoolTx(t, 1, 0, 0, 22, 3, nil, nil)
	addPoolTx(t, pool, replacement, nil)
	pool.SetGasTip(big.NewInt(4))
	waitAnnounced(t, pool)
	assertLive(t, pool, replacement, false)
	assertReleased(t, other, poolAddress(t, 1))
	for _, events := range []chan core.NewTxsEvent{discover, insert} {
		for _, want := range []*types.Transaction{first, replacement} {
			select {
			case got := <-events:
				if len(got.Txs) != 1 || got.Txs[0] != want {
					t.Fatalf("unexpected acceptance event: %v", got)
				}
			default:
				t.Fatal("missing acceptance event")
			}
		}
		select {
		case got := <-events:
			t.Fatalf("rejected or removed transaction emitted acceptance event: %v", got)
		default:
		}
	}
	// Raising the floor released exposure as well as the sender reservation.
	pool.SetGasTip(big.NewInt(1))
	addPoolTx(t, pool, first, nil)
	pool.Clear()
	assertReleased(t, other, poolAddress(t, 1))
	assertFramePoolConsistent(t, pool)
}

// Acceptance events must be sent without holding the pool lock: subscribers
// such as the broadcast loop read the pool before draining their channel.
func TestAcceptedEventsSubscriberReadsPool(t *testing.T) {
	for _, reorg := range []bool{false, true} {
		t.Run(fmt.Sprintf("reorg=%v", reorg), func(t *testing.T) {
			pool, chain, _ := setupFramePool(t, 10, nil)
			tx := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
			ancestor := chain.CurrentBlock()
			var old *types.Header
			if reorg {
				old = chain.extend(ancestor, 101, []*types.Transaction{tx}, nil, nil)
				pool.Reset(ancestor, old)
			}
			discover, insert := make(chan core.NewTxsEvent), make(chan core.NewTxsEvent)
			subDiscover := pool.SubscribeTransactions(discover, false)
			subInsert := pool.SubscribeTransactions(insert, true)
			defer subDiscover.Unsubscribe()
			defer subInsert.Unsubscribe()

			// The transaction becomes visible only after its commit, so every
			// event is sent after the subscriber has read the pool.
			received := make(chan []*types.Transaction, 1)
			go func() {
				for pool.Get(tx.Hash()) == nil {
					time.Sleep(time.Millisecond)
				}
				var got []*types.Transaction
				if !reorg {
					got = append(got, (<-discover).Txs...)
				}
				received <- append(got, (<-insert).Txs...)
			}()
			done := make(chan error, 1)
			go func() {
				if reorg {
					pool.Reset(old, chain.extend(ancestor, 102, nil, bal.NewConstructionBlockAccessList(), nil))
					done <- nil
				} else {
					done <- pool.Add([]*types.Transaction{tx}, true)[0]
				}
			}()
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("admission deadlocked with a subscriber reading the pool")
			}
			want := []*types.Transaction{tx, tx}
			if reorg {
				want = want[:1]
			}
			if got := <-received; !slices.Equal(got, want) {
				t.Fatalf("acceptance events: got %v, want %v", got, want)
			}
			assertFramePoolConsistent(t, pool)
		})
	}
}

// Admissions never wait on subscribers, and events are published in commit
// order: a replaced transaction is never announced after its replacement.
func TestAcceptedEventsCommitOrder(t *testing.T) {
	pool, _, _ := setupFramePool(t, 10, nil)
	discover, insert := make(chan core.NewTxsEvent), make(chan core.NewTxsEvent)
	subDiscover := pool.SubscribeTransactions(discover, false)
	subInsert := pool.SubscribeTransactions(insert, true)
	defer subDiscover.Unsubscribe()
	defer subInsert.Unsubscribe()

	txs := []*types.Transaction{
		signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil),
		signedPoolTx(t, 1, 0, 0, 22, 3, nil, nil), // Replaces the first.
		signedPoolTx(t, 2, 0, 0, 20, 2, nil, nil),
	}
	added := make(chan []error, 1)
	go func() { added <- pool.Add(txs, true) }() // Subscribers are not reading.
	select {
	case errs := <-added:
		for i, err := range errs {
			if err != nil {
				t.Fatalf("Add %d: %v", i, err)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("admission waited on a subscriber")
	}
	receive := func(events chan core.NewTxsEvent) *types.Transaction {
		select {
		case event := <-events:
			return event.Txs[0]
		case <-time.After(5 * time.Second):
			t.Fatal("missing acceptance event")
			return nil
		}
	}
	for i, want := range txs {
		if got := receive(discover); got != want {
			t.Fatalf("discovery %d: got %v, want %v", i, got.Hash(), want.Hash())
		}
		if got := receive(insert); got != want {
			t.Fatalf("insertion %d: got %v, want %v", i, got.Hash(), want.Hash())
		}
	}
}

// A huge configured capacity must not wrap the announcement bound to zero,
// which would drop or index an empty queue on the first acceptance.
func TestAcceptedEventsHugeCapacity(t *testing.T) {
	pool, _, _ := setupFramePool(t, 1<<62, nil)
	discover := make(chan core.NewTxsEvent, 1)
	sub := pool.SubscribeTransactions(discover, false)
	defer sub.Unsubscribe()
	tx := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
	addPoolTx(t, pool, tx, nil)
	select {
	case ev := <-discover:
		if len(ev.Txs) != 1 || ev.Txs[0].Hash() != tx.Hash() {
			t.Fatalf("unexpected event %v", ev.Txs)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("accepted transaction was not announced")
	}
}

// A stalled subscriber never blocks admission. The announcement queue keeps
// only the newest 4*GlobalSlots acceptances, and what is published once the
// subscriber resumes is still in commit order, ending with the latest.
func TestAcceptedEventsStalledSubscriber(t *testing.T) {
	pool, _, _ := setupFramePool(t, 2, nil)
	limit := pool.announceLimit()
	discover := make(chan core.NewTxsEvent)
	sub := pool.SubscribeTransactions(discover, false)
	defer sub.Unsubscribe()

	// Successive replacements, each accepted and announced.
	txs := make([]*types.Transaction, 3*limit)
	for i := range txs {
		txs[i] = signedPoolTx(t, 1, 0, 0, 20<<i, 2<<i, nil, nil)
	}
	added := make(chan struct{})
	go func() {
		defer close(added)
		for _, tx := range txs {
			if err := pool.Add([]*types.Transaction{tx}, true)[0]; err != nil {
				t.Error(err)
			}
		}
	}()
	select {
	case <-added:
	case <-time.After(5 * time.Second):
		t.Fatal("admission blocked on a stalled subscriber")
	}
	pool.lock.RLock()
	queued := len(pool.announces)
	pool.lock.RUnlock()
	if queued > limit {
		t.Fatalf("%d acceptances queued, bound %d", queued, limit)
	}
	order := make(map[*types.Transaction]int, len(txs))
	for i, tx := range txs {
		order[tx] = i
	}
	last, received := -1, 0
	for last != len(txs)-1 {
		select {
		case event := <-discover:
			i := order[event.Txs[0]]
			if i <= last {
				t.Fatalf("acceptance %d published after %d", i, last)
			}
			last = i
			received++
		case <-time.After(5 * time.Second):
			t.Fatalf("latest acceptance not published, last %d", last)
		}
	}
	// At most one acceptance was taken by the dispatcher before it stalled.
	if received > limit+1 {
		t.Fatalf("published %d of %d acceptances, bound %d", received, len(txs), limit+1)
	}
	waitAnnounced(t, pool)
}

func TestAdmissionHeadChangeRetry(t *testing.T) {
	pool, chain, other := setupFramePool(t, 10, nil)
	old := chain.CurrentBlock()
	entered, release := make(chan struct{}), make(chan struct{})
	chain.setHook(func(head *types.Header) {
		if head.Root == old.Root {
			close(entered)
			<-release
		}
	})
	tx := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
	result := make(chan error, 1)
	go func() { result <- pool.Add([]*types.Transaction{tx}, true)[0] }()
	<-entered
	next := chain.advance(101, func(s *state.StateDB) { s.SetNonce(poolAddress(t, 1), 1, tracing.NonceChangeUnspecified) })
	pool.Reset(old, next)
	close(release)
	if err := <-result; !errors.Is(err, core.ErrNonceTooLow) {
		t.Fatalf("stale validation committed: %v", err)
	}
	assertLive(t, pool, tx, false)
	assertReleased(t, other, poolAddress(t, 1))
	assertFramePoolConsistent(t, pool)
}

func TestAdmissionHeadChangeRetryBound(t *testing.T) {
	pool, chain, other := setupFramePool(t, 10, nil)
	old := chain.CurrentBlock()
	first, releaseFirst := make(chan struct{}), make(chan struct{})
	chain.setHook(func(head *types.Header) {
		if head.Root == old.Root {
			close(first)
			<-releaseFirst
		}
	})
	tx := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
	result := make(chan error, 1)
	go func() { result <- pool.Add([]*types.Transaction{tx}, true)[0] }()
	<-first
	next := chain.advance(101, nil)
	pool.Reset(old, next)
	second, releaseSecond := make(chan struct{}), make(chan struct{})
	chain.setHook(func(head *types.Header) {
		if head.Root == next.Root {
			close(second)
			<-releaseSecond
		}
	})
	close(releaseFirst)
	<-second
	pool.Reset(next, chain.advance(102, nil))
	close(releaseSecond)
	if err := <-result; !errors.Is(err, ErrHeadChanged) {
		t.Fatalf("got %v, want bounded retry error", err)
	}
	assertLive(t, pool, tx, false)
	assertReleased(t, other, poolAddress(t, 1))
	assertFramePoolConsistent(t, pool)
}

func TestDependenciesReleasedOnReplacementAndClear(t *testing.T) {
	slot := common.HexToHash("0x42")
	pool, _, other := setupFramePool(t, 10, func(s *state.StateDB) {
		code := approveCode(program.New().Push(slot).Op(vm.SLOAD, vm.POP), types.ApproveExecutionAndPayment)
		s.SetCode(poolAddress(t, 1), code, tracing.CodeChangeUnspecified)
	})
	first := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
	second := signedPoolTx(t, 1, 0, 0, 22, 3, nil, nil)
	addPoolTx(t, pool, first, nil)
	addPoolTx(t, pool, second, nil)
	assertFramePoolConsistent(t, pool)
	pool.Clear()
	assertFramePoolConsistent(t, pool)
	assertReleased(t, other, poolAddress(t, 1))
}

func TestCheckedPayerCostAndExposure(t *testing.T) {
	sample := signedPoolTx(t, 1, 20, 0, 20, 2, nil, nil)
	max := new(uint256.Int).SetAllOne()
	// Leave headroom for signature-byte calldata gas while making each cost
	// roughly two thirds of the maximum balance.
	fee := new(uint256.Int).Div(max, uint256.NewInt(sample.Gas()*3/2))
	pool, _, other := setupFramePool(t, 10, func(s *state.StateDB) {
		s.SetBalance(poolAddress(t, 20), max, tracing.BalanceChangeUnspecified)
	})
	first := signedPoolTx(t, 1, 20, 0, 20, 2, nil, func(tx *types.FrameTx) {
		tx.Fees.MaxFeePerGas = fee
	})
	addPoolTx(t, pool, first, nil)
	// Each tx is individually solvent, but summing them overflows uint256.
	second := signedPoolTx(t, 2, 20, 0, 20, 2, nil, func(tx *types.FrameTx) {
		tx.Fees.MaxFeePerGas = fee
	})
	addPoolTx(t, pool, second, core.ErrInsufficientFunds)
	overflow := signedPoolTx(t, 3, 20, 0, 20, 2, nil, func(tx *types.FrameTx) {
		tx.Fees.MaxFeePerGas = max
	})
	addPoolTx(t, pool, overflow, core.ErrInsufficientFunds)
	assertLive(t, pool, first, true)
	assertReleased(t, other, poolAddress(t, 2))
	assertReleased(t, other, poolAddress(t, 3))
	assertFramePoolConsistent(t, pool)
}

func TestReplacementFractionalBump(t *testing.T) {
	pool, _, _ := setupFramePool(t, 10, nil)
	first := signedPoolTx(t, 1, 0, 0, 101, 10, nil, nil)
	addPoolTx(t, pool, first, nil)
	// 111/101 is less than a 10% fee-cap increase; don't round down.
	addPoolTx(t, pool, signedPoolTx(t, 1, 0, 0, 111, 11, nil, nil), txpool.ErrReplaceUnderpriced)
	assertLive(t, pool, first, true)
	addPoolTx(t, pool, signedPoolTx(t, 1, 0, 0, 112, 11, nil, nil), nil)
}

func TestConcurrentPayerAdmission(t *testing.T) {
	var txs []*types.Transaction
	var largest uint256.Int
	for id := range 10 {
		tx := signedPoolTx(t, id+1, 20, 0, 20, 2, nil, nil)
		txs = append(txs, tx)
		if cost := txCost(tx); cost.Gt(&largest) {
			largest.Set(cost)
		}
	}
	pool, _, other := setupFramePool(t, 32, func(s *state.StateDB) {
		s.SetBalance(poolAddress(t, 20), new(uint256.Int).Mul(&largest, uint256.NewInt(5)), tracing.BalanceChangeUnspecified)
	})
	var wg sync.WaitGroup
	errs := make([]error, len(txs))
	for i, tx := range txs {
		wg.Go(func() { errs[i] = pool.Add([]*types.Transaction{tx}, true)[0] })
	}
	wg.Wait()
	var accepted int
	for i, err := range errs {
		if err == nil {
			accepted++
		} else {
			if !errors.Is(err, core.ErrInsufficientFunds) {
				t.Fatalf("unexpected admission error: %v", err)
			}
			assertReleased(t, other, *txs[i].FrameSender())
		}
	}
	if accepted != 5 {
		t.Fatalf("admitted %d transactions, want payer's capacity of 5", accepted)
	}
	assertFramePoolConsistent(t, pool)
}

func TestAdmissionCheapPoolRejections(t *testing.T) {
	deadline := uint64(101)
	for _, tc := range []struct {
		name   string
		makeTx func(*testing.T, *types.Transaction) *types.Transaction
		want   error
	}{
		{"known", func(_ *testing.T, tx *types.Transaction) *types.Transaction { return tx }, txpool.ErrAlreadyKnown},
		{"replacement fee cap", func(t *testing.T, _ *types.Transaction) *types.Transaction {
			return signedPoolTx(t, 1, 0, 0, 21, 3, nil, nil)
		}, txpool.ErrReplaceUnderpriced},
		{"replacement tip", func(t *testing.T, _ *types.Transaction) *types.Transaction {
			return signedPoolTx(t, 1, 0, 0, 22, 2, nil, nil)
		}, txpool.ErrReplaceUnderpriced},
		{"sender nonce", func(t *testing.T, _ *types.Transaction) *types.Transaction {
			return signedPoolTx(t, 1, 0, 1, 22, 3, nil, nil)
		}, ErrSenderPending},
		{"capacity tip", func(t *testing.T, _ *types.Transaction) *types.Transaction {
			return signedPoolTx(t, 2, 0, 0, 20, 1, nil, nil)
		}, txpool.ErrUnderpriced},
		{"capacity equal priority", func(t *testing.T, _ *types.Transaction) *types.Transaction {
			return signedPoolTx(t, 2, 0, 0, 20, 2, nil, nil)
		}, txpool.ErrUnderpriced},
		{"capacity expiry", func(t *testing.T, _ *types.Transaction) *types.Transaction {
			return signedPoolTx(t, 2, 0, 0, 100, 20, &deadline, nil)
		}, txpool.ErrUnderpriced},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool, chain, other := setupFramePool(t, 1, nil)
			victim := signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil)
			addPoolTx(t, pool, victim, nil)
			waitAnnounced(t, pool)
			tx := tc.makeTx(t, victim)
			if tc.want != txpool.ErrAlreadyKnown {
				inner := poolTxData(tx)
				inner.Signatures[0].Signature = make([]byte, 65)
				tx = types.NewTx(inner)
				// The standalone interface still verifies signatures instead
				// of consulting replacement or capacity policy.
				if err := pool.ValidateTxBasics(tx); !errors.Is(err, types.ErrFrameTxInvalidSignature) {
					t.Fatalf("ValidateTxBasics got %v, want invalid signature", err)
				}
			}
			var statesOpened int
			chain.setHook(func(*types.Header) { statesOpened++ })
			events := make(chan core.NewTxsEvent, 1)
			sub := pool.SubscribeTransactions(events, false)
			defer sub.Unsubscribe()
			addPoolTx(t, pool, tx, tc.want)
			waitAnnounced(t, pool)
			if statesOpened != 0 {
				t.Fatalf("cheap rejection opened %d simulation states", statesOpened)
			}
			select {
			case got := <-events:
				t.Fatalf("cheap rejection emitted acceptance: %v", got)
			default:
			}
			assertLive(t, pool, victim, true)
			assertReleased(t, other, poolAddress(t, 2))
			assertFramePoolConsistent(t, pool)
		})
	}
}

// Pool policy rechecked at commit can reject a candidate after its prefix ran.
// Such rejections carry the execution marker, except duplicates.
func TestAdmissionPoolPrechecksRecheckedAtCommit(t *testing.T) {
	exposed := signedPoolTx(t, 2, 20, 0, 40, 4, nil, nil)
	for _, tc := range []struct {
		name        string
		payer       int
		setup       func(*state.StateDB)
		replacement bool
		duplicate   bool
		want        error
	}{
		{"known", 0, nil, false, true, txpool.ErrAlreadyKnown},
		{"replacement", 0, nil, true, false, txpool.ErrReplaceUnderpriced},
		{"capacity", 0, nil, false, false, txpool.ErrUnderpriced},
		{"coded paymaster", 20, func(s *state.StateDB) {
			s.SetCode(poolAddress(t, 20), approveCode(program.New(), types.ApprovePayment), tracing.CodeChangeUnspecified)
		}, false, false, txpool.ErrInflightTxLimitReached},
		{"payer exposure", 20, func(s *state.StateDB) {
			s.SetBalance(poolAddress(t, 20), txCost(exposed), tracing.BalanceChangeUnspecified)
		}, false, false, core.ErrInsufficientFunds},
	} {
		t.Run(tc.name, func(t *testing.T) {
			slots := uint64(1)
			if tc.payer != 0 {
				slots = 10
			}
			pool, chain, other := setupFramePool(t, slots, tc.setup)
			candidate := signedPoolTx(t, 1, tc.payer, 0, 22, 3, nil, nil)
			winner := candidate
			if tc.replacement {
				addPoolTx(t, pool, signedPoolTx(t, 1, 0, 0, 20, 2, nil, nil), nil)
				winner = signedPoolTx(t, 1, 0, 0, 40, 4, nil, nil)
			} else if !tc.duplicate {
				winner = signedPoolTx(t, 2, tc.payer, 0, 40, 4, nil, nil)
			}
			entered, release := make(chan struct{}), make(chan struct{})
			var statesOpened atomic.Int32
			chain.setHook(func(*types.Header) {
				if statesOpened.Add(1) == 1 {
					close(entered)
					<-release
				}
			})
			result := make(chan error, 1)
			go func() { result <- pool.Add([]*types.Transaction{candidate}, true)[0] }()
			<-entered // The candidate passed all early checks.
			addPoolTx(t, pool, winner, nil)
			close(release)
			err := <-result
			if !errors.Is(err, tc.want) {
				t.Fatalf("stale pool precheck committed: got %v, want %v", err, tc.want)
			}
			if marked := errors.Is(err, txpool.ErrValidationExecuted); marked == tc.duplicate {
				t.Fatalf("rejection %v: execution marker %v", err, marked)
			}
			assertLive(t, pool, winner, true)
			if !tc.duplicate {
				assertLive(t, pool, candidate, false)
			}
			if !tc.replacement && !tc.duplicate {
				assertReleased(t, other, poolAddress(t, 1))
			}
			assertFramePoolConsistent(t, pool)
		})
	}
}

// Nonce, payer exposure and coded paymaster rejections known from the read
// state precede execution and are unmarked; failed prefixes are marked.
func TestAdmissionExecutionMarker(t *testing.T) {
	for _, tc := range []struct {
		name     string
		payer    int
		setup    func(*state.StateDB)
		nonce    uint64
		want     error
		executed bool
	}{
		{"nonce", 0, nil, 1, core.ErrNonceTooHigh, false},
		{"payer exposure", 20, func(s *state.StateDB) {
			s.SetBalance(poolAddress(t, 20), txCost(signedPoolTx(t, 1, 20, 0, 20, 2, nil, nil)), tracing.BalanceChangeUnspecified)
		}, 0, core.ErrInsufficientFunds, false},
		{"coded paymaster", 20, func(s *state.StateDB) {
			s.SetCode(poolAddress(t, 20), approveCode(program.New(), types.ApprovePayment), tracing.CodeChangeUnspecified)
		}, 0, txpool.ErrInflightTxLimitReached, false},
		{"trace violation", 0, func(s *state.StateDB) {
			s.SetCode(poolAddress(t, 2), approveCode(program.New().Op(vm.NUMBER, vm.POP), types.ApproveExecutionAndPayment), tracing.CodeChangeUnspecified)
		}, 0, ErrTraceViolation, true},
		{"failed verify", 0, func(s *state.StateDB) {
			s.SetCode(poolAddress(t, 2), []byte{byte(vm.PUSH0), byte(vm.DUP1), byte(vm.REVERT)}, tracing.CodeChangeUnspecified)
		}, 0, core.ErrFrameTxInvalidExecution, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool, _, _ := setupFramePool(t, 10, tc.setup)
			if tc.payer != 0 {
				// Occupies the coded paymaster's single slot.
				addPoolTx(t, pool, signedPoolTx(t, 1, tc.payer, 0, 20, 2, nil, nil), nil)
			}
			pool.simulations.Store(0)
			err := pool.Add([]*types.Transaction{signedPoolTx(t, 2, tc.payer, tc.nonce, 20, 2, nil, nil)}, true)[0]
			if !errors.Is(err, tc.want) || errors.Is(err, txpool.ErrValidationExecuted) != tc.executed {
				t.Fatalf("got %v, want %v with execution marker %v", err, tc.want, tc.executed)
			}
			want := uint64(0)
			if tc.executed {
				want = 1
			}
			assertSimulations(t, pool, want)
			assertFramePoolConsistent(t, pool)
		})
	}
}
