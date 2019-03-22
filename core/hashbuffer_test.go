// Copyright 2019 The go-ethereum Authors
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

package core

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func init() {
	rand.Seed(time.Now().Unix())
}
func dummyHeader(n int, prev common.Hash) *types.Header {
	return &types.Header{
		Number:     new(big.Int).SetUint64(uint64(n)),
		ParentHash: prev,
	}
}

func uniqueHeader(n int, prev common.Hash) *types.Header {
	return &types.Header{
		Number:     new(big.Int).SetUint64(uint64(n)),
		ParentHash: prev,
		GasUsed:    rand.Uint64(),
	}
}

// TestConsecutiveHashes does inserts and rollback, but uses contiguous chains
func TestConsecutiveHashes(t *testing.T) {
	t.Parallel()
	// This should we swapped out very quickly
	hs := newHashBuffer(uniqueHeader(0, common.Hash{0xaa}))

	parent := common.Hash{}
	expected := make(map[int]common.Hash)

	assertEmpty := func(num int) {
		h, found := hs.Get(uint64(num))
		if found {
			t.Fatalf("expected %d not to be present", num)
		}
		if h != (common.Hash{}) {
			t.Fatalf("expected empty hash, got %x", h)
		}

	}

	// test 10 entries
	for n := 1; n < 10; n++ {
		h := dummyHeader(n, parent)
		hs.Set(h)
		if _, lh := hs.Newest(); lh != h.Hash() {
			t.Fatalf("num %d, wrong last hash, got %x exp %x", n, lh, h.Hash())
		}
		expected[n] = h.Hash()
		parent = h.Hash()
	}

	n, _ := hs.Oldest()
	if n != 1 {
		t.Fatalf("wrong oldest, expected %d got %d", 1, n)
	}

	for n := 1; n < 10; n++ {
		got, _ := hs.Get(uint64(n))
		exp := expected[n]
		if got != exp {
			t.Errorf("num %d, got %x expected %x", n, got, exp)
		}
	}
	assertEmpty(11)
	assertEmpty(0)

	// Write another 300, overflowing the storage
	for n := 10; n < hashBufferElems+10; n++ {
		h := dummyHeader(n, parent)
		hs.Set(h)
		if _, lh := hs.Newest(); lh != h.Hash() {
			t.Fatalf("num %d, wrong last hash, got %x exp %x", n, lh, h.Hash())
		}
		expected[n] = h.Hash()
		parent = h.Hash()
	}
	x, _ := hs.Oldest()
	if x != 10 {
		t.Fatalf("wrong oldest, expected %d got %d", 10, x)
	}

	// The last 256 should be available
	for n := hashBufferElems + 10 - 1; n > 10; n-- {
		got, found := hs.Get(uint64(n))
		exp := expected[n]
		if !found {
			t.Fatalf("expected %d to be found", n)
		}
		if got != exp {
			t.Fatalf("num %d, got %x expected %x", n, got, exp)
		}
	}
	// The older ones should be flushed
	for ; n > 0; n-- {
		got, found := hs.Get(n)
		if found {
			t.Fatalf("expected %d to be flushed", n)
		}
		if got != (common.Hash{}) {
			t.Fatalf("expected empty hash, got %x", got)
		}
	}
}

func TestHashStorageNonContiguous(t *testing.T) {

	hs := newHashBuffer(uniqueHeader(0, common.Hash{0xff}))
	parent := common.Hash{}
	expected := make(map[int]common.Hash)

	for n := 1; n < 10; n++ {
		hs.Set(uniqueHeader(n, parent))
		hdr := uniqueHeader(n, parent)
		hs.Set(hdr)
		parent = hdr.Hash()
		expected[n] = hdr.Hash()
	}
	n, _ := hs.Oldest()
	if n != 1 {
		t.Fatalf("wrong oldest, expected %d got %d", 1, n)
	}
	for n := 1; n < 10; n++ {
		got, _ := hs.Get(uint64(n))
		exp := expected[n]
		if got != exp {
			t.Errorf("num %d, got %x expected %x", n, got, exp)
		}
	}
	// 9 headers there [ 1,2,3,4a,5,6,7,8,9]
	// Setting a new in the middle should change it to
	// [ 1, 2, 3, 4b]
	{
		parent = expected[4]
		hdr := uniqueHeader(5, parent)
		hs.Set(hdr)
		if hs.headNumber != 5 {
			t.Fatalf("expected head num 5, got %d", hs.headNumber)
		}
		got, found := hs.Get(5)
		if !found {
			t.Fatalf("expected hash to exist")
		}
		if !bytes.Equal(got[:], hdr.Hash().Bytes()) {
			t.Fatalf("expected %x, got %x", hdr.Hash(), got)
		}
	}

	// Set a totally new header at 3, should clean out everything else
	{
		hdr := uniqueHeader(4, common.Hash{0x1})
		hs.Set(hdr)
		if hs.headNumber != 4 {
			t.Fatalf("expected head num 4, got %d", hs.headNumber)
		}
		if hs.tailNumber != 4 {
			t.Fatalf("expected head num 4, got %d", hs.tailNumber)
		}
		if _, exist := hs.Get(3); exist {
			t.Fatalf("should be gone: %d", 3)
		}
		if _, exist := hs.Get(5); exist {
			t.Fatalf("should be gone: %d", 5)
		}
		if _, exist := hs.Get(0); exist {
			t.Fatalf("should be gone: %d", 0)
		}
		// 4 should be there
		got, found := hs.Get(4)
		if !found {
			t.Fatalf("expected hash to exist")
		}
		if !bytes.Equal(got[:], hdr.Hash().Bytes()) {
			t.Fatalf("expected %x, got %x", hdr.Hash(), got)
		}
	}

}
