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

// numPartitions is the number of slices the account hash space is divided
// into by GenerateTrie.
const numPartitions = 16

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

// generatePartition walks accounts whose first nibble equals `partition`,
// reconciling each account's Root with its flat storage and building
// both per-account storage subtries and the partition's slice of the
// account trie. Returns the raw (unstripped) partition root blob, or
// nil if the partition had no accounts at all.
func generatePartition(ctx context.Context, cancel <-chan struct{}, db ethdb.Database, scheme string, partition byte, rangeStart, rangeEnd common.Hash, scanned, updated *atomic.Int64) ([]byte, error) {
	iters := openRangeIterators(db, rangeStart)
	defer iters.release()
	batch := db.NewBatch()

	// Account-trie StackTrie for this partition. Persist every node except
	// the root. assembleRoot() may need to strip the leading-nibble extension
	// off the root, so we capture its bytes and return them instead.
	var rootBlob []byte
	acctTrie := trie.NewStackTrie(func(path []byte, hash common.Hash, blob []byte) {
		if len(path) == 0 {
			rootBlob = common.CopyBytes(blob)
			return
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
		account, err := types.FullAccount(iters.acct.Value())
		if err != nil {
			return nil, fmt.Errorf("decode account %x: %w", accountHash, err)
		}

		// Build the account's storage trie from the flat storage snapshot.
		// StackTrie's onTrieNode callback persists nodes as they finalize.
		stoTrie := trie.NewStackTrie(func(path []byte, hash common.Hash, blob []byte) {
			rawdb.WriteTrieNode(batch, accountHash, path, hash, blob, scheme)
		})

		// Compute the storage root by consuming matching slots from the
		// shared storage iterator. The inner loop terminates on Hold()
		// (slot belongs to a later account) or exhaustion.
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
			if err := stoTrie.Update(slotHash, iters.stor.Value()); err != nil {
				return nil, fmt.Errorf("storage stack trie update for %x: %w", accountHash, err)
			}
		}
		if err := iters.stor.Error(); err != nil {
			return nil, fmt.Errorf("storage iterator: %w", err)
		}
		computed := stoTrie.Hash()

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

		// Progress marker keeps the batch growing on a predictable
		// rate. The size check drives flush + iterator reopen so
		// pebble compactions aren't blocked by long-lived iterators.
		rawdb.WriteGenerateTrieProgress(batch, partition, accountHash)
		if batch.ValueSize() > ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return nil, fmt.Errorf("flush batch: %w", err)
			}
			batch.Reset()
			iters.reopen()
		}
	}
	if err := iters.acct.Error(); err != nil {
		return nil, fmt.Errorf("account iterator: %w", err)
	}

	// Finalize the partition's account trie. For a non-empty partition
	// this triggers the path=[] onTrieNode callback, populating
	// rootBlob. An empty partition never emits any node and leaves
	// rootBlob at nil.
	acctTrie.Hash()

	// Clear the progress marker since it's no longer needed once the
	// partition's batch is flushed.
	rawdb.DeleteGenerateTrieProgress(batch, partition)
	if err := batch.Write(); err != nil {
		return nil, fmt.Errorf("final partition batch write: %w", err)
	}
	return rootBlob, nil
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
func GenerateTrie(db ethdb.Database, scheme string, root common.Hash, cancel <-chan struct{}) error {
	start := time.Now()
	var (
		scanned atomic.Int64
		updated atomic.Int64
	)

	// partitionBlobs[i] holds the raw (unstripped) StackTrie root node
	// blob for partition i, or nil if the partition is empty.
	var partitionBlobs [numPartitions][]byte

	// For each partition, either skip (prior done marker found) or run
	// it. Prior runs can leave the partition's raw root blob in the done
	// marker. We recover it here so assembleRoot has everything it needs.
	ranges := hashRanges(numPartitions)
	eg, ctx := errgroup.WithContext(context.Background())
	for i, r := range ranges {
		partition := byte(i)
		rangeStart, rangeEnd := r[0], r[1]
		if blob, ok := rawdb.ReadGenerateTriePartitionDone(db, partition); ok {
			partitionBlobs[partition] = blob
			continue
		}
		eg.Go(func() error {
			blob, err := generatePartition(ctx, cancel, db, scheme, partition, rangeStart, rangeEnd, &scanned, &updated)
			if err != nil {
				return err
			}
			partitionBlobs[partition] = blob
			// Record completion only after the partition's batch has
			// flushed inside generatePartition, so this marker appears
			// on disk only when every write the partition did is durable.
			rawdb.WriteGenerateTriePartitionDone(db, partition, blob)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	// Assemble the top-level root from the partition blobs, verify it
	// matches the expected root, and clear all partition markers on
	// success.
	got, err := assembleRoot(db, scheme, partitionBlobs)
	if err != nil {
		return fmt.Errorf("assemble root: %w", err)
	}
	if got != root {
		return fmt.Errorf("state root mismatch: got %x, want %x", got, root)
	}
	batch := db.NewBatch()
	for i := range numPartitions {
		rawdb.DeleteGenerateTriePartitionDone(batch, byte(i))
	}
	if err := batch.Write(); err != nil {
		return fmt.Errorf("clear partition markers: %w", err)
	}
	log.Info("Generated state trie", "scanned", scanned.Load(), "updated", updated.Load(), "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// assembleRoot computes the canonical state root from the 16 raw
// partition root blobs and persists any newly-constructed nodes.
// The decision about whether to strip each partition's leading-nibble
// extension depends on how many partitions ended up populated:
//
//   - 0 populated: the state is empty, the root is types.EmptyRootHash,
//     nothing is written.
//   - 1 populated: the state's canonical root is that partition's
//     subtree directly, with its leading nibble still included. We
//     need to persist the partition's raw root node since generatePartition
//     deliberately didn't write it at path=[].
//   - 2+ populated: strip each partition so the leading-nibble extension
//     isn't double-traversed by the top-level branch, then pack the 16
//     stripped references into a fullNode, encode, hash, and persist that
//     branch as the state root.
func assembleRoot(db ethdb.Database, scheme string, partitionBlobs [numPartitions][]byte) (common.Hash, error) {
	var (
		populated int
		onlySlot  int
	)
	for i := range numPartitions {
		if partitionBlobs[i] != nil {
			populated++
			onlySlot = i
		}
	}
	if populated == 0 {
		return types.EmptyRootHash, nil
	}
	batch := db.NewBatch()
	if populated == 1 {
		// Persist the partition's raw root at path=[] (path scheme) or
		// at its hash (hash scheme). That node is the state root.
		blob := partitionBlobs[onlySlot]
		rootHash := crypto.Keccak256Hash(blob)
		rawdb.WriteTrieNode(batch, common.Hash{}, nil, rootHash, blob, scheme)
		if err := batch.Write(); err != nil {
			return common.Hash{}, fmt.Errorf("write single-partition root: %w", err)
		}
		return rootHash, nil
	}

	// populated >= 2: strip each partition and assemble a 17-slot branch.
	var children [17][]byte
	for i := range numPartitions {
		if partitionBlobs[i] == nil {
			continue
		}
		stripped, strippedBlob, err := trie.StripPartitionRoot(partitionBlobs[i], byte(i))
		if err != nil {
			return common.Hash{}, fmt.Errorf("strip partition %d: %w", i, err)
		}

		// Remember that strip returns nil for the common case 1.
		if strippedBlob != nil {
			// Strip constructed a new node that is alonger extension or leaf
			// partition (case 2/3). Persist it at path=[i] so path-scheme readers
			// traversing slot i of the top branch can find it.
			rawdb.WriteTrieNode(batch, common.Hash{}, []byte{byte(i)}, stripped, strippedBlob, scheme)
		}
		children[i] = stripped.Bytes()
	}
	rootBlob, rootHash, err := trie.AssembleBranch(children)
	if err != nil {
		return common.Hash{}, err
	}
	rawdb.WriteTrieNode(batch, common.Hash{}, nil, rootHash, rootBlob, scheme)
	if err := batch.Write(); err != nil {
		return common.Hash{}, fmt.Errorf("write root branch: %w", err)
	}
	return rootHash, nil
}
