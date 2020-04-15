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

package utils

import (
	"testing"
)

func TestValueExpiration(t *testing.T) {
	var cases = []struct {
		input      ExpiredValue
		timeOffset Fixed64
		expect     uint64
	}{
		{ExpiredValue{Base: 128, Exp: 0}, Uint64ToFixed64(0), 128},
		{ExpiredValue{Base: 128, Exp: 0}, Uint64ToFixed64(1), 64},
		{ExpiredValue{Base: 128, Exp: 0}, Uint64ToFixed64(2), 32},
		{ExpiredValue{Base: 128, Exp: 2}, Uint64ToFixed64(2), 128},
		{ExpiredValue{Base: 128, Exp: 2}, Uint64ToFixed64(3), 64},
	}
	for _, c := range cases {
		if got := c.input.Value(c.timeOffset); got != c.expect {
			t.Fatalf("Value mismatch, want=%d, got=%d", c.expect, got)
		}
	}
}

func TestValueAddition(t *testing.T) {
	var cases = []struct {
		input      ExpiredValue
		addend     int64
		timeOffset Fixed64
		expect     uint64
		expectNet  int64
	}{
		// Addition
		{ExpiredValue{Base: 128, Exp: 0}, 128, Uint64ToFixed64(0), 256, 128},
		{ExpiredValue{Base: 128, Exp: 2}, 128, Uint64ToFixed64(0), 640, 128},

		// Addition with offset
		{ExpiredValue{Base: 128, Exp: 0}, 128, Uint64ToFixed64(1), 192, 128},
		{ExpiredValue{Base: 128, Exp: 2}, 128, Uint64ToFixed64(1), 384, 128},
		{ExpiredValue{Base: 128, Exp: 2}, 128, Uint64ToFixed64(3), 192, 128},

		// Subtraction
		{ExpiredValue{Base: 128, Exp: 0}, -64, Uint64ToFixed64(0), 64, -64},
		{ExpiredValue{Base: 128, Exp: 0}, -128, Uint64ToFixed64(0), 0, -128},
		{ExpiredValue{Base: 128, Exp: 0}, -192, Uint64ToFixed64(0), 0, -128},

		// Subtraction with offset
		{ExpiredValue{Base: 128, Exp: 0}, -64, Uint64ToFixed64(1), 0, -64},
		{ExpiredValue{Base: 128, Exp: 0}, -128, Uint64ToFixed64(1), 0, -64},
		{ExpiredValue{Base: 128, Exp: 2}, -128, Uint64ToFixed64(1), 128, -128},
		{ExpiredValue{Base: 128, Exp: 2}, -128, Uint64ToFixed64(2), 0, -128},
	}
	for _, c := range cases {
		if net := c.input.Add(c.addend, c.timeOffset); net != c.expectNet {
			t.Fatalf("Net amount mismatch, want=%d, got=%d", c.expectNet, net)
		}
		if got := c.input.Value(c.timeOffset); got != c.expect {
			t.Fatalf("Value mismatch, want=%d, got=%d", c.expect, got)
		}
	}
}

func TestExpiredValueAddition(t *testing.T) {
	var cases = []struct {
		input      ExpiredValue
		another    ExpiredValue
		timeOffset Fixed64
		expect     uint64
	}{
		{ExpiredValue{Base: 128, Exp: 0}, ExpiredValue{Base: 128, Exp: 0}, Uint64ToFixed64(0), 256},
		{ExpiredValue{Base: 128, Exp: 1}, ExpiredValue{Base: 128, Exp: 0}, Uint64ToFixed64(0), 384},
		{ExpiredValue{Base: 128, Exp: 0}, ExpiredValue{Base: 128, Exp: 1}, Uint64ToFixed64(0), 384},
		{ExpiredValue{Base: 128, Exp: 0}, ExpiredValue{Base: 128, Exp: 0}, Uint64ToFixed64(1), 128},
	}
	for _, c := range cases {
		c.input.AddExp(c.another)
		if got := c.input.Value(c.timeOffset); got != c.expect {
			t.Fatalf("Value mismatch, want=%d, got=%d", c.expect, got)
		}
	}
}

func TestExpiredValueSubtraction(t *testing.T) {
	var cases = []struct {
		input      ExpiredValue
		another    ExpiredValue
		timeOffset Fixed64
		expect     uint64
	}{
		{ExpiredValue{Base: 128, Exp: 0}, ExpiredValue{Base: 128, Exp: 0}, Uint64ToFixed64(0), 0},
		{ExpiredValue{Base: 128, Exp: 0}, ExpiredValue{Base: 128, Exp: 1}, Uint64ToFixed64(0), 0},
		{ExpiredValue{Base: 128, Exp: 1}, ExpiredValue{Base: 128, Exp: 0}, Uint64ToFixed64(0), 128},
		{ExpiredValue{Base: 128, Exp: 1}, ExpiredValue{Base: 128, Exp: 0}, Uint64ToFixed64(1), 64},
	}
	for _, c := range cases {
		c.input.SubExp(c.another)
		if got := c.input.Value(c.timeOffset); got != c.expect {
			t.Fatalf("Value mismatch, want=%d, got=%d", c.expect, got)
		}
	}
}
