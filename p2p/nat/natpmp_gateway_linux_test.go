// Copyright 2026 The go-ethereum Authors
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

//go:build linux

package nat

import (
	"net"
	"testing"
)

func TestParseLinuxRouteTable(t *testing.T) {
	data := []byte(`Iface	Destination Gateway 	Flags RefCnt Use Metric Mask MTU Window IRTT
eth0	00000000	0101A8C0	0003	0	0	100	00000000	0	0	0
eth0	0001A8C0	00000000	0001	0	0	100	00FFFFFF	0	0	0
wlan0	00000000	FE01A8C0	0003	0	0	600	00000000	0	0	0
`)
	got := parseLinuxRouteTable(data)
	want := []net.IP{
		net.IPv4(192, 168, 1, 1).To4(),
		net.IPv4(192, 168, 1, 254).To4(),
	}
	if len(got) != len(want) {
		t.Fatalf("got %d gateways, want %d", len(got), len(want))
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Fatalf("gateway %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestParseLinuxRouteHexIPv4(t *testing.T) {
	got, err := parseLinuxRouteHexIPv4("0101A8C0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := net.IPv4(192, 168, 1, 1).To4()
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
