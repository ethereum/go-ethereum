package simulations

import (
	"testing"

	"github.com/maticnetwork/bor/p2p"
	"github.com/maticnetwork/bor/p2p/enode"
	"github.com/maticnetwork/bor/p2p/enr"
	"github.com/maticnetwork/bor/rpc"
)

// NoopService is the service that does not do anything
// but implements node.Service interface.
type NoopService struct {
	c map[enode.ID]chan struct{}
}

func NewNoopService(ackC map[enode.ID]chan struct{}) *NoopService {
	return &NoopService{
		c: ackC,
	}
}

func (t *NoopService) Protocols() []p2p.Protocol {
	return []p2p.Protocol{
		{
			Name:    "noop",
			Version: 666,
			Length:  0,
			Run: func(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
				if t.c != nil {
					t.c[peer.ID()] = make(chan struct{})
					close(t.c[peer.ID()])
				}
				rw.ReadMsg()
				return nil
			},
			NodeInfo: func() interface{} {
				return struct{}{}
			},
			PeerInfo: func(id enode.ID) interface{} {
				return struct{}{}
			},
			Attributes: []enr.Entry{},
		},
	}
}

func (t *NoopService) APIs() []rpc.API {
	return []rpc.API{}
}

func (t *NoopService) Start(server *p2p.Server) error {
	return nil
}

func (t *NoopService) Stop() error {
	return nil
}

func VerifyRing(t *testing.T, net *Network, ids []enode.ID) {
	t.Helper()
	n := len(ids)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			c := net.GetConn(ids[i], ids[j])
			if i == j-1 || (i == 0 && j == n-1) {
				if c == nil {
					t.Errorf("nodes %v and %v are not connected, but they should be", i, j)
				}
			} else {
				if c != nil {
					t.Errorf("nodes %v and %v are connected, but they should not be", i, j)
				}
			}
		}
	}
}

func VerifyChain(t *testing.T, net *Network, ids []enode.ID) {
	t.Helper()
	n := len(ids)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			c := net.GetConn(ids[i], ids[j])
			if i == j-1 {
				if c == nil {
					t.Errorf("nodes %v and %v are not connected, but they should be", i, j)
				}
			} else {
				if c != nil {
					t.Errorf("nodes %v and %v are connected, but they should not be", i, j)
				}
			}
		}
	}
}

func VerifyFull(t *testing.T, net *Network, ids []enode.ID) {
	t.Helper()
	n := len(ids)
	var connections int
	for i, lid := range ids {
		for _, rid := range ids[i+1:] {
			if net.GetConn(lid, rid) != nil {
				connections++
			}
		}
	}

	want := n * (n - 1) / 2
	if connections != want {
		t.Errorf("wrong number of connections, got: %v, want: %v", connections, want)
	}
}

func VerifyStar(t *testing.T, net *Network, ids []enode.ID, centerIndex int) {
	t.Helper()
	n := len(ids)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			c := net.GetConn(ids[i], ids[j])
			if i == centerIndex || j == centerIndex {
				if c == nil {
					t.Errorf("nodes %v and %v are not connected, but they should be", i, j)
				}
			} else {
				if c != nil {
					t.Errorf("nodes %v and %v are connected, but they should not be", i, j)
				}
			}
		}
	}
}
