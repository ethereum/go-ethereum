// Copyright 2015 The go-ethereum Authors
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

package vm

import "fmt"

// Memory implements a simple memory model for the ethereum virtual machine.
type Memory struct {
	store []byte
}

func NewMemory() *Memory {
	return &Memory{nil}
}

// Set sets offset + size to value
func (m *Memory) Set(offset, size uint64, value []byte) {
	// length of store may never be less than offset + size.
	// The store should be resized PRIOR to setting the memory
	if size > uint64(len(m.store)) {
		panic("INVALID memory: store empty")
	}

	// It's possible the offset is greater than 0 and size equals 0. This is because
	// the calcMemSize (common.go) could potentially return 0 when size is zero (NO-OP)
	if size > 0 {
		copy(m.store[offset:offset+size], value)
	}
}

// Resize resizes the memory to size
func (m *Memory) Resize(size uint64) {
	if uint64(m.Len()) < size {
		m.store = append(m.store, make([]byte, size-uint64(m.Len()))...)
	}
}

// Get returns offset + size as a new slice
func (self *Memory) Get(offset, size int64) (cpy []byte) {
	if size == 0 {
		return nil
	}

	if len(self.store) > int(offset) {
		cpy = make([]byte, size)
		copy(cpy, self.store[offset:offset+size])

		return
	}

	return
}

// GetPtr returns the offset + size
func (self *Memory) GetPtr(offset, size int64) []byte {
	if size == 0 {
		return nil
	}

	if len(self.store) > int(offset) {
		return self.store[offset : offset+size]
	}

	return nil
}

// Len returns the length of the backing slice
func (m *Memory) Len() int {
	return len(m.store)
}

// Data returns the backing slice
func (m *Memory) Data() []byte {
	return m.store
}

func (m *Memory) Print() {
	fmt.Printf("### mem %d bytes ###\n", len(m.store))
	if len(m.store) > 0 {
		addr := 0
		for i := 0; i+32 <= len(m.store); i += 32 {
			fmt.Printf("%03d: % x\n", addr, m.store[i:i+32])
			addr++
		}
	} else {
		fmt.Println("-- empty --")
	}
	fmt.Println("####################")
}
