package whisper5

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
)

type Filter struct {
	Src          *ecdsa.PublicKey           // Sender of the message
	Dst          *ecdsa.PublicKey           // Recipient of the message
	KeyAsym      *ecdsa.PrivateKey          // Private Key of recipient
	Topics       []TopicType                // Topics to filter messages with
	KeySym       []byte                     // Key associated with the Topic
	TopicKeyHash common.Hash                // The Keccak256Hash of the symmetric key
	PoW          int                        // Proof of work as described in the Whisper spec
	Fn           func(msg *ReceivedMessage) // Handler in case of a match
}

type Filters struct {
	id       int
	watchers map[int]*Filter
	ch       chan Envelope
	quit     chan struct{}
	whisper  *Whisper
}

func NewFilters(w *Whisper) *Filters {
	return &Filters{
		ch:       make(chan Envelope),
		watchers: make(map[int]*Filter),
		quit:     make(chan struct{}),
		whisper:  w,
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
				msg = envelope.Open(watcher)
			}
		}

		if match && msg != nil {
			watcher.Trigger(msg)
		}
	}

	if msg != nil {
		go self.whisper.addDecryptedMessage(msg)
	}
}

func (self Filter) expectsAsymmetricEncryption() bool {
	return self.KeyAsym != nil
}

func (self Filter) expectsSymmetricEncryption() bool {
	return self.KeySym != nil
}

func (self Filter) Trigger(msg *ReceivedMessage) {
	// todo: save msg hash in the filter
	self.Fn(msg)
}

func (self Filter) MatchMessage(msg *ReceivedMessage) bool {
	if self.PoW > 0 && msg.PoW < self.PoW {
		return false
	}

	if self.Src != nil && msg.Src != self.Src {
		return false
	}

	if self.expectsAsymmetricEncryption() && msg.isAsymmetricEncryption() {
		return self.Dst == msg.Dst
	} else if self.expectsSymmetricEncryption() && msg.isSymmetricEncryption() {
		// we need to compare the keys (or rather thier hashes), because of
		// possible collision (different keys can produce the same topic).
		// we also need to compare the topics, because they could be arbitrary (not related to KeySym).
		if self.TopicKeyHash == msg.TopicKeyHash {
			for _, t := range self.Topics {
				if t == msg.Topic {
					return true
				}
			}
			return false
		}
	}
	return false
}

func (self Filter) MatchEnvelope(envelope *Envelope) bool {
	if self.PoW > 0 && envelope.pow < self.PoW {
		return false
	}

	encryptionMethodMatch := false
	if self.expectsAsymmetricEncryption() && envelope.isAsymmetric() {
		encryptionMethodMatch = true
		if self.Topics == nil {
			return true // wildcard
		}
	} else if self.expectsSymmetricEncryption() && envelope.isSymmetric() {
		encryptionMethodMatch = true
	}

	if encryptionMethodMatch {
		for _, t := range self.Topics {
			if t == envelope.Topic {
				return true
			}
		}
		return false
	}
	return false
}
