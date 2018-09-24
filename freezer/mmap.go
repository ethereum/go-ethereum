// Copyright 2018 The go-ethereum Authors
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

package freezer

import (
	"os"
	"reflect"
	"unsafe"

	"github.com/edsrzf/mmap-go"
)

// openWithSize opens a file and ensures it is at least the provided bytes in
// size, growing it if necessary.
func openWithSize(path string, size uint64) (*os.File, error) {
	// Open the file for writing, potentially creating it
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	// Ensure the file's size is at least as much as requested
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if info.Size() < int64(size) {
		if err := file.Truncate(int64(size)); err != nil {
			file.Close()
			return nil, err
		}
	}
	// File is correctly open and of the correct size
	return file, err
}

// mmapBytes tries to memory map a file, creating it if it's non existent.
func mmapBytes(path string, size uint64) (*os.File, mmap.MMap, []byte, error) {
	// Open the file to memory map and ensure it's large enough
	file, err := openWithSize(path, size)
	if err != nil {
		return nil, nil, nil, err
	}
	// Memory map the file, cast to a byte slice and return
	mem, err := mmap.Map(file, mmap.RDWR, 0)
	if err != nil {
		file.Close()
		return nil, nil, nil, err
	}
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&mem))
	return file, mem, *(*[]byte)(unsafe.Pointer(&header)), nil
}

// mmapUints tries to memory map a file, creating it if it's non existent.
func mmapUints(path string, size uint64) (*os.File, mmap.MMap, []uint64, error) {
	// Open the file to memory map and ensure it's large enough
	file, err := openWithSize(path, size)
	if err != nil {
		return nil, nil, nil, err
	}
	// Memory map the file, cast to an uint64 slice and return
	mem, err := mmap.Map(file, mmap.RDWR, 0)
	if err != nil {
		file.Close()
		return nil, nil, nil, err
	}
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&mem))
	header.Len /= 8
	header.Cap /= 8

	return file, mem, *(*[]uint64)(unsafe.Pointer(&header)), nil
}
