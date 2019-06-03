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

package intervals

import "testing"

// Test tests Interval methods Add, Next and Last for various
// initial state.
func Test(t *testing.T) {
	for i, tc := range []struct {
		startLimit uint64
		initial    [][2]uint64
		start      uint64
		end        uint64
		expected   string
		nextStart  uint64
		nextEnd    uint64
		last       uint64
	}{
		{
			initial:   nil,
			start:     0,
			end:       0,
			expected:  "[[0 0]]",
			nextStart: 1,
			nextEnd:   0,
			last:      0,
		},
		{
			initial:   nil,
			start:     0,
			end:       10,
			expected:  "[[0 10]]",
			nextStart: 11,
			nextEnd:   0,
			last:      10,
		},
		{
			initial:   nil,
			start:     5,
			end:       15,
			expected:  "[[5 15]]",
			nextStart: 0,
			nextEnd:   4,
			last:      15,
		},
		{
			initial:   [][2]uint64{{0, 0}},
			start:     0,
			end:       0,
			expected:  "[[0 0]]",
			nextStart: 1,
			nextEnd:   0,
			last:      0,
		},
		{
			initial:   [][2]uint64{{0, 0}},
			start:     5,
			end:       15,
			expected:  "[[0 0] [5 15]]",
			nextStart: 1,
			nextEnd:   4,
			last:      15,
		},
		{
			initial:   [][2]uint64{{5, 15}},
			start:     5,
			end:       15,
			expected:  "[[5 15]]",
			nextStart: 0,
			nextEnd:   4,
			last:      15,
		},
		{
			initial:   [][2]uint64{{5, 15}},
			start:     5,
			end:       20,
			expected:  "[[5 20]]",
			nextStart: 0,
			nextEnd:   4,
			last:      20,
		},
		{
			initial:   [][2]uint64{{5, 15}},
			start:     10,
			end:       20,
			expected:  "[[5 20]]",
			nextStart: 0,
			nextEnd:   4,
			last:      20,
		},
		{
			initial:   [][2]uint64{{5, 15}},
			start:     0,
			end:       20,
			expected:  "[[0 20]]",
			nextStart: 21,
			nextEnd:   0,
			last:      20,
		},
		{
			initial:   [][2]uint64{{5, 15}},
			start:     2,
			end:       10,
			expected:  "[[2 15]]",
			nextStart: 0,
			nextEnd:   1,
			last:      15,
		},
		{
			initial:   [][2]uint64{{5, 15}},
			start:     2,
			end:       4,
			expected:  "[[2 15]]",
			nextStart: 0,
			nextEnd:   1,
			last:      15,
		},
		{
			initial:   [][2]uint64{{5, 15}},
			start:     2,
			end:       5,
			expected:  "[[2 15]]",
			nextStart: 0,
			nextEnd:   1,
			last:      15,
		},
		{
			initial:   [][2]uint64{{5, 15}},
			start:     2,
			end:       3,
			expected:  "[[2 3] [5 15]]",
			nextStart: 0,
			nextEnd:   1,
			last:      15,
		},
		{
			initial:   [][2]uint64{{5, 15}},
			start:     2,
			end:       4,
			expected:  "[[2 15]]",
			nextStart: 0,
			nextEnd:   1,
			last:      15,
		},
		{
			initial:   [][2]uint64{{0, 1}, {5, 15}},
			start:     2,
			end:       4,
			expected:  "[[0 15]]",
			nextStart: 16,
			nextEnd:   0,
			last:      15,
		},
		{
			initial:   [][2]uint64{{0, 5}, {15, 20}},
			start:     2,
			end:       10,
			expected:  "[[0 10] [15 20]]",
			nextStart: 11,
			nextEnd:   14,
			last:      20,
		},
		{
			initial:   [][2]uint64{{0, 5}, {15, 20}},
			start:     8,
			end:       18,
			expected:  "[[0 5] [8 20]]",
			nextStart: 6,
			nextEnd:   7,
			last:      20,
		},
		{
			initial:   [][2]uint64{{0, 5}, {15, 20}},
			start:     2,
			end:       17,
			expected:  "[[0 20]]",
			nextStart: 21,
			nextEnd:   0,
			last:      20,
		},
		{
			initial:   [][2]uint64{{0, 5}, {15, 20}},
			start:     2,
			end:       25,
			expected:  "[[0 25]]",
			nextStart: 26,
			nextEnd:   0,
			last:      25,
		},
		{
			initial:   [][2]uint64{{0, 5}, {15, 20}},
			start:     5,
			end:       14,
			expected:  "[[0 20]]",
			nextStart: 21,
			nextEnd:   0,
			last:      20,
		},
		{
			initial:   [][2]uint64{{0, 5}, {15, 20}},
			start:     6,
			end:       14,
			expected:  "[[0 20]]",
			nextStart: 21,
			nextEnd:   0,
			last:      20,
		},
		{
			initial:   [][2]uint64{{0, 5}, {15, 20}, {30, 40}},
			start:     6,
			end:       29,
			expected:  "[[0 40]]",
			nextStart: 41,
			nextEnd:   0,
			last:      40,
		},
		{
			initial:   [][2]uint64{{0, 5}, {15, 20}, {30, 40}, {50, 60}},
			start:     3,
			end:       55,
			expected:  "[[0 60]]",
			nextStart: 61,
			nextEnd:   0,
			last:      60,
		},
		{
			initial:   [][2]uint64{{0, 5}, {15, 20}, {30, 40}, {50, 60}},
			start:     21,
			end:       49,
			expected:  "[[0 5] [15 60]]",
			nextStart: 6,
			nextEnd:   14,
			last:      60,
		},
		{
			initial:   [][2]uint64{{0, 5}, {15, 20}, {30, 40}, {50, 60}},
			start:     0,
			end:       100,
			expected:  "[[0 100]]",
			nextStart: 101,
			nextEnd:   0,
			last:      100,
		},
		{
			startLimit: 100,
			initial:    nil,
			start:      0,
			end:        0,
			expected:   "[]",
			nextStart:  100,
			nextEnd:    0,
			last:       0,
		},
		{
			startLimit: 100,
			initial:    nil,
			start:      20,
			end:        30,
			expected:   "[]",
			nextStart:  100,
			nextEnd:    0,
			last:       0,
		},
		{
			startLimit: 100,
			initial:    nil,
			start:      50,
			end:        100,
			expected:   "[[100 100]]",
			nextStart:  101,
			nextEnd:    0,
			last:       100,
		},
		{
			startLimit: 100,
			initial:    nil,
			start:      50,
			end:        110,
			expected:   "[[100 110]]",
			nextStart:  111,
			nextEnd:    0,
			last:       110,
		},
		{
			startLimit: 100,
			initial:    nil,
			start:      120,
			end:        130,
			expected:   "[[120 130]]",
			nextStart:  100,
			nextEnd:    119,
			last:       130,
		},
		{
			startLimit: 100,
			initial:    nil,
			start:      120,
			end:        130,
			expected:   "[[120 130]]",
			nextStart:  100,
			nextEnd:    119,
			last:       130,
		},
	} {
		intervals := NewIntervals(tc.startLimit)
		intervals.ranges = tc.initial
		intervals.Add(tc.start, tc.end)
		got := intervals.String()
		if got != tc.expected {
			t.Errorf("interval #%d: expected %s, got %s", i, tc.expected, got)
		}
		nextStart, nextEnd := intervals.Next()
		if nextStart != tc.nextStart {
			t.Errorf("interval #%d, expected next start %d, got %d", i, tc.nextStart, nextStart)
		}
		if nextEnd != tc.nextEnd {
			t.Errorf("interval #%d, expected next end %d, got %d", i, tc.nextEnd, nextEnd)
		}
		last := intervals.Last()
		if last != tc.last {
			t.Errorf("interval #%d, expected last %d, got %d", i, tc.last, last)
		}
	}
}

func TestMerge(t *testing.T) {
	for i, tc := range []struct {
		initial  [][2]uint64
		merge    [][2]uint64
		expected string
	}{
		{
			initial:  nil,
			merge:    nil,
			expected: "[]",
		},
		{
			initial:  [][2]uint64{{10, 20}},
			merge:    nil,
			expected: "[[10 20]]",
		},
		{
			initial:  nil,
			merge:    [][2]uint64{{15, 25}},
			expected: "[[15 25]]",
		},
		{
			initial:  [][2]uint64{{0, 100}},
			merge:    [][2]uint64{{150, 250}},
			expected: "[[0 100] [150 250]]",
		},
		{
			initial:  [][2]uint64{{0, 100}},
			merge:    [][2]uint64{{101, 250}},
			expected: "[[0 250]]",
		},
		{
			initial:  [][2]uint64{{0, 10}, {30, 40}},
			merge:    [][2]uint64{{20, 25}, {41, 50}},
			expected: "[[0 10] [20 25] [30 50]]",
		},
		{
			initial:  [][2]uint64{{0, 5}, {15, 20}, {30, 40}, {50, 60}},
			merge:    [][2]uint64{{6, 25}},
			expected: "[[0 25] [30 40] [50 60]]",
		},
	} {
		intervals := NewIntervals(0)
		intervals.ranges = tc.initial
		m := NewIntervals(0)
		m.ranges = tc.merge

		intervals.Merge(m)

		got := intervals.String()
		if got != tc.expected {
			t.Errorf("interval #%d: expected %s, got %s", i, tc.expected, got)
		}
	}
}
