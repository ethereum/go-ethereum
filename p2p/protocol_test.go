package p2p

import (
	"fmt"
	"testing"
)

func TestBaseProtocolDisconnect(t *testing.T) {
	peer := NewPeer(NewSimpleClientIdentity("p1", "", "", "foo"), nil)
	peer.ourID = NewSimpleClientIdentity("p2", "", "", "bar")
	peer.pubkeyHook = func(*peerAddr) error { return nil }

	rw1, rw2 := MsgPipe()
	done := make(chan struct{})
	go func() {
		if err := expectMsg(rw2, handshakeMsg); err != nil {
			t.Error(err)
		}
		err := rw2.EncodeMsg(handshakeMsg,
			baseProtocolVersion,
			"",
			[]interface{}{},
			0,
			make([]byte, 64),
		)
		if err != nil {
			t.Error(err)
		}
		if err := expectMsg(rw2, getPeersMsg); err != nil {
			t.Error(err)
		}
		if err := rw2.EncodeMsg(discMsg, DiscQuitting); err != nil {
			t.Error(err)
		}
		close(done)
	}()

	if err := runBaseProtocol(peer, rw1); err == nil {
		t.Errorf("base protocol returned without error")
	} else if reason, ok := err.(discRequestedError); !ok || reason != DiscQuitting {
		t.Errorf("base protocol returned wrong error: %v", err)
	}
	<-done
}

func expectMsg(r MsgReader, code uint64) error {
	msg, err := r.ReadMsg()
	if err != nil {
		return err
	}
	if err := msg.Discard(); err != nil {
		return err
	}
	if msg.Code != code {
		return fmt.Errorf("wrong message code: got %d, expected %d", msg.Code, code)
	}
	return nil
}
