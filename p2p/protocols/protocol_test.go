package protocols

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

func init() {
	glog.SetV(logger.Detail)
	glog.SetToStderr(true)
}

// handshake message type
type hs0 struct {
	C uint
}

// message to kill/drop the peer with nodeId
type kill struct {
	C *adapters.NodeId
}

// message to drop connection
type drop struct {
}

/// protoHandshake represents module-independent aspects of the protocol and is
// the first message peers send and receive as part the initial exchange
type protoHandshake struct {
	Version   uint   // local and remote peer should have identical version
	NetworkId string // local and remote peer should have identical network id
}

// checkProtoHandshake verifies local and remote protoHandshakes match
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
func newProtocol(pp *TestPeerPool, wg *sync.WaitGroup) func(adapters.NodeAdapter) adapters.ProtoCall {
	ct := NewCodeMap("test", 42, 1024, &protoHandshake{}, &hs0{}, &kill{}, &drop{})
	return func(na adapters.NodeAdapter) adapters.ProtoCall {
		return func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			if wg != nil {
				wg.Add(1)
			}
			id := &adapters.NodeId{p.ID()}
			peer := NewPeer(p, ct, na.Messenger(rw), func() { na.Disconnect(id.Bytes()) })

			// demonstrates use of peerPool, killing another peer connection as a response to a message
			peer.Register(&kill{}, func(msg interface{}) error {
				id := msg.(*kill).C
				// pp.Get(id).Drop(fmt.Errorf("killed"))
				glog.V(logger.Detail).Infof("id %v killed", id)
				return nil
			})

			// for testing we can trigger self induced disconnect upon receiving drop message
			peer.Register(&drop{}, func(msg interface{}) error {
				glog.V(logger.Detail).Infof("dropped")
				// return fmt.Errorf("dropped")
				return
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

			lhs := &hs0{42}
			// module handshake demonstrating a simple repeatable exchange of same-type message
			hs, err = peer.Handshake(lhs)
			if err != nil {
				return err
			}

			if rmhs := hs.(*hs0); rmhs.C > lhs.C {
				return fmt.Errorf("handshake mismatch remote %v > local %v", rmhs.C, lhs.C)
			}

			peer.Register(lhs, func(msg interface{}) error {
				rhs := msg.(*hs0)
				if rhs.C > lhs.C {
					return fmt.Errorf("handshake mismatch remote %v > local %v", rhs.C, lhs.C)
				}
				lhs.C += rhs.C
				return peer.Send(lhs)
			})

			glog.V(logger.Detail).Infof("adding peer  %v", peer)
			pp.Add(peer)
			defer pp.Remove(peer)
			err = peer.Run()
			if wg != nil {
				wg.Done()
			}
			glog.V(logger.Detail).Infof("peer  %v protocol quitting: %v", peer, err)

			return err
		}
	}
}

func protocolTester(t *testing.T, pp *TestPeerPool, wg *sync.WaitGroup) *p2ptest.ExchangeSession {
	id := p2ptest.RandomNodeId()
	return p2ptest.NewProtocolTester(t, id, 2, newProtocol(pp, wg))
}

func protoHandshakeExchange(id *adapters.NodeId, proto *protoHandshake) []p2ptest.Exchange {

	return []p2ptest.Exchange{
		p2ptest.Exchange{
			Expects: []p2ptest.Expect{
				p2ptest.Expect{
					Code: 0,
					Msg:  &protoHandshake{42, "420"},
					Peer: id,
				},
			},
		},
		p2ptest.Exchange{
			Triggers: []p2ptest.Trigger{
				p2ptest.Trigger{
					Code: 0,
					Msg:  proto,
					Peer: id,
				},
			},
		},
	}
}

func runProtoHandshake(t *testing.T, proto *protoHandshake, errs ...error) {
	pp := NewTestPeerPool()
	s := protocolTester(t, pp, nil)
	// TODO: make this more than one handshake
	id := s.Ids[0]
	s.TestExchanges(protoHandshakeExchange(id, proto)...)
	var disconnects []*p2ptest.Disconnect
	for i, err := range errs {
		disconnects = append(disconnects, &p2ptest.Disconnect{Peer: s.Ids[i], Error: err})
	}
	s.TestDisconnected(disconnects...)
}

func TestProtoHandshakeVersionMismatch(t *testing.T) {
	runProtoHandshake(t, &protoHandshake{41, "420"}, fmt.Errorf("41 (!= 42)"))
}

func TestProtoHandshakeNetworkIdMismatch(t *testing.T) {
	runProtoHandshake(t, &protoHandshake{42, "421"}, fmt.Errorf("421 (!= 420)"))
}

func TestProtoHandshakeSuccess(t *testing.T) {
	runProtoHandshake(t, &protoHandshake{42, "420"})
}

func moduleHandshakeExchange(id *adapters.NodeId, resp uint) []p2ptest.Exchange {

	return []p2ptest.Exchange{
		p2ptest.Exchange{
			Expects: []p2ptest.Expect{
				p2ptest.Expect{
					Code: 1,
					Msg:  &hs0{42},
					Peer: id,
				},
			},
		},
		p2ptest.Exchange{
			Triggers: []p2ptest.Trigger{
				p2ptest.Trigger{
					Code: 1,
					Msg:  &hs0{resp},
					Peer: id,
				},
			},
		},
	}
}

func runModuleHandshake(t *testing.T, resp uint, errs ...error) {
	pp := NewTestPeerPool()
	s := protocolTester(t, pp, nil)
	id := s.Ids[0]
	s.TestExchanges(protoHandshakeExchange(id, &protoHandshake{42, "420"})...)
	s.TestExchanges(moduleHandshakeExchange(id, resp)...)
	var disconnects []*p2ptest.Disconnect
	for i, err := range errs {
		disconnects = append(disconnects, &p2ptest.Disconnect{Peer: s.Ids[i], Error: err})
	}
	s.TestDisconnected(disconnects...)
}

func TestModuleHandshakeError(t *testing.T) {
	runModuleHandshake(t, 43, fmt.Errorf("handshake mismatch remote 43 > local 42"))
}

func TestModuleHandshakeSuccess(t *testing.T) {
	runModuleHandshake(t, 42)
}

// testing complex interactions over multiple peers, relaying, dropping
func testMultiPeerSetup(a, b *adapters.NodeId) []p2ptest.Exchange {

	return []p2ptest.Exchange{
		p2ptest.Exchange{
			Label: "primary handshake",
			Expects: []p2ptest.Expect{
				p2ptest.Expect{
					Code: 0,
					Msg:  &protoHandshake{42, "420"},
					Peer: a,
				},
				p2ptest.Expect{
					Code: 0,
					Msg:  &protoHandshake{42, "420"},
					Peer: b,
				},
			},
		},
		p2ptest.Exchange{
			Label: "module handshake",
			Triggers: []p2ptest.Trigger{
				p2ptest.Trigger{
					Code: 0,
					Msg:  &protoHandshake{42, "420"},
					Peer: a,
				},
				p2ptest.Trigger{
					Code: 0,
					Msg:  &protoHandshake{42, "420"},
					Peer: b,
				},
			},
			Expects: []p2ptest.Expect{
				p2ptest.Expect{
					Code: 1,
					Msg:  &hs0{42},
					Peer: a,
				},
				p2ptest.Expect{
					Code: 1,
					Msg:  &hs0{42},
					Peer: b,
				},
			},
		},

		p2ptest.Exchange{Label: "alternative module handshake", Triggers: []p2ptest.Trigger{p2ptest.Trigger{Code: 1, Msg: &hs0{41}, Peer: a},
			p2ptest.Trigger{Code: 1, Msg: &hs0{41}, Peer: b}}},
		p2ptest.Exchange{Label: "repeated module handshake", Triggers: []p2ptest.Trigger{p2ptest.Trigger{Code: 1, Msg: &hs0{1}, Peer: a}}},
		p2ptest.Exchange{Label: "receiving repeated module handshake", Expects: []p2ptest.Expect{p2ptest.Expect{Code: 1, Msg: &hs0{43}, Peer: a}}}}
}

func runMultiplePeers(t *testing.T, peer int, errs ...error) {
	wg := &sync.WaitGroup{}
	pp := NewTestPeerPool()
	s := protocolTester(t, pp, wg)

	s.TestExchanges(testMultiPeerSetup(s.Ids[0], s.Ids[1])...)
	// after some exchanges of messages, we can test state changes
	// here this is simply demonstrated by the peerPool
	// after the handshake negotiations peers must be added to the pool
	// time.Sleep(1)
	for !pp.Has(s.Ids[0]) {
		time.Sleep(1)
		glog.V(logger.Detail).Infof("missing peer test-0: %v (%v)", pp, s.Ids)
	}
	// if !pp.Has(s.Ids[0]) {
	// 	t.Fatalf("missing peer test-0: %v (%v)", pp, s.Ids)
	// }
	if !pp.Has(s.Ids[1]) {
		t.Fatalf("missing peer test-1: %v (%v)", pp, s.Ids)
	}

	// sending kill request for peer with index <peer>
	s.TestExchanges(p2ptest.Exchange{
		Triggers: []p2ptest.Trigger{
			p2ptest.Trigger{
				Code: 2,
				Msg:  &kill{s.Ids[peer]},
				Peer: s.Ids[0],
			},
		},
	})

	// dropping the remaining peer
	s.TestExchanges(p2ptest.Exchange{
		Triggers: []p2ptest.Trigger{
			p2ptest.Trigger{
				Code: 3,
				Msg:  &drop{},
				Peer: s.Ids[(peer+1)%2],
			},
		},
	})
	wg.Wait()
	// check the actual discconnect errors on the individual peers
	var disconnects []*p2ptest.Disconnect
	for i, err := range errs {
		disconnects = append(disconnects, &p2ptest.Disconnect{Peer: s.Ids[i], Error: err})
	}
	s.TestDisconnected(disconnects...)
	// test if disconnected peers have been removed from peerPool
	if pp.Has(s.Ids[peer]) {
		t.Fatalf("peer test-%v not dropped: %v (%v)", peer, pp, s.Ids)
	}

}

func TestMultiplePeersDropSelf(t *testing.T) {
	runMultiplePeers(t, 0,
		fmt.Errorf("p2p: read or write on closed message pipe"),
		fmt.Errorf("Message handler error: (msg code 3): killed"),
	)
}

func TestMultiplePeersDropOther(t *testing.T) {
	runMultiplePeers(t, 1,
		fmt.Errorf("Message handler error: (msg code 3): dropped"),
		fmt.Errorf("p2p: read or write on closed message pipe"),
	)
}
