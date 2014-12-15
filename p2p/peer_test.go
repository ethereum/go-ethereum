package p2p

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"io"
	"io/ioutil"
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
			if err = msg.Discard(); err != nil {
				return err
			}
		}
	},
}

func testPeer(protos []Protocol) (net.Conn, *Peer, <-chan error) {
	conn1, conn2 := net.Pipe()
	id := NewSimpleClientIdentity("test", "0", "0", "public key")
	peer := newPeer(conn1, protos, nil)
	peer.ourID = id
	peer.pubkeyHook = func(*peerAddr) error { return nil }
	errc := make(chan error, 1)
	go func() {
		_, err := peer.loop()
		errc <- err
	}()
	return conn2, peer, errc
}

func TestPeerProtoReadMsg(t *testing.T) {
	defer testlog(t).detach()

	done := make(chan struct{})
	proto := Protocol{
		Name:   "a",
		Length: 5,
		Run: func(peer *Peer, rw MsgReadWriter) error {
			msg, err := rw.ReadMsg()
			if err != nil {
				t.Errorf("read error: %v", err)
			}
			if msg.Code != 2 {
				t.Errorf("incorrect msg code %d relayed to protocol", msg.Code)
			}
			data, err := ioutil.ReadAll(msg.Payload)
			if err != nil {
				t.Errorf("payload read error: %v", err)
			}
			expdata, _ := hex.DecodeString("0183303030")
			if !bytes.Equal(expdata, data) {
				t.Errorf("incorrect msg data %x", data)
			}
			close(done)
			return nil
		},
	}

	net, peer, errc := testPeer([]Protocol{proto})
	defer net.Close()
	peer.startSubprotocols([]Cap{proto.cap()})

	writeMsg(net, NewMsg(18, 1, "000"))
	select {
	case <-done:
	case err := <-errc:
		t.Errorf("peer returned: %v", err)
	case <-time.After(2 * time.Second):
		t.Errorf("receive timeout")
	}
}

func TestPeerProtoReadLargeMsg(t *testing.T) {
	defer testlog(t).detach()

	msgsize := uint32(10 * 1024 * 1024)
	done := make(chan struct{})
	proto := Protocol{
		Name:   "a",
		Length: 5,
		Run: func(peer *Peer, rw MsgReadWriter) error {
			msg, err := rw.ReadMsg()
			if err != nil {
				t.Errorf("read error: %v", err)
			}
			if msg.Size != msgsize+4 {
				t.Errorf("incorrect msg.Size, got %d, expected %d", msg.Size, msgsize)
			}
			msg.Discard()
			close(done)
			return nil
		},
	}

	net, peer, errc := testPeer([]Protocol{proto})
	defer net.Close()
	peer.startSubprotocols([]Cap{proto.cap()})

	writeMsg(net, NewMsg(18, make([]byte, msgsize)))
	select {
	case <-done:
	case err := <-errc:
		t.Errorf("peer returned: %v", err)
	case <-time.After(2 * time.Second):
		t.Errorf("receive timeout")
	}
}

func TestPeerProtoEncodeMsg(t *testing.T) {
	defer testlog(t).detach()

	proto := Protocol{
		Name:   "a",
		Length: 2,
		Run: func(peer *Peer, rw MsgReadWriter) error {
			if err := rw.EncodeMsg(2); err == nil {
				t.Error("expected error for out-of-range msg code, got nil")
			}
			if err := rw.EncodeMsg(1); err != nil {
				t.Errorf("write error: %v", err)
			}
			return nil
		},
	}
	net, peer, _ := testPeer([]Protocol{proto})
	defer net.Close()
	peer.startSubprotocols([]Cap{proto.cap()})

	bufr := bufio.NewReader(net)
	msg, err := readMsg(bufr)
	if err != nil {
		t.Errorf("read error: %v", err)
	}
	if msg.Code != 17 {
		t.Errorf("incorrect message code: got %d, expected %d", msg.Code, 17)
	}
}

func TestPeerWrite(t *testing.T) {
	defer testlog(t).detach()

	net, peer, peerErr := testPeer([]Protocol{discard})
	defer net.Close()
	peer.startSubprotocols([]Cap{discard.cap()})

	// test write errors
	if err := peer.writeProtoMsg("b", NewMsg(3)); err == nil {
		t.Errorf("expected error for unknown protocol, got nil")
	}
	if err := peer.writeProtoMsg("discard", NewMsg(8)); err == nil {
		t.Errorf("expected error for out-of-range msg code, got nil")
	} else if perr, ok := err.(*peerError); !ok || perr.Code != errInvalidMsgCode {
		t.Errorf("wrong error for out-of-range msg code, got %#v", err)
	}

	// setup for reading the message on the other end
	read := make(chan struct{})
	go func() {
		bufr := bufio.NewReader(net)
		msg, err := readMsg(bufr)
		if err != nil {
			t.Errorf("read error: %v", err)
		} else if msg.Code != 16 {
			t.Errorf("wrong code, got %d, expected %d", msg.Code, 16)
		}
		msg.Discard()
		close(read)
	}()

	// test succcessful write
	if err := peer.writeProtoMsg("discard", NewMsg(0)); err != nil {
		t.Errorf("expect no error for known protocol: %v", err)
	}
	select {
	case <-read:
	case err := <-peerErr:
		t.Fatalf("peer stopped: %v", err)
	}
}

func TestPeerActivity(t *testing.T) {
	// shorten inactivityTimeout while this test is running
	oldT := inactivityTimeout
	defer func() { inactivityTimeout = oldT }()
	inactivityTimeout = 20 * time.Millisecond

	net, peer, peerErr := testPeer([]Protocol{discard})
	defer net.Close()
	peer.startSubprotocols([]Cap{discard.cap()})

	sub := peer.activity.Subscribe(time.Time{})
	defer sub.Unsubscribe()

	for i := 0; i < 6; i++ {
		writeMsg(net, NewMsg(16))
		select {
		case <-sub.Chan():
		case <-time.After(inactivityTimeout / 2):
			t.Fatal("no event within ", inactivityTimeout/2)
		case err := <-peerErr:
			t.Fatal("peer error", err)
		}
	}

	select {
	case <-time.After(inactivityTimeout * 2):
	case <-sub.Chan():
		t.Fatal("got activity event while connection was inactive")
	case err := <-peerErr:
		t.Fatal("peer error", err)
	}
}

func TestNewPeer(t *testing.T) {
	id := NewSimpleClientIdentity("clientid", "version", "customid", "pubkey")
	caps := []Cap{{"foo", 2}, {"bar", 3}}
	p := NewPeer(id, caps)
	if !reflect.DeepEqual(p.Caps(), caps) {
		t.Errorf("Caps mismatch: got %v, expected %v", p.Caps(), caps)
	}
	if p.Identity() != id {
		t.Errorf("Identity mismatch: got %v, expected %v", p.Identity(), id)
	}
	// Should not hang.
	p.Disconnect(DiscAlreadyConnected)
}

func TestEOFSignal(t *testing.T) {
	rb := make([]byte, 10)

	// empty reader
	eof := make(chan struct{}, 1)
	sig := &eofSignal{new(bytes.Buffer), 0, eof}
	if n, err := sig.Read(rb); n != 0 || err != io.EOF {
		t.Errorf("Read returned unexpected values: (%v, %v)", n, err)
	}
	select {
	case <-eof:
	default:
		t.Error("EOF chan not signaled")
	}

	// count before error
	eof = make(chan struct{}, 1)
	sig = &eofSignal{bytes.NewBufferString("aaaaaaaa"), 4, eof}
	if n, err := sig.Read(rb); n != 8 || err != nil {
		t.Errorf("Read returned unexpected values: (%v, %v)", n, err)
	}
	select {
	case <-eof:
	default:
		t.Error("EOF chan not signaled")
	}

	// error before count
	eof = make(chan struct{}, 1)
	sig = &eofSignal{bytes.NewBufferString("aaaa"), 999, eof}
	if n, err := sig.Read(rb); n != 4 || err != nil {
		t.Errorf("Read returned unexpected values: (%v, %v)", n, err)
	}
	if n, err := sig.Read(rb); n != 0 || err != io.EOF {
		t.Errorf("Read returned unexpected values: (%v, %v)", n, err)
	}
	select {
	case <-eof:
	default:
		t.Error("EOF chan not signaled")
	}

	// no signal if neither occurs
	eof = make(chan struct{}, 1)
	sig = &eofSignal{bytes.NewBufferString("aaaaaaaaaaaaaaaaaaaaa"), 999, eof}
	if n, err := sig.Read(rb); n != 10 || err != nil {
		t.Errorf("Read returned unexpected values: (%v, %v)", n, err)
	}
	select {
	case <-eof:
		t.Error("unexpected EOF signal")
	default:
	}
}
