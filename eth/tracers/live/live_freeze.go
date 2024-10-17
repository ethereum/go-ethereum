package live

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	freezeThreshold = 64 // the max number of blocks to freeze in one batch
	kvdbTailKey     = "FilterFreezerTail"
)

func (l *live) freeze(maxKeepBlocks uint64) {
	var lastFinalized uint64
	var freezeErr error
	for {
		select {
		case <-l.stopCh:
			return
		case finalizedBlock := <-l.freezeCh:
			// Skip if the finalized block is not increasing
			if finalizedBlock <= lastFinalized {
				continue
			}
			lastFinalized = finalizedBlock

			// Check if error occurred in previous iteration
			if freezeErr != nil {
				log.Error("Error occurred in previous freezing, checking the log for more detail", "error", freezeErr)
				continue
			}

			tail := l.getFreezerTail()

			// Freeze at most freezeThreshold blocks
			freezeUpTo := min(finalizedBlock, tail+freezeThreshold)
			if freezeUpTo <= tail {
				continue
			}

			log.Info("Move traces from kvdb to frdb", "from", tail, "to", freezeUpTo-1)

			for blknum := tail; blknum < freezeUpTo; blknum++ {
				freezeErr = l.moveBlockToFreezer(blknum)
				if freezeErr != nil {
					log.Error("Failed to move block to freezer", "block", blknum, "error", freezeErr)
					break
				}
			}
			if freezeErr != nil {
				continue
			}

			// Update the tail of the freezer
			if err := l.updateFreezerTail(freezeUpTo); err != nil {
				log.Warn("Failed to update freezer tail", "old", tail, "new", freezeUpTo, "error", err)
				continue
			}

			// No need to prune
			if maxKeepBlocks == 0 {
				continue
			}

			frozen, err := l.frdb.Ancients()
			if err != nil {
				log.Error("Failed to get number of ancient items", "error", err)
				continue
			}

			// Not enough blocks to prune
			if frozen <= maxKeepBlocks {
				continue
			}

			// Prune old blocks if necessary
			itemsToPrune := min(freezeThreshold, frozen-maxKeepBlocks)
			from := l.offset.Load()
			head := from + itemsToPrune
			log.Info("Prune old blocks", "pruned", itemsToPrune, "from", from, "to", head)
			if err := l.pruneBlocksFromFreezer(frozen-itemsToPrune, head); err != nil {
				log.Error("Failed to prune blocks from freezer", "error", err)
			}
		}
	}
}

func (l *live) getFreezerTail() (tail uint64) {
	tailBytes, _ := l.kvdb.Get([]byte(kvdbTailKey))

	if len(tailBytes) > 0 {
		tail = binary.BigEndian.Uint64(tailBytes)
	} else {
		// If tail is 0 (not found in kvdb), use the offset
		tail = l.offset.Load()
	}
	return
}

func (l *live) updateFreezerTail(tail uint64) error {
	tailBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tailBytes, tail)
	return l.kvdb.Put([]byte(kvdbTailKey), tailBytes)
}

func (l *live) moveBlockToFreezer(blknum uint64) error {
	header, err := l.backend.HeaderByNumber(context.Background(), rpc.BlockNumber(blknum))
	if err != nil {
		return err
	}

	offset := l.offset.Load()

	size, err := l.frdb.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for name := range l.tracer.Tracers() {
			kvKey := toKVKey(name, blknum, header.Hash())
			data, err := l.kvdb.Get(kvKey)
			if err != nil {
				return err
			}

			if err := op.AppendRaw(toTraceTable(name), blknum-offset, data); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	log.Info("Move from kvdb to frdb", "blknum", blknum, "size", size)

	// Delete all entries for this prefix from kvdb, ignore error
	if err := l.deleteKVDBEntriesWithPrefix(blknum); err != nil {
		log.Error("Failed to delete entries from kvdb", "error", err)
	}

	return nil
}

func (l *live) deleteKVDBEntriesWithPrefix(blknum uint64) error {
	prefix := encodeNumber(blknum)
	batch := l.kvdb.NewBatch()
	it := l.kvdb.NewIterator(prefix, nil)
	defer it.Release()

	for it.Next() {
		if err := batch.Delete(it.Key()); err != nil {
			return fmt.Errorf("failed to add delete operation to batch: %w", err)
		}

		// Write batch if it's getting too large
		if batch.ValueSize() > ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return fmt.Errorf("failed to write batch: %w", err)
			}
			batch.Reset()
		}
	}

	if err := it.Error(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	// Write any remaining batch operations
	if batch.ValueSize() > 0 {
		if err := batch.Write(); err != nil {
			return fmt.Errorf("failed to write final batch: %w", err)
		}
	}

	return nil
}

func (l *live) pruneBlocksFromFreezer(items, newHead uint64) error {
	if _, err := l.frdb.TruncateHead(items); err != nil {
		return err
	}
	// Set the offset of the new head
	l.offset.Store(newHead)
	return nil
}
