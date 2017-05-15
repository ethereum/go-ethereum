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

	run := func(p *bzzPeer) error {
		dp := NewDiscovery(p, to)
		to.On(p)
		defer to.Off(p)
		log.Trace(fmt.Sprintf("kademlia on %v", p))
		return p.Run(dp.HandleMsg)
	}

	s := newBzzBaseTester(t, 1, addr, DiscoveryProtocol, run)
	defer s.Stop()

	s.TestExchanges(p2ptest.Exchange{
		Label: "outgoing SubPeersMsg",
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 3,
				Msg:  &subPeersMsg{Depth: 0},
				Peer: s.ProtocolTester.Ids[0],
			},
		},
	})
}
