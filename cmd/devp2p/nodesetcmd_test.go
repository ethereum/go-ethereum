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

package main

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

func TestDialableFilter(t *testing.T) {
	filter, err := dialableFilter(nil)
	if err != nil {
		t.Fatal(err)
	}

	makeNode := func(entries ...enr.Entry) nodeJSON {
		r := new(enr.Record)
		for _, e := range entries {
			r.Set(e)
		}
		key, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		if err := enode.SignV4(r, key); err != nil {
			t.Fatal(err)
		}
		n, err := enode.New(enode.ValidSchemes, r)
		if err != nil {
			t.Fatal(err)
		}
		return nodeJSON{N: n}
	}

	tests := []struct {
		name string
		node nodeJSON
		want bool
	}{
		{"tcp", makeNode(enr.TCP(30303)), true},
		{"tcp6", makeNode(enr.TCP6(30303)), true},
		{"quic", makeNode(enr.QUIC(30303)), true},
		{"no ports", makeNode(), false},
		{"udp only", makeNode(enr.UDP(30303)), false},
		{"zero tcp", makeNode(enr.TCP(0)), false},
		{"oversized tcp", makeNode(enr.WithEntry("tcp", uint32(70000))), false},
	}
	for _, tt := range tests {
		if got := filter(tt.node); got != tt.want {
			t.Errorf("%s: dialable = %v, want %v", tt.name, got, tt.want)
		}
	}
}
