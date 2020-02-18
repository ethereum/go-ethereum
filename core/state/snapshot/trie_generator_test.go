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
	"math/rand"
	"sync"
	"testing"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

func generateTrie(it AccountIterator, generatorFn trieGeneratorFn) common.Hash {
	var (
		in  = make(chan leaf)        // chan to pass leaves
		out = make(chan common.Hash) // chan to collect result
		wg  sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		generatorFn(in, out)
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
	rand.Seed(1338)
	// Stack three diff layers on top with various overlaps
	snaps.Update(common.HexToHash("0x02"), common.HexToHash("0x01"),
		randomAccountSet("0x11", "0x22", "0x33"), nil)
	// We call this once before the benchmark, so the creation of
	// sorted accountlists are not included in the results.
	head := snaps.Snapshot(common.HexToHash("0x02"))
	it := head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
	hash := generateTrie(it, AppendOnlyGenerate)
	if got, exp := hash, common.HexToHash("333a7c170a3d97bd53321d0f39b1a6b9a35b286ad2d3b3ced72ca339197c5dca"); exp != got {
		t.Fatalf("expected %x got %x", exp, got)
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
	rand.Seed(1337)
	// Stack three diff layers on top with various overlaps
	snaps.Update(common.HexToHash("0x02"), common.HexToHash("0x01"),
		randomAccountSet("0x11", "0x22", "0x33"), nil)
	// We call this once before the benchmark, so the creation of
	// sorted accountlists are not included in the results.
	head := snaps.Snapshot(common.HexToHash("0x02"))
	it := head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
	hash := generateTrie(it, AppendOnlyGenerate)
	if got, exp := hash, common.HexToHash("c9dd8a9602446bfcce27efbb0188a78761bf5473dd363f4ae2f17975a308344a"); exp != got {
		t.Fatalf("expected %x got %x", exp, got)
	}
}

// BenchmarkTrieGeneration/4K/standard-8         	     127	   9429425 ns/op	 6188077 B/op	   58026 allocs/op
// BenchmarkTrieGeneration/4K/pruning-8          	      72	  16544534 ns/op	 6617322 B/op	   55016 allocs/op
// BenchmarkTrieGeneration/4K/stack-8            	     159	   6452936 ns/op	 6308393 B/op	   12022 allocs/op
// BenchmarkTrieGeneration/10K/standard-8        	      50	  25025175 ns/op	17283703 B/op	  151023 allocs/op
// BenchmarkTrieGeneration/10K/pruning-8         	      28	  38141602 ns/op	16540254 B/op	  137520 allocs/op
// BenchmarkTrieGeneration/10K/stack-8           	      60	  18888649 ns/op	17557314 B/op	   30067 allocs/op
func BenchmarkTrieGeneration(b *testing.B) {
	// Get a fairly large trie
	// Create a custom account factory to recreate the same addresses
	makeAccounts := func(num int) map[common.Hash][]byte {
		accounts := make(map[common.Hash][]byte)
		for i := 0; i < num; i++ {
			h := common.Hash{}
			binary.BigEndian.PutUint64(h[:], uint64(i+1))
			accounts[h] = randomAccountWithSmall()
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
		b.Run("standard", func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			var got common.Hash
			for i := 0; i < b.N; i++ {
				it := head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
				got = generateTrie(it, StdGenerate)
			}
			b.StopTimer()
			if exp := common.HexToHash("fecc4e1fce05c888c8acc8baa2d7677a531714668b7a09b5ede6e3e110be266b"); got != exp{
				b.Fatalf("Error: got %x exp %x", got, exp)
			}
		})
		b.Run("pruning", func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			var got common.Hash
			for i := 0; i < b.N; i++ {
				it := head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
				got = generateTrie(it, PruneGenerate)
			}
			b.StopTimer()
			if exp := common.HexToHash("fecc4e1fce05c888c8acc8baa2d7677a531714668b7a09b5ede6e3e110be266b"); got != exp{
				b.Fatalf("Error: got %x exp %x", got, exp)
			}

		})
		b.Run("stack", func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			var got common.Hash
			for i := 0; i < b.N; i++ {
				it := head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
				got = generateTrie(it, StackGenerate)
			}
			b.StopTimer()
			if exp := common.HexToHash("fecc4e1fce05c888c8acc8baa2d7677a531714668b7a09b5ede6e3e110be266b"); got != exp{
				b.Fatalf("Error: got %x exp %x", got, exp)
			}

		})
	})
	b.Run("10K", func(b *testing.B) {
		// 4K accounts
		snaps.Update(common.HexToHash("0x02"), common.HexToHash("0x01"), makeAccounts(10000), nil)
		head := snaps.Snapshot(common.HexToHash("0x02"))
		// Call it once to make it create the lists before test starts
		head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
		b.Run("standard", func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				it := head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
				generateTrie(it, StdGenerate)
			}
		})
		b.Run("pruning", func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				it := head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
				generateTrie(it, PruneGenerate)
			}
		})
		b.Run("stack", func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				it := head.(*diffLayer).AccountIterator(common.HexToHash("0x00"))
				generateTrie(it, StackGenerate)
			}
		})
	})
}
