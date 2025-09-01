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
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/metrics"
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

// Add records trie access depths from the given node paths.
// If `owner` is the zero hash, accesses are attributed to the account trie;
// otherwise, they are attributed to the storage trie of that account.
func (s *WitnessStats) Add(nodes map[string][]byte, owner common.Hash) {
	// Extract paths from the nodes map
	paths := slices.Collect(maps.Keys(nodes))
	sort.Strings(paths)

	for i, path := range paths {
		// If current path is a prefix of the next path, it's not a leaf.
		// The last path is always a leaf.
		if i == len(paths)-1 || !strings.HasPrefix(paths[i+1], paths[i]) {
			if owner == (common.Hash{}) {
				s.accountTrie.add(int64(len(path)))
			} else {
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
