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

package v5test

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

func TestCheckSelfRecordAllowsIPv6UDPFallback(t *testing.T) {
	node := newTestNode(
		enr.IPv6Addr(mustAddr("2001:db8::1")),
		enr.UDP(30303),
	)
	summary, err := checkSelfRecord(node, netip.Addr{}, netip.Addr{})
	if err != nil {
		t.Fatalf("expected IPv6 endpoint with UDP fallback to be valid: %v", err)
	}
	if !strings.Contains(summary, "ip6=2001:db8::1 udp6=30303") {
		t.Fatalf("unexpected endpoint summary %q", summary)
	}
}

func TestCheckSelfRecordAllowsRedundantUDPWithIPv6Endpoint(t *testing.T) {
	node := newTestNode(
		enr.IPv6Addr(mustAddr("2001:db8::1")),
		enr.UDP(30303),
		enr.UDP6(30304),
	)
	summary, err := checkSelfRecord(node, netip.Addr{}, netip.Addr{})
	if err != nil {
		t.Fatalf("expected IPv6 endpoint with redundant UDP to be valid: %v", err)
	}
	if !strings.Contains(summary, "ip6=2001:db8::1 udp6=30304") {
		t.Fatalf("unexpected endpoint summary %q", summary)
	}
}

func TestCheckSelfRecordRejectsUDPWithoutIPAddress(t *testing.T) {
	node := newTestNode(enr.UDP(30303))
	_, err := checkSelfRecord(node, netip.Addr{}, netip.Addr{})
	if err == nil || !strings.Contains(err.Error(), "udp is present without ip or ip6") {
		t.Fatalf("expected UDP without IP address to be rejected, got %v", err)
	}
}

func TestCheckSelfRecordDoesNotRequireTCP(t *testing.T) {
	node := newTestNode(
		enr.IPv4Addr(mustAddr("203.0.113.1")),
		enr.IPv6Addr(mustAddr("2001:db8::1")),
		enr.UDP(30303),
	)
	summary, err := checkSelfRecord(node, mustAddr("203.0.113.1"), mustAddr("2001:db8::1"))
	if err != nil {
		t.Fatalf("expected discovery endpoints without TCP entries to be valid: %v", err)
	}
	if !strings.Contains(summary, "ip=203.0.113.1 udp=30303") {
		t.Fatalf("summary missing IPv4 endpoint: %q", summary)
	}
	if !strings.Contains(summary, "ip6=2001:db8::1 udp6=30303") {
		t.Fatalf("summary missing IPv6 endpoint: %q", summary)
	}
}

func TestCheckSelfRecordNeedsIPv6DiscoveryPort(t *testing.T) {
	node := newTestNode(enr.IPv6Addr(mustAddr("2001:db8::1")))
	_, err := checkSelfRecord(node, netip.Addr{}, mustAddr("2001:db8::1"))
	if err == nil || !strings.Contains(err.Error(), "missing udp6 entry or udp fallback") {
		t.Fatalf("expected missing IPv6 discovery port error, got %v", err)
	}
}

func TestCheckSelfRecordAllowsZeroSequence(t *testing.T) {
	record := newTestRecord(
		enr.IPv4Addr(mustAddr("203.0.113.1")),
		enr.UDP(30303),
	)
	record.SetSeq(0)
	node := enode.SignNull(record, enode.ID{})
	if _, err := checkSelfRecord(node, netip.Addr{}, netip.Addr{}); err != nil {
		t.Fatalf("expected sequence zero to be accepted: %v", err)
	}
}

func newTestRecord(entries ...enr.Entry) *enr.Record {
	var record enr.Record
	record.SetSeq(1)
	for _, entry := range entries {
		record.Set(entry)
	}
	return &record
}

func newTestNode(entries ...enr.Entry) *enode.Node {
	return enode.SignNull(newTestRecord(entries...), enode.ID{})
}

func mustAddr(raw string) netip.Addr {
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		panic(err)
	}
	return addr
}
