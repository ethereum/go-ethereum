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

package enode

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/netip"
	"testing"
	"testing/quick"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
)

var pyRecord, _ = hex.DecodeString("f884b8407098ad865b00a582051940cb9cf36836572411a47278783077011599ed5cd16b76f2635f4e234738f30813a89eb9137e3e3df5266e3a1f11df72ecf1145ccb9c01826964827634826970847f00000189736563703235366b31a103ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd31388375647082765f")

// TestPythonInterop checks that we can decode and verify a record produced by the Python
// implementation.
func TestPythonInterop(t *testing.T) {
	var r enr.Record
	if err := rlp.DecodeBytes(pyRecord, &r); err != nil {
		t.Fatalf("can't decode: %v", err)
	}
	n, err := New(ValidSchemes, &r)
	if err != nil {
		t.Fatalf("can't verify record: %v", err)
	}

	var (
		wantID  = HexID("a448f24c6d18e575453db13171562b71999873db5b286df957af199ec94617f7")
		wantSeq = uint64(1)
		wantIP  = enr.IPv4{127, 0, 0, 1}
		wantUDP = enr.UDP(30303)
	)
	if n.Seq() != wantSeq {
		t.Errorf("wrong seq: got %d, want %d", n.Seq(), wantSeq)
	}
	if n.ID() != wantID {
		t.Errorf("wrong id: got %x, want %x", n.ID(), wantID)
	}
	want := map[enr.Entry]interface{}{new(enr.IPv4): &wantIP, new(enr.UDP): &wantUDP}
	for k, v := range want {
		desc := fmt.Sprintf("loading key %q", k.ENRKey())
		if assert.NoError(t, n.Load(k), desc) {
			assert.Equal(t, k, v, desc)
		}
	}
}

func TestNodeEndpoints(t *testing.T) {
	id := HexID("00000000000000806ad9b61fa5ae014307ebdc964253adcd9f2c0a392aa11abc")
	type endpointTest struct {
		name     string
		node     *Node
		wantIP   netip.Addr
		wantUDP  int
		wantTCP  int
		wantQUIC int
	}
	tests := []endpointTest{
		{
			name: "no-addr",
			node: func() *Node {
				var r enr.Record
				return SignNull(&r, id)
			}(),
		},
		{
			name: "udp-only",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.UDP(9000))
				return SignNull(&r, id)
			}(),
		},
		{
			name: "tcp-only",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.TCP(9000))
				return SignNull(&r, id)
			}(),
		},
		{
			name: "quic-only",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.QUIC(9000))
				return SignNull(&r, id)
			}(),
		},
		{
			name: "quic6-only",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.QUIC6(9000))
				return SignNull(&r, id)
			}(),
		},
		{
			name: "ipv4-only-loopback",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.IPv4Addr(netip.MustParseAddr("127.0.0.1")))
				return SignNull(&r, id)
			}(),
			wantIP: netip.MustParseAddr("127.0.0.1"),
		},
		{
			name: "ipv4-only-unspecified",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.IPv4Addr(netip.MustParseAddr("0.0.0.0")))
				return SignNull(&r, id)
			}(),
			wantIP: netip.MustParseAddr("0.0.0.0"),
		},
		{
			name: "ipv4-only",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.IPv4Addr(netip.MustParseAddr("99.22.33.1")))
				return SignNull(&r, id)
			}(),
			wantIP: netip.MustParseAddr("99.22.33.1"),
		},
		{
			name: "ipv6-only",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.IPv6Addr(netip.MustParseAddr("2001::ff00:0042:8329")))
				return SignNull(&r, id)
			}(),
			wantIP: netip.MustParseAddr("2001::ff00:0042:8329"),
		},
		{
			name: "ipv4-loopback-and-ipv6-global",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.IPv4Addr(netip.MustParseAddr("127.0.0.1")))
				r.Set(enr.UDP(30304))
				r.Set(enr.IPv6Addr(netip.MustParseAddr("2001::ff00:0042:8329")))
				r.Set(enr.UDP6(30306))
				return SignNull(&r, id)
			}(),
			wantIP:  netip.MustParseAddr("2001::ff00:0042:8329"),
			wantUDP: 30306,
		},
		{
			name: "ipv4-unspecified-and-ipv6-loopback",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.IPv4Addr(netip.MustParseAddr("0.0.0.0")))
				r.Set(enr.IPv6Addr(netip.MustParseAddr("::1")))
				return SignNull(&r, id)
			}(),
			wantIP: netip.MustParseAddr("::1"),
		},
		{
			name: "ipv4-private-and-ipv6-global",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.IPv4Addr(netip.MustParseAddr("192.168.2.2")))
				r.Set(enr.UDP(30304))
				r.Set(enr.IPv6Addr(netip.MustParseAddr("2001::ff00:0042:8329")))
				r.Set(enr.UDP6(30306))
				return SignNull(&r, id)
			}(),
			wantIP:  netip.MustParseAddr("2001::ff00:0042:8329"),
			wantUDP: 30306,
		},
		{
			name: "ipv4-local-and-ipv6-global",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.IPv4Addr(netip.MustParseAddr("169.254.2.6")))
				r.Set(enr.UDP(30304))
				r.Set(enr.IPv6Addr(netip.MustParseAddr("2001::ff00:0042:8329")))
				r.Set(enr.UDP6(30306))
				return SignNull(&r, id)
			}(),
			wantIP:  netip.MustParseAddr("2001::ff00:0042:8329"),
			wantUDP: 30306,
		},
		{
			name: "ipv4-private-and-ipv6-private",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.IPv4Addr(netip.MustParseAddr("192.168.2.2")))
				r.Set(enr.UDP(30304))
				r.Set(enr.IPv6Addr(netip.MustParseAddr("fd00::abcd:1")))
				r.Set(enr.UDP6(30306))
				return SignNull(&r, id)
			}(),
			wantIP:  netip.MustParseAddr("192.168.2.2"),
			wantUDP: 30304,
		},
		{
			name: "ipv4-private-and-ipv6-link-local",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.IPv4Addr(netip.MustParseAddr("192.168.2.2")))
				r.Set(enr.UDP(30304))
				r.Set(enr.IPv6Addr(netip.MustParseAddr("fe80::1")))
				r.Set(enr.UDP6(30306))
				return SignNull(&r, id)
			}(),
			wantIP:  netip.MustParseAddr("192.168.2.2"),
			wantUDP: 30304,
		},
		{
			name: "ipv4-quic",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.IPv4Addr(netip.MustParseAddr("99.22.33.1")))
				r.Set(enr.QUIC(9001))
				return SignNull(&r, id)
			}(),
			wantIP:   netip.MustParseAddr("99.22.33.1"),
			wantQUIC: 9001,
		},
		{ // Because the node is IPv4, the quic6 entry won't be loaded.
			name: "ipv4-quic6",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.IPv4Addr(netip.MustParseAddr("99.22.33.1")))
				r.Set(enr.QUIC6(9001))
				return SignNull(&r, id)
			}(),
			wantIP: netip.MustParseAddr("99.22.33.1"),
		},
		{
			name: "ipv6-quic",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.IPv6Addr(netip.MustParseAddr("2001::ff00:0042:8329")))
				r.Set(enr.QUIC(9001))
				return SignNull(&r, id)
			}(),
			wantIP: netip.MustParseAddr("2001::ff00:0042:8329"),
		},
		{
			name: "ipv6-quic6",
			node: func() *Node {
				var r enr.Record
				r.Set(enr.IPv6Addr(netip.MustParseAddr("2001::ff00:0042:8329")))
				r.Set(enr.QUIC6(9001))
				return SignNull(&r, id)
			}(),
			wantIP:   netip.MustParseAddr("2001::ff00:0042:8329"),
			wantQUIC: 9001,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.wantIP != test.node.IPAddr() {
				t.Errorf("node has wrong IP %v, want %v", test.node.IPAddr(), test.wantIP)
			}
			if test.wantUDP != test.node.UDP() {
				t.Errorf("node has wrong UDP port %d, want %d", test.node.UDP(), test.wantUDP)
			}
			if test.wantTCP != test.node.TCP() {
				t.Errorf("node has wrong TCP port %d, want %d", test.node.TCP(), test.wantTCP)
			}
			if quic, _ := test.node.QUICEndpoint(); test.wantQUIC != int(quic.Port()) {
				t.Errorf("node has wrong QUIC port %d, want %d", quic.Port(), test.wantQUIC)
			}
		})
	}
}

func TestHexID(t *testing.T) {
	ref := ID{0, 0, 0, 0, 0, 0, 0, 128, 106, 217, 182, 31, 165, 174, 1, 67, 7, 235, 220, 150, 66, 83, 173, 205, 159, 44, 10, 57, 42, 161, 26, 188}
	id1 := HexID("0x00000000000000806ad9b61fa5ae014307ebdc964253adcd9f2c0a392aa11abc")
	id2 := HexID("00000000000000806ad9b61fa5ae014307ebdc964253adcd9f2c0a392aa11abc")

	if id1 != ref {
		t.Errorf("wrong id1\ngot  %v\nwant %v", id1[:], ref[:])
	}
	if id2 != ref {
		t.Errorf("wrong id2\ngot  %v\nwant %v", id2[:], ref[:])
	}
}

func TestID_textEncoding(t *testing.T) {
	ref := ID{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x10,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x20,
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x30,
		0x31, 0x32,
	}
	hex := "0102030405060708091011121314151617181920212223242526272829303132"

	text, err := ref.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(text, []byte(hex)) {
		t.Fatalf("text encoding did not match\nexpected: %s\ngot:      %s", hex, text)
	}

	id := new(ID)
	if err := id.UnmarshalText(text); err != nil {
		t.Fatal(err)
	}
	if *id != ref {
		t.Fatalf("text decoding did not match\nexpected: %s\ngot:      %s", ref, id)
	}
}

func TestID_distcmp(t *testing.T) {
	distcmpBig := func(target, a, b ID) int {
		tbig := new(big.Int).SetBytes(target[:])
		abig := new(big.Int).SetBytes(a[:])
		bbig := new(big.Int).SetBytes(b[:])
		return new(big.Int).Xor(tbig, abig).Cmp(new(big.Int).Xor(tbig, bbig))
	}
	if err := quick.CheckEqual(DistCmp, distcmpBig, nil); err != nil {
		t.Error(err)
	}
}

// The random tests is likely to miss the case where a and b are equal,
// this test checks it explicitly.
func TestID_distcmpEqual(t *testing.T) {
	base := ID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	x := ID{15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	if DistCmp(base, x, x) != 0 {
		t.Errorf("DistCmp(base, x, x) != 0")
	}
}

func TestID_logdist(t *testing.T) {
	logdistBig := func(a, b ID) int {
		abig, bbig := new(big.Int).SetBytes(a[:]), new(big.Int).SetBytes(b[:])
		return new(big.Int).Xor(abig, bbig).BitLen()
	}
	if err := quick.CheckEqual(LogDist, logdistBig, nil); err != nil {
		t.Error(err)
	}
}

// The random tests is likely to miss the case where a and b are equal,
// this test checks it explicitly.
func TestID_logdistEqual(t *testing.T) {
	x := ID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	if LogDist(x, x) != 0 {
		t.Errorf("LogDist(x, x) != 0")
	}
}
