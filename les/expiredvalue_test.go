// Copyright 2020 The go-ethereum Authors
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

package les

import "testing"

func TestValueExpiration(t *testing.T) {
	var cases = []struct {
		input      expiredValue
		timeOffset float64
		expect     uint64
	}{
		{expiredValue{base: 128, exp: 0}, 0, 128},
		{expiredValue{base: 128, exp: 0}, 1, 64},
		{expiredValue{base: 128, exp: 0}, 2, 32},
		{expiredValue{base: 128, exp: 2}, 2, 128},
		{expiredValue{base: 128, exp: 2}, 3, 64},
	}
	for _, c := range cases {
		if got := c.input.value(c.timeOffset); got != c.expect {
			t.Fatalf("Value mismatch, want=%d, got=%d", c.expect, got)
		}
	}
}

func TestValueAddition(t *testing.T) {
	var cases = []struct {
		input      expiredValue
		addend     int64
		timeOffset float64
		expect     uint64
		expectNet  int64
	}{
		// Addition
		{expiredValue{base: 128, exp: 0}, 128, 0, 256, 128},
		{expiredValue{base: 128, exp: 2}, 128, 0, 640, 128},

		// Addition with offset
		{expiredValue{base: 128, exp: 0}, 128, 1, 192, 128},
		{expiredValue{base: 128, exp: 2}, 128, 1, 384, 128},
		{expiredValue{base: 128, exp: 2}, 128, 3, 192, 128},

		// Subtraction
		{expiredValue{base: 128, exp: 0}, -64, 0, 64, -64},
		{expiredValue{base: 128, exp: 0}, -128, 0, 0, -128},
		{expiredValue{base: 128, exp: 0}, -192, 0, 0, -128},

		// Subtraction with offset
		{expiredValue{base: 128, exp: 0}, -64, 1, 0, -64},
		{expiredValue{base: 128, exp: 0}, -128, 1, 0, -64},
		{expiredValue{base: 128, exp: 2}, -128, 1, 128, -128},
		{expiredValue{base: 128, exp: 2}, -128, 2, 0, -128},
	}
	for _, c := range cases {
		if net := c.input.add(c.addend, c.timeOffset); net != c.expectNet {
			t.Fatalf("Net amount mismatch, want=%d, got=%d", c.expectNet, net)
		}
		if got := c.input.value(c.timeOffset); got != c.expect {
			t.Fatalf("Value mismatch, want=%d, got=%d", c.expect, got)
		}
	}
}

func TestExpiredValueAddition(t *testing.T) {
	var cases = []struct {
		input      expiredValue
		another    expiredValue
		timeOffset float64
		expect     uint64
	}{
		{expiredValue{base: 128, exp: 0}, expiredValue{base: 128, exp: 0}, 0, 256},
		{expiredValue{base: 128, exp: 1}, expiredValue{base: 128, exp: 0}, 0, 384},
		{expiredValue{base: 128, exp: 0}, expiredValue{base: 128, exp: 1}, 0, 384},
		{expiredValue{base: 128, exp: 0}, expiredValue{base: 128, exp: 0}, 1, 128},
	}
	for _, c := range cases {
		c.input.addExp(c.another)
		if got := c.input.value(c.timeOffset); got != c.expect {
			t.Fatalf("Value mismatch, want=%d, got=%d", c.expect, got)
		}
	}
}

func TestExpiredValueSubtraction(t *testing.T) {
	var cases = []struct {
		input      expiredValue
		another    expiredValue
		timeOffset float64
		expect     uint64
	}{
		{expiredValue{base: 128, exp: 0}, expiredValue{base: 128, exp: 0}, 0, 0},
		{expiredValue{base: 128, exp: 0}, expiredValue{base: 128, exp: 1}, 0, 0},
		{expiredValue{base: 128, exp: 1}, expiredValue{base: 128, exp: 0}, 0, 128},
		{expiredValue{base: 128, exp: 1}, expiredValue{base: 128, exp: 0}, 1, 64},
	}
	for _, c := range cases {
		c.input.subExp(c.another)
		if got := c.input.value(c.timeOffset); got != c.expect {
			t.Fatalf("Value mismatch, want=%d, got=%d", c.expect, got)
		}
	}
}
