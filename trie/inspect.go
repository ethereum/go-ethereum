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
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/tablewriter"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/triedb/database"
	"golang.org/x/sync/semaphore"
)

type inspector struct {
	triedb database.NodeDatabase
	trie   *Trie
	root   common.Hash

	storage bool
	stats   map[common.Hash]*triestat
	m       sync.Mutex

	sem *semaphore.Weighted
	wg  sync.WaitGroup
}

type triestat struct {
	depth [15]stat
}

func (s *triestat) maxDepth() int {
	depth := 0
	for i := range s.depth {
		if s.depth[i].short.Load() != 0 || s.depth[i].full.Load() != 0 || s.depth[i].value.Load() != 0 {
			depth = i
		}
	}
	return depth
}

type trieStatByDepth map[common.Hash]*triestat

func (s trieStatByDepth) sort() ([]common.Hash, []*triestat) {
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

func (s *triestat) add(n node, d uint32) {
	switch (n).(type) {
	case *shortNode:
		s.depth[d].short.Add(1)
	case *fullNode:
		s.depth[d].full.Add(1)
	case valueNode:
		s.depth[d].value.Add(1)
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
}

type stat struct {
	short atomic.Uint64
	full  atomic.Uint64
	value atomic.Uint64
}

func (s *stat) empty() bool {
	if s.full.Load() == 0 && s.short.Load() == 0 && s.value.Load() == 0 {
		return true
	}
	return false
}

func (s *stat) load() (uint64, uint64, uint64) {
	return s.short.Load(), s.full.Load(), s.value.Load()
}

func (s *stat) add(other *stat) *stat {
	s.short.Add(other.short.Load())
	s.full.Add(other.full.Load())
	s.value.Add(other.value.Load())
	return s
}

func InspectTrie(triedb database.NodeDatabase, root common.Hash, storage bool) error {
	return inspectTrie(triedb, root, storage)
}

func inspectTrie(triedb database.NodeDatabase, root common.Hash, storage bool) error {
	trie, err := New(TrieID(root), triedb)
	if err != nil {
		return fmt.Errorf("fail to open trie %s: %w", root, err)
	}
	in := inspector{
		triedb:  triedb,
		trie:    trie,
		root:    root,
		storage: storage,
		stats:   make(map[common.Hash]*triestat),
		sem:     semaphore.NewWeighted(int64(128)),
	}
	in.stats[root] = &triestat{}

	in.inspect(trie.root, 0, []byte{}, in.stats[root])
	in.wg.Wait()
	in.DisplayResult()
	return nil
}

func (in *inspector) inspect(n node, height uint32, path []byte, stat *triestat) {
	if n == nil {
		return
	}

	switch n := (n).(type) {
	case *shortNode:
		in.inspect(n.Val, height, append(path, n.Key...), stat)
	case *fullNode:
		for idx, child := range n.Children {
			if child == nil {
				continue
			}
			childPath := append(path, byte(idx))
			if in.sem.TryAcquire(1) {
				in.wg.Add(1)
				go func() {
					in.inspect(child, height+1, slices.Clone(childPath), stat)
					in.wg.Done()
				}()
			} else {
				in.inspect(child, height+1, childPath, stat)
			}
		}
	case hashNode:
		resolved, err := in.trie.resolveWithoutTrack(n, path)
		if err != nil {
			fmt.Printf("Resolve HashNode error: %v, TrieRoot: %v, Height: %v, Path: %v\n", err, in.trie.Hash().String(), height+1, path)
			return
		}
		in.inspect(resolved, height, path, stat)
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
		// TODO: update for 7702
		// if common.BytesToHash(account.CodeHash) == types.EmptyCodeHash {
		// 	inspect.eoaAccountNums.Add(1)
		// }
		if account.Root == (common.Hash{}) || account.Root == types.EmptyRootHash {
			break
		}

		// Start inspecting storage trie.
		if in.storage {
			owner := common.BytesToHash(hexToCompact(path))
			storage, err := New(StorageTrieID(in.root, owner, account.Root), in.triedb)
			if err != nil {
				fmt.Printf("New contract trie node: %v, error: %v, Height: %v, Path: %v\n", n, err, height, path)
				break
			}
			// contractTrie.opTracer.reset()
			stat := &triestat{}

			in.m.Lock()
			in.stats[owner] = stat
			in.m.Unlock()

			in.wg.Add(1)
			go func() {
				in.inspect(storage.root, 0, []byte{}, stat)
				in.wg.Done()
			}()
		}
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}

	// Record stats for current height
	stat.add(n, height)
}

func (s *triestat) display(title string) {
	// Shorten title if too long.
	if len(title) > 32 {
		title = title[0:8] + "..." + title[len(title)-8:len(title)]
	}

	b := new(strings.Builder)
	table := tablewriter.NewWriter(b)
	table.SetHeader([]string{title, "Level", "Short Nodes", "Full Node", "Value Node"})
	table.SetAlignment(1)

	stat := &stat{}
	for i := range s.depth {
		if s.depth[i].empty() {
			break
		}
		short, full, value := s.depth[i].load()
		table.Append([]string{"-", fmt.Sprint(i), fmt.Sprint(short), fmt.Sprint(full), fmt.Sprint(value)})
		stat.add(&s.depth[i])
	}
	short, full, value := stat.load()
	table.SetFooter([]string{"Total", "", fmt.Sprint(short), fmt.Sprint(full), fmt.Sprint(value)})
	table.Render()
	fmt.Print(b.String())
	fmt.Println("Max depth", s.maxDepth())
	fmt.Println()
}

func (in *inspector) DisplayResult() {
	fmt.Println("Results for trie", in.root)
	in.stats[in.root].display("Accounts trie")

	fmt.Println("===")
	fmt.Println()
	if in.storage {
		fmt.Println("Results for top storage tries")
		keys, stats := trieStatByDepth(in.stats).sort()
		for i := range keys[0:min(10, len(keys))] {
			fmt.Printf("%d: %s\n", i+1, keys[i])
			stats[i].display("storage trie")
		}
	}
}
