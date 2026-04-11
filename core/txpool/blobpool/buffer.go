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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
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
	tx    *types.Transaction
	peer  string
	added time.Time
}

type cellEntry struct {
	deliveries map[string]*PeerDelivery
	custody    *types.CustodyBitmap
	added      time.Time
}

type BlobBuffer struct {
	txs   map[common.Hash]*txEntry
	cells map[common.Hash]*cellEntry

	addToPool func(*PooledBlobTx) error
	dropPeer  func(string)
}

func NewBlobBuffer(addToPool func(*PooledBlobTx) error, dropPeer func(string)) *BlobBuffer {
	return &BlobBuffer{
		txs:       make(map[common.Hash]*txEntry),
		cells:     make(map[common.Hash]*cellEntry),
		addToPool: addToPool,
		dropPeer:  dropPeer,
	}
}

// AddTx buffers a blob transaction (without blobs) from an ETH/72 peer.
// If cells are already buffered, verification and pool insertion are attempted.
func (b *BlobBuffer) AddTx(tx *types.Transaction, peer string) error {
	b.evict()

	hash := tx.Hash()
	sidecar := tx.BlobTxSidecar()
	if sidecar == nil {
		return fmt.Errorf("blob transaction without sidecar")
	}
	// vhash check
	if err := sidecar.ValidateBlobCommitmentHashes(tx.BlobHashes()); err != nil {
		log.Warn("Commitment hash mismatch, dropping peer", "peer", peer, "err", err)
		b.dropPeer(peer)
		return err
	}
	// proof count check
	if len(sidecar.Proofs) < len(sidecar.Commitments)*kzg4844.CellProofsPerBlob {
		b.dropPeer(peer)
		return fmt.Errorf("insufficient proofs in sidecar")
	}
	// todo: I also considered performing additional validation for the metrics of the
	// tx_fetcher. This could be used to avoid sending GetCells requests when the
	// nonce is too low or the transaction is underpriced. However, doing so would
	// require taking buffered transactions into account as well, and would require
	// allowing the buffer to be part of the fetcher’s scheduling logic.
	// Therefore, I will leave this as a TODO for now.

	if entry, ok := b.cells[hash]; ok {
		return b.add(hash, tx, entry)
	}
	b.txs[hash] = &txEntry{tx: tx, peer: peer, added: time.Now()}
	return nil
}

// AddCells buffers per-peer cell deliveries from the blob fetcher.
// If the transaction is already buffered, verification and pool insertion are attempted.
func (b *BlobBuffer) AddCells(hash common.Hash, deliveries map[string]*PeerDelivery, custody *types.CustodyBitmap) error {
	b.evict()
	b.cells[hash] = &cellEntry{
		deliveries: deliveries,
		custody:    custody,
		added:      time.Now(),
	}

	if txe, ok := b.txs[hash]; ok {
		return b.add(hash, txe.tx, b.cells[hash])
	}
	return nil
}

// add verifies cells per-peer, sorts them, and adds to the pool.
func (b *BlobBuffer) add(hash common.Hash, tx *types.Transaction, cells *cellEntry) error {
	sidecar := tx.BlobTxSidecar()

	// Per-peer cell verification
	if badPeers := b.verifyCells(cells, sidecar); len(badPeers) > 0 {
		b.dropPeers(badPeers)
		delete(b.cells, hash)
		delete(b.txs, hash)
		return fmt.Errorf("cell verification failed")
	}
	blobCount := len(tx.BlobHashes())
	sorted, custody := sortCells(cells, blobCount)

	cellSidecar := &types.BlobTxCellSidecar{
		Version:     sidecar.Version,
		Cells:       sorted,
		Commitments: sidecar.Commitments,
		Proofs:      sidecar.Proofs,
		Custody:     *custody,
	}
	pooledTx := &PooledBlobTx{
		Transaction:     tx.WithoutBlobTxSidecar(),
		Sidecar:         cellSidecar,
		Size:            tx.Size(),
		SizeWithoutBlob: tx.WithoutBlob().Size(),
	}
	err := b.addToPool(pooledTx)
	delete(b.cells, hash)
	delete(b.txs, hash)
	return err
}

func (b *BlobBuffer) HasTx(hash common.Hash) bool {
	_, ok := b.txs[hash]
	return ok
}

func (b *BlobBuffer) HasCells(hash common.Hash) bool {
	_, ok := b.cells[hash]
	return ok
}

func (b *BlobBuffer) dropPeers(peers []string) {
	if b.dropPeer == nil {
		return
	}
	for _, p := range peers {
		b.dropPeer(p)
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

// verifyCells verifies each peer's cells against the sidecar.
// Returns the list of peers whose cells failed verification.
func (b *BlobBuffer) verifyCells(entry *cellEntry, sidecar *types.BlobTxSidecar) []string {
	var badPeers []string
	for peer, delivery := range entry.deliveries {
		if err := verifyPeerCells(delivery, sidecar); err != nil {
			log.Debug("Cell verification failed", "peer", peer, "err", err)
			badPeers = append(badPeers, peer)
		}
	}
	return badPeers
}

// verifyPeerCells verifies a single peer's cells against the sidecar proofs.
// delivery.Cells is blob-major: [blob0_cell0..blob0_cellN, blob1_cell0..blob1_cellN, ...]
func verifyPeerCells(delivery *PeerDelivery, sidecar *types.BlobTxSidecar) error {
	cellsPerBlob := len(delivery.Indices)
	blobCount := len(delivery.Cells) / cellsPerBlob
	if blobCount == 0 || blobCount != len(sidecar.Commitments) {
		return fmt.Errorf("blob count mismatch: delivery %d, commitments %d", blobCount, len(sidecar.Commitments))
	}
	// Extract proofs corresponding to this peer's cell indices
	var proofs []kzg4844.Proof
	for blobIdx := 0; blobIdx < blobCount; blobIdx++ {
		for _, cellIdx := range delivery.Indices {
			proofIdx := blobIdx*kzg4844.CellProofsPerBlob + int(cellIdx)
			if proofIdx >= len(sidecar.Proofs) {
				return fmt.Errorf("proof index out of range: %d", proofIdx)
			}
			proofs = append(proofs, sidecar.Proofs[proofIdx])
		}
	}
	return kzg4844.VerifyCells(delivery.Cells, sidecar.Commitments, proofs, delivery.Indices)
}

// sortCells merges all per-peer deliveries into a single flat cell array
// sorted by custody index.
//
// e.g.
// peer A: cells = [blob0_cell5, blob0_cell3, blob1_cell5, blob1_cell3]
// peer B: cells = [blob0_cell1, blob0_cell7, blob1_cell1, blob1_cell7]
// -> [blob0_cell1, blob0_cell3, blob0_cell5, blob0_cell7, blob1_cell1, blob1_cell3, blob1_cell5, blob1_cell7]
func sortCells(entry *cellEntry, blobCount int) ([]kzg4844.Cell, *types.CustodyBitmap) {
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
	return res, &custody
}
