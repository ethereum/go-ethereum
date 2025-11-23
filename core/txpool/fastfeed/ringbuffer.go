// Copyright 2024 The go-ethereum Authors
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

package fastfeed

import (
	"sync/atomic"
	"unsafe"
)

// RingBuffer is a lock-free single-producer, multiple-consumer ring buffer
// optimized for low-latency transaction propagation.
type RingBuffer struct {
	buffer   []unsafe.Pointer
	capacity uint64
	mask     uint64
	
	// Writer position (single producer)
	writePos atomic.Uint64
	
	// Padding to prevent false sharing
	_ [56]byte
	
	// Reader positions (multiple consumers)
	// Each consumer maintains its own read position
	readPositions []atomic.Uint64
	maxReaders    int
}

// NewRingBuffer creates a new ring buffer with the given capacity.
// Capacity must be a power of 2 for efficient masking.
func NewRingBuffer(capacity int, maxReaders int) *RingBuffer {
	if capacity&(capacity-1) != 0 {
		panic("capacity must be a power of 2")
	}
	
	rb := &RingBuffer{
		buffer:        make([]unsafe.Pointer, capacity),
		capacity:      uint64(capacity),
		mask:          uint64(capacity - 1),
		maxReaders:    maxReaders,
		readPositions: make([]atomic.Uint64, maxReaders),
	}
	
	return rb
}

// Write adds a new entry to the ring buffer.
// Returns false if the buffer is full (slowest reader is too far behind).
func (rb *RingBuffer) Write(data unsafe.Pointer) bool {
	writePos := rb.writePos.Load()
	
	// Check if we would overwrite unread data
	minReadPos := rb.getMinReadPos()
	if writePos-minReadPos >= rb.capacity {
		// Buffer is full, would overwrite unread data
		return false
	}
	
	// Write data
	idx := writePos & rb.mask
	atomic.StorePointer(&rb.buffer[idx], data)
	
	// Advance write position
	rb.writePos.Store(writePos + 1)
	
	return true
}

// Read reads the next entry for the given reader ID.
// Returns nil if no new data is available.
func (rb *RingBuffer) Read(readerID int) unsafe.Pointer {
	if readerID >= rb.maxReaders {
		return nil
	}
	
	readPos := rb.readPositions[readerID].Load()
	writePos := rb.writePos.Load()
	
	// Check if data is available
	if readPos >= writePos {
		return nil
	}
	
	// Check if data hasn't been overwritten
	if writePos-readPos > rb.capacity {
		// Data was overwritten, skip to oldest available
		readPos = writePos - rb.capacity
		rb.readPositions[readerID].Store(readPos)
	}
	
	// Read data
	idx := readPos & rb.mask
	data := atomic.LoadPointer(&rb.buffer[idx])
	
	// Advance read position
	rb.readPositions[readerID].Store(readPos + 1)
	
	return data
}

// Peek reads the next entry without advancing the read position.
func (rb *RingBuffer) Peek(readerID int) unsafe.Pointer {
	if readerID >= rb.maxReaders {
		return nil
	}
	
	readPos := rb.readPositions[readerID].Load()
	writePos := rb.writePos.Load()
	
	if readPos >= writePos {
		return nil
	}
	
	if writePos-readPos > rb.capacity {
		return nil
	}
	
	idx := readPos & rb.mask
	return atomic.LoadPointer(&rb.buffer[idx])
}

// Available returns the number of entries available to read for a given reader.
func (rb *RingBuffer) Available(readerID int) int {
	if readerID >= rb.maxReaders {
		return 0
	}
	
	readPos := rb.readPositions[readerID].Load()
	writePos := rb.writePos.Load()
	
	if readPos >= writePos {
		return 0
	}
	
	available := writePos - readPos
	if available > rb.capacity {
		available = rb.capacity
	}
	
	return int(available)
}

// getMinReadPos returns the minimum read position across all readers.
func (rb *RingBuffer) getMinReadPos() uint64 {
	min := rb.writePos.Load()
	
	for i := 0; i < rb.maxReaders; i++ {
		pos := rb.readPositions[i].Load()
		if pos < min {
			min = pos
		}
	}
	
	return min
}

// Reset resets a reader's position to the current write position.
// Useful for catching up when a reader falls too far behind.
func (rb *RingBuffer) Reset(readerID int) {
	if readerID < rb.maxReaders {
		rb.readPositions[readerID].Store(rb.writePos.Load())
	}
}

// Stats returns statistics about the ring buffer state.
type BufferStats struct {
	Capacity       int
	WritePosition  uint64
	ReadPositions  []uint64
	MinReadPos     uint64
	MaxLag         uint64
}

// Stats returns current buffer statistics.
func (rb *RingBuffer) Stats() BufferStats {
	writePos := rb.writePos.Load()
	readPositions := make([]uint64, rb.maxReaders)
	minReadPos := writePos
	
	for i := 0; i < rb.maxReaders; i++ {
		pos := rb.readPositions[i].Load()
		readPositions[i] = pos
		if pos < minReadPos {
			minReadPos = pos
		}
	}
	
	maxLag := uint64(0)
	if writePos > minReadPos {
		maxLag = writePos - minReadPos
	}
	
	return BufferStats{
		Capacity:      int(rb.capacity),
		WritePosition: writePos,
		ReadPositions: readPositions,
		MinReadPos:    minReadPos,
		MaxLag:        maxLag,
	}
}

