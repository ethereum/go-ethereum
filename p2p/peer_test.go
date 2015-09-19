// Copyright 2014 The go-ethereum Authors
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

package p2p

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
)

var discard = Protocol{
	Name:   "discard",
	Length: 1,
	Run: func(p *Peer, rw MsgReadWriter) error {
		for {
			msg, err := rw.ReadMsg()
			if err != nil {
				return err
			}
			fmt.Printf("discarding %d\n", msg.Code)
			if err = msg.Discard(); err != nil {
				return err
			}
		}
	},
}

func testPeer(protos []Protocol) (*devConn, *Peer, <-chan DiscReason) {
	fd1, fd2 := net.Pipe()
	k1, k2 := newkey(), newkey()
	c1 := &conn{transport: newDevConn(fd1, k1, &k2.PublicKey)}
	for _, p := range protos {
		c1.caps = append(c1.caps, p.cap())
	}
	peer := newPeer(c1, protos)
	errc := make(chan DiscReason, 1)
	go func() { errc <- peer.run() }()

	c2 := newDevConn(fd2, k2, nil)
	c2.addProtocols(len(protos))
	return c2, peer, errc
}

func TestPeerProtoReadMsg(t *testing.T) {
	done := make(chan struct{})
	proto := Protocol{
		Name:   "a",
		Length: 5,
		Run: func(peer *Peer, rw MsgReadWriter) error {
			if err := ExpectMsg(rw, 2, []uint{1}); err != nil {
				t.Error(err)
			}
			if err := ExpectMsg(rw, 3, []uint{2}); err != nil {
				t.Error(err)
			}
			if err := ExpectMsg(rw, 4, []uint{3}); err != nil {
				t.Error(err)
			}
			close(done)
			return nil
		},
	}

	conn, _, errc := testPeer([]Protocol{proto})
	defer conn.Close()

	Send(conn.protocols[1], 2, []uint{1})
	Send(conn.protocols[1], 3, []uint{2})
	Send(conn.protocols[1], 4, []uint{3})

	select {
	case <-done:
	case err := <-errc:
		t.Errorf("peer returned: %v", err)
	case <-time.After(2 * time.Second):
		t.Errorf("receive timeout")
	}
}

func TestPeerProtoEncodeMsg(t *testing.T) {
	proto := Protocol{
		Name:   "a",
		Length: 2,
		Run: func(peer *Peer, rw MsgReadWriter) error {
			if err := SendItems(rw, 2); err == nil {
				t.Error("expected error for out-of-range msg code, got nil")
			}
			if err := SendItems(rw, 1, "foo", "bar"); err != nil {
				t.Errorf("write error: %v", err)
			}
			return nil
		},
	}
	conn, _, _ := testPeer([]Protocol{proto})
	defer conn.Close()

	if err := ExpectMsg(conn.protocols[1], 1, []string{"foo", "bar"}); err != nil {
		t.Error(err)
	}
}

func TestPeerPing(t *testing.T) {
	conn, _, _ := testPeer(nil)
	defer conn.Close()
	if err := SendItems(conn.protocols[0], pingMsg); err != nil {
		t.Fatal(err)
	}
	if err := ExpectMsg(conn.protocols[0], pongMsg, nil); err != nil {
		t.Error(err)
	}
}

func TestPeerDisconnect(t *testing.T) {
	conn, _, disc := testPeer(nil)
	defer conn.Close()
	if err := SendItems(conn.protocols[0], discMsg, DiscQuitting); err != nil {
		t.Fatal(err)
	}
	select {
	case reason := <-disc:
		if reason != DiscRequested {
			t.Errorf("run returned wrong reason: got %v, want %v", reason, DiscRequested)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("peer did not return")
	}
}

// This test is supposed to verify that Peer can reliably handle
// multiple causes of disconnection occurring at the same time.
func TestPeerDisconnectRace(t *testing.T) {
	maybe := func() bool { return rand.Intn(1) == 1 }

	for i := 0; i < 100; i++ {
		protoclose := make(chan error, 1)
		protodisc := make(chan DiscReason, 1)
		conn, p, disc := testPeer([]Protocol{
			{
				Name:   "closereq",
				Run:    func(p *Peer, rw MsgReadWriter) error { return <-protoclose },
				Length: 1,
			},
			{
				Name:   "disconnect",
				Run:    func(p *Peer, rw MsgReadWriter) error { p.Disconnect(<-protodisc); return nil },
				Length: 1,
			},
		})
		conn.Handshake()

		// Simulate incoming messages.
		go SendItems(conn.protocols[1], 1)
		go SendItems(conn.protocols[2], 2)
		// Close the network connection.
		go conn.Close()
		// Make protocol "closereq" return.
		protoclose <- errors.New("protocol closed")
		// Make protocol "disconnect" call peer.Disconnect
		protodisc <- DiscAlreadyConnected
		// In some cases, simulate something else calling peer.Disconnect.
		if maybe() {
			go p.Disconnect(DiscInvalidIdentity)
		}
		// In some cases, simulate remote requesting a disconnect.
		if maybe() {
			go SendItems(conn.protocols[0], discMsg, DiscQuitting)
		}

		select {
		case <-disc:
		case <-time.After(2 * time.Second):
			// Peer.run should return quickly. If it doesn't the Peer
			// goroutines are probably deadlocked. Call panic in order to
			// show the stacks.
			panic("Peer.run took to long to return.")
		}
	}
}

func TestNewPeer(t *testing.T) {
	name := "nodename"
	caps := []Cap{{"foo", 2}, {"bar", 3}}
	id := randomID()
	p := NewPeer(id, name, caps)
	if p.ID() != id {
		t.Errorf("ID mismatch: got %v, expected %v", p.ID(), id)
	}
	if p.Name() != name {
		t.Errorf("Name mismatch: got %v, expected %v", p.Name(), name)
	}
	if !reflect.DeepEqual(p.Caps(), caps) {
		t.Errorf("Caps mismatch: got %v, expected %v", p.Caps(), caps)
	}

	p.Disconnect(DiscAlreadyConnected) // Should not hang
}

func TestMatchProtocols(t *testing.T) {
	tests := map[string]struct {
		Remote []Cap
		Local  []Protocol
		Match  []*protoRW
	}{
		"no remote caps": {
			Local: []Protocol{{Name: "a"}},
		},
		"no local protocols": {
			Remote: []Cap{{Name: "a"}},
		},
		"no mutual protocols": {
			Remote: []Cap{{Name: "a"}},
			Local:  []Protocol{{Name: "b"}},
		},
		"some matches": {
			Remote: []Cap{{Name: "local"}, {Name: "match1"}, {Name: "match2"}},
			Local:  []Protocol{{Name: "match1"}, {Name: "match2"}, {Name: "remote"}},
			Match: []*protoRW{
				{Protocol: Protocol{Name: "match1"}, offset: 16},
				{Protocol: Protocol{Name: "match2"}, offset: 16},
			},
		},
		"alphabetical ordering": {
			Remote: []Cap{{Name: "aa"}, {Name: "ab"}, {Name: "bb"}, {Name: "ba"}},
			Local:  []Protocol{{Name: "ba"}, {Name: "bb"}, {Name: "ab"}, {Name: "aa"}},
			Match: []*protoRW{
				{Protocol: Protocol{Name: "aa"}, offset: 16},
				{Protocol: Protocol{Name: "ab"}, offset: 16},
				{Protocol: Protocol{Name: "ba"}, offset: 16},
				{Protocol: Protocol{Name: "bb"}, offset: 16},
			},
		},
		"no mutual versions": {
			Remote: []Cap{{Version: 1}},
			Local:  []Protocol{{Version: 2}},
		},
		"multiple versions, single common": {
			Remote: []Cap{{Version: 1}, {Version: 2}},
			Local:  []Protocol{{Version: 2}, {Version: 3}},
			Match: []*protoRW{
				{Protocol: Protocol{Version: 2}, offset: 16},
			},
		},
		"multiple versions, multiple common": {
			Remote: []Cap{{Version: 1}, {Version: 2}, {Version: 3}, {Version: 4}},
			Local:  []Protocol{{Version: 2}, {Version: 3}},
			Match: []*protoRW{
				{Protocol: Protocol{Version: 3}, offset: 16},
			},
		},
		"version ordering": {
			Remote: []Cap{{Version: 4}, {Version: 1}, {Version: 3}, {Version: 2}},
			Local:  []Protocol{{Version: 2}, {Version: 3}, {Version: 1}},
			Match: []*protoRW{
				{Protocol: Protocol{Version: 3}, offset: 16},
			},
		},
		"versions overriding subprotocol lengths": {
			Remote: []Cap{{Version: 1}, {Version: 2}, {Version: 3}, {Name: "a"}},
			Local:  []Protocol{{Version: 1, Length: 1}, {Version: 2, Length: 2}, {Version: 3, Length: 3}, {Name: "a"}},
			Match: []*protoRW{
				{Protocol: Protocol{Version: 3, Length: 3}, offset: 16},
				{Protocol: Protocol{Name: "a"}, offset: 19},
			},
		},
	}

	for tname, tt := range tests {
		result := matchProtocols(tt.Local, tt.Remote)
		if !reflect.DeepEqual(result, tt.Match) {
			t.Errorf("%s: wrong result\ngot %s\nwant: %s", tname, spew.Sdump(result), spew.Sdump(tt.Match))
		}
	}
}
