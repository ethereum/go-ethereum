// Package protocols helpers_test make it easier to
// write protocol tests by providing convenience functions and structures
// protocols uses these helpers for  its own tests
// but ideally should sit in p2p/protocols/testing/ subpackage
package protocols

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

// a session represents a protocol running on multiple peer connections with single local node
type Session struct {
	Peers []*p2p.MsgPipeRW
	Errs  []error
	t     *testing.T
}

// exchanges are the basic units of protocol tests
// an exchange is defined on a session
type Exchange struct {
	Triggers []Trigger
	Expects  []Expect
}

// part of the exchange, incoming message from a set of peers
type Trigger struct {
	Msg     interface{}   // type of message to be sent
	Code    uint64        // code of message is given
	Peer    int           // the peer to send the message to
	Timeout time.Duration // timeout duration for the sending
}

type Expect struct {
	Msg     interface{}   // type of message to expect
	Code    uint64        // code of message is now given
	Peer    int           // the peer-connection index to expect the message from
	Timeout time.Duration // timeout duration of receiving
}

// NewSession creates a session by setting up a local peer with a prescribed set of peers
// wg if present allows wg.Wait() be used to block until all peers disconnect
// disconnect reason errors are written in session.Errs (correcponding to session,Peers)
func NewSession(t *testing.T, protocol *p2p.Protocol, peerCount int, wg *sync.WaitGroup) *Session {
	self := &Session{t: t, Errs: make([]error, peerCount)}
	caps := []p2p.Cap{p2p.Cap{protocol.Name, protocol.Version}}
	if wg != nil {
		wg.Add(peerCount)
	}
	var runs []func(int)
	var rws []p2p.MsgReadWriter
	// connect peerCount number of peers
	for i := 0; i < peerCount; i++ {
		rw, rrw := p2p.MsgPipe()
		self.Peers = append(self.Peers, rrw)
		self.Errs = append(self.Errs, nil)
		rws = append(rws, rw)
		key, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("unable to generate key")
		}
		pubkey := crypto.FromECDSAPub(&key.PublicKey)
		var id discover.NodeID
		copy(id[:], pubkey)
		runs = append(runs, func(j int) {
			name := fmt.Sprintf("test-%d", j)
			self.Errs[j] = protocol.Run(p2p.NewPeer(id, name, caps), rws[j])
			if wg != nil {
				wg.Done()
			}
		})
	}
	// start protocols on each peer connection
	for i, f := range runs {
		go f(i)
	}
	return self
}

// trigger sends messages from peers
func (self Session) trigger(trig Trigger) error {
	if self.Errs[trig.Peer] != nil {
		return fmt.Errorf("peer %v already disconnected with %v", trig.Peer, self.Errs[trig.Peer])
	}
	errc := make(chan error)
	go func() {
		errc <- p2p.Send(self.Peers[trig.Peer], trig.Code, trig.Msg)
	}()

	t := trig.Timeout
	if t == time.Duration(0) {
		t = 100 * time.Millisecond
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
func (self Session) expect(exp Expect) error {
	if exp.Msg == nil {
		panic("no message to expect")
	}
	if exp.Peer >= len(self.Errs) {
		panic(fmt.Sprintf("peer %v does not exist: %v", exp.Peer))
	}
	if self.Errs[exp.Peer] != nil {
		panic(fmt.Sprintf("peer %v already disconnected with: %v", exp.Peer, self.Errs))
	}
	errc := make(chan error)
	go func() {
		errc <- p2p.ExpectMsg(self.Peers[exp.Peer], exp.Code, exp.Msg)
	}()

	t := exp.Timeout
	if t == time.Duration(0) {
		t = 100 * time.Millisecond
	}
	alarm := time.NewTimer(t)
	select {
	case err := <-errc:
		return err
	case <-alarm.C:
		return fmt.Errorf("timout expecting %v sent to peer %v", exp.Msg, exp.Peer)
	}
	// fatal upon encountering first exchange error
}

// TestExchange tests a series of exchanges againsts the session
func (self Session) TestExchanges(exchanges ...Exchange) {
	// launch all triggers of this exchanges

	errc := make(chan error)
	ewg := &sync.WaitGroup{}
	for i, e := range exchanges {
		ewg.Add(1)
		wg := &sync.WaitGroup{}
		for _, trig := range e.Triggers {
			wg.Add(1)
			// separate go routing to allow parallel requests
			go func(t Trigger) {
				defer wg.Done()
				err := self.trigger(t)
				i++
				if err != nil {
					errc <- err
				}
			}(trig)
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
					errc <- err
				}
			}(ex)
		}

		// wait for all expectations
		wg.Wait()
		ewg.Done()

	}
	// wait for all expectations
	go func() {
		ewg.Wait()
		close(errc)
	}()

	// time out globally or finish when all expectations satisfied
	alarm := time.NewTimer(5000 * time.Millisecond)
	select {

	case err := <-errc:
		if err != nil {
			self.t.Fatalf("exchange failed with: %v", err)
		}
	case <-alarm.C:
		self.t.Fatalf("exchange timed out")
	}
}

func (self Session) TestDisconnects(errs ...error) {
	for i, err := range errs {
		if !((err == nil && self.Errs[i] == nil) || err != nil && self.Errs[i] != nil && err.Error() == self.Errs[i].Error()) {
			self.t.Fatalf("unexpected error on peer %v: '%v', wanted '%v'", i, self.Errs[i], err)
		}
	}
}
