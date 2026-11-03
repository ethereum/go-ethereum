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
	"net"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

// This test checks the protocol sets of browser and node connections. Browser
// peers may only run protocols with AllowBrowser set, node-to-node connections
// must not run them.
func TestProtocolsFor(t *testing.T) {
	var browserRec enr.Record
	browserNode, err := signBrowser(&browserRec, enode.ID{0x01})
	if err != nil {
		t.Fatal(err)
	}
	key := newkey()
	nodeNode := enode.NewV4(&key.PublicKey, net.IP{127, 0, 0, 1}, 30303, 30303)

	srv := &Server{
		Config: Config{
			Protocols: []Protocol{
				{Name: "eth", Version: 68},
				{Name: "snap", Version: 1},
				{Name: "webtx", Version: 1, AllowBrowser: true},
			},
		},
	}
	tests := []struct {
		name string
		conn *conn
		want []string
	}{
		{"browser", &conn{node: browserNode}, []string{"webtx"}},
		{"node", &conn{node: nodeNode}, []string{"eth", "snap"}},
		{"unknown identity", &conn{}, []string{"eth", "snap"}},
	}
	for _, tt := range tests {
		var got []string
		for _, p := range srv.protocolsFor(tt.conn) {
			got = append(got, p.Name)
		}
		if !slices.Equal(got, tt.want) {
			t.Fatalf("%s: protocols = %v, want %v", tt.name, got, tt.want)
		}
	}
}
