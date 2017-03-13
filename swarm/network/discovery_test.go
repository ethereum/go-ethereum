package network

import (
	"testing"

	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

func TestDiscovery(t *testing.T) {
	addr := RandomAddr()
	to := NewKademlia(addr.OAddr, NewKadParams())
	pp := NewHive(NewHiveParams(), to)
	ct := BzzCodeMap(HiveMsgs...)
	s := newBzzTester(t, 1, addr, pp, ct, nil)

	s.runHandshakes()
	s.TestExchanges(p2ptest.Exchange{
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 3,
				Msg:  &SubPeersMsg{ProxLimit: 0, MinProxBinSize: 8},
				Peer: s.ExchangeSession.Id(1),
			},
		},
	})
}
