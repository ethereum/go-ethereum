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

package trie

import (
	"fmt"
	"sync/atomic"
)

const trieStatLevels = 16

// LevelStats tracks the type and count of trie nodes at each level in a trie.
//
// Note: theoretically it is possible to have up to 64 trie levels, but
// LevelStats supports exactly 16 levels and panics on deeper paths.
type LevelStats struct {
	level [trieStatLevels]stat
}

// NewLevelStats creates an empty trie statistics collector.
func NewLevelStats() *LevelStats {
	return &LevelStats{}
}

// MaxDepth iterates each level and finds the deepest level with at least one
// trie node.
func (s *LevelStats) MaxDepth() int {
	depth := 0
	for i := range s.level {
		if s.level[i].short.Load() != 0 || s.level[i].full.Load() != 0 || s.level[i].value.Load() != 0 {
			depth = i
		}
	}
	return depth
}

// add increases the node count by one for the specified node type and depth.
func (s *LevelStats) add(n node, depth uint32) {
	d := int(depth)
	switch (n).(type) {
	case *shortNode:
		s.level[d].short.Add(1)
	case *fullNode:
		s.level[d].full.Add(1)
	case valueNode:
		s.level[d].value.Add(1)
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
}

// addSize increases the raw byte-size tally at the specified depth.
func (s *LevelStats) addSize(depth uint32, size uint64) {
	s.level[depth].size.Add(size)
}

// AddLeaf records a leaf depth. Witness collection reuses the value-node bucket
// for leaf accounting. It panics if the depth is outside [0, 15].
func (s *LevelStats) AddLeaf(depth int) {
	s.level[depth].value.Add(1)
}

// LeafDepths returns leaf counts grouped by depth.
func (s *LevelStats) LeafDepths() [trieStatLevels]int64 {
	var leaves [trieStatLevels]int64
	for i := range s.level {
		leaves[i] = int64(s.level[i].value.Load())
	}
	return leaves
}

// stat is a specific level's count of each node type.
type stat struct {
	short atomic.Uint64
	full  atomic.Uint64
	value atomic.Uint64
	size  atomic.Uint64
}

// empty is a helper that returns whether there are any trie nodes at the level.
func (s *stat) empty() bool {
	if s.full.Load() == 0 && s.short.Load() == 0 && s.value.Load() == 0 && s.size.Load() == 0 {
		return true
	}
	return false
}

// load is a helper that loads each node type's value.
func (s *stat) load() (uint64, uint64, uint64, uint64) {
	return s.short.Load(), s.full.Load(), s.value.Load(), s.size.Load()
}

// add is a helper that adds two level's stats together.
func (s *stat) add(other *stat) *stat {
	s.short.Add(other.short.Load())
	s.full.Add(other.full.Load())
	s.value.Add(other.value.Load())
	s.size.Add(other.size.Load())
	return s
}
