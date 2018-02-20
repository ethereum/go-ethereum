// Copyright 2016 The go-ethereum Authors
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

package network

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

/***
 *
 * - after connect, that outgoing subpeersmsg is sent
 *
 */
func TestDiscovery(t *testing.T) {
	addr := RandomAddr()
	to := NewKademlia(addr.OAddr, NewKadParams())

	run := func(p *BzzPeer) error {
		dp := newDiscovery(p, to)
		to.On(p)
		defer to.Off(p)
		log.Trace(fmt.Sprintf("kademlia on %v", p))
		return p.Run(dp.HandleMsg)
	}

	s := newBzzBaseTester(t, 1, addr, DiscoverySpec, run)
	defer s.Stop()

	s.TestExchanges(p2ptest.Exchange{
		Label: "outgoing SubPeersMsg",
		Expects: []p2ptest.Expect{
			{
				Code: 3,
				Msg:  &subPeersMsg{Depth: 0},
				Peer: s.ProtocolTester.IDs[0],
			},
		},
	})
}
