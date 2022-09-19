// Copyright 2022 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package tracers

import "testing"

func TestTracker(t *testing.T) {
	var cases = []struct {
		limit   int
		head    uint64
		calls   []uint64
		expHead uint64
	}{
		// Release in order
		{
			limit:   3,
			head:    0,
			calls:   []uint64{0, 1, 2},
			expHead: 3,
		},
		{
			limit:   3,
			head:    0,
			calls:   []uint64{0, 1, 2, 3, 4, 5},
			expHead: 6,
		},

		// Release out of order
		{
			limit:   3,
			head:    0,
			calls:   []uint64{1, 2, 0},
			expHead: 3,
		},
		{
			limit:   3,
			head:    0,
			calls:   []uint64{1, 2, 0, 5, 4, 3},
			expHead: 6,
		},
	}
	for _, c := range cases {
		tracker := newStateTracker(c.limit, c.head)
		for _, call := range c.calls {
			tracker.releaseState(call, func() {})
		}
		tracker.lock.RLock()
		head := tracker.head
		tracker.lock.RUnlock()

		if head != c.expHead {
			t.Fatalf("Unexpected head want %d got %d", c.expHead, head)
		}
	}
}
