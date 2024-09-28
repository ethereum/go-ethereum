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

func (f *live) freeze(maxKeepBlocks uint64) {
	var lastFinalized uint64
	for {
		select {
		case <-f.stopCh:
			return
		case finalizedBlock := <-f.freezeCh:
			if finalizedBlock <= lastFinalized {
				continue
			}
			lastFinalized = finalizedBlock

			tail := f.getFreezerTail()

			// Freeze at most freezeThreshold blocks
			freezeUpTo := min(finalizedBlock, tail+freezeThreshold)
			if freezeUpTo <= tail {
				continue
			}

			log.Info("Move traces from kvdb to frdb", "from", tail, "to", freezeUpTo-1)
			for blknum := tail; blknum < freezeUpTo; blknum++ {
				if err := f.moveBlockToFreezer(blknum); err != nil {
					log.Error("Failed to move block to freezer", "block", blknum, "error", err)
					break
				}
			}

			// Update the tail of the freezer
			tail = freezeUpTo
			if err := f.updateFreezerTail(tail); err != nil {
				log.Error("Failed to update freezer tail", "error", err)
			}

			frozen, _ := f.frdb.Ancients()
			offset := f.offset.Load()

			// No pruning
			if maxKeepBlocks == 0 || frozen <= maxKeepBlocks {
				continue
			}

			// Prune old blocks if necessary
			itemsToPrune := min(freezeThreshold, frozen-maxKeepBlocks)
			head := offset + itemsToPrune - 1
			log.Info("Prune old blocks", "pruned", itemsToPrune, "from", offset, "to", head)
			if err := f.pruneBlocksFromFreezer(frozen-itemsToPrune, head); err != nil {
				log.Error("Failed to prune blocks from freezer", "error", err)
			}
		}
	}
}

func (f *live) getFreezerTail() (tail uint64) {
	tailBytes, _ := f.kvdb.Get([]byte(kvdbTailKey))

	if len(tailBytes) > 0 {
		tail = binary.BigEndian.Uint64(tailBytes)
	} else {
		// If tail is 0 (not found in kvdb), use the offset
		tail = f.offset.Load()
	}
	return
}

func (f *live) updateFreezerTail(tail uint64) error {
	tailBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tailBytes, tail)
	return f.kvdb.Put([]byte(kvdbTailKey), tailBytes)
}

func (f *live) moveBlockToFreezer(blknum uint64) error {
	header, err := f.backend.HeaderByNumber(context.Background(), rpc.BlockNumber(blknum))
	if err != nil {
		return err
	}

	offset := f.offset.Load()

	size, err := f.frdb.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for name := range f.tracer.Tracers() {
			kvKey := toKVKey(name, blknum, header.Hash())
			data, err := f.kvdb.Get(kvKey)
			if err != nil {
				return err
			}

			table := toTraceTable(name)
			err = op.AppendRaw(table, blknum-offset, data)
			if err != nil {
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
	if err := f.deleteKVDBEntriesWithPrefix(blknum); err != nil {
		log.Error("Failed to delete entries from kvdb", "error", err)
	}

	return nil
}

func (f *live) deleteKVDBEntriesWithPrefix(blknum uint64) error {
	prefix := encodeNumber(blknum)
	batch := f.kvdb.NewBatch()
	it := f.kvdb.NewIterator(prefix, nil)
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

func (f *live) pruneBlocksFromFreezer(items, head uint64) error {
	if _, err := f.frdb.TruncateHead(items); err != nil {
		return err
	}
	// Head should be in sync with the on-mem offset
	f.offset.Store(head)
	return nil
}
