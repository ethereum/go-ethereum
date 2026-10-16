// Copyright 2018 The go-ethereum Authors
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
	crand "crypto/rand"
	"fmt"
	"math/rand/v2"
	"net/netip"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

const (
	opStatement = iota
	opContact
	opPredict
	opCheckFullCone
)

type iptrackTestEvent struct {
	op       int
	time     int // absolute, in milliseconds
	ip, from string
}

func TestIPTracker(t *testing.T) {
	tests := map[string][]iptrackTestEvent{
		"minStatements": {
			{opPredict, 0, "", ""},
			{opStatement, 0, "127.0.0.1:8000", "127.0.0.2"},
			{opPredict, 1000, "", ""},
			{opStatement, 1000, "127.0.0.1:8000", "127.0.0.3"},
			{opPredict, 1000, "", ""},
			{opStatement, 1000, "127.0.0.1:8000", "127.0.0.4"},
			{opPredict, 1000, "127.0.0.1:8000", ""},
		},
		"window": {
			{opStatement, 0, "127.0.0.1:8000", "127.0.0.2"},
			{opStatement, 2000, "127.0.0.1:8000", "127.0.0.3"},
			{opStatement, 3000, "127.0.0.1:8000", "127.0.0.4"},
			{opPredict, 10000, "127.0.0.1:8000", ""},
			{opPredict, 10001, "", ""}, // first statement expired
			{opStatement, 10100, "127.0.0.1:8000", "127.0.0.2"},
			{opPredict, 10200, "127.0.0.1:8000", ""},
		},
		"sameNetwork": {
			{opStatement, 0, "1.2.3.4:8000", "5.6.7.1"},
			{opStatement, 0, "1.2.3.4:8000", "5.6.7.2"},
			{opStatement, 0, "1.2.3.4:8000", "5.6.7.3"},
			{opStatement, 0, "1.2.3.4:8000", "5.6.7.4"},
			{opPredict, 0, "", ""},
			{opStatement, 0, "1.2.3.4:8000", "5.6.8.1"},
			{opStatement, 0, "1.2.3.4:8000", "5.6.9.1"},
			{opPredict, 0, "1.2.3.4:8000", ""},
		},
		"sameNetworkIPv6": {
			{opStatement, 0, "[2001:db8::1]:8000", "2001:db9::1"},
			{opStatement, 0, "[2001:db8::1]:8000", "2001:db9::2"},
			{opStatement, 0, "[2001:db8::1]:8000", "2001:db9:0:1::1"},
			{opPredict, 0, "", ""},
			{opStatement, 0, "[2001:db8::1]:8000", "2001:dba::1"},
			{opStatement, 0, "[2001:db8::1]:8000", "2001:dbb::1"},
			{opPredict, 0, "[2001:db8::1]:8000", ""},
		},
		"networkIPv6": {
			{opStatement, 0, "[2001:db8::1]:8000", "2001:db9:1::1"},
			{opStatement, 0, "[2001:db8::1]:8000", "2001:db9:2::1"},
			{opStatement, 0, "[2001:db8::1]:8000", "2001:db9:3::1"},
			{opPredict, 0, "[2001:db8::1]:8000", ""},
		},
		"lan": {
			{opStatement, 0, "1.2.3.4:8000", "192.168.0.1"},
			{opStatement, 0, "1.2.3.4:8000", "192.168.0.2"},
			{opStatement, 0, "1.2.3.4:8000", "192.168.0.3"},
			{opPredict, 0, "1.2.3.4:8000", ""},
		},
		"fullconeMapped": {
			{opContact, 0, "", "127.0.0.2"},
			{opStatement, 10, "127.0.0.1:8000", "::ffff:127.0.0.2"},
			{opCheckFullCone, 20, "false", ""},
		},
		"otherFamily": {
			{opStatement, 0, "1.2.3.4:8000", "2001:db9::1"},
			{opStatement, 0, "1.2.3.4:8000", "2001:dba::1"},
			{opStatement, 0, "1.2.3.4:8000", "2001:dbb::1"},
			{opPredict, 0, "", ""},
			{opStatement, 0, "[2001:db8::1]:8000", "5.6.7.1"},
			{opStatement, 0, "[2001:db8::1]:8000", "5.6.8.1"},
			{opStatement, 0, "[2001:db8::1]:8000", "5.6.9.1"},
			{opPredict, 0, "", ""},
			{opStatement, 0, "1.2.3.4:8000", "::ffff:5.6.7.1"},
			{opStatement, 0, "1.2.3.4:8000", "::ffff:5.6.8.1"},
			{opStatement, 0, "1.2.3.4:8000", "::ffff:5.6.9.1"},
			{opPredict, 0, "1.2.3.4:8000", ""},
		},
		"fullcone": {
			{opContact, 0, "", "127.0.0.2"},
			{opStatement, 10, "127.0.0.1:8000", "127.0.0.2"},
			{opContact, 2000, "", "127.0.0.3"},
			{opStatement, 2010, "127.0.0.1:8000", "127.0.0.3"},
			{opContact, 3000, "", "127.0.0.4"},
			{opStatement, 3010, "127.0.0.1:8000", "127.0.0.4"},
			{opCheckFullCone, 3500, "false", ""},
		},
		"fullcone_2": {
			{opContact, 0, "", "127.0.0.2"},
			{opStatement, 10, "127.0.0.1:8000", "127.0.0.2"},
			{opContact, 2000, "", "127.0.0.3"},
			{opStatement, 2010, "127.0.0.1:8000", "127.0.0.3"},
			{opStatement, 3000, "127.0.0.1:8000", "127.0.0.4"},
			{opContact, 3010, "", "127.0.0.4"},
			{opCheckFullCone, 3500, "true", ""},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) { runIPTrackerTest(t, test) })
	}
}

func runIPTrackerTest(t *testing.T, evs []iptrackTestEvent) {
	var (
		clock mclock.Simulated
		it    = NewIPTracker(10*time.Second, 10*time.Second, 3)
	)
	it.clock = &clock
	for i, ev := range evs {
		evtime := time.Duration(ev.time) * time.Millisecond
		clock.Run(evtime - time.Duration(clock.Now()))
		switch ev.op {
		case opStatement:
			it.AddStatement(netip.MustParseAddr(ev.from), netip.MustParseAddrPort(ev.ip))
		case opContact:
			it.AddContact(netip.MustParseAddr(ev.from))
		case opPredict:
			pred := it.PredictEndpoint()
			if ev.ip == "" {
				if pred.IsValid() {
					t.Errorf("op %d: wrong prediction %v, expected invalid", i, pred)
				}
			} else {
				if pred != netip.MustParseAddrPort(ev.ip) {
					t.Errorf("op %d: wrong prediction %v, want %q", i, pred, ev.ip)
				}
			}
		case opCheckFullCone:
			pred := fmt.Sprintf("%t", it.PredictFullConeNAT())
			if pred != ev.ip {
				t.Errorf("op %d: wrong prediction %s, want %s", i, pred, ev.ip)
			}
		}
	}
}

// This checks that old statements and contacts are GCed even if Predict* isn't called.
func TestIPTrackerForceGC(t *testing.T) {
	var (
		clock  mclock.Simulated
		window = 10 * time.Second
		rate   = 50 * time.Millisecond
		max    = int(window/rate) + 1
		it     = NewIPTracker(window, window, 3)
	)
	it.clock = &clock

	for i := 0; i < 5*max; i++ {
		var e1, e2 [4]byte
		crand.Read(e1[:])
		crand.Read(e2[:])
		it.AddStatement(netip.AddrFrom4(e1), netip.AddrPortFrom(netip.AddrFrom4(e2), 9000))
		it.AddContact(netip.AddrFrom4(e1))
		clock.Run(rate)
	}
	if len(it.contact) > 2*max {
		t.Errorf("contacts not GCed, have %d", len(it.contact))
	}
	if len(it.statements) > 2*max {
		t.Errorf("statements not GCed, have %d", len(it.statements))
	}
}

// This checks that the endpoint counts kept by AddStatement and GC match
// the statements.
func TestIPTrackerCounts(t *testing.T) {
	var (
		clock mclock.Simulated
		it    = NewIPTracker(10*time.Second, 10*time.Second, 2)
		rng   = rand.New(rand.NewPCG(1, 2))
	)
	it.clock = &clock
	for i := 0; i < 5000; i++ {
		host := netip.AddrFrom4([4]byte{1, byte(rng.IntN(8)), byte(rng.IntN(4)), byte(rng.IntN(4))})
		endpoint := netip.AddrPortFrom(netip.AddrFrom4([4]byte{2, 0, 0, byte(rng.IntN(4))}), 9000)
		it.AddStatement(host, endpoint)
		clock.Run(time.Duration(rng.IntN(100)) * time.Millisecond)

		pred := it.PredictEndpoint()
		counts, maxcount := countStatements(it)
		if maxcount < it.minStatements {
			if pred.IsValid() {
				t.Fatalf("op %d: prediction %v with max count %d", i, pred, maxcount)
			}
		} else if counts[pred] != maxcount {
			t.Fatalf("op %d: prediction %v has count %d, max is %d", i, pred, counts[pred], maxcount)
		}
		for ep, c := range counts {
			if it.counts.count[ep] != c {
				t.Fatalf("op %d: endpoint %v count %d, want %d", i, ep, it.counts.count[ep], c)
			}
		}
		if len(it.counts.count) != len(counts) {
			t.Fatalf("op %d: %d endpoints counted, want %d", i, len(it.counts.count), len(counts))
		}
		checkEndpointSets(t, &it.counts)
	}

	// All statements expire.
	clock.Run(11 * time.Second)
	if pred := it.PredictEndpoint(); pred.IsValid() {
		t.Fatalf("prediction %v after all statements expired", pred)
	}
	if len(it.statements) != 0 || it.order.Len() != 0 || len(it.votes) != 0 || len(it.counts.count) != 0 || len(it.counts.byCount) != 1 {
		t.Fatalf("state left after all statements expired: %d statements, %d votes, %d endpoints, %d sets",
			len(it.statements), len(it.votes), len(it.counts.count), len(it.counts.byCount))
	}
	it.AddStatement(netip.MustParseAddr("1.0.0.1"), netip.MustParseAddrPort("2.0.0.1:9000"))
	it.AddStatement(netip.MustParseAddr("1.0.1.1"), netip.MustParseAddrPort("2.0.0.1:9000"))
	if pred := it.PredictEndpoint(); pred != netip.MustParseAddrPort("2.0.0.1:9000") {
		t.Fatalf("wrong prediction %v after reuse", pred)
	}
}

// checkEndpointSets checks that every endpoint is in the set of its count.
func checkEndpointSets(t *testing.T, ec *endpointCounts) {
	t.Helper()
	n := 0
	for c, set := range ec.byCount {
		for i, ep := range set {
			if ec.count[ep] != c || ec.pos[ep] != i {
				t.Fatalf("endpoint %v at set %d index %d, has count %d pos %d", ep, c, i, ec.count[ep], ec.pos[ep])
			}
		}
		n += len(set)
	}
	if n != len(ec.count) || n != len(ec.pos) {
		t.Fatalf("%d endpoints in sets, %d counts, %d positions", n, len(ec.count), len(ec.pos))
	}
	if top := len(ec.byCount) - 1; top > 0 && len(ec.byCount[top]) == 0 {
		t.Fatalf("highest set %d is empty", top)
	}
}

// countStatements counts the networks stating each endpoint.
func countStatements(it *IPTracker) (map[netip.AddrPort]int, int) {
	var (
		counted  = make(map[netStatement]bool)
		counts   = make(map[netip.AddrPort]int)
		maxcount int
	)
	for host, el := range it.statements {
		st := el.Value.(*ipStatement)
		key := netStatement{statementNetwork(host), st.endpoint}
		if !counted[key] {
			counted[key] = true
			counts[st.endpoint]++
			maxcount = max(maxcount, counts[st.endpoint])
		}
	}
	return counts, maxcount
}

func BenchmarkIPTrackerPredict(b *testing.B) {
	var (
		clock mclock.Simulated
		it    = NewIPTracker(5*time.Minute, 10*time.Minute, 10)
	)
	it.clock = &clock
	for i := 0; i < 100000; i++ {
		host := netip.AddrFrom16([16]byte{0x20, 0x01, 0x0d, 0xb8, byte(i >> 16), byte(i >> 8), byte(i), 1})
		it.AddStatement(host, netip.MustParseAddrPort("[2001:db9::1]:30303"))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		it.PredictEndpoint()
	}
}

// This checks that statements with an invalid host or endpoint are ignored.
func TestIPTrackerInvalidStatement(t *testing.T) {
	it := NewIPTracker(10*time.Second, 10*time.Second, 1)
	it.AddStatement(netip.Addr{}, netip.MustParseAddrPort("[2001:db8::1]:8000"))
	it.AddStatement(netip.MustParseAddr("2001:db9::1"), netip.AddrPort{})
	if len(it.statements) != 0 {
		t.Fatalf("invalid statements recorded: %v", it.statements)
	}
}
