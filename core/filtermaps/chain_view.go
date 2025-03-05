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

// chainView represents an immutable view of a chain with a block hash, a
// block id and a set of receipts associated to each block number. Block id
// can be any unique identifier of the blocks.
// Note that id and receipts are expected to be available up to headNumber
// while the canonical block hash is only expected up to headNumber-1 so that
// it can be implemented by the block builder while the processed head hash
// is not known yet.
type chainView interface {
	headNumber() uint64
	getBlockHash(number uint64) common.Hash
	getBlockId(number uint64) common.Hash
	getReceipts(number uint64) types.Receipts
}

// equalViews returns true if the two chain views are equivalent.
func equalViews(cv1, cv2 chainView) bool {
	if cv1 == nil || cv2 == nil {
		return false
	}
	head1, head2 := cv1.headNumber(), cv2.headNumber()
	return head1 == head2 && cv1.getBlockId(head1) == cv2.getBlockId(head2)
}

// matchViews returns true if the two chain views are equivalent up until the
// specified block number. If the specified number is higher than one of the
// heads then false is returned.
func matchViews(cv1, cv2 chainView, number uint64) bool {
	if cv1 == nil || cv2 == nil {
		return false
	}
	head1 := cv1.headNumber()
	if head1 < number {
		return false
	}
	head2 := cv2.headNumber()
	if head2 < number {
		return false
	}
	if number == head1 || number == head2 {
		return cv1.getBlockId(number) == cv2.getBlockId(number)
	}
	return cv1.getBlockHash(number) == cv2.getBlockHash(number)
}

// blockchain defines functions required by the FilterMaps log indexer.
type blockchain interface {
	GetHeader(hash common.Hash, number uint64) *types.Header
	GetCanonicalHash(number uint64) common.Hash
	GetReceiptsByHash(hash common.Hash) types.Receipts
}

// StoredChainView implements chainView based on a given blockchain.
// Note that the view's head does not have to be the current canonical head
// of the underlying blockchain, it should only possess the block headers
// and receipts up until the expected chain view head.
// Also note that this implementation uses the canonical block hash as block
// id which works as long as the log index structure is not hashed into the
// block headers. Starting from the fork that hashes the log index to the
// block the id needs to be based on a set of fields that exactly defines the
// block but does not include the log index root itself.
type StoredChainView struct {
	chain  blockchain
	head   uint64
	hashes []common.Hash // block hashes starting backwards from headNumber until first canonical hash
}

// NewStoredChainView creates a new StoredChainView.
func NewStoredChainView(chain blockchain, number uint64, hash common.Hash) *StoredChainView {
	cv := &StoredChainView{
		chain:  chain,
		head:   number,
		hashes: []common.Hash{hash},
	}
	cv.extendNonCanonical()
	return cv
}

// headNumber implements chainView.
func (cv *StoredChainView) headNumber() uint64 {
	return cv.head
}

// getBlockHash implements chainView.
func (cv *StoredChainView) getBlockHash(number uint64) common.Hash {
	if number >= cv.head {
		panic("invalid block number")
	}
	return cv.blockHash(number)
}

// getBlockId implements chainView.
func (cv *StoredChainView) getBlockId(number uint64) common.Hash {
	if number > cv.head {
		panic("invalid block number")
	}
	return cv.blockHash(number)
}

// getReceipts implements chainView.
func (cv *StoredChainView) getReceipts(number uint64) types.Receipts {
	if number > cv.head {
		panic("invalid block number")
	}
	return cv.chain.GetReceiptsByHash(cv.blockHash(number))
}

// extendNonCanonical checks whether the previously known reverse list of head
// hashes still ends with one that is canonical on the underlying blockchain.
// If necessary then it traverses further back on the header chain and adds
// more hashes to the list.
func (cv *StoredChainView) extendNonCanonical() bool {
	for {
		hash, number := cv.hashes[len(cv.hashes)-1], cv.head-uint64(len(cv.hashes)-1)
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
func (cv *StoredChainView) blockHash(number uint64) common.Hash {
	if number+uint64(len(cv.hashes)) <= cv.head {
		hash := cv.chain.GetCanonicalHash(number)
		if !cv.extendNonCanonical() {
			return common.Hash{}
		}
		if number+uint64(len(cv.hashes)) <= cv.head {
			return hash
		}
	}
	return cv.hashes[cv.head-number]
}

// limitedChainView wraps a chainView and truncates it at a given head number.
type limitedChainView struct {
	parent chainView
	head   uint64
}

// newLimitedChainView returns a truncated view of the given parent.
func newLimitedChainView(parent chainView, headNumber uint64) chainView {
	if headNumber >= parent.headNumber() {
		return parent
	}
	return &limitedChainView{
		parent: parent,
		head:   headNumber,
	}
}

// headNumber implements chainView.
func (cv *limitedChainView) headNumber() uint64 {
	return cv.head
}

// getBlockHash implements chainView.
func (cv *limitedChainView) getBlockHash(number uint64) common.Hash {
	if number >= cv.head {
		panic("invalid block number")
	}
	return cv.parent.getBlockHash(number)
}

// getBlockId implements chainView.
func (cv *limitedChainView) getBlockId(number uint64) common.Hash {
	if number > cv.head {
		panic("invalid block number")
	}
	return cv.parent.getBlockId(number)
}

// getReceipts implements chainView.
func (cv *limitedChainView) getReceipts(number uint64) types.Receipts {
	if number > cv.head {
		panic("invalid block number")
	}
	return cv.parent.getReceipts(number)
}
