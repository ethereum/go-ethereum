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

package light

// Range represents a (possibly zero-length) range of integers (sync periods).
type Range struct {
	First     uint64
	AfterLast uint64
}

// IsEmpty returns true if the length of the range is zero.
func (a Range) IsEmpty() bool {
	return a.AfterLast == a.First
}

// Includes returns true if the range includes the given period.
func (a Range) Includes(period uint64) bool {
	return period >= a.First && period < a.AfterLast
}

// CanExpand returns true if the range can be expanded with the given period
// (either the range is empty or the new period is right before or after the range).
func (a Range) CanExpand(period uint64) bool {
	return a.IsEmpty() || (period+1 >= a.First && period <= a.AfterLast)
}

// Expand expands the range with the given period (assumes that CanExpand returned true).
func (a *Range) Expand(period uint64) {
	if a.IsEmpty() {
		a.First, a.AfterLast = period, period+1
		return
	}
	if a.Includes(period) {
		return
	}
	if a.First == period+1 {
		a.First--
		return
	}
	if a.AfterLast == period {
		a.AfterLast++
		return
	}
}
