package whisper

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"gopkg.in/fatih/set.v0"
)

const (
	protocolVersion = 0x02
)

type peer struct {
	host *Whisper
	peer *p2p.Peer
	ws   p2p.MsgReadWriter

	// XXX Eventually this is going to reach exceptional large space. We need an expiry here
	known *set.Set

	quit chan struct{}
}

func NewPeer(host *Whisper, p *p2p.Peer, ws p2p.MsgReadWriter) *peer {
	return &peer{host, p, ws, set.New(), make(chan struct{})}
}

func (self *peer) init() error {
	if err := self.handleStatus(); err != nil {
		return err
	}

	return nil
}

func (self *peer) start() {
	go self.update()
	self.peer.Infoln("whisper started")
}

func (self *peer) stop() {
	self.peer.Infoln("whisper stopped")

	close(self.quit)
}

func (self *peer) update() {
	relay := time.NewTicker(300 * time.Millisecond)
out:
	for {
		select {
		case <-relay.C:
			err := self.broadcast(self.host.envelopes())
			if err != nil {
				self.peer.Infoln("broadcast err:", err)
				break out
			}

		case <-self.quit:
			break out
		}
	}
}

func (self *peer) broadcast(envelopes []*Envelope) error {
	envs := make([]interface{}, len(envelopes))
	i := 0
	for _, envelope := range envelopes {
		if !self.known.Has(envelope.Hash()) {
			envs[i] = envelope
			self.known.Add(envelope.Hash())
			i++
		}
	}

	if i > 0 {
		if err := p2p.EncodeMsg(self.ws, envelopesMsg, envs[:i]...); err != nil {
			return err
		}
		self.peer.DebugDetailln("broadcasted", i, "message(s)")
	}

	return nil
}

func (self *peer) addKnown(envelope *Envelope) {
	self.known.Add(envelope.Hash())
}

func (self *peer) handleStatus() error {
	ws := self.ws
	if err := ws.WriteMsg(self.statusMsg()); err != nil {
		return err
	}
	msg, err := ws.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != statusMsg {
		return fmt.Errorf("peer send %x before status msg", msg.Code)
	}
	s := rlp.NewStream(msg.Payload)
	if _, err := s.List(); err != nil {
		return fmt.Errorf("bad status message: %v", err)
	}
	pv, err := s.Uint()
	if err != nil {
		return fmt.Errorf("bad status message: %v", err)
	}
	if pv != protocolVersion {
		return fmt.Errorf("protocol version mismatch %d != %d", pv, protocolVersion)
	}
	return msg.Discard() // ignore anything after protocol version
}

func (self *peer) statusMsg() p2p.Msg {
	return p2p.NewMsg(statusMsg, protocolVersion)
}
