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
	"encoding/binary"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb/internal"
	"golang.org/x/sync/errgroup"
)

// ErrCancelled is returned when GenerateTrie is aborted via its cancel
// channel before completing.
var ErrCancelled = internal.ErrCancelled

// GenerateStats reports per-run counters from GenerateTrie. Scanned is
// the number of accounts walked, Updated is how many had a stale Root
// field that was rewritten to match the recomputed storage root, and
// Deleted is the number of dangling storage slots removed.
type GenerateStats struct {
	Scanned int64
	Updated int64
	Deleted int64
}

// numPartitions is the number of slices the account hash space is divided
// into by GenerateTrie.
const numPartitions = 16

// Each partition covers 1/16 of the account hash space. We track progress
// by interpreting the top 8 bytes of an account hash as a uint64, so each
// partition spans 2^64 / 16 = 2^60. partitionFinished is stored in a
// partition's position when it completes.
const (
	partitionRangeSize = uint64(1) << 60
	partitionFinished  = ^uint64(0)
)

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

// flushIfFull writes and resets the batch once it grows past IdealBatchSize,
// then reopens the iterators.
func (r *rangeIterators) flushIfFull(batch ethdb.Batch, where string) error {
	if batch.ValueSize() <= ethdb.IdealBatchSize {
		return nil
	}
	if err := batch.Write(); err != nil {
		return fmt.Errorf("flush batch (%s): %w", where, err)
	}
	batch.Reset()
	r.reopen()
	return nil
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
	// pebble's Key() slice is invalidated by Release. Copy first so the new
	// iterator's lower bound isn't seeded from freed memory.
	next := common.CopyBytes(old.Key())
	old.Release()
	return openFlatIterator(db, prefix, next[len(prefix):], suffixLen)
}

// generatePartition walks accounts whose first nibble equals `partition`,
// reconciling each account's Root with its flat storage and building
// both per-account storage subtries and the partition's slice of the
// account trie. Returns the partition's stripped subtree root blob, or
// nil if the partition had no accounts at all.
func generatePartition(ctx context.Context, cancel <-chan struct{}, db ethdb.Database, scheme string, partition byte, rangeStart, rangeEnd common.Hash, scanned, updated, deleted *atomic.Int64, pos *atomic.Uint64) ([]byte, error) {
	iters := openRangeIterators(db, rangeStart)
	defer iters.release()

	batch := db.NewBatchWithSize(ethdb.IdealBatchSize)

	// Account-trie builder for this partition. It is fed account keys with
	// their leading nibble stripped and emits nodes at their absolute path
	// (prefixed with the partition nibble), so they line up with the full
	// trie without any post-hoc surgery.
	//
	// The subtree root is the only node emitted at path [partition]; we both
	// persist it (so the top-level branch can reference it) and capture its
	// bytes for assembleRoot, which needs them to either reference it or,
	// in the single-partition case, fold the leading nibble back in.
	var root []byte
	acctTrie := trie.NewPartialStackTrie(partition, func(path []byte, hash common.Hash, blob []byte) {
		if len(path) == 1 {
			root = common.CopyBytes(blob)
		}
		rawdb.WriteTrieNode(batch, common.Hash{}, path, hash, blob, scheme)
	})

	// Iterate through all the accounts.
	for iters.acct.Next() {
		select {
		case <-cancel:
			return nil, ErrCancelled
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		key := iters.acct.Key()
		var accountHash common.Hash
		copy(accountHash[:], key[len(rawdb.SnapshotAccountPrefix):])
		if bytes.Compare(accountHash[:], rangeEnd[:]) > 0 {
			break
		}
		scanned.Add(1)
		pos.Store(binary.BigEndian.Uint64(accountHash[:8]))

		// Decode the account object
		account, err := types.FullAccount(iters.acct.Value())
		if err != nil {
			return nil, fmt.Errorf("decode account %x: %w", accountHash, err)
		}

		// Build the account's storage trie from the flat storage snapshot.
		// StackTrie's onTrieNode callback persists nodes as they finalize.
		storageTrie := trie.NewStackTrie(func(path []byte, hash common.Hash, blob []byte) {
			rawdb.WriteTrieNode(batch, accountHash, path, hash, blob, scheme)
		})

		// Compute the storage root by consuming matching slots from the
		// shared storage iterator. The inner loop terminates on Hold()
		// (slot belongs to a later account) or exhaustion.
		lastDanglingAccount := make([]byte, common.HashLength)
		for iters.stor.Next() {
			// Re-check cancel.
			select {
			case <-cancel:
				return nil, ErrCancelled
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			var (
				sk             = iters.stor.Key()
				storageAccount = sk[len(rawdb.SnapshotStoragePrefix) : len(rawdb.SnapshotStoragePrefix)+common.HashLength]
				cmp            = bytes.Compare(storageAccount, accountHash[:])
			)
			// The slot belongs to an account whose hash is smaller than the one
			// currently being processed. This should be theoretically impossible,
			// so log it loudly and delete the dangling entry from the flat state.
			if cmp < 0 {
				if !bytes.Equal(lastDanglingAccount, storageAccount) {
					copy(lastDanglingAccount, storageAccount)
					log.Error("Unexpected storage entries for dangling account", "expected", accountHash, "got", common.BytesToHash(storageAccount))
				}
				deleted.Add(1)
				slotHash := sk[len(rawdb.SnapshotStoragePrefix)+common.HashLength:]
				rawdb.DeleteStorageSnapshot(batch, common.BytesToHash(storageAccount), common.BytesToHash(slotHash))
				if err := iters.flushIfFull(batch, "dangling"); err != nil {
					return nil, err
				}
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
			if err := storageTrie.Update(slotHash, iters.stor.Value()); err != nil {
				return nil, fmt.Errorf("storage stack trie update for %x: %w", accountHash, err)
			}
			if err := iters.flushIfFull(batch, "storage"); err != nil {
				return nil, err
			}
		}
		if err := iters.stor.Error(); err != nil {
			return nil, fmt.Errorf("storage iterator: %w", err)
		}
		computed := storageTrie.Hash()

		// If account.Root was stale, rewrite the flat-state entry. Then feed
		// the account, now with the correct Root, into this partition's
		// account trie.
		if computed != account.Root {
			account.Root = computed
			rawdb.WriteAccountSnapshot(batch, accountHash, types.SlimAccountRLP(*account))
			updated.Add(1)
		}
		fullAccount, err := rlp.EncodeToBytes(account)
		if err != nil {
			return nil, fmt.Errorf("encode account %x: %w", accountHash, err)
		}
		if err := acctTrie.Update(accountHash[:], fullAccount); err != nil {
			return nil, fmt.Errorf("account stack trie update for %x: %w", accountHash, err)
		}
		if err := iters.flushIfFull(batch, "account"); err != nil {
			return nil, err
		}
	}
	if err := iters.acct.Error(); err != nil {
		return nil, fmt.Errorf("account iterator: %w", err)
	}

	// The account iterator is exhausted (or has advanced past this partition),
	// but the storage iterator may still hold slots whose account hash falls
	// within this partition's range. Those slots belong to no existing account
	// and should be cleared.
	lastDanglingTail := make([]byte, common.HashLength)
	for iters.stor.Next() {
		select {
		case <-cancel:
			return nil, ErrCancelled
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		sk := iters.stor.Key()
		acct := sk[len(rawdb.SnapshotStoragePrefix) : len(rawdb.SnapshotStoragePrefix)+common.HashLength]
		if bytes.Compare(acct, rangeEnd[:]) > 0 {
			break
		}
		if !bytes.Equal(lastDanglingTail, acct) {
			copy(lastDanglingTail, acct)
			log.Error("Unexpected storage entries for dangling account", "addrhash", common.BytesToHash(acct))
		}
		deleted.Add(1)
		slotHash := sk[len(rawdb.SnapshotStoragePrefix)+common.HashLength:]
		rawdb.DeleteStorageSnapshot(batch, common.BytesToHash(acct), common.BytesToHash(slotHash))
		if err := iters.flushIfFull(batch, "dangling tail"); err != nil {
			return nil, err
		}
	}
	if err := iters.stor.Error(); err != nil {
		return nil, fmt.Errorf("storage iterator (dangling): %w", err)
	}

	// Finalize the partition's account trie. For a non-empty partition this
	// emits the subtree root at path [partition], populating rootBlob. An empty
	// partition never emits any node and leaves rootBlob at nil.
	acctTrie.Hash()

	if err := batch.Write(); err != nil {
		return nil, fmt.Errorf("final partition batch write: %w", err)
	}
	return root, nil
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
	for i := range total {
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
// data in the database. The account hash space is partitioned into 16
// slices aligned with the first-nibble branching of the MPT root. Each
// partition is processed by its own goroutine, which walks its slice,
// reconciles stale account.Root fields with flat storage, builds the
// per-account storage tries and the partition's slice of the account
// trie. Once every partition has produced its subtree root, the top-level
// branch is assembled and its hash verified against the expected root.
//
// Resume: on entry, any partition that has a "done" marker from a
// previous run is skipped. Its subtree blob is read from the marker
// and handed to assembleRoot directly. On a mid-run crash, only the
// in-flight partition(s) are redone.
func GenerateTrie(db ethdb.Database, scheme string, root common.Hash, cancel <-chan struct{}) (GenerateStats, error) {
	var (
		start        = time.Now()
		scanned      atomic.Int64
		updated      atomic.Int64
		deleted      atomic.Int64
		progress     [numPartitions]atomic.Uint64
		progressDone = make(chan struct{})

		// partitionBlobs[i] holds the root node for partition i, or nil if
		// the partition is empty.
		partitionBlobs [numPartitions][]byte
	)
	go tickProgress(progressDone, start, &scanned, &updated, &progress)
	defer close(progressDone)

	// For each partition, either skip (prior done marker found) or run
	// it. Prior runs can leave the partition's raw root blob in the done
	// marker. We recover it here so assembleRoot has everything it needs.
	var (
		ranges  = hashRanges(numPartitions)
		eg, ctx = errgroup.WithContext(context.Background())
	)
	for i, r := range ranges {
		partition := byte(i)
		rangeStart, rangeEnd := r[0], r[1]
		if blob, ok := rawdb.ReadGenerateTriePartitionDone(db, partition); ok {
			partitionBlobs[partition] = blob
			progress[partition].Store(partitionFinished)
			continue
		}
		eg.Go(func() error {
			start := time.Now()
			blob, err := generatePartition(ctx, cancel, db, scheme, partition, rangeStart, rangeEnd, &scanned, &updated, &deleted, &progress[partition])
			if err != nil {
				return err
			}
			log.Info("Partition done", "partition", partition, "elapsed", common.PrettyDuration(time.Since(start)))

			progress[partition].Store(partitionFinished)
			partitionBlobs[partition] = blob

			// Record completion only after the partition's batch has
			// flushed inside generatePartition, so this marker appears
			// on disk only when every write the partition did is durable.
			rawdb.WriteGenerateTriePartitionDone(db, partition, blob)
			return nil
		})
	}

	// Wait until all the partitions are fully generated
	if err := eg.Wait(); err != nil {
		return GenerateStats{}, err
	}

	// Assemble the top-level root from the partition blobs, verify it
	// matches the expected root, and clear all partition markers on
	// success.
	got, err := assembleRoot(db, scheme, partitionBlobs)
	if err != nil {
		return GenerateStats{}, fmt.Errorf("assemble root: %w", err)
	}
	if got != root {
		return GenerateStats{}, fmt.Errorf("state root mismatch: got %x, want %x", got, root)
	}

	// Clear the partition progress marker, ending the generation process.
	batch := db.NewBatch()
	for i := range numPartitions {
		rawdb.DeleteGenerateTriePartitionDone(batch, byte(i))
	}
	if err := batch.Write(); err != nil {
		return GenerateStats{}, fmt.Errorf("clear partition markers: %w", err)
	}
	log.Info("Generated state trie", "scanned", scanned.Load(), "updated", updated.Load(), "dangling-slots", deleted.Load(), "elapsed", common.PrettyDuration(time.Since(start)))
	return GenerateStats{
		Scanned: scanned.Load(),
		Updated: updated.Load(),
		Deleted: deleted.Load(),
	}, nil
}

// assembleRoot computes the canonical state root from the 16 partition subtree
// root blobs and persists the top-level node. Each partition was built with its
// leading nibble stripped, so its root blob is already the exact node the parent
// branch mounts in that slot, and the partition has already written it (and all
// its descendants) at their absolute paths. What's left depends on how many
// partitions ended up populated:
//
//   - 0 populated: the state is empty, the root is types.EmptyRootHash and
//     nothing is written.
//
//   - 1 populated: there is no top-level branch; the canonical root is that
//     lone partition's subtree with its leading nibble folded back in (see
//     trie.MountPartitionRoot). The new root node is written. If the fold
//     orphaned the old subtree root the partition left at [n], that node is
//     also deleted.
//
//   - 2+ populated: the canonical root is a 17-slot branch mounting each
//     partition's subtree root by hash. The subtree roots are already on disk,
//     so we only encode, hash, and persist the branch itself.
func assembleRoot(db ethdb.Database, scheme string, partitionBlobs [numPartitions][]byte) (common.Hash, error) {
	var (
		populated int
		partition int // last populated index, read only when populated == 1
		children  [17][]byte
	)

	// Loop through all partitions and count how many are populated, while
	// pre-filling the branch children array for the common 2+ case.
	for i := range numPartitions {
		if partitionBlobs[i] != nil {
			populated++
			partition = i
			children[i] = crypto.Keccak256(partitionBlobs[i])
		}
	}

	// No populated partitions: the state is empty.
	if populated == 0 {
		return types.EmptyRootHash, nil
	}

	// One populated partition: no top-level branch, so fold its leading nibble
	// back into the subtree root.
	if populated == 1 {
		rootHash, rootBlob, isOrphaned, err := trie.MountPartitionRoot(partitionBlobs[partition], byte(partition))
		if err != nil {
			return common.Hash{}, fmt.Errorf("mount partition %d: %w", partition, err)
		}
		batch := db.NewBatch()
		rawdb.WriteTrieNode(batch, common.Hash{}, nil, rootHash, rootBlob, scheme)
		if isOrphaned {
			// The folded root at nil does not reference [partition], so the copy
			// generatePartition wrote there is now unreferenced. Delete it so the
			// on-disk node set matches the canonical trie.
			staleHash := crypto.Keccak256Hash(partitionBlobs[partition])
			rawdb.DeleteTrieNode(batch, common.Hash{}, []byte{byte(partition)}, staleHash, scheme)
		}
		return rootHash, batch.Write()
	}

	// populated >= 2: mount each partition's subtree root (already persisted at
	// path [i]) into a 17-slot branch by hash, using the children array filled
	// above. Those hash references are valid because account-trie subtree roots
	// are always >= 32 bytes.
	rootBlob, rootHash, err := trie.AssembleBranch(children)
	if err != nil {
		return common.Hash{}, err
	}
	rawdb.WriteTrieNode(db, common.Hash{}, nil, rootHash, rootBlob, scheme)
	return rootHash, nil
}

// tickProgress logs an aggregate progress line every 30 seconds until done
// is closed. Cheap: a handful of atomic loads and one log line per tick.
func tickProgress(done <-chan struct{}, start time.Time, scanned, updated *atomic.Int64, progress *[numPartitions]atomic.Uint64) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			elapsed := time.Since(start)
			fraction := progressFraction(progress)
			eta := "n/a"
			if fraction > 0.005 {
				eta = common.PrettyDuration(time.Duration(float64(elapsed) * (1.0/fraction - 1.0))).String()
			}
			log.Info("Generating trie",
				"progress", fmt.Sprintf("%.1f%%", fraction*100), "eta", eta,
				"scanned", scanned.Load(), "updated", updated.Load(),
				"elapsed", common.PrettyDuration(elapsed),
				"acct/s", uint64(float64(scanned.Load())/elapsed.Seconds()))
		}
	}
}

// progressFraction averages each partition's iterator position (as a fraction
// of its hash range) into an overall completion estimate in [0, 1]. Keccak
// hashes are uniform, so keyspace position is a good proxy for work done.
func progressFraction(progress *[numPartitions]atomic.Uint64) float64 {
	var total float64
	for i := range numPartitions {
		p := progress[i].Load()
		switch {
		case p == partitionFinished:
			total += 1.0
		case p == 0:
			// not started yet
		default:
			rangeStart := uint64(i) * partitionRangeSize
			if p > rangeStart {
				rel := p - rangeStart
				if rel > partitionRangeSize {
					rel = partitionRangeSize
				}
				total += float64(rel) / float64(partitionRangeSize)
			}
		}
	}
	return total / float64(numPartitions)
}
