// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package testing

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
)

// ProtocolSession is a quasi simulation of a pivot node running
// a service and a number of dummy peers that can send (trigger) or
// receive (expect) messages
type ProtocolSession struct {
	Server  *p2p.Server
	IDs     []discover.NodeID
	adapter *adapters.SimAdapter
	events  chan *p2p.PeerEvent
}

// Exchange is the basic units of protocol tests
// the triggers and expects in the arrays are run immediately and asynchronously
// thus one cannot have multiple expects for the SAME peer with DIFFERENT message types
// because it's unpredictable which expect will receive which message
// (with expect #1 and #2, messages might be sent #2 and #1, and both expects will complain about wrong message code)
// an exchange is defined on a session
type Exchange struct {
	Label    string
	Triggers []Trigger
	Expects  []Expect
}

// Trigger is part of the exchange, incoming message for the pivot node
// sent by a peer
type Trigger struct {
	Msg     interface{}     // type of message to be sent
	Code    uint64          // code of message is given
	Peer    discover.NodeID // the peer to send the message to
	Timeout time.Duration   // timeout duration for the sending
}

// Expect is part of an exchange, outgoing message from the pivot node
// received by a peer
type Expect struct {
	Msg     interface{}     // type of message to expect
	Code    uint64          // code of message is now given
	Peer    discover.NodeID // the peer that expects the message
	Timeout time.Duration   // timeout duration for receiving
}

// Disconnect represents a disconnect event, used and checked by TestDisconnected
type Disconnect struct {
	Peer  discover.NodeID // discconnected peer
	Error error           // disconnect reason
}

// trigger sends messages from peers
func (self *ProtocolSession) trigger(trig Trigger) error {
	simNode, ok := self.adapter.GetNode(trig.Peer)
	if !ok {
		return fmt.Errorf("trigger: peer %v does not exist (1- %v)", trig.Peer, len(self.IDs))
	}
	mockNode, ok := simNode.Services()[0].(*mockNode)
	if !ok {
		return fmt.Errorf("trigger: peer %v is not a mock", trig.Peer)
	}

	errc := make(chan error)

	go func() {
		errc <- mockNode.Trigger(&trig)
	}()

	t := trig.Timeout
	if t == time.Duration(0) {
		t = 1000 * time.Millisecond
	}
	select {
	case err := <-errc:
		return err
	case <-time.After(t):
		return fmt.Errorf("timout expecting %v to send to peer %v", trig.Msg, trig.Peer)
	}
}

// expect checks an expectation of a message sent out by the pivot node
func (self *ProtocolSession) expect(exp Expect) error {
	if exp.Msg == nil {
		return errors.New("no message to expect")
	}
	simNode, ok := self.adapter.GetNode(exp.Peer)
	if !ok {
		return fmt.Errorf("trigger: peer %v does not exist (1- %v)", exp.Peer, len(self.IDs))
	}
	mockNode, ok := simNode.Services()[0].(*mockNode)
	if !ok {
		return fmt.Errorf("trigger: peer %v is not a mock", exp.Peer)
	}

	errc := make(chan error)
	go func() {
		errc <- mockNode.Expect(&exp)
	}()

	t := exp.Timeout
	if t == time.Duration(0) {
		t = 2000 * time.Millisecond
	}
	select {
	case err := <-errc:
		return err
	case <-time.After(t):
		return fmt.Errorf("timout expecting %v sent to peer %v", exp.Msg, exp.Peer)
	}
}

// TestExchanges tests a series of exchanges against the session
func (self *ProtocolSession) TestExchanges(exchanges ...Exchange) error {
	// launch all triggers of this exchanges

	for _, e := range exchanges {
		errc := make(chan error, len(e.Triggers)+len(e.Expects))
		for _, trig := range e.Triggers {
			errc <- self.trigger(trig)
		}

		// each expectation is spawned in separate go-routine
		// expectations of an exchange are conjunctive but unordered, i.e.,
		// only all of them arriving constitutes a pass
		// each expectation is meant to be for a different peer, otherwise they are expected to panic
		// testing of an exchange blocks until all expectations are decided
		// an expectation is decided if
		//  expected message arrives OR
		// an unexpected message arrives (panic)
		// times out on their individual timeout
		for _, ex := range e.Expects {
			// expect msg spawned to separate go routine
			go func(exp Expect) {
				errc <- self.expect(exp)
			}(ex)
		}

		// time out globally or finish when all expectations satisfied
		timeout := time.After(5 * time.Second)
		for i := 0; i < len(e.Triggers)+len(e.Expects); i++ {
			select {
			case err := <-errc:
				if err != nil {
					return fmt.Errorf("exchange failed with: %v", err)
				}
			case <-timeout:
				return fmt.Errorf("exchange %v: '%v' timed out", i, e.Label)
			}
		}
	}
	return nil
}

// TestDisconnected tests the disconnections given as arguments
// the disconnect structs describe what disconnect error is expected on which peer
func (self *ProtocolSession) TestDisconnected(disconnects ...*Disconnect) error {
	expects := make(map[discover.NodeID]error)
	for _, disconnect := range disconnects {
		expects[disconnect.Peer] = disconnect.Error
	}

	timeout := time.After(time.Second)
	for len(expects) > 0 {
		select {
		case event := <-self.events:
			if event.Type != p2p.PeerEventTypeDrop {
				continue
			}
			expectErr, ok := expects[event.Peer]
			if !ok {
				continue
			}

			if !(expectErr == nil && event.Error == "" || expectErr != nil && expectErr.Error() == event.Error) {
				return fmt.Errorf("unexpected error on peer %v. expected '%v', got '%v'", event.Peer, expectErr, event.Error)
			}
			delete(expects, event.Peer)
		case <-timeout:
			return fmt.Errorf("timed out waiting for peers to disconnect")
		}
	}
	return nil
}
