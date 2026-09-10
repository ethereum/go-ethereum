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

package types

import (
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// IndexBuilder accumulates IndexEntry values and computes a Merkle root.
type IndexBuilder struct {
	entries []IndexEntry
}

func NewIndexBuilder() *IndexBuilder {
	return &IndexBuilder{}
}

// AddBlockEntries adds entries for all logs in a block's receipts.
func (b *IndexBuilder) AddBlockEntries(blockNumber uint64, receipts Receipts) {
	for txIdx, receipt := range receipts {
		// Transaction entry
		b.entries = append(b.entries, EncodeEntry(EntryTypeTransaction, receipt.TxHash, blockNumber, uint32(txIdx), 0))
		for logIdx, log := range receipt.Logs {
			// Address entry
			addrHash := common.BytesToHash(log.Address.Bytes())
			b.entries = append(b.entries, EncodeEntry(EntryTypeLogAddress, addrHash, blockNumber, uint32(txIdx), uint32(logIdx)))
			// Topic entries
			for topicIdx, topic := range log.Topics {
				if topicIdx < 4 {
					b.entries = append(b.entries, EncodeEntry(EntryType(int(EntryTypeLogTopic0)+topicIdx), topic, blockNumber, uint32(txIdx), uint32(logIdx)))
				}
			}
		}
	}
}

// Entries returns the accumulated entries slice (unsorted). Callers that need
// sorted entries should call Build() or sort the returned slice themselves.
func (b *IndexBuilder) Entries() []IndexEntry {
	return b.entries
}

// Build sorts entries and returns a simple hash.
func (b *IndexBuilder) Build() common.Hash {
	sort.Slice(b.entries, func(i, j int) bool {
		return CompareEntries(b.entries[i], b.entries[j]) < 0
	})
	// Simple hash: Keccak256 of all concatenated entries
	var buf []byte
	for _, e := range b.entries {
		buf = append(buf, e[:]...)
	}
	return crypto.Keccak256Hash(buf)
}
