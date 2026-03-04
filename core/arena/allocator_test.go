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

package arena

import (
	"math/big"
	"testing"
	"unsafe"
)

// testStruct is a struct with mixed field types for alignment testing.
type testStruct struct {
	A uint64
	B uint32
	C byte
	D *big.Int
}

func TestBumpAllocatorAlignment(t *testing.T) {
	slab := make([]byte, 4096)
	alloc := NewBumpAllocator(slab)

	// Allocate a byte, then a uint64 — the uint64 must be properly aligned.
	alloc.RawAlloc(1, 1) // 1 byte, align 1

	var zero uint64
	ptr := alloc.RawAlloc(unsafe.Sizeof(zero), unsafe.Alignof(zero))
	addr := uintptr(ptr)
	if addr%unsafe.Alignof(zero) != 0 {
		t.Fatalf("uint64 pointer %x not aligned to %d", addr, unsafe.Alignof(zero))
	}
}

func TestBumpAllocatorZeroing(t *testing.T) {
	slab := make([]byte, 4096)
	alloc := NewBumpAllocator(slab)

	// Dirty the slab.
	for i := range slab {
		slab[i] = 0xFF
	}

	// Allocate and verify zeroed.
	ptr := alloc.RawAlloc(64, 1)
	data := unsafe.Slice((*byte)(ptr), 64)
	for i, b := range data {
		if b != 0 {
			t.Fatalf("byte %d not zeroed: got %x", i, b)
		}
	}
}

func TestBumpAllocatorMultiSlab(t *testing.T) {
	slab := make([]byte, 32)
	alloc := NewBumpAllocator(slab)

	// Allocation bigger than first slab triggers a new slab.
	alloc.RawAlloc(64, 1)
	if alloc.SlabCount() != 2 {
		t.Fatalf("expected 2 slabs, got %d", alloc.SlabCount())
	}
}

func TestBumpAllocatorTotalCap(t *testing.T) {
	slab := make([]byte, 32)
	alloc := NewBumpAllocator(slab)
	alloc.maxTotal = 128 // low cap for testing

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on total cap exceeded, got nil")
		}
	}()
	// First extra slab (32 bytes) → total 64, ok.
	alloc.RawAlloc(64, 1)
	// Second extra slab → total would exceed 128.
	alloc.RawAlloc(128, 1)
}

func TestBumpAllocatorReset(t *testing.T) {
	slab := make([]byte, 256)
	alloc := NewBumpAllocator(slab)

	alloc.RawAlloc(128, 1)
	if alloc.Used() == 0 {
		t.Fatal("expected non-zero Used after alloc")
	}

	alloc.Reset()
	if alloc.Used() != 0 {
		t.Fatalf("expected 0 Used after Reset, got %d", alloc.Used())
	}
	if alloc.Remaining() != 256 {
		t.Fatalf("expected 256 Remaining after Reset, got %d", alloc.Remaining())
	}

	// Should be able to allocate again after Reset.
	alloc.RawAlloc(128, 1)
}

func TestNewGenericHeap(t *testing.T) {
	alloc := &HeapAllocator{}

	v := New[uint64](alloc)
	if *v != 0 {
		t.Fatalf("expected zero value, got %d", *v)
	}
	*v = 42
	if *v != 42 {
		t.Fatal("heap-allocated uint64 not writable")
	}

	s := New[testStruct](alloc)
	if s.A != 0 || s.B != 0 || s.C != 0 || s.D != nil {
		t.Fatal("expected zero struct")
	}
	s.A = 1
	s.D = big.NewInt(99)
}

func TestNewGenericBump(t *testing.T) {
	slab := make([]byte, 4096)
	alloc := NewBumpAllocator(slab)

	v := New[uint64](alloc)
	if *v != 0 {
		t.Fatalf("expected zero value, got %d", *v)
	}
	*v = 42
	if *v != 42 {
		t.Fatal("bump-allocated uint64 not writable")
	}

	s := New[testStruct](alloc)
	if s.A != 0 || s.B != 0 || s.C != 0 || s.D != nil {
		t.Fatal("expected zero struct")
	}
	s.A = 1

	// Verify alignment.
	addr := uintptr(unsafe.Pointer(s))
	if addr%unsafe.Alignof(testStruct{}) != 0 {
		t.Fatalf("struct pointer %x not aligned to %d", addr, unsafe.Alignof(testStruct{}))
	}
}

func TestMakeSliceHeap(t *testing.T) {
	alloc := &HeapAllocator{}

	s := MakeSlice[uint64](alloc, 3, 8)
	if len(s) != 3 || cap(s) != 8 {
		t.Fatalf("unexpected len/cap: %d/%d", len(s), cap(s))
	}
	s[0] = 10
	s[1] = 20
	s[2] = 30
	if s[0] != 10 || s[1] != 20 || s[2] != 30 {
		t.Fatal("heap slice not writable")
	}
}

func TestMakeSliceBump(t *testing.T) {
	slab := make([]byte, 4096)
	alloc := NewBumpAllocator(slab)

	s := MakeSlice[uint64](alloc, 3, 8)
	if len(s) != 3 || cap(s) != 8 {
		t.Fatalf("unexpected len/cap: %d/%d", len(s), cap(s))
	}
	for i := range s {
		if s[i] != 0 {
			t.Fatalf("element %d not zeroed: %d", i, s[i])
		}
	}
	s[0] = 10
	s[1] = 20
	s[2] = 30
	if s[0] != 10 || s[1] != 20 || s[2] != 30 {
		t.Fatal("bump slice not writable")
	}
}

func TestHeapAllocatorReset(t *testing.T) {
	alloc := &HeapAllocator{}

	// RawAlloc pins data; Reset clears pins.
	alloc.RawAlloc(32, 1)
	alloc.RawAlloc(64, 1)
	if len(alloc.pins) != 2 {
		t.Fatalf("expected 2 pins, got %d", len(alloc.pins))
	}
	alloc.Reset()
	if len(alloc.pins) != 0 {
		t.Fatalf("expected 0 pins after Reset, got %d", len(alloc.pins))
	}
}

func TestDefaultHeap(t *testing.T) {
	// DefaultHeap should be usable as an Allocator.
	var a Allocator = DefaultHeap
	v := New[int](a)
	if *v != 0 {
		t.Fatal("expected zero from DefaultHeap")
	}
}

func TestBumpAllocatorNewBigInt(t *testing.T) {
	slab := make([]byte, 4096)
	alloc := NewBumpAllocator(slab)

	b := New[big.Int](alloc)
	if b.Sign() != 0 {
		t.Fatal("expected zero big.Int")
	}
	b.SetInt64(123456789)
	if b.Int64() != 123456789 {
		t.Fatalf("unexpected value: %d", b.Int64())
	}
}

func TestBumpResetAndReuse(t *testing.T) {
	slab := make([]byte, 256)
	alloc := NewBumpAllocator(slab)

	// Fill up most of the slab.
	for i := 0; i < 10; i++ {
		New[uint64](alloc)
	}
	used := alloc.Used()
	if used == 0 {
		t.Fatal("expected non-zero usage")
	}

	alloc.Reset()

	// After reset, should be able to allocate again from the start.
	for i := 0; i < 10; i++ {
		v := New[uint64](alloc)
		if *v != 0 {
			t.Fatalf("iteration %d: expected zero after reset, got %d", i, *v)
		}
	}
}
