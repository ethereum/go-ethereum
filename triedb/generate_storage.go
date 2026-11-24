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
	"github.com/ethereum/go-ethereum/trie"
	"golang.org/x/sync/errgroup"
)

const (
	numStoragePartitions = 16   // one subtree per leading slot-hash nibble
	storageProbeSlots    = 4096 // the number of probes for deciding the contract size

	// More than 4096 slots in the first 1/2048 of the keyspace suggests at
	// least 8 million slots across a uniformly distributed storage trie.
	storageProbeLimit uint32 = 1 << 21
)

// probeStorage checks a bounded prefix of one owner's flat storage. The
// partition iterator stays in place so small contracts can continue in order.
func (g *trieGenerator) probeStorage(ctx context.Context, cancel <-chan struct{}, owner common.Hash, firstSlot []byte) (bool, error) {
	// Short circuit if the first slot indicates the potential slot
	// number is less than 2k, avoiding opening the iterator.
	if binary.BigEndian.Uint32(firstSlot) >= storageProbeLimit {
		return false, nil
	}
	prefix := slices.Concat(rawdb.SnapshotStoragePrefix, owner[:])
	it := openFlatIterator(g.db, prefix, firstSlot, common.HashLength)
	defer it.Release()

	for i := 0; it.Next(); i++ {
		select {
		case <-cancel:
			return false, ErrCancelled
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		if binary.BigEndian.Uint32(it.Key()[len(prefix):]) >= storageProbeLimit {
			return false, nil
		}
		if i == storageProbeSlots {
			return true, nil
		}
	}
	return false, it.Error()
}

func (g *trieGenerator) splitStorage(ctx context.Context, cancel <-chan struct{}, owner common.Hash, batch ethdb.Batch) (common.Hash, error) {
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
			defer func() { slots <- struct{}{} }()
			root, err := g.generateStoragePartition(ctx, cancel, owner, byte(i))
			roots[i] = root
			return err
		})
	}
	if err := group.Wait(); err != nil {
		return common.Hash{}, err
	}
	if dispatchErr != nil {
		return common.Hash{}, dispatchErr
	}
	return assemblePartitionRoots(&roots, func(path []byte, hash common.Hash, blob []byte) {
		g.writeStorageNode(batch, owner, path, hash, blob)
	})
}

func (g *trieGenerator) generateStoragePartition(ctx context.Context, cancel <-chan struct{}, owner common.Hash, nibble byte) ([]byte, error) {
	prefix := slices.Concat(rawdb.SnapshotStoragePrefix, owner[:])

	it := openFlatIterator(g.db, prefix, []byte{nibble << 4}, common.HashLength)
	defer it.Release()

	batch := g.db.NewBatchWithSize(ethdb.IdealBatchSize)
	defer batch.Close()

	var root []byte
	subtree := trie.NewPartialStackTrie(nibble, func(path []byte, hash common.Hash, blob []byte) {
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

		if batch.ValueSize() > ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return nil, fmt.Errorf("flush storage partition %x for %x: %w", nibble, owner, err)
			}
			batch.Reset()
		}
	}
	if err := it.Error(); err != nil {
		return nil, fmt.Errorf("iterate storage partition %x for %x: %w", nibble, owner, err)
	}
	subtree.Hash()
	if batch.ValueSize() != 0 {
		if err := batch.Write(); err != nil {
			return nil, fmt.Errorf("flush storage partition %x for %x: %w", nibble, owner, err)
		}
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
