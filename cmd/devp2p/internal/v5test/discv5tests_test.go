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
	"net"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enr"
)

func TestValidateEndpointPairsAllowsIPv6UDPFallback(t *testing.T) {
	record := newTestRecord(
		enr.IPv6(net.ParseIP("2001:db8::1").To16()),
		enr.UDP(30303),
	)
	summary, err := validateEndpointPairs(record)
	if err != nil {
		t.Fatalf("expected IPv6 endpoint with UDP fallback to be valid: %v", err)
	}
	if !strings.Contains(summary, "ip6=2001:db8::1 udp6=30303") {
		t.Fatalf("unexpected endpoint summary %q", summary)
	}
}

func TestValidateEndpointPairsAllowsRedundantUDPWithIPv6Endpoint(t *testing.T) {
	record := newTestRecord(
		enr.IPv6(net.ParseIP("2001:db8::1").To16()),
		enr.UDP(30303),
		enr.UDP6(30304),
	)
	summary, err := validateEndpointPairs(record)
	if err != nil {
		t.Fatalf("expected IPv6 endpoint with redundant UDP to be valid: %v", err)
	}
	if !strings.Contains(summary, "ip6=2001:db8::1 udp6=30304") {
		t.Fatalf("unexpected endpoint summary %q", summary)
	}
}

func TestValidateEndpointPairsRejectsUDPWithoutIPAddress(t *testing.T) {
	record := newTestRecord(enr.UDP(30303))
	_, err := validateEndpointPairs(record)
	if err == nil || !strings.Contains(err.Error(), "udp is present without ip or ip6") {
		t.Fatalf("expected UDP without IP address to be rejected, got %v", err)
	}
}

func TestRequireExpectedEndpointEntriesDoesNotRequireTCP(t *testing.T) {
	record := newTestRecord(
		enr.IPv4(net.ParseIP("203.0.113.1").To4()),
		enr.IPv6(net.ParseIP("2001:db8::1").To16()),
		enr.UDP(30303),
	)
	summary, err := requireExpectedEndpointEntries(record, "203.0.113.1", "2001:db8::1")
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

func TestRequireExpectedEndpointEntriesNeedsIPv6DiscoveryPort(t *testing.T) {
	record := newTestRecord(enr.IPv6(net.ParseIP("2001:db8::1").To16()))
	_, err := requireExpectedEndpointEntries(record, "", "2001:db8::1")
	if err == nil || !strings.Contains(err.Error(), "missing udp6 entry or udp fallback") {
		t.Fatalf("expected missing IPv6 discovery port error, got %v", err)
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
