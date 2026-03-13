// Copyright 2025 The go-ethereum Authors
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

package pathdb

import (
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

const (
	// indexPruningThreshold defines the number of pruned histories that must
	// accumulate before triggering index pruning. This helps avoid scheduling
	// index pruning too frequently.
	indexPruningThreshold = 90000
)

// indexPruner is responsible for pruning stale index data from the tail side
// when old history objects are removed. It runs as a background goroutine and
// processes pruning signals whenever the history tail advances.
//
// The pruning operates at the block level: for each state element's index
// metadata, leading index blocks whose maximum history ID falls below the
// new tail are removed entirely. This avoids the need to decode individual
// block contents and is efficient because index blocks store monotonically
// increasing history IDs.
type indexPruner struct {
	disk    ethdb.KeyValueStore
	typ     historyType
	tail    atomic.Uint64 // Tail below which index entries can be pruned
	lastRun uint64        // The tail in the last pruning run
	trigger chan struct{} // Non-blocking signal that tail has advanced
	closed  chan struct{}
	wg      sync.WaitGroup
	log     log.Logger
}

// newIndexPruner creates and starts a new index pruner for the given history type.
func newIndexPruner(disk ethdb.KeyValueStore, typ historyType) *indexPruner {
	p := &indexPruner{
		disk:    disk,
		typ:     typ,
		trigger: make(chan struct{}, 1),
		closed:  make(chan struct{}),
		log:     log.New("type", typ.String()),
	}
	p.wg.Add(1)
	go p.run()
	return p
}

// prune signals the pruner that the history tail has advanced to the given ID.
// All index entries referencing history IDs below newTail can be removed.
func (p *indexPruner) prune(newTail uint64) {
	// Only update if the tail is actually advancing
	for {
		old := p.tail.Load()
		if newTail <= old {
			return
		}
		if p.tail.CompareAndSwap(old, newTail) {
			break
		}
	}
	// Non-blocking signal
	select {
	case p.trigger <- struct{}{}:
	default:
	}
}

// close shuts down the pruner and waits for it to finish.
func (p *indexPruner) close() {
	select {
	case <-p.closed:
		return
	default:
		close(p.closed)
		p.wg.Wait()
	}
}

// run is the main loop of the pruner. It waits for trigger signals and
// processes a small batch of entries on each trigger, advancing the cursor.
func (p *indexPruner) run() {
	defer p.wg.Done()

	for {
		select {
		case <-p.trigger:
			tail := p.tail.Load()
			if tail < p.lastRun || tail-p.lastRun < indexPruningThreshold {
				continue
			}
			if err := p.process(tail); err != nil {
				p.log.Error("Failed to prune index", "tail", tail, "err", err)
			} else {
				p.lastRun = tail
			}

		case <-p.closed:
			return
		}
	}
}

// process iterates all index metadata entries for the history type and prunes
// leading blocks whose max history ID is below the given tail.
func (p *indexPruner) process(tail uint64) error {
	var (
		err     error
		pruned  int
		scanned int
		start   = time.Now()
	)
	switch p.typ {
	case typeStateHistory:
		pn, sn, err := p.prunePrefix(rawdb.StateHistoryAccountMetadataPrefix, typeAccount, tail)
		if err != nil {
			return err
		}
		pruned += pn
		scanned += sn

		pn, sn, err = p.prunePrefix(rawdb.StateHistoryStorageMetadataPrefix, typeStorage, tail)
		if err != nil {
			return err
		}
		pruned += pn
		scanned += sn
		statePruneHistoryIndexTimer.UpdateSince(start)

	case typeTrienodeHistory:
		pruned, scanned, err = p.prunePrefix(rawdb.TrienodeHistoryMetadataPrefix, typeTrienode, tail)
		if err != nil {
			return err
		}
		trienodePruneHistoryIndexTimer.UpdateSince(start)

	default:
		panic("unknown history type")
	}
	if pruned > 0 {
		p.log.Info("Pruned stale index blocks", "pruned", pruned, "scanned", scanned, "tail", tail, "elapsed", common.PrettyDuration(time.Since(start)))
	}
	return nil
}

// prunePrefix scans up to indexPruneBatchSize metadata entries starting from
// the cursor position and prunes leading index blocks below the tail. The
// cursor advances after each cycle; when the prefix is fully scanned, the
// cursor resets so the next cycle starts from the beginning.
// Returns (prunedBlocks, scannedEntries, error).
func (p *indexPruner) prunePrefix(prefix []byte, elemType elementType, tail uint64) (int, int, error) {
	var (
		pruned  int
		scanned int
		batch   = p.disk.NewBatchWithSize(ethdb.IdealBatchSize)
	)
	it := p.disk.NewIterator(prefix, nil)
	defer it.Release()

	for it.Next() {
		// Check for shutdown
		select {
		case <-p.closed:
			return pruned, scanned, nil
		default:
		}
		scanned++
		key, value := it.Key(), it.Value()

		ident, bsize := p.identFromKey(key, prefix, elemType)
		n, err := p.pruneEntry(batch, ident, value, bsize, tail)
		if err != nil {
			p.log.Warn("Failed to prune index entry", "ident", ident, "err", err)
			continue
		}
		pruned += n

		if batch.ValueSize() >= ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return 0, 0, err
			}
			batch.Reset()
		}
	}
	if batch.ValueSize() > 0 {
		if err := batch.Write(); err != nil {
			return 0, 0, err
		}
	}
	return pruned, scanned, nil
}

// identFromKey reconstructs the stateIdent and bitmapSize from a metadata key.
func (p *indexPruner) identFromKey(key []byte, prefix []byte, elemType elementType) (stateIdent, int) {
	rest := key[len(prefix):]

	switch elemType {
	case typeAccount:
		// key = prefix + addressHash(32)
		var addrHash common.Hash
		copy(addrHash[:], rest[:32])
		return newAccountIdent(addrHash), 0

	case typeStorage:
		// key = prefix + addressHash(32) + storageHash(32)
		var addrHash, storHash common.Hash
		copy(addrHash[:], rest[:32])
		copy(storHash[:], rest[32:64])
		return newStorageIdent(addrHash, storHash), 0

	case typeTrienode:
		// key = prefix + addressHash(32) + path(variable)
		var addrHash common.Hash
		copy(addrHash[:], rest[:32])
		path := string(rest[32:])
		ident := newTrienodeIdent(addrHash, path)
		return ident, ident.bloomSize()

	default:
		panic("unknown element type")
	}
}

// pruneEntry checks a single metadata entry and removes leading index blocks
// whose max < tail. Returns the number of blocks pruned.
func (p *indexPruner) pruneEntry(batch ethdb.Batch, ident stateIdent, blob []byte, bsize int, tail uint64) (int, error) {
	// Fast path: the first 8 bytes of the metadata encode the max history ID
	// of the first index block (big-endian uint64). If it is >= tail, no
	// blocks can be pruned and we skip the full parse entirely.
	if len(blob) >= 8 && binary.BigEndian.Uint64(blob[:8]) >= tail {
		return 0, nil
	}
	descList, err := parseIndex(blob, bsize)
	if err != nil {
		return 0, err
	}
	// Find the number of leading blocks that can be entirely pruned.
	// A block can be pruned if its max history ID is strictly below
	// the tail.
	var count int
	for _, desc := range descList {
		if desc.max < tail {
			count++
		} else {
			break // blocks are ordered, no more to prune
		}
	}
	if count == 0 {
		return 0, nil
	}
	// Delete the pruned index blocks
	for i := 0; i < count; i++ {
		deleteStateIndexBlock(ident, batch, descList[i].id)
	}
	// Update or delete the metadata
	remaining := descList[count:]
	if len(remaining) == 0 {
		// All blocks pruned, remove the metadata entry entirely
		deleteStateIndex(ident, batch)
	} else {
		// Rewrite the metadata with the remaining blocks
		size := indexBlockDescSize + bsize
		buf := make([]byte, 0, size*len(remaining))
		for _, desc := range remaining {
			buf = append(buf, desc.encode()...)
		}
		writeStateIndex(ident, batch, buf)
	}
	return count, nil
}
