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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/tablewriter"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/triedb/database"
	"golang.org/x/sync/semaphore"
)

// inspector is used by the inner inspect function to coordinate across threads.
type inspector struct {
	triedb database.NodeDatabase
	root   common.Hash

	config *InspectConfig
	stats  map[common.Hash]*triestat
	m      sync.Mutex // protects stats

	sem *semaphore.Weighted
	wg  sync.WaitGroup
}

// InspectConfig is a set of options to control inspection and format the
// output. TopN will print the deepest min(len(results), N) storage tries.
// If path is set, a JSON version of the data will be written to a file at this path.
type InspectConfig struct {
	NoStorage bool
	TopN      int
	Path      string
}

// Inspect walks the trie with the given root and records the number and type of
// nodes at each depth. It works by recursively calling the inner inspect
// function on each child node.
func Inspect(triedb database.NodeDatabase, root common.Hash, config *InspectConfig) error {
	trie, err := New(TrieID(root), triedb)
	if err != nil {
		return fmt.Errorf("fail to open trie %s: %w", root, err)
	}
	if config == nil {
		config = &InspectConfig{}
	}
	in := inspector{
		triedb: triedb,
		root:   root,
		config: config,
		stats:  make(map[common.Hash]*triestat),
		sem:    semaphore.NewWeighted(int64(128)),
	}
	in.stats[root] = &triestat{}

	in.inspect(trie, trie.root, 0, []byte{}, in.stats[root])
	in.wg.Wait()
	if len(config.Path) > 0 {
		if err := in.writeJSON(); err != nil {
			log.Crit("Error during json encodeing", "error", err)
		}
	} else {
		in.displayResult()
	}
	return nil
}

// inspect is called recursively down the trie. At each level it records the
// node type encountered.
func (in *inspector) inspect(trie *Trie, n node, height uint32, path []byte, stat *triestat) {
	if n == nil {
		return
	}

	// Four types of nodes can be encountered:
	// - short: extend path with key, inspect single value.
	// - full: inspect all 17 children, spin up new threads when possible.
	// - hash: need to resolve node from disk, retry inspect on result.
	// - value: if account, begin inspecting storage trie.
	switch n := (n).(type) {
	case *shortNode:
		in.inspect(trie, n.Val, height+1, append(path, n.Key...), stat)
	case *fullNode:
		for idx, child := range n.Children {
			if child == nil {
				continue
			}
			childPath := append(path, byte(idx))
			if in.sem.TryAcquire(1) {
				in.wg.Add(1)
				go func() {
					in.inspect(trie, child, height+1, childPath, stat)
					in.wg.Done()
				}()
			} else {
				in.inspect(trie, child, height+1, childPath, stat)
			}
		}
	case hashNode:
		resolved, err := trie.resolveWithoutTrack(n, path)
		if err != nil {
			log.Error("Failed to resolve HashNode", "err", err, "trie", trie.Hash(), "height", height+1, "path", path)
			return
		}
		in.inspect(trie, resolved, height, path, stat)

		// Return early here so this level isn't recorded twice.
		return
	case valueNode:
		if !hasTerm(path) {
			break
		}
		var account types.StateAccount
		if err := rlp.Decode(bytes.NewReader(n), &account); err != nil {
			// Not an account value.
			break
		}
		if account.Root == (common.Hash{}) || account.Root == types.EmptyRootHash {
			// Account is empty, nothing further to inspect.
			break
		}

		// Start inspecting storage trie.
		if !in.config.NoStorage {
			owner := common.BytesToHash(hexToCompact(path))
			storage, err := New(StorageTrieID(in.root, owner, account.Root), in.triedb)
			if err != nil {
				log.Error("Failed to open account storage trie", "node", n, "error", err, "height", height, "path", common.Bytes2Hex(path))
				break
			}
			stat := &triestat{}

			in.m.Lock()
			in.stats[owner] = stat
			in.m.Unlock()

			in.wg.Add(1)
			go func() {
				in.inspect(storage, storage.root, 0, []byte{}, stat)
				in.wg.Done()
			}()
		}
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}

	// Record stats for current height
	stat.add(n, height)
}

// Display results prints out the inspect results.
func (in *inspector) displayResult() {
	fmt.Println("Results for trie", in.root)
	in.stats[in.root].display("Accounts trie")
	fmt.Println("===")
	fmt.Println()

	if !in.config.NoStorage {
		// Sort stats by max node depth.
		keys, stats := sortedTriestat(in.stats).sort()

		fmt.Println("Results for top storage tries")
		for i := range keys[0:min(in.config.TopN, len(keys))] {
			fmt.Printf("%d: %s\n", i+1, keys[i])
			stats[i].display("storage trie")
		}
	}
}

func (in *inspector) writeJSON() error {
	file, err := os.OpenFile(in.config.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)

	accountTrie := newJsonStat(in.stats[in.root], "account trie")
	if err := enc.Encode(accountTrie); err != nil {
		return err
	}
	if !in.config.NoStorage {
		// Sort stats by max node depth.
		keys, stats := sortedTriestat(in.stats).sort()
		for i := range keys[0:min(in.config.TopN, len(keys))] {
			storageTrie := newJsonStat(stats[i], fmt.Sprintf("%x", keys[i]))
			if err := enc.Encode(storageTrie); err != nil {
				return err
			}
		}
	}
	return nil
}

// triestat tracks the type and count of trie nodes at each level in the trie.
//
// Note: theoretically it is possible to have up to 64 trie level. Since it is
// unlikely to encounter such a large trie, the stats are capped at 16 levels to
// avoid substantial unneeded allocation.
type triestat struct {
	level [16]stat
}

// maxDepth iterates each level and finds the deepest level with at least one
// trie node.
func (s *triestat) maxDepth() int {
	depth := 0
	for i := range s.level {
		if s.level[i].short.Load() != 0 || s.level[i].full.Load() != 0 || s.level[i].value.Load() != 0 {
			depth = i
		}
	}
	return depth
}

// sortedTriestat implements sort().
type sortedTriestat map[common.Hash]*triestat

// sort returns the keys and triestats in decending order of the maximum trie
// node depth.
func (s sortedTriestat) sort() ([]common.Hash, []*triestat) {
	var (
		keys  = make([]common.Hash, 0, len(s))
		stats = make([]*triestat, 0, len(s))
	)
	for k := range s {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return s[keys[i]].maxDepth() > s[keys[j]].maxDepth() })
	for _, k := range keys {
		stats = append(stats, s[k])
	}
	return keys, stats
}

// add increases the node count by one for the specified node type and depth.
func (s *triestat) add(n node, d uint32) {
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

// stat is a specific level's count of each node type.
type stat struct {
	short atomic.Uint64
	full  atomic.Uint64
	value atomic.Uint64
}

// empty is a helper that returns whether there are any trie nodes at the level.
func (s *stat) empty() bool {
	if s.full.Load() == 0 && s.short.Load() == 0 && s.value.Load() == 0 {
		return true
	}
	return false
}

// load is a helper that loads each node type's value.
func (s *stat) load() (uint64, uint64, uint64) {
	return s.short.Load(), s.full.Load(), s.value.Load()
}

// add is a helper that adds two level's stats together.
func (s *stat) add(other *stat) *stat {
	s.short.Add(other.short.Load())
	s.full.Add(other.full.Load())
	s.value.Add(other.value.Load())
	return s
}

// display will print a table displaying the trie's node statistics.
func (s *triestat) display(title string) {
	// Shorten title if too long.
	if len(title) > 32 {
		title = title[0:8] + "..." + title[len(title)-8:]
	}

	b := new(strings.Builder)
	table := tablewriter.NewWriter(b)
	table.SetHeader([]string{title, "Level", "Short Nodes", "Full Node", "Value Node"})

	stat := &stat{}
	for i := range s.level {
		if s.level[i].empty() {
			break
		}
		short, full, value := s.level[i].load()
		table.AppendBulk([][]string{{"-", fmt.Sprint(i), fmt.Sprint(short), fmt.Sprint(full), fmt.Sprint(value)}})
		stat.add(&s.level[i])
	}
	short, full, value := stat.load()
	table.SetFooter([]string{"Total", "", fmt.Sprint(short), fmt.Sprint(full), fmt.Sprint(value)})
	table.Render()
	fmt.Print(b.String())
	fmt.Println("Max depth", s.maxDepth())
	fmt.Println()
}

type jsonLevel struct {
	Short uint64
	Full  uint64
	Value uint64
}

type jsonStat struct {
	Name    string
	Levels  []jsonLevel
	Summary jsonLevel
}

func newJsonStat(s *triestat, name string) *jsonStat {
	ret := jsonStat{Name: name, Summary: jsonLevel{}}
	for i := 0; i < len(s.level); i++ {
		// only count non-empty levels
		if s.level[i].empty() {
			continue
		}
		level := jsonLevel{
			Short: s.level[i].short.Load(),
			Full:  s.level[i].full.Load(),
			Value: s.level[i].value.Load(),
		}
		ret.Summary.Full += level.Full
		ret.Summary.Short += level.Short
		ret.Summary.Value += level.Value
		ret.Levels = append(ret.Levels, level)
	}
	return &ret
}
