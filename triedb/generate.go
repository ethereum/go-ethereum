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

package triedb

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb/internal"
	"golang.org/x/sync/errgroup"
)

// ErrCancelled is returned when GenerateTrie is aborted via its cancel
// channel before completing.
var ErrCancelled = internal.ErrCancelled

// updateStorageRootsProgressPrefix is the key prefix used to persist a
// per-partition progress marker during updateStorageRoots.
var updateStorageRootsProgressPrefix = []byte("triedb-updsr-")

func updateStorageRootsProgressKey(partition int) []byte {
	return append(updateStorageRootsProgressPrefix, byte(partition))
}

// kvAccountIterator wraps an ethdb.Iterator to iterate over account snapshot
// entries in the database, implementing internal.AccountIterator.
type kvAccountIterator struct {
	it   ethdb.Iterator
	hash common.Hash
}

func newKVAccountIterator(db ethdb.Iteratee) *kvAccountIterator {
	it := rawdb.NewKeyLengthIterator(
		db.NewIterator(rawdb.SnapshotAccountPrefix, nil),
		len(rawdb.SnapshotAccountPrefix)+common.HashLength,
	)
	return &kvAccountIterator{it: it}
}

func (it *kvAccountIterator) Next() bool {
	if !it.it.Next() {
		return false
	}
	key := it.it.Key()
	copy(it.hash[:], key[len(rawdb.SnapshotAccountPrefix):])
	return true
}

func (it *kvAccountIterator) Hash() common.Hash { return it.hash }
func (it *kvAccountIterator) Account() []byte   { return it.it.Value() }
func (it *kvAccountIterator) Error() error      { return it.it.Error() }
func (it *kvAccountIterator) Release()          { it.it.Release() }

// kvStorageIterator wraps an ethdb.Iterator to iterate over storage snapshot
// entries for a specific account, implementing internal.StorageIterator.
type kvStorageIterator struct {
	it   ethdb.Iterator
	hash common.Hash
}

func newKVStorageIterator(db ethdb.Iteratee, accountHash common.Hash) *kvStorageIterator {
	it := rawdb.IterateStorageSnapshots(db, accountHash)
	return &kvStorageIterator{it: it}
}

func (it *kvStorageIterator) Next() bool {
	if !it.it.Next() {
		return false
	}
	key := it.it.Key()
	copy(it.hash[:], key[len(rawdb.SnapshotStoragePrefix)+common.HashLength:])
	return true
}

func (it *kvStorageIterator) Hash() common.Hash { return it.hash }
func (it *kvStorageIterator) Slot() []byte      { return it.it.Value() }
func (it *kvStorageIterator) Error() error      { return it.it.Error() }
func (it *kvStorageIterator) Release()          { it.it.Release() }

// rangeIterators bundles the per-partition account and storage iterators.
type rangeIterators struct {
	db   ethdb.Database
	acct *internal.HoldableIterator
	stor *internal.HoldableIterator
}

func openRangeIterators(db ethdb.Database, start common.Hash) *rangeIterators {
	return &rangeIterators{
		db:   db,
		acct: openFlatIterator(db, rawdb.SnapshotAccountPrefix, start[:], common.HashLength),
		stor: openFlatIterator(db, rawdb.SnapshotStoragePrefix, start[:], 2*common.HashLength),
	}
}

// reopen releases both iterators and reopens them at their current
// positions. Invoked after each batch flush so pebble compactions aren't
// blocked by long-lived iterator snapshots. Follows the same pattern as
// triedb/pathdb/context.go.
func (r *rangeIterators) reopen() {
	r.acct = reopenFlatIterator(r.db, r.acct, rawdb.SnapshotAccountPrefix, common.HashLength)
	r.stor = reopenFlatIterator(r.db, r.stor, rawdb.SnapshotStoragePrefix, 2*common.HashLength)
}

func (r *rangeIterators) release() {
	r.acct.Release()
	r.stor.Release()
}

// openFlatIterator opens a length-filtered HoldableIterator over a snapshot
// prefix, seeked to the given start key (relative to the prefix).
func openFlatIterator(db ethdb.Database, prefix, start []byte, suffixLen int) *internal.HoldableIterator {
	it := db.NewIterator(prefix, start)
	return internal.NewHoldableIterator(rawdb.NewKeyLengthIterator(it, len(prefix)+suffixLen))
}

// reopenFlatIterator releases `old` and returns a new HoldableIterator
// positioned at the same key, or an empty iterator if `old` is exhausted.
func reopenFlatIterator(db ethdb.Database, old *internal.HoldableIterator, prefix []byte, suffixLen int) *internal.HoldableIterator {
	if !old.Next() {
		old.Release()
		return internal.NewHoldableIterator(memorydb.New().NewIterator(nil, nil))
	}
	next := old.Key()
	old.Release()
	return openFlatIterator(db, prefix, next[len(prefix):], suffixLen)
}

// updateStorageRoots walks flat-state accounts and updates each account's
// Root to match the storage root computed from its flat storage slots.
func updateStorageRoots(db ethdb.Database, cancel <-chan struct{}) error {
	start := time.Now()
	threads := runtime.NumCPU()
	var (
		batchMu sync.Mutex
		batch   = db.NewBatch()
		scanned atomic.Int64
		updated atomic.Int64
	)
	eg, ctx := errgroup.WithContext(context.Background())

	// Spawn one worker per hash-space partition. Each walker handles its
	// [rangeStart, rangeEnd] slice independently.  errgroup cancels ctx
	// on the first error so peers exit.
	for i, r := range hashRanges(threads) {
		partition := i
		rangeStart, rangeEnd := r[0], r[1]
		eg.Go(func() error {
			return updateStorageRootsInRange(ctx, cancel, db, partition, rangeStart, rangeEnd, &batchMu, batch, &scanned, &updated)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	// Clean up the progress markers now that every partition has finished
	// successfully.
	for i := 0; i < threads; i++ {
		batch.Delete(updateStorageRootsProgressKey(i))
	}
	if err := batch.Write(); err != nil {
		return fmt.Errorf("final batch write: %w", err)
	}
	log.Info("Updated stale storage roots", "scanned", scanned.Load(), "updated", updated.Load(), "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// updateStorageRootsInRange walks accounts whose hashes fall inside
// [rangeStart, rangeEnd] and fixes each account's Root to match its flat
// storage.
func updateStorageRootsInRange(ctx context.Context, cancel <-chan struct{}, db ethdb.Database, partition int, rangeStart, rangeEnd common.Hash, batchMu *sync.Mutex, batch ethdb.Batch, scanned, updated *atomic.Int64) error {
	iters := openRangeIterators(db, rangeStart)
	defer iters.release()

	// Iterate through all the accounts.
	for iters.acct.Next() {
		select {
		case <-cancel:
			return ErrCancelled
		case <-ctx.Done():
			return nil
		default:
		}
		key := iters.acct.Key()
		var accountHash common.Hash
		copy(accountHash[:], key[len(rawdb.SnapshotAccountPrefix):])
		if bytes.Compare(accountHash[:], rangeEnd[:]) > 0 {
			return nil
		}
		scanned.Add(1)
		account, err := types.FullAccount(iters.acct.Value())
		if err != nil {
			return fmt.Errorf("decode account %x: %w", accountHash, err)
		}

		// Compute the storage root by consuming matching slots from the
		// shared storage iterator. The inner loop terminates on Hold()
		// (slot belongs to a later account) or exhaustion.
		t := trie.NewStackTrie(nil)
		for iters.stor.Next() {
			sk := iters.stor.Key()
			storAcc := sk[len(rawdb.SnapshotStoragePrefix) : len(rawdb.SnapshotStoragePrefix)+common.HashLength]
			cmp := bytes.Compare(storAcc, accountHash[:])

			// The slot belongs to an account whose hash is before the one we're
			// processing. This only happens if an account was deleted but its flat
			// storage wasn't cleaned up. Skip the orphaned slot and advance.
			if cmp < 0 {
				continue
			}

			// The slot belongs to a later account. We're done with the current
			// account's slots, but we don't want to lose this slot. The slot might
			// belong to the next iteration of the account for-loop (or a later one).
			// Hold() the iterator so the next Next() call will re-serve this same
			// entry instead of advancing past it.
			if cmp > 0 {
				iters.stor.Hold()
				break
			}

			// The slot belongs to this account so we add it to the StackTrie.
			slotHash := sk[len(rawdb.SnapshotStoragePrefix)+common.HashLength:]
			if err := t.Update(slotHash, iters.stor.Value()); err != nil {
				return fmt.Errorf("stack trie update for %x: %w", accountHash, err)
			}
		}
		if err := iters.stor.Error(); err != nil {
			return fmt.Errorf("storage iterator: %w", err)
		}
		computed := t.Hash()

		// Update the account, progress marker, and (possibly) the batch.
		var (
			flushed  bool
			flushErr error
		)
		batchMu.Lock()
		if computed != account.Root {
			account.Root = computed
			rawdb.WriteAccountSnapshot(batch, accountHash, types.SlimAccountRLP(*account))
			updated.Add(1)
		}
		batch.Put(updateStorageRootsProgressKey(partition), accountHash[:])
		if batch.ValueSize() > ethdb.IdealBatchSize {
			flushErr = batch.Write()
			if flushErr == nil {
				batch.Reset()
				flushed = true
			}
		}
		batchMu.Unlock()
		if flushErr != nil {
			return fmt.Errorf("flush batch: %w", flushErr)
		}
		if flushed {
			iters.reopen()
		}
	}
	return iters.acct.Error()
}

// hashRanges returns hash pairs [start, end] that evenly partition the
// 256-bit hash space. The last partition absorbs the remainder so rounding
// doesn't leave hashes uncovered.
func hashRanges(total int) [][2]common.Hash {
	step := new(big.Int).Sub(
		new(big.Int).Div(
			new(big.Int).Exp(common.Big2, common.Big256, nil),
			big.NewInt(int64(total)),
		),
		common.Big1,
	)
	ranges := make([][2]common.Hash, total)
	var next common.Hash
	for i := 0; i < total; i++ {
		last := common.BigToHash(new(big.Int).Add(next.Big(), step))
		if i == total-1 {
			last = common.MaxHash
		}
		ranges[i] = [2]common.Hash{next, last}
		next = common.BigToHash(new(big.Int).Add(last.Big(), common.Big1))
	}
	return ranges
}

// GenerateTrie rebuilds all tries (storage + account) from flat snapshot
// data in the database. It first brings every account's Root into
// agreement with its flat storage, then builds tries using StackTrie with
// streaming node writes, and verifies that the computed state root matches
// the expected root.
func GenerateTrie(db ethdb.Database, scheme string, root common.Hash, cancel <-chan struct{}) error {
	if err := updateStorageRoots(db, cancel); err != nil {
		return err
	}
	acctIt := newKVAccountIterator(db)
	defer acctIt.Release()
	got, err := internal.GenerateTrieRoot(db, scheme, acctIt, common.Hash{}, internal.StackTrieGenerate, func(dst ethdb.KeyValueWriter, accountHash, codeHash common.Hash, stat *internal.GenerateStats) (common.Hash, error) {
		storageIt := newKVStorageIterator(db, accountHash)
		defer storageIt.Release()

		hash, err := internal.GenerateTrieRoot(dst, scheme, storageIt, accountHash, internal.StackTrieGenerate, nil, stat, false, cancel)
		if err != nil {
			return common.Hash{}, err
		}
		return hash, nil
	}, internal.NewGenerateStats(), true, cancel)
	if err != nil {
		return err
	}
	if got != root {
		return fmt.Errorf("state root mismatch: got %x, want %x", got, root)
	}
	return nil
}
