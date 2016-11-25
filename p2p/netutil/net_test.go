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

package netutil

import (
	"net"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestParseNetlist(t *testing.T) {
	var tests = []struct {
		input    string
		wantErr  error
		wantList *Netlist
	}{
		{
			input:    "",
			wantList: &Netlist{},
		},
		{
			input:    "127.0.0.0/8",
			wantErr:  nil,
			wantList: &Netlist{{IP: net.IP{127, 0, 0, 0}, Mask: net.CIDRMask(8, 32)}},
		},
		{
			input:   "127.0.0.0/44",
			wantErr: &net.ParseError{Type: "CIDR address", Text: "127.0.0.0/44"},
		},
		{
			input: "127.0.0.0/16, 23.23.23.23/24,",
			wantList: &Netlist{
				{IP: net.IP{127, 0, 0, 0}, Mask: net.CIDRMask(16, 32)},
				{IP: net.IP{23, 23, 23, 0}, Mask: net.CIDRMask(24, 32)},
			},
		},
	}

	for _, test := range tests {
		l, err := ParseNetlist(test.input)
		if !reflect.DeepEqual(err, test.wantErr) {
			t.Errorf("%q: got error %q, want %q", test.input, err, test.wantErr)
			continue
		}
		if !reflect.DeepEqual(l, test.wantList) {
			spew.Dump(l)
			spew.Dump(test.wantList)
			t.Errorf("%q: got %v, want %v", test.input, l, test.wantList)
		}
	}
}

func TestNilNetListContains(t *testing.T) {
	var list *Netlist
	checkContains(t, list.Contains, nil, []string{"1.2.3.4"})
}

func TestIsLAN(t *testing.T) {
	checkContains(t, IsLAN,
		[]string{ // included
			"0.0.0.0",
			"0.2.0.8",
			"127.0.0.1",
			"10.0.1.1",
			"10.22.0.3",
			"172.31.252.251",
			"192.168.1.4",
			"fe80::f4a1:8eff:fec5:9d9d",
			"febf::ab32:2233",
			"fc00::4",
		},
		[]string{ // excluded
			"192.0.2.1",
			"1.0.0.0",
			"172.32.0.1",
			"fec0::2233",
		},
	)
}

func TestIsSpecialNetwork(t *testing.T) {
	checkContains(t, IsSpecialNetwork,
		[]string{ // included
			"192.0.2.1",
			"192.0.2.44",
			"2001:db8:85a3:8d3:1319:8a2e:370:7348",
			"255.255.255.255",
			"224.0.0.22", // IPv4 multicast
			"ff05::1:3",  // IPv6 multicast
		},
		[]string{ // excluded
			"192.0.3.1",
			"1.0.0.0",
			"172.32.0.1",
			"fec0::2233",
		},
	)
}

func checkContains(t *testing.T, fn func(net.IP) bool, inc, exc []string) {
	for _, s := range inc {
		if !fn(parseIP(s)) {
			t.Error("returned false for included address", s)
		}
	}
	for _, s := range exc {
		if fn(parseIP(s)) {
			t.Error("returned true for excluded address", s)
		}
	}
}

func parseIP(s string) net.IP {
	ip := net.ParseIP(s)
	if ip == nil {
		panic("invalid " + s)
	}
	return ip
}

func TestCheckRelayIP(t *testing.T) {
	tests := []struct {
		sender, addr string
		want         error
	}{
		{"127.0.0.1", "0.0.0.0", errUnspecified},
		{"192.168.0.1", "0.0.0.0", errUnspecified},
		{"23.55.1.242", "0.0.0.0", errUnspecified},
		{"127.0.0.1", "255.255.255.255", errSpecial},
		{"192.168.0.1", "255.255.255.255", errSpecial},
		{"23.55.1.242", "255.255.255.255", errSpecial},
		{"192.168.0.1", "127.0.2.19", errLoopback},
		{"23.55.1.242", "192.168.0.1", errLAN},

		{"127.0.0.1", "127.0.2.19", nil},
		{"127.0.0.1", "192.168.0.1", nil},
		{"127.0.0.1", "23.55.1.242", nil},
		{"192.168.0.1", "192.168.0.1", nil},
		{"192.168.0.1", "23.55.1.242", nil},
		{"23.55.1.242", "23.55.1.242", nil},
	}

	for _, test := range tests {
		err := CheckRelayIP(parseIP(test.sender), parseIP(test.addr))
		if err != test.want {
			t.Errorf("%s from %s: got %q, want %q", test.addr, test.sender, err, test.want)
		}
	}
}

func BenchmarkCheckRelayIP(b *testing.B) {
	sender := parseIP("23.55.1.242")
	addr := parseIP("23.55.1.2")
	for i := 0; i < b.N; i++ {
		CheckRelayIP(sender, addr)
	}
}
