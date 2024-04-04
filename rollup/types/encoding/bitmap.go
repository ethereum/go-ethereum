package encoding

import (
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/core/types"
)

// ConstructSkippedBitmap constructs skipped L1 message bitmap of the batch.
func ConstructSkippedBitmap(batchIndex uint64, chunks []*Chunk, totalL1MessagePoppedBefore uint64) ([]byte, uint64, error) {
	// skipped L1 message bitmap, an array of 256-bit bitmaps
	var skippedBitmap []*big.Int

	// the first queue index that belongs to this batch
	baseIndex := totalL1MessagePoppedBefore

	// the next queue index that we need to process
	nextIndex := totalL1MessagePoppedBefore

	for chunkID, chunk := range chunks {
		for blockID, block := range chunk.Blocks {
			for _, tx := range block.Transactions {
				if tx.Type != types.L1MessageTxType {
					continue
				}
				currentIndex := tx.Nonce

				if currentIndex < nextIndex {
					return nil, 0, fmt.Errorf("unexpected batch payload, expected queue index: %d, got: %d. Batch index: %d, chunk index in batch: %d, block index in chunk: %d, block hash: %v, transaction hash: %v", nextIndex, currentIndex, batchIndex, chunkID, blockID, block.Header.Hash(), tx.TxHash)
				}

				// mark skipped messages
				for skippedIndex := nextIndex; skippedIndex < currentIndex; skippedIndex++ {
					quo := int((skippedIndex - baseIndex) / 256)
					rem := int((skippedIndex - baseIndex) % 256)
					for len(skippedBitmap) <= quo {
						bitmap := big.NewInt(0)
						skippedBitmap = append(skippedBitmap, bitmap)
					}
					skippedBitmap[quo].SetBit(skippedBitmap[quo], rem, 1)
				}

				// process included message
				quo := int((currentIndex - baseIndex) / 256)
				for len(skippedBitmap) <= quo {
					bitmap := big.NewInt(0)
					skippedBitmap = append(skippedBitmap, bitmap)
				}

				nextIndex = currentIndex + 1
			}
		}
	}

	bitmapBytes := make([]byte, len(skippedBitmap)*32)
	for ii, num := range skippedBitmap {
		bytes := num.Bytes()
		padding := 32 - len(bytes)
		copy(bitmapBytes[32*ii+padding:], bytes)
	}

	return bitmapBytes, nextIndex, nil
}
