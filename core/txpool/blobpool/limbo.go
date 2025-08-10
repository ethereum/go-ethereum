// Copyright 2023 The go-ethereum Authors
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
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/billy"
)

// limboBlob is a wrapper around an opaque blobset that also contains the tx hash
// to which it belongs as well as the block number in which it was included for
// finality eviction.
type limboBlob struct {
	TxHash common.Hash        // Owner transaction's hash to support resurrecting reorged txs
	Block  uint64             // Block in which the blob transaction was included
	Tx     *types.Transaction `rlp:"omitempty"`
	// Optional blob transaction metadata.
	TxMeta *blobTxMeta `rlp:"omitempty"`
	id     uint64      // the billy id of limboBlob
}

// limbo is a light, indexed database to temporarily store recently included
// blobs until they are finalized. The purpose is to support small reorgs, which
// would require pulling back up old blobs (which aren't part of the chain).
//
// TODO(karalabe): Currently updating the inclusion block of a blob needs a full db rewrite. Can we do without?
type limbo struct {
	store billy.Database             // Persistent data store for limboed blobs
	index map[common.Hash]*limboBlob // Mappings from tx hashes to datastore ids
}

// newLimbo opens and indexes a set of limboed blob transactions.
func newLimbo(datadir string, maxBlobsPerTransaction int) (*limbo, error) {
	l := &limbo{
		index: make(map[common.Hash]*limboBlob),
	}
	// Index all limboed blobs on disk and delete anything unprocessable
	var fails []uint64
	index := func(id uint64, size uint32, data []byte) {
		if l.parseBlob(id, data) != nil {
			fails = append(fails, id)
		}
	}
	store, err := billy.Open(billy.Options{Path: datadir, Repair: true}, newSlotter(maxBlobsPerTransaction), index)
	if err != nil {
		return nil, err
	}
	l.store = store

	if len(fails) > 0 {
		log.Warn("Dropping invalidated limboed blobs", "ids", fails)
		for _, id := range fails {
			if err := l.store.Delete(id); err != nil {
				l.Close()
				return nil, err
			}
		}
	}
	return l, nil
}

// Close closes down the underlying persistent store.
func (l *limbo) Close() error {
	return l.store.Close()
}

// parseBlob is a callback method on limbo creation that gets called for each
// limboed blob on disk to create the in-memory metadata index.
func (l *limbo) parseBlob(id uint64, data []byte) error {
	item := new(limboBlob)
	if err := rlp.DecodeBytes(data, item); err != nil {
		// This path is impossible unless the disk data representation changes
		// across restarts. For that ever improbable case, recover gracefully
		// by ignoring this data entry.
		log.Error("Failed to decode blob limbo entry", "id", id, "err", err)
		return err
	}
	if _, ok := l.index[item.TxHash]; ok {
		// This path is impossible, unless due to a programming error a blob gets
		// inserted into the limbo which was already part of if. Recover gracefully
		// by ignoring this data entry.
		log.Error("Dropping duplicate blob limbo entry", "owner", item.TxHash, "id", id)
		return errors.New("duplicate blob")
	}
	// Delete tx and set id.
	item.id = id
	l.index[item.TxHash] = item

	return nil
}

// existsAndSet checks whether a blob transaction is already tracked by the limbo.
func (l *limbo) existsAndSet(meta *blobTxMeta) bool {
	if item := l.index[meta.hash]; item != nil {
		if item.Tx != nil {
			item.TxMeta, item.Tx = meta, nil
		}
		return true
	}
	return false
}

// tryRepair attempts to repair the limbo by re-encoding all transactions that are
// currently in the limbo, but not yet stored in the database. This is useful
// when the limbo is created from a previous state, and the transactions are not
// yet stored in the database. The method will re-encode all transactions and
// store them in the database, updating the in-memory indices at the same time.
func (l *limbo) tryRepair(store billy.Database) error {
	for _, item := range l.index {
		if item.Tx == nil {
			continue
		}
		tx := item.Tx
		// Transaction permitted into the pool from a nonce and cost perspective,
		// insert it into the database and update the indices
		blob, err := rlp.EncodeToBytes(tx)
		if err != nil {
			log.Error("Failed to encode transaction for storage", "hash", tx.Hash(), "err", err)
			return err
		}
		id, err := store.Put(blob)
		if err != nil {
			return err
		}
		meta := newBlobTxMeta(id, tx.Size(), store.Size(id), tx)
		l.existsAndSet(meta)
	}
	return nil
}

// finalize evicts all blobs belonging to a recently finalized block or older.
func (l *limbo) finalize(final *types.Header, fn func(id uint64, txHash common.Hash)) {
	// Just in case there's no final block yet (network not yet merged, weird
	// restart, sethead, etc), fail gracefully.
	if final == nil {
		log.Warn("Nil finalized block cannot evict old blobs")
		return
	}
	for _, item := range l.index {
		if item.Block > final.Number.Uint64() {
			continue
		}
		if err := l.store.Delete(item.id); err != nil {
			log.Error("Failed to drop finalized blob", "block", item.Block, "id", item.id, "err", err)
		}
		delete(l.index, item.TxHash)
		if fn != nil {
			meta := item.TxMeta
			fn(meta.id, meta.hash)
		}
	}
}

// push stores a new blob transaction into the limbo, waiting until finality for
// it to be automatically evicted.
func (l *limbo) push(meta *blobTxMeta, block uint64) error {
	// If the blobs are already tracked by the limbo, consider it a programming
	// error. There's not much to do against it, but be loud.
	if _, ok := l.index[meta.hash]; ok {
		log.Error("Limbo cannot push already tracked blobs", "tx", meta.hash)
		return errors.New("already tracked blob transaction")
	}
	if err := l.setAndIndex(meta, block); err != nil {
		log.Error("Failed to set and index limboed blobs", "tx", meta.hash, "err", err)
		return err
	}
	return nil
}

// pull retrieves a previously pushed set of blobs back from the limbo, removing
// it at the same time. This method should be used when a previously included blob
// transaction gets reorged out.
func (l *limbo) pull(txhash common.Hash) (*blobTxMeta, error) {
	// If the blobs are not tracked by the limbo, there's not much to do. This
	// can happen for example if a blob transaction is mined without pushing it
	// into the network first.
	item, ok := l.index[txhash]
	if !ok {
		log.Trace("Limbo cannot pull non-tracked blobs", "tx", txhash)
		return nil, errors.New("unseen blob transaction")
	}
	if err := l.store.Delete(item.id); err != nil {
		return nil, err
	}
	return item.TxMeta, nil
}

// update changes the block number under which a blob transaction is tracked. This
// method should be used when a reorg changes a transaction's inclusion block.
//
// The method may log errors for various unexpected scenarios but will not return
// any of it since there's no clear error case. Some errors may be due to coding
// issues, others caused by signers mining MEV stuff or swapping transactions. In
// all cases, the pool needs to continue operating.
func (l *limbo) update(txhash common.Hash, block uint64) {
	// If the blobs are not tracked by the limbo, there's not much to do. This
	// can happen for example if a blob transaction is mined without pushing it
	// into the network first.
	item, ok := l.index[txhash]
	if !ok {
		log.Trace("Limbo cannot update non-tracked blobs", "tx", txhash)
		return
	}
	// If there was no change in the blob's inclusion block, don't mess around
	// with heavy database operations.
	if item.Block == block {
		log.Trace("Blob transaction unchanged in limbo", "tx", txhash, "block", block)
		return
	}
	// Retrieve the old blobs from the data store and write them back with a new
	// block number. IF anything fails, there's not much to do, go on.
	if err := l.store.Delete(item.id); err != nil {
		log.Error("Failed to drop old limboed blobs", "tx", txhash, "err", err)
		return
	}
	if err := l.setAndIndex(item.TxMeta, block); err != nil {
		log.Error("Failed to set and index limboed blobs", "tx", txhash, "err", err)
		return
	}
	log.Trace("Blob transaction updated in limbo", "tx", txhash, "old-block", item.Block, "new-block", block)
}

// setAndIndex assembles a limbo blob database entry and stores it, also updating
// the in-memory indices.
func (l *limbo) setAndIndex(meta *blobTxMeta, block uint64) error {
	txhash := meta.hash
	item := &limboBlob{
		TxHash: txhash,
		Block:  block,
		TxMeta: meta,
		Tx:     nil, // The tx is stored in the blob database, not here.
	}
	data, err := rlp.EncodeToBytes(item)
	if err != nil {
		panic(err) // cannot happen runtime, dev error
	}
	id, err := l.store.Put(data)
	if err != nil {
		return err
	}
	// Delete tx and set id.
	item.id = id
	l.index[txhash] = item

	return nil
}
