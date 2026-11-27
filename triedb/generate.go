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
	"github.com/ethereum/go-ethereum/ethdb"
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

// The top 8 bytes of the account hash span 2^60 positions per partition.
// Progress measures the distance from the partition's starting bound.
const (
	partitionRangeSize = uint64(1) << 60
	partitionFinished  = ^uint64(0)
)

// genCounters bundles the progress counters threaded through a GenerateTrie run.
type genCounters struct {
	accounts       atomic.Int64 // accounts scanned
	slots          atomic.Int64 // storage slots scanned
	accountUpdated atomic.Int64 // accounts whose stale storage Root was rewritten
	storageDeleted atomic.Int64 // dangling storage slots removed

	accountTrieNodes atomic.Int64 // generated account trie nodes
	accountTrieBytes atomic.Int64 // generated account trie bytes
	storageTrieNodes atomic.Int64 // generated storage trie nodes
	storageTrieBytes atomic.Int64 // generated storage trie bytes

	progress [numPartitions]atomic.Uint64 // distance scanned within each partition
}

type trieGenerator struct {
	db               ethdb.Database
	scheme           string
	ingester         ethdb.Ingester
	newIterator      iteratorFactory
	counters         genCounters
	availableWorkers chan struct{}
}

func newTrieGenerator(db ethdb.Database, scheme string) *trieGenerator {
	// Each account partition starts with one worker. A storage split can
	// borrow additional workers only from partitions that have finished.
	g := &trieGenerator{
		db:               db,
		scheme:           scheme,
		newIterator:      db.NewIterator,
		availableWorkers: make(chan struct{}, numPartitions),
	}
	// Switch to fast mode if the reverse iteration and database
	// ingestion are supported.
	if scheme == rawdb.PathScheme {
		store := rawdb.KeyValueStore(db)
		ingester, _ := store.(ethdb.Ingester)
		reverse, _ := store.(ethdb.ReverseIteratee)
		if ingester != nil && reverse != nil {
			g.ingester = ingester
			g.newIterator = reverse.NewReverseIterator
		}
	}
	return g
}

func (g *trieGenerator) newPartialStackTrie(nibble byte, onNode trie.OnTrieNode) *trie.PartialStackTrie {
	if g.ingester != nil {
		return trie.NewReversePartialStackTrie(nibble, onNode)
	}
	return trie.NewPartialStackTrie(nibble, onNode)
}

// generatePartition walks accounts whose first nibble equals `partition`,
// reconciling each account's Root with its flat storage and building
// both per-account storage subtries and the partition's slice of the
// account trie. Returns the partition's stripped subtree root blob, or
// nil if the partition had no accounts at all.
func (g *trieGenerator) generatePartition(ctx context.Context, cancel <-chan struct{}, partition byte, rangeStart, rangeEnd common.Hash) ([]byte, error) {
	counter := &g.counters
	reverse := g.ingester != nil

	start, end := rangeStart, rangeEnd
	if reverse {
		start, end = end, start
	}
	iters := newRangeIterators(g.newIterator, reverse, start)
	defer iters.release()

	batches := newPartitionBatches(g.db, g.ingester)
	defer batches.close()

	// Account-trie builder for this partition. It is fed account keys with
	// their leading nibble stripped and emits nodes at their absolute path
	// (prefixed with the partition nibble), so they line up with the full
	// trie without any post-hoc surgery.
	//
	// Hold the subtree root until assembly determines whether it is referenced
	// by hash, embedded or folded into a short node.
	var root []byte
	acctTrie := g.newPartialStackTrie(partition, func(path []byte, hash common.Hash, blob []byte) {
		if len(path) == 1 {
			root = common.CopyBytes(blob)
			return
		}
		g.writeAccountNode(batches.account, path, hash, blob)
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
		if iters.compare(accountHash[:], end[:]) > 0 {
			break
		}
		counter.accounts.Add(1)
		position := binary.BigEndian.Uint64(accountHash[:8]) % partitionRangeSize
		if reverse {
			position = partitionRangeSize - 1 - position
		}
		counter.progress[partition].Store(position)

		// Decode the account object
		account, err := types.FullAccount(iters.acct.Value())
		if err != nil {
			return nil, fmt.Errorf("decode account %x: %w", accountHash, err)
		}
		computed, err := g.generateAccountStorage(ctx, cancel, iters, batches, accountHash)
		if err != nil {
			return nil, err
		}
		// If account.Root was stale, rewrite the flat-state entry. Then feed
		// the account, now with the correct Root, into this partition's
		// account trie.
		if computed != account.Root {
			account.Root = computed
			rawdb.WriteAccountSnapshot(batches.flat, accountHash, types.SlimAccountRLP(*account))
			counter.accountUpdated.Add(1)
			if err := iters.flushIfFull(batches.flat, "account snapshot"); err != nil {
				return nil, err
			}
		}
		fullAccount, err := rlp.EncodeToBytes(account)
		if err != nil {
			return nil, fmt.Errorf("encode account %x: %w", accountHash, err)
		}
		if err := acctTrie.Update(accountHash[:], fullAccount); err != nil {
			return nil, fmt.Errorf("account stack trie update for %x: %w", accountHash, err)
		}
		if err := iters.flushIfFull(batches.account, "account"); err != nil {
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
	var lastDanglingTail []byte
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
		if iters.compare(acct, end[:]) > 0 {
			break
		}
		if !bytes.Equal(lastDanglingTail, acct) {
			lastDanglingTail = append(lastDanglingTail[:0], acct...)
			log.Error("Unexpected storage entries for dangling account", "addrhash", common.BytesToHash(acct))
		}
		counter.storageDeleted.Add(1)
		slotHash := sk[len(rawdb.SnapshotStoragePrefix)+common.HashLength:]
		rawdb.DeleteStorageSnapshot(batches.flat, common.BytesToHash(acct), common.BytesToHash(slotHash))
		if err := iters.flushIfFull(batches.flat, "dangling tail"); err != nil {
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

	if err := batches.flush(); err != nil {
		return nil, fmt.Errorf("final partition batch write: %w", err)
	}
	return root, nil
}

// generateAccountStorage builds one storage trie and leaves the scans positioned
// to continue with the next account.
func (g *trieGenerator) generateAccountStorage(ctx context.Context, cancel <-chan struct{}, iters *rangeIterators, batches *partitionBatches, accountHash common.Hash) (common.Hash, error) {
	var (
		storageTrie         *trie.StackTrie
		lastDanglingAccount []byte
		counter             = &g.counters
	)
	for iters.stor.Next() {
		// Re-check cancel.
		select {
		case <-cancel:
			return common.Hash{}, ErrCancelled
		case <-ctx.Done():
			return common.Hash{}, ctx.Err()
		default:
		}
		var (
			sk             = iters.stor.Key()
			storageAccount = sk[len(rawdb.SnapshotStoragePrefix) : len(rawdb.SnapshotStoragePrefix)+common.HashLength]
			cmp            = iters.compare(storageAccount, accountHash[:])
		)
		// The slot's owner precedes the current account in scan order.
		if cmp < 0 {
			if !bytes.Equal(lastDanglingAccount, storageAccount) {
				lastDanglingAccount = append(lastDanglingAccount[:0], storageAccount...)
				log.Error("Unexpected storage entries for dangling account", "expected", accountHash, "got", common.BytesToHash(storageAccount))
			}
			counter.storageDeleted.Add(1)
			slotHash := sk[len(rawdb.SnapshotStoragePrefix)+common.HashLength:]
			rawdb.DeleteStorageSnapshot(batches.flat, common.BytesToHash(storageAccount), common.BytesToHash(slotHash))

			if err := iters.flushIfFull(batches.flat, "dangling"); err != nil {
				return common.Hash{}, err
			}
			continue
		}

		// Keep the first slot of the next owner for the following scan.
		if cmp > 0 {
			iters.stor.Hold()
			break
		}

		// Generate the storage trie
		slotHash := sk[len(rawdb.SnapshotStoragePrefix)+common.HashLength:]
		if storageTrie == nil {
			split, err := g.probeStorage(ctx, cancel, accountHash, slotHash)
			if err != nil {
				return common.Hash{}, fmt.Errorf("probe storage for %x: %w", accountHash, err)
			}
			if split {
				return g.generateSplitStorage(ctx, cancel, iters, batches, accountHash)
			}
			onNode := func(path []byte, hash common.Hash, blob []byte) {
				g.writeStorageNode(batches.storage, accountHash, path, hash, blob)
			}
			if g.ingester != nil {
				storageTrie = trie.NewReverseStackTrie(onNode)
			} else {
				storageTrie = trie.NewStackTrie(onNode)
			}
		}
		if err := storageTrie.Update(slotHash, iters.stor.Value()); err != nil {
			return common.Hash{}, fmt.Errorf("storage stack trie update for %x: %w", accountHash, err)
		}
		counter.slots.Add(1)
		if err := iters.flushIfFull(batches.storage, "storage"); err != nil {
			return common.Hash{}, err
		}
	}
	if err := iters.stor.Error(); err != nil {
		return common.Hash{}, fmt.Errorf("storage iterator: %w", err)
	}
	if storageTrie == nil {
		return types.EmptyRootHash, nil
	}
	return storageTrie.Hash(), nil
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

// GenerateTrie builds all tries (storage + account) from flat snapshot
// data in the database. The account hash space is partitioned into 16
// slices aligned with the first-nibble branching of the MPT root. Each
// partition is processed by its own goroutine. Large storage tries are split
// into disjoint ranges that use available worker capacity. Once all account
// partitions have produced their subtree roots, generation assembles and
// verifies the state root.
//
// Path-scheme databases that support reverse iteration and SST ingestion use
// descending scans to generate sorted, disjoint trie-node batches.
//
// The caller must exclude other writers to the flat state and trie nodes
// during generation. An interrupted run may leave partial trie nodes, so the
// caller resets the path trie keyspace before rebuilding from scratch.
func GenerateTrie(db ethdb.Database, scheme string, root common.Hash, cancel <-chan struct{}) (GenerateStats, error) {
	return GenerateTrieWithProgress(db, scheme, root, cancel, nil)
}

// GenerateTrieWithProgress is GenerateTrie with live progress reporting.
func GenerateTrieWithProgress(db ethdb.Database, scheme string, root common.Hash, cancel <-chan struct{}, prog *atomic.Uint64) (GenerateStats, error) {
	var (
		start        = time.Now()
		gen          = newTrieGenerator(db, scheme)
		counter      = &gen.counters
		progressDone = make(chan struct{})

		// partitionBlobs[i] holds the root node for partition i, or nil if
		// the partition is empty.
		partitionBlobs [numPartitions][]byte
	)
	if gen.ingester != nil {
		// Persist the flat state and any earlier trie range deletions
		// before installing the first durable SST.
		if err := db.SyncKeyValue(); err != nil {
			return GenerateStats{}, fmt.Errorf("sync state before trie generation: %w", err)
		}
		log.Info("Using SST ingestion for trie generation")
	}
	go tickProgress(progressDone, start, counter, prog)
	defer close(progressDone)

	// Run every partition concurrently, each producing the subtree root
	// blob that assembleRoot needs.
	var (
		ranges  = hashRanges(numPartitions)
		eg, ctx = errgroup.WithContext(context.Background())
	)

	for i, r := range ranges {
		partition := byte(i)
		rangeStart, rangeEnd := r[0], r[1]

		eg.Go(func() error {
			defer func() { gen.availableWorkers <- struct{}{} }()

			start := time.Now()
			blob, err := gen.generatePartition(ctx, cancel, partition, rangeStart, rangeEnd)
			if err != nil {
				return err
			}
			log.Info("Partition done", "partition", partition, "elapsed", common.PrettyDuration(time.Since(start)))

			counter.progress[partition].Store(partitionFinished)
			partitionBlobs[partition] = blob
			return nil
		})
	}

	// Wait until all the partitions are fully generated
	if err := eg.Wait(); err != nil {
		return GenerateStats{}, err
	}
	// Assemble the top-level root from the partition blobs and verify it
	// matches the expected root.
	if err := gen.assembleRoot(&partitionBlobs, root); err != nil {
		return GenerateStats{}, fmt.Errorf("assemble root: %w", err)
	}
	if gen.ingester != nil {
		// The SSTs are durable once installed. Persist the ordinary root
		// nodes and flat repairs before the caller marks generation complete.
		if err := db.SyncKeyValue(); err != nil {
			return GenerateStats{}, fmt.Errorf("sync generated trie: %w", err)
		}
	}
	if prog != nil {
		prog.Store(100)
	}
	log.Info("Generated state trie",
		"accounts", counter.accounts.Load(), "slots", counter.slots.Load(),
		"account-nodes", counter.accountTrieNodes.Load(), "storage-nodes", counter.storageTrieNodes.Load(),
		"account-nodebytes", common.StorageSize(counter.accountTrieBytes.Load()), "storage-nodebytes", common.StorageSize(counter.storageTrieBytes.Load()),
		"updated-accounts", counter.accountUpdated.Load(), "dangling-slots", counter.storageDeleted.Load(),
		"elapsed", common.PrettyDuration(time.Since(start)))

	return GenerateStats{
		Scanned: counter.accounts.Load(),
		Updated: counter.accountUpdated.Load(),
		Deleted: counter.storageDeleted.Load(),
	}, nil
}

func (g *trieGenerator) assembleRoot(partitionBlobs *[numPartitions][]byte, want common.Hash) error {
	batch := g.db.NewBatch()
	defer batch.Close()

	got, err := assemblePartitionRoots(partitionBlobs, func(path []byte, hash common.Hash, blob []byte) {
		g.writeAccountNode(batch, path, hash, blob)
	})
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("state root mismatch: got %x, want %x", got, want)
	}
	if batch.ValueSize() != 0 {
		return batch.Write()
	}
	return nil
}

func (g *trieGenerator) writeAccountNode(batch ethdb.Batch, path []byte, hash common.Hash, blob []byte) {
	g.counters.accountTrieNodes.Add(1)
	if g.scheme == rawdb.PathScheme {
		g.counters.accountTrieBytes.Add(int64(len(path) + len(blob)))
	} else {
		g.counters.accountTrieBytes.Add(int64(common.HashLength + len(blob)))
	}
	rawdb.WriteTrieNode(batch, common.Hash{}, path, hash, blob, g.scheme)
}

// tickProgress logs an aggregate progress line every 30 seconds until done
// is closed. Cheap: a handful of atomic loads and one log line per tick.
func tickProgress(done <-chan struct{}, start time.Time, c *genCounters, prog *atomic.Uint64) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			elapsed := time.Since(start)
			fraction := progressFraction(&c.progress)

			// Notify the external subscriber about the generation progress
			if prog != nil {
				prog.Store(uint64(100 * fraction))
			}
			eta := "n/a"
			if fraction > 0.005 {
				eta = common.PrettyDuration(time.Duration(float64(elapsed) * (1.0/fraction - 1.0))).String()
			}
			log.Info("Generating trie",
				"progress", fmt.Sprintf("%.1f%%", fraction*100), "eta", eta,
				"accounts", c.accounts.Load(), "slots", c.slots.Load(),
				"account-updated", c.accountUpdated.Load(), "dangling-slots", c.storageDeleted.Load(),
				"elapsed", common.PrettyDuration(elapsed),
				"acct/s", uint64(float64(c.accounts.Load())/elapsed.Seconds()))
		}
	}
}

// progressFraction averages each partition's iterator position (as a fraction
// of its hash range) into an overall completion estimate in [0, 1]. Keccak
// hashes are uniform, so keyspace position is a good proxy for work done.
func progressFraction(progress *[numPartitions]atomic.Uint64) float64 {
	var total float64
	for i := range progress {
		p := progress[i].Load()
		if p == partitionFinished {
			total++
		} else {
			total += float64(p) / float64(partitionRangeSize)
		}
	}
	return total / float64(numPartitions)
}
