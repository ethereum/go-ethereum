package protocols

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

func init() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
}

// handshake message type
type hs0 struct {
	C uint
}

// message to kill/drop the peer with nodeID
type kill struct {
	C discover.NodeID
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
func newProtocol(pp *p2ptest.TestPeerPool) func(*p2p.Peer, p2p.MsgReadWriter) error {
	spec := &Spec{
		Name:       "test",
		Version:    42,
		MaxMsgSize: 10 * 1024,
		Messages: []interface{}{
			protoHandshake{},
			hs0{},
			kill{},
			drop{},
		},
	}
	return func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
		peer := NewPeer(p, rw, spec)

		// initiate one-off protohandshake and check validity
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		phs := &protoHandshake{42, networkId}
		hs, err := peer.Handshake(ctx, phs)
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
		hs, err = peer.Handshake(ctx, lhs)
		if err != nil {
			return err
		}

		if rmhs := hs.(*hs0); rmhs.C > lhs.C {
			return fmt.Errorf("handshake mismatch remote %v > local %v", rmhs.C, lhs.C)
		}

		handle := func(msg interface{}) error {
			switch msg := msg.(type) {

			case *protoHandshake:
				return errors.New("duplicate handshake")

			case *hs0:
				rhs := msg
				if rhs.C > lhs.C {
					return fmt.Errorf("handshake mismatch remote %v > local %v", rhs.C, lhs.C)
				}
				lhs.C += rhs.C
				return peer.Send(lhs)

			case *kill:
				// demonstrates use of peerPool, killing another peer connection as a response to a message
				id := msg.C
				pp.Get(id).Drop(errors.New("killed"))
				log.Trace(fmt.Sprintf("id %v killed", id))
				return nil

			case *drop:
				// for testing we can trigger self induced disconnect upon receiving drop message
				return errors.New("dropped")

			default:
				return fmt.Errorf("unknown message type: %T", msg)
			}
		}

		log.Trace(fmt.Sprintf("adding peer  %v", peer))
		pp.Add(peer)
		defer pp.Remove(peer)
		err = peer.Run(handle)
		log.Trace(fmt.Sprintf("peer  %v protocol quitting: %v", peer, err))

		return err
	}
}

func protocolTester(t *testing.T, pp *p2ptest.TestPeerPool) *p2ptest.ProtocolTester {
	conf := adapters.RandomNodeConfig()
	return p2ptest.NewProtocolTester(t, conf.ID, 2, newProtocol(pp))
}

func protoHandshakeExchange(id discover.NodeID, proto *protoHandshake) []p2ptest.Exchange {

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
	pp := p2ptest.NewTestPeerPool()
	s := protocolTester(t, pp)
	// TODO: make this more than one handshake
	id := s.IDs[0]
	if err := s.TestExchanges(protoHandshakeExchange(id, proto)...); err != nil {
		t.Fatal(err)
	}
	var disconnects []*p2ptest.Disconnect
	for i, err := range errs {
		disconnects = append(disconnects, &p2ptest.Disconnect{Peer: s.IDs[i], Error: err})
	}
	if err := s.TestDisconnected(disconnects...); err != nil {
		t.Fatal(err)
	}
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

func moduleHandshakeExchange(id discover.NodeID, resp uint) []p2ptest.Exchange {

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
	pp := p2ptest.NewTestPeerPool()
	s := protocolTester(t, pp)
	id := s.IDs[0]
	s.TestExchanges(protoHandshakeExchange(id, &protoHandshake{42, "420"})...)
	s.TestExchanges(moduleHandshakeExchange(id, resp)...)
	var disconnects []*p2ptest.Disconnect
	for i, err := range errs {
		disconnects = append(disconnects, &p2ptest.Disconnect{Peer: s.IDs[i], Error: err})
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
func testMultiPeerSetup(a, b discover.NodeID) []p2ptest.Exchange {

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
	pp := p2ptest.NewTestPeerPool()
	s := protocolTester(t, pp)

	s.TestExchanges(testMultiPeerSetup(s.IDs[0], s.IDs[1])...)
	// after some exchanges of messages, we can test state changes
	// here this is simply demonstrated by the peerPool
	// after the handshake negotiations peers must be added to the pool
	// time.Sleep(1)
	for !pp.Has(s.IDs[0]) {
		time.Sleep(1)
		log.Trace(fmt.Sprintf("missing peer test-0: %v (%v)", pp, s.IDs))
	}
	// if !pp.Has(s.IDs[0]) {
	// 	t.Fatalf("missing peer test-0: %v (%v)", pp, s.IDs)
	// }
	if !pp.Has(s.IDs[1]) {
		t.Fatalf("missing peer test-1: %v (%v)", pp, s.IDs)
	}

	// sending kill request for peer with index <peer>
	s.TestExchanges(p2ptest.Exchange{
		Triggers: []p2ptest.Trigger{
			p2ptest.Trigger{
				Code: 2,
				Msg:  &kill{s.IDs[peer]},
				Peer: s.IDs[0],
			},
		},
	})

	// dropping the remaining peer
	s.TestExchanges(p2ptest.Exchange{
		Triggers: []p2ptest.Trigger{
			p2ptest.Trigger{
				Code: 3,
				Msg:  &drop{},
				Peer: s.IDs[(peer+1)%2],
			},
		},
	})
	// check the actual discconnect errors on the individual peers
	var disconnects []*p2ptest.Disconnect
	for i, err := range errs {
		disconnects = append(disconnects, &p2ptest.Disconnect{Peer: s.IDs[i], Error: err})
	}
	s.TestDisconnected(disconnects...)
	// test if disconnected peers have been removed from peerPool
	if pp.Has(s.IDs[peer]) {
		t.Fatalf("peer test-%v not dropped: %v (%v)", peer, pp, s.IDs)
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
