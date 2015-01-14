package bzz

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
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

func newTestPeer() (peer *p2p.Peer) {
	// peer = NewPeer(&peerId{}, []p2p.Cap{})
	// peer.pubkeyHook = func(*peerAddr) error { return nil }
	// peer.ourID = &peerId{}
	// peer.listenAddr = &peerAddr{}
	// peer.otherPeers = func() []*Peer { return nil }
	return
}

func expectMsg(r p2p.MsgReader, code uint64) error {
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

func Test(t *testing.T) {}
