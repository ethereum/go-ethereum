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

type Range struct {
	First     uint64
	AfterLast uint64
}

func (a Range) IsEmpty() bool {
	return a.AfterLast == a.First
}

func (a Range) Includes(period uint64) bool {
	return period >= a.First && period < a.AfterLast
}

func (a Range) CanExpand(period uint64) bool {
	return a.IsEmpty() || (period+1 >= a.First && period <= a.AfterLast)
}

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
