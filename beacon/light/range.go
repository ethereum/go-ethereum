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
	Start, End uint64
}

// IsEmpty returns true if the length of the range is zero.
func (a Range) IsEmpty() bool {
	return a.End == a.Start
}

// Contains returns true if the range includes the given period.
func (a Range) Contains(period uint64) bool {
	return period >= a.Start && period < a.End
}

// CanExpand returns true if the range includes or can be expanded with the given
// period (either the range is empty or the given period is inside, right before or
// right after the range).
func (a Range) CanExpand(period uint64) bool {
	return a.IsEmpty() || (period+1 >= a.Start && period <= a.End)
}

// Expand expands the range with the given period.
// This method assumes that CanExpand returned true: otherwise this is a no-op.
func (a *Range) Expand(period uint64) {
	if a.IsEmpty() {
		a.Start, a.End = period, period+1
		return
	}
	if a.Start == period+1 {
		a.Start--
	}
	if a.End == period {
		a.End++
	}
}
