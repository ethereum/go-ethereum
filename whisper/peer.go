package whisper

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"gopkg.in/fatih/set.v0"
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
	self.peer.Debugln("whisper started")
}

func (self *peer) stop() {
	self.peer.Debugln("whisper stopped")

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
	envs := make([]*Envelope, 0, len(envelopes))
	for _, env := range envelopes {
		if !self.known.Has(env.Hash()) {
			envs = append(envs, env)
			self.known.Add(env.Hash())
		}
	}
	if len(envs) > 0 {
		if err := p2p.Send(self.ws, envelopesMsg, envs); err != nil {
			return err
		}
		self.peer.DebugDetailln("broadcasted", len(envs), "message(s)")
	}
	return nil
}

func (self *peer) addKnown(envelope *Envelope) {
	self.known.Add(envelope.Hash())
}

func (self *peer) handleStatus() error {
	ws := self.ws
	if err := p2p.SendItems(ws, statusMsg, protocolVersion); err != nil {
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
