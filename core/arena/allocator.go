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

// Allocator is the interface for arena-style allocators. RawAlloc returns
// a pointer to a zeroed region of at least `size` bytes, aligned to `align`.
// Reset releases all memory allocated since the last Reset (or since creation).
type Allocator interface {
	RawAlloc(size, align uintptr) unsafe.Pointer
	Reset()
}

// New allocates and zeros a single value of type T from the given allocator.
// When the allocator is a *HeapAllocator, this uses the standard `new(T)` and
// involves no unsafe operations.
func New[T any](a Allocator) *T {
	if _, ok := a.(*HeapAllocator); ok {
		return new(T)
	}
	var zero T
	size := unsafe.Sizeof(zero)
	align := unsafe.Alignof(zero)
	ptr := a.RawAlloc(size, align)
	return (*T)(ptr)
}

// MakeSlice allocates a slice of type []T with the given length and capacity
// from the allocator. When the allocator is a *HeapAllocator, this uses the
// standard `make([]T, length, capacity)` and involves no unsafe operations.
func MakeSlice[T any](a Allocator, length, capacity int) []T {
	if _, ok := a.(*HeapAllocator); ok {
		return make([]T, length, capacity)
	}
	var zero T
	elemSize := unsafe.Sizeof(zero)
	elemAlign := unsafe.Alignof(zero)
	totalSize := elemSize * uintptr(capacity)
	ptr := a.RawAlloc(totalSize, elemAlign)
	return unsafe.Slice((*T)(ptr), capacity)[:length]
}
