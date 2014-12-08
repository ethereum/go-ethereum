package whisper

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"gopkg.in/fatih/set.v0"
)

// MOVE ME
type Hash struct {
	hash string
}

var EmptyHash Hash

func H(hash []byte) Hash {
	return Hash{string(hash)}
}
func HS(hash string) Hash {
	return Hash{hash}
}

// MOVE ME END

const (
	statusMsg    = 0x0
	envelopesMsg = 0x01
)

type Whisper struct {
	pub, sec []byte
	protocol p2p.Protocol

	mmu      sync.RWMutex
	messages map[Hash]*Envelope
	expiry   map[int32]*set.SetNonTS

	quit chan struct{}
}

func New(pub, sec []byte) *Whisper {
	whisper := &Whisper{
		pub:      pub,
		sec:      sec,
		messages: make(map[Hash]*Envelope),
		expiry:   make(map[int32]*set.SetNonTS),
		quit:     make(chan struct{}),
	}
	go whisper.update()

	// p2p whisper sub protocol handler
	whisper.protocol = p2p.Protocol{
		Name:    "shh",
		Version: 2,
		Length:  2,
		Run:     whisper.msgHandler,
	}

	return whisper
}

func (self *Whisper) Stop() {
	close(self.quit)
}

func (self *Whisper) Send(ttl time.Duration, topics [][]byte, data *Message) {
	envelope := NewEnvelope(ttl, topics, data)
	envelope.Seal()

	self.add(envelope)
}

func (self *Whisper) msgHandler(peer *p2p.Peer, ws p2p.MsgReadWriter) error {
	wpeer := NewPeer(self, peer, ws)
	if err := wpeer.init(); err != nil {
		return err
	}
	go wpeer.start()

	for {
		msg, err := ws.ReadMsg()
		if err != nil {
			return err
		}

		envelope, err := NewEnvelopeFromReader(msg.Payload)
		if err != nil {
			peer.Infoln(err)
			continue
		}

		self.add(envelope)
	}
}

func (self *Whisper) add(envelope *Envelope) {
	self.mmu.Lock()
	defer self.mmu.Unlock()

	fmt.Println("received envelope", envelope)
	self.messages[envelope.Hash()] = envelope
	if self.expiry[envelope.Expiry] == nil {
		self.expiry[envelope.Expiry] = set.NewNonTS()
	}
	self.expiry[envelope.Expiry].Add(envelope.Hash())
}

func (self *Whisper) update() {
	expire := time.NewTicker(800 * time.Millisecond)
out:
	for {
		select {
		case <-expire.C:
			self.expire()
		case <-self.quit:
			break out
		}
	}
}
func (self *Whisper) expire() {
	self.mmu.Lock()
	defer self.mmu.Unlock()

	now := int32(time.Now().Unix())
	for then, hashSet := range self.expiry {
		if then > now {
			continue
		}

		hashSet.Each(func(v interface{}) bool {
			delete(self.messages, v.(Hash))
			return true
		})
		self.expiry[then].Clear()
	}
}

func (self *Whisper) envelopes() (envelopes []*Envelope) {
	self.mmu.RLock()
	defer self.mmu.RUnlock()

	envelopes = make([]*Envelope, len(self.messages))
	i := 0
	for _, envelope := range self.messages {
		envelopes[i] = envelope
		i++
	}

	return
}

func (self *Whisper) Protocol() p2p.Protocol {
	return self.protocol
}
