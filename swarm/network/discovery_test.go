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
	pp := p2ptest.NewTestPeerPool()
	//pp := NewHive(NewHiveParams(), to)
	ct := BzzCodeMap(HiveMsgs...)
	
	services := func(p Peer) error {
		dp := NewDiscovery(p, to)
		//pp.Add(p)
		to.On(dp)
		p.DisconnectHook(func(e interface{}) error {
			dp := e.(Peer)
			to.Off(dp)
			return nil
		})
		return nil
	}
	/*
	protocall := func (na adapters.NodeAdapter) adapters.ProtoCall {
		protocol := Bzz(addr.OverlayAddr(), na, ct, services, nil, nil)	
		return protocol.Run
	}
	
	s := p2ptest.NewProtocolTester(t, NodeId(addr), 1, protocall)
*/

	s := newBzzTester(t, addr, pp, ct, services)

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
