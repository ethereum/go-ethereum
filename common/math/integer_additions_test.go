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

package math

import (
	"math"
	"testing"
)

func TestSafeDiv(t *testing.T) {
	tests := []struct {
		x, y     uint64
		expected uint64
		isError  bool
	}{
		{10, 2, 5, false},
		{10, 3, 3, false},
		{0, 5, 0, false},
		{100, 1, 100, false},
		{math.MaxUint64, 2, math.MaxUint64 / 2, false},
		// Division by zero cases
		{10, 0, 0, true},
		{0, 0, 0, true},
		{math.MaxUint64, 0, 0, true},
	}

	for i, test := range tests {
		result, isError := SafeDiv(test.x, test.y)
		if isError != test.isError {
			t.Errorf("test %d: SafeDiv(%d, %d) error = %v, want %v", i, test.x, test.y, isError, test.isError)
		}
		if !isError && result != test.expected {
			t.Errorf("test %d: SafeDiv(%d, %d) = %d, want %d", i, test.x, test.y, result, test.expected)
		}
	}
}

func TestSafeMod(t *testing.T) {
	tests := []struct {
		x, y     uint64
		expected uint64
		isError  bool
	}{
		{10, 3, 1, false},
		{10, 2, 0, false},
		{0, 5, 0, false},
		{7, 7, 0, false},
		{math.MaxUint64, 2, 1, false},
		// Modulo by zero cases
		{10, 0, 0, true},
		{0, 0, 0, true},
	}

	for i, test := range tests {
		result, isError := SafeMod(test.x, test.y)
		if isError != test.isError {
			t.Errorf("test %d: SafeMod(%d, %d) error = %v, want %v", i, test.x, test.y, isError, test.isError)
		}
		if !isError && result != test.expected {
			t.Errorf("test %d: SafeMod(%d, %d) = %d, want %d", i, test.x, test.y, result, test.expected)
		}
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		x, y     uint64
		expected uint64
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 100, 0},
		{math.MaxUint64, 0, 0},
		{math.MaxUint64, math.MaxUint64 - 1, math.MaxUint64 - 1},
	}

	for i, test := range tests {
		result := Min(test.x, test.y)
		if result != test.expected {
			t.Errorf("test %d: Min(%d, %d) = %d, want %d", i, test.x, test.y, result, test.expected)
		}
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		x, y     uint64
		expected uint64
	}{
		{1, 2, 2},
		{2, 1, 2},
		{5, 5, 5},
		{0, 100, 100},
		{math.MaxUint64, 0, math.MaxUint64},
		{math.MaxUint64, math.MaxUint64 - 1, math.MaxUint64},
	}

	for i, test := range tests {
		result := Max(test.x, test.y)
		if result != test.expected {
			t.Errorf("test %d: Max(%d, %d) = %d, want %d", i, test.x, test.y, result, test.expected)
		}
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		x, min, max uint64
		expected    uint64
	}{
		{5, 0, 10, 5},      // within range
		{0, 5, 10, 5},      // below min
		{15, 5, 10, 10},    // above max
		{5, 5, 10, 5},      // at min
		{10, 5, 10, 10},    // at max
		{5, 5, 5, 5},       // min == max == x
		{0, 0, 0, 0},       // all zero
		{100, 0, 50, 50},   // clamped to max
	}

	for i, test := range tests {
		result := Clamp(test.x, test.min, test.max)
		if result != test.expected {
			t.Errorf("test %d: Clamp(%d, %d, %d) = %d, want %d", i, test.x, test.min, test.max, result, test.expected)
		}
	}
}

func TestAbsDiff(t *testing.T) {
	tests := []struct {
		x, y     uint64
		expected uint64
	}{
		{10, 3, 7},
		{3, 10, 7},
		{5, 5, 0},
		{0, 0, 0},
		{math.MaxUint64, 0, math.MaxUint64},
		{0, math.MaxUint64, math.MaxUint64},
		{100, 99, 1},
	}

	for i, test := range tests {
		result := AbsDiff(test.x, test.y)
		if result != test.expected {
			t.Errorf("test %d: AbsDiff(%d, %d) = %d, want %d", i, test.x, test.y, result, test.expected)
		}
	}
}
