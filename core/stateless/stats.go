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
	"encoding/json"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/trie"
)

var accountTrieLeavesAtDepth [16]*metrics.Counter
var storageTrieLeavesAtDepth [16]*metrics.Counter

func init() {
	for i := 0; i < 16; i++ {
		accountTrieLeavesAtDepth[i] = metrics.NewRegisteredCounter("witness/trie/account/leaves/depth_"+strconv.Itoa(i), nil)
		storageTrieLeavesAtDepth[i] = metrics.NewRegisteredCounter("witness/trie/storage/leaves/depth_"+strconv.Itoa(i), nil)
	}
}

// WitnessStats aggregates statistics for account and storage trie accesses.
type WitnessStats struct {
	accountTrie *trie.LevelStats
	storageTrie *trie.LevelStats
}

// NewWitnessStats creates a new WitnessStats collector.
func NewWitnessStats() *WitnessStats {
	return &WitnessStats{
		accountTrie: trie.NewLevelStats(),
		storageTrie: trie.NewLevelStats(),
	}
}

func (s *WitnessStats) init() {
	if s.accountTrie == nil {
		s.accountTrie = trie.NewLevelStats()
	}
	if s.storageTrie == nil {
		s.storageTrie = trie.NewLevelStats()
	}
}

// Add records trie access depths from the given node paths.
// If `owner` is the zero hash, accesses are attributed to the account trie;
// otherwise, they are attributed to the storage trie of that account.
func (s *WitnessStats) Add(nodes map[string][]byte, owner common.Hash) {
	s.init()

	// Extract paths from the nodes map.
	paths := slices.Collect(maps.Keys(nodes))
	sort.Strings(paths)

	ownerStat := s.accountTrie
	if owner != (common.Hash{}) {
		ownerStat = s.storageTrie
	}

	for i, path := range paths {
		// If current path is a prefix of the next path, it's not a leaf.
		// The last path is always a leaf.
		if i == len(paths)-1 || !strings.HasPrefix(paths[i+1], paths[i]) {
			ownerStat.AddLeaf(len(path))
		}
	}
}

// ReportMetrics reports the collected statistics to the global metrics registry.
func (s *WitnessStats) ReportMetrics(blockNumber uint64) {
	s.init()

	accountTrieLeaves := s.accountTrie.LeafDepths()
	storageTrieLeaves := s.storageTrie.LeafDepths()

	// Encode the metrics as JSON for easier consumption.
	accountLeavesJSON, _ := json.Marshal(accountTrieLeaves)
	storageLeavesJSON, _ := json.Marshal(storageTrieLeaves)

	// Log account trie depth statistics.
	log.Info("Account trie depth stats", "block", blockNumber, "leavesAtDepth", string(accountLeavesJSON))
	log.Info("Storage trie depth stats", "block", blockNumber, "leavesAtDepth", string(storageLeavesJSON))

	for i := 0; i < len(accountTrieLeavesAtDepth); i++ {
		accountTrieLeavesAtDepth[i].Inc(accountTrieLeaves[i])
		storageTrieLeavesAtDepth[i].Inc(storageTrieLeaves[i])
	}
}
