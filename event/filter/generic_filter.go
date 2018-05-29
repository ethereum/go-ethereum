// Copyright 2014 The go-ethereum Authors
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

package filter

type Generic struct {
	Str1, Str2, Str3 string
	Data             map[string]struct{}

	Fn func(data interface{})
}

// g = registered, f = incoming
func (g Generic) Compare(f Filter) bool {
	var strMatch, dataMatch = true, true

	filter := f.(Generic)
	if (len(g.Str1) > 0 && filter.Str1 != g.Str1) ||
		(len(g.Str2) > 0 && filter.Str2 != g.Str2) ||
		(len(g.Str3) > 0 && filter.Str3 != g.Str3) {
		strMatch = false
	}

	for k := range g.Data {
		if _, ok := filter.Data[k]; !ok {
			return false
		}
	}

	return strMatch && dataMatch
}

func (g Generic) Trigger(data interface{}) {
	g.Fn(data)
}
