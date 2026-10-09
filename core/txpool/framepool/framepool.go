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
	"errors"
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

var (
	// ErrHeadChanged is retryable: validation could not commit against a
	// reconciled head after the bounded head-change retry.
	ErrHeadChanged = errors.New("framepool head changed during validation")

	// ErrSenderPending rejects a second nonce from an already pooled sender.
	ErrSenderPending = errors.New("sender already has a pending frame transaction")

	// ErrClosed is returned when admission is attempted after Close.
	ErrClosed = errors.New("framepool is closed")
)

// BlockChain provides the chain configuration and fresh state at a header.
// StateAt must return a disposable state, independent of earlier returned states.
type BlockChain interface {
	Config() *params.ChainConfig
	CurrentBlock() *types.Header
	GetBlock(hash common.Hash, number uint64) *types.Block
	StateAt(header *types.Header) (*state.StateDB, error)
}

// frameTx holds immutable admission inputs and the latest prefix simulation.
// Only the effective-tip key, simulation result and heap index change on Reset.
type frameTx struct {
	*simResult
	tx           *types.Transaction
	sender       common.Address
	prefix       Prefix
	maxCost      uint256.Int
	feeCap       uint256.Int
	tipCap       uint256.Int
	effectiveTip *big.Int
	index        int // Position in the eviction heap, -1 when not in it
}

type payerUsage struct {
	reserved uint256.Int
	txs      map[common.Hash]*frameTx
	coded    int
}

// FramePool holds one publicly valid frame transaction per sender. It never
// queues nonce gaps and does not execute operations after the validation prefix.
//
// Admission rejects duplicates and uncompetitive transactions before signature
// verification and simulation, then rechecks pool policy at the atomic commit.
//
// Every payer's aggregate maximum cost is bounded by its head-state balance.
// Coded sponsors additionally obey the non-canonical paymaster cap; there is no
// canonical-paymaster exception. Blob-carrying frames are not supported.
//
// Capacity eviction prefers the nearest expiry, then the lowest next-block
// effective tip. Deadline equality remains valid. Reset uses block access lists
// to revalidate dependencies and recovers displaced transactions on bounded reorgs.
type FramePool struct {
	config         Config
	chain          BlockChain
	signer         types.Signer
	hasPendingAuth func(common.Address) bool
	reserver       txpool.Reserver

	head    atomic.Pointer[types.Header]
	gasTip  atomic.Pointer[uint256.Int]
	state   *state.StateDB // Read state; never passed to simulation.
	baseFee *big.Int       // Next-block fee used only for eviction pricing.

	lock         sync.RWMutex
	resetLock    sync.Mutex // Serializes Reset calls, not simulations with readers.
	generation   uint64
	resetDone    chan struct{} // Non-nil while Reset is in flight.
	unreconciled bool          // The last Reset could not open the new head state.
	closed       bool
	simulations  atomic.Uint64 // Counts prefix executions, including rejected ones.

	txs       map[common.Address]*frameTx
	all       map[common.Hash]*frameTx
	payers    map[common.Address]*payerUsage
	slots     uint64
	evict     evictHeap
	byAccount map[common.Address]map[common.Hash]accountFields
	bySlot    map[common.Address]map[common.Hash]map[common.Hash]struct{}

	// Acceptances are queued under lock in commit order and published by a
	// single dispatcher, so neither admission nor Reset waits on subscribers.
	// Subscribers are in-process and must keep draining their channels; if one
	// stalls, the queue keeps only the newest announceLimit acceptances.
	announces    []announcement
	enqueued     uint64        // Acceptances queued so far, guarded by lock.
	published    atomic.Uint64 // Acceptances published or dropped so far.
	overflowing  bool          // Dropping since the queue last drained, guarded by lock.
	announceWake chan struct{}
	quit         chan struct{} // Closed by Close to stop the dispatcher.
	dispatched   chan struct{} // Closed when the dispatcher exits.
	discoverFeed event.Feed
	insertFeed   event.Feed
	scope        event.SubscriptionScope
}

// announcement is an accepted transaction awaiting publication. Transactions
// recovered from displaced blocks are insertions, not new discoveries.
type announcement struct {
	tx    *types.Transaction
	reorg bool
}

var _ txpool.SubPool = (*FramePool)(nil)

// New constructs a frame pool. Init puts it in lockstep with the other subpools.
func New(config Config, chain BlockChain, hasPendingAuth func(common.Address) bool) *FramePool {
	p := &FramePool{
		config: config.sanitize(), chain: chain, signer: types.LatestSigner(chain.Config()),
		hasPendingAuth: hasPendingAuth,
		txs:            make(map[common.Address]*frameTx), all: make(map[common.Hash]*frameTx),
		payers:       make(map[common.Address]*payerUsage),
		byAccount:    make(map[common.Address]map[common.Hash]accountFields),
		bySlot:       make(map[common.Address]map[common.Hash]map[common.Hash]struct{}),
		announceWake: make(chan struct{}, 1),
	}
	p.gasTip.Store(new(uint256.Int))
	return p
}

// Filter reports whether this pool handles the transaction.
func (p *FramePool) Filter(tx *types.Transaction) bool { return p.FilterType(tx.Type()) }

// FilterType reports whether this pool handles the transaction type.
func (p *FramePool) FilterType(kind byte) bool { return kind == types.FrameTxType }

// Init installs the reconciled head, read state and address reservation handle,
// and starts the acceptance dispatcher.
func (p *FramePool) Init(gasTip uint64, head *types.Header, reserver txpool.Reserver) error {
	if head == nil {
		head = p.chain.CurrentBlock()
	}
	statedb, err := p.chain.StateAt(head)
	if err != nil {
		return err
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	p.reserver, p.state = reserver, statedb
	p.baseFee = eip1559.CalcBaseFee(p.chain.Config(), head)
	p.head.Store(head)
	p.gasTip.Store(uint256.NewInt(gasTip))
	pooltipGauge.Update(int64(gasTip))
	p.updateMetrics()
	if p.quit == nil {
		p.quit, p.dispatched = make(chan struct{}), make(chan struct{})
		go p.dispatch(p.quit, p.dispatched)
	}
	return nil
}

// Close releases every sender reservation, terminates event subscriptions and
// stops the dispatcher. Acceptances not yet published are dropped, since no
// subscription remains to receive them.
func (p *FramePool) Close() error {
	p.scope.Close()
	p.lock.Lock()
	if p.closed {
		p.lock.Unlock()
		return nil
	}
	p.closed = true
	p.clear()
	p.published.Add(uint64(len(p.announces)))
	p.announces = nil
	announceQueueGauge.Update(0)
	quit := p.quit
	p.lock.Unlock()
	if quit != nil {
		close(quit)
		<-p.dispatched
	}
	return nil
}

// announceLimit bounds the acceptances awaiting publication.
func (p *FramePool) announceLimit() int {
	return int(4 * min(p.config.GlobalSlots, (1<<20)/4))
}

// announce queues an acceptance in commit order. When a stalled subscriber
// lets the queue reach its bound, the oldest acceptances are dropped instead
// of blocking admission. The caller holds the lock.
func (p *FramePool) announce(tx *types.Transaction, reorg bool) {
	if len(p.announces) >= p.announceLimit() {
		if !p.overflowing {
			p.overflowing = true
			log.Warn("Framepool subscriber stalled, dropping oldest acceptance events", "queued", len(p.announces))
		}
		p.announces[0] = announcement{}
		p.announces = p.announces[1:]
		p.published.Add(1)
		announceDroppedMeter.Mark(1)
	}
	p.announces = append(p.announces, announcement{tx: tx, reorg: reorg})
	p.enqueued++
	announceQueueGauge.Update(int64(len(p.announces)))
	select {
	case p.announceWake <- struct{}{}:
	default:
	}
}

// dispatch publishes queued acceptances in commit order, one event per
// transaction: a discovery, unless recovered from a reorg, then an insertion.
// It never holds the pool lock while sending, so subscribers may read the pool.
// Acceptances are taken one at a time, so a stalled send holds back only one
// and everything else stays subject to the queue bound.
func (p *FramePool) dispatch(quit, done chan struct{}) {
	defer close(done)
	for {
		select {
		case <-p.announceWake:
		case <-quit:
			return
		}
		for {
			p.lock.Lock()
			if len(p.announces) == 0 {
				p.announces, p.overflowing = nil, false
				p.lock.Unlock()
				break
			}
			announced := p.announces[0]
			p.announces[0] = announcement{}
			p.announces = p.announces[1:]
			announceQueueGauge.Update(int64(len(p.announces)))
			p.lock.Unlock()

			event := core.NewTxsEvent{Txs: []*types.Transaction{announced.tx}}
			if !announced.reorg {
				p.discoverFeed.Send(event)
			}
			p.insertFeed.Send(event)
			p.published.Add(1)
		}
	}
}

// SetGasTip updates the tip-cap floor and releases transactions below it.
func (p *FramePool) SetGasTip(tip *big.Int) {
	p.lock.Lock()
	defer p.lock.Unlock()
	newTip := uint256.MustFromBig(tip)
	p.gasTip.Store(newTip)
	for _, entry := range p.txs {
		if entry.tipCap.Lt(newTip) {
			p.remove(entry, true)
			evictedMeter.Mark(1)
		}
	}
	pooltipGauge.Update(tip.Int64())
	p.updateMetrics()
}

// Has reports whether the transaction is currently accepted in the pool.
func (p *FramePool) Has(hash common.Hash) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.all[hash] != nil
}

// Get retrieves a live accepted transaction.
func (p *FramePool) Get(hash common.Hash) *types.Transaction {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if entry := p.all[hash]; entry != nil {
		return entry.tx
	}
	return nil
}

// GetRLP retrieves a live transaction's network encoding.
func (p *FramePool) GetRLP(hash common.Hash, _ uint) []byte {
	p.lock.RLock()
	defer p.lock.RUnlock()
	entry := p.all[hash]
	if entry == nil {
		return nil
	}
	encoded, err := rlp.EncodeToBytes(entry.tx)
	if err != nil {
		log.Error("Failed to encode framepool transaction", "hash", hash, "err", err)
		return nil
	}
	return encoded
}

// GetMetadata retrieves a live transaction's type and encoded size.
func (p *FramePool) GetMetadata(hash common.Hash) *txpool.TxMetadata {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if entry := p.all[hash]; entry != nil {
		return &txpool.TxMetadata{Type: types.FrameTxType, Size: entry.tx.Size(), SizeWithoutBlob: entry.tx.Size()}
	}
	return nil
}

// Pending returns at most one lazy transaction per sender, applying the miner's
// fee and gas constraints independently. Frame transactions carrying blobs are
// not admitted, so the blob-only view is empty.
func (p *FramePool) Pending(filter txpool.PendingFilter) (map[common.Address][]*txpool.LazyTransaction, int) {
	if filter.BlobTxs {
		return nil, 0
	}
	p.lock.RLock()
	defer p.lock.RUnlock()
	pending := make(map[common.Address][]*txpool.LazyTransaction, len(p.txs))
	for sender, entry := range p.txs {
		if filter.GasLimitCap != 0 && entry.tx.Gas() > filter.GasLimitCap {
			continue
		}
		if filter.BaseFee != nil && entry.feeCap.Lt(filter.BaseFee) {
			continue
		}
		if filter.MinTip != nil {
			tip := entry.tipCap
			if filter.BaseFee != nil {
				var available uint256.Int
				available.Sub(&entry.feeCap, filter.BaseFee)
				if available.Lt(&tip) {
					tip = available
				}
			}
			if tip.Lt(filter.MinTip) {
				continue
			}
		}
		pending[sender] = []*txpool.LazyTransaction{{
			Pool: p, Hash: entry.tx.Hash(), Time: entry.tx.Time(),
			GasFeeCap: new(uint256.Int).Set(&entry.feeCap), GasTipCap: new(uint256.Int).Set(&entry.tipCap),
			Gas: entry.tx.Gas(),
		}}
	}
	return pending, len(pending)
}

// SubscribeTransactions separates first discovery from reorg-inclusive insertion
// notifications, as in the blob pool. Recovered reorg transactions are inserted
// but not rediscovered. Events are published asynchronously, in commit order.
func (p *FramePool) SubscribeTransactions(ch chan<- core.NewTxsEvent, reorgs bool) event.Subscription {
	feed := &p.discoverFeed
	if reorgs {
		feed = &p.insertFeed
	}
	sub := feed.Subscribe(ch)
	tracked := p.scope.Track(sub)
	if tracked == nil {
		// Closed: an untracked subscription could block the final dispatch.
		sub.Unsubscribe()
	}
	return tracked
}

// Nonce returns the pending legacy account nonce. A keyed pending transaction
// does not consume that nonce, so RPC callers still see the head-state value.
// StateDB reads may populate its cache, so take a write lock.
func (p *FramePool) Nonce(addr common.Address) uint64 {
	p.lock.Lock()
	defer p.lock.Unlock()
	if entry := p.txs[addr]; entry != nil && types.HasLegacyNonceKeys(entry.tx.FrameNonceKeys()) {
		return entry.tx.Nonce() + 1
	}
	return p.state.GetNonce(addr)
}

// Stats returns the pending count; framepool never queues transactions.
func (p *FramePool) Stats() (int, int) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.txs), 0
}

// Content returns only pending transactions.
func (p *FramePool) Content() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	pending := make(map[common.Address][]*types.Transaction, len(p.txs))
	for sender, entry := range p.txs {
		pending[sender] = []*types.Transaction{entry.tx}
	}
	return pending, make(map[common.Address][]*types.Transaction)
}

// ContentFrom returns the sender's pending transaction, without a queue.
func (p *FramePool) ContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if entry := p.txs[addr]; entry != nil {
		return []*types.Transaction{entry.tx}, nil
	}
	return nil, nil
}

// Status reports pending for a live transaction and unknown otherwise.
func (p *FramePool) Status(hash common.Hash) txpool.TxStatus {
	if p.Has(hash) {
		return txpool.TxStatusPending
	}
	return txpool.TxStatusUnknown
}

// Clear releases all transactions, dependencies, payer exposure and sender holds.
func (p *FramePool) Clear() {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.clear()
}

func (p *FramePool) clear() {
	p.generation++
	for _, entry := range p.txs {
		p.remove(entry, true)
	}
	p.updateMetrics()
}

// insert and remove are the only admission mutation paths. Callers hold the
// write lock and have completed every fallible policy and reservation check.
func (p *FramePool) insert(entry *frameTx) {
	hash := entry.tx.Hash()
	p.txs[entry.sender], p.all[hash] = entry, entry
	p.slots += numSlots(entry.tx)
	heap.Push(&p.evict, entry)
	usage := p.payers[entry.payer]
	if usage == nil {
		usage = &payerUsage{txs: make(map[common.Hash]*frameTx)}
		p.payers[entry.payer] = usage
	}
	usage.reserved.Add(&usage.reserved, &entry.maxCost) // Checked before commit.
	usage.txs[hash] = entry
	if entry.codedPaymaster {
		usage.coded++
	}
	p.indexDependencies(entry)
}

func (p *FramePool) remove(entry *frameTx, release bool) {
	hash := entry.tx.Hash()
	delete(p.txs, entry.sender)
	delete(p.all, hash)
	p.slots -= numSlots(entry.tx)
	heap.Remove(&p.evict, entry.index)
	if usage := p.payers[entry.payer]; usage != nil && usage.txs[hash] != nil {
		usage.reserved.Sub(&usage.reserved, &entry.maxCost)
		delete(usage.txs, hash)
		if entry.codedPaymaster {
			usage.coded--
		}
		if len(usage.txs) == 0 {
			delete(p.payers, entry.payer)
		}
	}
	p.unindexDependencies(entry)
	if release {
		p.reserver.Release(entry.sender)
	}
}

func (p *FramePool) indexDependencies(entry *frameTx) {
	hash := entry.tx.Hash()
	for addr, fields := range entry.dependencies.accounts {
		if p.byAccount[addr] == nil {
			p.byAccount[addr] = make(map[common.Hash]accountFields)
		}
		p.byAccount[addr][hash] = fields
	}
	for location := range entry.dependencies.slots {
		addr, slot := location.address, location.slot
		if p.bySlot[addr] == nil {
			p.bySlot[addr] = make(map[common.Hash]map[common.Hash]struct{})
		}
		if p.bySlot[addr][slot] == nil {
			p.bySlot[addr][slot] = make(map[common.Hash]struct{})
		}
		p.bySlot[addr][slot][hash] = struct{}{}
	}
}

func (p *FramePool) unindexDependencies(entry *frameTx) {
	hash := entry.tx.Hash()
	for addr := range entry.dependencies.accounts {
		delete(p.byAccount[addr], hash)
		if len(p.byAccount[addr]) == 0 {
			delete(p.byAccount, addr)
		}
	}
	for location := range entry.dependencies.slots {
		addr, slot := location.address, location.slot
		delete(p.bySlot[addr][slot], hash)
		if len(p.bySlot[addr][slot]) == 0 {
			delete(p.bySlot[addr], slot)
		}
		if len(p.bySlot[addr]) == 0 {
			delete(p.bySlot, addr)
		}
	}
}

func (p *FramePool) updateMetrics() {
	pendingGauge.Update(int64(len(p.txs)))
	slotusedGauge.Update(int64(p.slots))
}
