package testing

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/adapters"
)

type ProtocolSession struct {
	TestNodeAdapter
	Ids    []*adapters.NodeId
	ignore []uint64
}

type TestMessenger interface {
	ExpectMsg(uint64, interface{}) error
	TriggerMsg(uint64, interface{}) error
}

type TestNodeAdapter interface {
	p2p.Server
	GetPeer(id *adapters.NodeId) *adapters.Peer
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

func NewProtocolSession(na TestNodeAdapter, ids []*adapters.NodeId) *ProtocolSession {
	ps := &ProtocolSession{
		TestNodeAdapter: na,
		Ids:             ids,
	}
	return ps
}

func (self *ProtocolSession) SetIgnoreCodes(ignore ...uint64) {
	self.ignore = ignore
}

// trigger sends messages from peers
func (self *ProtocolSession) trigger(trig Trigger) error {
	peer := self.GetPeer(trig.Peer)
	if peer == nil {
		panic(fmt.Sprintf("trigger: peer %v does not exist (1- %v)", trig.Peer, len(self.Ids)))
	}
	if peer.MsgReadWriteCloser == nil {
		return fmt.Errorf("trigger: peer %v unreachable", trig.Peer)
	}
	errc := make(chan error)

	go func() {
		log.Trace(fmt.Sprintf("trigger %v (%v)....", trig.Msg, trig.Code))
		errc <- p2p.Send(peer, trig.Code, trig.Msg)
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
		panic("no message to expect")
	}
	peer := self.GetPeer(exp.Peer)
	if peer == nil {
		panic(fmt.Sprintf("expect: peer %v does not exist (1- %v)", exp.Peer, len(self.Ids)))
	}
	if peer.MsgReadWriteCloser== nil {
		return fmt.Errorf("trigger: peer %v unreachable", exp.Peer)
	}

	errc := make(chan error)
	go func() {
		var err error
		ignored := true
		log.Trace("waiting for msg", "code", exp.Code, "msg", exp.Msg)
		for ignored {
			ignored = false
			err = p2p.ExpectMsg(peer, exp.Code, exp.Msg)
			// frail, but we can't know what code expectmsg got otherwise
			// can we do better error reporting in p2p.ExpectMsg()?
			if err != nil {
				if strings.Contains(err.Error(), "code") {
					re, _ := regexp.Compile("got ([0-9]+),")
					match := re.FindStringSubmatch(err.Error())
					if len(match) > 1 {
						for _, codetoignore := range self.ignore {
							codewegot, err := strconv.ParseUint(match[1], 10, 64)
							if err == nil {
								if codetoignore == codewegot {
									ignored = true
									log.Trace("ignore msg with wrong code", "received", codewegot, "expected", exp.Code)
									break
								}
							} else {
								log.Warn("expectmsg errormsg parse error?!")
							}
						}
					} else {
						log.Warn("expectmsg errormsg parse error?!")
						break
					}
				}
			}
		}
		errc <- err
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

func (self *ProtocolSession) Stop() {
	for _, id := range self.Ids {
		p := self.GetPeer(id)
		if p != nil && p.MsgReadWriteCloser != nil {
			p.Close()
		}
	}
}
