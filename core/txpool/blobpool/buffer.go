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

package blobpool

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	blobBufferTxFirstCounter    = metrics.NewRegisteredCounter("blobpool/buffer/txfirst", nil)
	blobBufferCellsFirstCounter = metrics.NewRegisteredCounter("blobpool/buffer/cellsfirst", nil)
	blobBufferTotalTx           = metrics.NewRegisteredGauge("blobpool/buffer/txcount", nil)
	blobBufferTotalCells        = metrics.NewRegisteredGauge("blobpool/buffer/cellcount", nil)
	blobBufferDupCellsCounter   = metrics.NewRegisteredCounter("blobpool/buffer/dupcells", nil)
	blobBufferTotalTxBytes      = metrics.NewRegisteredGauge("blobpool/buffer/txbytes", nil)
	blobBufferTotalCellBytes    = metrics.NewRegisteredGauge("blobpool/buffer/cellbytes", nil)
	blobBufferOverflowCounter   = metrics.NewRegisteredCounter("blobpool/buffer/overflow", nil)
)

const (
	bufferLifetime = 2 * time.Minute

	// maxBufferedBytes caps the total bytes of transactions and cells held in
	// the buffer. Cells are the larger part: a blobless eth/72 transaction is
	// ~37KB at the maximum blob count, while its cells range from ~98KB (one
	// node's custody) to ~786KB (full payload).
	maxBufferedBytes = 128 * 1024 * 1024

	// maxBufferedPeerBytes caps the bytes a single peer can hold in the buffer,
	// so a peer whose deliveries never complete stalls on its own allowance
	// instead of evicting everyone else. It matches the blob fetcher's per-peer
	// request budget, so a peer serving cells as fast as requested fits in it.
	maxBufferedPeerBytes = 16 * 1024 * 1024
)

// errPeerBufferFull is returned when a peer has exhausted its buffer allowance.
// It wraps ErrOutOfCapacity so the fetcher can tell it apart from a bad delivery.
var errPeerBufferFull = fmt.Errorf("%w: peer blob buffer allowance exhausted", txpool.ErrOutOfCapacity)

// PeerDelivery holds cells delivered by a single peer, in blob-major order.
type PeerDelivery struct {
	Cells   []kzg4844.Cell
	Indices []uint64
}

type txEntry struct {
	tx *types.Transaction

	// Technically it is not required to store peer information to drop properly.
	// This is mainly for per peer size limit check.
	peer  string    // Peer Identifier
	added time.Time // Timestamp when the tx is added
	size  uint64    // Encoded size, as accounted against the buffer limits
}

type cellEntry struct {
	deliveries map[string]*PeerDelivery
	custody    types.CustodyBitmap
	added      time.Time
	size       uint64 // Total cell bytes, as accounted against the buffer limits
}

// cellsSize returns the bytes occupied by a peer's cell delivery.
func cellsSize(delivery *PeerDelivery) uint64 {
	const cellSize = uint64(len(kzg4844.Cell{}))
	return uint64(len(delivery.Cells))*cellSize + uint64(len(delivery.Indices))*8
}

type BlobBuffer struct {
	mu sync.Mutex

	txs   map[common.Hash]*txEntry
	cells map[common.Hash]*cellEntry

	txBytes      uint64            // Encoded size of the buffered transactions
	cellBytes    uint64            // Size of the buffered cells
	peerBytes    map[string]uint64 // Buffered bytes of either kind, per contributing peer
	maxBytes     uint64            // Cap on txBytes plus cellBytes, lowered by tests
	maxPeerBytes uint64            // Cap on any single peerBytes entry, lowered by tests

	completed      []*BlobTxForPool
	completedCount atomic.Int32
	cb             BlobBufferFunctions
}

type BlobBufferFunctions struct {
	ValidateTx func(*types.Transaction) error
	AddToPool  func(*BlobTxForPool) error
	DropPeer   func(string)
}

func NewBlobBuffer(cb BlobBufferFunctions) *BlobBuffer {
	return &BlobBuffer{
		txs:          make(map[common.Hash]*txEntry),
		cells:        make(map[common.Hash]*cellEntry),
		peerBytes:    make(map[string]uint64),
		maxBytes:     maxBufferedBytes,
		maxPeerBytes: maxBufferedPeerBytes,
		cb:           cb,
	}
}

// Flush adds all completed entries to the pool and returns the hashes
// and corresponding errors (nil on success) for each attempted insert.
func (b *BlobBuffer) Flush() ([]common.Hash, []error) {
	// Read the count first and return early if there is nothing to do.
	// Flush is called very frequently from the blob fetcher so this
	// optimization is warranted.
	if b.completedCount.Load() == 0 {
		return nil, nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	txs := make([]common.Hash, len(b.completed))
	errs := make([]error, len(b.completed))
	for i, ptx := range b.completed {
		txs[i] = ptx.Tx.Hash()
		errs[i] = b.cb.AddToPool(ptx)
	}
	b.completed = nil
	b.completedCount.Store(0)
	return txs, errs
}

// AddTx buffers a blob transaction (without blobs) from an ETH/72 peer.
// If cells are already buffered, verification and pool insertion are attempted.
func (b *BlobBuffer) AddTx(txs []*types.Transaction, peer string) []error {
	b.mu.Lock()
	defer b.mu.Unlock()
	defer b.updateMetrics()()

	// First remove any timed-out entries.
	b.evict()

	errs := make([]error, len(txs))
	for i, tx := range txs {
		hash := tx.Hash()
		sidecar := tx.BlobTxSidecar()
		if sidecar == nil {
			errs[i] = fmt.Errorf("blob transaction without sidecar")
			continue
		}
		// tx validation (basic w/o lock)
		// error will be handled by tx fetcher
		if err := b.cb.ValidateTx(tx); err != nil {
			errs[i] = err
			continue
		}
		if entry, ok := b.cells[hash]; ok {
			b.storeCompleted(hash, tx, entry)
			continue
		}
		// Refuse the transaction if the peer has exhausted its allowance.
		size := tx.Size()
		if b.peerBytes[peer]+size > b.maxPeerBytes {
			errs[i] = errPeerBufferFull
			continue
		}
		// Make room by evicting from the peers holding the most.
		for b.buffered()+size > b.maxBytes && b.evictOne() {
		}
		blobBufferTxFirstCounter.Inc(1)
		b.insertTx(hash, tx, peer)
	}
	return errs
}

// AddCells buffers per-peer cell deliveries from the blob fetcher.
// If the transaction is already buffered, verification and pool insertion are attempted.
func (b *BlobBuffer) AddCells(hash common.Hash, deliveries map[string]*PeerDelivery, custody types.CustodyBitmap) {
	b.mu.Lock()
	defer b.mu.Unlock()
	defer b.updateMetrics()()

	// First remove any timed-out entries.
	b.evict()

	entry := &cellEntry{
		deliveries: deliveries,
		custody:    custody,
		added:      time.Now(),
	}
	for _, delivery := range deliveries {
		entry.size += cellsSize(delivery)
	}

	// Check if this delivery completes the entry
	if txe, ok := b.txs[hash]; ok {
		b.storeCompleted(hash, txe.tx, entry)
		return
	}

	// Otherwise make room and buffer the cells. Unlike transactions, cells
	// are not refused over the peer allowance: they have already been fetched,
	// so turning them away would only waste the retrieval.
	for b.buffered()+entry.size > b.maxBytes && b.evictOne() {
	}
	b.insertCells(hash, entry)

	blobBufferCellsFirstCounter.Inc(1)
}

// storeCompleted verifies cells per-peer, sorts them, and schedules them for
// addition into the pool. The actual addition happens in Flush().
func (b *BlobBuffer) storeCompleted(hash common.Hash, tx *types.Transaction, cells *cellEntry) {
	sidecar := tx.BlobTxSidecar()

	// Per-peer cell verification
	if badPeers := b.verifyCells(cells, sidecar); len(badPeers) > 0 {
		b.dropPeers(badPeers)
		b.removeCells(hash)
		b.removeTx(hash)
		return
	}
	blobCount := len(tx.BlobHashes())

	sorted, custody, err := sortCells(cells, blobCount)
	if err != nil {
		log.Warn("Dropping blob tx with overlapping cell deliveries", "hash", hash, "err", err)
		blobBufferDupCellsCounter.Inc(1)
		b.removeCells(hash)
		b.removeTx(hash)
		return
	}
	cellSidecar := types.BlobTxCellSidecar{
		Version:     sidecar.Version,
		Commitments: sidecar.Commitments,
		Proofs:      sidecar.Proofs,
		Cells:       sorted,
		Custody:     custody,
	}
	pooledTx := &BlobTxForPool{
		Tx:          tx.WithoutBlobTxSidecar(),
		CellSidecar: &cellSidecar,
	}

	b.completed = append(b.completed, pooledTx)
	b.completedCount.Add(1)
	b.removeCells(hash)
	b.removeTx(hash)
}

func (b *BlobBuffer) HasTx(hash common.Hash) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, ok := b.txs[hash]
	return ok
}

func (b *BlobBuffer) HasCells(hash common.Hash) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	_, ok := b.cells[hash]
	return ok
}

// insertTx buffers a transaction and accounts for its size. Any previous entry
// for the same hash is removed first.
func (b *BlobBuffer) insertTx(hash common.Hash, tx *types.Transaction, peer string) {
	b.removeTx(hash)

	size := tx.Size()
	b.txs[hash] = &txEntry{tx: tx, peer: peer, added: time.Now(), size: size}
	b.txBytes += size
	b.peerBytes[peer] += size
}

// removeTx drops a buffered transaction and releases its accounted size.
func (b *BlobBuffer) removeTx(hash common.Hash) {
	entry, ok := b.txs[hash]
	if !ok {
		return
	}
	b.txBytes -= min(b.txBytes, entry.size)
	b.releasePeer(entry.peer, entry.size)
	delete(b.txs, hash)
}

// insertCells buffers a set of cell deliveries and accounts for their size,
// charging each peer for its own delivery. Any previous entry for the same
// hash is removed first.
func (b *BlobBuffer) insertCells(hash common.Hash, entry *cellEntry) {
	b.removeCells(hash)

	b.cells[hash] = entry
	b.cellBytes += entry.size
	for peer, delivery := range entry.deliveries {
		b.peerBytes[peer] += cellsSize(delivery)
	}
}

// removeCells drops buffered cells and releases their accounted size.
func (b *BlobBuffer) removeCells(hash common.Hash) {
	entry, ok := b.cells[hash]
	if !ok {
		return
	}
	b.cellBytes -= min(b.cellBytes, entry.size)
	for peer, delivery := range entry.deliveries {
		b.releasePeer(peer, cellsSize(delivery))
	}
	delete(b.cells, hash)
}

// releasePeer refunds a peer's allowance.
func (b *BlobBuffer) releasePeer(peer string, size uint64) {
	if b.peerBytes[peer] <= size {
		delete(b.peerBytes, peer)
		return
	}
	b.peerBytes[peer] -= size
}

// buffered returns the total accounted bytes of transactions and cells.
func (b *BlobBuffer) buffered() uint64 {
	return b.txBytes + b.cellBytes
}

// evictOne drops the oldest entry of the peer holding the most bytes and
// reports whether anything was dropped. Evicting from the largest holder
// rather than the globally oldest entry keeps a few flooding peers from
// pushing out everyone else. Transactions are evicted before cells, as
// cells are more expensive to retrieve again.
func (b *BlobBuffer) evictOne() bool {
	var (
		worst string
		held  uint64
	)
	for peer, size := range b.peerBytes {
		if size > held {
			worst, held = peer, size
		}
	}
	var (
		oldest common.Hash
		added  time.Time
	)
	for hash, entry := range b.txs {
		if entry.peer != worst {
			continue
		}
		if added.IsZero() || entry.added.Before(added) {
			oldest, added = hash, entry.added
		}
	}
	if !added.IsZero() {
		blobBufferOverflowCounter.Inc(1)
		b.removeTx(oldest)
		return true
	}
	// No transactions left for the peer, fall back to its cells.
	for hash, entry := range b.cells {
		if _, ok := entry.deliveries[worst]; !ok {
			continue
		}
		if added.IsZero() || entry.added.Before(added) {
			oldest, added = hash, entry.added
		}
	}
	if added.IsZero() {
		return false
	}
	blobBufferOverflowCounter.Inc(1)
	b.removeCells(oldest)
	return true
}

func (b *BlobBuffer) dropPeers(peers []string) {
	if b.cb.DropPeer == nil {
		return
	}
	for _, p := range peers {
		b.cb.DropPeer(p)
	}
}

func (b *BlobBuffer) evict() {
	now := time.Now()
	for hash, entry := range b.txs {
		if now.Sub(entry.added) > bufferLifetime {
			b.removeTx(hash)
		}
	}
	for hash, entry := range b.cells {
		if now.Sub(entry.added) > bufferLifetime {
			b.removeCells(hash)
		}
	}
}

// updateMetrics updates the metrics gauges.
// This should be called at the start of any operation that changes the buffer
// content. The returned function is to be called at the end of the operation,
// usually with defer.
func (b *BlobBuffer) updateMetrics() func() {
	preTxCount := len(b.txs)
	preCellsCount := len(b.cells)
	preTxBytes := b.txBytes
	preCellBytes := b.cellBytes
	return func() {
		if len(b.txs) != preTxCount {
			blobBufferTotalTx.Update(int64(len(b.txs)))
		}
		if b.txBytes != preTxBytes {
			blobBufferTotalTxBytes.Update(int64(b.txBytes))
		}
		if b.cellBytes != preCellBytes {
			blobBufferTotalCellBytes.Update(int64(b.cellBytes))
		}
		if len(b.cells) != preCellsCount {
			blobBufferTotalCells.Update(int64(len(b.cells)))
		}
	}
}

// verifyCells verifies each peer's cells against the sidecar by treating each
// per-peer delivery as a mini BlobTxCellSidecar and reusing txpool.ValidateCells.
// Returns the list of peers whose cells failed verification.
func (b *BlobBuffer) verifyCells(entry *cellEntry, sidecar *types.BlobTxSidecar) []string {
	var badPeers []string
	for peer, delivery := range entry.deliveries {
		perPeer := &types.BlobTxCellSidecar{
			Version:     sidecar.Version,
			Cells:       delivery.Cells,
			Commitments: sidecar.Commitments,
			Proofs:      sidecar.Proofs,
			Custody:     types.NewCustodyBitmap(delivery.Indices),
		}
		if err := txpool.ValidateCells(perPeer); err != nil {
			log.Debug("Cell verification failed", "peer", peer, "err", err)
			badPeers = append(badPeers, peer)
		}
	}
	return badPeers
}

// sortCells merges all per-peer deliveries into a single flat cell array
// sorted by custody index.
//
// e.g.
// peer A: cells = [blob0_cell5, blob0_cell3, blob1_cell5, blob1_cell3]
// peer B: cells = [blob0_cell1, blob0_cell7, blob1_cell1, blob1_cell7]
// -> [blob0_cell1, blob0_cell3, blob0_cell5, blob0_cell7, blob1_cell1, blob1_cell3, blob1_cell5, blob1_cell7]
func sortCells(entry *cellEntry, blobCount int) ([]kzg4844.Cell, types.CustodyBitmap, error) {
	// indices per delivery
	var indices []uint64

	// 1. compose per blob cells
	blob := make([][]kzg4844.Cell, blobCount)
	for _, d := range entry.deliveries {
		n := len(d.Indices)
		indices = append(indices, d.Indices...)
		for b := range blobCount {
			blob[b] = append(blob[b], d.Cells[b*n:(b+1)*n]...)
		}
	}
	// Defensive operation, rejecting the deliveries with overlapping
	// custody index.
	custody := types.NewCustodyBitmap(indices)
	if custody.OneCount() != len(indices) {
		return nil, types.CustodyBitmap{}, errors.New("duplicate cell index across deliveries")
	}

	// 2. sort
	perm := make([]int, len(indices))
	for i := range perm {
		perm[i] = i
	}
	// perm represents the position of cells in sorted array
	slices.SortFunc(perm, func(a, b int) int {
		return cmp.Compare(indices[a], indices[b])
	})
	// reorder cells
	var res []kzg4844.Cell
	for b := range blobCount {
		for _, p := range perm {
			res = append(res, blob[b][p])
		}
	}
	return res, custody, nil
}
