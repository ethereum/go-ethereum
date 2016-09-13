package protocols

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

// handshake message type
type hs0 struct {
	C uint
}

// message to kill/drop the peer with index C
type kill struct {
	C discover.NodeID
}

// message to drop connection
type drop struct {
}

// example peerPool to demonstrate registration of peer connections
type peerPool struct {
	lock  sync.Mutex
	peers map[discover.NodeID]*Peer
}

func newPeerPool() *peerPool {
	return &peerPool{peers: make(map[discover.NodeID]*Peer)}
}

func (self *peerPool) add(p *Peer) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.peers[p.ID()] = p
}

func (self *peerPool) remove(p *Peer) {
	self.lock.Lock()
	defer self.lock.Unlock()
	delete(self.peers, p.ID())
}

func (self *peerPool) has(n discover.NodeID) bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	_, ok := self.peers[n]
	return ok
}

func (self *peerPool) get(n discover.NodeID) *Peer {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.peers[n]
}

/// protoHandshake represents module-independent aspects of the protocol and is
// the first message peers send and receive as part the initial exchange
type protoHandshake struct {
	Version   uint   // local and remote peer should have identical version
	NetworkId string // local and remote  peer should have identical network id
}

// function to check local and remote  protoHandshake matches
func checkProtoHandshake(local, remote *protoHandshake) error {

	if remote.NetworkId != local.NetworkId {
		return fmt.Errorf("%s (!= %s)", remote.NetworkId, local.NetworkId)
	}

	if remote.Version != local.Version {
		return fmt.Errorf("%d (!= %d)", remote.Version, local.Version)
	}
	return nil
}

const networkId = "420"

// newProtocol sets up a protocol
// the run function here demonstrates a typical protocol using peerPool, handshake
// and messages registered to handlers
func newProtocol() (*p2p.Protocol, *peerPool) {
	ct := NewCodeMap("test", 42, 1024, &protoHandshake{}, &hs0{}, &kill{}, &drop{})
	pp := newPeerPool()

	run := func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
		peer := NewTestPeer(p, rw, ct)

		// demonstrates use of peerPool, killing another peer connection as a response to a message
		peer.Register(&kill{}, func(msg interface{}) error {
			// panics if target.C out of range
			id := msg.(*kill).C
			// name := fmt.Sprintf("test-%d", i)
			pp.get(id).Drop()
			return nil
		})

		// for testing we can trigger self induced disconnect upon receiving drop message
		peer.Register(&drop{}, func(msg interface{}) error {
			return fmt.Errorf("received disconnect request")
		})

		// initiate one-off protohandshake and check validity
		phs := &protoHandshake{ct.Version, networkId}
		hs, err := peer.Handshake(phs)
		if err != nil {
			return err
		}
		rhs := hs.(*protoHandshake)
		err = checkProtoHandshake(phs, rhs)
		if err != nil {
			return err
		}

		lhs := &hs0{1}
		// module handshake demonstrating a simple repeatable exchange of same-type message
		hs, err = peer.Handshake(lhs)
		if err != nil {
			return err
		}
		rmhs := hs.(*hs0)
		if rmhs.C != lhs.C {
			return fmt.Errorf("handshake mismatch remote %v != local %v", rmhs.C, lhs.C)
		}

		peer.Register(lhs, func(msg interface{}) error {
			rhs := msg.(*hs0)
			if rhs.C != lhs.C {
				return fmt.Errorf("handshake mismatch remote %v != local %v", rhs.C, lhs.C)
			}
			return peer.Send(rhs)
		})

		// add/remove peer from pool
		pp.add(peer)
		defer pp.remove(peer)
		// this launches a forever read loop
		return peer.Run()
	}

	return &p2p.Protocol{
		Name:    ct.Name,
		Length:  uint64(len(ct.codes)),
		Version: 42,
		Run:     run,
	}, pp
}

func protoHandshakeExchange(proto *protoHandshake) []p2ptest.Exchange {

	return []p2ptest.Exchange{
		p2ptest.Exchange{
			Expects: []p2ptest.Expect{
				p2ptest.Expect{
					Code: 0,
					Msg:  &protoHandshake{42, "420"},
					Peer: 0,
				},
			},
		},
		p2ptest.Exchange{
			Triggers: []p2ptest.Trigger{
				p2ptest.Trigger{
					Code: 0,
					Msg:  proto,
					Peer: 0,
				},
			},
		},
	}
}

func runProtoHandshake(t *testing.T, proto *protoHandshake, err error) {
	p, _ := newProtocol()
	ids := p2ptest.RandomNodeIDs(t, 1)
	s := p2ptest.NewSession(t, p, ids, nil)

	s.TestExchanges(protoHandshakeExchange(proto)...)
	s.TestDisconnects(err)
}

func TestProtoHandshakeVersionMismatch(t *testing.T) {
	runProtoHandshake(t, &protoHandshake{41, "420"}, fmt.Errorf("41 (!= 42)"))
}

func TestProtoHandshakeNetworkIdMismatch(t *testing.T) {
	runProtoHandshake(t, &protoHandshake{42, "421"}, fmt.Errorf("421 (!= 420)"))
}

func TestProtoHandshakeSuccess(t *testing.T) {
	runProtoHandshake(t, &protoHandshake{42, "420"}, nil)
}

func moduleHandshakeExchange(resp uint) []p2ptest.Exchange {

	return []p2ptest.Exchange{
		p2ptest.Exchange{
			Expects: []p2ptest.Expect{
				p2ptest.Expect{
					Code: 1,
					Msg:  &hs0{1},
					Peer: 0,
				},
			},
		},
		p2ptest.Exchange{
			Triggers: []p2ptest.Trigger{
				p2ptest.Trigger{
					Code: 1,
					Msg:  &hs0{resp},
					Peer: 0,
				},
			},
		},
	}
}

func runModuleHandshake(t *testing.T, resp uint, err error) {
	p, _ := newProtocol()
	ids := p2ptest.RandomNodeIDs(t, 1)
	s := p2ptest.NewSession(t, p, ids, nil)

	s.TestExchanges(protoHandshakeExchange(&protoHandshake{42, "420"})...)
	s.TestExchanges(moduleHandshakeExchange(resp)...)
	s.TestDisconnects(err)
}

func TestModuleHandshakeError(t *testing.T) {
	runModuleHandshake(t, 42, fmt.Errorf("handshake mismatch remote 42 != local 1"))
}

func TestModuleHandshakeSuccess(t *testing.T) {
	runModuleHandshake(t, 1, nil)
}

// testing complex interactions over multiple peers, relaying, dropping
func testMultiPeerSetup() []p2ptest.Exchange {

	return []p2ptest.Exchange{
		p2ptest.Exchange{
			Expects: []p2ptest.Expect{
				p2ptest.Expect{
					Code: 0,
					Msg:  &protoHandshake{42, "420"},
					Peer: 0,
				},
				p2ptest.Expect{
					Code: 0,
					Msg:  &protoHandshake{42, "420"},
					Peer: 1,
				},
			},
		},
		p2ptest.Exchange{
			Triggers: []p2ptest.Trigger{
				p2ptest.Trigger{
					Code: 0,
					Msg:  &protoHandshake{42, "420"},
					Peer: 0,
				},
				p2ptest.Trigger{
					Code: 0,
					Msg:  &protoHandshake{42, "420"},
					Peer: 1,
				},
			},
			Expects: []p2ptest.Expect{
				p2ptest.Expect{
					Code: 1,
					Msg:  &hs0{1},
					Peer: 0,
				},
				p2ptest.Expect{
					Code: 1,
					Msg:  &hs0{1},
					Peer: 1,
				},
			},
		},
		p2ptest.Exchange{
			Triggers: []p2ptest.Trigger{
				p2ptest.Trigger{
					Code: 1,
					Msg:  &hs0{1},
					Peer: 0,
				},
				p2ptest.Trigger{
					Code: 1,
					Msg:  &hs0{1},
					Peer: 1,
				},
			},
		},
	}
}

func runMultiplePeers(t *testing.T, ids []discover.NodeID, peer int, errs ...error) {
	p, pp := newProtocol()
	wg := &sync.WaitGroup{}
	s := p2ptest.NewSession(t, p, ids, wg)

	s.TestExchanges(testMultiPeerSetup()...)
	// after some exchanges of messages, we can test state changes
	// here this is simply demonstrated by the peerPool
	// after the handshake negotiations peers must be addded to the pool
	if !pp.has(ids[0]) {
		t.Fatalf("missing peer test-0: %v", pp)
	}
	if !pp.has(ids[1]) {
		t.Fatalf("missing peer test-1: %v", pp)
	}

	// sending kill request for peer with index <peer>
	s.TestExchanges(p2ptest.Exchange{
		Triggers: []p2ptest.Trigger{
			p2ptest.Trigger{
				Code: 2,
				Msg:  &kill{ids[peer]},
				Peer: 0,
			},
		},
	})

	// dropping the remaining peer
	s.TestExchanges(p2ptest.Exchange{
		Triggers: []p2ptest.Trigger{
			p2ptest.Trigger{
				Code: 3,
				Msg:  &drop{},
				Peer: (peer + 1) % 2,
			},
		},
	})

	// since drops are asyncronous, for correct testing you need to wait
	//for all disconnections and error registration to complete or time out
	errc := make(chan bool)
	go func() {
		wg.Wait()
		close(errc)
	}()

	select {
	case <-errc:
	case <-time.NewTimer(1000 * time.Millisecond).C:
		t.Fatalf("timed out")
	}

	// test if disconnected peers have been removed from peerPool
	if pp.has(ids[peer]) {
		t.Fatalf("peer test-% not dropped: %v", peer, pp)
	}
	// check the actual discconnect errors on the individual peers
	s.TestDisconnects(errs...)

}

func TestMultiplePeersDropSelf(t *testing.T) {
	ids := p2ptest.RandomNodeIDs(t, 2)
	runMultiplePeers(t, ids, 0, fmt.Errorf("p2p: read or write on closed message pipe"), fmt.Errorf("Message handler error: (msg code 3): received disconnect request"))
}

func TestMultiplePeersDropOther(t *testing.T) {
	ids := p2ptest.RandomNodeIDs(t, 2)
	runMultiplePeers(t, ids, 1, fmt.Errorf("Message handler error: (msg code 3): received disconnect request"), fmt.Errorf("p2p: read or write on closed message pipe"))
}
