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

package ethapi

import (
	"testing"

	"github.com/ethereum/go-ethereum/p2p"
)

func TestNetAPIListening(t *testing.T) {
	tests := []struct {
		name string
		net  *p2p.Server
		want bool
	}{
		// A node with no P2P TCP listener (e.g. a --dev node) must report
		// that it is not listening for network connections.
		{"no p2p listener", &p2p.Server{}, false},
		// Without a P2P server there is no listener to report.
		{"no p2p server", nil, false},
		// A node with an enabled P2P listener keeps reporting true.
		{"p2p listener enabled", &p2p.Server{Config: p2p.Config{ListenAddr: ":30303"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := NewNetAPI(tt.net, 1)
			if got := api.Listening(); got != tt.want {
				t.Errorf("Listening() = %v, want %v", got, tt.want)
			}
		})
	}
}
