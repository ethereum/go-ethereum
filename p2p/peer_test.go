package p2p

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
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

func testPeer(protos []Protocol) (io.Closer, *conn, *Peer, <-chan DiscReason) {
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

	return p1, &conn{p2, hs2}, peer, errc
}

func TestPeerProtoReadMsg(t *testing.T) {
	defer testlog(t).detach()

	done := make(chan struct{})
	proto := Protocol{
		Name:   "a",
		Length: 5,
		Run: func(peer *Peer, rw MsgReadWriter) error {
			if err := expectMsg(rw, 2, []uint{1}); err != nil {
				t.Error(err)
			}
			if err := expectMsg(rw, 3, []uint{2}); err != nil {
				t.Error(err)
			}
			if err := expectMsg(rw, 4, []uint{3}); err != nil {
				t.Error(err)
			}
			close(done)
			return nil
		},
	}

	closer, rw, _, errc := testPeer([]Protocol{proto})
	defer closer.Close()

	EncodeMsg(rw, baseProtocolLength+2, 1)
	EncodeMsg(rw, baseProtocolLength+3, 2)
	EncodeMsg(rw, baseProtocolLength+4, 3)

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

	closer, rw, _, errc := testPeer([]Protocol{proto})
	defer closer.Close()

	EncodeMsg(rw, 18, make([]byte, msgsize))
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
			if err := EncodeMsg(rw, 2); err == nil {
				t.Error("expected error for out-of-range msg code, got nil")
			}
			if err := EncodeMsg(rw, 1, "foo", "bar"); err != nil {
				t.Errorf("write error: %v", err)
			}
			return nil
		},
	}
	closer, rw, _, _ := testPeer([]Protocol{proto})
	defer closer.Close()

	if err := expectMsg(rw, 17, []string{"foo", "bar"}); err != nil {
		t.Error(err)
	}
}

func TestPeerWriteForBroadcast(t *testing.T) {
	defer testlog(t).detach()

	closer, rw, peer, peerErr := testPeer([]Protocol{discard})
	defer closer.Close()

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
		if err := expectMsg(rw, 16, nil); err != nil {
			t.Error(err)
		}
		close(read)
	}()

	// test successful write
	if err := peer.writeProtoMsg("discard", NewMsg(0)); err != nil {
		t.Errorf("expect no error for known protocol: %v", err)
	}
	select {
	case <-read:
	case err := <-peerErr:
		t.Fatalf("peer stopped: %v", err)
	}
}

func TestPeerPing(t *testing.T) {
	defer testlog(t).detach()

	closer, rw, _, _ := testPeer(nil)
	defer closer.Close()
	if err := EncodeMsg(rw, pingMsg); err != nil {
		t.Fatal(err)
	}
	if err := expectMsg(rw, pongMsg, nil); err != nil {
		t.Error(err)
	}
}

func TestPeerDisconnect(t *testing.T) {
	defer testlog(t).detach()

	closer, rw, _, disc := testPeer(nil)
	defer closer.Close()
	if err := EncodeMsg(rw, discMsg, DiscQuitting); err != nil {
		t.Fatal(err)
	}
	if err := expectMsg(rw, discMsg, []interface{}{DiscRequested}); err != nil {
		t.Error(err)
	}
	closer.Close() // make test end faster
	if reason := <-disc; reason != DiscRequested {
		t.Errorf("run returned wrong reason: got %v, want %v", reason, DiscRequested)
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

// expectMsg reads a message from r and verifies that its
// code and encoded RLP content match the provided values.
// If content is nil, the payload is discarded and not verified.
func expectMsg(r MsgReader, code uint64, content interface{}) error {
	msg, err := r.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != code {
		return fmt.Errorf("message code mismatch: got %d, expected %d", msg.Code, code)
	}
	if content == nil {
		return msg.Discard()
	} else {
		contentEnc, err := rlp.EncodeToBytes(content)
		if err != nil {
			panic("content encode error: " + err.Error())
		}
		// skip over list header in encoded value. this is temporary.
		contentEncR := bytes.NewReader(contentEnc)
		if k, _, err := rlp.NewStream(contentEncR).Kind(); k != rlp.List || err != nil {
			panic("content must encode as RLP list")
		}
		contentEnc = contentEnc[len(contentEnc)-contentEncR.Len():]

		actualContent, err := ioutil.ReadAll(msg.Payload)
		if err != nil {
			return err
		}
		if !bytes.Equal(actualContent, contentEnc) {
			return fmt.Errorf("message payload mismatch:\ngot:  %x\nwant: %x", actualContent, contentEnc)
		}
	}
	return nil
}
