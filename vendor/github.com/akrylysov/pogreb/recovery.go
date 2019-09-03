package pogreb

import (
	"math"
	"sort"
)

func align51264(n int64) int64 {
	return (n + 511) &^ 511
}

func truncateFiles(db *DB) error {
	db.index.size = align51264(db.index.size)
	if err := db.index.Truncate(db.index.size); err != nil {
		return err
	}
	if err := db.index.Mmap(db.index.size); err != nil {
		return err
	}
	db.data.size = align51264(db.data.size)
	if err := db.data.Truncate(db.data.size); err != nil {
		return err
	}
	if err := db.data.Mmap(db.data.size); err != nil {
		return err
	}
	return nil
}

func getUsedBlocks(db *DB) (uint32, []block, error) {
	var itemCount uint32
	var usedBlocks []block
	for bucketIdx := uint32(0); bucketIdx < db.nBuckets; bucketIdx++ {
		err := db.forEachBucket(bucketIdx, func(b bucketHandle) (bool, error) {
			for i := 0; i < slotsPerBucket; i++ {
				sl := b.slots[i]
				if sl.kvOffset == 0 {
					return true, nil
				}
				itemCount++
				usedBlocks = append(usedBlocks, block{size: align512(sl.kvSize()), offset: sl.kvOffset})
			}
			if b.next != 0 {
				usedBlocks = append(usedBlocks, block{size: bucketSize, offset: b.next})
			}
			return false, nil
		})
		if err != nil {
			return 0, nil, err
		}
	}
	return itemCount, usedBlocks, nil
}

func recoverSplitCrash(db *DB) error {
	if db.nBuckets == 1 {
		return nil
	}
	prevnBuckets := db.nBuckets - 1
	prevLevel := uint8(math.Floor(math.Log2(float64(prevnBuckets))))
	prevSplitBucketIdx := prevnBuckets - (uint32(1) << prevLevel)
	splitCrash := false
	err := db.forEachBucket(prevSplitBucketIdx, func(b bucketHandle) (bool, error) {
		for i := 0; i < slotsPerBucket; i++ {
			sl := b.slots[i]
			if sl.kvOffset == 0 {
				return true, nil
			}
			if db.bucketIndex(sl.hash) != prevSplitBucketIdx {
				splitCrash = true
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	if !splitCrash {
		return nil
	}
	logger.Print("Detected split crash. Truncating index file...")
	if err := db.index.Truncate(db.index.size - int64(bucketSize)); err != nil {
		return err
	}
	db.index.size -= int64(bucketSize)
	if err := db.index.Mmap(db.index.size); err != nil {
		return err
	}
	db.nBuckets = prevnBuckets
	db.level = prevLevel
	db.splitBucketIdx = prevSplitBucketIdx
	return nil
}

func recoverFreeList(db *DB, usedBlocks []block) error {
	if len(usedBlocks) == 0 {
		return nil
	}
	sort.Slice(usedBlocks, func(i, j int) bool {
		return usedBlocks[i].offset < usedBlocks[j].offset
	})
	fl := freelist{}
	expectedOff := int64(headerSize)
	for _, bl := range usedBlocks {
		if bl.offset > expectedOff {
			fl.free(expectedOff, uint32(bl.offset-expectedOff))
		}
		expectedOff = bl.offset + int64(bl.size)
	}
	lastBlock := usedBlocks[len(usedBlocks)-1]
	lastOffset := int64(lastBlock.size) + lastBlock.offset
	if db.data.size > lastOffset {
		fl.free(lastOffset, uint32(db.data.size-lastOffset))
		logger.Println(lastBlock, db.data.size)
	}
	logger.Printf("Recovered freelist. Old len=%d; new len=%d\n", len(db.data.fl.blocks), len(fl.blocks))
	db.data.fl = fl
	return nil
}

func (db *DB) recover() error {
	logger.Println("Performing recovery...")
	logger.Printf("Index file size=%d; data file size=%d\n", db.index.size, db.data.size)
	logger.Printf("Header dbInfo %+v\n", db.dbInfo)

	// Truncate index and data files.
	if err := truncateFiles(db); err != nil {
		return err
	}

	// Recover header.
	db.nBuckets = uint32((db.index.size - int64(headerSize)) / int64(bucketSize))
	db.level = uint8(math.Floor(math.Log2(float64(db.nBuckets))))
	db.splitBucketIdx = db.nBuckets - (uint32(1) << db.level)
	itemCount, usedBlocks, err := getUsedBlocks(db)
	if err != nil {
		return err
	}
	db.count = itemCount

	// Check if crash occurred during split.
	if err := recoverSplitCrash(db); err != nil {
		return err
	}
	logger.Printf("Recovered dbInfo %+v\n", db.dbInfo)

	// Recover free list.
	if err := recoverFreeList(db, usedBlocks); err != nil {
		return err
	}
	logger.Println("Recovery complete.")
	return nil
}
