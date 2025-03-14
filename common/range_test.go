// Copyright 2025 The go-ethereum Authors
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

package common

import (
	"slices"
	"testing"
)

func TestRangeIter(t *testing.T) {
	r := NewRange[uint32](1, 7)
	values := slices.Collect(r.Iter())
	if !slices.Equal(values, []uint32{1, 2, 3, 4, 5, 6, 7}) {
		t.Fatalf("wrong iter values: %v", values)
	}

	empty := NewRange[uint32](1, 0)
	values = slices.Collect(empty.Iter())
	if !slices.Equal(values, []uint32{}) {
		t.Fatalf("wrong iter values: %v", values)
	}
}
