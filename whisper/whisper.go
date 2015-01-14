package whisper

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event/filter"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/obscuren/ecies"
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

func (self Hash) Compare(other Hash) int {
	return bytes.Compare([]byte(self.hash), []byte(other.hash))
}

// MOVE ME END

const (
	statusMsg    = 0x0
	envelopesMsg = 0x01
)

type MessageEvent struct {
	To      *ecdsa.PrivateKey
	From    *ecdsa.PublicKey
	Message *Message
}

const DefaultTtl = 50 * time.Second

var wlogger = logger.NewLogger("SHH")

type Whisper struct {
	protocol p2p.Protocol
	filters  *filter.Filters

	mmu      sync.RWMutex
	messages map[Hash]*Envelope
	expiry   map[uint32]*set.SetNonTS

	quit chan struct{}

	keys []*ecdsa.PrivateKey
}

func New() *Whisper {
	whisper := &Whisper{
		messages: make(map[Hash]*Envelope),
		filters:  filter.New(),
		expiry:   make(map[uint32]*set.SetNonTS),
		quit:     make(chan struct{}),
	}
	whisper.filters.Start()

	// p2p whisper sub protocol handler
	whisper.protocol = p2p.Protocol{
		Name:    "shh",
		Version: 2,
		Length:  2,
		Run:     whisper.msgHandler,
	}

	return whisper
}

func (self *Whisper) Start() {
	wlogger.Infoln("Whisper started")
	go self.update()
}

func (self *Whisper) Stop() {
	close(self.quit)
}

func (self *Whisper) Send(envelope *Envelope) error {
	return self.add(envelope)
}

func (self *Whisper) NewIdentity() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	self.keys = append(self.keys, key)

	return key
}

func (self *Whisper) HasIdentity(key *ecdsa.PrivateKey) bool {
	for _, key := range self.keys {
		if key.D.Cmp(key.D) == 0 {
			return true
		}
	}
	return false
}

func (self *Whisper) Watch(opts Filter) int {
	return self.filters.Install(filter.Generic{
		Str1: string(crypto.FromECDSA(opts.To)),
		Str2: string(crypto.FromECDSAPub(opts.From)),
		Data: bytesToMap(opts.Topics),
		Fn: func(data interface{}) {
			opts.Fn(data.(*Message))
		},
	})
}

func (self *Whisper) Messages(id int) (messages []*Message) {
	filter := self.filters.Get(id)
	if filter != nil {
		for _, e := range self.messages {
			if msg, key := self.open(e); msg != nil {
				f := createFilter(msg, e.Topics, key)
				if self.filters.Match(filter, f) {
					messages = append(messages, msg)
				}
			}
		}
	}

	return
}

// Main handler for passing whisper messages to whisper peer objects
func (self *Whisper) msgHandler(peer *p2p.Peer, ws p2p.MsgReadWriter) error {
	wpeer := NewPeer(self, peer, ws)
	// initialise whisper peer (handshake/status)
	if err := wpeer.init(); err != nil {
		return err
	}
	// kick of the main handler for broadcasting/managing envelopes
	go wpeer.start()
	defer wpeer.stop()

	// Main *read* loop. Writing is done by the peer it self.
	for {
		msg, err := ws.ReadMsg()
		if err != nil {
			return err
		}

		var envelopes []*Envelope
		if err := msg.Decode(&envelopes); err != nil {
			peer.Infoln(err)
			continue
		}

		for _, envelope := range envelopes {
			if err := self.add(envelope); err != nil {
				// TODO Punish peer here. Invalid envelope.
				peer.Infoln(err)
			}
			wpeer.addKnown(envelope)
		}
	}
}

// takes care of adding envelopes to the messages pool. At this moment no sanity checks are being performed.
func (self *Whisper) add(envelope *Envelope) error {
	if !envelope.valid() {
		return errors.New("invalid pow provided for envelope")
	}

	self.mmu.Lock()
	defer self.mmu.Unlock()

	hash := envelope.Hash()
	self.messages[hash] = envelope
	if self.expiry[envelope.Expiry] == nil {
		self.expiry[envelope.Expiry] = set.NewNonTS()
	}

	if !self.expiry[envelope.Expiry].Has(hash) {
		self.expiry[envelope.Expiry].Add(hash)
		go self.postEvent(envelope)
	}

	wlogger.DebugDetailln("added whisper message")

	return nil
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

	now := uint32(time.Now().Unix())
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

func (self *Whisper) postEvent(envelope *Envelope) {
	if message, key := self.open(envelope); message != nil {
		self.filters.Notify(createFilter(message, envelope.Topics, key), message)
	}
}

func (self *Whisper) open(envelope *Envelope) (*Message, *ecdsa.PrivateKey) {
	for _, key := range self.keys {
		if message, err := envelope.Open(key); err == nil || (err != nil && err == ecies.ErrInvalidPublicKey) {
			return message, key
		}
	}

	return nil, nil
}

func (self *Whisper) Protocol() p2p.Protocol {
	return self.protocol
}

func createFilter(message *Message, topics [][]byte, key *ecdsa.PrivateKey) filter.Filter {
	return filter.Generic{
		Str1: string(crypto.FromECDSA(key)), Str2: string(crypto.FromECDSAPub(message.Recover())),
		Data: bytesToMap(topics),
	}
}
