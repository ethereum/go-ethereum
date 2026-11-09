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

package core

// This file holds the chain writer, which takes the database write of a block
// off the engine API response path. InsertBlockWithoutSetHead hands the write
// over once the block is validated, and the writer runs it in two phases.
//
// The first phase makes the state of the block readable. It commits the state,
// writes the new contract code and adds the diff layer of the block to the path
// database. Next to the commit, the same loop flattens the diff layers that the
// new one would push too far down into the disk layer, appending their state
// history on the way. That only needs the layers below the parent. The second
// phase persists the block. It writes the header, body, receipts and preimages
// in one batch. Both phases keep the order the blocks came in, and the second
// one can lag behind the first by a few blocks.
//
// Only a block that builds on the last one handed over gets queued, a block on
// another branch is written synchronously. So the queued blocks form one line,
// and no flattening in the queue drops a layer of the blocks before it.
//
// Until its data is on disk, the chain serves a queued block from memory and
// counts its state as present, and the last block written counts as present
// too. Opening that state waits for the first phase. Processing a child block
// waits for the flattening as well, since reading the path database would wait
// for it anyway, and reports that wait as commit time.
//
// QueueHead moves the chain head in memory right away. Until its markers are on
// disk, the chain serves the queued head by number and its transactions by hash
// from memory too. The markers are written after the second phase of the block,
// so they never reach disk before the block they point at, and the chain events
// go out after them. Any other update of the chain waits for all the queued
// work first.

import (
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// maxQueuedWrites caps the blocks on their way to disk. One more waits for the oldest.
const maxQueuedWrites = 8

var (
	stateWriteTimer   = metrics.NewRegisteredResettingTimer("chain/write/state", nil)
	flattenWriteTimer = metrics.NewRegisteredResettingTimer("chain/write/flatten", nil)
	persistWriteTimer = metrics.NewRegisteredResettingTimer("chain/write/persist", nil)
	headWriteTimer    = metrics.NewRegisteredResettingTimer("chain/write/head", nil)
)

// writeJob is the write of a validated block.
type writeJob struct {
	block      *types.Block
	header     *types.Header
	receipts   types.Receipts
	preimages  map[common.Hash][]byte
	parentRoot common.Hash
	rules      params.Rules
	statedb    *state.StateDB // dropped once the state is committed

	committed chan struct{} // closed once the state is readable
	stateTime time.Duration // time the state commit took

	flattened   chan struct{} // closed once the state is committed and the old layers are flattened
	flattenTime time.Duration // time the flattening took

	encodeOnce  sync.Once
	receiptsRLP rlp.RawValue // receipts in their storage encoding, for readers
}

// headTx locates a transaction in a queued head.
type headTx struct {
	block *types.Block
	index uint64
}

// chainWriter writes the blocks handed over by InsertBlockWithoutSetHead in the background.
type chainWriter struct {
	bc *BlockChain

	lock      sync.RWMutex
	jobs      map[common.Hash]*writeJob // blocks whose data is not on disk yet
	tail      chan struct{}             // closed once the last queued task has run
	head      *types.Block              // last queued head, nil if none since the last drain
	heads     map[uint64]*types.Block   // queued heads whose markers are not on disk yet
	headTxs   map[common.Hash]headTx    // transactions of those heads, whose lookups are not on disk yet
	validated common.Hash               // last block handed over
	persisted common.Hash               // state root of the last block written, zero if none since the last drain
	running   bool                      // whether the loops are started
	closed    bool                      // whether the writer is stopped

	slots   chan struct{}  // held by every job until its block data is on disk
	stateCh chan *writeJob // jobs waiting for their first phase
	taskCh  chan func()    // second phases and head updates, in order
	quit    chan struct{}
	wg      sync.WaitGroup

	hook atomic.Pointer[func(step string, block *types.Block)] // see SetWriteHookForTesting
}

// newChainWriter creates the writer of the chain. Its loops start with the first job.
func newChainWriter(bc *BlockChain) *chainWriter {
	return &chainWriter{
		bc:      bc,
		jobs:    make(map[common.Hash]*writeJob),
		heads:   make(map[uint64]*types.Block),
		headTxs: make(map[common.Hash]headTx),
		slots:   make(chan struct{}, maxQueuedWrites),
		stateCh: make(chan *writeJob, maxQueuedWrites),
		// A head update needs a block handed over after the previous update,
		// so there are never more than twice the jobs plus one tasks queued.
		taskCh: make(chan func(), 2*maxQueuedWrites+1),
		quit:   make(chan struct{}),
	}
}

// newWriteJob prepares the write of a validated block for the chain writer.
func (bc *BlockChain) newWriteJob(block *types.Block, receipts types.Receipts, statedb *state.StateDB, parentRoot common.Hash) (*writeJob, error) {
	// Same check as the synchronous write does before writing anything
	if !bc.HasHeader(block.ParentHash(), block.NumberU64()-1) {
		return nil, consensus.ErrUnknownAncestor
	}
	return &writeJob{
		block:      block,
		header:     block.Header(),
		receipts:   receipts,
		preimages:  statedb.Preimages(),
		parentRoot: parentRoot,
		rules:      bc.chainConfig.Rules(block.Number(), block.Difficulty().Sign() == 0, block.Time()),
		statedb:    statedb,
		committed:  make(chan struct{}),
		flattened:  make(chan struct{}),
	}, nil
}

// canDeferWrite reports whether the write of the block can be left to the chain writer.
func (bc *BlockChain) canDeferWrite(block *types.Block) bool {
	// Only the merkle state in the path database can have its layers flattened later
	if bc.triedb.Scheme() != rawdb.PathScheme || bc.chainConfig.IsUBT(block.Number(), block.Time()) {
		return false
	}
	// The state hook of a live tracer expects the state update before the import returns
	if bc.logger != nil && bc.logger.OnStateUpdate != nil {
		return false
	}
	// Self validation builds a witness, which the state commit still adds to
	return !bc.cfg.StatelessSelfValidation
}

// QueueHead queues head as the new chain head behind its block write, or returns false if it can't.
func (bc *BlockChain) QueueHead(head *types.Block) bool {
	// A repeated update for the last queued or the current head has nothing to do
	if bc.writer.isHead(head.Hash(), bc.CurrentBlock()) {
		return true
	}
	if !bc.chainmu.TryLock() {
		return false
	}
	defer bc.chainmu.Unlock()

	// Only the last block handed over can be queued, on top of the last queued head
	if bc.writer.queueHead(head, bc.CurrentBlock()) {
		return true
	}
	// The caller falls back to SetCanonical, which works on what's on disk
	bc.writer.drain()
	return false
}

// WaitWrites blocks until the block writes and head updates queued so far have landed.
func (bc *BlockChain) WaitWrites() {
	bc.writer.wait()
}

// SetWriteHookForTesting sets a hook run before each writer step: state, persist, head or drain.
func (bc *BlockChain) SetWriteHookForTesting(hook func(step string, block *types.Block)) {
	bc.writer.hook.Store(&hook)
}

// queue hands the job to the writer. It assumes the chain mutex is held.
func (w *chainWriter) queue(job *writeJob) {
	// Wait for a free slot, the oldest block frees its slot once it's on disk
	w.slots <- struct{}{}

	// Serve the block from memory until its data is on disk
	hash, done := job.block.Hash(), make(chan struct{})

	w.lock.Lock()
	if !w.running {
		w.running = true
		w.wg.Add(2)
		go w.stateLoop()
		go w.persistLoop()
	}
	w.jobs[hash] = job
	w.validated = hash
	w.tail = done
	w.lock.Unlock()

	// Queue both phases. The chain mutex keeps the queues in the same order,
	// and the second phase waits for the first one of its block.
	w.stateCh <- job
	w.taskCh <- func() {
		<-job.committed
		w.runHook("persist", job.block)
		w.persist(job)

		// Readers go to the database from now on, but the state still counts
		// as present without asking it, see hasState
		w.lock.Lock()
		delete(w.jobs, hash)
		w.persisted = job.header.Root
		w.lock.Unlock()

		<-w.slots
		close(done)
	}
}

// queueHead queues the head update behind the write of its block. It assumes the chain mutex is held.
func (w *chainWriter) queueHead(head *types.Block, current *types.Header) bool {
	w.lock.Lock()

	// The head has to extend the last queued head, or the current one if none is queued
	base := current.Hash()
	if w.head != nil {
		base = w.head.Hash()
	}
	if head.Hash() == base {
		w.lock.Unlock()
		return true
	}
	if head.Hash() != w.validated || head.ParentHash() != base {
		w.lock.Unlock()
		return false
	}
	done := make(chan struct{})
	w.head = head
	w.tail = done

	// Serve the head by number and its transactions by hash until their markers
	// and lookups are on disk
	w.heads[head.NumberU64()] = head
	for i, tx := range head.Transactions() {
		w.headTxs[tx.Hash()] = headTx{block: head, index: uint64(i)}
	}
	w.lock.Unlock()

	// The chain moves to the head in memory right away, the lookups above
	// resolve it for anyone who sees it
	w.bc.setCurrentHead(head.Header())

	// The markers get written after the second phase of the block, so the block
	// data is on disk before them. It doesn't need the chain mutex, every other
	// head update waits for the writer first.
	w.taskCh <- func() {
		w.runHook("head", head)
		start := time.Now()
		w.bc.writeHeadMarkers(head)

		// Lookups go to the database from now on
		w.lock.Lock()
		delete(w.heads, head.NumberU64())
		for _, tx := range head.Transactions() {
			delete(w.headTxs, tx.Hash())
		}
		w.lock.Unlock()

		// The events go out once everything they point at is on disk
		w.bc.sendHeadEvents(head, start)
		headWriteTimer.UpdateSince(start)
		close(done)
	}
	return true
}

// canonicalHash returns the hash of the queued head with the given number, or zero if there's none.
func (w *chainWriter) canonicalHash(number uint64) common.Hash {
	w.lock.RLock()
	defer w.lock.RUnlock()

	if head := w.heads[number]; head != nil {
		return head.Hash()
	}
	return common.Hash{}
}

// headTransaction returns a transaction of a queued head with its block and index, or nil if there's none.
func (w *chainWriter) headTransaction(hash common.Hash) (*types.Transaction, *types.Block, uint64) {
	w.lock.RLock()
	defer w.lock.RUnlock()

	loc, ok := w.headTxs[hash]
	if !ok {
		return nil, nil, 0
	}
	return loc.block.Transactions()[loc.index], loc.block, loc.index
}

// extends reports whether the block builds on the last block handed over, or none is since the last drain.
func (w *chainWriter) extends(block *types.Block) bool {
	w.lock.RLock()
	defer w.lock.RUnlock()

	return w.validated == (common.Hash{}) || block.ParentHash() == w.validated
}

// isHead reports whether hash is the last queued head, or the current head if none is queued.
func (w *chainWriter) isHead(hash common.Hash, current *types.Header) bool {
	w.lock.RLock()
	defer w.lock.RUnlock()

	if w.head != nil {
		return w.head.Hash() == hash
	}
	return current.Hash() == hash
}

// stateLoop runs the first phase of the handed over writes and the flattening next to each, in order.
func (w *chainWriter) stateLoop() {
	defer w.wg.Done()
	for {
		select {
		case job := <-w.stateCh:
			// The flattening only needs the layers below the parent, so it runs
			// next to the commit instead of after it
			capped := make(chan struct{})
			go func() {
				w.flatten(job)
				close(capped)
			}()
			w.runHook("state", job.block)
			w.commitState(job)
			close(job.committed)

			// A child block waits for both, so no flattening holds up its reads
			<-capped
			close(job.flattened)
		case <-w.quit:
			return
		}
	}
}

// persistLoop runs the second phases and the head updates in the order they were queued.
func (w *chainWriter) persistLoop() {
	defer w.wg.Done()
	for {
		select {
		case task := <-w.taskCh:
			task()
		case <-w.quit:
			return
		}
	}
}

// commitState commits the state of the block and adds its diff layer, which makes it readable.
func (w *chainWriter) commitState(job *writeJob) {
	start := time.Now()
	if _, err := job.statedb.CommitLayer(job.rules, job.block.NumberU64()); err != nil {
		// The block is valid and accepted already, so the database is broken
		log.Crit("Failed to commit block state", "number", job.block.Number(), "hash", job.block.Hash(), "err", err)
	}
	accountCommitTimer.Update(job.statedb.AccountCommits)
	storageCommitTimer.Update(job.statedb.StorageCommits)
	triedbCommitTimer.Update(job.statedb.DatabaseCommits)

	// Let go of the state, only the block data is needed from here on
	job.statedb = nil

	job.stateTime = time.Since(start)
	stateWriteTimer.Update(job.stateTime)
}

// flatten makes room for the layer of the block, flattening the diff layers that would end up too far down into the disk layer.
func (w *chainWriter) flatten(job *writeJob) {
	start := time.Now()

	// A block that doesn't change the state adds no layer to make room for
	if job.block.Root() != job.parentRoot {
		if err := w.bc.triedb.CapLayers(job.parentRoot); err != nil {
			log.Crit("Failed to flatten state layers", "number", job.block.Number(), "hash", job.block.Hash(), "err", err)
		}
	}
	job.flattenTime = time.Since(start)
	flattenWriteTimer.Update(job.flattenTime)
}

// persist writes the block data, while the flattening may still run on the state loop.
func (w *chainWriter) persist(job *writeJob) {
	start := time.Now()
	w.bc.writeBlockData(job.block, job.receipts, job.preimages)
	elapsed := time.Since(start)
	persistWriteTimer.Update(elapsed)

	// The block only counts as written once its flattening is done too, so
	// draining the writer covers the flattening
	<-job.flattened

	// The slow block log stops at the handover, so the write gets a line of
	// its own when that log is on
	if limit := w.bc.slowBlockThreshold; limit >= 0 && job.stateTime+job.flattenTime+elapsed >= limit {
		log.Info("Wrote block", "number", job.block.Number(), "hash", job.block.Hash(),
			"state", common.PrettyDuration(job.stateTime), "flatten", common.PrettyDuration(job.flattenTime),
			"persist", common.PrettyDuration(elapsed))
	}
}

// wait blocks until the tasks queued so far have run.
func (w *chainWriter) wait() {
	w.lock.RLock()
	tail := w.tail
	w.lock.RUnlock()

	if tail == nil {
		return
	}
	select {
	case <-tail:
	default:
		// Tell the test hook this caller has to wait
		w.runHook("drain", nil)
		<-tail
	}
}

// drain lets the queued work land before a synchronous chain update. It assumes the chain mutex is held.
func (w *chainWriter) drain() {
	w.wait()

	// The synchronous update may move the head, so the next queued head starts
	// over from it. It may also drop the last state written, so forget that too.
	w.lock.Lock()
	w.head, w.validated, w.persisted = nil, common.Hash{}, common.Hash{}
	w.lock.Unlock()
}

// close lets the queued work land and stops the writer.
func (w *chainWriter) close() {
	w.drain()

	w.lock.Lock()
	if w.closed {
		w.lock.Unlock()
		return
	}
	w.closed = true
	running := w.running
	w.lock.Unlock()

	if running {
		close(w.quit)
		w.wg.Wait()
	}
}

// job returns the queued write of the block with the given hash, or nil.
func (w *chainWriter) job(hash common.Hash) *writeJob {
	w.lock.RLock()
	defer w.lock.RUnlock()

	return w.jobs[hash]
}

// header returns the header of a queued block, or nil.
func (w *chainWriter) header(hash common.Hash) *types.Header {
	if job := w.job(hash); job != nil {
		return job.header
	}
	return nil
}

// hasState reports whether a queued block, or the last one written, has the given state root.
func (w *chainWriter) hasState(root common.Hash) bool {
	w.lock.RLock()
	defer w.lock.RUnlock()

	// The last block written is still one of the newest layers. The queued
	// blocks all build on it, so their flattening keeps it, and whatever else
	// could drop it drains the writer first. Answering here keeps a check of
	// the head state from waiting on a flattening.
	if root == w.persisted && root != (common.Hash{}) {
		return true
	}
	for _, job := range w.jobs {
		if job.header.Root == root {
			return true
		}
	}
	return false
}

// waitState blocks until the given state is readable, if a queued block is still committing it.
func (w *chainWriter) waitState(root common.Hash) {
	w.waitJob(root, func(job *writeJob) chan struct{} { return job.committed })
}

// waitFlattened blocks until a queued block with the given state root is done flattening, and returns how long that took.
func (w *chainWriter) waitFlattened(root common.Hash) time.Duration {
	return w.waitJob(root, func(job *writeJob) chan struct{} { return job.flattened })
}

// waitJob blocks until the step of a queued block with the given root is done, and returns how long it waited.
func (w *chainWriter) waitJob(root common.Hash, step func(*writeJob) chan struct{}) time.Duration {
	// Any queued block with this root will do
	w.lock.RLock()
	var pending *writeJob
	for _, job := range w.jobs {
		if job.header.Root != root {
			continue
		}
		select {
		case <-step(job):
			w.lock.RUnlock()
			return 0
		default:
			pending = job
		}
	}
	w.lock.RUnlock()

	if pending == nil {
		return 0
	}
	start := time.Now()
	<-step(pending)
	return time.Since(start)
}

// runHook calls the test hook if one is set.
func (w *chainWriter) runHook(step string, block *types.Block) {
	if hook := w.hook.Load(); hook != nil {
		(*hook)(step, block)
	}
}

// encodedReceipts returns the receipts of the block in their storage encoding.
func (job *writeJob) encodedReceipts() rlp.RawValue {
	job.encodeOnce.Do(func() {
		storage := make([]*types.ReceiptForStorage, len(job.receipts))
		for i, receipt := range job.receipts {
			storage[i] = (*types.ReceiptForStorage)(receipt)
		}
		job.receiptsRLP, _ = rlp.EncodeToBytes(storage)
	})
	return job.receiptsRLP
}

// rawReceipts returns a copy of the receipts as the database would, without derived fields.
func (job *writeJob) rawReceipts() types.Receipts {
	// Decode a fresh copy, the receipts of the job are still being written
	var storage []*types.ReceiptForStorage
	if err := rlp.DecodeBytes(job.encodedReceipts(), &storage); err != nil {
		log.Error("Invalid receipt array RLP", "hash", job.block.Hash(), "err", err)
		return nil
	}
	receipts := make(types.Receipts, len(storage))
	for i, receipt := range storage {
		receipts[i] = (*types.Receipt)(receipt)
	}
	return receipts
}

// derivedReceipts returns a copy of the receipts with all the fields derived.
func (job *writeJob) derivedReceipts(config *params.ChainConfig) types.Receipts {
	receipts := job.rawReceipts()
	if receipts == nil {
		return nil
	}
	// Derive the fields from the block, like rawdb.ReadReceipts does
	var (
		block        = job.block
		blobGasPrice *big.Int
	)
	if block.ExcessBlobGas() != nil {
		blobGasPrice = eip4844.CalcBlobFee(config, job.header)
	}
	if err := receipts.DeriveFields(config, block.Hash(), block.NumberU64(), block.Time(), block.BaseFee(), blobGasPrice, block.Transactions()); err != nil {
		log.Error("Failed to derive block receipts fields", "hash", block.Hash(), "number", block.NumberU64(), "err", err)
		return nil
	}
	return receipts
}
