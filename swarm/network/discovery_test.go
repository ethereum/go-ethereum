package network

import (
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/adapters"
	// "github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

func TestDiscovery(t *testing.T) {
	addr := RandomAddr()
	to := NewTestOverlay(addr.OverlayAddr())
	pp := NewHive(NewHiveParams(), to)
	ct := BzzCodeMap(HiveMsgs...)
	s := newDiscoveryTester(t, 0, addr, pp, ct, nil)
	s.TestExchanges(p2ptest.Exchange{
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 1,
				Msg:  &subPeersMsg{uint(o)},
				Peer: id,
			},
		},
	})
}
