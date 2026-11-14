// Copyright 2015 The go-ethereum Authors
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

package nat

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// This test checks that autodisc doesn't hang and returns
// consistent results when multiple goroutines call its methods
// concurrently.
func TestAutoDiscRace(t *testing.T) {
	ad := startautodisc("thing", func() Interface {
		time.Sleep(500 * time.Millisecond)
		return ExtIP{33, 44, 55, 66}
	})

	// Spawn a few concurrent calls to ad.ExternalIP.
	type rval struct {
		ip  net.IP
		err error
	}
	results := make(chan rval, 50)
	for i := 0; i < cap(results); i++ {
		go func() {
			ip, err := ad.ExternalIP()
			results <- rval{ip, err}
		}()
	}

	// Check that they all return the correct result within the deadline.
	deadline := time.After(2 * time.Second)
	for i := 0; i < cap(results); i++ {
		select {
		case <-deadline:
			t.Fatal("deadline exceeded")
		case rval := <-results:
			if rval.err != nil {
				t.Errorf("result %d: unexpected error: %v", i, rval.err)
			}
			wantIP := net.IP{33, 44, 55, 66}
			if !rval.ip.Equal(wantIP) {
				t.Errorf("result %d: got IP %v, want %v", i, rval.ip, wantIP)
			}
		}
	}
}

// stun should work well
func TestParseStun(t *testing.T) {
	testcases := []struct {
		natStr string
		want   *stun
	}{
		{"stun", &stun{serverList: strings.Split(stunDefaultServers, "\n")}},
		{"stun:1.2.3.4:1234", &stun{serverList: []string{"1.2.3.4:1234"}}},
		{"stun:[2001:db8::1]:3478", &stun{serverList: []string{"[2001:db8::1]:3478"}}},
	}

	for _, tc := range testcases {
		nat, err := Parse(tc.natStr)
		if err != nil {
			t.Errorf("should no err, but get %v", err)
		}
		stun := nat.(*stun)
		assert.Equal(t, stun.serverList, tc.want.serverList)
	}
}

func TestParseExtIP(t *testing.T) {
	tests := []struct {
		spec string
		want Interface
	}{
		{"extip:77.12.33.4", ExtIP(net.ParseIP("77.12.33.4"))},
		{"extip:2001:db8::1", ExtIP(net.ParseIP("2001:db8::1"))},
		{"extip:77.12.33.4,2001:db8::1", ExtIPs{IPv4: net.ParseIP("77.12.33.4").To4(), IPv6: net.ParseIP("2001:db8::1")}},
		{"extip:2001:db8::1,77.12.33.4", ExtIPs{IPv4: net.ParseIP("77.12.33.4").To4(), IPv6: net.ParseIP("2001:db8::1")}},
		{"extip:0.0.0.0,2001:db8::1", ExtIPs{IPv4: net.IPv4zero.To4(), IPv6: net.ParseIP("2001:db8::1")}},
		{"extip:77.12.33.4,::", ExtIPs{IPv4: net.ParseIP("77.12.33.4").To4(), IPv6: net.IPv6unspecified}},
	}
	for _, tc := range tests {
		got, err := Parse(tc.spec)
		if err != nil {
			t.Errorf("%q: unexpected error %v", tc.spec, err)
			continue
		}
		assert.Equal(t, tc.want, got, tc.spec)
	}

	bad := []string{
		"extip:",
		"extip:77.12.33.4,",
		"extip:77.12.33.4,10.0.0.1",
		"extip:2001:db8::1,2001:db8::2",
		"extip:0.0.0.0,::",
		"extip:77.12.33.4,2001:db8::1,::",
	}
	for _, spec := range bad {
		if _, err := Parse(spec); err == nil {
			t.Errorf("%q: expected error", spec)
		}
	}

	n, _ := Parse("extip:0.0.0.0,2001:db8::1")
	text, _ := n.(ExtIPs).MarshalText()
	assert.Equal(t, "extip:0.0.0.0,2001:db8::1", string(text))
	ip, _ := n.ExternalIP()
	assert.Equal(t, net.ParseIP("2001:db8::1"), ip)
}

func TestParseRejectsAddressPairForOtherMechanisms(t *testing.T) {
	for _, mech := range []string{"none", "off", "any", "auto", "on", "upnp", "pmp", "natpmp", "nat-pmp"} {
		spec := mech + ":192.0.2.1,2001:db8::1"
		if _, err := Parse(spec); err == nil {
			t.Errorf("%q: expected invalid IP address", spec)
		}
	}
}
