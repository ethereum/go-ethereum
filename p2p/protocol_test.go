package p2p

import (
	"fmt"
	"net"
	"reflect"
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

func TestBaseProtocolPeers(t *testing.T) {
	id1 := NewSimpleClientIdentity("p1", "", "", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	id2 := NewSimpleClientIdentity("p2", "", "", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	cannedPeerList := []*peerAddr{
		{IP: net.ParseIP("1.2.3.4"), Port: 2222, Pubkey: []byte{}},
		{IP: net.ParseIP("5.6.7.8"), Port: 3333, Pubkey: []byte{}},
	}
	rw1, rw2 := MsgPipe()

	// run matcher, close pipe when addresses have arrived
	addrChan := make(chan *peerAddr, len(cannedPeerList))
	go func() {
		for _, want := range cannedPeerList {
			got := <-addrChan
			t.Logf("got peer: %+v", got)
			if !reflect.DeepEqual(want, got) {
				t.Errorf("mismatch: got %#v, want %#v", got, want)
			}
		}
		rw1.Close()
	}()

	// run first peer
	peer1 := NewPeer(id2, nil)
	peer1.ourID = id1
	peer1.pubkeyHook = func(*peerAddr) error { return nil }
	peer1.otherPeers = func() []*Peer {
		pl := make([]*Peer, len(cannedPeerList))
		for i, addr := range cannedPeerList {
			pl[i] = &Peer{listenAddr: addr}
		}
		return pl
	}
	go runBaseProtocol(peer1, rw2)

	// run second peer
	peer2 := NewPeer(id1, nil)
	peer2.ourID = id2
	peer2.pubkeyHook = func(*peerAddr) error { return nil }
	peer2.otherPeers = func() []*Peer { return nil }
	peer2.newPeerAddr = addrChan // feed peer suggestions into matcher
	if err := runBaseProtocol(peer2, rw1); err != ErrPipeClosed {
		t.Errorf("peer2 terminated with unexpected error: %v", err)
	}
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
