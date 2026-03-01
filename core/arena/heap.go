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

import "unsafe"

// DefaultHeap is the package-level HeapAllocator used as the nil-allocator default.
var DefaultHeap = &HeapAllocator{}

// HeapAllocator is a dummy allocator that delegates to Go's built-in heap.
// The generic helpers (New[T], MakeSlice[T]) detect it via type assertion and
// use new(T)/make([]T) directly, avoiding any unsafe operations. RawAlloc is
// provided as a fallback for callers that go through the Allocator interface
// without the generic helpers.
type HeapAllocator struct {
	pins []any // GC roots for objects allocated via RawAlloc
}

// RawAlloc allocates size bytes on the Go heap and pins the backing array to
// prevent GC collection. This is only used as a fallback; the generic helpers
// bypass it entirely via the type assertion fast path.
func (h *HeapAllocator) RawAlloc(size, align uintptr) unsafe.Pointer {
	buf := make([]byte, size)
	h.pins = append(h.pins, buf)
	return unsafe.Pointer(unsafe.SliceData(buf))
}

// Reset clears the pins slice, allowing GC to collect all RawAlloc'd memory.
func (h *HeapAllocator) Reset() {
	h.pins = h.pins[:0]
}
