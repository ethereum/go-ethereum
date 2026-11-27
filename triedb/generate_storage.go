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
	"context"
	"encoding/binary"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"golang.org/x/sync/errgroup"
)

const (
	numStoragePartitions = 16   // one subtree per leading slot-hash nibble
	storageProbeSlots    = 4096 // the number of probes for deciding the contract size

	// More than 4096 slots in a 1/2048 edge of the keyspace suggests at
	// least 8 million slots across a uniformly distributed storage trie.
	storageProbeLimit uint32 = 1 << 21
)

func (g *trieGenerator) inStorageProbe(slot []byte) bool {
	position := binary.BigEndian.Uint32(slot)
	if g.ingester != nil {
		position = ^position
	}
	return position < storageProbeLimit
}

// probeStorage checks the leading edge of the owner's slots in scan order.
// The partition iterator stays in place so small contracts can continue.
func (g *trieGenerator) probeStorage(ctx context.Context, cancel <-chan struct{}, owner common.Hash, firstSlot []byte) (bool, error) {
	if !g.inStorageProbe(firstSlot) {
		return false, nil
	}
	prefix := slices.Concat(rawdb.SnapshotStoragePrefix, owner[:])
	it := openFlatIterator(g.newIterator, prefix, firstSlot, common.HashLength)
	defer it.Release()

	for i := 0; it.Next(); i++ {
		select {
		case <-cancel:
			return false, ErrCancelled
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		if !g.inStorageProbe(it.Key()[len(prefix):]) {
			return false, nil
		}
		if i == storageProbeSlots {
			return true, nil
		}
	}
	return false, it.Error()
}

func (g *trieGenerator) generateSplitStorage(ctx context.Context, cancel <-chan struct{}, iters *rangeIterators, batches *partitionBatches, owner common.Hash) (common.Hash, error) {
	// Release the parent read views while the shards work on this owner.
	iters.release()

	// Flush buffered storage nodes from earlier owners before reusing this
	// batch for the large contract's shards. For example, an SST containing
	// f2/... and f1/0/... would span the f1/1/... shard, since
	// f1/0/... < f1/1/... < f2/... (owner hashes f1,f2 abbreviated).
	if err := batches.storage.flush(); err != nil {
		return common.Hash{}, fmt.Errorf("flush before splitting storage %x: %w", owner, err)
	}
	root, err := g.splitStorage(ctx, cancel, owner, batches)
	if err != nil {
		return common.Hash{}, fmt.Errorf("split storage %x: %w", owner, err)
	}
	iters.resumeAfterStorage(owner)
	return root, nil
}

func (g *trieGenerator) splitStorage(ctx context.Context, cancel <-chan struct{}, owner common.Hash, batches *partitionBatches) (common.Hash, error) {
	// The suspended account worker lends its own slot to this contract.
	ownerSlot := make(chan struct{}, 1)
	ownerSlot <- struct{}{}
	group, ctx := errgroup.WithContext(ctx)

	var (
		roots       [numStoragePartitions][]byte
		dispatchErr error
	)
schedule:
	for i := range numStoragePartitions {
		var slots chan struct{}
		select {
		case <-cancel:
			dispatchErr = ErrCancelled
			break schedule
		case <-ctx.Done():
			dispatchErr = ctx.Err()
			break schedule
		case <-ownerSlot:
			slots = ownerSlot
		case <-g.availableWorkers:
			slots = g.availableWorkers
		}
		group.Go(func() error {
			var batch *generationBatch
			if slots == ownerSlot {
				// The worker of own partition with batch initialized
				batch = batches.storage
			} else {
				// The worker borrowed from other finished partition
				defer func() { slots <- struct{}{} }()
				batch = newGenerationBatch(g.db, g.ingester)
				defer batch.Close()
			}
			root, err := g.generateStoragePartition(ctx, cancel, owner, byte(i), batch)
			if err != nil {
				return err
			}
			roots[i] = root
			if slots == ownerSlot {
				ownerSlot <- struct{}{}
			}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return common.Hash{}, err
	}
	if dispatchErr != nil {
		return common.Hash{}, dispatchErr
	}
	root, err := assemblePartitionRoots(&roots, func(path []byte, hash common.Hash, blob []byte) {
		// An SST containing several shard roots would span descendant
		// ranges already ingested by other shards.
		g.writeStorageNode(batches.flat, owner, path, hash, blob)
	})
	if err != nil {
		return common.Hash{}, err
	}
	if batches.flat.full() {
		if err := batches.flat.flush(); err != nil {
			return common.Hash{}, err
		}
	}
	return root, nil
}

func (g *trieGenerator) generateStoragePartition(ctx context.Context, cancel <-chan struct{}, owner common.Hash, nibble byte, batch *generationBatch) ([]byte, error) {
	prefix := slices.Concat(rawdb.SnapshotStoragePrefix, owner[:])

	start := []byte{nibble << 4}
	if g.ingester != nil {
		last := common.MaxHash
		last[0] = nibble<<4 | 0xf
		start = last[:]
	}
	it := openFlatIterator(g.newIterator, prefix, start, common.HashLength)
	defer func() { it.Release() }()

	var root []byte
	subtree := g.newPartialStackTrie(nibble, func(path []byte, hash common.Hash, blob []byte) {
		if len(path) == 1 {
			// The parent may embed or fold this node when assembling the root.
			root = common.CopyBytes(blob)
			return
		}
		g.writeStorageNode(batch, owner, path, hash, blob)
	})
	for it.Next() {
		select {
		case <-cancel:
			return nil, ErrCancelled
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		slot := it.Key()[len(prefix):]
		if slot[0]>>4 != nibble {
			break
		}
		if err := subtree.Update(slot, it.Value()); err != nil {
			return nil, fmt.Errorf("build storage partition %x for %x: %w", nibble, owner, err)
		}
		g.counters.slots.Add(1)

		if batch.full() {
			if err := batch.flush(); err != nil {
				return nil, fmt.Errorf("flush storage partition %x for %x: %w", nibble, owner, err)
			}
			var err error
			it, err = reopenFlatIterator(g.newIterator, it, prefix, common.HashLength)
			if err != nil {
				return nil, fmt.Errorf("reopen storage partition %x for %x: %w", nibble, owner, err)
			}
		}
	}
	if err := it.Error(); err != nil {
		return nil, fmt.Errorf("iterate storage partition %x for %x: %w", nibble, owner, err)
	}
	subtree.Hash()
	if err := batch.flush(); err != nil {
		return nil, fmt.Errorf("flush storage partition %x for %x: %w", nibble, owner, err)
	}
	return root, nil
}

func (g *trieGenerator) writeStorageNode(dst ethdb.KeyValueWriter, owner common.Hash, path []byte, hash common.Hash, blob []byte) {
	var size int
	if g.scheme == rawdb.PathScheme {
		size = common.HashLength + len(path) + len(blob)
	} else {
		size = common.HashLength + len(blob)
	}
	g.counters.storageTrieBytes.Add(int64(size))
	g.counters.storageTrieNodes.Add(1)

	rawdb.WriteTrieNode(dst, owner, path, hash, blob, g.scheme)
}
