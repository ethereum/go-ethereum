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
	"container/list"
	"net/netip"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

// IPTracker predicts the external endpoint, i.e. IP address and port, of the local host
// based on statements made by other hosts.
type IPTracker struct {
	window        time.Duration
	contactWindow time.Duration
	minStatements int
	clock         mclock.Clock
	statements    map[netip.Addr]*list.Element // of *ipStatement, in order
	order         list.List                    // statements, oldest first
	votes         map[netStatement]int         // hosts per network and endpoint
	counts        endpointCounts               // networks per endpoint
	predicted     netip.AddrPort
	contact       map[netip.Addr]mclock.AbsTime
	lastContactGC mclock.AbsTime
}

type ipStatement struct {
	host     netip.Addr
	endpoint netip.AddrPort
	time     mclock.AbsTime
}

// netStatement is an endpoint stated by the hosts of one network.
type netStatement struct {
	network  netip.Prefix
	endpoint netip.AddrPort
}

// NewIPTracker creates an IP tracker.
//
// The window parameters configure the amount of past network events which are kept. The
// minStatements parameter enforces a minimum number of statements which must be recorded
// before any prediction is made. Higher values for these parameters decrease 'flapping' of
// predictions as network conditions change. Window duration values should typically be in
// the range of minutes.
func NewIPTracker(window, contactWindow time.Duration, minStatements int) *IPTracker {
	return &IPTracker{
		window:        window,
		contactWindow: contactWindow,
		statements:    make(map[netip.Addr]*list.Element),
		votes:         make(map[netStatement]int),
		counts:        newEndpointCounts(),
		minStatements: minStatements,
		contact:       make(map[netip.Addr]mclock.AbsTime),
		clock:         mclock.System{},
	}
}

// PredictFullConeNAT checks whether the local host is behind full cone NAT. It predicts by
// checking whether any statement has been received from a node we didn't contact before
// the statement was made.
func (it *IPTracker) PredictFullConeNAT() bool {
	now := it.clock.Now()
	it.gcContact(now)
	it.gcStatements(now)
	for el := it.order.Front(); el != nil; el = el.Next() {
		st := el.Value.(*ipStatement)
		if c, ok := it.contact[st.host]; !ok || c > st.time {
			return true
		}
	}
	return false
}

// PredictEndpoint returns the current prediction of the external endpoint.
func (it *IPTracker) PredictEndpoint() netip.AddrPort {
	it.gcStatements(it.clock.Now())

	// The current strategy is simple: find the endpoint with most statements.
	// Hosts in the same network count as one. The counts are kept up to date as
	// statements are added and removed, so this does not scan the statements.
	max, maxcount := it.counts.max()
	if maxcount == 0 || maxcount < it.minStatements {
		return netip.AddrPort{}
	}
	if it.counts.count[it.predicted] != maxcount {
		it.predicted = max
	}
	return it.predicted
}

// statementNetwork returns the network of a host making statements. Hosts in
// the same public /24 (IPv4) or /48 (IPv6) network are likely controlled by
// the same party, so their statements count as one. Hosts on the LAN, e.g. of
// a local test network, count individually.
func statementNetwork(host netip.Addr) netip.Prefix {
	bits := 48
	if host.Is4() {
		bits = 24
	}
	if AddrIsLAN(host) {
		bits = host.BitLen()
	}
	network, _ := host.Prefix(bits)
	return network
}

// AddStatement records that a certain host thinks our external endpoint is the one given.
// Statements about an endpoint of a different address family than the host are ignored.
func (it *IPTracker) AddStatement(host netip.Addr, endpoint netip.AddrPort) {
	host = host.Unmap()
	endpoint = netip.AddrPortFrom(endpoint.Addr().Unmap(), endpoint.Port())
	if !host.IsValid() || !endpoint.IsValid() || host.Is4() != endpoint.Addr().Is4() {
		return
	}
	now := it.clock.Now()
	if el, ok := it.statements[host]; ok {
		it.removeStatement(el)
	}
	it.statements[host] = it.order.PushBack(&ipStatement{host, endpoint, now})
	key := netStatement{statementNetwork(host), endpoint}
	if it.votes[key]++; it.votes[key] == 1 {
		it.counts.inc(endpoint)
	}
	it.gcStatements(now)
}

// AddContact records that a packet containing our endpoint information has been sent to a
// certain host.
func (it *IPTracker) AddContact(host netip.Addr) {
	host = host.Unmap()
	now := it.clock.Now()
	it.contact[host] = now
	if time.Duration(now-it.lastContactGC) >= it.contactWindow {
		it.gcContact(now)
	}
}

// gcStatements removes statements older than the window. Statements are kept in
// the order they were made, so this only visits the ones it removes.
func (it *IPTracker) gcStatements(now mclock.AbsTime) {
	cutoff := now.Add(-it.window)
	for {
		el := it.order.Front()
		if el == nil || el.Value.(*ipStatement).time >= cutoff {
			return
		}
		it.removeStatement(el)
	}
}

func (it *IPTracker) removeStatement(el *list.Element) {
	st := it.order.Remove(el).(*ipStatement)
	delete(it.statements, st.host)
	key := netStatement{statementNetwork(st.host), st.endpoint}
	if it.votes[key]--; it.votes[key] == 0 {
		delete(it.votes, key)
		it.counts.dec(st.endpoint)
	}
}

func (it *IPTracker) gcContact(now mclock.AbsTime) {
	it.lastContactGC = now
	cutoff := now.Add(-it.contactWindow)
	for host, ct := range it.contact {
		if ct < cutoff {
			delete(it.contact, host)
		}
	}
}

// endpointCounts counts the networks stating each endpoint. Counts change by one
// at a time, and endpoints are kept in sets by count, so an endpoint with the
// highest count is found in constant time.
type endpointCounts struct {
	count   map[netip.AddrPort]int // networks stating the endpoint
	pos     map[netip.AddrPort]int // index of the endpoint in byCount[count]
	byCount [][]netip.AddrPort     // endpoints by count; byCount[0] is unused
}

func newEndpointCounts() endpointCounts {
	return endpointCounts{
		count:   make(map[netip.AddrPort]int),
		pos:     make(map[netip.AddrPort]int),
		byCount: make([][]netip.AddrPort, 1),
	}
}

// max returns an endpoint with the highest count, and the count.
func (ec *endpointCounts) max() (netip.AddrPort, int) {
	top := len(ec.byCount) - 1
	if top == 0 {
		return netip.AddrPort{}, 0
	}
	return ec.byCount[top][0], top
}

func (ec *endpointCounts) inc(ep netip.AddrPort) {
	c := ec.count[ep]
	if c > 0 {
		ec.remove(ep, c)
	}
	c++
	if c == len(ec.byCount) {
		ec.byCount = append(ec.byCount, nil)
	}
	ec.add(ep, c)
}

func (ec *endpointCounts) dec(ep netip.AddrPort) {
	c := ec.count[ep]
	ec.remove(ep, c)
	if c--; c > 0 {
		ec.add(ep, c)
	} else {
		delete(ec.count, ep)
		delete(ec.pos, ep)
	}
	if top := len(ec.byCount) - 1; top > 0 && len(ec.byCount[top]) == 0 {
		ec.byCount[top] = nil
		ec.byCount = ec.byCount[:top]
	}
}

func (ec *endpointCounts) add(ep netip.AddrPort, c int) {
	ec.count[ep] = c
	ec.pos[ep] = len(ec.byCount[c])
	ec.byCount[c] = append(ec.byCount[c], ep)
}

func (ec *endpointCounts) remove(ep netip.AddrPort, c int) {
	set, i := ec.byCount[c], ec.pos[ep]
	last := set[len(set)-1]
	set[i] = last
	ec.pos[last] = i
	set = set[:len(set)-1]
	// Release the space of sets that have shrunk, so the sets don't keep their
	// largest size.
	if len(set) < cap(set)/4 {
		set = append([]netip.AddrPort(nil), set...)
	}
	ec.byCount[c] = set
}
