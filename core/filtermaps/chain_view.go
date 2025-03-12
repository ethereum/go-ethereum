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
	chain      blockchain
	headNumber uint64
	hashes     []common.Hash // block hashes starting backwards from headNumber until first canonical hash
}

// NewChainView creates a new ChainView.
func NewChainView(chain blockchain, number uint64, hash common.Hash) *ChainView {
	cv := &ChainView{
		chain:      chain,
		headNumber: number,
		hashes:     []common.Hash{hash},
	}
	cv.extendNonCanonical()
	return cv
}

// getBlockHash returns the block hash belonging to the given block number.
// Note that the hash of the head block is not returned because ChainView might
// represent a view where the head block is currently being created.
func (cv *ChainView) getBlockHash(number uint64) common.Hash {
	if number >= cv.headNumber {
		panic("invalid block number")
	}
	return cv.blockHash(number)
}

// getBlockId returns the unique block id belonging to the given block number.
// Note that it is currently equal to the block hash. In the future it might
// be a different id for future blocks if the log index root becomes part of
// consensus and therefore rendering the index with the new head will happen
// before the hash of that new head is available.
func (cv *ChainView) getBlockId(number uint64) common.Hash {
	if number > cv.headNumber {
		panic("invalid block number")
	}
	return cv.blockHash(number)
}

// getReceipts returns the set of receipts belonging to the block at the given
// block number.
func (cv *ChainView) getReceipts(number uint64) types.Receipts {
	if number > cv.headNumber {
		panic("invalid block number")
	}
	return cv.chain.GetReceiptsByHash(cv.blockHash(number))
}

// limitedView returns a new chain view that is a truncated version of the parent view.
func (cv *ChainView) limitedView(newHead uint64) *ChainView {
	if newHead >= cv.headNumber {
		return cv
	}
	return NewChainView(cv.chain, newHead, cv.blockHash(newHead))
}

// equalViews returns true if the two chain views are equivalent.
func equalViews(cv1, cv2 *ChainView) bool {
	if cv1 == nil || cv2 == nil {
		return false
	}
	return cv1.headNumber == cv2.headNumber && cv1.getBlockId(cv1.headNumber) == cv2.getBlockId(cv2.headNumber)
}

// matchViews returns true if the two chain views are equivalent up until the
// specified block number. If the specified number is higher than one of the
// heads then false is returned.
func matchViews(cv1, cv2 *ChainView, number uint64) bool {
	if cv1 == nil || cv2 == nil {
		return false
	}
	if cv1.headNumber < number || cv2.headNumber < number {
		return false
	}
	if number == cv1.headNumber || number == cv2.headNumber {
		return cv1.getBlockId(number) == cv2.getBlockId(number)
	}
	return cv1.getBlockHash(number) == cv2.getBlockHash(number)
}

// extendNonCanonical checks whether the previously known reverse list of head
// hashes still ends with one that is canonical on the underlying blockchain.
// If necessary then it traverses further back on the header chain and adds
// more hashes to the list.
func (cv *ChainView) extendNonCanonical() bool {
	for {
		hash, number := cv.hashes[len(cv.hashes)-1], cv.headNumber-uint64(len(cv.hashes)-1)
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
	if number+uint64(len(cv.hashes)) <= cv.headNumber {
		hash := cv.chain.GetCanonicalHash(number)
		if !cv.extendNonCanonical() {
			return common.Hash{}
		}
		if number+uint64(len(cv.hashes)) <= cv.headNumber {
			return hash
		}
	}
	return cv.hashes[cv.headNumber-number]
}
