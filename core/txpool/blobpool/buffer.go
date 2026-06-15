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
)

const (
	bufferLifetime = 2 * time.Minute
)

// PeerDelivery holds cells delivered by a single peer, in blob-major order.
type PeerDelivery struct {
	Cells   []kzg4844.Cell
	Indices []uint64
}

type txEntry struct {
	tx *types.Transaction
	// Technically it is not required to store peer information to drop properly.
	// This is mainly for per peer size limit check.
	peer  string
	added time.Time
}

type cellEntry struct {
	deliveries map[string]*PeerDelivery
	custody    types.CustodyBitmap
	added      time.Time
}

type BlobBuffer struct {
	mu sync.Mutex

	txs   map[common.Hash]*txEntry
	cells map[common.Hash]*cellEntry

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
		txs:   make(map[common.Hash]*txEntry),
		cells: make(map[common.Hash]*cellEntry),
		cb:    cb,
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
		blobBufferTxFirstCounter.Inc(1)
		b.txs[hash] = &txEntry{tx: tx, peer: peer, added: time.Now()}
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

	b.cells[hash] = &cellEntry{
		deliveries: deliveries,
		custody:    custody,
		added:      time.Now(),
	}
	if txe, ok := b.txs[hash]; ok {
		b.storeCompleted(hash, txe.tx, b.cells[hash])
	}
	blobBufferCellsFirstCounter.Inc(1)
}

// storeCompleted verifies cells per-peer, sorts them, and schedules them for
// addition into the pool. The actual addition happens in Flush().
func (b *BlobBuffer) storeCompleted(hash common.Hash, tx *types.Transaction, cells *cellEntry) {
	sidecar := tx.BlobTxSidecar()

	// Per-peer cell verification
	if badPeers := b.verifyCells(cells, sidecar); len(badPeers) > 0 {
		b.dropPeers(badPeers)
		delete(b.cells, hash)
		delete(b.txs, hash)
	}
	blobCount := len(tx.BlobHashes())
	sorted, custody := sortCells(cells, blobCount)

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
	delete(b.cells, hash)
	delete(b.txs, hash)
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
			delete(b.txs, hash)
		}
	}
	for hash, entry := range b.cells {
		if now.Sub(entry.added) > bufferLifetime {
			delete(b.cells, hash)
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
	return func() {
		if len(b.txs) != preTxCount {
			blobBufferTotalTx.Update(int64(len(b.txs)))
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
func sortCells(entry *cellEntry, blobCount int) ([]kzg4844.Cell, types.CustodyBitmap) {
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

	custody := types.NewCustodyBitmap(indices)
	return res, custody
}
