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
	"maps"
	"math/big"
	"math/rand/v2"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
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
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// Stress account roles; setupFramePool funds keys 1-32.
const (
	stressPlain     = 12 // Keys 1-12 self-relay or use a sponsor.
	stressContested = 15 // Keys 13-15 are also toggled by a second reserver.
	stressSlot      = 16 // Approves only while stressSlotKey is zero.
	stressHelped    = 17 // Calls stressHelper, whose code blocks may break.
	stressBurner    = 18 // Burns its whole verification budget, then fails.
	stressSponsor   = 20 // Keys 20-22 are codeless sponsors funding a few txs.
	stressPaymaster = 23 // Coded paymaster; blocks may remove its code.
	stressHelper    = 30
	stressBystander = 31 // Never a dependency of any transaction.
	stressAccounts  = 32

	stressWorkers = 8
)

var stressSlotKey = common.HexToHash("0x42")

// stressAddErrors are the only admission outcomes of a well-formed run. Any
// other error, such as a reservation fault, fails the test.
var stressAddErrors = []error{
	txpool.ErrAlreadyKnown, txpool.ErrAlreadyReserved, txpool.ErrInflightTxLimitReached,
	txpool.ErrReplaceUnderpriced, txpool.ErrTxGasPriceTooLow, txpool.ErrUnderpriced,
	ErrSenderPending, ErrHeadChanged, ErrClosed, ErrTraceViolation,
	core.ErrInsufficientFunds, core.ErrNonceTooLow, core.ErrNonceTooHigh, core.ErrFrameTxInvalidExecution,
	types.ErrFrameTxInvalidSignature, types.ErrFrameTxSignerMismatch,
}

// auditReserver records double holds and releases of the pool's own senders,
// which the shared tracker tolerates or only logs.
type auditReserver struct {
	txpool.Reserver
	lock   sync.Mutex
	held   map[common.Address]bool
	faults []string
}

func (r *auditReserver) Hold(addr common.Address) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.held[addr] {
		r.faults = append(r.faults, fmt.Sprintf("double hold of %v", addr))
	}
	if err := r.Reserver.Hold(addr); err != nil {
		return err
	}
	r.held[addr] = true
	return nil
}

func (r *auditReserver) Release(addr common.Address) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if !r.held[addr] {
		r.faults = append(r.faults, fmt.Sprintf("release of unheld %v", addr))
	}
	delete(r.held, addr)
	if err := r.Reserver.Release(addr); err != nil {
		r.faults = append(r.faults, fmt.Sprintf("release of %v: %v", addr, err))
		return err
	}
	return nil
}

func (r *auditReserver) snapshot() (map[common.Address]bool, []string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	return maps.Clone(r.held), slices.Clone(r.faults)
}

// stressMutation is one state change, applied identically to a block's state
// and its access list so selective revalidation sees an exact description.
type stressMutation struct {
	kind    byte // 'n'once, 'b'alance, 'c'ode, 's'torage write or storage 'r'ead
	addr    common.Address
	nonce   uint64
	balance *uint256.Int
	code    []byte
	slot    common.Hash
	value   common.Hash
}

func (m *stressMutation) apply(s *state.StateDB) {
	switch m.kind {
	case 'n':
		s.SetNonce(m.addr, m.nonce, tracing.NonceChangeUnspecified)
	case 'b':
		s.SetBalance(m.addr, m.balance, tracing.BalanceChangeUnspecified)
	case 'c':
		s.SetCode(m.addr, m.code, tracing.CodeChangeUnspecified)
	case 's':
		s.SetState(m.addr, m.slot, m.value)
	}
}

func (m *stressMutation) record(list *bal.ConstructionBlockAccessList, index uint32) {
	switch m.kind {
	case 'n':
		list.NonceChange(m.addr, index, m.nonce)
	case 'b':
		list.BalanceChange(index, m.addr, m.balance)
	case 'c':
		list.CodeChange(m.addr, index, m.code)
	case 's':
		list.StorageWrite(index, m.addr, m.slot, m.value)
	case 'r':
		list.StorageRead(m.addr, m.slot)
	}
}

// poolStress drives one pool from concurrent admission, reader, reservation
// and chain goroutines. Every operation holds the gate shared, so quiescent
// checkpoints take it exclusively.
type poolStress struct {
	t     *testing.T
	seed  uint64
	pool  *FramePool
	chain *testBlockChain
	other txpool.Reserver
	holds *auditReserver
	addrs []common.Address // addrs[id] is the address of poolKey(id)
	unit  *uint256.Int     // Maximum cost of a sponsored tx at fee cap 30

	gate  sync.RWMutex
	stop  atomic.Bool
	audit chan struct{} // The chain goroutine requests a checkpoint after each reset
	halt  chan struct{} // Closed on shutdown, unblocking audit requests

	head   *types.Header // Last head passed to Reset, owned by the chain goroutine
	canon  []*types.Header
	resets atomic.Int64

	reserveLock sync.Mutex
	reserved    map[common.Address]bool // Holds of the second reserver

	rejectOnly sync.Map // Forged and burner hashes, never to be accepted or announced

	lock     sync.Mutex
	accepted map[common.Hash]int // Successful Add results per hash
	included map[common.Hash]bool
	recent   []common.Hash
	invalid  []*types.Transaction // Recent reject-only txs, placed into blocks
	outcomes map[string]int

	discovered, inserted map[common.Hash]int // Owned by the event consumer
}

func stressParams(t *testing.T) (uint64, int) {
	seed, ops := uint64(time.Now().UnixNano()), uint64(400)
	if testing.Short() {
		ops = 100
	}
	for name, value := range map[string]*uint64{"FRAMEPOOL_STRESS_SEED": &seed, "FRAMEPOOL_STRESS_OPS": &ops} {
		if env := os.Getenv(name); env != "" {
			parsed, err := strconv.ParseUint(env, 10, 64)
			if err != nil {
				t.Fatalf("invalid %s: %v", name, err)
			}
			*value = parsed
		}
	}
	return seed, int(ops)
}

// rng derives an independent stream per goroutine from the run seed.
func (s *poolStress) rng(stream uint64) *rand.Rand {
	return rand.New(rand.NewPCG(s.seed, stream))
}

func newPoolStress(t *testing.T, seed uint64) *poolStress {
	unit := txCost(signedPoolTx(t, 1, stressSponsor, 0, 30, 1, nil, nil))
	burner, loop := program.New().Jumpdest()
	pool, chain, other := setupFramePool(t, 10, func(s *state.StateDB) {
		for id := stressSponsor; id < stressPaymaster; id++ {
			s.SetBalance(poolAddress(t, id), new(uint256.Int).Mul(unit, uint256.NewInt(4)), tracing.BalanceChangeUnspecified)
		}
		slot := program.New().Push(stressSlotKey).Op(vm.SLOAD).Push(0).Op(vm.JUMPI)
		s.SetCode(poolAddress(t, stressSlot), approveCode(slot, types.ApproveExecutionAndPayment), tracing.CodeChangeUnspecified)
		helped := program.New().Call(nil, poolAddress(t, stressHelper), 0, 0, 0, 0, 0).Op(vm.POP)
		s.SetCode(poolAddress(t, stressHelped), approveCode(helped, types.ApproveExecutionAndPayment), tracing.CodeChangeUnspecified)
		s.SetCode(poolAddress(t, stressHelper), []byte{byte(vm.STOP)}, tracing.CodeChangeUnspecified)
		s.SetCode(poolAddress(t, stressBurner), burner.Jump(loop).Bytes(), tracing.CodeChangeUnspecified)
		s.SetCode(poolAddress(t, stressPaymaster), approveCode(program.New(), types.ApprovePayment), tracing.CodeChangeUnspecified)
	})
	// Audit the pool's own reservations before any concurrent use.
	holds := &auditReserver{Reserver: pool.reserver, held: make(map[common.Address]bool)}
	pool.reserver = holds
	s := &poolStress{
		t: t, seed: seed, pool: pool, chain: chain, other: other, holds: holds, unit: unit,
		addrs: make([]common.Address, stressAccounts+1),
		head:  chain.CurrentBlock(), canon: []*types.Header{chain.CurrentBlock()},
		audit: make(chan struct{}), halt: make(chan struct{}),
		reserved: make(map[common.Address]bool), accepted: make(map[common.Hash]int),
		included: make(map[common.Hash]bool), outcomes: make(map[string]int),
		discovered: make(map[common.Hash]int), inserted: make(map[common.Hash]int),
	}
	for id := 1; id <= stressAccounts; id++ {
		s.addrs[id] = poolAddress(t, id)
	}
	// Delay some state openings to widen the windows between an admission's
	// head snapshot and its commit, and between a reset's phases.
	chain.setHook(func(*types.Header) {
		if rand.IntN(4) == 0 {
			time.Sleep(time.Duration(rand.IntN(500)) * time.Microsecond)
		}
	})
	return s
}

// fail is safe from any goroutine and stops the remaining operations.
func (s *poolStress) fail(format string, args ...any) {
	s.t.Helper()
	s.t.Errorf("seed %d: %s", s.seed, fmt.Sprintf(format, args...))
	s.stop.Store(true)
}

func (s *poolStress) isRejectOnly(hash common.Hash) bool {
	_, ok := s.rejectOnly.Load(hash)
	return ok
}

func (s *poolStress) pickSender(rng *rand.Rand) int {
	switch n := rng.IntN(10); {
	case n < 6:
		return 1 + rng.IntN(stressPlain)
	case n < 8:
		return stressPlain + 1 + rng.IntN(stressContested-stressPlain)
	case n < 9:
		return stressSlot
	default:
		return stressHelped
	}
}

// pickPayer sponsors only codeless senders; coded senders approve both scopes.
func (s *poolStress) pickPayer(rng *rand.Rand, sender int) int {
	if sender > stressContested || rng.IntN(2) == 0 {
		return 0
	}
	return stressSponsor + rng.IntN(stressPaymaster-stressSponsor+1)
}

// nonce returns the pending nonce for a replacement, or the head nonce.
func (s *poolStress) nonce(sender int) uint64 {
	if pending, _ := s.pool.ContentFrom(s.addrs[sender]); len(pending) != 0 {
		return pending[0].Nonce()
	}
	return s.pool.Nonce(s.addrs[sender])
}

func (s *poolStress) fresh(rng *rand.Rand, deadline *uint64) *types.Transaction {
	sender := s.pickSender(rng)
	fee := []uint64{20, 25, 30, 40}[rng.IntN(4)]
	return signedPoolTx(s.t, sender, s.pickPayer(rng, sender), s.nonce(sender), fee, 1+rng.Uint64N(6), deadline, nil)
}

// replacement bumps both fees by at least the 10% price bump, or keeps the
// tip so the replacement is underpriced. The payer may move.
func (s *poolStress) replacement(rng *rand.Rand, bump bool) *types.Transaction {
	sender := s.pickSender(rng)
	pending, _ := s.pool.ContentFrom(s.addrs[sender])
	if len(pending) == 0 {
		return s.fresh(rng, nil)
	}
	old := pending[0]
	fee, tip := old.GasFeeCap().Uint64(), old.GasTipCap().Uint64()
	if bump {
		fee, tip = fee+(fee+9)/10+rng.Uint64N(3), tip+(tip+9)/10
	} else {
		fee += rng.Uint64N(3)
	}
	return signedPoolTx(s.t, sender, s.pickPayer(rng, sender), old.Nonce(), fee, min(tip, fee), nil, nil)
}

func (s *poolStress) forged(rng *rand.Rand) *types.Transaction {
	inner := poolTxData(s.fresh(rng, nil))
	sig := &inner.Signatures[rng.IntN(len(inner.Signatures))]
	sig.Signature = slices.Clone(sig.Signature)
	sig.Signature[1+rng.IntN(64)] ^= 0x01
	return types.NewTx(inner)
}

// burner models a sender contract that spends the full verification budget
// and then fails; every variant has a unique hash.
func (s *poolStress) burner(rng *rand.Rand) *types.Transaction {
	return signedPoolTx(s.t, stressBurner, 0, s.nonce(stressBurner), 20+rng.Uint64N(1000), 1+rng.Uint64N(6), nil, func(tx *types.FrameTx) {
		tx.Frames[0].GasLimits.Execution = MaxVerifyGas - types.FrameTxSignatureGas(&tx.Signatures[0])
	})
}

func (s *poolStress) add(txs []*types.Transaction, rejectOnly []bool) {
	for i, tx := range txs {
		if rejectOnly[i] {
			s.rejectOnly.Store(tx.Hash(), struct{}{})
		}
	}
	errs := s.pool.Add(txs, false)
	s.lock.Lock()
	defer s.lock.Unlock()
	for i, err := range errs {
		hash := txs[i].Hash()
		if s.recent = append(s.recent, hash); len(s.recent) > 256 {
			s.recent = s.recent[1:]
		}
		if rejectOnly[i] {
			if s.invalid = append(s.invalid, txs[i]); len(s.invalid) > 16 {
				s.invalid = s.invalid[1:]
			}
			if err == nil {
				s.fail("accepted invalid transaction %v", hash)
			}
		}
		if err == nil {
			s.accepted[hash]++
			s.outcomes["accepted"]++
			continue
		}
		known := slices.IndexFunc(stressAddErrors, func(want error) bool { return errors.Is(err, want) })
		if known < 0 {
			s.fail("unexpected admission error for %v: %v", hash, err)
			continue
		}
		s.outcomes[stressAddErrors[known].Error()]++
	}
}

func (s *poolStress) recentHash(rng *rand.Rand) common.Hash {
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.recent) == 0 {
		return common.Hash{}
	}
	return s.recent[rng.IntN(len(s.recent))]
}

func (s *poolStress) worker(id, ops int) {
	rng := s.rng(uint64(id))
	for range ops {
		if s.stop.Load() {
			return
		}
		s.gate.RLock()
		s.step(rng)
		s.gate.RUnlock()
	}
}

// sponsored batches transactions from several senders on one codeless sponsor.
func (s *poolStress) sponsored(rng *rand.Rand) ([]*types.Transaction, []bool) {
	sponsor := stressSponsor + rng.IntN(stressPaymaster-stressSponsor)
	var txs []*types.Transaction
	for _, index := range rng.Perm(stressContested)[:2+rng.IntN(3)] {
		sender := index + 1
		txs = append(txs, signedPoolTx(s.t, sender, sponsor, s.nonce(sender), 20+rng.Uint64N(11), 1+rng.Uint64N(6), nil, nil))
	}
	return txs, make([]bool, len(txs))
}

func (s *poolStress) step(rng *rand.Rand) {
	switch n := rng.IntN(100); {
	case n < 20:
		s.add([]*types.Transaction{s.fresh(rng, nil)}, []bool{false})
	case n < 28:
		s.add(s.sponsored(rng))
	case n < 38:
		s.add([]*types.Transaction{s.replacement(rng, true)}, []bool{false})
	case n < 42:
		s.add([]*types.Transaction{s.replacement(rng, false)}, []bool{false})
	case n < 50:
		s.add([]*types.Transaction{s.forged(rng)}, []bool{true})
	case n < 53:
		s.add([]*types.Transaction{s.burner(rng)}, []bool{true})
	case n < 57:
		sender := s.pickSender(rng)
		nonce := s.nonce(sender)
		if nonce == 0 || rng.IntN(2) == 0 {
			nonce += 1 + rng.Uint64N(2)
		} else {
			nonce--
		}
		s.add([]*types.Transaction{signedPoolTx(s.t, sender, s.pickPayer(rng, sender), nonce, 30, 2, nil, nil)}, []bool{false})
	case n < 63:
		deadline := s.pool.head.Load().Time + rng.Uint64N(4)
		if rng.IntN(5) == 0 {
			deadline -= 4
		}
		s.add([]*types.Transaction{s.fresh(rng, &deadline)}, []bool{false})
	case n < 66:
		if tx := s.pool.Get(s.recentHash(rng)); tx != nil {
			s.add([]*types.Transaction{tx}, []bool{false})
		}
	case n < 70:
		s.add([]*types.Transaction{s.fresh(rng, nil), s.forged(rng), s.replacement(rng, true)}, []bool{false, true, false})
	case n < 76:
		s.toggleReservation(rng)
	case n < 96:
		s.read(rng)
	case n < 98:
		tip := int64(1)
		if rng.IntN(2) == 0 {
			tip += rng.Int64N(4)
		}
		s.pool.SetGasTip(big.NewInt(tip))
	default:
		s.pool.Clear()
	}
}

// toggleReservation holds or releases a sender through the second handle.
func (s *poolStress) toggleReservation(rng *rand.Rand) {
	id := stressPlain + 1 + rng.IntN(stressContested-stressPlain)
	if rng.IntN(4) == 0 {
		id = 1 + rng.IntN(stressHelped)
	}
	addr := s.addrs[id]
	s.reserveLock.Lock()
	defer s.reserveLock.Unlock()
	if s.reserved[addr] {
		if err := s.other.Release(addr); err != nil {
			s.fail("second reserver release of %v: %v", addr, err)
		}
		delete(s.reserved, addr)
		return
	}
	switch err := s.other.Hold(addr); {
	case err == nil:
		s.reserved[addr] = true
	case !errors.Is(err, txpool.ErrAlreadyReserved):
		s.fail("second reserver hold of %v: %v", addr, err)
	}
}

// read checks every reader's result is internally consistent and never
// exposes a reject-only transaction.
func (s *poolStress) read(rng *rand.Rand) {
	check := func(tx *types.Transaction, sender common.Address) {
		if tx == nil {
			return
		}
		if s.isRejectOnly(tx.Hash()) {
			s.fail("reader exposed invalid transaction %v", tx.Hash())
		}
		if *tx.FrameSender() != sender {
			s.fail("transaction %v listed under sender %v", tx.Hash(), sender)
		}
	}
	switch rng.IntN(5) {
	case 0:
		var filter txpool.PendingFilter
		if rng.IntN(2) == 0 {
			filter.MinTip = uint256.NewInt(rng.Uint64N(5))
		}
		if rng.IntN(2) == 0 {
			filter.BaseFee = uint256.NewInt(10 + rng.Uint64N(25))
		}
		if rng.IntN(3) == 0 {
			filter.GasLimitCap = 50_000 + rng.Uint64N(150_000)
		}
		pending, count := s.pool.Pending(filter)
		if count != len(pending) {
			s.fail("pending count %d, %d senders", count, len(pending))
		}
		for sender, lazies := range pending {
			if len(lazies) != 1 {
				s.fail("sender %v has %d pending transactions", sender, len(lazies))
				continue
			}
			lazy := lazies[0]
			if (filter.BaseFee != nil && lazy.GasFeeCap.Lt(filter.BaseFee)) || (filter.GasLimitCap != 0 && lazy.Gas > filter.GasLimitCap) {
				s.fail("pending transaction %v violates filter", lazy.Hash)
			}
			if tx := lazy.Resolve(); tx != nil && tx.Hash() != lazy.Hash {
				s.fail("lazy transaction %v resolved to %v", lazy.Hash, tx.Hash())
			} else {
				check(tx, sender)
			}
		}
	case 1:
		pending, queued := s.pool.Content()
		if len(queued) != 0 {
			s.fail("framepool reported queued transactions")
		}
		for sender, txs := range pending {
			if len(txs) != 1 {
				s.fail("sender %v has %d pending transactions", sender, len(txs))
			}
			check(txs[0], sender)
		}
	case 2:
		sender := s.addrs[1+rng.IntN(stressBurner)]
		pending, queued := s.pool.ContentFrom(sender)
		if len(pending) > 1 || len(queued) != 0 {
			s.fail("sender %v content %d pending, %d queued", sender, len(pending), len(queued))
		}
		for _, tx := range pending {
			check(tx, sender)
		}
		s.pool.Nonce(sender)
	case 3:
		hash := s.recentHash(rng)
		if tx := s.pool.Get(hash); tx != nil {
			if tx.Hash() != hash {
				s.fail("Get(%v) returned %v", hash, tx.Hash())
			}
			check(tx, *tx.FrameSender())
		}
		if blob := s.pool.GetRLP(hash, 0); blob != nil {
			var tx types.Transaction
			if err := rlp.DecodeBytes(blob, &tx); err != nil || tx.Hash() != hash {
				s.fail("GetRLP(%v) decoded to %v: %v", hash, tx.Hash(), err)
			}
		}
		if meta := s.pool.GetMetadata(hash); meta != nil && meta.Type != types.FrameTxType {
			s.fail("metadata of %v has type %d", hash, meta.Type)
		}
		s.pool.Has(hash)
		s.pool.Status(hash)
	default:
		if pending, queued := s.pool.Stats(); uint64(pending) > s.pool.config.GlobalSlots || queued != 0 {
			s.fail("stats report %d pending, %d queued", pending, queued)
		}
	}
}

func (s *poolStress) chainWorker(steps int) {
	rng := s.rng(stressWorkers)
	for range steps {
		if s.stop.Load() {
			return
		}
		s.gate.RLock()
		s.chainStep(rng)
		s.gate.RUnlock()
		select {
		case s.audit <- struct{}{}:
		case <-s.halt:
			return
		}
	}
}

// chainStep resets the pool across a linear advance (possibly coalesced), a
// reorg onto a competing branch, or an unchanged head.
func (s *poolStress) chainStep(rng *rand.Rand) {
	old := s.head
	next := old
	switch n := rng.IntN(20); {
	case n < 4 && len(s.canon) > 1:
		depth := 1 + rng.IntN(min(3, len(s.canon)-1))
		var discarded []*types.Transaction
		for _, header := range s.canon[len(s.canon)-depth:] {
			discarded = append(discarded, s.chain.GetBlock(header.Hash(), header.Number.Uint64()).Transactions()...)
		}
		s.canon = s.canon[:len(s.canon)-depth]
		next = s.canon[len(s.canon)-1]
		for range depth + rng.IntN(2) {
			next = s.buildBlock(rng, next, discarded)
			s.canon = append(s.canon, next)
		}
	case n < 5:
	default:
		blocks := 1
		if rng.IntN(4) == 0 {
			blocks += 1 + rng.IntN(2)
		}
		for range blocks {
			next = s.buildBlock(rng, next, nil)
			s.canon = append(s.canon, next)
		}
	}
	s.pool.Reset(old, next)
	s.head = next
	s.resets.Add(1)
}

// buildBlock includes some carried and pending transactions, applies random
// dependency changes and attaches an exact access list, or none at all.
func (s *poolStress) buildBlock(rng *rand.Rand, parent *types.Header, carry []*types.Transaction) *types.Header {
	scratch, err := s.chain.StateAt(parent)
	if err != nil {
		s.fail("state at %d: %v", parent.Number, err)
		return parent
	}
	var (
		txs  []*types.Transaction
		muts []stressMutation
	)
	mutate := func(m stressMutation) {
		m.apply(scratch)
		muts = append(muts, m)
	}
	var candidates []*types.Transaction
	for _, tx := range carry {
		if rng.IntN(2) == 0 {
			candidates = append(candidates, tx)
		}
	}
	pending, _ := s.pool.Content()
	for _, sender := range slices.SortedFunc(maps.Keys(pending), common.Address.Cmp) {
		if rng.IntN(4) == 0 {
			candidates = append(candidates, pending[sender][0])
		}
	}
	s.lock.Lock()
	if len(s.invalid) != 0 && rng.IntN(3) == 0 {
		candidates = append(candidates, s.invalid[rng.IntN(len(s.invalid))])
	}
	s.lock.Unlock()
	for _, tx := range candidates {
		if slices.Contains(txs, tx) {
			continue
		}
		sender := *tx.FrameSender()
		if s.isRejectOnly(tx.Hash()) {
			txs = append(txs, tx) // Body only: recovery must still reject it.
			continue
		}
		if scratch.GetNonce(sender) != tx.Nonce() {
			continue
		}
		prefix, err := ClassifyPrefix(tx.Frames(), sender, tx.FrameSignatures())
		if err != nil {
			s.fail("pooled transaction %v has no prefix: %v", tx.Hash(), err)
			continue
		}
		txs = append(txs, tx)
		mutate(stressMutation{kind: 'n', addr: sender, nonce: tx.Nonce() + 1})
		payer := tx.Frames()[prefix.End].ResolvedTarget(sender)
		balance, underflow := new(uint256.Int).SubOverflow(scratch.GetBalance(payer), uint256.NewInt(tx.Gas()*10))
		if underflow {
			balance.Clear()
		}
		mutate(stressMutation{kind: 'b', addr: payer, balance: balance})
	}
	for range rng.IntN(4) {
		mutate(s.randomMutation(rng, scratch))
	}
	var accesses *bal.ConstructionBlockAccessList
	if rng.IntN(8) != 0 {
		accesses = bal.NewConstructionBlockAccessList()
		for i := range muts {
			muts[i].record(accesses, uint32(i+1))
		}
	}
	s.lock.Lock()
	for _, tx := range txs {
		s.included[tx.Hash()] = true
	}
	s.lock.Unlock()
	return s.chain.extend(parent, parent.Time+1, txs, accesses, func(statedb *state.StateDB) {
		for i := range muts {
			muts[i].apply(statedb)
		}
	})
}

// randomMutation changes a dependency field, an unrelated field of a known
// dependency, or an account no transaction depends on.
func (s *poolStress) randomMutation(rng *rand.Rand, scratch *state.StateDB) stressMutation {
	sponsor := s.addrs[stressSponsor+rng.IntN(stressPaymaster-stressSponsor)]
	switch rng.IntN(13) {
	case 0:
		addr := s.addrs[1+rng.IntN(stressHelped)]
		return stressMutation{kind: 'n', addr: addr, nonce: scratch.GetNonce(addr) + 1}
	case 1: // A one-wei transfer to a sponsor revalidates all it sponsors.
		balance := new(uint256.Int).AddUint64(scratch.GetBalance(sponsor), 1)
		return stressMutation{kind: 'b', addr: sponsor, balance: balance}
	case 2, 11, 12: // Refund between half a cost and six costs, or drain part of the balance.
		balance := new(uint256.Int).Mul(s.unit, uint256.NewInt(1+rng.Uint64N(12)))
		balance.Rsh(balance, 1)
		if rng.IntN(2) == 0 {
			balance.Mul(scratch.GetBalance(sponsor), uint256.NewInt(rng.Uint64N(4)))
			balance.Rsh(balance, 2)
		}
		return stressMutation{kind: 'b', addr: sponsor, balance: balance}
	case 3:
		balance := uint256.NewInt(1e18)
		if rng.IntN(2) == 0 {
			balance = new(uint256.Int).Rsh(s.unit, 1)
		}
		return stressMutation{kind: 'b', addr: s.addrs[1+rng.IntN(stressContested)], balance: balance}
	case 4:
		code := [][]byte{{byte(vm.STOP)}, {byte(vm.STOP)}, {byte(vm.INVALID)}, nil}[rng.IntN(4)]
		return stressMutation{kind: 'c', addr: s.addrs[stressHelper], code: code}
	case 5:
		addr := s.addrs[stressPaymaster]
		if rng.IntN(3) == 0 {
			addr = sponsor
		}
		var code []byte
		if rng.IntN(2) == 0 {
			code = approveCode(program.New(), types.ApprovePayment)
		}
		return stressMutation{kind: 'c', addr: addr, code: code}
	case 6:
		if rng.IntN(3) == 0 {
			return stressMutation{kind: 's', addr: s.addrs[stressSlot], slot: common.HexToHash("0x43"), value: common.HexToHash("0x01")}
		}
		var value common.Hash
		if rng.IntN(3) == 0 {
			value = common.HexToHash("0x01")
		}
		return stressMutation{kind: 's', addr: s.addrs[stressSlot], slot: stressSlotKey, value: value}
	case 7:
		addr := s.addrs[stressBystander]
		switch rng.IntN(4) {
		case 0:
			return stressMutation{kind: 'n', addr: addr, nonce: scratch.GetNonce(addr) + 1}
		case 1:
			return stressMutation{kind: 'b', addr: addr, balance: uint256.NewInt(rng.Uint64())}
		case 2:
			return stressMutation{kind: 'c', addr: addr, code: []byte{byte(vm.STOP)}}
		default:
			return stressMutation{kind: 's', addr: addr, slot: stressSlotKey, value: common.HexToHash("0x01")}
		}
	case 8: // Payers depend on balance and code, never on their nonce.
		return stressMutation{kind: 'n', addr: sponsor, nonce: scratch.GetNonce(sponsor) + 1}
	case 9:
		return stressMutation{kind: 'r', addr: s.addrs[stressSlot], slot: stressSlotKey}
	default:
		code := params.FrameTxExpiryVerifierCode
		if rng.IntN(3) == 0 {
			code = []byte{byte(vm.STOP)}
		}
		return stressMutation{kind: 'c', addr: params.FrameTxExpiryVerifier, code: code}
	}
}

// checkpoint quiesces every goroutine and verifies the pool against an
// independent revalidation of all live transactions at the reset head.
func (s *poolStress) checkpoint(t *testing.T) {
	s.gate.Lock()
	defer s.gate.Unlock()
	assertFramePoolConsistent(t, s.pool)

	p := s.pool
	p.lock.RLock()
	defer p.lock.RUnlock()
	head := p.head.Load()
	if head != s.head {
		t.Fatalf("seed %d: pool head %d differs from reset head %d", s.seed, head.Number, s.head.Number)
	}
	statedb, err := s.chain.StateAt(head)
	if err != nil {
		t.Fatal(err)
	}
	if p.slots > p.config.GlobalSlots {
		t.Fatalf("seed %d: %d slots used, capacity %d", s.seed, p.slots, p.config.GlobalSlots)
	}
	// Only the pool and the second reserver hold senders, so Has on the second
	// handle reports exactly the pool's reservations.
	held, faults := s.holds.snapshot()
	if len(faults) != 0 {
		t.Fatalf("seed %d: reservation faults: %v", s.seed, faults)
	}
	for _, addr := range s.addrs[1:] {
		live := p.txs[addr] != nil
		if s.other.Has(addr) != live || held[addr] != live {
			t.Fatalf("seed %d: sender %v live=%v, pool hold=%v, recorded=%v", s.seed, addr, live, s.other.Has(addr), held[addr])
		}
	}
	for payer, usage := range p.payers {
		if balance := statedb.GetBalance(payer); usage.reserved.Gt(balance) {
			t.Fatalf("seed %d: payer %v exposure %v exceeds head balance %v", s.seed, payer, &usage.reserved, balance)
		}
	}
	for sender, entry := range p.txs {
		hash := entry.tx.Hash()
		if s.isRejectOnly(hash) {
			t.Fatalf("seed %d: invalid transaction %v is live", s.seed, hash)
		}
		if entry.tipCap.Lt(p.gasTip.Load()) {
			t.Fatalf("seed %d: transaction %v tip below pool minimum", s.seed, hash)
		}
		// Selective revalidation must match a full revalidation at the head.
		fresh, err := simulate(p.chain.Config(), head, statedb.Copy(), entry.tx, entry.prefix)
		if err != nil {
			t.Fatalf("seed %d: live transaction %v of %v is invalid at head %d: %v", s.seed, hash, sender, head.Number, err)
		}
		if !reflect.DeepEqual(fresh, entry.simResult) {
			t.Fatalf("seed %d: stale simulation of %v at head %d: have %+v, want %+v", s.seed, hash, head.Number, entry.simResult, fresh)
		}
	}
}

func (s *poolStress) consume(discover, insert <-chan core.NewTxsEvent, quit <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	record := func(seen map[common.Hash]int, ev core.NewTxsEvent) {
		for _, tx := range ev.Txs {
			if s.isRejectOnly(tx.Hash()) {
				s.fail("announced invalid transaction %v", tx.Hash())
			}
			seen[tx.Hash()]++
		}
	}
	for {
		select {
		case ev := <-discover:
			record(s.discovered, ev)
		case ev := <-insert:
			record(s.inserted, ev)
		case <-quit:
			for {
				select {
				case ev := <-discover:
					record(s.discovered, ev)
				case ev := <-insert:
					record(s.inserted, ev)
				default:
					return
				}
			}
		}
	}
}

// checkEvents requires one discovery per accepted Add, and insertions beyond
// discoveries only for transactions recovered from a displaced block.
func (s *poolStress) checkEvents(t *testing.T) {
	for hash, count := range s.accepted {
		if s.discovered[hash] != count {
			t.Errorf("seed %d: transaction %v accepted %d times, discovered %d", s.seed, hash, count, s.discovered[hash])
		}
	}
	for hash, count := range s.discovered {
		if s.accepted[hash] == 0 {
			t.Errorf("seed %d: transaction %v discovered %d times, never accepted", s.seed, hash, count)
		}
	}
	for hash, count := range s.inserted {
		if count < s.discovered[hash] || (count > s.discovered[hash] && !s.included[hash]) {
			t.Errorf("seed %d: transaction %v inserted %d times, discovered %d, in block %v", s.seed, hash, count, s.discovered[hash], s.included[hash])
		}
	}
}

// closeRace closes the pool under concurrent admissions and a reset. Nothing
// may be admitted or remain reserved afterwards.
func (s *poolStress) closeRace(t *testing.T) {
	var wg sync.WaitGroup
	for id := range 4 {
		wg.Go(func() {
			rng := s.rng(stressWorkers + 1 + uint64(id))
			for range 4 {
				s.add([]*types.Transaction{s.fresh(rng, nil)}, []bool{false})
			}
		})
	}
	wg.Go(func() { s.chainStep(s.rng(stressWorkers + 5)) })
	runtime.Gosched()
	if err := s.pool.Close(); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if t.Failed() {
		return
	}
	tx := signedPoolTx(t, 1, 0, s.pool.Nonce(s.addrs[1]), 40, 6, nil, nil)
	if err := s.pool.Add([]*types.Transaction{tx}, false)[0]; !errors.Is(err, ErrClosed) {
		t.Fatalf("seed %d: admission after close: %v", s.seed, err)
	}
	held, faults := s.holds.snapshot()
	if len(held) != 0 || len(faults) != 0 {
		t.Fatalf("seed %d: after close, holds %v, faults %v", s.seed, held, faults)
	}
	for _, addr := range s.addrs[1:] {
		if s.other.Has(addr) {
			t.Fatalf("seed %d: sender %v still reserved after close", s.seed, addr)
		}
	}
	assertFramePoolConsistent(t, s.pool)
	if count, _ := s.pool.Stats(); count != 0 {
		t.Fatalf("seed %d: %d transactions after close", s.seed, count)
	}
}

// TestFramePoolStress interleaves admissions of valid, replacement, forged,
// budget-burning and reserved-sender transactions with resets across
// selective, fallback and reorg paths, tip changes, clears, readers and a
// competing reserver. Quiescent checkpoints compare the pool with a full
// revalidation. FRAMEPOOL_STRESS_SEED and FRAMEPOOL_STRESS_OPS reproduce or
// lengthen a run.
func TestFramePoolStress(t *testing.T) {
	seed, ops := stressParams(t)
	t.Logf("seed %d, %d operations per worker", seed, ops)
	s := newPoolStress(t, seed)

	timeout := 2 * time.Minute
	if deadline, ok := t.Deadline(); ok {
		timeout = min(timeout, time.Until(deadline)-5*time.Second)
	}
	watchdog := time.AfterFunc(timeout, func() {
		debug.SetTraceback("all")
		panic(fmt.Sprintf("framepool stress stalled for %v, seed %d", timeout, seed))
	})
	defer watchdog.Stop()
	baseline := runtime.NumGoroutine()

	discover, insert := make(chan core.NewTxsEvent, 64), make(chan core.NewTxsEvent, 64)
	subs := []event.Subscription{s.pool.SubscribeTransactions(discover, false), s.pool.SubscribeTransactions(insert, true)}
	quit, consumed := make(chan struct{}), make(chan struct{})
	go s.consume(discover, insert, quit, consumed)

	var wg sync.WaitGroup
	for id := range stressWorkers {
		wg.Go(func() { s.worker(id, ops) })
	}
	wg.Go(func() { s.chainWorker(ops / 2) })
	finished := make(chan struct{})
	go func() { wg.Wait(); close(finished) }()

	var stopped bool
	shutdown := func() {
		if stopped {
			return
		}
		stopped = true
		s.stop.Store(true)
		close(s.halt)
		<-finished
		waitAnnounced(t, s.pool) // Publication is asynchronous.
		for _, sub := range subs {
			sub.Unsubscribe()
		}
		close(quit)
		<-consumed
	}
	defer shutdown() // Also after a failed checkpoint
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for running := true; running; {
		select {
		case <-finished:
			running = false
		case <-s.audit:
			s.checkpoint(t)
		case <-ticker.C:
			s.checkpoint(t)
		}
	}
	shutdown()
	if t.Failed() {
		return
	}
	s.checkpoint(t)
	s.checkEvents(t)
	reinjected := 0
	for hash, count := range s.inserted {
		reinjected += count - s.discovered[hash]
	}
	t.Logf("%d resets, %d reorg recoveries, admission outcomes %v", s.resets.Load(), reinjected, s.outcomes)
	if s.outcomes["accepted"] == 0 || s.resets.Load() == 0 {
		t.Fatalf("seed %d: vacuous run", seed)
	}

	// Releasing the second reserver's holds must leave only the pool's.
	s.reserveLock.Lock()
	for addr := range s.reserved {
		if err := s.other.Release(addr); err != nil {
			t.Fatal(err)
		}
	}
	clear(s.reserved)
	s.reserveLock.Unlock()
	s.checkpoint(t)
	s.closeRace(t)

	for deadline := time.Now().Add(5 * time.Second); runtime.NumGoroutine() > baseline; time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<20)
			t.Fatalf("seed %d: %d goroutines remain, %d before the run\n%s", seed, runtime.NumGoroutine(), baseline, buf[:runtime.Stack(buf, true)])
		}
	}
}
