// Copyright 2025 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSub(t *testing.T) {
	for _, tt := range [][3][]int{ // start, sub, want
		{{}, {}, {}},
		{{0}, {}, {0}},
		{{}, {0}, {}},
		{{0, 1}, {0}, {1}},
		{{0, 1}, {1}, {0}},
	} {
		in, sub := tt[0], tt[1]
		want := tt[2]
		got := From(in...).Sub(From(sub...)).Slice()
		assert.Equalf(t, want, got, "Set(%v).Sub(%v)", in, sub)
	}
}

func TestIntersect(t *testing.T) {
	for _, tt := range [][3][]int{ // L, R, intersection
		{{}, {}, {}},
		{{0}, {}, {}},
		{{0}, {0}, {0}},
		{{0, 1}, {0}, {0}},
		{{0, 1}, {1}, {1}},
	} {
		want := tt[2]

		for i := 0; i <= 1; i++ { // commutativity
			lhs, rhs := tt[i], tt[1-i]
			got := From(lhs...).Intersect(From(rhs...)).Slice()
			assert.Equalf(t, want, got, "Set(%v).Intersect(%v)", lhs, rhs)
		}
	}
}
