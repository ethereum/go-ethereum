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

package stateless

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	accountTrieDepthAvg = metrics.NewRegisteredGauge("witness/trie/account/depth/avg", nil)
	accountTrieDepthMin = metrics.NewRegisteredGauge("witness/trie/account/depth/min", nil)
	accountTrieDepthMax = metrics.NewRegisteredGauge("witness/trie/account/depth/max", nil)

	storageTrieDepthAvg = metrics.NewRegisteredGauge("witness/trie/storage/depth/avg", nil)
	storageTrieDepthMin = metrics.NewRegisteredGauge("witness/trie/storage/depth/min", nil)
	storageTrieDepthMax = metrics.NewRegisteredGauge("witness/trie/storage/depth/max", nil)
)

// depthStats tracks min/avg/max statistics for trie access depths.
type depthStats struct {
	totalDepth int64
	samples    int64
	minDepth   int64
	maxDepth   int64
}

// newDepthStats creates a new depthStats with default values.
func newDepthStats() *depthStats {
	return &depthStats{minDepth: -1}
}

// add records a new depth sample.
func (d *depthStats) add(n int64) {
	if n < 0 {
		return
	}
	d.totalDepth += n
	d.samples++

	if d.minDepth == -1 || n < d.minDepth {
		d.minDepth = n
	}
	if n > d.maxDepth {
		d.maxDepth = n
	}
}

// report uploads the collected statistics into the provided gauges.
func (d *depthStats) report(maxGauge, minGauge, avgGauge *metrics.Gauge) {
	if d.samples == 0 {
		return
	}
	maxGauge.Update(d.maxDepth)
	minGauge.Update(d.minDepth)
	avgGauge.Update(d.totalDepth / d.samples)
}

// WitnessStats aggregates statistics for account and storage trie accesses.
type WitnessStats struct {
	accountTrie *depthStats
	storageTrie *depthStats
}

// NewWitnessStats creates a new WitnessStats collector.
func NewWitnessStats() *WitnessStats {
	return &WitnessStats{
		accountTrie: newDepthStats(),
		storageTrie: newDepthStats(),
	}
}

// isLeafNode checks if the given RLP-encoded node data represents a leaf node.
// In Ethereum's Modified Merkle Patricia Trie, a leaf node is identified by:
// - Having exactly 2 RLP list elements (for both shortNode and leafNode encodings)
// - The second element being a value (not a hash reference to another node)
func isLeafNode(nodeData []byte) bool {
	if len(nodeData) == 0 {
		return false
	}
	
	// Decode the RLP list
	var elems [][]byte
	if err := rlp.DecodeBytes(nodeData, &elems); err != nil {
		return false
	}
	
	// A leaf node in MPT has exactly 2 elements: [key, value]
	// An extension node also has 2 elements but the value is a hash (32 bytes)
	if len(elems) != 2 {
		return false // Branch nodes have 17 elements
	}
	
	// If the second element is 32 bytes, it's likely a hash reference (extension node)
	// Leaf nodes typically have values that are not exactly 32 bytes
	// However, this is not a perfect heuristic as values could be 32 bytes
	// A more accurate check would require checking the key's terminator flag
	
	// Check if the key has a terminator (indicates leaf node)
	// In compact encoding, the first nibble of the key indicates the node type
	if len(elems[0]) > 0 {
		// Get the first byte which contains the flags
		flags := elems[0][0]
		// Check if the terminator flag is set (bit 5)
		// Leaf nodes have the terminator flag set (0x20 or 0x30)
		return (flags & 0x20) != 0
	}
	
	return false
}

// Add records trie access depths from the given node paths.
// If `owner` is the zero hash, accesses are attributed to the account trie;
// otherwise, they are attributed to the storage trie of that account.
func (s *WitnessStats) Add(nodes map[string][]byte, owner common.Hash) {
	if owner == (common.Hash{}) {
		for path, nodeData := range nodes {
			// Only record depth for leaf nodes
			if isLeafNode(nodeData) {
				s.accountTrie.add(int64(len(path)))
			}
		}
	} else {
		for path, nodeData := range nodes {
			// Only record depth for leaf nodes
			if isLeafNode(nodeData) {
				s.storageTrie.add(int64(len(path)))
			}
		}
	}
}

// ReportMetrics reports the collected statistics to the global metrics registry.
func (s *WitnessStats) ReportMetrics() {
	s.accountTrie.report(accountTrieDepthMax, accountTrieDepthMin, accountTrieDepthAvg)
	s.storageTrie.report(storageTrieDepthMax, storageTrieDepthMin, storageTrieDepthAvg)
}
