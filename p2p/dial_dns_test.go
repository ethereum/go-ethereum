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

package p2p

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

func TestDialDNSAddressUpdates(t *testing.T) {
	ip4 := netip.MustParseAddr("192.0.2.1")
	ip6 := netip.MustParseAddr("2001:db8::1")
	lookupErr := errors.New("DNS lookup failed")
	tests := []struct {
		name                     string
		old4, old6, want4, want6 netip.Addr
		answers                  []netip.Addr
		lookupErr, wantErr       error
		unchanged                bool
	}{
		{name: "IPv4 to IPv6", old4: ip4, answers: []netip.Addr{ip6}, want6: ip6},
		{name: "IPv6 to IPv4", old6: ip6, answers: []netip.Addr{ip4}, want4: ip4},
		{name: "dual stack to IPv6", old4: ip4, old6: ip6, answers: []netip.Addr{ip6}, want6: ip6},
		{name: "dual stack to IPv4", old4: ip4, old6: ip6, answers: []netip.Addr{ip4}, want4: ip4},
		{name: "IPv4 to dual stack", old4: ip4, answers: []netip.Addr{ip4, ip6}, want4: ip4, want6: ip6},
		{name: "IPv6 to dual stack", old6: ip6, answers: []netip.Addr{ip4, ip6}, want4: ip4, want6: ip6},
		{name: "unchanged", old4: ip4, old6: ip6, answers: []netip.Addr{ip4, ip6}, want4: ip4, want6: ip6, unchanged: true},
		{name: "lookup error", old4: ip4, old6: ip6, lookupErr: lookupErr, wantErr: lookupErr, want4: ip4, want6: ip6, unchanged: true},
		{name: "empty answer", old4: ip4, old6: ip6, wantErr: errNoResolvedIP, want4: ip4, want6: ip6, unchanged: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rec enr.Record
			rec.Set(enr.TCP(30303))
			rec.Set(enr.UDP(30301))
			rec.Set(enr.WithEntry("metadata", "preserved"))
			if tt.old4.IsValid() {
				rec.Set(enr.IPv4Addr(tt.old4))
			}
			if tt.old6.IsValid() {
				rec.Set(enr.IPv6Addr(tt.old6))
			}
			rec.SetSeq(42)
			node := enode.SignNull(&rec, uintID(1)).WithHostname("peer.example")
			d := &dialScheduler{
				dialConfig: dialConfig{log: testlog.Logger(t, log.LvlTrace)},
				dnsLookupFunc: func(_ context.Context, network, name string) ([]netip.Addr, error) {
					if network != "ip" || name != node.Hostname() {
						t.Fatalf("unexpected lookup: %s %s", network, name)
					}
					return tt.answers, tt.lookupErr
				},
			}
			resolved, err := d.dnsResolveHostname(node)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error: have %v, want %v", err, tt.wantErr)
			}
			if tt.unchanged && resolved != node {
				t.Fatal("unchanged lookup replaced the original node")
			}
			checkIPs := func(n *enode.Node, want4, want6 netip.Addr) {
				t.Helper()
				var got4, got6 netip.Addr
				err4 := n.Load((*enr.IPv4Addr)(&got4))
				err6 := n.Load((*enr.IPv6Addr)(&got6))
				if got4 != want4 || got6 != want6 || (!want4.IsValid() && !enr.IsNotFound(err4)) || (!want6.IsValid() && !enr.IsNotFound(err6)) {
					t.Errorf("IP entries: have %v/%v, want %v/%v", got4, got6, want4, want6)
				}
			}
			checkIPs(resolved, tt.want4, tt.want6)
			checkIPs(node, tt.old4, tt.old6)
			wantIP := tt.want4
			if !wantIP.IsValid() {
				wantIP = tt.want6
			}
			endpoint, ok := resolved.TCPEndpoint()
			if !ok || endpoint != netip.AddrPortFrom(wantIP, 30303) {
				t.Errorf("TCP endpoint: have %v, want %v:30303", endpoint, wantIP)
			}
			var metadata string
			if err := resolved.Load(enr.WithEntry("metadata", &metadata)); err != nil || metadata != "preserved" {
				t.Errorf("metadata was not preserved: %q, %v", metadata, err)
			}
			if resolved.ID() != node.ID() || resolved.Seq() != node.Seq() || resolved.Hostname() != node.Hostname() || resolved.UDP() != 30301 {
				t.Fatal("node identity, sequence, hostname or UDP port changed")
			}
		})
	}
}
