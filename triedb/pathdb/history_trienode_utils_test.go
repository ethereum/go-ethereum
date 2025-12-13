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

package pathdb

import (
	"testing"
)

func TestIsAncestor(t *testing.T) {
	suites := []struct {
		x, y uint16
		want bool
	}{
		{0, 1, true},
		{0, 16, true},
		{0, 17, true},
		{0, 272, true},

		{1, 0, false},
		{1, 2, false},
		{1, 17, true},
		{1, 18, true},
		{17, 273, true},
		{1, 1, false},
	}
	for _, tc := range suites {
		result := isAncestor(tc.x, tc.y)
		if result != tc.want {
			t.Fatalf("isAncestor(%d, %d) = %v, want %v", tc.x, tc.y, result, tc.want)
		}
	}
}
