package testing

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/adapters"
)

type ProtocolSession struct {
	TestNodeAdapter
	Ids []*adapters.NodeId
}

type TestMessenger interface {
	ExpectMsg(uint64, interface{}) error
	TriggerMsg(uint64, interface{}) error
}

type TestNodeAdapter interface {
	GetPeer(id *adapters.NodeId) *adapters.Peer
	Connect([]byte) error
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

func NewProtocolSession(na adapters.NodeAdapter, ids []*adapters.NodeId) *ProtocolSession {
	ps := &ProtocolSession{
		TestNodeAdapter: na.(TestNodeAdapter),
		Ids:             ids,
	}
	return ps
}

// trigger sends messages from peers
func (self *ProtocolSession) trigger(trig Trigger) error {
	peer := self.GetPeer(trig.Peer)
	if peer == nil {
		panic(fmt.Sprintf("trigger: peer %v does not exist (1- %v)", trig.Peer, len(self.Ids)))
	}
	m := peer.Messenger
	if m == nil {
		return fmt.Errorf("trigger: peer %v unreachable", trig.Peer)
	}
	errc := make(chan error)

	go func() {
		glog.V(logger.Detail).Infof("trigger %v (%v)....", trig.Msg, trig.Code)
		errc <- m.(TestMessenger).TriggerMsg(trig.Code, trig.Msg)
		glog.V(logger.Detail).Infof("triggered %v (%v)", trig.Msg, trig.Code)
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
		panic("no message to expect")
	}
	peer := self.GetPeer(exp.Peer)
	if peer == nil {
		panic(fmt.Sprintf("expect: peer %v does not exist (1- %v)", exp.Peer, len(self.Ids)))
	}
	m := peer.Messenger
	if m == nil {
		return fmt.Errorf("trigger: peer %v unreachable", exp.Peer)
	}

	errc := make(chan error)
	go func() {
		glog.V(logger.Detail).Infof("waiting for msg, %v", exp.Msg)
		errc <- m.(TestMessenger).ExpectMsg(exp.Code, exp.Msg)
	}()

	t := exp.Timeout
	if t == time.Duration(0) {
		t = 1000 * time.Millisecond
	}
	alarm := time.NewTimer(t)
	select {
	case err := <-errc:
		glog.V(logger.Detail).Infof("expected msg arrives with error %v", err)
		return err
	case <-alarm.C:
		glog.V(logger.Detail).Infof("caught timeout")
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
					glog.V(logger.Detail).Infof("expect msg fails %v", err)
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
				glog.V(logger.Detail).Infof("exchange %v: '%v' run successfully", i, e.Label)
			}
		case <-alarm.C:
			return fmt.Errorf("exchange %v: '%v' timed out", i, e.Label)
		}
	}
	return nil
}

func (self *ProtocolSession) TestDisconnected(disconnects ...*Disconnect) error {
	for _, disconnect := range disconnects {
		id := disconnect.Peer
		err := disconnect.Error
		peer := self.GetPeer(id)

		alarm := time.NewTimer(1000 * time.Millisecond)
		select {
		case derr := <-peer.Errc:
			if !((err == nil && derr == nil) || err != nil && derr != nil && err.Error() == derr.Error()) {
				return fmt.Errorf("unexpected error on peer %v. expected '%v', got '%v'", id, err, derr)
			}
		case <-alarm.C:
			return fmt.Errorf("timed out waiting for peer %v to disconnect", id)
		}
	}
	return nil
}
