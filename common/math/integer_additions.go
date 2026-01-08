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

// SafeDiv returns x/y and checks for division by zero.
// Returns (0, true) if y is zero (indicating an error condition).
// Returns (x/y, false) if the division is valid.
func SafeDiv(x, y uint64) (uint64, bool) {
	if y == 0 {
		return 0, true
	}
	return x / y, false
}

// SafeMod returns x%y and checks for division by zero.
// Returns (0, true) if y is zero (indicating an error condition).
// Returns (x%y, false) if the operation is valid.
func SafeMod(x, y uint64) (uint64, bool) {
	if y == 0 {
		return 0, true
	}
	return x % y, false
}

// Min returns the smaller of x or y.
func Min(x, y uint64) uint64 {
	if x < y {
		return x
	}
	return y
}

// Max returns the larger of x or y.
func Max(x, y uint64) uint64 {
	if x > y {
		return x
	}
	return y
}

// Clamp returns x clamped to the inclusive range [min, max].
// If x is less than min, min is returned.
// If x is greater than max, max is returned.
// Otherwise, x is returned unchanged.
func Clamp(x, min, max uint64) uint64 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

// AbsDiff returns the absolute difference between x and y.
// This is useful when you need |x - y| without worrying about underflow.
func AbsDiff(x, y uint64) uint64 {
	if x > y {
		return x - y
	}
	return y - x
}
