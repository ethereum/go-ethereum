// Copyright 2024 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/common"
)

type txMetadata struct {
	id   uint64 // the billy id of transction
	size uint64 // the RLP encoded size of transaction (blobs are included)
}

// lookup maps blob versioned hashes to transaction hashes that include them,
// transaction hashes to billy entries that include them, transaction hashes
// to the transaction size
type lookup struct {
	blobIndex map[common.Hash]map[common.Hash]struct{}
	txIndex   map[common.Hash]*txMetadata
}

// newLookup creates a new index for tracking blob to tx; and tx to billy mappings.
func newLookup() *lookup {
	return &lookup{
		blobIndex: make(map[common.Hash]map[common.Hash]struct{}),
		txIndex:   make(map[common.Hash]*txMetadata),
	}
}

// exists returns whether a transaction is already tracked or not.
func (l *lookup) exists(txhash common.Hash) bool {
	_, exists := l.txIndex[txhash]
	return exists
}

// storeidOfTx returns the datastore storage item id of a transaction.
func (l *lookup) storeidOfTx(txhash common.Hash) (uint64, bool) {
	meta, ok := l.txIndex[txhash]
	if !ok {
		return 0, false
	}
	return meta.id, true
}

// storeidOfBlob returns the datastore storage item id of a blob.
func (l *lookup) storeidOfBlob(vhash common.Hash) (uint64, bool) {
	// If the blob is unknown, return a miss
	txs, ok := l.blobIndex[vhash]
	if !ok {
		return 0, false
	}
	// If the blob is known, return any tx for it
	for tx := range txs {
		return l.storeidOfTx(tx)
	}
	return 0, false // Weird, don't choke
}

// sizeOfTx returns the RLP-encoded size of transaction
func (l *lookup) sizeOfTx(txhash common.Hash) (uint64, bool) {
	meta, ok := l.txIndex[txhash]
	if !ok {
		return 0, false
	}
	return meta.size, true
}

// track inserts a new set of mappings from blob versioned hashes to transaction
// hashes; and from transaction hashes to datastore storage item ids.
func (l *lookup) track(tx *blobTxMeta) {
	// Map all the blobs to the transaction hash
	for _, vhash := range tx.vhashes {
		if _, ok := l.blobIndex[vhash]; !ok {
			l.blobIndex[vhash] = make(map[common.Hash]struct{})
		}
		l.blobIndex[vhash][tx.hash] = struct{}{} // may be double mapped if a tx contains the same blob twice
	}
	// Map the transaction hash to the datastore id and RLP-encoded transaction size
	l.txIndex[tx.hash] = &txMetadata{
		id:   tx.id,
		size: tx.size,
	}
}

// untrack removes a set of mappings from blob versioned hashes to transaction
// hashes from the blob index.
func (l *lookup) untrack(tx *blobTxMeta) {
	// Unmap the transaction hash from the datastore id
	delete(l.txIndex, tx.hash)

	// Unmap all the blobs from the transaction hash
	for _, vhash := range tx.vhashes {
		delete(l.blobIndex[vhash], tx.hash) // may be double deleted if a tx contains the same blob twice
		if len(l.blobIndex[vhash]) == 0 {
			delete(l.blobIndex, vhash)
		}
	}
}
