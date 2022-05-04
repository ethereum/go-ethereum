// Copyright 2022 The go-ethereum Authors
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

package snapshot

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

const (
	snapAccount = "account" // Identifier of account snapshot generation
	snapStorage = "storage" // Identifier of storage snapshot generation
)

// generatorStats is a collection of statistics gathered by the snapshot generator
// for logging purposes.
type generatorStats struct {
	origin   uint64             // Origin prefix where generation started
	start    time.Time          // Timestamp when generation started
	accounts uint64             // Number of accounts indexed(generated or recovered)
	slots    uint64             // Number of storage slots indexed(generated or recovered)
	dangling uint64             // Number of dangling storage slots
	storage  common.StorageSize // Total account and storage slot size(generation or recovery)
}

// Log creates an contextual log with the given message and the context pulled
// from the internally maintained statistics.
func (gs *generatorStats) Log(msg string, root common.Hash, marker []byte) {
	var ctx []interface{}
	if root != (common.Hash{}) {
		ctx = append(ctx, []interface{}{"root", root}...)
	}
	// Figure out whether we're after or within an account
	switch len(marker) {
	case common.HashLength:
		ctx = append(ctx, []interface{}{"at", common.BytesToHash(marker)}...)
	case 2 * common.HashLength:
		ctx = append(ctx, []interface{}{
			"in", common.BytesToHash(marker[:common.HashLength]),
			"at", common.BytesToHash(marker[common.HashLength:]),
		}...)
	}
	// Add the usual measurements
	ctx = append(ctx, []interface{}{
		"accounts", gs.accounts,
		"slots", gs.slots,
		"storage", gs.storage,
		"dangling", gs.dangling,
		"elapsed", common.PrettyDuration(time.Since(gs.start)),
	}...)
	// Calculate the estimated indexing time based on current stats
	if len(marker) > 0 {
		if done := binary.BigEndian.Uint64(marker[:8]) - gs.origin; done > 0 {
			left := math.MaxUint64 - binary.BigEndian.Uint64(marker[:8])

			speed := done/uint64(time.Since(gs.start)/time.Millisecond+1) + 1 // +1s to avoid division by zero
			ctx = append(ctx, []interface{}{
				"eta", common.PrettyDuration(time.Duration(left/speed) * time.Millisecond),
			}...)
		}
	}
	log.Info(msg, ctx...)
}

// generatorContext carries a few global values to be shared by all generation functions.
type generatorContext struct {
	stats   *generatorStats     // Generation statistic collection
	db      ethdb.KeyValueStore // Key-value store containing the snapshot data
	account ethdb.Iterator      // Iterator of account snapshot data
	storage ethdb.Iterator      // Iterator of storage snapshot data
	batch   ethdb.Batch         // Database batch for writing batch data atomically
	logged  time.Time           // The timestamp when last generation progress was displayed
}

// newGeneratorContext initializes the context for generation.
func newGeneratorContext(stats *generatorStats, db ethdb.KeyValueStore, accMarker []byte, storageMarker []byte) *generatorContext {
	ctx := &generatorContext{
		stats:  stats,
		db:     db,
		batch:  db.NewBatch(),
		logged: time.Now(),
	}
	ctx.openIterator(snapAccount, accMarker)
	ctx.openIterator(snapStorage, storageMarker)
	return ctx
}

// openIterator constructs global account and storage snapshot iterators
// at the interrupted position. These iterators should be reopened from time
// to time to avoid blocking leveldb compaction for a long time.
func (ctx *generatorContext) openIterator(kind string, start []byte) {
	if kind == snapAccount {
		iter := ctx.db.NewIterator(rawdb.SnapshotAccountPrefix, start)
		ctx.account = rawdb.NewKeyLengthIterator(iter, 1+common.HashLength)
		return
	}
	iter := ctx.db.NewIterator(rawdb.SnapshotStoragePrefix, start)
	ctx.storage = rawdb.NewKeyLengthIterator(iter, 1+2*common.HashLength)
}

// reopenIterators releases the held two global database iterators and
// reopens them in the interruption position. It's aim for not blocking
// leveldb compaction.
func (ctx *generatorContext) reopenIterators() {
	for i, iter := range []ethdb.Iterator{ctx.account, ctx.storage} {
		var (
			key = iter.Key()
			cur = key[1:]
		)
		kind := snapAccount
		if i == 1 {
			kind = snapStorage
		}
		iter.Release()
		ctx.openIterator(kind, cur)
	}
}

// close releases all the held resources.
func (ctx *generatorContext) close() {
	ctx.account.Release()
	ctx.storage.Release()
}

// iterator returns the corresponding iterator specified by the kind.
func (ctx *generatorContext) iterator(kind string) ethdb.Iterator {
	if kind == snapAccount {
		return ctx.account
	}
	return ctx.storage
}

// removeStorageBefore, iterates and deletes all storage snapshots starting
// from the current iterator position until the specified account. When the
// iterator touches the storage located in the given account range, or the
// storage is larger than the given account range, it stops and moves back
// the iterator a step.
func (ctx *generatorContext) removeStorageBefore(account common.Hash) {
	var (
		count uint64
		iter  = ctx.storage
	)
	for iter.Next() {
		key := iter.Key()
		if bytes.Compare(key[1:1+common.HashLength], account.Bytes()) >= 0 {
			iter.Prev()
			return
		}
		ctx.batch.Delete(key)
		count += 1
	}
	ctx.stats.dangling += count
}

// removeStorageAt iterates and deletes all storage snapshots which are located
// in the specified account range. When the iterator touches the storage which
// is larger than the given account range, it stops and moves back the iterator
// a step. An error will be returned if the initial position of iterator is not
// in the given account range.
func (ctx *generatorContext) removeStorageAt(account common.Hash) error {
	var (
		count uint64
		iter  = ctx.iterator(snapStorage)
	)
	for iter.Next() {
		key := iter.Key()
		cmp := bytes.Compare(key[1:1+common.HashLength], account.Bytes())
		if cmp < 0 {
			return errors.New("invalid iterator position")
		}
		if cmp > 0 {
			iter.Prev()
			break
		}
		ctx.batch.Delete(key)
		count += 1
	}
	ctx.stats.dangling += count
	return nil
}

// removeStorageLeft starting from the current iterator position, iterate and
// delete all storage snapshots left.
func (ctx *generatorContext) removeStorageLeft() {
	var (
		count uint64
		iter  = ctx.iterator(snapStorage)
	)
	for iter.Next() {
		ctx.batch.Delete(iter.Key())
		count += 1
	}
	ctx.stats.dangling += count
}
