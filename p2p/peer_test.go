package p2p

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"testing"
	"time"
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

func testPeer(protos []Protocol) (func(), *conn, *Peer, <-chan DiscReason) {
	fd1, _ := net.Pipe()
	hs1 := &protoHandshake{ID: randomID(), Version: baseProtocolVersion}
	hs2 := &protoHandshake{ID: randomID(), Version: baseProtocolVersion}
	for _, p := range protos {
		hs1.Caps = append(hs1.Caps, p.cap())
		hs2.Caps = append(hs2.Caps, p.cap())
	}

	p1, p2 := MsgPipe()
	peer := newPeer(fd1, &conn{p1, hs1}, protos)
	errc := make(chan DiscReason, 1)
	go func() { errc <- peer.run() }()

	closer := func() {
		p1.Close()
		fd1.Close()
	}
	return closer, &conn{p2, hs2}, peer, errc
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

	closer, rw, _, errc := testPeer([]Protocol{proto})
	defer closer()

	Send(rw, baseProtocolLength+2, []uint{1})
	Send(rw, baseProtocolLength+3, []uint{2})
	Send(rw, baseProtocolLength+4, []uint{3})

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
	closer, rw, _, _ := testPeer([]Protocol{proto})
	defer closer()

	if err := ExpectMsg(rw, 17, []string{"foo", "bar"}); err != nil {
		t.Error(err)
	}
}

func TestPeerPing(t *testing.T) {
	closer, rw, _, _ := testPeer(nil)
	defer closer()
	if err := SendItems(rw, pingMsg); err != nil {
		t.Fatal(err)
	}
	if err := ExpectMsg(rw, pongMsg, nil); err != nil {
		t.Error(err)
	}
}

func TestPeerDisconnect(t *testing.T) {
	closer, rw, _, disc := testPeer(nil)
	defer closer()
	if err := SendItems(rw, discMsg, DiscQuitting); err != nil {
		t.Fatal(err)
	}
	select {
	case reason := <-disc:
		if reason != DiscQuitting {
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

	for i := 0; i < 1000; i++ {
		protoclose := make(chan error)
		protodisc := make(chan DiscReason)
		closer, rw, p, disc := testPeer([]Protocol{
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

		// Simulate incoming messages.
		go SendItems(rw, baseProtocolLength+1)
		go SendItems(rw, baseProtocolLength+2)
		// Close the network connection.
		go closer()
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
			go SendItems(rw, discMsg, DiscQuitting)
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
