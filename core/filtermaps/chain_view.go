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

package filtermaps

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// blockchain represents the underlying blockchain of ChainView.
type blockchain interface {
	GetHeader(hash common.Hash, number uint64) *types.Header
	GetCanonicalHash(number uint64) common.Hash
	GetReceiptsByHash(hash common.Hash) types.Receipts
}

// ChainView represents an immutable view of a chain with a block id and a set
// of receipts associated to each block number and a block hash associated with
// all block numbers except the head block. This is because in the future
// ChainView might represent a view where the head block is currently being
// created. Block id is a unique identifier that can also be calculated for the
// head block.
// Note that the view's head does not have to be the current canonical head
// of the underlying blockchain, it should only possess the block headers
// and receipts up until the expected chain view head.
type ChainView struct {
	chain        blockchain
	finishedHead uint64
	hashes       []common.Hash // block hashes starting backwards from finishedHead until first canonical hash
}

// NewChainView creates a new ChainView.
func NewChainView(chain blockchain, number uint64, hash common.Hash) *ChainView {
	cv := &ChainView{
		chain:        chain,
		finishedHead: number,
		hashes:       []common.Hash{hash},
	}
	cv.extendNonCanonical()
	return cv
}

// ProcessedHead returns the highest block number where BlockId and Receipts
// are available.
// Note that though in the current implementation ProcessedHead is always equal
// to FinishedHead, the called should expect that ProcessedHead can be one block
// higher in case the view represents a chain during block processing, when the
// receipts and block id are already available but the header is not finished
// yet and therefore the block hash cannot be calculated.
func (cv *ChainView) ProcessedHead() uint64 {
	return cv.finishedHead
}

// FinishedHead returns the highest block number where BlockHash and Header
// are available.
func (cv *ChainView) FinishedHead() uint64 {
	return cv.finishedHead
}

// BlockHash returns the block hash belonging to the given block number.
// Note that the hash of the head block is not returned because ChainView might
// represent a view where the head block is currently being created.
func (cv *ChainView) BlockHash(number uint64) common.Hash {
	if number > cv.finishedHead {
		panic("invalid block number")
	}
	return cv.blockHash(number)
}

// BlockId returns the unique block id belonging to the given block number.
// Note that it is currently equal to the block hash. In the future it might
// be a different id for future blocks if the log index root becomes part of
// consensus and therefore rendering the index with the new head will happen
// before the hash of that new head is available.
func (cv *ChainView) BlockId(number uint64) common.Hash {
	if number > cv.finishedHead {
		panic("invalid block number")
	}
	return cv.blockHash(number)
}

// Header returns the block header at the given block number.
func (cv *ChainView) Header(number uint64) *types.Header {
	return cv.chain.GetHeader(cv.BlockHash(number), number)
}

// Receipts returns the set of receipts belonging to the block at the given
// block number.
func (cv *ChainView) Receipts(number uint64) types.Receipts {
	if number > cv.finishedHead {
		panic("invalid block number")
	}
	blockHash := cv.blockHash(number)
	if blockHash == (common.Hash{}) {
		log.Error("Chain view: block hash unavailable", "number", number, "head", cv.finishedHead)
	}
	return cv.chain.GetReceiptsByHash(blockHash)
}

// SharedRange returns the block range shared by two chain views.
func (cv *ChainView) SharedRange(cv2 *ChainView) common.Range[uint64] {
	if cv == nil || cv2 == nil {
		return common.Range[uint64]{}
	}
	var sharedLen uint64
	for n := min(cv.finishedHead+1-uint64(len(cv.hashes)), cv2.finishedHead+1-uint64(len(cv2.hashes))); n <= cv.finishedHead && n <= cv2.finishedHead && cv.blockHash(n) == cv2.blockHash(n); n++ {
		sharedLen = n + 1
	}
	return common.NewRange(0, sharedLen)
}

// limitedView returns a new chain view that is a truncated version of the parent view.
func (cv *ChainView) limitedView(newHead uint64) *ChainView {
	if newHead >= cv.finishedHead {
		return cv
	}
	return NewChainView(cv.chain, newHead, cv.blockHash(newHead))
}

// equalViews returns true if the two chain views are equivalent.
func equalViews(cv1, cv2 *ChainView) bool {
	if cv1 == nil || cv2 == nil {
		return false
	}
	return cv1.finishedHead == cv2.finishedHead && cv1.BlockId(cv1.finishedHead) == cv2.BlockId(cv2.finishedHead)
}

// matchViews returns true if the two chain views are equivalent up until the
// specified block number. If the specified number is higher than one of the
// heads then false is returned.
func matchViews(cv1, cv2 *ChainView, number uint64) bool {
	if cv1 == nil || cv2 == nil {
		return false
	}
	if cv1.finishedHead < number || cv2.finishedHead < number {
		return false
	}
	if number == cv1.finishedHead || number == cv2.finishedHead {
		return cv1.BlockId(number) == cv2.BlockId(number)
	}
	return cv1.BlockHash(number) == cv2.BlockHash(number)
}

// extendNonCanonical checks whether the previously known reverse list of head
// hashes still ends with one that is canonical on the underlying blockchain.
// If necessary then it traverses further back on the header chain and adds
// more hashes to the list.
func (cv *ChainView) extendNonCanonical() bool {
	for {
		hash, number := cv.hashes[len(cv.hashes)-1], cv.finishedHead-uint64(len(cv.hashes)-1)
		if cv.chain.GetCanonicalHash(number) == hash {
			return true
		}
		if number == 0 {
			log.Error("Unknown genesis block hash found")
			return false
		}
		header := cv.chain.GetHeader(hash, number)
		if header == nil {
			log.Error("Header not found", "number", number, "hash", hash)
			return false
		}
		cv.hashes = append(cv.hashes, header.ParentHash)
	}
}

// blockHash returns the given block hash without doing the head number check.
func (cv *ChainView) blockHash(number uint64) common.Hash {
	if number+uint64(len(cv.hashes)) <= cv.finishedHead {
		hash := cv.chain.GetCanonicalHash(number)
		if !cv.extendNonCanonical() {
			return common.Hash{}
		}
		if number+uint64(len(cv.hashes)) <= cv.finishedHead {
			return hash
		}
	}
	return cv.hashes[cv.finishedHead-number]
}
