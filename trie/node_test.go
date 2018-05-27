// Copyright 2016 The go-ethereum Authors
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

package trie

import "testing"

func TestCanUnload(t *testing.T) {
	tests := []struct {
		flag                 nodeFlag
		cachegen, cachelimit uint16
		want                 bool
	}{
		{
			flag: nodeFlag{dirty: true, gen: 0},
			want: false,
		},
		{
			flag:     nodeFlag{dirty: false, gen: 0},
			cachegen: 0, cachelimit: 0,
			want: true,
		},
		{
			flag:     nodeFlag{dirty: false, gen: 65534},
			cachegen: 65535, cachelimit: 1,
			want: true,
		},
		{
			flag:     nodeFlag{dirty: false, gen: 65534},
			cachegen: 0, cachelimit: 1,
			want: true,
		},
		{
			flag:     nodeFlag{dirty: false, gen: 1},
			cachegen: 65535, cachelimit: 1,
			want: true,
		},
	}

	for _, test := range tests {
		if got := test.flag.canUnload(test.cachegen, test.cachelimit); got != test.want {
			t.Errorf("%+v\n   got %t, want %t", test, got, test.want)
		}
	}
}
