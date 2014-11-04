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

	"github.com/ethereum/go-ethereum/ethutil"
)

func init() {
	ethlog.AddLogSystem(ethlog.NewStdLogSystem(os.Stdout, log.LstdFlags, ethlog.DebugLevel))
}

func setupMessenger(handlers Handlers) (net.Conn, *Peer, *messenger) {
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
		return fmt.Errorf("first message should be handshake, got %x", msg.Code)
	}
	if err := msg.Discard(); err != nil {
		return err
	}
	// send empty handshake
	pubkey := make([]byte, 64)
	msg = NewMsg(handshakeMsg, p2pVersion, "testid", nil, 9999, pubkey)
	return writeMsg(w, msg)
}

type testMsg struct {
	code MsgCode
	data *ethutil.Value
}

type testProto struct {
	recv chan testMsg
}

func (*testProto) Offset() MsgCode { return 5 }

func (tp *testProto) Start(peer *Peer, rw MsgReadWriter) error {
	return MsgLoop(rw, 1024, func(code MsgCode, data *ethutil.Value) error {
		logger.Debugf("testprotocol got msg: %d\n", code)
		tp.recv <- testMsg{code, data}
		return nil
	})
}

func TestRead(t *testing.T) {
	testProtocol := &testProto{make(chan testMsg)}
	handlers := Handlers{"a": func() Protocol { return testProtocol }}
	net, peer, mess := setupMessenger(handlers)
	bufr := bufio.NewReader(net)
	defer peer.Stop()
	if err := performTestHandshake(bufr, net); err != nil {
		t.Fatalf("handshake failed: %v", err)
	}

	mess.setRemoteProtocols([]string{"a"})
	writeMsg(net, NewMsg(17, uint32(1), "000"))
	select {
	case msg := <-testProtocol.recv:
		if msg.code != 1 {
			t.Errorf("incorrect msg code %d relayed to protocol", msg.code)
		}
		expdata := []interface{}{1, []byte{0x30, 0x30, 0x30}}
		if !reflect.DeepEqual(msg.data.Slice(), expdata) {
			t.Errorf("incorrect msg data %#v", msg.data.Slice())
		}
	case <-time.After(2 * time.Second):
		t.Errorf("receive timeout")
	}
}

func TestWriteProtoMsg(t *testing.T) {
	handlers := make(Handlers)
	testProtocol := &testProto{recv: make(chan testMsg, 1)}
	handlers["a"] = func() Protocol { return testProtocol }
	net, peer, mess := setupMessenger(handlers)
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
	if err := mess.writeProtoMsg("a", NewMsg(3)); err != nil {
		t.Errorf("expect no error for known protocol: %v", err)
	}
	select {
	case msg := <-read:
		if msg.Code != 19 {
			t.Errorf("wrong code, got %d, expected %d", msg.Code, 19)
		}
		msg.Discard()
	case err := <-readerr:
		t.Errorf("read error: %v", err)
	}
}

func TestPulse(t *testing.T) {
	net, peer, _ := setupMessenger(nil)
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
		t.Errorf("expected ping message, got %x", msg.Code)
	}
	if d := after.Sub(before); d < pingTimeout {
		t.Errorf("ping sent too early after %v, expected at least %v", d, pingTimeout)
	}
}
