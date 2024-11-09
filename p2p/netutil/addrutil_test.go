// Copyright 2024 The go-ethereum Authors
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
	"net/netip"
	"path/filepath"
	"testing"
)

// customNetAddr is a custom implementation of net.Addr for testing purposes.
type customNetAddr struct{}

func (c *customNetAddr) Network() string { return "custom" }
func (c *customNetAddr) String() string  { return "custom" }

func TestAddrAddr(t *testing.T) {
	tempDir := t.TempDir()
	tests := []struct {
		name string
		addr net.Addr
		want netip.Addr
	}{
		{
			name: "IPAddr IPv4",
			addr: &net.IPAddr{IP: net.ParseIP("192.0.2.1")},
			want: netip.MustParseAddr("192.0.2.1"),
		},
		{
			name: "IPAddr IPv6",
			addr: &net.IPAddr{IP: net.ParseIP("2001:db8::1")},
			want: netip.MustParseAddr("2001:db8::1"),
		},
		{
			name: "TCPAddr IPv4",
			addr: &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 8080},
			want: netip.MustParseAddr("192.0.2.1"),
		},
		{
			name: "TCPAddr IPv6",
			addr: &net.TCPAddr{IP: net.ParseIP("2001:db8::1"), Port: 8080},
			want: netip.MustParseAddr("2001:db8::1"),
		},
		{
			name: "UDPAddr IPv4",
			addr: &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 8080},
			want: netip.MustParseAddr("192.0.2.1"),
		},
		{
			name: "UDPAddr IPv6",
			addr: &net.UDPAddr{IP: net.ParseIP("2001:db8::1"), Port: 8080},
			want: netip.MustParseAddr("2001:db8::1"),
		},
		{
			name: "Unsupported Addr type",
			addr: &net.UnixAddr{Name: filepath.Join(tempDir, "test.sock"), Net: "unix"},
			want: netip.Addr{},
		},
		{
			name: "Nil input",
			addr: nil,
			want: netip.Addr{},
		},
		{
			name: "Custom net.Addr implementation",
			addr: &customNetAddr{},
			want: netip.Addr{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddrAddr(tt.addr); got != tt.want {
				t.Errorf("AddrAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIPToAddr(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		want netip.Addr
	}{
		{
			name: "IPv4",
			ip:   net.ParseIP("192.0.2.1"),
			want: netip.MustParseAddr("192.0.2.1"),
		},
		{
			name: "IPv6",
			ip:   net.ParseIP("2001:db8::1"),
			want: netip.MustParseAddr("2001:db8::1"),
		},
		{
			name: "Invalid IP",
			ip:   net.IP{1, 2, 3},
			want: netip.Addr{},
		},
		{
			name: "Invalid IP (5 octets)",
			ip:   net.IP{192, 0, 2, 1, 1},
			want: netip.Addr{},
		},
		{
			name: "IPv4-mapped IPv6",
			ip:   net.ParseIP("::ffff:192.0.2.1"),
			want: netip.MustParseAddr("192.0.2.1"),
		},
		{
			name: "Nil input",
			ip:   nil,
			want: netip.Addr{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IPToAddr(tt.ip); got != tt.want {
				t.Errorf("IPToAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}
