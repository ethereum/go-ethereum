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

func TestParseExpectedIPRejectsUnusableAddresses(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		ipv6    bool
		wantErr bool
	}{
		{name: "valid ipv4", raw: "192.0.2.1"},
		{name: "valid ipv6", raw: "2001:db8::1", ipv6: true},
		{name: "wrong family", raw: "2001:db8::1", wantErr: true},
		{name: "v4 mapped ipv6", raw: "::ffff:192.0.2.1", ipv6: true, wantErr: true},
		{name: "zoned ipv6", raw: "fe80::1%eth0", ipv6: true, wantErr: true},
		{name: "unspecified ipv4", raw: "0.0.0.0", wantErr: true},
		{name: "unspecified ipv6", raw: "::", ipv6: true, wantErr: true},
		{name: "multicast ipv4", raw: "224.0.0.1", wantErr: true},
		{name: "multicast ipv6", raw: "ff02::1", ipv6: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseExpectedIP(tt.raw, tt.ipv6)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
