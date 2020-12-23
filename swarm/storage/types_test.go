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

package storage

import (
	"strconv"
	"testing"
)

// TestProximity validates Proximity function with explicit
// values in a table-driven test. It is highly dependant on
// MaxPO constant and it validates cases up to MaxPO=32.
func TestProximity(t *testing.T) {
	// integer from base2 encoded string
	bx := func(s string) uint8 {
		i, err := strconv.ParseUint(s, 2, 8)
		if err != nil {
			t.Fatal(err)
		}
		return uint8(i)
	}
	// adjust expected bins in respect to MaxPO
	limitPO := func(po uint8) uint8 {
		if po > MaxPO {
			return MaxPO
		}
		return po
	}
	base := []byte{bx("00000000"), bx("00000000"), bx("00000000"), bx("00000000")}
	for _, tc := range []struct {
		addr []byte
		po   uint8
	}{
		{
			addr: base,
			po:   MaxPO,
		},
		{
			addr: []byte{bx("10000000"), bx("00000000"), bx("00000000"), bx("00000000")},
			po:   limitPO(0),
		},
		{
			addr: []byte{bx("01000000"), bx("00000000"), bx("00000000"), bx("00000000")},
			po:   limitPO(1),
		},
		{
			addr: []byte{bx("00100000"), bx("00000000"), bx("00000000"), bx("00000000")},
			po:   limitPO(2),
		},
		{
			addr: []byte{bx("00010000"), bx("00000000"), bx("00000000"), bx("00000000")},
			po:   limitPO(3),
		},
		{
			addr: []byte{bx("00001000"), bx("00000000"), bx("00000000"), bx("00000000")},
			po:   limitPO(4),
		},
		{
			addr: []byte{bx("00000100"), bx("00000000"), bx("00000000"), bx("00000000")},
			po:   limitPO(5),
		},
		{
			addr: []byte{bx("00000010"), bx("00000000"), bx("00000000"), bx("00000000")},
			po:   limitPO(6),
		},
		{
			addr: []byte{bx("00000001"), bx("00000000"), bx("00000000"), bx("00000000")},
			po:   limitPO(7),
		},
		{
			addr: []byte{bx("00000000"), bx("10000000"), bx("00000000"), bx("00000000")},
			po:   limitPO(8),
		},
		{
			addr: []byte{bx("00000000"), bx("01000000"), bx("00000000"), bx("00000000")},
			po:   limitPO(9),
		},
		{
			addr: []byte{bx("00000000"), bx("00100000"), bx("00000000"), bx("00000000")},
			po:   limitPO(10),
		},
		{
			addr: []byte{bx("00000000"), bx("00010000"), bx("00000000"), bx("00000000")},
			po:   limitPO(11),
		},
		{
			addr: []byte{bx("00000000"), bx("00001000"), bx("00000000"), bx("00000000")},
			po:   limitPO(12),
		},
		{
			addr: []byte{bx("00000000"), bx("00000100"), bx("00000000"), bx("00000000")},
			po:   limitPO(13),
		},
		{
			addr: []byte{bx("00000000"), bx("00000010"), bx("00000000"), bx("00000000")},
			po:   limitPO(14),
		},
		{
			addr: []byte{bx("00000000"), bx("00000001"), bx("00000000"), bx("00000000")},
			po:   limitPO(15),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("10000000"), bx("00000000")},
			po:   limitPO(16),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("01000000"), bx("00000000")},
			po:   limitPO(17),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("00100000"), bx("00000000")},
			po:   limitPO(18),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("00010000"), bx("00000000")},
			po:   limitPO(19),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("00001000"), bx("00000000")},
			po:   limitPO(20),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("00000100"), bx("00000000")},
			po:   limitPO(21),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("00000010"), bx("00000000")},
			po:   limitPO(22),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("00000001"), bx("00000000")},
			po:   limitPO(23),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("00000000"), bx("10000000")},
			po:   limitPO(24),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("00000000"), bx("01000000")},
			po:   limitPO(25),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("00000000"), bx("00100000")},
			po:   limitPO(26),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("00000000"), bx("00010000")},
			po:   limitPO(27),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("00000000"), bx("00001000")},
			po:   limitPO(28),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("00000000"), bx("00000100")},
			po:   limitPO(29),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("00000000"), bx("00000010")},
			po:   limitPO(30),
		},
		{
			addr: []byte{bx("00000000"), bx("00000000"), bx("00000000"), bx("00000001")},
			po:   limitPO(31),
		},
	} {
		got := uint8(Proximity(base, tc.addr))
		if got != tc.po {
			t.Errorf("got %v bin, want %v", got, tc.po)
		}
	}
}
