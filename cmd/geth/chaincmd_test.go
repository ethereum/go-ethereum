// Copyright 2025 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import "testing"

func TestParseRange(t *testing.T) {
	var cases = []struct {
		input    string
		valid    bool
		expStart uint64
		expEnd   uint64
	}{
		{
			input:    "0",
			valid:    true,
			expStart: 0,
			expEnd:   0,
		},
		{
			input:    "500",
			valid:    true,
			expStart: 500,
			expEnd:   500,
		},
		{
			input:    "-1",
			valid:    false,
			expStart: 0,
			expEnd:   0,
		},
		{
			input:    "1-1",
			valid:    true,
			expStart: 1,
			expEnd:   1,
		},
		{
			input:    "0-1",
			valid:    true,
			expStart: 0,
			expEnd:   1,
		},
		{
			input:    "1-0",
			valid:    false,
			expStart: 0,
			expEnd:   0,
		},
		{
			input:    "1-1000",
			valid:    true,
			expStart: 1,
			expEnd:   1000,
		},
		{
			input:    "1-1-",
			valid:    false,
			expStart: 0,
			expEnd:   0,
		},
		{
			input:    "-1-1",
			valid:    false,
			expStart: 0,
			expEnd:   0,
		},
	}
	for _, c := range cases {
		start, end, valid := parseRange(c.input)
		if valid != c.valid {
			t.Errorf("Unexpected result, want: %t, got: %t", c.valid, valid)
			continue
		}
		if valid {
			if c.expStart != start {
				t.Errorf("Unexpected start, want: %d, got: %d", c.expStart, start)
			}
			if c.expEnd != end {
				t.Errorf("Unexpected end, want: %d, got: %d", c.expEnd, end)
			}
		}
	}
}
