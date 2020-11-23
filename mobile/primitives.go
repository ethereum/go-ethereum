// Copyright 2016 The go-ethereum Authors
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

// Contains various wrappers for primitive types.

package geth

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// Strings represents s slice of strs.
type Strings struct{ strs []string }

// Size returns the number of strs in the slice.
func (s *Strings) Size() int {
	return len(s.strs)
}

// Get returns the string at the given index from the slice.
func (s *Strings) Get(index int) (str string, _ error) {
	if index < 0 || index >= len(s.strs) {
		return "", errors.New("index out of bounds")
	}
	return s.strs[index], nil
}

// Set sets the string at the given index in the slice.
func (s *Strings) Set(index int, str string) error {
	if index < 0 || index >= len(s.strs) {
		return errors.New("index out of bounds")
	}
	s.strs[index] = str
	return nil
}

// String implements the Stringer interface.
func (s *Strings) String() string {
	return fmt.Sprintf("%v", s.strs)
}

// Bools represents a slice of bool.
type Bools struct{ bools []bool }

// Size returns the number of bool in the slice.
func (bs *Bools) Size() int {
	return len(bs.bools)
}

// Get returns the bool at the given index from the slice.
func (bs *Bools) Get(index int) (b bool, _ error) {
	if index < 0 || index >= len(bs.bools) {
		return false, errors.New("index out of bounds")
	}
	return bs.bools[index], nil
}

// Set sets the bool at the given index in the slice.
func (bs *Bools) Set(index int, b bool) error {
	if index < 0 || index >= len(bs.bools) {
		return errors.New("index out of bounds")
	}
	bs.bools[index] = b
	return nil
}

// String implements the Stringer interface.
func (bs *Bools) String() string {
	return fmt.Sprintf("%v", bs.bools)
}

// Binaries represents a slice of byte slice
type Binaries struct{ binaries [][]byte }

// Size returns the number of byte slice in the slice.
func (bs *Binaries) Size() int {
	return len(bs.binaries)
}

// Get returns the byte slice at the given index from the slice.
func (bs *Binaries) Get(index int) (binary []byte, _ error) {
	if index < 0 || index >= len(bs.binaries) {
		return nil, errors.New("index out of bounds")
	}
	return common.CopyBytes(bs.binaries[index]), nil
}

// Set sets the byte slice at the given index in the slice.
func (bs *Binaries) Set(index int, binary []byte) error {
	if index < 0 || index >= len(bs.binaries) {
		return errors.New("index out of bounds")
	}
	bs.binaries[index] = common.CopyBytes(binary)
	return nil
}

// String implements the Stringer interface.
func (bs *Binaries) String() string {
	return fmt.Sprintf("%v", bs.binaries)
}
