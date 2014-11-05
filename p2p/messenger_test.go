package p2p

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	logpkg "github.com/ethereum/go-ethereum/logger"
)

func init() {
	logpkg.AddLogSystem(logpkg.NewStdLogSystem(os.Stdout, log.LstdFlags, logpkg.DebugLevel))
}

func testMessenger(handlers Handlers) (net.Conn, *Peer, *messenger) {
	conn1, conn2 := net.Pipe()
	id := NewSimpleClientIdentity("test", "0", "0", "public key")
	server := New(nil, conn1.LocalAddr(), id, handlers, 10, NewBlacklist())
	peer := server.addPeer(conn1, conn1.RemoteAddr(), true, 0)
	return conn2, peer, peer.messenger
}

func performTestHandshake(r *bufio.Reader, w io.Writer) error {
	// read remote handshake
	msg, err := readMsg(r)
	if err != nil {
		return fmt.Errorf("read error: %v", err)
	}
	if msg.Code != handshakeMsg {
		return fmt.Errorf("first message should be handshake, got %d", msg.Code)
	}
	if err := msg.Discard(); err != nil {
		return err
	}
	// send empty handshake
	pubkey := make([]byte, 64)
	msg = NewMsg(handshakeMsg, p2pVersion, "testid", nil, 9999, pubkey)
	return writeMsg(w, msg)
}

type testProtocol struct {
	offset MsgCode
	f      func(MsgReadWriter)
}

func (p *testProtocol) Offset() MsgCode {
	return p.offset
}

func (p *testProtocol) Start(peer *Peer, rw MsgReadWriter) error {
	p.f(rw)
	return nil
}

func TestRead(t *testing.T) {
	done := make(chan struct{})
	handlers := Handlers{
		"a": &testProtocol{5, func(rw MsgReadWriter) {
			msg, err := rw.ReadMsg()
			if err != nil {
				t.Errorf("read error: %v", err)
			}
			if msg.Code != 2 {
				t.Errorf("incorrect msg code %d relayed to protocol", msg.Code)
			}
			data, err := msg.Data()
			if err != nil {
				t.Errorf("data decoding error: %v", err)
			}
			expdata := []interface{}{1, []byte{0x30, 0x30, 0x30}}
			if !reflect.DeepEqual(data.Slice(), expdata) {
				t.Errorf("incorrect msg data %#v", data.Slice())
			}
			close(done)
		}},
	}

	net, peer, m := testMessenger(handlers)
	defer peer.Stop()
	bufr := bufio.NewReader(net)
	if err := performTestHandshake(bufr, net); err != nil {
		t.Fatalf("handshake failed: %v", err)
	}
	m.setRemoteProtocols([]string{"a"})

	writeMsg(net, NewMsg(18, 1, "000"))
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Errorf("receive timeout")
	}
}

func TestWriteFromProto(t *testing.T) {
	handlers := Handlers{
		"a": &testProtocol{2, func(rw MsgReadWriter) {
			if err := rw.WriteMsg(NewMsg(2)); err == nil {
				t.Error("expected error for out-of-range msg code, got nil")
			}
			if err := rw.WriteMsg(NewMsg(1)); err != nil {
				t.Errorf("write error: %v", err)
			}
		}},
	}
	net, peer, mess := testMessenger(handlers)
	defer peer.Stop()
	bufr := bufio.NewReader(net)
	if err := performTestHandshake(bufr, net); err != nil {
		t.Fatalf("handshake failed: %v", err)
	}
	mess.setRemoteProtocols([]string{"a"})

	msg, err := readMsg(bufr)
	if err != nil {
		t.Errorf("read error: %v")
	}
	if msg.Code != 17 {
		t.Errorf("incorrect message code: got %d, expected %d", msg.Code, 17)
	}
}

var discardProto = &testProtocol{1, func(rw MsgReadWriter) {
	for {
		msg, err := rw.ReadMsg()
		if err != nil {
			return
		}
		if err = msg.Discard(); err != nil {
			return
		}
	}
}}

func TestMessengerWriteProtoMsg(t *testing.T) {
	handlers := Handlers{"a": discardProto}
	net, peer, mess := testMessenger(handlers)
	defer peer.Stop()
	bufr := bufio.NewReader(net)
	if err := performTestHandshake(bufr, net); err != nil {
		t.Fatalf("handshake failed: %v", err)
	}
	mess.setRemoteProtocols([]string{"a"})

	// test write errors
	if err := mess.writeProtoMsg("b", NewMsg(3)); err == nil {
		t.Errorf("expected error for unknown protocol, got nil")
	}
	if err := mess.writeProtoMsg("a", NewMsg(8)); err == nil {
		t.Errorf("expected error for out-of-range msg code, got nil")
	} else if perr, ok := err.(*PeerError); !ok || perr.Code != InvalidMsgCode {
		t.Errorf("wrong error for out-of-range msg code, got %#v")
	}

	// test succcessful write
	read, readerr := make(chan Msg), make(chan error)
	go func() {
		if msg, err := readMsg(bufr); err != nil {
			readerr <- err
		} else {
			read <- msg
		}
	}()
	if err := mess.writeProtoMsg("a", NewMsg(0)); err != nil {
		t.Errorf("expect no error for known protocol: %v", err)
	}
	select {
	case msg := <-read:
		if msg.Code != 16 {
			t.Errorf("wrong code, got %d, expected %d", msg.Code, 16)
		}
		msg.Discard()
	case err := <-readerr:
		t.Errorf("read error: %v", err)
	}
}

func TestPulse(t *testing.T) {
	net, peer, _ := testMessenger(nil)
	defer peer.Stop()
	bufr := bufio.NewReader(net)
	if err := performTestHandshake(bufr, net); err != nil {
		t.Fatalf("handshake failed: %v", err)
	}

	before := time.Now()
	msg, err := readMsg(bufr)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	after := time.Now()
	if msg.Code != pingMsg {
		t.Errorf("expected ping message, got %d", msg.Code)
	}
	if d := after.Sub(before); d < pingTimeout {
		t.Errorf("ping sent too early after %v, expected at least %v", d, pingTimeout)
	}
}
