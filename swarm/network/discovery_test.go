package network

import (
	"testing"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
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
		glog.V(logger.Detail).Infof("kademlia on %v", p)
		p.DisconnectHook(func(err error) {
			to.Off(p)
		})
		return nil
	}

	s := newBzzBaseTester(t, 1, addr, ct, services)

	s.runHandshakes()
	// o := 0
	s.TestExchanges(p2ptest.Exchange{
		Label: "outgoing SubPeersMsg",
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 3,
				Msg:  &SubPeersMsg{ProxLimit: 0, MinProxBinSize: 8},
				Peer: s.ProtocolTester.Ids[0],
			},
		},
	})
}
