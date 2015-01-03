package p2p

import (
	"fmt"
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

func testPeerFree() (peer *Peer) {
	peer = NewPeer(&peerId{}, []Cap{})
	peer.pubkeyHook = func(*peerAddr) error { return nil }
	peer.ourID = &peerId{}
	peer.listenAddr = &peerAddr{}
	return
}

func TestPeersMsg(t *testing.T) {
	var peers []*Peer
	for i := 0; i < 3; i++ {
		peers = append(peers, testPeerFree())
	}
	peer1 := testPeerFree()
	peer1.newPeerAddr = make(chan *peerAddr)
	peer1.otherPeers = func() []*Peer {
		return peers
	}

	peer2 := testPeerFree()
	peer2.newPeerAddr = make(chan *peerAddr)
	peer2.otherPeers = func() []*Peer {
		return peers
	}

	rw1, rw2 := MsgPipe()
	fmt.Printf("all set up\n	")

	done := make(chan struct{})
	go func() {
		fmt.Printf("expect handshake\n	")

		if err := expectMsg(rw2, handshakeMsg); err != nil {
			t.Error(err)
		}
		fmt.Printf("send handshake\n	")

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
		fmt.Printf("send getPeers msg\n")

		if err := rw2.EncodeMsg(getPeersMsg); err != nil {
			t.Error(err)
		}
		fmt.Printf("expecting peersMsg\n")
		var msg Msg
		if msg, err = rw2.ReadMsg(); err != nil {
			t.Error(err)
			return
		}

		var addrs []*peerAddr
		fmt.Printf("got peersMsg\n")
		if err := msg.Decode(&addrs); err != nil {
			t.Errorf("msg %v : %v", msg, err)
		}
		fmt.Printf("decoding done\n")

		if len(addrs) != 3 {
			t.Errorf("too few peer addresses, expected %v, got %v", 3, len(addrs))
		}
		fmt.Printf("count ok\n")

		for i, p := range peers {
			if i == len(addrs) {
				break
			}
			addr := addrs[i]
			fmt.Printf("addr %v: %v\n", i, addr)
			if addr != p.listenAddr {
				t.Errorf("incorrect peer address %v (%v)", addr, i)
			}
			if addr == nil {
				t.Errorf("no processing %v", i)
			}
		}
		fmt.Printf("complete\n")
		if err := expectMsg(rw2, peersMsg); err != nil {
			t.Error(err)
		}

		if err := rw2.EncodeMsg(discMsg, DiscQuitting); err != nil {
			t.Error(err)
		}

		close(done)
		fmt.Printf("done channel closed")
	}()

	fmt.Printf("proto")

	if err := runBaseProtocol(peer1, rw1); err == nil {
		t.Errorf("base protocol returned without error")
	} else if reason, ok := err.(discRequestedError); !ok || reason != DiscQuitting {
		t.Errorf("base protocol returned wrong error: %v", err)
	}

	<-done
	t.Error("oops")
}

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
