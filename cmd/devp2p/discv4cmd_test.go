// Copyright 2026 The go-ethereum Authors
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

func TestParseExtAddrPortRange(t *testing.T) {
	tests := []struct {
		input string
		ok    bool
		port  int
	}{
		{input: "127.0.0.1", ok: true, port: 0},
		{input: "127.0.0.1:30303", ok: true, port: 30303},
		{input: "127.0.0.1:65535", ok: true, port: 65535},
		{input: "127.0.0.1:65536", ok: false},
		{input: "127.0.0.1:-1", ok: false},
		{input: "[2001:db8::1]:30303", ok: true, port: 30303},
	}
	for _, tc := range tests {
		_, port, ok := parseExtAddr(tc.input)
		if ok != tc.ok {
			t.Fatalf("parseExtAddr(%q) ok=%v, want %v", tc.input, ok, tc.ok)
		}
		if ok && port != tc.port {
			t.Fatalf("parseExtAddr(%q) port=%d, want %d", tc.input, port, tc.port)
		}
	}
}
