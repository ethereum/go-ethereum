// Copyright 2020 The go-ethereum Authors
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

package snapshot

import (
	"encoding/binary"
	"sync"
	"testing"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

func generateTrie(it AccountIterator, generator *trieGenerator) common.Hash {
	var (
		in  = make(chan leaf)        // chan to pass leaves
		out = make(chan common.Hash) // chan to collect result
		wg  sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		generator.Generate3(in, out)
		wg.Done()
	}()
	// Feed leaves
	for it.Next() {
		in <- leaf{it.Hash(), it.Account()}
	}
	close(in)
	result := <-out
	wg.Wait()
	return result

}

func TestTrieGeneration(t *testing.T) {
	// Create an empty base layer and a snapshot tree out of it
	base := &diskLayer{
		diskdb: rawdb.NewMemoryDatabase(),
		root:   common.HexToHash("0x01"),
		cache:  fastcache.New(1024 * 500),
	}
	snaps := &Tree{
		layers: map[common.Hash]snapshot{
			base.root: base,
		},
	}
	// Stack three diff layers on top with various overlaps
	snaps.Update(common.HexToHash("0x02"), common.HexToHash("0x01"),
		randomAccountSet("0x11", "0x22", "0x33"), nil)
	// We call this once before the benchmark, so the creation of
	// sorted accountlists are not included in the results.
	head := snaps.Snapshot(common.HexToHash("0x02"))
	it := head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
	generator := &trieGenerator{}
	hash := generateTrie(it, generator)
	if exp, got := hash, common.HexToHash("807fbe7d4e4c62b80b1e7f682bb13ed409467df2a5903e5af44b88f6b08d0519"); exp != got {
		t.Fatalf("expected %v got %v", exp, got)
	}
}

func TestTrieGenerationAppendonly(t *testing.T) {
	// Create an empty base layer and a snapshot tree out of it
	base := &diskLayer{
		diskdb: rawdb.NewMemoryDatabase(),
		root:   common.HexToHash("0x01"),
		cache:  fastcache.New(1024 * 500),
	}
	snaps := &Tree{
		layers: map[common.Hash]snapshot{
			base.root: base,
		},
	}
	// Stack three diff layers on top with various overlaps
	snaps.Update(common.HexToHash("0x02"), common.HexToHash("0x01"),
		randomAccountSet("0x11", "0x22", "0x33"), nil)
	// We call this once before the benchmark, so the creation of
	// sorted accountlists are not included in the results.
	head := snaps.Snapshot(common.HexToHash("0x02"))
	it := head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
	generator := &trieGenerator{}
	hash := generateTrie(it, generator)
	if exp, got := hash, common.HexToHash("807fbe7d4e4c62b80b1e7f682bb13ed409467df2a5903e5af44b88f6b08d0519"); exp != got {
		t.Fatalf("expected %v got %v", exp, got)
	}
}

func BenchmarkTrieGeneration(b *testing.B) {
	// Get a fairly large trie
	// Create a custom account factory to recreate the same addresses
	makeAccounts := func(num int) map[common.Hash][]byte {
		accounts := make(map[common.Hash][]byte)
		for i := 0; i < num; i++ {
			h := common.Hash{}
			binary.BigEndian.PutUint64(h[:], uint64(i+1))
			accounts[h] = randomAccount()
		}
		return accounts
	}
	// Build up a large stack of snapshots
	base := &diskLayer{
		diskdb: rawdb.NewMemoryDatabase(),
		root:   common.HexToHash("0x01"),
		cache:  fastcache.New(1024 * 500),
	}
	snaps := &Tree{
		layers: map[common.Hash]snapshot{
			base.root: base,
		},
	}
	b.Run("4K", func(b *testing.B) {
		// 4K accounts
		snaps.Update(common.HexToHash("0x02"), common.HexToHash("0x01"), makeAccounts(4000), nil)
		head := snaps.Snapshot(common.HexToHash("0x02"))
		// Call it once to make it create the lists before test starts
		head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			it := head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
			generator := &trieGenerator{}
			generateTrie(it, generator)
		}
	})
	b.Run("10K", func(b *testing.B) {
		// 4K accounts
		snaps.Update(common.HexToHash("0x02"), common.HexToHash("0x01"), makeAccounts(10000), nil)
		head := snaps.Snapshot(common.HexToHash("0x02"))
		// Call it once to make it create the lists before test starts
		head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			it := head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
			generator := &trieGenerator{}
			generateTrie(it, generator)
		}
	})
}
