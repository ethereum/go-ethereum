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
	"fmt"
	"unsafe"
)

const (
	defaultSlabSize = 8 << 20 // 8 MiB per slab
	maxTotalSize    = 1 << 30 // 1 GiB total cap across all slabs
)

// BumpAllocator is a multi-slab bump/arena allocator. It sub-allocates from
// pre-allocated byte slabs, growing by adding new slabs when the current one
// is exhausted. It is not thread-safe. All allocated memory is freed at once
// via Reset.
//
// Zeroing is performed lazily at allocation time (not on Reset), using
// clear() which compiles to an optimized memclr intrinsic.
type BumpAllocator struct {
	slabs    [][]byte // all slabs (index 0 = first allocated)
	current  int      // index of the active slab in slabs
	offset   uintptr  // offset within the current slab
	slabSize int      // size of each new slab
	total    uintptr  // total bytes allocated across all slabs
	maxTotal uintptr  // maximum total bytes (DoS protection)
	peak     uintptr  // high-water mark of Used() across resets
}

// NewBumpAllocator creates a BumpAllocator with a single initial slab.
// The slab size and total cap use defaults (8 MiB per slab, 1 GiB cap).
func NewBumpAllocator(slab []byte) *BumpAllocator {
	return &BumpAllocator{
		slabs:    [][]byte{slab},
		slabSize: len(slab),
		total:    uintptr(len(slab)),
		maxTotal: maxTotalSize,
	}
}

// RawAlloc returns a pointer to a zeroed region of at least `size` bytes,
// aligned to `align`, from the current slab. If the current slab is
// exhausted, a new slab is allocated. Panics if the total cap is exceeded.
func (b *BumpAllocator) RawAlloc(size, align uintptr) unsafe.Pointer {
	slab := b.slabs[b.current]

	// Align the current offset up to the required alignment.
	aligned := (b.offset + align - 1) &^ (align - 1)
	end := aligned + size

	if end > uintptr(len(slab)) {
		// Current slab exhausted — try the next retained slab or allocate a new one.
		b.current++
		if b.current < len(b.slabs) {
			// Reuse a previously allocated slab.
			slab = b.slabs[b.current]
		} else {
			// Allocate a new slab, at least large enough for this request.
			newSize := b.slabSize
			if int(size) > newSize {
				newSize = int(size + align) // oversized allocation
			}
			if b.total+uintptr(newSize) > b.maxTotal {
				panic(fmt.Sprintf("arena: total allocation exceeds %d byte cap", b.maxTotal))
			}
			slab = make([]byte, newSize)
			b.slabs = append(b.slabs, slab)
			b.total += uintptr(newSize)
		}
		b.offset = 0
		aligned = 0 // offset 0 is always aligned
		end = size
	}

	// Zero the region using clear() which compiles to optimized memclr.
	clear(slab[aligned:end])

	b.offset = end
	return unsafe.Pointer(&slab[aligned])
}

// Reset rewinds the allocator to the first slab. Retained slabs are kept
// for reuse. Zeroing is deferred to the next RawAlloc call, so Reset is O(1).
func (b *BumpAllocator) Reset() {
	used := b.Used()
	if used > b.peak {
		b.peak = used
	}
	b.current = 0
	b.offset = 0
}

// Used returns the number of bytes currently allocated (across all active slabs).
func (b *BumpAllocator) Used() uintptr {
	var total uintptr
	for i := 0; i < b.current; i++ {
		total += uintptr(len(b.slabs[i]))
	}
	total += b.offset
	return total
}

// Remaining returns the number of bytes left in the current slab.
func (b *BumpAllocator) Remaining() uintptr {
	return uintptr(len(b.slabs[b.current])) - b.offset
}

// SlabCount returns the number of slabs allocated.
func (b *BumpAllocator) SlabCount() int {
	return len(b.slabs)
}

// TotalCapacity returns the total bytes across all slabs.
func (b *BumpAllocator) TotalCapacity() uintptr {
	return b.total
}

// Peak returns the high-water mark of Used() across all resets.
func (b *BumpAllocator) Peak() uintptr {
	// Check current usage too (might not have been Reset yet).
	if used := b.Used(); used > b.peak {
		return used
	}
	return b.peak
}
