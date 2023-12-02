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

// periodRange represents a (possibly zero-length) range of integers (sync periods).
type periodRange struct {
	Start, End uint64
}

// isEmpty returns true if the length of the range is zero.
func (a periodRange) isEmpty() bool {
	return a.End == a.Start
}

// contains returns true if the range includes the given period.
func (a periodRange) contains(period uint64) bool {
	return period >= a.Start && period < a.End
}

// canExpand returns true if the range includes or can be expanded with the given
// period (either the range is empty or the given period is inside, right before or
// right after the range).
func (a periodRange) canExpand(period uint64) bool {
	return a.isEmpty() || (period+1 >= a.Start && period <= a.End)
}

// expand expands the range with the given period.
// This method assumes that canExpand returned true: otherwise this is a no-op.
func (a *periodRange) expand(period uint64) {
	if a.isEmpty() {
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

// split splits the range into two ranges. The 'fromPeriod' will be the first
// element in the second range (if present).
// The original range is unchanged by this operation
func (a *periodRange) split(fromPeriod uint64) (periodRange, periodRange) {
	if fromPeriod <= a.Start {
		// First range empty, everything in second range,
		return periodRange{}, *a
	}
	if fromPeriod >= a.End {
		// Second range empty, everything in first range,
		return *a, periodRange{}
	}
	x := periodRange{a.Start, fromPeriod}
	y := periodRange{fromPeriod, a.End}
	return x, y
}

// each invokes the supplied function fn once per period in range
func (a *periodRange) each(fn func(uint64)) {
	for p := a.Start; p < a.End; p++ {
		fn(p)
	}
}
