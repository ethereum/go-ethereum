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

package legacypool

import (
	"crypto/ecdsa"
	"maps"
	"math/big"
	"math/rand"
	"runtime"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// TestFeeBucket checks that buckets are ordered as the fee caps in them, are
// at most 1/32 wide above 64 and cover every uint256.
func TestFeeBucket(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	values := []*uint256.Int{
		uint256.NewInt(0), uint256.NewInt(1), uint256.NewInt(63), uint256.NewInt(64), uint256.NewInt(65),
		uint256.NewInt(127), uint256.NewInt(128), uint256.NewInt(1e9), uint256.NewInt(1e9 + 1),
		new(uint256.Int).Lsh(uint256.NewInt(1), 64), new(uint256.Int).Lsh(uint256.NewInt(1), 255),
		new(uint256.Int).SetAllOne(),
	}
	for i := 0; i < 2000; i++ {
		var v uint256.Int
		v.SetUint64(rng.Uint64())
		v.Rsh(&v, uint(rng.Intn(64)))
		v.Lsh(&v, uint(rng.Intn(200)))
		values = append(values, &v)
	}
	slices.SortFunc(values, func(a, b *uint256.Int) int { return a.Cmp(b) })

	for i, v := range values {
		k := feeBucket(v)
		if int(k) >= feeBucketCount {
			t.Fatalf("fee cap %v: bucket %d, beyond %d", v, k, feeBucketCount)
		}
		if i > 0 && k < feeBucket(values[i-1]) {
			t.Fatalf("fee cap %v: bucket %d below %d of the smaller %v", v, k, feeBucket(values[i-1]), values[i-1])
		}
		if v.LtUint64(64) && uint64(k) != v.Uint64() {
			t.Fatalf("fee cap %v: bucket %d, want its own", v, k)
		}
	}
	// Adjacent buckets' boundaries are contiguous across doublings.
	for _, v := range values {
		if v.LtUint64(64) || v.BitLen() == 256 {
			continue
		}
		var lo, width, next uint256.Int
		lo.Rsh(v, uint(v.BitLen()-6))
		lo.Lsh(&lo, uint(v.BitLen()-6)) // the bucket's lowest fee cap
		width.Lsh(uint256.NewInt(1), uint(v.BitLen()-6))
		next.Add(&lo, &width)
		if feeBucket(&lo) != feeBucket(v) || feeBucket(&next) != feeBucket(v)+1 {
			t.Fatalf("fee cap %v: bucket %d, lowest %v in %d, next %v in %d", v, feeBucket(v), &lo, feeBucket(&lo), &next, feeBucket(&next))
		}
	}
}

// TestHeadIndexAtLeast changes pending lists at random, through every list
// operation, and checks that atLeast returns exactly the accounts whose first
// transaction has a fee cap at least the one asked for.
func TestHeadIndexAtLeast(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	idx := newHeadIndex()
	key, _ := crypto.GenerateKey()
	lists := make(map[common.Address]*list)
	for i := 0; i < 200; i++ {
		addr := common.Address{byte(i), byte(i >> 8)}
		lists[addr] = newPendingList(addr, idx)
	}
	randomFee := func() *big.Int { return big.NewInt(rng.Int63n(1 << uint(1+rng.Intn(40)))) }
	for step := 0; step < 20000; step++ {
		var (
			addr = common.Address{byte(rng.Intn(200)), 0}
			l    = lists[addr]
			n    = uint64(l.Len())
		)
		if first := l.Flatten(); len(first) > 0 {
			n += first[0].Nonce()
		}
		switch rng.Intn(8) {
		case 0, 1, 2: // the next nonce, or a replacement
			nonce := n
			if n > 0 && rng.Intn(3) == 0 {
				nonce = n - 1 - uint64(rng.Intn(min(int(n), 3)))
			}
			tip := randomFee()
			l.Add(dynamicFeeTx(nonce, 21000, new(big.Int).Add(tip, randomFee()), tip, key), 0)
		case 3:
			l.Forward(n - uint64(rng.Intn(3)))
		case 4:
			if txs := l.Flatten(); len(txs) > 0 {
				l.Remove(txs[rng.Intn(len(txs))])
			}
		case 5:
			l.Cap(rng.Intn(3))
		case 6:
			l.Filter(uint256.NewInt(uint64(rng.Int63n(1<<45))), 30000)
		case 7:
			l.Ready(n)
		}
		if step%100 != 0 {
			continue
		}
		feeCap := uint256.NewInt(uint64(rng.Int63n(1 << uint(1+rng.Intn(40)))))
		got := make(map[common.Address]bool)
		idx.atLeast(feeCap, func(addr common.Address, l *list) {
			if got[addr] || lists[addr] != l {
				t.Fatalf("step %d: account %x returned twice or with another list", step, addr)
			}
			got[addr] = true
		})
		for addr, l := range lists {
			txs := l.Flatten()
			want := len(txs) > 0 && txs[0].GasFeeCapIntCmp(feeCap.ToBig()) >= 0
			if got[addr] != want {
				t.Fatalf("step %d, fee cap %v: account %x returned %v, want %v", step, feeCap, addr, got[addr], want)
			}
		}
	}
}

// TestHeadIndexAtLeastExtremes checks atLeast at the ends of the fee cap
// range, 0, 2^255 and the largest uint256, with accounts indexed at and
// around each.
func TestHeadIndexAtLeastExtremes(t *testing.T) {
	var (
		idx  = newHeadIndex()
		key  = func() *ecdsa.PrivateKey { k, _ := crypto.GenerateKey(); return k }()
		half = new(uint256.Int).Lsh(uint256.NewInt(1), 255)
		most = new(uint256.Int).SetAllOne()
		caps = []*uint256.Int{
			uint256.NewInt(0), uint256.NewInt(1), new(uint256.Int).SubUint64(half, 1), half,
			new(uint256.Int).AddUint64(half, 1), new(uint256.Int).SubUint64(most, 1), most,
		}
		feeCaps = make(map[common.Address]*uint256.Int)
	)
	for i, feeCap := range caps {
		addr := common.Address{byte(i)}
		l := newPendingList(addr, idx)
		l.txs.Put(dynamicFeeTx(0, 21000, feeCap.ToBig(), big.NewInt(0), key)) // fee caps beyond Add's cost check
		idx.update(addr, l)
		feeCaps[addr] = feeCap
	}
	for _, threshold := range append(caps, uint256.NewInt(2), new(uint256.Int).Rsh(most, 1)) {
		var got int
		idx.atLeast(threshold, func(addr common.Address, l *list) {
			if feeCaps[addr].Lt(threshold) {
				t.Errorf("at least %v: account with fee cap %v returned", threshold, feeCaps[addr])
			}
			got++
		})
		var want int
		for _, feeCap := range feeCaps {
			if !feeCap.Lt(threshold) {
				want++
			}
		}
		if got != want {
			t.Errorf("at least %v: %d accounts returned, want %d", threshold, got, want)
		}
	}
}

// TestHeadIndexCompaction checks that a bucket's map is rebuilt once it holds
// less than a quarter of its peak, keeping every account, and that a drained
// bucket doesn't keep the size it had.
func TestHeadIndexCompaction(t *testing.T) {
	key, _ := crypto.GenerateKey()
	full, empty := newList(true), newList(true)
	full.txs.Put(dynamicFeeTx(0, 21000, big.NewInt(1<<20), big.NewInt(0), key))
	k := feeBucket(uint256.NewInt(1 << 20))
	addr := func(a int) common.Address { return common.BigToAddress(big.NewInt(int64(a + 1))) }

	idx := newHeadIndex()
	for a := range 64 {
		idx.update(addr(a), full)
	}
	for a := 63; a >= 16; a-- {
		idx.update(addr(a), empty)
	}
	if b := idx.buckets[k]; b.peak != 64 || len(b.entries) != 16 {
		t.Fatalf("at a quarter of the peak: %d accounts, peak %d; want 16, 64", len(b.entries), b.peak)
	}
	idx.update(addr(15), empty)
	if b := idx.buckets[k]; b.peak != 15 || len(b.entries) != 15 {
		t.Fatalf("below a quarter of the peak: %d accounts, peak %d; want 15, 15", len(b.entries), b.peak)
	}
	for a := range 15 {
		if idx.bucket[addr(a)] != k || idx.buckets[k].entries[addr(a)].list != full {
			t.Fatalf("account %d lost in the rebuild", a)
		}
	}
	if idx.used[k/64]&(1<<(k%64)) == 0 {
		t.Fatal("rebuilt bucket not marked used")
	}
	for a := range 15 {
		idx.update(addr(a), empty)
	}
	if idx.buckets[k] != nil || idx.used[k/64]&(1<<(k%64)) != 0 {
		t.Fatal("empty bucket kept")
	}
	// maps.Clone copies a map at its size, so the bytes of a clone measure it:
	// a bucket drained from 4,096 accounts to one must copy as a small map.
	for a := range 4096 {
		idx.update(addr(a), full)
	}
	for a := 4095; a >= 1; a-- {
		idx.update(addr(a), empty)
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	clone := maps.Clone(idx.buckets[k].entries)
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(clone)
	if n := after.TotalAlloc - before.TotalAlloc; n > 64<<10 {
		t.Errorf("the drained bucket's map copies as %d bytes", n)
	}
}
