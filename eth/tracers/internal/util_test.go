// Copyright 2023 The go-ethereum Authors
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
package internal

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
)

func TestMemCopying(t *testing.T) {
	for i, tc := range []struct {
		memsize  int64
		offset   int64
		size     int64
		wantErr  string
		wantSize int
	}{
		{0, 0, 100, "", 100},    // Should pad up to 100
		{0, 100, 0, "", 0},      // No need to pad (0 size)
		{100, 50, 100, "", 100}, // Should pad 100-150
		{100, 50, 5, "", 5},     // Wanted range fully within memory
		{100, -50, 0, "offset or size must not be negative", 0},                        // Error
		{0, 1, 1024*1024 + 1, "reached limit for padding memory slice: 1048578", 0},    // Error
		{10, 0, 1024*1024 + 100, "reached limit for padding memory slice: 1048666", 0}, // Error

	} {
		mem := vm.NewMemory()
		mem.Resize(uint64(tc.memsize))
		cpy, err := GetMemoryCopyPadded(mem.Data(), tc.offset, tc.size)
		if want := tc.wantErr; want != "" {
			if err == nil {
				t.Fatalf("test %d: want '%v' have no error", i, want)
			}
			if have := err.Error(); want != have {
				t.Fatalf("test %d: want '%v' have '%v'", i, want, have)
			}
			continue
		}
		if err != nil {
			t.Fatalf("test %d: unexpected error: %v", i, err)
		}
		if want, have := tc.wantSize, len(cpy); have != want {
			t.Fatalf("test %d: want %v have %v", i, want, have)
		}
	}
}
