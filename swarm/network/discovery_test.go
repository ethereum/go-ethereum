package network

import (
	"testing"

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
	ct := BzzCodeMap(HiveMsgs...)

	services := func(p Peer) error {
		dp := NewDiscovery(p, to)
		to.On(dp)
		p.DisconnectHook(func(e interface{}) error {
			dp := e.(Peer)
			to.Off(dp)
			return nil
		})
		return nil
	}

	s := newBzzBaseTester(t, 1, addr, ct, services)

	s.runHandshakes()
	s.TestExchanges(p2ptest.Exchange{
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 3,
				Msg:  &SubPeersMsg{ProxLimit: 0, MinProxBinSize: 8},
				Peer: s.ExchangeSession.Ids[0],
			},
		},
	})
}
