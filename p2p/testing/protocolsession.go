package testing

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
)

type ProtocolSession struct {
	Server  *p2p.Server
	Ids     []*adapters.NodeId
	adapter *adapters.SimAdapter
	events  chan *p2p.PeerEvent
}

// exchanges are the basic units of protocol tests
// the triggers and expects in the arrays are run immediately and asynchronously
// thus one cannot have multiple expects for the SAME peer with the DIFFERENT messagetypes
// because it's unpredictable which expect will receive which message
// (with expect #1 and #2, messages might be sent #2 and #1, and both expects will complain about wrong message code)
// an exchange is defined on a session
type Exchange struct {
	Label    string
	Triggers []Trigger
	Expects  []Expect
}

// part of the exchange, incoming message from a set of peers
type Trigger struct {
	Msg     interface{}      // type of message to be sent
	Code    uint64           // code of message is given
	Peer    *adapters.NodeId // the peer to send the message to
	Timeout time.Duration    // timeout duration for the sending
}

type Expect struct {
	Msg     interface{}      // type of message to expect
	Code    uint64           // code of message is now given
	Peer    *adapters.NodeId // the peer that expects the message
	Timeout time.Duration    // timeout duration for receiving
}

type Disconnect struct {
	Peer  *adapters.NodeId // discconnected peer
	Error error            // disconnect reason
}

// trigger sends messages from peers
func (self *ProtocolSession) trigger(trig Trigger) error {
	simNode, ok := self.adapter.GetNode(trig.Peer.NodeID)
	if !ok {
		return fmt.Errorf("trigger: peer %v does not exist (1- %v)", trig.Peer, len(self.Ids))
	}
	mockNode, ok := simNode.GetService("mock").(*mockNode)
	if !ok {
		return fmt.Errorf("trigger: peer %v is not a mock", trig.Peer)
	}

	errc := make(chan error)

	go func() {
		log.Trace(fmt.Sprintf("trigger %v (%v)....", trig.Msg, trig.Code))
		errc <- mockNode.Trigger(&trig)
		log.Trace(fmt.Sprintf("triggered %v (%v)", trig.Msg, trig.Code))
	}()

	t := trig.Timeout
	if t == time.Duration(0) {
		t = 1000 * time.Millisecond
	}
	alarm := time.NewTimer(t)
	select {
	case err := <-errc:
		return err
	case <-alarm.C:
		return fmt.Errorf("timout expecting %v to send to peer %v", trig.Msg, trig.Peer)
	}
}

// expect checks an expectation
func (self *ProtocolSession) expect(exp Expect) error {
	if exp.Msg == nil {
		return errors.New("no message to expect")
	}
	simNode, ok := self.adapter.GetNode(exp.Peer.NodeID)
	if !ok {
		return fmt.Errorf("trigger: peer %v does not exist (1- %v)", exp.Peer, len(self.Ids))
	}
	mockNode, ok := simNode.GetService("mock").(*mockNode)
	if !ok {
		return fmt.Errorf("trigger: peer %v is not a mock", exp.Peer)
	}

	errc := make(chan error)
	go func() {
		log.Trace(fmt.Sprintf("waiting for msg, %v", exp.Msg))
		errc <- mockNode.Expect(&exp)
	}()

	t := exp.Timeout
	if t == time.Duration(0) {
		t = 2000 * time.Millisecond
	}
	alarm := time.NewTimer(t)
	select {
	case err := <-errc:
		log.Trace(fmt.Sprintf("expected msg arrives with error %v", err))
		return err
	case <-alarm.C:
		return fmt.Errorf("timout expecting %v sent to peer %v", exp.Msg, exp.Peer)
	}
}

// TestExchange tests a series of exchanges againsts the session
func (self *ProtocolSession) TestExchanges(exchanges ...Exchange) error {
	// launch all triggers of this exchanges

	for i, e := range exchanges {
		errc := make(chan error)
		wg := &sync.WaitGroup{}
		for _, trig := range e.Triggers {
			err := self.trigger(trig)
			if err != nil {
				errc <- err
			}
		}

		// each expectation is spawned in separate go-routine
		// expectations of an exchange are conjunctive but uordered, i.e., only all of them arriving constitutes a pass
		// each expectation is meant to be for a different peer, otherwise they are expected to panic
		// testing of an exchange blocks until all expectations are decided
		// an expectation is decided if
		//  expected message arrives OR
		// an unexpected message arrives (panic)
		// times out on their individual tiemeout
		for _, ex := range e.Expects {
			wg.Add(1)
			// expect msg spawned to separate go routine
			go func(exp Expect) {
				defer wg.Done()
				err := self.expect(exp)
				if err != nil {
					log.Trace(fmt.Sprintf("expect msg fails %v", err))
					errc <- err
				}
			}(ex)
		}

		// wait for all expectations
		go func() {
			wg.Wait()
			close(errc)
		}()

		// time out globally or finish when all expectations satisfied
		alarm := time.NewTimer(1000 * time.Millisecond)
		select {

		case err := <-errc:
			if err != nil {
				return fmt.Errorf("exchange failed with: %v", err)
			} else {
				log.Trace(fmt.Sprintf("exchange %v: '%v' run successfully", i, e.Label))
			}
		case <-alarm.C:
			return fmt.Errorf("exchange %v: '%v' timed out", i, e.Label)
		}
	}
	return nil
}

func (self *ProtocolSession) TestDisconnected(disconnects ...*Disconnect) error {
	expects := make(map[discover.NodeID]error)
	for _, disconnect := range disconnects {
		expects[disconnect.Peer.NodeID] = disconnect.Error
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

			if !((expectErr == nil && event.Error == "") || expectErr != nil && event.Error != "" && expectErr.Error() == event.Error) {
				return fmt.Errorf("unexpected error on peer %v. expected '%v', got '%v'", event.Peer, expectErr, event.Error)
			}
			delete(expects, event.Peer)
		case <-timeout:
			return fmt.Errorf("timed out waiting for peers to disconnect")
		}
	}
	return nil
}
