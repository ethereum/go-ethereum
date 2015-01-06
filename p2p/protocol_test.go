package p2p

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

type peerId struct {
	pubkey []byte
}

func (self *peerId) String() string {
	return fmt.Sprintf("test peer %x", self.Pubkey()[:4])
}

func (self *peerId) Pubkey() (pubkey []byte) {
	pubkey = self.pubkey
	if len(pubkey) == 0 {
		pubkey = crypto.GenerateNewKeyPair().PublicKey
		self.pubkey = pubkey
	}
	return
}

func newTestPeer() (peer *Peer) {
	peer = NewPeer(&peerId{}, []Cap{})
	peer.pubkeyHook = func(*peerAddr) error { return nil }
	peer.ourID = &peerId{}
	peer.listenAddr = &peerAddr{}
	peer.otherPeers = func() []*Peer { return nil }
	return
}

func TestBaseProtocolPeers(t *testing.T) {
	cannedPeerList := []*peerAddr{
		{IP: net.ParseIP("1.2.3.4"), Port: 2222, Pubkey: []byte{}},
		{IP: net.ParseIP("5.6.7.8"), Port: 3333, Pubkey: []byte{}},
	}
	var ownAddr *peerAddr = &peerAddr{IP: net.ParseIP("1.3.5.7"), Port: 1111, Pubkey: []byte{}}
	rw1, rw2 := MsgPipe()
	// run matcher, close pipe when addresses have arrived
	addrChan := make(chan *peerAddr, len(cannedPeerList))
	go func() {
		for _, want := range cannedPeerList {
			got := <-addrChan
			t.Logf("got peer: %+v", got)
			if !reflect.DeepEqual(want, got) {
				t.Errorf("mismatch:  got %#v, want %#v", got, want)
			}
		}
		close(addrChan)
		var own []*peerAddr
		var got *peerAddr
		for got = range addrChan {
			own = append(own, got)
		}
		if len(own) != 1 || !reflect.DeepEqual(ownAddr, own[0]) {
			t.Errorf("mismatch: peers own address is incorrectly or not given, got %v, want %#v", ownAddr)
		}
		rw2.Close()
	}()
	// run first peer
	peer1 := newTestPeer()
	peer1.ourListenAddr = ownAddr
	peer1.otherPeers = func() []*Peer {
		pl := make([]*Peer, len(cannedPeerList))
		for i, addr := range cannedPeerList {
			pl[i] = &Peer{listenAddr: addr}
		}
		return pl
	}
	go runBaseProtocol(peer1, rw1)
	// run second peer
	peer2 := newTestPeer()
	peer2.newPeerAddr = addrChan // feed peer suggestions into matcher
	if err := runBaseProtocol(peer2, rw2); err != ErrPipeClosed {
		t.Errorf("peer2 terminated with unexpected error: %v", err)
	}
}

func TestBaseProtocolDisconnect(t *testing.T) {
	peer := NewPeer(&peerId{}, nil)
	peer.ourID = &peerId{}
	peer.pubkeyHook = func(*peerAddr) error { return nil }

	rw1, rw2 := MsgPipe()
	done := make(chan struct{})
	go func() {
		if err := expectMsg(rw2, handshakeMsg); err != nil {
			t.Error(err)
		}
		err := EncodeMsg(rw2, handshakeMsg,
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
		if err := EncodeMsg(rw2, discMsg, DiscQuitting); err != nil {
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
