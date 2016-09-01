package whisper5

import (
	"bytes"
	"crypto/ecdsa"
)

var empty = TopicType{0, 0, 0, 0}

type Filter struct {
	Src          *ecdsa.PublicKey           // Sender of the message
	Dst          *ecdsa.PublicKey           // Recipient of the message
	KeyAsym      *ecdsa.PrivateKey          // Private Key of recipient
	Topic        TopicType                  // Topics to filter messages with
	KeySym       []byte                     // Key associated with the Topic
	TopicKeyHash []byte                     // The Keccak256Hash of the key, associated with the Topic
	PoW          int                        // Proof of work as described in the Whisper spec
	Fn           func(msg *ReceivedMessage) // Handler in case of a match
}

type Filters struct {
	id       int
	watchers map[int]*Filter
	ch       chan Envelope
	quit     chan struct{}
}

func NewFilters() *Filters {
	return &Filters{
		ch:       make(chan Envelope),
		watchers: make(map[int]*Filter),
		quit:     make(chan struct{}),
	}
}

func (self *Filters) Start() {
	go self.loop()
}

func (self *Filters) Stop() {
	close(self.quit)
}

func (self *Filters) Notify(env *Envelope) {
	self.ch <- *env
}

func (self *Filters) Install(watcher *Filter) int {
	self.watchers[self.id] = watcher
	ret := self.id
	self.id++
	return ret
}

func (self *Filters) Uninstall(id int) {
	delete(self.watchers, id)
}

func (self *Filters) Get(i int) *Filter {
	return self.watchers[i]
}

func (self *Filters) loop() {
	for {
		select {
		case <-self.quit:
			return
		case envelope := <-self.ch:
			self.processEnvelope(&envelope)
		}
	}
}

func (self *Filters) processEnvelope(envelope *Envelope) {
	var msg *ReceivedMessage
	for _, watcher := range self.watchers {
		match := false
		if msg != nil {
			match = watcher.MatchMessage(msg)
		} else {
			match = watcher.MatchEnvelope(envelope)
			if match {
				msg = envelope.Open(watcher) // todo: fill all the fields & validate
			}
		}

		if match && msg != nil {
			watcher.Trigger(msg)
		}
	}
}

func (self Filter) expectsPublicKeyEncryption() bool {
	return self.KeyAsym != nil
}

func (self Filter) expectsTopicEncryption() bool {
	return self.KeySym != nil
}

func (self Filter) Trigger(msg *ReceivedMessage) {
	go self.Fn(msg) // todo: review
}

func (self Filter) MatchMessage(msg *ReceivedMessage) bool {
	if self.PoW > 0 && msg.PoW < self.PoW {
		return false
	}

	if self.expectsPublicKeyEncryption() && msg.isAsymmetric() {
		return self.Dst == msg.Dst
	} else if self.expectsTopicEncryption() && msg.isSymmetric() {
		// we need to compare the keys (or rather thier hashes), because of
		// possible collision (different keys can produce the same topic).
		// we also need to compare the topics, because they could be arbitrary (not related to KeySym).
		if self.Topic == msg.Topic && bytes.Equal(self.TopicKeyHash, msg.TopicKeyHash) {
			return true
		}
	}
	return false
}

func (self Filter) MatchEnvelope(envelope *Envelope) bool {
	if self.PoW > 0 && envelope.pow < self.PoW {
		return false
	}

	encryptionMethodMatch := false
	if self.expectsPublicKeyEncryption() && envelope.isAsymmetric() {
		encryptionMethodMatch = true
	} else if self.expectsTopicEncryption() && envelope.isSymmetric() {
		encryptionMethodMatch = true
	}

	if encryptionMethodMatch {
		if self.Topic == empty || self.Topic == envelope.Topic {
			return true
		}
	}
	return false
}
