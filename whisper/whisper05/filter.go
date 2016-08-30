// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package whisper05

import (
	"crypto/ecdsa"

	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type Filter struct {
	Src          *ecdsa.PublicKey  // Sender of the message
	Dst          *ecdsa.PublicKey  // Recipient of the message
	KeyAsym      *ecdsa.PrivateKey // Private Key of recipient
	Topics       []TopicType       // Topics to filter messages with
	KeySym       []byte            // Key associated with the Topic
	TopicKeyHash common.Hash       // The Keccak256Hash of the symmetric key
	PoW          float64           // Proof of work as described in the Whisper spec
	acceptP2P    bool              // Indicates whether this filter is interested in direct peer-to-peer messages

	messages map[common.Hash]*ReceivedMessage
	mutex    sync.RWMutex
}

type Filters struct {
	id       int
	watchers map[int]*Filter
	whisper  *Whisper
	mutex    sync.RWMutex
}

func NewFilters(w *Whisper) *Filters {
	return &Filters{
		watchers: make(map[int]*Filter),
		whisper:  w,
	}
}

func (self *Filters) Install(watcher *Filter) int {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	self.watchers[self.id] = watcher
	ret := self.id
	self.id++
	return ret
}

func (self *Filters) Uninstall(id int) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	delete(self.watchers, id)
}

func (self *Filters) Get(i int) *Filter {
	self.mutex.RLock()
	defer self.mutex.RUnlock()
	return self.watchers[i]
}

func (self *Filters) NotifyWatchers(env *Envelope, messageCode uint64) {
	self.mutex.RLock()
	var msg *ReceivedMessage
	for _, watcher := range self.watchers {
		if messageCode == p2pCode && !watcher.acceptP2P {
			continue
		}

		match := false
		if msg != nil {
			match = watcher.MatchMessage(msg)
		} else {
			match = watcher.MatchEnvelope(env)
			if match {
				msg = env.Open(watcher)
			}
		}

		if match && msg != nil {
			watcher.Trigger(msg)
		}
	}
	self.mutex.RUnlock() // we need to unlock before calling addDecryptedMessage

	if msg != nil {
		self.whisper.addDecryptedMessage(msg)
	}
}

func (self *Filter) expectsAsymmetricEncryption() bool {
	return self.KeyAsym != nil
}

func (self *Filter) expectsSymmetricEncryption() bool {
	return self.KeySym != nil
}

func (self *Filter) Trigger(msg *ReceivedMessage) {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	if _, exist := self.messages[msg.EnvelopeHash]; !exist {
		self.messages[msg.EnvelopeHash] = msg
	}
}

func (self *Filter) retrieve() (all []*ReceivedMessage) {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	all = make([]*ReceivedMessage, 0, len(self.messages))
	for _, msg := range self.messages {
		all = append(all, msg)
	}
	self.messages = make(map[common.Hash]*ReceivedMessage) // delete old messages
	return all
}

func (self *Filter) MatchMessage(msg *ReceivedMessage) bool {
	if self.PoW > 0 && msg.PoW < self.PoW {
		return false
	}

	if self.Src != nil && !isEqual(msg.Src, self.Src) {
		return false
	}

	if self.expectsAsymmetricEncryption() && msg.isAsymmetricEncryption() {
		// if Dst match, ignore the topic
		return isEqual(self.Dst, msg.Dst)
	} else if self.expectsSymmetricEncryption() && msg.isSymmetricEncryption() {
		// check if that both the key and the topic match
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

func (self *Filter) MatchEnvelope(envelope *Envelope) bool {
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
	}

	return false
}

func isEqual(a, b *ecdsa.PublicKey) bool {
	if !validatePublicKey(a) {
		return false
	} else if !validatePublicKey(b) {
		return false
	}
	// the Curve is always the same, just compare the points
	return a.X.Cmp(b.X) == 0 && a.Y.Cmp(b.Y) == 0
}
